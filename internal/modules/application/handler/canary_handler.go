package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"devops/internal/models"
	"devops/internal/service/kubernetes"
)

// CanaryHandler 灰度发布处理器
type CanaryHandler struct {
	db           *gorm.DB
	clientMgr    *kubernetes.K8sClientManager
	istioService *kubernetes.IstioService
}

// NewCanaryHandler 创建灰度发布处理器
func NewCanaryHandler(db *gorm.DB, clientMgr *kubernetes.K8sClientManager, istioService *kubernetes.IstioService) *CanaryHandler {
	return &CanaryHandler{db: db, clientMgr: clientMgr, istioService: istioService}
}

// RegisterRoutes 注册路由
func (h *CanaryHandler) RegisterRoutes(r *gin.RouterGroup) {
	// 全局蓝绿部署路由（不需要应用ID）
	// 注意：灰度发布路由已在 deploy_check_handler.go 中注册 (/deploy/canary/*)
	bg := r.Group("/deploy/bluegreen")
	{
		bg.GET("/list", h.ListAllBlueGreenDeployments)
		bg.POST("/start", h.StartBlueGreenDeployment)
		bg.GET("/:id/status", h.GetBlueGreenStatus)
		bg.POST("/:id/switch", h.SwitchBlueGreenByID)
		bg.POST("/:id/rollback", h.RollbackBlueGreenByID)
		bg.POST("/:id/cleanup", h.CleanupBlueGreen)
	}

	// 应用级别的路由
	g := r.Group("/applications/:id/release")
	{
		// 金丝雀发布
		g.GET("/canary", h.ListCanaryReleases)
		g.POST("/canary", h.CreateCanaryRelease)
		g.GET("/canary/:releaseId", h.GetCanaryRelease)
		g.PUT("/canary/:releaseId", h.UpdateCanaryRelease)
		g.POST("/canary/:releaseId/start", h.StartCanaryRelease)
		g.POST("/canary/:releaseId/pause", h.PauseCanaryRelease)
		g.POST("/canary/:releaseId/resume", h.ResumeCanaryRelease)
		g.POST("/canary/:releaseId/rollback", h.RollbackCanaryRelease)
		g.POST("/canary/:releaseId/complete", h.CompleteCanaryRelease)
		g.DELETE("/canary/:releaseId", h.DeleteCanaryRelease)

		// 蓝绿部署
		g.GET("/bluegreen", h.ListBlueGreenDeployments)
		g.POST("/bluegreen", h.CreateBlueGreenDeployment)
		g.GET("/bluegreen/:deployId", h.GetBlueGreenDeployment)
		g.POST("/bluegreen/:deployId/switch", h.SwitchBlueGreen)
		g.DELETE("/bluegreen/:deployId", h.DeleteBlueGreenDeployment)
	}
}

// getApp 获取应用信息
func (h *CanaryHandler) getApp(c *gin.Context) (*models.Application, error) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	var app models.Application
	if err := h.db.First(&app, id).Error; err != nil {
		return nil, err
	}
	return &app, nil
}

// ========== 金丝雀发布 ==========

// ListAllCanaryReleases 获取所有灰度发布列表（全局）
func (h *CanaryHandler) ListAllCanaryReleases(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	appID := c.Query("application_id")
	envName := c.Query("env_name")
	status := c.Query("status")

	query := h.db.Model(&models.CanaryRelease{})

	if appID != "" {
		query = query.Where("app_id = ?", appID)
	}
	if envName != "" {
		query = query.Where("env_name = ?", envName)
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}

	var total int64
	query.Count(&total)

	var releases []models.CanaryRelease
	query.Order("created_at DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&releases)

	// 填充应用名称
	type ReleaseWithApp struct {
		models.CanaryRelease
		AppName string `json:"app_name"`
	}
	var result []ReleaseWithApp
	for _, r := range releases {
		var app models.Application
		h.db.Select("name").First(&app, r.AppID)
		result = append(result, ReleaseWithApp{
			CanaryRelease: r,
			AppName:       app.Name,
		})
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"list": result, "total": total}})
}

// StartCanaryDeployment 创建并启动灰度发布（全局）
func (h *CanaryHandler) StartCanaryDeployment(c *gin.Context) {
	var req struct {
		ApplicationID     uint64 `json:"application_id" binding:"required"`
		EnvName           string `json:"env_name" binding:"required"`
		ImageTag          string `json:"image_tag" binding:"required"`
		CanaryPercent     int    `json:"canary_percent"`
		CanaryHeader      string `json:"canary_header"`
		CanaryHeaderValue string `json:"canary_header_value"`
		CanaryCookie      string `json:"canary_cookie"`
		Description       string `json:"description"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	var app models.Application
	if err := h.db.First(&app, req.ApplicationID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "应用不存在"})
		return
	}

	// 获取当前版本作为稳定版本
	stableVersion := "current"
	if app.K8sDeployment != "" {
		stableVersion = app.K8sDeployment
	}

	if req.CanaryPercent <= 0 {
		req.CanaryPercent = 10
	}

	release := models.CanaryRelease{
		AppID:           req.ApplicationID,
		Name:            fmt.Sprintf("%s-canary-%d", app.Name, time.Now().Unix()),
		EnvName:         req.EnvName,
		StableVersion:   stableVersion,
		CanaryVersion:   req.ImageTag,
		CurrentWeight:   req.CanaryPercent,
		TargetWeight:    100,
		WeightIncrement: 10,
		Status:          "canary_running",
	}

	now := time.Now()
	release.StartedAt = &now

	if err := h.db.Create(&release).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	// 同步到 K8s
	if app.K8sClusterID != nil && app.K8sNamespace != "" {
		h.syncCanaryRouting(c.Request.Context(), &app, &release)
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "灰度发布已启动", "data": release})
}

// GetCanaryStatus 获取灰度发布状态
func (h *CanaryHandler) GetCanaryStatus(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	var release models.CanaryRelease
	if err := h.db.First(&release, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "发布不存在"})
		return
	}

	var app models.Application
	h.db.Select("name").First(&app, release.AppID)

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{
		"id":             release.ID,
		"app_id":         release.AppID,
		"app_name":       app.Name,
		"env_name":       release.EnvName,
		"image_tag":      release.CanaryVersion,
		"canary_percent": release.CurrentWeight,
		"status":         release.Status,
		"created_at":     release.CreatedAt,
		"started_at":     release.StartedAt,
	}})
}

// AdjustCanaryPercent 调整灰度比例
func (h *CanaryHandler) AdjustCanaryPercent(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	var release models.CanaryRelease
	if err := h.db.First(&release, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "发布不存在"})
		return
	}

	var req struct {
		Percent int `json:"percent" binding:"required,min=1,max=100"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	h.db.Model(&release).Update("current_weight", req.Percent)

	// 同步到 K8s
	var app models.Application
	if err := h.db.First(&app, release.AppID).Error; err == nil {
		if app.K8sClusterID != nil && app.K8sNamespace != "" {
			release.CurrentWeight = req.Percent
			h.syncCanaryRouting(c.Request.Context(), &app, &release)
		}
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "灰度比例已调整"})
}

// PromoteCanary 全量发布
func (h *CanaryHandler) PromoteCanary(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	var release models.CanaryRelease
	if err := h.db.First(&release, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "发布不存在"})
		return
	}

	now := time.Now()
	h.db.Model(&release).Updates(map[string]any{
		"status":         "success",
		"current_weight": 100,
		"completed_at":   &now,
	})

	// 同步到 K8s - 100% 流量到新版本
	var app models.Application
	if err := h.db.First(&app, release.AppID).Error; err == nil {
		if app.K8sClusterID != nil && app.K8sNamespace != "" {
			release.CurrentWeight = 100
			h.syncCanaryRouting(c.Request.Context(), &app, &release)
		}
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "全量发布成功"})
}

// RollbackCanaryByID 回滚灰度发布（通过ID）
func (h *CanaryHandler) RollbackCanaryByID(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	var release models.CanaryRelease
	if err := h.db.First(&release, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "发布不存在"})
		return
	}

	now := time.Now()
	h.db.Model(&release).Updates(map[string]any{
		"status":         "rolled_back",
		"current_weight": 0,
		"completed_at":   &now,
	})

	// 同步到 K8s - 100% 流量回到稳定版本
	var app models.Application
	if err := h.db.First(&app, release.AppID).Error; err == nil {
		if app.K8sClusterID != nil && app.K8sNamespace != "" {
			release.CurrentWeight = 0
			h.syncCanaryRouting(c.Request.Context(), &app, &release)
		}
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "回滚成功"})
}

// ListCanaryReleases 获取金丝雀发布列表
func (h *CanaryHandler) ListCanaryReleases(c *gin.Context) {
	app, err := h.getApp(c)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "应用不存在"})
		return
	}

	var releases []models.CanaryRelease
	h.db.Where("app_id = ?", app.ID).Order("created_at DESC").Find(&releases)

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"items": releases}})
}

// CreateCanaryRelease 创建金丝雀发布
func (h *CanaryHandler) CreateCanaryRelease(c *gin.Context) {
	app, err := h.getApp(c)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "应用不存在"})
		return
	}

	var release models.CanaryRelease
	if err := c.ShouldBindJSON(&release); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	release.AppID = uint64(app.ID)
	release.Status = "pending"
	release.CurrentWeight = 0

	if err := h.db.Create(&release).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "创建成功", "data": release})
}

// GetCanaryRelease 获取金丝雀发布详情
func (h *CanaryHandler) GetCanaryRelease(c *gin.Context) {
	app, err := h.getApp(c)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "应用不存在"})
		return
	}

	releaseID, _ := strconv.ParseUint(c.Param("releaseId"), 10, 64)
	var release models.CanaryRelease
	if err := h.db.Where("id = ? AND app_id = ?", releaseID, app.ID).First(&release).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "发布不存在"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": release})
}

// UpdateCanaryRelease 更新金丝雀发布配置
func (h *CanaryHandler) UpdateCanaryRelease(c *gin.Context) {
	app, err := h.getApp(c)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "应用不存在"})
		return
	}

	releaseID, _ := strconv.ParseUint(c.Param("releaseId"), 10, 64)
	var release models.CanaryRelease
	if err := h.db.Where("id = ? AND app_id = ?", releaseID, app.ID).First(&release).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "发布不存在"})
		return
	}

	var req models.CanaryRelease
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	h.db.Model(&release).Updates(map[string]interface{}{
		"name":              req.Name,
		"canary_version":    req.CanaryVersion,
		"target_weight":     req.TargetWeight,
		"weight_increment":  req.WeightIncrement,
		"interval_seconds":  req.IntervalSeconds,
		"success_threshold": req.SuccessThreshold,
		"latency_threshold": req.LatencyThreshold,
		"auto_rollback":     req.AutoRollback,
	})

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "更新成功"})
}

// StartCanaryRelease 开始金丝雀发布
func (h *CanaryHandler) StartCanaryRelease(c *gin.Context) {
	app, err := h.getApp(c)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "应用不存在"})
		return
	}

	releaseID, _ := strconv.ParseUint(c.Param("releaseId"), 10, 64)
	var release models.CanaryRelease
	if err := h.db.Where("id = ? AND app_id = ?", releaseID, app.ID).First(&release).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "发布不存在"})
		return
	}

	if release.Status != "pending" && release.Status != "paused" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "当前状态不允许启动"})
		return
	}

	now := time.Now()
	h.db.Model(&release).Updates(map[string]interface{}{
		"status":     "running",
		"started_at": &now,
	})

	// 同步到 K8s - 创建初始路由规则
	if app.K8sClusterID != nil && app.K8sNamespace != "" {
		h.syncCanaryRouting(c.Request.Context(), app, &release)
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "发布已启动"})
}

// PauseCanaryRelease 暂停金丝雀发布
func (h *CanaryHandler) PauseCanaryRelease(c *gin.Context) {
	app, err := h.getApp(c)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "应用不存在"})
		return
	}

	releaseID, _ := strconv.ParseUint(c.Param("releaseId"), 10, 64)
	h.db.Model(&models.CanaryRelease{}).Where("id = ? AND app_id = ?", releaseID, app.ID).
		Update("status", "paused")

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "发布已暂停"})
}

// ResumeCanaryRelease 恢复金丝雀发布
func (h *CanaryHandler) ResumeCanaryRelease(c *gin.Context) {
	app, err := h.getApp(c)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "应用不存在"})
		return
	}

	releaseID, _ := strconv.ParseUint(c.Param("releaseId"), 10, 64)
	h.db.Model(&models.CanaryRelease{}).Where("id = ? AND app_id = ?", releaseID, app.ID).
		Update("status", "running")

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "发布已恢复"})
}

// RollbackCanaryRelease 回滚金丝雀发布
func (h *CanaryHandler) RollbackCanaryRelease(c *gin.Context) {
	app, err := h.getApp(c)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "应用不存在"})
		return
	}

	releaseID, _ := strconv.ParseUint(c.Param("releaseId"), 10, 64)
	var release models.CanaryRelease
	if err := h.db.Where("id = ? AND app_id = ?", releaseID, app.ID).First(&release).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "发布不存在"})
		return
	}

	now := time.Now()
	h.db.Model(&release).Updates(map[string]interface{}{
		"status":         "rollback",
		"current_weight": 0,
		"completed_at":   &now,
	})

	// 同步到 K8s - 将所有流量切回稳定版本
	if app.K8sClusterID != nil && app.K8sNamespace != "" {
		release.CurrentWeight = 0
		h.syncCanaryRouting(c.Request.Context(), app, &release)
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "已回滚"})
}

// CompleteCanaryRelease 完成金丝雀发布
func (h *CanaryHandler) CompleteCanaryRelease(c *gin.Context) {
	app, err := h.getApp(c)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "应用不存在"})
		return
	}

	releaseID, _ := strconv.ParseUint(c.Param("releaseId"), 10, 64)
	var release models.CanaryRelease
	if err := h.db.Where("id = ? AND app_id = ?", releaseID, app.ID).First(&release).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "发布不存在"})
		return
	}

	now := time.Now()
	h.db.Model(&release).Updates(map[string]interface{}{
		"status":         "completed",
		"current_weight": 100,
		"completed_at":   &now,
	})

	// 同步到 K8s - 将所有流量切到金丝雀版本
	if app.K8sClusterID != nil && app.K8sNamespace != "" {
		release.CurrentWeight = 100
		h.syncCanaryRouting(c.Request.Context(), app, &release)
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "发布完成"})
}

// DeleteCanaryRelease 删除金丝雀发布
func (h *CanaryHandler) DeleteCanaryRelease(c *gin.Context) {
	app, err := h.getApp(c)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "应用不存在"})
		return
	}

	releaseID, _ := strconv.ParseUint(c.Param("releaseId"), 10, 64)
	h.db.Where("id = ? AND app_id = ?", releaseID, app.ID).Delete(&models.CanaryRelease{})

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "删除成功"})
}

// syncCanaryRouting 同步金丝雀路由到 K8s
func (h *CanaryHandler) syncCanaryRouting(ctx context.Context, app *models.Application, release *models.CanaryRelease) error {
	stableWeight := 100 - release.CurrentWeight
	canaryWeight := release.CurrentWeight

	// 创建路由规则
	rules := []models.TrafficRoutingRule{
		{
			Name:      fmt.Sprintf("canary-%d", release.ID),
			RouteType: "weight",
			Destinations: models.JSONDestinations{
				{Subset: "stable", Weight: stableWeight},
				{Subset: "canary", Weight: canaryWeight},
			},
			Enabled: true,
		},
	}

	return h.istioService.SyncRoutingRules(ctx, *app.K8sClusterID, app.K8sNamespace, app, rules)
}

// ========== 蓝绿部署 ==========

// ListAllBlueGreenDeployments 获取所有蓝绿部署列表（全局）
func (h *CanaryHandler) ListAllBlueGreenDeployments(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	appID := c.Query("application_id")
	envName := c.Query("env_name")
	status := c.Query("status")

	query := h.db.Model(&models.BlueGreenDeployment{})

	if appID != "" {
		query = query.Where("app_id = ?", appID)
	}
	if envName != "" {
		query = query.Where("env_name = ?", envName)
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}

	var total int64
	query.Count(&total)

	var deployments []models.BlueGreenDeployment
	query.Order("created_at DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&deployments)

	// 填充应用名称
	type DeployWithApp struct {
		models.BlueGreenDeployment
		AppName string `json:"app_name"`
	}
	var result []DeployWithApp
	for _, d := range deployments {
		var app models.Application
		h.db.Select("name").First(&app, d.AppID)
		result = append(result, DeployWithApp{
			BlueGreenDeployment: d,
			AppName:             app.Name,
		})
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"list": result, "total": total}})
}

// StartBlueGreenDeployment 创建并启动蓝绿部署（全局）
func (h *CanaryHandler) StartBlueGreenDeployment(c *gin.Context) {
	var req struct {
		ApplicationID uint64 `json:"application_id" binding:"required"`
		EnvName       string `json:"env_name" binding:"required"`
		GreenImageTag string `json:"green_image_tag" binding:"required"`
		Replicas      int    `json:"replicas"`
		Description   string `json:"description"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	var app models.Application
	if err := h.db.First(&app, req.ApplicationID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "应用不存在"})
		return
	}

	// 获取当前版本作为蓝版本
	blueImageTag := "current"
	if app.K8sDeployment != "" {
		blueImageTag = app.K8sDeployment
	}

	deploy := models.BlueGreenDeployment{
		AppID:        req.ApplicationID,
		Name:         fmt.Sprintf("%s-bluegreen-%d", app.Name, time.Now().Unix()),
		EnvName:      req.EnvName,
		BlueVersion:  blueImageTag,
		GreenVersion: req.GreenImageTag,
		ActiveColor:  "blue",
		Status:       "pending",
		Replicas:     req.Replicas,
	}

	if deploy.Replicas == 0 {
		deploy.Replicas = 2
	}

	if err := h.db.Create(&deploy).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "蓝绿部署已创建", "data": deploy})
}

// GetBlueGreenStatus 获取蓝绿部署状态
func (h *CanaryHandler) GetBlueGreenStatus(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	var deploy models.BlueGreenDeployment
	if err := h.db.First(&deploy, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "部署不存在"})
		return
	}

	var app models.Application
	h.db.Select("name").First(&app, deploy.AppID)

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{
		"id":              deploy.ID,
		"app_id":          deploy.AppID,
		"app_name":        app.Name,
		"env_name":        deploy.EnvName,
		"blue_image_tag":  deploy.BlueVersion,
		"green_image_tag": deploy.GreenVersion,
		"active_version":  deploy.ActiveColor,
		"status":          deploy.Status,
		"replicas":        deploy.Replicas,
		"created_at":      deploy.CreatedAt,
		"switched_at":     deploy.SwitchedAt,
	}})
}

// SwitchBlueGreenByID 通过ID切换蓝绿部署
func (h *CanaryHandler) SwitchBlueGreenByID(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	var deploy models.BlueGreenDeployment
	if err := h.db.First(&deploy, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "部署不存在"})
		return
	}

	var app models.Application
	if err := h.db.First(&app, deploy.AppID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "应用不存在"})
		return
	}

	// 切换颜色
	newColor := "green"
	newStatus := "switched"
	if deploy.ActiveColor == "green" {
		newColor = "blue"
		newStatus = "switched"
	}

	now := time.Now()
	h.db.Model(&deploy).Updates(map[string]interface{}{
		"active_color": newColor,
		"status":       newStatus,
		"switched_at":  &now,
	})

	// 同步到 K8s
	if app.K8sClusterID != nil && app.K8sNamespace != "" {
		h.syncBlueGreenRouting(c.Request.Context(), &app, &deploy, newColor)
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": fmt.Sprintf("已切换到 %s 版本", newColor)})
}

// RollbackBlueGreenByID 回滚蓝绿部署
func (h *CanaryHandler) RollbackBlueGreenByID(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	var deploy models.BlueGreenDeployment
	if err := h.db.First(&deploy, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "部署不存在"})
		return
	}

	var app models.Application
	if err := h.db.First(&app, deploy.AppID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "应用不存在"})
		return
	}

	// 回滚到蓝版本
	now := time.Now()
	h.db.Model(&deploy).Updates(map[string]interface{}{
		"active_color": "blue",
		"status":       "rolled_back",
		"switched_at":  &now,
	})

	// 同步到 K8s
	if app.K8sClusterID != nil && app.K8sNamespace != "" {
		h.syncBlueGreenRouting(c.Request.Context(), &app, &deploy, "blue")
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "已回滚到蓝版本"})
}

// CleanupBlueGreen 清理蓝绿部署旧版本
func (h *CanaryHandler) CleanupBlueGreen(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	var deploy models.BlueGreenDeployment
	if err := h.db.First(&deploy, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "部署不存在"})
		return
	}

	if deploy.Status != "switched" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "只有已切换状态的部署才能清理"})
		return
	}

	// 标记为已完成
	h.db.Model(&deploy).Update("status", "completed")

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "清理完成"})
}

// ListBlueGreenDeployments 获取蓝绿部署列表
func (h *CanaryHandler) ListBlueGreenDeployments(c *gin.Context) {
	app, err := h.getApp(c)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "应用不存在"})
		return
	}

	var deployments []models.BlueGreenDeployment
	h.db.Where("app_id = ?", app.ID).Order("created_at DESC").Find(&deployments)

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"items": deployments}})
}

// CreateBlueGreenDeployment 创建蓝绿部署
func (h *CanaryHandler) CreateBlueGreenDeployment(c *gin.Context) {
	app, err := h.getApp(c)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "应用不存在"})
		return
	}

	var deploy models.BlueGreenDeployment
	if err := c.ShouldBindJSON(&deploy); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	deploy.AppID = uint64(app.ID)
	deploy.Status = "blue_active"
	deploy.ActiveColor = "blue"

	if err := h.db.Create(&deploy).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "创建成功", "data": deploy})
}

// GetBlueGreenDeployment 获取蓝绿部署详情
func (h *CanaryHandler) GetBlueGreenDeployment(c *gin.Context) {
	app, err := h.getApp(c)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "应用不存在"})
		return
	}

	deployID, _ := strconv.ParseUint(c.Param("deployId"), 10, 64)
	var deploy models.BlueGreenDeployment
	if err := h.db.Where("id = ? AND app_id = ?", deployID, app.ID).First(&deploy).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "部署不存在"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": deploy})
}

// SwitchBlueGreen 切换蓝绿部署
func (h *CanaryHandler) SwitchBlueGreen(c *gin.Context) {
	app, err := h.getApp(c)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "应用不存在"})
		return
	}

	deployID, _ := strconv.ParseUint(c.Param("deployId"), 10, 64)
	var deploy models.BlueGreenDeployment
	if err := h.db.Where("id = ? AND app_id = ?", deployID, app.ID).First(&deploy).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "部署不存在"})
		return
	}

	// 切换颜色
	newColor := "green"
	newStatus := "green_active"
	if deploy.ActiveColor == "green" {
		newColor = "blue"
		newStatus = "blue_active"
	}

	now := time.Now()
	h.db.Model(&deploy).Updates(map[string]interface{}{
		"active_color": newColor,
		"status":       newStatus,
		"switched_at":  &now,
	})

	// 同步到 K8s
	if app.K8sClusterID != nil && app.K8sNamespace != "" {
		h.syncBlueGreenRouting(c.Request.Context(), app, &deploy, newColor)
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": fmt.Sprintf("已切换到 %s 环境", newColor)})
}

// DeleteBlueGreenDeployment 删除蓝绿部署
func (h *CanaryHandler) DeleteBlueGreenDeployment(c *gin.Context) {
	app, err := h.getApp(c)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "应用不存在"})
		return
	}

	deployID, _ := strconv.ParseUint(c.Param("deployId"), 10, 64)
	h.db.Where("id = ? AND app_id = ?", deployID, app.ID).Delete(&models.BlueGreenDeployment{})

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "删除成功"})
}

// syncBlueGreenRouting 同步蓝绿路由到 K8s
func (h *CanaryHandler) syncBlueGreenRouting(ctx context.Context, app *models.Application, deploy *models.BlueGreenDeployment, activeColor string) error {
	// 创建路由规则 - 100% 流量到活跃颜色
	rules := []models.TrafficRoutingRule{
		{
			Name:      fmt.Sprintf("bluegreen-%d", deploy.ID),
			RouteType: "weight",
			Destinations: models.JSONDestinations{
				{Subset: activeColor, Weight: 100},
			},
			Enabled: true,
		},
	}

	return h.istioService.SyncRoutingRules(ctx, *app.K8sClusterID, app.K8sNamespace, app, rules)
}

// IncrementCanaryWeight 增加金丝雀权重（供定时任务调用）
func (h *CanaryHandler) IncrementCanaryWeight(releaseID uint64) error {
	var release models.CanaryRelease
	if err := h.db.First(&release, releaseID).Error; err != nil {
		return err
	}

	if release.Status != "running" {
		return nil
	}

	newWeight := release.CurrentWeight + release.WeightIncrement
	if newWeight > release.TargetWeight {
		newWeight = release.TargetWeight
	}

	h.db.Model(&release).Update("current_weight", newWeight)

	// 获取应用信息并同步路由
	var app models.Application
	if err := h.db.First(&app, release.AppID).Error; err != nil {
		return err
	}

	if app.K8sClusterID != nil && app.K8sNamespace != "" {
		release.CurrentWeight = newWeight
		h.syncCanaryRouting(context.Background(), &app, &release)
	}

	// 如果达到目标权重，标记为完成
	if newWeight >= release.TargetWeight {
		now := time.Now()
		h.db.Model(&release).Updates(map[string]interface{}{
			"status":       "completed",
			"completed_at": &now,
		})
	}

	return nil
}

// SaveRuleVersion 保存规则版本（用于回滚）
func (h *CanaryHandler) SaveRuleVersion(appID uint64, ruleType string, ruleID uint64, content interface{}, operator string) error {
	// 获取当前最大版本号
	var maxVersion int
	h.db.Model(&models.TrafficRuleVersion{}).
		Where("app_id = ? AND rule_type = ? AND rule_id = ?", appID, ruleType, ruleID).
		Select("COALESCE(MAX(version), 0)").Scan(&maxVersion)

	contentJSON, _ := json.Marshal(content)
	version := models.TrafficRuleVersion{
		AppID:    appID,
		RuleType: ruleType,
		RuleID:   ruleID,
		Version:  maxVersion + 1,
		Content:  string(contentJSON),
		Operator: operator,
	}

	return h.db.Create(&version).Error
}
