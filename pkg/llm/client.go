// Package llm 提供大语言模型客户端接口和实现
package llm

import (
	"context"
)

// Client LLM客户端接口
type Client interface {
	// Chat 发送聊天请求，返回流式响应通道
	Chat(ctx context.Context, req *ChatRequest) (<-chan StreamChunk, error)

	// ChatSync 同步聊天请求，返回完整响应
	ChatSync(ctx context.Context, req *ChatRequest) (*ChatResponse, error)

	// CountTokens 估算token数量
	CountTokens(text string) int
}

// ChatRequest 聊天请求
type ChatRequest struct {
	Model       string        `json:"model"`
	Messages    []ChatMessage `json:"messages"`
	Tools       []Tool        `json:"tools,omitempty"`
	Temperature float64       `json:"temperature,omitempty"`
	MaxTokens   int           `json:"max_tokens,omitempty"`
	Stream      bool          `json:"stream"`
}

// ChatMessage 聊天消息
type ChatMessage struct {
	Role       string     `json:"role"` // system, user, assistant, tool
	Content    string     `json:"content"`
	Name       string     `json:"name,omitempty"` // tool name
	ToolCallID string     `json:"tool_call_id,omitempty"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
}

// Tool 工具定义
type Tool struct {
	Type     string      `json:"type"` // function
	Function FunctionDef `json:"function"`
}

// FunctionDef 函数定义
type FunctionDef struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

// ToolCall 工具调用
type ToolCall struct {
	ID       string       `json:"id"`
	Type     string       `json:"type"` // function
	Function FunctionCall `json:"function"`
}

// FunctionCall 函数调用
type FunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"` // JSON string
}

// StreamChunk 流式响应块
type StreamChunk struct {
	Type     ChunkType `json:"type"` // content, tool_call, done, error
	Content  string    `json:"content,omitempty"`
	ToolCall *ToolCall `json:"tool_call,omitempty"`
	Error    string    `json:"error,omitempty"`
	Usage    *Usage    `json:"usage,omitempty"`
}

// ChunkType 响应块类型
type ChunkType string

const (
	ChunkTypeContent  ChunkType = "content"
	ChunkTypeToolCall ChunkType = "tool_call"
	ChunkTypeDone     ChunkType = "done"
	ChunkTypeError    ChunkType = "error"
)

// ChatResponse 聊天响应
type ChatResponse struct {
	ID        string      `json:"id"`
	Model     string      `json:"model"`
	Message   ChatMessage `json:"message"`
	Usage     Usage       `json:"usage"`
	ToolCalls []ToolCall  `json:"tool_calls,omitempty"`
}

// Usage Token使用量
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// Config LLM客户端配置
type Config struct {
	Provider       string  `json:"provider"` // openai, azure, qwen, zhipu, ollama
	APIURL         string  `json:"api_url"`
	APIKey         string  `json:"api_key"`
	Model          string  `json:"model"`
	MaxTokens      int     `json:"max_tokens"`
	Temperature    float64 `json:"temperature"`
	TimeoutSeconds int     `json:"timeout_seconds"`
}

// NewClient 创建LLM客户端
func NewClient(cfg Config) (Client, error) {
	switch cfg.Provider {
	case "openai", "azure", "qwen", "zhipu", "ollama", "deepseek", "custom":
		return NewOpenAIClient(cfg)
	default:
		return NewOpenAIClient(cfg) // 默认使用OpenAI兼容接口
	}
}
