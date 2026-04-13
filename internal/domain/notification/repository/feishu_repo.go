package repository

import (
	"context"

	"gorm.io/gorm"

	"devops/internal/domain/notification/model"
)

// FeishuRequestRepository 飞书请求仓储
type FeishuRequestRepository struct {
	db *gorm.DB
}

func NewFeishuRequestRepository(db *gorm.DB) *FeishuRequestRepository {
	return &FeishuRequestRepository{db: db}
}

func (r *FeishuRequestRepository) Create(ctx context.Context, req *model.FeishuRequest) error {
	return r.db.WithContext(ctx).Create(req).Error
}

func (r *FeishuRequestRepository) GetByRequestID(ctx context.Context, requestID string) (*model.FeishuRequest, error) {
	var req model.FeishuRequest
	err := r.db.WithContext(ctx).Where("request_id = ?", requestID).First(&req).Error
	if err != nil {
		return nil, err
	}
	return &req, nil
}

func (r *FeishuRequestRepository) Update(ctx context.Context, req *model.FeishuRequest) error {
	return r.db.WithContext(ctx).Save(req).Error
}

func (r *FeishuRequestRepository) Delete(ctx context.Context, requestID string) error {
	return r.db.WithContext(ctx).Where("request_id = ?", requestID).Delete(&model.FeishuRequest{}).Error
}

// FeishuAppRepository 飞书应用仓储
type FeishuAppRepository struct {
	db *gorm.DB
}

func NewFeishuAppRepository(db *gorm.DB) *FeishuAppRepository {
	return &FeishuAppRepository{db: db}
}

func (r *FeishuAppRepository) Create(ctx context.Context, app *model.FeishuApp) error {
	return r.db.WithContext(ctx).Create(app).Error
}

func (r *FeishuAppRepository) GetByID(ctx context.Context, id uint) (*model.FeishuApp, error) {
	var app model.FeishuApp
	err := r.db.WithContext(ctx).First(&app, id).Error
	if err != nil {
		return nil, err
	}
	return &app, nil
}

func (r *FeishuAppRepository) GetByAppID(ctx context.Context, appID string) (*model.FeishuApp, error) {
	var app model.FeishuApp
	err := r.db.WithContext(ctx).Where("app_id = ?", appID).First(&app).Error
	if err != nil {
		return nil, err
	}
	return &app, nil
}

func (r *FeishuAppRepository) List(ctx context.Context, page, pageSize int) ([]model.FeishuApp, int64, error) {
	var list []model.FeishuApp
	var total int64

	query := r.db.WithContext(ctx).Model(&model.FeishuApp{})
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).Order("created_at DESC").Find(&list).Error; err != nil {
		return nil, 0, err
	}

	return list, total, nil
}

func (r *FeishuAppRepository) Update(ctx context.Context, app *model.FeishuApp) error {
	return r.db.WithContext(ctx).Model(app).Where("id = ?", app.ID).Updates(map[string]any{
		"name":        app.Name,
		"app_id":      app.AppID,
		"app_secret":  app.AppSecret,
		"webhook":     app.Webhook,
		"project":     app.Project,
		"description": app.Description,
		"status":      app.Status,
		"is_default":  app.IsDefault,
		"created_by":  app.CreatedBy,
	}).Error
}

func (r *FeishuAppRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&model.FeishuApp{}, id).Error
}

func (r *FeishuAppRepository) GetDefault(ctx context.Context) (*model.FeishuApp, error) {
	var app model.FeishuApp
	err := r.db.WithContext(ctx).Where("is_default = ? AND status = ?", true, "active").First(&app).Error
	if err != nil {
		return nil, err
	}
	return &app, nil
}

func (r *FeishuAppRepository) SetDefault(ctx context.Context, id uint) error {
	if err := r.db.WithContext(ctx).Model(&model.FeishuApp{}).Where("is_default = ?", true).Update("is_default", false).Error; err != nil {
		return err
	}
	return r.db.WithContext(ctx).Model(&model.FeishuApp{}).Where("id = ?", id).Update("is_default", true).Error
}

// FeishuBotRepository 飞书机器人仓储
type FeishuBotRepository struct {
	db *gorm.DB
}

func NewFeishuBotRepository(db *gorm.DB) *FeishuBotRepository {
	return &FeishuBotRepository{db: db}
}

func (r *FeishuBotRepository) Create(ctx context.Context, bot *model.FeishuBot) error {
	return r.db.WithContext(ctx).Create(bot).Error
}

func (r *FeishuBotRepository) GetByID(ctx context.Context, id uint) (*model.FeishuBot, error) {
	var bot model.FeishuBot
	err := r.db.WithContext(ctx).First(&bot, id).Error
	if err != nil {
		return nil, err
	}
	return &bot, nil
}

func (r *FeishuBotRepository) List(ctx context.Context, page, pageSize int) ([]model.FeishuBot, int64, error) {
	var list []model.FeishuBot
	var total int64

	query := r.db.WithContext(ctx).Model(&model.FeishuBot{})
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).Order("created_at DESC").Find(&list).Error; err != nil {
		return nil, 0, err
	}

	return list, total, nil
}

func (r *FeishuBotRepository) Update(ctx context.Context, bot *model.FeishuBot) error {
	return r.db.WithContext(ctx).Model(bot).Where("id = ?", bot.ID).Updates(map[string]any{
		"name":                bot.Name,
		"webhook_url":         bot.WebhookURL,
		"project":             bot.Project,
		"secret":              bot.Secret,
		"description":         bot.Description,
		"status":              bot.Status,
		"message_template_id": bot.MessageTemplateID,
		"created_by":          bot.CreatedBy,
	}).Error
}

func (r *FeishuBotRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&model.FeishuBot{}, id).Error
}

// FeishuMessageLogRepository 飞书消息日志仓储
type FeishuMessageLogRepository struct {
	db *gorm.DB
}

func NewFeishuMessageLogRepository(db *gorm.DB) *FeishuMessageLogRepository {
	return &FeishuMessageLogRepository{db: db}
}

func (r *FeishuMessageLogRepository) Create(ctx context.Context, log *model.FeishuMessageLog) error {
	return r.db.WithContext(ctx).Create(log).Error
}

func (r *FeishuMessageLogRepository) List(ctx context.Context, page, pageSize int, msgType, source string) ([]model.FeishuMessageLog, int64, error) {
	var list []model.FeishuMessageLog
	var total int64

	query := r.db.WithContext(ctx).Model(&model.FeishuMessageLog{})
	if msgType != "" {
		query = query.Where("msg_type = ?", msgType)
	}
	if source != "" {
		query = query.Where("source = ?", source)
	}
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).Order("created_at DESC").Find(&list).Error; err != nil {
		return nil, 0, err
	}

	return list, total, nil
}

func (r *FeishuMessageLogRepository) GetByID(ctx context.Context, id uint) (*model.FeishuMessageLog, error) {
	var log model.FeishuMessageLog
	err := r.db.WithContext(ctx).First(&log, id).Error
	if err != nil {
		return nil, err
	}
	return &log, nil
}

// FeishuUserTokenRepository 飞书用户令牌仓储
type FeishuUserTokenRepository struct {
	db *gorm.DB
}

func NewFeishuUserTokenRepository(db *gorm.DB) *FeishuUserTokenRepository {
	return &FeishuUserTokenRepository{db: db}
}

func (r *FeishuUserTokenRepository) GetByAppID(ctx context.Context, appID string) (*model.FeishuUserToken, error) {
	var token model.FeishuUserToken
	err := r.db.WithContext(ctx).Where("app_id = ?", appID).First(&token).Error
	if err != nil {
		return nil, err
	}
	return &token, nil
}

func (r *FeishuUserTokenRepository) Save(ctx context.Context, token *model.FeishuUserToken) error {
	var existing model.FeishuUserToken
	err := r.db.WithContext(ctx).Where("app_id = ?", token.AppID).First(&existing).Error
	if err == nil {
		return r.db.WithContext(ctx).Model(&existing).Updates(map[string]any{
			"access_token":  token.AccessToken,
			"refresh_token": token.RefreshToken,
			"expires_at":    token.ExpiresAt,
		}).Error
	}
	return r.db.WithContext(ctx).Create(token).Error
}

func (r *FeishuUserTokenRepository) Delete(ctx context.Context, appID string) error {
	return r.db.WithContext(ctx).Where("app_id = ?", appID).Delete(&model.FeishuUserToken{}).Error
}

// GetLatest 获取最新的一条用户令牌（用于 app_id 不匹配时的 fallback）
func (r *FeishuUserTokenRepository) GetLatest(ctx context.Context) (*model.FeishuUserToken, error) {
	var token model.FeishuUserToken
	err := r.db.WithContext(ctx).Order("updated_at DESC").First(&token).Error
	if err != nil {
		return nil, err
	}
	return &token, nil
}
