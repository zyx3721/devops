// Package handler 流水线模块处理器
// 本文件实现流水线可视化编排功能
package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"devops/internal/models"
	"devops/pkg/dto"
)

// DesignerHandler 流水线设计器处理器
type DesignerHandler struct {
	db *gorm.DB
}

// NewDesignerHandler 创建设计器处理器
func NewDesignerHandler(db *gorm.DB) *DesignerHandler {
	return &DesignerHandler{db: db}
}

// RegisterRoutes 注册路由
func (h *DesignerHandler) RegisterRoutes(r *gin.RouterGroup) {
	g := r.Group("/pipeline/designer")
	{
		// 获取流水线设计数据
		g.GET("/:id", h.GetDesign)
		// 保存流水线设计
		g.PUT("/:id", h.SaveDesign)
		// 验证流水线设计
		g.POST("/:id/validate", h.ValidateDesign)
		// 预览流水线 YAML
		g.GET("/:id/preview", h.PreviewYAML)
		// 导出流水线配置
		g.GET("/:id/export", h.ExportConfig)
		// 导入流水线配置
		g.POST("/import", h.ImportConfig)
	}
}

// DesignData 设计数据
type DesignData struct {
	Pipeline  *models.Pipeline   `json:"pipeline"`
	Stages    []dto.Stage        `json:"stages"`
	Variables []dto.Variable     `json:"variables"`
	Triggers  *dto.TriggerConfig `json:"triggers,omitempty"`
	Metadata  *DesignMetadata    `json:"metadata,omitempty"`
}

// DesignMetadata 设计元数据（用于可视化编排）
type DesignMetadata struct {
	CanvasWidth  int                 `json:"canvas_width"`
	CanvasHeight int                 `json:"canvas_height"`
	Zoom         float64             `json:"zoom"`
	Positions    map[string]Position `json:"positions"`   // 阶段/步骤位置
	Connections  []Connection        `json:"connections"` // 连接线
}

// Position 位置信息
type Position struct {
	X int `json:"x"`
	Y int `json:"y"`
}

// Connection 连接信息
type Connection struct {
	From string `json:"from"`
	To   string `json:"to"`
	Type string `json:"type"` // success, failure, always
}

// GetDesign 获取流水线设计数据
// @Summary 获取流水线设计数据
// @Tags 流水线设计器
// @Param id path int true "流水线ID"
// @Success 200 {object} gin.H
// @Router /pipeline/designer/{id} [get]
func (h *DesignerHandler) GetDesign(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)

	var pipeline models.Pipeline
	if err := h.db.First(&pipeline, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "流水线不存在"})
		return
	}

	design := &DesignData{
		Pipeline: &pipeline,
	}

	// 解析配置
	if pipeline.ConfigJSON != "" {
		var config struct {
			Stages    []dto.Stage     `json:"stages"`
			Variables []dto.Variable  `json:"variables"`
			Metadata  *DesignMetadata `json:"metadata,omitempty"`
		}
		if err := json.Unmarshal([]byte(pipeline.ConfigJSON), &config); err == nil {
			design.Stages = config.Stages
			design.Variables = config.Variables
			design.Metadata = config.Metadata
		}
	}

	// 解析触发器配置
	if pipeline.TriggerConfigJSON != "" {
		var triggers dto.TriggerConfig
		if err := json.Unmarshal([]byte(pipeline.TriggerConfigJSON), &triggers); err == nil {
			design.Triggers = &triggers
		}
	}

	// 如果没有元数据，生成默认布局
	if design.Metadata == nil {
		design.Metadata = h.generateDefaultLayout(design.Stages)
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": design})
}

// SaveDesign 保存流水线设计
// @Summary 保存流水线设计
// @Tags 流水线设计器
// @Param id path int true "流水线ID"
// @Param body body DesignData true "设计数据"
// @Success 200 {object} gin.H
// @Router /pipeline/designer/{id} [put]
func (h *DesignerHandler) SaveDesign(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)

	var pipeline models.Pipeline
	if err := h.db.First(&pipeline, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "流水线不存在"})
		return
	}

	var design DesignData
	if err := c.ShouldBindJSON(&design); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误: " + err.Error()})
		return
	}

	// 验证设计
	if err := h.validateDesign(&design); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "设计验证失败: " + err.Error()})
		return
	}

	// 序列化配置
	config := struct {
		Stages    []dto.Stage     `json:"stages"`
		Variables []dto.Variable  `json:"variables"`
		Metadata  *DesignMetadata `json:"metadata,omitempty"`
	}{
		Stages:    design.Stages,
		Variables: design.Variables,
		Metadata:  design.Metadata,
	}
	configJSON, err := json.Marshal(config)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "序列化配置失败"})
		return
	}

	// 序列化触发器配置
	var triggerJSON []byte
	if design.Triggers != nil {
		triggerJSON, _ = json.Marshal(design.Triggers)
	}

	// 更新流水线
	updates := map[string]any{
		"config_json": string(configJSON),
	}
	if len(triggerJSON) > 0 {
		updates["trigger_config_json"] = string(triggerJSON)
	}

	if err := h.db.Model(&pipeline).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "保存失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "保存成功"})
}

// ValidateDesign 验证流水线设计
// @Summary 验证流水线设计
// @Tags 流水线设计器
// @Param id path int true "流水线ID"
// @Param body body DesignData true "设计数据"
// @Success 200 {object} gin.H
// @Router /pipeline/designer/{id}/validate [post]
func (h *DesignerHandler) ValidateDesign(c *gin.Context) {
	var design DesignData
	if err := c.ShouldBindJSON(&design); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误: " + err.Error()})
		return
	}

	errors := h.validateDesignWithDetails(&design)
	if len(errors) > 0 {
		c.JSON(http.StatusOK, gin.H{
			"code":    0,
			"data":    gin.H{"valid": false, "errors": errors},
			"message": "验证失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"data":    gin.H{"valid": true, "errors": []string{}},
		"message": "验证通过",
	})
}

// PreviewYAML 预览流水线 YAML
// @Summary 预览流水线 YAML
// @Tags 流水线设计器
// @Param id path int true "流水线ID"
// @Success 200 {object} gin.H
// @Router /pipeline/designer/{id}/preview [get]
func (h *DesignerHandler) PreviewYAML(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)

	var pipeline models.Pipeline
	if err := h.db.First(&pipeline, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "流水线不存在"})
		return
	}

	// 解析配置
	var config struct {
		Stages    []dto.Stage    `json:"stages"`
		Variables []dto.Variable `json:"variables"`
	}
	if pipeline.ConfigJSON != "" {
		json.Unmarshal([]byte(pipeline.ConfigJSON), &config)
	}

	// 生成 YAML 预览
	yaml := h.generateYAML(&pipeline, config.Stages, config.Variables)

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"yaml": yaml}})
}

// ExportConfig 导出流水线配置
// @Summary 导出流水线配置
// @Tags 流水线设计器
// @Param id path int true "流水线ID"
// @Success 200 {object} gin.H
// @Router /pipeline/designer/{id}/export [get]
func (h *DesignerHandler) ExportConfig(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)

	var pipeline models.Pipeline
	if err := h.db.First(&pipeline, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "流水线不存在"})
		return
	}

	export := map[string]any{
		"name":           pipeline.Name,
		"description":    pipeline.Description,
		"config":         json.RawMessage(pipeline.ConfigJSON),
		"trigger_config": json.RawMessage(pipeline.TriggerConfigJSON),
		"version":        "1.0",
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": export})
}

// ImportConfig 导入流水线配置
// @Summary 导入流水线配置
// @Tags 流水线设计器
// @Param body body ImportConfigRequest true "导入请求"
// @Success 200 {object} gin.H
// @Router /pipeline/designer/import [post]
func (h *DesignerHandler) ImportConfig(c *gin.Context) {
	var req ImportConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误: " + err.Error()})
		return
	}

	// 验证配置
	var config map[string]any
	if err := json.Unmarshal([]byte(req.Config), &config); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "配置格式错误"})
		return
	}

	// 创建流水线
	userID := c.GetUint("user_id")
	pipeline := &models.Pipeline{
		Name:              req.Name,
		Description:       req.Description,
		ProjectID:         req.ProjectID,
		ConfigJSON:        req.Config,
		TriggerConfigJSON: req.TriggerConfig,
		Status:            "active",
		CreatedBy:         &userID,
	}

	if err := h.db.Create(pipeline).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "导入失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": pipeline, "message": "导入成功"})
}

// validateDesign 验证设计
func (h *DesignerHandler) validateDesign(design *DesignData) error {
	errors := h.validateDesignWithDetails(design)
	if len(errors) > 0 {
		return &ValidationError{Message: errors[0]}
	}
	return nil
}

// validateDesignWithDetails 验证设计并返回详细错误
func (h *DesignerHandler) validateDesignWithDetails(design *DesignData) []string {
	var errors []string

	if len(design.Stages) == 0 {
		errors = append(errors, "至少需要一个阶段")
		return errors
	}

	stageIDs := make(map[string]bool)
	for _, stage := range design.Stages {
		if stage.ID == "" {
			errors = append(errors, "阶段ID不能为空")
			continue
		}
		if stageIDs[stage.ID] {
			errors = append(errors, "阶段ID重复: "+stage.ID)
			continue
		}
		stageIDs[stage.ID] = true

		if stage.Name == "" {
			errors = append(errors, "阶段名称不能为空: "+stage.ID)
		}

		// 检查依赖
		for _, dep := range stage.DependsOn {
			if !stageIDs[dep] {
				errors = append(errors, "依赖的阶段不存在: "+dep)
			}
		}

		// 检查步骤
		if len(stage.Steps) == 0 {
			errors = append(errors, "阶段至少需要一个步骤: "+stage.Name)
		}

		stepIDs := make(map[string]bool)
		for _, step := range stage.Steps {
			if step.ID == "" {
				errors = append(errors, "步骤ID不能为空")
				continue
			}
			if stepIDs[step.ID] {
				errors = append(errors, "步骤ID重复: "+step.ID)
				continue
			}
			stepIDs[step.ID] = true

			if step.Type == "" {
				errors = append(errors, "步骤类型不能为空: "+step.ID)
			}
		}
	}

	// 检查循环依赖
	if h.hasCyclicDependency(design.Stages) {
		errors = append(errors, "存在循环依赖")
	}

	return errors
}

// hasCyclicDependency 检查循环依赖
func (h *DesignerHandler) hasCyclicDependency(stages []dto.Stage) bool {
	// 构建依赖图
	graph := make(map[string][]string)
	for _, stage := range stages {
		graph[stage.ID] = stage.DependsOn
	}

	// DFS 检测环
	visited := make(map[string]int) // 0: 未访问, 1: 访问中, 2: 已完成
	var hasCycle bool

	var dfs func(node string)
	dfs = func(node string) {
		if hasCycle {
			return
		}
		if visited[node] == 1 {
			hasCycle = true
			return
		}
		if visited[node] == 2 {
			return
		}

		visited[node] = 1
		for _, dep := range graph[node] {
			dfs(dep)
		}
		visited[node] = 2
	}

	for _, stage := range stages {
		if visited[stage.ID] == 0 {
			dfs(stage.ID)
		}
	}

	return hasCycle
}

// generateDefaultLayout 生成默认布局
func (h *DesignerHandler) generateDefaultLayout(stages []dto.Stage) *DesignMetadata {
	metadata := &DesignMetadata{
		CanvasWidth:  1200,
		CanvasHeight: 800,
		Zoom:         1.0,
		Positions:    make(map[string]Position),
		Connections:  make([]Connection, 0),
	}

	// 简单的水平布局
	x := 100
	y := 100
	stageWidth := 200
	stageGap := 50

	for i, stage := range stages {
		metadata.Positions[stage.ID] = Position{X: x + i*(stageWidth+stageGap), Y: y}

		// 添加连接
		if i > 0 {
			metadata.Connections = append(metadata.Connections, Connection{
				From: stages[i-1].ID,
				To:   stage.ID,
				Type: "success",
			})
		}
	}

	return metadata
}

// generateYAML 生成 YAML 预览
func (h *DesignerHandler) generateYAML(pipeline *models.Pipeline, stages []dto.Stage, variables []dto.Variable) string {
	yaml := "# 流水线: " + pipeline.Name + "\n"
	yaml += "# 描述: " + pipeline.Description + "\n\n"

	// 变量
	if len(variables) > 0 {
		yaml += "variables:\n"
		for _, v := range variables {
			yaml += "  " + v.Name + ": " + v.Value + "\n"
		}
		yaml += "\n"
	}

	// 阶段
	yaml += "stages:\n"
	for _, stage := range stages {
		yaml += "  - name: " + stage.Name + "\n"
		yaml += "    id: " + stage.ID + "\n"
		if len(stage.DependsOn) > 0 {
			yaml += "    depends_on:\n"
			for _, dep := range stage.DependsOn {
				yaml += "      - " + dep + "\n"
			}
		}
		yaml += "    steps:\n"
		for _, step := range stage.Steps {
			yaml += "      - name: " + step.Name + "\n"
			yaml += "        type: " + step.Type + "\n"
			if step.Timeout > 0 {
				yaml += "        timeout: " + strconv.Itoa(step.Timeout) + "s\n"
			}
		}
		yaml += "\n"
	}

	return yaml
}

// ImportConfigRequest 导入配置请求
type ImportConfigRequest struct {
	Name          string `json:"name" binding:"required"`
	Description   string `json:"description"`
	ProjectID     *uint  `json:"project_id"`
	Config        string `json:"config" binding:"required"`
	TriggerConfig string `json:"trigger_config"`
}

// ValidationError 验证错误
type ValidationError struct {
	Message string
}

func (e *ValidationError) Error() string {
	return e.Message
}
