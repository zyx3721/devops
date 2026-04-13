package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"devops/internal/config"
	"devops/internal/models/system"
	"devops/internal/repository"
	"devops/internal/service/notification"
	apperrors "devops/pkg/errors"
	"devops/pkg/ioc"
	"devops/pkg/middleware"
)

func init() {
	ioc.Api.RegisterContainer("TemplateHandler", &TemplateApiHandler{})
}

type TemplateApiHandler struct {
	handler *TemplateHandler
}

func (h *TemplateApiHandler) Init() error {
	cfg, err := config.LoadConfig()
	if err != nil {
		return err
	}

	db := repository.GetDB(context.Background())
	h.handler = NewTemplateHandler(db)

	root := cfg.Application.GinRootRouter().Group("notification/templates")
	root.Use(middleware.AuthMiddleware())
	h.Register(root)

	return nil
}

func (h *TemplateApiHandler) Register(r gin.IRouter) {
	r.GET("", h.handler.List)
	r.POST("", h.handler.Create)
	r.GET("/:id", h.handler.Get)
	r.PUT("/:id", h.handler.Update)
	r.DELETE("/:id", h.handler.Delete)
	r.POST("/preview", h.handler.Preview)
	r.POST("/send", h.handler.SendToWebhook)
}

type TemplateHandler struct {
	repo *repository.MessageTemplateRepository
	svc  *notification.TemplateService
}

func NewTemplateHandler(db *gorm.DB) *TemplateHandler {
	repo := repository.NewMessageTemplateRepository(db)
	return &TemplateHandler{
		repo: repo,
		svc:  notification.NewTemplateService(repo),
	}
}

// List 获取模板列表
func (h *TemplateHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	keyword := c.Query("keyword")

	list, total, err := h.repo.List(c.Request.Context(), page, pageSize, keyword)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    apperrors.Success,
		"message": "success",
		"data": gin.H{
			"list":  list,
			"total": total,
		},
	})
}

// Create 创建模板
func (h *TemplateHandler) Create(c *gin.Context) {
	var req system.MessageTemplate
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": apperrors.ErrCodeInvalidParams, "message": err.Error()})
		return
	}

	if err := h.repo.Create(c.Request.Context(), &req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": apperrors.Success, "message": "success", "data": req})
}

// Get 获取单个模板
func (h *TemplateHandler) Get(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": apperrors.ErrCodeInvalidParams, "message": "Invalid ID"})
		return
	}

	template, err := h.repo.GetByID(c.Request.Context(), uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": apperrors.ErrCodeNotFound, "message": "Template not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": apperrors.Success, "message": "success", "data": template})
}

// Update 更新模板
func (h *TemplateHandler) Update(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": apperrors.ErrCodeInvalidParams, "message": "Invalid ID"})
		return
	}

	var req system.MessageTemplate
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": apperrors.ErrCodeInvalidParams, "message": err.Error()})
		return
	}

	req.ID = uint(id)
	if err := h.repo.Update(c.Request.Context(), &req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": apperrors.Success, "message": "success", "data": req})
}

// Delete 删除模板
func (h *TemplateHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": apperrors.ErrCodeInvalidParams, "message": "Invalid ID"})
		return
	}

	if err := h.repo.Delete(c.Request.Context(), uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": apperrors.Success, "message": "success"})
}

// PreviewRequest 预览请求
type PreviewRequest struct {
	TemplateName string                 `json:"template_name"` // 优先使用模板名称查找
	TemplateID   uint                   `json:"template_id"`   // 或者指定ID
	Content      string                 `json:"content"`       // 或者直接提供模板内容（用于调试未保存的模板）
	Data         map[string]interface{} `json:"data"`          // 渲染数据
}

// Preview 预览模板渲染结果
func (h *TemplateHandler) Preview(c *gin.Context) {
	var req PreviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": apperrors.ErrCodeInvalidParams, "message": err.Error()})
		return
	}

	var renderedContent string
	var err error

	// 1. 如果直接提供了内容，使用直接渲染
	if req.Content != "" {
		renderedContent, err = h.svc.RenderContent(c.Request.Context(), req.Content, req.Data)
	} else if req.TemplateID > 0 {
		renderedContent, err = h.svc.RenderByID(c.Request.Context(), req.TemplateID, req.Data)
	} else if req.TemplateName != "" {
		renderedContent, err = h.svc.Render(c.Request.Context(), req.TemplateName, req.Data)
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "content, template_id or template_name is required"})
		return
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "Render failed: " + err.Error()})
		return
	}

	// 尝试解析 JSON 验证格式是否正确
	var jsonCheck interface{}
	isValidJSON := true
	var jsonError string
	if jsonErr := json.Unmarshal([]byte(renderedContent), &jsonCheck); jsonErr != nil {
		isValidJSON = false
		jsonError = jsonErr.Error()
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    apperrors.Success,
		"message": "success",
		"data": gin.H{
			"content":    renderedContent,
			"valid_json": isValidJSON,
			"json_error": jsonError,
		},
	})
}
