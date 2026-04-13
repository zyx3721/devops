package dingtalk

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"time"

	"devops/pkg/logger"
)

// Client 钉钉客户端
type Client struct {
	appKey        string
	appSecret     string
	agentID       int64
	logger        *logger.Logger
	httpClient    *http.Client
	accessToken   string
	tokenExpireAt time.Time
	mu            sync.RWMutex
}

// NewClient 创建钉钉客户端
func NewClient(appKey, appSecret string, agentID int64) *Client {
	return &Client{
		appKey:    appKey,
		appSecret: appSecret,
		agentID:   agentID,
		logger:    logger.NewLogger("INFO"),
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

	if c.accessToken != "" && time.Now().Before(c.tokenExpireAt) {
		return c.accessToken, nil
	}

	tokenURL := fmt.Sprintf("https://oapi.dingtalk.com/gettoken?appkey=%s&appsecret=%s", c.appKey, c.appSecret)

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

	c.logger.Info("Dingtalk access token obtained, expires at: %v", c.tokenExpireAt)
	return c.accessToken, nil
}

// SendWorkMessage 发送工作通知消息
func (c *Client) SendWorkMessage(ctx context.Context, userIDList string, msgType string, content any) error {
	token, err := c.GetAccessToken(ctx)
	if err != nil {
		return err
	}

	sendURL := fmt.Sprintf("https://oapi.dingtalk.com/topapi/message/corpconversation/asyncsend_v2?access_token=%s", token)

	msg := map[string]any{
		"agent_id":    c.agentID,
		"userid_list": userIDList,
		"msg":         content,
	}

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

	c.logger.Info("Work message sent successfully")
	return nil
}

// SendWebhookMessage 发送Webhook消息
func (c *Client) SendWebhookMessage(ctx context.Context, webhookURL, secret string, msg *WebhookMessage) error {
	if secret != "" {
		timestamp := time.Now().UnixMilli()
		sign := c.sign(timestamp, secret)
		webhookURL = fmt.Sprintf("%s&timestamp=%d&sign=%s", webhookURL, timestamp, url.QueryEscape(sign))
	}

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

func (c *Client) sign(timestamp int64, secret string) string {
	stringToSign := fmt.Sprintf("%d\n%s", timestamp, secret)
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(stringToSign))
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

// SearchUser 搜索用户
func (c *Client) SearchUser(ctx context.Context, query string) ([]UserInfo, error) {
	token, err := c.GetAccessToken(ctx)
	if err != nil {
		return nil, err
	}

	searchURL := fmt.Sprintf("https://oapi.dingtalk.com/topapi/user/search?access_token=%s", token)

	payload := map[string]any{
		"query":  query,
		"offset": 0,
		"size":   20,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal request failed: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", searchURL, bytes.NewBuffer(data))
	if err != nil {
		return nil, fmt.Errorf("create request failed: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

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
	if resultData, ok := result["result"].(map[string]any); ok {
		if list, ok := resultData["list"].([]any); ok {
			for _, item := range list {
				if u, ok := item.(map[string]any); ok {
					user := UserInfo{
						UserID: getString(u, "userid"),
						Name:   getString(u, "name"),
						Mobile: getString(u, "mobile"),
						Email:  getString(u, "email"),
						Avatar: getString(u, "avatar"),
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

// GetLogger 获取日志记录器
func (c *Client) GetLogger() *logger.Logger {
	return c.logger
}
