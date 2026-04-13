package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"devops/internal/service/logs"
	"devops/pkg/dto"
	"devops/pkg/logger"
	"devops/pkg/response"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

// LogHandler 日志处理器
type LogHandler struct {
	queryService      *logs.QueryService
	streamService     *logs.StreamService
	parserService     *logs.ParserService
	contextService    *logs.ContextService
	savedQueryService *logs.SavedQueryService
}

// NewLogHandler 创建日志处理器
func NewLogHandler(query *logs.QueryService, stream *logs.StreamService, parser *logs.ParserService, context *logs.ContextService, savedQuery *logs.SavedQueryService) *LogHandler {
	return &LogHandler{
		queryService:      query,
		streamService:     stream,
		parserService:     parser,
		contextService:    context,
		savedQueryService: savedQuery,
	}
}

// StreamLogs WebSocket 日志流
// @Summary WebSocket 日志流
// @Tags 日志中心
// @Param cluster_id query int true "集群ID"
// @Param namespace query string true "命名空间"
// @Param pod_name query string true "Pod名称"
// @Param container query string false "容器名称"
// @Param tail_lines query int false "初始行数"
// @Param follow query bool false "是否实时跟踪"
// @Router /api/v1/logs/stream [get]
func (h *LogHandler) StreamLogs(c *gin.Context) {
	clusterID, _ := strconv.ParseInt(c.Query("cluster_id"), 10, 64)
	namespace := c.Query("namespace")
	podName := c.Query("pod_name")
	container := c.Query("container")
	tailLines, _ := strconv.ParseInt(c.DefaultQuery("tail_lines", "100"), 10, 64)
	follow := c.Query("follow") != "false"
	keyword := c.Query("keyword")
	level := c.Query("level")

	// 升级为 WebSocket
	ws, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		logger.L().Error("WebSocket upgrade failed: %v", err)
		return
	}

	req := &dto.LogStreamRequest{
		ClusterID: clusterID,
		Namespace: namespace,
		PodName:   podName,
		Container: container,
		TailLines: tailLines,
		Follow:    follow,
		Keyword:   keyword,
		Level:     level,
	}

	connID := uuid.New().String()
	conn := logs.NewStreamConnection(connID, ws, req)

	if err := h.streamService.HandleConnection(c.Request.Context(), conn); err != nil {
		logger.L().Error("Stream logs error: %v", err)
	}
}

// QueryLogs 查询日志
// @Summary 查询日志
// @Tags 日志中心
// @Accept json
// @Produce json
// @Param cluster_id query int true "集群ID"
// @Param namespace query string true "命名空间"
// @Param pod_names query []string false "Pod名称列表"
// @Param keyword query string false "关键词"
// @Param regex query string false "正则表达式"
// @Param level query string false "日志级别"
// @Param start_time query string false "开始时间"
// @Param end_time query string false "结束时间"
// @Param page query int false "页码"
// @Param page_size query int false "每页数量"
// @Success 200 {object} response.Response{data=dto.LogQueryResponse}
// @Router /api/v1/logs/query [get]
func (h *LogHandler) QueryLogs(c *gin.Context) {
	var req dto.LogQueryRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	// 设置默认值
	if req.Page < 1 {
		req.Page = 1
	}
	if req.PageSize < 1 {
		req.PageSize = 100
	}
	if req.PageSize > 1000 {
		req.PageSize = 1000
	}

	resp, err := h.queryService.Query(c.Request.Context(), &req)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, resp)
}

// GetLogContext 获取日志上下文
// @Summary 获取日志上下文
// @Tags 日志中心
// @Accept json
// @Produce json
// @Param cluster_id query int true "集群ID"
// @Param namespace query string true "命名空间"
// @Param pod_name query string true "Pod名称"
// @Param container query string false "容器名称"
// @Param timestamp query string true "时间戳"
// @Param lines_before query int false "前面行数"
// @Param lines_after query int false "后面行数"
// @Success 200 {object} response.Response{data=dto.LogContextResponse}
// @Router /api/v1/logs/context [get]
func (h *LogHandler) GetLogContext(c *gin.Context) {
	var req dto.LogContextRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	// 设置默认值
	if req.LinesBefore <= 0 {
		req.LinesBefore = 100
	}
	if req.LinesAfter <= 0 {
		req.LinesAfter = 100
	}

	// 获取上下文日志
	queryReq := &dto.LogQueryRequest{
		ClusterID: req.ClusterID,
		Namespace: req.Namespace,
		PodNames:  []string{req.PodName},
		EndTime:   req.Timestamp,
		PageSize:  req.LinesBefore + req.LinesAfter + 1,
		Order:     "desc",
	}
	if req.Container != "" {
		queryReq.Containers = []string{req.Container}
	}

	resp, err := h.queryService.Query(c.Request.Context(), queryReq)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	// 构建上下文响应
	contextResp := &dto.LogContextResponse{
		Before:      []dto.LogEntry{},
		After:       []dto.LogEntry{},
		TotalBefore: 0,
		TotalAfter:  0,
	}

	// 查找当前行并分割
	for i, entry := range resp.Items {
		if entry.Timestamp == req.Timestamp {
			contextResp.Current = entry
			if i > 0 {
				contextResp.Before = resp.Items[:i]
				contextResp.TotalBefore = len(contextResp.Before)
			}
			if i < len(resp.Items)-1 {
				contextResp.After = resp.Items[i+1:]
				contextResp.TotalAfter = len(contextResp.After)
			}
			break
		}
	}

	response.Success(c, contextResp)
}

// GetContainers 获取容器列表
// @Summary 获取Pod容器列表
// @Tags 日志中心
// @Param cluster_id path int true "集群ID"
// @Param namespace path string true "命名空间"
// @Param pod_name path string true "Pod名称"
// @Success 200 {object} response.Response{data=[]string}
// @Router /api/v1/logs/containers/{cluster_id}/{namespace}/{pod_name} [get]
func (h *LogHandler) GetContainers(c *gin.Context) {
	clusterID, _ := strconv.ParseInt(c.Param("cluster_id"), 10, 64)
	namespace := c.Param("namespace")
	podName := c.Param("pod_name")

	containers, err := h.queryService.GetContainers(c.Request.Context(), clusterID, namespace, podName)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, containers)
}

// ParseLog 解析日志
// @Summary 解析日志内容
// @Tags 日志中心
// @Accept json
// @Produce json
// @Param body body dto.ParseTestRequest true "解析请求"
// @Success 200 {object} response.Response{data=dto.ParseTestResponse}
// @Router /api/v1/logs/parse [post]
func (h *LogHandler) ParseLog(c *gin.Context) {
	var req dto.ParseTestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	resp := h.parserService.TestTemplate(&req)
	response.Success(c, resp)
}

// StreamMultiPodLogs 多 Pod WebSocket 日志流
// @Summary 多 Pod WebSocket 日志流
// @Tags 日志中心
// @Param cluster_id query int true "集群ID"
// @Param namespace query string true "命名空间"
// @Param pods query string true "Pod列表(JSON格式)"
// @Param tail_lines query int false "初始行数"
// @Router /api/v1/logs/stream/multi [get]
func (h *LogHandler) StreamMultiPodLogs(c *gin.Context) {
	clusterID, _ := strconv.ParseInt(c.Query("cluster_id"), 10, 64)
	namespace := c.Query("namespace")
	tailLines, _ := strconv.ParseInt(c.DefaultQuery("tail_lines", "100"), 10, 64)
	keyword := c.Query("keyword")
	level := c.Query("level")

	// 解析 pods 参数
	podsJSON := c.Query("pods")
	var pods []dto.PodLogRequest
	if podsJSON != "" {
		if err := json.Unmarshal([]byte(podsJSON), &pods); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid pods parameter"})
			return
		}
	}

	// 验证 Pod 数量限制
	if len(pods) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "At least one pod is required"})
		return
	}
	if len(pods) > 10 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Maximum 10 pods allowed"})
		return
	}

	// 升级为 WebSocket
	ws, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		logger.L().Error("WebSocket upgrade failed: %v", err)
		return
	}

	req := &dto.MultiPodLogStreamRequest{
		ClusterID: clusterID,
		Namespace: namespace,
		Pods:      pods,
		TailLines: tailLines,
		Follow:    true,
		Keyword:   keyword,
		Level:     level,
	}

	connID := uuid.New().String()
	conn := logs.NewMultiPodStreamConnection(connID, ws, req)

	if err := h.streamService.HandleMultiPodConnection(c.Request.Context(), conn); err != nil {
		logger.L().Error("Multi-pod stream logs error: %v", err)
	}
}

// ListSavedQueries 获取快捷查询列表
// @Summary 获取快捷查询列表
// @Tags 日志中心
// @Produce json
// @Param include_shared query bool false "是否包含共享查询"
// @Success 200 {object} response.Response{data=[]dto.SavedQueryResponse}
// @Router /api/v1/logs/saved-queries [get]
func (h *LogHandler) ListSavedQueries(c *gin.Context) {
	userID, _ := c.Get("user_id")
	includeShared := c.Query("include_shared") == "true"

	uid, ok := userID.(uint)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "用户未登录")
		return
	}

	queries, err := h.savedQueryService.List(int64(uid), includeShared)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, queries)
}

// CreateSavedQuery 创建快捷查询
// @Summary 创建快捷查询
// @Tags 日志中心
// @Accept json
// @Produce json
// @Param body body dto.SavedQueryRequest true "查询请求"
// @Success 200 {object} response.Response{data=dto.SavedQueryResponse}
// @Router /api/v1/logs/saved-queries [post]
func (h *LogHandler) CreateSavedQuery(c *gin.Context) {
	userID, _ := c.Get("user_id")
	uid, ok := userID.(uint)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "用户未登录")
		return
	}

	var req dto.SavedQueryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	result, err := h.savedQueryService.Create(int64(uid), &req)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, result)
}

// UpdateSavedQuery 更新快捷查询
// @Summary 更新快捷查询
// @Tags 日志中心
// @Accept json
// @Produce json
// @Param id path int true "查询ID"
// @Param body body dto.SavedQueryRequest true "查询请求"
// @Success 200 {object} response.Response{data=dto.SavedQueryResponse}
// @Router /api/v1/logs/saved-queries/{id} [put]
func (h *LogHandler) UpdateSavedQuery(c *gin.Context) {
	userID, _ := c.Get("user_id")
	uid, ok := userID.(uint)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "用户未登录")
		return
	}
	queryID, _ := strconv.ParseInt(c.Param("id"), 10, 64)

	var req dto.SavedQueryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	result, err := h.savedQueryService.Update(int64(uid), queryID, &req)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, result)
}

// DeleteSavedQuery 删除快捷查询
// @Summary 删除快捷查询
// @Tags 日志中心
// @Param id path int true "查询ID"
// @Success 200 {object} response.Response
// @Router /api/v1/logs/saved-queries/{id} [delete]
func (h *LogHandler) DeleteSavedQuery(c *gin.Context) {
	userID, _ := c.Get("user_id")
	uid, ok := userID.(uint)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "用户未登录")
		return
	}
	queryID, _ := strconv.ParseInt(c.Param("id"), 10, 64)

	if err := h.savedQueryService.Delete(int64(uid), queryID); err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, nil)
}

// UseSavedQuery 使用快捷查询
// @Summary 使用快捷查询（增加使用次数）
// @Tags 日志中心
// @Param id path int true "查询ID"
// @Success 200 {object} response.Response
// @Router /api/v1/logs/saved-queries/{id}/use [post]
func (h *LogHandler) UseSavedQuery(c *gin.Context) {
	queryID, _ := strconv.ParseInt(c.Param("id"), 10, 64)

	if err := h.savedQueryService.Use(queryID); err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, nil)
}
