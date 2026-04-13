// Package feishu 飞书客户端封装
// 本文件包含用户相关的方法
package feishu

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// ========== 用户管理 ==========

// SearchUser 搜索用户
// 优先使用 user_access_token（支持按姓名搜索），降级使用 tenant_access_token
func (c *Client) SearchUser(ctx context.Context, query string) ([]map[string]any, error) {
	c.logger.Debug("Searching user with query: %s", query)

	if c.HasUserToken() {
		token, err := c.GetUserAccessToken(ctx)
		if err == nil && token != "" {
			return c.searchUserWithToken(ctx, query, token)
		}
		c.logger.Warn("Failed to get user token, falling back to batch_get_id: %v", err)
	}

	return c.searchUserByEmailOrMobile(ctx, query)
}

// searchUserWithToken 使用 user_access_token 搜索用户
func (c *Client) searchUserWithToken(ctx context.Context, query, token string) ([]map[string]any, error) {
	searchURL := "https://open.feishu.cn/open-apis/search/v1/user?user_id_type=open_id&page_size=20"

	payload := map[string]any{
		"query": query,
	}

	payloadData, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal search payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", searchURL, bytes.NewBuffer(payloadData))
	if err != nil {
		return nil, fmt.Errorf("failed to create search request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send search request: %w", err)
	}
	defer resp.Body.Close()

	var response map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode search response: %w", err)
	}

	if code, ok := response["code"].(float64); ok && code != 0 {
		msg, _ := response["msg"].(string)
		c.logger.Error("Search API error: code=%v, msg=%s", code, msg)
		return nil, fmt.Errorf("搜索失败: %s", msg)
	}

	data, ok := response["data"].(map[string]any)
	if !ok {
		return []map[string]any{}, nil
	}

	users, ok := data["users"].([]any)
	if !ok {
		return []map[string]any{}, nil
	}

	result := make([]map[string]any, 0, len(users))
	for _, u := range users {
		if user, ok := u.(map[string]any); ok {
			result = append(result, user)
		}
	}

	c.logger.Info("Found %d users for query: %s (using user token)", len(result), query)
	return result, nil
}

// searchUserByEmailOrMobile 通过邮箱或手机号搜索用户
func (c *Client) searchUserByEmailOrMobile(ctx context.Context, query string) ([]map[string]any, error) {
	token, err := c.GetTenantAccessToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get tenant access token: %w", err)
	}

	searchURL := "https://open.feishu.cn/open-apis/contact/v3/users/batch_get_id?user_id_type=open_id"

	payload := map[string]any{
		"emails":  []string{query},
		"mobiles": []string{query},
	}

	payloadData, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal search payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", searchURL, bytes.NewBuffer(payloadData))
	if err != nil {
		return nil, fmt.Errorf("failed to create search request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send search request: %w", err)
	}
	defer resp.Body.Close()

	var response map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode search response: %w", err)
	}

	if code, ok := response["code"].(float64); ok && code != 0 {
		msg, _ := response["msg"].(string)
		c.logger.Error("Search API error: code=%v, msg=%s", code, msg)
		return nil, fmt.Errorf("搜索失败: %s (请输入邮箱或手机号，或配置 user_token 以支持姓名搜索)", msg)
	}

	data, ok := response["data"].(map[string]any)
	if !ok {
		return []map[string]any{}, nil
	}

	userList, ok := data["user_list"].([]any)
	if !ok {
		return []map[string]any{}, nil
	}

	result := make([]map[string]any, 0, len(userList))
	for _, u := range userList {
		if user, ok := u.(map[string]any); ok {
			if _, hasID := user["user_id"]; hasID {
				result = append(result, user)
			}
		}
	}

	c.logger.Info("Found %d users for query: %s (using tenant token)", len(result), query)
	return result, nil
}

// GetUserByID 根据用户ID获取用户信息
func (c *Client) GetUserByID(ctx context.Context, userID, userIDType string) (map[string]any, error) {
	c.logger.Debug("Getting user info for: %s (type: %s)", userID, userIDType)

	token, err := c.GetTenantAccessToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get tenant access token: %w", err)
	}

	url := fmt.Sprintf("https://open.feishu.cn/open-apis/contact/v3/users/%s?user_id_type=%s", userID, userIDType)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	var response map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if code, ok := response["code"].(float64); ok && code != 0 {
		msg, _ := response["msg"].(string)
		return nil, fmt.Errorf("API error: code=%v, msg=%s", code, msg)
	}

	data, ok := response["data"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("invalid response format")
	}

	user, ok := data["user"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("user not found in response")
	}

	return user, nil
}
