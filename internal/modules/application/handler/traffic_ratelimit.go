// Package handler 应用模块处理器
// 本文件包含限流规则相关的处理器方法
package handler

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"devops/internal/models"
)

// ========== 限流规则 ==========

// ListRateLimits 获取限流规则列表
// @Summary 获取应用的限流规则列表
// @Tags 流量治理-限流
// @Param id path int true "应用ID"
// @Success 200 {object} gin.H
// @Router /applications/{id}/traffic/ratelimits [get]
func (h *TrafficHandler) ListRateLimits(c *gin.Context) {
	app, err := h.getApp(c)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "应用不存在"})
		return
	}

	var rules []models.TrafficRateLimitRule
	h.db.Where("app_id = ?", app.ID).Order("priority ASC").Find(&rules)

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"items": rules}})
}

// CreateRateLimit 创建限流规则
// @Summary 创建限流规则
// @Tags 流量治理-限流
// @Param id path int true "应用ID"
// @Param body body models.TrafficRateLimitRule true "限流规则"
// @Success 200 {object} gin.H
// @Router /applications/{id}/traffic/ratelimits [post]
func (h *TrafficHandler) CreateRateLimit(c *gin.Context) {
	app, err := h.getApp(c)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "应用不存在"})
		return
	}

	var rule models.TrafficRateLimitRule
	if err := c.ShouldBindJSON(&rule); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	rule.AppID = uint64(app.ID)
	if err := h.db.Create(&rule).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	// 同步到 K8s EnvoyFilter
	var syncErr error
	var syncMsg string
	if rule.Enabled {
		syncErr = h.syncRateLimitToK8s(app, &rule)
		if syncErr != nil {
			syncMsg = syncErr.Error()
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"code":       0,
		"message":    "创建成功",
		"data":       rule,
		"k8s_synced": syncErr == nil && rule.Enabled,
		"k8s_error":  syncMsg,
	})
}

// UpdateRateLimit 更新限流规则
// @Summary 更新限流规则
// @Tags 流量治理-限流
// @Param id path int true "应用ID"
// @Param ruleId path int true "规则ID"
// @Param body body models.TrafficRateLimitRule true "限流规则"
// @Success 200 {object} gin.H
// @Router /applications/{id}/traffic/ratelimits/{ruleId} [put]
func (h *TrafficHandler) UpdateRateLimit(c *gin.Context) {
	app, err := h.getApp(c)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "应用不存在"})
		return
	}

	ruleID, _ := strconv.ParseUint(c.Param("ruleId"), 10, 64)
	var rule models.TrafficRateLimitRule
	if err := h.db.Where("id = ? AND app_id = ?", ruleID, app.ID).First(&rule).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "规则不存在"})
		return
	}

	var req models.TrafficRateLimitRule
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	// 更新字段
	updates := map[string]any{
		"name":             req.Name,
		"description":      req.Description,
		"resource_type":    req.ResourceType,
		"resource":         req.Resource,
		"method":           req.Method,
		"strategy":         req.Strategy,
		"threshold":        req.Threshold,
		"burst":            req.Burst,
		"queue_size":       req.QueueSize,
		"control_behavior": req.ControlBehavior,
		"warm_up_period":   req.WarmUpPeriod,
		"max_queue_time":   req.MaxQueueTime,
		"limit_dimensions": req.LimitDimensions,
		"limit_header":     req.LimitHeader,
		"rejected_code":    req.RejectedCode,
		"rejected_message": req.RejectedMessage,
		"enabled":          req.Enabled,
		"priority":         req.Priority,
	}

	h.db.Model(&rule).Updates(updates)

	// 同步到 K8s
	if req.Enabled {
		h.syncRateLimitToK8s(app, &rule)
	} else {
		h.deleteRateLimitFromK8s(app, uint(ruleID))
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "更新成功"})
}

// DeleteRateLimit 删除限流规则
// @Summary 删除限流规则
// @Tags 流量治理-限流
// @Param id path int true "应用ID"
// @Param ruleId path int true "规则ID"
// @Success 200 {object} gin.H
// @Router /applications/{id}/traffic/ratelimits/{ruleId} [delete]
func (h *TrafficHandler) DeleteRateLimit(c *gin.Context) {
	app, err := h.getApp(c)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "应用不存在"})
		return
	}

	ruleID, _ := strconv.ParseUint(c.Param("ruleId"), 10, 64)
	h.db.Where("id = ? AND app_id = ?", ruleID, app.ID).Delete(&models.TrafficRateLimitRule{})

	// 从 K8s 删除
	h.deleteRateLimitFromK8s(app, uint(ruleID))

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "删除成功"})
}

// syncRateLimitToK8s 同步限流规则到 K8s EnvoyFilter
// 将限流规则转换为 Istio EnvoyFilter 资源并应用到集群
func (h *TrafficHandler) syncRateLimitToK8s(app *models.Application, rule *models.TrafficRateLimitRule) error {
	if app.K8sClusterID == nil || app.K8sNamespace == "" {
		return nil
	}

	client, err := h.getDynamicClient(*app.K8sClusterID)
	if err != nil {
		return err
	}

	filterName := fmt.Sprintf("%s-ratelimit-%d", app.Name, rule.ID)
	envoyFilter := &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "networking.istio.io/v1alpha3",
			"kind":       "EnvoyFilter",
			"metadata": map[string]any{
				"name":      filterName,
				"namespace": app.K8sNamespace,
				"labels": map[string]any{
					"app":        app.Name,
					"managed-by": "devops-platform",
				},
			},
			"spec": map[string]any{
				"workloadSelector": map[string]any{
					"labels": map[string]any{
						"app": app.K8sDeployment,
					},
				},
				"configPatches": []any{
					map[string]any{
						"applyTo": "HTTP_FILTER",
						"match": map[string]any{
							"context": "SIDECAR_INBOUND",
							"listener": map[string]any{
								"filterChain": map[string]any{
									"filter": map[string]any{
										"name": "envoy.filters.network.http_connection_manager",
									},
								},
							},
						},
						"patch": map[string]any{
							"operation": "INSERT_BEFORE",
							"value": map[string]any{
								"name": "envoy.filters.http.local_ratelimit",
								"typed_config": map[string]any{
									"@type":       "type.googleapis.com/udpa.type.v1.TypedStruct",
									"type_url":    "type.googleapis.com/envoy.extensions.filters.http.local_ratelimit.v3.LocalRateLimit",
									"stat_prefix": "http_local_rate_limiter",
									"token_bucket": map[string]any{
										"max_tokens":      rule.Threshold,
										"tokens_per_fill": rule.Threshold,
										"fill_interval":   "1s",
									},
								},
							},
						},
					},
				},
			},
		},
	}

	ctx := context.Background()
	_, err = client.Resource(envoyFilterGVR).Namespace(app.K8sNamespace).Create(ctx, envoyFilter, metav1.CreateOptions{})
	if err != nil {
		_, err = client.Resource(envoyFilterGVR).Namespace(app.K8sNamespace).Update(ctx, envoyFilter, metav1.UpdateOptions{})
	}
	return err
}

// deleteRateLimitFromK8s 从 K8s 删除限流规则
func (h *TrafficHandler) deleteRateLimitFromK8s(app *models.Application, ruleID uint) error {
	if app.K8sClusterID == nil || app.K8sNamespace == "" {
		return nil
	}

	client, err := h.getDynamicClient(*app.K8sClusterID)
	if err != nil {
		return err
	}

	filterName := fmt.Sprintf("%s-ratelimit-%d", app.Name, ruleID)
	return client.Resource(envoyFilterGVR).Namespace(app.K8sNamespace).Delete(
		context.Background(), filterName, metav1.DeleteOptions{},
	)
}
