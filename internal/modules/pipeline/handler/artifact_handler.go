package handler

import (
	"fmt"
	"strconv"

	"github.com/gin-gonic/gin"

	"devops/internal/config"
	"devops/internal/models"
	"devops/internal/service/pipeline"
	"devops/pkg/dto"
	"devops/pkg/ioc"
	"devops/pkg/middleware"
	"devops/pkg/response"
)

func init() {
	ioc.Api.RegisterContainer("ArtifactHandler", &ArtifactApiHandler{})
}

// ArtifactApiHandler IOC容器注册的处理器
type ArtifactApiHandler struct {
	handler *ArtifactHandler
}

func (h *ArtifactApiHandler) Init() error {
	cfg, err := config.LoadConfig()
	if err != nil {
		return err
	}

	db := cfg.GetDB()
	artifactSvc := pipeline.NewArtifactService(db)
	cacheSvc := pipeline.NewCacheService(db)

	h.handler = NewArtifactHandler(artifactSvc, cacheSvc)

	root := cfg.Application.GinRootRouter().Group("artifacts")
	root.Use(middleware.AuthMiddleware())
	h.Register(root)

	// 缓存路由
	cacheRoot := cfg.Application.GinRootRouter().Group("build-caches")
	cacheRoot.Use(middleware.AuthMiddleware())
	h.RegisterCache(cacheRoot)

	return nil
}

func (h *ArtifactApiHandler) Register(r gin.IRouter) {
	r.GET("", h.handler.ListArtifacts)
	r.GET("/:id", h.handler.GetArtifact)
	r.POST("", h.handler.CreateArtifact)
	r.DELETE("/:id", middleware.RequireAdmin(), h.handler.DeleteArtifact)
}

func (h *ArtifactApiHandler) RegisterCache(r gin.IRouter) {
	r.GET("", h.handler.ListCaches)
	r.DELETE("/:id", middleware.RequireAdmin(), h.handler.DeleteCache)
	r.DELETE("/pipeline/:pipeline_id", middleware.RequireAdmin(), h.handler.DeletePipelineCaches)
}

// ArtifactHandler 制品处理器
type ArtifactHandler struct {
	artifactSvc *pipeline.ArtifactService
	cacheSvc    *pipeline.CacheService
}

// NewArtifactHandler 创建制品处理器
func NewArtifactHandler(artifactSvc *pipeline.ArtifactService, cacheSvc *pipeline.CacheService) *ArtifactHandler {
	return &ArtifactHandler{
		artifactSvc: artifactSvc,
		cacheSvc:    cacheSvc,
	}
}

// ListArtifacts 获取制品列表
func (h *ArtifactHandler) ListArtifacts(c *gin.Context) {
	var req dto.ArtifactListRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	result, err := h.artifactSvc.List(c.Request.Context(), &req)
	if err != nil {
		response.FromError(c, err)
		return
	}

	response.Success(c, result)
}

// GetArtifact 获取制品详情
func (h *ArtifactHandler) GetArtifact(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "无效的ID")
		return
	}

	result, err := h.artifactSvc.Get(c.Request.Context(), uint(id))
	if err != nil {
		response.FromError(c, err)
		return
	}

	response.Success(c, result)
}

// CreateArtifact 创建制品
func (h *ArtifactHandler) CreateArtifact(c *gin.Context) {
	var req dto.ArtifactCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	result, err := h.artifactSvc.Create(c.Request.Context(), &req)
	if err != nil {
		response.FromError(c, err)
		return
	}

	response.SuccessWithMessage(c, "创建成功", result)
}

// DeleteArtifact 删除制品
func (h *ArtifactHandler) DeleteArtifact(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "无效的ID")
		return
	}

	if err := h.artifactSvc.Delete(c.Request.Context(), uint(id)); err != nil {
		response.FromError(c, err)
		return
	}

	response.SuccessWithMessage(c, "删除成功", nil)
}

// ListCaches 获取缓存列表
func (h *ArtifactHandler) ListCaches(c *gin.Context) {
	var req dto.BuildCacheListRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	page := req.Page
	if page < 1 {
		page = 1
	}
	pageSize := req.PageSize
	if pageSize < 1 {
		pageSize = 20
	}

	caches, total, err := h.cacheSvc.ListCaches(c.Request.Context(), uint(req.PipelineID), page, pageSize)
	if err != nil {
		response.FromError(c, err)
		return
	}

	items := make([]dto.BuildCacheItem, len(caches))
	for i, cache := range caches {
		items[i] = dto.BuildCacheItem{
			ID:          cache.ID,
			PipelineID:  cache.PipelineID,
			CacheKey:    cache.CacheKey,
			StoragePath: cache.StoragePath,
			Size:        cache.Size,
			SizeHuman:   formatSize(cache.Size),
			HitCount:    cache.HitCount,
			LastUsedAt:  cache.LastUsedAt,
			ExpiresAt:   cache.ExpiresAt,
			CreatedAt:   cache.CreatedAt,
		}
	}

	result := &dto.BuildCacheListResponse{
		Total: int(total),
		Items: items,
	}

	response.Success(c, result)
}

// DeleteCache 删除缓存
func (h *ArtifactHandler) DeleteCache(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "无效的ID")
		return
	}

	// 先查询缓存获取 pipeline_id 和 cache_key
	var cache models.BuildCache
	cfg, _ := config.LoadConfig()
	db := cfg.GetDB()
	if err := db.First(&cache, id).Error; err != nil {
		response.NotFound(c, "缓存不存在")
		return
	}

	if err := h.cacheSvc.InvalidateCache(c.Request.Context(), uint(cache.PipelineID), cache.CacheKey); err != nil {
		response.FromError(c, err)
		return
	}

	response.SuccessWithMessage(c, "删除成功", nil)
}

// DeletePipelineCaches 删除流水线的所有缓存
func (h *ArtifactHandler) DeletePipelineCaches(c *gin.Context) {
	pipelineID, err := strconv.ParseUint(c.Param("pipeline_id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "无效的流水线ID")
		return
	}

	if err := h.cacheSvc.InvalidatePipelineCaches(c.Request.Context(), uint(pipelineID)); err != nil {
		response.FromError(c, err)
		return
	}

	response.SuccessWithMessage(c, "删除成功", nil)
}

// formatSize 格式化文件大小
func formatSize(size int64) string {
	if size < 1024 {
		return fmt.Sprintf("%d B", size)
	}
	if size < 1024*1024 {
		return fmt.Sprintf("%.2f KB", float64(size)/1024)
	}
	if size < 1024*1024*1024 {
		return fmt.Sprintf("%.2f MB", float64(size)/(1024*1024))
	}
	return fmt.Sprintf("%.2f GB", float64(size)/(1024*1024*1024))
}
