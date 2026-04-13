package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"devops/internal/models/ai"
	"devops/internal/repository"
	"devops/pkg/response"
)

// ConfigHandler LLM配置处理器
type ConfigHandler struct {
	db   *gorm.DB
	repo *repository.AILLMConfigRepository
}

// NewConfigHandler 创建配置处理器
func NewConfigHandler(db *gorm.DB) *ConfigHandler {
	return &ConfigHandler{
		db:   db,
		repo: repository.NewAILLMConfigRepository(db),
	}
}

// CreateConfigRequest 创建配置请求
type CreateConfigRequest struct {
	Name           string  `json:"name" binding:"required"`
	Provider       string  `json:"provider" binding:"required"`
	APIURL         string  `json:"api_url" binding:"required"`
	APIKey         string  `json:"api_key" binding:"required"`
	ModelName      string  `json:"model_name" binding:"required"`
	MaxTokens      int     `json:"max_tokens"`
	Temperature    float64 `json:"temperature"`
	TimeoutSeconds int     `json:"timeout_seconds"`
	IsDefault      bool    `json:"is_default"`
	Description    string  `json:"description"`
}

// List 获取配置列表
// @Summary 获取LLM配置列表
// @Tags AI Config
// @Produce json
// @Success 200 {object} response.Response
// @Router /api/v1/ai/config [get]
func (h *ConfigHandler) List(c *gin.Context) {
	configs, err := h.repo.List(c.Request.Context())
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "获取配置失败: "+err.Error())
		return
	}

	// 隐藏API Key
	for i := range configs {
		configs[i].APIKeyEncrypted = "******"
	}

	response.Success(c, configs)
}

// Get 获取配置详情
// @Summary 获取LLM配置详情
// @Tags AI Config
// @Produce json
// @Param id path int true "配置ID"
// @Success 200 {object} response.Response
// @Router /api/v1/ai/config/{id} [get]
func (h *ConfigHandler) Get(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "无效的ID")
		return
	}

	config, err := h.repo.GetByID(c.Request.Context(), uint(id))
	if err != nil {
		response.Error(c, http.StatusNotFound, "配置不存在")
		return
	}

	// 隐藏API Key
	config.APIKeyEncrypted = "******"

	response.Success(c, config)
}

// Create 创建配置
// @Summary 创建LLM配置
// @Tags AI Config
// @Accept json
// @Produce json
// @Param request body CreateConfigRequest true "配置内容"
// @Success 200 {object} response.Response
// @Router /api/v1/ai/config [post]
func (h *ConfigHandler) Create(c *gin.Context) {
	var req CreateConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}

	userID := getUserID(c)

	config := &ai.AILLMConfig{
		Name:            req.Name,
		Provider:        ai.LLMProvider(req.Provider),
		APIURL:          req.APIURL,
		APIKeyEncrypted: req.APIKey, // 实际应该加密存储
		ModelName:       req.ModelName,
		MaxTokens:       req.MaxTokens,
		Temperature:     req.Temperature,
		TimeoutSeconds:  req.TimeoutSeconds,
		IsDefault:       req.IsDefault,
		IsActive:        true,
		Description:     req.Description,
		CreatedBy:       &userID,
		UpdatedBy:       &userID,
	}

	// 设置默认值
	if config.MaxTokens == 0 {
		config.MaxTokens = 4096
	}
	if config.Temperature == 0 {
		config.Temperature = 0.7
	}
	if config.TimeoutSeconds == 0 {
		config.TimeoutSeconds = 60
	}

	if err := h.repo.Create(c.Request.Context(), config); err != nil {
		response.Error(c, http.StatusInternalServerError, "创建失败: "+err.Error())
		return
	}

	// 如果设置为默认，更新其他配置
	if config.IsDefault {
		h.repo.SetDefault(c.Request.Context(), config.ID)
	}

	config.APIKeyEncrypted = "******"
	response.Success(c, config)
}

// UpdateConfigRequest 更新配置请求
type UpdateConfigRequest struct {
	Name           string  `json:"name"`
	Provider       string  `json:"provider"`
	APIURL         string  `json:"api_url"`
	APIKey         string  `json:"api_key"`
	ModelName      string  `json:"model_name"`
	MaxTokens      int     `json:"max_tokens"`
	Temperature    float64 `json:"temperature"`
	TimeoutSeconds int     `json:"timeout_seconds"`
	IsDefault      bool    `json:"is_default"`
	IsActive       bool    `json:"is_active"`
	Description    string  `json:"description"`
}

// Update 更新配置
// @Summary 更新LLM配置
// @Tags AI Config
// @Accept json
// @Produce json
// @Param id path int true "配置ID"
// @Param request body UpdateConfigRequest true "配置内容"
// @Success 200 {object} response.Response
// @Router /api/v1/ai/config/{id} [put]
func (h *ConfigHandler) Update(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "无效的ID")
		return
	}

	var req UpdateConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}

	config, err := h.repo.GetByID(c.Request.Context(), uint(id))
	if err != nil {
		response.Error(c, http.StatusNotFound, "配置不存在")
		return
	}

	userID := getUserID(c)

	// 更新字段
	if req.Name != "" {
		config.Name = req.Name
	}
	if req.Provider != "" {
		config.Provider = ai.LLMProvider(req.Provider)
	}
	if req.APIURL != "" {
		config.APIURL = req.APIURL
	}
	if req.APIKey != "" && req.APIKey != "******" {
		config.APIKeyEncrypted = req.APIKey
	}
	if req.ModelName != "" {
		config.ModelName = req.ModelName
	}
	if req.MaxTokens > 0 {
		config.MaxTokens = req.MaxTokens
	}
	if req.Temperature > 0 {
		config.Temperature = req.Temperature
	}
	if req.TimeoutSeconds > 0 {
		config.TimeoutSeconds = req.TimeoutSeconds
	}
	config.IsDefault = req.IsDefault
	config.IsActive = req.IsActive
	config.Description = req.Description
	config.UpdatedBy = &userID

	if err := h.repo.Update(c.Request.Context(), config); err != nil {
		response.Error(c, http.StatusInternalServerError, "更新失败: "+err.Error())
		return
	}

	// 如果设置为默认，更新其他配置
	if config.IsDefault {
		h.repo.SetDefault(c.Request.Context(), config.ID)
	}

	config.APIKeyEncrypted = "******"
	response.Success(c, config)
}

// Delete 删除配置
// @Summary 删除LLM配置
// @Tags AI Config
// @Param id path int true "配置ID"
// @Success 200 {object} response.Response
// @Router /api/v1/ai/config/{id} [delete]
func (h *ConfigHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "无效的ID")
		return
	}

	if err := h.repo.Delete(c.Request.Context(), uint(id)); err != nil {
		response.Error(c, http.StatusInternalServerError, "删除失败: "+err.Error())
		return
	}

	response.Success(c, nil)
}

// SetDefault 设置默认配置
// @Summary 设置默认LLM配置
// @Tags AI Config
// @Param id path int true "配置ID"
// @Success 200 {object} response.Response
// @Router /api/v1/ai/config/{id}/default [post]
func (h *ConfigHandler) SetDefault(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "无效的ID")
		return
	}

	if err := h.repo.SetDefault(c.Request.Context(), uint(id)); err != nil {
		response.Error(c, http.StatusInternalServerError, "设置失败: "+err.Error())
		return
	}

	response.Success(c, nil)
}

// GetProviders 获取支持的提供商列表
// @Summary 获取支持的LLM提供商列表
// @Tags AI Config
// @Produce json
// @Success 200 {object} response.Response
// @Router /api/v1/ai/config/providers [get]
func (h *ConfigHandler) GetProviders(c *gin.Context) {
	providers := []map[string]string{
		{"value": "openai", "label": "OpenAI"},
		{"value": "azure", "label": "Azure OpenAI"},
		{"value": "deepseek", "label": "DeepSeek"},
		{"value": "qwen", "label": "通义千问"},
		{"value": "zhipu", "label": "智谱AI"},
		{"value": "ollama", "label": "Ollama (本地)"},
		{"value": "custom", "label": "自定义 (OpenAI兼容)"},
	}

	response.Success(c, providers)
}

// RegisterRoutes 注册LLM配置相关路由
func (h *ConfigHandler) RegisterRoutes(r *gin.RouterGroup) {
	g := r.Group("/ai/config")
	{
		g.GET("", h.List)
		g.POST("", h.Create)
		g.GET("/providers", h.GetProviders)
		g.GET("/:id", h.Get)
		g.PUT("/:id", h.Update)
		g.DELETE("/:id", h.Delete)
		g.POST("/:id/default", h.SetDefault)
	}
}
