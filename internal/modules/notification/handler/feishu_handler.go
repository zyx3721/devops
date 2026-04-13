package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"devops/internal/config"
	"devops/internal/domain/notification/service/feishu"
	"devops/internal/models"
	"devops/internal/repository"
	apperrors "devops/pkg/errors"
	"devops/pkg/ioc"
	"devops/pkg/logger"
)

func init() {
	ioc.Api.RegisterContainer("FeishuHandler", &FeishuApiHandler{})
}

type FeishuApiHandler struct {
	handler *FeishuHandler
}

func (h *FeishuApiHandler) Init() error {
	cfg, _ := config.LoadConfig()

	// 初始化飞书存储
	feishu.InitFeishuStore(cfg.GetDB())

	client := feishu.NewClient(cfg)
	h.handler = NewFeishuHandler(client, cfg.GetDB())

	// 设置 token 仓储并从数据库加载 token
	tokenRepo := repository.NewFeishuUserTokenRepository(cfg.GetDB())
	client.SetTokenRepository(tokenRepo)
	client.LoadTokenFromDB()

	// 初始化回调处理器
	feishu.InitCallbackHandler(client)

	// 初始化回调管理器并启动所有应用的回调
	callbackMgr := feishu.InitCallbackManager(cfg.GetDB(), cfg)
	callbackMgr.StartAllCallbacks()

	// 启动 user token 自动刷新任务
	client.StartTokenRefreshTask()

	root := cfg.Application.GinRootRouter().Group("feishu")
	h.Register(root)

	return nil
}

func (h *FeishuApiHandler) Register(r gin.IRouter) {
	r.POST("/api/send-card", h.handler.SendCard)
	r.POST("/send-message", h.handler.SendMessage)
	r.GET("/version", h.handler.Version)

	// 回调管理
	r.GET("/callback/status", h.handler.GetCallbackStatus)
	r.POST("/callback/refresh", h.handler.RefreshCallbacks)

	// 消息日志
	r.GET("/logs", h.handler.ListMessageLogs)
	r.GET("/logs/:id", h.handler.GetMessageLog)

	// 用户搜索
	r.POST("/user/search", h.handler.SearchUser)
	r.GET("/user/:id", h.handler.GetUser)
	r.POST("/user/token", h.handler.SetUserToken)
	r.GET("/user/token/status", h.handler.GetUserTokenStatus)

	// OAuth 授权
	r.GET("/oauth/authorize", h.handler.OAuthAuthorize)
	r.GET("/oauth/callback", h.handler.OAuthCallback)

	// 群聊管理
	r.GET("/chat", h.handler.ListChats)
	r.POST("/chat", h.handler.CreateChat)
	r.POST("/chat/:id/members", h.handler.AddChatMembers)

	// 飞书应用管理
	app := r.Group("/app")
	{
		app.GET("", h.handler.ListApps)
		app.GET("/:id", h.handler.GetApp)
		app.POST("", h.handler.CreateApp)
		app.PUT("/:id", h.handler.UpdateApp)
		app.DELETE("/:id", h.handler.DeleteApp)
		app.POST("/:id/default", h.handler.SetDefaultApp)
		app.GET("/:id/bindings", h.handler.GetAppBindings)
	}

	// 飞书机器人管理
	bot := r.Group("/bot")
	{
		bot.GET("", h.handler.ListBots)
		bot.GET("/:id", h.handler.GetBot)
		bot.POST("", h.handler.CreateBot)
		bot.PUT("/:id", h.handler.UpdateBot)
		bot.DELETE("/:id", h.handler.DeleteBot)
	}
}

// FeishuHandler HTTP处理器
type FeishuHandler struct {
	client  *feishu.Client
	logger  *logger.Logger
	sender  feishu.Sender
	appRepo *repository.FeishuAppRepository
	botRepo *repository.FeishuBotRepository
	logRepo *repository.FeishuMessageLogRepository
}

// NewFeishuHandler 创建新的HTTP处理器
func NewFeishuHandler(client *feishu.Client, db interface{}) *FeishuHandler {
	h := &FeishuHandler{
		client: client,
		logger: logger.NewLogger("INFO"),
		sender: feishu.NewAPISender(client),
	}
	if gormDB, ok := db.(*gorm.DB); ok {
		h.appRepo = repository.NewFeishuAppRepository(gormDB)
		h.botRepo = repository.NewFeishuBotRepository(gormDB)
		h.logRepo = repository.NewFeishuMessageLogRepository(gormDB)
	}
	return h
}

// SendCard 发送动态生成的灰度发布卡片
func (h *FeishuHandler) SendCard(c *gin.Context) {
	var req feishu.SendGrayCardRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("Invalid request body: %v", err)
		h.writeError(c, http.StatusBadRequest, fmt.Sprintf("Invalid request body: %v", err))
		return
	}

	if req.ReceiveID == "" || req.ReceiveIDType == "" {
		h.writeError(c, http.StatusBadRequest, "receive_id and receive_id_type are required")
		return
	}

	requestID := fmt.Sprintf("req_%d", time.Now().UnixNano())

	req.CardData.ReceiveID = req.ReceiveID
	req.CardData.ReceiveIDType = req.ReceiveIDType

	feishu.GlobalStore.Save(requestID, req.CardData)

	displayCardData := req.CardData
	hasGray := false
	for _, s := range req.CardData.Services {
		for _, a := range s.Actions {
			if strings.EqualFold(a, "gray") || a == "灰度" {
				hasGray = true
				break
			}
		}
		if hasGray {
			break
		}
	}

	if hasGray {
		var filteredServices []feishu.Service
		for _, s := range req.CardData.Services {
			hasGrayAction := false
			for _, a := range s.Actions {
				if strings.EqualFold(a, "gray") || a == "灰度" {
					hasGrayAction = true
					break
				}
			}

			if hasGrayAction {
				newService := s
				newActions := []string{}
				for _, a := range s.Actions {
					if strings.EqualFold(a, "official") || strings.EqualFold(a, "release") || a == "正式" {
						continue
					}
					newActions = append(newActions, a)
				}
				newService.Actions = newActions
				filteredServices = append(filteredServices, newService)
			}
		}
		displayCardData.Services = filteredServices
	}

	cardContent := feishu.BuildCard(displayCardData, requestID, nil, nil)

	cardBytes, err := json.Marshal(cardContent)
	if err != nil {
		h.logger.Error("Failed to marshal card content: %v", err)
		h.writeError(c, http.StatusInternalServerError, "Failed to build card content")
		return
	}

	ctx := c.Request.Context()
	err = h.sender.Send(ctx, req.ReceiveID, req.ReceiveIDType, "interactive", string(cardBytes))

	// 记录日志
	if h.logRepo != nil {
		logEntry := &models.FeishuMessageLog{
			MsgType:       "interactive",
			ReceiveID:     req.ReceiveID,
			ReceiveIDType: req.ReceiveIDType,
			Content:       string(cardBytes),
			Title:         req.CardData.Title,
			Source:        "manual",
			Status:        "success",
		}
		if err != nil {
			logEntry.Status = "failed"
			logEntry.ErrorMsg = err.Error()
		}
		h.logRepo.Create(ctx, logEntry)
	}

	if err != nil {
		h.logger.Error("Failed to send gray card: %v", err)
		h.writeError(c, http.StatusInternalServerError, fmt.Sprintf("Failed to send gray card: %v", err))
		return
	}

	h.writeSuccess(c, map[string]string{
		"message": "Gray release card sent successfully",
	})
}

// SendMessage 发送消息
func (h *FeishuHandler) SendMessage(c *gin.Context) {
	var req feishu.SendRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("Invalid request body: %v", err)
		h.writeError(c, http.StatusBadRequest, fmt.Sprintf("Invalid request body: %v", err))
		return
	}

	if req.ReceiveID == "" || req.ReceiveIDType == "" || req.MsgType == "" {
		h.writeError(c, http.StatusBadRequest, "receive_id, receive_id_type and msg_type are required")
		return
	}

	ctx := c.Request.Context()
	err := h.sender.Send(ctx, req.ReceiveID, req.ReceiveIDType, req.MsgType, string(req.Content))

	// 记录日志
	if h.logRepo != nil {
		logEntry := &models.FeishuMessageLog{
			MsgType:       req.MsgType,
			ReceiveID:     req.ReceiveID,
			ReceiveIDType: req.ReceiveIDType,
			Content:       string(req.Content),
			Source:        "manual",
			Status:        "success",
		}
		if err != nil {
			logEntry.Status = "failed"
			logEntry.ErrorMsg = err.Error()
		}
		h.logRepo.Create(ctx, logEntry)
	}

	if err != nil {
		h.logger.Error("Failed to send message: %v", err)
		h.writeError(c, http.StatusInternalServerError, fmt.Sprintf("Failed to send message: %v", err))
		return
	}

	h.writeSuccess(c, map[string]string{
		"message": "Message sent successfully",
	})
}

// Version 版本信息接口
func (h *FeishuHandler) Version(c *gin.Context) {
	h.writeSuccess(c, map[string]string{
		"version": os.Getenv("VERSION"),
	})
}

// writeSuccess 写入成功响应
func (h *FeishuHandler) writeSuccess(c *gin.Context, data interface{}) {
	response := feishu.APIResponse{
		Code:    apperrors.Success,
		Message: "success",
		Data:    data,
	}
	c.JSON(http.StatusOK, response)
}

// writeError 写入错误响应
func (h *FeishuHandler) writeError(c *gin.Context, statusCode int, message string) {
	code := apperrors.ErrCodeInternalError
	if statusCode >= 400 && statusCode < 500 {
		code = apperrors.ErrCodeInvalidParams
	}
	response := feishu.APIResponse{
		Code:    code,
		Message: message,
		Data:    nil,
	}
	c.JSON(statusCode, response)
}
