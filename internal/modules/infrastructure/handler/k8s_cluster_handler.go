package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"devops/internal/config"
	"devops/internal/service/kubernetes"
	"devops/pkg/dto"
	apperrors "devops/pkg/errors"
	"devops/pkg/ioc"
	"devops/pkg/middleware"
)

func init() {
	ioc.Api.RegisterContainer("K8sClusterHandler", &K8sClusterApiHandler{})
}

type K8sClusterApiHandler struct {
	handler *K8sClusterHandler
}

func (h *K8sClusterApiHandler) Init() error {
	cfg, _ := config.LoadConfig()
	db := cfg.GetDB()
	svc := kubernetes.NewK8sClusterService(db)
	h.handler = NewK8sClusterHandler(svc, db)

	// 主路由 k8s-clusters
	root := cfg.Application.GinRootRouter().Group("k8s-clusters")
	root.Use(middleware.AuthMiddleware())
	h.Register(root)

	// 兼容路由 k8s/clusters
	compat := cfg.Application.GinRootRouter().Group("k8s/clusters")
	compat.Use(middleware.AuthMiddleware())
	h.Register(compat)

	return nil
}

func (h *K8sClusterApiHandler) Register(r gin.IRouter) {
	// 查看权限
	r.GET("", h.handler.GetK8sClusterList)
	r.GET("/default", h.handler.GetDefaultK8sCluster)
	r.GET("/:id", h.handler.GetK8sCluster)
	r.GET("/:id/feishu-apps", h.handler.GetFeishuApps)
	r.GET("/:id/namespaces", h.handler.GetNamespaces)
	r.GET("/:id/pods", h.handler.GetPods)

	// Istio 和服务相关
	r.GET("/:id/istio/status", h.handler.GetIstioStatus)
	r.GET("/:id/namespaces/:ns/services", h.handler.ListServices)

	// 管理权限（K8s集群是全局资源，使用超级管理员权限检查）
	r.POST("", middleware.RequireSuperAdmin(), h.handler.CreateK8sCluster)
	r.PUT("/:id", middleware.RequireSuperAdmin(), h.handler.UpdateK8sCluster)
	r.PUT("/:id/default", middleware.RequireSuperAdmin(), h.handler.SetDefaultK8sCluster)
	r.DELETE("/:id", middleware.RequireSuperAdmin(), h.handler.DeleteK8sCluster)
	r.POST("/:id/test-connection", middleware.RequireSuperAdmin(), h.handler.TestConnection)
	r.PUT("/:id/feishu-apps", middleware.RequireSuperAdmin(), h.handler.BindFeishuApps)
}

type K8sClusterHandler struct {
	svc kubernetes.K8sClusterService
	db  *gorm.DB
}

func NewK8sClusterHandler(svc kubernetes.K8sClusterService, db *gorm.DB) *K8sClusterHandler {
	return &K8sClusterHandler{svc: svc, db: db}
}

func (h *K8sClusterHandler) CreateK8sCluster(c *gin.Context) {
	var req dto.CreateK8sClusterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": apperrors.ErrCodeInvalidParams, "message": "参数错误", "error": err.Error()})
		return
	}

	result, err := h.svc.CreateK8sCluster(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": apperrors.ErrCodeInternalError, "message": "创建K8s集群失败", "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": apperrors.Success, "message": "创建成功", "data": result})
}

func (h *K8sClusterHandler) GetK8sCluster(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": apperrors.ErrCodeInvalidParams, "message": "ID格式错误"})
		return
	}

	result, err := h.svc.GetK8sCluster(c.Request.Context(), uint(id))
	if err != nil {
		if err == apperrors.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{"code": apperrors.ErrCodeNotFound, "message": "K8s集群不存在"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"code": apperrors.ErrCodeInternalError, "message": "查询K8s集群失败", "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": apperrors.Success, "message": "success", "data": result})
}

func (h *K8sClusterHandler) GetK8sClusterList(c *gin.Context) {
	var req dto.K8sClusterListRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": apperrors.ErrCodeInvalidParams, "message": "参数错误", "error": err.Error()})
		return
	}

	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 20
	}

	result, err := h.svc.GetK8sClusterList(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": apperrors.ErrCodeInternalError, "message": "查询K8s集群列表失败", "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": apperrors.Success, "message": "success", "data": result})
}

func (h *K8sClusterHandler) UpdateK8sCluster(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": apperrors.ErrCodeInvalidParams, "message": "ID格式错误"})
		return
	}

	var req dto.UpdateK8sClusterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": apperrors.ErrCodeInvalidParams, "message": "参数错误", "error": err.Error()})
		return
	}

	result, err := h.svc.UpdateK8sCluster(c.Request.Context(), uint(id), &req)
	if err != nil {
		if err == apperrors.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{"code": apperrors.ErrCodeNotFound, "message": "K8s集群不存在"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"code": apperrors.ErrCodeInternalError, "message": "更新K8s集群失败", "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": apperrors.Success, "message": "更新成功", "data": result})
}

func (h *K8sClusterHandler) DeleteK8sCluster(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": apperrors.ErrCodeInvalidParams, "message": "ID格式错误"})
		return
	}

	if err := h.svc.DeleteK8sCluster(c.Request.Context(), uint(id)); err != nil {
		if err == apperrors.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{"code": apperrors.ErrCodeNotFound, "message": "K8s集群不存在"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"code": apperrors.ErrCodeInternalError, "message": "删除K8s集群失败", "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": apperrors.Success, "message": "删除成功"})
}

func (h *K8sClusterHandler) SetDefaultK8sCluster(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": apperrors.ErrCodeInvalidParams, "message": "ID格式错误"})
		return
	}

	if err := h.svc.SetDefaultK8sCluster(c.Request.Context(), uint(id)); err != nil {
		if err == apperrors.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{"code": apperrors.ErrCodeNotFound, "message": "K8s集群不存在"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"code": apperrors.ErrCodeInternalError, "message": "设置默认K8s集群失败", "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": apperrors.Success, "message": "设置成功"})
}

func (h *K8sClusterHandler) GetDefaultK8sCluster(c *gin.Context) {
	result, err := h.svc.GetDefaultK8sCluster(c.Request.Context())
	if err != nil {
		if err == apperrors.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{"code": apperrors.ErrCodeNotFound, "message": "没有可用的K8s集群"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"code": apperrors.ErrCodeInternalError, "message": "查询默认K8s集群失败", "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": apperrors.Success, "message": "success", "data": result})
}

func (h *K8sClusterHandler) GetFeishuApps(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": apperrors.ErrCodeInvalidParams, "message": "ID格式错误"})
		return
	}

	result, err := h.svc.GetFeishuApps(c.Request.Context(), uint(id))
	if err != nil {
		if err == apperrors.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{"code": apperrors.ErrCodeNotFound, "message": "K8s集群不存在"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"code": apperrors.ErrCodeInternalError, "message": "查询绑定的飞书应用失败", "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": apperrors.Success, "message": "success", "data": result})
}

func (h *K8sClusterHandler) BindFeishuApps(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": apperrors.ErrCodeInvalidParams, "message": "ID格式错误"})
		return
	}

	var req dto.BindFeishuAppsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": apperrors.ErrCodeInvalidParams, "message": "参数错误", "error": err.Error()})
		return
	}

	if err := h.svc.BindFeishuApps(c.Request.Context(), uint(id), req.AppIDs); err != nil {
		if err == apperrors.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{"code": apperrors.ErrCodeNotFound, "message": "K8s集群不存在"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"code": apperrors.ErrCodeInternalError, "message": "绑定飞书应用失败", "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": apperrors.Success, "message": "绑定成功"})
}

func (h *K8sClusterHandler) TestConnection(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": apperrors.ErrCodeInvalidParams, "message": "ID格式错误"})
		return
	}

	result, err := h.svc.TestConnection(c.Request.Context(), uint(id))
	if err != nil {
		if err == apperrors.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{"code": apperrors.ErrCodeNotFound, "message": "K8s集群不存在"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"code": apperrors.ErrCodeInternalError, "message": "测试连接失败", "error": err.Error()})
		return
	}

	// 即使连接失败也返回 200，让前端根据 connected 字段判断
	// result.Error 中包含了具体的错误原因
	c.JSON(http.StatusOK, gin.H{"code": apperrors.Success, "message": "success", "data": result})
}

// GetNamespaces 获取命名空间列表
func (h *K8sClusterHandler) GetNamespaces(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": apperrors.ErrCodeInvalidParams, "message": "ID格式错误"})
		return
	}

	result, err := h.svc.GetNamespaces(c.Request.Context(), uint(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": apperrors.ErrCodeInternalError, "message": "获取命名空间失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": apperrors.Success, "message": "success", "data": result})
}

// GetPods 获取Pod列表
func (h *K8sClusterHandler) GetPods(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": apperrors.ErrCodeInvalidParams, "message": "ID格式错误"})
		return
	}

	namespace := c.Query("namespace")
	if namespace == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": apperrors.ErrCodeInvalidParams, "message": "命名空间必填"})
		return
	}

	result, err := h.svc.GetPods(c.Request.Context(), uint(id), namespace)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": apperrors.ErrCodeInternalError, "message": "获取Pod列表失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": apperrors.Success, "message": "success", "data": result})
}

// GetIstioStatus 获取 Istio 状态
func (h *K8sClusterHandler) GetIstioStatus(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": apperrors.ErrCodeInvalidParams, "message": "ID格式错误"})
		return
	}

	clientMgr := kubernetes.NewK8sClientManager(h.db)
	clientset, err := clientMgr.GetClient(c.Request.Context(), uint(id))
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"enabled": false}})
		return
	}

	// 检查 istio-system 命名空间是否存在
	_, err = clientset.CoreV1().Namespaces().Get(c.Request.Context(), "istio-system", metav1.GetOptions{})
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"enabled": false}})
		return
	}

	// 检查 istiod deployment 是否存在
	_, err = clientset.AppsV1().Deployments("istio-system").Get(c.Request.Context(), "istiod", metav1.GetOptions{})
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"enabled": false}})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"enabled": true}})
}

// ListServices 获取命名空间下的服务列表
func (h *K8sClusterHandler) ListServices(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": apperrors.ErrCodeInvalidParams, "message": "ID格式错误"})
		return
	}

	namespace := c.Param("ns")
	if namespace == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": apperrors.ErrCodeInvalidParams, "message": "命名空间不能为空"})
		return
	}

	clientMgr := kubernetes.NewK8sClientManager(h.db)
	clientset, err := clientMgr.GetClient(c.Request.Context(), uint(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": apperrors.ErrCodeInternalError, "message": "获取K8s客户端失败", "error": err.Error()})
		return
	}

	services, err := clientset.CoreV1().Services(namespace).List(c.Request.Context(), metav1.ListOptions{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": apperrors.ErrCodeInternalError, "message": "获取服务列表失败", "error": err.Error()})
		return
	}

	// 转换为简化的服务列表
	items := make([]gin.H, 0, len(services.Items))
	for _, svc := range services.Items {
		items = append(items, gin.H{
			"name":      svc.Name,
			"namespace": svc.Namespace,
			"metadata": gin.H{
				"name":      svc.Name,
				"namespace": svc.Namespace,
			},
		})
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"items": items}})
}
