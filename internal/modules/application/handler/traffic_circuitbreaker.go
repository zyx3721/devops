// Package handler 应用模块处理器
// 本文件包含熔断规则相关的处理器方法
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

// ========== 熔断规则 ==========

// ListCircuitBreakers 获取熔断规则列表
// @Summary 获取应用的熔断规则列表
// @Tags 流量治理-熔断
// @Param id path int true "应用ID"
// @Success 200 {object} gin.H
// @Router /applications/{id}/traffic/circuitbreakers [get]
func (h *TrafficHandler) ListCircuitBreakers(c *gin.Context) {
	app, err := h.getApp(c)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "应用不存在"})
		return
	}

	var rules []models.TrafficCircuitBreakerRule
	h.db.Where("app_id = ?", app.ID).Find(&rules)

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"items": rules}})
}

// CreateCircuitBreaker 创建熔断规则
// @Summary 创建熔断规则
// @Tags 流量治理-熔断
// @Param id path int true "应用ID"
// @Param body body models.TrafficCircuitBreakerRule true "熔断规则"
// @Success 200 {object} gin.H
// @Router /applications/{id}/traffic/circuitbreakers [post]
func (h *TrafficHandler) CreateCircuitBreaker(c *gin.Context) {
	app, err := h.getApp(c)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "应用不存在"})
		return
	}

	var rule models.TrafficCircuitBreakerRule
	if err := c.ShouldBindJSON(&rule); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	rule.AppID = uint64(app.ID)
	rule.CircuitStatus = "closed"
	if err := h.db.Create(&rule).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	// 同步到 K8s DestinationRule
	var syncErr error
	var syncMsg string
	if rule.Enabled {
		syncErr = h.syncCircuitBreakerToK8s(app)
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

// UpdateCircuitBreaker 更新熔断规则
// @Summary 更新熔断规则
// @Tags 流量治理-熔断
// @Param id path int true "应用ID"
// @Param ruleId path int true "规则ID"
// @Param body body models.TrafficCircuitBreakerRule true "熔断规则"
// @Success 200 {object} gin.H
// @Router /applications/{id}/traffic/circuitbreakers/{ruleId} [put]
func (h *TrafficHandler) UpdateCircuitBreaker(c *gin.Context) {
	app, err := h.getApp(c)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "应用不存在"})
		return
	}

	ruleID, _ := strconv.ParseUint(c.Param("ruleId"), 10, 64)
	var rule models.TrafficCircuitBreakerRule
	if err := h.db.Where("id = ? AND app_id = ?", ruleID, app.ID).First(&rule).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "规则不存在"})
		return
	}

	var req models.TrafficCircuitBreakerRule
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	updates := map[string]any{
		"name":               req.Name,
		"resource":           req.Resource,
		"strategy":           req.Strategy,
		"slow_rt_threshold":  req.SlowRtThreshold,
		"threshold":          req.Threshold,
		"stat_interval":      req.StatInterval,
		"min_request_amount": req.MinRequestAmount,
		"recovery_timeout":   req.RecoveryTimeout,
		"probe_num":          req.ProbeNum,
		"fallback_strategy":  req.FallbackStrategy,
		"fallback_value":     req.FallbackValue,
		"fallback_service":   req.FallbackService,
		"enabled":            req.Enabled,
	}

	h.db.Model(&rule).Updates(updates)

	// 同步到 K8s
	h.syncCircuitBreakerToK8s(app)

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "更新成功"})
}

// DeleteCircuitBreaker 删除熔断规则
// @Summary 删除熔断规则
// @Tags 流量治理-熔断
// @Param id path int true "应用ID"
// @Param ruleId path int true "规则ID"
// @Success 200 {object} gin.H
// @Router /applications/{id}/traffic/circuitbreakers/{ruleId} [delete]
func (h *TrafficHandler) DeleteCircuitBreaker(c *gin.Context) {
	app, err := h.getApp(c)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "应用不存在"})
		return
	}

	ruleID, _ := strconv.ParseUint(c.Param("ruleId"), 10, 64)
	h.db.Where("id = ? AND app_id = ?", ruleID, app.ID).Delete(&models.TrafficCircuitBreakerRule{})

	// 重新同步 K8s
	h.syncCircuitBreakerToK8s(app)

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "删除成功"})
}

// syncCircuitBreakerToK8s 同步熔断配置到 K8s DestinationRule
// 将熔断规则转换为 Istio DestinationRule 的 outlierDetection 配置
func (h *TrafficHandler) syncCircuitBreakerToK8s(app *models.Application) error {
	if app.K8sClusterID == nil || app.K8sNamespace == "" {
		return nil
	}

	client, err := h.getDynamicClient(*app.K8sClusterID)
	if err != nil {
		return err
	}

	// 获取所有启用的熔断规则
	var rules []models.TrafficCircuitBreakerRule
	h.db.Where("app_id = ? AND enabled = ?", app.ID, true).Find(&rules)

	// 计算熔断参数
	consecutiveErrors := 5
	interval := "10s"
	baseEjectionTime := "30s"
	maxEjectionPercent := 100

	for _, rule := range rules {
		if rule.Strategy == "error_count" {
			consecutiveErrors = int(rule.Threshold)
		}
		interval = fmt.Sprintf("%ds", rule.StatInterval)
		baseEjectionTime = fmt.Sprintf("%ds", rule.RecoveryTimeout)
	}

	drName := fmt.Sprintf("%s-dr", app.Name)
	dr := &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "networking.istio.io/v1beta1",
			"kind":       "DestinationRule",
			"metadata": map[string]any{
				"name":      drName,
				"namespace": app.K8sNamespace,
				"labels": map[string]any{
					"app":        app.Name,
					"managed-by": "devops-platform",
				},
			},
			"spec": map[string]any{
				"host": app.K8sDeployment,
				"trafficPolicy": map[string]any{
					"outlierDetection": map[string]any{
						"consecutive5xxErrors": consecutiveErrors,
						"interval":             interval,
						"baseEjectionTime":     baseEjectionTime,
						"maxEjectionPercent":   maxEjectionPercent,
					},
				},
			},
		},
	}

	ctx := context.Background()
	_, err = client.Resource(destinationRuleGVR).Namespace(app.K8sNamespace).Create(ctx, dr, metav1.CreateOptions{})
	if err != nil {
		_, err = client.Resource(destinationRuleGVR).Namespace(app.K8sNamespace).Update(ctx, dr, metav1.UpdateOptions{})
	}
	return err
}

// GetCircuitBreakerConfig 获取熔断配置（配置型，兼容旧版前端）
// @Summary 获取熔断配置
// @Tags 流量治理-熔断
// @Param id path int true "应用ID"
// @Success 200 {object} gin.H
// @Router /applications/{id}/traffic/circuitbreaker [get]
func (h *TrafficHandler) GetCircuitBreakerConfig(c *gin.Context) {
	app, err := h.getApp(c)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "应用不存在"})
		return
	}

	// 构建熔断配置
	circuitBreakerConfig := gin.H{
		"enabled":              false,
		"consecutive_errors":   5,
		"interval":             "10s",
		"base_ejection_time":   "30s",
		"max_ejection_percent": 100,
	}

	// 构建连接池配置
	connectionPoolConfig := gin.H{
		"http2_max_requests":  1024,
		"tcp_max_connections": 1024,
		"connect_timeout":     "10s",
	}

	// 尝试从负载均衡配置中获取连接池配置
	var lbConfigs []models.TrafficLoadBalanceConfig
	h.db.Where("app_id = ?", app.ID).Find(&lbConfigs)
	if len(lbConfigs) > 0 {
		lbConfig := lbConfigs[0]
		connectionPoolConfig["http2_max_requests"] = lbConfig.HTTPMaxConnections
		connectionPoolConfig["tcp_max_connections"] = lbConfig.TCPMaxConnections
		connectionPoolConfig["connect_timeout"] = lbConfig.TCPConnectTimeout
	}

	// 尝试从熔断规则中获取配置
	var rules []models.TrafficCircuitBreakerRule
	h.db.Where("app_id = ? AND enabled = ?", app.ID, true).Find(&rules)
	if len(rules) > 0 {
		rule := rules[0]
		circuitBreakerConfig["enabled"] = true
		circuitBreakerConfig["consecutive_errors"] = int(rule.Threshold)
		circuitBreakerConfig["interval"] = fmt.Sprintf("%ds", rule.StatInterval)
		circuitBreakerConfig["base_ejection_time"] = fmt.Sprintf("%ds", rule.RecoveryTimeout)
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"circuit_breaker": circuitBreakerConfig,
			"connection_pool": connectionPoolConfig,
		},
	})
}

// UpdateCircuitBreakerConfig 更新熔断配置（配置型，兼容旧版前端）
// @Summary 更新熔断配置
// @Tags 流量治理-熔断
// @Param id path int true "应用ID"
// @Param body body map[string]any true "熔断配置"
// @Success 200 {object} gin.H
// @Router /applications/{id}/traffic/circuitbreaker [put]
func (h *TrafficHandler) UpdateCircuitBreakerConfig(c *gin.Context) {
	app, err := h.getApp(c)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "应用不存在"})
		return
	}

	var req map[string]any
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	// 更新负载均衡配置（连接池部分）
	var lbConfig models.TrafficLoadBalanceConfig
	var lbConfigs []models.TrafficLoadBalanceConfig
	h.db.Where("app_id = ?", app.ID).Limit(1).Find(&lbConfigs)

	isNewLbConfig := len(lbConfigs) == 0
	if !isNewLbConfig {
		lbConfig = lbConfigs[0]
	} else {
		// 创建新配置
		lbConfig.AppID = uint64(app.ID)
		lbConfig.LbPolicy = "round_robin"
	}

	if val, ok := req["http2_max_requests"].(float64); ok {
		lbConfig.HTTPMaxConnections = int(val)
	}
	if val, ok := req["tcp_max_connections"].(float64); ok {
		lbConfig.TCPMaxConnections = int(val)
	}
	if val, ok := req["connect_timeout"].(string); ok {
		lbConfig.TCPConnectTimeout = val
	}

	if isNewLbConfig {
		h.db.Create(&lbConfig)
	} else {
		h.db.Save(&lbConfig)
	}

	// 更新熔断规则
	enabled, _ := req["enabled"].(bool)
	if enabled {
		// 查找或创建熔断规则
		var rule models.TrafficCircuitBreakerRule
		var rules []models.TrafficCircuitBreakerRule
		h.db.Where("app_id = ?", app.ID).Limit(1).Find(&rules)

		isNewRule := len(rules) == 0
		if !isNewRule {
			rule = rules[0]
		} else {
			// 创建新规则
			rule.AppID = uint64(app.ID)
			rule.Name = "默认熔断规则"
			rule.Resource = "*"
			rule.Strategy = "error_count"
			rule.CircuitStatus = "closed"
		}

		if val, ok := req["consecutive_errors"].(float64); ok {
			rule.Threshold = val
		}
		if val, ok := req["interval"].(string); ok {
			// 解析 "10s" 格式
			var seconds int
			fmt.Sscanf(val, "%ds", &seconds)
			rule.StatInterval = seconds
		}
		if val, ok := req["base_ejection_time"].(string); ok {
			var seconds int
			fmt.Sscanf(val, "%ds", &seconds)
			rule.RecoveryTimeout = seconds
		}

		rule.Enabled = true

		if isNewRule {
			h.db.Create(&rule)
		} else {
			h.db.Save(&rule)
		}

		// 同步到 K8s
		h.syncCircuitBreakerToK8s(app)
	} else {
		// 禁用所有熔断规则
		h.db.Model(&models.TrafficCircuitBreakerRule{}).Where("app_id = ?", app.ID).Update("enabled", false)
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "保存成功"})
}
