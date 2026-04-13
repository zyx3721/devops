// Package handler 制品管理模块处理器
package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"devops/internal/models/artifact"
	artifactSvc "devops/internal/service/artifact"
)

// ArtifactHandler 制品管理处理器
type ArtifactHandler struct {
	db          *gorm.DB
	artifactSvc *artifactSvc.ArtifactService
	scanSvc     *artifactSvc.ScanService
}

// NewArtifactHandler 创建制品管理处理器
func NewArtifactHandler(db *gorm.DB) *ArtifactHandler {
	return &ArtifactHandler{
		db:          db,
		artifactSvc: artifactSvc.NewArtifactService(db),
		scanSvc:     artifactSvc.NewScanService(db),
	}
}

// RegisterRoutes 注册路由
func (h *ArtifactHandler) RegisterRoutes(r *gin.RouterGroup) {
	// 仓库管理
	repo := r.Group("/artifact/repositories")
	{
		repo.GET("", h.ListRepositories)
		repo.GET("/:id", h.GetRepository)
		repo.POST("", h.CreateRepository)
		repo.PUT("/:id", h.UpdateRepository)
		repo.DELETE("/:id", h.DeleteRepository)
		// 连接测试和监控
		repo.POST("/:id/test", h.TestConnection)
		repo.POST("/refresh-status", h.RefreshAllStatus)
		repo.GET("/:id/history", h.GetConnectionHistory)
	}

	// 版本管理 - 使用不同的路径避免与 pipeline 模块冲突
	ver := r.Group("/artifact/versions")
	{
		ver.GET("", h.ListVersions)
		ver.GET("/:version_id", h.GetVersion)
		ver.POST("", h.CreateVersion)
		ver.DELETE("/:version_id", h.DeleteVersion)
		ver.POST("/:version_id/release", h.ReleaseVersion)
		ver.POST("/:version_id/download", h.DownloadVersion)
	}

	// 扫描管理
	scan := r.Group("/artifact/scan")
	{
		scan.POST("", h.StartScan)
		scan.GET("/results/:version_id", h.GetScanResults)
		scan.GET("/stats/:repo_id", h.GetScanStats)
	}
}

// ========== 仓库管理 ==========

// ListRepositories 获取仓库列表
// @Summary 获取制品仓库列表
// @Tags 制品管理
// @Param type query string false "仓库类型"
// @Success 200 {object} gin.H
// @Router /artifact/repositories [get]
func (h *ArtifactHandler) ListRepositories(c *gin.Context) {
	repoType := c.Query("type")

	repos, err := h.artifactSvc.ListRepositories(c.Request.Context(), repoType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "获取仓库列表失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"items": repos}})
}

// GetRepository 获取仓库详情
// @Summary 获取仓库详情
// @Tags 制品管理
// @Param id path int true "仓库ID"
// @Success 200 {object} gin.H
// @Router /artifact/repositories/{id} [get]
func (h *ArtifactHandler) GetRepository(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)

	repo, err := h.artifactSvc.GetRepository(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "仓库不存在"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": repo})
}

// CreateRepository 创建仓库
// @Summary 创建制品仓库
// @Tags 制品管理
// @Param body body artifact.Repository true "仓库信息"
// @Success 200 {object} gin.H
// @Router /artifact/repositories [post]
func (h *ArtifactHandler) CreateRepository(c *gin.Context) {
	var repo artifact.Repository
	if err := c.ShouldBindJSON(&repo); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	repo.CreatedBy = c.GetString("username")

	if err := h.artifactSvc.CreateRepository(c.Request.Context(), &repo); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "创建失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": repo, "message": "创建成功"})
}

// UpdateRepository 更新仓库
// @Summary 更新制品仓库
// @Tags 制品管理
// @Param id path int true "仓库ID"
// @Param body body artifact.Repository true "仓库信息"
// @Success 200 {object} gin.H
// @Router /artifact/repositories/{id} [put]
func (h *ArtifactHandler) UpdateRepository(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)

	var repo artifact.Repository
	if err := c.ShouldBindJSON(&repo); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}
	repo.ID = id

	if err := h.artifactSvc.UpdateRepository(c.Request.Context(), &repo); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "更新失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "更新成功"})
}

// DeleteRepository 删除仓库
// @Summary 删除制品仓库
// @Tags 制品管理
// @Param id path int true "仓库ID"
// @Success 200 {object} gin.H
// @Router /artifact/repositories/{id} [delete]
func (h *ArtifactHandler) DeleteRepository(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)

	if err := h.artifactSvc.DeleteRepository(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "删除成功"})
}

// TestConnection 测试仓库连接
// @Summary 测试制品仓库连接
// @Tags 制品管理
// @Param id path int true "仓库ID"
// @Success 200 {object} gin.H
// @Router /artifact/repositories/{id}/test [post]
func (h *ArtifactHandler) TestConnection(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)

	result, err := h.artifactSvc.TestConnection(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "测试失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": result})
}

// RefreshAllStatus 刷新所有仓库状态
// @Summary 刷新所有制品仓库连接状态
// @Tags 制品管理
// @Success 200 {object} gin.H
// @Router /artifact/repositories/refresh-status [post]
func (h *ArtifactHandler) RefreshAllStatus(c *gin.Context) {
	if err := h.artifactSvc.RefreshAllStatus(c.Request.Context()); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "刷新失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "刷新成功"})
}

// GetConnectionHistory 获取连接历史记录
// @Summary 获取制品仓库连接历史记录
// @Tags 制品管理
// @Param id path int true "仓库ID"
// @Param page query int false "页码"
// @Param page_size query int false "每页数量"
// @Success 200 {object} gin.H
// @Router /artifact/repositories/{id}/history [get]
func (h *ArtifactHandler) GetConnectionHistory(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "50"))

	result, err := h.artifactSvc.GetConnectionHistory(c.Request.Context(), id, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "获取历史记录失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": result})
}

// ========== 制品管理 ==========

// ListArtifacts 获取制品列表
// @Summary 获取制品列表
// @Tags 制品管理
// @Param repo_id query int false "仓库ID"
// @Param keyword query string false "关键词"
// @Param page query int false "页码"
// @Param page_size query int false "每页数量"
// @Success 200 {object} gin.H
// @Router /artifacts [get]
func (h *ArtifactHandler) ListArtifacts(c *gin.Context) {
	repoID, _ := strconv.ParseUint(c.Query("repo_id"), 10, 64)
	keyword := c.Query("keyword")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	result, err := h.artifactSvc.ListArtifacts(c.Request.Context(), repoID, keyword, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "获取制品列表失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": result})
}

// SearchArtifacts 搜索制品
// @Summary 搜索制品
// @Tags 制品管理
// @Param keyword query string true "关键词"
// @Param type query string false "制品类型"
// @Success 200 {object} gin.H
// @Router /artifacts/search [get]
func (h *ArtifactHandler) SearchArtifacts(c *gin.Context) {
	keyword := c.Query("keyword")
	artType := c.Query("type")

	if keyword == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "关键词不能为空"})
		return
	}

	arts, err := h.artifactSvc.SearchArtifacts(c.Request.Context(), keyword, artType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "搜索失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"items": arts}})
}

// GetArtifact 获取制品详情
// @Summary 获取制品详情
// @Tags 制品管理
// @Param id path int true "制品ID"
// @Success 200 {object} gin.H
// @Router /artifacts/{id} [get]
func (h *ArtifactHandler) GetArtifact(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)

	art, err := h.artifactSvc.GetArtifact(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "制品不存在"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": art})
}

// CreateArtifact 创建制品
// @Summary 创建制品
// @Tags 制品管理
// @Param body body artifact.Artifact true "制品信息"
// @Success 200 {object} gin.H
// @Router /artifacts [post]
func (h *ArtifactHandler) CreateArtifact(c *gin.Context) {
	var art artifact.Artifact
	if err := c.ShouldBindJSON(&art); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	art.CreatedBy = c.GetString("username")

	if err := h.artifactSvc.CreateArtifact(c.Request.Context(), &art); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "创建失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": art, "message": "创建成功"})
}

// UpdateArtifact 更新制品
// @Summary 更新制品
// @Tags 制品管理
// @Param id path int true "制品ID"
// @Param body body artifact.Artifact true "制品信息"
// @Success 200 {object} gin.H
// @Router /artifacts/{id} [put]
func (h *ArtifactHandler) UpdateArtifact(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)

	var art artifact.Artifact
	if err := c.ShouldBindJSON(&art); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}
	art.ID = id

	if err := h.artifactSvc.UpdateArtifact(c.Request.Context(), &art); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "更新失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "更新成功"})
}

// DeleteArtifact 删除制品
// @Summary 删除制品
// @Tags 制品管理
// @Param id path int true "制品ID"
// @Success 200 {object} gin.H
// @Router /artifacts/{id} [delete]
func (h *ArtifactHandler) DeleteArtifact(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)

	if err := h.artifactSvc.DeleteArtifact(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "删除失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "删除成功"})
}

// ========== 版本管理 ==========

// ListVersions 获取版本列表
// @Summary 获取制品版本列表
// @Tags 制品管理
// @Param artifact_id path int true "制品ID"
// @Param page query int false "页码"
// @Param page_size query int false "每页数量"
// @Success 200 {object} gin.H
// @Router /artifacts/{artifact_id}/versions [get]
func (h *ArtifactHandler) ListVersions(c *gin.Context) {
	artifactID, _ := strconv.ParseUint(c.Param("artifact_id"), 10, 64)
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	result, err := h.artifactSvc.ListVersions(c.Request.Context(), artifactID, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "获取版本列表失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": result})
}

// GetVersion 获取版本详情
// @Summary 获取版本详情
// @Tags 制品管理
// @Param artifact_id path int true "制品ID"
// @Param version_id path int true "版本ID"
// @Success 200 {object} gin.H
// @Router /artifacts/{artifact_id}/versions/{version_id} [get]
func (h *ArtifactHandler) GetVersion(c *gin.Context) {
	versionID, _ := strconv.ParseUint(c.Param("version_id"), 10, 64)

	ver, err := h.artifactSvc.GetVersion(c.Request.Context(), versionID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "版本不存在"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": ver})
}

// CreateVersion 创建版本
// @Summary 创建制品版本
// @Tags 制品管理
// @Param artifact_id path int true "制品ID"
// @Param body body artifact.ArtifactVersion true "版本信息"
// @Success 200 {object} gin.H
// @Router /artifacts/{artifact_id}/versions [post]
func (h *ArtifactHandler) CreateVersion(c *gin.Context) {
	artifactID, _ := strconv.ParseUint(c.Param("artifact_id"), 10, 64)

	var ver artifact.ArtifactVersion
	if err := c.ShouldBindJSON(&ver); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}
	ver.ArtifactID = artifactID

	if err := h.artifactSvc.CreateVersion(c.Request.Context(), &ver); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "创建失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": ver, "message": "创建成功"})
}

// DeleteVersion 删除版本
// @Summary 删除制品版本
// @Tags 制品管理
// @Param artifact_id path int true "制品ID"
// @Param version_id path int true "版本ID"
// @Success 200 {object} gin.H
// @Router /artifacts/{artifact_id}/versions/{version_id} [delete]
func (h *ArtifactHandler) DeleteVersion(c *gin.Context) {
	versionID, _ := strconv.ParseUint(c.Param("version_id"), 10, 64)

	if err := h.artifactSvc.DeleteVersion(c.Request.Context(), versionID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "删除失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "删除成功"})
}

// ReleaseVersion 发布版本
// @Summary 发布制品版本
// @Tags 制品管理
// @Param artifact_id path int true "制品ID"
// @Param version_id path int true "版本ID"
// @Success 200 {object} gin.H
// @Router /artifacts/{artifact_id}/versions/{version_id}/release [post]
func (h *ArtifactHandler) ReleaseVersion(c *gin.Context) {
	versionID, _ := strconv.ParseUint(c.Param("version_id"), 10, 64)
	username := c.GetString("username")

	if err := h.artifactSvc.ReleaseVersion(c.Request.Context(), versionID, username); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "发布失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "发布成功"})
}

// DownloadVersion 下载版本
// @Summary 下载制品版本
// @Tags 制品管理
// @Param artifact_id path int true "制品ID"
// @Param version_id path int true "版本ID"
// @Success 200 {object} gin.H
// @Router /artifacts/{artifact_id}/versions/{version_id}/download [post]
func (h *ArtifactHandler) DownloadVersion(c *gin.Context) {
	versionID, _ := strconv.ParseUint(c.Param("version_id"), 10, 64)

	ver, err := h.artifactSvc.GetVersion(c.Request.Context(), versionID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "版本不存在"})
		return
	}

	// 增加下载次数
	h.artifactSvc.IncrementDownload(c.Request.Context(), versionID)

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"download_url": ver.DownloadURL,
			"checksum":     ver.Checksum,
		},
	})
}

// ========== 扫描管理 ==========

// StartScan 开始扫描
// @Summary 开始制品扫描
// @Tags 制品管理
// @Param body body artifactSvc.ScanRequest true "扫描请求"
// @Success 200 {object} gin.H
// @Router /artifact/scan [post]
func (h *ArtifactHandler) StartScan(c *gin.Context) {
	var req artifactSvc.ScanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	if err := h.scanSvc.StartScan(c.Request.Context(), &req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "扫描失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "扫描已启动"})
}

// GetScanResults 获取扫描结果
// @Summary 获取扫描结果
// @Tags 制品管理
// @Param version_id path int true "版本ID"
// @Success 200 {object} gin.H
// @Router /artifact/scan/results/{version_id} [get]
func (h *ArtifactHandler) GetScanResults(c *gin.Context) {
	versionID, _ := strconv.ParseUint(c.Param("version_id"), 10, 64)

	results, err := h.scanSvc.GetScanResults(c.Request.Context(), versionID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "获取扫描结果失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"items": results}})
}

// GetScanStats 获取扫描统计
// @Summary 获取扫描统计
// @Tags 制品管理
// @Param repo_id path int true "仓库ID"
// @Success 200 {object} gin.H
// @Router /artifact/scan/stats/{repo_id} [get]
func (h *ArtifactHandler) GetScanStats(c *gin.Context) {
	repoID, _ := strconv.ParseUint(c.Param("repo_id"), 10, 64)

	stats, err := h.scanSvc.GetScanStats(c.Request.Context(), repoID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "获取统计失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": stats})
}
