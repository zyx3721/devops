package handler

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"devops/internal/config"
	"devops/internal/service/jenkins"
	"devops/pkg/dto"
	apperrors "devops/pkg/errors"
	"devops/pkg/ioc"
	"devops/pkg/middleware"
)

func init() {
	ioc.Api.RegisterContainer("JenkinsInstanceHandler", &JenkinsInstanceApiHandler{})
}

type JenkinsInstanceApiHandler struct {
	handler *JenkinsInstanceHandler
}

func (h *JenkinsInstanceApiHandler) Init() error {
	cfg, _ := config.LoadConfig()
	db := cfg.GetDB()
	svc := jenkins.NewJenkinsInstanceService(db)
	jobSvc := jenkins.NewJobService(db)
	h.handler = NewJenkinsInstanceHandler(svc, jobSvc, db)

	root := cfg.Application.GinRootRouter().Group("jenkins-instances")
	root.Use(middleware.AuthMiddleware())
	h.Register(root)

	return nil
}

func (h *JenkinsInstanceApiHandler) Register(r gin.IRouter) {
	// 查看权限
	r.GET("", h.handler.GetJenkinsInstanceList)
	r.GET("/default", h.handler.GetDefaultJenkinsInstance)
	r.GET("/:id", h.handler.GetJenkinsInstance)
	r.GET("/:id/feishu-apps", h.handler.GetFeishuApps)
	r.GET("/:id/jobs", h.handler.GetJobs)
	r.GET("/:id/jobs/:jobName/builds", h.handler.GetJobBuilds)

	// 管理权限
	r.POST("", middleware.RequireAdmin(), h.handler.CreateJenkinsInstance)
	r.PUT("/:id", middleware.RequireAdmin(), h.handler.UpdateJenkinsInstance)
	r.PUT("/:id/default", middleware.RequireAdmin(), h.handler.SetDefaultJenkinsInstance)
	r.DELETE("/:id", middleware.RequireAdmin(), h.handler.DeleteJenkinsInstance)
	r.POST("/:id/test-connection", middleware.RequireAdmin(), h.handler.TestConnection)
	r.PUT("/:id/feishu-apps", middleware.RequireAdmin(), h.handler.BindFeishuApps)
}

type JenkinsInstanceHandler struct {
	svc    jenkins.JenkinsInstanceService
	jobSvc jenkins.JobService
	db     *gorm.DB
}

func NewJenkinsInstanceHandler(svc jenkins.JenkinsInstanceService, jobSvc jenkins.JobService, db *gorm.DB) *JenkinsInstanceHandler {
	return &JenkinsInstanceHandler{svc: svc, jobSvc: jobSvc, db: db}
}

func (h *JenkinsInstanceHandler) CreateJenkinsInstance(c *gin.Context) {
	var req dto.CreateJenkinsInstanceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": apperrors.ErrCodeInvalidParams, "message": "参数错误", "error": err.Error()})
		return
	}

	result, err := h.svc.CreateJenkinsInstance(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": apperrors.ErrCodeInternalError, "message": "创建Jenkins实例失败", "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": apperrors.Success, "message": "创建成功", "data": result})
}

func (h *JenkinsInstanceHandler) GetJenkinsInstance(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": apperrors.ErrCodeInvalidParams, "message": "ID格式错误"})
		return
	}

	result, err := h.svc.GetJenkinsInstance(c.Request.Context(), uint(id))
	if err != nil {
		if err == apperrors.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{"code": apperrors.ErrCodeNotFound, "message": "Jenkins实例不存在"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"code": apperrors.ErrCodeInternalError, "message": "查询Jenkins实例失败", "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": apperrors.Success, "message": "success", "data": result})
}

func (h *JenkinsInstanceHandler) GetJenkinsInstanceList(c *gin.Context) {
	var req dto.JenkinsInstanceListRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": apperrors.ErrCodeInvalidParams, "message": "参数错误", "error": err.Error()})
		return
	}

	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 20
	}

	result, err := h.svc.GetJenkinsInstanceList(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": apperrors.ErrCodeInternalError, "message": "查询Jenkins实例列表失败", "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": apperrors.Success, "message": "success", "data": result})
}

func (h *JenkinsInstanceHandler) UpdateJenkinsInstance(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": apperrors.ErrCodeInvalidParams, "message": "ID格式错误"})
		return
	}

	var req dto.UpdateJenkinsInstanceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": apperrors.ErrCodeInvalidParams, "message": "参数错误", "error": err.Error()})
		return
	}

	result, err := h.svc.UpdateJenkinsInstance(c.Request.Context(), uint(id), &req)
	if err != nil {
		if err == apperrors.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{"code": apperrors.ErrCodeNotFound, "message": "Jenkins实例不存在"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"code": apperrors.ErrCodeInternalError, "message": "更新Jenkins实例失败", "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": apperrors.Success, "message": "更新成功", "data": result})
}

func (h *JenkinsInstanceHandler) DeleteJenkinsInstance(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": apperrors.ErrCodeInvalidParams, "message": "ID格式错误"})
		return
	}

	if err := h.svc.DeleteJenkinsInstance(c.Request.Context(), uint(id)); err != nil {
		if err == apperrors.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{"code": apperrors.ErrCodeNotFound, "message": "Jenkins实例不存在"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"code": apperrors.ErrCodeInternalError, "message": "删除Jenkins实例失败", "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": apperrors.Success, "message": "删除成功"})
}

func (h *JenkinsInstanceHandler) SetDefaultJenkinsInstance(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": apperrors.ErrCodeInvalidParams, "message": "ID格式错误"})
		return
	}

	if err := h.svc.SetDefaultJenkinsInstance(c.Request.Context(), uint(id)); err != nil {
		if err == apperrors.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{"code": apperrors.ErrCodeNotFound, "message": "Jenkins实例不存在"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"code": apperrors.ErrCodeInternalError, "message": "设置默认Jenkins实例失败", "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": apperrors.Success, "message": "设置成功"})
}

func (h *JenkinsInstanceHandler) GetDefaultJenkinsInstance(c *gin.Context) {
	result, err := h.svc.GetDefaultJenkinsInstance(c.Request.Context())
	if err != nil {
		if err == apperrors.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{"code": apperrors.ErrCodeNotFound, "message": "没有可用的Jenkins实例"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"code": apperrors.ErrCodeInternalError, "message": "查询默认Jenkins实例失败", "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": apperrors.Success, "message": "success", "data": result})
}

func (h *JenkinsInstanceHandler) GetFeishuApps(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": apperrors.ErrCodeInvalidParams, "message": "ID格式错误"})
		return
	}

	result, err := h.svc.GetFeishuApps(c.Request.Context(), uint(id))
	if err != nil {
		if err == apperrors.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{"code": apperrors.ErrCodeNotFound, "message": "Jenkins实例不存在"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"code": apperrors.ErrCodeInternalError, "message": "查询绑定的飞书应用失败", "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": apperrors.Success, "message": "success", "data": result})
}

func (h *JenkinsInstanceHandler) BindFeishuApps(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": apperrors.ErrCodeInvalidParams, "message": "ID格式错误"})
		return
	}

	var req dto.BindFeishuAppsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": apperrors.ErrCodeInvalidParams, "message": "参数错误", "error": err.Error()})
		return
	}

	if err := h.svc.BindFeishuApps(c.Request.Context(), uint(id), req.AppIDs); err != nil {
		if err == apperrors.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{"code": apperrors.ErrCodeNotFound, "message": "Jenkins实例不存在"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"code": apperrors.ErrCodeInternalError, "message": "绑定飞书应用失败", "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": apperrors.Success, "message": "绑定成功"})
}

func (h *JenkinsInstanceHandler) GetJobs(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": apperrors.ErrCodeInvalidParams, "message": "ID格式错误"})
		return
	}

	result, err := h.jobSvc.GetJobs(c.Request.Context(), uint(id))
	if err != nil {
		if err == apperrors.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{"code": apperrors.ErrCodeNotFound, "message": "Jenkins实例不存在"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"code": apperrors.ErrCodeInternalError, "message": "获取Job列表失败", "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": apperrors.Success, "message": "success", "data": result})
}

func (h *JenkinsInstanceHandler) GetJobBuilds(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": apperrors.ErrCodeInvalidParams, "message": "ID格式错误"})
		return
	}

	jobName := c.Param("jobName")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	result, err := h.jobSvc.GetJobBuilds(c.Request.Context(), uint(id), jobName, limit)
	if err != nil {
		if err == apperrors.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{"code": apperrors.ErrCodeNotFound, "message": "Jenkins实例不存在"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"code": apperrors.ErrCodeInternalError, "message": "获取构建列表失败", "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": apperrors.Success, "message": "success", "data": result})
}

func (h *JenkinsInstanceHandler) TestConnection(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": apperrors.ErrCodeInvalidParams, "message": "ID格式错误"})
		return
	}

	// 捕获可能的 panic
	defer func() {
		if r := recover(); r != nil {
			c.JSON(http.StatusOK, gin.H{
				"code":    apperrors.Success,
				"message": "success",
				"data": map[string]interface{}{
					"connected": false,
					"error":     fmt.Sprintf("连接异常: %v", r),
				},
			})
		}
	}()

	result, err := h.svc.TestConnection(c.Request.Context(), uint(id))
	if err != nil {
		if err == apperrors.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{"code": apperrors.ErrCodeNotFound, "message": "Jenkins实例不存在"})
			return
		}
		// 返回 200 但标记连接失败，显示具体错误
		c.JSON(http.StatusOK, gin.H{
			"code":    apperrors.Success,
			"message": "success",
			"data": map[string]interface{}{
				"connected": false,
				"error":     err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": apperrors.Success, "message": "success", "data": result})
}
