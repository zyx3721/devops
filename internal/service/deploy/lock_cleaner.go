package deploy

import (
	"context"
	"devops/pkg/logger"
	"time"
)

type LockCleaner struct {
	lockService *LockService
	stopCh      chan struct{}
}

func NewLockCleaner(lockService *LockService) *LockCleaner {
	return &LockCleaner{
		lockService: lockService,
		stopCh:      make(chan struct{}),
	}
}

// Start 启动锁清理器
func (c *LockCleaner) Start() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	logger.L().Info("发布锁清理器已启动")

	for {
		select {
		case <-ticker.C:
			c.clean(context.Background())
		case <-c.stopCh:
			logger.L().Info("发布锁清理器已停止")
			return
		}
	}
}

// Stop 停止锁清理器
func (c *LockCleaner) Stop() {
	close(c.stopCh)
}

// clean 清理过期锁
func (c *LockCleaner) clean(ctx context.Context) {
	if err := c.lockService.CleanExpired(ctx); err != nil {
		logger.L().Error("清理过期发布锁失败: %v", err)
	}
}
