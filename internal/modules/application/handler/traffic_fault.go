// Package handler 应用模块处理器
// 本文件包含故障注入相关的处理器方法
package handler

import (
	"context"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"devops/internal/models"
)

// ========== 故障注入 ==========

// ListFaults 获取故障注入列表
// @Summary 获取应用的故障注入规则列表
// @Tags 流量治理-故障注入
// @Param id path int true "应用ID"
// @Success 200 {object} gin.H
// @Router /applications/{id}/traffic/faults [get]
func (h *TrafficHandler) ListFaults(c *gin.Context) {
	app, err := h.getApp(c)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "应用不存在"})
		return
	}

	var rules []models.TrafficFaultRule
	h.db.Where("app_id = ?", app.ID).Find(&rules)

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"items": rules}})
}

// CreateFault 创建故障注入
// @Summary 创建故障注入规则
// @Tags 流量治理-故障注入
// @Param id path int true "应用ID"
// @Param body body models.TrafficFaultRule true "故障注入规则"
// @Success 200 {object} gin.H
// @Router /applications/{id}/traffic/faults [post]
func (h *TrafficHandler) CreateFault(c *gin.Context) {
	app, err := h.getApp(c)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "应用不存在"})
		return
	}

	var rule models.TrafficFaultRule
	if err := c.ShouldBindJSON(&rule); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	rule.AppID = uint64(app.ID)
	if err := h.db.Create(&rule).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	// 同步到 K8s
	if rule.Enabled {
		h.syncFaultToK8s(app)
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "创建成功", "data": rule})
}

// UpdateFault 更新故障注入
// @Summary 更新故障注入规则
// @Tags 流量治理-故障注入
// @Param id path int true "应用ID"
// @Param ruleId path int true "规则ID"
// @Param body body models.TrafficFaultRule true "故障注入规则"
// @Success 200 {object} gin.H
// @Router /applications/{id}/traffic/faults/{ruleId} [put]
func (h *TrafficHandler) UpdateFault(c *gin.Context) {
	app, err := h.getApp(c)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "应用不存在"})
		return
	}

	ruleID, _ := strconv.ParseUint(c.Param("ruleId"), 10, 64)
	var req models.TrafficFaultRule
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	h.db.Model(&models.TrafficFaultRule{}).Where("id = ? AND app_id = ?", ruleID, app.ID).Updates(map[string]any{
		"type":           req.Type,
		"path":           req.Path,
		"delay_duration": req.DelayDuration,
		"abort_code":     req.AbortCode,
		"abort_message":  req.AbortMessage,
		"percentage":     req.Percentage,
		"enabled":        req.Enabled,
	})

	// 同步到 K8s
	h.syncFaultToK8s(app)

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "更新成功"})
}

// DeleteFault 删除故障注入
// @Summary 删除故障注入规则
// @Tags 流量治理-故障注入
// @Param id path int true "应用ID"
// @Param ruleId path int true "规则ID"
// @Success 200 {object} gin.H
// @Router /applications/{id}/traffic/faults/{ruleId} [delete]
func (h *TrafficHandler) DeleteFault(c *gin.Context) {
	app, err := h.getApp(c)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "应用不存在"})
		return
	}

	ruleID, _ := strconv.ParseUint(c.Param("ruleId"), 10, 64)
	h.db.Where("id = ? AND app_id = ?", ruleID, app.ID).Delete(&models.TrafficFaultRule{})

	// 重新同步 K8s
	h.syncFaultToK8s(app)

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "删除成功"})
}

// syncFaultToK8s 同步故障注入到 K8s VirtualService
// 将故障注入规则转换为 Istio VirtualService 的 fault 配置
func (h *TrafficHandler) syncFaultToK8s(app *models.Application) error {
	if app.K8sClusterID == nil || app.K8sNamespace == "" {
		return nil
	}

	client, err := h.getDynamicClient(*app.K8sClusterID)
	if err != nil {
		return err
	}

	// 获取所有启用的故障注入规则
	var rules []models.TrafficFaultRule
	h.db.Where("app_id = ? AND enabled = ?", app.ID, true).Find(&rules)

	httpRoutes := []any{}
	for _, rule := range rules {
		route := map[string]any{
			"match": []any{
				map[string]any{
					"uri": map[string]any{
						"prefix": rule.Path,
					},
				},
			},
			"route": []any{
				map[string]any{
					"destination": map[string]any{
						"host": app.K8sDeployment,
					},
				},
			},
		}

		fault := map[string]any{}
		if rule.Type == "delay" {
			fault["delay"] = map[string]any{
				"percentage": map[string]any{
					"value": float64(rule.Percentage),
				},
				"fixedDelay": rule.DelayDuration,
			}
		} else if rule.Type == "abort" {
			fault["abort"] = map[string]any{
				"percentage": map[string]any{
					"value": float64(rule.Percentage),
				},
				"httpStatus": rule.AbortCode,
			}
		}
		route["fault"] = fault

		httpRoutes = append(httpRoutes, route)
	}

	// 添加默认路由
	httpRoutes = append(httpRoutes, map[string]any{
		"route": []any{
			map[string]any{
				"destination": map[string]any{
					"host": app.K8sDeployment,
				},
			},
		},
	})

	vsName := app.Name + "-fault-vs"
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
