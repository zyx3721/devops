package repository

import (
	"context"

	"gorm.io/gorm"

	"devops/internal/models"
)

// ApprovalNodeRepository 审批节点仓库
type ApprovalNodeRepository struct {
	db *gorm.DB
}

// NewApprovalNodeRepository 创建审批节点仓库
func NewApprovalNodeRepository(db *gorm.DB) *ApprovalNodeRepository {
	return &ApprovalNodeRepository{db: db}
}

// Create 创建审批节点
func (r *ApprovalNodeRepository) Create(ctx context.Context, node *models.ApprovalNode) error {
	return r.db.WithContext(ctx).Create(node).Error
}

// Update 更新审批节点
func (r *ApprovalNodeRepository) Update(ctx context.Context, node *models.ApprovalNode) error {
	return r.db.WithContext(ctx).Save(node).Error
}

// Delete 删除审批节点
func (r *ApprovalNodeRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&models.ApprovalNode{}, id).Error
}

// GetByID 根据ID获取节点
func (r *ApprovalNodeRepository) GetByID(ctx context.Context, id uint) (*models.ApprovalNode, error) {
	var node models.ApprovalNode
	err := r.db.WithContext(ctx).First(&node, id).Error
	if err != nil {
		return nil, err
	}
	return &node, nil
}

// GetByChainID 获取审批链的所有节点
func (r *ApprovalNodeRepository) GetByChainID(ctx context.Context, chainID uint) ([]models.ApprovalNode, error) {
	var nodes []models.ApprovalNode
	err := r.db.WithContext(ctx).
		Where("chain_id = ?", chainID).
		Order("node_order ASC").
		Find(&nodes).Error
	return nodes, err
}

// GetMaxOrder 获取审批链中最大的节点顺序
func (r *ApprovalNodeRepository) GetMaxOrder(ctx context.Context, chainID uint) (int, error) {
	var maxOrder int
	err := r.db.WithContext(ctx).
		Model(&models.ApprovalNode{}).
		Where("chain_id = ?", chainID).
		Select("COALESCE(MAX(node_order), 0)").
		Scan(&maxOrder).Error
	return maxOrder, err
}

// ReorderNodes 重新排序节点
func (r *ApprovalNodeRepository) ReorderNodes(ctx context.Context, chainID uint, nodeOrders []struct {
	ID    uint
	Order int
}) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, no := range nodeOrders {
			if err := tx.Model(&models.ApprovalNode{}).
				Where("id = ? AND chain_id = ?", no.ID, chainID).
				Update("node_order", no.Order).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// DeleteByChainID 删除审批链的所有节点
func (r *ApprovalNodeRepository) DeleteByChainID(ctx context.Context, chainID uint) error {
	return r.db.WithContext(ctx).
		Where("chain_id = ?", chainID).
		Delete(&models.ApprovalNode{}).Error
}
