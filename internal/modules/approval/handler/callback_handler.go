package handler

import (
	"io"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"devops/internal/config"
	"devops/internal/repository"
	"devops/internal/service/approval"
	"devops/pkg/ioc"
	"devops/pkg/logger"
	"devops/pkg/response"
)

var callbackLog = logger.L().WithField("module", "approval_callback_handler")

func init() {
	ioc.Api.RegisterContainer("ApprovalCallbackHandler", &ApprovalCallbackApiHandler{})
}

type ApprovalCallbackApiHandler struct {
	handler *CallbackHandler
}

func (h *ApprovalCallbackApiHandler) Init() error {
	cfg, err := config.LoadConfig()
	if err != nil {
		return err
	}

	db := cfg.GetDB()
	h.handler = NewCallbackHandler(db)

	// 回调路由不需要认证
	root := cfg.Application.GinRootRouter().Group("callback")
	h.Register(root)

	return nil
}

func (h *ApprovalCallbackApiHandler) Register(r gin.IRouter) {
	// 飞书审批回调
	r.POST("/feishu/approval", h.handler.HandleFeishuCallback)
	// 钉钉审批回调
	r.POST("/dingtalk/approval", h.handler.HandleDingTalkCallback)
	// 企业微信审批回调
	r.POST("/wecom/approval", h.handler.HandleWeComCallback)
}

// CallbackHandler 回调处理器
type CallbackHandler struct {
	callbackHandler *approval.CallbackHandler
}

// NewCallbackHandler 创建回调处理器
func NewCallbackHandler(db *gorm.DB) *CallbackHandler {
	nodeInstanceRepo := repository.NewApprovalNodeInstanceRepository(db)
	actionRepo := repository.NewApprovalActionRepository(db)
	instanceRepo := repository.NewApprovalInstanceRepository(db)

	nodeExecutor := approval.NewNodeExecutor(nodeInstanceRepo, actionRepo, instanceRepo)
	callbackHandler := approval.NewCallbackHandler(nodeExecutor)

	return &CallbackHandler{
		callbackHandler: callbackHandler,
	}
}

// HandleFeishuCallback 处理飞书卡片回调
// @Summary 飞书审批回调
// @Description 处理飞书卡片按钮点击回调
// @Tags 回调
// @Accept json
// @Produce json
// @Success 200 {object} map[string]any "成功"
// @Router /callback/feishu/approval [post]
func (h *CallbackHandler) HandleFeishuCallback(c *gin.Context) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		callbackLog.WithError(err).Error("读取请求体失败")
		response.BadRequest(c, "读取请求体失败")
		return
	}

	signature := c.GetHeader("X-Lark-Signature")
	timestamp := c.GetHeader("X-Lark-Request-Timestamp")

	result, err := h.callbackHandler.HandleFeishuCallback(c.Request.Context(), body, signature, timestamp)
	if err != nil {
		callbackLog.WithError(err).Error("处理飞书回调失败")
		response.InternalError(c, err.Error())
		return
	}

	c.JSON(200, result)
}

// HandleDingTalkCallback 处理钉钉回调
// @Summary 钉钉审批回调
// @Description 处理钉钉卡片按钮点击回调
// @Tags 回调
// @Accept json
// @Produce json
// @Success 200 {object} map[string]any "成功"
// @Router /callback/dingtalk/approval [post]
func (h *CallbackHandler) HandleDingTalkCallback(c *gin.Context) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		callbackLog.WithError(err).Error("读取请求体失败")
		response.BadRequest(c, "读取请求体失败")
		return
	}

	signature := c.GetHeader("sign")
	timestamp := c.GetHeader("timestamp")

	result, err := h.callbackHandler.HandleDingTalkCallback(c.Request.Context(), body, signature, timestamp)
	if err != nil {
		callbackLog.WithError(err).Error("处理钉钉回调失败")
		response.InternalError(c, err.Error())
		return
	}

	c.JSON(200, result)
}

// HandleWeComCallback 处理企业微信回调
// @Summary 企业微信审批回调
// @Description 处理企业微信卡片按钮点击回调
// @Tags 回调
// @Accept json
// @Produce json
// @Success 200 {object} map[string]any "成功"
// @Router /callback/wecom/approval [post]
func (h *CallbackHandler) HandleWeComCallback(c *gin.Context) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		callbackLog.WithError(err).Error("读取请求体失败")
		response.BadRequest(c, "读取请求体失败")
		return
	}

	msgSignature := c.Query("msg_signature")
	timestamp := c.Query("timestamp")
	nonce := c.Query("nonce")

	result, err := h.callbackHandler.HandleWeComCallback(c.Request.Context(), body, msgSignature, timestamp, nonce)
	if err != nil {
		callbackLog.WithError(err).Error("处理企业微信回调失败")
		response.InternalError(c, err.Error())
		return
	}

	c.JSON(200, result)
}
