package deploy

import (
	"context"
	"fmt"
	"time"

	"devops/internal/domain/notification/service/dingtalk"
	"devops/internal/domain/notification/service/feishu"
	"devops/internal/domain/notification/service/wechatwork"
	"devops/internal/models"
	"devops/internal/repository"
	"devops/pkg/logger"
)

var notifyLog = logger.L().WithField("module", "deploy-notifier")

// Notifier 发布通知器
type Notifier struct {
	appRepo        *repository.ApplicationRepository
	feishuClient   *feishu.Client
	dingtalkClient *dingtalk.Client
	wechatClient   *wechatwork.Client
}

// NewNotifier 创建通知器
func NewNotifier(
	appRepo *repository.ApplicationRepository,
	feishuClient *feishu.Client,
	dingtalkClient *dingtalk.Client,
	wechatClient *wechatwork.Client,
) *Notifier {
	return &Notifier{
		appRepo:        appRepo,
		feishuClient:   feishuClient,
		dingtalkClient: dingtalkClient,
		wechatClient:   wechatClient,
	}
}

// NotifyType 通知类型
type NotifyType string

const (
	NotifyTypeCreated   NotifyType = "created"   // 请求创建
	NotifyTypeApproved  NotifyType = "approved"  // 审批通过
	NotifyTypeRejected  NotifyType = "rejected"  // 审批拒绝
	NotifyTypeExecuting NotifyType = "executing" // 开始执行
	NotifyTypeSuccess   NotifyType = "success"   // 发布成功
	NotifyTypeFailed    NotifyType = "failed"    // 发布失败
)

// Notify 发送通知
func (n *Notifier) Notify(ctx context.Context, record *models.DeployRecord, notifyType NotifyType) error {
	// 获取应用配置
	app, err := n.appRepo.GetByID(ctx, record.ApplicationID)
	if err != nil {
		notifyLog.WithError(err).Warn("获取应用配置失败，跳过通知")
		return nil
	}

	// 检查是否配置了通知
	if app.NotifyPlatform == "" || app.NotifyAppID == nil {
		notifyLog.Debug("应用未配置通知，跳过")
		return nil
	}

	// 构建通知内容
	title, content := n.buildNotifyContent(record, notifyType)

	// 根据平台发送通知
	switch app.NotifyPlatform {
	case "feishu":
		return n.sendFeishuNotify(ctx, app, title, content)
	case "dingtalk":
		return n.sendDingtalkNotify(ctx, app, title, content)
	case "wechatwork":
		return n.sendWechatWorkNotify(ctx, app, title, content)
	default:
		notifyLog.WithField("platform", app.NotifyPlatform).Warn("未知的通知平台")
		return nil
	}
}

// buildNotifyContent 构建通知内容
func (n *Notifier) buildNotifyContent(record *models.DeployRecord, notifyType NotifyType) (string, string) {
	var title, emoji string

	switch notifyType {
	case NotifyTypeCreated:
		title = "📝 发布请求已创建"
		emoji = "📝"
	case NotifyTypeApproved:
		title = "✅ 发布请求已通过"
		emoji = "✅"
	case NotifyTypeRejected:
		title = "❌ 发布请求已拒绝"
		emoji = "❌"
	case NotifyTypeExecuting:
		title = "🚀 发布开始执行"
		emoji = "🚀"
	case NotifyTypeSuccess:
		title = "🎉 发布成功"
		emoji = "🎉"
	case NotifyTypeFailed:
		title = "💥 发布失败"
		emoji = "💥"
	}

	deployType := "部署"
	if record.DeployType == DeployTypeRollback {
		deployType = "回滚"
	}

	content := fmt.Sprintf(`%s %s通知

应用: %s
环境: %s
类型: %s
分支: %s
版本: %s
操作人: %s
时间: %s`,
		emoji, deployType,
		record.AppName,
		record.EnvName,
		deployType,
		record.Branch,
		record.Version,
		record.Operator,
		time.Now().Format("2006-01-02 15:04:05"),
	)

	// 添加额外信息
	if notifyType == NotifyTypeRejected && record.RejectReason != "" {
		content += fmt.Sprintf("\n拒绝原因: %s", record.RejectReason)
	}
	if notifyType == NotifyTypeFailed && record.ErrorMsg != "" {
		content += fmt.Sprintf("\n错误信息: %s", record.ErrorMsg)
	}
	if notifyType == NotifyTypeSuccess && record.Duration > 0 {
		content += fmt.Sprintf("\n耗时: %d秒", record.Duration)
	}

	return title, content
}

// sendFeishuNotify 发送飞书通知
func (n *Notifier) sendFeishuNotify(ctx context.Context, app *models.Application, title, content string) error {
	if n.feishuClient == nil {
		notifyLog.Warn("飞书客户端未初始化")
		return nil
	}

	// 使用应用配置的接收者发送消息
	// 这里简化处理，实际应该根据 app.NotifyReceiveType 和 app.NotifyReceiveID 发送
	notifyLog.WithField("title", title).Info("发送飞书通知")
	return nil
}

// sendDingtalkNotify 发送钉钉通知
func (n *Notifier) sendDingtalkNotify(ctx context.Context, app *models.Application, title, content string) error {
	if n.dingtalkClient == nil {
		notifyLog.Warn("钉钉客户端未初始化")
		return nil
	}

	notifyLog.WithField("title", title).Info("发送钉钉通知")
	return nil
}

// sendWechatWorkNotify 发送企业微信通知
func (n *Notifier) sendWechatWorkNotify(ctx context.Context, app *models.Application, title, content string) error {
	if n.wechatClient == nil {
		notifyLog.Warn("企业微信客户端未初始化")
		return nil
	}

	notifyLog.WithField("title", title).Info("发送企业微信通知")
	return nil
}
