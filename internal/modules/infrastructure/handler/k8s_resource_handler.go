package handler

import (
	"net/http"
	"slices"
	"strconv"

	"github.com/gin-gonic/gin"

	"devops/internal/config"
	"devops/internal/service/kubernetes"
	"devops/pkg/dto"
	apperrors "devops/pkg/errors"
	"devops/pkg/ioc"
	"devops/pkg/logger"
	"devops/pkg/middleware"
)

// clusterScopedResources 集群级资源（不需要 namespace）
var clusterScopedResources = map[string]bool{
	"node": true, "nodes": true,
	"pv": true, "pvs": true, "persistentvolume": true, "persistentvolumes": true,
	"storageclass": true, "storageclasses": true,
	"namespace": true, "namespaces": true,
}

func init() {
	ioc.Api.RegisterContainer("K8sResourceHandler", &K8sResourceApiHandler{})
}

type K8sResourceApiHandler struct {
	handler *K8sResourceHandler
}

func (h *K8sResourceApiHandler) Init() error {
	cfg, _ := config.LoadConfig()
	db := cfg.GetDB()
	svc := kubernetes.NewK8sResourceService(db)
	h.handler = NewK8sResourceHandler(svc)

	root := cfg.Application.GinRootRouter().Group("k8s-clusters/:id/resources")
	root.Use(middleware.AuthMiddleware())
	h.Register(root)

	return nil
}

func (h *K8sResourceApiHandler) Register(r gin.IRouter) {
	// 查看权限 - 所有登录用户
	r.GET("/namespaces", h.handler.GetNamespaces)
	r.GET("/deployments", h.handler.GetDeployments)
	r.GET("/deployments/:name/pods", h.handler.GetDeploymentPods)
	r.GET("/statefulsets", h.handler.GetStatefulSets)
	r.GET("/statefulsets/:name/pods", h.handler.GetStatefulSetPods)
	r.GET("/daemonsets", h.handler.GetDaemonSets)
	r.GET("/daemonsets/:name/pods", h.handler.GetDaemonSetPods)
	r.GET("/jobs", h.handler.GetJobs)
	r.GET("/cronjobs", h.handler.GetCronJobs)
	r.GET("/pods", h.handler.GetPods)
	r.GET("/pods/:podName/logs", h.handler.GetPodLogs)
	r.GET("/services", h.handler.GetServices)
	r.GET("/services/:name/pods", h.handler.GetServicePods)
	r.GET("/ingresses", h.handler.GetIngresses)
	r.GET("/endpoints", h.handler.GetEndpoints)
	r.GET("/configmaps", h.handler.GetConfigMaps)
	r.GET("/secrets", h.handler.GetSecrets)
	r.GET("/serviceaccounts", h.handler.GetServiceAccounts)
	r.GET("/pvcs", h.handler.GetPVCs)
	r.GET("/pvs", h.handler.GetPVs)
	r.GET("/storageclasses", h.handler.GetStorageClasses)
	r.GET("/nodes", h.handler.GetNodes)
	r.GET("/nodes/:nodeName", h.handler.GetNodeDetail)
	r.GET("/events", h.handler.GetEvents)

	// 操作权限 - 需要管理员
	r.POST("/namespaces", middleware.RequireAdmin(), h.handler.CreateNamespace)
	r.DELETE("/namespaces/:name", middleware.RequireAdmin(), h.handler.DeleteNamespace)
	r.DELETE("/pods/:podName", middleware.RequireAdmin(), h.handler.DeletePod)
	r.POST("/deployments/:deploymentName/restart", middleware.RequireAdmin(), h.handler.RestartDeployment)
	r.POST("/deployments/:deploymentName/scale", middleware.RequireAdmin(), h.handler.ScaleDeployment)
	r.GET("/nodes/join-command", middleware.RequireAdmin(), h.handler.GetJoinCommand)
	r.POST("/nodes/:nodeName/cordon", middleware.RequireAdmin(), h.handler.CordonNode)
	r.POST("/nodes/:nodeName/uncordon", middleware.RequireAdmin(), h.handler.UncordonNode)
	r.POST("/nodes/:nodeName/taints", middleware.RequireAdmin(), h.handler.AddNodeTaint)
	r.DELETE("/nodes/:nodeName/taints", middleware.RequireAdmin(), h.handler.RemoveNodeTaint)
	r.PUT("/nodes/:nodeName/labels", middleware.RequireAdmin(), h.handler.UpdateNodeLabels)
	r.GET("/events/resource", h.handler.GetResourceEvents)
	// 资源详情
	r.GET("/detail/:resourceType/:name", h.handler.GetResourceDetail)
	// YAML 操作
	r.GET("/yaml/:resourceType/:name", h.handler.GetResourceYAML)
	r.POST("/apply", h.handler.ApplyResourceYAML)
	r.DELETE("/:resourceType/:name", h.handler.DeleteResource)
}

type K8sResourceHandler struct {
	svc kubernetes.K8sResourceService
}

func NewK8sResourceHandler(svc kubernetes.K8sResourceService) *K8sResourceHandler {
	return &K8sResourceHandler{svc: svc}
}

func (h *K8sResourceHandler) getClusterID(c *gin.Context) (uint, error) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		return 0, err
	}
	return uint(id), nil
}

func (h *K8sResourceHandler) GetNamespaces(c *gin.Context) {
	log := logger.L().WithField("handler", "GetNamespaces")
	clusterID, err := h.getClusterID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": apperrors.ErrCodeInvalidParams, "message": "集群ID格式错误"})
		return
	}

	log.Info("获取命名空间", "clusterID", clusterID)
	result, err := h.svc.GetNamespaces(c.Request.Context(), clusterID)
	if err != nil {
		log.Error("获取命名空间失败", "clusterID", clusterID, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"code": apperrors.ErrCodeInternalError, "message": "获取命名空间失败", "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": apperrors.Success, "message": "success", "data": result})
}

func (h *K8sResourceHandler) CreateNamespace(c *gin.Context) {
	clusterID, err := h.getClusterID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": apperrors.ErrCodeInvalidParams, "message": "集群ID格式错误"})
		return
	}

	var req dto.CreateNamespaceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": apperrors.ErrCodeInvalidParams, "message": "参数错误", "error": err.Error()})
		return
	}

	if req.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": apperrors.ErrCodeInvalidParams, "message": "命名空间名称不能为空"})
		return
	}

	if err := h.svc.CreateNamespace(c.Request.Context(), clusterID, req.Name, req.Labels); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": apperrors.ErrCodeInternalError, "message": "创建命名空间失败", "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": apperrors.Success, "message": "创建成功"})
}

func (h *K8sResourceHandler) DeleteNamespace(c *gin.Context) {
	clusterID, err := h.getClusterID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": apperrors.ErrCodeInvalidParams, "message": "集群ID格式错误"})
		return
	}

	name := c.Param("name")
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": apperrors.ErrCodeInvalidParams, "message": "命名空间名称不能为空"})
		return
	}

	// 禁止删除系统命名空间
	systemNamespaces := []string{"default", "kube-system", "kube-public", "kube-node-lease"}
	if slices.Contains(systemNamespaces, name) {
		c.JSON(http.StatusBadRequest, gin.H{"code": apperrors.ErrCodeInvalidParams, "message": "不能删除系统命名空间"})
		return
	}

	if err := h.svc.DeleteNamespace(c.Request.Context(), clusterID, name); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": apperrors.ErrCodeInternalError, "message": "删除命名空间失败", "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": apperrors.Success, "message": "删除成功"})
}

func (h *K8sResourceHandler) GetDeployments(c *gin.Context) {
	clusterID, err := h.getClusterID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": apperrors.ErrCodeInvalidParams, "message": "集群ID格式错误"})
		return
	}

	namespace := c.DefaultQuery("namespace", "")
	result, err := h.svc.GetDeployments(c.Request.Context(), clusterID, namespace)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": apperrors.ErrCodeInternalError, "message": "获取Deployment失败", "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": apperrors.Success, "message": "success", "data": result})
}

func (h *K8sResourceHandler) GetStatefulSets(c *gin.Context) {
	clusterID, err := h.getClusterID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": apperrors.ErrCodeInvalidParams, "message": "集群ID格式错误"})
		return
	}

	namespace := c.DefaultQuery("namespace", "")
	result, err := h.svc.GetStatefulSets(c.Request.Context(), clusterID, namespace)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": apperrors.ErrCodeInternalError, "message": "获取StatefulSet失败", "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": apperrors.Success, "message": "success", "data": result})
}

func (h *K8sResourceHandler) GetDaemonSets(c *gin.Context) {
	clusterID, err := h.getClusterID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": apperrors.ErrCodeInvalidParams, "message": "集群ID格式错误"})
		return
	}

	namespace := c.DefaultQuery("namespace", "")
	result, err := h.svc.GetDaemonSets(c.Request.Context(), clusterID, namespace)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": apperrors.ErrCodeInternalError, "message": "获取DaemonSet失败", "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": apperrors.Success, "message": "success", "data": result})
}

func (h *K8sResourceHandler) GetJobs(c *gin.Context) {
	clusterID, err := h.getClusterID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": apperrors.ErrCodeInvalidParams, "message": "集群ID格式错误"})
		return
	}

	namespace := c.DefaultQuery("namespace", "")
	result, err := h.svc.GetJobs(c.Request.Context(), clusterID, namespace)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": apperrors.ErrCodeInternalError, "message": "获取Job失败", "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": apperrors.Success, "message": "success", "data": result})
}

func (h *K8sResourceHandler) GetCronJobs(c *gin.Context) {
	clusterID, err := h.getClusterID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": apperrors.ErrCodeInvalidParams, "message": "集群ID格式错误"})
		return
	}

	namespace := c.DefaultQuery("namespace", "")
	result, err := h.svc.GetCronJobs(c.Request.Context(), clusterID, namespace)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": apperrors.ErrCodeInternalError, "message": "获取CronJob失败", "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": apperrors.Success, "message": "success", "data": result})
}

func (h *K8sResourceHandler) GetPods(c *gin.Context) {
	clusterID, err := h.getClusterID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": apperrors.ErrCodeInvalidParams, "message": "集群ID格式错误"})
		return
	}

	namespace := c.DefaultQuery("namespace", "")
	result, err := h.svc.GetPods(c.Request.Context(), clusterID, namespace)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": apperrors.ErrCodeInternalError, "message": "获取Pod失败", "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": apperrors.Success, "message": "success", "data": result})
}

func (h *K8sResourceHandler) GetServices(c *gin.Context) {
	clusterID, err := h.getClusterID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": apperrors.ErrCodeInvalidParams, "message": "集群ID格式错误"})
		return
	}

	namespace := c.DefaultQuery("namespace", "")
	result, err := h.svc.GetServices(c.Request.Context(), clusterID, namespace)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": apperrors.ErrCodeInternalError, "message": "获取Service失败", "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": apperrors.Success, "message": "success", "data": result})
}

func (h *K8sResourceHandler) GetIngresses(c *gin.Context) {
	clusterID, err := h.getClusterID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": apperrors.ErrCodeInvalidParams, "message": "集群ID格式错误"})
		return
	}

	namespace := c.DefaultQuery("namespace", "")
	result, err := h.svc.GetIngresses(c.Request.Context(), clusterID, namespace)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": apperrors.ErrCodeInternalError, "message": "获取Ingress失败", "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": apperrors.Success, "message": "success", "data": result})
}

func (h *K8sResourceHandler) GetConfigMaps(c *gin.Context) {
	clusterID, err := h.getClusterID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": apperrors.ErrCodeInvalidParams, "message": "集群ID格式错误"})
		return
	}

	namespace := c.DefaultQuery("namespace", "")
	result, err := h.svc.GetConfigMaps(c.Request.Context(), clusterID, namespace)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": apperrors.ErrCodeInternalError, "message": "获取ConfigMap失败", "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": apperrors.Success, "message": "success", "data": result})
}

func (h *K8sResourceHandler) GetSecrets(c *gin.Context) {
	clusterID, err := h.getClusterID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": apperrors.ErrCodeInvalidParams, "message": "集群ID格式错误"})
		return
	}

	namespace := c.DefaultQuery("namespace", "")
	result, err := h.svc.GetSecrets(c.Request.Context(), clusterID, namespace)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": apperrors.ErrCodeInternalError, "message": "获取Secret失败", "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": apperrors.Success, "message": "success", "data": result})
}

func (h *K8sResourceHandler) GetPVCs(c *gin.Context) {
	clusterID, err := h.getClusterID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": apperrors.ErrCodeInvalidParams, "message": "集群ID格式错误"})
		return
	}

	namespace := c.DefaultQuery("namespace", "")
	result, err := h.svc.GetPVCs(c.Request.Context(), clusterID, namespace)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": apperrors.ErrCodeInternalError, "message": "获取PVC失败", "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": apperrors.Success, "message": "success", "data": result})
}

func (h *K8sResourceHandler) GetPodLogs(c *gin.Context) {
	clusterID, err := h.getClusterID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": apperrors.ErrCodeInvalidParams, "message": "集群ID格式错误"})
		return
	}

	namespace := c.Query("namespace")
	if namespace == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": apperrors.ErrCodeInvalidParams, "message": "namespace不能为空"})
		return
	}

	podName := c.Param("podName")
	container := c.Query("container")
	tailLines := int64(100)
	if t := c.Query("tail"); t != "" {
		if parsed, err := strconv.ParseInt(t, 10, 64); err == nil {
			tailLines = parsed
		}
	}

	logs, err := h.svc.GetPodLogs(c.Request.Context(), clusterID, namespace, podName, container, tailLines)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": apperrors.ErrCodeInternalError, "message": "获取日志失败", "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": apperrors.Success, "message": "success", "data": logs})
}

func (h *K8sResourceHandler) DeletePod(c *gin.Context) {
	clusterID, err := h.getClusterID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": apperrors.ErrCodeInvalidParams, "message": "集群ID格式错误"})
		return
	}

	namespace := c.Query("namespace")
	if namespace == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": apperrors.ErrCodeInvalidParams, "message": "namespace不能为空"})
		return
	}

	podName := c.Param("podName")
	if err := h.svc.DeletePod(c.Request.Context(), clusterID, namespace, podName); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": apperrors.ErrCodeInternalError, "message": "删除Pod失败", "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": apperrors.Success, "message": "删除成功"})
}

func (h *K8sResourceHandler) RestartDeployment(c *gin.Context) {
	clusterID, err := h.getClusterID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": apperrors.ErrCodeInvalidParams, "message": "集群ID格式错误"})
		return
	}

	namespace := c.Query("namespace")
	if namespace == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": apperrors.ErrCodeInvalidParams, "message": "namespace不能为空"})
		return
	}

	deploymentName := c.Param("deploymentName")
	if err := h.svc.RestartDeployment(c.Request.Context(), clusterID, namespace, deploymentName); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": apperrors.ErrCodeInternalError, "message": "重启Deployment失败", "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": apperrors.Success, "message": "重启成功"})
}

func (h *K8sResourceHandler) ScaleDeployment(c *gin.Context) {
	clusterID, err := h.getClusterID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": apperrors.ErrCodeInvalidParams, "message": "集群ID格式错误"})
		return
	}

	namespace := c.Query("namespace")
	if namespace == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": apperrors.ErrCodeInvalidParams, "message": "namespace不能为空"})
		return
	}

	var req dto.ScaleDeploymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": apperrors.ErrCodeInvalidParams, "message": "参数错误", "error": err.Error()})
		return
	}

	deploymentName := c.Param("deploymentName")
	if err := h.svc.ScaleDeployment(c.Request.Context(), clusterID, namespace, deploymentName, req.Replicas); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": apperrors.ErrCodeInternalError, "message": "调整副本数失败", "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": apperrors.Success, "message": "调整成功"})
}

func (h *K8sResourceHandler) GetResourceYAML(c *gin.Context) {
	clusterID, err := h.getClusterID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": apperrors.ErrCodeInvalidParams, "message": "集群ID格式错误"})
		return
	}

	resourceType := c.Param("resourceType")
	name := c.Param("name")
	namespace := c.Query("namespace")

	if !clusterScopedResources[resourceType] && namespace == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": apperrors.ErrCodeInvalidParams, "message": "namespace不能为空"})
		return
	}

	yaml, err := h.svc.GetResourceYAML(c.Request.Context(), clusterID, resourceType, namespace, name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": apperrors.ErrCodeInternalError, "message": "获取资源YAML失败", "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": apperrors.Success, "message": "success", "data": yaml})
}

func (h *K8sResourceHandler) ApplyResourceYAML(c *gin.Context) {
	clusterID, err := h.getClusterID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": apperrors.ErrCodeInvalidParams, "message": "集群ID格式错误"})
		return
	}

	var req dto.ApplyResourceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": apperrors.ErrCodeInvalidParams, "message": "参数错误: " + err.Error()})
		return
	}

	if req.YAML == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": apperrors.ErrCodeInvalidParams, "message": "YAML内容不能为空"})
		return
	}

	// 使用结构化日志记录 YAML 应用操作
	log := logger.L().WithField("cluster_id", clusterID)
	yamlPreview := req.YAML
	if len(yamlPreview) > 100 {
		yamlPreview = yamlPreview[:100] + "..."
	}
	log.Debug("ApplyYAML: YAML preview: %s", yamlPreview)

	if err := h.svc.ApplyResourceYAML(c.Request.Context(), clusterID, req.YAML); err != nil {
		log.WithError(err).Error("ApplyYAML failed")
		c.JSON(http.StatusInternalServerError, gin.H{"code": apperrors.ErrCodeInternalError, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": apperrors.Success, "message": "应用成功"})
}

func (h *K8sResourceHandler) DeleteResource(c *gin.Context) {
	clusterID, err := h.getClusterID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": apperrors.ErrCodeInvalidParams, "message": "集群ID格式错误"})
		return
	}

	resourceType := c.Param("resourceType")
	name := c.Param("name")
	namespace := c.Query("namespace")

	if !clusterScopedResources[resourceType] && namespace == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": apperrors.ErrCodeInvalidParams, "message": "namespace不能为空"})
		return
	}

	if err := h.svc.DeleteResource(c.Request.Context(), clusterID, resourceType, namespace, name); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": apperrors.ErrCodeInternalError, "message": "删除资源失败", "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": apperrors.Success, "message": "删除成功"})
}

// 新增的 handler 方法

func (h *K8sResourceHandler) GetEndpoints(c *gin.Context) {
	clusterID, err := h.getClusterID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": apperrors.ErrCodeInvalidParams, "message": "集群ID格式错误"})
		return
	}

	namespace := c.DefaultQuery("namespace", "")
	result, err := h.svc.GetEndpoints(c.Request.Context(), clusterID, namespace)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": apperrors.ErrCodeInternalError, "message": "获取Endpoints失败", "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": apperrors.Success, "message": "success", "data": result})
}

func (h *K8sResourceHandler) GetServiceAccounts(c *gin.Context) {
	clusterID, err := h.getClusterID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": apperrors.ErrCodeInvalidParams, "message": "集群ID格式错误"})
		return
	}

	namespace := c.DefaultQuery("namespace", "")
	result, err := h.svc.GetServiceAccounts(c.Request.Context(), clusterID, namespace)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": apperrors.ErrCodeInternalError, "message": "获取ServiceAccount失败", "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": apperrors.Success, "message": "success", "data": result})
}

func (h *K8sResourceHandler) GetPVs(c *gin.Context) {
	clusterID, err := h.getClusterID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": apperrors.ErrCodeInvalidParams, "message": "集群ID格式错误"})
		return
	}

	result, err := h.svc.GetPVs(c.Request.Context(), clusterID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": apperrors.ErrCodeInternalError, "message": "获取PV失败", "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": apperrors.Success, "message": "success", "data": result})
}

func (h *K8sResourceHandler) GetStorageClasses(c *gin.Context) {
	clusterID, err := h.getClusterID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": apperrors.ErrCodeInvalidParams, "message": "集群ID格式错误"})
		return
	}

	result, err := h.svc.GetStorageClasses(c.Request.Context(), clusterID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": apperrors.ErrCodeInternalError, "message": "获取StorageClass失败", "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": apperrors.Success, "message": "success", "data": result})
}

func (h *K8sResourceHandler) GetNodes(c *gin.Context) {
	clusterID, err := h.getClusterID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": apperrors.ErrCodeInvalidParams, "message": "集群ID格式错误"})
		return
	}

	result, err := h.svc.GetNodes(c.Request.Context(), clusterID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": apperrors.ErrCodeInternalError, "message": "获取节点失败", "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": apperrors.Success, "message": "success", "data": result})
}

func (h *K8sResourceHandler) GetNodeDetail(c *gin.Context) {
	clusterID, err := h.getClusterID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": apperrors.ErrCodeInvalidParams, "message": "集群ID格式错误"})
		return
	}

	nodeName := c.Param("nodeName")
	result, err := h.svc.GetNodeDetail(c.Request.Context(), clusterID, nodeName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": apperrors.ErrCodeInternalError, "message": "获取节点详情失败", "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": apperrors.Success, "message": "success", "data": result})
}

func (h *K8sResourceHandler) CordonNode(c *gin.Context) {
	clusterID, err := h.getClusterID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": apperrors.ErrCodeInvalidParams, "message": "集群ID格式错误"})
		return
	}

	nodeName := c.Param("nodeName")
	if err := h.svc.CordonNode(c.Request.Context(), clusterID, nodeName); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": apperrors.ErrCodeInternalError, "message": "设置节点不可调度失败", "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": apperrors.Success, "message": "设置成功"})
}

func (h *K8sResourceHandler) UncordonNode(c *gin.Context) {
	clusterID, err := h.getClusterID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": apperrors.ErrCodeInvalidParams, "message": "集群ID格式错误"})
		return
	}

	nodeName := c.Param("nodeName")
	if err := h.svc.UncordonNode(c.Request.Context(), clusterID, nodeName); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": apperrors.ErrCodeInternalError, "message": "设置节点可调度失败", "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": apperrors.Success, "message": "设置成功"})
}

func (h *K8sResourceHandler) AddNodeTaint(c *gin.Context) {
	clusterID, err := h.getClusterID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": apperrors.ErrCodeInvalidParams, "message": "集群ID格式错误"})
		return
	}

	var req dto.AddNodeTaintRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": apperrors.ErrCodeInvalidParams, "message": "参数错误", "error": err.Error()})
		return
	}

	nodeName := c.Param("nodeName")
	taint := dto.K8sNodeTaint{Key: req.Key, Value: req.Value, Effect: req.Effect}
	if err := h.svc.AddNodeTaint(c.Request.Context(), clusterID, nodeName, taint); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": apperrors.ErrCodeInternalError, "message": "添加污点失败", "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": apperrors.Success, "message": "添加成功"})
}

func (h *K8sResourceHandler) RemoveNodeTaint(c *gin.Context) {
	clusterID, err := h.getClusterID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": apperrors.ErrCodeInvalidParams, "message": "集群ID格式错误"})
		return
	}

	nodeName := c.Param("nodeName")
	taintKey := c.Query("key")
	taintEffect := c.Query("effect")

	if taintKey == "" || taintEffect == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": apperrors.ErrCodeInvalidParams, "message": "key和effect不能为空"})
		return
	}

	if err := h.svc.RemoveNodeTaint(c.Request.Context(), clusterID, nodeName, taintKey, taintEffect); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": apperrors.ErrCodeInternalError, "message": "移除污点失败", "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": apperrors.Success, "message": "移除成功"})
}

func (h *K8sResourceHandler) UpdateNodeLabels(c *gin.Context) {
	clusterID, err := h.getClusterID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": apperrors.ErrCodeInvalidParams, "message": "集群ID格式错误"})
		return
	}

	var req dto.UpdateNodeLabelsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": apperrors.ErrCodeInvalidParams, "message": "参数错误", "error": err.Error()})
		return
	}

	nodeName := c.Param("nodeName")
	if err := h.svc.UpdateNodeLabels(c.Request.Context(), clusterID, nodeName, req.Labels); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": apperrors.ErrCodeInternalError, "message": "更新标签失败", "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": apperrors.Success, "message": "更新成功"})
}

func (h *K8sResourceHandler) GetEvents(c *gin.Context) {
	clusterID, err := h.getClusterID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": apperrors.ErrCodeInvalidParams, "message": "集群ID格式错误"})
		return
	}

	namespace := c.DefaultQuery("namespace", "")
	result, err := h.svc.GetEvents(c.Request.Context(), clusterID, namespace)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": apperrors.ErrCodeInternalError, "message": "获取事件失败", "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": apperrors.Success, "message": "success", "data": result})
}

func (h *K8sResourceHandler) GetJoinCommand(c *gin.Context) {
	clusterID, err := h.getClusterID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": apperrors.ErrCodeInvalidParams, "message": "集群ID格式错误"})
		return
	}

	result, err := h.svc.GetJoinCommand(c.Request.Context(), clusterID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": apperrors.ErrCodeInternalError, "message": "生成加入命令失败", "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": apperrors.Success, "message": "success", "data": result})
}

// 获取资源详情
func (h *K8sResourceHandler) GetResourceDetail(c *gin.Context) {
	clusterID, err := h.getClusterID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": apperrors.ErrCodeInvalidParams, "message": "集群ID格式错误"})
		return
	}

	resourceType := c.Param("resourceType")
	name := c.Param("name")
	namespace := c.Query("namespace")

	result, err := h.svc.GetResourceDetail(c.Request.Context(), clusterID, resourceType, namespace, name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": apperrors.ErrCodeInternalError, "message": "获取资源详情失败", "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": apperrors.Success, "message": "success", "data": result})
}

// 获取资源相关事件
func (h *K8sResourceHandler) GetResourceEvents(c *gin.Context) {
	clusterID, err := h.getClusterID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": apperrors.ErrCodeInvalidParams, "message": "集群ID格式错误"})
		return
	}

	resourceType := c.Query("resource_type")
	namespace := c.Query("namespace")
	name := c.Query("name")

	result, err := h.svc.GetResourceEvents(c.Request.Context(), clusterID, resourceType, namespace, name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": apperrors.ErrCodeInternalError, "message": "获取事件失败", "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": apperrors.Success, "message": "success", "data": result})
}

// 获取 Deployment 关联的 Pods
func (h *K8sResourceHandler) GetDeploymentPods(c *gin.Context) {
	clusterID, err := h.getClusterID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": apperrors.ErrCodeInvalidParams, "message": "集群ID格式错误"})
		return
	}

	namespace := c.Query("namespace")
	name := c.Param("name")

	result, err := h.svc.GetRelatedPods(c.Request.Context(), clusterID, "deployment", namespace, name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": apperrors.ErrCodeInternalError, "message": "获取关联Pod失败", "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": apperrors.Success, "message": "success", "data": result})
}

// 获取 StatefulSet 关联的 Pods
func (h *K8sResourceHandler) GetStatefulSetPods(c *gin.Context) {
	clusterID, err := h.getClusterID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": apperrors.ErrCodeInvalidParams, "message": "集群ID格式错误"})
		return
	}

	namespace := c.Query("namespace")
	name := c.Param("name")

	result, err := h.svc.GetRelatedPods(c.Request.Context(), clusterID, "statefulset", namespace, name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": apperrors.ErrCodeInternalError, "message": "获取关联Pod失败", "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": apperrors.Success, "message": "success", "data": result})
}

// 获取 DaemonSet 关联的 Pods
func (h *K8sResourceHandler) GetDaemonSetPods(c *gin.Context) {
	clusterID, err := h.getClusterID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": apperrors.ErrCodeInvalidParams, "message": "集群ID格式错误"})
		return
	}

	namespace := c.Query("namespace")
	name := c.Param("name")

	result, err := h.svc.GetRelatedPods(c.Request.Context(), clusterID, "daemonset", namespace, name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": apperrors.ErrCodeInternalError, "message": "获取关联Pod失败", "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": apperrors.Success, "message": "success", "data": result})
}

// 获取 Service 后端 Pods
func (h *K8sResourceHandler) GetServicePods(c *gin.Context) {
	clusterID, err := h.getClusterID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": apperrors.ErrCodeInvalidParams, "message": "集群ID格式错误"})
		return
	}

	namespace := c.Query("namespace")
	name := c.Param("name")

	result, err := h.svc.GetServicePods(c.Request.Context(), clusterID, namespace, name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": apperrors.ErrCodeInternalError, "message": "获取Service后端Pod失败", "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": apperrors.Success, "message": "success", "data": result})
}
