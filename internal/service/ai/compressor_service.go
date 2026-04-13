package ai

import (
	"context"
	"fmt"
	"strings"

	"gorm.io/gorm"

	"devops/internal/models/ai"
	"devops/internal/repository"
	"devops/pkg/llm"
)

// CompressorService 会话压缩服务
type CompressorService struct {
	convRepo  *repository.AIConversationRepository
	msgRepo   *repository.AIMessageRepository
	llmClient llm.Client
	db        *gorm.DB
}

// NewCompressorService 创建会话压缩服务
func NewCompressorService(db *gorm.DB, llmClient llm.Client) *CompressorService {
	return &CompressorService{
		convRepo:  repository.NewAIConversationRepository(db),
		msgRepo:   repository.NewAIMessageRepository(db),
		llmClient: llmClient,
		db:        db,
	}
}

// CompressConfig 压缩配置
type CompressConfig struct {
	MaxRounds       int // 最大对话轮数（超过则压缩）
	KeepRecentCount int // 保留最近的消息数量
	MaxTokens       int // 压缩后的最大token数
}

// DefaultCompressConfig 默认压缩配置
var DefaultCompressConfig = CompressConfig{
	MaxRounds:       10,
	KeepRecentCount: 6, // 保留最近3轮对话
	MaxTokens:       4096,
}

// Compress 压缩会话历史
func (s *CompressorService) Compress(ctx context.Context, conversationID string, config *CompressConfig) error {
	if config == nil {
		config = &DefaultCompressConfig
	}

	// 获取所有消息
	messages, err := s.msgRepo.GetByConversationID(ctx, conversationID, 0)
	if err != nil {
		return fmt.Errorf("get messages: %w", err)
	}

	// 检查是否需要压缩
	if len(messages) <= config.KeepRecentCount {
		return nil // 不需要压缩
	}

	// 分离要压缩的消息和要保留的消息
	compressCount := len(messages) - config.KeepRecentCount
	toCompress := messages[:compressCount]
	// toKeep := messages[compressCount:]

	// 生成摘要
	summary, err := s.generateSummary(ctx, toCompress)
	if err != nil {
		return fmt.Errorf("generate summary: %w", err)
	}

	// 在事务中执行压缩
	return s.db.Transaction(func(tx *gorm.DB) error {
		msgRepo := repository.NewAIMessageRepository(tx)

		// 删除旧消息
		if err := msgRepo.DeleteOldMessages(ctx, conversationID, config.KeepRecentCount); err != nil {
			return fmt.Errorf("delete old messages: %w", err)
		}

		// 添加摘要消息作为系统消息
		summaryMsg := &ai.AIMessage{
			ConversationID: conversationID,
			Role:           ai.RoleSystem,
			Content:        fmt.Sprintf("[历史对话摘要]\n%s", summary),
			TokenCount:     llm.EstimateTokens(summary),
			Status:         ai.StatusComplete,
		}

		if err := msgRepo.Create(ctx, summaryMsg); err != nil {
			return fmt.Errorf("create summary message: %w", err)
		}

		return nil
	})
}

// generateSummary 生成对话摘要
func (s *CompressorService) generateSummary(ctx context.Context, messages []ai.AIMessage) (string, error) {
	if s.llmClient == nil {
		// 如果没有LLM客户端，使用简单的摘要方法
		return s.simpleCompress(messages), nil
	}

	// 构建摘要请求
	var content strings.Builder
	content.WriteString("请将以下对话历史压缩成简洁的摘要，保留关键信息和上下文：\n\n")

	for _, msg := range messages {
		role := "用户"
		if msg.Role == ai.RoleAssistant {
			role = "助手"
		} else if msg.Role == ai.RoleSystem {
			role = "系统"
		}
		content.WriteString(fmt.Sprintf("%s: %s\n", role, msg.Content))
	}

	content.WriteString("\n请用简洁的语言总结上述对话的要点：")

	req := &llm.ChatRequest{
		Messages: []llm.ChatMessage{
			{Role: "user", Content: content.String()},
		},
		MaxTokens:   500,
		Temperature: 0.3,
	}

	resp, err := s.llmClient.ChatSync(ctx, req)
	if err != nil {
		// 如果LLM调用失败，使用简单压缩
		return s.simpleCompress(messages), nil
	}

	return resp.Message.Content, nil
}

// simpleCompress 简单压缩（不使用LLM）
func (s *CompressorService) simpleCompress(messages []ai.AIMessage) string {
	var summary strings.Builder
	summary.WriteString("之前的对话要点：\n")

	// 提取每条消息的关键内容
	for i, msg := range messages {
		if msg.Role == ai.RoleUser {
			// 用户消息：提取前50个字符
			content := truncateString(msg.Content, 50)
			summary.WriteString(fmt.Sprintf("- 用户问题%d: %s\n", i/2+1, content))
		} else if msg.Role == ai.RoleAssistant {
			// 助手消息：提取前100个字符
			content := truncateString(msg.Content, 100)
			summary.WriteString(fmt.Sprintf("- 助手回答%d: %s\n", i/2+1, content))
		}
	}

	return summary.String()
}

// truncateString 截断字符串
func truncateString(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen]) + "..."
}

// NeedsCompression 检查是否需要压缩
func (s *CompressorService) NeedsCompression(ctx context.Context, conversationID string, maxRounds int) (bool, error) {
	count, err := s.msgRepo.CountByConversationID(ctx, conversationID)
	if err != nil {
		return false, err
	}
	// 每轮对话包含用户消息和助手回复
	return count > int64(maxRounds*2), nil
}

// GetCompressedMessages 获取压缩后的消息列表（用于LLM调用）
func (s *CompressorService) GetCompressedMessages(ctx context.Context, conversationID string, maxTokens int) ([]llm.ChatMessage, error) {
	messages, err := s.msgRepo.GetByConversationID(ctx, conversationID, 0)
	if err != nil {
		return nil, err
	}

	// 计算总token数
	totalTokens := 0
	for _, msg := range messages {
		totalTokens += msg.TokenCount
	}

	// 如果不超过限制，直接返回
	if totalTokens <= maxTokens {
		return s.convertToLLMMessages(messages), nil
	}

	// 需要压缩：保留最近的消息，直到达到token限制
	var result []ai.AIMessage
	currentTokens := 0

	// 从后往前遍历
	for i := len(messages) - 1; i >= 0; i-- {
		msg := messages[i]
		if currentTokens+msg.TokenCount > maxTokens {
			break
		}
		result = append([]ai.AIMessage{msg}, result...)
		currentTokens += msg.TokenCount
	}

	return s.convertToLLMMessages(result), nil
}

// convertToLLMMessages 转换为LLM消息格式
func (s *CompressorService) convertToLLMMessages(messages []ai.AIMessage) []llm.ChatMessage {
	result := make([]llm.ChatMessage, len(messages))
	for i, msg := range messages {
		result[i] = llm.ChatMessage{
			Role:       string(msg.Role),
			Content:    msg.Content,
			ToolCallID: msg.ToolCallID,
		}
	}
	return result
}
