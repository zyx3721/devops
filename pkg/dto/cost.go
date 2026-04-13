package dto

import "time"

// ===== 成本概览 =====

// CostOverviewResponse 成本概览响应
type CostOverviewResponse struct {
	// 总成本
	TotalCost   float64 `json:"total_cost"`
	CPUCost     float64 `json:"cpu_cost"`
	MemoryCost  float64 `json:"memory_cost"`
	StorageCost float64 `json:"storage_cost"`

	// 环比变化
	CostChange     float64 `json:"cost_change"`      // 成本变化（元）
	CostChangeRate float64 `json:"cost_change_rate"` // 成本变化率（%）

	// 同比变化
	YoYCostChange     float64 `json:"yoy_cost_change"`      // 同比变化（元）
	YoYCostChangeRate float64 `json:"yoy_cost_change_rate"` // 同比变化率（%）

	// 资源利用率
	AvgCPUUsage    float64 `json:"avg_cpu_usage"`    // 平均 CPU 利用率（%）
	AvgMemoryUsage float64 `json:"avg_memory_usage"` // 平均内存利用率（%）

	// 资源浪费
	WastedCost       float64 `json:"wasted_cost"`       // 浪费成本（闲置+超配）
	WastedPercentage float64 `json:"wasted_percentage"` // 浪费占比（%）
	IdleResources    int     `json:"idle_resources"`    // 闲置资源数
	ZombieResources  int     `json:"zombie_resources"`  // 僵尸资源数（长期无流量）

	// 优化空间
	PotentialSavings float64 `json:"potential_savings"` // 潜在节省（元）
	SuggestionCount  int     `json:"suggestion_count"`  // 优化建议数量

	// 预算
	BudgetTotal   float64 `json:"budget_total"`   // 预算总额
	BudgetUsed    float64 `json:"budget_used"`    // 已使用预算
	BudgetPercent float64 `json:"budget_percent"` // 预算使用率（%）

	// 预测
	PredictedCost float64 `json:"predicted_cost"` // 预测月底成本

	// 统计时间范围
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
}

// ===== 成本趋势 =====

// CostTrendRequest 成本趋势请求
type CostTrendRequest struct {
	ClusterID uint   `form:"cluster_id"`
	Period    string `form:"period"`    // day/week/month
	Days      int    `form:"days"`      // 天数，默认 30
	Dimension string `form:"dimension"` // namespace/app/team
}

// CostTrendItem 成本趋势项
type CostTrendItem struct {
	Date          string  `json:"date"`
	TotalCost     float64 `json:"total_cost"`
	CPUCost       float64 `json:"cpu_cost"`
	MemoryCost    float64 `json:"memory_cost"`
	StorageCost   float64 `json:"storage_cost"`
	PredictedCost float64 `json:"predicted_cost"` // 预测值（用于未来日期）
}

// CostTrendResponse 成本趋势响应
type CostTrendResponse struct {
	Items           []CostTrendItem `json:"items"`
	TrendDirection  string          `json:"trend_direction"`  // up/down/stable
	TrendPercentage float64         `json:"trend_percentage"` // 趋势变化率
	Prediction      []CostTrendItem `json:"prediction"`       // 未来7天预测
	Anomalies       []CostAnomaly   `json:"anomalies"`        // 异常点
}

// CostAnomaly 成本异常
type CostAnomaly struct {
	Date         string  `json:"date"`
	ActualCost   float64 `json:"actual_cost"`
	ExpectedCost float64 `json:"expected_cost"`
	Deviation    float64 `json:"deviation"` // 偏差百分比
	Reason       string  `json:"reason"`    // 可能原因
}

// ===== 成本分布 =====

// CostDistributionRequest 成本分布请求
type CostDistributionRequest struct {
	ClusterID uint   `form:"cluster_id"`
	Dimension string `form:"dimension"` // namespace/app/team/resource_type
	StartTime string `form:"start_time"`
	EndTime   string `form:"end_time"`
	TopN      int    `form:"top_n"` // 返回 Top N，默认 10
}

// CostDistributionItem 成本分布项
type CostDistributionItem struct {
	Name        string  `json:"name"`
	TotalCost   float64 `json:"total_cost"`
	CPUCost     float64 `json:"cpu_cost"`
	MemoryCost  float64 `json:"memory_cost"`
	StorageCost float64 `json:"storage_cost"`
	Percentage  float64 `json:"percentage"` // 占比（%）
}

// CostDistributionResponse 成本分布响应
type CostDistributionResponse struct {
	Dimension string                 `json:"dimension"`
	Items     []CostDistributionItem `json:"items"`
	Total     float64                `json:"total"`
}

// ===== 资源利用率 =====

// ResourceUsageRequest 资源利用率请求
type ResourceUsageRequest struct {
	ClusterID uint   `form:"cluster_id"`
	Namespace string `form:"namespace"`
	Status    string `form:"status"` // idle/underutilized/normal/overprovisioned
	TopN      int    `form:"top_n"`  // 返回 Top N，默认 20
}

// ResourceUsageItem 资源利用率项
type ResourceUsageItem struct {
	Namespace    string `json:"namespace"`
	ResourceType string `json:"resource_type"`
	ResourceName string `json:"resource_name"`
	AppName      string `json:"app_name"`

	CPURequest   float64 `json:"cpu_request"`
	CPULimit     float64 `json:"cpu_limit"`
	CPUUsage     float64 `json:"cpu_usage"`
	CPUUsageRate float64 `json:"cpu_usage_rate"` // CPU 利用率（%）

	MemoryRequest   float64 `json:"memory_request"`
	MemoryLimit     float64 `json:"memory_limit"`
	MemoryUsage     float64 `json:"memory_usage"`
	MemoryUsageRate float64 `json:"memory_usage_rate"` // 内存利用率（%）

	TotalCost  float64 `json:"total_cost"`
	WastedCost float64 `json:"wasted_cost"` // 浪费成本
	Status     string  `json:"status"`      // normal/underutilized/overprovisioned/idle
	IdleDays   int     `json:"idle_days"`   // 闲置天数
	LastActive string  `json:"last_active"` // 最后活跃时间
	Suggestion string  `json:"suggestion"`  // 优化建议
}

// ResourceUsageResponse 资源利用率响应
type ResourceUsageResponse struct {
	Items          []ResourceUsageItem `json:"items"`
	AvgCPUUsage    float64             `json:"avg_cpu_usage"`
	AvgMemoryUsage float64             `json:"avg_memory_usage"`
	TotalWasted    float64             `json:"total_wasted"`    // 总浪费成本
	IdleCount      int                 `json:"idle_count"`      // 闲置资源数
	UnderutilCount int                 `json:"underutil_count"` // 低利用资源数
	Summary        UsageSummary        `json:"summary"`
}

// UsageSummary 利用率汇总
type UsageSummary struct {
	TotalResources  int     `json:"total_resources"`
	IdleResources   int     `json:"idle_resources"`   // <10%
	LowResources    int     `json:"low_resources"`    // 10-30%
	NormalResources int     `json:"normal_resources"` // 30-70%
	HighResources   int     `json:"high_resources"`   // >70%
	AvgEfficiency   float64 `json:"avg_efficiency"`   // 平均效率分
}

// ===== 成本优化建议 =====

// CostSuggestionItem 成本优化建议项
type CostSuggestionItem struct {
	ID             uint    `json:"id"`
	ClusterID      uint    `json:"cluster_id"`
	ClusterName    string  `json:"cluster_name"`
	Namespace      string  `json:"namespace"`
	ResourceType   string  `json:"resource_type"`
	ResourceName   string  `json:"resource_name"`
	SuggestionType string  `json:"suggestion_type"`
	Severity       string  `json:"severity"`
	Title          string  `json:"title"`
	Description    string  `json:"description"`
	CurrentCost    float64 `json:"current_cost"`
	OptimizedCost  float64 `json:"optimized_cost"`
	Savings        float64 `json:"savings"`
	SavingsPercent float64 `json:"savings_percent"`
	Status         string  `json:"status"`
	CreatedAt      string  `json:"created_at"`
}

// CostSuggestionListResponse 成本优化建议列表响应
type CostSuggestionListResponse struct {
	Items        []CostSuggestionItem `json:"items"`
	Total        int64                `json:"total"`
	TotalSavings float64              `json:"total_savings"`
	HighCount    int                  `json:"high_count"`
	MediumCount  int                  `json:"medium_count"`
	LowCount     int                  `json:"low_count"`
}

// ApplySuggestionRequest 应用优化建议请求
type ApplySuggestionRequest struct {
	SuggestionID uint `json:"suggestion_id" binding:"required"`
}

// ===== 成本配置 =====

// CostConfigRequest 成本配置请求
type CostConfigRequest struct {
	CPUPricePerCore   float64 `json:"cpu_price_per_core" binding:"required,min=0"`
	MemoryPricePerGB  float64 `json:"memory_price_per_gb" binding:"required,min=0"`
	StoragePricePerGB float64 `json:"storage_price_per_gb" binding:"required,min=0"`
	Currency          string  `json:"currency"`
}

// CostConfigResponse 成本配置响应
type CostConfigResponse struct {
	ClusterID         uint    `json:"cluster_id"`
	CPUPricePerCore   float64 `json:"cpu_price_per_core"`
	MemoryPricePerGB  float64 `json:"memory_price_per_gb"`
	StoragePricePerGB float64 `json:"storage_price_per_gb"`
	Currency          string  `json:"currency"`
}

// ===== 成本报表 =====

// CostReportRequest 成本报表请求
type CostReportRequest struct {
	ClusterID uint   `form:"cluster_id"`
	StartTime string `form:"start_time" binding:"required"`
	EndTime   string `form:"end_time" binding:"required"`
	GroupBy   string `form:"group_by"` // namespace/app/team
}

// CostReportItem 成本报表项
type CostReportItem struct {
	Name           string  `json:"name"`
	CPUCost        float64 `json:"cpu_cost"`
	MemoryCost     float64 `json:"memory_cost"`
	StorageCost    float64 `json:"storage_cost"`
	TotalCost      float64 `json:"total_cost"`
	AvgCPUUsage    float64 `json:"avg_cpu_usage"`
	AvgMemoryUsage float64 `json:"avg_memory_usage"`
	Percentage     float64 `json:"percentage"`
}

// CostReportResponse 成本报表响应
type CostReportResponse struct {
	StartTime string           `json:"start_time"`
	EndTime   string           `json:"end_time"`
	TotalCost float64          `json:"total_cost"`
	Items     []CostReportItem `json:"items"`
}

// ===== 成本预测 =====

// CostForecastRequest 成本预测请求
type CostForecastRequest struct {
	ClusterID uint `form:"cluster_id"`
	Days      int  `form:"days"` // 预测天数，默认30
}

// CostForecastResponse 成本预测响应
type CostForecastResponse struct {
	CurrentMonthCost   float64          `json:"current_month_cost"`   // 本月已产生成本
	PredictedMonthCost float64          `json:"predicted_month_cost"` // 预测月底成本
	NextMonthCost      float64          `json:"next_month_cost"`      // 预测下月成本
	DailyForecast      []ForecastItem   `json:"daily_forecast"`       // 每日预测
	Confidence         float64          `json:"confidence"`           // 预测置信度
	Factors            []ForecastFactor `json:"factors"`              // 影响因素
}

// ForecastItem 预测项
type ForecastItem struct {
	Date          string  `json:"date"`
	PredictedCost float64 `json:"predicted_cost"`
	LowerBound    float64 `json:"lower_bound"` // 下限
	UpperBound    float64 `json:"upper_bound"` // 上限
}

// ForecastFactor 预测影响因素
type ForecastFactor struct {
	Name   string  `json:"name"`
	Impact float64 `json:"impact"` // 影响程度 -100 ~ 100
	Trend  string  `json:"trend"`  // up/down/stable
}

// ===== 成本对比 =====

// CostCompareRequest 成本对比请求
type CostCompareRequest struct {
	ClusterID    uint   `form:"cluster_id"`
	Period1Start string `form:"period1_start"`
	Period1End   string `form:"period1_end"`
	Period2Start string `form:"period2_start"`
	Period2End   string `form:"period2_end"`
	Dimension    string `form:"dimension"` // namespace/app/team
}

// CostCompareResponse 成本对比响应
type CostCompareResponse struct {
	Period1      CostPeriodSummary `json:"period1"`
	Period2      CostPeriodSummary `json:"period2"`
	Changes      []CostChangeItem  `json:"changes"`
	TopIncreases []CostChangeItem  `json:"top_increases"` // 增长最多
	TopDecreases []CostChangeItem  `json:"top_decreases"` // 下降最多
}

// CostPeriodSummary 周期成本汇总
type CostPeriodSummary struct {
	StartTime   string  `json:"start_time"`
	EndTime     string  `json:"end_time"`
	TotalCost   float64 `json:"total_cost"`
	CPUCost     float64 `json:"cpu_cost"`
	MemoryCost  float64 `json:"memory_cost"`
	StorageCost float64 `json:"storage_cost"`
	AvgDaily    float64 `json:"avg_daily"`
}

// CostChangeItem 成本变化项
type CostChangeItem struct {
	Name          string  `json:"name"`
	Period1Cost   float64 `json:"period1_cost"`
	Period2Cost   float64 `json:"period2_cost"`
	Change        float64 `json:"change"`
	ChangePercent float64 `json:"change_percent"`
}

// ===== 成本预算 =====

// CostBudgetRequest 成本预算请求
type CostBudgetRequest struct {
	ClusterID      uint    `json:"cluster_id"`
	Namespace      string  `json:"namespace"`       // 可选，按命名空间设置预算
	MonthlyBudget  float64 `json:"monthly_budget"`  // 月度预算
	AlertThreshold float64 `json:"alert_threshold"` // 告警阈值（%）
}

// CostBudgetResponse 成本预算响应
type CostBudgetResponse struct {
	ID             uint    `json:"id"`
	ClusterID      uint    `json:"cluster_id"`
	Namespace      string  `json:"namespace"`
	MonthlyBudget  float64 `json:"monthly_budget"`
	CurrentCost    float64 `json:"current_cost"`
	UsagePercent   float64 `json:"usage_percent"`
	AlertThreshold float64 `json:"alert_threshold"`
	Status         string  `json:"status"`          // normal/warning/exceeded
	PredictedUsage float64 `json:"predicted_usage"` // 预测月底使用率
}

// CostBudgetListResponse 预算列表响应
type CostBudgetListResponse struct {
	Items       []CostBudgetResponse `json:"items"`
	TotalBudget float64              `json:"total_budget"`
	TotalUsed   float64              `json:"total_used"`
	OverBudget  int                  `json:"over_budget"` // 超预算数量
	AtRisk      int                  `json:"at_risk"`     // 风险数量
}

// ===== 成本分摊 =====

// CostAllocationRequest 成本分摊请求
type CostAllocationRequest struct {
	ClusterID uint   `form:"cluster_id"`
	StartTime string `form:"start_time"`
	EndTime   string `form:"end_time"`
	GroupBy   string `form:"group_by"` // team/project/namespace/app
}

// CostAllocationItem 成本分摊项
type CostAllocationItem struct {
	Name          string  `json:"name"`
	CPUCost       float64 `json:"cpu_cost"`
	MemoryCost    float64 `json:"memory_cost"`
	StorageCost   float64 `json:"storage_cost"`
	NetworkCost   float64 `json:"network_cost"`
	SharedCost    float64 `json:"shared_cost"` // 分摊的公共成本
	TotalCost     float64 `json:"total_cost"`
	Percentage    float64 `json:"percentage"`
	ResourceCount int     `json:"resource_count"` // 资源数量
	AvgEfficiency float64 `json:"avg_efficiency"` // 平均效率
}

// CostAllocationResponse 成本分摊响应
type CostAllocationResponse struct {
	StartTime     string               `json:"start_time"`
	EndTime       string               `json:"end_time"`
	TotalCost     float64              `json:"total_cost"`
	SharedCost    float64              `json:"shared_cost"`    // 公共成本
	AllocatedCost float64              `json:"allocated_cost"` // 已分摊成本
	Items         []CostAllocationItem `json:"items"`
}

// ===== 资源浪费检测 =====

// WasteDetectionRequest 浪费检测请求
type WasteDetectionRequest struct {
	ClusterID uint `form:"cluster_id"`
	Days      int  `form:"days"` // 检测周期，默认7天
}

// WasteDetectionResponse 浪费检测响应
type WasteDetectionResponse struct {
	TotalWaste      float64      `json:"total_waste"`      // 总浪费成本
	WastePercent    float64      `json:"waste_percent"`    // 浪费占比
	IdleResources   []WasteItem  `json:"idle_resources"`   // 闲置资源
	Overprovisioned []WasteItem  `json:"overprovisioned"`  // 超配资源
	ZombieResources []WasteItem  `json:"zombie_resources"` // 僵尸资源
	UnusedVolumes   []WasteItem  `json:"unused_volumes"`   // 未使用存储
	Summary         WasteSummary `json:"summary"`
}

// WasteItem 浪费项
type WasteItem struct {
	ClusterID    uint    `json:"cluster_id"`
	ClusterName  string  `json:"cluster_name"`
	Namespace    string  `json:"namespace"`
	ResourceType string  `json:"resource_type"`
	ResourceName string  `json:"resource_name"`
	WasteType    string  `json:"waste_type"`    // idle/overprovisioned/zombie/unused
	WasteCost    float64 `json:"waste_cost"`    // 浪费成本
	CurrentUsage float64 `json:"current_usage"` // 当前使用率
	IdleDays     int     `json:"idle_days"`     // 闲置天数
	Suggestion   string  `json:"suggestion"`    // 优化建议
	Impact       string  `json:"impact"`        // high/medium/low
}

// WasteSummary 浪费汇总
type WasteSummary struct {
	IdleCount            int     `json:"idle_count"`
	IdleCost             float64 `json:"idle_cost"`
	OverprovisionedCount int     `json:"overprovisioned_count"`
	OverprovisionedCost  float64 `json:"overprovisioned_cost"`
	ZombieCount          int     `json:"zombie_count"`
	ZombieCost           float64 `json:"zombie_cost"`
	UnusedVolumeCount    int     `json:"unused_volume_count"`
	UnusedVolumeCost     float64 `json:"unused_volume_cost"`
}

// ===== 成本优化建议增强 =====

// CostSuggestionDetail 优化建议详情
type CostSuggestionDetail struct {
	CostSuggestionItem
	CurrentConfig   ResourceConfig `json:"current_config"`
	SuggestedConfig ResourceConfig `json:"suggested_config"`
	RiskLevel       string         `json:"risk_level"`     // low/medium/high
	Difficulty      string         `json:"difficulty"`     // easy/medium/hard
	EstimatedTime   string         `json:"estimated_time"` // 预计执行时间
	Prerequisites   []string       `json:"prerequisites"`  // 前置条件
	Steps           []string       `json:"steps"`          // 执行步骤
}

// ResourceConfig 资源配置
type ResourceConfig struct {
	CPURequest    string `json:"cpu_request"`
	CPULimit      string `json:"cpu_limit"`
	MemoryRequest string `json:"memory_request"`
	MemoryLimit   string `json:"memory_limit"`
	Replicas      int    `json:"replicas"`
}

// ===== 成本健康评分 =====

// CostHealthScoreResponse 成本健康评分响应
type CostHealthScoreResponse struct {
	OverallScore    int              `json:"overall_score"`   // 总分 0-100
	Grade           string           `json:"grade"`           // A/B/C/D/F
	Dimensions      []ScoreDimension `json:"dimensions"`      // 各维度得分
	Recommendations []string         `json:"recommendations"` // 改进建议
	Trend           string           `json:"trend"`           // improving/stable/declining
}

// ScoreDimension 评分维度
type ScoreDimension struct {
	Name        string `json:"name"`
	Score       int    `json:"score"`
	MaxScore    int    `json:"max_score"`
	Description string `json:"description"`
	Status      string `json:"status"` // good/warning/critical
}

// ===== 成本对比分析 =====

// CostComparisonRequest 成本对比请求
type CostComparisonRequest struct {
	ClusterID    uint   `form:"cluster_id"`
	Period1Start string `form:"period1_start" binding:"required"`
	Period1End   string `form:"period1_end" binding:"required"`
	Period2Start string `form:"period2_start" binding:"required"`
	Period2End   string `form:"period2_end" binding:"required"`
}

// CostComparisonResponse 成本对比响应
type CostComparisonResponse struct {
	Period1             PeriodInfo            `json:"period1"`
	Period2             PeriodInfo            `json:"period2"`
	TotalChange         float64               `json:"total_change"`
	TotalChangeRate     float64               `json:"total_change_rate"`
	CPUChange           float64               `json:"cpu_change"`
	CPUChangeRate       float64               `json:"cpu_change_rate"`
	MemoryChange        float64               `json:"memory_change"`
	MemoryChangeRate    float64               `json:"memory_change_rate"`
	StorageChange       float64               `json:"storage_change"`
	StorageChangeRate   float64               `json:"storage_change_rate"`
	NamespaceComparison []NamespaceComparison `json:"namespace_comparison"`
}

// PeriodInfo 周期信息
type PeriodInfo struct {
	Start string         `json:"start"`
	End   string         `json:"end"`
	Data  PeriodCostData `json:"data"`
}

// PeriodCostData 周期成本数据
type PeriodCostData struct {
	TotalCost      float64 `json:"total_cost"`
	CPUCost        float64 `json:"cpu_cost"`
	MemoryCost     float64 `json:"memory_cost"`
	StorageCost    float64 `json:"storage_cost"`
	AvgCPUUsage    float64 `json:"avg_cpu_usage"`
	AvgMemoryUsage float64 `json:"avg_memory_usage"`
}

// NamespaceComparison 命名空间对比
type NamespaceComparison struct {
	Namespace   string  `json:"namespace"`
	Period1Cost float64 `json:"period1_cost"`
	Period2Cost float64 `json:"period2_cost"`
	Change      float64 `json:"change"`
	ChangeRate  float64 `json:"change_rate"`
}

// ===== 成本告警 =====

// CostAlertItem 成本告警项
type CostAlertItem struct {
	ID          uint    `json:"id"`
	ClusterID   uint    `json:"cluster_id"`
	AlertType   string  `json:"alert_type"`
	Severity    string  `json:"severity"`
	Title       string  `json:"title"`
	Message     string  `json:"message"`
	Threshold   float64 `json:"threshold"`
	ActualValue float64 `json:"actual_value"`
	Status      string  `json:"status"`
	CreatedAt   string  `json:"created_at"`
}

// ===== 成本报表导出 =====

// CostExportRequest 成本报表导出请求
type CostExportRequest struct {
	ClusterID  uint   `form:"cluster_id"`
	StartTime  string `form:"start_time" binding:"required"`
	EndTime    string `form:"end_time" binding:"required"`
	ReportType string `form:"report_type"` // overview/comparison
}

// ===== 应用维度成本分析 =====

// AppCostRequest 应用成本请求
type AppCostRequest struct {
	ClusterID uint   `form:"cluster_id"`
	StartTime string `form:"start_time"`
	EndTime   string `form:"end_time"`
	TopN      int    `form:"top_n"`
}

// AppCostItem 应用成本项
type AppCostItem struct {
	AppName         string  `json:"app_name"`
	Namespace       string  `json:"namespace"`
	ResourceCount   int     `json:"resource_count"`
	CPURequest      float64 `json:"cpu_request"`
	CPUUsage        float64 `json:"cpu_usage"`
	CPUUsageRate    float64 `json:"cpu_usage_rate"`
	MemoryRequest   float64 `json:"memory_request"`
	MemoryUsage     float64 `json:"memory_usage"`
	MemoryUsageRate float64 `json:"memory_usage_rate"`
	CPUCost         float64 `json:"cpu_cost"`
	MemoryCost      float64 `json:"memory_cost"`
	StorageCost     float64 `json:"storage_cost"`
	TotalCost       float64 `json:"total_cost"`
	Percentage      float64 `json:"percentage"`
	Efficiency      float64 `json:"efficiency"` // 效率评分
}

// AppCostResponse 应用成本响应
type AppCostResponse struct {
	Items       []AppCostItem `json:"items"`
	TotalCost   float64       `json:"total_cost"`
	TotalApps   int           `json:"total_apps"`
	AvgCost     float64       `json:"avg_cost"`
	TopCostApps []string      `json:"top_cost_apps"`
}

// ===== 团队维度成本分析 =====

// TeamCostRequest 团队成本请求
type TeamCostRequest struct {
	ClusterID uint   `form:"cluster_id"`
	StartTime string `form:"start_time"`
	EndTime   string `form:"end_time"`
}

// TeamCostItem 团队成本项
type TeamCostItem struct {
	TeamName      string  `json:"team_name"`
	AppCount      int     `json:"app_count"`
	ResourceCount int     `json:"resource_count"`
	CPUCost       float64 `json:"cpu_cost"`
	MemoryCost    float64 `json:"memory_cost"`
	StorageCost   float64 `json:"storage_cost"`
	TotalCost     float64 `json:"total_cost"`
	Percentage    float64 `json:"percentage"`
	AvgEfficiency float64 `json:"avg_efficiency"`
	WastedCost    float64 `json:"wasted_cost"`
	BudgetUsed    float64 `json:"budget_used"`    // 预算使用率
	MonthlyBudget float64 `json:"monthly_budget"` // 月度预算
}

// TeamCostResponse 团队成本响应
type TeamCostResponse struct {
	Items      []TeamCostItem `json:"items"`
	TotalCost  float64        `json:"total_cost"`
	TotalTeams int            `json:"total_teams"`
	SharedCost float64        `json:"shared_cost"` // 未分配到团队的公共成本
}

// ===== 节点成本分析 =====

// NodeCostRequest 节点成本请求
type NodeCostRequest struct {
	ClusterID uint `form:"cluster_id"`
}

// NodeCostItem 节点成本项
type NodeCostItem struct {
	NodeName          string  `json:"node_name"`
	NodeIP            string  `json:"node_ip"`
	NodeType          string  `json:"node_type"` // master/worker
	InstanceType      string  `json:"instance_type"`
	CPUCapacity       float64 `json:"cpu_capacity"`
	CPUAllocatable    float64 `json:"cpu_allocatable"`
	CPURequested      float64 `json:"cpu_requested"`
	CPUUsage          float64 `json:"cpu_usage"`
	CPUUsageRate      float64 `json:"cpu_usage_rate"`
	MemoryCapacity    float64 `json:"memory_capacity"`
	MemoryAllocatable float64 `json:"memory_allocatable"`
	MemoryRequested   float64 `json:"memory_requested"`
	MemoryUsage       float64 `json:"memory_usage"`
	MemoryUsageRate   float64 `json:"memory_usage_rate"`
	PodCount          int     `json:"pod_count"`
	PodCapacity       int     `json:"pod_capacity"`
	EstimatedCost     float64 `json:"estimated_cost"` // 节点估算成本
	Efficiency        float64 `json:"efficiency"`     // 效率评分
	Status            string  `json:"status"`         // Ready/NotReady
}

// NodeCostResponse 节点成本响应
type NodeCostResponse struct {
	Items          []NodeCostItem `json:"items"`
	TotalNodes     int            `json:"total_nodes"`
	TotalCPU       float64        `json:"total_cpu"`
	TotalMemory    float64        `json:"total_memory"`
	AvgCPUUsage    float64        `json:"avg_cpu_usage"`
	AvgMemoryUsage float64        `json:"avg_memory_usage"`
	TotalCost      float64        `json:"total_cost"`
	UnderutilNodes int            `json:"underutil_nodes"` // 低利用率节点数
}

// ===== PVC 存储成本分析 =====

// PVCCostRequest PVC成本请求
type PVCCostRequest struct {
	ClusterID uint   `form:"cluster_id"`
	Namespace string `form:"namespace"`
}

// PVCCostItem PVC成本项
type PVCCostItem struct {
	Namespace    string  `json:"namespace"`
	PVCName      string  `json:"pvc_name"`
	StorageClass string  `json:"storage_class"`
	Capacity     float64 `json:"capacity"`     // GB
	Used         float64 `json:"used"`         // GB
	UsageRate    float64 `json:"usage_rate"`   // 使用率
	MonthlyCost  float64 `json:"monthly_cost"` // 月成本
	BoundPod     string  `json:"bound_pod"`    // 绑定的Pod
	Status       string  `json:"status"`       // Bound/Pending/Available
	IsUnused     bool    `json:"is_unused"`    // 是否未使用
	CreatedAt    string  `json:"created_at"`
}

// PVCCostResponse PVC成本响应
type PVCCostResponse struct {
	Items         []PVCCostItem `json:"items"`
	TotalPVCs     int           `json:"total_pvcs"`
	TotalCapacity float64       `json:"total_capacity"`
	TotalUsed     float64       `json:"total_used"`
	TotalCost     float64       `json:"total_cost"`
	UnusedPVCs    int           `json:"unused_pvcs"`
	UnusedCost    float64       `json:"unused_cost"`
	AvgUsageRate  float64       `json:"avg_usage_rate"`
}

// ===== 成本分摊报表 =====

// CostAllocationReportRequest 成本分摊报表请求
type CostAllocationReportRequest struct {
	ClusterID     uint   `form:"cluster_id"`
	StartTime     string `form:"start_time" binding:"required"`
	EndTime       string `form:"end_time" binding:"required"`
	GroupBy       string `form:"group_by"`       // team/namespace/app
	IncludeShared bool   `form:"include_shared"` // 是否分摊公共成本
}

// CostAllocationReportItem 成本分摊报表项
type CostAllocationReportItem struct {
	Name          string  `json:"name"`
	DirectCost    float64 `json:"direct_cost"` // 直接成本
	SharedCost    float64 `json:"shared_cost"` // 分摊的公共成本
	TotalCost     float64 `json:"total_cost"`  // 总成本
	CPUCost       float64 `json:"cpu_cost"`
	MemoryCost    float64 `json:"memory_cost"`
	StorageCost   float64 `json:"storage_cost"`
	Percentage    float64 `json:"percentage"`
	ResourceCount int     `json:"resource_count"`
	AvgEfficiency float64 `json:"avg_efficiency"`
}

// CostAllocationReportResponse 成本分摊报表响应
type CostAllocationReportResponse struct {
	StartTime       string                     `json:"start_time"`
	EndTime         string                     `json:"end_time"`
	GroupBy         string                     `json:"group_by"`
	TotalCost       float64                    `json:"total_cost"`
	DirectCost      float64                    `json:"direct_cost"`
	SharedCost      float64                    `json:"shared_cost"`
	Items           []CostAllocationReportItem `json:"items"`
	UnallocatedCost float64                    `json:"unallocated_cost"` // 未分配成本
}

// ===== 环境维度成本分析 =====

// EnvCostRequest 环境成本请求
type EnvCostRequest struct {
	ClusterID uint   `form:"cluster_id"`
	StartTime string `form:"start_time"`
	EndTime   string `form:"end_time"`
}

// EnvCostItem 环境成本项
type EnvCostItem struct {
	Environment    string  `json:"environment"` // dev/test/staging/prod
	NamespaceCount int     `json:"namespace_count"`
	AppCount       int     `json:"app_count"`
	CPUCost        float64 `json:"cpu_cost"`
	MemoryCost     float64 `json:"memory_cost"`
	StorageCost    float64 `json:"storage_cost"`
	TotalCost      float64 `json:"total_cost"`
	Percentage     float64 `json:"percentage"`
	AvgEfficiency  float64 `json:"avg_efficiency"`
}

// EnvCostResponse 环境成本响应
type EnvCostResponse struct {
	Items     []EnvCostItem `json:"items"`
	TotalCost float64       `json:"total_cost"`
}
