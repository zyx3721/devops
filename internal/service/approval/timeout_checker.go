package approval

import (
	"context"
	"devops/internal/models"
	"devops/pkg/logger"
	"time"

	"gorm.io/gorm"
)

// TimeoutChecker 审批超时检查器
// 负责检查 DeployRecord 的审批超时，并同步更新相关的 ApprovalInstance
type TimeoutChecker struct {
	db          *gorm.DB
	ruleService *RuleService
	stopCh      chan struct{}
	// 记录已发送提醒的记录ID，避免重复发送
	reminderSent map[uint]bool
}

func NewTimeoutChecker(db *gorm.DB, ruleService *RuleService) *TimeoutChecker {
	return &TimeoutChecker{
		db:           db,
		ruleService:  ruleService,
		stopCh:       make(chan struct{}),
		reminderSent: make(map[uint]bool),
	}
}

// Start 启动超时检查器
func (c *TimeoutChecker) Start() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	logger.L().Info("审批超时检查器已启动")

	for {
		select {
		case <-ticker.C:
			c.checkTimeout(context.Background())
		case <-c.stopCh:
			logger.L().Info("审批超时检查器已停止")
			return
		}
	}
}

// Stop 停止超时检查器
func (c *TimeoutChecker) Stop() {
	close(c.stopCh)
}

// checkTimeout 检查超时的审批
func (c *TimeoutChecker) checkTimeout(ctx context.Context) {
	var records []models.DeployRecord

	// 查询所有 pending 状态的记录
	err := c.db.Where("status = ? AND need_approval = ?", "pending", true).Find(&records).Error
	if err != nil {
		logger.L().Error("查询待审批记录失败: %v", err)
		return
	}

	now := time.Now()
	for _, record := range records {
		// 获取超时时间
		timeoutMinutes := c.ruleService.GetTimeoutMinutes(record.ApplicationID, record.EnvName)
		deadline := record.CreatedAt.Add(time.Duration(timeoutMinutes) * time.Minute)

		if now.After(deadline) {
			// 已超时，自动取消
			c.cancelTimeout(ctx, &record)
			// 清理提醒记录
			delete(c.reminderSent, record.ID)
		} else if now.After(deadline.Add(-5 * time.Minute)) {
			// 剩余5分钟，发送提醒（只发送一次）
			if !c.reminderSent[record.ID] {
				c.sendTimeoutReminder(&record, deadline)
				c.reminderSent[record.ID] = true
			}
		}
	}

	// 清理已完成记录的提醒状态（防止内存泄漏）
	c.cleanupReminderCache(ctx)
}

// cancelTimeout 超时自动取消
// 同时更新 DeployRecord 和 ApprovalInstance 的状态
func (c *TimeoutChecker) cancelTimeout(ctx context.Context, record *models.DeployRecord) {
	now := time.Now()

	// 使用事务确保数据一致性
	err := c.db.Transaction(func(tx *gorm.DB) error {
		// 1. 更新 DeployRecord 状态
		updates := map[string]any{
			"status":        "cancelled",
			"reject_reason": "审批超时自动取消",
			"finished_at":   now,
		}
		if err := tx.Model(record).Updates(updates).Error; err != nil {
			return err
		}

		// 2. 如果有关联的审批实例，同步更新其状态
		if record.ApprovalChainID != nil {
			// 查找关联的审批实例
			var instance models.ApprovalInstance
			err := tx.Where("deploy_record_id = ? AND status = ?", record.ID, "pending").
				First(&instance).Error
			if err == nil {
				// 更新审批实例状态
				instanceUpdates := map[string]any{
					"status":       "cancelled",
					"completed_at": now,
				}
				if err := tx.Model(&instance).Updates(instanceUpdates).Error; err != nil {
					logger.L().Error("更新审批实例状态失败: instance_id=%d, error=%v", instance.ID, err)
					// 不返回错误，继续执行
				}

				// 更新所有 pending 状态的节点实例为 timeout
				if err := tx.Model(&models.ApprovalNodeInstance{}).
					Where("instance_id = ? AND status = ?", instance.ID, "pending").
					Updates(map[string]any{
						"status":       "timeout",
						"completed_at": now,
					}).Error; err != nil {
					logger.L().Error("更新节点实例状态失败: instance_id=%d, error=%v", instance.ID, err)
				}
			}
		}

		return nil
	})

	if err != nil {
		logger.L().Error("超时取消审批失败: record_id=%d, error=%v", record.ID, err)
		return
	}

	logger.L().Info("审批超时自动取消: record_id=%d, app=%s, env=%s", record.ID, record.AppName, record.EnvName)

	// TODO: 发送超时取消通知
}

// sendTimeoutReminder 发送超时提醒
func (c *TimeoutChecker) sendTimeoutReminder(record *models.DeployRecord, deadline time.Time) {
	remaining := time.Until(deadline)
	logger.L().Info("审批即将超时提醒: record_id=%d, app=%s, env=%s, remaining=%v",
		record.ID, record.AppName, record.EnvName, remaining)

	// TODO: 发送超时提醒通知
}

// cleanupReminderCache 清理已完成记录的提醒缓存
func (c *TimeoutChecker) cleanupReminderCache(ctx context.Context) {
	if len(c.reminderSent) < 100 {
		return // 缓存较小时不清理
	}

	// 获取所有已完成的记录ID
	var completedIDs []uint
	for recordID := range c.reminderSent {
		var count int64
		c.db.Model(&models.DeployRecord{}).
			Where("id = ? AND status NOT IN (?)", recordID, []string{"pending"}).
			Count(&count)
		if count > 0 {
			completedIDs = append(completedIDs, recordID)
		}
	}

	// 清理
	for _, id := range completedIDs {
		delete(c.reminderSent, id)
	}
}
