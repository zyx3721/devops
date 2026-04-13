package deploy

import (
	"context"
	"devops/internal/models"
	"devops/pkg/logger"
	"errors"
	"time"

	"gorm.io/gorm"
)

var (
	ErrDeployLocked = errors.New("该应用环境正在发布中，请稍后重试")
	ErrNotLockOwner = errors.New("您不是锁的持有者")
)

type LockService struct {
	db *gorm.DB
}

func NewLockService(db *gorm.DB) *LockService {
	return &LockService{db: db}
}

// Acquire 获取发布锁
func (s *LockService) Acquire(ctx context.Context, appID uint, env string, recordID uint, userID uint, userName string) error {
	// 检查是否已有活跃的锁
	locked, existingLock, err := s.IsLocked(ctx, appID, env)
	if err != nil {
		return err
	}
	if locked {
		return errors.New("该应用环境正在发布中，锁定人: " + existingLock.LockedByName)
	}

	// 创建新锁，默认30分钟过期
	lock := &models.DeployLock{
		ApplicationID: appID,
		EnvName:       env,
		RecordID:      recordID,
		LockedBy:      userID,
		LockedByName:  userName,
		ExpiresAt:     time.Now().Add(30 * time.Minute),
		Status:        "active",
	}

	if err := s.db.Create(lock).Error; err != nil {
		return err
	}

	logger.L().Info("获取发布锁: app_id=%d, env=%s, user=%s", appID, env, userName)
	return nil
}

// Release 释放发布锁
func (s *LockService) Release(ctx context.Context, appID uint, env string) error {
	now := time.Now()
	result := s.db.Model(&models.DeployLock{}).
		Where("application_id = ? AND env_name = ? AND status = ?", appID, env, "active").
		Updates(map[string]any{
			"status":      "released",
			"released_at": now,
		})

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected > 0 {
		logger.L().Info("释放发布锁: app_id=%d, env=%s", appID, env)
	}
	return nil
}

// IsLocked 检查是否被锁定
func (s *LockService) IsLocked(ctx context.Context, appID uint, env string) (bool, *models.DeployLock, error) {
	var lock models.DeployLock
	err := s.db.Where("application_id = ? AND env_name = ? AND status = ? AND expires_at > ?",
		appID, env, "active", time.Now()).First(&lock).Error

	if err == gorm.ErrRecordNotFound {
		return false, nil, nil
	}
	if err != nil {
		return false, nil, err
	}
	return true, &lock, nil
}

// ForceRelease 强制释放锁（管理员）
func (s *LockService) ForceRelease(ctx context.Context, appID uint, env string, adminID uint, reason string) error {
	now := time.Now()
	result := s.db.Model(&models.DeployLock{}).
		Where("application_id = ? AND env_name = ? AND status = ?", appID, env, "active").
		Updates(map[string]any{
			"status":         "released",
			"released_at":    now,
			"released_by":    adminID,
			"release_reason": reason,
		})

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return ErrLockNotFound
	}

	logger.L().Info("强制释放发布锁: app_id=%d, env=%s, admin_id=%d, reason=%s", appID, env, adminID, reason)
	return nil
}

// GetActiveLocks 获取所有活跃的锁
func (s *LockService) GetActiveLocks(ctx context.Context) ([]models.DeployLock, error) {
	var locks []models.DeployLock
	err := s.db.Where("status = ? AND expires_at > ?", "active", time.Now()).
		Order("created_at DESC").
		Find(&locks).Error
	return locks, err
}

// CleanExpired 清理过期的锁
func (s *LockService) CleanExpired(ctx context.Context) error {
	now := time.Now()
	result := s.db.Model(&models.DeployLock{}).
		Where("status = ? AND expires_at < ?", "active", now).
		Updates(map[string]any{
			"status":      "expired",
			"released_at": now,
		})

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected > 0 {
		logger.L().Info("清理过期发布锁: count=%d", result.RowsAffected)
	}
	return nil
}
