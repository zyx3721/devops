package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"devops/internal/config"
	"devops/internal/models"
	"devops/internal/service/pipeline"
	"devops/pkg/ioc"
	"devops/pkg/middleware"
	"devops/pkg/response"
)

func init() {
	ioc.Api.RegisterContainer("RegistryHandler", &RegistryApiHandler{})
}

// RegistryApiHandler IOC容器注册的处理器
type RegistryApiHandler struct {
	handler *RegistryHandler
}

func (h *RegistryApiHandler) Init() error {
	cfg, err := config.LoadConfig()
	if err != nil {
		return err
	}

	db := cfg.GetDB()
	registrySvc := pipeline.NewRegistryService(db)

	h.handler = NewRegistryHandler(registrySvc)

	root := cfg.Application.GinRootRouter().Group("artifact-registries")
	root.Use(middleware.AuthMiddleware())
	h.Register(root)

	return nil
}

func (h *RegistryApiHandler) Register(r gin.IRouter) {
	r.GET("", h.handler.List)
	r.GET("/:id", h.handler.Get)
	r.POST("", h.handler.Create)
	r.PUT("/:id", h.handler.Update)
	r.DELETE("/:id", middleware.RequireAdmin(), h.handler.Delete)
	r.POST("/:id/test", h.handler.TestConnection)
}

// RegistryHandler 制品库处理器
type RegistryHandler struct {
	registrySvc *pipeline.RegistryService
}

// NewRegistryHandler 创建制品库处理器
func NewRegistryHandler(registrySvc *pipeline.RegistryService) *RegistryHandler {
	return &RegistryHandler{
		registrySvc: registrySvc,
	}
}

// RegistryRequest 制品库请求
type RegistryRequest struct {
	Name        string `json:"name" binding:"required"`
	Type        string `json:"type" binding:"required"`
	URL         string `json:"url" binding:"required"`
	Username    string `json:"username"`
	Password    string `json:"password"`
	Description string `json:"description"`
	IsDefault   bool   `json:"is_default"`
}

// List 获取制品库列表
func (h *RegistryHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	items, total, err := h.registrySvc.List(c.Request.Context(), page, pageSize)
	if err != nil {
		response.InternalError(c, "获取制品库列表失败")
		return
	}

	response.Success(c, gin.H{
		"items": items,
		"total": total,
		"page":  page,
	})
}

// Get 获取制品库详情
func (h *RegistryHandler) Get(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "无效的ID")
		return
	}

	result, err := h.registrySvc.Get(c.Request.Context(), uint(id))
	if err != nil {
		response.NotFound(c, "制品库不存在")
		return
	}

	response.Success(c, result)
}

// Create 创建制品库
func (h *RegistryHandler) Create(c *gin.Context) {
	var req RegistryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	registry := &models.ArtifactRegistry{
		Name:        req.Name,
		Type:        req.Type,
		URL:         req.URL,
		Username:    req.Username,
		Password:    req.Password,
		Description: req.Description,
		IsDefault:   req.IsDefault,
	}

	if err := h.registrySvc.Create(c.Request.Context(), registry); err != nil {
		response.InternalError(c, "创建制品库失败")
		return
	}

	response.SuccessWithMessage(c, "创建成功", registry)
}

// Update 更新制品库
func (h *RegistryHandler) Update(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "无效的ID")
		return
	}

	var req RegistryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	registry := &models.ArtifactRegistry{
		Name:        req.Name,
		Type:        req.Type,
		URL:         req.URL,
		Username:    req.Username,
		Password:    req.Password,
		Description: req.Description,
		IsDefault:   req.IsDefault,
	}

	if err := h.registrySvc.Update(c.Request.Context(), uint(id), registry); err != nil {
		response.InternalError(c, "更新制品库失败")
		return
	}

	response.SuccessWithMessage(c, "更新成功", nil)
}

// Delete 删除制品库
func (h *RegistryHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "无效的ID")
		return
	}

	if err := h.registrySvc.Delete(c.Request.Context(), uint(id)); err != nil {
		response.InternalError(c, "删除制品库失败")
		return
	}

	response.SuccessWithMessage(c, "删除成功", nil)
}

// TestConnection 测试制品库连接
func (h *RegistryHandler) TestConnection(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "无效的ID")
		return
	}

	connected, errMsg, err := h.registrySvc.TestConnection(c.Request.Context(), uint(id))
	if err != nil {
		response.NotFound(c, "制品库不存在")
		return
	}

	response.Success(c, gin.H{
		"connected": connected,
		"error":     errMsg,
	})
}
