// Package handler 应用模块处理器
// 本文件包含流量治理处理器的核心结构和路由注册
//
// 流量治理功能按功能拆分为多个文件：
//   - traffic_handler.go      - 核心结构、路由注册、公共方法
//   - traffic_ratelimit.go    - 限流规则 CRUD
//   - traffic_circuitbreaker.go - 熔断规则 CRUD
//   - traffic_routing.go      - 流量路由 CRUD
//   - traffic_loadbalance.go  - 负载均衡配置
//   - traffic_timeout.go      - 超时重试配置
//   - traffic_mirror.go       - 流量镜像 CRUD
//   - traffic_fault.go        - 故障注入 CRUD
package handler

import (
	"context"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"

	"devops/internal/models"
	"devops/internal/service/kubernetes"
)

// TrafficHandler 流量治理处理器
// 提供流量治理相关的 API 接口
type TrafficHandler struct {
	db        *gorm.DB                       // 数据库连接
	clientMgr *kubernetes.K8sClientManager   // K8s 客户端管理器
}

// NewTrafficHandler 创建流量治理处理器
func NewTrafficHandler(db *gorm.DB, clientMgr *kubernetes.K8sClientManager) *TrafficHandler {
	return &TrafficHandler{db: db, clientMgr: clientMgr}
}

// RegisterRoutes 注册路由
// 将所有流量治理相关的路由注册到指定的路由组
func (h *TrafficHandler) RegisterRoutes(r *gin.RouterGroup) {
	g := r.Group("/applications/:id/traffic")
	{
		// 限流规则 CRUD
		g.GET("/ratelimits", h.ListRateLimits)
		g.POST("/ratelimits", h.CreateRateLimit)
		g.PUT("/ratelimits/:ruleId", h.UpdateRateLimit)
		g.DELETE("/ratelimits/:ruleId", h.DeleteRateLimit)

		// 熔断规则 CRUD
		g.GET("/circuitbreakers", h.ListCircuitBreakers)
		g.POST("/circuitbreakers", h.CreateCircuitBreaker)
		g.PUT("/circuitbreakers/:ruleId", h.UpdateCircuitBreaker)
		g.DELETE("/circuitbreakers/:ruleId", h.DeleteCircuitBreaker)

		// 熔断配置（配置型，兼容旧版前端）
		g.GET("/circuitbreaker", h.GetCircuitBreakerConfig)
		g.PUT("/circuitbreaker", h.UpdateCircuitBreakerConfig)

		// 流量路由 CRUD
		g.GET("/routes", h.ListRoutes)
		g.POST("/routes", h.CreateRoute)
		g.PUT("/routes/:ruleId", h.UpdateRoute)
		g.DELETE("/routes/:ruleId", h.DeleteRoute)

		// 负载均衡配置
		g.GET("/loadbalance", h.GetLoadBalance)
		g.PUT("/loadbalance", h.UpdateLoadBalance)

		// 超时重试配置
		g.GET("/timeout", h.GetTimeout)
		g.PUT("/timeout", h.UpdateTimeout)

		// 流量镜像 CRUD
		g.GET("/mirrors", h.ListMirrors)
		g.POST("/mirrors", h.CreateMirror)
		g.PUT("/mirrors/:ruleId", h.UpdateMirror)
		g.DELETE("/mirrors/:ruleId", h.DeleteMirror)

		// 故障注入 CRUD
		g.GET("/faults", h.ListFaults)
		g.POST("/faults", h.CreateFault)
		g.PUT("/faults/:ruleId", h.UpdateFault)
		g.DELETE("/faults/:ruleId", h.DeleteFault)
	}
}

// ========== Istio GVR 定义 ==========

// Istio 资源的 GroupVersionResource 定义
var (
	// virtualServiceGVR VirtualService 资源定义
	virtualServiceGVR = schema.GroupVersionResource{
		Group: "networking.istio.io", Version: "v1beta1", Resource: "virtualservices",
	}
	// destinationRuleGVR DestinationRule 资源定义
	destinationRuleGVR = schema.GroupVersionResource{
		Group: "networking.istio.io", Version: "v1beta1", Resource: "destinationrules",
	}
	// envoyFilterGVR EnvoyFilter 资源定义
	envoyFilterGVR = schema.GroupVersionResource{
		Group: "networking.istio.io", Version: "v1alpha3", Resource: "envoyfilters",
	}
)

// ========== 公共方法 ==========

// getApp 获取应用信息
// 从请求路径中解析应用ID并查询数据库
func (h *TrafficHandler) getApp(c *gin.Context) (*models.Application, error) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	var app models.Application
	if err := h.db.First(&app, id).Error; err != nil {
		return nil, err
	}
	return &app, nil
}

// getDynamicClient 获取 K8s 动态客户端
// 根据集群ID获取对应的 K8s 动态客户端
func (h *TrafficHandler) getDynamicClient(clusterID uint) (dynamic.Interface, error) {
	config, err := h.clientMgr.GetConfig(context.Background(), clusterID)
	if err != nil {
		return nil, err
	}
	return dynamic.NewForConfig(config)
}
