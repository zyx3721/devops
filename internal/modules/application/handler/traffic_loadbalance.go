// Package handler 应用模块处理器
// 本文件包含负载均衡配置相关的处理器方法
package handler

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"devops/internal/models"
)

// ========== 负载均衡配置 ==========

// GetLoadBalance 获取负载均衡配置
// @Summary 获取应用的负载均衡配置
// @Tags 流量治理-负载均衡
// @Param id path int true "应用ID"
// @Success 200 {object} gin.H
// @Router /applications/{id}/traffic/loadbalance [get]
func (h *TrafficHandler) GetLoadBalance(c *gin.Context) {
	app, err := h.getApp(c)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "应用不存在"})
		return
	}

	var configs []models.TrafficLoadBalanceConfig
	h.db.Where("app_id = ?", app.ID).Limit(1).Find(&configs)
	if len(configs) == 0 {
		c.JSON(http.StatusOK, gin.H{"code": 0, "data": nil})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": configs[0]})
}

// UpdateLoadBalance 更新负载均衡配置
// @Summary 更新负载均衡配置
// @Tags 流量治理-负载均衡
// @Param id path int true "应用ID"
// @Param body body models.TrafficLoadBalanceConfig true "负载均衡配置"
// @Success 200 {object} gin.H
// @Router /applications/{id}/traffic/loadbalance [put]
func (h *TrafficHandler) UpdateLoadBalance(c *gin.Context) {
	app, err := h.getApp(c)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "应用不存在"})
		return
	}

	var req models.TrafficLoadBalanceConfig
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	var config models.TrafficLoadBalanceConfig
	var configs []models.TrafficLoadBalanceConfig
	h.db.Where("app_id = ?", app.ID).Limit(1).Find(&configs)

	if len(configs) == 0 {
		// 创建新配置
		req.AppID = uint64(app.ID)
		if err := h.db.Create(&req).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
			return
		}
	} else {
		// 更新现有配置
		config = configs[0]
		h.db.Model(&config).Updates(map[string]any{
			"lb_policy":                  req.LbPolicy,
			"hash_key":                   req.HashKey,
			"hash_key_name":              req.HashKeyName,
			"ring_size":                  req.RingSize,
			"choice_count":               req.ChoiceCount,
			"warmup_duration":            req.WarmupDuration,
			"health_check_enabled":       req.HealthCheckEnabled,
			"health_check_path":          req.HealthCheckPath,
			"health_check_interval":      req.HealthCheckInterval,
			"health_check_timeout":       req.HealthCheckTimeout,
			"healthy_threshold":          req.HealthyThreshold,
			"unhealthy_threshold":        req.UnhealthyThreshold,
			"http_max_connections":       req.HTTPMaxConnections,
			"http_max_requests_per_conn": req.HTTPMaxRequestsPerConn,
			"http_max_pending_requests":  req.HTTPMaxPendingRequests,
			"http_max_retries":           req.HTTPMaxRetries,
			"http_idle_timeout":          req.HTTPIdleTimeout,
			"tcp_max_connections":        req.TCPMaxConnections,
			"tcp_connect_timeout":        req.TCPConnectTimeout,
			"tcp_keepalive_enabled":      req.TCPKeepaliveEnabled,
			"tcp_keepalive_interval":     req.TCPKeepaliveInterval,
		})
	}

	// 同步到 K8s DestinationRule
	h.syncLoadBalanceToK8s(app, &req)

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "保存成功"})
}

// syncLoadBalanceToK8s 同步负载均衡配置到 K8s
// 将负载均衡配置转换为 Istio DestinationRule 资源
func (h *TrafficHandler) syncLoadBalanceToK8s(app *models.Application, config *models.TrafficLoadBalanceConfig) error {
	if app.K8sClusterID == nil || app.K8sNamespace == "" {
		return nil
	}

	client, err := h.getDynamicClient(*app.K8sClusterID)
	if err != nil {
		return err
	}

	// 构建负载均衡策略
	trafficPolicy := map[string]any{
		"connectionPool": map[string]any{
			"tcp": map[string]any{
				"maxConnections": config.TCPMaxConnections,
				"connectTimeout": config.TCPConnectTimeout,
			},
			"http": map[string]any{
				"http2MaxRequests":         config.HTTPMaxConnections,
				"maxRequestsPerConnection": config.HTTPMaxRequestsPerConn,
				"maxPendingRequests":       config.HTTPMaxPendingRequests,
				"maxRetries":               config.HTTPMaxRetries,
				"idleTimeout":              config.HTTPIdleTimeout,
			},
		},
	}

	// 设置负载均衡算法
	lbSettings := map[string]any{}
	switch config.LbPolicy {
	case "round_robin":
		lbSettings["simple"] = "ROUND_ROBIN"
	case "random":
		lbSettings["simple"] = "RANDOM"
	case "least_request":
		lbSettings["simple"] = "LEAST_REQUEST"
	case "passthrough":
		lbSettings["simple"] = "PASSTHROUGH"
	case "consistent_hash":
		hashKey := map[string]any{}
		switch config.HashKey {
		case "header":
			hashKey["httpHeaderName"] = config.HashKeyName
		case "cookie":
			hashKey["httpCookie"] = map[string]any{
				"name": config.HashKeyName,
				"ttl":  "0s",
			}
		case "source_ip":
			hashKey["useSourceIp"] = true
		case "query_param":
			hashKey["httpQueryParameterName"] = config.HashKeyName
		}
		lbSettings["consistentHash"] = hashKey
	}
	trafficPolicy["loadBalancer"] = lbSettings

	drName := fmt.Sprintf("%s-lb-dr", app.Name)
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
				"host":          app.K8sDeployment,
				"trafficPolicy": trafficPolicy,
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
