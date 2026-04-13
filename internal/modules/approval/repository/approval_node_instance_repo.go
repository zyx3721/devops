package repository

import (
	"context"
	"time"

	"gorm.io/gorm"

	"devops/internal/models"
)

// ApprovalNodeInstanceRepository 节点实例仓库
type ApprovalNodeInstanceRepository struct {
	db *gorm.DB
}

// NewApprovalNodeInstanceRepository 创建节点实例仓库
func NewApprovalNodeInstanceRepository(db *gorm.DB) *ApprovalNodeInstanceRepository {
	return &ApprovalNodeInstanceRepository{db: db}
}

// Create 创建节点实例
func (r *ApprovalNodeInstanceRepository) Create(ctx context.Context, nodeInstance *models.ApprovalNodeInstance) error {
	return r.db.WithContext(ctx).Create(nodeInstance).Error
}

// CreateBatch 批量创建节点实例
func (r *ApprovalNodeInstanceRepository) CreateBatch(ctx context.Context, nodeInstances []models.ApprovalNodeInstance) error {
	return r.db.WithContext(ctx).Create(&nodeInstances).Error
}

// Update 更新节点实例
func (r *ApprovalNodeInstanceRepository) Update(ctx context.Context, nodeInstance *models.ApprovalNodeInstance) error {
	return r.db.WithContext(ctx).Save(nodeInstance).Error
}

// GetByID 根据ID获取节点实例
func (r *ApprovalNodeInstanceRepository) GetByID(ctx context.Context, id uint) (*models.ApprovalNodeInstance, error) {
	var nodeInstance models.ApprovalNodeInstance
	err := r.db.WithContext(ctx).First(&nodeInstance, id).Error
	if err != nil {
		return nil, err
	}
	return &nodeInstance, nil
}

// GetWithActions 获取节点实例及其审批动作
func (r *ApprovalNodeInstanceRepository) GetWithActions(ctx context.Context, id uint) (*models.ApprovalNodeInstance, error) {
	var nodeInstance models.ApprovalNodeInstance
	err := r.db.WithContext(ctx).
		Preload("Actions", func(db *gorm.DB) *gorm.DB {
			return db.Order("created_at ASC")
		}).
		First(&nodeInstance, id).Error
	if err != nil {
		return nil, err
	}
	return &nodeInstance, nil
}

// GetByInstanceID 获取审批实例的所有节点实例
func (r *ApprovalNodeInstanceRepository) GetByInstanceID(ctx context.Context, instanceID uint) ([]models.ApprovalNodeInstance, error) {
	var nodeInstances []models.ApprovalNodeInstance
	err := r.db.WithContext(ctx).
		Where("instance_id = ?", instanceID).
		Order("node_order ASC").
		Find(&nodeInstances).Error
	return nodeInstances, err
}

// GetByInstanceIDAndOrder 根据实例ID和节点顺序获取节点实例
func (r *ApprovalNodeInstanceRepository) GetByInstanceIDAndOrder(ctx context.Context, instanceID uint, nodeOrder int) (*models.ApprovalNodeInstance, error) {
	var nodeInstance models.ApprovalNodeInstance
	err := r.db.WithContext(ctx).
		Where("instance_id = ? AND node_order = ?", instanceID, nodeOrder).
		First(&nodeInstance).Error
	if err != nil {
		return nil, err
	}
	return &nodeInstance, nil
}

// UpdateStatus 更新节点实例状态
func (r *ApprovalNodeInstanceRepository) UpdateStatus(ctx context.Context, id uint, status string, finishedAt *time.Time) error {
	updates := map[string]interface{}{
		"status": status,
	}
	if finishedAt != nil {
		updates["finished_at"] = finishedAt
	}
	return r.db.WithContext(ctx).
		Model(&models.ApprovalNodeInstance{}).
		Where("id = ?", id).
		Updates(updates).Error
}

// Activate 激活节点实例
func (r *ApprovalNodeInstanceRepository) Activate(ctx context.Context, id uint, timeoutAt *time.Time) error {
	now := time.Now()
	updates := map[string]interface{}{
		"status":       "active",
		"activated_at": &now,
	}
	if timeoutAt != nil {
		updates["timeout_at"] = timeoutAt
	}
	return r.db.WithContext(ctx).
		Model(&models.ApprovalNodeInstance{}).
		Where("id = ?", id).
		Updates(updates).Error
}

// IncrementApproved 增加已通过人数
func (r *ApprovalNodeInstanceRepository) IncrementApproved(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).
		Model(&models.ApprovalNodeInstance{}).
		Where("id = ?", id).
		UpdateColumn("approved_count", gorm.Expr("approved_count + 1")).Error
}

// IncrementRejected 增加已拒绝人数
func (r *ApprovalNodeInstanceRepository) IncrementRejected(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).
		Model(&models.ApprovalNodeInstance{}).
		Where("id = ?", id).
		UpdateColumn("rejected_count", gorm.Expr("rejected_count + 1")).Error
}

// GetTimeoutNodes 获取超时的节点实例
func (r *ApprovalNodeInstanceRepository) GetTimeoutNodes(ctx context.Context) ([]models.ApprovalNodeInstance, error) {
	var nodeInstances []models.ApprovalNodeInstance
	now := time.Now()
	err := r.db.WithContext(ctx).
		Where("status = ? AND timeout_at IS NOT NULL AND timeout_at <= ?", "active", now).
		Find(&nodeInstances).Error
	return nodeInstances, err
}

// GetNearTimeoutNodes 获取即将超时的节点实例（剩余10分钟）
func (r *ApprovalNodeInstanceRepository) GetNearTimeoutNodes(ctx context.Context, reminderMinutes int) ([]models.ApprovalNodeInstance, error) {
	var nodeInstances []models.ApprovalNodeInstance
	now := time.Now()
	reminderTime := now.Add(time.Duration(reminderMinutes) * time.Minute)
	err := r.db.WithContext(ctx).
		Where("status = ? AND timeout_at IS NOT NULL AND timeout_at > ? AND timeout_at <= ?", "active", now, reminderTime).
		Find(&nodeInstances).Error
	return nodeInstances, err
}

// UpdateApprovers 更新审批人列表（用于转交）
func (r *ApprovalNodeInstanceRepository) UpdateApprovers(ctx context.Context, id uint, approvers string) error {
	return r.db.WithContext(ctx).
		Model(&models.ApprovalNodeInstance{}).
		Where("id = ?", id).
		Update("approvers", approvers).Error
}

// GetActiveByApprover 获取某用户待审批的活跃节点实例
func (r *ApprovalNodeInstanceRepository) GetActiveByApprover(ctx context.Context, userID uint) ([]models.ApprovalNodeInstance, error) {
	var nodeInstances []models.ApprovalNodeInstance
	// 使用 FIND_IN_SET 查找审批人
	err := r.db.WithContext(ctx).
		Where("status = ? AND FIND_IN_SET(?, approvers) > 0", "active", userID).
		Order("created_at DESC").
		Find(&nodeInstances).Error
	return nodeInstances, err
}
