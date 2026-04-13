package handler

import (
	"encoding/json"
	"net/http"
	"slices"
	"strconv"

	"github.com/gin-gonic/gin"

	"devops/internal/models"
	"devops/internal/models/traffic"
)

// RegisterTemplateRoutes 注册模板路由
func (h *TrafficHandler) RegisterTemplateRoutes(r *gin.RouterGroup) {
	g := r.Group("/traffic/templates")
	{
		g.GET("", h.ListTemplates)
		g.GET("/:id", h.GetTemplate)
		g.POST("", h.CreateTemplate)
		g.PUT("/:id", h.UpdateTemplate)
		g.DELETE("/:id", h.DeleteTemplate)
		g.POST("/:id/apply/:appId", h.ApplyTemplate)
		g.POST("/export", h.ExportRules)
		g.POST("/import", h.ImportRules)
	}
}

// ListTemplates 获取模板列表
// @Summary 获取流量治理规则模板列表
// @Tags 流量治理模板
// @Param category query string false "模板分类"
// @Param rule_type query string false "规则类型"
// @Success 200 {object} gin.H
// @Router /traffic/templates [get]
func (h *TrafficHandler) ListTemplates(c *gin.Context) {
	category := c.Query("category")
	ruleType := c.Query("rule_type")

	query := h.db.Model(&models.TrafficRuleTemplate{})
	if category != "" {
		query = query.Where("category = ?", category)
	}
	if ruleType != "" {
		query = query.Where("rule_type = ?", ruleType)
	}

	var templates []models.TrafficRuleTemplate
	query.Order("is_builtin DESC, usage_count DESC, created_at DESC").Find(&templates)

	// 如果没有模板，初始化内置模板
	if len(templates) == 0 {
		h.initBuiltinTemplates()
		query.Find(&templates)
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"items": templates}})
}

// GetTemplate 获取模板详情
// @Summary 获取模板详情
// @Tags 流量治理模板
// @Param id path int true "模板ID"
// @Success 200 {object} gin.H
// @Router /traffic/templates/{id} [get]
func (h *TrafficHandler) GetTemplate(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)

	var template models.TrafficRuleTemplate
	if err := h.db.First(&template, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "模板不存在"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": template})
}

// CreateTemplate 创建模板
// @Summary 创建流量治理规则模板
// @Tags 流量治理模板
// @Param body body models.TrafficRuleTemplate true "模板信息"
// @Success 200 {object} gin.H
// @Router /traffic/templates [post]
func (h *TrafficHandler) CreateTemplate(c *gin.Context) {
	var template models.TrafficRuleTemplate
	if err := c.ShouldBindJSON(&template); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误: " + err.Error()})
		return
	}

	// 验证必填字段
	if template.Name == "" || template.RuleType == "" || template.Content == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "名称、规则类型和内容不能为空"})
		return
	}

	// 验证 JSON 格式
	var contentMap map[string]any
	if err := json.Unmarshal([]byte(template.Content), &contentMap); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "规则内容必须是有效的 JSON"})
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
// @Summary 更新流量治理规则模板
// @Tags 流量治理模板
// @Param id path int true "模板ID"
// @Param body body models.TrafficRuleTemplate true "模板信息"
// @Success 200 {object} gin.H
// @Router /traffic/templates/{id} [put]
func (h *TrafficHandler) UpdateTemplate(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)

	var template models.TrafficRuleTemplate
	if err := h.db.First(&template, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "模板不存在"})
		return
	}

	// 内置模板不允许修改
	if template.IsBuiltin {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "内置模板不允许修改"})
		return
	}

	var updates models.TrafficRuleTemplate
	if err := c.ShouldBindJSON(&updates); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误: " + err.Error()})
		return
	}

	// 验证 JSON 格式
	if updates.Content != "" {
		var contentMap map[string]any
		if err := json.Unmarshal([]byte(updates.Content), &contentMap); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "规则内容必须是有效的 JSON"})
			return
		}
	}

	if err := h.db.Model(&template).Updates(map[string]any{
		"name":        updates.Name,
		"description": updates.Description,
		"category":    updates.Category,
		"rule_type":   updates.RuleType,
		"content":     updates.Content,
		"is_public":   updates.IsPublic,
	}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "更新失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "更新成功"})
}

// DeleteTemplate 删除模板
// @Summary 删除流量治理规则模板
// @Tags 流量治理模板
// @Param id path int true "模板ID"
// @Success 200 {object} gin.H
// @Router /traffic/templates/{id} [delete]
func (h *TrafficHandler) DeleteTemplate(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)

	var template models.TrafficRuleTemplate
	if err := h.db.First(&template, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "模板不存在"})
		return
	}

	// 内置模板不允许删除
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

// ApplyTemplate 应用模板到应用
// @Summary 将模板应用到指定应用
// @Tags 流量治理模板
// @Param id path int true "模板ID"
// @Param appId path int true "应用ID"
// @Success 200 {object} gin.H
// @Router /traffic/templates/{id}/apply/{appId} [post]
func (h *TrafficHandler) ApplyTemplate(c *gin.Context) {
	templateID, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	appID, _ := strconv.ParseUint(c.Param("appId"), 10, 64)

	// 获取模板
	var template models.TrafficRuleTemplate
	if err := h.db.First(&template, templateID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "模板不存在"})
		return
	}

	// 获取应用
	var app models.Application
	if err := h.db.First(&app, appID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "应用不存在"})
		return
	}

	// 根据规则类型创建对应的规则
	var err error
	switch template.RuleType {
	case "ratelimit":
		err = h.applyRateLimitTemplate(&app, &template)
	case "circuitbreaker":
		err = h.applyCircuitBreakerTemplate(&app, &template)
	case "routing":
		err = h.applyRoutingTemplate(&app, &template)
	case "loadbalance":
		err = h.applyLoadBalanceTemplate(&app, &template)
	case "timeout":
		err = h.applyTimeoutTemplate(&app, &template)
	case "mirror":
		err = h.applyMirrorTemplate(&app, &template)
	case "fault":
		err = h.applyFaultTemplate(&app, &template)
	default:
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "不支持的规则类型"})
		return
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "应用模板失败: " + err.Error()})
		return
	}

	// 更新使用次数
	h.db.Model(&template).UpdateColumn("usage_count", template.UsageCount+1)

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "模板应用成功"})
}

// ExportRules 导出规则
// @Summary 导出应用的流量治理规则
// @Tags 流量治理模板
// @Param body body ExportRequest true "导出请求"
// @Success 200 {object} gin.H
// @Router /traffic/templates/export [post]
func (h *TrafficHandler) ExportRules(c *gin.Context) {
	var req ExportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	result := make(map[string]any)

	// 导出限流规则
	if slices.Contains(req.RuleTypes, "ratelimit") || len(req.RuleTypes) == 0 {
		var rules []models.TrafficRateLimitRule
		h.db.Where("app_id = ?", req.AppID).Find(&rules)
		result["ratelimit"] = rules
	}

	// 导出熔断规则
	if slices.Contains(req.RuleTypes, "circuitbreaker") || len(req.RuleTypes) == 0 {
		var rules []models.TrafficCircuitBreakerRule
		h.db.Where("app_id = ?", req.AppID).Find(&rules)
		result["circuitbreaker"] = rules
	}

	// 导出路由规则
	if slices.Contains(req.RuleTypes, "routing") || len(req.RuleTypes) == 0 {
		var rules []models.TrafficRoutingRule
		h.db.Where("app_id = ?", req.AppID).Find(&rules)
		result["routing"] = rules
	}

	// 导出负载均衡配置
	if slices.Contains(req.RuleTypes, "loadbalance") || len(req.RuleTypes) == 0 {
		var config models.TrafficLoadBalanceConfig
		h.db.Where("app_id = ?", req.AppID).First(&config)
		if config.ID > 0 {
			result["loadbalance"] = config
		}
	}

	// 导出超时配置
	if slices.Contains(req.RuleTypes, "timeout") || len(req.RuleTypes) == 0 {
		var config models.TrafficTimeoutConfig
		h.db.Where("app_id = ?", req.AppID).First(&config)
		if config.ID > 0 {
			result["timeout"] = config
		}
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": result})
}

// ImportRules 导入规则
// @Summary 导入流量治理规则到应用
// @Tags 流量治理模板
// @Param body body ImportRequest true "导入请求"
// @Success 200 {object} gin.H
// @Router /traffic/templates/import [post]
func (h *TrafficHandler) ImportRules(c *gin.Context) {
	var req ImportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	// 获取应用
	var app models.Application
	if err := h.db.First(&app, req.AppID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "应用不存在"})
		return
	}

	imported := make(map[string]int)

	// 导入限流规则
	if rules, ok := req.Rules["ratelimit"].([]any); ok {
		for _, r := range rules {
			ruleJSON, _ := json.Marshal(r)
			var rule models.TrafficRateLimitRule
			json.Unmarshal(ruleJSON, &rule)
			rule.ID = 0
			rule.AppID = uint64(req.AppID)
			h.db.Create(&rule)
		}
		imported["ratelimit"] = len(rules)
	}

	// 导入熔断规则
	if rules, ok := req.Rules["circuitbreaker"].([]any); ok {
		for _, r := range rules {
			ruleJSON, _ := json.Marshal(r)
			var rule models.TrafficCircuitBreakerRule
			json.Unmarshal(ruleJSON, &rule)
			rule.ID = 0
			rule.AppID = uint64(req.AppID)
			h.db.Create(&rule)
		}
		imported["circuitbreaker"] = len(rules)
	}

	// 导入路由规则
	if rules, ok := req.Rules["routing"].([]any); ok {
		for _, r := range rules {
			ruleJSON, _ := json.Marshal(r)
			var rule models.TrafficRoutingRule
			json.Unmarshal(ruleJSON, &rule)
			rule.ID = 0
			rule.AppID = uint64(req.AppID)
			h.db.Create(&rule)
		}
		imported["routing"] = len(rules)
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": imported, "message": "导入成功"})
}

// ExportRequest 导出请求
type ExportRequest struct {
	AppID     uint     `json:"app_id" binding:"required"`
	RuleTypes []string `json:"rule_types"` // 为空则导出全部
}

// ImportRequest 导入请求
type ImportRequest struct {
	AppID uint           `json:"app_id" binding:"required"`
	Rules map[string]any `json:"rules" binding:"required"`
}

// initBuiltinTemplates 初始化内置模板
func (h *TrafficHandler) initBuiltinTemplates() {
	for _, t := range traffic.BuiltinTemplates {
		var existing models.TrafficRuleTemplate
		if h.db.Where("name = ?", t.Name).First(&existing).Error == nil {
			continue // 已存在
		}
		h.db.Create(&t)
	}
}

// applyRateLimitTemplate 应用限流模板
func (h *TrafficHandler) applyRateLimitTemplate(app *models.Application, template *models.TrafficRuleTemplate) error {
	var rule models.TrafficRateLimitRule
	if err := json.Unmarshal([]byte(template.Content), &rule); err != nil {
		return err
	}
	rule.AppID = uint64(app.ID)
	rule.Name = template.Name + " (从模板创建)"
	rule.Enabled = true
	return h.db.Create(&rule).Error
}

// applyCircuitBreakerTemplate 应用熔断模板
func (h *TrafficHandler) applyCircuitBreakerTemplate(app *models.Application, template *models.TrafficRuleTemplate) error {
	var rule models.TrafficCircuitBreakerRule
	if err := json.Unmarshal([]byte(template.Content), &rule); err != nil {
		return err
	}
	rule.AppID = uint64(app.ID)
	rule.Name = template.Name + " (从模板创建)"
	rule.Enabled = true
	return h.db.Create(&rule).Error
}

// applyRoutingTemplate 应用路由模板
func (h *TrafficHandler) applyRoutingTemplate(app *models.Application, template *models.TrafficRuleTemplate) error {
	var rule models.TrafficRoutingRule
	if err := json.Unmarshal([]byte(template.Content), &rule); err != nil {
		return err
	}
	rule.AppID = uint64(app.ID)
	rule.Name = template.Name + " (从模板创建)"
	rule.Enabled = true
	return h.db.Create(&rule).Error
}

// applyLoadBalanceTemplate 应用负载均衡模板
func (h *TrafficHandler) applyLoadBalanceTemplate(app *models.Application, template *models.TrafficRuleTemplate) error {
	var config models.TrafficLoadBalanceConfig
	if err := json.Unmarshal([]byte(template.Content), &config); err != nil {
		return err
	}
	config.AppID = uint64(app.ID)

	// 检查是否已存在配置
	var existing models.TrafficLoadBalanceConfig
	if h.db.Where("app_id = ?", app.ID).First(&existing).Error == nil {
		// 更新现有配置
		return h.db.Model(&existing).Updates(&config).Error
	}
	return h.db.Create(&config).Error
}

// applyTimeoutTemplate 应用超时模板
func (h *TrafficHandler) applyTimeoutTemplate(app *models.Application, template *models.TrafficRuleTemplate) error {
	var config models.TrafficTimeoutConfig
	if err := json.Unmarshal([]byte(template.Content), &config); err != nil {
		return err
	}
	config.AppID = uint64(app.ID)

	// 检查是否已存在配置
	var existing models.TrafficTimeoutConfig
	if h.db.Where("app_id = ?", app.ID).First(&existing).Error == nil {
		// 更新现有配置
		return h.db.Model(&existing).Updates(&config).Error
	}
	return h.db.Create(&config).Error
}

// applyMirrorTemplate 应用镜像模板
func (h *TrafficHandler) applyMirrorTemplate(app *models.Application, template *models.TrafficRuleTemplate) error {
	var rule models.TrafficMirrorRule
	if err := json.Unmarshal([]byte(template.Content), &rule); err != nil {
		return err
	}
	rule.AppID = uint64(app.ID)
	rule.Enabled = true
	return h.db.Create(&rule).Error
}

// applyFaultTemplate 应用故障注入模板
func (h *TrafficHandler) applyFaultTemplate(app *models.Application, template *models.TrafficRuleTemplate) error {
	var rule models.TrafficFaultRule
	if err := json.Unmarshal([]byte(template.Content), &rule); err != nil {
		return err
	}
	rule.AppID = uint64(app.ID)
	// 故障注入默认禁用，需要手动启用
	rule.Enabled = false
	return h.db.Create(&rule).Error
}
