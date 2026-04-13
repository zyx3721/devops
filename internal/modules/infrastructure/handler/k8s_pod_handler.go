package handler

import (
	"devops/internal/service/kubernetes"
	"devops/pkg/logger"
	"devops/pkg/response"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type K8sPodHandler struct {
	podService      *kubernetes.K8sPodService
	terminalService *kubernetes.K8sTerminalService
}

func NewK8sPodHandler(podSvc *kubernetes.K8sPodService, termSvc *kubernetes.K8sTerminalService) *K8sPodHandler {
	return &K8sPodHandler{
		podService:      podSvc,
		terminalService: termSvc,
	}
}

// ListPods 获取 Pod 列表
// @Summary 获取 Pod 列表
// @Tags K8s Pod
// @Param id path int true "集群ID"
// @Param ns path string true "命名空间"
// @Param label_selector query string false "标签选择器"
// @Success 200 {object} response.Response
// @Router /api/v1/k8s/clusters/{id}/namespaces/{ns}/pods [get]
func (h *K8sPodHandler) ListPods(c *gin.Context) {
	clusterID, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	namespace := c.Param("ns")
	labelSelector := c.Query("label_selector")

	pods, err := h.podService.ListPods(c.Request.Context(), uint(clusterID), namespace, labelSelector)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, pods)
}

// GetPod 获取 Pod 详情
// @Summary 获取 Pod 详情
// @Tags K8s Pod
// @Param id path int true "集群ID"
// @Param ns path string true "命名空间"
// @Param name path string true "Pod名称"
// @Success 200 {object} response.Response
// @Router /api/v1/k8s/clusters/{id}/namespaces/{ns}/pods/{name} [get]
func (h *K8sPodHandler) GetPod(c *gin.Context) {
	clusterID, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	namespace := c.Param("ns")
	name := c.Param("name")

	pod, err := h.podService.GetPod(c.Request.Context(), uint(clusterID), namespace, name)
	if err != nil {
		response.Error(c, http.StatusNotFound, err.Error())
		return
	}

	response.Success(c, pod)
}

// DeletePod 删除 Pod
// @Summary 删除 Pod
// @Tags K8s Pod
// @Param id path int true "集群ID"
// @Param ns path string true "命名空间"
// @Param name path string true "Pod名称"
// @Success 200 {object} response.Response
// @Router /api/v1/k8s/clusters/{id}/namespaces/{ns}/pods/{name} [delete]
func (h *K8sPodHandler) DeletePod(c *gin.Context) {
	clusterID, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	namespace := c.Param("ns")
	name := c.Param("name")

	if err := h.podService.DeletePod(c.Request.Context(), uint(clusterID), namespace, name); err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, nil)
}

// GetPodContainers 获取 Pod 容器列表
// @Summary 获取 Pod 容器列表
// @Tags K8s Pod
// @Param id path int true "集群ID"
// @Param ns path string true "命名空间"
// @Param name path string true "Pod名称"
// @Success 200 {object} response.Response
// @Router /api/v1/k8s/clusters/{id}/namespaces/{ns}/pods/{name}/containers [get]
func (h *K8sPodHandler) GetPodContainers(c *gin.Context) {
	clusterID, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	namespace := c.Param("ns")
	name := c.Param("name")

	containers, err := h.podService.GetPodContainers(c.Request.Context(), uint(clusterID), namespace, name)
	if err != nil {
		response.Error(c, http.StatusNotFound, err.Error())
		return
	}

	response.Success(c, containers)
}

// GetPodLogs 获取 Pod 日志
// @Summary 获取 Pod 日志
// @Tags K8s Pod
// @Param id path int true "集群ID"
// @Param ns path string true "命名空间"
// @Param name path string true "Pod名称"
// @Param container query string false "容器名称"
// @Param tail_lines query int false "行数"
// @Param timestamps query bool false "是否显示时间戳"
// @Success 200 {object} response.Response
// @Router /api/v1/k8s/clusters/{id}/namespaces/{ns}/pods/{name}/logs [get]
func (h *K8sPodHandler) GetPodLogs(c *gin.Context) {
	clusterID, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	namespace := c.Param("ns")
	name := c.Param("name")
	container := c.Query("container")
	tailLines, _ := strconv.ParseInt(c.DefaultQuery("tail_lines", "100"), 10, 64)
	timestamps := c.Query("timestamps") == "true"

	req := &kubernetes.LogRequest{
		ClusterID:  uint(clusterID),
		Namespace:  namespace,
		PodName:    name,
		Container:  container,
		TailLines:  tailLines,
		Timestamps: timestamps,
	}

	logs, err := h.podService.GetLogs(c.Request.Context(), req)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, gin.H{"logs": logs})
}

// StreamPodLogs WebSocket 流式日志
// @Summary WebSocket 流式日志
// @Tags K8s Pod
// @Param id path int true "集群ID"
// @Param ns path string true "命名空间"
// @Param name path string true "Pod名称"
// @Router /api/v1/k8s/clusters/{id}/namespaces/{ns}/pods/{name}/logs/stream [get]
func (h *K8sPodHandler) StreamPodLogs(c *gin.Context) {
	clusterID, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	namespace := c.Param("ns")
	name := c.Param("name")
	container := c.Query("container")
	tailLines, _ := strconv.ParseInt(c.DefaultQuery("tail_lines", "100"), 10, 64)
	timestamps := c.Query("timestamps") == "true"

	// 升级为 WebSocket
	ws, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		logger.L().Error("WebSocket upgrade failed: %v", err)
		return
	}
	defer ws.Close()

	req := &kubernetes.LogRequest{
		ClusterID:  uint(clusterID),
		Namespace:  namespace,
		PodName:    name,
		Container:  container,
		TailLines:  tailLines,
		Follow:     true,
		Timestamps: timestamps,
	}

	// 创建一个 writer 将日志写入 WebSocket
	writer := &wsWriter{ws: ws}
	ctx := c.Request.Context()

	if err := h.podService.StreamLogs(ctx, req, writer); err != nil {
		logger.L().Error("Stream logs error: %v", err)
	}
}

// DownloadPodLogs 下载 Pod 日志
// @Summary 下载 Pod 日志
// @Tags K8s Pod
// @Param id path int true "集群ID"
// @Param ns path string true "命名空间"
// @Param name path string true "Pod名称"
// @Param container query string false "容器名称"
// @Router /api/v1/k8s/clusters/{id}/namespaces/{ns}/pods/{name}/logs/download [get]
func (h *K8sPodHandler) DownloadPodLogs(c *gin.Context) {
	clusterID, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	namespace := c.Param("ns")
	name := c.Param("name")
	container := c.Query("container")

	req := &kubernetes.LogRequest{
		ClusterID:  uint(clusterID),
		Namespace:  namespace,
		PodName:    name,
		Container:  container,
		TailLines:  10000, // 下载最多10000行
		Timestamps: true,
	}

	logs, err := h.podService.GetLogs(c.Request.Context(), req)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	filename := name + ".log"
	if container != "" {
		filename = name + "-" + container + ".log"
	}

	c.Header("Content-Disposition", "attachment; filename="+filename)
	c.Header("Content-Type", "text/plain")
	c.String(http.StatusOK, logs)
}

// PodTerminal WebSocket Pod 终端
// @Summary WebSocket Pod 终端
// @Tags K8s Pod
// @Param id path int true "集群ID"
// @Param ns path string true "命名空间"
// @Param name path string true "Pod名称"
// @Param container query string false "容器名称"
// @Param shell query string false "Shell类型"
// @Router /api/v1/k8s/clusters/{id}/namespaces/{ns}/pods/{name}/terminal [get]
func (h *K8sPodHandler) PodTerminal(c *gin.Context) {
	clusterID, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	namespace := c.Param("ns")
	name := c.Param("name")
	container := c.Query("container")
	shell := c.DefaultQuery("shell", "/bin/sh")

	logger.L().Info("终端连接请求: cluster=%d, ns=%s, pod=%s, container=%s, shell=%s",
		clusterID, namespace, name, container, shell)

	// 如果没有指定容器，获取第一个容器
	if container == "" {
		containers, err := h.podService.GetPodContainers(c.Request.Context(), uint(clusterID), namespace, name)
		if err != nil {
			logger.L().Error("获取容器列表失败: %v", err)
			c.JSON(400, gin.H{"code": 400, "message": "获取容器列表失败: " + err.Error()})
			return
		}
		if len(containers) > 0 {
			container = containers[0].Name
			logger.L().Info("使用默认容器: %s", container)
		} else {
			c.JSON(400, gin.H{"code": 400, "message": "Pod 没有容器"})
			return
		}
	}

	logger.L().Info("开始 WebSocket 升级...")

	// 升级为 WebSocket
	ws, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		logger.L().Error("WebSocket upgrade failed: %v", err)
		return
	}
	defer ws.Close()

	logger.L().Info("WebSocket 升级成功，创建终端会话...")

	session := h.terminalService.NewTerminalSession(uint(clusterID), namespace, name, container, shell)
	defer session.Close()

	ctx := c.Request.Context()
	if err := h.terminalService.HandleTerminal(ctx, session, ws); err != nil {
		logger.L().Error("Terminal error: %v", err)
	}

	logger.L().Info("终端会话结束")
}

// wsWriter WebSocket writer
type wsWriter struct {
	ws *websocket.Conn
}

func (w *wsWriter) Write(p []byte) (int, error) {
	// 直接发送整个数据块
	msg := map[string]string{
		"type": "log",
		"data": string(p),
	}
	if err := w.ws.WriteJSON(msg); err != nil {
		return 0, err
	}
	return len(p), nil
}
