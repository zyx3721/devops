package handler

import (
	"devops/internal/service/deploy"
	"devops/pkg/middleware"
	"devops/pkg/response"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type DeployLockHandler struct {
	service *deploy.LockService
}

func NewDeployLockHandler(service *deploy.LockService) *DeployLockHandler {
	return &DeployLockHandler{service: service}
}

// List 获取活跃的发布锁列表
// @Summary 获取发布锁列表
// @Tags 发布管理
// @Produce json
// @Success 200 {object} response.Response
// @Router /api/v1/deploy/locks [get]
func (h *DeployLockHandler) List(c *gin.Context) {
	locks, err := h.service.GetActiveLocks(c.Request.Context())
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "获取发布锁列表失败")
		return
	}

	response.Success(c, locks)
}

// ForceRelease 强制释放发布锁
// @Summary 强制释放发布锁
// @Tags 发布管理
// @Accept json
// @Produce json
// @Param id path int true "锁ID"
// @Param body body object true "释放原因"
// @Success 200 {object} response.Response
// @Router /api/v1/deploy/locks/{id}/release [post]
func (h *DeployLockHandler) ForceRelease(c *gin.Context) {
	appIDStr := c.Query("app_id")
	env := c.Query("env")

	if appIDStr == "" || env == "" {
		response.Error(c, http.StatusBadRequest, "缺少必要参数")
		return
	}

	appID, err := strconv.ParseUint(appIDStr, 10, 32)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "无效的应用ID")
		return
	}

	var req struct {
		Reason string `json:"reason" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "请填写释放原因")
		return
	}

	uid, ok := middleware.GetUserID(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "用户未登录")
		return
	}

	if err := h.service.ForceRelease(c.Request.Context(), uint(appID), env, uid, req.Reason); err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, nil)
}

// CheckLock 检查发布锁状态
// @Summary 检查发布锁状态
// @Tags 发布管理
// @Param app_id query int true "应用ID"
// @Param env query string true "环境"
// @Success 200 {object} response.Response
// @Router /api/v1/deploy/locks/check [get]
func (h *DeployLockHandler) CheckLock(c *gin.Context) {
	appIDStr := c.Query("app_id")
	env := c.Query("env")

	if appIDStr == "" || env == "" {
		response.Error(c, http.StatusBadRequest, "缺少必要参数")
		return
	}

	appID, err := strconv.ParseUint(appIDStr, 10, 32)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "无效的应用ID")
		return
	}

	locked, lock, err := h.service.IsLocked(c.Request.Context(), uint(appID), env)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "检查发布锁失败")
		return
	}

	response.Success(c, gin.H{
		"locked": locked,
		"lock":   lock,
	})
}
