package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"

	"devops/internal/config"
	"devops/internal/service/kubernetes"
	"devops/pkg/ioc"
	"devops/pkg/logger"
	"devops/pkg/middleware"
	"devops/pkg/response"
)

func init() {
	ioc.Api.RegisterContainer("K8sResourceHandler", &K8sResourceApiHandler{})
}

type K8sResourceApiHandler struct {
	handler *K8sResourceHandler
}

func (h *K8sResourceApiHandler) Init() error {
	cfg, _ := config.LoadConfig()
	db := cfg.GetDB()
	clientMgr := kubernetes.NewK8sClientManager(db)

	namespaceSvc := kubernetes.NewK8sNamespaceService(clientMgr)
	deploymentSvc := kubernetes.NewK8sDeploymentService(clientMgr)
	podSvc := kubernetes.NewK8sPodService(clientMgr)
	serviceSvc := kubernetes.NewK8sServiceService(clientMgr)

	h.handler = NewK8sResourceHandler(namespaceSvc, deploymentSvc, podSvc, serviceSvc)

	root := cfg.Application.GinRootRouter().Group("k8s/clusters")
	root.Use(middleware.AuthMiddleware())
	h.Register(root)

	return nil
}

func (h *K8sResourceApiHandler) Register(r gin.IRouter) {
	h.handler.RegisterRoutes(r)
}

// K8sResourceHandler K8s 资源管理处理器
type K8sResourceHandler struct {
	namespaceSvc  kubernetes.NamespaceService
	deploymentSvc *kubernetes.K8sDeploymentService
	podSvc        *kubernetes.K8sPodService
	serviceSvc    *kubernetes.K8sServiceService
}

// NewK8sResourceHandler 创建资源处理器
func NewK8sResourceHandler(
	namespaceSvc kubernetes.NamespaceService,
	deploymentSvc *kubernetes.K8sDeploymentService,
	podSvc *kubernetes.K8sPodService,
	serviceSvc *kubernetes.K8sServiceService,
) *K8sResourceHandler {
	return &K8sResourceHandler{
		namespaceSvc:  namespaceSvc,
		deploymentSvc: deploymentSvc,
		podSvc:        podSvc,
		serviceSvc:    serviceSvc,
	}
}

// RegisterRoutes 注册路由
func (h *K8sResourceHandler) RegisterRoutes(r gin.IRouter) {
	// Namespace 路由
	r.GET("/:id/namespaces", h.ListNamespaces)
	r.GET("/:id/namespaces/:name", h.GetNamespace)

	// Deployment 路由
	r.GET("/:id/deployments", h.ListDeployments)
	r.GET("/:id/deployments/:name", h.GetDeployment)
	r.POST("/:id/deployments/:name/restart", h.RestartDeployment)
	r.POST("/:id/deployments/:name/scale", h.ScaleDeployment)

	// Pod 路由
	r.GET("/:id/pods", h.ListPods)
	r.GET("/:id/pods/:name", h.GetPod)
	r.DELETE("/:id/pods/:name", h.DeletePod)
	r.GET("/:id/pods/:name/logs", h.GetPodLogs)
	r.GET("/:id/pods/:name/logs/stream", h.StreamPodLogs)

	// Service 路由
	r.GET("/:id/services", h.ListServices)
	r.GET("/:id/services/:name", h.GetService)
}

// ===== Namespace Handlers =====

// ListNamespaces 获取命名空间列表
func (h *K8sResourceHandler) ListNamespaces(c *gin.Context) {
	clusterID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的集群ID")
		return
	}

	namespaces, err := h.namespaceSvc.ListNamespaces(c.Request.Context(), uint(clusterID))
	if err != nil {
		response.InternalError(c, "获取命名空间列表失败: "+err.Error())
		return
	}

	response.Success(c, namespaces)
}

// GetNamespace 获取命名空间详情
func (h *K8sResourceHandler) GetNamespace(c *gin.Context) {
	clusterID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的集群ID")
		return
	}

	name := c.Param("name")
	if name == "" {
		response.BadRequest(c, "命名空间名称不能为空")
		return
	}

	namespace, err := h.namespaceSvc.GetNamespace(c.Request.Context(), uint(clusterID), name)
	if err != nil {
		response.InternalError(c, "获取命名空间详情失败: "+err.Error())
		return
	}

	response.Success(c, namespace)
}

// ===== Deployment Handlers =====

// ListDeployments 获取 Deployment 列表
func (h *K8sResourceHandler) ListDeployments(c *gin.Context) {
	clusterID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的集群ID")
		return
	}

	namespace := c.Query("namespace")
	if namespace == "" {
		namespace = "default"
	}

	deployments, err := h.deploymentSvc.ListDeployments(c.Request.Context(), uint(clusterID), namespace)
	if err != nil {
		response.InternalError(c, "获取Deployment列表失败: "+err.Error())
		return
	}

	response.Success(c, deployments)
}

// GetDeployment 获取 Deployment 详情
func (h *K8sResourceHandler) GetDeployment(c *gin.Context) {
	clusterID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的集群ID")
		return
	}

	namespace := c.Query("namespace")
	if namespace == "" {
		namespace = "default"
	}

	name := c.Param("name")
	if name == "" {
		response.BadRequest(c, "Deployment名称不能为空")
		return
	}

	deployment, err := h.deploymentSvc.GetDeployment(c.Request.Context(), uint(clusterID), namespace, name)
	if err != nil {
		response.InternalError(c, "获取Deployment详情失败: "+err.Error())
		return
	}

	response.Success(c, deployment)
}

// RestartDeployment 重启 Deployment
func (h *K8sResourceHandler) RestartDeployment(c *gin.Context) {
	clusterID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的集群ID")
		return
	}

	namespace := c.Query("namespace")
	if namespace == "" {
		namespace = "default"
	}

	name := c.Param("name")
	if name == "" {
		response.BadRequest(c, "Deployment名称不能为空")
		return
	}

	if err := h.deploymentSvc.Restart(c.Request.Context(), uint(clusterID), namespace, name); err != nil {
		response.InternalError(c, "重启Deployment失败: "+err.Error())
		return
	}

	response.Success(c, gin.H{"message": "重启操作已提交"})
}

// ScaleDeploymentRequest 扩缩容请求
type ScaleDeploymentRequest struct {
	Replicas int32 `json:"replicas" binding:"required,min=0,max=100"`
}

// ScaleDeployment 扩缩容 Deployment
func (h *K8sResourceHandler) ScaleDeployment(c *gin.Context) {
	clusterID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的集群ID")
		return
	}

	namespace := c.Query("namespace")
	if namespace == "" {
		namespace = "default"
	}

	name := c.Param("name")
	if name == "" {
		response.BadRequest(c, "Deployment名称不能为空")
		return
	}

	var req ScaleDeploymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "请求参数错误: "+err.Error())
		return
	}

	if err := h.deploymentSvc.Scale(c.Request.Context(), uint(clusterID), namespace, name, req.Replicas); err != nil {
		response.InternalError(c, "扩缩容失败: "+err.Error())
		return
	}

	response.Success(c, gin.H{"message": "扩缩容操作已提交"})
}

// ===== Pod Handlers =====

// ListPods 获取 Pod 列表
func (h *K8sResourceHandler) ListPods(c *gin.Context) {
	clusterID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的集群ID")
		return
	}

	namespace := c.Query("namespace")
	if namespace == "" {
		namespace = "default"
	}

	labelSelector := c.Query("label_selector")

	pods, err := h.podSvc.ListPods(c.Request.Context(), uint(clusterID), namespace, labelSelector)
	if err != nil {
		response.InternalError(c, "获取Pod列表失败: "+err.Error())
		return
	}

	response.Success(c, pods)
}

// GetPod 获取 Pod 详情
func (h *K8sResourceHandler) GetPod(c *gin.Context) {
	clusterID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的集群ID")
		return
	}

	namespace := c.Query("namespace")
	if namespace == "" {
		namespace = "default"
	}

	name := c.Param("name")
	if name == "" {
		response.BadRequest(c, "Pod名称不能为空")
		return
	}

	pod, err := h.podSvc.GetPod(c.Request.Context(), uint(clusterID), namespace, name)
	if err != nil {
		response.InternalError(c, "获取Pod详情失败: "+err.Error())
		return
	}

	response.Success(c, pod)
}

// DeletePod 删除 Pod
func (h *K8sResourceHandler) DeletePod(c *gin.Context) {
	clusterID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的集群ID")
		return
	}

	namespace := c.Query("namespace")
	if namespace == "" {
		namespace = "default"
	}

	name := c.Param("name")
	if name == "" {
		response.BadRequest(c, "Pod名称不能为空")
		return
	}

	if err := h.podSvc.DeletePod(c.Request.Context(), uint(clusterID), namespace, name); err != nil {
		response.InternalError(c, "删除Pod失败: "+err.Error())
		return
	}

	response.Success(c, gin.H{"message": "Pod已删除"})
}

// GetPodLogs 获取 Pod 日志
func (h *K8sResourceHandler) GetPodLogs(c *gin.Context) {
	clusterID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的集群ID")
		return
	}

	namespace := c.Query("namespace")
	if namespace == "" {
		namespace = "default"
	}

	name := c.Param("name")
	if name == "" {
		response.BadRequest(c, "Pod名称不能为空")
		return
	}

	container := c.Query("container")
	tailLines := int64(100)
	if tl := c.Query("tail_lines"); tl != "" {
		if parsed, err := strconv.ParseInt(tl, 10, 64); err == nil {
			tailLines = parsed
		}
	}

	timestamps := c.Query("timestamps") == "true"

	req := &kubernetes.LogRequest{
		ClusterID:  uint(clusterID),
		Namespace:  namespace,
		PodName:    name,
		Container:  container,
		TailLines:  tailLines,
		Timestamps: timestamps,
	}

	logs, err := h.podSvc.GetLogs(c.Request.Context(), req)
	if err != nil {
		response.InternalError(c, "获取日志失败: "+err.Error())
		return
	}

	response.Success(c, gin.H{"logs": logs})
}

// StreamPodLogs 流式获取 Pod 日志（WebSocket）
func (h *K8sResourceHandler) StreamPodLogs(c *gin.Context) {
	log := logger.L().WithField("handler", "StreamPodLogs")

	clusterID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的集群ID"})
		return
	}

	namespace := c.Query("namespace")
	if namespace == "" {
		namespace = "default"
	}

	name := c.Param("name")
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Pod名称不能为空"})
		return
	}

	container := c.Query("container")
	tailLines := int64(100)
	if tl := c.Query("tail_lines"); tl != "" {
		if parsed, err := strconv.ParseInt(tl, 10, 64); err == nil {
			tailLines = parsed
		}
	}

	timestamps := c.Query("timestamps") == "true"

	// 升级为 WebSocket
	conn, err := wsUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.WithField("error", err).Error("WebSocket 升级失败")
		return
	}
	defer conn.Close()

	req := &kubernetes.LogRequest{
		ClusterID:  uint(clusterID),
		Namespace:  namespace,
		PodName:    name,
		Container:  container,
		TailLines:  tailLines,
		Follow:     true,
		Timestamps: timestamps,
	}

	log.WithField("pod", name).WithField("namespace", namespace).Info("日志流连接")

	// 创建 WebSocket writer
	writer := &wsWriter{conn: conn}

	if err := h.podSvc.StreamLogs(c.Request.Context(), req, writer); err != nil {
		log.WithField("error", err).Error("日志流失败")
		conn.WriteMessage(websocket.TextMessage, []byte("\r\nError: "+err.Error()+"\r\n"))
	}
}

// wsWriter WebSocket writer 适配器
type wsWriter struct {
	conn *websocket.Conn
}

func (w *wsWriter) Write(p []byte) (n int, err error) {
	if err := w.conn.WriteMessage(websocket.TextMessage, p); err != nil {
		return 0, err
	}
	return len(p), nil
}

// ===== Service Handlers =====

// ListServices 获取 Service 列表
func (h *K8sResourceHandler) ListServices(c *gin.Context) {
	clusterID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的集群ID")
		return
	}

	namespace := c.Query("namespace")
	if namespace == "" {
		namespace = "default"
	}

	services, err := h.serviceSvc.ListServices(c.Request.Context(), uint(clusterID), namespace)
	if err != nil {
		response.InternalError(c, "获取Service列表失败: "+err.Error())
		return
	}

	response.Success(c, services)
}

// GetService 获取 Service 详情
func (h *K8sResourceHandler) GetService(c *gin.Context) {
	clusterID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的集群ID")
		return
	}

	namespace := c.Query("namespace")
	if namespace == "" {
		namespace = "default"
	}

	name := c.Param("name")
	if name == "" {
		response.BadRequest(c, "Service名称不能为空")
		return
	}

	service, err := h.serviceSvc.GetService(c.Request.Context(), uint(clusterID), namespace, name)
	if err != nil {
		response.InternalError(c, "获取Service详情失败: "+err.Error())
		return
	}

	response.Success(c, service)
}
