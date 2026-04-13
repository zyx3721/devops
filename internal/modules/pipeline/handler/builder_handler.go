package handler

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"devops/internal/config"
	"devops/internal/service/pipeline"
	"devops/pkg/ioc"
	"devops/pkg/middleware"
	"devops/pkg/response"
)

func init() {
	ioc.Api.RegisterContainer("BuilderHandler", &BuilderApiHandler{})
}

// BuilderApiHandler IOC容器注册的处理器
type BuilderApiHandler struct {
	handler *BuilderHandler
}

func (h *BuilderApiHandler) Init() error {
	cfg, err := config.LoadConfig()
	if err != nil {
		return err
	}

	db := cfg.GetDB()
	runSvc := pipeline.NewRunService(db)

	h.handler = NewBuilderHandler(runSvc)

	root := cfg.Application.GinRootRouter().Group("builders")
	root.Use(middleware.AuthMiddleware())
	h.Register(root)

	return nil
}

func (h *BuilderApiHandler) Register(r gin.IRouter) {
	r.GET("/pods", h.handler.ListBuilderPods)
	r.DELETE("/pods/:pod_name", middleware.RequireAdmin(), h.handler.DeleteBuilderPod)
	r.PUT("/config", middleware.RequireAdmin(), h.handler.SaveConfig)
	r.GET("/config", h.handler.GetConfig)
}

// BuilderHandler 构建器处理器
type BuilderHandler struct {
	runSvc *pipeline.RunService
}

// NewBuilderHandler 创建构建器处理器
func NewBuilderHandler(runSvc *pipeline.RunService) *BuilderHandler {
	return &BuilderHandler{runSvc: runSvc}
}

// ListBuilderPods 获取活跃的构建 Pod 列表
func (h *BuilderHandler) ListBuilderPods(c *gin.Context) {
	pods := h.runSvc.GetActiveBuilderPods()
	response.Success(c, gin.H{
		"items": pods,
		"total": len(pods),
	})
}

// DeleteBuilderPod 删除构建 Pod
func (h *BuilderHandler) DeleteBuilderPod(c *gin.Context) {
	podName := c.Param("pod_name")
	clusterID, _ := strconv.ParseUint(c.Query("cluster_id"), 10, 64)
	namespace := c.Query("namespace")

	if podName == "" || clusterID == 0 || namespace == "" {
		response.BadRequest(c, "缺少必要参数")
		return
	}

	if err := h.runSvc.DeleteBuilderPod(c.Request.Context(), uint(clusterID), namespace, podName); err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.SuccessWithMessage(c, "删除成功", nil)
}

// BuilderConfigRequest 构建器配置请求
type BuilderConfigRequest struct {
	IdleTimeoutMinutes int    `json:"idle_timeout_minutes"`
	StorageType        string `json:"storage_type"`
	PVCName            string `json:"pvc_name"`
	PVCSizeGi          int    `json:"pvc_size_gi"`
	StorageClass       string `json:"storage_class"`
	AccessMode         string `json:"access_mode"`
	HostPath           string `json:"host_path"`
	CPURequest         string `json:"cpu_request"`
	CPULimit           string `json:"cpu_limit"`
	MemoryRequest      string `json:"memory_request"`
	MemoryLimit        string `json:"memory_limit"`
}

// SaveConfig 保存构建器配置
func (h *BuilderHandler) SaveConfig(c *gin.Context) {
	var req BuilderConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	// 设置空闲超时
	if req.IdleTimeoutMinutes > 0 {
		h.runSvc.SetBuilderIdleTimeout(time.Duration(req.IdleTimeoutMinutes) * time.Minute)
	}

	// 保存配置（包含 IdleTimeoutMinutes）
	h.runSvc.SetBuilderConfig(&pipeline.BuilderConfig{
		IdleTimeoutMinutes: req.IdleTimeoutMinutes,
		StorageType:        req.StorageType,
		PVCName:            req.PVCName,
		PVCSizeGi:          req.PVCSizeGi,
		StorageClass:       req.StorageClass,
		AccessMode:         req.AccessMode,
		HostPath:           req.HostPath,
		CPURequest:         req.CPURequest,
		CPULimit:           req.CPULimit,
		MemoryRequest:      req.MemoryRequest,
		MemoryLimit:        req.MemoryLimit,
	})

	response.SuccessWithMessage(c, "保存成功", nil)
}

// GetConfig 获取构建器配置
func (h *BuilderHandler) GetConfig(c *gin.Context) {
	config := h.runSvc.GetBuilderConfig()
	response.Success(c, config)
}
