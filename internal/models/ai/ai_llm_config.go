package ai

import (
	"time"

	"gorm.io/gorm"
)

// LLMProvider LLM提供商
type LLMProvider string

const (
	ProviderOpenAI   LLMProvider = "openai"
	ProviderAzure    LLMProvider = "azure"
	ProviderQwen     LLMProvider = "qwen"     // 通义千问
	ProviderZhipu    LLMProvider = "zhipu"    // 智谱AI
	ProviderOllama   LLMProvider = "ollama"   // 本地部署
	ProviderDeepSeek LLMProvider = "deepseek" // DeepSeek
	ProviderCustom   LLMProvider = "custom"   // 自定义OpenAI兼容API
)

// AILLMConfig AI LLM配置模型
type AILLMConfig struct {
	ID              uint           `gorm:"primaryKey" json:"id"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
	DeletedAt       gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
	Name            string         `gorm:"size:50;uniqueIndex;not null" json:"name"`            // 配置名称
	Provider        LLMProvider    `gorm:"size:50;not null;index" json:"provider"`              // 提供商
	APIURL          string         `gorm:"column:api_url;size:255;not null" json:"api_url"`     // API地址
	APIKeyEncrypted string         `gorm:"column:api_key_encrypted;size:512;not null" json:"-"` // 加密的API密钥
	ModelName       string         `gorm:"size:100;not null" json:"model_name"`                 // 模型名称
	MaxTokens       int            `gorm:"default:4096" json:"max_tokens"`                      // 最大Token数
	Temperature     float64        `gorm:"type:decimal(3,2);default:0.70" json:"temperature"`   // 温度参数
	TimeoutSeconds  int            `gorm:"default:60" json:"timeout_seconds"`                   // 请求超时时间(秒)
	IsDefault       bool           `gorm:"default:false;index" json:"is_default"`               // 是否默认配置
	IsActive        bool           `gorm:"default:true;index" json:"is_active"`                 // 是否启用
	Description     string         `gorm:"size:500" json:"description"`                         // 描述
	CreatedBy       *uint          `json:"created_by"`                                          // 创建人ID
	UpdatedBy       *uint          `json:"updated_by"`                                          // 更新人ID
}

// TableName 指定表名
func (AILLMConfig) TableName() string {
	return "ai_llm_configs"
}

// AIMessageFeedback AI消息反馈模型
type AIMessageFeedback struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	CreatedAt      time.Time `json:"created_at"`
	UserID         uint      `gorm:"not null;uniqueIndex:idx_ai_feedback_user_msg" json:"user_id"`                  // 用户ID
	MessageID      string    `gorm:"size:36;not null;uniqueIndex:idx_ai_feedback_user_msg;index" json:"message_id"` // 消息ID
	ConversationID string    `gorm:"size:36;not null;index" json:"conversation_id"`                                 // 会话ID
	FeedbackType   string    `gorm:"size:20;not null;index" json:"feedback_type"`                                   // 反馈类型: like/dislike
	Comment        string    `gorm:"type:text" json:"comment"`                                                      // 反馈评论
}

// TableName 指定表名
func (AIMessageFeedback) TableName() string {
	return "ai_message_feedbacks"
}

// FeedbackType 反馈类型常量
const (
	FeedbackLike    = "like"
	FeedbackDislike = "dislike"
)

// DefaultLLMConfigs 默认LLM配置列表
var DefaultLLMConfigs = []AILLMConfig{
	{
		Name:           "openai-gpt4",
		Provider:       ProviderOpenAI,
		APIURL:         "https://api.openai.com/v1",
		ModelName:      "gpt-4-turbo-preview",
		MaxTokens:      4096,
		Temperature:    0.7,
		TimeoutSeconds: 60,
		IsDefault:      true,
		IsActive:       false, // 需要配置API Key后启用
		Description:    "OpenAI GPT-4 Turbo",
	},
	{
		Name:           "deepseek-chat",
		Provider:       ProviderDeepSeek,
		APIURL:         "https://api.deepseek.com/v1",
		ModelName:      "deepseek-chat",
		MaxTokens:      4096,
		Temperature:    0.7,
		TimeoutSeconds: 60,
		IsDefault:      false,
		IsActive:       false,
		Description:    "DeepSeek Chat - 高性价比大模型",
	},
	{
		Name:           "deepseek-coder",
		Provider:       ProviderDeepSeek,
		APIURL:         "https://api.deepseek.com/v1",
		ModelName:      "deepseek-coder",
		MaxTokens:      4096,
		Temperature:    0.7,
		TimeoutSeconds: 60,
		IsDefault:      false,
		IsActive:       false,
		Description:    "DeepSeek Coder - 代码专用模型",
	},
	{
		Name:           "qwen-turbo",
		Provider:       ProviderQwen,
		APIURL:         "https://dashscope.aliyuncs.com/api/v1",
		ModelName:      "qwen-turbo",
		MaxTokens:      4096,
		Temperature:    0.7,
		TimeoutSeconds: 60,
		IsDefault:      false,
		IsActive:       false,
		Description:    "阿里云通义千问",
	},
	{
		Name:           "ollama-local",
		Provider:       ProviderOllama,
		APIURL:         "http://localhost:11434/v1",
		ModelName:      "llama2",
		MaxTokens:      4096,
		Temperature:    0.7,
		TimeoutSeconds: 120,
		IsDefault:      false,
		IsActive:       false,
		Description:    "本地Ollama部署",
	},
	{
		Name:           "custom-api",
		Provider:       ProviderCustom,
		APIURL:         "http://your-api-server/v1",
		ModelName:      "your-model",
		MaxTokens:      4096,
		Temperature:    0.7,
		TimeoutSeconds: 60,
		IsDefault:      false,
		IsActive:       false,
		Description:    "自定义OpenAI兼容API",
	},
}
