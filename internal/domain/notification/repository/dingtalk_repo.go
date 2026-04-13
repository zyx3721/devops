package repository

import (
	"context"

	"gorm.io/gorm"

	"devops/internal/domain/notification/model"
)

// DingtalkAppRepository 钉钉应用仓储
type DingtalkAppRepository struct {
	db *gorm.DB
}

func NewDingtalkAppRepository(db *gorm.DB) *DingtalkAppRepository {
	return &DingtalkAppRepository{db: db}
}

func (r *DingtalkAppRepository) Create(ctx context.Context, app *model.DingtalkApp) error {
	return r.db.WithContext(ctx).Create(app).Error
}

func (r *DingtalkAppRepository) GetByID(ctx context.Context, id uint) (*model.DingtalkApp, error) {
	var app model.DingtalkApp
	err := r.db.WithContext(ctx).First(&app, id).Error
	if err != nil {
		return nil, err
	}
	return &app, nil
}

func (r *DingtalkAppRepository) List(ctx context.Context, page, pageSize int) ([]model.DingtalkApp, int64, error) {
	var list []model.DingtalkApp
	var total int64

	query := r.db.WithContext(ctx).Model(&model.DingtalkApp{})
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).Order("created_at DESC").Find(&list).Error; err != nil {
		return nil, 0, err
	}

	return list, total, nil
}

func (r *DingtalkAppRepository) Update(ctx context.Context, app *model.DingtalkApp) error {
	return r.db.WithContext(ctx).Model(app).Where("id = ?", app.ID).Updates(map[string]any{
		"name":        app.Name,
		"app_key":     app.AppKey,
		"app_secret":  app.AppSecret,
		"agent_id":    app.AgentID,
		"project":     app.Project,
		"description": app.Description,
		"status":      app.Status,
		"is_default":  app.IsDefault,
	}).Error
}

func (r *DingtalkAppRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&model.DingtalkApp{}, id).Error
}

func (r *DingtalkAppRepository) GetDefault(ctx context.Context) (*model.DingtalkApp, error) {
	var app model.DingtalkApp
	err := r.db.WithContext(ctx).Where("is_default = ? AND status = ?", true, "active").First(&app).Error
	if err != nil {
		return nil, err
	}
	return &app, nil
}

func (r *DingtalkAppRepository) SetDefault(ctx context.Context, id uint) error {
	if err := r.db.WithContext(ctx).Model(&model.DingtalkApp{}).Where("is_default = ?", true).Update("is_default", false).Error; err != nil {
		return err
	}
	return r.db.WithContext(ctx).Model(&model.DingtalkApp{}).Where("id = ?", id).Update("is_default", true).Error
}

// DingtalkBotRepository 钉钉机器人仓储
type DingtalkBotRepository struct {
	db *gorm.DB
}

func NewDingtalkBotRepository(db *gorm.DB) *DingtalkBotRepository {
	return &DingtalkBotRepository{db: db}
}

func (r *DingtalkBotRepository) Create(ctx context.Context, bot *model.DingtalkBot) error {
	return r.db.WithContext(ctx).Create(bot).Error
}

func (r *DingtalkBotRepository) GetByID(ctx context.Context, id uint) (*model.DingtalkBot, error) {
	var bot model.DingtalkBot
	err := r.db.WithContext(ctx).First(&bot, id).Error
	if err != nil {
		return nil, err
	}
	return &bot, nil
}

func (r *DingtalkBotRepository) List(ctx context.Context, page, pageSize int) ([]model.DingtalkBot, int64, error) {
	var list []model.DingtalkBot
	var total int64

	query := r.db.WithContext(ctx).Model(&model.DingtalkBot{})
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).Order("created_at DESC").Find(&list).Error; err != nil {
		return nil, 0, err
	}

	return list, total, nil
}

func (r *DingtalkBotRepository) Update(ctx context.Context, bot *model.DingtalkBot) error {
	return r.db.WithContext(ctx).Model(bot).Where("id = ?", bot.ID).Updates(map[string]any{
		"name":        bot.Name,
		"webhook_url": bot.WebhookURL,
		"secret":      bot.Secret,
		"description": bot.Description,
		"status":      bot.Status,
	}).Error
}

func (r *DingtalkBotRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&model.DingtalkBot{}, id).Error
}

// DingtalkMessageLogRepository 钉钉消息日志仓储
type DingtalkMessageLogRepository struct {
	db *gorm.DB
}

func NewDingtalkMessageLogRepository(db *gorm.DB) *DingtalkMessageLogRepository {
	return &DingtalkMessageLogRepository{db: db}
}

func (r *DingtalkMessageLogRepository) Create(ctx context.Context, log *model.DingtalkMessageLog) error {
	return r.db.WithContext(ctx).Create(log).Error
}

func (r *DingtalkMessageLogRepository) List(ctx context.Context, page, pageSize int, msgType, source string) ([]model.DingtalkMessageLog, int64, error) {
	var list []model.DingtalkMessageLog
	var total int64

	query := r.db.WithContext(ctx).Model(&model.DingtalkMessageLog{})
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
