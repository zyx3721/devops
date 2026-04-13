package handler

import (
	"devops/internal/config"
	"devops/internal/service/kubernetes"
	"devops/pkg/ioc"
	"devops/pkg/middleware"

	"github.com/gin-gonic/gin"
)

func init() {
	ioc.Api.RegisterContainer("TrafficHandler", &TrafficApiHandler{})
	ioc.Api.RegisterContainer("TrafficMonitorHandler", &TrafficMonitorApiHandler{})
	ioc.Api.RegisterContainer("CanaryHandler", &CanaryApiHandler{})
	ioc.Api.RegisterContainer("TrafficTestHandler", &TrafficTestApiHandler{})
}

// TrafficApiHandler 流量治理 API Handler IOC 包装器
type TrafficApiHandler struct {
	handler *TrafficHandler
}

// Init 初始化 Handler
func (h *TrafficApiHandler) Init() error {
	cfg, err := config.LoadConfig()
	if err != nil {
		return err
	}

	db := cfg.GetDB()
	clientMgr := kubernetes.NewK8sClientManager(db)
	h.handler = NewTrafficHandler(db, clientMgr)

	root := cfg.Application.GinRootRouter().(*gin.RouterGroup)
	trafficGroup := root.Group("")
	trafficGroup.Use(middleware.AuthMiddleware())
	h.handler.RegisterRoutes(trafficGroup)

	return nil
}

// TrafficMonitorApiHandler 流量监控 API Handler IOC 包装器
type TrafficMonitorApiHandler struct {
	handler *TrafficMonitorHandler
}

// Init 初始化 Handler
func (h *TrafficMonitorApiHandler) Init() error {
	cfg, err := config.LoadConfig()
	if err != nil {
		return err
	}

	db := cfg.GetDB()
	clientMgr := kubernetes.NewK8sClientManager(db)
	istioService := kubernetes.NewIstioService(clientMgr, db)
	h.handler = NewTrafficMonitorHandler(db, istioService)

	root := cfg.Application.GinRootRouter().(*gin.RouterGroup)
	monitorGroup := root.Group("")
	monitorGroup.Use(middleware.AuthMiddleware())
	h.handler.RegisterRoutes(monitorGroup)

	return nil
}

// CanaryApiHandler 灰度发布 API Handler IOC 包装器
type CanaryApiHandler struct {
	handler *CanaryHandler
}

// Init 初始化 Handler
func (h *CanaryApiHandler) Init() error {
	cfg, err := config.LoadConfig()
	if err != nil {
		return err
	}

	db := cfg.GetDB()
	clientMgr := kubernetes.NewK8sClientManager(db)
	istioService := kubernetes.NewIstioService(clientMgr, db)
	h.handler = NewCanaryHandler(db, clientMgr, istioService)

	root := cfg.Application.GinRootRouter().(*gin.RouterGroup)
	canaryGroup := root.Group("")
	canaryGroup.Use(middleware.AuthMiddleware())
	h.handler.RegisterRoutes(canaryGroup)

	return nil
}

// TrafficTestApiHandler 流量测试 API Handler IOC 包装器
type TrafficTestApiHandler struct {
	handler *TrafficTestHandler
}

// Init 初始化 Handler
func (h *TrafficTestApiHandler) Init() error {
	cfg, err := config.LoadConfig()
	if err != nil {
		return err
	}

	db := cfg.GetDB()
	clientMgr := kubernetes.NewK8sClientManager(db)
	istioService := kubernetes.NewIstioService(clientMgr, db)
	h.handler = NewTrafficTestHandler(db, istioService)

	root := cfg.Application.GinRootRouter().(*gin.RouterGroup)
	testGroup := root.Group("")
	testGroup.Use(middleware.AuthMiddleware())
	h.handler.RegisterRoutes(testGroup)

	return nil
}
