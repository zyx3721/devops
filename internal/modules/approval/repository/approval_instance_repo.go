package repository

import (
	"context"
	"time"

	"gorm.io/gorm"

	"devops/internal/models"
)

// ApprovalInstanceRepository 审批实例仓库
type ApprovalInstanceRepository struct {
	db *gorm.DB
}

// NewApprovalInstanceRepository 创建审批实例仓库
func NewApprovalInstanceRepository(db *gorm.DB) *ApprovalInstanceRepository {
	return &ApprovalInstanceRepository{db: db}
}

// Create 创建审批实例
func (r *ApprovalInstanceRepository) Create(ctx context.Context, instance *models.ApprovalInstance) error {
	return r.db.WithContext(ctx).Create(instance).Error
}

// Update 更新审批实例
func (r *ApprovalInstanceRepository) Update(ctx context.Context, instance *models.ApprovalInstance) error {
	return r.db.WithContext(ctx).Save(instance).Error
}

// GetByID 根据ID获取审批实例
func (r *ApprovalInstanceRepository) GetByID(ctx context.Context, id uint) (*models.ApprovalInstance, error) {
	var instance models.ApprovalInstance
	err := r.db.WithContext(ctx).First(&instance, id).Error
	if err != nil {
		return nil, err
	}
	return &instance, nil
}

// GetWithNodeInstances 获取审批实例及其节点实例
func (r *ApprovalInstanceRepository) GetWithNodeInstances(ctx context.Context, id uint) (*models.ApprovalInstance, error) {
	var instance models.ApprovalInstance
	err := r.db.WithContext(ctx).
		Preload("NodeInstances", func(db *gorm.DB) *gorm.DB {
			return db.Order("node_order ASC")
		}).
		Preload("NodeInstances.Actions", func(db *gorm.DB) *gorm.DB {
			return db.Order("created_at ASC")
		}).
		First(&instance, id).Error
	if err != nil {
		return nil, err
	}
	return &instance, nil
}

// GetByRecordID 根据部署记录ID获取审批实例
func (r *ApprovalInstanceRepository) GetByRecordID(ctx context.Context, recordID uint) (*models.ApprovalInstance, error) {
	var instance models.ApprovalInstance
	err := r.db.WithContext(ctx).
		Where("record_id = ?", recordID).
		First(&instance).Error
	if err != nil {
		return nil, err
	}
	return &instance, nil
}

// InstanceFilter 审批实例筛选条件
type InstanceFilter struct {
	ChainID   *uint
	Status    string
	StartTime *time.Time
	EndTime   *time.Time
}

// List 获取审批实例列表
func (r *ApprovalInstanceRepository) List(ctx context.Context, filter InstanceFilter, page, pageSize int) ([]models.ApprovalInstance, int64, error) {
	var instances []models.ApprovalInstance
	var total int64

	query := r.db.WithContext(ctx).Model(&models.ApprovalInstance{})

	if filter.ChainID != nil {
		query = query.Where("chain_id = ?", *filter.ChainID)
	}
	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}
	if filter.StartTime != nil {
		query = query.Where("created_at >= ?", *filter.StartTime)
	}
	if filter.EndTime != nil {
		query = query.Where("created_at <= ?", *filter.EndTime)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	err := query.
		Preload("NodeInstances", func(db *gorm.DB) *gorm.DB {
			return db.Order("node_order ASC")
		}).
		Order("created_at DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&instances).Error

	return instances, total, err
}

// UpdateStatus 更新审批实例状态
func (r *ApprovalInstanceRepository) UpdateStatus(ctx context.Context, id uint, status string, finishedAt *time.Time) error {
	updates := map[string]interface{}{
		"status": status,
	}
	if finishedAt != nil {
		updates["finished_at"] = finishedAt
	}
	return r.db.WithContext(ctx).
		Model(&models.ApprovalInstance{}).
		Where("id = ?", id).
		Updates(updates).Error
}

// UpdateCurrentNode 更新当前节点顺序
func (r *ApprovalInstanceRepository) UpdateCurrentNode(ctx context.Context, id uint, nodeOrder int) error {
	return r.db.WithContext(ctx).
		Model(&models.ApprovalInstance{}).
		Where("id = ?", id).
		Update("current_node_order", nodeOrder).Error
}

// GetPendingByApprover 获取某用户待审批的实例
func (r *ApprovalInstanceRepository) GetPendingByApprover(ctx context.Context, userID uint) ([]models.ApprovalInstance, error) {
	var instances []models.ApprovalInstance

	// 查找所有 active 状态的节点实例，且审批人包含该用户
	userIDStr := "%" + string(rune(userID)) + "%"
	
	err := r.db.WithContext(ctx).
		Joins("JOIN approval_node_instances ON approval_node_instances.instance_id = approval_instances.id").
		Where("approval_node_instances.status = ?", "active").
		Where("approval_node_instances.approvers LIKE ?", userIDStr).
		Preload("NodeInstances", func(db *gorm.DB) *gorm.DB {
			return db.Where("status = ?", "active").Order("node_order ASC")
		}).
		Find(&instances).Error

	return instances, err
}

// Cancel 取消审批实例
func (r *ApprovalInstanceRepository) Cancel(ctx context.Context, id uint, reason string) error {
	now := time.Now()
	return r.db.WithContext(ctx).
		Model(&models.ApprovalInstance{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":        "cancelled",
			"cancel_reason": reason,
			"finished_at":   &now,
		}).Error
}
