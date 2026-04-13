package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	apperrors "devops/pkg/errors"
)

// SendRequest 发送请求
type SendRequest struct {
	TemplateName string                 `json:"template_name" binding:"required"`
	Data         map[string]interface{} `json:"data" binding:"required"`
	WebhookURL   string                 `json:"webhook_url" binding:"required"` // 仅支持 Webhook URL
}

// SendToWebhook 发送模板消息到 Webhook
func (h *TemplateHandler) SendToWebhook(c *gin.Context) {
	var req SendRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": apperrors.ErrCodeInvalidParams, "message": err.Error()})
		return
	}

	// 1. 渲染模板
	content, err := h.svc.Render(c.Request.Context(), req.TemplateName, req.Data)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "Render failed: " + err.Error()})
		return
	}

	// 2. 构造飞书消息体
	// 注意：Webhook 接口需要的格式可能与 Content 渲染出的格式略有不同
	// 渲染出的 Content 通常是卡片本身的 JSON (header, body 等)
	// Webhook 需要包装一层 {"msg_type": "interactive", "card": ...}
	// 或者如果 Content 已经是包装好的，就直接发
	
	// 假设模板渲染出来的是 Card 对象 (header, elements 等)
	var cardObj interface{}
	if err := json.Unmarshal([]byte(content), &cardObj); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "Invalid template content JSON"})
		return
	}

	payload := map[string]interface{}{
		"msg_type": "interactive",
		"card":     cardObj,
	}
	
	payloadBytes, _ := json.Marshal(payload)

	// 3. 发送 HTTP 请求
	httpReq, err := http.NewRequestWithContext(c.Request.Context(), "POST", req.WebhookURL, bytes.NewBuffer(payloadBytes))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "Failed to create request: " + err.Error()})
		return
	}
	httpReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(httpReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "Failed to send webhook: " + err.Error()})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": fmt.Sprintf("Webhook returned status: %d", resp.StatusCode)})
		return
	}
	
	// 解析响应看是否有错误码
	var respData map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&respData); err == nil {
		if code, ok := respData["code"].(float64); ok && code != 0 {
			msg, _ := respData["msg"].(string)
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": fmt.Sprintf("Feishu error: %v %s", code, msg)})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    apperrors.Success,
		"message": "success",
		"data": gin.H{
			"content": content,
		},
	})
}
