package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"devops/internal/config"
	"devops/internal/models"
	apperrors "devops/pkg/errors"
	"devops/pkg/ioc"
)

func init() {
	ioc.Api.RegisterContainer("DashboardHandler", &DashboardApiHandler{})
}

type DashboardApiHandler struct {
	handler *DashboardHandler
}

func (h *DashboardApiHandler) Init() error {
	cfg, err := config.LoadConfig()
	if err != nil {
		return err
	}

	h.handler = NewDashboardHandler(cfg)

	root := cfg.Application.GinRootRouter().Group("dashboard")
	h.Register(root)

	return nil
}

func (h *DashboardApiHandler) Register(r gin.IRouter) {
	r.GET("/stats", h.handler.GetStats)
	r.GET("/health-overview", h.handler.GetHealthOverview)
	r.GET("/recent-alerts", h.handler.GetRecentAlerts)
	r.GET("/recent-audits", h.handler.GetRecentAudits)
}

type DashboardHandler struct {
	cfg *config.Config
}

func NewDashboardHandler(cfg *config.Config) *DashboardHandler {
	return &DashboardHandler{cfg: cfg}
}

type DashboardStats struct {
	JenkinsInstances int64 `json:"jenkinsInstances"`
	K8sClusters      int64 `json:"k8sClusters"`
	Users            int64 `json:"users"`
	HealthChecks     int64 `json:"healthChecks"`
	AlertsToday      int64 `json:"alertsToday"`
	AuditsToday      int64 `json:"auditsToday"`
}

func (h *DashboardHandler) GetStats(c *gin.Context) {
	db := h.cfg.GetDB()

	var stats DashboardStats
	today := time.Now().Format("2006-01-02")

	db.Model(&models.JenkinsInstance{}).Where("deleted_at IS NULL").Count(&stats.JenkinsInstances)
	db.Model(&models.K8sCluster{}).Where("deleted_at IS NULL").Count(&stats.K8sClusters)
	db.Model(&models.User{}).Where("deleted_at IS NULL").Count(&stats.Users)
	db.Model(&models.HealthCheckConfig{}).Where("deleted_at IS NULL AND enabled = ?", true).Count(&stats.HealthChecks)
	db.Model(&models.AlertHistory{}).Where("DATE(created_at) = ?", today).Count(&stats.AlertsToday)
	db.Model(&models.AuditLog{}).Where("DATE(created_at) = ?", today).Count(&stats.AuditsToday)

	c.JSON(http.StatusOK, gin.H{
		"code":    apperrors.Success,
		"message": "success",
		"data":    stats,
	})
}

type HealthOverview struct {
	Status    string `json:"status"`
	Healthy   int64  `json:"healthy"`
	Unhealthy int64  `json:"unhealthy"`
	Unknown   int64  `json:"unknown"`
	Total     int64  `json:"total"`
}

func (h *DashboardHandler) GetHealthOverview(c *gin.Context) {
	db := h.cfg.GetDB()

	var overview HealthOverview

	db.Model(&models.HealthCheckConfig{}).Where("deleted_at IS NULL AND enabled = ?", true).Count(&overview.Total)
	db.Model(&models.HealthCheckConfig{}).Where("deleted_at IS NULL AND enabled = ? AND last_status = ?", true, "healthy").Count(&overview.Healthy)
	db.Model(&models.HealthCheckConfig{}).Where("deleted_at IS NULL AND enabled = ? AND last_status = ?", true, "unhealthy").Count(&overview.Unhealthy)
	overview.Unknown = overview.Total - overview.Healthy - overview.Unhealthy

	if overview.Unhealthy > 0 {
		overview.Status = "unhealthy"
	} else if overview.Unknown > 0 && overview.Healthy == 0 {
		overview.Status = "unknown"
	} else {
		overview.Status = "healthy"
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    apperrors.Success,
		"message": "success",
		"data":    overview,
	})
}

type RecentAlert struct {
	ID        uint      `json:"id"`
	Type      string    `json:"type"`
	Level     string    `json:"level"`
	Title     string    `json:"title"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

func (h *DashboardHandler) GetRecentAlerts(c *gin.Context) {
	db := h.cfg.GetDB()

	var alerts []RecentAlert
	db.Model(&models.AlertHistory{}).
		Select("id, type, level, title, status, created_at").
		Order("created_at DESC").
		Limit(10).
		Find(&alerts)

	c.JSON(http.StatusOK, gin.H{
		"code":    apperrors.Success,
		"message": "success",
		"data":    alerts,
	})
}

type RecentAudit struct {
	ID           uint      `json:"id"`
	Username     string    `json:"username"`
	Action       string    `json:"action"`
	ResourceType string    `json:"resource_type"`
	ResourceID   string    `json:"resource_id"`
	CreatedAt    time.Time `json:"created_at"`
}

func (h *DashboardHandler) GetRecentAudits(c *gin.Context) {
	db := h.cfg.GetDB()

	var audits []RecentAudit
	db.Model(&models.AuditLog{}).
		Select("id, username, action, resource_type, resource_id, created_at").
		Order("created_at DESC").
		Limit(10).
		Find(&audits)

	c.JSON(http.StatusOK, gin.H{
		"code":    apperrors.Success,
		"message": "success",
		"data":    audits,
	})
}
