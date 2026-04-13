package feishu

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"os"
)

// Sender 消息发送接口
type Sender interface {
	Send(ctx context.Context, receiveID, receiveIDType, msgType, content string) error
}

// APISender 使用飞书API发送消息
type APISender struct {
	client *Client
}

// NewAPISender 创建API发送器
func NewAPISender(client *Client) *APISender {
	return &APISender{client: client}
}

// Send 发送消息
func (s *APISender) Send(ctx context.Context, receiveID, receiveIDType, msgType, content string) error {
	return s.client.SendMessage(ctx, receiveID, receiveIDType, msgType, content)
}

// WebhookSender 使用Webhook发送消息
type WebhookSender struct {
	httpClient *http.Client
	url        string
}

// NewWebhookSender 创建Webhook发送器
func NewWebhookSender() *WebhookSender {
	return &WebhookSender{
		httpClient: &http.Client{},
		url:        os.Getenv("FEISHU_WEBHOOK_URL"),
	}
}

// Send 发送消息
func (s *WebhookSender) Send(ctx context.Context, receiveID, receiveIDType, msgType, content string) error {
	if s.url == "" {
		return nil
	}
	payload := map[string]any{"msg_type": msgType}
	if msgType == "text" {
		var m map[string]string
		_ = json.Unmarshal([]byte(content), &m)
		payload["content"] = m
	} else {
		var card map[string]any
		_ = json.Unmarshal([]byte(content), &card)
		payload["card"] = card
	}
	data, _ := json.Marshal(payload)
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, s.url, bytes.NewBuffer(data))
	req.Header.Set("Content-Type", "application/json")
	_, err := s.httpClient.Do(req)
	if err != nil {
		return err
	}
	return nil
}
