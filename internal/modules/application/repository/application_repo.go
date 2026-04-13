package repository

import (
	"context"
	"time"

	"gorm.io/gorm"

	"devops/internal/models"
)

type ApplicationRepository struct {
	db *gorm.DB
}

func NewApplicationRepository(db *gorm.DB) *ApplicationRepository {
	return &ApplicationRepository{db: db}
}

func (r *ApplicationRepository) Create(ctx context.Context, app *models.Application) error {
	return r.db.WithContext(ctx).Create(app).Error
}

func (r *ApplicationRepository) Update(ctx context.Context, app *models.Application) error {
	return r.db.WithContext(ctx).Save(app).Error
}

func (r *ApplicationRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&models.Application{}, id).Error
}

func (r *ApplicationRepository) GetByID(ctx context.Context, id uint) (*models.Application, error) {
	var app models.Application
	if err := r.db.WithContext(ctx).First(&app, id).Error; err != nil {
		return nil, err
	}
	return &app, nil
}

func (r *ApplicationRepository) GetByName(ctx context.Context, name string) (*models.Application, error) {
	var app models.Application
	if err := r.db.WithContext(ctx).Where("name = ?", name).First(&app).Error; err != nil {
		return nil, err
	}
	return &app, nil
}

func (r *ApplicationRepository) List(ctx context.Context, filter ApplicationFilter, page, pageSize int) ([]models.Application, int64, error) {
	var apps []models.Application
	var total int64

	query := r.db.WithContext(ctx).Model(&models.Application{})

	if filter.Name != "" {
		query = query.Where("name LIKE ? OR display_name LIKE ?", "%"+filter.Name+"%", "%"+filter.Name+"%")
	}
	if filter.Team != "" {
		query = query.Where("team = ?", filter.Team)
	}
	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}
	if filter.Language != "" {
		query = query.Where("language = ?", filter.Language)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&apps).Error; err != nil {
		return nil, 0, err
	}

	return apps, total, nil
}

func (r *ApplicationRepository) GetAllTeams(ctx context.Context) ([]string, error) {
	var teams []string
	if err := r.db.WithContext(ctx).Model(&models.Application{}).Distinct("team").Where("team != ''").Pluck("team", &teams).Error; err != nil {
		return nil, err
	}
	return teams, nil
}

type ApplicationFilter struct {
	Name     string
	Team     string
	Status   string
	Language string
}

// ApplicationEnvRepository 应用环境仓库
type ApplicationEnvRepository struct {
	db *gorm.DB
}

func NewApplicationEnvRepository(db *gorm.DB) *ApplicationEnvRepository {
	return &ApplicationEnvRepository{db: db}
}

func (r *ApplicationEnvRepository) Create(ctx context.Context, env *models.ApplicationEnv) error {
	return r.db.WithContext(ctx).Create(env).Error
}

func (r *ApplicationEnvRepository) Update(ctx context.Context, env *models.ApplicationEnv) error {
	return r.db.WithContext(ctx).Save(env).Error
}

func (r *ApplicationEnvRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&models.ApplicationEnv{}, id).Error
}

func (r *ApplicationEnvRepository) GetByAppID(ctx context.Context, appID uint) ([]models.ApplicationEnv, error) {
	var envs []models.ApplicationEnv
	if err := r.db.WithContext(ctx).Where("application_id = ?", appID).Find(&envs).Error; err != nil {
		return nil, err
	}
	return envs, nil
}

// DeployRecordRepository 部署记录仓库
type DeployRecordRepository struct {
	db *gorm.DB
}

func NewDeployRecordRepository(db *gorm.DB) *DeployRecordRepository {
	return &DeployRecordRepository{db: db}
}

func (r *DeployRecordRepository) Create(ctx context.Context, record *models.DeployRecord) error {
	return r.db.WithContext(ctx).Create(record).Error
}

func (r *DeployRecordRepository) Update(ctx context.Context, record *models.DeployRecord) error {
	return r.db.WithContext(ctx).Save(record).Error
}

func (r *DeployRecordRepository) GetByID(ctx context.Context, id uint) (*models.DeployRecord, error) {
	var record models.DeployRecord
	if err := r.db.WithContext(ctx).First(&record, id).Error; err != nil {
		return nil, err
	}
	return &record, nil
}

func (r *DeployRecordRepository) List(ctx context.Context, filter DeployRecordFilter, page, pageSize int) ([]models.DeployRecord, int64, error) {
	var records []models.DeployRecord
	var total int64

	query := r.db.WithContext(ctx).Model(&models.DeployRecord{})

	if filter.ApplicationID > 0 {
		query = query.Where("application_id = ?", filter.ApplicationID)
	}
	if filter.AppName != "" {
		query = query.Where("app_name LIKE ?", "%"+filter.AppName+"%")
	}
	if filter.EnvName != "" {
		query = query.Where("env_name = ?", filter.EnvName)
	}
	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&records).Error; err != nil {
		return nil, 0, err
	}

	return records, total, nil
}

type DeployRecordFilter struct {
	ApplicationID uint
	AppName       string
	EnvName       string
	Status        string
	NeedApproval  *bool
	DeployType    string
}

// UpdateStatus 更新记录状态
func (r *DeployRecordRepository) UpdateStatus(ctx context.Context, id uint, status string, updates map[string]interface{}) error {
	updates["status"] = status
	return r.db.WithContext(ctx).Model(&models.DeployRecord{}).Where("id = ?", id).Updates(updates).Error
}

// GetLatestSuccess 获取最近一次成功的部署记录
func (r *DeployRecordRepository) GetLatestSuccess(ctx context.Context, appID uint, envName string) (*models.DeployRecord, error) {
	var record models.DeployRecord
	err := r.db.WithContext(ctx).
		Where("application_id = ? AND env_name = ? AND status = ?", appID, envName, "success").
		Order("created_at DESC").
		First(&record).Error
	if err != nil {
		return nil, err
	}
	return &record, nil
}

// GetPendingApproval 获取待审批的记录
func (r *DeployRecordRepository) GetPendingApproval(ctx context.Context, appID uint, envName string) ([]models.DeployRecord, error) {
	var records []models.DeployRecord
	err := r.db.WithContext(ctx).
		Where("application_id = ? AND env_name = ? AND status = ? AND need_approval = ?", appID, envName, "pending", true).
		Find(&records).Error
	return records, err
}

// GetStats 获取统计数据
func (r *DeployRecordRepository) GetStats(ctx context.Context, filter DeployStatsFilter) (*DeployStats, error) {
	var stats DeployStats

	query := r.db.WithContext(ctx).Model(&models.DeployRecord{})

	if filter.ApplicationID > 0 {
		query = query.Where("application_id = ?", filter.ApplicationID)
	}
	if filter.EnvName != "" {
		query = query.Where("env_name = ?", filter.EnvName)
	}
	if !filter.StartTime.IsZero() {
		query = query.Where("created_at >= ?", filter.StartTime)
	}
	if !filter.EndTime.IsZero() {
		query = query.Where("created_at <= ?", filter.EndTime)
	}

	// 总数
	if err := query.Count(&stats.Total).Error; err != nil {
		return nil, err
	}

	// 成功数
	var successCount int64
	r.db.WithContext(ctx).Model(&models.DeployRecord{}).
		Where("status = ?", "success").
		Count(&successCount)
	stats.Success = successCount

	// 失败数
	var failedCount int64
	r.db.WithContext(ctx).Model(&models.DeployRecord{}).
		Where("status = ?", "failed").
		Count(&failedCount)
	stats.Failed = failedCount

	// 平均耗时
	var avgDuration float64
	r.db.WithContext(ctx).Model(&models.DeployRecord{}).
		Where("status = ? AND duration > 0", "success").
		Select("COALESCE(AVG(duration), 0)").Scan(&avgDuration)
	stats.AvgDuration = int(avgDuration)

	// 成功率
	if stats.Total > 0 {
		stats.SuccessRate = float64(stats.Success) / float64(stats.Total) * 100
	}

	return &stats, nil
}

type DeployStatsFilter struct {
	ApplicationID uint
	EnvName       string
	StartTime     time.Time
	EndTime       time.Time
}

type DeployStats struct {
	Total       int64   `json:"total"`
	Success     int64   `json:"success"`
	Failed      int64   `json:"failed"`
	SuccessRate float64 `json:"success_rate"`
	AvgDuration int     `json:"avg_duration"`
}
