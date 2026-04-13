package security

import (
	"context"
	"time"

	"gorm.io/gorm"

	"devops/internal/models"
	"devops/pkg/dto"
	"devops/pkg/logger"
)

// RuleService 合规规则服务
type RuleService struct {
	db *gorm.DB
}

// NewRuleService 创建合规规则服务
func NewRuleService(db *gorm.DB) *RuleService {
	return &RuleService{db: db}
}

// List 获取规则列表
func (s *RuleService) List(ctx context.Context, category string, enabled *bool) ([]dto.ComplianceRuleItem, error) {
	var rules []models.ComplianceRule

	query := s.db.Model(&models.ComplianceRule{})
	if category != "" {
		query = query.Where("category = ?", category)
	}
	if enabled != nil {
		query = query.Where("enabled = ?", *enabled)
	}

	if err := query.Order("severity, name").Find(&rules).Error; err != nil {
		return nil, err
	}

	items := make([]dto.ComplianceRuleItem, 0, len(rules))
	for _, rule := range rules {
		items = append(items, dto.ComplianceRuleItem{
			ID:          rule.ID,
			Name:        rule.Name,
			Description: rule.Description,
			Severity:    rule.Severity,
			Category:    rule.Category,
			CheckType:   rule.CheckType,
			Enabled:     rule.Enabled,
			Remediation: rule.Remediation,
			CreatedAt:   rule.CreatedAt,
		})
	}

	return items, nil
}

// Get 获取规则详情
func (s *RuleService) Get(ctx context.Context, id uint) (*models.ComplianceRule, error) {
	var rule models.ComplianceRule
	if err := s.db.First(&rule, id).Error; err != nil {
		return nil, err
	}
	return &rule, nil
}

// Create 创建规则
func (s *RuleService) Create(ctx context.Context, req *dto.ComplianceRuleRequest) error {
	log := logger.L().WithField("name", req.Name)

	rule := &models.ComplianceRule{
		Name:          req.Name,
		Description:   req.Description,
		Severity:      req.Severity,
		Category:      req.Category,
		CheckType:     "custom",
		Enabled:       req.Enabled,
		ConditionJSON: req.ConditionJSON,
		Remediation:   req.Remediation,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	if err := s.db.Create(rule).Error; err != nil {
		log.WithField("error", err).Error("创建规则失败")
		return err
	}

	log.Info("创建规则成功")
	return nil
}

// Update 更新规则
func (s *RuleService) Update(ctx context.Context, req *dto.ComplianceRuleRequest) error {
	log := logger.L().WithField("id", req.ID)

	var rule models.ComplianceRule
	if err := s.db.First(&rule, req.ID).Error; err != nil {
		return err
	}

	// 内置规则只能修改启用状态
	if rule.CheckType == "builtin" {
		rule.Enabled = req.Enabled
	} else {
		rule.Name = req.Name
		rule.Description = req.Description
		rule.Severity = req.Severity
		rule.Category = req.Category
		rule.Enabled = req.Enabled
		rule.ConditionJSON = req.ConditionJSON
		rule.Remediation = req.Remediation
	}
	rule.UpdatedAt = time.Now()

	if err := s.db.Save(&rule).Error; err != nil {
		log.WithField("error", err).Error("更新规则失败")
		return err
	}

	log.Info("更新规则成功")
	return nil
}

// Delete 删除规则
func (s *RuleService) Delete(ctx context.Context, id uint) error {
	var rule models.ComplianceRule
	if err := s.db.First(&rule, id).Error; err != nil {
		return err
	}

	// 内置规则不能删除
	if rule.CheckType == "builtin" {
		return gorm.ErrRecordNotFound
	}

	return s.db.Delete(&rule).Error
}

// ToggleEnabled 切换启用状态
func (s *RuleService) ToggleEnabled(ctx context.Context, id uint) error {
	var rule models.ComplianceRule
	if err := s.db.First(&rule, id).Error; err != nil {
		return err
	}

	rule.Enabled = !rule.Enabled
	rule.UpdatedAt = time.Now()

	return s.db.Save(&rule).Error
}
