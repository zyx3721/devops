package repository

import (
	"context"

	"gorm.io/gorm"

	"devops/internal/models"
)

type K8sClusterRepository struct {
	db *gorm.DB
}

func NewK8sClusterRepository(db *gorm.DB) *K8sClusterRepository {
	return &K8sClusterRepository{db: db}
}

func (r *K8sClusterRepository) GetByID(ctx context.Context, id uint) (*models.K8sCluster, error) {
	var cluster models.K8sCluster
	if err := r.db.WithContext(ctx).First(&cluster, id).Error; err != nil {
		return nil, err
	}
	return &cluster, nil
}

func (r *K8sClusterRepository) List(ctx context.Context, page, pageSize int) ([]models.K8sCluster, int64, error) {
	var clusters []models.K8sCluster
	var total int64

	if err := r.db.WithContext(ctx).Model(&models.K8sCluster{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := r.db.WithContext(ctx).Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&clusters).Error; err != nil {
		return nil, 0, err
	}

	return clusters, total, nil
}

func (r *K8sClusterRepository) GetAll(ctx context.Context) ([]models.K8sCluster, error) {
	var clusters []models.K8sCluster
	if err := r.db.WithContext(ctx).Where("status = ?", "active").Find(&clusters).Error; err != nil {
		return nil, err
	}
	return clusters, nil
}
