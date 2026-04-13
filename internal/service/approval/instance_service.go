package approval

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	"devops/internal/models"
	"devops/internal/repository"
)

var (
	ErrInstanceNotFound   = errors.New("审批实例不存在")
	ErrInstanceNotPending = errors.New("审批实例状态不是待处理")
	ErrNotRequester       = errors.New("只有申请人可以取消审批")
)

// InstanceService 审批实例服务
type InstanceService struct {
	instanceRepo     *repository.ApprovalInstanceRepository
	nodeInstanceRepo *repository.ApprovalNodeInstanceRepository
	chainService     *ChainService
	nodeExecutor     *NodeExecutor
	approverResolver *ApproverResolver
	deployTrigger    DeployTrigger
}

// NewInstanceService 创建审批实例服务
func NewInstanceService(
	instanceRepo *repository.ApprovalInstanceRepository,
	nodeInstanceRepo *repository.ApprovalNodeInstanceRepository,
	chainService *ChainService,
	nodeExecutor *NodeExecutor,
	approverResolver *ApproverResolver,
) *InstanceService {
	return &InstanceService{
		instanceRepo:     instanceRepo,
		nodeInstanceRepo: nodeInstanceRepo,
		chainService:     chainService,
		nodeExecutor:     nodeExecutor,
		approverResolver: approverResolver,
	}
}

// SetDeployTrigger 设置部署触发器（用于依赖注入）
func (s *InstanceService) SetDeployTrigger(trigger DeployTrigger) {
	s.deployTrigger = trigger
}

// Create 创建审批实例
func (s *InstanceService) Create(ctx context.Context, recordID uint, chain *models.ApprovalChain) (*models.ApprovalInstance, error) {
	return s.CreateWithAppID(ctx, recordID, chain, chain.AppID)
}

// CreateWithAppID 创建审批实例（指定应用ID用于解析审批人）
func (s *InstanceService) CreateWithAppID(ctx context.Context, recordID uint, chain *models.ApprovalChain, appID uint) (*models.ApprovalInstance, error) {
	if len(chain.Nodes) == 0 {
		return nil, ErrNoNodesInChain
	}

	now := time.Now()

	// 创建审批实例
	instance := &models.ApprovalInstance{
		RecordID:         recordID,
		ChainID:          chain.ID,
		ChainName:        chain.Name,
		Status:           "pending",
		CurrentNodeOrder: 1,
		StartedAt:        &now,
	}

	if err := s.instanceRepo.Create(ctx, instance); err != nil {
		return nil, err
	}

	// 创建节点实例，解析审批人
	nodeInstances := make([]models.ApprovalNodeInstance, 0, len(chain.Nodes))
	for _, node := range chain.Nodes {
		// 解析审批人
		resolvedApprovers := node.Approvers
		if s.approverResolver != nil {
			resolved, err := s.approverResolver.ResolveApprovers(ctx, node.ApproverType, node.Approvers, appID)
			if err == nil && resolved != "" {
				resolvedApprovers = resolved
			}
		}

		nodeInstance := models.ApprovalNodeInstance{
			InstanceID:    instance.ID,
			NodeID:        node.ID,
			NodeName:      node.Name,
			NodeOrder:     node.NodeOrder,
			ApproveMode:   node.ApproveMode,
			ApproveCount:  node.ApproveCount,
			ApproverType:  node.ApproverType,
			Approvers:     resolvedApprovers, // 使用解析后的审批人ID
			Status:        "pending",
			RejectOnAny:   node.RejectOnAny,
			TimeoutAction: node.TimeoutAction,
		}
		nodeInstances = append(nodeInstances, nodeInstance)
	}

	if err := s.nodeInstanceRepo.CreateBatch(ctx, nodeInstances); err != nil {
		return nil, err
	}

	// 激活第一个节点
	firstNodeInstance, err := s.nodeInstanceRepo.GetByInstanceIDAndOrder(ctx, instance.ID, 1)
	if err != nil {
		return nil, err
	}

	// 计算超时时间
	timeoutMinutes := chain.TimeoutMinutes
	if chain.Nodes[0].TimeoutMinutes > 0 {
		timeoutMinutes = chain.Nodes[0].TimeoutMinutes
	}
	timeoutAt := now.Add(time.Duration(timeoutMinutes) * time.Minute)

	if err := s.nodeInstanceRepo.Activate(ctx, firstNodeInstance.ID, &timeoutAt); err != nil {
		return nil, err
	}

	// 重新获取完整的实例信息
	return s.instanceRepo.GetWithNodeInstances(ctx, instance.ID)
}

// GetByID 根据ID获取审批实例
func (s *InstanceService) GetByID(ctx context.Context, id uint) (*models.ApprovalInstance, error) {
	instance, err := s.instanceRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrInstanceNotFound
		}
		return nil, err
	}
	return instance, nil
}

// GetWithNodeInstances 获取审批实例及其节点实例
func (s *InstanceService) GetWithNodeInstances(ctx context.Context, id uint) (*models.ApprovalInstance, error) {
	instance, err := s.instanceRepo.GetWithNodeInstances(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrInstanceNotFound
		}
		return nil, err
	}
	return instance, nil
}

// GetByRecordID 根据部署记录ID获取审批实例
func (s *InstanceService) GetByRecordID(ctx context.Context, recordID uint) (*models.ApprovalInstance, error) {
	instance, err := s.instanceRepo.GetByRecordID(ctx, recordID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return instance, nil
}

// List 获取审批实例列表
func (s *InstanceService) List(ctx context.Context, page, pageSize int, status string) ([]models.ApprovalInstance, int64, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	filter := repository.InstanceFilter{
		Status: status,
	}
	return s.instanceRepo.List(ctx, filter, page, pageSize)
}

// Cancel 取消审批实例
func (s *InstanceService) Cancel(ctx context.Context, instanceID uint, reason string) error {
	instance, err := s.instanceRepo.GetByID(ctx, instanceID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrInstanceNotFound
		}
		return err
	}

	if instance.Status != "pending" {
		return ErrInstanceNotPending
	}

	// 取消审批实例
	if err := s.instanceRepo.Cancel(ctx, instanceID, reason); err != nil {
		return err
	}

	// 取消所有活跃的节点实例
	nodeInstances, err := s.nodeInstanceRepo.GetByInstanceID(ctx, instanceID)
	if err != nil {
		return err
	}

	now := time.Now()
	for _, ni := range nodeInstances {
		if ni.Status == "active" || ni.Status == "pending" {
			if err := s.nodeInstanceRepo.UpdateStatus(ctx, ni.ID, "cancelled", &now); err != nil {
				return err
			}
		}
	}

	return nil
}

// GetPendingList 获取某用户待审批的实例列表
func (s *InstanceService) GetPendingList(ctx context.Context, userID uint) ([]models.ApprovalInstance, error) {
	// 获取用户待审批的活跃节点实例
	nodeInstances, err := s.nodeInstanceRepo.GetActiveByApprover(ctx, userID)
	if err != nil {
		return nil, err
	}

	if len(nodeInstances) == 0 {
		return []models.ApprovalInstance{}, nil
	}

	// 获取对应的审批实例
	instanceIDs := make([]uint, 0, len(nodeInstances))
	for _, ni := range nodeInstances {
		instanceIDs = append(instanceIDs, ni.InstanceID)
	}

	// 去重
	uniqueIDs := make(map[uint]bool)
	for _, id := range instanceIDs {
		uniqueIDs[id] = true
	}

	instances := make([]models.ApprovalInstance, 0, len(uniqueIDs))
	for id := range uniqueIDs {
		instance, err := s.instanceRepo.GetWithNodeInstances(ctx, id)
		if err != nil {
			continue
		}
		instances = append(instances, *instance)
	}

	return instances, nil
}

// AdvanceToNextNode 推进到下一个节点
func (s *InstanceService) AdvanceToNextNode(ctx context.Context, instanceID uint) error {
	instance, err := s.instanceRepo.GetWithNodeInstances(ctx, instanceID)
	if err != nil {
		return err
	}

	// 找到下一个节点
	nextOrder := instance.CurrentNodeOrder + 1
	var nextNodeInstance *models.ApprovalNodeInstance
	for i := range instance.NodeInstances {
		if instance.NodeInstances[i].NodeOrder == nextOrder {
			nextNodeInstance = &instance.NodeInstances[i]
			break
		}
	}

	if nextNodeInstance == nil {
		// 没有下一个节点，审批完成
		now := time.Now()
		if err := s.instanceRepo.UpdateStatus(ctx, instanceID, "approved", &now); err != nil {
			return err
		}

		// 触发部署执行
		if s.deployTrigger != nil {
			go func() {
				triggerCtx := context.Background()
				if err := s.deployTrigger.TriggerDeployAfterApproval(triggerCtx, instance.RecordID); err != nil {
					// 记录错误但不影响审批流程
					fmt.Printf("[InstanceService] 触发部署失败: record_id=%d, err=%v\n", instance.RecordID, err)
				}
			}()
		}
		return nil
	}

	// 更新当前节点顺序
	if err := s.instanceRepo.UpdateCurrentNode(ctx, instanceID, nextOrder); err != nil {
		return err
	}

	// 获取审批链以计算超时时间
	chain, err := s.chainService.GetWithNodes(ctx, instance.ChainID)
	if err != nil {
		return err
	}

	// 计算超时时间
	timeoutMinutes := chain.TimeoutMinutes
	for _, node := range chain.Nodes {
		if node.NodeOrder == nextOrder && node.TimeoutMinutes > 0 {
			timeoutMinutes = node.TimeoutMinutes
			break
		}
	}
	timeoutAt := time.Now().Add(time.Duration(timeoutMinutes) * time.Minute)

	// 激活下一个节点
	return s.nodeInstanceRepo.Activate(ctx, nextNodeInstance.ID, &timeoutAt)
}

// RejectInstance 拒绝审批实例
func (s *InstanceService) RejectInstance(ctx context.Context, instanceID uint) error {
	now := time.Now()
	return s.instanceRepo.UpdateStatus(ctx, instanceID, "rejected", &now)
}

// GetStats 获取审批统计
func (s *InstanceService) GetStats(ctx context.Context) (*ApprovalStats, error) {
	filter := repository.InstanceFilter{}

	// 获取所有实例
	instances, total, err := s.instanceRepo.List(ctx, filter, 1, 10000)
	if err != nil {
		return nil, err
	}

	stats := &ApprovalStats{
		Total: total,
	}

	var totalDuration int64
	var completedCount int64

	for _, inst := range instances {
		switch inst.Status {
		case "approved":
			stats.Approved++
		case "rejected":
			stats.Rejected++
		case "cancelled":
			stats.Cancelled++
		case "pending":
			stats.Pending++
		}

		if inst.FinishedAt != nil && inst.StartedAt != nil {
			duration := inst.FinishedAt.Sub(*inst.StartedAt).Seconds()
			totalDuration += int64(duration)
			completedCount++
		}
	}

	if completedCount > 0 {
		stats.AvgDurationSeconds = totalDuration / completedCount
	}

	if stats.Total > 0 {
		stats.ApprovalRate = float64(stats.Approved) / float64(stats.Total) * 100
	}

	return stats, nil
}

// ApprovalStats 审批统计
type ApprovalStats struct {
	Total              int64   `json:"total"`
	Approved           int64   `json:"approved"`
	Rejected           int64   `json:"rejected"`
	Cancelled          int64   `json:"cancelled"`
	Pending            int64   `json:"pending"`
	ApprovalRate       float64 `json:"approval_rate"`
	AvgDurationSeconds int64   `json:"avg_duration_seconds"`
}

// FormatDuration 格式化平均时长
func (s *ApprovalStats) FormatDuration() string {
	if s.AvgDurationSeconds == 0 {
		return "-"
	}
	minutes := s.AvgDurationSeconds / 60
	if minutes < 60 {
		return fmt.Sprintf("%d分钟", minutes)
	}
	hours := minutes / 60
	mins := minutes % 60
	return fmt.Sprintf("%d小时%d分钟", hours, mins)
}
