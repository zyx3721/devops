package handler

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"devops/internal/service/logs"
	"devops/pkg/dto"
	"devops/pkg/response"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// ExportHandler 日志导出处理器
type ExportHandler struct {
	queryService *logs.QueryService
	tasks        sync.Map // map[string]*ExportTask
}

// ExportTask 导出任务
type ExportTask struct {
	ID        string
	Status    string // pending/processing/completed/failed
	Progress  int
	URL       string
	Error     string
	Data      []byte
	CreatedAt time.Time
	cancel    context.CancelFunc
}

// NewExportHandler 创建导出处理器
func NewExportHandler(query *logs.QueryService) *ExportHandler {
	return &ExportHandler{
		queryService: query,
	}
}

// ExportLogs 导出日志
// @Summary 导出日志
// @Tags 日志中心
// @Accept json
// @Produce json
// @Param body body dto.LogExportRequest true "导出请求"
// @Success 200 {object} response.Response{data=dto.LogExportResponse}
// @Router /api/v1/logs/export [post]
func (h *ExportHandler) ExportLogs(c *gin.Context) {
	var req dto.LogExportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	// 创建带超时的 context（5分钟超时）
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)

	// 创建导出任务
	taskID := uuid.New().String()
	task := &ExportTask{
		ID:        taskID,
		Status:    "pending",
		Progress:  0,
		CreatedAt: time.Now(),
		cancel:    cancel,
	}
	h.tasks.Store(taskID, task)

	// 异步执行导出
	go h.processExport(ctx, task, &req)

	response.Success(c, dto.LogExportResponse{
		TaskID:   taskID,
		Status:   task.Status,
		Progress: task.Progress,
	})
}

// processExport 处理导出任务
func (h *ExportHandler) processExport(ctx context.Context, task *ExportTask, req *dto.LogExportRequest) {
	defer task.cancel()

	task.Status = "processing"
	task.Progress = 10

	// 查询日志
	queryReq := &dto.LogQueryRequest{
		ClusterID: req.ClusterID,
		Namespace: req.Namespace,
		PodNames:  req.PodNames,
		StartTime: req.StartTime,
		EndTime:   req.EndTime,
		Keyword:   req.Keyword,
		Level:     req.Level,
		PageSize:  100000, // 最大导出10万条
	}

	resp, err := h.queryService.Query(ctx, queryReq)
	if err != nil {
		task.Status = "failed"
		task.Error = err.Error()
		return
	}

	task.Progress = 50

	// 根据格式生成数据
	var data []byte
	switch req.Format {
	case "txt":
		data = h.formatTXT(resp.Items)
	case "json":
		data, _ = h.formatJSON(resp.Items)
	case "csv":
		data = h.formatCSV(resp.Items)
	default:
		data = h.formatTXT(resp.Items)
	}

	task.Progress = 80

	// 如果数据超过100MB，压缩
	if len(data) > 100*1024*1024 {
		data, err = h.compressData(data, req.Format)
		if err != nil {
			task.Status = "failed"
			task.Error = err.Error()
			return
		}
	}

	task.Data = data
	task.Progress = 100
	task.Status = "completed"
	task.URL = fmt.Sprintf("/api/v1/logs/export/%s/download", task.ID)
}

// formatTXT 格式化为 TXT
func (h *ExportHandler) formatTXT(logs []dto.LogEntry) []byte {
	var buf bytes.Buffer
	for _, log := range logs {
		if log.PodName != "" {
			buf.WriteString(fmt.Sprintf("[%s][%s][%s][%s] %s\n",
				log.Timestamp, log.PodName, log.Container, log.Level, log.Content))
		} else {
			buf.WriteString(fmt.Sprintf("[%s][%s] %s\n",
				log.Timestamp, log.Level, log.Content))
		}
	}
	return buf.Bytes()
}

// formatJSON 格式化为 JSON
func (h *ExportHandler) formatJSON(logs []dto.LogEntry) ([]byte, error) {
	return json.MarshalIndent(logs, "", "  ")
}

// formatCSV 格式化为 CSV
func (h *ExportHandler) formatCSV(logs []dto.LogEntry) []byte {
	var buf bytes.Buffer
	writer := csv.NewWriter(&buf)

	// 写入表头
	writer.Write([]string{"Timestamp", "Pod", "Container", "Level", "Content"})

	// 写入数据
	for _, log := range logs {
		writer.Write([]string{
			log.Timestamp,
			log.PodName,
			log.Container,
			log.Level,
			log.Content,
		})
	}

	writer.Flush()
	return buf.Bytes()
}

// compressData 压缩数据
func (h *ExportHandler) compressData(data []byte, format string) ([]byte, error) {
	var buf bytes.Buffer
	zipWriter := zip.NewWriter(&buf)

	filename := fmt.Sprintf("logs.%s", format)
	writer, err := zipWriter.Create(filename)
	if err != nil {
		return nil, err
	}

	_, err = writer.Write(data)
	if err != nil {
		return nil, err
	}

	if err := zipWriter.Close(); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// GetExportStatus 获取导出状态
// @Summary 获取导出任务状态
// @Tags 日志中心
// @Param task_id path string true "任务ID"
// @Success 200 {object} response.Response{data=dto.LogExportResponse}
// @Router /api/v1/logs/export/{task_id} [get]
func (h *ExportHandler) GetExportStatus(c *gin.Context) {
	taskID := c.Param("task_id")

	taskI, ok := h.tasks.Load(taskID)
	if !ok {
		response.Error(c, http.StatusNotFound, "任务不存在")
		return
	}

	task := taskI.(*ExportTask)
	response.Success(c, dto.LogExportResponse{
		TaskID:   task.ID,
		Status:   task.Status,
		Progress: task.Progress,
		URL:      task.URL,
		Error:    task.Error,
	})
}

// DownloadExport 下载导出文件
// @Summary 下载导出文件
// @Tags 日志中心
// @Param task_id path string true "任务ID"
// @Produce octet-stream
// @Router /api/v1/logs/export/{task_id}/download [get]
func (h *ExportHandler) DownloadExport(c *gin.Context) {
	taskID := c.Param("task_id")

	taskI, ok := h.tasks.Load(taskID)
	if !ok {
		response.Error(c, http.StatusNotFound, "任务不存在")
		return
	}

	task := taskI.(*ExportTask)
	if task.Status != "completed" {
		response.Error(c, http.StatusBadRequest, "任务未完成")
		return
	}

	// 设置响应头
	filename := fmt.Sprintf("logs_%s.txt", time.Now().Format("20060102_150405"))
	if len(task.Data) > 100*1024*1024 {
		filename = fmt.Sprintf("logs_%s.zip", time.Now().Format("20060102_150405"))
	}

	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	c.Header("Content-Type", "application/octet-stream")
	c.Header("Content-Length", fmt.Sprintf("%d", len(task.Data)))
	c.Data(http.StatusOK, "application/octet-stream", task.Data)

	// 下载后删除任务
	h.tasks.Delete(taskID)
}

// CancelExport 取消导出任务
// @Summary 取消导出任务
// @Tags 日志中心
// @Param task_id path string true "任务ID"
// @Success 200 {object} response.Response
// @Router /api/v1/logs/export/{task_id}/cancel [post]
func (h *ExportHandler) CancelExport(c *gin.Context) {
	taskID := c.Param("task_id")

	taskI, ok := h.tasks.Load(taskID)
	if !ok {
		response.Error(c, http.StatusNotFound, "任务不存在")
		return
	}

	task := taskI.(*ExportTask)
	if task.Status == "processing" {
		task.Status = "failed"
		task.Error = "用户取消"
	}

	h.tasks.Delete(taskID)
	response.Success(c, nil)
}
