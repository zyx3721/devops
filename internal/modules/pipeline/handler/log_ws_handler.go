package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"gorm.io/gorm"

	"devops/internal/config"
	"devops/internal/models"
	"devops/internal/service/pipeline"
	"devops/pkg/ioc"
	"devops/pkg/logger"
	"devops/pkg/middleware"
)

var wsLog = logger.L().WithField("module", "pipeline_ws")

func init() {
	ioc.Api.RegisterContainer("PipelineLogWSHandler", &PipelineLogWSApiHandler{})
}

type PipelineLogWSApiHandler struct {
	handler *LogWSHandler
}

func (h *PipelineLogWSApiHandler) Init() error {
	cfg, err := config.LoadConfig()
	if err != nil {
		return err
	}

	db := cfg.GetDB()
	h.handler = NewLogWSHandler(db)

	root := cfg.Application.GinRootRouter().Group("pipelines")
	root.Use(middleware.AuthMiddleware())
	h.Register(root)

	return nil
}

func (h *PipelineLogWSApiHandler) Register(r gin.IRouter) {
	// WebSocket 日志流
	r.GET("/runs/:id/logs/stream", h.handler.StreamLogs)
	// 按步骤获取日志流
	r.GET("/runs/:id/steps/:step_id/logs/stream", h.handler.StreamStepLogs)
}

// LogWSHandler WebSocket 日志处理器
type LogWSHandler struct {
	db         *gorm.DB
	logService *pipeline.LogService
	upgrader   websocket.Upgrader
	clients    map[uint]map[*websocket.Conn]bool // runID -> connections
	mu         sync.RWMutex
}

// NewLogWSHandler 创建 WebSocket 日志处理器
func NewLogWSHandler(db *gorm.DB) *LogWSHandler {
	return &LogWSHandler{
		db:         db,
		logService: pipeline.NewLogService(db),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // 允许所有来源，生产环境应该限制
			},
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
		},
		clients: make(map[uint]map[*websocket.Conn]bool),
	}
}

// LogMessage WebSocket 日志消息
type LogMessage struct {
	Type      string `json:"type"`                  // log, status, error
	RunID     uint   `json:"run_id"`                // 流水线运行ID
	StepRunID uint   `json:"step_run_id,omitempty"` // 步骤运行ID
	StepName  string `json:"step_name,omitempty"`   // 步骤名称
	Content   string `json:"content"`               // 日志内容
	Timestamp int64  `json:"timestamp"`             // 时间戳
	Level     string `json:"level,omitempty"`       // info, warn, error
}

// StreamLogs 流式获取流水线运行日志
// @Summary WebSocket 日志流
// @Description 通过 WebSocket 实时获取流水线运行日志
// @Tags 流水线
// @Param id path int true "运行ID"
// @Success 101 {string} string "WebSocket 连接成功"
// @Router /pipelines/runs/{id}/logs/stream [get]
func (h *LogWSHandler) StreamLogs(c *gin.Context) {
	runID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的运行ID"})
		return
	}

	// 验证运行记录存在
	var pipelineRun models.PipelineRun
	if err := h.db.First(&pipelineRun, runID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "运行记录不存在"})
		return
	}

	// 升级为 WebSocket 连接
	conn, err := h.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		wsLog.WithError(err).Error("WebSocket 升级失败")
		return
	}
	defer conn.Close()

	// 注册客户端
	h.registerClient(uint(runID), conn)
	defer h.unregisterClient(uint(runID), conn)

	wsLog.WithField("run_id", runID).Info("WebSocket 连接建立")

	// 发送历史日志
	h.sendHistoryLogs(conn, uint(runID))

	// 如果运行还在进行中，开始流式推送
	if pipelineRun.Status == "running" || pipelineRun.Status == "pending" {
		h.streamRunLogs(c.Request.Context(), conn, uint(runID))
	}

	// 保持连接，等待客户端关闭
	h.handleClientMessages(conn)
}

// StreamStepLogs 流式获取步骤日志
// @Summary WebSocket 步骤日志流
// @Description 通过 WebSocket 实时获取步骤运行日志
// @Tags 流水线
// @Param id path int true "运行ID"
// @Param step_id path int true "步骤运行ID"
// @Success 101 {string} string "WebSocket 连接成功"
// @Router /pipelines/runs/{id}/steps/{step_id}/logs/stream [get]
func (h *LogWSHandler) StreamStepLogs(c *gin.Context) {
	runID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的运行ID"})
		return
	}

	stepRunID, err := strconv.ParseUint(c.Param("step_id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的步骤ID"})
		return
	}

	// 验证步骤运行记录存在
	var stepRun models.StepRun
	if err := h.db.First(&stepRun, stepRunID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "步骤运行记录不存在"})
		return
	}

	// 升级为 WebSocket 连接
	conn, err := h.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		wsLog.WithError(err).Error("WebSocket 升级失败")
		return
	}
	defer conn.Close()

	wsLog.WithField("run_id", runID).WithField("step_run_id", stepRunID).Info("步骤日志 WebSocket 连接建立")

	// 发送历史日志
	h.sendStepHistoryLogs(conn, uint(stepRunID))

	// 如果步骤还在运行，开始流式推送
	if stepRun.Status == "running" {
		h.streamStepLogs(c.Request.Context(), conn, uint(runID), uint(stepRunID))
	}

	// 保持连接
	h.handleClientMessages(conn)
}

// registerClient 注册客户端
func (h *LogWSHandler) registerClient(runID uint, conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.clients[runID] == nil {
		h.clients[runID] = make(map[*websocket.Conn]bool)
	}
	h.clients[runID][conn] = true
}

// unregisterClient 注销客户端
func (h *LogWSHandler) unregisterClient(runID uint, conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.clients[runID] != nil {
		delete(h.clients[runID], conn)
		if len(h.clients[runID]) == 0 {
			delete(h.clients, runID)
		}
	}
}

// sendHistoryLogs 发送历史日志
func (h *LogWSHandler) sendHistoryLogs(conn *websocket.Conn, runID uint) {
	// 获取所有阶段运行记录
	var stageRuns []models.StageRun
	if err := h.db.Where("pipeline_run_id = ?", runID).Find(&stageRuns).Error; err != nil {
		wsLog.WithError(err).Error("获取阶段运行记录失败")
		return
	}

	// 获取所有步骤运行记录
	var stageRunIDs []uint
	for _, sr := range stageRuns {
		stageRunIDs = append(stageRunIDs, sr.ID)
	}

	if len(stageRunIDs) == 0 {
		return
	}

	var stepRuns []models.StepRun
	if err := h.db.Where("stage_run_id IN ?", stageRunIDs).Order("started_at ASC").Find(&stepRuns).Error; err != nil {
		wsLog.WithError(err).Error("获取步骤运行记录失败")
		return
	}

	for _, stepRun := range stepRuns {
		if stepRun.Logs != "" {
			msg := LogMessage{
				Type:      "log",
				RunID:     runID,
				StepRunID: stepRun.ID,
				StepName:  stepRun.StepName,
				Content:   h.logService.SanitizeLogs(stepRun.Logs),
				Timestamp: time.Now().UnixMilli(),
				Level:     "info",
			}
			h.sendMessage(conn, msg)
		}
	}
}

// sendStepHistoryLogs 发送步骤历史日志
func (h *LogWSHandler) sendStepHistoryLogs(conn *websocket.Conn, stepRunID uint) {
	logs, err := h.logService.GetStepRunLogs(context.Background(), stepRunID)
	if err != nil {
		wsLog.WithError(err).Error("获取步骤日志失败")
		return
	}

	if logs != "" {
		var stepRun models.StepRun
		h.db.First(&stepRun, stepRunID)

		msg := LogMessage{
			Type:      "log",
			StepRunID: stepRunID,
			StepName:  stepRun.StepName,
			Content:   logs,
			Timestamp: time.Now().UnixMilli(),
			Level:     "info",
		}
		h.sendMessage(conn, msg)
	}
}

// streamRunLogs 流式推送运行日志
func (h *LogWSHandler) streamRunLogs(ctx context.Context, conn *websocket.Conn, runID uint) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	lastLogLengths := make(map[uint]int) // 记录每个步骤的日志长度

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// 检查运行状态
			var pipelineRun models.PipelineRun
			if err := h.db.First(&pipelineRun, runID).Error; err != nil {
				return
			}

			// 获取所有阶段运行记录
			var stageRuns []models.StageRun
			h.db.Where("pipeline_run_id = ?", runID).Find(&stageRuns)

			var stageRunIDs []uint
			for _, sr := range stageRuns {
				stageRunIDs = append(stageRunIDs, sr.ID)
			}

			if len(stageRunIDs) == 0 {
				continue
			}

			// 获取正在运行的步骤
			var stepRuns []models.StepRun
			h.db.Where("stage_run_id IN ? AND status = ?", stageRunIDs, "running").Find(&stepRuns)

			for _, stepRun := range stepRuns {
				// 获取新增的日志
				logs, err := h.logService.GetStepRunLogs(ctx, stepRun.ID)
				if err != nil {
					continue
				}

				lastLen := lastLogLengths[stepRun.ID]
				if len(logs) > lastLen {
					newLogs := logs[lastLen:]
					lastLogLengths[stepRun.ID] = len(logs)

					msg := LogMessage{
						Type:      "log",
						RunID:     runID,
						StepRunID: stepRun.ID,
						StepName:  stepRun.StepName,
						Content:   newLogs,
						Timestamp: time.Now().UnixMilli(),
						Level:     "info",
					}
					h.sendMessage(conn, msg)
				}
			}

			// 如果运行完成，发送状态消息并退出
			if pipelineRun.Status != "running" && pipelineRun.Status != "pending" {
				msg := LogMessage{
					Type:      "status",
					RunID:     runID,
					Content:   pipelineRun.Status,
					Timestamp: time.Now().UnixMilli(),
				}
				h.sendMessage(conn, msg)
				return
			}
		}
	}
}

// streamStepLogs 流式推送步骤日志
func (h *LogWSHandler) streamStepLogs(ctx context.Context, conn *websocket.Conn, runID, stepRunID uint) {
	var stepRun models.StepRun
	if err := h.db.First(&stepRun, stepRunID).Error; err != nil {
		return
	}

	// 如果有关联的构建任务，使用 K8s 日志流
	if stepRun.BuildJobID != nil {
		h.logService.StreamBuildJobLogs(ctx, *stepRun.BuildJobID, func(line string) error {
			msg := LogMessage{
				Type:      "log",
				RunID:     runID,
				StepRunID: stepRunID,
				StepName:  stepRun.StepName,
				Content:   line,
				Timestamp: time.Now().UnixMilli(),
				Level:     "info",
			}
			return h.sendMessage(conn, msg)
		})
	}
}

// sendMessage 发送消息
func (h *LogWSHandler) sendMessage(conn *websocket.Conn, msg LogMessage) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	return conn.WriteMessage(websocket.TextMessage, data)
}

// handleClientMessages 处理客户端消息
func (h *LogWSHandler) handleClientMessages(conn *websocket.Conn) {
	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				wsLog.WithError(err).Warn("WebSocket 连接异常关闭")
			}
			return
		}
	}
}

// BroadcastLog 广播日志到所有订阅的客户端
func (h *LogWSHandler) BroadcastLog(runID uint, msg LogMessage) {
	h.mu.RLock()
	clients := h.clients[runID]
	h.mu.RUnlock()

	for conn := range clients {
		if err := h.sendMessage(conn, msg); err != nil {
			wsLog.WithError(err).Warn("广播日志失败")
			h.unregisterClient(runID, conn)
		}
	}
}

// GetConnectedClients 获取连接的客户端数量
func (h *LogWSHandler) GetConnectedClients(runID uint) int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients[runID])
}
