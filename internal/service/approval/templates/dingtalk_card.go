package templates

import (
	"fmt"
	"time"
)

// DingTalkCardBuilder 钉钉卡片构建器
type DingTalkCardBuilder struct{}

// NewDingTalkCardBuilder 创建钉钉卡片构建器
func NewDingTalkCardBuilder() *DingTalkCardBuilder {
	return &DingTalkCardBuilder{}
}

// BuildApprovalRequestCard 构建审批请求卡片（ActionCard 类型）
func (b *DingTalkCardBuilder) BuildApprovalRequestCard(params ApprovalRequestParams) map[string]any {
	timeoutInfo := ""
	if params.TimeoutAt != nil {
		timeoutInfo = fmt.Sprintf("\n\n⏰ 超时时间: %s", params.TimeoutAt.Format("2006-01-02 15:04:05"))
	}

	modeText := getModeText(params.ApproveMode, params.ApproveCount)

	markdown := fmt.Sprintf(`### 🔔 发布审批请求

**应用名称**: %s

**部署环境**: %s

**版本**: %s

**申请人**: %s

**审批模式**: %s

**当前节点**: %s (第%d节点)

**发布说明**: %s%s`,
		params.AppName,
		params.EnvName,
		params.Version,
		params.Operator,
		modeText,
		params.NodeName,
		params.NodeOrder,
		params.Description,
		timeoutInfo,
	)

	return map[string]any{
		"msgtype": "actionCard",
		"actionCard": map[string]any{
			"title":          "发布审批请求",
			"text":           markdown,
			"btnOrientation": "0", // 按钮竖直排列
			"btns": []any{
				map[string]any{
					"title":     "✅ 通过",
					"actionURL": fmt.Sprintf("%s?action=approve&node_instance_id=%d", params.CallbackURL, params.NodeInstanceID),
				},
				map[string]any{
					"title":     "❌ 拒绝",
					"actionURL": fmt.Sprintf("%s?action=reject&node_instance_id=%d", params.CallbackURL, params.NodeInstanceID),
				},
				map[string]any{
					"title":     "📋 查看详情",
					"actionURL": fmt.Sprintf("/approval/instances/%d", params.InstanceID),
				},
			},
		},
	}
}

// BuildApprovalResultCard 构建审批结果卡片
func (b *DingTalkCardBuilder) BuildApprovalResultCard(params ApprovalResultParams) map[string]any {
	var title, resultText string
	switch params.Result {
	case "approved":
		title = "✅ 审批已通过"
		resultText = "您的发布申请已通过审批，即将开始部署"
	case "rejected":
		title = "❌ 审批已拒绝"
		resultText = "您的发布申请已被拒绝"
	case "cancelled":
		title = "⚪ 审批已取消"
		resultText = "审批已被取消"
	case "timeout":
		title = "⏰ 审批已超时"
		resultText = "审批已超时自动取消"
	default:
		title = "📋 审批状态更新"
		resultText = fmt.Sprintf("审批状态: %s", params.Result)
	}

	markdown := fmt.Sprintf(`### %s

%s

**审批链**: %s

**操作人**: %s

**完成时间**: %s`,
		title,
		resultText,
		params.ChainName,
		params.Operator,
		time.Now().Format("2006-01-02 15:04:05"),
	)

	return map[string]any{
		"msgtype": "markdown",
		"markdown": map[string]any{
			"title": title,
			"text":  markdown,
		},
	}
}

// BuildTimeoutReminderCard 构建超时提醒卡片
func (b *DingTalkCardBuilder) BuildTimeoutReminderCard(params TimeoutReminderParams) map[string]any {
	remainingTime := ""
	if params.TimeoutAt != nil {
		remaining := time.Until(*params.TimeoutAt)
		if remaining > 0 {
			remainingTime = fmt.Sprintf("剩余 %d 分钟", int(remaining.Minutes()))
		}
	}

	markdown := fmt.Sprintf(`### ⏰ 审批即将超时提醒

您有一个审批请求即将超时，请尽快处理！

**节点名称**: %s

**%s**`,
		params.NodeName,
		remainingTime,
	)

	return map[string]any{
		"msgtype": "actionCard",
		"actionCard": map[string]any{
			"title":       "审批即将超时提醒",
			"text":        markdown,
			"singleTitle": "立即处理",
			"singleURL":   fmt.Sprintf("/approval/instances/%d", params.InstanceID),
		},
	}
}

// DingTalkTextMessage 钉钉文本消息
func DingTalkTextMessage(content string, atMobiles []string, isAtAll bool) map[string]any {
	return map[string]any{
		"msgtype": "text",
		"text": map[string]any{
			"content": content,
		},
		"at": map[string]any{
			"atMobiles": atMobiles,
			"isAtAll":   isAtAll,
		},
	}
}
