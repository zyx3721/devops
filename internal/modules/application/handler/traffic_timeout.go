// Package handler 应用模块处理器
// 本文件包含超时重试配置相关的处理器方法
package handler

import (
	"context"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"devops/internal/models"
)

// ========== 超时重试配置 ==========

// GetTimeout 获取超时配置
// @Summary 获取应用的超时重试配置
// @Tags 流量治理-超时重试
// @Param id path int true "应用ID"
// @Success 200 {object} gin.H
// @Router /applications/{id}/traffic/timeout [get]
func (h *TrafficHandler) GetTimeout(c *gin.Context) {
	app, err := h.getApp(c)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "应用不存在"})
		return
	}

	var configs []models.TrafficTimeoutConfig
	h.db.Where("app_id = ?", app.ID).Limit(1).Find(&configs)
	if len(configs) == 0 {
		c.JSON(http.StatusOK, gin.H{"code": 0, "data": nil})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": configs[0]})
}

// UpdateTimeout 更新超时配置
// @Summary 更新超时重试配置
// @Tags 流量治理-超时重试
// @Param id path int true "应用ID"
// @Param body body models.TrafficTimeoutConfig true "超时配置"
// @Success 200 {object} gin.H
// @Router /applications/{id}/traffic/timeout [put]
func (h *TrafficHandler) UpdateTimeout(c *gin.Context) {
	app, err := h.getApp(c)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "应用不存在"})
		return
	}

	var req models.TrafficTimeoutConfig
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	var config models.TrafficTimeoutConfig
	var configs []models.TrafficTimeoutConfig
	h.db.Where("app_id = ?", app.ID).Limit(1).Find(&configs)

	if len(configs) == 0 {
		req.AppID = uint64(app.ID)
		if err := h.db.Create(&req).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
			return
		}
	} else {
		config = configs[0]
		h.db.Model(&config).Updates(map[string]any{
			"timeout":         req.Timeout,
			"retries":         req.Retries,
			"per_try_timeout": req.PerTryTimeout,
			"retry_on":        req.RetryOn,
		})
	}

	// 同步到 K8s VirtualService
	h.syncTimeoutToK8s(app, &req)

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "保存成功"})
}

// syncTimeoutToK8s 同步超时配置到 K8s
// 将超时重试配置转换为 Istio VirtualService 资源
func (h *TrafficHandler) syncTimeoutToK8s(app *models.Application, config *models.TrafficTimeoutConfig) error {
	if app.K8sClusterID == nil || app.K8sNamespace == "" {
		return nil
	}

	client, err := h.getDynamicClient(*app.K8sClusterID)
	if err != nil {
		return err
	}

	retryOnStr := "5xx"
	if len(config.RetryOn) > 0 {
		retryOnStr = strings.Join(config.RetryOn, ",")
	}

	vsName := app.Name + "-timeout-vs"
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
				"http": []any{
					map[string]any{
						"timeout": config.Timeout,
						"retries": map[string]any{
							"attempts":      config.Retries,
							"perTryTimeout": config.PerTryTimeout,
							"retryOn":       retryOnStr,
						},
						"route": []any{
							map[string]any{
								"destination": map[string]any{
									"host": app.K8sDeployment,
								},
							},
						},
					},
				},
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
