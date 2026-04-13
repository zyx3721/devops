package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"devops/internal/config"
	"devops/internal/repository"
	"devops/pkg/ioc"
	"devops/pkg/middleware"
)

func init() {
	ioc.Api.RegisterContainer("AuditHandler", &AuditApiHandler{})
}

type AuditApiHandler struct {
	handler *AuditHandler
}

func (h *AuditApiHandler) Init() error {
	cfg, err := config.LoadConfig()
	if err != nil {
		return err
	}

	db := cfg.GetDB()
	h.handler = NewAuditHandler(db)

	root := cfg.Application.GinRootRouter().Group("audit")
	root.Use(middleware.AuthMiddleware())
	h.Register(root)

	return nil
}

func (h *AuditApiHandler) Register(r gin.IRouter) {
	r.GET("/logs", h.handler.ListLogs)
	r.GET("/stats", h.handler.GetStats)
}

type AuditHandler struct {
	repo *repository.AuditLogRepository
	db   *gorm.DB
}

func NewAuditHandler(db *gorm.DB) *AuditHandler {
	return &AuditHandler{
		repo: repository.NewAuditLogRepository(db),
		db:   db,
	}
}

func (h *AuditHandler) ListLogs(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	filter := repository.AuditLogFilter{
		Username: c.Query("username"),
		Action:   c.Query("action"),
		Resource: c.Query("resource"),
		Status:   c.Query("status"),
	}

	if startTime := c.Query("start_time"); startTime != "" {
		if t, err := time.Parse("2006-01-02", startTime); err == nil {
			filter.StartTime = t
		}
	}
	if endTime := c.Query("end_time"); endTime != "" {
		if t, err := time.Parse("2006-01-02", endTime); err == nil {
			filter.EndTime = t.Add(24 * time.Hour)
		}
	}

	logs, total, err := h.repo.List(c.Request.Context(), filter, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "Success",
		"data": gin.H{
			"list":  logs,
			"total": total,
		},
	})
}

func (h *AuditHandler) GetStats(c *gin.Context) {
	type StatItem struct {
		Name  string `json:"name"`
		Count int64  `json:"count"`
	}

	// 按操作类型统计
	var actionStats []StatItem
	h.db.Raw(`
		SELECT action as name, COUNT(*) as count 
		FROM audit_logs 
		WHERE created_at >= DATE_SUB(NOW(), INTERVAL 7 DAY)
		GROUP BY action 
		ORDER BY count DESC
	`).Scan(&actionStats)

	// 按资源类型统计
	var resourceStats []StatItem
	h.db.Raw(`
		SELECT resource_type as name, COUNT(*) as count 
		FROM audit_logs 
		WHERE created_at >= DATE_SUB(NOW(), INTERVAL 7 DAY)
		GROUP BY resource_type 
		ORDER BY count DESC
	`).Scan(&resourceStats)

	// 按用户统计
	var userStats []StatItem
	h.db.Raw(`
		SELECT username as name, COUNT(*) as count 
		FROM audit_logs 
		WHERE created_at >= DATE_SUB(NOW(), INTERVAL 7 DAY) AND username != ''
		GROUP BY username 
		ORDER BY count DESC
		LIMIT 10
	`).Scan(&userStats)

	// 今日操作数
	var todayCount int64
	h.db.Raw(`SELECT COUNT(*) FROM audit_logs WHERE DATE(created_at) = CURDATE()`).Scan(&todayCount)

	// 本周操作数
	var weekCount int64
	h.db.Raw(`SELECT COUNT(*) FROM audit_logs WHERE created_at >= DATE_SUB(NOW(), INTERVAL 7 DAY)`).Scan(&weekCount)

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "Success",
		"data": gin.H{
			"action_stats":   actionStats,
			"resource_stats": resourceStats,
			"user_stats":     userStats,
			"today_count":    todayCount,
			"week_count":     weekCount,
		},
	})
}
