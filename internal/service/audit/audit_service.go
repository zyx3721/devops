package audit

import (
	"context"
	"encoding/json"
	"time"

	"gorm.io/gorm"

	"devops/internal/models/system"
	"devops/pkg/logger"
)

// AuditAction 审计动作
type AuditAction string

const (
	ActionCreate AuditAction = "create"
	ActionUpdate AuditAction = "update"
	ActionDelete AuditAction = "delete"
	ActionRead   AuditAction = "read"
	ActionLogin  AuditAction = "login"
	ActionLogout AuditAction = "logout"
	ActionExport AuditAction = "export"
	ActionImport AuditAction = "import"
)

// AuditStatus 审计状态
type AuditStatus string

const (
	StatusSuccess AuditStatus = "success"
	StatusFailed  AuditStatus = "failed"
)

// AuditEntry 审计条目
type AuditEntry struct {
	TenantID     *uint
	UserID       *uint
	Username     string
	Action       AuditAction
	ResourceType string
	ResourceID   *uint
	ResourceName string
	OldValue     interface{}
	NewValue     interface{}
	IPAddress    string
	UserAgent    string
	RequestID    string
	TraceID      string
	Status       AuditStatus
	ErrorMessage string
	Duration     int64
}

// AuditService 审计服务
// 支持同步/异步日志记录、批量插入、自动清理等功能
type AuditService struct {
	db         *gorm.DB
	logChan    chan *AuditEntry // 异步日志通道
	batchSize  int              // 批量插入大小（默认 100）
	flushTimer time.Duration    // 刷新间隔（默认 1 秒）
	stopCh     chan struct{}    // 停止信号
}

// NewAuditService 创建审计服务
func NewAuditService(db *gorm.DB) *AuditService {
	s := &AuditService{
		db:         db,
		logChan:    make(chan *AuditEntry, 1000), // 缓冲 1000 条日志
		batchSize:  100,                          // 每批 100 条
		flushTimer: 1 * time.Second,              // 每秒刷新一次
		stopCh:     make(chan struct{}),
	}

	// 启动批量写入 Worker
	go s.startBatchWorker()

	return s
}

// startBatchWorker 启动批量写入 Worker
// 使用缓冲区收集日志，定时或达到批量大小时批量插入数据库
func (s *AuditService) startBatchWorker() {
	buffer := make([]*AuditEntry, 0, s.batchSize)
	ticker := time.NewTicker(s.flushTimer)
	defer ticker.Stop()

	for {
		select {
		case entry := <-s.logChan:
			buffer = append(buffer, entry)
			// 达到批量大小，立即刷新
			if len(buffer) >= s.batchSize {
				s.flushBatch(buffer)
				buffer = buffer[:0] // 清空缓冲区
			}

		case <-ticker.C:
			// 定时刷新（即使未达到批量大小）
			if len(buffer) > 0 {
				s.flushBatch(buffer)
				buffer = buffer[:0]
			}

		case <-s.stopCh:
			// 停止前刷新剩余日志
			if len(buffer) > 0 {
				s.flushBatch(buffer)
			}
			return
		}
	}
}

// flushBatch 批量插入日志
func (s *AuditService) flushBatch(entries []*AuditEntry) {
	if len(entries) == 0 {
		return
	}

	logs := make([]system.AuditLog, len(entries))
	for i, entry := range entries {
		logs[i] = s.convertToModel(entry)
	}

	// 批量插入
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := s.db.WithContext(ctx).CreateInBatches(logs, len(logs)).Error; err != nil {
		logger.L().WithError(err).WithField("count", len(logs)).Error("批量插入审计日志失败")
	} else {
		logger.L().WithField("count", len(logs)).Debug("批量插入审计日志成功")
	}
}

// convertToModel 将 AuditEntry 转换为 system.AuditLog
func (s *AuditService) convertToModel(entry *AuditEntry) system.AuditLog {
	var oldValue, newValue json.RawMessage

	if entry.OldValue != nil {
		if data, err := json.Marshal(entry.OldValue); err == nil {
			oldValue = data
		}
	}

	if entry.NewValue != nil {
		if data, err := json.Marshal(entry.NewValue); err == nil {
			newValue = data
		}
	}

	return system.AuditLog{
		TenantID:     entry.TenantID,
		UserID:       entry.UserID,
		Username:     entry.Username,
		Action:       string(entry.Action),
		ResourceType: entry.ResourceType,
		ResourceID:   entry.ResourceID,
		ResourceName: entry.ResourceName,
		OldValue:     oldValue,
		NewValue:     newValue,
		IPAddress:    entry.IPAddress,
		UserAgent:    entry.UserAgent,
		RequestID:    entry.RequestID,
		TraceID:      entry.TraceID,
		Status:       string(entry.Status),
		ErrorMessage: entry.ErrorMessage,
		Duration:     entry.Duration,
		CreatedAt:    time.Now(),
	}
}

// Log 同步记录审计日志
func (s *AuditService) Log(ctx context.Context, entry *AuditEntry) error {
	log := s.convertToModel(entry)
	return s.db.WithContext(ctx).Create(&log).Error
}

// LogAsync 异步记录审计日志
// 日志会被放入缓冲通道，由后台 Worker 批量插入
func (s *AuditService) LogAsync(entry *AuditEntry) {
	select {
	case s.logChan <- entry:
		// 成功放入通道
	default:
		// 通道已满，记录警告并丢弃
		logger.L().Warn("审计日志通道已满，日志被丢弃")
	}
}

// Stop 停止审计服务
// 会等待所有缓冲的日志写入完成
func (s *AuditService) Stop() {
	close(s.stopCh)
	// 等待 Worker 退出
	time.Sleep(2 * time.Second)
}

// QueryRequest 查询请求
type QueryRequest struct {
	TenantID     *uint
	UserID       *uint
	Action       string
	ResourceType string
	ResourceID   *uint
	RequestID    string
	Status       string
	StartTime    *time.Time
	EndTime      *time.Time
	Page         int
	PageSize     int
}

// QueryResponse 查询响应
type QueryResponse struct {
	List  []system.AuditLog `json:"list"`
	Total int64             `json:"total"`
}

// Query 查询审计日志
func (s *AuditService) Query(ctx context.Context, req *QueryRequest) (*QueryResponse, error) {
	query := s.db.WithContext(ctx).Model(&system.AuditLog{})

	if req.TenantID != nil {
		query = query.Where("tenant_id = ?", *req.TenantID)
	}
	if req.UserID != nil {
		query = query.Where("user_id = ?", *req.UserID)
	}
	if req.Action != "" {
		query = query.Where("action = ?", req.Action)
	}
	if req.ResourceType != "" {
		query = query.Where("resource_type = ?", req.ResourceType)
	}
	if req.ResourceID != nil {
		query = query.Where("resource_id = ?", *req.ResourceID)
	}
	if req.RequestID != "" {
		query = query.Where("request_id = ?", req.RequestID)
	}
	if req.Status != "" {
		query = query.Where("status = ?", req.Status)
	}
	if req.StartTime != nil {
		query = query.Where("created_at >= ?", *req.StartTime)
	}
	if req.EndTime != nil {
		query = query.Where("created_at <= ?", *req.EndTime)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 20
	}

	var logs []system.AuditLog
	offset := (req.Page - 1) * req.PageSize
	if err := query.Order("created_at DESC").Offset(offset).Limit(req.PageSize).Find(&logs).Error; err != nil {
		return nil, err
	}

	return &QueryResponse{
		List:  logs,
		Total: total,
	}, nil
}

// GetByRequestID 根据请求 ID 获取审计日志
func (s *AuditService) GetByRequestID(ctx context.Context, requestID string) ([]system.AuditLog, error) {
	var logs []system.AuditLog
	err := s.db.WithContext(ctx).Where("request_id = ?", requestID).Order("created_at ASC").Find(&logs).Error
	return logs, err
}

// GetByTraceID 根据追踪 ID 获取审计日志
func (s *AuditService) GetByTraceID(ctx context.Context, traceID string) ([]system.AuditLog, error) {
	var logs []system.AuditLog
	err := s.db.WithContext(ctx).Where("trace_id = ?", traceID).Order("created_at ASC").Find(&logs).Error
	return logs, err
}

// GetResourceHistory 获取资源变更历史
func (s *AuditService) GetResourceHistory(ctx context.Context, resourceType string, resourceID uint) ([]system.AuditLog, error) {
	var logs []system.AuditLog
	err := s.db.WithContext(ctx).
		Where("resource_type = ? AND resource_id = ?", resourceType, resourceID).
		Order("created_at DESC").
		Find(&logs).Error
	return logs, err
}

// Export 导出审计日志
func (s *AuditService) Export(ctx context.Context, req *QueryRequest) ([]system.AuditLog, error) {
	query := s.db.WithContext(ctx).Model(&system.AuditLog{})

	if req.TenantID != nil {
		query = query.Where("tenant_id = ?", *req.TenantID)
	}
	if req.StartTime != nil {
		query = query.Where("created_at >= ?", *req.StartTime)
	}
	if req.EndTime != nil {
		query = query.Where("created_at <= ?", *req.EndTime)
	}

	var logs []system.AuditLog
	// 限制导出数量
	if err := query.Order("created_at DESC").Limit(10000).Find(&logs).Error; err != nil {
		return nil, err
	}

	return logs, nil
}

// GetStats 获取审计统计
func (s *AuditService) GetStats(ctx context.Context, tenantID *uint, days int) (map[string]interface{}, error) {
	startTime := time.Now().AddDate(0, 0, -days)

	query := s.db.WithContext(ctx).Model(&system.AuditLog{}).Where("created_at >= ?", startTime)
	if tenantID != nil {
		query = query.Where("tenant_id = ?", *tenantID)
	}

	// 总数
	var total int64
	query.Count(&total)

	// 按动作统计
	var actionStats []struct {
		Action string `json:"action"`
		Count  int64  `json:"count"`
	}
	s.db.WithContext(ctx).Model(&system.AuditLog{}).
		Select("action, count(*) as count").
		Where("created_at >= ?", startTime).
		Group("action").
		Scan(&actionStats)

	// 按状态统计
	var statusStats []struct {
		Status string `json:"status"`
		Count  int64  `json:"count"`
	}
	s.db.WithContext(ctx).Model(&system.AuditLog{}).
		Select("status, count(*) as count").
		Where("created_at >= ?", startTime).
		Group("status").
		Scan(&statusStats)

	return map[string]interface{}{
		"total":        total,
		"by_action":    actionStats,
		"by_status":    statusStats,
		"period_days":  days,
		"period_start": startTime,
	}, nil
}

// Cleanup 清理过期审计日志
func (s *AuditService) Cleanup(ctx context.Context, retentionDays int) (int64, error) {
	cutoff := time.Now().AddDate(0, 0, -retentionDays)
	result := s.db.WithContext(ctx).Where("created_at < ?", cutoff).Delete(&system.AuditLog{})
	return result.RowsAffected, result.Error
}

// StartAutoCleanup 启动自动清理
// retentionDays: 保留天数，超过此天数的日志将被删除
func (s *AuditService) StartAutoCleanup(retentionDays int) {
	go func() {
		ticker := time.NewTicker(24 * time.Hour) // 每天执行一次
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				ctx := context.Background()
				deleted, err := s.Cleanup(ctx, retentionDays)
				if err != nil {
					logger.L().WithError(err).Error("自动清理审计日志失败")
				} else {
					logger.L().WithField("deleted", deleted).Info("自动清理审计日志成功")
				}
			case <-s.stopCh:
				return
			}
		}
	}()
}
