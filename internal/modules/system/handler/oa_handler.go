package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"devops/internal/config"
	"devops/internal/domain/notification/service/feishu"
	"devops/internal/models"
	"devops/internal/repository"
	"devops/internal/service/oa"
	"devops/pkg/dto"
	apperrors "devops/pkg/errors"
	"devops/pkg/ioc"
)

func init() {
	ioc.Api.RegisterContainer("OAHandler", &OAApiHandler{})
}

type OAApiHandler struct {
	handler *OAHandler
}

func (h *OAApiHandler) Init() error {
	cfg, err := config.LoadConfig()
	if err != nil {
		return err
	}

	db := cfg.GetDB()

	// 初始化 OA 存储
	oa.InitOAStore(db)

	// 初始化飞书客户端
	feishuClient := feishu.NewClient(cfg)

	// 初始化并启动同步服务
	syncService := oa.InitSyncService(db)
	syncService.SetFeishuClient(feishuClient)
	// 设置Redis客户端
	if rdb := cfg.GetRedis(); rdb != nil {
		syncService.SetRedis(rdb)
	}
	syncService.Start()

	h.handler = NewOAHandler(db, feishuClient)

	root := cfg.Application.GinRootRouter().Group("oa")
	h.Register(root)

	return nil
}

func (h *OAApiHandler) Register(r gin.IRouter) {
	// OA 数据接口 - 保持原有路径供外部系统调用
	r.POST("/api/store-json", h.handler.StoreJsonHandler)
	r.GET("/api/get-json/:id", h.handler.GetJsonHandler)
	r.GET("/api/get-json-all", h.handler.GetJsonBatchHandler)
	r.GET("/api/get-latest-json", h.handler.GetLatestJsonHandler)

	// OA 同步数据接口
	r.GET("/data", h.handler.ListOAData)
	r.DELETE("/data/:id", h.handler.DeleteOAData)

	// OA 地址管理接口
	addr := r.Group("/address")
	{
		addr.GET("", h.handler.ListAddresses)
		addr.GET("/:id", h.handler.GetAddress)
		addr.POST("", h.handler.CreateAddress)
		addr.PUT("/:id", h.handler.UpdateAddress)
		addr.DELETE("/:id", h.handler.DeleteAddress)
		addr.POST("/:id/test-connection", h.handler.TestAddressConnection)
	}

	// 通知配置管理接口
	notify := r.Group("/notify")
	{
		notify.GET("", h.handler.ListNotifyConfigs)
		notify.GET("/:id", h.handler.GetNotifyConfig)
		notify.POST("", h.handler.CreateNotifyConfig)
		notify.PUT("/:id", h.handler.UpdateNotifyConfig)
		notify.DELETE("/:id", h.handler.DeleteNotifyConfig)
		notify.POST("/:id/default", h.handler.SetDefaultNotifyConfig)
	}

	// 同步管理接口
	sync := r.Group("/sync")
	{
		sync.GET("/status", h.handler.GetSyncStatus)
		sync.POST("/now", h.handler.SyncNow)
		sync.POST("/force", h.handler.SyncNowForce)
		sync.POST("/test-card/:unique_id", h.handler.TestSendCard)
	}
}

type OAHandler struct {
	addrRepo     *repository.OAAddressRepository
	dataRepo     *repository.OADataRepository
	notifyRepo   *repository.OANotifyConfigRepository
	feishuClient *feishu.Client
}

func NewOAHandler(db *gorm.DB, feishuClient *feishu.Client) *OAHandler {
	return &OAHandler{
		addrRepo:     repository.NewOAAddressRepository(db),
		dataRepo:     repository.NewOADataRepository(db),
		notifyRepo:   repository.NewOANotifyConfigRepository(db),
		feishuClient: feishuClient,
	}
}

func (h *OAHandler) StoreJsonHandler(c *gin.Context) {
	body, err := c.GetRawData()
	if err != nil {
		c.JSON(http.StatusInternalServerError, oa.APIResponse{
			Code:    apperrors.ErrCodeInternalError,
			Message: "Failed to read request body",
		})
		return
	}

	if !json.Valid(body) {
		c.JSON(http.StatusBadRequest, oa.APIResponse{
			Code:    apperrors.ErrCodeInvalidParams,
			Message: "Invalid JSON",
		})
		return
	}

	var originalData map[string]interface{}
	if err1 := json.Unmarshal(body, &originalData); err1 != nil {
		var jsonString string
		if err2 := json.Unmarshal(body, &jsonString); err2 == nil {
			if err3 := json.Unmarshal([]byte(jsonString), &originalData); err3 != nil {
				originalData = map[string]interface{}{
					"_raw_json_string": jsonString,
					"_parse_error":     err3.Error(),
				}
			}
		} else {
			var genericData interface{}
			if err4 := json.Unmarshal(body, &genericData); err4 == nil {
				originalData = map[string]interface{}{
					"_parsed_data": genericData,
					"_data_type":   fmt.Sprintf("%T", genericData),
				}
			} else {
				c.JSON(http.StatusBadRequest, oa.APIResponse{
					Code:    apperrors.ErrCodeInvalidParams,
					Message: "无效的JSON格式: " + err1.Error(),
				})
				return
			}
		}
	}

	id := uuid.New().String()[:8]
	timestamp := time.Now().Format("20060102_150405")
	uniqueID := fmt.Sprintf("%s_%s", timestamp, id)

	storedJSON := oa.StoredJSON{
		ID:           uniqueID,
		ReceivedAt:   time.Now().Format(time.RFC3339),
		IPAddress:    c.ClientIP(),
		UserAgent:    c.GetHeader("User-Agent"),
		OriginalData: originalData,
	}

	err = oa.SaveToDisk(uniqueID, storedJSON)
	if err != nil {
		c.JSON(http.StatusInternalServerError, oa.APIResponse{
			Code:    apperrors.ErrCodeInternalError,
			Message: "Failed to save data",
		})
		return
	}

	c.JSON(http.StatusOK, oa.APIResponse{
		Code:    apperrors.Success,
		Message: "Success",
		Data: map[string]interface{}{
			"id": uniqueID,
		},
	})
}

func (h *OAHandler) GetJsonHandler(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, oa.APIResponse{
			Code:    apperrors.ErrCodeInvalidParams,
			Message: "ID is required",
		})
		return
	}

	req, err := oa.LoadFromDisk(id)
	if err != nil {
		c.JSON(http.StatusNotFound, oa.APIResponse{
			Code:    apperrors.ErrCodeNotFound,
			Message: "Data not found",
		})
		return
	}

	c.JSON(http.StatusOK, oa.APIResponse{
		Code:    apperrors.Success,
		Message: "Success",
		Data:    req,
	})
}

func (h *OAHandler) GetJsonBatchHandler(c *gin.Context) {
	reqs, err := oa.LoadJsonFromDiskALL()
	if err != nil {
		c.JSON(http.StatusOK, oa.APIResponse{
			Code:    apperrors.Success,
			Message: "Success",
			Data:    []interface{}{},
		})
		return
	}

	// 转换为数组格式
	var list []map[string]interface{}
	for _, v := range reqs {
		if item, ok := v.(map[string]interface{}); ok {
			list = append(list, item)
		}
	}

	if list == nil {
		list = []map[string]interface{}{}
	}

	c.JSON(http.StatusOK, oa.APIResponse{
		Code:    apperrors.Success,
		Message: "Success",
		Data:    list,
	})
}

func (h *OAHandler) GetLatestJsonHandler(c *gin.Context) {
	latestFile, err := oa.GetLatestJsonFileContent()
	if err != nil {
		c.JSON(http.StatusOK, oa.APIResponse{
			Code:    apperrors.Success,
			Message: "No data found",
			Data: map[string]interface{}{
				"latest_file": nil,
			},
		})
		return
	}

	c.JSON(http.StatusOK, oa.APIResponse{
		Code:    apperrors.Success,
		Message: "Success",
		Data: map[string]interface{}{
			"latest_file": latestFile,
		},
	})
}

// OA 地址管理

func (h *OAHandler) ListAddresses(c *gin.Context) {
	if h.addrRepo == nil {
		c.JSON(http.StatusOK, gin.H{"code": 0, "message": "Success", "data": []interface{}{}})
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "100"))

	list, total, err := h.addrRepo.List(c.Request.Context(), page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "Success",
		"data": gin.H{
			"list":  list,
			"total": total,
		},
	})
}

func (h *OAHandler) GetAddress(c *gin.Context) {
	if h.addrRepo == nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "Not found"})
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "Invalid ID"})
		return
	}

	addr, err := h.addrRepo.GetByID(c.Request.Context(), uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "Not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "Success", "data": addr})
}

func (h *OAHandler) CreateAddress(c *gin.Context) {
	if h.addrRepo == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "Repository not initialized"})
		return
	}

	var addr models.OAAddress
	if err := c.ShouldBindJSON(&addr); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	// 创建时清除ID，让数据库自动生成
	addr.ID = 0

	if err := h.addrRepo.Create(c.Request.Context(), &addr); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "Success", "data": addr})
}

func (h *OAHandler) UpdateAddress(c *gin.Context) {
	if h.addrRepo == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "Repository not initialized"})
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "Invalid ID"})
		return
	}

	var addr models.OAAddress
	if err := c.ShouldBindJSON(&addr); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	addr.ID = uint(id)
	if err := h.addrRepo.Update(c.Request.Context(), &addr); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "Success", "data": addr})
}

func (h *OAHandler) DeleteAddress(c *gin.Context) {
	if h.addrRepo == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "Repository not initialized"})
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "Invalid ID"})
		return
	}

	if err := h.addrRepo.Delete(c.Request.Context(), uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "Success"})
}

// TestAddressConnection 测试 OA 地址连通性
func (h *OAHandler) TestAddressConnection(c *gin.Context) {
	if h.addrRepo == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "Repository not initialized"})
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "Invalid ID"})
		return
	}

	addr, err := h.addrRepo.GetByID(c.Request.Context(), uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "Address not found"})
		return
	}

	result := dto.ConnectionTestResult{}
	startTime := time.Now()

	// 创建 HTTP 客户端，设置超时
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	// 发送 GET 请求测试连通性
	resp, err := client.Get(addr.URL)
	result.ResponseTimeMs = time.Since(startTime).Milliseconds()

	if err != nil {
		result.Connected = false
		result.Error = err.Error()
		c.JSON(http.StatusOK, gin.H{"code": 0, "message": "Success", "data": result})
		return
	}
	defer resp.Body.Close()

	result.Connected = true
	result.Version = fmt.Sprintf("HTTP %s", resp.Proto)
	result.ServerVersion = resp.Header.Get("Server")

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "Success", "data": result})
}

// 同步管理

func (h *OAHandler) GetSyncStatus(c *gin.Context) {
	syncService := oa.GetSyncService()
	if syncService == nil {
		c.JSON(http.StatusOK, gin.H{
			"code":    0,
			"message": "Success",
			"data": gin.H{
				"running":      false,
				"synced_count": 0,
			},
		})
		return
	}

	status := syncService.GetSyncStatus()
	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "Success",
		"data":    status,
	})
}

func (h *OAHandler) SyncNow(c *gin.Context) {
	syncService := oa.GetSyncService()
	if syncService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "Sync service not initialized"})
		return
	}

	syncService.SyncNow()
	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "Sync triggered",
	})
}

// SyncNowForce 强制重新同步（清除缓存）
func (h *OAHandler) SyncNowForce(c *gin.Context) {
	syncService := oa.GetSyncService()
	if syncService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "Sync service not initialized"})
		return
	}

	syncService.SyncNowForce()
	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "Force sync triggered, cache cleared",
	})
}

// 通知配置管理

func (h *OAHandler) ListNotifyConfigs(c *gin.Context) {
	if h.notifyRepo == nil {
		c.JSON(http.StatusOK, gin.H{"code": 0, "message": "Success", "data": gin.H{"list": []interface{}{}, "total": 0}})
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "100"))

	list, total, err := h.notifyRepo.List(c.Request.Context(), page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "Success",
		"data": gin.H{
			"list":  list,
			"total": total,
		},
	})
}

func (h *OAHandler) GetNotifyConfig(c *gin.Context) {
	if h.notifyRepo == nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "Not found"})
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "Invalid ID"})
		return
	}

	config, err := h.notifyRepo.GetByID(c.Request.Context(), uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "Not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "Success", "data": config})
}

func (h *OAHandler) CreateNotifyConfig(c *gin.Context) {
	if h.notifyRepo == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "Repository not initialized"})
		return
	}

	var config models.OANotifyConfig
	if err := c.ShouldBindJSON(&config); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	config.ID = 0
	if err := h.notifyRepo.Create(c.Request.Context(), &config); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "Success", "data": config})
}

func (h *OAHandler) UpdateNotifyConfig(c *gin.Context) {
	if h.notifyRepo == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "Repository not initialized"})
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "Invalid ID"})
		return
	}

	var config models.OANotifyConfig
	if err := c.ShouldBindJSON(&config); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	config.ID = uint(id)
	if err := h.notifyRepo.Update(c.Request.Context(), &config); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "Success", "data": config})
}

func (h *OAHandler) DeleteNotifyConfig(c *gin.Context) {
	if h.notifyRepo == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "Repository not initialized"})
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "Invalid ID"})
		return
	}

	if err := h.notifyRepo.Delete(c.Request.Context(), uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "Success"})
}

func (h *OAHandler) SetDefaultNotifyConfig(c *gin.Context) {
	if h.notifyRepo == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "Repository not initialized"})
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "Invalid ID"})
		return
	}

	if err := h.notifyRepo.SetDefault(c.Request.Context(), uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "Success"})
}

// TestSendCard 测试发送卡片
func (h *OAHandler) TestSendCard(c *gin.Context) {
	uniqueID := c.Param("unique_id")
	if uniqueID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "unique_id is required"})
		return
	}

	syncService := oa.GetSyncService()
	if syncService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "Sync service not initialized"})
		return
	}

	if err := syncService.TestSendCard(uniqueID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "Test card sent"})
}

// ListOAData 获取同步的OA数据列表，支持按来源搜索
func (h *OAHandler) ListOAData(c *gin.Context) {
	if h.dataRepo == nil {
		c.JSON(http.StatusOK, gin.H{"code": 0, "message": "Success", "data": gin.H{"list": []interface{}{}, "total": 0}})
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	source := c.Query("source") // 按来源搜索

	list, total, err := h.dataRepo.ListBySource(c.Request.Context(), source, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "Success",
		"data": gin.H{
			"list":  list,
			"total": total,
		},
	})
}

// DeleteOAData 删除OA数据
func (h *OAHandler) DeleteOAData(c *gin.Context) {
	if h.dataRepo == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "Repository not initialized"})
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "Invalid ID"})
		return
	}

	// 先获取数据的unique_id，用于清除缓存
	data, err := h.dataRepo.GetByID(c.Request.Context(), uint(id))
	if err == nil && data != nil {
		// 从同步缓存中移除
		syncService := oa.GetSyncService()
		if syncService != nil {
			syncService.RemoveFromCache(data.UniqueID)
		}
	}

	if err := h.dataRepo.Delete(c.Request.Context(), uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "Success"})
}
