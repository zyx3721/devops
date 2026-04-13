package templates

import (
	"fmt"
	"time"
)

// FeishuCardBuilder 飞书卡片构建器
type FeishuCardBuilder struct{}

// NewFeishuCardBuilder 创建飞书卡片构建器
func NewFeishuCardBuilder() *FeishuCardBuilder {
	return &FeishuCardBuilder{}
}

// ApprovalRequestParams 审批请求卡片参数
type ApprovalRequestParams struct {
	InstanceID     uint
	NodeInstanceID uint
	AppName        string
	EnvName        string
	Version        string
	Operator       string
	Description    string
	NodeName       string
	NodeOrder      int
	ApproveMode    string
	ApproveCount   int
	TimeoutAt      *time.Time
	CallbackURL    string
}

// BuildApprovalRequestCard 构建审批请求卡片
func (b *FeishuCardBuilder) BuildApprovalRequestCard(params ApprovalRequestParams) map[string]any {
	timeoutInfo := ""
	if params.TimeoutAt != nil {
		timeoutInfo = fmt.Sprintf("⏰ 超时时间: %s", params.TimeoutAt.Format("2006-01-02 15:04:05"))
	}

	modeText := getModeText(params.ApproveMode, params.ApproveCount)

	return map[string]any{
		"config": map[string]any{
			"wide_screen_mode": true,
		},
		"header": map[string]any{
			"title": map[string]any{
				"tag":     "plain_text",
				"content": "🔔 发布审批请求",
			},
			"template": "blue",
		},
		"elements": []any{
			map[string]any{
				"tag": "div",
				"fields": []any{
					map[string]any{
						"is_short": true,
						"text": map[string]any{
							"tag":     "lark_md",
							"content": fmt.Sprintf("**应用名称**\n%s", params.AppName),
						},
					},
					map[string]any{
						"is_short": true,
						"text": map[string]any{
							"tag":     "lark_md",
							"content": fmt.Sprintf("**部署环境**\n%s", params.EnvName),
						},
					},
					map[string]any{
						"is_short": true,
						"text": map[string]any{
							"tag":     "lark_md",
							"content": fmt.Sprintf("**版本**\n%s", params.Version),
						},
					},
					map[string]any{
						"is_short": true,
						"text": map[string]any{
							"tag":     "lark_md",
							"content": fmt.Sprintf("**申请人**\n%s", params.Operator),
						},
					},
					map[string]any{
						"is_short": true,
						"text": map[string]any{
							"tag":     "lark_md",
							"content": fmt.Sprintf("**审批模式**\n%s", modeText),
						},
					},
				},
			},
			map[string]any{
				"tag": "div",
				"text": map[string]any{
					"tag":     "lark_md",
					"content": fmt.Sprintf("**当前节点**: %s (第%d节点)", params.NodeName, params.NodeOrder),
				},
			},
			map[string]any{
				"tag": "div",
				"text": map[string]any{
					"tag":     "lark_md",
					"content": fmt.Sprintf("**发布说明**: %s", params.Description),
				},
			},
			map[string]any{
				"tag": "div",
				"text": map[string]any{
					"tag":     "lark_md",
					"content": timeoutInfo,
				},
			},
			map[string]any{
				"tag": "hr",
			},
			map[string]any{
				"tag": "action",
				"actions": []any{
					map[string]any{
						"tag": "button",
						"text": map[string]any{
							"tag":     "plain_text",
							"content": "✅ 通过",
						},
						"type": "primary",
						"value": map[string]any{
							"action":           "approve",
							"node_instance_id": params.NodeInstanceID,
						},
					},
					map[string]any{
						"tag": "button",
						"text": map[string]any{
							"tag":     "plain_text",
							"content": "❌ 拒绝",
						},
						"type": "danger",
						"value": map[string]any{
							"action":           "reject",
							"node_instance_id": params.NodeInstanceID,
						},
					},
					map[string]any{
						"tag": "button",
						"text": map[string]any{
							"tag":     "plain_text",
							"content": "📋 查看详情",
						},
						"type": "default",
						"url":  fmt.Sprintf("/approval/instances/%d", params.InstanceID),
					},
				},
			},
		},
	}
}

// ApprovalResultParams 审批结果卡片参数
type ApprovalResultParams struct {
	InstanceID uint
	ChainName  string
	Result     string // approved, rejected, cancelled
	Operator   string
}

// BuildApprovalResultCard 构建审批结果卡片
func (b *FeishuCardBuilder) BuildApprovalResultCard(params ApprovalResultParams) map[string]any {
	var headerTemplate, headerTitle, resultText string
	switch params.Result {
	case "approved":
		headerTemplate = "green"
		headerTitle = "✅ 审批已通过"
		resultText = "您的发布申请已通过审批，即将开始部署"
	case "rejected":
		headerTemplate = "red"
		headerTitle = "❌ 审批已拒绝"
		resultText = "您的发布申请已被拒绝"
	case "cancelled":
		headerTemplate = "grey"
		headerTitle = "⚪ 审批已取消"
		resultText = "审批已被取消"
	case "timeout":
		headerTemplate = "orange"
		headerTitle = "⏰ 审批已超时"
		resultText = "审批已超时自动取消"
	default:
		headerTemplate = "blue"
		headerTitle = "📋 审批状态更新"
		resultText = fmt.Sprintf("审批状态: %s", params.Result)
	}

	return map[string]any{
		"config": map[string]any{
			"wide_screen_mode": true,
		},
		"header": map[string]any{
			"title": map[string]any{
				"tag":     "plain_text",
				"content": headerTitle,
			},
			"template": headerTemplate,
		},
		"elements": []any{
			map[string]any{
				"tag": "div",
				"text": map[string]any{
					"tag":     "lark_md",
					"content": resultText,
				},
			},
			map[string]any{
				"tag": "div",
				"fields": []any{
					map[string]any{
						"is_short": true,
						"text": map[string]any{
							"tag":     "lark_md",
							"content": fmt.Sprintf("**审批链**\n%s", params.ChainName),
						},
					},
					map[string]any{
						"is_short": true,
						"text": map[string]any{
							"tag":     "lark_md",
							"content": fmt.Sprintf("**操作人**\n%s", params.Operator),
						},
					},
				},
			},
			map[string]any{
				"tag": "div",
				"text": map[string]any{
					"tag":     "lark_md",
					"content": fmt.Sprintf("**完成时间**: %s", time.Now().Format("2006-01-02 15:04:05")),
				},
			},
		},
	}
}

// TimeoutReminderParams 超时提醒卡片参数
type TimeoutReminderParams struct {
	InstanceID     uint
	NodeInstanceID uint
	NodeName       string
	TimeoutAt      *time.Time
}

// BuildTimeoutReminderCard 构建超时提醒卡片
func (b *FeishuCardBuilder) BuildTimeoutReminderCard(params TimeoutReminderParams) map[string]any {
	remainingTime := ""
	if params.TimeoutAt != nil {
		remaining := time.Until(*params.TimeoutAt)
		if remaining > 0 {
			remainingTime = fmt.Sprintf("剩余 %d 分钟", int(remaining.Minutes()))
		}
	}

	return map[string]any{
		"config": map[string]any{
			"wide_screen_mode": true,
		},
		"header": map[string]any{
			"title": map[string]any{
				"tag":     "plain_text",
				"content": "⏰ 审批即将超时提醒",
			},
			"template": "orange",
		},
		"elements": []any{
			map[string]any{
				"tag": "div",
				"text": map[string]any{
					"tag":     "lark_md",
					"content": fmt.Sprintf("您有一个审批请求即将超时，请尽快处理！\n\n**节点名称**: %s\n**%s**", params.NodeName, remainingTime),
				},
			},
			map[string]any{
				"tag": "hr",
			},
			map[string]any{
				"tag": "action",
				"actions": []any{
					map[string]any{
						"tag": "button",
						"text": map[string]any{
							"tag":     "plain_text",
							"content": "立即处理",
						},
						"type": "primary",
						"url":  fmt.Sprintf("/approval/instances/%d", params.InstanceID),
					},
				},
			},
		},
	}
}

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
