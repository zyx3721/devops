package handler

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"devops/internal/config"
	"devops/internal/models"
	"devops/internal/models/monitoring"
	"devops/internal/repository"
	"devops/internal/service/notification"
	"devops/pkg/ioc"
	"devops/pkg/logger"
)

var gatewayLog = logger.L().WithField("module", "alert_gateway")

func init() {
	ioc.Api.RegisterContainer("GatewayHandler", &GatewayApiHandler{})
}

type GatewayApiHandler struct {
	handler *GatewayHandler
}

func (h *GatewayApiHandler) Init() error {
	cfg, err := config.LoadConfig()
	if err != nil {
		return err
	}

	db := cfg.GetDB()
	h.handler = NewGatewayHandler(db)

	root := cfg.Application.GinRootRouter().Group("gateway")
	{
		// 接收告警事件
		root.POST("/event/:source", h.handler.Ingest)
		// 交互回调
		root.POST("/callback/:channel", h.handler.HandleCallback)
	}

	return nil
}

// Register 仅用于满足接口要求，逻辑已在 Init 中处理
func (h *GatewayApiHandler) Register(r gin.IRouter) {
	// do nothing
}

type GatewayHandler struct {
	configRepo  *repository.AlertConfigRepository
	historyRepo *repository.AlertHistoryRepository
	silenceRepo *repository.AlertSilenceRepository
	tmplSvc     *notification.TemplateService
	db          *gorm.DB
	// 复用 AlertHandler 的部分逻辑
	alertHandler *AlertHandler
}

func NewGatewayHandler(db *gorm.DB) *GatewayHandler {
	h := &GatewayHandler{
		configRepo:   repository.NewAlertConfigRepository(db),
		historyRepo:  repository.NewAlertHistoryRepository(db),
		silenceRepo:  repository.NewAlertSilenceRepository(db),
		tmplSvc:      notification.NewTemplateService(repository.NewMessageTemplateRepository(db)),
		db:           db,
		alertHandler: NewAlertHandler(db), // 复用
	}
	h.ensureDefaultConfig()
	return h
}

// ensureDefaultConfig 确保存在默认告警配置
func (h *GatewayHandler) ensureDefaultConfig() {
	var count int64
	h.db.Model(&models.AlertConfig{}).Where("name = ?", "Default").Count(&count)
	if count == 0 {
		defaultConfig := models.AlertConfig{
			Name:        "Default",
			Type:        "default",
			Enabled:     true,
			Platform:    "feishu", // 默认假设用飞书，或者根据实际情况调整
			Description: "系统自动创建的默认兜底告警配置",
			Channels:    "[]",
			Conditions:  "{}",
		}
		if err := h.db.Create(&defaultConfig).Error; err != nil {
			gatewayLog.Error("Failed to create default alert config: %v", err)
		} else {
			gatewayLog.Info("Created default alert config 'Default'")
		}
	}
}

// Ingest 接收外部告警
func (h *GatewayHandler) Ingest(c *gin.Context) {
	source := c.Param("source")
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to read body"})
		return
	}

	// 1. 解析数据 (Adapter)
	events, err := h.parseEvents(source, body)
	if err != nil {
		gatewayLog.Warn("Failed to parse events from %s: %v", source, err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if len(events) == 0 {
		c.JSON(http.StatusOK, gin.H{"message": "No events parsed"})
		return
	}

	// 2. 处理每个事件
	processedCount := 0
	for _, event := range events {
		// 2.1 匹配 AlertConfig (路由)
		config, err := h.matchConfig(event)
		if err != nil {
			gatewayLog.Debug("No matching config for event: %s", event.Title)
			// 可以选择记录到“未匹配告警”日志中
			continue
		}

		// 2.2 静默检查 (Silence)
		if h.isSilenced(event) {
			gatewayLog.Info("Event silenced: %s", event.Title)
			// 记录静默历史？
			h.recordHistory(c, config, event, "silenced", "Silenced by rule")
			continue
		}

		// 2.3 发送告警 (Dispatch)
		// 复用 AlertHandler 的逻辑，或者重写
		go h.dispatchAlert(config, event)
		processedCount++
	}

	c.JSON(http.StatusOK, gin.H{
		"message": fmt.Sprintf("Processed %d events", processedCount),
	})
}

// parseEvents 根据来源解析数据
func (h *GatewayHandler) parseEvents(source string, body []byte) ([]monitoring.AlertEvent, error) {
	switch source {
	case "prometheus":
		return parsePrometheusWebhook(body)
	case "grafana":
		return parseGrafanaWebhook(body)
	case "default", "generic":
		// 尝试解析为标准 AlertEvent
		var event monitoring.AlertEvent
		if err := json.Unmarshal(body, &event); err != nil {
			return nil, err
		}
		// 补充指纹
		if event.Fingerprint == "" {
			event.Fingerprint = generateFingerprint(event.Title, event.Labels)
		}
		return []monitoring.AlertEvent{event}, nil
	default:
		return nil, fmt.Errorf("unsupported source: %s", source)
	}
}

// matchConfig 匹配告警配置
// 策略：优先匹配 Labels 中的 alert_config_id，其次匹配 Name = event.Title，最后匹配 Labels 包含关系
func (h *GatewayHandler) matchConfig(event monitoring.AlertEvent) (*models.AlertConfig, error) {
	// 1. 如果 Labels 中指定了 config_id 或 config_name
	if name, ok := event.Labels["alertname"]; ok {
		var configs []models.AlertConfig
		if h.db.Where("name = ? AND enabled = ?", name, true).Find(&configs); len(configs) > 0 {
			return &configs[0], nil
		}
	}

	// 2. 尝试匹配 Title
	var configs []models.AlertConfig
	if h.db.Where("name = ? AND enabled = ?", event.Title, true).Find(&configs); len(configs) > 0 {
		return &configs[0], nil
	}

	// 3. 默认配置 (可选)
	// return nil, fmt.Errorf("no config matched")

	// 为了演示，如果没有匹配到，我们尝试查找一个名为 "Default" 的配置
	if h.db.Where("name = ? AND enabled = ?", "Default", true).Find(&configs); len(configs) > 0 {
		return &configs[0], nil
	}

	return nil, fmt.Errorf("no config matched")
}

// isSilenced 检查是否静默
func (h *GatewayHandler) isSilenced(event monitoring.AlertEvent) bool {
	// 查询所有 Active 的静默规则
	// 优化：应该在 Repo 层做，这里简单实现
	var silences []models.AlertSilence
	now := time.Now()
	h.db.Where("status = ? AND start_time <= ? AND end_time >= ?", "active", now, now).Find(&silences)

	for _, silence := range silences {
		if silence.Matchers == "" {
			// 全局静默？通常 matchers 为空表示匹配该类型的所有，或者无效
			if silence.Type == "all" {
				return true
			}
			continue
		}

		// 解析 Matchers JSON
		var matchers map[string]interface{}
		if err := json.Unmarshal([]byte(silence.Matchers), &matchers); err != nil {
			continue
		}

		if match(event, matchers) {
			return true
		}
	}
	return false
}

func match(event monitoring.AlertEvent, matchers map[string]interface{}) bool {
	// 简单的匹配逻辑：所有 matchers 必须满足
	// 支持 key: value (精确匹配)
	// 支持 key: [list] (包含匹配)

	// 合并 event 的属性用于匹配
	attributes := make(map[string]string)
	for k, v := range event.Labels {
		attributes[k] = v
	}
	attributes["title"] = event.Title
	attributes["level"] = event.Level
	attributes["source"] = event.Source

	for k, v := range matchers {
		attrVal, exists := attributes[k]
		if !exists {
			return false // 属性不存在，不匹配
		}

		switch matcherVal := v.(type) {
		case string:
			if attrVal != matcherVal {
				return false
			}
		case []interface{}:
			// 列表中的任意一个匹配即可
			found := false
			for _, item := range matcherVal {
				if s, ok := item.(string); ok && s == attrVal {
					found = true
					break
				}
			}
			if !found {
				return false
			}
		default:
			// 不支持的类型，忽略或视为不匹配
			return false
		}
	}
	return true
}

// dispatchAlert 发送告警
func (h *GatewayHandler) dispatchAlert(config *models.AlertConfig, event monitoring.AlertEvent) {
	// 构造数据用于模板渲染
	data := map[string]interface{}{
		"Title":       event.Title,
		"Content":     event.Content,
		"Level":       event.Level,
		"Source":      event.Source,
		"Labels":      event.Labels,
		"Fingerprint": event.Fingerprint,
		"Time":        time.Now().Format("2006-01-02 15:04:05"),
	}

	// 调用 AlertHandler 的异步处理逻辑
	// 但 AlertHandler.processAlertAsync 需要 gin.Context 里的参数，我们这里直接调用底层逻辑
	// 我们需要在 AlertHandler 中提取一个公共方法，或者在这里重写。
	// 为了复用，我们重写一部分逻辑，因为这里的数据结构更丰富 (AlertEvent)

	ctx := context.Background()
	var content string
	var err error

	if config.TemplateID != nil && *config.TemplateID > 0 {
		content, err = h.tmplSvc.RenderByID(ctx, uint(*config.TemplateID), data)
	}

	if err != nil || content == "" {
		content = fmt.Sprintf("Alert: %s\nDetails: %s\nLevel: %s", event.Title, event.Content, event.Level)
	}

	// 发送
	successCount := h.alertHandler.sendToChannels(config, content)

	// 记录历史
	h.recordHistory(nil, config, event, "sent", "")

	if successCount == 0 {
		gatewayLog.Error("Failed to send alert %s to any channel", event.Title)
	}
}

func (h *GatewayHandler) recordHistory(c *gin.Context, config *models.AlertConfig, event monitoring.AlertEvent, status string, errorMsg string) {
	history := &models.AlertHistory{
		AlertConfigID: config.ID,
		Type:          config.Type,
		Title:         event.Title,
		Content:       event.Content,
		Level:         event.Level,
		Status:        status,
		AckStatus:     "pending",
		SourceID:      event.SourceID,
		SourceURL:     event.SourceURL,
		ErrorMsg:      errorMsg,
		Silenced:      status == "silenced",
	}

	if status == "silenced" {
		history.AckStatus = "resolved" // 静默的告警视为已处理？或者 pending 但不需要通知
		// history.ResolveComment = "Silenced by rule"
	}

	h.historyRepo.Create(context.Background(), history)
}

func generateFingerprint(title string, labels map[string]string) string {
	// 简单实现：MD5(Title + SortedLabels)
	// 这里简化为 MD5(Title)
	hash := md5.Sum([]byte(title))
	return hex.EncodeToString(hash[:])
}

// ---------------- Adapters ----------------

func parsePrometheusWebhook(body []byte) ([]monitoring.AlertEvent, error) {
	// 简化的 Prometheus Alertmanager Webhook 结构
	type PromAlert struct {
		Status       string            `json:"status"`
		Labels       map[string]string `json:"labels"`
		Annotations  map[string]string `json:"annotations"`
		StartsAt     time.Time         `json:"startsAt"`
		EndsAt       time.Time         `json:"endsAt"`
		GeneratorURL string            `json:"generatorURL"`
		Fingerprint  string            `json:"fingerprint"`
	}
	type PromWebhook struct {
		Alerts []PromAlert `json:"alerts"`
	}

	var payload PromWebhook
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, err
	}

	var events []monitoring.AlertEvent
	for _, alert := range payload.Alerts {
		// 只处理 firing 的告警
		if alert.Status != "firing" {
			continue
		}

		title := alert.Labels["alertname"]
		if title == "" {
			title = "Prometheus Alert"
		}

		content := alert.Annotations["description"]
		if content == "" {
			content = alert.Annotations["summary"]
		}

		event := monitoring.AlertEvent{
			Fingerprint: alert.Fingerprint,
			Title:       title,
			Content:     content,
			Level:       normalizeLevel(alert.Labels["severity"]),
			Status:      alert.Status,
			Source:      "prometheus",
			SourceID:    alert.Fingerprint,
			SourceURL:   alert.GeneratorURL,
			Labels:      alert.Labels,
			StartsAt:    alert.StartsAt,
			RawData:     alert,
		}
		if !alert.EndsAt.IsZero() {
			event.EndsAt = &alert.EndsAt
		}

		events = append(events, event)
	}
	return events, nil
}

func parseGrafanaWebhook(body []byte) ([]monitoring.AlertEvent, error) {
	// Grafana Legacy Webhook 结构
	type GrafanaAlert struct {
		Title    string            `json:"title"`
		Message  string            `json:"message"`
		State    string            `json:"state"`
		RuleName string            `json:"ruleName"`
		RuleUrl  string            `json:"ruleUrl"`
		Tags     map[string]string `json:"tags"`
	}

	var alert GrafanaAlert
	if err := json.Unmarshal(body, &alert); err != nil {
		return nil, err
	}

	if alert.State != "alerting" {
		return nil, nil // 忽略 ok/no_data 状态
	}

	event := monitoring.AlertEvent{
		Title:     alert.Title,
		Content:   alert.Message,
		Level:     "warning", // Grafana 默认没有 severity，除非在 tags 里
		Status:    "firing",
		Source:    "grafana",
		SourceID:  fmt.Sprintf("%s-%d", alert.RuleName, time.Now().Unix()),
		SourceURL: alert.RuleUrl,
		Labels:    alert.Tags,
		StartsAt:  time.Now(),
		RawData:   alert,
	}
	return []monitoring.AlertEvent{event}, nil
}

func normalizeLevel(l string) string {
	switch strings.ToLower(l) {
	case "critical", "crit":
		return "critical"
	case "error", "err":
		return "error"
	case "warning", "warn":
		return "warning"
	default:
		return "info"
	}
}

// HandleCallback 处理交互式卡片回调
func (h *GatewayHandler) HandleCallback(c *gin.Context) {
	channel := c.Param("channel")

	var payload map[string]interface{}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON"})
		return
	}

	// 飞书 URL 验证
	if typeVal, ok := payload["type"].(string); ok && typeVal == "url_verification" {
		c.JSON(http.StatusOK, gin.H{"challenge": payload["challenge"]})
		return
	}

	// 处理飞书卡片交互
	// 结构参考: https://open.feishu.cn/document/ukTMukTMukTM/uYjL24iN/interactive-cards/card-action-callback
	if channel == "feishu" {
		if action, ok := payload["action"].(map[string]interface{}); ok {
			value := action["value"].(map[string]interface{})
			tag := action["tag"].(string)

			if tag == "button" {
				// 获取参数
				historyIDRaw := value["history_id"]
				actionType := value["action"] // "ack" or "resolve"

				// 转换 ID
				var historyID uint
				if idFloat, ok := historyIDRaw.(float64); ok {
					historyID = uint(idFloat)
				} else {
					gatewayLog.Warn("Invalid history_id in callback: %v", historyIDRaw)
					c.JSON(http.StatusOK, gin.H{}) // 返回空响应避免报错
					return
				}

				// 更新数据库
				ctx := context.Background()
				history, err := h.historyRepo.GetByID(ctx, historyID)
				if err != nil {
					gatewayLog.Error("History not found: %d", historyID)
					c.JSON(http.StatusOK, gin.H{})
					return
				}

				// 获取用户信息 (飞书 user_id)
				feishuUserID := payload["user_id"].(string)
				// 这里可以将 feishuUserID 映射为系统 UserID，或者直接记录 feishuUserID
				// 暂时只更新状态

				now := time.Now()
				var replyText string

				if actionType == "ack" {
					if history.AckStatus == "pending" {
						history.AckStatus = "acked"
						history.AckAt = &now
						// history.AckBy = ... // 需要用户映射系统
						h.historyRepo.Update(ctx, history)
						replyText = fmt.Sprintf("✅ 已由 %s 认领", feishuUserID)
					} else {
						replyText = fmt.Sprintf("⚠️ 告警当前状态: %s", history.AckStatus)
					}
				} else if actionType == "resolve" {
					history.AckStatus = "resolved"
					history.ResolvedAt = &now
					h.historyRepo.Update(ctx, history)
					replyText = fmt.Sprintf("✅ 已由 %s 解决", feishuUserID)
				}

				// 返回 Toast 提示
				c.JSON(http.StatusOK, gin.H{
					"toast": gin.H{
						"type":    "success",
						"content": replyText,
					},
				})
				return
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{"message": "Callback received"})
}
