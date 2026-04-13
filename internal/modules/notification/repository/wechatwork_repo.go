package repository

import (
	"context"

	"gorm.io/gorm"

	"devops/internal/models"
)

// WechatWorkAppRepository 企业微信应用仓储
type WechatWorkAppRepository struct {
	db *gorm.DB
}

func NewWechatWorkAppRepository(db *gorm.DB) *WechatWorkAppRepository {
	return &WechatWorkAppRepository{db: db}
}

func (r *WechatWorkAppRepository) Create(ctx context.Context, app *models.WechatWorkApp) error {
	return r.db.WithContext(ctx).Create(app).Error
}

func (r *WechatWorkAppRepository) GetByID(ctx context.Context, id uint) (*models.WechatWorkApp, error) {
	var app models.WechatWorkApp
	err := r.db.WithContext(ctx).First(&app, id).Error
	if err != nil {
		return nil, err
	}
	return &app, nil
}

func (r *WechatWorkAppRepository) List(ctx context.Context, page, pageSize int) ([]models.WechatWorkApp, int64, error) {
	var list []models.WechatWorkApp
	var total int64

	query := r.db.WithContext(ctx).Model(&models.WechatWorkApp{})
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).Order("created_at DESC").Find(&list).Error; err != nil {
		return nil, 0, err
	}

	return list, total, nil
}

func (r *WechatWorkAppRepository) Update(ctx context.Context, app *models.WechatWorkApp) error {
	return r.db.WithContext(ctx).Model(app).Where("id = ?", app.ID).Updates(map[string]interface{}{
		"name":        app.Name,
		"corp_id":     app.CorpID,
		"agent_id":    app.AgentID,
		"secret":      app.Secret,
		"project":     app.Project,
		"description": app.Description,
		"status":      app.Status,
		"is_default":  app.IsDefault,
	}).Error
}

func (r *WechatWorkAppRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&models.WechatWorkApp{}, id).Error
}

func (r *WechatWorkAppRepository) GetDefault(ctx context.Context) (*models.WechatWorkApp, error) {
	var app models.WechatWorkApp
	err := r.db.WithContext(ctx).Where("is_default = ? AND status = ?", true, "active").First(&app).Error
	if err != nil {
		return nil, err
	}
	return &app, nil
}

func (r *WechatWorkAppRepository) SetDefault(ctx context.Context, id uint) error {
	if err := r.db.WithContext(ctx).Model(&models.WechatWorkApp{}).Where("is_default = ?", true).Update("is_default", false).Error; err != nil {
		return err
	}
	return r.db.WithContext(ctx).Model(&models.WechatWorkApp{}).Where("id = ?", id).Update("is_default", true).Error
}

// WechatWorkBotRepository 企业微信机器人仓储
type WechatWorkBotRepository struct {
	db *gorm.DB
}

func NewWechatWorkBotRepository(db *gorm.DB) *WechatWorkBotRepository {
	return &WechatWorkBotRepository{db: db}
}

func (r *WechatWorkBotRepository) Create(ctx context.Context, bot *models.WechatWorkBot) error {
	return r.db.WithContext(ctx).Create(bot).Error
}

func (r *WechatWorkBotRepository) GetByID(ctx context.Context, id uint) (*models.WechatWorkBot, error) {
	var bot models.WechatWorkBot
	err := r.db.WithContext(ctx).First(&bot, id).Error
	if err != nil {
		return nil, err
	}
	return &bot, nil
}

func (r *WechatWorkBotRepository) List(ctx context.Context, page, pageSize int) ([]models.WechatWorkBot, int64, error) {
	var list []models.WechatWorkBot
	var total int64

	query := r.db.WithContext(ctx).Model(&models.WechatWorkBot{})
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).Order("created_at DESC").Find(&list).Error; err != nil {
		return nil, 0, err
	}

	return list, total, nil
}

func (r *WechatWorkBotRepository) Update(ctx context.Context, bot *models.WechatWorkBot) error {
	return r.db.WithContext(ctx).Model(bot).Where("id = ?", bot.ID).Updates(map[string]interface{}{
		"name":        bot.Name,
		"webhook_url": bot.WebhookURL,
		"description": bot.Description,
		"status":      bot.Status,
	}).Error
}

func (r *WechatWorkBotRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&models.WechatWorkBot{}, id).Error
}

// WechatWorkMessageLogRepository 企业微信消息日志仓储
type WechatWorkMessageLogRepository struct {
	db *gorm.DB
}

func NewWechatWorkMessageLogRepository(db *gorm.DB) *WechatWorkMessageLogRepository {
	return &WechatWorkMessageLogRepository{db: db}
}

func (r *WechatWorkMessageLogRepository) Create(ctx context.Context, log *models.WechatWorkMessageLog) error {
	return r.db.WithContext(ctx).Create(log).Error
}

func (r *WechatWorkMessageLogRepository) List(ctx context.Context, page, pageSize int, msgType, source string) ([]models.WechatWorkMessageLog, int64, error) {
	var list []models.WechatWorkMessageLog
	var total int64

	query := r.db.WithContext(ctx).Model(&models.WechatWorkMessageLog{})
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
