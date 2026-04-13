package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"devops/internal/config"
	"devops/internal/models"
	"devops/internal/service/security"
	"devops/pkg/dto"
	"devops/pkg/ioc"
	"devops/pkg/middleware"
	"devops/pkg/response"
)

func init() {
	ioc.Api.RegisterContainer("SecurityHandler", &SecurityApiHandler{})
}

// SecurityApiHandler IOC容器注册的处理器
type SecurityApiHandler struct {
	handler *SecurityHandler
}

func (h *SecurityApiHandler) Init() error {
	cfg, err := config.LoadConfig()
	if err != nil {
		return err
	}

	db := cfg.GetDB()
	svc := security.NewSecurityService(db)
	registrySvc := security.NewRegistryService(db)
	ruleSvc := security.NewRuleService(db)

	h.handler = NewSecurityHandler(svc, registrySvc, ruleSvc)

	root := cfg.Application.GinRootRouter().Group("security")
	root.Use(middleware.AuthMiddleware())
	h.Register(root)

	return nil
}

func (h *SecurityApiHandler) Register(r gin.IRouter) {
	// 安全概览
	r.GET("/overview", h.handler.GetOverview)

	// 镜像扫描
	r.POST("/scan", h.handler.ScanImage)
	r.GET("/scans", h.handler.GetScanHistory)
	r.GET("/scans/:id", h.handler.GetScanResult)

	// 镜像仓库
	r.GET("/registries", h.handler.ListRegistries)
	r.POST("/registries", middleware.RequireAdmin(), h.handler.CreateRegistry)
	r.PUT("/registries/:id", middleware.RequireAdmin(), h.handler.UpdateRegistry)
	r.DELETE("/registries/:id", middleware.RequireAdmin(), h.handler.DeleteRegistry)
	r.POST("/registries/test", h.handler.TestRegistryConnection)
	r.GET("/registries/:id/images", h.handler.ListRegistryImages)

	// 配置检查
	r.POST("/config-check", h.handler.RunConfigCheck)
	r.GET("/config-checks", h.handler.GetConfigCheckHistory)
	r.GET("/config-checks/:id", h.handler.GetConfigCheckResult)

	// 合规规则
	r.GET("/rules", h.handler.ListRules)
	r.POST("/rules", middleware.RequireAdmin(), h.handler.CreateRule)
	r.PUT("/rules/:id", middleware.RequireAdmin(), h.handler.UpdateRule)
	r.DELETE("/rules/:id", middleware.RequireAdmin(), h.handler.DeleteRule)
	r.POST("/rules/:id/toggle", middleware.RequireAdmin(), h.handler.ToggleRule)

	// 审计日志
	r.GET("/audit-logs", h.handler.ListAuditLogs)
	r.GET("/audit-logs/export", h.handler.ExportAuditLogs)
}

// SecurityHandler 安全处理器
type SecurityHandler struct {
	svc         *security.SecurityService
	registrySvc *security.RegistryService
	ruleSvc     *security.RuleService
}

// NewSecurityHandler 创建安全处理器
func NewSecurityHandler(svc *security.SecurityService, registrySvc *security.RegistryService, ruleSvc *security.RuleService) *SecurityHandler {
	return &SecurityHandler{
		svc:         svc,
		registrySvc: registrySvc,
		ruleSvc:     ruleSvc,
	}
}

// GetOverview 获取安全概览
func (h *SecurityHandler) GetOverview(c *gin.Context) {
	clusterID, _ := strconv.ParseUint(c.Query("cluster_id"), 10, 64)

	result, err := h.svc.GetOverview(c.Request.Context(), uint(clusterID))
	if err != nil {
		response.FromError(c, err)
		return
	}

	response.Success(c, result)
}

// ScanImage 扫描镜像
func (h *SecurityHandler) ScanImage(c *gin.Context) {
	var req dto.ScanImageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	result, err := h.svc.GetImageScanner().ScanImage(c.Request.Context(), &req)
	if err != nil {
		response.FromError(c, err)
		return
	}

	// 记录审计日志
	h.svc.GetAuditLogger().Log(c.Request.Context(), &models.SecurityAuditLog{
		UserID:       getUserID(c),
		Username:     getUsername(c),
		Action:       "scan",
		ResourceType: "image",
		ResourceName: req.Image,
		Result:       "success",
		ClientIP:     c.ClientIP(),
	})

	response.Success(c, result)
}

// GetScanHistory 获取扫描历史
func (h *SecurityHandler) GetScanHistory(c *gin.Context) {
	var req dto.ScanHistoryRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	result, err := h.svc.GetImageScanner().GetScanHistory(c.Request.Context(), &req)
	if err != nil {
		response.FromError(c, err)
		return
	}

	response.Success(c, result)
}

// GetScanResult 获取扫描结果
func (h *SecurityHandler) GetScanResult(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "无效的ID")
		return
	}

	result, err := h.svc.GetImageScanner().GetScanResult(c.Request.Context(), uint(id))
	if err != nil {
		response.FromError(c, err)
		return
	}

	response.Success(c, result)
}

// ListRegistries 获取仓库列表
func (h *SecurityHandler) ListRegistries(c *gin.Context) {
	result, err := h.registrySvc.List(c.Request.Context())
	if err != nil {
		response.FromError(c, err)
		return
	}

	response.Success(c, result)
}

// CreateRegistry 创建仓库
func (h *SecurityHandler) CreateRegistry(c *gin.Context) {
	var req dto.ImageRegistryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	if err := h.registrySvc.Create(c.Request.Context(), &req); err != nil {
		response.FromError(c, err)
		return
	}

	response.SuccessWithMessage(c, "创建成功", nil)
}

// UpdateRegistry 更新仓库
func (h *SecurityHandler) UpdateRegistry(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "无效的ID")
		return
	}

	var req dto.ImageRegistryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}
	req.ID = uint(id)

	if err := h.registrySvc.Update(c.Request.Context(), &req); err != nil {
		response.FromError(c, err)
		return
	}

	response.SuccessWithMessage(c, "更新成功", nil)
}

// DeleteRegistry 删除仓库
func (h *SecurityHandler) DeleteRegistry(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "无效的ID")
		return
	}

	if err := h.registrySvc.Delete(c.Request.Context(), uint(id)); err != nil {
		response.FromError(c, err)
		return
	}

	response.SuccessWithMessage(c, "删除成功", nil)
}

// TestRegistryConnection 测试仓库连接
func (h *SecurityHandler) TestRegistryConnection(c *gin.Context) {
	var req dto.ImageRegistryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	if err := h.registrySvc.TestConnection(c.Request.Context(), &req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	response.SuccessWithMessage(c, "连接成功", nil)
}

// ListRegistryImages 列出仓库镜像
func (h *SecurityHandler) ListRegistryImages(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "无效的ID")
		return
	}

	result, err := h.registrySvc.ListImages(c.Request.Context(), uint(id))
	if err != nil {
		response.FromError(c, err)
		return
	}

	response.Success(c, result)
}

// RunConfigCheck 运行配置检查
func (h *SecurityHandler) RunConfigCheck(c *gin.Context) {
	var req dto.ConfigCheckRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	result, err := h.svc.GetConfigChecker().RunCheck(c.Request.Context(), &req)
	if err != nil {
		response.FromError(c, err)
		return
	}

	response.Success(c, result)
}

// GetConfigCheckHistory 获取配置检查历史
func (h *SecurityHandler) GetConfigCheckHistory(c *gin.Context) {
	clusterID, _ := strconv.ParseUint(c.Query("cluster_id"), 10, 64)
	page, _ := strconv.Atoi(c.Query("page"))
	pageSize, _ := strconv.Atoi(c.Query("page_size"))

	result, err := h.svc.GetConfigChecker().GetCheckHistory(c.Request.Context(), uint(clusterID), page, pageSize)
	if err != nil {
		response.FromError(c, err)
		return
	}

	if result == nil {
		response.Success(c, &dto.ConfigCheckHistoryResponse{Total: 0, Items: []dto.ConfigCheckItem{}})
		return
	}

	response.Success(c, result)
}

// GetConfigCheckResult 获取配置检查结果
func (h *SecurityHandler) GetConfigCheckResult(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "无效的ID")
		return
	}

	result, err := h.svc.GetConfigChecker().GetCheckResult(c.Request.Context(), uint(id))
	if err != nil {
		response.FromError(c, err)
		return
	}

	response.Success(c, result)
}

// ListRules 获取规则列表
func (h *SecurityHandler) ListRules(c *gin.Context) {
	category := c.Query("category")
	var enabled *bool
	if e := c.Query("enabled"); e != "" {
		b := e == "true"
		enabled = &b
	}

	result, err := h.ruleSvc.List(c.Request.Context(), category, enabled)
	if err != nil {
		response.FromError(c, err)
		return
	}

	response.Success(c, result)
}

// CreateRule 创建规则
func (h *SecurityHandler) CreateRule(c *gin.Context) {
	var req dto.ComplianceRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	if err := h.ruleSvc.Create(c.Request.Context(), &req); err != nil {
		response.FromError(c, err)
		return
	}

	response.SuccessWithMessage(c, "创建成功", nil)
}

// UpdateRule 更新规则
func (h *SecurityHandler) UpdateRule(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "无效的ID")
		return
	}

	var req dto.ComplianceRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}
	req.ID = uint(id)

	if err := h.ruleSvc.Update(c.Request.Context(), &req); err != nil {
		response.FromError(c, err)
		return
	}

	response.SuccessWithMessage(c, "更新成功", nil)
}

// DeleteRule 删除规则
func (h *SecurityHandler) DeleteRule(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "无效的ID")
		return
	}

	if err := h.ruleSvc.Delete(c.Request.Context(), uint(id)); err != nil {
		response.FromError(c, err)
		return
	}

	response.SuccessWithMessage(c, "删除成功", nil)
}

// ToggleRule 切换规则启用状态
func (h *SecurityHandler) ToggleRule(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "无效的ID")
		return
	}

	if err := h.ruleSvc.ToggleEnabled(c.Request.Context(), uint(id)); err != nil {
		response.FromError(c, err)
		return
	}

	response.SuccessWithMessage(c, "操作成功", nil)
}

// ListAuditLogs 获取审计日志
func (h *SecurityHandler) ListAuditLogs(c *gin.Context) {
	var req dto.AuditLogRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	result, err := h.svc.GetAuditLogger().List(c.Request.Context(), &req)
	if err != nil {
		response.FromError(c, err)
		return
	}

	response.Success(c, result)
}

// ExportAuditLogs 导出审计日志
func (h *SecurityHandler) ExportAuditLogs(c *gin.Context) {
	var req dto.AuditLogRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	format := c.DefaultQuery("format", "csv")

	data, contentType, err := h.svc.GetAuditLogger().Export(c.Request.Context(), &req, format)
	if err != nil {
		response.FromError(c, err)
		return
	}

	filename := "audit_logs." + format
	c.Header("Content-Type", contentType)
	c.Header("Content-Disposition", "attachment; filename="+filename)
	c.Data(200, contentType, data)
}

// 辅助函数
func getUserID(c *gin.Context) *uint {
	if id, exists := c.Get("userID"); exists {
		uid := id.(uint)
		return &uid
	}
	return nil
}

func getUsername(c *gin.Context) string {
	if name, exists := c.Get("username"); exists {
		return name.(string)
	}
	return ""
}
