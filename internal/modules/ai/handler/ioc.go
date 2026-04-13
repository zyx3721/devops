// Package handler AI模块处理器
// 本文件实现AI Copilot的IOC注册
package handler

import (
	"context"

	"devops/internal/config"
	"devops/internal/repository"
	"devops/pkg/ioc"
	"devops/pkg/llm"
	"devops/pkg/logger"
	"devops/pkg/middleware"

	"github.com/gin-gonic/gin"
)

func init() {
	ioc.Api.RegisterContainer("AIHandler", &AIApiHandler{})
	ioc.Api.RegisterContainer("AIKnowledgeHandler", &KnowledgeApiHandler{})
	ioc.Api.RegisterContainer("AIConfigHandler", &ConfigApiHandler{})
}

// AIApiHandler AI聊天 API Handler IOC 包装器
type AIApiHandler struct {
	handler *AIHandler
}

// Init 初始化 Handler
func (h *AIApiHandler) Init() error {
	cfg, err := config.LoadConfig()
	if err != nil {
		return err
	}

	db := cfg.GetDB()

	// 创建 LLM 客户端工厂函数，每次调用时获取最新配置
	llmClientFactory := func() (llm.Client, error) {
		configRepo := repository.NewAILLMConfigRepository(db)
		llmConfig, err := configRepo.GetDefault(context.Background())
		if err != nil || llmConfig == nil {
			logger.L().Warn("未找到默认LLM配置，AI功能可能不可用")
			return nil, err
		}

		if !llmConfig.IsActive {
			logger.L().Warn("默认LLM配置未启用")
			return nil, nil
		}

		logger.L().Info("创建LLM客户端", "provider", llmConfig.Provider, "model", llmConfig.ModelName)
		return llm.NewOpenAIClient(llm.Config{
			Provider:       string(llmConfig.Provider),
			APIURL:         llmConfig.APIURL,
			APIKey:         llmConfig.APIKeyEncrypted,
			Model:          llmConfig.ModelName,
			MaxTokens:      llmConfig.MaxTokens,
			Temperature:    llmConfig.Temperature,
			TimeoutSeconds: llmConfig.TimeoutSeconds,
		})
	}

	h.handler = NewAIHandlerWithFactory(db, llmClientFactory)

	root := cfg.Application.GinRootRouter().(*gin.RouterGroup)
	aiGroup := root.Group("")
	aiGroup.Use(middleware.AuthMiddleware())
	h.handler.RegisterRoutes(aiGroup)

	return nil
}

// KnowledgeApiHandler 知识库 API Handler IOC 包装器
type KnowledgeApiHandler struct {
	handler *KnowledgeHandler
}

// Init 初始化 Handler
func (h *KnowledgeApiHandler) Init() error {
	cfg, err := config.LoadConfig()
	if err != nil {
		return err
	}

	db := cfg.GetDB()
	h.handler = NewKnowledgeHandler(db)

	root := cfg.Application.GinRootRouter().(*gin.RouterGroup)
	knowledgeGroup := root.Group("")
	knowledgeGroup.Use(middleware.AuthMiddleware())
	h.handler.RegisterRoutes(knowledgeGroup)

	return nil
}

// ConfigApiHandler LLM配置 API Handler IOC 包装器
type ConfigApiHandler struct {
	handler *ConfigHandler
}

// Init 初始化 Handler
func (h *ConfigApiHandler) Init() error {
	cfg, err := config.LoadConfig()
	if err != nil {
		return err
	}

	db := cfg.GetDB()
	h.handler = NewConfigHandler(db)

	root := cfg.Application.GinRootRouter().(*gin.RouterGroup)
	configGroup := root.Group("")
	configGroup.Use(middleware.AuthMiddleware())
	h.handler.RegisterRoutes(configGroup)

	return nil
}
