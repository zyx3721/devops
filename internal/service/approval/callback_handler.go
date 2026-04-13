package approval

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"devops/pkg/logger"
)

var callbackLog = logger.L().WithField("module", "approval_callback")

// CallbackHandler 审批回调处理器
type CallbackHandler struct {
	nodeExecutor     *NodeExecutor
	feishuSecret     string // 飞书应用签名密钥
	dingtalkSecret   string // 钉钉机器人签名密钥
	wecomToken       string // 企业微信 Token
	wecomEncodingKey string // 企业微信 EncodingAESKey
}

// NewCallbackHandler 创建回调处理器
func NewCallbackHandler(nodeExecutor *NodeExecutor) *CallbackHandler {
	return &CallbackHandler{
		nodeExecutor: nodeExecutor,
	}
}

// SetFeishuSecret 设置飞书签名密钥
func (h *CallbackHandler) SetFeishuSecret(secret string) {
	h.feishuSecret = secret
}

// SetDingTalkSecret 设置钉钉签名密钥
func (h *CallbackHandler) SetDingTalkSecret(secret string) {
	h.dingtalkSecret = secret
}

// SetWeComConfig 设置企业微信配置
func (h *CallbackHandler) SetWeComConfig(token, encodingKey string) {
	h.wecomToken = token
	h.wecomEncodingKey = encodingKey
}

// HandleApprove 处理审批通过回调
func (h *CallbackHandler) HandleApprove(ctx context.Context, nodeInstanceID uint, userID uint, userName string, comment string) error {
	if h.nodeExecutor == nil {
		return fmt.Errorf("node executor not initialized")
	}
	return h.nodeExecutor.Approve(ctx, nodeInstanceID, userID, userName, comment)
}

// HandleReject 处理审批拒绝回调
func (h *CallbackHandler) HandleReject(ctx context.Context, nodeInstanceID uint, userID uint, userName string, comment string) error {
	if h.nodeExecutor == nil {
		return fmt.Errorf("node executor not initialized")
	}
	return h.nodeExecutor.Reject(ctx, nodeInstanceID, userID, userName, comment)
}

// FeishuCardCallback 飞书卡片回调数据
type FeishuCardCallback struct {
	Challenge string `json:"challenge"` // URL 验证时的 challenge
	Token     string `json:"token"`
	Type      string `json:"type"`
	Event     struct {
		Operator struct {
			OpenID string `json:"open_id"`
			UserID string `json:"user_id"`
		} `json:"operator"`
		Action struct {
			Value map[string]any `json:"value"`
			Tag   string         `json:"tag"`
		} `json:"action"`
	} `json:"event"`
}

// HandleFeishuCallback 处理飞书卡片回调
func (h *CallbackHandler) HandleFeishuCallback(ctx context.Context, body []byte, signature string, timestamp string) (map[string]any, error) {
	// 验证签名
	if h.feishuSecret != "" {
		if !h.verifyFeishuSignature(body, signature, timestamp) {
			return nil, fmt.Errorf("invalid signature")
		}
	}

	var callback FeishuCardCallback
	if err := json.Unmarshal(body, &callback); err != nil {
		return nil, fmt.Errorf("parse callback failed: %w", err)
	}

	// URL 验证
	if callback.Challenge != "" {
		return map[string]any{"challenge": callback.Challenge}, nil
	}

	// 解析操作
	action, ok := callback.Event.Action.Value["action"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid action")
	}

	nodeInstanceIDFloat, ok := callback.Event.Action.Value["node_instance_id"].(float64)
	if !ok {
		return nil, fmt.Errorf("invalid node_instance_id")
	}
	nodeInstanceID := uint(nodeInstanceIDFloat)

	// 获取用户信息（这里简化处理，实际应该通过飞书 API 获取用户详情）
	userID := uint(0)
	userName := callback.Event.Operator.UserID
	if userName == "" {
		userName = callback.Event.Operator.OpenID
	}

	callbackLog.WithField("action", action).WithField("node_instance_id", nodeInstanceID).
		WithField("user", userName).Info("处理飞书卡片回调")

	switch action {
	case "approve":
		if err := h.HandleApprove(ctx, nodeInstanceID, userID, userName, "通过飞书卡片审批"); err != nil {
			return nil, err
		}
		return map[string]any{
			"toast": map[string]any{
				"type":    "success",
				"content": "审批通过成功",
			},
		}, nil
	case "reject":
		if err := h.HandleReject(ctx, nodeInstanceID, userID, userName, "通过飞书卡片拒绝"); err != nil {
			return nil, err
		}
		return map[string]any{
			"toast": map[string]any{
				"type":    "info",
				"content": "已拒绝该审批",
			},
		}, nil
	default:
		return nil, fmt.Errorf("unknown action: %s", action)
	}
}

// verifyFeishuSignature 验证飞书签名
func (h *CallbackHandler) verifyFeishuSignature(body []byte, signature string, timestamp string) bool {
	stringToSign := timestamp + "\n" + h.feishuSecret + "\n" + string(body)
	mac := hmac.New(sha256.New, []byte(h.feishuSecret))
	mac.Write([]byte(stringToSign))
	expectedSignature := base64.StdEncoding.EncodeToString(mac.Sum(nil))
	return signature == expectedSignature
}

// DingTalkCallback 钉钉回调数据
type DingTalkCallback struct {
	MsgType string `json:"msgtype"`
	Text    struct {
		Content string `json:"content"`
	} `json:"text"`
	// ActionCard 回调
	ActionCardCallback struct {
		ActionCardValue string `json:"actionCardValue"`
	} `json:"actionCardCallback"`
	// 发送者信息
	SenderID       string `json:"senderId"`
	SenderNick     string `json:"senderNick"`
	ConversationID string `json:"conversationId"`
}

// HandleDingTalkCallback 处理钉钉回调
func (h *CallbackHandler) HandleDingTalkCallback(ctx context.Context, body []byte, signature string, timestamp string) (map[string]any, error) {
	// 验证签名
	if h.dingtalkSecret != "" {
		if !h.verifyDingTalkSignature(timestamp, signature) {
			return nil, fmt.Errorf("invalid signature")
		}
	}

	var callback DingTalkCallback
	if err := json.Unmarshal(body, &callback); err != nil {
		return nil, fmt.Errorf("parse callback failed: %w", err)
	}

	// 解析 ActionCard 回调值
	// 格式: action=approve&node_instance_id=123
	params := parseQueryString(callback.ActionCardCallback.ActionCardValue)
	action := params["action"]
	nodeInstanceIDStr := params["node_instance_id"]

	if action == "" || nodeInstanceIDStr == "" {
		return nil, fmt.Errorf("invalid callback params")
	}

	nodeInstanceID, err := strconv.ParseUint(nodeInstanceIDStr, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid node_instance_id: %w", err)
	}

	userName := callback.SenderNick
	if userName == "" {
		userName = callback.SenderID
	}

	callbackLog.WithField("action", action).WithField("node_instance_id", nodeInstanceID).
		WithField("user", userName).Info("处理钉钉回调")

	switch action {
	case "approve":
		if err := h.HandleApprove(ctx, uint(nodeInstanceID), 0, userName, "通过钉钉卡片审批"); err != nil {
			return nil, err
		}
		return map[string]any{
			"msgtype": "text",
			"text": map[string]any{
				"content": "✅ 审批通过成功",
			},
		}, nil
	case "reject":
		if err := h.HandleReject(ctx, uint(nodeInstanceID), 0, userName, "通过钉钉卡片拒绝"); err != nil {
			return nil, err
		}
		return map[string]any{
			"msgtype": "text",
			"text": map[string]any{
				"content": "❌ 已拒绝该审批",
			},
		}, nil
	default:
		return nil, fmt.Errorf("unknown action: %s", action)
	}
}

// verifyDingTalkSignature 验证钉钉签名
func (h *CallbackHandler) verifyDingTalkSignature(timestamp string, signature string) bool {
	stringToSign := timestamp + "\n" + h.dingtalkSecret
	mac := hmac.New(sha256.New, []byte(h.dingtalkSecret))
	mac.Write([]byte(stringToSign))
	expectedSignature := base64.StdEncoding.EncodeToString(mac.Sum(nil))
	return signature == expectedSignature
}

// WeComCallback 企业微信回调数据
type WeComCallback struct {
	MsgType string `json:"MsgType"`
	Event   string `json:"Event"`
	// 模板卡片回调
	TaskID       string `json:"TaskId"`
	CardType     string `json:"CardType"`
	ResponseCode string `json:"ResponseCode"`
	SelectedItem struct {
		QuestionKey string `json:"QuestionKey"`
		OptionIds   string `json:"OptionIds"`
	} `json:"SelectedItem"`
	// 按钮回调
	EventKey string `json:"EventKey"` // 格式: approve_123 或 reject_123
	// 用户信息
	FromUserName string `json:"FromUserName"`
}

// HandleWeComCallback 处理企业微信回调
func (h *CallbackHandler) HandleWeComCallback(ctx context.Context, body []byte, msgSignature string, timestamp string, nonce string) (map[string]any, error) {
	// 验证签名（简化处理，实际需要完整的企业微信签名验证）
	if h.wecomToken != "" {
		if !h.verifyWeComSignature(msgSignature, timestamp, nonce, string(body)) {
			return nil, fmt.Errorf("invalid signature")
		}
	}

	var callback WeComCallback
	if err := json.Unmarshal(body, &callback); err != nil {
		return nil, fmt.Errorf("parse callback failed: %w", err)
	}

	// 解析按钮回调
	// EventKey 格式: approve_123 或 reject_123
	parts := strings.SplitN(callback.EventKey, "_", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid event key format")
	}

	action := parts[0]
	nodeInstanceID, err := strconv.ParseUint(parts[1], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid node_instance_id: %w", err)
	}

	userName := callback.FromUserName

	callbackLog.WithField("action", action).WithField("node_instance_id", nodeInstanceID).
		WithField("user", userName).Info("处理企业微信回调")

	switch action {
	case "approve":
		if err := h.HandleApprove(ctx, uint(nodeInstanceID), 0, userName, "通过企业微信卡片审批"); err != nil {
			return nil, err
		}
		return map[string]any{
			"response_code": "success",
			"replace_card": map[string]any{
				"card_type": "text_notice",
				"main_title": map[string]any{
					"title": "✅ 审批通过成功",
				},
			},
		}, nil
	case "reject":
		if err := h.HandleReject(ctx, uint(nodeInstanceID), 0, userName, "通过企业微信卡片拒绝"); err != nil {
			return nil, err
		}
		return map[string]any{
			"response_code": "success",
			"replace_card": map[string]any{
				"card_type": "text_notice",
				"main_title": map[string]any{
					"title": "❌ 已拒绝该审批",
				},
			},
		}, nil
	default:
		return nil, fmt.Errorf("unknown action: %s", action)
	}
}

// verifyWeComSignature 验证企业微信签名
func (h *CallbackHandler) verifyWeComSignature(msgSignature string, timestamp string, nonce string, body string) bool {
	// 简化实现，实际需要按企业微信文档进行完整验证
	// 1. 将 token、timestamp、nonce、body 按字典序排序
	// 2. 拼接后进行 SHA1 签名
	// 3. 与 msgSignature 比较
	strs := []string{h.wecomToken, timestamp, nonce, body}
	// 排序并拼接
	sortedStr := strings.Join(strs, "")
	hash := sha256.Sum256([]byte(sortedStr))
	expectedSignature := fmt.Sprintf("%x", hash)
	return msgSignature == expectedSignature
}

// parseQueryString 解析查询字符串
func parseQueryString(query string) map[string]string {
	result := make(map[string]string)
	pairs := strings.Split(query, "&")
	for _, pair := range pairs {
		kv := strings.SplitN(pair, "=", 2)
		if len(kv) == 2 {
			result[kv[0]] = kv[1]
		}
	}
	return result
}

// GenerateDingTalkSignature 生成钉钉签名（用于发送消息时）
func GenerateDingTalkSignature(secret string) (string, string) {
	timestamp := fmt.Sprintf("%d", time.Now().UnixMilli())
	stringToSign := timestamp + "\n" + secret
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(stringToSign))
	signature := base64.StdEncoding.EncodeToString(mac.Sum(nil))
	return timestamp, signature
}
