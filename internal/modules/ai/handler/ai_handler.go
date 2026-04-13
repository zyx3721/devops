// Package handler 提供AI模块的HTTP处理器
package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"devops/internal/models/ai"
	aiservice "devops/internal/service/ai"
	"devops/pkg/llm"
	"devops/pkg/logger"
	"devops/pkg/response"
)

// LLMClientFactory LLM客户端工厂函数
type LLMClientFactory func() (llm.Client, error)

// AIHandler AI处理器
type AIHandler struct {
	db               *gorm.DB
	aiService        *aiservice.AIService
	llmClientFactory LLMClientFactory
}

// NewAIHandler 创建AI处理器
func NewAIHandler(db *gorm.DB, llmClient llm.Client) *AIHandler {
	return &AIHandler{
		db:        db,
		aiService: aiservice.NewAIService(db, llmClient, nil),
	}
}

// NewAIHandlerWithFactory 使用工厂函数创建AI处理器
func NewAIHandlerWithFactory(db *gorm.DB, factory LLMClientFactory) *AIHandler {
	// 初始化时尝试创建一次客户端
	var llmClient llm.Client
	if factory != nil {
		client, err := factory()
		if err == nil && client != nil {
			llmClient = client
		}
	}

	return &AIHandler{
		db:               db,
		aiService:        aiservice.NewAIService(db, llmClient, nil),
		llmClientFactory: factory,
	}
}

// refreshLLMClient 刷新LLM客户端
func (h *AIHandler) refreshLLMClient() {
	if h.llmClientFactory == nil {
		return
	}

	client, err := h.llmClientFactory()
	if err != nil {
		logger.L().Error("刷新LLM客户端失败", "error", err)
		return
	}

	if client != nil {
		h.aiService.UpdateLLMClient(client)
	}
}

// ChatRequest 聊天请求
type ChatRequest struct {
	ConversationID string          `json:"conversation_id"`
	Message        string          `json:"message" binding:"required"`
	Context        *ai.PageContext `json:"context"`
}

// Chat 发送消息
// @Summary 发送消息给AI助手
// @Tags AI
// @Accept json
// @Produce json
// @Param request body ChatRequest true "聊天请求"
// @Success 200 {object} response.Response
// @Router /api/v1/ai/chat [post]
func (h *AIHandler) Chat(c *gin.Context) {
	var req ChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}

	// 每次聊天前刷新LLM客户端配置
	h.refreshLLMClient()

	// 获取用户信息
	userID := getUserID(c)
	username := getUsername(c)

	// 调用AI服务
	resp, err := h.aiService.Chat(c.Request.Context(), &aiservice.ChatRequest{
		ConversationID: req.ConversationID,
		Message:        req.Message,
		Context:        req.Context,
		UserID:         userID,
		Username:       username,
	})
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "处理失败: "+err.Error())
		return
	}

	response.Success(c, resp)
}

// Stream SSE流式响应
// @Summary 获取AI响应流
// @Tags AI
// @Produce text/event-stream
// @Param message_id path string true "消息ID"
// @Success 200 {string} string "SSE流"
// @Router /api/v1/ai/stream/{message_id} [get]
func (h *AIHandler) Stream(c *gin.Context) {
	log := logger.L().WithField("handler", "Stream")
	messageID := c.Param("message_id")
	if messageID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "message_id is required"})
		return
	}

	userID := getUserID(c)
	username := getUsername(c)

	// 设置SSE响应头
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	// 使用 ResponseController 设置单独的写超时（5分钟）
	rc := http.NewResponseController(c.Writer)
	rc.SetWriteDeadline(time.Now().Add(5 * time.Minute))

	// 每次聊天前刷新LLM客户端配置
	h.refreshLLMClient()

	// 获取流式响应
	streamCh, err := h.aiService.StreamChat(c.Request.Context(), messageID, userID, username)
	if err != nil {
		log.Error("StreamChat失败: %v", err)
		c.SSEvent("error", gin.H{"error": err.Error()})
		return
	}

	log.Info("开始流式响应, messageID=%s", messageID)

	// 发送流式响应
	for {
		select {
		case chunk, ok := <-streamCh:
			if !ok {
				log.Info("流通道已关闭")
				return
			}

			switch chunk.Type {
			case llm.ChunkTypeContent:
				c.SSEvent("content", gin.H{"content": chunk.Content})
				c.Writer.Flush()
				// 每次写入后延长超时
				rc.SetWriteDeadline(time.Now().Add(5 * time.Minute))
			case llm.ChunkTypeToolCall:
				c.SSEvent("tool_call", gin.H{"tool_call": chunk.ToolCall})
				c.Writer.Flush()
			case llm.ChunkTypeDone:
				log.Info("流式响应完成")
				c.SSEvent("done", gin.H{"usage": chunk.Usage})
				c.Writer.Flush()
				return
			case llm.ChunkTypeError:
				log.Error("流式响应错误: %s", chunk.Error)
				c.SSEvent("error", gin.H{"error": chunk.Error})
				c.Writer.Flush()
				return
			}

		case <-c.Request.Context().Done():
			log.Info("客户端断开连接")
			return
		}
	}
}

// GetHistory 获取会话历史
// @Summary 获取会话历史列表
// @Tags AI
// @Produce json
// @Param page query int false "页码"
// @Param page_size query int false "每页数量"
// @Success 200 {object} response.Response
// @Router /api/v1/ai/history [get]
func (h *AIHandler) GetHistory(c *gin.Context) {
	userID := getUserID(c)
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	conversations, total, err := h.aiService.GetHistory(c.Request.Context(), userID, page, pageSize)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "获取历史失败: "+err.Error())
		return
	}

	response.Page(c, conversations, total, page, pageSize)
}

// GetConversation 获取会话详情
// @Summary 获取会话详情
// @Tags AI
// @Produce json
// @Param id path string true "会话ID"
// @Param limit query int false "消息数量限制"
// @Success 200 {object} response.Response
// @Router /api/v1/ai/conversation/{id} [get]
func (h *AIHandler) GetConversation(c *gin.Context) {
	conversationID := c.Param("id")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))

	conversation, err := h.aiService.GetConversation(c.Request.Context(), conversationID, limit)
	if err != nil {
		response.Error(c, http.StatusNotFound, "会话不存在")
		return
	}

	response.Success(c, conversation)
}

// DeleteConversation 删除会话
// @Summary 删除会话
// @Tags AI
// @Param id path string true "会话ID"
// @Success 200 {object} response.Response
// @Router /api/v1/ai/conversation/{id} [delete]
func (h *AIHandler) DeleteConversation(c *gin.Context) {
	conversationID := c.Param("id")

	if err := h.aiService.DeleteConversation(c.Request.Context(), conversationID); err != nil {
		response.Error(c, http.StatusInternalServerError, "删除失败: "+err.Error())
		return
	}

	response.Success(c, nil)
}

// ExecuteRequest 执行操作请求
type ExecuteRequest struct {
	Action         string                 `json:"action" binding:"required"`
	Params         map[string]interface{} `json:"params"`
	ConversationID string                 `json:"conversation_id"`
	MessageID      string                 `json:"message_id"`
}

// FeedbackRequest 消息反馈请求
type FeedbackRequest struct {
	Rating  string `json:"rating" binding:"required,oneof=like dislike"` // like/dislike
	Comment string `json:"comment"`                                      // 可选评论
}

// Execute 执行操作
// @Summary 执行AI建议的操作
// @Tags AI
// @Accept json
// @Produce json
// @Param request body ExecuteRequest true "执行请求"
// @Success 200 {object} response.Response
// @Router /api/v1/ai/execute [post]
func (h *AIHandler) Execute(c *gin.Context) {
	var req ExecuteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}

	userID := getUserID(c)
	username := getUsername(c)

	result, err := h.aiService.ExecuteAction(
		c.Request.Context(),
		userID,
		username,
		req.Action,
		req.Params,
		req.ConversationID,
		req.MessageID,
	)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "执行失败: "+err.Error())
		return
	}

	response.Success(c, result)
}

// Feedback 提交消息反馈
// @Summary 提交消息反馈
// @Tags AI
// @Accept json
// @Produce json
// @Param message_id path string true "消息ID"
// @Param request body FeedbackRequest true "反馈请求"
// @Success 200 {object} response.Response
// @Router /api/v1/ai/message/{message_id}/feedback [post]
func (h *AIHandler) Feedback(c *gin.Context) {
	messageID := c.Param("message_id")
	if messageID == "" {
		response.Error(c, http.StatusBadRequest, "message_id is required")
		return
	}

	var req FeedbackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}

	if err := h.aiService.SubmitFeedback(c.Request.Context(), messageID, req.Rating, req.Comment); err != nil {
		response.Error(c, http.StatusInternalServerError, "提交反馈失败: "+err.Error())
		return
	}

	response.Success(c, nil)
}

// 辅助函数

func getUserID(c *gin.Context) uint {
	if id, exists := c.Get("user_id"); exists {
		if userID, ok := id.(uint); ok {
			return userID
		}
	}
	return 0
}

func getUsername(c *gin.Context) string {
	if name, exists := c.Get("username"); exists {
		if username, ok := name.(string); ok {
			return username
		}
	}
	return ""
}

// RegisterRoutes 注册AI聊天相关路由
func (h *AIHandler) RegisterRoutes(r *gin.RouterGroup) {
	g := r.Group("/ai")
	{
		// 聊天相关
		g.POST("/chat", h.Chat)
		g.GET("/stream/:message_id", h.Stream)
		g.GET("/history", h.GetHistory)
		g.GET("/conversation/:id", h.GetConversation)
		g.DELETE("/conversation/:id", h.DeleteConversation)
		g.POST("/execute", h.Execute)
		// 消息反馈
		g.POST("/message/:message_id/feedback", h.Feedback)
	}
}
