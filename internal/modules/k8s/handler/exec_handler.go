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
	ioc.Api.RegisterContainer("K8sExecHandler", &K8sExecApiHandler{})
}

type K8sExecApiHandler struct {
	handler *K8sExecHandler
}

func (h *K8sExecApiHandler) Init() error {
	cfg, _ := config.LoadConfig()
	db := cfg.GetDB()
	clientMgr := kubernetes.NewK8sClientManager(db)
	execSvc := kubernetes.NewK8sExecService(clientMgr)
	h.handler = NewK8sExecHandler(execSvc)

	root := cfg.Application.GinRootRouter().Group("k8s/exec")
	root.Use(middleware.AuthMiddleware())
	h.Register(root)

	return nil
}

func (h *K8sExecApiHandler) Register(r gin.IRouter) {
	r.GET("/shell", h.handler.WebShell)
	r.GET("/shells", h.handler.GetAvailableShells)
}

// K8sExecHandler 执行处理器
type K8sExecHandler struct {
	execSvc *kubernetes.K8sExecService
}

func NewK8sExecHandler(execSvc *kubernetes.K8sExecService) *K8sExecHandler {
	return &K8sExecHandler{execSvc: execSvc}
}

var wsUpgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// WebShell WebSocket 终端
func (h *K8sExecHandler) WebShell(c *gin.Context) {
	log := logger.L().WithField("handler", "WebShell")

	clusterID, _ := strconv.ParseUint(c.Query("cluster_id"), 10, 32)
	namespace := c.Query("namespace")
	podName := c.Query("pod")
	container := c.Query("container")
	shell := c.Query("shell")

	if clusterID == 0 || namespace == "" || podName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cluster_id, namespace, pod 必填"})
		return
	}

	if shell == "" {
		shell = "/bin/sh"
	}

	// 升级为 WebSocket
	conn, err := wsUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.WithField("error", err).Error("WebSocket 升级失败")
		return
	}
	defer conn.Close()

	req := &kubernetes.ExecRequest{
		ClusterID: uint(clusterID),
		Namespace: namespace,
		PodName:   podName,
		Container: container,
		Command:   shell,
	}

	log.WithField("pod", podName).WithField("namespace", namespace).Info("WebShell 连接")

	if err := h.execSvc.ExecInPodWithWebSocket(c.Request.Context(), req, conn); err != nil {
		log.WithField("error", err).Error("WebShell 执行失败")
		// 尝试发送错误消息
		conn.WriteMessage(websocket.TextMessage, []byte("\r\n\033[31mError: "+err.Error()+"\033[0m\r\n"))
	}
}

// GetAvailableShells 获取可用的 shell
func (h *K8sExecHandler) GetAvailableShells(c *gin.Context) {
	clusterID, _ := strconv.ParseUint(c.Query("cluster_id"), 10, 32)
	namespace := c.Query("namespace")
	podName := c.Query("pod")
	container := c.Query("container")

	if clusterID == 0 || namespace == "" || podName == "" {
		response.BadRequest(c, "cluster_id, namespace, pod 必填")
		return
	}

	shells, err := h.execSvc.GetPodShells(c.Request.Context(), uint(clusterID), namespace, podName, container)
	if err != nil {
		response.InternalError(c, "获取 shell 列表失败: "+err.Error())
		return
	}

	response.Success(c, shells)
}
