package notification

import (
	"context"

	"gorm.io/gorm"

	"devops/internal/models/system"
	"devops/internal/modules/system/repository"
)

// InitDefaultTemplates 初始化默认消息模板
func InitDefaultTemplates(db *gorm.DB) error {
	repo := repository.NewMessageTemplateRepository(db)
	svc := NewTemplateService(repo)

	defaults := []system.MessageTemplate{
		{
			Name:        "SSL_CERT_ALERT",
			Type:        "card",
			Description: "SSL证书过期告警卡片",
			IsActive:    true,
			Content: `{
  "config": { "wide_screen_mode": true },
  "header": {
    "title": { "tag": "plain_text", "content": "{{.Title}}" },
    "template": "{{.HeaderColor}}"
  },
  "elements": [
    {
      "tag": "div",
      "text": {
        "tag": "lark_md",
        "content": "**域名**: {{.Domain}}\n**告警级别**: {{.AlertLevel}}\n**剩余天数**: {{.DaysRemaining}}天\n**过期时间**: {{.ExpiryDate}}\n**颁发者**: {{.Issuer}}"
      }
    }
  ]
}`,
		},
		{
			Name:        "HEALTH_CHECK_ALERT",
			Type:        "card",
			Description: "健康检查失败告警卡片",
			IsActive:    true,
			Content: `{
  "config": { "wide_screen_mode": true },
  "header": {
    "title": { "tag": "plain_text", "content": "{{.Title}}" },
    "template": "red"
  },
  "elements": [
    {
      "tag": "div",
      "text": {
        "tag": "lark_md",
        "content": "**名称**: {{.Name}}\n**类型**: {{.Type}}\n**状态**: Unhealthy\n**错误信息**: {{.ErrorMsg}}\n**时间**: {{.Time}}"
      }
    }
  ]
}`,
		},
		{
			Name:        "COST_ANOMALY",
			Type:        "card",
			Description: "成本异常告警卡片",
			IsActive:    true,
			Content: `{
  "config": { "wide_screen_mode": true },
  "header": {
    "title": { "tag": "plain_text", "content": "⚠️ 成本异常告警" },
    "template": "orange"
  },
  "elements": [
    {
      "tag": "div",
      "text": {
        "tag": "lark_md",
        "content": "**日期**: {{.Date}}\n**实际成本**: ¥{{.ActualCost}}\n**预期成本**: ¥{{.ExpectedCost}}\n**偏差**: {{.Deviation}}%\n**说明**: {{.Message}}"
      }
    }
  ]
}`,
		},
		{
			Name:        "COST_WASTE",
			Type:        "card",
			Description: "资源浪费告警卡片",
			IsActive:    true,
			Content: `{
  "config": { "wide_screen_mode": true },
  "header": {
    "title": { "tag": "plain_text", "content": "💡 资源浪费提示" },
    "template": "green"
  },
  "elements": [
    {
      "tag": "div",
      "text": {
        "tag": "lark_md",
        "content": "**浪费成本**: ¥{{.WastedCost}}\n**闲置资源**: {{.IdleCount}} 个\n**超配资源**: {{.OverCount}} 个\n**建议**: {{.Message}}"
      }
    }
  ]
}`,
		},
		{
			Name:        "COST_BUDGET_EXCEEDED",
			Type:        "card",
			Description: "成本预算超支告警卡片",
			IsActive:    true,
			Content: `{
  "config": { "wide_screen_mode": true },
  "header": {
    "title": { "tag": "plain_text", "content": "🔴 成本预算超支" },
    "template": "red"
  },
  "elements": [
    {
      "tag": "div",
      "text": {
        "tag": "lark_md",
        "content": "**项目**: {{.Project}}\n**当前花费**: ¥{{.CurrentCost}}\n**预算**: ¥{{.Budget}}\n**超支**: ¥{{.Overrun}}\n**使用率**: {{.UsageRate}}%\n**警告**: {{.Message}}"
      }
    }
  ]
}`,
		},
		{
			Name:        "COST_BUDGET_WARNING",
			Type:        "card",
			Description: "成本预算预警卡片",
			IsActive:    true,
			Content: `{
  "config": { "wide_screen_mode": true },
  "header": {
    "title": { "tag": "plain_text", "content": "💰 成本预算预警" },
    "template": "orange"
  },
  "elements": [
    {
      "tag": "div",
      "text": {
        "tag": "lark_md",
        "content": "**项目**: {{.Project}}\n**当前花费**: ¥{{.CurrentCost}}\n**预算**: ¥{{.Budget}}\n**使用率**: {{.UsageRate}}%\n**警告**: {{.Message}}"
      }
    }
  ]
}`,
		},
		{
			Name:        "APPROVAL_REQUEST",
			Type:        "card",
			Description: "发布审批请求卡片",
			IsActive:    true,
			Content: `{
  "config": { "wide_screen_mode": true },
  "header": {
    "title": { "tag": "plain_text", "content": "🔔 发布审批请求" },
    "template": "blue"
  },
  "elements": [
    {
      "tag": "div",
      "fields": [
        { "is_short": true, "text": { "tag": "lark_md", "content": "**应用名称**\n{{.AppName}}" } },
        { "is_short": true, "text": { "tag": "lark_md", "content": "**部署环境**\n{{.EnvName}}" } },
        { "is_short": true, "text": { "tag": "lark_md", "content": "**申请人**\n{{.Operator}}" } },
        { "is_short": true, "text": { "tag": "lark_md", "content": "**审批模式**\n{{.ModeText}}" } }
      ]
    },
    {
      "tag": "div",
      "text": { "tag": "lark_md", "content": "**当前节点**: {{.NodeName}} (第{{.NodeOrder}}节点)" }
    },
    {
      "tag": "div",
      "text": { "tag": "lark_md", "content": "**发布说明**: {{.Description}}" }
    },
    {{if .TimeoutInfo}}
    {
      "tag": "div",
      "text": { "tag": "lark_md", "content": "{{.TimeoutInfo}}" }
    },
    {{end}}
    { "tag": "hr" },
    {
      "tag": "action",
      "actions": [
        {
          "tag": "button",
          "text": { "tag": "plain_text", "content": "✅ 通过" },
          "type": "primary",
          "value": { "action": "approve", "node_instance_id": "{{.NodeInstanceID}}" }
        },
        {
          "tag": "button",
          "text": { "tag": "plain_text", "content": "❌ 拒绝" },
          "type": "danger",
          "value": { "action": "reject", "node_instance_id": "{{.NodeInstanceID}}" }
        },
        {
          "tag": "button",
          "text": { "tag": "plain_text", "content": "📋 查看详情" },
          "type": "default",
          "url": "/approval/instances/{{.InstanceID}}"
        }
      ]
    }
  ]
}`,
		},
		{
			Name:        "JENKINS_FLOW_CARD",
			Type:        "card",
			Description: "Jenkins 发布申请卡片",
			IsActive:    true,
			Content: `{
  "config": { "wide_screen_mode": true },
  "header": {
    "title": { "tag": "plain_text", "content": "{{.Title}}" },
    "template": "blue"
  },
  "elements": [
    {{range $index, $service := .Services}}
    {{if $index}},{"tag": "hr"},{{end}}
    {
      "tag": "div",
      "text": { "tag": "lark_md", "content": "**服务**: {{$service.Name}}\n**Object ID**: {{$service.ObjectID}}" }
    },
    {
      "tag": "action",
      "actions": [
        {{range $actionIndex, $action := $service.Actions}}
        {{if $actionIndex}},{{end}}
        {
          "tag": "button",
          "text": { "tag": "plain_text", "content": "{{$action}}" },
          "type": "primary",
          "value": { "action": "{{$action}}", "service": "{{$service.Name}}", "request_id": "{{$.RequestID}}" }
        }
        {{end}}
      ]
    }
    {{end}}
  ]
}`,
		},
		{
			Name:        "APPROVAL_RESULT",
			Type:        "card",
			Description: "审批结果通知卡片",
			IsActive:    true,
			Content: `{
  "config": { "wide_screen_mode": true },
  "header": {
    "title": { "tag": "plain_text", "content": "{{.Title}}" },
    "template": "{{.HeaderColor}}"
  },
  "elements": [
    {
      "tag": "div",
      "text": { "tag": "lark_md", "content": "{{.ResultText}}" }
    },
    {
      "tag": "div",
      "fields": [
        { "is_short": true, "text": { "tag": "lark_md", "content": "**审批链**\n{{.ChainName}}" } },
        { "is_short": true, "text": { "tag": "lark_md", "content": "**操作人**\n{{.Operator}}" } }
      ]
    },
    {
      "tag": "div",
      "text": { "tag": "lark_md", "content": "**完成时间**: {{.Time}}" }
    }
  ]
}`,
		},
		{
			Name:        "APPROVAL_TIMEOUT_REMINDER",
			Type:        "card",
			Description: "审批超时提醒卡片",
			IsActive:    true,
			Content: `{
  "config": { "wide_screen_mode": true },
  "header": {
    "title": { "tag": "plain_text", "content": "⏰ 审批即将超时提醒" },
    "template": "orange"
  },
  "elements": [
    {
      "tag": "div",
      "text": { "tag": "lark_md", "content": "您有一个审批请求即将超时，请尽快处理！\n\n**节点名称**: {{.NodeName}}\n**{{.RemainingTime}}**" }
    },
    { "tag": "hr" },
    {
      "tag": "action",
      "actions": [
        {
          "tag": "button",
          "text": { "tag": "plain_text", "content": "立即处理" },
          "type": "primary",
          "url": "/approval/instances/{{.InstanceID}}"
        }
      ]
    }
  ]
}`,
		},
		{
			Name:        "APPROVAL_TIMEOUT_CANCELLED",
			Type:        "card",
			Description: "审批超时取消卡片",
			IsActive:    true,
			Content: `{
  "config": { "wide_screen_mode": true },
  "header": {
    "title": { "tag": "plain_text", "content": "⏰ 审批已超时取消" },
    "template": "red"
  },
  "elements": [
    {
      "tag": "div",
      "text": { "tag": "lark_md", "content": "您的发布申请因审批超时已被自动取消。\n\n**审批链**: {{.ChainName}}\n**取消时间**: {{.Time}}" }
    },
    { "tag": "hr" },
    {
      "tag": "note",
      "elements": [
        { "tag": "plain_text", "content": "如需重新发布，请重新提交发布申请" }
      ]
    }
  ]
}`,
		},
	}
	
	return svc.EnsureDefaultTemplates(context.Background(), defaults)
}
