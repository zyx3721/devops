package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"devops/internal/config"
	"devops/internal/models"
	"devops/internal/repository"
	"devops/internal/service/notification"
	"devops/pkg/ioc"
	"devops/pkg/logger"
	"devops/pkg/middleware"
)

var alertLog = logger.L().WithField("module", "alert")

func init() {
	ioc.Api.RegisterContainer("AlertHandler", &AlertApiHandler{})
}

type AlertApiHandler struct {
	handler *AlertHandler
}

func (h *AlertApiHandler) Init() error {
	cfg, err := config.LoadConfig()
	if err != nil {
		return err
	}

	db := cfg.GetDB()
	h.handler = NewAlertHandler(db)

	root := cfg.Application.GinRootRouter().Group("alert")

	// 分离公开接口和需要认证的接口
	// 1. 公开接口 (Trigger)
	root.POST("/trigger", h.handler.TriggerAlert)

	// 2. 需要认证的管理接口
	authorized := root.Group("")
	authorized.Use(middleware.AuthMiddleware())
	h.Register(authorized)

	return nil
}

func (h *AlertApiHandler) Register(r gin.IRouter) {
	// 告警配置 - 查看
	r.GET("/configs", h.handler.ListConfigs)
	r.GET("/configs/:id", h.handler.GetConfig)
	// 告警配置 - 管理
	r.POST("/configs", middleware.RequireAdmin(), h.handler.CreateConfig)
	r.PUT("/configs/:id", middleware.RequireAdmin(), h.handler.UpdateConfig)
	r.DELETE("/configs/:id", middleware.RequireAdmin(), h.handler.DeleteConfig)
	r.POST("/configs/:id/toggle", middleware.RequireAdmin(), h.handler.ToggleConfig)

	// 告警历史 - 查看和操作
	r.GET("/histories", h.handler.ListHistories)
	r.POST("/histories/:id/ack", h.handler.AckAlert)
	r.POST("/histories/:id/resolve", h.handler.ResolveAlert)

	// 告警静默 - 查看
	r.GET("/silences", h.handler.ListSilences)
	r.GET("/silences/:id", h.handler.GetSilence)
	// 告警静默 - 管理
	r.POST("/silences", middleware.RequireAdmin(), h.handler.CreateSilence)
	r.PUT("/silences/:id", middleware.RequireAdmin(), h.handler.UpdateSilence)
	r.DELETE("/silences/:id", middleware.RequireAdmin(), h.handler.DeleteSilence)
	r.POST("/silences/:id/cancel", middleware.RequireAdmin(), h.handler.CancelSilence)

	// 告警升级 - 查看
	r.GET("/escalations", h.handler.ListEscalations)
	r.GET("/escalations/:id", h.handler.GetEscalation)
	// 告警升级 - 管理
	r.POST("/escalations", middleware.RequireAdmin(), h.handler.CreateEscalation)
	r.PUT("/escalations/:id", middleware.RequireAdmin(), h.handler.UpdateEscalation)
	r.DELETE("/escalations/:id", middleware.RequireAdmin(), h.handler.DeleteEscalation)
	r.POST("/escalations/:id/toggle", middleware.RequireAdmin(), h.handler.ToggleEscalation)

	// 告警统计
	r.GET("/stats", h.handler.GetStats)
	r.GET("/trend", h.handler.GetTrend)

	// 告警触发接口已在 Init 中注册为免认证
}

type AlertHandler struct {
	configRepo     *repository.AlertConfigRepository
	historyRepo    *repository.AlertHistoryRepository
	silenceRepo    *repository.AlertSilenceRepository
	escalationRepo *repository.AlertEscalationRepository
	db             *gorm.DB
	tmplSvc        *notification.TemplateService
}

func NewAlertHandler(db *gorm.DB) *AlertHandler {
	return &AlertHandler{
		configRepo:     repository.NewAlertConfigRepository(db),
		historyRepo:    repository.NewAlertHistoryRepository(db),
		silenceRepo:    repository.NewAlertSilenceRepository(db),
		escalationRepo: repository.NewAlertEscalationRepository(db),
		db:             db,
		tmplSvc:        notification.NewTemplateService(repository.NewMessageTemplateRepository(db)),
	}
}

func (h *AlertHandler) ListConfigs(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	alertType := c.Query("type")

	configs, total, err := h.configRepo.List(c.Request.Context(), alertType, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "Success",
		"data": gin.H{
			"list":  configs,
			"total": total,
		},
	})
}

func (h *AlertHandler) GetConfig(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "Invalid ID"})
		return
	}

	config, err := h.configRepo.GetByID(c.Request.Context(), uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "Not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "Success", "data": config})
}

func (h *AlertHandler) CreateConfig(c *gin.Context) {
	var config models.AlertConfig
	if err := c.ShouldBindJSON(&config); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	config.ID = 0
	if userID, ok := middleware.GetUserID(c); ok {
		config.CreatedBy = &userID
	}

	if err := h.configRepo.Create(c.Request.Context(), &config); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "Success", "data": config})
}

func (h *AlertHandler) UpdateConfig(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "Invalid ID"})
		return
	}

	var config models.AlertConfig
	if err := c.ShouldBindJSON(&config); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	config.ID = uint(id)
	if err := h.configRepo.Update(c.Request.Context(), &config); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "Success", "data": config})
}

func (h *AlertHandler) DeleteConfig(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "Invalid ID"})
		return
	}

	if err := h.configRepo.Delete(c.Request.Context(), uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "Success"})
}

func (h *AlertHandler) ToggleConfig(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "Invalid ID"})
		return
	}

	config, err := h.configRepo.GetByID(c.Request.Context(), uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "Not found"})
		return
	}

	config.Enabled = !config.Enabled
	if err := h.configRepo.Update(c.Request.Context(), config); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "Success", "data": config})
}

func (h *AlertHandler) ListHistories(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	alertType := c.Query("type")
	ackStatus := c.Query("ack_status")

	histories, total, err := h.historyRepo.List(c.Request.Context(), alertType, ackStatus, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "Success",
		"data": gin.H{
			"list":  histories,
			"total": total,
		},
	})
}

// AckAlert 确认告警
func (h *AlertHandler) AckAlert(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "Invalid ID"})
		return
	}

	history, err := h.historyRepo.GetByID(c.Request.Context(), uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "Not found"})
		return
	}

	if history.AckStatus != "pending" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "告警已被处理"})
		return
	}

	now := time.Now()
	history.AckStatus = "acked"
	history.AckAt = &now
	if userID, ok := middleware.GetUserID(c); ok {
		history.AckBy = &userID
	}

	if err := h.historyRepo.Update(c.Request.Context(), history); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "Success", "data": history})
}

// ResolveAlert 解决告警
func (h *AlertHandler) ResolveAlert(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "Invalid ID"})
		return
	}

	var req struct {
		Comment string `json:"comment"`
	}
	c.ShouldBindJSON(&req)

	history, err := h.historyRepo.GetByID(c.Request.Context(), uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "Not found"})
		return
	}

	if history.AckStatus == "resolved" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "告警已解决"})
		return
	}

	now := time.Now()
	history.AckStatus = "resolved"
	history.ResolvedAt = &now
	history.ResolveComment = req.Comment
	if userID, ok := middleware.GetUserID(c); ok {
		history.ResolvedBy = &userID
	}

	if err := h.historyRepo.Update(c.Request.Context(), history); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "Success", "data": history})
}

// ========== 告警静默 ==========

func (h *AlertHandler) ListSilences(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	status := c.Query("status")

	// 先过期旧规则
	h.silenceRepo.ExpireOldSilences(c.Request.Context())

	silences, total, err := h.silenceRepo.List(c.Request.Context(), status, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "Success",
		"data": gin.H{
			"list":  silences,
			"total": total,
		},
	})
}

func (h *AlertHandler) GetSilence(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "Invalid ID"})
		return
	}

	silence, err := h.silenceRepo.GetByID(c.Request.Context(), uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "Not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "Success", "data": silence})
}

func (h *AlertHandler) CreateSilence(c *gin.Context) {
	var silence models.AlertSilence
	if err := c.ShouldBindJSON(&silence); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	silence.ID = 0
	silence.Status = "active"
	if userID, ok := middleware.GetUserID(c); ok {
		silence.CreatedBy = &userID
	}

	if err := h.silenceRepo.Create(c.Request.Context(), &silence); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "Success", "data": silence})
}

func (h *AlertHandler) UpdateSilence(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "Invalid ID"})
		return
	}

	var silence models.AlertSilence
	if err := c.ShouldBindJSON(&silence); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	silence.ID = uint(id)
	if err := h.silenceRepo.Update(c.Request.Context(), &silence); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "Success", "data": silence})
}

func (h *AlertHandler) DeleteSilence(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "Invalid ID"})
		return
	}

	if err := h.silenceRepo.Delete(c.Request.Context(), uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "Success"})
}

func (h *AlertHandler) CancelSilence(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "Invalid ID"})
		return
	}

	silence, err := h.silenceRepo.GetByID(c.Request.Context(), uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "Not found"})
		return
	}

	silence.Status = "cancelled"
	if err := h.silenceRepo.Update(c.Request.Context(), silence); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "Success", "data": silence})
}

// ========== 告警升级 ==========

func (h *AlertHandler) ListEscalations(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	escalations, total, err := h.escalationRepo.List(c.Request.Context(), page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "Success",
		"data": gin.H{
			"list":  escalations,
			"total": total,
		},
	})
}

func (h *AlertHandler) GetEscalation(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "Invalid ID"})
		return
	}

	escalation, err := h.escalationRepo.GetByID(c.Request.Context(), uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "Not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "Success", "data": escalation})
}

func (h *AlertHandler) CreateEscalation(c *gin.Context) {
	var escalation models.AlertEscalation
	if err := c.ShouldBindJSON(&escalation); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	escalation.ID = 0
	if userID, ok := middleware.GetUserID(c); ok {
		escalation.CreatedBy = &userID
	}

	if err := h.escalationRepo.Create(c.Request.Context(), &escalation); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "Success", "data": escalation})
}

func (h *AlertHandler) UpdateEscalation(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "Invalid ID"})
		return
	}

	var escalation models.AlertEscalation
	if err := c.ShouldBindJSON(&escalation); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	escalation.ID = uint(id)
	if err := h.escalationRepo.Update(c.Request.Context(), &escalation); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "Success", "data": escalation})
}

func (h *AlertHandler) DeleteEscalation(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "Invalid ID"})
		return
	}

	if err := h.escalationRepo.Delete(c.Request.Context(), uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "Success"})
}

func (h *AlertHandler) ToggleEscalation(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "Invalid ID"})
		return
	}

	escalation, err := h.escalationRepo.GetByID(c.Request.Context(), uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "Not found"})
		return
	}

	escalation.Enabled = !escalation.Enabled
	if err := h.escalationRepo.Update(c.Request.Context(), escalation); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "Success", "data": escalation})
}

func (h *AlertHandler) GetStats(c *gin.Context) {
	type StatItem struct {
		Name  string `json:"name"`
		Count int64  `json:"count"`
	}

	// 按类型统计告警数
	var typeStats []StatItem
	h.db.Raw(`
		SELECT type as name, COUNT(*) as count 
		FROM alert_histories 
		WHERE created_at >= DATE_SUB(NOW(), INTERVAL 7 DAY)
		GROUP BY type
	`).Scan(&typeStats)

	// 按级别统计
	var levelStats []StatItem
	h.db.Raw(`
		SELECT level as name, COUNT(*) as count 
		FROM alert_histories 
		WHERE created_at >= DATE_SUB(NOW(), INTERVAL 7 DAY)
		GROUP BY level
	`).Scan(&levelStats)

	// 按确认状态统计
	var ackStats []StatItem
	h.db.Raw(`
		SELECT ack_status as name, COUNT(*) as count 
		FROM alert_histories 
		WHERE created_at >= DATE_SUB(NOW(), INTERVAL 7 DAY)
		GROUP BY ack_status
	`).Scan(&ackStats)

	// 今日告警数
	var todayCount int64
	h.db.Raw(`SELECT COUNT(*) FROM alert_histories WHERE DATE(created_at) = CURDATE()`).Scan(&todayCount)

	// 待处理告警数
	var pendingCount int64
	h.db.Raw(`SELECT COUNT(*) FROM alert_histories WHERE ack_status = 'pending'`).Scan(&pendingCount)

	// 启用的配置数
	var enabledCount int64
	h.db.Raw(`SELECT COUNT(*) FROM alert_configs WHERE enabled = true`).Scan(&enabledCount)

	// 活跃静默规则数
	var activeSilenceCount int64
	h.db.Raw(`SELECT COUNT(*) FROM alert_silences WHERE status = 'active' AND start_time <= NOW() AND end_time >= NOW()`).Scan(&activeSilenceCount)

	// 启用的升级规则数
	var enabledEscalationCount int64
	h.db.Raw(`SELECT COUNT(*) FROM alert_escalations WHERE enabled = true`).Scan(&enabledEscalationCount)

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "Success",
		"data": gin.H{
			"type_stats":               typeStats,
			"level_stats":              levelStats,
			"ack_stats":                ackStats,
			"today_count":              todayCount,
			"pending_count":            pendingCount,
			"enabled_count":            enabledCount,
			"active_silence_count":     activeSilenceCount,
			"enabled_escalation_count": enabledEscalationCount,
		},
	})
}

// GetTrend 获取告警趋势
func (h *AlertHandler) GetTrend(c *gin.Context) {
	days, _ := strconv.Atoi(c.DefaultQuery("days", "7"))
	if days <= 0 || days > 30 {
		days = 7
	}

	type TrendItem struct {
		Date  string `json:"date"`
		Count int64  `json:"count"`
	}

	var items []TrendItem
	h.db.Raw(`
		SELECT DATE_FORMAT(created_at, '%m-%d') as date, COUNT(*) as count 
		FROM alert_histories 
		WHERE created_at >= DATE_SUB(NOW(), INTERVAL ? DAY)
		GROUP BY DATE_FORMAT(created_at, '%m-%d')
		ORDER BY MIN(created_at)
	`, days).Scan(&items)

	// 补全没有数据的日期
	dateMap := make(map[string]int64)
	for _, item := range items {
		dateMap[item.Date] = item.Count
	}

	result := make([]TrendItem, 0, days)
	var total int64
	now := time.Now()
	for i := days - 1; i >= 0; i-- {
		d := now.AddDate(0, 0, -i)
		dateStr := d.Format("01-02")
		count := dateMap[dateStr]
		total += count
		result = append(result, TrendItem{Date: dateStr, Count: count})
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "Success",
		"data": gin.H{
			"items": result,
			"total": total,
		},
	})
}
