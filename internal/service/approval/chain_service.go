package approval

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"devops/internal/models"
	"devops/internal/repository"
)

var (
	ErrChainNotFound     = errors.New("审批链不存在")
	ErrChainInUse        = errors.New("审批链有进行中的审批实例，无法删除")
	ErrChainNameRequired = errors.New("审批链名称不能为空")
	ErrNoNodesInChain    = errors.New("审批链至少需要一个节点")
)

// ChainService 审批链服务
type ChainService struct {
	chainRepo *repository.ApprovalChainRepository
	nodeRepo  *repository.ApprovalNodeRepository
}

// NewChainService 创建审批链服务
func NewChainService(
	chainRepo *repository.ApprovalChainRepository,
	nodeRepo *repository.ApprovalNodeRepository,
) *ChainService {
	return &ChainService{
		chainRepo: chainRepo,
		nodeRepo:  nodeRepo,
	}
}

// Create 创建审批链
func (s *ChainService) Create(ctx context.Context, chain *models.ApprovalChain) error {
	if chain.Name == "" {
		return ErrChainNameRequired
	}

	// 设置默认值
	if chain.Env == "" {
		chain.Env = "*"
	}
	if chain.TimeoutMinutes <= 0 {
		chain.TimeoutMinutes = 60
	}
	if chain.TimeoutAction == "" {
		chain.TimeoutAction = "auto_cancel"
	}

	return s.chainRepo.Create(ctx, chain)
}

// Update 更新审批链
func (s *ChainService) Update(ctx context.Context, chain *models.ApprovalChain) error {
	existing, err := s.chainRepo.GetByID(ctx, chain.ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrChainNotFound
		}
		return err
	}

	// 合并更新：只更新非零值字段
	if chain.Name != "" {
		existing.Name = chain.Name
	}
	if chain.Description != "" {
		existing.Description = chain.Description
	}
	if chain.AppID != 0 || chain.Name != "" { // 如果是完整更新，允许设置 AppID 为 0
		existing.AppID = chain.AppID
	}
	if chain.Env != "" {
		existing.Env = chain.Env
	}
	if chain.Priority != 0 {
		existing.Priority = chain.Priority
	}
	if chain.TimeoutMinutes != 0 {
		existing.TimeoutMinutes = chain.TimeoutMinutes
	}
	if chain.TimeoutAction != "" {
		existing.TimeoutAction = chain.TimeoutAction
	}
	// 布尔值需要特殊处理，因为 false 也是有效值
	existing.Enabled = chain.Enabled
	existing.AllowEmergency = chain.AllowEmergency

	return s.chainRepo.Update(ctx, existing)
}

// Delete 删除审批链
func (s *ChainService) Delete(ctx context.Context, id uint) error {
	// 检查是否存在
	_, err := s.chainRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrChainNotFound
		}
		return err
	}

	// 检查是否有进行中的审批实例
	hasActive, err := s.chainRepo.HasActiveInstances(ctx, id)
	if err != nil {
		return err
	}
	if hasActive {
		return ErrChainInUse
	}

	// 删除节点
	if err := s.nodeRepo.DeleteByChainID(ctx, id); err != nil {
		return err
	}

	// 删除审批链（软删除）
	return s.chainRepo.Delete(ctx, id)
}

// GetByID 根据ID获取审批链
func (s *ChainService) GetByID(ctx context.Context, id uint) (*models.ApprovalChain, error) {
	chain, err := s.chainRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrChainNotFound
		}
		return nil, err
	}
	return chain, nil
}

// GetWithNodes 获取审批链及其节点
func (s *ChainService) GetWithNodes(ctx context.Context, id uint) (*models.ApprovalChain, error) {
	chain, err := s.chainRepo.GetWithNodes(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrChainNotFound
		}
		return nil, err
	}
	return chain, nil
}

// List 获取审批链列表
func (s *ChainService) List(ctx context.Context, page, pageSize int, appID *uint, env string) ([]models.ApprovalChain, int64, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	filter := repository.ChainFilter{
		AppID: appID,
		Env:   env,
	}
	return s.chainRepo.List(ctx, filter, page, pageSize)
}

// Match 根据应用和环境匹配审批链
// 优先级：应用+环境 > 应用+* > 0+环境 > 0+*
func (s *ChainService) Match(ctx context.Context, appID uint, env string) (*models.ApprovalChain, error) {
	chain, err := s.chainRepo.Match(ctx, appID, env)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil // 没有匹配的审批链
		}
		return nil, err
	}
	return chain, nil
}

// NeedApproval 检查是否需要审批
func (s *ChainService) NeedApproval(ctx context.Context, appID uint, env string) (bool, *models.ApprovalChain, error) {
	chain, err := s.Match(ctx, appID, env)
	if err != nil {
		return false, nil, err
	}
	if chain == nil {
		return false, nil, nil
	}
	if !chain.Enabled {
		return false, nil, nil
	}
	if len(chain.Nodes) == 0 {
		return false, nil, nil
	}
	return true, chain, nil
}

// AddNode 添加审批节点
func (s *ChainService) AddNode(ctx context.Context, node *models.ApprovalNode) error {
	// 检查审批链是否存在
	_, err := s.chainRepo.GetByID(ctx, node.ChainID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrChainNotFound
		}
		return err
	}

	// 获取最大顺序号
	maxOrder, err := s.nodeRepo.GetMaxOrder(ctx, node.ChainID)
	if err != nil {
		return err
	}

	// 如果没有指定顺序，则追加到最后
	if node.NodeOrder <= 0 {
		node.NodeOrder = maxOrder + 1
	}

	// 设置默认值
	if node.ApproveMode == "" {
		node.ApproveMode = "any"
	}
	if node.ApproveCount <= 0 {
		node.ApproveCount = 1
	}

	return s.nodeRepo.Create(ctx, node)
}

// UpdateNode 更新审批节点
func (s *ChainService) UpdateNode(ctx context.Context, node *models.ApprovalNode) error {
	existing, err := s.nodeRepo.GetByID(ctx, node.ID)
	if err != nil {
		return err
	}

	// 保留链ID
	node.ChainID = existing.ChainID
	node.CreatedAt = existing.CreatedAt

	return s.nodeRepo.Update(ctx, node)
}

// DeleteNode 删除审批节点
func (s *ChainService) DeleteNode(ctx context.Context, id uint) error {
	return s.nodeRepo.Delete(ctx, id)
}

// GetNodes 获取审批链的所有节点
func (s *ChainService) GetNodes(ctx context.Context, chainID uint) ([]models.ApprovalNode, error) {
	return s.nodeRepo.GetByChainID(ctx, chainID)
}

// ReorderNodes 重新排序节点
func (s *ChainService) ReorderNodes(ctx context.Context, chainID uint, nodeIDs []uint) error {
	nodeOrders := make([]struct {
		ID    uint
		Order int
	}, len(nodeIDs))
	for i, id := range nodeIDs {
		nodeOrders[i] = struct {
			ID    uint
			Order int
		}{ID: id, Order: i + 1}
	}
	return s.nodeRepo.ReorderNodes(ctx, chainID, nodeOrders)
}

// SetEnabled 设置审批链启用状态
func (s *ChainService) SetEnabled(ctx context.Context, id uint, enabled bool) error {
	chain, err := s.chainRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrChainNotFound
		}
		return err
	}

	chain.Enabled = enabled
	return s.chainRepo.Update(ctx, chain)
}
