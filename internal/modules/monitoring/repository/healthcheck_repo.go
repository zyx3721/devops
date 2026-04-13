package repository

import (
	"context"
	"time"

	"gorm.io/gorm"

	"devops/internal/models"
)

type HealthCheckConfigRepository struct {
	db *gorm.DB
}

func NewHealthCheckConfigRepository(db *gorm.DB) *HealthCheckConfigRepository {
	return &HealthCheckConfigRepository{db: db}
}

func (r *HealthCheckConfigRepository) Create(ctx context.Context, config *models.HealthCheckConfig) error {
	return r.db.WithContext(ctx).Create(config).Error
}

func (r *HealthCheckConfigRepository) Update(ctx context.Context, config *models.HealthCheckConfig) error {
	return r.db.WithContext(ctx).Save(config).Error
}

func (r *HealthCheckConfigRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&models.HealthCheckConfig{}, id).Error
}

func (r *HealthCheckConfigRepository) GetByID(ctx context.Context, id uint) (*models.HealthCheckConfig, error) {
	var config models.HealthCheckConfig
	if err := r.db.WithContext(ctx).First(&config, id).Error; err != nil {
		return nil, err
	}
	return &config, nil
}

func (r *HealthCheckConfigRepository) List(ctx context.Context, checkType string, page, pageSize int) ([]models.HealthCheckConfig, int64, error) {
	var configs []models.HealthCheckConfig
	var total int64

	query := r.db.WithContext(ctx).Model(&models.HealthCheckConfig{})
	if checkType != "" {
		query = query.Where("type = ?", checkType)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&configs).Error; err != nil {
		return nil, 0, err
	}

	return configs, total, nil
}

func (r *HealthCheckConfigRepository) GetEnabled(ctx context.Context) ([]models.HealthCheckConfig, error) {
	var configs []models.HealthCheckConfig
	if err := r.db.WithContext(ctx).Where("enabled = ?", true).Find(&configs).Error; err != nil {
		return nil, err
	}
	return configs, nil
}

func (r *HealthCheckConfigRepository) UpdateStatus(ctx context.Context, id uint, status, errorMsg string) error {
	now := time.Now()
	return r.db.WithContext(ctx).Model(&models.HealthCheckConfig{}).Where("id = ?", id).Updates(map[string]interface{}{
		"last_check_at": now,
		"last_status":   status,
		"last_error":    errorMsg,
	}).Error
}

// GetByType 按类型查询配置
func (r *HealthCheckConfigRepository) GetByType(ctx context.Context, checkType string) ([]models.HealthCheckConfig, error) {
	var configs []models.HealthCheckConfig
	if err := r.db.WithContext(ctx).Where("type = ?", checkType).Find(&configs).Error; err != nil {
		return nil, err
	}
	return configs, nil
}

// CertInfo 证书信息
type CertInfo struct {
	ExpiryDate    time.Time
	DaysRemaining int
	Issuer        string
	Subject       string
	SerialNumber  string
}

// UpdateCertInfo 更新证书信息
func (r *HealthCheckConfigRepository) UpdateCertInfo(ctx context.Context, id uint, certInfo *CertInfo) error {
	return r.db.WithContext(ctx).Model(&models.HealthCheckConfig{}).Where("id = ?", id).Updates(map[string]interface{}{
		"cert_expiry_date":    certInfo.ExpiryDate,
		"cert_days_remaining": certInfo.DaysRemaining,
		"cert_issuer":         certInfo.Issuer,
		"cert_subject":        certInfo.Subject,
		"cert_serial_number":  certInfo.SerialNumber,
	}).Error
}

// BatchCreate 批量创建配置
func (r *HealthCheckConfigRepository) BatchCreate(ctx context.Context, configs []*models.HealthCheckConfig) error {
	return r.db.WithContext(ctx).Create(configs).Error
}

// GetExpiringCerts 查询即将过期的证书
// days: 查询剩余天数小于等于指定天数的证书
func (r *HealthCheckConfigRepository) GetExpiringCerts(ctx context.Context, days int) ([]models.HealthCheckConfig, error) {
	var configs []models.HealthCheckConfig
	if err := r.db.WithContext(ctx).
		Where("type = ? AND enabled = ? AND cert_days_remaining IS NOT NULL AND cert_days_remaining <= ?", "ssl_cert", true, days).
		Order("cert_days_remaining ASC").
		Find(&configs).Error; err != nil {
		return nil, err
	}
	return configs, nil
}

// UpdateAlertInfo 更新告警信息
func (r *HealthCheckConfigRepository) UpdateAlertInfo(ctx context.Context, id uint, alertLevel string, alertTime time.Time) error {
	return r.db.WithContext(ctx).Model(&models.HealthCheckConfig{}).Where("id = ?", id).Updates(map[string]interface{}{
		"last_alert_level": alertLevel,
		"last_alert_at":    alertTime,
	}).Error
}

// ListFilters 列表查询过滤条件
type ListFilters struct {
	Type             string // 检查类型
	AlertLevel       string // 告警级别
	Keyword          string // 关键字搜索
	MaxDaysRemaining int    // 最大剩余天数
	SortBy           string // 排序字段
}

// ListWithFilters 带过滤条件的列表查询
func (r *HealthCheckConfigRepository) ListWithFilters(ctx context.Context, filters *ListFilters, page, pageSize int) ([]models.HealthCheckConfig, int64, error) {
	var configs []models.HealthCheckConfig
	var total int64

	query := r.db.WithContext(ctx).Model(&models.HealthCheckConfig{})

	// 应用过滤条件
	if filters.Type != "" {
		query = query.Where("type = ?", filters.Type)
	}
	if filters.AlertLevel != "" {
		query = query.Where("last_alert_level = ?", filters.AlertLevel)
	}
	if filters.Keyword != "" {
		query = query.Where("name LIKE ? OR url LIKE ?", "%"+filters.Keyword+"%", "%"+filters.Keyword+"%")
	}
	if filters.MaxDaysRemaining > 0 {
		query = query.Where("cert_days_remaining <= ?", filters.MaxDaysRemaining)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 应用排序 - 转换前端排序字段为数据库列名
	orderBy := "created_at DESC" // 默认排序
	switch filters.SortBy {
	case "days_asc":
		orderBy = "cert_days_remaining ASC"
	case "days_desc":
		orderBy = "cert_days_remaining DESC"
	case "created_desc":
		orderBy = "created_at DESC"
	case "created_asc":
		orderBy = "created_at ASC"
	default:
		if filters.SortBy != "" {
			orderBy = filters.SortBy
		}
	}
	query = query.Order(orderBy)

	offset := (page - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).Find(&configs).Error; err != nil {
		return nil, 0, err
	}

	return configs, total, nil
}

// HealthCheckHistoryRepository 健康检查历史仓库
type HealthCheckHistoryRepository struct {
	db *gorm.DB
}

func NewHealthCheckHistoryRepository(db *gorm.DB) *HealthCheckHistoryRepository {
	return &HealthCheckHistoryRepository{db: db}
}

func (r *HealthCheckHistoryRepository) Create(ctx context.Context, history *models.HealthCheckHistory) error {
	return r.db.WithContext(ctx).Create(history).Error
}

func (r *HealthCheckHistoryRepository) List(ctx context.Context, configID uint, page, pageSize int) ([]models.HealthCheckHistory, int64, error) {
	var histories []models.HealthCheckHistory
	var total int64

	query := r.db.WithContext(ctx).Model(&models.HealthCheckHistory{})
	if configID > 0 {
		query = query.Where("config_id = ?", configID)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&histories).Error; err != nil {
		return nil, 0, err
	}

	return histories, total, nil
}

func (r *HealthCheckHistoryRepository) CleanOld(ctx context.Context, days int) error {
	return r.db.WithContext(ctx).Where("created_at < DATE_SUB(NOW(), INTERVAL ? DAY)", days).Delete(&models.HealthCheckHistory{}).Error
}
