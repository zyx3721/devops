package security

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"time"

	"gorm.io/gorm"

	"devops/internal/models"
	"devops/pkg/dto"
	"devops/pkg/logger"
)

// AuditLoggerService 审计日志服务
type AuditLoggerService struct {
	db       *gorm.DB
	logChan  chan *models.SecurityAuditLog
	stopChan chan struct{}
}

// NewAuditLoggerService 创建审计日志服务
func NewAuditLoggerService(db *gorm.DB) *AuditLoggerService {
	s := &AuditLoggerService{
		db:       db,
		logChan:  make(chan *models.SecurityAuditLog, 1000),
		stopChan: make(chan struct{}),
	}

	// 启动异步写入协程
	go s.asyncWriter()

	return s
}

// asyncWriter 异步写入
func (s *AuditLoggerService) asyncWriter() {
	batch := make([]*models.SecurityAuditLog, 0, 100)
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case log := <-s.logChan:
			batch = append(batch, log)
			if len(batch) >= 100 {
				s.writeBatch(batch)
				batch = batch[:0]
			}
		case <-ticker.C:
			if len(batch) > 0 {
				s.writeBatch(batch)
				batch = batch[:0]
			}
		case <-s.stopChan:
			if len(batch) > 0 {
				s.writeBatch(batch)
			}
			return
		}
	}
}

// writeBatch 批量写入
func (s *AuditLoggerService) writeBatch(logs []*models.SecurityAuditLog) {
	if len(logs) == 0 {
		return
	}

	if err := s.db.CreateInBatches(logs, 100).Error; err != nil {
		logger.L().WithField("error", err).Error("批量写入审计日志失败")
	}
}

// Stop 停止服务
func (s *AuditLoggerService) Stop() {
	close(s.stopChan)
}

// Log 记录审计日志（异步）
func (s *AuditLoggerService) Log(ctx context.Context, log *models.SecurityAuditLog) {
	log.CreatedAt = time.Now()

	select {
	case s.logChan <- log:
	default:
		// 通道满了，直接写入
		if err := s.db.Create(log).Error; err != nil {
			logger.L().WithField("error", err).Error("写入审计日志失败")
		}
	}
}

// LogSync 同步记录审计日志
func (s *AuditLoggerService) LogSync(ctx context.Context, log *models.SecurityAuditLog) error {
	log.CreatedAt = time.Now()
	return s.db.Create(log).Error
}

// List 查询审计日志
func (s *AuditLoggerService) List(ctx context.Context, req *dto.AuditLogRequest) (*dto.AuditLogResponse, error) {
	var logs []models.SecurityAuditLog
	var total int64

	query := s.db.Model(&models.SecurityAuditLog{})

	if req.UserID > 0 {
		query = query.Where("user_id = ?", req.UserID)
	}
	if req.Action != "" {
		query = query.Where("action = ?", req.Action)
	}
	if req.ResourceType != "" {
		query = query.Where("resource_type = ?", req.ResourceType)
	}
	if req.ClusterID > 0 {
		query = query.Where("cluster_id = ?", req.ClusterID)
	}
	if req.StartTime != "" {
		startTime, err := time.Parse("2006-01-02", req.StartTime)
		if err == nil {
			query = query.Where("created_at >= ?", startTime)
		}
	}
	if req.EndTime != "" {
		endTime, err := time.Parse("2006-01-02", req.EndTime)
		if err == nil {
			query = query.Where("created_at < ?", endTime.AddDate(0, 0, 1))
		}
	}

	query.Count(&total)

	page := req.Page
	if page <= 0 {
		page = 1
	}
	pageSize := req.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}

	query.Order("created_at DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&logs)

	items := make([]dto.AuditLogItem, 0, len(logs))
	for _, log := range logs {
		items = append(items, dto.AuditLogItem{
			ID:           log.ID,
			UserID:       log.UserID,
			Username:     log.Username,
			Action:       log.Action,
			ResourceType: log.ResourceType,
			ResourceName: log.ResourceName,
			Namespace:    log.Namespace,
			ClusterID:    log.ClusterID,
			ClusterName:  log.ClusterName,
			Detail:       log.Detail,
			Result:       log.Result,
			ClientIP:     log.ClientIP,
			CreatedAt:    log.CreatedAt,
		})
	}

	return &dto.AuditLogResponse{
		Total: int(total),
		Items: items,
	}, nil
}

// Export 导出审计日志
func (s *AuditLoggerService) Export(ctx context.Context, req *dto.AuditLogRequest, format string) ([]byte, string, error) {
	// 获取所有符合条件的日志
	req.Page = 1
	req.PageSize = 10000 // 最多导出1万条

	result, err := s.List(ctx, req)
	if err != nil {
		return nil, "", err
	}

	switch format {
	case "json":
		data, err := json.MarshalIndent(result.Items, "", "  ")
		return data, "application/json", err

	case "csv":
		return s.exportCSV(result.Items)

	default:
		return nil, "", fmt.Errorf("不支持的导出格式: %s", format)
	}
}

// exportCSV 导出CSV
func (s *AuditLoggerService) exportCSV(items []dto.AuditLogItem) ([]byte, string, error) {
	var buf []byte
	writer := csv.NewWriter(&csvBuffer{buf: &buf})

	// 写入表头
	headers := []string{"ID", "用户", "操作", "资源类型", "资源名称", "命名空间", "集群", "结果", "IP", "时间"}
	writer.Write(headers)

	// 写入数据
	for _, item := range items {
		row := []string{
			fmt.Sprintf("%d", item.ID),
			item.Username,
			item.Action,
			item.ResourceType,
			item.ResourceName,
			item.Namespace,
			item.ClusterName,
			item.Result,
			item.ClientIP,
			item.CreatedAt.Format("2006-01-02 15:04:05"),
		}
		writer.Write(row)
	}

	writer.Flush()
	return buf, "text/csv", writer.Error()
}

// csvBuffer CSV缓冲区
type csvBuffer struct {
	buf *[]byte
}

func (b *csvBuffer) Write(p []byte) (n int, err error) {
	*b.buf = append(*b.buf, p...)
	return len(p), nil
}
