package repository

import (
	"context"
	"time"

	"gorm.io/gorm"

	"devops/internal/models"
)

// DeployLockRepository 发布锁仓库
type DeployLockRepository struct {
	db *gorm.DB
}

func NewDeployLockRepository(db *gorm.DB) *DeployLockRepository {
	return &DeployLockRepository{db: db}
}

// AcquireLock 获取锁（如果已存在活跃锁则返回错误）
func (r *DeployLockRepository) AcquireLock(ctx context.Context, lock *models.DeployLock) error {
	// 先检查是否存在活跃锁
	var count int64
	err := r.db.WithContext(ctx).Model(&models.DeployLock{}).
		Where("application_id = ? AND env_name = ? AND status = ?", lock.ApplicationID, lock.EnvName, "active").
		Count(&count).Error
	if err != nil {
		return err
	}
	if count > 0 {
		return gorm.ErrDuplicatedKey
	}

	return r.db.WithContext(ctx).Create(lock).Error
}

// ReleaseLock 释放锁
func (r *DeployLockRepository) ReleaseLock(ctx context.Context, appID uint, envName string, releasedBy uint, reason string) error {
	now := time.Now()
	return r.db.WithContext(ctx).Model(&models.DeployLock{}).
		Where("application_id = ? AND env_name = ? AND status = ?", appID, envName, "active").
		Updates(map[string]interface{}{
			"status":         "released",
			"released_at":    &now,
			"released_by":    &releasedBy,
			"release_reason": reason,
		}).Error
}

// GetActiveLock 获取活跃锁
func (r *DeployLockRepository) GetActiveLock(ctx context.Context, appID uint, envName string) (*models.DeployLock, error) {
	var lock models.DeployLock
	err := r.db.WithContext(ctx).
		Where("application_id = ? AND env_name = ? AND status = ?", appID, envName, "active").
		First(&lock).Error
	if err != nil {
		return nil, err
	}
	return &lock, nil
}

// ExpireLocks 过期超时的锁
func (r *DeployLockRepository) ExpireLocks(ctx context.Context) (int64, error) {
	result := r.db.WithContext(ctx).Model(&models.DeployLock{}).
		Where("status = ? AND expires_at < ?", "active", time.Now()).
		Updates(map[string]interface{}{
			"status":         "expired",
			"release_reason": "自动过期",
		})
	return result.RowsAffected, result.Error
}

// GetByRecordID 根据记录ID获取锁
func (r *DeployLockRepository) GetByRecordID(ctx context.Context, recordID uint) (*models.DeployLock, error) {
	var lock models.DeployLock
	err := r.db.WithContext(ctx).Where("record_id = ?", recordID).First(&lock).Error
	if err != nil {
		return nil, err
	}
	return &lock, nil
}

// ApprovalRecordRepository 审批记录仓库
type ApprovalRecordRepository struct {
	db *gorm.DB
}

func NewApprovalRecordRepository(db *gorm.DB) *ApprovalRecordRepository {
	return &ApprovalRecordRepository{db: db}
}

func (r *ApprovalRecordRepository) Create(ctx context.Context, record *models.ApprovalRecord) error {
	return r.db.WithContext(ctx).Create(record).Error
}

func (r *ApprovalRecordRepository) GetByRecordID(ctx context.Context, recordID uint) ([]models.ApprovalRecord, error) {
	var records []models.ApprovalRecord
	err := r.db.WithContext(ctx).Where("record_id = ?", recordID).Order("created_at DESC").Find(&records).Error
	return records, err
}
