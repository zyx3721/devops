package handler

import (
	"net/http"
	"strconv"

	"devops/internal/models"
	"devops/pkg/dto"
	"devops/pkg/response"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// HighlightHandler 染色规则处理器
type HighlightHandler struct {
	db *gorm.DB
}

// NewHighlightHandler 创建染色规则处理器
func NewHighlightHandler(db *gorm.DB) *HighlightHandler {
	return &HighlightHandler{db: db}
}

// ListHighlightRules 获取染色规则列表
// @Summary 获取染色规则列表
// @Tags 日志中心
// @Param include_preset query bool false "是否包含预设规则"
// @Success 200 {object} response.Response{data=[]dto.HighlightRuleResponse}
// @Router /api/v1/logs/highlight-rules [get]
func (h *HighlightHandler) ListHighlightRules(c *gin.Context) {
	userID := c.GetInt64("user_id")
	includePreset := c.Query("include_preset") != "false"

	var rules []models.LogHighlightRule
	query := h.db.Where("user_id = ?", userID)
	if includePreset {
		query = query.Or("is_preset = ?", true)
	}
	query.Order("priority ASC").Find(&rules)

	var resp []dto.HighlightRuleResponse
	for _, rule := range rules {
		resp = append(resp, dto.HighlightRuleResponse{
			ID:         rule.ID,
			UserID:     rule.UserID,
			Name:       rule.Name,
			MatchType:  rule.MatchType,
			MatchValue: rule.MatchValue,
			FgColor:    rule.FgColor,
			BgColor:    rule.BgColor,
			Priority:   rule.Priority,
			Enabled:    rule.Enabled,
			IsPreset:   rule.IsPreset,
			CreatedAt:  rule.CreatedAt,
		})
	}

	response.Success(c, resp)
}

// CreateHighlightRule 创建染色规则
// @Summary 创建染色规则
// @Tags 日志中心
// @Accept json
// @Produce json
// @Param body body dto.HighlightRuleRequest true "规则信息"
// @Success 200 {object} response.Response{data=dto.HighlightRuleResponse}
// @Router /api/v1/logs/highlight-rules [post]
func (h *HighlightHandler) CreateHighlightRule(c *gin.Context) {
	var req dto.HighlightRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	userID := c.GetInt64("user_id")

	rule := models.LogHighlightRule{
		UserID:     userID,
		Name:       req.Name,
		MatchType:  req.MatchType,
		MatchValue: req.MatchValue,
		FgColor:    req.FgColor,
		BgColor:    req.BgColor,
		Priority:   req.Priority,
		Enabled:    req.Enabled,
		IsPreset:   false,
	}

	if err := h.db.Create(&rule).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, dto.HighlightRuleResponse{
		ID:         rule.ID,
		UserID:     rule.UserID,
		Name:       rule.Name,
		MatchType:  rule.MatchType,
		MatchValue: rule.MatchValue,
		FgColor:    rule.FgColor,
		BgColor:    rule.BgColor,
		Priority:   rule.Priority,
		Enabled:    rule.Enabled,
		IsPreset:   rule.IsPreset,
		CreatedAt:  rule.CreatedAt,
	})
}

// UpdateHighlightRule 更新染色规则
// @Summary 更新染色规则
// @Tags 日志中心
// @Accept json
// @Produce json
// @Param id path int true "规则ID"
// @Param body body dto.HighlightRuleRequest true "规则信息"
// @Success 200 {object} response.Response{data=dto.HighlightRuleResponse}
// @Router /api/v1/logs/highlight-rules/{id} [put]
func (h *HighlightHandler) UpdateHighlightRule(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	userID := c.GetInt64("user_id")

	var rule models.LogHighlightRule
	if err := h.db.First(&rule, id).Error; err != nil {
		response.Error(c, http.StatusNotFound, "规则不存在")
		return
	}

	// 不能修改预设规则
	if rule.IsPreset {
		response.Error(c, http.StatusForbidden, "不能修改预设规则")
		return
	}

	// 只能修改自己的规则
	if rule.UserID != userID {
		response.Error(c, http.StatusForbidden, "无权修改此规则")
		return
	}

	var req dto.HighlightRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	rule.Name = req.Name
	rule.MatchType = req.MatchType
	rule.MatchValue = req.MatchValue
	rule.FgColor = req.FgColor
	rule.BgColor = req.BgColor
	rule.Priority = req.Priority
	rule.Enabled = req.Enabled

	if err := h.db.Save(&rule).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, dto.HighlightRuleResponse{
		ID:         rule.ID,
		UserID:     rule.UserID,
		Name:       rule.Name,
		MatchType:  rule.MatchType,
		MatchValue: rule.MatchValue,
		FgColor:    rule.FgColor,
		BgColor:    rule.BgColor,
		Priority:   rule.Priority,
		Enabled:    rule.Enabled,
		IsPreset:   rule.IsPreset,
		CreatedAt:  rule.CreatedAt,
	})
}

// DeleteHighlightRule 删除染色规则
// @Summary 删除染色规则
// @Tags 日志中心
// @Param id path int true "规则ID"
// @Success 200 {object} response.Response
// @Router /api/v1/logs/highlight-rules/{id} [delete]
func (h *HighlightHandler) DeleteHighlightRule(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	userID := c.GetInt64("user_id")

	var rule models.LogHighlightRule
	if err := h.db.First(&rule, id).Error; err != nil {
		response.Error(c, http.StatusNotFound, "规则不存在")
		return
	}

	// 不能删除预设规则
	if rule.IsPreset {
		response.Error(c, http.StatusForbidden, "不能删除预设规则")
		return
	}

	// 只能删除自己的规则
	if rule.UserID != userID {
		response.Error(c, http.StatusForbidden, "无权删除此规则")
		return
	}

	if err := h.db.Delete(&rule).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, nil)
}

// ToggleHighlightRule 启用/禁用染色规则
// @Summary 启用/禁用染色规则
// @Tags 日志中心
// @Param id path int true "规则ID"
// @Success 200 {object} response.Response
// @Router /api/v1/logs/highlight-rules/{id}/toggle [post]
func (h *HighlightHandler) ToggleHighlightRule(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	userID := c.GetInt64("user_id")

	var rule models.LogHighlightRule
	if err := h.db.First(&rule, id).Error; err != nil {
		response.Error(c, http.StatusNotFound, "规则不存在")
		return
	}

	// 只能操作自己的规则或预设规则
	if rule.UserID != userID && !rule.IsPreset {
		response.Error(c, http.StatusForbidden, "无权操作此规则")
		return
	}

	rule.Enabled = !rule.Enabled
	if err := h.db.Save(&rule).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, gin.H{"enabled": rule.Enabled})
}

// GetPresetRules 获取预设规则
// @Summary 获取预设染色规则
// @Tags 日志中心
// @Success 200 {object} response.Response{data=[]dto.HighlightRuleResponse}
// @Router /api/v1/logs/highlight-rules/presets [get]
func (h *HighlightHandler) GetPresetRules(c *gin.Context) {
	var rules []models.LogHighlightRule
	h.db.Where("is_preset = ?", true).Order("priority ASC").Find(&rules)

	var resp []dto.HighlightRuleResponse
	for _, rule := range rules {
		resp = append(resp, dto.HighlightRuleResponse{
			ID:         rule.ID,
			UserID:     rule.UserID,
			Name:       rule.Name,
			MatchType:  rule.MatchType,
			MatchValue: rule.MatchValue,
			FgColor:    rule.FgColor,
			BgColor:    rule.BgColor,
			Priority:   rule.Priority,
			Enabled:    rule.Enabled,
			IsPreset:   rule.IsPreset,
			CreatedAt:  rule.CreatedAt,
		})
	}

	response.Success(c, resp)
}
