package handler

import (
	"devops/internal/models"
	"devops/internal/service/approval"
	"devops/pkg/response"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type DeployWindowHandler struct {
	service *approval.WindowService
}

func NewDeployWindowHandler(service *approval.WindowService) *DeployWindowHandler {
	return &DeployWindowHandler{service: service}
}

// List 获取发布窗口列表
// @Summary 获取发布窗口列表
// @Tags 审批管理
// @Produce json
// @Param app_id query int false "应用ID"
// @Success 200 {object} response.Response
// @Router /api/v1/approval/windows [get]
func (h *DeployWindowHandler) List(c *gin.Context) {
	var appID *uint
	if appIDStr := c.Query("app_id"); appIDStr != "" {
		id, err := strconv.ParseUint(appIDStr, 10, 32)
		if err == nil {
			appIDUint := uint(id)
			appID = &appIDUint
		}
	}

	windows, err := h.service.List(appID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "获取发布窗口失败")
		return
	}

	response.Success(c, windows)
}

// Create 创建发布窗口
// @Summary 创建发布窗口
// @Tags 审批管理
// @Accept json
// @Produce json
// @Param window body models.DeployWindow true "发布窗口"
// @Success 200 {object} response.Response
// @Router /api/v1/approval/windows [post]
func (h *DeployWindowHandler) Create(c *gin.Context) {
	var window models.DeployWindow
	if err := c.ShouldBindJSON(&window); err != nil {
		response.Error(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}

	// 获取当前用户ID
	userID, _ := c.Get("user_id")
	if uid, ok := userID.(uint); ok {
		window.CreatedBy = uid
	}

	if err := h.service.Create(&window); err != nil {
		response.Error(c, http.StatusInternalServerError, "创建发布窗口失败: "+err.Error())
		return
	}

	response.Success(c, window)
}

// Update 更新发布窗口
// @Summary 更新发布窗口
// @Tags 审批管理
// @Accept json
// @Produce json
// @Param id path int true "窗口ID"
// @Param window body models.DeployWindow true "发布窗口"
// @Success 200 {object} response.Response
// @Router /api/v1/approval/windows/{id} [put]
func (h *DeployWindowHandler) Update(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "无效的ID")
		return
	}

	var window models.DeployWindow
	if err := c.ShouldBindJSON(&window); err != nil {
		response.Error(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}

	window.ID = uint(id)
	if err := h.service.Update(&window); err != nil {
		response.Error(c, http.StatusInternalServerError, "更新发布窗口失败: "+err.Error())
		return
	}

	response.Success(c, window)
}

// Delete 删除发布窗口
// @Summary 删除发布窗口
// @Tags 审批管理
// @Param id path int true "窗口ID"
// @Success 200 {object} response.Response
// @Router /api/v1/approval/windows/{id} [delete]
func (h *DeployWindowHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "无效的ID")
		return
	}

	if err := h.service.Delete(uint(id)); err != nil {
		response.Error(c, http.StatusInternalServerError, "删除发布窗口失败: "+err.Error())
		return
	}

	response.Success(c, nil)
}

// GetByID 获取单个发布窗口
// @Summary 获取单个发布窗口
// @Tags 审批管理
// @Param id path int true "窗口ID"
// @Success 200 {object} response.Response
// @Router /api/v1/approval/windows/{id} [get]
func (h *DeployWindowHandler) GetByID(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "无效的ID")
		return
	}

	window, err := h.service.GetByID(uint(id))
	if err != nil {
		response.Error(c, http.StatusNotFound, "发布窗口不存在")
		return
	}

	response.Success(c, window)
}

// CheckWindow 检查当前是否在发布窗口内
// @Summary 检查发布窗口
// @Tags 审批管理
// @Param app_id query int true "应用ID"
// @Param env query string true "环境"
// @Success 200 {object} response.Response
// @Router /api/v1/approval/windows/check [get]
func (h *DeployWindowHandler) CheckWindow(c *gin.Context) {
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

	inWindow, allowEmergency, err := h.service.IsInWindow(uint(appID), env)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "检查发布窗口失败")
		return
	}

	response.Success(c, gin.H{
		"in_window":       inWindow,
		"allow_emergency": allowEmergency,
	})
}
