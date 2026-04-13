package repository

import (
	"context"

	"gorm.io/gorm"

	"devops/internal/models/system"
)

// MessageTemplateRepository 消息模板仓储
type MessageTemplateRepository struct {
	db *gorm.DB
}

func NewMessageTemplateRepository(db *gorm.DB) *MessageTemplateRepository {
	return &MessageTemplateRepository{db: db}
}

func (r *MessageTemplateRepository) Create(ctx context.Context, template *system.MessageTemplate) error {
	return r.db.WithContext(ctx).Create(template).Error
}

func (r *MessageTemplateRepository) GetByID(ctx context.Context, id uint) (*system.MessageTemplate, error) {
	var template system.MessageTemplate
	err := r.db.WithContext(ctx).First(&template, id).Error
	if err != nil {
		return nil, err
	}
	return &template, nil
}

func (r *MessageTemplateRepository) GetByName(ctx context.Context, name string) (*system.MessageTemplate, error) {
	var template system.MessageTemplate
	err := r.db.WithContext(ctx).Where("name = ?", name).First(&template).Error
	if err != nil {
		return nil, err
	}
	return &template, nil
}

func (r *MessageTemplateRepository) List(ctx context.Context, page, pageSize int, keyword string) ([]system.MessageTemplate, int64, error) {
	var list []system.MessageTemplate
	var total int64

	query := r.db.WithContext(ctx).Model(&system.MessageTemplate{})
	if keyword != "" {
		query = query.Where("name LIKE ? OR description LIKE ?", "%"+keyword+"%", "%"+keyword+"%")
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

func (r *MessageTemplateRepository) Update(ctx context.Context, template *system.MessageTemplate) error {
	return r.db.WithContext(ctx).Model(template).Where("id = ?", template.ID).Updates(map[string]interface{}{
		"name":        template.Name,
		"type":        template.Type,
		"content":     template.Content,
		"description": template.Description,
		"is_active":   template.IsActive,
	}).Error
}

func (r *MessageTemplateRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&system.MessageTemplate{}, id).Error
}
