// Package feishu 飞书客户端封装
// 本文件包含群聊相关的方法
package feishu

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// ========== 群聊管理 ==========

// CreateChat 创建群聊
// 返回创建的群聊 chat_id
func (c *Client) CreateChat(ctx context.Context, name, description string, userIDs []string, userIDType string) (string, error) {
	c.logger.Debug("Creating chat: %s with %d users", name, len(userIDs))

	token, err := c.GetTenantAccessToken(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get tenant access token: %w", err)
	}

	url := fmt.Sprintf("https://open.feishu.cn/open-apis/im/v1/chats?user_id_type=%s", userIDType)

	payload := map[string]any{
		"name":         name,
		"description":  description,
		"user_id_list": userIDs,
	}

	payloadData, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(payloadData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	var response map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if code, ok := response["code"].(float64); ok && code != 0 {
		msg, _ := response["msg"].(string)
		return "", fmt.Errorf("API error: code=%v, msg=%s", code, msg)
	}

	data, ok := response["data"].(map[string]any)
	if !ok {
		return "", fmt.Errorf("invalid response format")
	}

	chatID, ok := data["chat_id"].(string)
	if !ok {
		return "", fmt.Errorf("chat_id not found in response")
	}

	c.logger.Info("Chat created successfully, chat_id: %s", chatID)
	return chatID, nil
}

// AddChatMembers 添加群成员
func (c *Client) AddChatMembers(ctx context.Context, chatID string, userIDs []string, userIDType string) error {
	c.logger.Debug("Adding %d members to chat: %s", len(userIDs), chatID)

	token, err := c.GetTenantAccessToken(ctx)
	if err != nil {
		return fmt.Errorf("failed to get tenant access token: %w", err)
	}

	url := fmt.Sprintf("https://open.feishu.cn/open-apis/im/v1/chats/%s/members?member_id_type=%s", chatID, userIDType)

	payload := map[string]any{
		"id_list": userIDs,
	}

	payloadData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(payloadData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	var response map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	if code, ok := response["code"].(float64); ok && code != 0 {
		msg, _ := response["msg"].(string)
		return fmt.Errorf("API error: code=%v, msg=%s", code, msg)
	}

	c.logger.Info("Members added successfully to chat: %s", chatID)
	return nil
}

// GetChatList 获取群列表
// 返回群列表和下一页的 page_token
func (c *Client) GetChatList(ctx context.Context, pageToken string, pageSize int) ([]map[string]any, string, error) {
	c.logger.Debug("Getting chat list")

	token, err := c.GetTenantAccessToken(ctx)
	if err != nil {
		c.logger.Debug("Failed to get tenant access token: %v", err)
		return nil, "", fmt.Errorf("feishu not configured or token unavailable: %w", err)
	}

	url := fmt.Sprintf("https://open.feishu.cn/open-apis/im/v1/chats?page_size=%d", pageSize)
	if pageToken != "" {
		url += "&page_token=" + pageToken
	}

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	var response map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, "", fmt.Errorf("failed to decode response: %w", err)
	}

	if code, ok := response["code"].(float64); ok && code != 0 {
		msg, _ := response["msg"].(string)
		return nil, "", fmt.Errorf("API error: code=%v, msg=%s", code, msg)
	}

	data, ok := response["data"].(map[string]any)
	if !ok {
		return []map[string]any{}, "", nil
	}

	items, ok := data["items"].([]any)
	if !ok {
		return []map[string]any{}, "", nil
	}

	result := make([]map[string]any, 0, len(items))
	for _, item := range items {
		if chat, ok := item.(map[string]any); ok {
			result = append(result, chat)
		}
	}

	nextPageToken, _ := data["page_token"].(string)

	return result, nextPageToken, nil
}
