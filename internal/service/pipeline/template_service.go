package pipeline

import (
	"context"
	"encoding/json"
	"time"

	"gorm.io/gorm"

	"devops/internal/models"
	"devops/pkg/dto"
)

// TemplateService 模板服务
type TemplateService struct {
	db *gorm.DB
}

// NewTemplateService 创建模板服务
func NewTemplateService(db *gorm.DB) *TemplateService {
	return &TemplateService{db: db}
}

// List 获取模板列表
func (s *TemplateService) List(ctx context.Context, category string) ([]dto.PipelineTemplateItem, error) {
	var templates []models.PipelineTemplate

	query := s.db.Model(&models.PipelineTemplate{})
	if category != "" {
		query = query.Where("category = ?", category)
	}

	if err := query.Order("is_builtin DESC, name").Find(&templates).Error; err != nil {
		return nil, err
	}

	items := make([]dto.PipelineTemplateItem, 0, len(templates))
	for _, t := range templates {
		items = append(items, dto.PipelineTemplateItem{
			ID:          t.ID,
			Name:        t.Name,
			Description: t.Description,
			Category:    t.Category,
			IsBuiltin:   t.IsBuiltin,
			CreatedAt:   t.CreatedAt,
		})
	}

	return items, nil
}

// Get 获取模板详情
func (s *TemplateService) Get(ctx context.Context, id uint) (*dto.PipelineTemplateDetailResponse, error) {
	var template models.PipelineTemplate
	if err := s.db.First(&template, id).Error; err != nil {
		return nil, err
	}

	result := &dto.PipelineTemplateDetailResponse{
		ID:          template.ID,
		Name:        template.Name,
		Description: template.Description,
		Category:    template.Category,
		IsBuiltin:   template.IsBuiltin,
		CreatedAt:   template.CreatedAt,
	}

	// 解析配置
	if template.ConfigJSON != "" {
		var config struct {
			Stages []dto.Stage `json:"stages"`
		}
		if err := json.Unmarshal([]byte(template.ConfigJSON), &config); err == nil {
			result.Stages = config.Stages
		}
	}

	return result, nil
}

// CreateFromTemplate 从模板创建流水线
func (s *TemplateService) CreateFromTemplate(ctx context.Context, req *dto.CreateFromTemplateRequest, userID uint) error {
	// 获取模板
	var template models.PipelineTemplate
	if err := s.db.First(&template, req.TemplateID).Error; err != nil {
		return err
	}

	// 创建流水线
	pipeline := &models.Pipeline{
		Name:              req.Name,
		Description:       req.Description,
		ProjectID:         req.ProjectID,
		ConfigJSON:        template.ConfigJSON,
		TriggerConfigJSON: `{"manual":true}`,
		Status:            "active",
		CreatedBy:         &userID,
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}

	return s.db.Create(pipeline).Error
}
