// Package handler 流水线模块处理器
// 本文件实现流水线模板和设计器的 IOC 注册
package handler

import (
	"devops/internal/config"
	"devops/pkg/ioc"
	"devops/pkg/middleware"

	"github.com/gin-gonic/gin"
)

func init() {
	ioc.Api.RegisterContainer("PipelineTemplateHandler", &PipelineTemplateApiHandler{})
	ioc.Api.RegisterContainer("PipelineDesignerHandler", &PipelineDesignerApiHandler{})
	ioc.Api.RegisterContainer("BuildHandler", &BuildApiHandler{})
}

// PipelineTemplateApiHandler 流水线模板 API Handler IOC 包装器
type PipelineTemplateApiHandler struct {
	handler *TemplateHandler
}

// Init 初始化 Handler
func (h *PipelineTemplateApiHandler) Init() error {
	cfg, err := config.LoadConfig()
	if err != nil {
		return err
	}

	db := cfg.GetDB()
	h.handler = NewTemplateHandler(db)

	root := cfg.Application.GinRootRouter().(*gin.RouterGroup)
	templateGroup := root.Group("")
	templateGroup.Use(middleware.AuthMiddleware())
	h.handler.RegisterRoutes(templateGroup)

	return nil
}

// PipelineDesignerApiHandler 流水线设计器 API Handler IOC 包装器
type PipelineDesignerApiHandler struct {
	handler *DesignerHandler
}

// Init 初始化 Handler
func (h *PipelineDesignerApiHandler) Init() error {
	cfg, err := config.LoadConfig()
	if err != nil {
		return err
	}

	db := cfg.GetDB()
	h.handler = NewDesignerHandler(db)

	root := cfg.Application.GinRootRouter().(*gin.RouterGroup)
	designerGroup := root.Group("")
	designerGroup.Use(middleware.AuthMiddleware())
	h.handler.RegisterRoutes(designerGroup)

	return nil
}


// BuildApiHandler 构建优化 API Handler IOC 包装器
type BuildApiHandler struct {
	handler *BuildHandler
}

// Init 初始化 Handler
func (h *BuildApiHandler) Init() error {
	cfg, err := config.LoadConfig()
	if err != nil {
		return err
	}

	db := cfg.GetDB()
	h.handler = NewBuildHandler(db)

	root := cfg.Application.GinRootRouter().(*gin.RouterGroup)
	buildGroup := root.Group("")
	buildGroup.Use(middleware.AuthMiddleware())
	h.handler.RegisterRoutes(buildGroup)

	return nil
}
