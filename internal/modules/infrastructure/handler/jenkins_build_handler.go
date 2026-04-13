package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"devops/internal/config"
	"devops/internal/service/jenkins"
	apperrors "devops/pkg/errors"
	"devops/pkg/ioc"
	"devops/pkg/middleware"
)

func init() {
	ioc.Api.RegisterContainer("JenkinsBuildHandler", &JenkinsBuildApiHandler{})
}

type JenkinsBuildApiHandler struct {
	handler *JenkinsBuildHandler
}

func (h *JenkinsBuildApiHandler) Init() error {
	cfg, _ := config.LoadConfig()
	_ = cfg
	client := jenkins.NewClient()
	h.handler = NewJenkinsBuildHandler(client)

	root := cfg.Application.GinRootRouter().Group("jenkins")
	root.Use(middleware.AuthMiddleware())
	h.Register(root)

	return nil
}

func (h *JenkinsBuildApiHandler) Register(r gin.IRouter) {
	r.POST("/build", h.handler.TriggerBuild)
	r.GET("/build/:queueId/status", h.handler.GetBuildStatus)
	r.GET("/job/:jobName/build/:buildNumber", h.handler.GetBuildInfo)
	r.GET("/jobs", h.handler.GetJobList)
}

type JenkinsBuildHandler struct {
	client *jenkins.Client
}

func NewJenkinsBuildHandler(client *jenkins.Client) *JenkinsBuildHandler {
	return &JenkinsBuildHandler{client: client}
}

type TriggerBuildRequest struct {
	JobName      string `json:"job_name" binding:"required"`
	Branch       string `json:"branch" binding:"required"`
	DeployType   string `json:"deploy_type"`
	ImageVersion string `json:"image_version"`
}

func (h *JenkinsBuildHandler) TriggerBuild(c *gin.Context) {
	var req TriggerBuildRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": apperrors.ErrCodeInvalidParams, "message": "参数错误", "error": err.Error()})
		return
	}

	buildReq := jenkins.BuildRequest{
		JobName:      req.JobName,
		Branch:       req.Branch,
		DeployType:   req.DeployType,
		ImageVersion: req.ImageVersion,
	}

	queueID, err := h.client.Build(c.Request.Context(), buildReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": apperrors.ErrCodeJenkinsBuild, "message": "触发构建失败", "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": apperrors.Success, "message": "构建已触发", "data": gin.H{"queue_id": queueID}})
}

func (h *JenkinsBuildHandler) GetBuildStatus(c *gin.Context) {
	queueIDStr := c.Param("queueId")
	queueID, err := strconv.ParseInt(queueIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": apperrors.ErrCodeInvalidParams, "message": "队列ID格式错误"})
		return
	}

	buildNumber, err := h.client.WaitForBuildToStart(c.Request.Context(), queueID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": apperrors.ErrCodeJenkinsBuild, "message": "获取构建状态失败", "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": apperrors.Success, "message": "success", "data": gin.H{"build_number": buildNumber}})
}

func (h *JenkinsBuildHandler) GetBuildInfo(c *gin.Context) {
	jobName := c.Param("jobName")
	buildNumberStr := c.Param("buildNumber")
	buildNumber, err := strconv.Atoi(buildNumberStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": apperrors.ErrCodeInvalidParams, "message": "构建号格式错误"})
		return
	}

	build, err := h.client.GetJobBuildInfo(c.Request.Context(), jobName, buildNumber)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": apperrors.ErrCodeJenkinsAPI, "message": "获取构建信息失败", "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    apperrors.Success,
		"message": "success",
		"data": gin.H{
			"number":    build.GetBuildNumber(),
			"result":    build.GetResult(),
			"building":  build.IsRunning(c.Request.Context()),
			"timestamp": build.GetTimestamp(),
			"duration":  build.GetDuration(),
			"url":       build.GetUrl(),
		},
	})
}

func (h *JenkinsBuildHandler) GetJobList(c *gin.Context) {
	jobs, err := h.client.GetAllJobs(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": apperrors.ErrCodeJenkinsAPI, "message": "获取Job列表失败", "error": err.Error()})
		return
	}

	var jobList []gin.H
	for _, job := range jobs {
		jobList = append(jobList, gin.H{
			"name":  job.GetName(),
			"url":   job.Raw.URL,
			"color": job.Raw.Color,
		})
	}

	c.JSON(http.StatusOK, gin.H{"code": apperrors.Success, "message": "success", "data": jobList})
}
