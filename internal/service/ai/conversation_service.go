// Package ai 提供AI Copilot相关服务
package ai

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"devops/internal/models/ai"
	"devops/internal/repository"
	"devops/pkg/llm"
)

// ConversationService 会话管理服务
type ConversationService struct {
	convRepo *repository.AIConversationRepository
	msgRepo  *repository.AIMessageRepository
	db       *gorm.DB
}

// NewConversationService 创建会话管理服务
func NewConversationService(db *gorm.DB) *ConversationService {
	return &ConversationService{
		convRepo: repository.NewAIConversationRepository(db),
		msgRepo:  repository.NewAIMessageRepository(db),
		db:       db,
	}
}

// GetOrCreate 获取或创建会话
func (s *ConversationService) GetOrCreate(ctx context.Context, userID uint, conversationID string) (*ai.AIConversation, error) {
	if conversationID != "" {
		conv, err := s.convRepo.GetByID(ctx, conversationID)
		if err == nil && conv.UserID == userID {
			return conv, nil
		}
	}

	// 创建新会话
	return s.Create(ctx, userID, nil)
}

// Create 创建新会话
func (s *ConversationService) Create(ctx context.Context, userID uint, pageCtx *ai.PageContext) (*ai.AIConversation, error) {
	conv := &ai.AIConversation{
		ID:           uuid.New().String(),
		UserID:       userID,
		Title:        "新对话",
		Context:      pageCtx,
		MessageCount: 0,
	}

	if err := s.convRepo.Create(ctx, conv); err != nil {
		return nil, fmt.Errorf("create conversation: %w", err)
	}

	return conv, nil
}

// GetByID 根据ID获取会话
func (s *ConversationService) GetByID(ctx context.Context, id string) (*ai.AIConversation, error) {
	return s.convRepo.GetByID(ctx, id)
}

// GetByIDWithMessages 获取会话及其消息
func (s *ConversationService) GetByIDWithMessages(ctx context.Context, id string, messageLimit int) (*ai.AIConversation, error) {
	return s.convRepo.GetByIDWithMessages(ctx, id, messageLimit)
}

// GetUserConversations 获取用户的会话列表
func (s *ConversationService) GetUserConversations(ctx context.Context, userID uint, page, pageSize int) ([]ai.AIConversation, int64, error) {
	return s.convRepo.GetByUserID(ctx, userID, page, pageSize)
}

// AddMessage 添加消息到会话
func (s *ConversationService) AddMessage(ctx context.Context, conversationID string, role ai.MessageRole, content string) (*ai.AIMessage, error) {
	msg := &ai.AIMessage{
		ID:             uuid.New().String(),
		ConversationID: conversationID,
		Role:           role,
		Content:        content,
		TokenCount:     llm.EstimateTokens(content),
		Status:         ai.StatusComplete,
	}

	if err := s.msgRepo.Create(ctx, msg); err != nil {
		return nil, fmt.Errorf("create message: %w", err)
	}

	// 更新会话消息计数
	if err := s.convRepo.UpdateMessageCount(ctx, conversationID); err != nil {
		// 不影响主流程
	}

	// 如果是第一条用户消息，更新会话标题
	count, _ := s.msgRepo.CountByConversationID(ctx, conversationID)
	if count == 1 && role == ai.RoleUser {
		title := s.generateTitle(content)
		_ = s.convRepo.UpdateTitle(ctx, conversationID, title)
	}

	return msg, nil
}

// AddAssistantMessage 添加助手消息（支持流式）
func (s *ConversationService) AddAssistantMessage(ctx context.Context, conversationID string) (*ai.AIMessage, error) {
	msg := &ai.AIMessage{
		ID:             uuid.New().String(),
		ConversationID: conversationID,
		Role:           ai.RoleAssistant,
		Content:        "",
		Status:         ai.StatusStreaming,
	}

	if err := s.msgRepo.Create(ctx, msg); err != nil {
		return nil, fmt.Errorf("create message: %w", err)
	}

	return msg, nil
}

// UpdateMessageContent 更新消息内容（流式完成后）
func (s *ConversationService) UpdateMessageContent(ctx context.Context, messageID string, content string, tokenCount int) error {
	return s.msgRepo.UpdateContent(ctx, messageID, content, tokenCount)
}

// UpdateMessageStatus 更新消息状态
func (s *ConversationService) UpdateMessageStatus(ctx context.Context, messageID string, status ai.MessageStatus, errorMsg string) error {
	return s.msgRepo.UpdateStatus(ctx, messageID, status, errorMsg)
}

// GetHistory 获取会话历史消息
func (s *ConversationService) GetHistory(ctx context.Context, conversationID string, limit int) ([]ai.AIMessage, error) {
	return s.msgRepo.GetByConversationID(ctx, conversationID, limit)
}

// GetRecentMessages 获取最近的消息
func (s *ConversationService) GetRecentMessages(ctx context.Context, conversationID string, limit int) ([]ai.AIMessage, error) {
	return s.msgRepo.GetRecentMessages(ctx, conversationID, limit)
}

// UpdateContext 更新会话上下文
func (s *ConversationService) UpdateContext(ctx context.Context, conversationID string, pageCtx *ai.PageContext) error {
	return s.convRepo.UpdateContext(ctx, conversationID, pageCtx)
}

// Delete 删除会话
func (s *ConversationService) Delete(ctx context.Context, id string) error {
	return s.convRepo.Delete(ctx, id)
}

// generateTitle 根据第一条消息生成标题
func (s *ConversationService) generateTitle(content string) string {
	// 截取前30个字符作为标题
	runes := []rune(content)
	if len(runes) > 30 {
		return string(runes[:30]) + "..."
	}
	return content
}

// GetMessageCount 获取会话消息数量
func (s *ConversationService) GetMessageCount(ctx context.Context, conversationID string) (int64, error) {
	return s.msgRepo.CountByConversationID(ctx, conversationID)
}

// NeedsCompression 检查会话是否需要压缩
func (s *ConversationService) NeedsCompression(ctx context.Context, conversationID string, maxRounds int) (bool, error) {
	count, err := s.msgRepo.CountByConversationID(ctx, conversationID)
	if err != nil {
		return false, err
	}
	// 每轮对话包含用户消息和助手回复，所以乘以2
	return count > int64(maxRounds*2), nil
}

// GetMessagesForLLM 获取用于LLM调用的消息列表
func (s *ConversationService) GetMessagesForLLM(ctx context.Context, conversationID string, maxMessages int) ([]llm.ChatMessage, error) {
	messages, err := s.msgRepo.GetRecentMessages(ctx, conversationID, maxMessages)
	if err != nil {
		return nil, err
	}

	result := make([]llm.ChatMessage, len(messages))
	for i, msg := range messages {
		result[i] = llm.ChatMessage{
			Role:       string(msg.Role),
			Content:    msg.Content,
			Name:       "",
			ToolCallID: msg.ToolCallID,
		}
	}

	return result, nil
}

// ExportHistory 导出会话历史
func (s *ConversationService) ExportHistory(ctx context.Context, conversationID string) (*ConversationExport, error) {
	conv, err := s.convRepo.GetByIDWithMessages(ctx, conversationID, 0)
	if err != nil {
		return nil, err
	}

	export := &ConversationExport{
		ID:        conv.ID,
		Title:     conv.Title,
		CreatedAt: conv.CreatedAt,
		Messages:  make([]MessageExport, len(conv.Messages)),
	}

	for i, msg := range conv.Messages {
		export.Messages[i] = MessageExport{
			Role:      string(msg.Role),
			Content:   msg.Content,
			Timestamp: msg.CreatedAt,
		}
	}

	return export, nil
}

// ConversationExport 会话导出格式
type ConversationExport struct {
	ID        string          `json:"id"`
	Title     string          `json:"title"`
	CreatedAt time.Time       `json:"created_at"`
	Messages  []MessageExport `json:"messages"`
}

// MessageExport 消息导出格式
type MessageExport struct {
	Role      string    `json:"role"`
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
}
