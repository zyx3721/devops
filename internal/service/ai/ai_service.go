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

// AIService AI主服务
type AIService struct {
	db               *gorm.DB
	llmClient        llm.Client
	convService      *ConversationService
	compressor       *CompressorService
	toolExecutor     *ToolExecutor
	knowledgeService *KnowledgeService
	contextBuilder   *ContextBuilder
	configRepo       *repository.AILLMConfigRepository
	sanitizer        *llm.Sanitizer
}

// NewAIService 创建AI服务
func NewAIService(db *gorm.DB, llmClient llm.Client, permChecker PermissionChecker) *AIService {
	return &AIService{
		db:               db,
		llmClient:        llmClient,
		convService:      NewConversationService(db),
		compressor:       NewCompressorService(db, llmClient),
		toolExecutor:     NewToolExecutor(db, permChecker),
		knowledgeService: NewKnowledgeService(db),
		contextBuilder:   NewContextBuilder(),
		configRepo:       repository.NewAILLMConfigRepository(db),
		sanitizer:        llm.NewSanitizer(),
	}
}

// ChatRequest 聊天请求
type ChatRequest struct {
	ConversationID string          `json:"conversation_id"`
	Message        string          `json:"message"`
	Context        *ai.PageContext `json:"context"`
	UserID         uint            `json:"user_id"`
	Username       string          `json:"username"`
}

// ChatResponse 聊天响应
type ChatResponse struct {
	ConversationID string `json:"conversation_id"`
	MessageID      string `json:"message_id"`
	StreamURL      string `json:"stream_url"`
}

// Chat 处理聊天请求
func (s *AIService) Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	// 获取或创建会话
	conv, err := s.convService.GetOrCreate(ctx, req.UserID, req.ConversationID)
	if err != nil {
		return nil, fmt.Errorf("get or create conversation: %w", err)
	}

	// 更新上下文
	if req.Context != nil {
		if err := s.convService.UpdateContext(ctx, conv.ID, req.Context); err != nil {
			// 不影响主流程
		}
	}

	// 添加用户消息
	_, err = s.convService.AddMessage(ctx, conv.ID, ai.RoleUser, req.Message)
	if err != nil {
		return nil, fmt.Errorf("add user message: %w", err)
	}

	// 创建助手消息（流式）
	assistantMsg, err := s.convService.AddAssistantMessage(ctx, conv.ID)
	if err != nil {
		return nil, fmt.Errorf("add assistant message: %w", err)
	}

	return &ChatResponse{
		ConversationID: conv.ID,
		MessageID:      assistantMsg.ID,
		StreamURL:      fmt.Sprintf("/api/v1/ai/stream/%s", assistantMsg.ID),
	}, nil
}

// StreamChat 流式聊天处理
func (s *AIService) StreamChat(ctx context.Context, messageID string, userID uint, username string) (<-chan llm.StreamChunk, error) {
	// 检查 LLM 客户端
	if s.llmClient == nil {
		return nil, fmt.Errorf("LLM客户端未配置，请先在系统设置中配置并启用LLM")
	}

	// 获取消息
	msg, err := s.convService.msgRepo.GetByID(ctx, messageID)
	if err != nil {
		return nil, fmt.Errorf("get message: %w", err)
	}

	// 获取会话
	conv, err := s.convService.GetByIDWithMessages(ctx, msg.ConversationID, 20)
	if err != nil {
		return nil, fmt.Errorf("get conversation: %w", err)
	}

	// 构建LLM请求
	llmReq, err := s.buildLLMRequest(ctx, conv, userID)
	if err != nil {
		return nil, fmt.Errorf("build llm request: %w", err)
	}

	// 调用LLM
	streamCh, err := s.llmClient.Chat(ctx, llmReq)
	if err != nil {
		return nil, fmt.Errorf("call llm: %w", err)
	}

	// 创建输出通道
	outputCh := make(chan llm.StreamChunk, 100)

	// 处理流式响应
	go s.processStream(ctx, streamCh, outputCh, messageID, conv.ID, userID, username)

	return outputCh, nil
}

// processStream 处理流式响应
func (s *AIService) processStream(ctx context.Context, inputCh <-chan llm.StreamChunk, outputCh chan<- llm.StreamChunk, messageID, conversationID string, userID uint, username string) {
	defer close(outputCh)

	var contentBuilder strings.Builder
	var toolCalls []llm.ToolCall

	for chunk := range inputCh {
		switch chunk.Type {
		case llm.ChunkTypeContent:
			contentBuilder.WriteString(chunk.Content)
			outputCh <- chunk

		case llm.ChunkTypeToolCall:
			if chunk.ToolCall != nil {
				toolCalls = append(toolCalls, *chunk.ToolCall)
			}
			outputCh <- chunk

		case llm.ChunkTypeDone:
			// 保存消息内容
			content := contentBuilder.String()
			tokenCount := 0
			if chunk.Usage != nil {
				tokenCount = chunk.Usage.CompletionTokens
				llm.GlobalTokenCounter.Add(chunk.Usage.TotalTokens)
			}

			if err := s.convService.UpdateMessageContent(ctx, messageID, content, tokenCount); err != nil {
				// 记录错误但不影响响应
			}

			// 更新会话消息计数
			s.convService.convRepo.UpdateMessageCount(ctx, conversationID)

			// 处理工具调用
			if len(toolCalls) > 0 {
				s.handleToolCalls(ctx, toolCalls, conversationID, messageID, userID, username, outputCh)
			}

			outputCh <- chunk

		case llm.ChunkTypeError:
			s.convService.UpdateMessageStatus(ctx, messageID, ai.StatusError, chunk.Error)
			outputCh <- chunk
		}
	}
}

// handleToolCalls 处理工具调用
func (s *AIService) handleToolCalls(ctx context.Context, toolCalls []llm.ToolCall, conversationID, messageID string, userID uint, username string, outputCh chan<- llm.StreamChunk) {
	for _, tc := range toolCalls {
		toolName, params, err := ParseToolCall(&tc)
		if err != nil {
			continue
		}

		result, err := s.toolExecutor.Execute(ctx, userID, username, toolName, params, conversationID, messageID)
		if err != nil {
			outputCh <- llm.StreamChunk{
				Type:  llm.ChunkTypeError,
				Error: fmt.Sprintf("执行工具 %s 失败: %v", toolName, err),
			}
			continue
		}

		// 如果需要确认，发送确认请求
		if result.NeedConfirm {
			outputCh <- llm.StreamChunk{
				Type:    llm.ChunkTypeContent,
				Content: fmt.Sprintf("\n\n⚠️ %s\n", result.ConfirmMsg),
			}
		}
	}
}

// buildLLMRequest 构建LLM请求
func (s *AIService) buildLLMRequest(ctx context.Context, conv *ai.AIConversation, userID uint) (*llm.ChatRequest, error) {
	// 构建系统提示词
	systemPrompt, err := s.buildSystemPrompt(ctx, conv.Context)
	if err != nil {
		return nil, err
	}

	// 构建消息列表
	messages := []llm.ChatMessage{
		{Role: "system", Content: systemPrompt},
	}

	// 添加历史消息
	for _, msg := range conv.Messages {
		// 过滤敏感信息
		content := s.sanitizer.Sanitize(msg.Content)
		messages = append(messages, llm.ChatMessage{
			Role:    string(msg.Role),
			Content: content,
		})
	}

	// 获取可用工具
	tools := s.toolExecutor.GetAvailableTools(ctx, userID)

	return &llm.ChatRequest{
		Messages: messages,
		Tools:    tools,
		Stream:   true,
	}, nil
}

// buildSystemPrompt 构建系统提示词
func (s *AIService) buildSystemPrompt(ctx context.Context, pageCtx *ai.PageContext) (string, error) {
	// 构建上下文字符串
	contextStr := s.contextBuilder.BuildContextString(pageCtx)

	// 获取相关知识
	knowledgeStr, _ := s.knowledgeService.BuildKnowledgeContext(ctx, pageCtx, 2000)

	return fmt.Sprintf(SystemPromptTemplate, contextStr, knowledgeStr), nil
}

// ExecuteAction 执行操作
func (s *AIService) ExecuteAction(ctx context.Context, userID uint, username string, action string, params map[string]interface{}, conversationID, messageID string) (*ExecuteResult, error) {
	return s.toolExecutor.Execute(ctx, userID, username, action, params, conversationID, messageID)
}

// GetHistory 获取会话历史
func (s *AIService) GetHistory(ctx context.Context, userID uint, page, pageSize int) ([]ai.AIConversation, int64, error) {
	return s.convService.GetUserConversations(ctx, userID, page, pageSize)
}

// GetConversation 获取会话详情
func (s *AIService) GetConversation(ctx context.Context, conversationID string, messageLimit int) (*ai.AIConversation, error) {
	return s.convService.GetByIDWithMessages(ctx, conversationID, messageLimit)
}

// DeleteConversation 删除会话
func (s *AIService) DeleteConversation(ctx context.Context, conversationID string) error {
	return s.convService.Delete(ctx, conversationID)
}

// SubmitFeedback 提交消息反馈
func (s *AIService) SubmitFeedback(ctx context.Context, messageID, rating, comment string) error {
	return s.convService.msgRepo.UpdateFeedback(ctx, messageID, rating, comment)
}

// GetLLMConfig 获取LLM配置
func (s *AIService) GetLLMConfig(ctx context.Context) (*ai.AILLMConfig, error) {
	return s.configRepo.GetDefault(ctx)
}

// UpdateLLMClient 更新LLM客户端
func (s *AIService) UpdateLLMClient(client llm.Client) {
	s.llmClient = client
	s.compressor.llmClient = client
}

// SystemPromptTemplate 系统提示词模板
const SystemPromptTemplate = `你是 DevOps 平台的智能运维助手，名叫"小运"。你的职责是：

1. **系统指导**：帮助用户了解和使用 DevOps 平台的各项功能
2. **问题诊断**：分析日志、告警，帮助定位问题根因
3. **操作建议**：提供运维操作建议和最佳实践
4. **执行操作**：在用户确认后执行运维操作

## 当前上下文
%s

## 可用工具
你可以使用以下工具来帮助用户：
- query_logs: 查询应用日志
- query_alerts: 查询告警信息
- query_metrics: 查询监控指标
- restart_app: 重启应用
- scale_pod: 调整 Pod 副本数
- rollback: 回滚到指定版本
- silence_alert: 静默告警
- query_knowledge: 查询系统使用文档

## 回复规范
1. 使用中文回复
2. 技术术语保持英文
3. 操作建议要具体可执行
4. 危险操作要明确提醒风险
5. 不确定的信息要说明

## 系统功能知识
%s
`
