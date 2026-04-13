// Package handler 应用模块处理器
// 本文件包含流量镜像相关的处理器方法
package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"devops/internal/models"
)

// ========== 流量镜像 ==========

// ListMirrors 获取流量镜像列表
// @Summary 获取应用的流量镜像规则列表
// @Tags 流量治理-镜像
// @Param id path int true "应用ID"
// @Success 200 {object} gin.H
// @Router /applications/{id}/traffic/mirrors [get]
func (h *TrafficHandler) ListMirrors(c *gin.Context) {
	app, err := h.getApp(c)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "应用不存在"})
		return
	}

	var rules []models.TrafficMirrorRule
	h.db.Where("app_id = ?", app.ID).Find(&rules)

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"items": rules}})
}

// CreateMirror 创建流量镜像
// @Summary 创建流量镜像规则
// @Tags 流量治理-镜像
// @Param id path int true "应用ID"
// @Param body body models.TrafficMirrorRule true "镜像规则"
// @Success 200 {object} gin.H
// @Router /applications/{id}/traffic/mirrors [post]
func (h *TrafficHandler) CreateMirror(c *gin.Context) {
	app, err := h.getApp(c)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "应用不存在"})
		return
	}

	var rule models.TrafficMirrorRule
	if err := c.ShouldBindJSON(&rule); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	rule.AppID = uint64(app.ID)
	if err := h.db.Create(&rule).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "创建成功", "data": rule})
}

// UpdateMirror 更新流量镜像
// @Summary 更新流量镜像规则
// @Tags 流量治理-镜像
// @Param id path int true "应用ID"
// @Param ruleId path int true "规则ID"
// @Param body body models.TrafficMirrorRule true "镜像规则"
// @Success 200 {object} gin.H
// @Router /applications/{id}/traffic/mirrors/{ruleId} [put]
func (h *TrafficHandler) UpdateMirror(c *gin.Context) {
	app, err := h.getApp(c)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "应用不存在"})
		return
	}

	ruleID, _ := strconv.ParseUint(c.Param("ruleId"), 10, 64)
	var req models.TrafficMirrorRule
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	h.db.Model(&models.TrafficMirrorRule{}).Where("id = ? AND app_id = ?", ruleID, app.ID).Updates(map[string]any{
		"target_service": req.TargetService,
		"target_subset":  req.TargetSubset,
		"percentage":     req.Percentage,
		"enabled":        req.Enabled,
	})

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "更新成功"})
}

// DeleteMirror 删除流量镜像
// @Summary 删除流量镜像规则
// @Tags 流量治理-镜像
// @Param id path int true "应用ID"
// @Param ruleId path int true "规则ID"
// @Success 200 {object} gin.H
// @Router /applications/{id}/traffic/mirrors/{ruleId} [delete]
func (h *TrafficHandler) DeleteMirror(c *gin.Context) {
	app, err := h.getApp(c)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "应用不存在"})
		return
	}

	ruleID, _ := strconv.ParseUint(c.Param("ruleId"), 10, 64)
	h.db.Where("id = ? AND app_id = ?", ruleID, app.ID).Delete(&models.TrafficMirrorRule{})

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "删除成功"})
}
