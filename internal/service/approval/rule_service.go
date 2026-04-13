package approval

import (
	"devops/internal/models"
	"devops/internal/repository"
	"strconv"
	"strings"
)

type RuleService struct {
	repo *repository.ApprovalRuleRepository
}

func NewRuleService(repo *repository.ApprovalRuleRepository) *RuleService {
	return &RuleService{repo: repo}
}

// Create 创建审批规则
func (s *RuleService) Create(rule *models.ApprovalRule) error {
	return s.repo.Create(rule)
}

// Update 更新审批规则
func (s *RuleService) Update(rule *models.ApprovalRule) error {
	return s.repo.Update(rule)
}

// Delete 删除审批规则
func (s *RuleService) Delete(id uint) error {
	return s.repo.Delete(id)
}

// GetByID 根据ID获取规则
func (s *RuleService) GetByID(id uint) (*models.ApprovalRule, error) {
	return s.repo.GetByID(id)
}

// List 获取规则列表
func (s *RuleService) List(appID *uint) ([]models.ApprovalRule, error) {
	return s.repo.List(appID)
}

// NeedApproval 检查是否需要审批，返回是否需要审批和审批人ID列表
func (s *RuleService) NeedApproval(appID uint, env string) (bool, []uint, error) {
	rule, err := s.repo.GetByAppEnv(appID, env)
	if err != nil {
		// 没有找到规则，默认不需要审批
		return false, nil, nil
	}

	if !rule.NeedApproval || !rule.Enabled {
		return false, nil, nil
	}

	// 解析审批人ID列表
	approverIDs := parseApproverIDs(rule.Approvers)
	return true, approverIDs, nil
}

// GetTimeoutMinutes 获取审批超时时间
func (s *RuleService) GetTimeoutMinutes(appID uint, env string) int {
	rule, err := s.repo.GetByAppEnv(appID, env)
	if err != nil {
		return 30 // 默认30分钟
	}
	if rule.TimeoutMinutes <= 0 {
		return 30
	}
	return rule.TimeoutMinutes
}

// parseApproverIDs 解析审批人ID字符串
func parseApproverIDs(approvers string) []uint {
	if approvers == "" {
		return nil
	}
	parts := strings.Split(approvers, ",")
	var ids []uint
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		id, err := strconv.ParseUint(p, 10, 32)
		if err == nil {
			ids = append(ids, uint(id))
		}
	}
	return ids
}
