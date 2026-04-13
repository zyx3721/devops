package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"devops/internal/service/feature"
)

// FeatureFlagHandler 功能开关处理器
type FeatureFlagHandler struct {
	service feature.FeatureFlagService
}

// NewFeatureFlagHandler 创建功能开关处理器
func NewFeatureFlagHandler(db *gorm.DB) *FeatureFlagHandler {
	return &FeatureFlagHandler{
		service: feature.NewFeatureFlagService(db),
	}
}

// RegisterRoutes 注册路由
func (h *FeatureFlagHandler) RegisterRoutes(r *gin.RouterGroup) {
	g := r.Group("/feature-flags")
	{
		g.GET("", h.List)
		g.POST("", h.Create)
		g.GET("/:name", h.Get)
		g.PUT("/:name", h.Update)
		g.DELETE("/:name", h.Delete)
		g.GET("/:name/check", h.Check)
		g.GET("/stats", h.Stats)
	}
}

// List 获取功能开关列表
func (h *FeatureFlagHandler) List(c *gin.Context) {
	flags, err := h.service.List(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"items": flags,
			"total": len(flags),
		},
	})
}

// Create 创建功能开关
func (h *FeatureFlagHandler) Create(c *gin.Context) {
	var req feature.CreateFeatureFlagRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误: " + err.Error()})
		return
	}

	flag, err := h.service.Create(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "创建成功",
		"data":    flag,
	})
}

// Get 获取功能开关详情
func (h *FeatureFlagHandler) Get(c *gin.Context) {
	name := c.Param("name")
	flag, err := h.service.Get(c.Request.Context(), name)
	if err != nil {
		if err == feature.ErrFeatureFlagNotFound {
			c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "功能开关不存在"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": flag})
}

// Update 更新功能开关
func (h *FeatureFlagHandler) Update(c *gin.Context) {
	name := c.Param("name")
	var req feature.UpdateFeatureFlagRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误: " + err.Error()})
		return
	}

	if err := h.service.Update(c.Request.Context(), name, &req); err != nil {
		if err == feature.ErrFeatureFlagNotFound {
			c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "功能开关不存在"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "更新成功"})
}

// Delete 删除功能开关
func (h *FeatureFlagHandler) Delete(c *gin.Context) {
	name := c.Param("name")
	if err := h.service.Delete(c.Request.Context(), name); err != nil {
		if err == feature.ErrFeatureFlagNotFound {
			c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "功能开关不存在"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "删除成功"})
}

// Check 检查功能开关状态
func (h *FeatureFlagHandler) Check(c *gin.Context) {
	name := c.Param("name")
	tenantIDStr := c.Query("tenant_id")
	userIDStr := c.Query("user_id")

	var enabled bool
	if userIDStr != "" {
		userID, _ := strconv.ParseUint(userIDStr, 10, 64)
		enabled = h.service.IsEnabledForUser(c.Request.Context(), name, uint(userID))
	} else if tenantIDStr != "" {
		tenantID, _ := strconv.ParseUint(tenantIDStr, 10, 64)
		enabled = h.service.IsEnabled(c.Request.Context(), name, uint(tenantID))
	} else {
		enabled = h.service.IsEnabled(c.Request.Context(), name, 0)
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"name":    name,
			"enabled": enabled,
		},
	})
}

// Stats 获取统计信息
func (h *FeatureFlagHandler) Stats(c *gin.Context) {
	flags, err := h.service.List(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	var total, enabled, disabled, rollout int
	total = len(flags)
	for _, f := range flags {
		if f.IsEnabled {
			if f.RolloutPercentage > 0 && f.RolloutPercentage < 100 {
				rollout++
			} else {
				enabled++
			}
		} else {
			disabled++
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"total":    total,
			"enabled":  enabled,
			"disabled": disabled,
			"rollout":  rollout,
		},
	})
}
