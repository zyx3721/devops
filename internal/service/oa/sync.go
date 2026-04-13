package oa

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
	"gorm.io/gorm"

	"devops/internal/domain/notification/service/feishu"
	"devops/internal/models"
	"devops/internal/repository"
	"devops/pkg/logger"
)

const (
	redisSyncedIDsKey = "oa:synced_ids" // Redis Set key for synced IDs
)

var (
	syncLogger  = logger.NewLogger("INFO")
	syncService *OASyncService
	syncOnce    sync.Once
)

// OASyncService OA数据同步服务
type OASyncService struct {
	db               *gorm.DB
	rdb              *redis.Client // Redis客户端
	oaRepo           *repository.OADataRepository
	addrRepo         *repository.OAAddressRepository
	appRepo          *repository.FeishuAppRepository
	notifyConfigRepo *repository.OANotifyConfigRepository
	msgLogRepo       *repository.FeishuMessageLogRepository
	httpClient       *http.Client
	stopChan         chan struct{}
	running          bool
	mu               sync.Mutex
	syncedIDs        map[string]bool // 内存缓存（Redis不可用时的降级方案）
	syncedMu         sync.RWMutex
	feishuClient     *feishu.Client // 飞书客户端（默认）
}

// InitSyncService 初始化同步服务
func InitSyncService(db *gorm.DB) *OASyncService {
	syncOnce.Do(func() {
		syncService = &OASyncService{
			db:               db,
			oaRepo:           repository.NewOADataRepository(db),
			addrRepo:         repository.NewOAAddressRepository(db),
			appRepo:          repository.NewFeishuAppRepository(db),
			notifyConfigRepo: repository.NewOANotifyConfigRepository(db),
			msgLogRepo:       repository.NewFeishuMessageLogRepository(db),
			httpClient: &http.Client{
				Timeout: 30 * time.Second,
			},
			stopChan:  make(chan struct{}),
			syncedIDs: make(map[string]bool),
		}
	})
	return syncService
}

// SetRedis 设置Redis客户端
func (s *OASyncService) SetRedis(rdb *redis.Client) {
	s.rdb = rdb
	if rdb != nil {
		// 从数据库加载已有ID到Redis
		s.loadExistingIDsToRedis()
	}
}

// SetFeishuClient 设置飞书客户端
func (s *OASyncService) SetFeishuClient(client *feishu.Client) {
	s.feishuClient = client
}

// GetSyncService 获取同步服务实例
func GetSyncService() *OASyncService {
	return syncService
}

// loadExistingIDsToRedis 从数据库加载已存在的ID到Redis
func (s *OASyncService) loadExistingIDsToRedis() {
	if s.rdb == nil {
		return
	}
	ctx := context.Background()
	dataList, _, err := s.oaRepo.List(ctx, 1, 10000)
	if err != nil {
		syncLogger.Error("Failed to load existing OA data IDs: %v", err)
		return
	}
	if len(dataList) == 0 {
		return
	}
	// 批量添加到Redis Set
	ids := make([]interface{}, len(dataList))
	for i, data := range dataList {
		ids[i] = data.UniqueID
	}
	if err := s.rdb.SAdd(ctx, redisSyncedIDsKey, ids...).Err(); err != nil {
		syncLogger.Error("Failed to add IDs to Redis: %v", err)
		return
	}
	syncLogger.Info("Loaded %d existing OA data IDs to Redis", len(dataList))
}

// loadExistingIDs 加载已存在的ID到内存缓存（降级方案）
func (s *OASyncService) loadExistingIDs() {
	ctx := context.Background()
	dataList, _, err := s.oaRepo.List(ctx, 1, 10000)
	if err != nil {
		syncLogger.Error("Failed to load existing OA data IDs: %v", err)
		return
	}
	s.syncedMu.Lock()
	defer s.syncedMu.Unlock()
	for _, data := range dataList {
		s.syncedIDs[data.UniqueID] = true
	}
	syncLogger.Info("Loaded %d existing OA data IDs to memory cache", len(s.syncedIDs))
}

// isIDSynced 检查ID是否已同步
func (s *OASyncService) isIDSynced(ctx context.Context, uniqueID string) bool {
	// 优先使用Redis
	if s.rdb != nil {
		exists, err := s.rdb.SIsMember(ctx, redisSyncedIDsKey, uniqueID).Result()
		if err == nil {
			return exists
		}
		syncLogger.Warn("Redis check failed, falling back to memory: %v", err)
	}
	// 降级到内存缓存
	s.syncedMu.RLock()
	exists := s.syncedIDs[uniqueID]
	s.syncedMu.RUnlock()
	return exists
}

// markIDSynced 标记ID为已同步
func (s *OASyncService) markIDSynced(ctx context.Context, uniqueID string) {
	// 优先使用Redis
	if s.rdb != nil {
		if err := s.rdb.SAdd(ctx, redisSyncedIDsKey, uniqueID).Err(); err != nil {
			syncLogger.Warn("Failed to add ID to Redis: %v", err)
		}
	}
	// 同时更新内存缓存
	s.syncedMu.Lock()
	s.syncedIDs[uniqueID] = true
	s.syncedMu.Unlock()
}

// Start 启动同步服务
func (s *OASyncService) Start() {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return
	}
	s.running = true
	s.mu.Unlock()

	syncLogger.Info("OA sync service started")

	// 启动时先执行一次同步
	go s.syncAll()

	// 定时任务：每分钟执行
	ticker := time.NewTicker(1 * time.Minute)
	go func() {
		for {
			select {
			case <-ticker.C:
				s.syncAll()
			case <-s.stopChan:
				ticker.Stop()
				syncLogger.Info("OA sync service stopped")
				return
			}
		}
	}()
}

// Stop 停止同步服务
func (s *OASyncService) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.running {
		return
	}
	s.running = false
	close(s.stopChan)
}

// syncAll 同步所有配置的OA地址
func (s *OASyncService) syncAll() {
	ctx := context.Background()

	// 获取所有启用的OA地址
	addresses, _, err := s.addrRepo.List(ctx, 1, 100)
	if err != nil {
		syncLogger.Error("Failed to get OA addresses: %v", err)
		return
	}

	if len(addresses) == 0 {
		syncLogger.Debug("No OA addresses configured, skipping sync")
		return
	}

	for _, addr := range addresses {
		if addr.Status != "active" {
			continue
		}
		s.syncFromAddress(ctx, &addr)
	}
}

// syncFromAddress 从单个OA地址同步数据
func (s *OASyncService) syncFromAddress(ctx context.Context, addr *models.OAAddress) {
	// 智能处理URL：去掉末尾斜杠和已有的API路径后缀
	baseURL := strings.TrimSuffix(addr.URL, "/")
	// 去掉可能已经存在的后缀
	baseURL = strings.TrimSuffix(baseURL, "/get-json-all")
	baseURL = strings.TrimSuffix(baseURL, "/get-latest-json")
	baseURL = strings.TrimSuffix(baseURL, "/store-json")
	baseURL = strings.TrimSuffix(baseURL, "/api")

	apiURL := fmt.Sprintf("%s/api/get-json-all", baseURL)

	syncLogger.Info("Syncing from OA: %s -> %s", addr.Name, apiURL)

	// 发起HTTP请求
	resp, err := s.httpClient.Get(apiURL)
	if err != nil {
		syncLogger.Error("Failed to fetch from %s: %v", apiURL, err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		syncLogger.Error("OA API returned status %d for %s", resp.StatusCode, apiURL)
		return
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		syncLogger.Error("Failed to read response body: %v", err)
		return
	}

	// 解析响应
	var apiResp struct {
		Code    int                    `json:"code"`
		Message string                 `json:"message"`
		Data    map[string]interface{} `json:"data"`
	}

	if err := json.Unmarshal(body, &apiResp); err != nil {
		syncLogger.Error("Failed to parse API response: %v", err)
		return
	}

	// code 为 0 或 200 都表示成功
	if apiResp.Code != 0 && apiResp.Code != 200 {
		syncLogger.Error("OA API error: code=%d, message=%s", apiResp.Code, apiResp.Message)
		return
	}

	// 处理数据
	newCount := 0
	if apiResp.Data == nil {
		syncLogger.Info("No data returned from %s", addr.Name)
		return
	}

	for uniqueID, data := range apiResp.Data {
		// 检查是否已同步
		if s.isIDSynced(ctx, uniqueID) {
			syncLogger.Debug("Skipping already synced data: %s", uniqueID)
			continue
		}

		syncLogger.Info("Processing new OA data: %s", uniqueID)

		// 保存到数据库
		dataJSON, _ := json.Marshal(data)
		oaData := &models.OAData{
			UniqueID:     uniqueID,
			Source:       addr.Name, // 来源OA地址名称
			OriginalData: string(dataJSON),
			IPAddress:    addr.URL,
		}

		// 尝试从data中提取更多信息
		if dataMap, ok := data.(map[string]interface{}); ok {
			if ip, ok := dataMap["ip_address"].(string); ok {
				oaData.IPAddress = ip
			}
			if ua, ok := dataMap["user_agent"].(string); ok {
				oaData.UserAgent = ua
			}
		}

		// 尝试保存到数据库（可能已存在）
		dbErr := s.oaRepo.Create(ctx, oaData)
		if dbErr != nil {
			syncLogger.Debug("OA data %s already exists in database: %v", uniqueID, dbErr)
		}

		// 标记为已同步
		s.markIDSynced(ctx, uniqueID)

		// 只有数据库插入成功才计数
		if dbErr == nil {
			newCount++
		}

		// 尝试解析并发送飞书卡片（无论数据库是否已存在都发送）
		if dataMap, ok := data.(map[string]interface{}); ok {
			s.processOADataAndSendCard(ctx, uniqueID, dataMap)
		}
	}

	if newCount > 0 {
		syncLogger.Info("Synced %d new records from %s", newCount, addr.Name)
	}
}

// processOADataAndSendCard 处理OA数据并发送飞书卡片
func (s *OASyncService) processOADataAndSendCard(ctx context.Context, uniqueID string, data map[string]interface{}) {
	// 获取所有启用的通知配置
	configs, err := s.notifyConfigRepo.ListActive(ctx)
	if err != nil {
		syncLogger.Error("Failed to get notify configs: %v", err)
		return
	}
	if len(configs) == 0 {
		syncLogger.Warn("No active notify configs found, skipping card send for %s", uniqueID)
		return
	}

	syncLogger.Info("Found %d active notify configs for OA data %s", len(configs), uniqueID)

	// 尝试从 original_data 中提取服务信息
	// OA数据结构: data["original_data"]["fwm"]
	var originalData map[string]interface{}
	if od, ok := data["original_data"].(map[string]interface{}); ok {
		originalData = od
		syncLogger.Debug("Using nested original_data for %s", uniqueID)
	} else {
		// 直接使用 data 作为 originalData（兼容其他格式）
		originalData = data
		syncLogger.Debug("Using data directly as original_data for %s", uniqueID)
	}

	// 检查是否是 OA 审批数据（包含 fwm 字段）
	fwm, ok := originalData["fwm"].(string)
	if !ok || fwm == "" {
		syncLogger.Info("No fwm field found in OA data %s, skipping card send", uniqueID)
		return
	}

	syncLogger.Info("Processing OA data %s with fwm: %s", uniqueID, fwm[:min(len(fwm), 100)])

	// 解析服务列表
	services := s.parseServicesFromFwm(fwm)
	if len(services) == 0 {
		syncLogger.Warn("No valid services parsed from fwm for %s", uniqueID)
		return
	}

	syncLogger.Info("Parsed %d services from OA data %s", len(services), uniqueID)

	// 获取标题信息
	title := "应用发布申请"
	if xqmc, ok := originalData["xqmc"].(string); ok && xqmc != "" {
		title = fmt.Sprintf("应用发布申请 - %s", strings.ReplaceAll(xqmc, "&nbsp;", " "))
	}
	if lcbh, ok := originalData["lcbh"].(string); ok && lcbh != "" {
		title = fmt.Sprintf("%s (%s)", title, lcbh)
	}

	// 向每个配置的接收者发送卡片
	for _, config := range configs {
		syncLogger.Info("Sending card to: %s (receive_id: %s, type: %s)", config.Name, config.ReceiveID, config.ReceiveIDType)
		s.sendCardToReceiver(ctx, uniqueID, title, services, &config)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// getMapKeys 获取map的所有key（用于调试）
func getMapKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// sendCardToReceiver 向单个接收者发送卡片
func (s *OASyncService) sendCardToReceiver(ctx context.Context, uniqueID, title string, services []feishu.Service, config *models.OANotifyConfig) {
	// 获取飞书客户端
	client := s.getFeishuClientByAppID(ctx, config.AppID)
	if client == nil {
		syncLogger.Error("Failed to get Feishu client for config: %s", config.Name)
		return
	}

	// 生成请求ID
	requestID := fmt.Sprintf("oa_%s_%d_%d", uniqueID, config.ID, time.Now().UnixNano())

	// 构建卡片请求
	cardReq := feishu.GrayCardRequest{
		Title:         title,
		Services:      services,
		ReceiveID:     config.ReceiveID,
		ReceiveIDType: config.ReceiveIDType,
	}

	// 保存到飞书存储
	feishu.GlobalStore.Save(requestID, cardReq)

	// 构建并发送卡片
	cardContent := feishu.BuildCard(cardReq, requestID, nil, nil)
	cardBytes, err := json.Marshal(cardContent)
	if err != nil {
		syncLogger.Error("Failed to marshal card content: %v", err)
		return
	}

	err = client.SendMessage(ctx, config.ReceiveID, config.ReceiveIDType, "interactive", string(cardBytes))

	// 记录日志
	if s.msgLogRepo != nil {
		logEntry := &models.FeishuMessageLog{
			MsgType:       "interactive",
			ReceiveID:     config.ReceiveID,
			ReceiveIDType: config.ReceiveIDType,
			Content:       string(cardBytes),
			Title:         title,
			Source:        "oa_sync",
			Status:        "success",
			AppID:         config.AppID,
		}
		if err != nil {
			logEntry.Status = "failed"
			logEntry.ErrorMsg = err.Error()
		}
		s.msgLogRepo.Create(ctx, logEntry)
	}

	if err != nil {
		syncLogger.Error("Failed to send Feishu card to %s (%s): %v", config.Name, config.ReceiveID, err)
		return
	}

	syncLogger.Info("Sent Feishu card for OA data %s to %s with %d services", uniqueID, config.Name, len(services))
}

// getFeishuClientByAppID 根据应用ID获取飞书客户端
func (s *OASyncService) getFeishuClientByAppID(ctx context.Context, appID uint) *feishu.Client {
	// 如果配置了特定应用ID，从数据库获取应用信息创建客户端
	if appID > 0 && s.appRepo != nil {
		app, err := s.appRepo.GetByID(ctx, appID)
		if err == nil && app != nil && app.Status == "active" {
			syncLogger.Debug("Using Feishu app: %s (ID: %d)", app.Name, app.ID)
			return feishu.NewClientWithApp(app.AppID, app.AppSecret)
		}
		syncLogger.Warn("Configured Feishu app (ID: %d) not found or inactive, falling back to default", appID)
	}

	// 使用默认客户端
	return s.feishuClient
}

// parseServicesFromFwm 从 fwm 字段解析服务列表
func (s *OASyncService) parseServicesFromFwm(fwm string) []feishu.Service {
	var services []feishu.Service

	// fwm 格式: "服务名&nbsp;&nbsp;分支<br>服务名&nbsp;&nbsp;分支"
	// 替换 HTML 实体
	fwm = strings.ReplaceAll(fwm, "&nbsp;", " ")

	// 按 <br> 分割
	lines := strings.Split(fwm, "<br>")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// 按空格分割获取服务名和分支
		parts := strings.Fields(line)
		if len(parts) < 2 {
			syncLogger.Warn("Invalid service format: %s", line)
			continue
		}

		serviceName := parts[0]
		branch := parts[1]

		// 默认操作：灰度、回滚、重启
		actions := []string{"gray", "rollback", "restart"}

		services = append(services, feishu.Service{
			Name:     serviceName,
			ObjectID: serviceName,
			Actions:  actions,
			Branches: []string{branch},
		})
	}

	return services
}

// RemoveFromCache 从缓存中移除ID
func (s *OASyncService) RemoveFromCache(uniqueID string) {
	ctx := context.Background()
	// 从Redis移除
	if s.rdb != nil {
		if err := s.rdb.SRem(ctx, redisSyncedIDsKey, uniqueID).Err(); err != nil {
			syncLogger.Warn("Failed to remove ID from Redis: %v", err)
		}
	}
	// 从内存缓存移除
	s.syncedMu.Lock()
	delete(s.syncedIDs, uniqueID)
	s.syncedMu.Unlock()
	syncLogger.Debug("Removed %s from sync cache", uniqueID)
}

// SyncNow 立即执行一次同步
func (s *OASyncService) SyncNow() {
	syncLogger.Info("Manual sync triggered")
	go s.syncAll()
}

// SyncNowForce 强制重新同步（清除缓存）
func (s *OASyncService) SyncNowForce() {
	syncLogger.Info("Force sync triggered, clearing cache")
	ctx := context.Background()
	// 清除Redis缓存
	if s.rdb != nil {
		if err := s.rdb.Del(ctx, redisSyncedIDsKey).Err(); err != nil {
			syncLogger.Warn("Failed to clear Redis cache: %v", err)
		}
	}
	// 清除内存缓存
	s.syncedMu.Lock()
	s.syncedIDs = make(map[string]bool)
	s.syncedMu.Unlock()
	go s.syncAll()
}

// GetSyncStatus 获取同步状态
func (s *OASyncService) GetSyncStatus() map[string]interface{} {
	ctx := context.Background()
	var count int64

	// 优先从Redis获取数量
	if s.rdb != nil {
		c, err := s.rdb.SCard(ctx, redisSyncedIDsKey).Result()
		if err == nil {
			count = c
		}
	}
	// 如果Redis不可用，从内存获取
	if count == 0 {
		s.syncedMu.RLock()
		count = int64(len(s.syncedIDs))
		s.syncedMu.RUnlock()
	}

	s.mu.Lock()
	running := s.running
	s.mu.Unlock()

	return map[string]interface{}{
		"running":      running,
		"synced_count": count,
		"use_redis":    s.rdb != nil,
	}
}

// TestSendCard 测试发送卡片（用于调试）
func (s *OASyncService) TestSendCard(uniqueID string) error {
	ctx := context.Background()

	// 从数据库获取OA数据
	oaData, err := s.oaRepo.GetByUniqueID(ctx, uniqueID)
	if err != nil {
		return fmt.Errorf("OA data not found: %v", err)
	}

	// 解析原始数据
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(oaData.OriginalData), &data); err != nil {
		return fmt.Errorf("failed to parse original data: %v", err)
	}

	syncLogger.Info("Test sending card for OA data: %s", uniqueID)
	s.processOADataAndSendCard(ctx, uniqueID, data)
	return nil
}
