package repository

import (
	"context"

	"gorm.io/gorm"

	"devops/internal/models"
)

type JenkinsInstanceRepository struct {
	db *gorm.DB
}

func NewJenkinsInstanceRepository(db *gorm.DB) *JenkinsInstanceRepository {
	return &JenkinsInstanceRepository{db: db}
}

func (r *JenkinsInstanceRepository) GetByID(ctx context.Context, id uint) (*models.JenkinsInstance, error) {
	var instance models.JenkinsInstance
	if err := r.db.WithContext(ctx).First(&instance, id).Error; err != nil {
		return nil, err
	}
	return &instance, nil
}

func (r *JenkinsInstanceRepository) List(ctx context.Context, page, pageSize int) ([]models.JenkinsInstance, int64, error) {
	var instances []models.JenkinsInstance
	var total int64

	if err := r.db.WithContext(ctx).Model(&models.JenkinsInstance{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := r.db.WithContext(ctx).Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&instances).Error; err != nil {
		return nil, 0, err
	}

	return instances, total, nil
}

func (r *JenkinsInstanceRepository) GetAll(ctx context.Context) ([]models.JenkinsInstance, error) {
	var instances []models.JenkinsInstance
	if err := r.db.WithContext(ctx).Where("status = ?", "active").Find(&instances).Error; err != nil {
		return nil, err
	}
	return instances, nil
}
