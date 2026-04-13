// Package feishu 飞书客户端封装
// 本文件包含消息发送相关的方法
package feishu

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// ========== 消息发送 ==========

// CreateCard 创建飞书卡片
// 返回卡片ID（本地生成，用于追踪）
func (c *Client) CreateCard(ctx context.Context, title, content string) (string, error) {
	c.logger.Debug("Creating card with title: %s", title)

	cardJSON := fmt.Sprintf(`{
		"schema":"2.0",
		"header":{
			"title":{
				"content":"%s",
				"tag":"plain_text"
			}
		},
		"body":{
			"elements":[
				{
					"tag":"markdown",
					"content":"%s"
				}
			]
		}
	}`, title, content)

	c.logger.Debug("Card content: %s", cardJSON)

	cardID := fmt.Sprintf("card_%s", generateUUID())

	c.logger.Info("Card created successfully, card_id: %s", cardID)
	return cardID, nil
}

// SendMessage 发送消息
// 支持 text 和 interactive 两种消息类型
func (c *Client) SendMessage(ctx context.Context, receiveID, receiveIdType, msgType, content string) error {
	c.logger.Debug("Sending message to %s, type: %s", receiveID, msgType)

	token, err := c.GetTenantAccessToken(ctx)
	if err != nil {
		c.logger.Error("Failed to get tenant access token: %v", err)
		return fmt.Errorf("failed to get tenant access token: %w", err)
	}

	sendURL := fmt.Sprintf("https://open.feishu.cn/open-apis/im/v1/messages?receive_id_type=%s", receiveIdType)

	messagePayload := map[string]any{
		"receive_id": receiveID,
		"msg_type":   msgType,
	}

	switch msgType {
	case "text":
		messagePayload["content"] = content
	case "interactive":
		messagePayload["content"] = content
	default:
		c.logger.Error("Unsupported message type: %s", msgType)
		return fmt.Errorf("unsupported message type: %s", msgType)
	}

	payloadData, err := json.Marshal(messagePayload)
	if err != nil {
		c.logger.Error("Failed to marshal message payload: %v", err)
		return fmt.Errorf("failed to marshal message payload: %w", err)
	}

	c.logger.Debug("Message payload: %s", string(payloadData))

	req, err := http.NewRequestWithContext(ctx, "POST", sendURL, bytes.NewBuffer(payloadData))
	if err != nil {
		c.logger.Error("Failed to create message request: %v", err)
		return fmt.Errorf("failed to create message request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.Error("Failed to send message request: %v", err)
		return fmt.Errorf("failed to send message request: %w", err)
	}
	defer resp.Body.Close()

	var response map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		c.logger.Error("Failed to decode message response: %v", err)
		return fmt.Errorf("failed to decode message response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		c.logger.Error("Message send failed: status=%d, response=%v", resp.StatusCode, response)
		return fmt.Errorf("message send failed: status=%d, response=%v", resp.StatusCode, response)
	}

	if code, ok := response["code"].(float64); ok && code != 0 {
		msg, _ := response["msg"].(string)
		c.logger.Error("Message API error: code=%v, msg=%s", code, msg)
		return fmt.Errorf("message API error: code=%v, msg=%s", code, msg)
	}

	data, ok := response["data"].(map[string]any)
	if !ok {
		c.logger.Error("Invalid response data format: %v", response)
		return fmt.Errorf("invalid response data format")
	}

	messageID, ok := data["message_id"].(string)
	if !ok {
		c.logger.Error("Message ID not found in response: %v", data)
		return fmt.Errorf("message ID not found in response")
	}

	c.logger.Info("Message sent successfully, message_id: %s", messageID)

	return nil
}
