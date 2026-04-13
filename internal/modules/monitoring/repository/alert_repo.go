package repository

import (
	"context"
	"time"

	"gorm.io/gorm"

	"devops/internal/models"
)

type AlertConfigRepository struct {
	db *gorm.DB
}

func NewAlertConfigRepository(db *gorm.DB) *AlertConfigRepository {
	return &AlertConfigRepository{db: db}
}

func (r *AlertConfigRepository) Create(ctx context.Context, config *models.AlertConfig) error {
	return r.db.WithContext(ctx).Create(config).Error
}

func (r *AlertConfigRepository) Update(ctx context.Context, config *models.AlertConfig) error {
	return r.db.WithContext(ctx).Save(config).Error
}

func (r *AlertConfigRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&models.AlertConfig{}, id).Error
}

func (r *AlertConfigRepository) GetByID(ctx context.Context, id uint) (*models.AlertConfig, error) {
	var config models.AlertConfig
	if err := r.db.WithContext(ctx).First(&config, id).Error; err != nil {
		return nil, err
	}
	return &config, nil
}

func (r *AlertConfigRepository) List(ctx context.Context, alertType string, page, pageSize int) ([]models.AlertConfig, int64, error) {
	var configs []models.AlertConfig
	var total int64

	query := r.db.WithContext(ctx).Model(&models.AlertConfig{})
	if alertType != "" {
		query = query.Where("type = ?", alertType)
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

func (r *AlertConfigRepository) GetEnabledByType(ctx context.Context, alertType string) ([]models.AlertConfig, error) {
	var configs []models.AlertConfig
	if err := r.db.WithContext(ctx).Where("type = ? AND enabled = ?", alertType, true).Find(&configs).Error; err != nil {
		return nil, err
	}
	return configs, nil
}

// AlertHistoryRepository 告警历史仓库
type AlertHistoryRepository struct {
	db *gorm.DB
}

func NewAlertHistoryRepository(db *gorm.DB) *AlertHistoryRepository {
	return &AlertHistoryRepository{db: db}
}

func (r *AlertHistoryRepository) Create(ctx context.Context, history *models.AlertHistory) error {
	return r.db.WithContext(ctx).Create(history).Error
}

func (r *AlertHistoryRepository) GetByID(ctx context.Context, id uint) (*models.AlertHistory, error) {
	var history models.AlertHistory
	if err := r.db.WithContext(ctx).First(&history, id).Error; err != nil {
		return nil, err
	}
	return &history, nil
}

func (r *AlertHistoryRepository) Update(ctx context.Context, history *models.AlertHistory) error {
	return r.db.WithContext(ctx).Save(history).Error
}

func (r *AlertHistoryRepository) List(ctx context.Context, alertType, ackStatus string, page, pageSize int) ([]models.AlertHistory, int64, error) {
	var histories []models.AlertHistory
	var total int64

	query := r.db.WithContext(ctx).Model(&models.AlertHistory{})
	if alertType != "" {
		query = query.Where("type = ?", alertType)
	}
	if ackStatus != "" {
		query = query.Where("ack_status = ?", ackStatus)
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

// GetPendingForEscalation 获取待升级的告警
func (r *AlertHistoryRepository) GetPendingForEscalation(ctx context.Context, level string, beforeTime time.Time) ([]models.AlertHistory, error) {
	var histories []models.AlertHistory
	err := r.db.WithContext(ctx).
		Where("ack_status = ? AND level = ? AND escalated = ? AND created_at < ?", "pending", level, false, beforeTime).
		Find(&histories).Error
	return histories, err
}

// AlertSilenceRepository 告警静默仓库
type AlertSilenceRepository struct {
	db *gorm.DB
}

func NewAlertSilenceRepository(db *gorm.DB) *AlertSilenceRepository {
	return &AlertSilenceRepository{db: db}
}

func (r *AlertSilenceRepository) Create(ctx context.Context, silence *models.AlertSilence) error {
	return r.db.WithContext(ctx).Create(silence).Error
}

func (r *AlertSilenceRepository) Update(ctx context.Context, silence *models.AlertSilence) error {
	return r.db.WithContext(ctx).Save(silence).Error
}

func (r *AlertSilenceRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&models.AlertSilence{}, id).Error
}

func (r *AlertSilenceRepository) GetByID(ctx context.Context, id uint) (*models.AlertSilence, error) {
	var silence models.AlertSilence
	if err := r.db.WithContext(ctx).First(&silence, id).Error; err != nil {
		return nil, err
	}
	return &silence, nil
}

func (r *AlertSilenceRepository) List(ctx context.Context, status string, page, pageSize int) ([]models.AlertSilence, int64, error) {
	var silences []models.AlertSilence
	var total int64

	query := r.db.WithContext(ctx).Model(&models.AlertSilence{})
	if status != "" {
		query = query.Where("status = ?", status)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&silences).Error; err != nil {
		return nil, 0, err
	}

	return silences, total, nil
}

// GetActiveSilences 获取当前生效的静默规则
func (r *AlertSilenceRepository) GetActiveSilences(ctx context.Context, alertType string) ([]models.AlertSilence, error) {
	var silences []models.AlertSilence
	now := time.Now()
	query := r.db.WithContext(ctx).
		Where("status = ? AND start_time <= ? AND end_time >= ?", "active", now, now)
	if alertType != "" {
		query = query.Where("type IN (?, 'all')", alertType)
	}
	err := query.Find(&silences).Error
	return silences, err
}

// ExpireOldSilences 过期旧的静默规则
func (r *AlertSilenceRepository) ExpireOldSilences(ctx context.Context) error {
	return r.db.WithContext(ctx).
		Model(&models.AlertSilence{}).
		Where("status = ? AND end_time < ?", "active", time.Now()).
		Update("status", "expired").Error
}

// AlertEscalationRepository 告警升级仓库
type AlertEscalationRepository struct {
	db *gorm.DB
}

func NewAlertEscalationRepository(db *gorm.DB) *AlertEscalationRepository {
	return &AlertEscalationRepository{db: db}
}

func (r *AlertEscalationRepository) Create(ctx context.Context, escalation *models.AlertEscalation) error {
	return r.db.WithContext(ctx).Create(escalation).Error
}

func (r *AlertEscalationRepository) Update(ctx context.Context, escalation *models.AlertEscalation) error {
	return r.db.WithContext(ctx).Save(escalation).Error
}

func (r *AlertEscalationRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&models.AlertEscalation{}, id).Error
}

func (r *AlertEscalationRepository) GetByID(ctx context.Context, id uint) (*models.AlertEscalation, error) {
	var escalation models.AlertEscalation
	if err := r.db.WithContext(ctx).First(&escalation, id).Error; err != nil {
		return nil, err
	}
	return &escalation, nil
}

func (r *AlertEscalationRepository) List(ctx context.Context, page, pageSize int) ([]models.AlertEscalation, int64, error) {
	var escalations []models.AlertEscalation
	var total int64

	query := r.db.WithContext(ctx).Model(&models.AlertEscalation{})
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&escalations).Error; err != nil {
		return nil, 0, err
	}

	return escalations, total, nil
}

// GetEnabledByLevel 获取指定级别的启用升级规则
func (r *AlertEscalationRepository) GetEnabledByLevel(ctx context.Context, level string) ([]models.AlertEscalation, error) {
	var escalations []models.AlertEscalation
	err := r.db.WithContext(ctx).
		Where("enabled = ? AND level = ?", true, level).
		Find(&escalations).Error
	return escalations, err
}

// AlertEscalationLogRepository 告警升级记录仓库
type AlertEscalationLogRepository struct {
	db *gorm.DB
}

func NewAlertEscalationLogRepository(db *gorm.DB) *AlertEscalationLogRepository {
	return &AlertEscalationLogRepository{db: db}
}

func (r *AlertEscalationLogRepository) Create(ctx context.Context, log *models.AlertEscalationLog) error {
	return r.db.WithContext(ctx).Create(log).Error
}

func (r *AlertEscalationLogRepository) ListByAlertID(ctx context.Context, alertHistoryID uint) ([]models.AlertEscalationLog, error) {
	var logs []models.AlertEscalationLog
	err := r.db.WithContext(ctx).Where("alert_history_id = ?", alertHistoryID).Order("created_at DESC").Find(&logs).Error
	return logs, err
}
