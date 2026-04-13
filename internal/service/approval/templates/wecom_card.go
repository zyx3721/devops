package templates

import (
	"fmt"
	"time"
)

// WeComCardBuilder 企业微信卡片构建器
type WeComCardBuilder struct{}

// NewWeComCardBuilder 创建企业微信卡片构建器
func NewWeComCardBuilder() *WeComCardBuilder {
	return &WeComCardBuilder{}
}

// BuildApprovalRequestCard 构建审批请求卡片（模板卡片类型）
func (b *WeComCardBuilder) BuildApprovalRequestCard(params ApprovalRequestParams) map[string]any {
	timeoutInfo := ""
	if params.TimeoutAt != nil {
		timeoutInfo = params.TimeoutAt.Format("2006-01-02 15:04:05")
	}

	modeText := getModeText(params.ApproveMode, params.ApproveCount)

	return map[string]any{
		"msgtype": "template_card",
		"template_card": map[string]any{
			"card_type": "button_interaction",
			"source": map[string]any{
				"icon_url": "",
				"desc":     "DevOps平台",
			},
			"main_title": map[string]any{
				"title": "🔔 发布审批请求",
				"desc":  fmt.Sprintf("%s - %s", params.AppName, params.EnvName),
			},
			"horizontal_content_list": []any{
				map[string]any{
					"keyname": "应用名称",
					"value":   params.AppName,
				},
				map[string]any{
					"keyname": "部署环境",
					"value":   params.EnvName,
				},
				map[string]any{
					"keyname": "版本",
					"value":   params.Version,
				},
				map[string]any{
					"keyname": "申请人",
					"value":   params.Operator,
				},
				map[string]any{
					"keyname": "审批模式",
					"value":   modeText,
				},
				map[string]any{
					"keyname": "当前节点",
					"value":   fmt.Sprintf("%s (第%d节点)", params.NodeName, params.NodeOrder),
				},
			},
			"sub_title_text": params.Description,
			"card_action": map[string]any{
				"type": 1,
				"url":  fmt.Sprintf("/approval/instances/%d", params.InstanceID),
			},
			"button_list": []any{
				map[string]any{
					"text":  "通过",
					"style": 1, // 绿色
					"key":   fmt.Sprintf("approve_%d", params.NodeInstanceID),
				},
				map[string]any{
					"text":  "拒绝",
					"style": 3, // 红色
					"key":   fmt.Sprintf("reject_%d", params.NodeInstanceID),
				},
			},
			"task_id": fmt.Sprintf("approval_%d_%d", params.InstanceID, params.NodeInstanceID),
		},
		"_timeout_info": timeoutInfo, // 额外信息，用于显示
	}
}

// BuildApprovalResultCard 构建审批结果卡片
func (b *WeComCardBuilder) BuildApprovalResultCard(params ApprovalResultParams) map[string]any {
	var title, resultText, emphasisColor string
	switch params.Result {
	case "approved":
		title = "✅ 审批已通过"
		resultText = "您的发布申请已通过审批，即将开始部署"
		emphasisColor = "green"
	case "rejected":
		title = "❌ 审批已拒绝"
		resultText = "您的发布申请已被拒绝"
		emphasisColor = "red"
	case "cancelled":
		title = "⚪ 审批已取消"
		resultText = "审批已被取消"
		emphasisColor = "gray"
	case "timeout":
		title = "⏰ 审批已超时"
		resultText = "审批已超时自动取消"
		emphasisColor = "orange"
	default:
		title = "📋 审批状态更新"
		resultText = fmt.Sprintf("审批状态: %s", params.Result)
		emphasisColor = "blue"
	}

	return map[string]any{
		"msgtype": "template_card",
		"template_card": map[string]any{
			"card_type": "text_notice",
			"source": map[string]any{
				"icon_url": "",
				"desc":     "DevOps平台",
			},
			"main_title": map[string]any{
				"title": title,
			},
			"emphasis_content": map[string]any{
				"title": resultText,
				"desc":  emphasisColor,
			},
			"horizontal_content_list": []any{
				map[string]any{
					"keyname": "审批链",
					"value":   params.ChainName,
				},
				map[string]any{
					"keyname": "操作人",
					"value":   params.Operator,
				},
				map[string]any{
					"keyname": "完成时间",
					"value":   time.Now().Format("2006-01-02 15:04:05"),
				},
			},
			"card_action": map[string]any{
				"type": 1,
				"url":  fmt.Sprintf("/approval/instances/%d", params.InstanceID),
			},
		},
	}
}

// BuildTimeoutReminderCard 构建超时提醒卡片
func (b *WeComCardBuilder) BuildTimeoutReminderCard(params TimeoutReminderParams) map[string]any {
	remainingTime := ""
	if params.TimeoutAt != nil {
		remaining := time.Until(*params.TimeoutAt)
		if remaining > 0 {
			remainingTime = fmt.Sprintf("剩余 %d 分钟", int(remaining.Minutes()))
		}
	}

	return map[string]any{
		"msgtype": "template_card",
		"template_card": map[string]any{
			"card_type": "text_notice",
			"source": map[string]any{
				"icon_url": "",
				"desc":     "DevOps平台",
			},
			"main_title": map[string]any{
				"title": "⏰ 审批即将超时提醒",
			},
			"emphasis_content": map[string]any{
				"title": "请尽快处理",
				"desc":  remainingTime,
			},
			"horizontal_content_list": []any{
				map[string]any{
					"keyname": "节点名称",
					"value":   params.NodeName,
				},
			},
			"card_action": map[string]any{
				"type": 1,
				"url":  fmt.Sprintf("/approval/instances/%d", params.InstanceID),
			},
			"jump_list": []any{
				map[string]any{
					"type":  1,
					"title": "立即处理",
					"url":   fmt.Sprintf("/approval/instances/%d", params.InstanceID),
				},
			},
		},
	}
}

// WeComMarkdownMessage 企业微信 Markdown 消息
func WeComMarkdownMessage(content string) map[string]any {
	return map[string]any{
		"msgtype": "markdown",
		"markdown": map[string]any{
			"content": content,
		},
	}
}

// WeComTextMessage 企业微信文本消息
func WeComTextMessage(content string, mentionedList []string, mentionedMobileList []string) map[string]any {
	return map[string]any{
		"msgtype": "text",
		"text": map[string]any{
			"content":               content,
			"mentioned_list":        mentionedList,
			"mentioned_mobile_list": mentionedMobileList,
		},
	}
}
