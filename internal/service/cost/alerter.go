package cost

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"devops/internal/domain/notification/service/feishu"
	"devops/internal/models"
	"devops/internal/repository"
	"devops/internal/service/notification"
	"devops/pkg/logger"
)

// CostAlerter 成本告警器
type CostAlerter struct {
	db              *gorm.DB
	log             *logger.Logger
	oaNotifyRepo    *repository.OANotifyConfigRepository
	feishuAppRepo   *repository.FeishuAppRepository
	templateService *notification.TemplateService
}

// NewCostAlerter 创建成本告警器
func NewCostAlerter(db *gorm.DB) *CostAlerter {
	return &CostAlerter{
		db:              db,
		log:             logger.NewLogger("CostAlerter"),
		oaNotifyRepo:    repository.NewOANotifyConfigRepository(db),
		feishuAppRepo:   repository.NewFeishuAppRepository(db),
		templateService: notification.NewTemplateService(repository.NewMessageTemplateRepository(db)),
	}
}

// SendBudgetWarning 发送预算预警
func (a *CostAlerter) SendBudgetWarning(ctx context.Context, budget *models.CostBudget, currentCost, usagePercent float64) {
	title := "成本预算预警"
	namespace := budget.Namespace
	if namespace == "" {
		namespace = "集群级别"
	}

	message := fmt.Sprintf(
		"【预算预警】\n"+
			"命名空间: %s\n"+
			"月度预算: ¥%.2f\n"+
			"当前成本: ¥%.2f\n"+
			"使用率: %.1f%%\n"+
			"告警阈值: %.0f%%\n"+
			"请及时关注成本控制",
		namespace, budget.MonthlyBudget, currentCost, usagePercent, budget.AlertThreshold,
	)

	// 保存告警记录
	alert := models.CostAlert{
		ClusterID:   budget.ClusterID,
		BudgetID:    &budget.ID,
		AlertType:   "budget_warning",
		Severity:    "warning",
		Title:       title,
		Message:     message,
		Threshold:   budget.AlertThreshold,
		ActualValue: usagePercent,
		Status:      "active",
	}
	a.db.Create(&alert)

	// 发送通知
	data := map[string]interface{}{
		"Project":     namespace,
		"CurrentCost": fmt.Sprintf("%.2f", currentCost),
		"Budget":      fmt.Sprintf("%.2f", budget.MonthlyBudget),
		"UsageRate":   fmt.Sprintf("%.1f", usagePercent),
		"Message":     "请及时关注成本控制",
	}
	a.sendNotification(ctx, "COST_BUDGET_WARNING", data)

	a.log.WithField("budget_id", budget.ID).
		WithField("usage_percent", usagePercent).
		Warn("发送预算预警")
}

// SendBudgetExceeded 发送预算超支告警
func (a *CostAlerter) SendBudgetExceeded(ctx context.Context, budget *models.CostBudget, currentCost, usagePercent float64) {
	title := "成本预算超支"
	namespace := budget.Namespace
	if namespace == "" {
		namespace = "集群级别"
	}

	message := fmt.Sprintf(
		"【预算超支】\n"+
			"命名空间: %s\n"+
			"月度预算: ¥%.2f\n"+
			"当前成本: ¥%.2f\n"+
			"超支金额: ¥%.2f\n"+
			"使用率: %.1f%%\n"+
			"请立即采取措施控制成本",
		namespace, budget.MonthlyBudget, currentCost, currentCost-budget.MonthlyBudget, usagePercent,
	)

	alert := models.CostAlert{
		ClusterID:   budget.ClusterID,
		BudgetID:    &budget.ID,
		AlertType:   "budget_exceeded",
		Severity:    "critical",
		Title:       title,
		Message:     message,
		Threshold:   100,
		ActualValue: usagePercent,
		Status:      "active",
	}
	a.db.Create(&alert)

	// 发送通知
	data := map[string]interface{}{
		"Project":     namespace,
		"CurrentCost": fmt.Sprintf("%.2f", currentCost),
		"Budget":      fmt.Sprintf("%.2f", budget.MonthlyBudget),
		"Overrun":     fmt.Sprintf("%.2f", currentCost-budget.MonthlyBudget),
		"UsageRate":   fmt.Sprintf("%.1f", usagePercent),
		"Message":     "请立即采取措施控制成本",
	}
	a.sendNotification(ctx, "COST_BUDGET_EXCEEDED", data)

	a.log.WithField("budget_id", budget.ID).
		WithField("usage_percent", usagePercent).
		Error("发送预算超支告警")
}

// SendCostAnomaly 发送成本异常告警
func (a *CostAlerter) SendCostAnomaly(ctx context.Context, clusterID uint, date string, actualCost, expectedCost, deviation float64) {
	title := "成本异常告警"
	message := fmt.Sprintf(
		"【成本异常】\n"+
			"日期: %s\n"+
			"实际成本: ¥%.2f\n"+
			"预期成本: ¥%.2f\n"+
			"偏差: %.1f%%\n"+
			"可能原因: 资源突增或配置变更\n"+
			"请检查近期资源变化",
		date, actualCost, expectedCost, deviation,
	)

	alert := models.CostAlert{
		ClusterID:   clusterID,
		AlertType:   "anomaly",
		Severity:    "warning",
		Title:       title,
		Message:     message,
		Threshold:   expectedCost,
		ActualValue: actualCost,
		Status:      "active",
	}
	a.db.Create(&alert)

	// 发送通知
	data := map[string]interface{}{
		"Date":         date,
		"ActualCost":   fmt.Sprintf("%.2f", actualCost),
		"ExpectedCost": fmt.Sprintf("%.2f", expectedCost),
		"Deviation":    fmt.Sprintf("%.1f", deviation),
		"Message":      "可能原因: 资源突增或配置变更\n请检查近期资源变化",
	}
	a.sendNotification(ctx, "COST_ANOMALY", data)

	a.log.WithField("cluster_id", clusterID).
		WithField("deviation", deviation).
		Warn("发送成本异常告警")
}

// SendWasteAlert 发送资源浪费告警
func (a *CostAlerter) SendWasteAlert(ctx context.Context, clusterID uint, wastedCost float64, idleCount, overCount int) {
	title := "资源浪费告警"
	message := fmt.Sprintf(
		"【资源浪费】\n"+
			"浪费成本: ¥%.2f\n"+
			"闲置资源: %d 个\n"+
			"超配资源: %d 个\n"+
			"建议清理闲置资源并优化超配资源",
		wastedCost, idleCount, overCount,
	)

	alert := models.CostAlert{
		ClusterID:   clusterID,
		AlertType:   "waste",
		Severity:    "info",
		Title:       title,
		Message:     message,
		ActualValue: wastedCost,
		Status:      "active",
	}
	a.db.Create(&alert)

	// 发送通知
	data := map[string]interface{}{
		"WastedCost": fmt.Sprintf("%.2f", wastedCost),
		"IdleCount":  idleCount,
		"OverCount":  overCount,
		"Message":    "建议清理闲置资源并优化超配资源",
	}
	a.sendNotification(ctx, "COST_WASTE", data)

	a.log.WithField("cluster_id", clusterID).
		WithField("wasted_cost", wastedCost).
		Info("发送资源浪费告警")
}

func (a *CostAlerter) getFeishuClient(ctx context.Context) (*feishu.Client, error) {
	notifyConfig, err := a.oaNotifyRepo.GetDefault(ctx)
	if err != nil {
		return nil, err
	}
	// 如果没有默认配置，尝试直接获取默认飞书应用
	if notifyConfig == nil {
		app, err := a.feishuAppRepo.GetDefault(ctx)
		if err != nil {
			return nil, err
		}
		if app == nil {
			return nil, fmt.Errorf("no default feishu app found")
		}
		return feishu.NewClientWithApp(app.AppID, app.AppSecret), nil
	}

	var feishuApp *models.FeishuApp
	if notifyConfig.AppID > 0 {
		feishuApp, err = a.feishuAppRepo.GetByID(ctx, notifyConfig.AppID)
	} else {
		feishuApp, err = a.feishuAppRepo.GetDefault(ctx)
	}

	if err != nil {
		return nil, err
	}
	if feishuApp == nil {
		return nil, fmt.Errorf("no feishu app found")
	}

	return feishu.NewClientWithApp(feishuApp.AppID, feishuApp.AppSecret), nil
}

// sendNotification 发送通知
func (a *CostAlerter) sendNotification(ctx context.Context, templateName string, data map[string]interface{}) {
	// 获取飞书客户端
	feishuClient, err := a.getFeishuClient(ctx)
	if err != nil {
		a.log.WithField("error", err.Error()).Warn("获取飞书客户端失败，跳过发送")
		return
	}

	// 渲染模板
	content, err := a.templateService.Render(ctx, templateName, data)
	if err != nil {
		a.log.WithField("error", err.Error()).Error("渲染模板失败")
		return
	}

	// 获取接收人 (暂时使用 default config 的 receiver)
	notifyConfig, _ := a.oaNotifyRepo.GetDefault(ctx)
	receiveID := ""
	receiveIDType := "open_id"
	if notifyConfig != nil {
		receiveID = notifyConfig.ReceiveID
		receiveIDType = notifyConfig.ReceiveIDType
	}

	if err := feishuClient.SendMessage(ctx, receiveID, receiveIDType, "interactive", content); err != nil {
		a.log.WithField("error", err.Error()).Warn("发送飞书通知失败")
	}
}
