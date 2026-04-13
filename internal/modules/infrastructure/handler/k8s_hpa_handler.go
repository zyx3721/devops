package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"devops/internal/config"
	"devops/internal/service/kubernetes"
	"devops/pkg/dto"
	apperrors "devops/pkg/errors"
	"devops/pkg/ioc"
	"devops/pkg/middleware"
)

func init() {
	ioc.Api.RegisterContainer("K8sHPAHandler", &K8sHPAApiHandler{})
}

type K8sHPAApiHandler struct {
	handler     *K8sHPAHandler
	cronHandler *CronHPAHandler
}

func (h *K8sHPAApiHandler) Init() error {
	cfg, _ := config.LoadConfig()
	db := cfg.GetDB()
	clientMgr := kubernetes.NewK8sClientManager(db)
	svc := kubernetes.NewK8sHPAService(clientMgr)
	cronSvc := kubernetes.NewCronHPAService(db, clientMgr)
	h.handler = NewK8sHPAHandler(svc)
	h.cronHandler = NewCronHPAHandler(cronSvc)

	root := cfg.Application.GinRootRouter().Group("k8s")
	root.Use(middleware.AuthMiddleware())
	h.Register(root)

	return nil
}

func (h *K8sHPAApiHandler) Register(r gin.IRouter) {
	// HPA 管理
	hpa := r.Group("/clusters/:id/hpa")
	hpa.GET("", h.handler.ListHPAs)
	hpa.GET("/:namespace/:name", h.handler.GetHPA)
	hpa.POST("", middleware.RequireAdmin(), h.handler.CreateHPA)
	hpa.PUT("/:namespace/:name", middleware.RequireAdmin(), h.handler.UpdateHPA)
	hpa.DELETE("/:namespace/:name", middleware.RequireAdmin(), h.handler.DeleteHPA)

	// CronHPA 管理
	cronHPA := r.Group("/clusters/:id/cron-hpa")
	cronHPA.GET("", h.cronHandler.ListCronHPAs)
	cronHPA.GET("/:namespace/:name", h.cronHandler.GetCronHPA)
	cronHPA.POST("", middleware.RequireAdmin(), h.cronHandler.CreateCronHPA)
	cronHPA.PUT("/:namespace/:name", middleware.RequireAdmin(), h.cronHandler.UpdateCronHPA)
	cronHPA.DELETE("/:namespace/:name", middleware.RequireAdmin(), h.cronHandler.DeleteCronHPA)

	// ResourceQuota 管理
	quota := r.Group("/clusters/:id/quotas")
	quota.GET("", h.handler.ListResourceQuotas)
	quota.POST("", middleware.RequireAdmin(), h.handler.CreateResourceQuota)
	quota.DELETE("/:namespace/:name", middleware.RequireAdmin(), h.handler.DeleteResourceQuota)

	// LimitRange 管理
	lr := r.Group("/clusters/:id/limitranges")
	lr.GET("", h.handler.ListLimitRanges)
}

type K8sHPAHandler struct {
	svc *kubernetes.K8sHPAService
}

func NewK8sHPAHandler(svc *kubernetes.K8sHPAService) *K8sHPAHandler {
	return &K8sHPAHandler{svc: svc}
}

// ListHPAs 获取 HPA 列表
func (h *K8sHPAHandler) ListHPAs(c *gin.Context) {
	clusterID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": apperrors.ErrCodeInvalidParams, "message": "集群ID格式不正确"})
		return
	}

	namespace := c.DefaultQuery("namespace", "")
	result, err := h.svc.ListHPAs(c.Request.Context(), uint(clusterID), namespace)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": apperrors.ErrCodeK8sDeploy, "message": "获取HPA列表失败，请检查集群连接"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": apperrors.Success, "message": "success", "data": result})
}

// GetHPA 获取单个 HPA
func (h *K8sHPAHandler) GetHPA(c *gin.Context) {
	clusterID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": apperrors.ErrCodeInvalidParams, "message": "集群ID格式不正确"})
		return
	}

	namespace := c.Param("namespace")
	name := c.Param("name")

	result, err := h.svc.GetHPA(c.Request.Context(), uint(clusterID), namespace, name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": apperrors.ErrCodeNotFound, "message": "HPA不存在或已被删除"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": apperrors.Success, "message": "success", "data": result})
}

// CreateHPA 创建 HPA
func (h *K8sHPAHandler) CreateHPA(c *gin.Context) {
	clusterID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": apperrors.ErrCodeInvalidParams, "message": "集群ID格式不正确"})
		return
	}

	var req dto.CreateHPARequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": apperrors.ErrCodeInvalidParams, "message": "请填写完整的HPA配置信息"})
		return
	}

	if err := h.svc.CreateHPA(c.Request.Context(), uint(clusterID), &req); err != nil {
		errMsg := "创建HPA失败"
		if contains(err.Error(), "already exists") {
			errMsg = "HPA已存在，请勿重复创建"
		} else if contains(err.Error(), "not found") {
			errMsg = "目标资源不存在，请检查Deployment/StatefulSet名称"
		}
		c.JSON(http.StatusInternalServerError, gin.H{"code": apperrors.ErrCodeK8sDeploy, "message": errMsg})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": apperrors.Success, "message": "HPA创建成功"})
}

// UpdateHPA 更新 HPA
func (h *K8sHPAHandler) UpdateHPA(c *gin.Context) {
	clusterID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": apperrors.ErrCodeInvalidParams, "message": "集群ID格式不正确"})
		return
	}

	namespace := c.Param("namespace")
	name := c.Param("name")

	var req dto.UpdateHPARequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": apperrors.ErrCodeInvalidParams, "message": "请填写完整的HPA配置信息"})
		return
	}

	if err := h.svc.UpdateHPA(c.Request.Context(), uint(clusterID), namespace, name, &req); err != nil {
		errMsg := "更新HPA失败"
		if contains(err.Error(), "not found") {
			errMsg = "HPA不存在或已被删除"
		}
		c.JSON(http.StatusInternalServerError, gin.H{"code": apperrors.ErrCodeK8sDeploy, "message": errMsg})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": apperrors.Success, "message": "HPA更新成功"})
}

// DeleteHPA 删除 HPA
func (h *K8sHPAHandler) DeleteHPA(c *gin.Context) {
	clusterID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": apperrors.ErrCodeInvalidParams, "message": "集群ID格式不正确"})
		return
	}

	namespace := c.Param("namespace")
	name := c.Param("name")

	if err := h.svc.DeleteHPA(c.Request.Context(), uint(clusterID), namespace, name); err != nil {
		errMsg := "删除HPA失败"
		if contains(err.Error(), "not found") {
			errMsg = "HPA不存在或已被删除"
		}
		c.JSON(http.StatusInternalServerError, gin.H{"code": apperrors.ErrCodeK8sDeploy, "message": errMsg})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": apperrors.Success, "message": "HPA删除成功"})
}

// ListResourceQuotas 获取资源配额列表
func (h *K8sHPAHandler) ListResourceQuotas(c *gin.Context) {
	clusterID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": apperrors.ErrCodeInvalidParams, "message": "集群ID格式不正确"})
		return
	}

	namespace := c.DefaultQuery("namespace", "")
	result, err := h.svc.ListResourceQuotas(c.Request.Context(), uint(clusterID), namespace)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": apperrors.ErrCodeK8sDeploy, "message": "获取资源配额失败，请检查集群连接"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": apperrors.Success, "message": "success", "data": result})
}

// CreateResourceQuota 创建资源配额
func (h *K8sHPAHandler) CreateResourceQuota(c *gin.Context) {
	clusterID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": apperrors.ErrCodeInvalidParams, "message": "集群ID格式不正确"})
		return
	}

	var req dto.CreateResourceQuotaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": apperrors.ErrCodeInvalidParams, "message": "请填写完整的配额配置信息"})
		return
	}

	if err := h.svc.CreateResourceQuota(c.Request.Context(), uint(clusterID), &req); err != nil {
		errMsg := "创建资源配额失败"
		if contains(err.Error(), "already exists") {
			errMsg = "资源配额已存在，请勿重复创建"
		} else if contains(err.Error(), "invalid") {
			errMsg = "资源配额格式不正确，请检查配置"
		}
		c.JSON(http.StatusInternalServerError, gin.H{"code": apperrors.ErrCodeK8sDeploy, "message": errMsg})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": apperrors.Success, "message": "资源配额创建成功"})
}

// DeleteResourceQuota 删除资源配额
func (h *K8sHPAHandler) DeleteResourceQuota(c *gin.Context) {
	clusterID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": apperrors.ErrCodeInvalidParams, "message": "集群ID格式不正确"})
		return
	}

	namespace := c.Param("namespace")
	name := c.Param("name")

	if err := h.svc.DeleteResourceQuota(c.Request.Context(), uint(clusterID), namespace, name); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": apperrors.ErrCodeK8sDeploy, "message": "删除资源配额失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": apperrors.Success, "message": "资源配额删除成功"})
}

// ListLimitRanges 获取 LimitRange 列表
func (h *K8sHPAHandler) ListLimitRanges(c *gin.Context) {
	clusterID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": apperrors.ErrCodeInvalidParams, "message": "集群ID格式不正确"})
		return
	}

	namespace := c.DefaultQuery("namespace", "")
	result, err := h.svc.ListLimitRanges(c.Request.Context(), uint(clusterID), namespace)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": apperrors.ErrCodeK8sDeploy, "message": "获取LimitRange失败，请检查集群连接"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": apperrors.Success, "message": "success", "data": result})
}

// contains 检查字符串是否包含子串
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsLower(s, substr))
}

func containsLower(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if equalFoldAt(s, i, substr) {
			return true
		}
	}
	return false
}

func equalFoldAt(s string, i int, substr string) bool {
	for j := 0; j < len(substr); j++ {
		c1 := s[i+j]
		c2 := substr[j]
		if c1 != c2 && toLower(c1) != toLower(c2) {
			return false
		}
	}
	return true
}

func toLower(c byte) byte {
	if c >= 'A' && c <= 'Z' {
		return c + 32
	}
	return c
}

// ==================== CronHPA Handler ====================

type CronHPAHandler struct {
	svc *kubernetes.CronHPAService
}

func NewCronHPAHandler(svc *kubernetes.CronHPAService) *CronHPAHandler {
	return &CronHPAHandler{svc: svc}
}

// ListCronHPAs 获取 CronHPA 列表
func (h *CronHPAHandler) ListCronHPAs(c *gin.Context) {
	clusterID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": apperrors.ErrCodeInvalidParams, "message": "集群ID格式不正确"})
		return
	}

	namespace := c.DefaultQuery("namespace", "")
	result, err := h.svc.ListCronHPAs(c.Request.Context(), uint(clusterID), namespace)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": apperrors.ErrCodeK8sDeploy, "message": "获取CronHPA列表失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": apperrors.Success, "message": "success", "data": result})
}

// GetCronHPA 获取单个 CronHPA
func (h *CronHPAHandler) GetCronHPA(c *gin.Context) {
	clusterID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": apperrors.ErrCodeInvalidParams, "message": "集群ID格式不正确"})
		return
	}

	namespace := c.Param("namespace")
	name := c.Param("name")

	result, err := h.svc.GetCronHPA(c.Request.Context(), uint(clusterID), namespace, name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": apperrors.ErrCodeNotFound, "message": "CronHPA不存在或已被删除"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": apperrors.Success, "message": "success", "data": result})
}

// CreateCronHPA 创建 CronHPA
func (h *CronHPAHandler) CreateCronHPA(c *gin.Context) {
	clusterID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": apperrors.ErrCodeInvalidParams, "message": "集群ID格式不正确"})
		return
	}

	var req dto.CreateCronHPARequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": apperrors.ErrCodeInvalidParams, "message": "请填写完整的CronHPA配置信息"})
		return
	}

	if err := h.svc.CreateCronHPA(c.Request.Context(), uint(clusterID), &req); err != nil {
		errMsg := "创建CronHPA失败"
		if contains(err.Error(), "无效的cron表达式") {
			errMsg = err.Error()
		}
		c.JSON(http.StatusInternalServerError, gin.H{"code": apperrors.ErrCodeInvalidParams, "message": errMsg})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": apperrors.Success, "message": "CronHPA创建成功"})
}

// UpdateCronHPA 更新 CronHPA
func (h *CronHPAHandler) UpdateCronHPA(c *gin.Context) {
	clusterID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": apperrors.ErrCodeInvalidParams, "message": "集群ID格式不正确"})
		return
	}

	namespace := c.Param("namespace")
	name := c.Param("name")

	var req dto.UpdateCronHPARequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": apperrors.ErrCodeInvalidParams, "message": "请填写完整的CronHPA配置信息"})
		return
	}

	if err := h.svc.UpdateCronHPA(c.Request.Context(), uint(clusterID), namespace, name, &req); err != nil {
		errMsg := "更新CronHPA失败"
		if contains(err.Error(), "无效的cron表达式") {
			errMsg = err.Error()
		} else if contains(err.Error(), "不存在") {
			errMsg = "CronHPA不存在或已被删除"
		}
		c.JSON(http.StatusInternalServerError, gin.H{"code": apperrors.ErrCodeK8sDeploy, "message": errMsg})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": apperrors.Success, "message": "CronHPA更新成功"})
}

// DeleteCronHPA 删除 CronHPA
func (h *CronHPAHandler) DeleteCronHPA(c *gin.Context) {
	clusterID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": apperrors.ErrCodeInvalidParams, "message": "集群ID格式不正确"})
		return
	}

	namespace := c.Param("namespace")
	name := c.Param("name")

	if err := h.svc.DeleteCronHPA(c.Request.Context(), uint(clusterID), namespace, name); err != nil {
		errMsg := "删除CronHPA失败"
		if contains(err.Error(), "不存在") {
			errMsg = "CronHPA不存在或已被删除"
		}
		c.JSON(http.StatusInternalServerError, gin.H{"code": apperrors.ErrCodeK8sDeploy, "message": errMsg})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": apperrors.Success, "message": "CronHPA删除成功"})
}
