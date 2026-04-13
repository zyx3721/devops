// Package feishu 飞书客户端封装
// 本文件包含令牌管理相关的方法
package feishu

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"devops/internal/models"
)

// ========== 租户令牌管理 ==========

// GetTenantAccessToken 获取飞书应用的 tenant_access_token
// 如果缓存的令牌有效则直接返回，否则重新获取
func (c *Client) GetTenantAccessToken(ctx context.Context) (string, error) {
	// 验证配置是否完整
	if c.appID == "" || c.appSecret == "" {
		c.logger.Debug("Feishu app_id or app_secret not configured, skipping token fetch")
		return "", fmt.Errorf("feishu app_id or app_secret not configured")
	}

	c.mu.RLock()
	valid := c.tenantToken != "" && time.Now().Before(c.tokenExpireAt)
	token := c.tenantToken
	c.mu.RUnlock()
	if valid {
		c.logger.Debug("Using cached tenant access token")
		return token, nil
	}

	c.logger.Info("Fetching new tenant access token")

	tokenURL := "https://open.feishu.cn/open-apis/auth/v3/tenant_access_token/internal"

	payload := map[string]string{
		"app_id":     c.appID,
		"app_secret": c.appSecret,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		c.logger.Error("Failed to marshal token request: %v", err)
		return "", fmt.Errorf("failed to marshal token request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", tokenURL, bytes.NewBuffer(data))
	if err != nil {
		c.logger.Error("Failed to create token request: %v", err)
		return "", fmt.Errorf("failed to create token request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.Error("Failed to send token request: %v", err)
		return "", fmt.Errorf("failed to send token request: %w", err)
	}
	defer resp.Body.Close()

	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		c.logger.Error("Failed to decode token response: %v", err)
		return "", fmt.Errorf("failed to decode token response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		c.logger.Error("Token request failed: status=%d, response=%v", resp.StatusCode, result)
		return "", fmt.Errorf("token request failed: status=%d, response=%v", resp.StatusCode, result)
	}

	if code, ok := result["code"].(float64); ok && code != 0 {
		msg, _ := result["msg"].(string)
		c.logger.Error("Token API error: code=%v, msg=%s", code, msg)
		return "", fmt.Errorf("token API error: code=%v, msg=%s", code, msg)
	}

	token, tokenOk := result["tenant_access_token"].(string)
	if !tokenOk {
		c.logger.Error("Invalid token format in response: %v", result)
		return "", fmt.Errorf("invalid token format in response")
	}

	expire, _ := result["expire"].(float64)
	c.mu.Lock()
	if expire > 0 {
		c.tokenExpireAt = time.Now().Add(time.Duration(expire-600) * time.Second)
	} else {
		c.tokenExpireAt = time.Now().Add(50 * time.Minute)
	}
	c.tenantToken = token
	c.mu.Unlock()
	c.logger.Info("Successfully obtained tenant access token, expires at: %v", c.tokenExpireAt)

	return token, nil
}

// getTenantTokenInternal 内部获取 tenant token（不加锁）
func (c *Client) getTenantTokenInternal(ctx context.Context) (string, error) {
	if c.tenantToken != "" && time.Now().Before(c.tokenExpireAt) {
		return c.tenantToken, nil
	}

	tokenURL := "https://open.feishu.cn/open-apis/auth/v3/tenant_access_token/internal"

	payload := map[string]string{
		"app_id":     c.appID,
		"app_secret": c.appSecret,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", tokenURL, bytes.NewBuffer(data))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	token, _ := result["tenant_access_token"].(string)
	expire, _ := result["expire"].(float64)

	if token != "" {
		c.tenantToken = token
		if expire > 0 {
			c.tokenExpireAt = time.Now().Add(time.Duration(expire-600) * time.Second)
		} else {
			c.tokenExpireAt = time.Now().Add(50 * time.Minute)
		}
	}

	return token, nil
}

// ========== 用户令牌管理 ==========

// SetUserToken 设置用户访问令牌（初始设置）
func (c *Client) SetUserToken(userToken, refreshToken string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.userToken = userToken
	c.refreshToken = refreshToken
	c.userTokenExpire = time.Now().Add(2 * time.Hour)
	c.logger.Info("User token set, will expire at: %v, has_refresh_token: %v", c.userTokenExpire, refreshToken != "")

	c.saveTokenToDBInternal()
}

// LoadTokenFromDB 从数据库加载 token
func (c *Client) LoadTokenFromDB() {
	if c.tokenRepo == nil {
		return
	}

	token, err := c.tokenRepo.GetByAppID(context.Background(), c.appID)
	if err != nil {
		c.logger.Debug("No saved user token found for app %s", c.appID)
		return
	}

	c.mu.Lock()
	c.userToken = token.AccessToken
	c.refreshToken = token.RefreshToken
	c.userTokenExpire = token.ExpiresAt
	c.mu.Unlock()

	c.logger.Info("Loaded user token from database, expires at: %v", token.ExpiresAt)
}

// saveTokenToDBInternal 保存 token 到数据库（内部方法，需要已持有锁）
func (c *Client) saveTokenToDBInternal() {
	if c.tokenRepo == nil {
		return
	}

	token := &models.FeishuUserToken{
		AppID:        c.appID,
		AccessToken:  c.userToken,
		RefreshToken: c.refreshToken,
		ExpiresAt:    c.userTokenExpire,
	}

	if err := c.tokenRepo.Save(context.Background(), token); err != nil {
		c.logger.Error("Failed to save user token to database: %v", err)
	} else {
		c.logger.Debug("User token saved to database")
	}
}

// StartTokenRefreshTask 启动 token 自动刷新任务
func (c *Client) StartTokenRefreshTask() {
	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()

		for range ticker.C {
			c.mu.RLock()
			hasRefresh := c.refreshToken != ""
			c.mu.RUnlock()

			if hasRefresh {
				c.logger.Info("Auto refreshing user token...")
				_, err := c.RefreshUserToken(context.Background())
				if err != nil {
					c.logger.Error("Auto refresh user token failed: %v", err)
				} else {
					c.logger.Info("Auto refresh user token success")
				}
			}
		}
	}()
	c.logger.Info("User token auto refresh task started (every 1 hour)")
}

// GetUserAccessToken 获取用户访问令牌
func (c *Client) GetUserAccessToken(ctx context.Context) (string, error) {
	c.mu.RLock()
	token := c.userToken
	valid := token != "" && time.Now().Before(c.userTokenExpire)
	hasRefresh := c.refreshToken != ""
	c.mu.RUnlock()

	if valid {
		return token, nil
	}

	if hasRefresh {
		return c.RefreshUserToken(ctx)
	}

	if token == "" {
		return "", fmt.Errorf("no user token available")
	}

	return "", fmt.Errorf("user token expired and no refresh token available")
}

// RefreshUserToken 刷新用户访问令牌
func (c *Client) RefreshUserToken(ctx context.Context) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.refreshToken == "" {
		return "", fmt.Errorf("no refresh token available, please set user token first")
	}

	c.logger.Info("Refreshing user access token")

	refreshURL := "https://open.feishu.cn/open-apis/authen/v2/oauth/token"

	payload := map[string]string{
		"grant_type":    "refresh_token",
		"refresh_token": c.refreshToken,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal refresh request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", refreshURL, bytes.NewBuffer(data))
	if err != nil {
		return "", fmt.Errorf("failed to create refresh request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	auth := c.appID + ":" + c.appSecret
	req.Header.Set("Authorization", "Basic "+base64Encode(auth))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send refresh request: %w", err)
	}
	defer resp.Body.Close()

	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode refresh response: %w", err)
	}

	if code, ok := result["code"].(float64); ok && code != 0 {
		msg, _ := result["msg"].(string)
		c.logger.Error("Refresh token failed: code=%v, msg=%s", code, msg)
		return "", fmt.Errorf("refresh token failed: %s", msg)
	}

	newToken, _ := result["access_token"].(string)
	newRefresh, _ := result["refresh_token"].(string)
	expire, _ := result["expires_in"].(float64)

	if newToken == "" {
		return "", fmt.Errorf("no access_token in refresh response")
	}

	c.userToken = newToken
	if newRefresh != "" {
		c.refreshToken = newRefresh
	}
	if expire > 0 {
		c.userTokenExpire = time.Now().Add(time.Duration(expire-300) * time.Second)
	} else {
		c.userTokenExpire = time.Now().Add(2 * time.Hour)
	}

	c.logger.Info("User token refreshed, expires at: %v", c.userTokenExpire)

	c.saveTokenToDBInternal()

	return c.userToken, nil
}

// base64Encode base64 编码
func base64Encode(s string) string {
	return base64.StdEncoding.EncodeToString([]byte(s))
}

// GetRefreshToken 获取当前的 refresh_token（用于保存）
func (c *Client) GetRefreshToken() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.refreshToken
}

// HasUserToken 检查是否有用户令牌
func (c *Client) HasUserToken() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.userToken != "" || c.refreshToken != ""
}
