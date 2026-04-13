package repository

import (
	"context"

	"gorm.io/gorm"

	"devops/internal/models"
)

// ApprovalChainRepository 审批链仓库
type ApprovalChainRepository struct {
	db *gorm.DB
}

// NewApprovalChainRepository 创建审批链仓库
func NewApprovalChainRepository(db *gorm.DB) *ApprovalChainRepository {
	return &ApprovalChainRepository{db: db}
}

// Create 创建审批链
func (r *ApprovalChainRepository) Create(ctx context.Context, chain *models.ApprovalChain) error {
	return r.db.WithContext(ctx).Create(chain).Error
}

// Update 更新审批链
func (r *ApprovalChainRepository) Update(ctx context.Context, chain *models.ApprovalChain) error {
	return r.db.WithContext(ctx).Save(chain).Error
}

// Delete 删除审批链（软删除）
func (r *ApprovalChainRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&models.ApprovalChain{}, id).Error
}

// GetByID 根据ID获取审批链
func (r *ApprovalChainRepository) GetByID(ctx context.Context, id uint) (*models.ApprovalChain, error) {
	var chain models.ApprovalChain
	err := r.db.WithContext(ctx).First(&chain, id).Error
	if err != nil {
		return nil, err
	}
	return &chain, nil
}

// GetWithNodes 获取审批链及其节点
func (r *ApprovalChainRepository) GetWithNodes(ctx context.Context, id uint) (*models.ApprovalChain, error) {
	var chain models.ApprovalChain
	err := r.db.WithContext(ctx).
		Preload("Nodes", func(db *gorm.DB) *gorm.DB {
			return db.Order("node_order ASC")
		}).
		First(&chain, id).Error
	if err != nil {
		return nil, err
	}
	return &chain, nil
}

// ChainFilter 审批链筛选条件
type ChainFilter struct {
	AppID   *uint
	Env     string
	Enabled *bool
}

// List 获取审批链列表
func (r *ApprovalChainRepository) List(ctx context.Context, filter ChainFilter, page, pageSize int) ([]models.ApprovalChain, int64, error) {
	var chains []models.ApprovalChain
	var total int64

	query := r.db.WithContext(ctx).Model(&models.ApprovalChain{})

	if filter.AppID != nil {
		query = query.Where("app_id = ?", *filter.AppID)
	}
	if filter.Env != "" {
		query = query.Where("env = ?", filter.Env)
	}
	if filter.Enabled != nil {
		query = query.Where("enabled = ?", *filter.Enabled)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	err := query.
		Preload("Nodes", func(db *gorm.DB) *gorm.DB {
			return db.Order("node_order ASC")
		}).
		Order("priority DESC, created_at DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&chains).Error

	return chains, total, err
}

// Match 根据应用和环境匹配审批链
// 优先级：应用+环境 > 应用+* > 0+环境 > 0+*
func (r *ApprovalChainRepository) Match(ctx context.Context, appID uint, env string) (*models.ApprovalChain, error) {
	var chain models.ApprovalChain

	// 按优先级查找
	// 1. 精确匹配：应用ID + 环境
	err := r.db.WithContext(ctx).
		Where("app_id = ? AND env = ? AND enabled = ?", appID, env, true).
		Order("priority DESC").
		First(&chain).Error
	if err == nil {
		return r.GetWithNodes(ctx, chain.ID)
	}
	if err != gorm.ErrRecordNotFound {
		return nil, err
	}

	// 2. 应用ID + 所有环境
	err = r.db.WithContext(ctx).
		Where("app_id = ? AND env = ? AND enabled = ?", appID, "*", true).
		Order("priority DESC").
		First(&chain).Error
	if err == nil {
		return r.GetWithNodes(ctx, chain.ID)
	}
	if err != gorm.ErrRecordNotFound {
		return nil, err
	}

	// 3. 全局 + 环境
	err = r.db.WithContext(ctx).
		Where("app_id = ? AND env = ? AND enabled = ?", 0, env, true).
		Order("priority DESC").
		First(&chain).Error
	if err == nil {
		return r.GetWithNodes(ctx, chain.ID)
	}
	if err != gorm.ErrRecordNotFound {
		return nil, err
	}

	// 4. 全局 + 所有环境
	err = r.db.WithContext(ctx).
		Where("app_id = ? AND env = ? AND enabled = ?", 0, "*", true).
		Order("priority DESC").
		First(&chain).Error
	if err == nil {
		return r.GetWithNodes(ctx, chain.ID)
	}

	return nil, err
}

// HasActiveInstances 检查是否有进行中的审批实例
func (r *ApprovalChainRepository) HasActiveInstances(ctx context.Context, chainID uint) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&models.ApprovalInstance{}).
		Where("chain_id = ? AND status = ?", chainID, "pending").
		Count(&count).Error
	return count > 0, err
}
