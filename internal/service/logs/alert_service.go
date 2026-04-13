package logs

import (
	"context"
	"regexp"
	"strings"
	"sync"
	"time"

	"devops/internal/models"
	"devops/internal/modules/monitoring/repository"
	"devops/pkg/dto"
	"devops/pkg/logger"

	"gorm.io/gorm"
)

// AlertService 日志告警服务
type AlertService struct {
	db           *gorm.DB
	adapter      *K8sLogAdapter
	rules        sync.Map // map[int64]*models.LogAlertRule
	aggregator   *AlertAggregator
	notifier     AlertNotifier
	silenceRepo  *repository.AlertSilenceRepository
	stopCh       chan struct{}
	watcherCount int
}

// AlertNotifier 告警通知接口
type AlertNotifier interface {
	Send(ctx context.Context, rule *models.LogAlertRule, history *models.LogAlertHistory) error
}

// AlertAggregator 告警聚合器
type AlertAggregator struct {
	mu       sync.Mutex
	counters map[string]*alertCounter
}

type alertCounter struct {
	count     int
	firstTime time.Time
	lastTime  time.Time
	samples   []string
}

// NewAlertService 创建告警服务
func NewAlertService(db *gorm.DB, adapter *K8sLogAdapter, notifier AlertNotifier, silenceRepo *repository.AlertSilenceRepository) *AlertService {
	return &AlertService{
		db:          db,
		adapter:     adapter,
		notifier:    notifier,
		silenceRepo: silenceRepo,
		aggregator: &AlertAggregator{
			counters: make(map[string]*alertCounter),
		},
		stopCh: make(chan struct{}),
	}
}

// Start 启动告警服务
func (s *AlertService) Start(ctx context.Context) error {
	// 加载所有启用的规则
	if err := s.loadRules(); err != nil {
		return err
	}

	// 启动规则监控
	go s.watchRules(ctx)

	// 启动聚合检查
	go s.checkAggregation(ctx)

	return nil
}

// Stop 停止告警服务
func (s *AlertService) Stop() {
	close(s.stopCh)
}

// loadRules 加载告警规则
func (s *AlertService) loadRules() error {
	var rules []models.LogAlertRule
	if err := s.db.Where("enabled = ?", true).Find(&rules).Error; err != nil {
		return err
	}

	for _, rule := range rules {
		r := rule
		s.rules.Store(rule.ID, &r)
	}

	return nil
}

// watchRules 监控规则变化
func (s *AlertService) watchRules(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stopCh:
			return
		case <-ticker.C:
			s.loadRules()
		}
	}
}

// checkAggregation 检查聚合告警
func (s *AlertService) checkAggregation(ctx context.Context) {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stopCh:
			return
		case <-ticker.C:
			s.processAggregatedAlerts(ctx)
		}
	}
}

// processAggregatedAlerts 处理聚合告警
func (s *AlertService) processAggregatedAlerts(ctx context.Context) {
	s.aggregator.mu.Lock()
	defer s.aggregator.mu.Unlock()

	now := time.Now()
	for key, counter := range s.aggregator.counters {
		// 检查是否超过聚合时间
		if now.Sub(counter.firstTime) >= time.Minute {
			// 发送聚合告警
			s.sendAggregatedAlert(ctx, key, counter)
			delete(s.aggregator.counters, key)
		}
	}
}

// sendAggregatedAlert 发送聚合告警
func (s *AlertService) sendAggregatedAlert(ctx context.Context, key string, counter *alertCounter) {
	// 解析 key 获取规则 ID
	parts := strings.Split(key, ":")
	if len(parts) < 1 {
		return
	}

	// 记录告警历史
	history := &models.LogAlertHistory{
		AlertCount: counter.count,
		Status:     "pending",
		CreatedAt:  time.Now(),
	}

	if counter.count > 0 && len(counter.samples) > 0 {
		history.MatchedContent = counter.samples[0]
	}

	s.db.Create(history)
}

// CheckLog 检查日志是否触发告警
func (s *AlertService) CheckLog(entry *dto.LogEntry) {
	s.rules.Range(func(key, value interface{}) bool {
		rule := value.(*models.LogAlertRule)
		if s.matchRule(rule, entry) {
			s.handleMatch(rule, entry)
		}
		return true
	})
}

// matchRule 检查日志是否匹配规则
func (s *AlertService) matchRule(rule *models.LogAlertRule, entry *dto.LogEntry) bool {
	switch rule.MatchType {
	case "keyword":
		return strings.Contains(strings.ToLower(entry.Content), strings.ToLower(rule.MatchValue))
	case "regex":
		matched, _ := regexp.MatchString(rule.MatchValue, entry.Content)
		return matched
	case "level":
		return strings.EqualFold(entry.Level, rule.MatchValue)
	default:
		return false
	}
}

// handleMatch 处理匹配的日志
func (s *AlertService) handleMatch(rule *models.LogAlertRule, entry *dto.LogEntry) {
	// 如果设置了聚合时间，则聚合处理
	if rule.AggregateMin > 0 {
		s.aggregateAlert(rule, entry)
		return
	}

	// 直接发送告警
	s.sendAlert(context.Background(), rule, entry)
}

// aggregateAlert 聚合告警
func (s *AlertService) aggregateAlert(rule *models.LogAlertRule, entry *dto.LogEntry) {
	s.aggregator.mu.Lock()
	defer s.aggregator.mu.Unlock()

	key := s.getAggregateKey(rule, entry)
	counter, exists := s.aggregator.counters[key]
	if !exists {
		counter = &alertCounter{
			count:     0,
			firstTime: time.Now(),
			samples:   make([]string, 0, 5),
		}
		s.aggregator.counters[key] = counter
	}

	counter.count++
	counter.lastTime = time.Now()
	if len(counter.samples) < 5 {
		counter.samples = append(counter.samples, entry.Content)
	}
}

// getAggregateKey 获取聚合键
func (s *AlertService) getAggregateKey(rule *models.LogAlertRule, entry *dto.LogEntry) string {
	return strings.Join([]string{
		string(rune(rule.ID)),
		entry.PodName,
		entry.Container,
	}, ":")
}

// sendAlert 发送告警
func (s *AlertService) sendAlert(ctx context.Context, rule *models.LogAlertRule, entry *dto.LogEntry) {
	history := &models.LogAlertHistory{
		RuleID:         rule.ID,
		ClusterID:      rule.ClusterID,
		Namespace:      rule.Namespace,
		PodName:        entry.PodName,
		Container:      entry.Container,
		MatchedContent: entry.Content,
		AlertCount:     1,
		Status:         "pending",
		CreatedAt:      time.Now(),
	}

	// 检查是否被静默
	if s.silenceRepo != nil {
		silence, isSilenced := s.checkSilence(ctx, rule, entry)
		if isSilenced {
			history.Silenced = true
			history.SilenceID = &silence.ID
			history.Status = "silenced"

			// 记录到数据库但不发送通知
			if err := s.db.Create(history).Error; err != nil {
				logger.L().Error("创建告警历史失败: %v", err)
			}

			logger.L().Info("告警被静默: rule_id=%d, silence_id=%d, namespace=%s, pod=%s",
				rule.ID, silence.ID, rule.Namespace, entry.PodName)
			return
		}
	}

	if err := s.db.Create(history).Error; err != nil {
		logger.L().Error("创建告警历史失败: %v", err)
		return
	}

	// 发送通知
	if s.notifier != nil {
		go func() {
			if err := s.notifier.Send(ctx, rule, history); err != nil {
				logger.L().Error("发送告警通知失败: %v", err)
				s.db.Model(history).Updates(map[string]interface{}{
					"status":    "failed",
					"error_msg": err.Error(),
				})
			} else {
				now := time.Now()
				s.db.Model(history).Updates(map[string]interface{}{
					"status":  "sent",
					"sent_at": &now,
				})
			}
		}()
	}
}

// checkSilence 检查告警是否被静默
func (s *AlertService) checkSilence(ctx context.Context, rule *models.LogAlertRule, entry *dto.LogEntry) (*models.AlertSilence, bool) {
	// 获取当前生效的静默规则
	silences, err := s.silenceRepo.GetActiveSilences(ctx, "log_alert")
	if err != nil {
		logger.L().Error("获取静默规则失败: %v", err)
		return nil, false
	}

	// 遍历静默规则，检查是否匹配
	for _, silence := range silences {
		if s.matchSilence(&silence, rule, entry) {
			return &silence, true
		}
	}

	return nil, false
}

// matchSilence 检查告警是否匹配静默规则
func (s *AlertService) matchSilence(silence *models.AlertSilence, rule *models.LogAlertRule, entry *dto.LogEntry) bool {
	// 如果 Matchers 为空，则匹配所有
	if silence.Matchers == "" {
		return true
	}

	// 简单的键值对匹配（格式：namespace=xxx,cluster=yyy）
	matchers := strings.Split(silence.Matchers, ",")
	for _, matcher := range matchers {
		parts := strings.SplitN(strings.TrimSpace(matcher), "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		pattern := strings.TrimSpace(parts[1])

		switch key {
		case "namespace":
			if !matchPattern(pattern, rule.Namespace) {
				return false
			}
		case "pod_name":
			if !matchPattern(pattern, entry.PodName) {
				return false
			}
		case "container":
			if !matchPattern(pattern, entry.Container) {
				return false
			}
		case "level":
			if !matchPattern(pattern, entry.Level) {
				return false
			}
		}
	}

	return true
}

// matchPattern 匹配模式（支持通配符和正则）
func matchPattern(pattern, value string) bool {
	// 通配符匹配
	if pattern == "*" {
		return true
	}

	// 精确匹配
	if pattern == value {
		return true
	}

	// 正则匹配
	matched, _ := regexp.MatchString(pattern, value)
	return matched
}

// CRUD 操作

// ListRules 获取告警规则列表
func (s *AlertService) ListRules(clusterID int64, namespace string) ([]dto.LogAlertRuleResponse, error) {
	var rules []models.LogAlertRule
	query := s.db.Model(&models.LogAlertRule{})

	if clusterID > 0 {
		query = query.Where("cluster_id = ?", clusterID)
	}
	if namespace != "" {
		query = query.Where("namespace = ? OR namespace = ''", namespace)
	}

	if err := query.Order("created_at DESC").Find(&rules).Error; err != nil {
		return nil, err
	}

	result := make([]dto.LogAlertRuleResponse, len(rules))
	for i, rule := range rules {
		result[i] = s.ruleToResponse(&rule)
	}

	return result, nil
}

// CreateRule 创建告警规则
func (s *AlertService) CreateRule(userID int64, req *dto.LogAlertRuleRequest) (*dto.LogAlertRuleResponse, error) {
	rule := &models.LogAlertRule{
		Name:         req.Name,
		ClusterID:    req.ClusterID,
		Namespace:    req.Namespace,
		MatchType:    req.MatchType,
		MatchValue:   req.MatchValue,
		Level:        req.Level,
		Channels:     models.JSONArray(req.Channels),
		Enabled:      req.Enabled,
		AggregateMin: req.AggregateMin,
		CreatedBy:    userID,
	}

	if err := s.db.Create(rule).Error; err != nil {
		return nil, err
	}

	// 更新缓存
	if rule.Enabled {
		s.rules.Store(rule.ID, rule)
	}

	resp := s.ruleToResponse(rule)
	return &resp, nil
}

// UpdateRule 更新告警规则
func (s *AlertService) UpdateRule(ruleID int64, req *dto.LogAlertRuleRequest) (*dto.LogAlertRuleResponse, error) {
	var rule models.LogAlertRule
	if err := s.db.First(&rule, ruleID).Error; err != nil {
		return nil, err
	}

	rule.Name = req.Name
	rule.ClusterID = req.ClusterID
	rule.Namespace = req.Namespace
	rule.MatchType = req.MatchType
	rule.MatchValue = req.MatchValue
	rule.Level = req.Level
	rule.Channels = models.JSONArray(req.Channels)
	rule.Enabled = req.Enabled
	rule.AggregateMin = req.AggregateMin

	if err := s.db.Save(&rule).Error; err != nil {
		return nil, err
	}

	// 更新缓存
	if rule.Enabled {
		s.rules.Store(rule.ID, &rule)
	} else {
		s.rules.Delete(rule.ID)
	}

	resp := s.ruleToResponse(&rule)
	return &resp, nil
}

// DeleteRule 删除告警规则
func (s *AlertService) DeleteRule(ruleID int64) error {
	if err := s.db.Delete(&models.LogAlertRule{}, ruleID).Error; err != nil {
		return err
	}
	s.rules.Delete(ruleID)
	return nil
}

// ToggleRule 切换规则状态
func (s *AlertService) ToggleRule(ruleID int64) (bool, error) {
	var rule models.LogAlertRule
	if err := s.db.First(&rule, ruleID).Error; err != nil {
		return false, err
	}

	rule.Enabled = !rule.Enabled
	if err := s.db.Save(&rule).Error; err != nil {
		return false, err
	}

	if rule.Enabled {
		s.rules.Store(rule.ID, &rule)
	} else {
		s.rules.Delete(rule.ID)
	}

	return rule.Enabled, nil
}

// ListHistory 获取告警历史
func (s *AlertService) ListHistory(ruleID int64, page, pageSize int) ([]dto.LogAlertHistoryResponse, int64, error) {
	var histories []models.LogAlertHistory
	var total int64

	query := s.db.Model(&models.LogAlertHistory{})
	if ruleID > 0 {
		query = query.Where("rule_id = ?", ruleID)
	}

	query.Count(&total)

	if err := query.Order("created_at DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&histories).Error; err != nil {
		return nil, 0, err
	}

	result := make([]dto.LogAlertHistoryResponse, len(histories))
	for i, h := range histories {
		result[i] = dto.LogAlertHistoryResponse{
			ID:             h.ID,
			RuleID:         h.RuleID,
			ClusterID:      h.ClusterID,
			Namespace:      h.Namespace,
			PodName:        h.PodName,
			Container:      h.Container,
			MatchedContent: h.MatchedContent,
			AlertCount:     h.AlertCount,
			Status:         h.Status,
			SentAt:         h.SentAt,
			ErrorMsg:       h.ErrorMsg,
			CreatedAt:      h.CreatedAt,
		}
	}

	return result, total, nil
}

func (s *AlertService) ruleToResponse(rule *models.LogAlertRule) dto.LogAlertRuleResponse {
	channels := []string(rule.Channels)
	if channels == nil {
		channels = []string{}
	}

	return dto.LogAlertRuleResponse{
		ID:           rule.ID,
		Name:         rule.Name,
		ClusterID:    rule.ClusterID,
		Namespace:    rule.Namespace,
		MatchType:    rule.MatchType,
		MatchValue:   rule.MatchValue,
		Level:        rule.Level,
		Channels:     channels,
		Enabled:      rule.Enabled,
		AggregateMin: rule.AggregateMin,
		CreatedBy:    rule.CreatedBy,
		CreatedAt:    rule.CreatedAt,
		UpdatedAt:    rule.UpdatedAt,
	}
}
