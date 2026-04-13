package repository

import (
	"context"

	"gorm.io/gorm"

	"devops/internal/models"
)

// ApprovalActionRepository 审批动作仓库
type ApprovalActionRepository struct {
	db *gorm.DB
}

// NewApprovalActionRepository 创建审批动作仓库
func NewApprovalActionRepository(db *gorm.DB) *ApprovalActionRepository {
	return &ApprovalActionRepository{db: db}
}

// Create 创建审批动作
func (r *ApprovalActionRepository) Create(ctx context.Context, action *models.ApprovalAction) error {
	return r.db.WithContext(ctx).Create(action).Error
}

// GetByID 根据ID获取审批动作
func (r *ApprovalActionRepository) GetByID(ctx context.Context, id uint) (*models.ApprovalAction, error) {
	var action models.ApprovalAction
	err := r.db.WithContext(ctx).First(&action, id).Error
	if err != nil {
		return nil, err
	}
	return &action, nil
}

// GetByNodeInstanceID 获取节点实例的所有审批动作
func (r *ApprovalActionRepository) GetByNodeInstanceID(ctx context.Context, nodeInstanceID uint) ([]models.ApprovalAction, error) {
	var actions []models.ApprovalAction
	err := r.db.WithContext(ctx).
		Where("node_instance_id = ?", nodeInstanceID).
		Order("created_at ASC").
		Find(&actions).Error
	return actions, err
}

// HasUserApproved 检查用户是否已对该节点进行过审批
func (r *ApprovalActionRepository) HasUserApproved(ctx context.Context, nodeInstanceID uint, userID uint) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&models.ApprovalAction{}).
		Where("node_instance_id = ? AND user_id = ? AND action IN (?, ?)", nodeInstanceID, userID, "approve", "reject").
		Count(&count).Error
	return count > 0, err
}

// GetUserAction 获取用户对该节点的审批动作
func (r *ApprovalActionRepository) GetUserAction(ctx context.Context, nodeInstanceID uint, userID uint) (*models.ApprovalAction, error) {
	var action models.ApprovalAction
	err := r.db.WithContext(ctx).
		Where("node_instance_id = ? AND user_id = ?", nodeInstanceID, userID).
		First(&action).Error
	if err != nil {
		return nil, err
	}
	return &action, nil
}

// CountByNodeInstanceAndAction 统计节点实例的某种动作数量
func (r *ApprovalActionRepository) CountByNodeInstanceAndAction(ctx context.Context, nodeInstanceID uint, action string) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&models.ApprovalAction{}).
		Where("node_instance_id = ? AND action = ?", nodeInstanceID, action).
		Count(&count).Error
	return count, err
}
