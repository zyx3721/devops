package healthcheck

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"gorm.io/gorm"

	"devops/internal/config"
	"devops/internal/domain/notification/service/feishu"
	"devops/internal/models"
	"devops/internal/repository"
	"devops/internal/service/jenkins"
	"devops/internal/service/kubernetes"
	"devops/internal/service/notification"
	"devops/pkg/logger"
)

var (
	checker     *HealthChecker
	checkerOnce sync.Once
)

type HealthChecker struct {
	db                *gorm.DB
	configRepo        *repository.HealthCheckConfigRepository
	historyRepo       *repository.HealthCheckHistoryRepository
	jenkinsRepo       *repository.JenkinsInstanceRepository
	k8sRepo           *repository.K8sClusterRepository
	oaAddrRepo        *repository.OAAddressRepository
	oaNotifyRepo      *repository.OANotifyConfigRepository
	feishuAppRepo     *repository.FeishuAppRepository
	dingtalkAppRepo   *repository.DingtalkAppRepository
	wechatWorkAppRepo *repository.WechatWorkAppRepository
	templateService   *notification.TemplateService
	stopChan          chan struct{}
	running           bool
	mu                sync.Mutex
	log               *logger.Logger
	semaphore         chan struct{} // 并发控制信号�?
}

func InitHealthChecker(db *gorm.DB, cfg *config.Config) *HealthChecker {
	checkerOnce.Do(func() {
		checker = &HealthChecker{
			db:                db,
			configRepo:        repository.NewHealthCheckConfigRepository(db),
			historyRepo:       repository.NewHealthCheckHistoryRepository(db),
			jenkinsRepo:       repository.NewJenkinsInstanceRepository(db),
			k8sRepo:           repository.NewK8sClusterRepository(db),
			oaAddrRepo:        repository.NewOAAddressRepository(db),
			oaNotifyRepo:      repository.NewOANotifyConfigRepository(db),
			feishuAppRepo:     repository.NewFeishuAppRepository(db),
			dingtalkAppRepo:   repository.NewDingtalkAppRepository(db),
			wechatWorkAppRepo: repository.NewWechatWorkAppRepository(db),
			templateService:   notification.NewTemplateService(repository.NewMessageTemplateRepository(db)),
			stopChan:          make(chan struct{}),
			semaphore:         make(chan struct{}, 10), // 最�?0个并�?
			log:               logger.NewLogger("healthcheck"),
		}
	})
	return checker
}

func GetHealthChecker() *HealthChecker {
	return checker
}

func (h *HealthChecker) Start() {
	h.mu.Lock()
	if h.running {
		h.mu.Unlock()
		return
	}
	h.running = true
	h.mu.Unlock()

	h.log.Info("Health checker started")

	go h.runLoop()
}

func (h *HealthChecker) Stop() {
	h.mu.Lock()
	defer h.mu.Unlock()

	if !h.running {
		return
	}

	close(h.stopChan)
	h.running = false
	h.log.Info("Health checker stopped")
}

func (h *HealthChecker) runLoop() {
	ticker := time.NewTicker(60 * time.Second) // 每分钟检查一次是否有需要执行的检�?
	defer ticker.Stop()

	h.log.Info("Health checker loop started, checking every 60 seconds")

	// 启动时立即执行一�?
	h.checkAll()

	for {
		select {
		case <-h.stopChan:
			return
		case <-ticker.C:
			h.checkAll()
		}
	}
}

func (h *HealthChecker) checkAll() {
	ctx := context.Background()
	configs, err := h.configRepo.GetEnabled(ctx)
	if err != nil {
		h.log.Error("Failed to get health check configs: %v", err)
		return
	}

	h.log.Info("Found %d enabled health check configs", len(configs))

	if len(configs) == 0 {
		return
	}

	now := time.Now()
	checkedCount := 0
	for _, config := range configs {
		// 检查是否到了检查时�?
		if config.LastCheckAt != nil {
			nextCheck := config.LastCheckAt.Add(time.Duration(config.Interval) * time.Second)
			if now.Before(nextCheck) {
				h.log.Debug("Skipping %s: next check at %v", config.Name, nextCheck)
				continue
			}
		}

		checkedCount++
		// 使用信号量控制并发数�?
		go func(cfg models.HealthCheckConfig) {
			// 获取信号量，如果已达到最大并发数则阻�?
			h.semaphore <- struct{}{}
			defer func() {
				// 释放信号�?
				<-h.semaphore
				// 捕获panic，确保单个检查失败不影响其他检�?
				if r := recover(); r != nil {
					h.log.Error("Panic in health check for %s: %v", cfg.Name, r)
				}
			}()
			h.checkOne(ctx, &cfg)
		}(config)
	}

	if checkedCount > 0 {
		h.log.Info("Health check triggered: %d configs", checkedCount)
	}
}

func (h *HealthChecker) checkOne(ctx context.Context, config *models.HealthCheckConfig) {
	h.log.Info("Checking: %s (%s)", config.Name, config.Type)

	startTime := time.Now()
	var status string
	var errorMsg string
	var responseTimeMs int64

	switch config.Type {
	case "jenkins":
		status, errorMsg, responseTimeMs = h.checkJenkins(ctx, config)
	case "k8s":
		status, errorMsg, responseTimeMs = h.checkK8s(ctx, config)
	case "oa":
		status, errorMsg, responseTimeMs = h.checkOA(ctx, config)
	case "custom":
		status, errorMsg, responseTimeMs = h.checkCustomURL(ctx, config)
	case "ssl_cert":
		status, errorMsg, responseTimeMs = h.checkSSLCert(ctx, config)
	default:
		status = "unknown"
		errorMsg = "Unknown check type"
	}

	if responseTimeMs == 0 {
		responseTimeMs = time.Since(startTime).Milliseconds()
	}

	// 更新配置状�?
	h.configRepo.UpdateStatus(ctx, config.ID, status, errorMsg)

	// 记录历史
	history := &models.HealthCheckHistory{
		CreatedAt:      time.Now(),
		ConfigID:       config.ID,
		ConfigName:     config.Name,
		Type:           config.Type,
		TargetName:     config.TargetName,
		Status:         status,
		ResponseTimeMs: responseTimeMs,
		ErrorMsg:       errorMsg,
	}

	// 如果状态变为不健康且启用了告警，发送告�?
	if status == "unhealthy" && config.AlertEnabled && config.LastStatus != "unhealthy" {
		h.sendAlert(ctx, config, errorMsg)
		history.AlertSent = true
	}

	h.historyRepo.Create(ctx, history)

	if status == "unhealthy" {
		h.log.Warn("Health check failed: %s (%s) - %s", config.Name, config.Type, errorMsg)
	} else {
		h.log.Info("Health check passed: %s (%s) - %dms", config.Name, config.Type, responseTimeMs)
	}
}

func (h *HealthChecker) checkJenkins(ctx context.Context, config *models.HealthCheckConfig) (string, string, int64) {
	if config.TargetID == 0 {
		return "unhealthy", "No target ID specified", 0
	}

	instance, err := h.jenkinsRepo.GetByID(ctx, config.TargetID)
	if err != nil {
		return "unhealthy", fmt.Sprintf("Failed to get Jenkins instance: %v", err), 0
	}

	svc := jenkins.NewJenkinsInstanceService(h.db)
	result, err := svc.TestConnection(ctx, config.TargetID)
	if err != nil {
		return "unhealthy", err.Error(), 0
	}

	if !result.Connected {
		return "unhealthy", result.Error, result.ResponseTimeMs
	}

	// 更新 target name
	if config.TargetName == "" {
		config.TargetName = instance.Name
		h.configRepo.Update(ctx, config)
	}

	return "healthy", "", result.ResponseTimeMs
}

func (h *HealthChecker) checkK8s(ctx context.Context, config *models.HealthCheckConfig) (string, string, int64) {
	if config.TargetID == 0 {
		return "unhealthy", "No target ID specified", 0
	}

	cluster, err := h.k8sRepo.GetByID(ctx, config.TargetID)
	if err != nil {
		return "unhealthy", fmt.Sprintf("Failed to get K8s cluster: %v", err), 0
	}

	svc := kubernetes.NewK8sClusterService(h.db)
	result, err := svc.TestConnection(ctx, config.TargetID)
	if err != nil {
		return "unhealthy", err.Error(), 0
	}

	if !result.Connected {
		return "unhealthy", result.Error, result.ResponseTimeMs
	}

	// 更新 target name
	if config.TargetName == "" {
		config.TargetName = cluster.Name
		h.configRepo.Update(ctx, config)
	}

	return "healthy", "", result.ResponseTimeMs
}

func (h *HealthChecker) checkOA(ctx context.Context, config *models.HealthCheckConfig) (string, string, int64) {
	if config.TargetID == 0 && config.URL == "" {
		return "unhealthy", "No target ID or URL specified", 0
	}

	url := config.URL
	if config.TargetID > 0 {
		addr, err := h.oaAddrRepo.GetByID(ctx, config.TargetID)
		if err != nil {
			return "unhealthy", fmt.Sprintf("Failed to get OA address: %v", err), 0
		}
		url = addr.URL
		if config.TargetName == "" {
			config.TargetName = addr.Name
			h.configRepo.Update(ctx, config)
		}
	}

	return h.checkURL(url, config.Timeout)
}

func (h *HealthChecker) checkCustomURL(ctx context.Context, config *models.HealthCheckConfig) (string, string, int64) {
	if config.URL == "" {
		return "unhealthy", "No URL specified", 0
	}

	return h.checkURL(config.URL, config.Timeout)
}

func (h *HealthChecker) checkURL(url string, timeout int) (string, string, int64) {
	if timeout <= 0 {
		timeout = 10
	}

	client := &http.Client{
		Timeout: time.Duration(timeout) * time.Second,
	}

	startTime := time.Now()
	resp, err := client.Get(url)
	responseTimeMs := time.Since(startTime).Milliseconds()

	if err != nil {
		return "unhealthy", err.Error(), responseTimeMs
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return "unhealthy", fmt.Sprintf("HTTP %d", resp.StatusCode), responseTimeMs
	}

	return "healthy", "", responseTimeMs
}

func (h *HealthChecker) checkSSLCert(ctx context.Context, config *models.HealthCheckConfig) (string, string, int64) {
	if config.URL == "" {
		h.log.WithFields(map[string]interface{}{
			"config_id":   config.ID,
			"config_name": config.Name,
		}).Error("No domain specified for SSL certificate check")
		return "unhealthy", "No domain specified", 0
	}

	// 创建SSL证书检查器
	timeout := time.Duration(config.Timeout) * time.Second
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	checker := NewSSLCertChecker(timeout)

	h.log.WithFields(map[string]interface{}{
		"config_id":   config.ID,
		"config_name": config.Name,
		"domain":      config.URL,
		"timeout":     timeout,
	}).Debug("Starting SSL certificate check")

	// 执行证书检查（包含告警级别判断�?
	result, err := checker.CheckSSLCertWithAlertLevel(
		config.URL,
		config.CriticalDays,
		config.WarningDays,
		config.NoticeDays,
	)

	if err != nil {
		h.log.WithFields(map[string]interface{}{
			"config_id":   config.ID,
			"config_name": config.Name,
			"domain":      config.URL,
			"error_type":  result.ErrorType,
			"error":       err.Error(),
		}).Error("SSL certificate check failed")
		return "unhealthy", err.Error(), 0
	}

	if result.Status == "unhealthy" {
		h.log.WithFields(map[string]interface{}{
			"config_id":   config.ID,
			"config_name": config.Name,
			"domain":      config.URL,
			"error_type":  result.ErrorType,
			"error":       result.ErrorMsg,
		}).Warn("SSL certificate check returned unhealthy status")
		return "unhealthy", result.ErrorMsg, result.ResponseTimeMs
	}

	// 更新证书信息
	certInfo := &repository.CertInfo{
		ExpiryDate:    result.ExpiryDate,
		DaysRemaining: result.DaysRemaining,
		Issuer:        result.Issuer,
		Subject:       result.Subject,
		SerialNumber:  result.SerialNumber,
	}
	if err := h.configRepo.UpdateCertInfo(ctx, config.ID, certInfo); err != nil {
		h.log.WithFields(map[string]interface{}{
			"config_id":   config.ID,
			"config_name": config.Name,
			"error":       err.Error(),
		}).Error("Failed to update cert info")
	}

	// 检查是否需要告�?
	shouldAlert, alertLevel := h.shouldSendAlert(config, result.AlertLevel)
	if shouldAlert {
		h.log.WithFields(map[string]interface{}{
			"config_id":      config.ID,
			"config_name":    config.Name,
			"domain":         config.URL,
			"alert_level":    alertLevel,
			"days_remaining": result.DaysRemaining,
		}).Info("Sending SSL certificate alert")

		h.sendCertAlert(ctx, config, result)
		if err := h.configRepo.UpdateAlertInfo(ctx, config.ID, alertLevel, time.Now()); err != nil {
			h.log.WithFields(map[string]interface{}{
				"config_id":   config.ID,
				"config_name": config.Name,
				"error":       err.Error(),
			}).Error("Failed to update alert info")
		}
	}

	h.log.WithFields(map[string]interface{}{
		"config_id":      config.ID,
		"config_name":    config.Name,
		"domain":         config.URL,
		"days_remaining": result.DaysRemaining,
		"alert_level":    result.AlertLevel,
		"response_ms":    result.ResponseTimeMs,
	}).Info("SSL certificate check completed successfully")

	return "healthy", "", result.ResponseTimeMs
}

// shouldSendAlert 判断是否应该发送告�?
// 返回: (是否发送告�? 告警级别)
func (h *HealthChecker) shouldSendAlert(config *models.HealthCheckConfig, newAlertLevel string) (bool, string) {
	// 如果未启用告警，不发�?
	if !config.AlertEnabled {
		return false, ""
	}

	// 如果是normal级别，不发送告�?
	if newAlertLevel == "normal" {
		return false, ""
	}

	// 如果告警级别升级，立即发送（忽略冷却期）
	if h.isAlertLevelUpgrade(config.LastAlertLevel, newAlertLevel) {
		return true, newAlertLevel
	}

	// 如果告警级别相同，检查冷却期
	if config.LastAlertLevel == newAlertLevel {
		if config.LastAlertAt != nil {
			cooldownPeriod := 24 * time.Hour // 默认24小时冷却�?
			if time.Since(*config.LastAlertAt) < cooldownPeriod {
				return false, ""
			}
		}
		return true, newAlertLevel
	}

	// 告警级别降级，不发送告�?
	return false, ""
}

// isAlertLevelUpgrade 判断告警级别是否升级
func (h *HealthChecker) isAlertLevelUpgrade(oldLevel, newLevel string) bool {
	levels := map[string]int{
		"":         0,
		"normal":   0,
		"notice":   1,
		"warning":  2,
		"critical": 3,
		"expired":  4,
	}
	return levels[newLevel] > levels[oldLevel]
}

// sendCertAlert 发送证书告警
func (h *HealthChecker) sendCertAlert(ctx context.Context, config *models.HealthCheckConfig, result *CertCheckResult) {
	// 根据告警级别选择emoji
	var emoji string
	switch result.AlertLevel {
	case "expired":
		emoji = "🔴"
	case "critical":
		emoji = "🟠"
	case "warning":
		emoji = "🟡"
	case "notice":
		emoji = "🔵"
	default:
		emoji = "⚪"
	}

	// 构建告警消息
	message := fmt.Sprintf(
		"%s SSL证书告警\n"+
			"域名: %s\n"+
			"告警级别: %s\n"+
			"证书剩余天数: %d天\n"+
			"证书过期时间: %s\n"+
			"证书颁发者: %s\n"+
			"证书主题: %s",
		emoji,
		config.URL,
		result.AlertLevel,
		result.DaysRemaining,
		result.ExpiryDate.Format("2006-01-02 15:04:05"),
		result.Issuer,
		result.Subject,
	)

	h.log.WithFields(map[string]interface{}{
		"config_id":      config.ID,
		"config_name":    config.Name,
		"domain":         config.URL,
		"alert_level":    result.AlertLevel,
		"days_remaining": result.DaysRemaining,
		"expiry_date":    result.ExpiryDate.Format("2006-01-02"),
		"issuer":         result.Issuer,
		"subject":        result.Subject,
	}).Warn("SSL Certificate Alert triggered")

	h.log.WithFields(map[string]interface{}{
		"config_id": config.ID,
		"message":   message,
	}).Info("Alert message prepared")

	// 获取默认通知配置
	notifyConfig, err := h.oaNotifyRepo.GetDefault(ctx)
	if err != nil {
		h.log.Error("Failed to get default notification config: %v", err)
		return
	}

	if notifyConfig != nil {
		// 尝试获取飞书应用配置
		var feishuApp *models.FeishuApp
		var err error
		if notifyConfig.AppID > 0 {
			feishuApp, err = h.feishuAppRepo.GetByID(ctx, notifyConfig.AppID)
		} else {
			// 如果没有指定AppID，尝试获取默认应用
			feishuApp, err = h.feishuAppRepo.GetDefault(ctx)
		}

		if err == nil && feishuApp != nil {
			// 创建临时的飞书客户端
			feishuClient := feishu.NewClientWithApp(feishuApp.AppID, feishuApp.AppSecret)

			// 确定卡片颜色
			headerColor := "blue"
			switch result.AlertLevel {
			case "critical", "expired":
				headerColor = "red"
			case "warning":
				headerColor = "orange"
			}

			// 使用模板渲染卡片
			data := map[string]interface{}{
				"Title":         fmt.Sprintf("%s SSL证书告警", emoji),
				"HeaderColor":   headerColor,
				"Domain":        config.URL,
				"AlertLevel":    result.AlertLevel,
				"DaysRemaining": result.DaysRemaining,
				"ExpiryDate":    result.ExpiryDate.Format("2006-01-02 15:04:05"),
				"Issuer":        result.Issuer,
			}

			cardContent, err := h.templateService.Render(ctx, "SSL_CERT_ALERT", data)
			if err != nil {
				h.log.Error("Failed to render SSL alert template: %v", err)
				// 降级处理：使用默认的简单文本或其他方式
				return
			}

			err = feishuClient.SendMessage(ctx, notifyConfig.ReceiveID, notifyConfig.ReceiveIDType, "interactive", cardContent)
			if err != nil {
				h.log.Error("Failed to send Feishu message: %v", err)
			} else {
				h.log.Info("Feishu alert sent successfully")
			}
		} else {
			h.log.Warn("No valid Feishu App configuration found for alerting")
		}
	}
}

func (h *HealthChecker) sendAlert(ctx context.Context, config *models.HealthCheckConfig, errorMsg string) {
	h.log.Warn("Alert: %s (%s) is unhealthy - %s", config.Name, config.Type, errorMsg)

	// 获取默认通知配置
	notifyConfig, err := h.oaNotifyRepo.GetDefault(ctx)
	if err != nil {
		h.log.Error("Failed to get default notification config: %v", err)
		return
	}

	if notifyConfig != nil {
		// 尝试获取飞书应用配置
		var feishuApp *models.FeishuApp
		var err error
		if notifyConfig.AppID > 0 {
			feishuApp, err = h.feishuAppRepo.GetByID(ctx, notifyConfig.AppID)
		} else {
			feishuApp, err = h.feishuAppRepo.GetDefault(ctx)
		}

		if err == nil && feishuApp != nil {
			// 创建临时的飞书客户端
			feishuClient := feishu.NewClientWithApp(feishuApp.AppID, feishuApp.AppSecret)

			// 使用模板渲染卡片
			data := map[string]interface{}{
				"Title":    "🔴 健康检查告警",
				"Name":     config.Name,
				"Type":     config.Type,
				"ErrorMsg": errorMsg,
				"Time":     time.Now().Format("2006-01-02 15:04:05"),
			}

			cardContent, err := h.templateService.Render(ctx, "HEALTH_CHECK_ALERT", data)
			if err != nil {
				h.log.Error("Failed to render health check alert template: %v", err)
				return
			}

			err = feishuClient.SendMessage(ctx, notifyConfig.ReceiveID, notifyConfig.ReceiveIDType, "interactive", cardContent)
			if err != nil {
				h.log.Error("Failed to send Feishu message: %v", err)
			} else {
				h.log.Info("Feishu alert sent successfully")
			}
		} else {
			h.log.Warn("No valid Feishu App configuration found for alerting")
		}
	}
}

// CheckNow 立即执行指定配置的检�?
func (h *HealthChecker) CheckNow(ctx context.Context, configID uint) (*models.HealthCheckHistory, error) {
	config, err := h.configRepo.GetByID(ctx, configID)
	if err != nil {
		return nil, err
	}

	startTime := time.Now()
	var status string
	var errorMsg string
	var responseTimeMs int64

	switch config.Type {
	case "jenkins":
		status, errorMsg, responseTimeMs = h.checkJenkins(ctx, config)
	case "k8s":
		status, errorMsg, responseTimeMs = h.checkK8s(ctx, config)
	case "oa":
		status, errorMsg, responseTimeMs = h.checkOA(ctx, config)
	case "custom":
		status, errorMsg, responseTimeMs = h.checkCustomURL(ctx, config)
	case "ssl_cert":
		status, errorMsg, responseTimeMs = h.checkSSLCert(ctx, config)
	default:
		status = "unknown"
		errorMsg = "Unknown check type"
	}

	if responseTimeMs == 0 {
		responseTimeMs = time.Since(startTime).Milliseconds()
	}

	// 更新配置状态
	h.configRepo.UpdateStatus(ctx, config.ID, status, errorMsg)

	// 记录历史
	history := &models.HealthCheckHistory{
		CreatedAt:      time.Now(),
		ConfigID:       config.ID,
		ConfigName:     config.Name,
		Type:           config.Type,
		TargetName:     config.TargetName,
		Status:         status,
		ResponseTimeMs: responseTimeMs,
		ErrorMsg:       errorMsg,
	}

	// 如果是 SSL 证书检查，需要重新获取配置以获取最新的证书信息
	if config.Type == "ssl_cert" {
		updatedConfig, err := h.configRepo.GetByID(ctx, config.ID)
		if err == nil {
			history.CertDaysRemaining = updatedConfig.CertDaysRemaining
			history.CertExpiryDate = updatedConfig.CertExpiryDate
			history.AlertLevel = updatedConfig.LastAlertLevel
		}
	}

	h.historyRepo.Create(ctx, history)

	return history, nil
}
