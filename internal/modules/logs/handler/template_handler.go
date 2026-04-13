package handler

import (
	"net/http"
	"strconv"

	"devops/internal/models"
	"devops/internal/service/logs"
	"devops/pkg/dto"
	"devops/pkg/response"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// TemplateHandler 解析模板处理器
type TemplateHandler struct {
	db            *gorm.DB
	parserService *logs.ParserService
}

// NewTemplateHandler 创建解析模板处理器
func NewTemplateHandler(db *gorm.DB, parserService *logs.ParserService) *TemplateHandler {
	return &TemplateHandler{
		db:            db,
		parserService: parserService,
	}
}

// ListTemplates 获取解析模板列表
// @Summary 获取解析模板列表
// @Tags 日志解析
// @Param include_preset query bool false "是否包含预设模板"
// @Success 200 {object} response.Response{data=[]dto.ParseTemplateResponse}
// @Router /api/v1/logs/parse-templates [get]
func (h *TemplateHandler) ListTemplates(c *gin.Context) {
	userID := c.GetInt64("user_id")
	includePreset := c.Query("include_preset") != "false"

	var templates []models.LogParseTemplate
	query := h.db.Where("created_by = ?", userID)
	if includePreset {
		query = query.Or("is_preset = ?", true)
	}

	if err := query.Order("is_preset DESC, created_at DESC").Find(&templates).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	result := make([]dto.ParseTemplateResponse, len(templates))
	for i, t := range templates {
		result[i] = templateToResponse(&t)
	}

	response.Success(c, result)
}

// GetPresetTemplates 获取预设模板
// @Summary 获取预设模板
// @Tags 日志解析
// @Success 200 {object} response.Response{data=[]dto.ParseTemplateResponse}
// @Router /api/v1/logs/parse-templates/presets [get]
func (h *TemplateHandler) GetPresetTemplates(c *gin.Context) {
	var templates []models.LogParseTemplate
	if err := h.db.Where("is_preset = ?", true).Find(&templates).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	result := make([]dto.ParseTemplateResponse, len(templates))
	for i, t := range templates {
		result[i] = templateToResponse(&t)
	}

	response.Success(c, result)
}

// CreateTemplate 创建解析模板
// @Summary 创建解析模板
// @Tags 日志解析
// @Accept json
// @Produce json
// @Param body body dto.ParseTemplateRequest true "解析模板"
// @Success 200 {object} response.Response{data=dto.ParseTemplateResponse}
// @Router /api/v1/logs/parse-templates [post]
func (h *TemplateHandler) CreateTemplate(c *gin.Context) {
	var req dto.ParseTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	userID := c.GetInt64("user_id")

	template := &models.LogParseTemplate{
		Name:        req.Name,
		Description: req.Description,
		Type:        req.Type,
		Pattern:     req.Pattern,
		IsPreset:    false,
		Enabled:     req.Enabled,
		CreatedBy:   userID,
	}

	if err := h.db.Create(template).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, templateToResponse(template))
}

// UpdateTemplate 更新解析模板
// @Summary 更新解析模板
// @Tags 日志解析
// @Accept json
// @Produce json
// @Param id path int true "模板ID"
// @Param body body dto.ParseTemplateRequest true "解析模板"
// @Success 200 {object} response.Response{data=dto.ParseTemplateResponse}
// @Router /api/v1/logs/parse-templates/{id} [put]
func (h *TemplateHandler) UpdateTemplate(c *gin.Context) {
	templateID, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	userID := c.GetInt64("user_id")

	var template models.LogParseTemplate
	if err := h.db.First(&template, templateID).Error; err != nil {
		response.Error(c, http.StatusNotFound, "模板不存在")
		return
	}

	// 不能修改预设模板
	if template.IsPreset {
		response.Error(c, http.StatusForbidden, "不能修改预设模板")
		return
	}

	// 只能修改自己的模板
	if template.CreatedBy != userID {
		response.Error(c, http.StatusForbidden, "无权修改此模板")
		return
	}

	var req dto.ParseTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	template.Name = req.Name
	template.Description = req.Description
	template.Type = req.Type
	template.Pattern = req.Pattern
	template.Enabled = req.Enabled

	if err := h.db.Save(&template).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, templateToResponse(&template))
}

// DeleteTemplate 删除解析模板
// @Summary 删除解析模板
// @Tags 日志解析
// @Param id path int true "模板ID"
// @Success 200 {object} response.Response
// @Router /api/v1/logs/parse-templates/{id} [delete]
func (h *TemplateHandler) DeleteTemplate(c *gin.Context) {
	templateID, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	userID := c.GetInt64("user_id")

	var template models.LogParseTemplate
	if err := h.db.First(&template, templateID).Error; err != nil {
		response.Error(c, http.StatusNotFound, "模板不存在")
		return
	}

	// 不能删除预设模板
	if template.IsPreset {
		response.Error(c, http.StatusForbidden, "不能删除预设模板")
		return
	}

	// 只能删除自己的模板
	if template.CreatedBy != userID {
		response.Error(c, http.StatusForbidden, "无权删除此模板")
		return
	}

	if err := h.db.Delete(&template).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, nil)
}

// TestTemplate 测试解析模板
// @Summary 测试解析模板
// @Tags 日志解析
// @Accept json
// @Produce json
// @Param body body dto.ParseTestRequest true "测试请求"
// @Success 200 {object} response.Response{data=dto.ParseTestResponse}
// @Router /api/v1/logs/parse-templates/test [post]
func (h *TemplateHandler) TestTemplate(c *gin.Context) {
	var req dto.ParseTestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	resp := h.parserService.TestTemplate(&req)
	response.Success(c, resp)
}

func templateToResponse(t *models.LogParseTemplate) dto.ParseTemplateResponse {
	return dto.ParseTemplateResponse{
		ID:          t.ID,
		Name:        t.Name,
		Description: t.Description,
		Type:        t.Type,
		Pattern:     t.Pattern,
		IsPreset:    t.IsPreset,
		Enabled:     t.Enabled,
		CreatedBy:   t.CreatedBy,
		CreatedAt:   t.CreatedAt,
	}
}
