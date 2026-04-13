package wechatwork

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"devops/pkg/logger"
)

// Client 企业微信客户端
type Client struct {
	corpID        string
	agentID       int64
	secret        string
	logger        *logger.Logger
	httpClient    *http.Client
	accessToken   string
	tokenExpireAt time.Time
	mu            sync.RWMutex
}

// NewClient 创建企业微信客户端
func NewClient(corpID string, agentID int64, secret string) *Client {
	return &Client{
		corpID:  corpID,
		agentID: agentID,
		secret:  secret,
		logger:  logger.NewLogger("INFO"),
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// GetAccessToken 获取访问令牌
func (c *Client) GetAccessToken(ctx context.Context) (string, error) {
	c.mu.RLock()
	if c.accessToken != "" && time.Now().Before(c.tokenExpireAt) {
		token := c.accessToken
		c.mu.RUnlock()
		return token, nil
	}
	c.mu.RUnlock()

	c.mu.Lock()
	defer c.mu.Unlock()

	// 双重检查
	if c.accessToken != "" && time.Now().Before(c.tokenExpireAt) {
		return c.accessToken, nil
	}

	tokenURL := fmt.Sprintf("https://qyapi.weixin.qq.com/cgi-bin/gettoken?corpid=%s&corpsecret=%s", c.corpID, c.secret)

	req, err := http.NewRequestWithContext(ctx, "GET", tokenURL, nil)
	if err != nil {
		return "", fmt.Errorf("create request failed: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	var result TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decode response failed: %w", err)
	}

	if result.ErrCode != 0 {
		return "", fmt.Errorf("API error: %d - %s", result.ErrCode, result.ErrMsg)
	}

	c.accessToken = result.AccessToken
	c.tokenExpireAt = time.Now().Add(time.Duration(result.ExpiresIn-300) * time.Second)

	c.logger.Info("WechatWork access token obtained, expires at: %v", c.tokenExpireAt)
	return c.accessToken, nil
}

// SendMessage 发送应用消息
func (c *Client) SendMessage(ctx context.Context, msg *AppMessage) error {
	token, err := c.GetAccessToken(ctx)
	if err != nil {
		return err
	}

	sendURL := fmt.Sprintf("https://qyapi.weixin.qq.com/cgi-bin/message/send?access_token=%s", token)

	msg.AgentID = c.agentID

	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal message failed: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", sendURL, bytes.NewBuffer(data))
	if err != nil {
		return fmt.Errorf("create request failed: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("decode response failed: %w", err)
	}

	if errcode, ok := result["errcode"].(float64); ok && errcode != 0 {
		errmsg, _ := result["errmsg"].(string)
		return fmt.Errorf("API error: %v - %s", errcode, errmsg)
	}

	c.logger.Info("App message sent successfully")
	return nil
}

// SendWebhookMessage 发送Webhook消息
func (c *Client) SendWebhookMessage(ctx context.Context, webhookURL string, msg *WebhookMessage) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal message failed: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", webhookURL, bytes.NewBuffer(data))
	if err != nil {
		return fmt.Errorf("create request failed: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("decode response failed: %w", err)
	}

	if errcode, ok := result["errcode"].(float64); ok && errcode != 0 {
		errmsg, _ := result["errmsg"].(string)
		return fmt.Errorf("webhook error: %v - %s", errcode, errmsg)
	}

	c.logger.Info("Webhook message sent successfully")
	return nil
}

// SearchUser 搜索用户
func (c *Client) SearchUser(ctx context.Context, query string) ([]UserInfo, error) {
	token, err := c.GetAccessToken(ctx)
	if err != nil {
		return nil, err
	}

	searchURL := fmt.Sprintf("https://qyapi.weixin.qq.com/cgi-bin/user/simplelist?access_token=%s&department_id=1&fetch_child=1", token)

	req, err := http.NewRequestWithContext(ctx, "GET", searchURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request failed: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response failed: %w", err)
	}

	if errcode, ok := result["errcode"].(float64); ok && errcode != 0 {
		errmsg, _ := result["errmsg"].(string)
		return nil, fmt.Errorf("API error: %v - %s", errcode, errmsg)
	}

	var users []UserInfo
	if userList, ok := result["userlist"].([]any); ok {
		for _, item := range userList {
			if u, ok := item.(map[string]any); ok {
				name := getString(u, "name")
				if query == "" || containsIgnoreCase(name, query) {
					user := UserInfo{
						UserID: getString(u, "userid"),
						Name:   name,
					}
					users = append(users, user)
				}
			}
		}
	}

	return users, nil
}

func getString(m map[string]any, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

func containsIgnoreCase(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0)
}

// GetLogger 获取日志记录器
func (c *Client) GetLogger() *logger.Logger {
	return c.logger
}
