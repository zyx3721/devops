// Package handler 流水线模块处理器
// 本文件实现构建优化相关 API
package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"devops/internal/models/pipeline"
	pipelineSvc "devops/internal/service/pipeline"
)

// BuildHandler 构建优化处理器
type BuildHandler struct {
	db       *gorm.DB
	cacheSvc *pipelineSvc.CacheService
	quotaSvc *pipelineSvc.ResourceQuotaService
}

// NewBuildHandler 创建构建优化处理器
func NewBuildHandler(db *gorm.DB) *BuildHandler {
	return &BuildHandler{
		db:       db,
		cacheSvc: pipelineSvc.NewCacheService(db),
		quotaSvc: pipelineSvc.NewResourceQuotaService(db),
	}
}

// RegisterRoutes 注册路由
func (h *BuildHandler) RegisterRoutes(r *gin.RouterGroup) {
	// 构建缓存
	cache := r.Group("/build/cache")
	{
		cache.GET("", h.ListCaches)
		cache.GET("/stats", h.GetCacheStats)
		cache.DELETE("/:id", h.DeleteCache)
		cache.POST("/clean", h.CleanExpiredCaches)
	}

	// 资源配额
	quota := r.Group("/build/quota")
	{
		quota.GET("", h.ListQuotas)
		quota.GET("/:id", h.GetQuota)
		quota.POST("", h.CreateQuota)
		quota.PUT("/:id", h.UpdateQuota)
		quota.DELETE("/:id", h.DeleteQuota)
		quota.GET("/check/:pipeline_id", h.CheckQuota)
	}

	// 并行构建配置
	parallel := r.Group("/build/parallel")
	{
		parallel.GET("/:pipeline_id", h.GetParallelConfig)
		parallel.PUT("/:pipeline_id", h.UpdateParallelConfig)
	}

	// 使用统计
	r.GET("/build/usage/stats", h.GetUsageStats)
}

// ========== 缓存管理 ==========

// ListCaches 获取缓存列表
// @Summary 获取构建缓存列表
// @Tags 构建优化
// @Param pipeline_id query int false "流水线ID"
// @Param page query int false "页码"
// @Param page_size query int false "每页数量"
// @Success 200 {object} gin.H
// @Router /build/cache [get]
func (h *BuildHandler) ListCaches(c *gin.Context) {
	pipelineID, _ := strconv.ParseUint(c.Query("pipeline_id"), 10, 64)
	page, _ := strconv.Atoi(c.Query("page"))
	if page < 1 {
		page = 1
	}
	pageSize, _ := strconv.Atoi(c.Query("page_size"))
	if pageSize < 1 {
		pageSize = 20
	}

	caches, total, err := h.cacheSvc.ListCaches(c.Request.Context(), uint(pipelineID), page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "获取缓存列表失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"items": caches, "total": total}})
}

// GetCacheStats 获取缓存统计
// @Summary 获取缓存统计
// @Tags 构建优化
// @Param pipeline_id query int false "流水线ID"
// @Success 200 {object} gin.H
// @Router /build/cache/stats [get]
func (h *BuildHandler) GetCacheStats(c *gin.Context) {
	pipelineID, _ := strconv.ParseUint(c.Query("pipeline_id"), 10, 64)

	stats, err := h.cacheSvc.GetCacheStats(c.Request.Context(), uint(pipelineID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "获取统计失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": stats})
}

// DeleteCache 删除缓存
// @Summary 删除构建缓存
// @Tags 构建优化
// @Param id path int true "缓存ID"
// @Success 200 {object} gin.H
// @Router /build/cache/{id} [delete]
func (h *BuildHandler) DeleteCache(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)

	// 先查询缓存获取 pipeline_id 和 cache_key
	var cache pipeline.BuildCache
	if err := h.db.First(&cache, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "缓存不存在"})
		return
	}

	if err := h.cacheSvc.InvalidateCache(c.Request.Context(), uint(cache.PipelineID), cache.CacheKey); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "删除失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "删除成功"})
}

// CleanExpiredCaches 清理过期缓存
// @Summary 清理过期缓存
// @Tags 构建优化
// @Success 200 {object} gin.H
// @Router /build/cache/clean [post]
func (h *BuildHandler) CleanExpiredCaches(c *gin.Context) {
	count, err := h.cacheSvc.CleanupExpired(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "清理失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "清理成功", "data": gin.H{"cleaned_count": count}})
}

// ========== 资源配额 ==========

// ListQuotas 获取配额列表
// @Summary 获取资源配额列表
// @Tags 构建优化
// @Param project_id query int false "项目ID"
// @Success 200 {object} gin.H
// @Router /build/quota [get]
func (h *BuildHandler) ListQuotas(c *gin.Context) {
	var projectID *uint64
	if pid := c.Query("project_id"); pid != "" {
		id, _ := strconv.ParseUint(pid, 10, 64)
		projectID = &id
	}

	quotas, err := h.quotaSvc.ListQuotas(c.Request.Context(), projectID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "获取配额列表失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"items": quotas}})
}

// GetQuota 获取配额详情
// @Summary 获取资源配额详情
// @Tags 构建优化
// @Param id path int true "配额ID"
// @Success 200 {object} gin.H
// @Router /build/quota/{id} [get]
func (h *BuildHandler) GetQuota(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)

	quota, err := h.quotaSvc.GetQuota(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "配额不存在"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": quota})
}

// CreateQuota 创建配额
// @Summary 创建资源配额
// @Tags 构建优化
// @Param body body pipeline.BuildResourceQuota true "配额信息"
// @Success 200 {object} gin.H
// @Router /build/quota [post]
func (h *BuildHandler) CreateQuota(c *gin.Context) {
	var quota pipeline.BuildResourceQuota
	if err := c.ShouldBindJSON(&quota); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	if err := h.quotaSvc.CreateQuota(c.Request.Context(), &quota); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "创建失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": quota, "message": "创建成功"})
}

// UpdateQuota 更新配额
// @Summary 更新资源配额
// @Tags 构建优化
// @Param id path int true "配额ID"
// @Param body body pipeline.BuildResourceQuota true "配额信息"
// @Success 200 {object} gin.H
// @Router /build/quota/{id} [put]
func (h *BuildHandler) UpdateQuota(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)

	var quota pipeline.BuildResourceQuota
	if err := c.ShouldBindJSON(&quota); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}
	quota.ID = id

	if err := h.quotaSvc.UpdateQuota(c.Request.Context(), &quota); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "更新失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "更新成功"})
}

// DeleteQuota 删除配额
// @Summary 删除资源配额
// @Tags 构建优化
// @Param id path int true "配额ID"
// @Success 200 {object} gin.H
// @Router /build/quota/{id} [delete]
func (h *BuildHandler) DeleteQuota(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)

	if err := h.quotaSvc.DeleteQuota(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "删除失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "删除成功"})
}

// CheckQuota 检查配额
// @Summary 检查构建配额
// @Tags 构建优化
// @Param pipeline_id path int true "流水线ID"
// @Success 200 {object} gin.H
// @Router /build/quota/check/{pipeline_id} [get]
func (h *BuildHandler) CheckQuota(c *gin.Context) {
	pipelineID, _ := strconv.ParseUint(c.Param("pipeline_id"), 10, 64)

	result, err := h.quotaSvc.CheckQuota(c.Request.Context(), pipelineID, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "检查失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": result})
}

// ========== 并行构建 ==========

// GetParallelConfig 获取并行构建配置
// @Summary 获取并行构建配置
// @Tags 构建优化
// @Param pipeline_id path int true "流水线ID"
// @Success 200 {object} gin.H
// @Router /build/parallel/{pipeline_id} [get]
func (h *BuildHandler) GetParallelConfig(c *gin.Context) {
	pipelineID, _ := strconv.ParseUint(c.Param("pipeline_id"), 10, 64)

	var config pipeline.ParallelBuildConfig
	if err := h.db.Where("pipeline_id = ?", pipelineID).First(&config).Error; err != nil {
		// 返回默认配置
		config = pipeline.ParallelBuildConfig{
			PipelineID:  pipelineID,
			Enabled:     true,
			MaxParallel: 3,
			FailFast:    true,
		}
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": config})
}

// UpdateParallelConfig 更新并行构建配置
// @Summary 更新并行构建配置
// @Tags 构建优化
// @Param pipeline_id path int true "流水线ID"
// @Param body body pipeline.ParallelBuildConfig true "配置信息"
// @Success 200 {object} gin.H
// @Router /build/parallel/{pipeline_id} [put]
func (h *BuildHandler) UpdateParallelConfig(c *gin.Context) {
	pipelineID, _ := strconv.ParseUint(c.Param("pipeline_id"), 10, 64)

	var config pipeline.ParallelBuildConfig
	if err := c.ShouldBindJSON(&config); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}
	config.PipelineID = pipelineID

	// 使用 upsert
	if err := h.db.Where("pipeline_id = ?", pipelineID).Assign(config).FirstOrCreate(&config).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "保存失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "保存成功"})
}

// ========== 使用统计 ==========

// GetUsageStats 获取使用统计
// @Summary 获取构建使用统计
// @Tags 构建优化
// @Param pipeline_id query int true "流水线ID"
// @Param start_time query string false "开始时间"
// @Param end_time query string false "结束时间"
// @Success 200 {object} gin.H
// @Router /build/usage/stats [get]
func (h *BuildHandler) GetUsageStats(c *gin.Context) {
	pipelineID, _ := strconv.ParseUint(c.Query("pipeline_id"), 10, 64)

	startTime := time.Now().AddDate(0, 0, -7) // 默认最近7天
	endTime := time.Now()

	if st := c.Query("start_time"); st != "" {
		if t, err := time.Parse("2006-01-02", st); err == nil {
			startTime = t
		}
	}
	if et := c.Query("end_time"); et != "" {
		if t, err := time.Parse("2006-01-02", et); err == nil {
			endTime = t
		}
	}

	stats, err := h.quotaSvc.GetUsageStats(c.Request.Context(), pipelineID, startTime, endTime)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "获取统计失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": stats})
}
