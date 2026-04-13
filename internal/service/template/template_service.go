package template

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"

	"devops/internal/models"
)

var (
	ErrTemplateNotFound   = errors.New("模板不存在")
	ErrTemplateSlugExists = errors.New("模板标识已存在")
	ErrInvalidTemplateSlug = errors.New("无效的模板标识")
)

// PipelineTemplate 流水线模板
type PipelineTemplate struct {
	ID          uint                   `gorm:"primarykey" json:"id"`
	TenantID    *uint                  `gorm:"index" json:"tenant_id,omitempty"`
	Name        string                 `gorm:"size:100;not null" json:"name"`
	Slug        string                 `gorm:"size:100;not null" json:"slug"`
	Description string                 `gorm:"type:text" json:"description,omitempty"`
	Category    string                 `gorm:"size:50" json:"category,omitempty"`
	Tags        models.JSONMap         `gorm:"type:json" json:"tags,omitempty"`
	ConfigJSON  models.JSONMap         `gorm:"type:json;not null" json:"config_json"`
	IsPublic    bool                   `gorm:"default:false" json:"is_public"`
	IsOfficial  bool                   `gorm:"default:false" json:"is_official"`
	UsageCount  int                    `gorm:"default:0" json:"usage_count"`
	Rating      float64                `gorm:"type:decimal(2,1);default:0" json:"rating"`
	RatingCount int                    `gorm:"default:0" json:"rating_count"`
	Version     string                 `gorm:"size:20;default:'1.0.0'" json:"version"`
	CreatedBy   *uint                  `json:"created_by,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

func (PipelineTemplate) TableName() string { return "pipeline_templates" }

// TemplateService 模板服务接口
type TemplateService interface {
	// 模板 CRUD
	Create(ctx context.Context, tenantID *uint, req *CreateTemplateRequest) (*PipelineTemplate, error)
	Get(ctx context.Context, id uint) (*PipelineTemplate, error)
	GetBySlug(ctx context.Context, tenantID *uint, slug string) (*PipelineTemplate, error)
	Update(ctx context.Context, id uint, req *UpdateTemplateRequest) error
	Delete(ctx context.Context, id uint) error

	// 模板列表
	List(ctx context.Context, filter *TemplateFilter) ([]PipelineTemplate, int64, error)
	ListPublic(ctx context.Context, filter *TemplateFilter) ([]PipelineTemplate, int64, error)
	ListByTenant(ctx context.Context, tenantID uint, filter *TemplateFilter) ([]PipelineTemplate, int64, error)
	ListOfficial(ctx context.Context, filter *TemplateFilter) ([]PipelineTemplate, int64, error)

	// 模板使用
	UseTemplate(ctx context.Context, templateID uint) (*PipelineTemplate, error)
	RateTemplate(ctx context.Context, templateID uint, rating float64) error

	// 分类
	ListCategories(ctx context.Context) ([]string, error)
}

// CreateTemplateRequest 创建模板请求
type CreateTemplateRequest struct {
	Name        string                 `json:"name" binding:"required,min=2,max=100"`
	Slug        string                 `json:"slug" binding:"required,min=2,max=100"`
	Description string                 `json:"description,omitempty"`
	Category    string                 `json:"category,omitempty"`
	Tags        []string               `json:"tags,omitempty"`
	ConfigJSON  map[string]interface{} `json:"config_json" binding:"required"`
	IsPublic    bool                   `json:"is_public"`
	CreatedBy   uint                   `json:"-"`
}

// UpdateTemplateRequest 更新模板请求
type UpdateTemplateRequest struct {
	Name        string                 `json:"name,omitempty"`
	Description string                 `json:"description,omitempty"`
	Category    string                 `json:"category,omitempty"`
	Tags        []string               `json:"tags,omitempty"`
	ConfigJSON  map[string]interface{} `json:"config_json,omitempty"`
	IsPublic    *bool                  `json:"is_public,omitempty"`
	Version     string                 `json:"version,omitempty"`
}

// TemplateFilter 模板过滤条件
type TemplateFilter struct {
	Category string `json:"category,omitempty"`
	Keyword  string `json:"keyword,omitempty"`
	Page     int    `json:"page"`
	PageSize int    `json:"page_size"`
	OrderBy  string `json:"order_by,omitempty"` // usage_count, rating, created_at
}

// templateServiceImpl 模板服务实现
type templateServiceImpl struct {
	db *gorm.DB
}

// NewTemplateService 创建模板服务
func NewTemplateService(db *gorm.DB) TemplateService {
	return &templateServiceImpl{db: db}
}

func (s *templateServiceImpl) Create(ctx context.Context, tenantID *uint, req *CreateTemplateRequest) (*PipelineTemplate, error) {
	slug := strings.ToLower(strings.TrimSpace(req.Slug))

	// 检查 slug 是否已存在
	var count int64
	query := s.db.WithContext(ctx).Model(&PipelineTemplate{}).Where("slug = ?", slug)
	if tenantID != nil {
		query = query.Where("tenant_id = ? OR tenant_id IS NULL", *tenantID)
	} else {
		query = query.Where("tenant_id IS NULL")
	}
	if err := query.Count(&count).Error; err != nil {
		return nil, fmt.Errorf("检查模板标识失败: %w", err)
	}
	if count > 0 {
		return nil, ErrTemplateSlugExists
	}

	template := &PipelineTemplate{
		TenantID:    tenantID,
		Name:        strings.TrimSpace(req.Name),
		Slug:        slug,
		Description: req.Description,
		Category:    req.Category,
		Tags:        map[string]interface{}{"tags": req.Tags},
		ConfigJSON:  req.ConfigJSON,
		IsPublic:    req.IsPublic,
		IsOfficial:  tenantID == nil,
		Version:     "1.0.0",
		CreatedBy:   &req.CreatedBy,
	}

	if err := s.db.WithContext(ctx).Create(template).Error; err != nil {
		return nil, fmt.Errorf("创建模板失败: %w", err)
	}

	return template, nil
}

func (s *templateServiceImpl) Get(ctx context.Context, id uint) (*PipelineTemplate, error) {
	var template PipelineTemplate
	if err := s.db.WithContext(ctx).First(&template, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrTemplateNotFound
		}
		return nil, fmt.Errorf("查询模板失败: %w", err)
	}
	return &template, nil
}

func (s *templateServiceImpl) GetBySlug(ctx context.Context, tenantID *uint, slug string) (*PipelineTemplate, error) {
	var template PipelineTemplate
	query := s.db.WithContext(ctx).Where("slug = ?", slug)
	if tenantID != nil {
		query = query.Where("tenant_id = ? OR tenant_id IS NULL", *tenantID)
	}
	if err := query.First(&template).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrTemplateNotFound
		}
		return nil, fmt.Errorf("查询模板失败: %w", err)
	}
	return &template, nil
}

func (s *templateServiceImpl) Update(ctx context.Context, id uint, req *UpdateTemplateRequest) error {
	updates := make(map[string]interface{})
	if req.Name != "" {
		updates["name"] = strings.TrimSpace(req.Name)
	}
	if req.Description != "" {
		updates["description"] = req.Description
	}
	if req.Category != "" {
		updates["category"] = req.Category
	}
	if req.Tags != nil {
		updates["tags"] = map[string]interface{}{"tags": req.Tags}
	}
	if req.ConfigJSON != nil {
		updates["config_json"] = req.ConfigJSON
	}
	if req.IsPublic != nil {
		updates["is_public"] = *req.IsPublic
	}
	if req.Version != "" {
		updates["version"] = req.Version
	}

	if len(updates) == 0 {
		return nil
	}

	result := s.db.WithContext(ctx).Model(&PipelineTemplate{}).Where("id = ?", id).Updates(updates)
	if result.Error != nil {
		return fmt.Errorf("更新模板失败: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrTemplateNotFound
	}
	return nil
}

func (s *templateServiceImpl) Delete(ctx context.Context, id uint) error {
	result := s.db.WithContext(ctx).Delete(&PipelineTemplate{}, id)
	if result.Error != nil {
		return fmt.Errorf("删除模板失败: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrTemplateNotFound
	}
	return nil
}

func (s *templateServiceImpl) List(ctx context.Context, filter *TemplateFilter) ([]PipelineTemplate, int64, error) {
	return s.listWithQuery(ctx, s.db.WithContext(ctx).Model(&PipelineTemplate{}), filter)
}

func (s *templateServiceImpl) ListPublic(ctx context.Context, filter *TemplateFilter) ([]PipelineTemplate, int64, error) {
	query := s.db.WithContext(ctx).Model(&PipelineTemplate{}).Where("is_public = ?", true)
	return s.listWithQuery(ctx, query, filter)
}

func (s *templateServiceImpl) ListByTenant(ctx context.Context, tenantID uint, filter *TemplateFilter) ([]PipelineTemplate, int64, error) {
	query := s.db.WithContext(ctx).Model(&PipelineTemplate{}).
		Where("tenant_id = ? OR (is_public = ? AND tenant_id IS NULL)", tenantID, true)
	return s.listWithQuery(ctx, query, filter)
}

func (s *templateServiceImpl) ListOfficial(ctx context.Context, filter *TemplateFilter) ([]PipelineTemplate, int64, error) {
	query := s.db.WithContext(ctx).Model(&PipelineTemplate{}).Where("is_official = ?", true)
	return s.listWithQuery(ctx, query, filter)
}

func (s *templateServiceImpl) listWithQuery(ctx context.Context, query *gorm.DB, filter *TemplateFilter) ([]PipelineTemplate, int64, error) {
	if filter.Category != "" {
		query = query.Where("category = ?", filter.Category)
	}
	if filter.Keyword != "" {
		keyword := "%" + filter.Keyword + "%"
		query = query.Where("name LIKE ? OR description LIKE ?", keyword, keyword)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("统计模板数量失败: %w", err)
	}

	// 排序
	orderBy := "created_at DESC"
	switch filter.OrderBy {
	case "usage_count":
		orderBy = "usage_count DESC"
	case "rating":
		orderBy = "rating DESC"
	case "created_at":
		orderBy = "created_at DESC"
	}
	query = query.Order(orderBy)

	// 分页
	if filter.Page <= 0 {
		filter.Page = 1
	}
	if filter.PageSize <= 0 {
		filter.PageSize = 20
	}
	offset := (filter.Page - 1) * filter.PageSize
	query = query.Offset(offset).Limit(filter.PageSize)

	var templates []PipelineTemplate
	if err := query.Find(&templates).Error; err != nil {
		return nil, 0, fmt.Errorf("查询模板列表失败: %w", err)
	}

	return templates, total, nil
}

func (s *templateServiceImpl) UseTemplate(ctx context.Context, templateID uint) (*PipelineTemplate, error) {
	template, err := s.Get(ctx, templateID)
	if err != nil {
		return nil, err
	}

	// 增加使用次数
	s.db.WithContext(ctx).Model(&PipelineTemplate{}).
		Where("id = ?", templateID).
		UpdateColumn("usage_count", gorm.Expr("usage_count + 1"))

	return template, nil
}

func (s *templateServiceImpl) RateTemplate(ctx context.Context, templateID uint, rating float64) error {
	if rating < 0 || rating > 5 {
		return errors.New("评分必须在 0-5 之间")
	}

	template, err := s.Get(ctx, templateID)
	if err != nil {
		return err
	}

	// 计算新的平均评分
	newRatingCount := template.RatingCount + 1
	newRating := (template.Rating*float64(template.RatingCount) + rating) / float64(newRatingCount)

	return s.db.WithContext(ctx).Model(&PipelineTemplate{}).
		Where("id = ?", templateID).
		Updates(map[string]interface{}{
			"rating":       newRating,
			"rating_count": newRatingCount,
		}).Error
}

func (s *templateServiceImpl) ListCategories(ctx context.Context) ([]string, error) {
	var categories []string
	err := s.db.WithContext(ctx).Model(&PipelineTemplate{}).
		Distinct("category").
		Where("category IS NOT NULL AND category != ''").
		Pluck("category", &categories).Error
	if err != nil {
		return nil, fmt.Errorf("查询分类失败: %w", err)
	}
	return categories, nil
}
