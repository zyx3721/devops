package handler

import (
	"devops/internal/service/pipeline"
	"devops/pkg/logger"
	"devops/pkg/response"
	"io"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// WebhookHandler Webhook 处理器
type WebhookHandler struct {
	webhookService *pipeline.WebhookService
	gitService     *pipeline.GitService
}

// NewWebhookHandler 创建 Webhook 处理器
func NewWebhookHandler(db *gorm.DB, runService *pipeline.RunService) *WebhookHandler {
	return &WebhookHandler{
		webhookService: pipeline.NewWebhookService(db, runService),
		gitService:     pipeline.NewGitService(db),
	}
}

// RegisterRoutes 注册路由
func (h *WebhookHandler) RegisterRoutes(r *gin.RouterGroup) {
	webhook := r.Group("/webhook")
	{
		webhook.POST("/github/:repoId", h.HandleGitHub)
		webhook.POST("/gitlab/:repoId", h.HandleGitLab)
		webhook.POST("/gitee/:repoId", h.HandleGitee)
	}
}

// HandleGitHub 处理 GitHub Webhook
// @Summary GitHub Webhook
// @Tags Webhook
// @Accept json
// @Produce json
// @Param repoId path int true "仓库ID"
// @Success 200 {object} response.Response
// @Router /api/v1/webhook/github/{repoId} [post]
func (h *WebhookHandler) HandleGitHub(c *gin.Context) {
	log := logger.L().WithField("handler", "webhook").WithField("provider", "github")

	repoID, err := strconv.ParseUint(c.Param("repoId"), 10, 64)
	if err != nil {
		response.BadRequest(c, "无效的仓库ID")
		return
	}

	// 读取 payload
	payload, err := io.ReadAll(c.Request.Body)
	if err != nil {
		log.WithField("error", err).Error("读取 payload 失败")
		response.BadRequest(c, "读取请求体失败")
		return
	}

	// 获取仓库信息验证签名
	repo, err := h.gitService.GetByID(c.Request.Context(), uint(repoID))
	if err != nil {
		log.WithField("error", err).Error("获取仓库信息失败")
		response.NotFound(c, "仓库不存在")
		return
	}

	// 验证签名
	signature := c.GetHeader("X-Hub-Signature-256")
	if !h.webhookService.VerifyGitHubSignature(payload, signature, repo.WebhookSecret) {
		log.Warn("签名验证失败")
		response.Unauthorized(c, "签名验证失败")
		return
	}

	// 解析 Webhook
	headers := map[string]string{
		"X-GitHub-Event":    c.GetHeader("X-GitHub-Event"),
		"X-GitHub-Delivery": c.GetHeader("X-GitHub-Delivery"),
	}

	wp, err := h.webhookService.HandleGitHubWebhook(c.Request.Context(), payload, headers)
	if err != nil {
		log.WithField("error", err).Error("解析 Webhook 失败")
		h.webhookService.SaveWebhookLog(c.Request.Context(), uint(repoID), wp, 0, err)
		response.BadRequest(c, "解析 Webhook 失败")
		return
	}

	// 触发流水线
	run, err := h.webhookService.TriggerPipeline(c.Request.Context(), uint(repoID), wp)
	var runID uint
	if run != nil {
		runID = run.ID
	}

	// 保存日志
	h.webhookService.SaveWebhookLog(c.Request.Context(), uint(repoID), wp, runID, err)

	if err != nil {
		log.WithField("error", err).Error("触发流水线失败")
		response.InternalError(c, "触发流水线失败")
		return
	}

	if run != nil {
		response.Success(c, gin.H{"run_id": run.ID, "message": "流水线已触发"})
	} else {
		response.Success(c, gin.H{"message": "Webhook 已接收，无匹配的触发条件"})
	}
}

// HandleGitLab 处理 GitLab Webhook
// @Summary GitLab Webhook
// @Tags Webhook
// @Accept json
// @Produce json
// @Param repoId path int true "仓库ID"
// @Success 200 {object} response.Response
// @Router /api/v1/webhook/gitlab/{repoId} [post]
func (h *WebhookHandler) HandleGitLab(c *gin.Context) {
	log := logger.L().WithField("handler", "webhook").WithField("provider", "gitlab")

	repoID, err := strconv.ParseUint(c.Param("repoId"), 10, 64)
	if err != nil {
		response.BadRequest(c, "无效的仓库ID")
		return
	}

	payload, err := io.ReadAll(c.Request.Body)
	if err != nil {
		log.WithField("error", err).Error("读取 payload 失败")
		response.BadRequest(c, "读取请求体失败")
		return
	}

	repo, err := h.gitService.GetByID(c.Request.Context(), uint(repoID))
	if err != nil {
		response.NotFound(c, "仓库不存在")
		return
	}

	// 验证 Token
	token := c.GetHeader("X-Gitlab-Token")
	if !h.webhookService.VerifyGitLabToken(token, repo.WebhookSecret) {
		response.Unauthorized(c, "Token 验证失败")
		return
	}

	headers := map[string]string{
		"X-Gitlab-Event": c.GetHeader("X-Gitlab-Event"),
		"X-Gitlab-Token": token,
	}

	wp, err := h.webhookService.HandleGitLabWebhook(c.Request.Context(), payload, headers)
	if err != nil {
		h.webhookService.SaveWebhookLog(c.Request.Context(), uint(repoID), wp, 0, err)
		response.BadRequest(c, "解析 Webhook 失败")
		return
	}

	run, err := h.webhookService.TriggerPipeline(c.Request.Context(), uint(repoID), wp)
	var runID uint
	if run != nil {
		runID = run.ID
	}
	h.webhookService.SaveWebhookLog(c.Request.Context(), uint(repoID), wp, runID, err)

	if run != nil {
		response.Success(c, gin.H{"run_id": run.ID})
	} else {
		response.Success(c, gin.H{"message": "已接收"})
	}
}

// HandleGitee 处理 Gitee Webhook
// @Summary Gitee Webhook
// @Tags Webhook
// @Accept json
// @Produce json
// @Param repoId path int true "仓库ID"
// @Success 200 {object} response.Response
// @Router /api/v1/webhook/gitee/{repoId} [post]
func (h *WebhookHandler) HandleGitee(c *gin.Context) {
	log := logger.L().WithField("handler", "webhook").WithField("provider", "gitee")

	repoID, err := strconv.ParseUint(c.Param("repoId"), 10, 64)
	if err != nil {
		response.BadRequest(c, "无效的仓库ID")
		return
	}

	payload, err := io.ReadAll(c.Request.Body)
	if err != nil {
		log.WithField("error", err).Error("读取 payload 失败")
		response.BadRequest(c, "读取请求体失败")
		return
	}

	repo, err := h.gitService.GetByID(c.Request.Context(), uint(repoID))
	if err != nil {
		response.NotFound(c, "仓库不存在")
		return
	}

	// 验证签名
	signature := c.GetHeader("X-Gitee-Token")
	if !h.webhookService.VerifyGiteeSignature(payload, signature, repo.WebhookSecret) {
		response.Unauthorized(c, "签名验证失败")
		return
	}

	headers := map[string]string{
		"X-Gitee-Event": c.GetHeader("X-Gitee-Event"),
		"X-Gitee-Token": signature,
	}

	wp, err := h.webhookService.HandleGiteeWebhook(c.Request.Context(), payload, headers)
	if err != nil {
		h.webhookService.SaveWebhookLog(c.Request.Context(), uint(repoID), wp, 0, err)
		response.BadRequest(c, "解析 Webhook 失败")
		return
	}

	run, err := h.webhookService.TriggerPipeline(c.Request.Context(), uint(repoID), wp)
	var runID uint
	if run != nil {
		runID = run.ID
	}
	h.webhookService.SaveWebhookLog(c.Request.Context(), uint(repoID), wp, runID, err)

	if run != nil {
		response.Success(c, gin.H{"run_id": run.ID})
	} else {
		response.Success(c, gin.H{"message": "已接收"})
	}
}
