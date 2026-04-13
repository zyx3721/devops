// Package handler 应用模块处理器
// 本文件包含流量路由相关的处理器方法
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

// ========== 流量路由 ==========

// ListRoutes 获取路由规则列表
// @Summary 获取应用的路由规则列表
// @Tags 流量治理-路由
// @Param id path int true "应用ID"
// @Success 200 {object} gin.H
// @Router /applications/{id}/traffic/routes [get]
func (h *TrafficHandler) ListRoutes(c *gin.Context) {
	app, err := h.getApp(c)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "应用不存在"})
		return
	}

	var rules []models.TrafficRoutingRule
	h.db.Where("app_id = ?", app.ID).Order("priority ASC").Find(&rules)

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"items": rules}})
}

// CreateRoute 创建路由规则
// @Summary 创建路由规则
// @Tags 流量治理-路由
// @Param id path int true "应用ID"
// @Param body body models.TrafficRoutingRule true "路由规则"
// @Success 200 {object} gin.H
// @Router /applications/{id}/traffic/routes [post]
func (h *TrafficHandler) CreateRoute(c *gin.Context) {
	app, err := h.getApp(c)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "应用不存在"})
		return
	}

	var rule models.TrafficRoutingRule
	if err := c.ShouldBindJSON(&rule); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	rule.AppID = uint64(app.ID)
	if err := h.db.Create(&rule).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	// 同步到 K8s VirtualService
	var syncErr error
	var syncMsg string
	if rule.Enabled {
		syncErr = h.syncRoutesToK8s(app)
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

// UpdateRoute 更新路由规则
// @Summary 更新路由规则
// @Tags 流量治理-路由
// @Param id path int true "应用ID"
// @Param ruleId path int true "规则ID"
// @Param body body models.TrafficRoutingRule true "路由规则"
// @Success 200 {object} gin.H
// @Router /applications/{id}/traffic/routes/{ruleId} [put]
func (h *TrafficHandler) UpdateRoute(c *gin.Context) {
	app, err := h.getApp(c)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "应用不存在"})
		return
	}

	ruleID, _ := strconv.ParseUint(c.Param("ruleId"), 10, 64)
	var rule models.TrafficRoutingRule
	if err := h.db.Where("id = ? AND app_id = ?", ruleID, app.ID).First(&rule).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "规则不存在"})
		return
	}

	var req models.TrafficRoutingRule
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	updates := map[string]any{
		"name":           req.Name,
		"description":    req.Description,
		"priority":       req.Priority,
		"route_type":     req.RouteType,
		"destinations":   req.Destinations,
		"match_key":      req.MatchKey,
		"match_operator": req.MatchOperator,
		"match_value":    req.MatchValue,
		"target_subset":  req.TargetSubset,
		"enabled":        req.Enabled,
	}

	h.db.Model(&rule).Updates(updates)

	// 同步到 K8s
	h.syncRoutesToK8s(app)

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "更新成功"})
}

// DeleteRoute 删除路由规则
// @Summary 删除路由规则
// @Tags 流量治理-路由
// @Param id path int true "应用ID"
// @Param ruleId path int true "规则ID"
// @Success 200 {object} gin.H
// @Router /applications/{id}/traffic/routes/{ruleId} [delete]
func (h *TrafficHandler) DeleteRoute(c *gin.Context) {
	app, err := h.getApp(c)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "应用不存在"})
		return
	}

	ruleID, _ := strconv.ParseUint(c.Param("ruleId"), 10, 64)
	h.db.Where("id = ? AND app_id = ?", ruleID, app.ID).Delete(&models.TrafficRoutingRule{})

	// 重新同步 K8s
	h.syncRoutesToK8s(app)

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "删除成功"})
}

// syncRoutesToK8s 同步路由规则到 K8s VirtualService
// 将路由规则转换为 Istio VirtualService 资源
func (h *TrafficHandler) syncRoutesToK8s(app *models.Application) error {
	if app.K8sClusterID == nil || app.K8sNamespace == "" {
		return nil
	}

	client, err := h.getDynamicClient(*app.K8sClusterID)
	if err != nil {
		return err
	}

	// 获取所有启用的路由规则
	var rules []models.TrafficRoutingRule
	h.db.Where("app_id = ? AND enabled = ?", app.ID, true).Order("priority ASC").Find(&rules)

	// 构建 HTTP 路由
	httpRoutes := []any{}
	for _, rule := range rules {
		route := map[string]any{}

		if rule.RouteType == "weight" && len(rule.Destinations) > 0 {
			// 权重路由
			destinations := []any{}
			for _, dest := range rule.Destinations {
				destinations = append(destinations, map[string]any{
					"destination": map[string]any{
						"host":   app.K8sDeployment,
						"subset": dest.Subset,
					},
					"weight": dest.Weight,
				})
			}
			route["route"] = destinations
		} else {
			// 条件路由
			match := []any{}
			matchCondition := map[string]any{}

			switch rule.RouteType {
			case "header":
				matchCondition["headers"] = map[string]any{
					rule.MatchKey: buildMatchValue(rule.MatchOperator, rule.MatchValue),
				}
			case "cookie":
				matchCondition["headers"] = map[string]any{
					"cookie": map[string]any{
						"regex": fmt.Sprintf(".*%s=%s.*", rule.MatchKey, rule.MatchValue),
					},
				}
			case "param":
				matchCondition["queryParams"] = map[string]any{
					rule.MatchKey: buildMatchValue(rule.MatchOperator, rule.MatchValue),
				}
			}

			if len(matchCondition) > 0 {
				match = append(match, matchCondition)
				route["match"] = match
			}

			route["route"] = []any{
				map[string]any{
					"destination": map[string]any{
						"host":   app.K8sDeployment,
						"subset": rule.TargetSubset,
					},
				},
			}
		}

		httpRoutes = append(httpRoutes, route)
	}

	// 添加默认路由
	if len(httpRoutes) == 0 {
		httpRoutes = append(httpRoutes, map[string]any{
			"route": []any{
				map[string]any{
					"destination": map[string]any{
						"host": app.K8sDeployment,
					},
				},
			},
		})
	}

	vsName := fmt.Sprintf("%s-vs", app.Name)
	vs := &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "networking.istio.io/v1beta1",
			"kind":       "VirtualService",
			"metadata": map[string]any{
				"name":      vsName,
				"namespace": app.K8sNamespace,
				"labels": map[string]any{
					"app":        app.Name,
					"managed-by": "devops-platform",
				},
			},
			"spec": map[string]any{
				"hosts": []string{app.K8sDeployment},
				"http":  httpRoutes,
			},
		},
	}

	ctx := context.Background()
	_, err = client.Resource(virtualServiceGVR).Namespace(app.K8sNamespace).Create(ctx, vs, metav1.CreateOptions{})
	if err != nil {
		_, err = client.Resource(virtualServiceGVR).Namespace(app.K8sNamespace).Update(ctx, vs, metav1.UpdateOptions{})
	}
	return err
}

// buildMatchValue 构建路由匹配条件
// 根据操作符类型返回对应的 Istio 匹配配置
func buildMatchValue(operator, value string) map[string]any {
	switch operator {
	case "exact":
		return map[string]any{"exact": value}
	case "prefix":
		return map[string]any{"prefix": value}
	case "regex":
		return map[string]any{"regex": value}
	case "present":
		return map[string]any{"regex": ".*"}
	default:
		return map[string]any{"exact": value}
	}
}
