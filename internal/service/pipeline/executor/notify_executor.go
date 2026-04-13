package executor

import (
	"bytes"
	"context"
	"devops/pkg/logger"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"text/template"
	"time"
)

// PipelineNotifyExecutor 流水线通知执行器
type PipelineNotifyExecutor struct {
	httpClient *http.Client
}

// NewPipelineNotifyExecutor 创建流水线通知执行器
func NewPipelineNotifyExecutor() *PipelineNotifyExecutor {
	return &PipelineNotifyExecutor{
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// NotifyConfig 通知配置
type NotifyConfig struct {
	Type       string            `json:"type"`        // feishu, dingtalk, wechat, email, webhook
	WebhookURL string            `json:"webhook_url"` // Webhook URL
	Secret     string            `json:"secret"`      // 签名密钥
	Template   string            `json:"template"`    // 消息模板
	AtUsers    []string          `json:"at_users"`    // @用户列表
	AtAll      bool              `json:"at_all"`      // @所有人
	Extra      map[string]string `json:"extra"`       // 额外配置
}

// NotifyContext 通知上下文
type NotifyContext struct {
	PipelineName string `json:"pipeline_name"`
	PipelineID   uint   `json:"pipeline_id"`
	RunID        uint   `json:"run_id"`
	Status       string `json:"status"`
	TriggerBy    string `json:"trigger_by"`
	GitBranch    string `json:"git_branch"`
	GitCommit    string `json:"git_commit"`
	GitMessage   string `json:"git_message"`
	Duration     int    `json:"duration"`
	StartedAt    string `json:"started_at"`
	FinishedAt   string `json:"finished_at"`
	URL          string `json:"url"` // 详情页 URL
}

// Execute 执行通知
func (e *PipelineNotifyExecutor) Execute(ctx context.Context, config *NotifyConfig, notifyCtx *NotifyContext) error {
	log := logger.L().WithField("notify_type", config.Type)
	log.Info("发送通知")

	switch config.Type {
	case "feishu":
		return e.sendFeishu(ctx, config, notifyCtx)
	case "dingtalk":
		return e.sendDingtalk(ctx, config, notifyCtx)
	case "wechat":
		return e.sendWechat(ctx, config, notifyCtx)
	case "webhook":
		return e.sendWebhook(ctx, config, notifyCtx)
	default:
		return fmt.Errorf("不支持的通知类型: %s", config.Type)
	}
}

// sendFeishu 发送飞书通知
func (e *PipelineNotifyExecutor) sendFeishu(ctx context.Context, config *NotifyConfig, notifyCtx *NotifyContext) error {
	// 构建消息内容
	content := e.renderTemplate(config.Template, notifyCtx)
	if content == "" {
		content = e.defaultFeishuContent(notifyCtx)
	}

	// 飞书卡片消息
	msg := map[string]any{
		"msg_type": "interactive",
		"card": map[string]any{
			"config": map[string]any{
				"wide_screen_mode": true,
			},
			"header": map[string]any{
				"title": map[string]any{
					"tag":     "plain_text",
					"content": fmt.Sprintf("流水线 %s %s", notifyCtx.PipelineName, e.statusText(notifyCtx.Status)),
				},
				"template": e.statusColor(notifyCtx.Status),
			},
			"elements": []map[string]any{
				{
					"tag": "div",
					"text": map[string]any{
						"tag":     "lark_md",
						"content": content,
					},
				},
				{
					"tag": "action",
					"actions": []map[string]any{
						{
							"tag": "button",
							"text": map[string]any{
								"tag":     "plain_text",
								"content": "查看详情",
							},
							"url":  notifyCtx.URL,
							"type": "primary",
						},
					},
				},
			},
		},
	}

	return e.postJSON(ctx, config.WebhookURL, msg)
}

func (e *PipelineNotifyExecutor) defaultFeishuContent(ctx *NotifyContext) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("**执行ID**: %d\n", ctx.RunID))
	sb.WriteString(fmt.Sprintf("**触发人**: %s\n", ctx.TriggerBy))
	if ctx.GitBranch != "" {
		sb.WriteString(fmt.Sprintf("**分支**: %s\n", ctx.GitBranch))
	}
	if ctx.GitCommit != "" {
		sb.WriteString(fmt.Sprintf("**提交**: %s\n", ctx.GitCommit[:8]))
	}
	if ctx.Duration > 0 {
		sb.WriteString(fmt.Sprintf("**耗时**: %s\n", e.formatDuration(ctx.Duration)))
	}
	return sb.String()
}

// sendDingtalk 发送钉钉通知
func (e *PipelineNotifyExecutor) sendDingtalk(ctx context.Context, config *NotifyConfig, notifyCtx *NotifyContext) error {
	content := e.renderTemplate(config.Template, notifyCtx)
	if content == "" {
		content = e.defaultDingtalkContent(notifyCtx)
	}

	// 钉钉 Markdown 消息
	msg := map[string]any{
		"msgtype": "markdown",
		"markdown": map[string]any{
			"title": fmt.Sprintf("流水线 %s %s", notifyCtx.PipelineName, e.statusText(notifyCtx.Status)),
			"text":  content,
		},
	}

	// @用户
	if len(config.AtUsers) > 0 || config.AtAll {
		msg["at"] = map[string]any{
			"atMobiles": config.AtUsers,
			"isAtAll":   config.AtAll,
		}
	}

	return e.postJSON(ctx, config.WebhookURL, msg)
}

func (e *PipelineNotifyExecutor) defaultDingtalkContent(ctx *NotifyContext) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("### 流水线 %s %s\n\n", ctx.PipelineName, e.statusEmoji(ctx.Status)))
	sb.WriteString(fmt.Sprintf("- **执行ID**: %d\n", ctx.RunID))
	sb.WriteString(fmt.Sprintf("- **触发人**: %s\n", ctx.TriggerBy))
	if ctx.GitBranch != "" {
		sb.WriteString(fmt.Sprintf("- **分支**: %s\n", ctx.GitBranch))
	}
	if ctx.GitCommit != "" {
		sb.WriteString(fmt.Sprintf("- **提交**: %s\n", ctx.GitCommit[:8]))
	}
	if ctx.Duration > 0 {
		sb.WriteString(fmt.Sprintf("- **耗时**: %s\n", e.formatDuration(ctx.Duration)))
	}
	sb.WriteString(fmt.Sprintf("\n[查看详情](%s)", ctx.URL))
	return sb.String()
}

// sendWechat 发送企业微信通知
func (e *PipelineNotifyExecutor) sendWechat(ctx context.Context, config *NotifyConfig, notifyCtx *NotifyContext) error {
	content := e.renderTemplate(config.Template, notifyCtx)
	if content == "" {
		content = e.defaultWechatContent(notifyCtx)
	}

	// 企业微信 Markdown 消息
	msg := map[string]any{
		"msgtype": "markdown",
		"markdown": map[string]any{
			"content": content,
		},
	}

	return e.postJSON(ctx, config.WebhookURL, msg)
}

func (e *PipelineNotifyExecutor) defaultWechatContent(ctx *NotifyContext) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("## 流水线 %s %s\n", ctx.PipelineName, e.statusEmoji(ctx.Status)))
	sb.WriteString(fmt.Sprintf("> 执行ID: <font color=\"comment\">%d</font>\n", ctx.RunID))
	sb.WriteString(fmt.Sprintf("> 触发人: <font color=\"comment\">%s</font>\n", ctx.TriggerBy))
	if ctx.GitBranch != "" {
		sb.WriteString(fmt.Sprintf("> 分支: <font color=\"comment\">%s</font>\n", ctx.GitBranch))
	}
	if ctx.Duration > 0 {
		sb.WriteString(fmt.Sprintf("> 耗时: <font color=\"comment\">%s</font>\n", e.formatDuration(ctx.Duration)))
	}
	sb.WriteString(fmt.Sprintf("\n[查看详情](%s)", ctx.URL))
	return sb.String()
}

// sendWebhook 发送通用 Webhook 通知
func (e *PipelineNotifyExecutor) sendWebhook(ctx context.Context, config *NotifyConfig, notifyCtx *NotifyContext) error {
	// 直接发送 JSON 格式的上下文
	return e.postJSON(ctx, config.WebhookURL, notifyCtx)
}

// postJSON 发送 JSON POST 请求
func (e *PipelineNotifyExecutor) postJSON(ctx context.Context, url string, data any) error {
	body, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("序列化消息失败: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("创建请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := e.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("发送请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("请求失败: %d - %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// renderTemplate 渲染模板
func (e *PipelineNotifyExecutor) renderTemplate(tmplStr string, ctx *NotifyContext) string {
	if tmplStr == "" {
		return ""
	}

	tmpl, err := template.New("notify").Parse(tmplStr)
	if err != nil {
		logger.L().WithField("error", err).Warn("解析通知模板失败")
		return ""
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, ctx); err != nil {
		logger.L().WithField("error", err).Warn("渲染通知模板失败")
		return ""
	}

	return buf.String()
}

// statusText 状态文本
func (e *PipelineNotifyExecutor) statusText(status string) string {
	switch status {
	case "success":
		return "构建成功"
	case "failed":
		return "构建失败"
	case "cancelled":
		return "已取消"
	case "running":
		return "运行中"
	default:
		return status
	}
}

// statusEmoji 状态 Emoji
func (e *PipelineNotifyExecutor) statusEmoji(status string) string {
	switch status {
	case "success":
		return "✅"
	case "failed":
		return "❌"
	case "cancelled":
		return "⚠️"
	case "running":
		return "🔄"
	default:
		return "📋"
	}
}

// statusColor 状态颜色 (飞书)
func (e *PipelineNotifyExecutor) statusColor(status string) string {
	switch status {
	case "success":
		return "green"
	case "failed":
		return "red"
	case "cancelled":
		return "orange"
	default:
		return "blue"
	}
}

// formatDuration 格式化时长
func (e *PipelineNotifyExecutor) formatDuration(seconds int) string {
	if seconds < 60 {
		return fmt.Sprintf("%d秒", seconds)
	}
	if seconds < 3600 {
		return fmt.Sprintf("%d分%d秒", seconds/60, seconds%60)
	}
	return fmt.Sprintf("%d时%d分", seconds/3600, (seconds%3600)/60)
}

// PipelineNotifyService 流水线通知服务
type PipelineNotifyService struct {
	executor *PipelineNotifyExecutor
}

// NewPipelineNotifyService 创建流水线通知服务
func NewPipelineNotifyService() *PipelineNotifyService {
	return &PipelineNotifyService{
		executor: NewPipelineNotifyExecutor(),
	}
}

// SendPipelineNotification 发送流水线通知
func (s *PipelineNotifyService) SendPipelineNotification(ctx context.Context, configs []NotifyConfig, notifyCtx *NotifyContext) []error {
	var errors []error

	for _, config := range configs {
		if err := s.executor.Execute(ctx, &config, notifyCtx); err != nil {
			logger.L().WithField("type", config.Type).WithField("error", err).Error("发送通知失败")
			errors = append(errors, err)
		}
	}

	return errors
}

// ParseNotifyConfigs 解析通知配置
func ParseNotifyConfigs(configJSON string) ([]NotifyConfig, error) {
	if configJSON == "" {
		return nil, nil
	}

	var configs []NotifyConfig
	if err := json.Unmarshal([]byte(configJSON), &configs); err != nil {
		return nil, fmt.Errorf("解析通知配置失败: %w", err)
	}

	return configs, nil
}
