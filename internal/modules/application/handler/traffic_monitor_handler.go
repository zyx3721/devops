package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"devops/internal/models"
	"devops/internal/service/kubernetes"
)

// TrafficMonitorHandler 流量监控处理器
type TrafficMonitorHandler struct {
	db           *gorm.DB
	istioService *kubernetes.IstioService
}

// NewTrafficMonitorHandler 创建流量监控处理器
func NewTrafficMonitorHandler(db *gorm.DB, istioService *kubernetes.IstioService) *TrafficMonitorHandler {
	return &TrafficMonitorHandler{db: db, istioService: istioService}
}

// RegisterRoutes 注册路由
func (h *TrafficMonitorHandler) RegisterRoutes(r *gin.RouterGroup) {
	g := r.Group("/applications/:id/traffic")
	{
		// 流量统计
		g.GET("/stats", h.GetTrafficStats)
		g.GET("/stats/history", h.GetTrafficStatsHistory)

		// 熔断状态
		g.GET("/circuitbreaker/status", h.GetCircuitBreakerStatus)

		// 规则版本管理
		g.GET("/versions", h.ListRuleVersions)
		g.POST("/versions/:versionId/rollback", h.RollbackRule)

		// Istio 资源查看
		g.GET("/istio/virtualservices", h.ListVirtualServices)
		g.GET("/istio/destinationrules", h.ListDestinationRules)
		g.GET("/istio/gateways", h.ListGateways)
	}
}

// getApp 获取应用信息
func (h *TrafficMonitorHandler) getApp(c *gin.Context) (*models.Application, error) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	var app models.Application
	if err := h.db.First(&app, id).Error; err != nil {
		return nil, err
	}
	return &app, nil
}

// GetTrafficStats 获取流量统计
func (h *TrafficMonitorHandler) GetTrafficStats(c *gin.Context) {
	app, err := h.getApp(c)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "应用不存在"})
		return
	}

	stats, err := h.istioService.GetTrafficStats(c.Request.Context(), uint64(app.ID), app.K8sNamespace, app.Name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": stats})
}

// GetTrafficStatsHistory 获取流量统计历史
func (h *TrafficMonitorHandler) GetTrafficStatsHistory(c *gin.Context) {
	app, err := h.getApp(c)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "应用不存在"})
		return
	}

	// 获取时间范围
	hours, _ := strconv.Atoi(c.DefaultQuery("hours", "24"))
	if hours <= 0 || hours > 168 {
		hours = 24
	}

	startTime := time.Now().Add(-time.Duration(hours) * time.Hour)

	var stats []models.TrafficStatistics
	h.db.Where("app_id = ? AND timestamp >= ?", app.ID, startTime).
		Order("timestamp ASC").Find(&stats)

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"items": stats}})
}

// GetCircuitBreakerStatus 获取熔断状态
func (h *TrafficMonitorHandler) GetCircuitBreakerStatus(c *gin.Context) {
	app, err := h.getApp(c)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "应用不存在"})
		return
	}

	status, err := h.istioService.GetCircuitBreakerStatus(uint64(app.ID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"items": status}})
}

// ListRuleVersions 获取规则版本列表
func (h *TrafficMonitorHandler) ListRuleVersions(c *gin.Context) {
	app, err := h.getApp(c)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "应用不存在"})
		return
	}

	ruleType := c.Query("rule_type")
	ruleID, _ := strconv.ParseUint(c.Query("rule_id"), 10, 64)

	query := h.db.Where("app_id = ?", app.ID)
	if ruleType != "" {
		query = query.Where("rule_type = ?", ruleType)
	}
	if ruleID > 0 {
		query = query.Where("rule_id = ?", ruleID)
	}

	var versions []models.TrafficRuleVersion
	query.Order("created_at DESC").Limit(50).Find(&versions)

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"items": versions}})
}

// RollbackRule 回滚规则到指定版本
// 支持回滚的规则类型: ratelimit, circuitbreaker, routing, loadbalance, timeout, mirror, fault
func (h *TrafficMonitorHandler) RollbackRule(c *gin.Context) {
	app, err := h.getApp(c)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "应用不存在"})
		return
	}

	versionID, _ := strconv.ParseUint(c.Param("versionId"), 10, 64)

	var version models.TrafficRuleVersion
	if err := h.db.Where("id = ? AND app_id = ?", versionID, app.ID).First(&version).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "版本不存在"})
		return
	}

	// 获取操作人信息
	operator := c.GetString("username")
	if operator == "" {
		operator = "system"
	}

	// 根据规则类型执行回滚
	if err := h.executeRollback(c.Request.Context(), app, &version, operator); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "回滚失败: " + err.Error()})
		return
	}

	// 记录操作日志
	h.db.Create(&models.TrafficOperationLog{
		AppID:     uint64(app.ID),
		RuleType:  version.RuleType,
		RuleID:    version.RuleID,
		Operation: "rollback",
		Operator:  operator,
		NewValue:  "回滚到版本 " + strconv.FormatInt(int64(version.Version), 10),
	})

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "回滚成功"})
}

// executeRollback 执行规则回滚
func (h *TrafficMonitorHandler) executeRollback(ctx context.Context, app *models.Application, version *models.TrafficRuleVersion, _ string) error {
	switch version.RuleType {
	case "ratelimit":
		return h.rollbackRateLimitRule(app, version)
	case "circuitbreaker":
		return h.rollbackCircuitBreakerRule(ctx, app, version)
	case "routing":
		return h.rollbackRoutingRule(ctx, app, version)
	case "loadbalance":
		return h.rollbackLoadBalanceConfig(ctx, app, version)
	case "timeout":
		return h.rollbackTimeoutConfig(ctx, app, version)
	case "mirror":
		return h.rollbackMirrorRule(ctx, app, version)
	case "fault":
		return h.rollbackFaultRule(ctx, app, version)
	default:
		return fmt.Errorf("不支持的规则类型: %s", version.RuleType)
	}
}

// rollbackRateLimitRule 回滚限流规则
func (h *TrafficMonitorHandler) rollbackRateLimitRule(app *models.Application, version *models.TrafficRuleVersion) error {
	_ = app // 限流规则不需要同步到 Istio
	var rule models.TrafficRateLimitRule
	if err := json.Unmarshal([]byte(version.Content), &rule); err != nil {
		return fmt.Errorf("解析规则内容失败: %w", err)
	}

	// 更新数据库中的规则
	return h.db.Model(&models.TrafficRateLimitRule{}).
		Where("id = ?", version.RuleID).
		Updates(map[string]any{
			"name":             rule.Name,
			"resource":         rule.Resource,
			"strategy":         rule.Strategy,
			"threshold":        rule.Threshold,
			"burst":            rule.Burst,
			"control_behavior": rule.ControlBehavior,
			"enabled":          rule.Enabled,
		}).Error
}

// rollbackCircuitBreakerRule 回滚熔断规则
func (h *TrafficMonitorHandler) rollbackCircuitBreakerRule(ctx context.Context, app *models.Application, version *models.TrafficRuleVersion) error {
	var rule models.TrafficCircuitBreakerRule
	if err := json.Unmarshal([]byte(version.Content), &rule); err != nil {
		return fmt.Errorf("解析规则内容失败: %w", err)
	}

	// 更新数据库中的规则
	if err := h.db.Model(&models.TrafficCircuitBreakerRule{}).
		Where("id = ?", version.RuleID).
		Updates(map[string]any{
			"name":               rule.Name,
			"resource":           rule.Resource,
			"strategy":           rule.Strategy,
			"threshold":          rule.Threshold,
			"stat_interval":      rule.StatInterval,
			"recovery_timeout":   rule.RecoveryTimeout,
			"min_request_amount": rule.MinRequestAmount,
			"slow_rt_threshold":  rule.SlowRtThreshold,
			"enabled":            rule.Enabled,
		}).Error; err != nil {
		return err
	}

	// 同步到 Istio
	if app.K8sClusterID != nil && app.K8sNamespace != "" {
		var cbRules []models.TrafficCircuitBreakerRule
		h.db.Where("app_id = ? AND enabled = ?", app.ID, true).Find(&cbRules)
		return h.istioService.SyncDestinationRule(ctx, *app.K8sClusterID, app.K8sNamespace, app, nil, cbRules)
	}

	return nil
}

// rollbackRoutingRule 回滚路由规则
func (h *TrafficMonitorHandler) rollbackRoutingRule(ctx context.Context, app *models.Application, version *models.TrafficRuleVersion) error {
	var rule models.TrafficRoutingRule
	if err := json.Unmarshal([]byte(version.Content), &rule); err != nil {
		return fmt.Errorf("解析规则内容失败: %w", err)
	}

	// 更新数据库中的规则
	if err := h.db.Model(&models.TrafficRoutingRule{}).
		Where("id = ?", version.RuleID).
		Updates(map[string]any{
			"name":           rule.Name,
			"route_type":     rule.RouteType,
			"match_key":      rule.MatchKey,
			"match_operator": rule.MatchOperator,
			"match_value":    rule.MatchValue,
			"target_subset":  rule.TargetSubset,
			"priority":       rule.Priority,
			"enabled":        rule.Enabled,
		}).Error; err != nil {
		return err
	}

	// 同步到 Istio
	if app.K8sClusterID != nil && app.K8sNamespace != "" {
		var rules []models.TrafficRoutingRule
		h.db.Where("app_id = ? AND enabled = ?", app.ID, true).Order("priority DESC").Find(&rules)
		return h.istioService.SyncRoutingRules(ctx, *app.K8sClusterID, app.K8sNamespace, app, rules)
	}

	return nil
}

// rollbackLoadBalanceConfig 回滚负载均衡配置
func (h *TrafficMonitorHandler) rollbackLoadBalanceConfig(ctx context.Context, app *models.Application, version *models.TrafficRuleVersion) error {
	var config models.TrafficLoadBalanceConfig
	if err := json.Unmarshal([]byte(version.Content), &config); err != nil {
		return fmt.Errorf("解析配置内容失败: %w", err)
	}

	// 更新数据库中的配置
	if err := h.db.Model(&models.TrafficLoadBalanceConfig{}).
		Where("id = ?", version.RuleID).
		Updates(map[string]any{
			"lb_policy":                  config.LbPolicy,
			"hash_key":                   config.HashKey,
			"hash_key_name":              config.HashKeyName,
			"ring_size":                  config.RingSize,
			"http_max_connections":       config.HTTPMaxConnections,
			"http_max_pending_requests":  config.HTTPMaxPendingRequests,
			"http_max_requests_per_conn": config.HTTPMaxRequestsPerConn,
			"http_max_retries":           config.HTTPMaxRetries,
			"http_idle_timeout":          config.HTTPIdleTimeout,
			"tcp_max_connections":        config.TCPMaxConnections,
			"tcp_connect_timeout":        config.TCPConnectTimeout,
		}).Error; err != nil {
		return err
	}

	// 同步到 Istio
	if app.K8sClusterID != nil && app.K8sNamespace != "" {
		var lbConfig models.TrafficLoadBalanceConfig
		h.db.Where("app_id = ?", app.ID).First(&lbConfig)
		return h.istioService.SyncDestinationRule(ctx, *app.K8sClusterID, app.K8sNamespace, app, &lbConfig, nil)
	}

	return nil
}

// rollbackTimeoutConfig 回滚超时配置
func (h *TrafficMonitorHandler) rollbackTimeoutConfig(ctx context.Context, app *models.Application, version *models.TrafficRuleVersion) error {
	var config models.TrafficTimeoutConfig
	if err := json.Unmarshal([]byte(version.Content), &config); err != nil {
		return fmt.Errorf("解析配置内容失败: %w", err)
	}

	// 更新数据库中的配置
	if err := h.db.Model(&models.TrafficTimeoutConfig{}).
		Where("id = ?", version.RuleID).
		Updates(map[string]any{
			"timeout":         config.Timeout,
			"retries":         config.Retries,
			"per_try_timeout": config.PerTryTimeout,
			"retry_on":        config.RetryOn,
		}).Error; err != nil {
		return err
	}

	// 同步到 Istio
	if app.K8sClusterID != nil && app.K8sNamespace != "" {
		return h.istioService.SyncTimeoutRetry(ctx, *app.K8sClusterID, app.K8sNamespace, app, &config)
	}

	return nil
}

// rollbackMirrorRule 回滚流量镜像规则
func (h *TrafficMonitorHandler) rollbackMirrorRule(ctx context.Context, app *models.Application, version *models.TrafficRuleVersion) error {
	var rule models.TrafficMirrorRule
	if err := json.Unmarshal([]byte(version.Content), &rule); err != nil {
		return fmt.Errorf("解析规则内容失败: %w", err)
	}

	// 更新数据库中的规则
	if err := h.db.Model(&models.TrafficMirrorRule{}).
		Where("id = ?", version.RuleID).
		Updates(map[string]any{
			"target_service": rule.TargetService,
			"target_subset":  rule.TargetSubset,
			"percentage":     rule.Percentage,
			"enabled":        rule.Enabled,
		}).Error; err != nil {
		return err
	}

	// 同步到 Istio
	if app.K8sClusterID != nil && app.K8sNamespace != "" {
		return h.istioService.SyncMirrorRule(ctx, *app.K8sClusterID, app.K8sNamespace, app, &rule)
	}

	return nil
}

// rollbackFaultRule 回滚故障注入规则
func (h *TrafficMonitorHandler) rollbackFaultRule(ctx context.Context, app *models.Application, version *models.TrafficRuleVersion) error {
	var rule models.TrafficFaultRule
	if err := json.Unmarshal([]byte(version.Content), &rule); err != nil {
		return fmt.Errorf("解析规则内容失败: %w", err)
	}

	// 更新数据库中的规则
	if err := h.db.Model(&models.TrafficFaultRule{}).
		Where("id = ?", version.RuleID).
		Updates(map[string]any{
			"type":           rule.Type,
			"percentage":     rule.Percentage,
			"delay_duration": rule.DelayDuration,
			"abort_code":     rule.AbortCode,
			"path":           rule.Path,
			"enabled":        rule.Enabled,
		}).Error; err != nil {
		return err
	}

	// 同步到 Istio
	if app.K8sClusterID != nil && app.K8sNamespace != "" {
		return h.istioService.SyncFaultRule(ctx, *app.K8sClusterID, app.K8sNamespace, app, &rule)
	}

	return nil
}

// ListVirtualServices 列出 VirtualService
func (h *TrafficMonitorHandler) ListVirtualServices(c *gin.Context) {
	app, err := h.getApp(c)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "应用不存在"})
		return
	}

	if app.K8sClusterID == nil || app.K8sNamespace == "" {
		c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"items": []any{}}})
		return
	}

	list, err := h.istioService.ListVirtualServices(c.Request.Context(), *app.K8sClusterID, app.K8sNamespace)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"items": list.Items}})
}

// ListDestinationRules 列出 DestinationRule
func (h *TrafficMonitorHandler) ListDestinationRules(c *gin.Context) {
	app, err := h.getApp(c)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "应用不存在"})
		return
	}

	if app.K8sClusterID == nil || app.K8sNamespace == "" {
		c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"items": []any{}}})
		return
	}

	list, err := h.istioService.ListDestinationRules(c.Request.Context(), *app.K8sClusterID, app.K8sNamespace)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"items": list.Items}})
}

// ListGateways 列出 Gateway
func (h *TrafficMonitorHandler) ListGateways(c *gin.Context) {
	app, err := h.getApp(c)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "应用不存在"})
		return
	}

	if app.K8sClusterID == nil || app.K8sNamespace == "" {
		c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"items": []any{}}})
		return
	}

	list, err := h.istioService.ListGateways(c.Request.Context(), *app.K8sClusterID, app.K8sNamespace)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"items": list.Items}})
}
