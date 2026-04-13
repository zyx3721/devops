package handler

import (
	"devops/internal/config"
	"devops/internal/service/kubernetes"
	"devops/pkg/ioc"
	"devops/pkg/middleware"

	"github.com/gin-gonic/gin"
)

func init() {
	ioc.Api.RegisterContainer("K8sOpsHandler", &K8sOpsIOC{})
}

type K8sOpsIOC struct {
	podHandler        *K8sPodHandler
	deploymentHandler *K8sDeploymentHandler
	metricsHandler    *K8sMetricsHandler
}

func (h *K8sOpsIOC) Init() error {
	cfg, err := config.LoadConfig()
	if err != nil {
		return err
	}

	db := cfg.GetDB()

	// 创建 K8s 客户端管理器
	clientMgr := kubernetes.NewK8sClientManager(db)

	// 创建服务
	podService := kubernetes.NewK8sPodService(clientMgr)
	terminalService := kubernetes.NewK8sTerminalService(clientMgr)
	deploymentService := kubernetes.NewK8sDeploymentService(clientMgr)
	metricsService := kubernetes.NewK8sMetricsService(db)

	// 创建 Handler
	h.podHandler = NewK8sPodHandler(podService, terminalService)
	h.deploymentHandler = NewK8sDeploymentHandler(deploymentService)
	h.metricsHandler = NewK8sMetricsHandler(metricsService)

	// 注册路由
	root := cfg.Application.GinRootRouter().Group("k8s/clusters/:id/namespaces/:ns")
	root.Use(middleware.AuthMiddleware())
	h.Register(root)

	// 注册集群级别的 metrics 路由
	clusterRoot := cfg.Application.GinRootRouter().Group("k8s/clusters/:id")
	clusterRoot.Use(middleware.AuthMiddleware())
	h.RegisterClusterRoutes(clusterRoot)

	return nil
}

func (h *K8sOpsIOC) Register(r gin.IRouter) {
	// Pod 管理
	pods := r.Group("/pods")
	{
		pods.GET("", h.podHandler.ListPods)
		pods.GET("/:name", h.podHandler.GetPod)
		pods.DELETE("/:name", h.podHandler.DeletePod)
		pods.GET("/:name/containers", h.podHandler.GetPodContainers)
		pods.GET("/:name/logs", h.podHandler.GetPodLogs)
		pods.GET("/:name/logs/stream", h.podHandler.StreamPodLogs)
		pods.GET("/:name/logs/download", h.podHandler.DownloadPodLogs)
		pods.GET("/:name/terminal", h.podHandler.PodTerminal)
		pods.GET("/:name/metrics", h.metricsHandler.GetPodMetrics)
	}

	// Deployment 管理
	deployments := r.Group("/deployments")
	{
		deployments.GET("", h.deploymentHandler.ListDeployments)
		deployments.GET("/:name", h.deploymentHandler.GetDeployment)
		deployments.PUT("/:name/image", h.deploymentHandler.UpdateImage)
		deployments.PUT("/:name/scale", h.deploymentHandler.Scale)
		deployments.POST("/:name/restart", h.deploymentHandler.Restart)
		deployments.GET("/:name/revisions", h.deploymentHandler.GetRevisionHistory)
		deployments.POST("/:name/rollback", h.deploymentHandler.Rollback)
		deployments.GET("/:name/progress", h.deploymentHandler.GetUpdateProgress)
	}

	// 命名空间级别的 Metrics
	metrics := r.Group("/metrics")
	{
		metrics.GET("/pods", h.metricsHandler.GetPodListMetrics)
	}
}

// RegisterClusterRoutes 注册集群级别的路由
func (h *K8sOpsIOC) RegisterClusterRoutes(r gin.IRouter) {
	// 集群级别的 Metrics
	metrics := r.Group("/metrics")
	{
		metrics.GET("/status", h.metricsHandler.CheckMetricsServer)
		metrics.GET("/nodes", h.metricsHandler.GetNodeMetrics)
	}

	// 全局 Deployment 管理 (跨命名空间)
	deployments := r.Group("/deployments")
	{
		deployments.GET("", h.deploymentHandler.ListDeployments)
	}
}
