package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"devops/internal/models/ai"
	"devops/internal/repository"
	aiservice "devops/internal/service/ai"
	"devops/pkg/response"
)

// KnowledgeHandler 知识库处理器
type KnowledgeHandler struct {
	db      *gorm.DB
	service *aiservice.KnowledgeService
}

// NewKnowledgeHandler 创建知识库处理器
func NewKnowledgeHandler(db *gorm.DB) *KnowledgeHandler {
	return &KnowledgeHandler{
		db:      db,
		service: aiservice.NewKnowledgeService(db),
	}
}

// CreateKnowledgeRequest 创建知识请求
type CreateKnowledgeRequest struct {
	Title    string   `json:"title" binding:"required"`
	Content  string   `json:"content" binding:"required"`
	Category string   `json:"category" binding:"required"`
	Tags     []string `json:"tags"`
}

// Create 创建知识条目
// @Summary 创建知识条目
// @Tags AI Knowledge
// @Accept json
// @Produce json
// @Param request body CreateKnowledgeRequest true "知识内容"
// @Success 200 {object} response.Response
// @Router /api/v1/ai/knowledge [post]
func (h *KnowledgeHandler) Create(c *gin.Context) {
	var req CreateKnowledgeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}

	userID := getUserID(c)

	doc := ai.Document{
		Title:    req.Title,
		Content:  req.Content,
		Category: req.Category,
		Tags:     req.Tags,
	}

	knowledge, err := h.service.AddDocument(c.Request.Context(), doc, userID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "创建失败: "+err.Error())
		return
	}

	response.Success(c, knowledge)
}

// List 获取知识列表
// @Summary 获取知识列表
// @Tags AI Knowledge
// @Produce json
// @Param category query string false "分类"
// @Param keyword query string false "关键词"
// @Param page query int false "页码"
// @Param page_size query int false "每页数量"
// @Success 200 {object} response.Response
// @Router /api/v1/ai/knowledge [get]
func (h *KnowledgeHandler) List(c *gin.Context) {
	category := c.Query("category")
	keyword := c.Query("keyword")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	filter := repository.AIKnowledgeFilter{
		Category: category,
		Keyword:  keyword,
	}

	items, total, err := h.service.List(c.Request.Context(), filter, page, pageSize)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "获取失败: "+err.Error())
		return
	}

	response.Page(c, items, total, page, pageSize)
}

// Get 获取知识详情
// @Summary 获取知识详情
// @Tags AI Knowledge
// @Produce json
// @Param id path int true "知识ID"
// @Success 200 {object} response.Response
// @Router /api/v1/ai/knowledge/{id} [get]
func (h *KnowledgeHandler) Get(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "无效的ID")
		return
	}

	knowledge, err := h.service.GetByID(c.Request.Context(), uint(id))
	if err != nil {
		response.Error(c, http.StatusNotFound, "知识不存在")
		return
	}

	response.Success(c, knowledge)
}

// UpdateKnowledgeRequest 更新知识请求
type UpdateKnowledgeRequest struct {
	Title    string   `json:"title"`
	Content  string   `json:"content"`
	Category string   `json:"category"`
	Tags     []string `json:"tags"`
}

// Update 更新知识条目
// @Summary 更新知识条目
// @Tags AI Knowledge
// @Accept json
// @Produce json
// @Param id path int true "知识ID"
// @Param request body UpdateKnowledgeRequest true "知识内容"
// @Success 200 {object} response.Response
// @Router /api/v1/ai/knowledge/{id} [put]
func (h *KnowledgeHandler) Update(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "无效的ID")
		return
	}

	var req UpdateKnowledgeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}

	userID := getUserID(c)

	doc := ai.Document{
		Title:    req.Title,
		Content:  req.Content,
		Category: req.Category,
		Tags:     req.Tags,
	}

	knowledge, err := h.service.UpdateDocument(c.Request.Context(), uint(id), doc, userID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "更新失败: "+err.Error())
		return
	}

	response.Success(c, knowledge)
}

// Delete 删除知识条目
// @Summary 删除知识条目
// @Tags AI Knowledge
// @Param id path int true "知识ID"
// @Success 200 {object} response.Response
// @Router /api/v1/ai/knowledge/{id} [delete]
func (h *KnowledgeHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "无效的ID")
		return
	}

	if err := h.service.Delete(c.Request.Context(), uint(id)); err != nil {
		response.Error(c, http.StatusInternalServerError, "删除失败: "+err.Error())
		return
	}

	response.Success(c, nil)
}

// Search 搜索知识
// @Summary 搜索知识
// @Tags AI Knowledge
// @Produce json
// @Param q query string true "搜索关键词"
// @Param limit query int false "返回数量限制"
// @Success 200 {object} response.Response
// @Router /api/v1/ai/knowledge/search [get]
func (h *KnowledgeHandler) Search(c *gin.Context) {
	query := c.Query("q")
	if query == "" {
		response.Error(c, http.StatusBadRequest, "搜索关键词不能为空")
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	items, err := h.service.Search(c.Request.Context(), query, limit)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "搜索失败: "+err.Error())
		return
	}

	response.Success(c, items)
}

// GetCategories 获取所有分类
// @Summary 获取所有分类
// @Tags AI Knowledge
// @Produce json
// @Success 200 {object} response.Response
// @Router /api/v1/ai/knowledge/categories [get]
func (h *KnowledgeHandler) GetCategories(c *gin.Context) {
	categories, err := h.service.GetAllCategories(c.Request.Context())
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "获取分类失败: "+err.Error())
		return
	}

	// 添加显示名称
	result := make([]map[string]string, len(categories))
	for i, cat := range categories {
		result[i] = map[string]string{
			"value": cat,
			"label": aiservice.GetCategoryDisplayName(ai.KnowledgeCategory(cat)),
		}
	}

	response.Success(c, result)
}

// RegisterRoutes 注册知识库相关路由
func (h *KnowledgeHandler) RegisterRoutes(r *gin.RouterGroup) {
	g := r.Group("/ai/knowledge")
	{
		g.POST("", h.Create)
		g.GET("", h.List)
		g.GET("/search", h.Search)
		g.GET("/categories", h.GetCategories)
		g.GET("/:id", h.Get)
		g.PUT("/:id", h.Update)
		g.DELETE("/:id", h.Delete)
	}
}
