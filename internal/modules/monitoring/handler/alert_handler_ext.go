package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	"devops/internal/models"
)

// Helper functions for sending notifications
func sendFeishuWebhook(url, content string) error {
	payload := map[string]interface{}{
		"msg_type": "interactive",
		"card":     json.RawMessage(content),
	}
	if !json.Valid([]byte(content)) {
		payload = map[string]interface{}{
			"msg_type": "text",
			"content": map[string]string{
				"text": content,
			},
		}
	}

	payloadBytes, _ := json.Marshal(payload)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(payloadBytes))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("feishu webhook status: %d", resp.StatusCode)
	}
	return nil
}

func sendDingTalkWebhook(url, secret, content string) error {
	// 简单实现：钉钉支持 markdown
	// 注意：实际生产中需要处理 secret 签名
	payload := map[string]interface{}{
		"msgtype": "markdown",
		"markdown": map[string]string{
			"title": "Alert Notification",
			"text":  content, // 钉钉不支持飞书的卡片 JSON，这里假设 content 是兼容的或者是纯文本
			// 实际情况：TemplateService 渲染出来的是飞书 JSON，直接发给钉钉会乱码。
			// 改进：TemplateService 应该支持 target_type，或者我们在这里做一个简单的转换
		},
	}

	// 简单的 JSON 转换尝试：如果是飞书卡片 JSON，提取纯文本
	if json.Valid([]byte(content)) {
		var cardObj struct {
			Header struct {
				Title struct {
					Content string `json:"content"`
				} `json:"title"`
			} `json:"header"`
			Elements []interface{} `json:"elements"` // 简化处理
		}
		if err := json.Unmarshal([]byte(content), &cardObj); err == nil {
			// 重新构造为钉钉 Markdown
			payload["markdown"] = map[string]string{
				"title": cardObj.Header.Title.Content,
				"text":  fmt.Sprintf("### %s\n\n(详见飞书卡片)", cardObj.Header.Title.Content),
			}
		}
	}

	payloadBytes, _ := json.Marshal(payload)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(payloadBytes))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("dingtalk webhook status: %d", resp.StatusCode)
	}
	return nil
}

func sendEmail(to, subject, title, content string) error {
	// 这里仅打印日志模拟发送，实际需要 SMTP 配置
	// 在生产环境中，应该注入一个 EmailService
	fmt.Printf("Mock Sending Email to %s: [%s] %s\n", to, title, content)
	return nil
}

// TriggerAlert 触发告警
func (h *AlertHandler) TriggerAlert(c *gin.Context) {
	var req struct {
		ConfigID uint                   `json:"config_id"` // 告警配置ID
		RuleName string                 `json:"rule_name"` // 或通过规则名称查找
		Data     map[string]interface{} `json:"data"`      // 告警数据
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	// 1. 查找告警配置
	var config *models.AlertConfig
	var err error

	if req.ConfigID > 0 {
		config, err = h.configRepo.GetByID(c.Request.Context(), req.ConfigID)
	} else if req.RuleName != "" {
		// 需要在 configRepo 中添加 GetByName 方法，或者这里临时用 List 查找
		// 暂时用 List 模拟查找
		configs, _, _ := h.configRepo.List(c.Request.Context(), "", 1, 1000)
		for _, cfg := range configs {
			if cfg.Name == req.RuleName {
				tempCfg := cfg
				config = &tempCfg
				break
			}
		}
		if config == nil {
			err = fmt.Errorf("rule not found: %s", req.RuleName)
		}
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "config_id or rule_name is required"})
		return
	}

	if err != nil || config == nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "Alert config not found"})
		return
	}

	if !config.Enabled {
		c.JSON(http.StatusOK, gin.H{"code": 0, "message": "Alert config is disabled"})
		return
	}

	// 异步处理告警发送
	go h.processAlertAsync(config, req.Data)

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "Alert triggered (processing in background)",
	})
}

func (h *AlertHandler) processAlertAsync(config *models.AlertConfig, data map[string]interface{}) {
	// 使用新的上下文，因为 gin.Context 在请求结束后会失效
	ctx := context.Background()

	// 2. 渲染模板
	var content string
	var err error
	if config.TemplateID != nil && *config.TemplateID > 0 {
		content, err = h.tmplSvc.RenderByID(ctx, uint(*config.TemplateID), data)
		if err != nil {
			alertLog.Error("Failed to render template: %v", err)
			// 降级：使用默认格式
			content = fmt.Sprintf("Alert: %s\nData: %v", config.Name, data)
		}
	} else {
		// 没有模板，使用默认格式
		content = fmt.Sprintf("Alert: %s\nData: %v", config.Name, data)
	}

	// 3. 发送通知
	successCount := h.sendToChannels(config, content)

	// 4. 记录历史
	history := &models.AlertHistory{
		AlertConfigID: config.ID,
		Type:          config.Type,
		Title:         config.Name,
		Content:       content,
		Status:        "sent",
		AckStatus:     "pending",
	}
	if successCount == 0 && config.Channels != "" {
		history.Status = "failed"
		history.ErrorMsg = "All channels failed"
	}
	h.historyRepo.Create(ctx, history)
}

func (h *AlertHandler) sendToChannels(config *models.AlertConfig, content string) int {
	type ChannelConfig struct {
		Type      string `json:"type"`
		URL       string `json:"url"`
		Secret    string `json:"secret"`
		ReceiveID string `json:"receive_id"`
	}
	var channels []ChannelConfig
	if config.Channels != "" {
		_ = json.Unmarshal([]byte(config.Channels), &channels)
	}

	successCount := 0
	for _, ch := range channels {
		var err error
		switch ch.Type {
		case "webhook", "feishu":
			err = sendFeishuWebhook(ch.URL, content)
		case "dingtalk":
			err = sendDingTalkWebhook(ch.URL, ch.Secret, content)
		case "email":
			err = sendEmail(ch.URL, ch.Secret, config.Name, content)
		default:
			alertLog.Warn("Unknown channel type: %s", ch.Type)
			continue
		}

		if err == nil {
			successCount++
		} else {
			alertLog.Error("Failed to send to %s: %v", ch.Type, err)
		}
	}
	return successCount
}
