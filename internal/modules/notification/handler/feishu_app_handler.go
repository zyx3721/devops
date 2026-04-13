package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"devops/internal/config"
	"devops/internal/models"
)

// 飞书应用管理

func (h *FeishuHandler) ListApps(c *gin.Context) {
	if h.appRepo == nil {
		c.JSON(http.StatusOK, gin.H{"code": 0, "message": "Success", "data": gin.H{"list": []interface{}{}, "total": 0}})
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "100"))

	list, total, err := h.appRepo.List(c.Request.Context(), page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "Success",
		"data": gin.H{
			"list":  list,
			"total": total,
		},
	})
}

func (h *FeishuHandler) GetApp(c *gin.Context) {
	if h.appRepo == nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "Not found"})
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "Invalid ID"})
		return
	}

	app, err := h.appRepo.GetByID(c.Request.Context(), uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "Not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "Success", "data": app})
}

func (h *FeishuHandler) CreateApp(c *gin.Context) {
	if h.appRepo == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "Repository not initialized"})
		return
	}

	var app models.FeishuApp
	if err := c.ShouldBindJSON(&app); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	if app.AppID == "" || app.AppSecret == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "app_id and app_secret are required"})
		return
	}

	app.ID = 0
	if app.Status == "" {
		app.Status = "active"
	}

	if err := h.appRepo.Create(c.Request.Context(), &app); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "Success", "data": app})
}

func (h *FeishuHandler) UpdateApp(c *gin.Context) {
	if h.appRepo == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "Repository not initialized"})
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "Invalid ID"})
		return
	}

	var app models.FeishuApp
	if err := c.ShouldBindJSON(&app); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	app.ID = uint(id)
	if err := h.appRepo.Update(c.Request.Context(), &app); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "Success", "data": app})
}

func (h *FeishuHandler) DeleteApp(c *gin.Context) {
	if h.appRepo == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "Repository not initialized"})
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "Invalid ID"})
		return
	}

	if err := h.appRepo.Delete(c.Request.Context(), uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "Success"})
}

func (h *FeishuHandler) SetDefaultApp(c *gin.Context) {
	if h.appRepo == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "Repository not initialized"})
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "Invalid ID"})
		return
	}

	if err := h.appRepo.SetDefault(c.Request.Context(), uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "Success"})
}

// GetAppBindings 获取飞书应用绑定的 Jenkins 实例和 K8s 集群
func (h *FeishuHandler) GetAppBindings(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "Invalid ID"})
		return
	}

	cfg, _ := config.LoadConfig()
	db := cfg.GetDB()

	// 获取绑定的 Jenkins 实例
	var jenkinsBindings []struct {
		ID   uint   `json:"id"`
		Name string `json:"name"`
		URL  string `json:"url"`
	}
	db.Table("jenkins_instances").
		Select("jenkins_instances.id, jenkins_instances.name, jenkins_instances.url").
		Joins("JOIN jenkins_feishu_apps ON jenkins_feishu_apps.jenkins_instance_id = jenkins_instances.id").
		Where("jenkins_feishu_apps.feishu_app_id = ?", id).
		Scan(&jenkinsBindings)

	// 获取绑定的 K8s 集群
	var k8sBindings []struct {
		ID   uint   `json:"id"`
		Name string `json:"name"`
	}
	db.Table("k8s_clusters").
		Select("k8s_clusters.id, k8s_clusters.name").
		Joins("JOIN k8s_cluster_feishu_apps ON k8s_cluster_feishu_apps.k8s_cluster_id = k8s_clusters.id").
		Where("k8s_cluster_feishu_apps.feishu_app_id = ?", id).
		Scan(&k8sBindings)

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "Success",
		"data": gin.H{
			"jenkins_instances": jenkinsBindings,
			"k8s_clusters":      k8sBindings,
		},
	})
}

// 飞书机器人管理

func (h *FeishuHandler) ListBots(c *gin.Context) {
	if h.botRepo == nil {
		c.JSON(http.StatusOK, gin.H{"code": 0, "message": "Success", "data": gin.H{"list": []interface{}{}, "total": 0}})
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "100"))

	list, total, err := h.botRepo.List(c.Request.Context(), page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "Success",
		"data": gin.H{
			"list":  list,
			"total": total,
		},
	})
}

func (h *FeishuHandler) GetBot(c *gin.Context) {
	if h.botRepo == nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "Not found"})
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "Invalid ID"})
		return
	}

	bot, err := h.botRepo.GetByID(c.Request.Context(), uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "Not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "Success", "data": bot})
}

func (h *FeishuHandler) CreateBot(c *gin.Context) {
	if h.botRepo == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "Repository not initialized"})
		return
	}

	var bot models.FeishuBot
	if err := c.ShouldBindJSON(&bot); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	if bot.Name == "" || bot.WebhookURL == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "name and webhook_url are required"})
		return
	}

	bot.ID = 0
	if bot.Status == "" {
		bot.Status = "active"
	}

	if err := h.botRepo.Create(c.Request.Context(), &bot); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "Success", "data": bot})
}

func (h *FeishuHandler) UpdateBot(c *gin.Context) {
	if h.botRepo == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "Repository not initialized"})
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "Invalid ID"})
		return
	}

	var bot models.FeishuBot
	if err := c.ShouldBindJSON(&bot); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	bot.ID = uint(id)
	if err := h.botRepo.Update(c.Request.Context(), &bot); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "Success", "data": bot})
}

func (h *FeishuHandler) DeleteBot(c *gin.Context) {
	if h.botRepo == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "Repository not initialized"})
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "Invalid ID"})
		return
	}

	if err := h.botRepo.Delete(c.Request.Context(), uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "Success"})
}
