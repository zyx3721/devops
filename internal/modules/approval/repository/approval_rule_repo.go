package repository

import (
	"devops/internal/models"

	"gorm.io/gorm"
)

type ApprovalRuleRepository struct {
	db *gorm.DB
}

func NewApprovalRuleRepository(db *gorm.DB) *ApprovalRuleRepository {
	return &ApprovalRuleRepository{db: db}
}

// Create 创建审批规则
func (r *ApprovalRuleRepository) Create(rule *models.ApprovalRule) error {
	return r.db.Create(rule).Error
}

// Update 更新审批规则
func (r *ApprovalRuleRepository) Update(rule *models.ApprovalRule) error {
	return r.db.Save(rule).Error
}

// Delete 删除审批规则
func (r *ApprovalRuleRepository) Delete(id uint) error {
	return r.db.Delete(&models.ApprovalRule{}, id).Error
}

// GetByID 根据ID获取规则
func (r *ApprovalRuleRepository) GetByID(id uint) (*models.ApprovalRule, error) {
	var rule models.ApprovalRule
	err := r.db.First(&rule, id).Error
	if err != nil {
		return nil, err
	}
	return &rule, nil
}

// List 获取规则列表
func (r *ApprovalRuleRepository) List(appID *uint) ([]models.ApprovalRule, error) {
	var rules []models.ApprovalRule
	query := r.db.Model(&models.ApprovalRule{})
	if appID != nil {
		query = query.Where("app_id = ? OR app_id = 0", *appID)
	}
	err := query.Order("app_id ASC, env ASC").Find(&rules).Error
	return rules, err
}

// GetByAppEnv 根据应用和环境获取规则（优先应用级别，其次全局）
func (r *ApprovalRuleRepository) GetByAppEnv(appID uint, env string) (*models.ApprovalRule, error) {
	var rule models.ApprovalRule
	// 先查应用级别的规则
	err := r.db.Where("app_id = ? AND env = ? AND enabled = ?", appID, env, true).First(&rule).Error
	if err == nil {
		return &rule, nil
	}
	// 再查全局规则
	err = r.db.Where("app_id = 0 AND (env = ? OR env = '*') AND enabled = ?", env, true).First(&rule).Error
	if err != nil {
		return nil, err
	}
	return &rule, nil
}

// GetGlobalRules 获取全局规则
func (r *ApprovalRuleRepository) GetGlobalRules() ([]models.ApprovalRule, error) {
	var rules []models.ApprovalRule
	err := r.db.Where("app_id = 0").Find(&rules).Error
	return rules, err
}
