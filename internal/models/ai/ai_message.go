package ai

import (
	"database/sql/driver"
	"encoding/json"
	"time"

	"gorm.io/gorm"
)

// MessageRole 消息角色
type MessageRole string

const (
	RoleUser      MessageRole = "user"
	RoleAssistant MessageRole = "assistant"
	RoleSystem    MessageRole = "system"
	RoleTool      MessageRole = "tool"
)

// MessageStatus 消息状态
type MessageStatus string

const (
	StatusPending   MessageStatus = "pending"
	StatusStreaming MessageStatus = "streaming"
	StatusComplete  MessageStatus = "complete"
	StatusError     MessageStatus = "error"
)

// ToolCall 工具调用信息
type ToolCall struct {
	ID       string       `json:"id"`
	Type     string       `json:"type"` // function
	Function FunctionCall `json:"function"`
}

// FunctionCall 函数调用信息
type FunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"` // JSON string
}

// ToolCalls 工具调用列表
type ToolCalls []ToolCall

// Scan 实现 sql.Scanner 接口
func (t *ToolCalls) Scan(value any) error {
	if value == nil {
		*t = nil
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(bytes, t)
}

// Value 实现 driver.Valuer 接口
func (t ToolCalls) Value() (driver.Value, error) {
	if t == nil {
		return nil, nil
	}
	return json.Marshal(t)
}

// AIMessage AI消息模型
type AIMessage struct {
	ID             string        `gorm:"primaryKey;size:36" json:"id"` // UUID
	CreatedAt      time.Time     `gorm:"index" json:"created_at"`
	ConversationID string        `gorm:"size:36;not null;index" json:"conversation_id"` // 会话ID
	Role           MessageRole   `gorm:"size:20;not null;index" json:"role"`            // 角色
	Content        string        `gorm:"type:text;not null" json:"content"`             // 消息内容
	ToolCalls      ToolCalls     `gorm:"type:json" json:"tool_calls,omitempty"`         // 工具调用
	ToolCallID     string        `gorm:"size:100" json:"tool_call_id,omitempty"`        // 工具调用ID
	TokenCount     int           `gorm:"default:0" json:"token_count"`                  // Token数量
	Status         MessageStatus `gorm:"size:20;default:'complete'" json:"status"`      // 状态
	ErrorMsg       string        `gorm:"type:text" json:"error_msg,omitempty"`          // 错误信息
	// 用户反馈字段
	FeedbackRating  string     `gorm:"size:20" json:"feedback_rating,omitempty"`    // like/dislike
	FeedbackComment string     `gorm:"type:text" json:"feedback_comment,omitempty"` // 反馈评论
	FeedbackAt      *time.Time `json:"feedback_at,omitempty"`                       // 反馈时间（指针类型避免零值问题）
}

// TableName 指定表名
func (AIMessage) TableName() string {
	return "ai_messages"
}

// BeforeCreate 创建前钩子
func (m *AIMessage) BeforeCreate(tx *gorm.DB) error {
	if m.ID == "" {
		m.ID = generateUUID()
	}
	if m.Status == "" {
		m.Status = StatusComplete
	}
	return nil
}

// ActionButton AI建议的操作按钮
type ActionButton struct {
	ID              string         `json:"id"`
	Label           string         `json:"label"`
	Action          string         `json:"action"`
	Params          map[string]any `json:"params"`
	ConfirmRequired bool           `json:"confirm_required"`
}
