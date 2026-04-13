package handler

import (
	"devops/internal/models"
	"devops/internal/service/approval"
	"devops/pkg/response"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type ApprovalRuleHandler struct {
	service *approval.RuleService
}

func NewApprovalRuleHandler(service *approval.RuleService) *ApprovalRuleHandler {
	return &ApprovalRuleHandler{service: service}
}

// List 获取审批规则列表
// @Summary 获取审批规则列表
// @Tags 审批管理
// @Produce json
// @Param app_id query int false "应用ID"
// @Success 200 {object} response.Response
// @Router /api/v1/approval/rules [get]
func (h *ApprovalRuleHandler) List(c *gin.Context) {
	var appID *uint
	if appIDStr := c.Query("app_id"); appIDStr != "" {
		id, err := strconv.ParseUint(appIDStr, 10, 32)
		if err == nil {
			appIDUint := uint(id)
			appID = &appIDUint
		}
	}

	rules, err := h.service.List(appID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "获取审批规则失败")
		return
	}

	response.Success(c, rules)
}

// Create 创建审批规则
// @Summary 创建审批规则
// @Tags 审批管理
// @Accept json
// @Produce json
// @Param rule body models.ApprovalRule true "审批规则"
// @Success 200 {object} response.Response
// @Router /api/v1/approval/rules [post]
func (h *ApprovalRuleHandler) Create(c *gin.Context) {
	var rule models.ApprovalRule
	if err := c.ShouldBindJSON(&rule); err != nil {
		response.Error(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}

	// 获取当前用户ID
	userID, _ := c.Get("user_id")
	if uid, ok := userID.(uint); ok {
		rule.CreatedBy = uid
	}

	if err := h.service.Create(&rule); err != nil {
		response.Error(c, http.StatusInternalServerError, "创建审批规则失败: "+err.Error())
		return
	}

	response.Success(c, rule)
}

// Update 更新审批规则
// @Summary 更新审批规则
// @Tags 审批管理
// @Accept json
// @Produce json
// @Param id path int true "规则ID"
// @Param rule body models.ApprovalRule true "审批规则"
// @Success 200 {object} response.Response
// @Router /api/v1/approval/rules/{id} [put]
func (h *ApprovalRuleHandler) Update(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "无效的ID")
		return
	}

	var rule models.ApprovalRule
	if err := c.ShouldBindJSON(&rule); err != nil {
		response.Error(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}

	rule.ID = uint(id)
	if err := h.service.Update(&rule); err != nil {
		response.Error(c, http.StatusInternalServerError, "更新审批规则失败: "+err.Error())
		return
	}

	response.Success(c, rule)
}

// Delete 删除审批规则
// @Summary 删除审批规则
// @Tags 审批管理
// @Param id path int true "规则ID"
// @Success 200 {object} response.Response
// @Router /api/v1/approval/rules/{id} [delete]
func (h *ApprovalRuleHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "无效的ID")
		return
	}

	if err := h.service.Delete(uint(id)); err != nil {
		response.Error(c, http.StatusInternalServerError, "删除审批规则失败: "+err.Error())
		return
	}

	response.Success(c, nil)
}

// GetByID 获取单个审批规则
// @Summary 获取单个审批规则
// @Tags 审批管理
// @Param id path int true "规则ID"
// @Success 200 {object} response.Response
// @Router /api/v1/approval/rules/{id} [get]
func (h *ApprovalRuleHandler) GetByID(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "无效的ID")
		return
	}

	rule, err := h.service.GetByID(uint(id))
	if err != nil {
		response.Error(c, http.StatusNotFound, "审批规则不存在")
		return
	}

	response.Success(c, rule)
}
