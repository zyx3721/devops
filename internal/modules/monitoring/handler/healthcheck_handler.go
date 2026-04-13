package handler

import (
	"context"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"devops/internal/config"
	"devops/internal/models"
	"devops/internal/repository"
	"devops/internal/service/healthcheck"
	"devops/pkg/ioc"
	"devops/pkg/logger"
	"devops/pkg/middleware"
	"devops/pkg/response"
)

var hcLog = logger.L().WithField("module", "healthcheck")

func init() {
	ioc.Api.RegisterContainer("HealthCheckHandler", &HealthCheckApiHandler{})
}

type HealthCheckApiHandler struct {
	handler *HealthCheckHandler
}

func (h *HealthCheckApiHandler) Init() error {
	cfg, err := config.LoadConfig()
	if err != nil {
		return err
	}

	db := cfg.GetDB()

	// 初始化并启动健康检查服务
	checker := healthcheck.InitHealthChecker(db, cfg)
	checker.Start()

	h.handler = NewHealthCheckHandler(db)

	root := cfg.Application.GinRootRouter().Group("healthcheck")
	root.Use(middleware.AuthMiddleware())
	h.Register(root)

	return nil
}

func (h *HealthCheckApiHandler) Register(r gin.IRouter) {
	// 配置管理 - 查看
	r.GET("/configs", h.handler.ListConfigs)
	r.GET("/configs/:id", h.handler.GetConfig)
	// 配置管理 - 需要管理员
	r.POST("/configs", middleware.RequireAdmin(), h.handler.CreateConfig)
	r.PUT("/configs/:id", middleware.RequireAdmin(), h.handler.UpdateConfig)
	r.DELETE("/configs/:id", middleware.RequireAdmin(), h.handler.DeleteConfig)
	r.POST("/configs/:id/toggle", middleware.RequireAdmin(), h.handler.ToggleConfig)
	r.POST("/configs/:id/check", middleware.RequireAdmin(), h.handler.CheckNow)

	// SSL域名查询
	r.GET("/ssl-domains", h.handler.ListSSLDomains)
	r.GET("/ssl-domains/expiring", h.handler.GetExpiringCerts)
	r.GET("/ssl-domains/export", h.handler.ExportCertReport)

	// SSL域名批量管理 - 需要管理员
	r.POST("/ssl-domains/import", middleware.RequireAdmin(), h.handler.ImportSSLDomains)
	r.PUT("/ssl-domains/alert-config", middleware.RequireAdmin(), h.handler.BatchUpdateAlertConfig)

	// 历史记录
	r.GET("/histories", h.handler.ListHistories)

	// 统计
	r.GET("/stats", h.handler.GetStats)
	r.GET("/status", h.handler.GetOverallStatus)
}

type HealthCheckHandler struct {
	configRepo  *repository.HealthCheckConfigRepository
	historyRepo *repository.HealthCheckHistoryRepository
	db          *gorm.DB
}

func NewHealthCheckHandler(db *gorm.DB) *HealthCheckHandler {
	return &HealthCheckHandler{
		configRepo:  repository.NewHealthCheckConfigRepository(db),
		historyRepo: repository.NewHealthCheckHistoryRepository(db),
		db:          db,
	}
}

func (h *HealthCheckHandler) ListConfigs(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	// 构建过滤条件
	filters := &repository.ListFilters{
		Type:       c.Query("type"),
		AlertLevel: c.Query("alert_level"),
		Keyword:    c.Query("keyword"),
	}

	// 解析剩余天数过滤条件
	if maxDaysStr := c.Query("max_days_remaining"); maxDaysStr != "" {
		if maxDays, err := strconv.Atoi(maxDaysStr); err == nil && maxDays > 0 {
			filters.MaxDaysRemaining = maxDays
		}
	}

	// 解析排序参数
	sortBy := c.Query("sort_by")
	sortOrder := c.DefaultQuery("sort_order", "desc")
	if sortBy != "" {
		// 验证排序字段，防止SQL注入
		validSortFields := map[string]bool{
			"cert_days_remaining": true,
			"created_at":          true,
			"name":                true,
			"last_check_at":       true,
		}
		if validSortFields[sortBy] {
			if sortOrder == "asc" {
				filters.SortBy = sortBy + " ASC"
			} else {
				filters.SortBy = sortBy + " DESC"
			}
		}
	}

	configs, total, err := h.configRepo.ListWithFilters(c.Request.Context(), filters, page, pageSize)
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

func (h *HealthCheckHandler) GetConfig(c *gin.Context) {
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

func (h *HealthCheckHandler) CreateConfig(c *gin.Context) {
	var config models.HealthCheckConfig
	if err := c.ShouldBindJSON(&config); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	config.ID = 0
	config.LastStatus = "unknown"
	if userID, ok := middleware.GetUserID(c); ok {
		config.CreatedBy = &userID
	}

	if err := h.configRepo.Create(c.Request.Context(), &config); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "Success", "data": config})
}

func (h *HealthCheckHandler) UpdateConfig(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "Invalid ID"})
		return
	}

	// 先获取原有配置
	existing, err := h.configRepo.GetByID(c.Request.Context(), uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "配置不存在"})
		return
	}

	// 绑定更新数据
	var input models.HealthCheckConfig
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	// 更新字段
	existing.Name = input.Name
	existing.Type = input.Type
	existing.TargetID = input.TargetID
	existing.TargetName = input.TargetName
	existing.URL = input.URL
	existing.Interval = input.Interval
	existing.Timeout = input.Timeout
	existing.RetryCount = input.RetryCount
	existing.Enabled = input.Enabled
	existing.AlertEnabled = input.AlertEnabled
	existing.AlertPlatform = input.AlertPlatform
	existing.AlertBotID = input.AlertBotID

	if err := h.configRepo.Update(c.Request.Context(), existing); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "Success", "data": existing})
}

func (h *HealthCheckHandler) DeleteConfig(c *gin.Context) {
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

func (h *HealthCheckHandler) ToggleConfig(c *gin.Context) {
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

func (h *HealthCheckHandler) CheckNow(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "Invalid ID"})
		return
	}

	checker := healthcheck.GetHealthChecker()
	if checker == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "Health checker not initialized"})
		return
	}

	history, err := checker.CheckNow(c.Request.Context(), uint(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "Success", "data": history})
}

func (h *HealthCheckHandler) ListHistories(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	configID, _ := strconv.ParseUint(c.Query("config_id"), 10, 64)

	histories, total, err := h.historyRepo.List(c.Request.Context(), uint(configID), page, pageSize)
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

func (h *HealthCheckHandler) GetStats(c *gin.Context) {
	type StatItem struct {
		Name  string `json:"name"`
		Count int64  `json:"count"`
	}

	// 按类型统计配置数
	var typeStats []StatItem
	h.db.Raw(`SELECT type as name, COUNT(*) as count FROM health_check_configs GROUP BY type`).Scan(&typeStats)

	// 按状态统计
	var statusStats []StatItem
	h.db.Raw(`SELECT last_status as name, COUNT(*) as count FROM health_check_configs WHERE enabled = true GROUP BY last_status`).Scan(&statusStats)

	// 启用的配置数
	var enabledCount int64
	h.db.Raw(`SELECT COUNT(*) FROM health_check_configs WHERE enabled = true`).Scan(&enabledCount)

	// 健康的配置数
	var healthyCount int64
	h.db.Raw(`SELECT COUNT(*) FROM health_check_configs WHERE enabled = true AND last_status = 'healthy'`).Scan(&healthyCount)

	// 不健康的配置数
	var unhealthyCount int64
	h.db.Raw(`SELECT COUNT(*) FROM health_check_configs WHERE enabled = true AND last_status = 'unhealthy'`).Scan(&unhealthyCount)

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "Success",
		"data": gin.H{
			"type_stats":      typeStats,
			"status_stats":    statusStats,
			"enabled_count":   enabledCount,
			"healthy_count":   healthyCount,
			"unhealthy_count": unhealthyCount,
		},
	})
}

func (h *HealthCheckHandler) GetOverallStatus(c *gin.Context) {
	var configs []models.HealthCheckConfig
	h.db.Where("enabled = ?", true).Find(&configs)

	var healthy, unhealthy, unknown int
	for _, config := range configs {
		switch config.LastStatus {
		case "healthy":
			healthy++
		case "unhealthy":
			unhealthy++
		default:
			unknown++
		}
	}

	status := "healthy"
	if unhealthy > 0 {
		status = "unhealthy"
	} else if unknown > 0 && healthy == 0 {
		status = "unknown"
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "Success",
		"data": gin.H{
			"status":    status,
			"healthy":   healthy,
			"unhealthy": unhealthy,
			"unknown":   unknown,
			"total":     len(configs),
		},
	})
}

// ImportDomainsRequest 批量导入域名请求
type ImportDomainsRequest struct {
	Domains       []string `json:"domains" binding:"required"`               // 域名列表
	Interval      int      `json:"interval" binding:"required,min=60"`       // 检查间隔(秒)，最小60秒
	Timeout       int      `json:"timeout" binding:"required,min=1,max=300"` // 超时时间(秒)
	CriticalDays  int      `json:"critical_days" binding:"required,min=1"`   // 严重告警阈值（天）
	WarningDays   int      `json:"warning_days" binding:"required,min=1"`    // 警告告警阈值（天）
	NoticeDays    int      `json:"notice_days" binding:"required,min=1"`     // 提醒告警阈值（天）
	AlertEnabled  bool     `json:"alert_enabled"`                            // 是否启用告警
	AlertPlatform string   `json:"alert_platform"`                           // 告警平台
	AlertBotID    *uint    `json:"alert_bot_id"`                             // 告警机器人ID
}

// FailedDomain 失败的域名信息
type FailedDomain struct {
	Domain string `json:"domain"`
	Error  string `json:"error"`
}

// ImportSSLDomains 批量导入SSL域名
// POST /healthcheck/ssl-domains/import
func (h *HealthCheckHandler) ImportSSLDomains(c *gin.Context) {
	var req ImportDomainsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	// 验证告警阈值的合理性
	if req.CriticalDays >= req.WarningDays || req.WarningDays >= req.NoticeDays {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "Invalid alert thresholds: critical_days < warning_days < notice_days required",
		})
		return
	}

	// 获取当前用户ID
	var createdBy *uint
	if userID, ok := middleware.GetUserID(c); ok {
		createdBy = &userID
	}

	// 处理域名列表
	var validConfigs []*models.HealthCheckConfig
	var failedDomains []FailedDomain
	processedDomains := make(map[string]bool) // 用于去重

	for _, domain := range req.Domains {
		// 去除空格
		domain = strings.TrimSpace(domain)
		if domain == "" {
			continue
		}

		// 检查域名格式
		if !isValidDomain(domain) {
			failedDomains = append(failedDomains, FailedDomain{
				Domain: domain,
				Error:  "Invalid domain format",
			})
			continue
		}

		// 标准化域名（用于去重检查）
		normalizedDomain := normalizeDomain(domain)

		// 检查是否重复（在当前批次中）
		if processedDomains[normalizedDomain] {
			failedDomains = append(failedDomains, FailedDomain{
				Domain: domain,
				Error:  "Duplicate domain in request",
			})
			continue
		}

		// 检查数据库中是否已存在
		exists, err := h.isDomainExists(c.Request.Context(), normalizedDomain)
		if err != nil {
			hcLog.Error("Failed to check domain existence: %s, error: %v", domain, err)
			failedDomains = append(failedDomains, FailedDomain{
				Domain: domain,
				Error:  "Database error",
			})
			continue
		}
		if exists {
			failedDomains = append(failedDomains, FailedDomain{
				Domain: domain,
				Error:  "Domain already exists",
			})
			continue
		}

		// 创建配置
		config := &models.HealthCheckConfig{
			Name:          domain + " SSL证书",
			Type:          "ssl_cert",
			URL:           domain,
			Interval:      req.Interval,
			Timeout:       req.Timeout,
			RetryCount:    3,
			Enabled:       true,
			AlertEnabled:  req.AlertEnabled,
			AlertPlatform: req.AlertPlatform,
			AlertBotID:    req.AlertBotID,
			LastStatus:    "unknown",
			CriticalDays:  req.CriticalDays,
			WarningDays:   req.WarningDays,
			NoticeDays:    req.NoticeDays,
			CreatedBy:     createdBy,
		}

		validConfigs = append(validConfigs, config)
		processedDomains[normalizedDomain] = true
	}

	// 批量创建配置
	successCount := 0
	if len(validConfigs) > 0 {
		if err := h.configRepo.BatchCreate(c.Request.Context(), validConfigs); err != nil {
			hcLog.Error("Failed to batch create SSL domain configs: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"code":    500,
				"message": "Failed to create configurations: " + err.Error(),
			})
			return
		}
		successCount = len(validConfigs)
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "Success",
		"data": gin.H{
			"success_count":  successCount,
			"failed_count":   len(failedDomains),
			"failed_domains": failedDomains,
		},
	})
}

// isValidDomain 验证域名格式
func isValidDomain(domain string) bool {
	// 移除协议前缀
	domain = strings.TrimPrefix(domain, "https://")
	domain = strings.TrimPrefix(domain, "http://")

	// 移除路径部分
	if idx := strings.Index(domain, "/"); idx != -1 {
		domain = domain[:idx]
	}

	// 检查是否为空
	if domain == "" {
		return false
	}

	// 分离主机名和端口
	host := domain
	if strings.Contains(domain, ":") {
		var err error
		host, _, err = net.SplitHostPort(domain)
		if err != nil {
			return false
		}
	}

	// 检查主机名格式
	// 允许域名、IP地址
	if host == "" {
		return false
	}

	// 简单的域名格式检查
	// 域名应该包含字母、数字、点、连字符
	for _, ch := range host {
		if !((ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') ||
			(ch >= '0' && ch <= '9') || ch == '.' || ch == '-') {
			return false
		}
	}

	// 不能以点或连字符开头/结尾
	if strings.HasPrefix(host, ".") || strings.HasPrefix(host, "-") ||
		strings.HasSuffix(host, ".") || strings.HasSuffix(host, "-") {
		return false
	}

	// 不能包含连续的点
	if strings.Contains(host, "..") {
		return false
	}

	return true
}

// normalizeDomain 标准化域名（用于去重检查）
func normalizeDomain(domain string) string {
	// 移除协议前缀
	domain = strings.TrimPrefix(domain, "https://")
	domain = strings.TrimPrefix(domain, "http://")

	// 移除路径部分
	if idx := strings.Index(domain, "/"); idx != -1 {
		domain = domain[:idx]
	}

	// 转换为小写
	domain = strings.ToLower(domain)

	// 如果没有端口，添加默认端口443
	if !strings.Contains(domain, ":") {
		domain = domain + ":443"
	}

	return domain
}

// isDomainExists 检查域名是否已存在
func (h *HealthCheckHandler) isDomainExists(ctx context.Context, normalizedDomain string) (bool, error) {
	// 获取所有ssl_cert类型的配置
	configs, err := h.configRepo.GetByType(ctx, "ssl_cert")
	if err != nil {
		return false, err
	}

	// 检查是否存在相同的标准化域名
	for _, config := range configs {
		if normalizeDomain(config.URL) == normalizedDomain {
			return true, nil
		}
	}

	return false, nil
}

// BatchAlertConfigRequest 批量更新告警配置请求
type BatchAlertConfigRequest struct {
	ConfigIDs    []uint `json:"config_ids" binding:"required,min=1"`    // 配置ID列表
	CriticalDays int    `json:"critical_days" binding:"required,min=1"` // 严重告警阈值（天）
	WarningDays  int    `json:"warning_days" binding:"required,min=1"`  // 警告告警阈值（天）
	NoticeDays   int    `json:"notice_days" binding:"required,min=1"`   // 提醒告警阈值（天）
}

// BatchUpdateAlertConfig 批量更新告警配置
// PUT /healthcheck/ssl-domains/alert-config
func (h *HealthCheckHandler) BatchUpdateAlertConfig(c *gin.Context) {
	var req BatchAlertConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	// 验证告警阈值的合理性
	if req.CriticalDays >= req.WarningDays || req.WarningDays >= req.NoticeDays {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "Invalid alert thresholds: critical_days < warning_days < notice_days required",
		})
		return
	}

	// 批量更新配置
	updatedCount := 0
	for _, configID := range req.ConfigIDs {
		// 获取配置
		config, err := h.configRepo.GetByID(c.Request.Context(), configID)
		if err != nil {
			hcLog.Warn("Failed to get config %d: %v", configID, err)
			continue
		}

		// 更新告警阈值
		config.CriticalDays = req.CriticalDays
		config.WarningDays = req.WarningDays
		config.NoticeDays = req.NoticeDays

		// 保存更新
		if err := h.configRepo.Update(c.Request.Context(), config); err != nil {
			hcLog.Error("Failed to update config %d: %v", configID, err)
			continue
		}

		updatedCount++
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "Success",
		"data": gin.H{
			"updated_count": updatedCount,
		},
	})
}

// GetExpiringCerts 获取即将过期的证书
// GET /healthcheck/ssl-domains/expiring?days=30
func (h *HealthCheckHandler) GetExpiringCerts(c *gin.Context) {
	// 获取查询参数，默认30天
	days, err := strconv.Atoi(c.DefaultQuery("days", "30"))
	if err != nil || days < 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "Invalid days parameter",
		})
		return
	}

	// 查询即将过期的证书
	configs, err := h.configRepo.GetExpiringCerts(c.Request.Context(), days)
	if err != nil {
		hcLog.Error("Failed to get expiring certs: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "Success",
		"data": gin.H{
			"list":  configs,
			"total": len(configs),
		},
	})
}

// CertReportItem 证书报告项
type CertReportItem struct {
	Domain        string  `json:"domain"`
	ExpiryDate    *string `json:"expiry_date"`
	DaysRemaining *int    `json:"days_remaining"`
	AlertLevel    string  `json:"alert_level"`
	Issuer        string  `json:"issuer"`
	Subject       string  `json:"subject"`
	SerialNumber  string  `json:"serial_number"`
	LastCheckAt   *string `json:"last_check_at"`
	Status        string  `json:"status"`
	Enabled       bool    `json:"enabled"`
}

// ListSSLDomains 查询SSL证书列表
// GET /healthcheck/ssl-domains?page=1&page_size=20&alert_level=warning&keyword=example&sort_by=days_asc
func (h *HealthCheckHandler) ListSSLDomains(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	alertLevel := c.Query("alert_level")
	keyword := c.Query("keyword")
	sortBy := c.DefaultQuery("sort_by", "days_asc")

	// 构建过滤条件
	filters := &repository.ListFilters{
		Type:       "ssl_cert",
		AlertLevel: alertLevel,
		Keyword:    keyword,
		SortBy:     sortBy,
	}

	// 查询配置列表
	configs, total, err := h.configRepo.ListWithFilters(c.Request.Context(), filters, page, pageSize)
	if err != nil {
		hcLog.Error("Failed to list SSL domains: %v", err)
		response.InternalError(c, "查询失败")
		return
	}

	// 转换为响应格式
	list := make([]map[string]interface{}, 0, len(configs))
	for _, cfg := range configs {
		item := map[string]interface{}{
			"id":                  cfg.ID,
			"name":                cfg.Name,
			"url":                 cfg.URL,
			"type":                cfg.Type,
			"interval":            cfg.Interval,
			"timeout":             cfg.Timeout,
			"retry_count":         cfg.RetryCount,
			"enabled":             cfg.Enabled,
			"alert_enabled":       cfg.AlertEnabled,
			"alert_platform":      cfg.AlertPlatform,
			"alert_bot_id":        cfg.AlertBotID,
			"cert_expiry_date":    cfg.CertExpiryDate,
			"cert_days_remaining": cfg.CertDaysRemaining,
			"cert_issuer":         cfg.CertIssuer,
			"cert_subject":        cfg.CertSubject,
			"cert_serial_number":  cfg.CertSerialNumber,
			"critical_days":       cfg.CriticalDays,
			"warning_days":        cfg.WarningDays,
			"notice_days":         cfg.NoticeDays,
			"last_alert_level":    cfg.LastAlertLevel,
			"last_alert_at":       cfg.LastAlertAt,
			"last_check_at":       cfg.LastCheckAt,
			"last_status":         cfg.LastStatus,
			"created_at":          cfg.CreatedAt,
			"updated_at":          cfg.UpdatedAt,
		}
		list = append(list, item)
	}

	response.Success(c, gin.H{
		"list":  list,
		"total": total,
	})
}

// CertReport 证书报告
type CertReport struct {
	ExportTime   string           `json:"export_time"`
	TotalCount   int              `json:"total_count"`
	Certificates []CertReportItem `json:"certificates"`
}

// ExportCertReport 导出证书报告
// GET /healthcheck/ssl-domains/export?format=json
func (h *HealthCheckHandler) ExportCertReport(c *gin.Context) {
	// 获取格式参数，目前只支持JSON
	format := c.DefaultQuery("format", "json")
	if format != "json" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "Unsupported format, only 'json' is supported",
		})
		return
	}

	// 查询所有ssl_cert类型的配置
	configs, err := h.configRepo.GetByType(c.Request.Context(), "ssl_cert")
	if err != nil {
		hcLog.Error("Failed to get SSL cert configs: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": err.Error(),
		})
		return
	}

	// 构建报告数据
	report := CertReport{
		ExportTime:   time.Now().Format("2006-01-02T15:04:05Z07:00"),
		TotalCount:   len(configs),
		Certificates: make([]CertReportItem, 0, len(configs)),
	}

	for _, config := range configs {
		item := CertReportItem{
			Domain:        config.URL,
			DaysRemaining: config.CertDaysRemaining,
			AlertLevel:    config.LastAlertLevel,
			Issuer:        config.CertIssuer,
			Subject:       config.CertSubject,
			SerialNumber:  config.CertSerialNumber,
			Status:        config.LastStatus,
			Enabled:       config.Enabled,
		}

		// 格式化时间字段
		if config.CertExpiryDate != nil {
			expiryStr := config.CertExpiryDate.Format("2006-01-02T15:04:05Z07:00")
			item.ExpiryDate = &expiryStr
		}

		if config.LastCheckAt != nil {
			checkStr := config.LastCheckAt.Format("2006-01-02T15:04:05Z07:00")
			item.LastCheckAt = &checkStr
		}

		report.Certificates = append(report.Certificates, item)
	}

	// 设置响应头，触发文件下载
	c.Header("Content-Type", "application/json")
	c.Header("Content-Disposition", "attachment; filename=ssl-cert-report.json")

	// 返回JSON数据
	c.JSON(http.StatusOK, report)
}
