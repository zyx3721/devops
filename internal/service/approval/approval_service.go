package approval

import (
	"context"
	"devops/internal/models"
	"devops/pkg/logger"
	"errors"
	"fmt"
	"slices"
	"time"

	"gorm.io/gorm"
)

var (
	ErrApprovalNotFound = errors.New("审批记录不存在")
	ErrAlreadyApproved  = errors.New("该记录已被审批")
	ErrNotApprover      = errors.New("您不是该节点的审批人，无权进行此操作")
	ErrApprovalTimeout  = errors.New("审批已超时")
	ErrRecordNotPending = errors.New("记录状态不是待审批")
)

// DeployTrigger 部署触发器接口
type DeployTrigger interface {
	TriggerDeployAfterApproval(ctx context.Context, recordID uint) error
}

type ApprovalService struct {
	db                  *gorm.DB
	ruleService         *RuleService
	deployTrigger       DeployTrigger
	notificationService *NotificationService
}

func NewApprovalService(db *gorm.DB, ruleService *RuleService, notificationService *NotificationService) *ApprovalService {
	return &ApprovalService{
		db:                  db,
		ruleService:         ruleService,
		notificationService: notificationService,
	}
}

// SetDeployTrigger 设置部署触发器（用于依赖注入）
func (s *ApprovalService) SetDeployTrigger(trigger DeployTrigger) {
	s.deployTrigger = trigger
}

// Submit 提交审批（创建部署记录时调用）
func (s *ApprovalService) Submit(ctx context.Context, record *models.DeployRecord) error {
	needApproval, approverIDs, err := s.ruleService.NeedApproval(record.ApplicationID, record.EnvName)
	if err != nil {
		return err
	}

	record.NeedApproval = needApproval
	if needApproval {
		record.Status = "pending"

		// 发送审批请求通知
		if s.notificationService != nil && len(approverIDs) > 0 {
			// 获取应用信息
			var app models.Application
			if err := s.db.First(&app, record.ApplicationID).Error; err != nil {
				logger.L().WithError(err).Error("获取应用信息失败: app_id=%d", record.ApplicationID)
			} else {
				// 构建通知请求
				req := &ApprovalNotifyRequest{
					Approvers:   convertApproverIDsToStrings(approverIDs),
					AppName:     app.Name,
					EnvName:     record.EnvName,
					Operator:    record.Operator,
					Description: record.Description,
				}

				// 异步发送通知
				go func() {
					notifyCtx := context.WithoutCancel(ctx)
					if err := s.notificationService.SendApprovalRequest(notifyCtx, req); err != nil {
						logger.L().WithError(err).Error("发送审批请求通知失败: record_id=%d", record.ID)
					} else {
						logger.L().Info("发送审批请求通知成功: record_id=%d, approvers=%v", record.ID, req.Approvers)
					}
				}()
			}
		}
	}

	return nil
}

// convertApproverIDsToStrings 将审批人ID列表转换为字符串列表
func convertApproverIDsToStrings(approverIDs []uint) []string {
	result := make([]string, len(approverIDs))
	for i, id := range approverIDs {
		result[i] = fmt.Sprintf("%d", id)
	}
	return result
}

// Approve 审批通过
func (s *ApprovalService) Approve(ctx context.Context, recordID uint, approverID uint, approverName string, comment string) error {
	var record models.DeployRecord

	err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.First(&record, recordID).Error; err != nil {
			return ErrApprovalNotFound
		}

		if record.Status != "pending" {
			return ErrRecordNotPending
		}

		// 更新部署记录状态
		now := time.Now()
		updates := map[string]any{
			"status":        "approved",
			"approver_id":   approverID,
			"approver_name": approverName,
			"approved_at":   now,
		}
		if err := tx.Model(&record).Updates(updates).Error; err != nil {
			return err
		}

		// 创建审批记录
		approvalRecord := &models.ApprovalRecord{
			RecordID:     recordID,
			ApproverID:   approverID,
			ApproverName: approverName,
			Action:       "approve",
			Comment:      comment,
		}
		if err := tx.Create(approvalRecord).Error; err != nil {
			return err
		}

		logger.L().Info("审批通过: record_id=%d, approver=%s", recordID, approverName)
		return nil
	})

	if err != nil {
		return err
	}

	// 发送审批结果通知给发起人
	if s.notificationService != nil {
		go func() {
			notifyCtx := context.WithoutCancel(ctx)
			// 获取应用信息
			var app models.Application
			s.db.First(&app, record.ApplicationID)

			// 构建审批实例（简化版，用于通知）
			instance := &models.ApprovalInstance{
				ID:        recordID,
				ChainName: fmt.Sprintf("%s-%s 发布审批", app.Name, record.EnvName),
			}

			if err := s.notificationService.SendApprovalResult(notifyCtx, instance, "approved", approverName); err != nil {
				logger.L().WithError(err).Error("发送审批结果通知失败: record_id=%d", recordID)
			} else {
				logger.L().Info("发送审批通过通知成功: record_id=%d, requester=%s", recordID, record.Operator)
			}
		}()
	}

	// 审批通过后自动触发部署（使用传入的 context 而非 Background）
	if s.deployTrigger != nil {
		// 创建一个新的 context，避免原 context 被取消影响异步操作
		triggerCtx := context.WithoutCancel(ctx)
		go func() {
			if triggerErr := s.deployTrigger.TriggerDeployAfterApproval(triggerCtx, recordID); triggerErr != nil {
				logger.L().WithError(triggerErr).Error("审批通过后触发部署失败: record_id=%d", recordID)
			}
		}()
	}

	return nil
}

// Reject 审批拒绝
func (s *ApprovalService) Reject(ctx context.Context, recordID uint, approverID uint, approverName string, reason string) error {
	var record models.DeployRecord

	err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.First(&record, recordID).Error; err != nil {
			return ErrApprovalNotFound
		}

		if record.Status != "pending" {
			return ErrRecordNotPending
		}

		// 更新部署记录状态
		now := time.Now()
		updates := map[string]any{
			"status":        "rejected",
			"approver_id":   approverID,
			"approver_name": approverName,
			"approved_at":   now,
			"reject_reason": reason,
		}
		if err := tx.Model(&record).Updates(updates).Error; err != nil {
			return err
		}

		// 创建审批记录
		approvalRecord := &models.ApprovalRecord{
			RecordID:     recordID,
			ApproverID:   approverID,
			ApproverName: approverName,
			Action:       "reject",
			Comment:      reason,
		}
		if err := tx.Create(approvalRecord).Error; err != nil {
			return err
		}

		logger.L().Info("审批拒绝: record_id=%d, approver=%s, reason=%s", recordID, approverName, reason)
		return nil
	})

	if err != nil {
		return err
	}

	// 发送审批结果通知给发起人
	if s.notificationService != nil {
		go func() {
			notifyCtx := context.WithoutCancel(ctx)
			// 获取应用信息
			var app models.Application
			s.db.First(&app, record.ApplicationID)

			// 构建审批实例（简化版，用于通知）
			instance := &models.ApprovalInstance{
				ID:        recordID,
				ChainName: fmt.Sprintf("%s-%s 发布审批", app.Name, record.EnvName),
			}

			if err := s.notificationService.SendApprovalResult(notifyCtx, instance, "rejected", approverName); err != nil {
				logger.L().WithError(err).Error("发送审批结果通知失败: record_id=%d", recordID)
			} else {
				logger.L().Info("发送审批拒绝通知成功: record_id=%d, requester=%s", recordID, record.Operator)
			}
		}()
	}

	return nil
}

// Cancel 取消审批
func (s *ApprovalService) Cancel(ctx context.Context, recordID uint, userID uint, userName string) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		var record models.DeployRecord
		if err := tx.First(&record, recordID).Error; err != nil {
			return ErrApprovalNotFound
		}

		if record.Status != "pending" {
			return ErrRecordNotPending
		}

		// 只有申请人可以取消
		if record.OperatorID != userID {
			return errors.New("只有申请人可以取消审批")
		}

		// 更新部署记录状态
		updates := map[string]any{
			"status": "cancelled",
		}
		if err := tx.Model(&record).Updates(updates).Error; err != nil {
			return err
		}

		logger.L().Info("审批取消: record_id=%d, user=%s", recordID, userName)
		return nil
	})
}

// GetPendingList 获取待审批列表
func (s *ApprovalService) GetPendingList(ctx context.Context, approverID uint) ([]models.DeployRecord, error) {
	var records []models.DeployRecord
	err := s.db.Where("status = ? AND need_approval = ?", "pending", true).
		Order("created_at DESC").
		Find(&records).Error
	return records, err
}

// GetHistory 获取审批历史
func (s *ApprovalService) GetHistory(ctx context.Context, page, pageSize int, appID *uint, env string, status string) ([]models.DeployRecord, int64, error) {
	var records []models.DeployRecord
	var total int64

	query := s.db.Model(&models.DeployRecord{}).Where("need_approval = ?", true)

	if appID != nil && *appID > 0 {
		query = query.Where("application_id = ?", *appID)
	}
	if env != "" {
		query = query.Where("env_name = ?", env)
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	err := query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&records).Error
	return records, total, err
}

// GetApprovalRecords 获取某个部署记录的审批记录
func (s *ApprovalService) GetApprovalRecords(ctx context.Context, recordID uint) ([]models.ApprovalRecord, error) {
	var records []models.ApprovalRecord
	err := s.db.Where("record_id = ?", recordID).Order("created_at ASC").Find(&records).Error
	return records, err
}

// GetHistoryForExport 获取审批历史用于导出（不分页）
func (s *ApprovalService) GetHistoryForExport(ctx context.Context, env, status, startTime, endTime string) ([]map[string]interface{}, error) {
	query := s.db.Model(&models.DeployRecord{}).
		Select("deploy_records.*, applications.name as app_name").
		Joins("LEFT JOIN applications ON applications.id = deploy_records.application_id").
		Where("deploy_records.need_approval = ?", true)

	if env != "" {
		query = query.Where("deploy_records.env_name = ?", env)
	}
	if status != "" {
		query = query.Where("deploy_records.status = ?", status)
	}
	if startTime != "" {
		query = query.Where("deploy_records.created_at >= ?", startTime)
	}
	if endTime != "" {
		query = query.Where("deploy_records.created_at <= ?", endTime+" 23:59:59")
	}

	var results []map[string]interface{}
	err := query.Order("deploy_records.created_at DESC").Limit(10000).Find(&results).Error
	return results, err
}

// GetDeployRequestDetail 获取发布请求详情
func (s *ApprovalService) GetDeployRequestDetail(ctx context.Context, recordID uint, userID uint) (*models.DeployRecord, []models.ApprovalRecord, []map[string]interface{}, bool, error) {
	var record models.DeployRecord
	if err := s.db.First(&record, recordID).Error; err != nil {
		return nil, nil, nil, false, err
	}

	// 获取审批记录
	var approvalRecords []models.ApprovalRecord
	s.db.Where("record_id = ?", recordID).Order("created_at ASC").Find(&approvalRecords)

	// 获取审批人列表
	approvers := []map[string]interface{}{}
	needApproval, approverIDs, _ := s.ruleService.NeedApproval(record.ApplicationID, record.EnvName)
	if needApproval && len(approverIDs) > 0 {
		var users []models.User
		s.db.Where("id IN ?", approverIDs).Find(&users)
		for _, u := range users {
			approved := false
			rejected := false
			for _, ar := range approvalRecords {
				if ar.ApproverID == u.ID {
					if ar.Action == "approve" {
						approved = true
					} else if ar.Action == "reject" {
						rejected = true
					}
				}
			}
			approvers = append(approvers, map[string]interface{}{
				"id":       u.ID,
				"name":     u.Username,
				"role":     "审批人",
				"approved": approved,
				"rejected": rejected,
			})
		}
	}

	// 检查当前用户是否可以审批
	canApprove := false
	if record.Status == "pending" {
		canApprove = slices.Contains(approverIDs, userID)
	}

	return &record, approvalRecords, approvers, canApprove, nil
}

// CheckApprovalRequired 检查是否需要审批
func (s *ApprovalService) CheckApprovalRequired(ctx context.Context, appID uint, env string) (bool, []string, error) {
	needApproval, approverIDs, err := s.ruleService.NeedApproval(appID, env)
	if err != nil {
		return false, nil, err
	}

	var approverNames []string
	if needApproval && len(approverIDs) > 0 {
		var users []models.User
		s.db.Where("id IN ?", approverIDs).Find(&users)
		for _, u := range users {
			approverNames = append(approverNames, u.Username)
		}
	}

	return needApproval, approverNames, nil
}
