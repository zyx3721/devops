// Package handler 流水线模块处理器
package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"devops/internal/models"
	"devops/internal/models/pipeline"
)

// TemplateHandler 流水线模板处理器
type TemplateHandler struct {
	db *gorm.DB
}

// NewTemplateHandler 创建模板处理器
func NewTemplateHandler(db *gorm.DB) *TemplateHandler {
	return &TemplateHandler{db: db}
}

// RegisterRoutes 注册路由
func (h *TemplateHandler) RegisterRoutes(r *gin.RouterGroup) {
	g := r.Group("/pipeline/templates")
	{
		// 流水线模板
		g.GET("", h.ListTemplates)
		g.GET("/:id", h.GetTemplate)
		g.POST("", h.CreateTemplate)
		g.PUT("/:id", h.UpdateTemplate)
		g.DELETE("/:id", h.DeleteTemplate)
		g.POST("/:id/apply", h.ApplyTemplate)
		g.POST("/:id/rate", h.RateTemplate)

		// 模板分类、标签、收藏
		g.GET("/categories", h.GetCategories)
		g.GET("/tags", h.GetTags)
		g.GET("/favorites", h.GetFavorites)

		// 阶段模板
		g.GET("/stages", h.ListStageTemplates)

		// 步骤模板
		g.GET("/steps", h.ListStepTemplates)
	}
}

// ListTemplates 获取流水线模板列表
// @Summary 获取流水线模板列表
// @Tags 流水线模板
// @Param category query string false "分类"
// @Param language query string false "编程语言"
// @Param keyword query string false "关键词"
// @Success 200 {object} gin.H
// @Router /pipeline/templates [get]
func (h *TemplateHandler) ListTemplates(c *gin.Context) {
	category := c.Query("category")
	language := c.Query("language")
	keyword := c.Query("keyword")

	query := h.db.Model(&pipeline.PipelineTemplate{}).Where("is_public = ?", true)

	if category != "" {
		query = query.Where("category = ?", category)
	}
	if language != "" {
		query = query.Where("language = ?", language)
	}
	if keyword != "" {
		query = query.Where("name LIKE ? OR description LIKE ?", "%"+keyword+"%", "%"+keyword+"%")
	}

	var templates []pipeline.PipelineTemplate
	query.Order("is_builtin DESC, usage_count DESC, rating DESC, created_at DESC").Find(&templates)

	// 如果没有模板，初始化内置模板
	if len(templates) == 0 {
		h.initBuiltinTemplates()
		query.Find(&templates)
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"items": templates}})
}

// GetTemplate 获取模板详情
// @Summary 获取模板详情
// @Tags 流水线模板
// @Param id path int true "模板ID"
// @Success 200 {object} gin.H
// @Router /pipeline/templates/{id} [get]
func (h *TemplateHandler) GetTemplate(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)

	var template pipeline.PipelineTemplate
	if err := h.db.First(&template, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "模板不存在"})
		return
	}

	// 解析配置
	var config map[string]any
	json.Unmarshal([]byte(template.ConfigJSON), &config)

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"template": template,
			"config":   config,
		},
	})
}

// CreateTemplate 创建模板
// @Summary 创建流水线模板
// @Tags 流水线模板
// @Param body body pipeline.PipelineTemplate true "模板信息"
// @Success 200 {object} gin.H
// @Router /pipeline/templates [post]
func (h *TemplateHandler) CreateTemplate(c *gin.Context) {
	var template pipeline.PipelineTemplate
	if err := c.ShouldBindJSON(&template); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误: " + err.Error()})
		return
	}

	if template.Name == "" || template.ConfigJSON == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "名称和配置不能为空"})
		return
	}

	// 验证 JSON 格式
	var config map[string]any
	if err := json.Unmarshal([]byte(template.ConfigJSON), &config); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "配置必须是有效的 JSON"})
		return
	}

	template.IsBuiltin = false
	template.CreatedBy = c.GetString("username")

	if err := h.db.Create(&template).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "创建失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": template, "message": "创建成功"})
}

// UpdateTemplate 更新模板
// @Summary 更新流水线模板
// @Tags 流水线模板
// @Param id path int true "模板ID"
// @Param body body pipeline.PipelineTemplate true "模板信息"
// @Success 200 {object} gin.H
// @Router /pipeline/templates/{id} [put]
func (h *TemplateHandler) UpdateTemplate(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)

	var template pipeline.PipelineTemplate
	if err := h.db.First(&template, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "模板不存在"})
		return
	}

	if template.IsBuiltin {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "内置模板不允许修改"})
		return
	}

	var updates pipeline.PipelineTemplate
	if err := c.ShouldBindJSON(&updates); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误: " + err.Error()})
		return
	}

	if updates.ConfigJSON != "" {
		var config map[string]any
		if err := json.Unmarshal([]byte(updates.ConfigJSON), &config); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "配置必须是有效的 JSON"})
			return
		}
	}

	if err := h.db.Model(&template).Updates(map[string]any{
		"name":        updates.Name,
		"description": updates.Description,
		"category":    updates.Category,
		"language":    updates.Language,
		"framework":   updates.Framework,
		"config_json": updates.ConfigJSON,
		"is_public":   updates.IsPublic,
	}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "更新失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "更新成功"})
}

// DeleteTemplate 删除模板
// @Summary 删除流水线模板
// @Tags 流水线模板
// @Param id path int true "模板ID"
// @Success 200 {object} gin.H
// @Router /pipeline/templates/{id} [delete]
func (h *TemplateHandler) DeleteTemplate(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)

	var template pipeline.PipelineTemplate
	if err := h.db.First(&template, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "模板不存在"})
		return
	}

	if template.IsBuiltin {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "内置模板不允许删除"})
		return
	}

	if err := h.db.Delete(&template).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "删除失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "删除成功"})
}

// ApplyTemplate 应用模板创建流水线
// @Summary 应用模板创建流水线
// @Tags 流水线模板
// @Param id path int true "模板ID"
// @Param body body ApplyTemplateRequest true "应用请求"
// @Success 200 {object} gin.H
// @Router /pipeline/templates/{id}/apply [post]
func (h *TemplateHandler) ApplyTemplate(c *gin.Context) {
	templateID, _ := strconv.ParseUint(c.Param("id"), 10, 64)

	var req ApplyTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误: " + err.Error()})
		return
	}

	// 获取模板
	var template pipeline.PipelineTemplate
	if err := h.db.First(&template, templateID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "模板不存在"})
		return
	}

	// 创建流水线
	userID := c.GetUint("user_id")
	newPipeline := &models.Pipeline{
		Name:           req.Name,
		Description:    req.Description,
		ProjectID:      req.ProjectID,
		GitRepoID:      req.GitRepoID,
		GitBranch:      req.GitBranch,
		BuildClusterID: req.BuildClusterID,
		BuildNamespace: req.BuildNamespace,
		ConfigJSON:     template.ConfigJSON,
		Status:         "active",
		CreatedBy:      &userID,
	}

	if err := h.db.Create(newPipeline).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "创建流水线失败: " + err.Error()})
		return
	}

	// 更新模板使用次数
	h.db.Model(&template).UpdateColumn("usage_count", template.UsageCount+1)

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": newPipeline, "message": "流水线创建成功"})
}

// RateTemplate 评价模板
// @Summary 评价流水线模板
// @Tags 流水线模板
// @Param id path int true "模板ID"
// @Param body body RateTemplateRequest true "评价请求"
// @Success 200 {object} gin.H
// @Router /pipeline/templates/{id}/rate [post]
func (h *TemplateHandler) RateTemplate(c *gin.Context) {
	templateID, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	userID := c.GetUint("user_id")

	var req RateTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误: " + err.Error()})
		return
	}

	if req.Rating < 1 || req.Rating > 5 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "评分必须在 1-5 之间"})
		return
	}

	// 检查是否已评价
	var existing pipeline.PipelineTemplateRating
	if h.db.Where("template_id = ? AND user_id = ?", templateID, userID).First(&existing).Error == nil {
		// 更新评价
		h.db.Model(&existing).Updates(map[string]any{
			"rating":  req.Rating,
			"comment": req.Comment,
		})
	} else {
		// 创建评价
		rating := &pipeline.PipelineTemplateRating{
			TemplateID: templateID,
			UserID:     userID,
			Rating:     req.Rating,
			Comment:    req.Comment,
		}
		h.db.Create(rating)
	}

	// 更新模板平均评分
	var avgRating float64
	var count int64
	h.db.Model(&pipeline.PipelineTemplateRating{}).
		Where("template_id = ?", templateID).
		Select("AVG(rating)").Scan(&avgRating)
	h.db.Model(&pipeline.PipelineTemplateRating{}).
		Where("template_id = ?", templateID).Count(&count)

	h.db.Model(&pipeline.PipelineTemplate{}).Where("id = ?", templateID).
		Updates(map[string]any{
			"rating":       avgRating,
			"rating_count": count,
		})

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "评价成功"})
}

// ListStageTemplates 获取阶段模板列表
// @Summary 获取阶段模板列表
// @Tags 流水线模板
// @Success 200 {object} gin.H
// @Router /pipeline/templates/stages [get]
func (h *TemplateHandler) ListStageTemplates(c *gin.Context) {
	var templates []pipeline.PipelineStageTemplate
	h.db.Order("sort_order ASC").Find(&templates)

	// 如果没有模板，初始化内置模板
	if len(templates) == 0 {
		h.initBuiltinStageTemplates()
		h.db.Order("sort_order ASC").Find(&templates)
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"items": templates}})
}

// ListStepTemplates 获取步骤模板列表
// @Summary 获取步骤模板列表
// @Tags 流水线模板
// @Param category query string false "分类"
// @Success 200 {object} gin.H
// @Router /pipeline/templates/steps [get]
func (h *TemplateHandler) ListStepTemplates(c *gin.Context) {
	category := c.Query("category")

	query := h.db.Model(&pipeline.PipelineStepTemplate{})
	if category != "" {
		query = query.Where("category = ?", category)
	}

	var templates []pipeline.PipelineStepTemplate
	query.Order("sort_order ASC").Find(&templates)

	// 如果没有模板，初始化内置模板
	if len(templates) == 0 {
		h.initBuiltinStepTemplates()
		query.Order("sort_order ASC").Find(&templates)
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"items": templates}})
}

// initBuiltinTemplates 初始化内置流水线模板
func (h *TemplateHandler) initBuiltinTemplates() {
	for _, t := range pipeline.BuiltinPipelineTemplates {
		var existing pipeline.PipelineTemplate
		if h.db.Where("name = ?", t.Name).First(&existing).Error == nil {
			continue
		}
		h.db.Create(&t)
	}
}

// initBuiltinStageTemplates 初始化内置阶段模板
func (h *TemplateHandler) initBuiltinStageTemplates() {
	for _, t := range pipeline.BuiltinStageTemplates {
		var existing pipeline.PipelineStageTemplate
		if h.db.Where("name = ?", t.Name).First(&existing).Error == nil {
			continue
		}
		h.db.Create(&t)
	}
}

// initBuiltinStepTemplates 初始化内置步骤模板
func (h *TemplateHandler) initBuiltinStepTemplates() {
	for _, t := range pipeline.BuiltinStepTemplates {
		var existing pipeline.PipelineStepTemplate
		if h.db.Where("name = ?", t.Name).First(&existing).Error == nil {
			continue
		}
		h.db.Create(&t)
	}
}

// ApplyTemplateRequest 应用模板请求
type ApplyTemplateRequest struct {
	Name           string `json:"name" binding:"required"`
	Description    string `json:"description"`
	ProjectID      *uint  `json:"project_id"`
	GitRepoID      *uint  `json:"git_repo_id"`
	GitBranch      string `json:"git_branch"`
	BuildClusterID *uint  `json:"build_cluster_id"`
	BuildNamespace string `json:"build_namespace"`
}

// RateTemplateRequest 评价模板请求
type RateTemplateRequest struct {
	Rating  int    `json:"rating" binding:"required"`
	Comment string `json:"comment"`
}

// GetCategories 获取模板分类列表
// @Summary 获取模板分类列表
// @Tags 流水线模板
// @Success 200 {object} gin.H
// @Router /pipeline/templates/categories [get]
func (h *TemplateHandler) GetCategories(c *gin.Context) {
	// 定义所有可用的分类
	categories := []map[string]interface{}{
		{
			"value":       "build",
			"label":       "构建",
			"description": "代码构建和编译",
			"icon":        "build",
		},
		{
			"value":       "deploy",
			"label":       "部署",
			"description": "应用部署到各种环境",
			"icon":        "deploy",
		},
		{
			"value":       "test",
			"label":       "测试",
			"description": "自动化测试和质量检查",
			"icon":        "test",
		},
		{
			"value":       "release",
			"label":       "发布",
			"description": "版本发布和交付",
			"icon":        "release",
		},
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"items": categories,
		},
	})
}

// GetTags 获取模板标签列表
// @Summary 获取模板标签列表
// @Tags 流水线模板
// @Success 200 {object} gin.H
// @Router /pipeline/templates/tags [get]
func (h *TemplateHandler) GetTags(c *gin.Context) {
	// 从数据库中获取所有使用的语言和框架作为标签
	var languages []string
	h.db.Model(&pipeline.PipelineTemplate{}).
		Where("language IS NOT NULL AND language != ''").
		Distinct("language").
		Pluck("language", &languages)

	var frameworks []string
	h.db.Model(&pipeline.PipelineTemplate{}).
		Where("framework IS NOT NULL AND framework != ''").
		Distinct("framework").
		Pluck("framework", &frameworks)

	// 组合标签
	tags := []map[string]interface{}{}

	// 添加语言标签
	for _, lang := range languages {
		tags = append(tags, map[string]interface{}{
			"value": lang,
			"label": lang,
			"type":  "language",
		})
	}

	// 添加框架标签
	for _, fw := range frameworks {
		tags = append(tags, map[string]interface{}{
			"value": fw,
			"label": fw,
			"type":  "framework",
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"items": tags,
		},
	})
}

// GetFavorites 获取收藏的模板列表
// @Summary 获取收藏的模板列表
// @Tags 流水线模板
// @Success 200 {object} gin.H
// @Router /pipeline/templates/favorites [get]
func (h *TemplateHandler) GetFavorites(c *gin.Context) {
	// TODO: 实现用户收藏功能，需要用户认证和收藏表
	// 目前返回空列表

	// 从请求头或上下文中获取用户ID
	// userID := c.GetString("user_id")

	// 如果有用户收藏表，可以这样查询：
	// var favorites []pipeline.PipelineTemplate
	// h.db.Joins("JOIN user_template_favorites ON user_template_favorites.template_id = pipeline_templates.id").
	//     Where("user_template_favorites.user_id = ?", userID).
	//     Find(&favorites)

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"items": []interface{}{},
		},
		"message": "收藏功能待实现",
	})
}
