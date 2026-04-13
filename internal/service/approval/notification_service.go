package approval

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"devops/internal/domain/notification/service/feishu"
	"devops/internal/models"
	"devops/internal/repository"
	"devops/internal/service/notification"

	"gorm.io/gorm"
)

// NotificationService 审批通知服务
type NotificationService struct {
	oaNotifyRepo    *repository.OANotifyConfigRepository
	feishuAppRepo   *repository.FeishuAppRepository
	templateService *notification.TemplateService
}

// NewNotificationService 创建通知服务
func NewNotificationService(db *gorm.DB) *NotificationService {
	return &NotificationService{
		oaNotifyRepo:    repository.NewOANotifyConfigRepository(db),
		feishuAppRepo:   repository.NewFeishuAppRepository(db),
		templateService: notification.NewTemplateService(repository.NewMessageTemplateRepository(db)),
	}
}

// ApprovalNotifyRequest 审批通知请求
type ApprovalNotifyRequest struct {
	Instance     *models.ApprovalInstance
	NodeInstance *models.ApprovalNodeInstance
	Approvers    []string // 审批人用户ID列表
	AppName      string
	EnvName      string
	Operator     string
	Description  string
}

func (s *NotificationService) getFeishuClient(ctx context.Context) (*feishu.Client, error) {
	notifyConfig, err := s.oaNotifyRepo.GetDefault(ctx)
	if err != nil {
		return nil, err
	}
	if notifyConfig == nil {
		app, err := s.feishuAppRepo.GetDefault(ctx)
		if err != nil {
			return nil, err
		}
		if app == nil {
			return nil, fmt.Errorf("no default feishu app found")
		}
		return feishu.NewClientWithApp(app.AppID, app.AppSecret), nil
	}

	var feishuApp *models.FeishuApp
	if notifyConfig.AppID > 0 {
		feishuApp, err = s.feishuAppRepo.GetByID(ctx, notifyConfig.AppID)
	} else {
		feishuApp, err = s.feishuAppRepo.GetDefault(ctx)
	}

	if err != nil {
		return nil, err
	}
	if feishuApp == nil {
		return nil, fmt.Errorf("no feishu app found")
	}

	return feishu.NewClientWithApp(feishuApp.AppID, feishuApp.AppSecret), nil
}

func (s *NotificationService) sendCard(ctx context.Context, receiverID, templateName string, data map[string]interface{}) error {
	client, err := s.getFeishuClient(ctx)
	if err != nil {
		return err
	}

	content, err := s.templateService.Render(ctx, templateName, data)
	if err != nil {
		return fmt.Errorf("render template error: %w", err)
	}

	return client.SendMessage(ctx, receiverID, "user_id", "interactive", content)
}

// SendApprovalRequest 发送审批请求通知
func (s *NotificationService) SendApprovalRequest(ctx context.Context, req *ApprovalNotifyRequest) error {
	timeoutInfo := ""
	if req.NodeInstance.TimeoutAt != nil {
		timeoutInfo = fmt.Sprintf("⏰ 超时时间: %s", req.NodeInstance.TimeoutAt.Format("2006-01-02 15:04:05"))
	}
	modeText := getModeText(req.NodeInstance.ApproveMode, req.NodeInstance.ApproveCount)

	data := map[string]interface{}{
		"AppName":        req.AppName,
		"EnvName":        req.EnvName,
		"Operator":       req.Operator,
		"ModeText":       modeText,
		"NodeName":       req.NodeInstance.NodeName,
		"NodeOrder":      req.NodeInstance.NodeOrder,
		"Description":    req.Description,
		"TimeoutInfo":    timeoutInfo,
		"NodeInstanceID": req.NodeInstance.ID,
		"InstanceID":     req.Instance.ID,
	}

	for _, approverID := range req.Approvers {
		if err := s.sendCard(ctx, approverID, "APPROVAL_REQUEST", data); err != nil {
			log.Printf("[NotificationService] 发送审批请求通知失败: approver=%s, err=%v", approverID, err)
		} else {
			log.Printf("[NotificationService] 发送审批请求通知成功: approver=%s", approverID)
		}
	}

	return nil
}

// SendApprovalResult 发送审批结果通知
func (s *NotificationService) SendApprovalResult(ctx context.Context, instance *models.ApprovalInstance, result string, operator string) error {
	var headerTemplate, headerTitle, resultText string
	switch result {
	case "approved":
		headerTemplate = "green"
		headerTitle = "✅ 审批已通过"
		resultText = "您的发布申请已通过审批"
	case "rejected":
		headerTemplate = "red"
		headerTitle = "❌ 审批已拒绝"
		resultText = "您的发布申请已被拒绝"
	case "cancelled":
		headerTemplate = "grey"
		headerTitle = "⚪ 审批已取消"
		resultText = "审批已被取消"
	default:
		headerTemplate = "blue"
		headerTitle = "📋 审批状态更新"
		resultText = fmt.Sprintf("审批状态: %s", result)
	}

	// 准备模板数据 (即使目前不发送，也保留逻辑结构)
	_ = map[string]interface{}{
		"Title":       headerTitle,
		"HeaderColor": headerTemplate,
		"ResultText":  resultText,
		"ChainName":   instance.ChainName,
		"Operator":    operator,
		"Time":        time.Now().Format("2006-01-02 15:04:05"),
	}

	// 通知发起人
	// TODO: 获取发起人的飞书用户ID
	log.Printf("[NotificationService] 审批结果通知: instance=%d, result=%s", instance.ID, result)

	return nil
}

// SendTimeoutReminder 发送超时提醒通知
func (s *NotificationService) SendTimeoutReminder(ctx context.Context, nodeInstance *models.ApprovalNodeInstance, approvers []string) error {
	remainingTime := ""
	if nodeInstance.TimeoutAt != nil {
		remaining := time.Until(*nodeInstance.TimeoutAt)
		if remaining > 0 {
			remainingTime = fmt.Sprintf("剩余 %d 分钟", int(remaining.Minutes()))
		}
	}

	data := map[string]interface{}{
		"NodeName":      nodeInstance.NodeName,
		"RemainingTime": remainingTime,
		"InstanceID":    nodeInstance.InstanceID,
	}

	for _, approverID := range approvers {
		if err := s.sendCard(ctx, approverID, "APPROVAL_TIMEOUT_REMINDER", data); err != nil {
			log.Printf("[NotificationService] 发送超时提醒失败: approver=%s, err=%v", approverID, err)
		}
	}

	return nil
}

// SendTimeoutCancelled 发送超时取消通知
func (s *NotificationService) SendTimeoutCancelled(ctx context.Context, instance *models.ApprovalInstance, requesterID string) error {
	data := map[string]interface{}{
		"ChainName": instance.ChainName,
		"Time":      time.Now().Format("2006-01-02 15:04:05"),
	}

	// 通知发起人
	if requesterID != "" {
		if err := s.sendCard(ctx, requesterID, "APPROVAL_TIMEOUT_CANCELLED", data); err != nil {
			log.Printf("[NotificationService] 发送超时取消通知失败: requester=%s, err=%v", requesterID, err)
		} else {
			log.Printf("[NotificationService] 发送超时取消通知成功: requester=%s", requesterID)
		}
	}

	return nil
}

// getModeText 获取审批模式文本
func getModeText(mode string, count int) string {
	switch mode {
	case "any":
		return "任一人通过"
	case "all":
		return "所有人通过"
	case "count":
		return fmt.Sprintf("%d人通过", count)
	default:
		return mode
	}
}

// ParseApprovers 解析审批人ID列表
func ParseApprovers(approvers string) []string {
	if approvers == "" {
		return nil
	}
	return strings.Split(approvers, ",")
}
