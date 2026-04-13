package approval

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"

	"devops/internal/models"
	"devops/internal/repository"
)

var (
	ErrNodeInstanceNotFound = errors.New("节点实例不存在")
	ErrNodeNotActive        = errors.New("节点不是活跃状态")
	ErrAlreadyActioned      = errors.New("您已经对该节点进行过操作")
	ErrNodeAlreadyCompleted = errors.New("节点已完成，无法操作")
	ErrTransferToSelf       = errors.New("不能转交给自己")
)

// NodeExecutor 节点执行器
type NodeExecutor struct {
	nodeInstanceRepo *repository.ApprovalNodeInstanceRepository
	actionRepo       *repository.ApprovalActionRepository
	instanceRepo     *repository.ApprovalInstanceRepository
}

// NewNodeExecutor 创建节点执行器
func NewNodeExecutor(
	nodeInstanceRepo *repository.ApprovalNodeInstanceRepository,
	actionRepo *repository.ApprovalActionRepository,
	instanceRepo *repository.ApprovalInstanceRepository,
) *NodeExecutor {
	return &NodeExecutor{
		nodeInstanceRepo: nodeInstanceRepo,
		actionRepo:       actionRepo,
		instanceRepo:     instanceRepo,
	}
}

// Approve 审批通过
func (e *NodeExecutor) Approve(ctx context.Context, nodeInstanceID uint, userID uint, userName string, comment string) error {
	nodeInstance, err := e.nodeInstanceRepo.GetByID(ctx, nodeInstanceID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrNodeInstanceNotFound
		}
		return err
	}

	// 检查节点状态
	if nodeInstance.Status != "active" {
		if nodeInstance.Status == "approved" || nodeInstance.Status == "rejected" || nodeInstance.Status == "timeout" {
			return ErrNodeAlreadyCompleted
		}
		return ErrNodeNotActive
	}

	// 检查是否是审批人
	if !e.isApprover(nodeInstance.Approvers, userID) {
		return ErrNotApprover
	}

	// 检查是否已经操作过
	hasActioned, err := e.actionRepo.HasUserApproved(ctx, nodeInstanceID, userID)
	if err != nil {
		return err
	}
	if hasActioned {
		return ErrAlreadyActioned
	}

	// 创建审批动作
	action := &models.ApprovalAction{
		NodeInstanceID: nodeInstanceID,
		UserID:         userID,
		UserName:       userName,
		Action:         "approve",
		Comment:        comment,
	}
	if err := e.actionRepo.Create(ctx, action); err != nil {
		return err
	}

	// 增加已通过人数
	if err := e.nodeInstanceRepo.IncrementApproved(ctx, nodeInstanceID); err != nil {
		return err
	}

	// 重新获取节点实例以检查是否完成
	nodeInstance, err = e.nodeInstanceRepo.GetByID(ctx, nodeInstanceID)
	if err != nil {
		return err
	}

	// 检查节点是否完成
	completed, status := e.CheckComplete(nodeInstance)
	if completed {
		now := time.Now()
		if err := e.nodeInstanceRepo.UpdateStatus(ctx, nodeInstanceID, status, &now); err != nil {
			return err
		}

		// 如果节点通过，推进到下一个节点
		if status == "approved" {
			return e.advanceInstance(ctx, nodeInstance.InstanceID)
		}
	}

	return nil
}

// Reject 审批拒绝
func (e *NodeExecutor) Reject(ctx context.Context, nodeInstanceID uint, userID uint, userName string, reason string) error {
	nodeInstance, err := e.nodeInstanceRepo.GetByID(ctx, nodeInstanceID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrNodeInstanceNotFound
		}
		return err
	}

	// 检查节点状态
	if nodeInstance.Status != "active" {
		if nodeInstance.Status == "approved" || nodeInstance.Status == "rejected" || nodeInstance.Status == "timeout" {
			return ErrNodeAlreadyCompleted
		}
		return ErrNodeNotActive
	}

	// 检查是否是审批人
	if !e.isApprover(nodeInstance.Approvers, userID) {
		return ErrNotApprover
	}

	// 检查是否已经操作过
	hasActioned, err := e.actionRepo.HasUserApproved(ctx, nodeInstanceID, userID)
	if err != nil {
		return err
	}
	if hasActioned {
		return ErrAlreadyActioned
	}

	// 创建审批动作
	action := &models.ApprovalAction{
		NodeInstanceID: nodeInstanceID,
		UserID:         userID,
		UserName:       userName,
		Action:         "reject",
		Comment:        reason,
	}
	if err := e.actionRepo.Create(ctx, action); err != nil {
		return err
	}

	// 增加已拒绝人数
	if err := e.nodeInstanceRepo.IncrementRejected(ctx, nodeInstanceID); err != nil {
		return err
	}

	// 如果配置了任一人拒绝即拒绝，则直接拒绝节点
	if nodeInstance.RejectOnAny {
		now := time.Now()
		if err := e.nodeInstanceRepo.UpdateStatus(ctx, nodeInstanceID, "rejected", &now); err != nil {
			return err
		}

		// 拒绝整个审批实例
		return e.rejectInstance(ctx, nodeInstance.InstanceID)
	}

	// 重新获取节点实例以检查是否完成
	nodeInstance, err = e.nodeInstanceRepo.GetByID(ctx, nodeInstanceID)
	if err != nil {
		return err
	}

	// 检查节点是否完成
	completed, status := e.CheckComplete(nodeInstance)
	if completed {
		now := time.Now()
		if err := e.nodeInstanceRepo.UpdateStatus(ctx, nodeInstanceID, status, &now); err != nil {
			return err
		}

		if status == "rejected" {
			return e.rejectInstance(ctx, nodeInstance.InstanceID)
		}
	}

	return nil
}

// Transfer 转交审批
func (e *NodeExecutor) Transfer(ctx context.Context, nodeInstanceID uint, fromUserID uint, fromUserName string, toUserID uint, toUserName string, reason string) error {
	if fromUserID == toUserID {
		return ErrTransferToSelf
	}

	nodeInstance, err := e.nodeInstanceRepo.GetByID(ctx, nodeInstanceID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrNodeInstanceNotFound
		}
		return err
	}

	// 检查节点状态
	if nodeInstance.Status != "active" {
		if nodeInstance.Status == "approved" || nodeInstance.Status == "rejected" || nodeInstance.Status == "timeout" {
			return ErrNodeAlreadyCompleted
		}
		return ErrNodeNotActive
	}

	// 检查是否是审批人
	if !e.isApprover(nodeInstance.Approvers, fromUserID) {
		return ErrNotApprover
	}

	// 创建转交动作
	action := &models.ApprovalAction{
		NodeInstanceID: nodeInstanceID,
		UserID:         fromUserID,
		UserName:       fromUserName,
		Action:         "transfer",
		Comment:        reason,
		TransferTo:     &toUserID,
		TransferToName: toUserName,
	}
	if err := e.actionRepo.Create(ctx, action); err != nil {
		return err
	}

	// 更新审批人列表：移除原审批人，添加新审批人
	newApprovers := e.replaceApprover(nodeInstance.Approvers, fromUserID, toUserID)
	return e.nodeInstanceRepo.UpdateApprovers(ctx, nodeInstanceID, newApprovers)
}

// CheckComplete 检查节点是否完成
// 返回：是否完成，完成状态（approved/rejected）
func (e *NodeExecutor) CheckComplete(nodeInstance *models.ApprovalNodeInstance) (bool, string) {
	totalApprovers := len(strings.Split(nodeInstance.Approvers, ","))

	switch nodeInstance.ApproveMode {
	case "any":
		// 任一人通过即可
		if nodeInstance.ApprovedCount >= 1 {
			return true, "approved"
		}
	case "all":
		// 所有人都要通过
		if nodeInstance.ApprovedCount >= totalApprovers {
			return true, "approved"
		}
		// 如果有人拒绝且不是 reject_on_any，检查是否所有人都已操作
		if nodeInstance.ApprovedCount+nodeInstance.RejectedCount >= totalApprovers {
			if nodeInstance.RejectedCount > 0 {
				return true, "rejected"
			}
		}
	case "count":
		// 指定人数通过
		if nodeInstance.ApprovedCount >= nodeInstance.ApproveCount {
			return true, "approved"
		}
		// 检查是否已经不可能达到指定人数
		remaining := totalApprovers - nodeInstance.ApprovedCount - nodeInstance.RejectedCount
		if nodeInstance.ApprovedCount+remaining < nodeInstance.ApproveCount {
			return true, "rejected"
		}
	}

	return false, ""
}

// isApprover 检查用户是否是审批人
func (e *NodeExecutor) isApprover(approvers string, userID uint) bool {
	userIDStr := strconv.FormatUint(uint64(userID), 10)
	approverList := strings.Split(approvers, ",")
	for _, a := range approverList {
		if strings.TrimSpace(a) == userIDStr {
			return true
		}
	}
	return false
}

// replaceApprover 替换审批人
func (e *NodeExecutor) replaceApprover(approvers string, fromUserID, toUserID uint) string {
	fromStr := strconv.FormatUint(uint64(fromUserID), 10)
	toStr := strconv.FormatUint(uint64(toUserID), 10)

	approverList := strings.Split(approvers, ",")
	for i, a := range approverList {
		if strings.TrimSpace(a) == fromStr {
			approverList[i] = toStr
			break
		}
	}
	return strings.Join(approverList, ",")
}

// advanceInstance 推进审批实例到下一个节点
func (e *NodeExecutor) advanceInstance(ctx context.Context, instanceID uint) error {
	instance, err := e.instanceRepo.GetWithNodeInstances(ctx, instanceID)
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
		return e.instanceRepo.UpdateStatus(ctx, instanceID, "approved", &now)
	}

	// 更新当前节点顺序
	if err := e.instanceRepo.UpdateCurrentNode(ctx, instanceID, nextOrder); err != nil {
		return err
	}

	// 激活下一个节点（使用默认超时时间60分钟）
	timeoutAt := time.Now().Add(60 * time.Minute)
	return e.nodeInstanceRepo.Activate(ctx, nextNodeInstance.ID, &timeoutAt)
}

// rejectInstance 拒绝审批实例
func (e *NodeExecutor) rejectInstance(ctx context.Context, instanceID uint) error {
	now := time.Now()
	return e.instanceRepo.UpdateStatus(ctx, instanceID, "rejected", &now)
}

// GetNodeInstance 获取节点实例
func (e *NodeExecutor) GetNodeInstance(ctx context.Context, id uint) (*models.ApprovalNodeInstance, error) {
	nodeInstance, err := e.nodeInstanceRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNodeInstanceNotFound
		}
		return nil, err
	}
	return nodeInstance, nil
}

// GetNodeInstanceWithActions 获取节点实例及其审批动作
func (e *NodeExecutor) GetNodeInstanceWithActions(ctx context.Context, id uint) (*models.ApprovalNodeInstance, error) {
	nodeInstance, err := e.nodeInstanceRepo.GetWithActions(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNodeInstanceNotFound
		}
		return nil, err
	}
	return nodeInstance, nil
}
