package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"devops/internal/config"
	"devops/internal/service/deploy"
	"devops/internal/service/kubernetes"
	"devops/pkg/dto"
	apperrors "devops/pkg/errors"
	"devops/pkg/ioc"
	"devops/pkg/middleware"
)

func init() {
	ioc.Api.RegisterContainer("DeployCheckHandler", &DeployCheckApiHandler{})
}

type DeployCheckApiHandler struct {
	handler *DeployCheckHandler
}

func (h *DeployCheckApiHandler) Init() error {
	cfg, _ := config.LoadConfig()
	db := cfg.GetDB()
	clientMgr := kubernetes.NewK8sClientManager(db)
	checkSvc := deploy.NewDeployCheckService(db, clientMgr)
	canarySvc := deploy.NewCanaryService(db, clientMgr)
	rollbackSvc := deploy.NewAutoRollbackService(db, clientMgr)
	h.handler = NewDeployCheckHandler(checkSvc, canarySvc, rollbackSvc)

	root := cfg.Application.GinRootRouter().Group("deploy")
	root.Use(middleware.AuthMiddleware())
	h.Register(root)

	return nil
}

func (h *DeployCheckApiHandler) Register(r gin.IRouter) {
	// 部署前置检查
	r.POST("/pre-check", h.handler.PreCheck)

	// 灰度发布
	canary := r.Group("/canary")
	canary.GET("/list", h.handler.ListCanary)
	canary.POST("/start", h.handler.StartCanary)
	canary.GET("/:record_id/status", h.handler.GetCanaryStatus)
	canary.POST("/:record_id/adjust", h.handler.AdjustCanary)
	canary.POST("/:record_id/promote", h.handler.PromoteCanary)
	canary.POST("/:record_id/rollback", h.handler.RollbackCanary)

	// 自动回滚 - 使用 auto-rollback 避免与现有 rollback 路由冲突
	autoRollback := r.Group("/auto-rollback")
	autoRollback.GET("/:record_id/health", h.handler.GetDeployHealth)
	autoRollback.POST("/:record_id/config", h.handler.UpdateRollbackConfig)
}

type DeployCheckHandler struct {
	checkSvc    *deploy.DeployCheckService
	canarySvc   *deploy.CanaryService
	rollbackSvc *deploy.AutoRollbackService
}

func NewDeployCheckHandler(checkSvc *deploy.DeployCheckService, canarySvc *deploy.CanaryService, rollbackSvc *deploy.AutoRollbackService) *DeployCheckHandler {
	return &DeployCheckHandler{
		checkSvc:    checkSvc,
		canarySvc:   canarySvc,
		rollbackSvc: rollbackSvc,
	}
}

// PreCheck 部署前置检查
func (h *DeployCheckHandler) PreCheck(c *gin.Context) {
	var req dto.DeployPreCheckRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": apperrors.ErrCodeInvalidParams, "message": "请填写完整的检查配置信息"})
		return
	}

	result, err := h.checkSvc.PreCheck(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": apperrors.ErrCodeK8sDeploy, "message": "部署前置检查失败，请检查集群连接和资源配置"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": apperrors.Success, "message": "success", "data": result})
}

// StartCanary 开始灰度发布
func (h *DeployCheckHandler) StartCanary(c *gin.Context) {
	var req dto.CanaryDeployRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": apperrors.ErrCodeInvalidParams, "message": "请填写完整的灰度发布配置"})
		return
	}

	result, err := h.canarySvc.StartCanary(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": apperrors.ErrCodeK8sDeploy, "message": "启动灰度发布失败，请检查应用配置和集群状态"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": apperrors.Success, "message": "灰度发布已启动", "data": result})
}

// GetCanaryStatus 获取灰度状态
func (h *DeployCheckHandler) GetCanaryStatus(c *gin.Context) {
	recordID, err := strconv.ParseUint(c.Param("record_id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": apperrors.ErrCodeInvalidParams, "message": "记录ID格式不正确"})
		return
	}

	result, err := h.canarySvc.GetCanaryStatus(c.Request.Context(), uint(recordID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": apperrors.ErrCodeNotFound, "message": "灰度发布记录不存在或已过期"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": apperrors.Success, "message": "success", "data": result})
}

// PromoteCanary 灰度全量发布
func (h *DeployCheckHandler) PromoteCanary(c *gin.Context) {
	recordID, err := strconv.ParseUint(c.Param("record_id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": apperrors.ErrCodeInvalidParams, "message": "记录ID格式不正确"})
		return
	}

	if err := h.canarySvc.Promote(c.Request.Context(), uint(recordID)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": apperrors.ErrCodeK8sDeploy, "message": "全量发布失败，请检查灰度状态和集群连接"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": apperrors.Success, "message": "全量发布成功"})
}

// RollbackCanary 灰度回滚
func (h *DeployCheckHandler) RollbackCanary(c *gin.Context) {
	recordID, err := strconv.ParseUint(c.Param("record_id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": apperrors.ErrCodeInvalidParams, "message": "记录ID格式不正确"})
		return
	}

	var req dto.CanaryRollbackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		req.Reason = ""
	}

	if err := h.canarySvc.Rollback(c.Request.Context(), uint(recordID), req.Reason); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": apperrors.ErrCodeK8sDeploy, "message": "回滚失败，请检查灰度状态和集群连接"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": apperrors.Success, "message": "回滚成功"})
}

// ListCanary 灰度发布列表
func (h *DeployCheckHandler) ListCanary(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	appID, _ := strconv.Atoi(c.Query("application_id"))
	envName := c.Query("env_name")
	status := c.Query("status")

	result, total, err := h.canarySvc.ListCanary(c.Request.Context(), page, pageSize, uint(appID), envName, status)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": apperrors.ErrCodeDBQuery, "message": "获取灰度列表失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    apperrors.Success,
		"message": "success",
		"data": gin.H{
			"list":  result,
			"total": total,
		},
	})
}

// AdjustCanary 调整灰度比例
func (h *DeployCheckHandler) AdjustCanary(c *gin.Context) {
	recordID, err := strconv.ParseUint(c.Param("record_id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": apperrors.ErrCodeInvalidParams, "message": "记录ID格式不正确"})
		return
	}

	var req struct {
		Percent int `json:"percent" binding:"required,min=1,max=100"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": apperrors.ErrCodeInvalidParams, "message": "灰度比例必须在1-100之间"})
		return
	}

	if err := h.canarySvc.AdjustCanary(c.Request.Context(), uint(recordID), req.Percent); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": apperrors.ErrCodeK8sDeploy, "message": "调整灰度比例失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": apperrors.Success, "message": "灰度比例已调整"})
}

// GetDeployHealth 获取部署健康状态
func (h *DeployCheckHandler) GetDeployHealth(c *gin.Context) {
	recordID, err := strconv.ParseUint(c.Param("record_id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": apperrors.ErrCodeInvalidParams, "message": "记录ID格式不正确"})
		return
	}

	result, err := h.rollbackSvc.GetHealthStatus(c.Request.Context(), uint(recordID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": apperrors.ErrCodeNotFound, "message": "部署记录不存在或健康检查配置未启用"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": apperrors.Success, "message": "success", "data": result})
}

// UpdateRollbackConfig 更新自动回滚配置
func (h *DeployCheckHandler) UpdateRollbackConfig(c *gin.Context) {
	recordID, err := strconv.ParseUint(c.Param("record_id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": apperrors.ErrCodeInvalidParams, "message": "记录ID格式不正确"})
		return
	}

	var req dto.AutoRollbackConfig
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": apperrors.ErrCodeInvalidParams, "message": "请填写完整的回滚配置信息"})
		return
	}

	if err := h.rollbackSvc.UpdateConfig(c.Request.Context(), uint(recordID), &req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": apperrors.ErrCodeDBQuery, "message": "更新回滚配置失败，请稍后重试"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": apperrors.Success, "message": "回滚配置更新成功"})
}
