package pipeline

import (
	"context"
	"time"

	"gorm.io/gorm"

	"devops/internal/models"
	"devops/pkg/dto"
)

// VariableService 变量服务
type VariableService struct {
	db *gorm.DB
}

// NewVariableService 创建变量服务
func NewVariableService(db *gorm.DB) *VariableService {
	return &VariableService{db: db}
}

// List 获取变量列表
func (s *VariableService) List(ctx context.Context, scope string, pipelineID uint) ([]dto.VariableItem, error) {
	var variables []models.PipelineVariable

	query := s.db.Model(&models.PipelineVariable{})
	if scope != "" {
		query = query.Where("scope = ?", scope)
	}
	if pipelineID > 0 {
		query = query.Where("pipeline_id = ? OR scope = 'global'", pipelineID)
	}

	if err := query.Order("scope, name").Find(&variables).Error; err != nil {
		return nil, err
	}

	items := make([]dto.VariableItem, 0, len(variables))
	for _, v := range variables {
		value := v.Value
		if v.IsSecret {
			value = "******"
		}

		items = append(items, dto.VariableItem{
			ID:         v.ID,
			Name:       v.Name,
			Value:      value,
			IsSecret:   v.IsSecret,
			Scope:      v.Scope,
			PipelineID: v.PipelineID,
			CreatedAt:  v.CreatedAt,
		})
	}

	return items, nil
}

// Create 创建变量
func (s *VariableService) Create(ctx context.Context, req *dto.VariableRequest) error {
	variable := &models.PipelineVariable{
		Name:       req.Name,
		Value:      req.Value,
		IsSecret:   req.IsSecret,
		Scope:      req.Scope,
		PipelineID: req.PipelineID,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	if variable.Scope == "" {
		variable.Scope = "global"
	}

	return s.db.Create(variable).Error
}

// Update 更新变量
func (s *VariableService) Update(ctx context.Context, req *dto.VariableRequest) error {
	var variable models.PipelineVariable
	if err := s.db.First(&variable, req.ID).Error; err != nil {
		return err
	}

	variable.Name = req.Name
	variable.Value = req.Value
	variable.IsSecret = req.IsSecret
	variable.Scope = req.Scope
	variable.PipelineID = req.PipelineID
	variable.UpdatedAt = time.Now()

	return s.db.Save(&variable).Error
}

// Delete 删除变量
func (s *VariableService) Delete(ctx context.Context, id uint) error {
	return s.db.Delete(&models.PipelineVariable{}, id).Error
}

// GetByPipeline 获取流水线的所有变量（包括全局）
func (s *VariableService) GetByPipeline(ctx context.Context, pipelineID uint) (map[string]string, error) {
	var variables []models.PipelineVariable

	// 获取全局变量和流水线变量
	if err := s.db.Where("scope = 'global' OR pipeline_id = ?", pipelineID).Find(&variables).Error; err != nil {
		return nil, err
	}

	result := make(map[string]string)
	for _, v := range variables {
		result[v.Name] = v.Value
	}

	return result, nil
}
