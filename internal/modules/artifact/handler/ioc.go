// Package handler 制品管理模块处理器
package handler

import (
	"devops/internal/config"
	"devops/pkg/ioc"
	"devops/pkg/middleware"

	"github.com/gin-gonic/gin"
)

func init() {
	ioc.Api.RegisterContainer("ArtifactRepositoryHandler", &ArtifactApiHandler{})
}

// ArtifactApiHandler 制品管理 API Handler IOC 包装器
type ArtifactApiHandler struct {
	handler *ArtifactHandler
}

// Init 初始化 Handler
func (h *ArtifactApiHandler) Init() error {
	cfg, err := config.LoadConfig()
	if err != nil {
		return err
	}

	db := cfg.GetDB()
	h.handler = NewArtifactHandler(db)

	root := cfg.Application.GinRootRouter().(*gin.RouterGroup)
	artifactGroup := root.Group("")
	artifactGroup.Use(middleware.AuthMiddleware())
	h.handler.RegisterRoutes(artifactGroup)

	return nil
}
