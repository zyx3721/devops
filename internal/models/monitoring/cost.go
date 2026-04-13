// Package monitoring 定义监控告警相关的数据模型
// 本文件包含成本相关的模型定义
package monitoring

import (
	"time"

	"gorm.io/gorm"
)

// ==================== 成本模型 ====================

// ResourceCost 资源成本记录
// 记录资源的成本数据
type ResourceCost struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	ClusterID    uint      `gorm:"index:idx_cluster_time" json:"cluster_id"`  // 集群ID
	Namespace    string    `gorm:"size:100;index:idx_namespace" json:"namespace"` // 命名空间
	ResourceType string    `gorm:"size:50" json:"resource_type"`              // 资源类型
	ResourceName string    `gorm:"size:200" json:"resource_name"`             // 资源名称
	AppName      string    `gorm:"size:100;index:idx_app" json:"app_name"`    // 应用名称
	TeamName     string    `gorm:"size:100;index:idx_team" json:"team_name"`  // 团队名称
	CPURequest   float64   `json:"cpu_request"`                               // CPU 请求值
	CPULimit     float64   `json:"cpu_limit"`                                 // CPU 限制值
	CPUUsage     float64   `json:"cpu_usage"`                                 // CPU 实际使用
	CPUCost      float64   `json:"cpu_cost"`                                  // CPU 成本
	MemoryRequest float64  `json:"memory_request"`                            // 内存请求值
	MemoryLimit  float64   `json:"memory_limit"`                              // 内存限制值
	MemoryUsage  float64   `json:"memory_usage"`                              // 内存实际使用
	MemoryCost   float64   `json:"memory_cost"`                               // 内存成本
	StorageSize  float64   `json:"storage_size"`                              // 存储大小
	StorageCost  float64   `json:"storage_cost"`                              // 存储成本
	TotalCost    float64   `json:"total_cost"`                                // 总成本
	RecordedAt   time.Time `gorm:"index:idx_cluster_time" json:"recorded_at"` // 记录时间
	CreatedAt    time.Time `json:"created_at"`
}

// TableName 指定表名
func (ResourceCost) TableName() string {
	return "resource_costs"
}


// CostSummary 成本汇总（按天/周/月）
// 汇总成本数据
type CostSummary struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	ClusterID      uint      `gorm:"index:idx_cluster_period" json:"cluster_id"`  // 集群ID
	Period         string    `gorm:"size:20;index:idx_cluster_period" json:"period"` // 周期: daily/weekly/monthly
	PeriodStart    time.Time `gorm:"index:idx_period_start" json:"period_start"`  // 周期开始
	PeriodEnd      time.Time `json:"period_end"`                                  // 周期结束
	Dimension      string    `gorm:"size:50" json:"dimension"`                    // 维度
	DimensionValue string    `gorm:"size:200" json:"dimension_value"`             // 维度值
	CPUCost        float64   `json:"cpu_cost"`                                    // CPU 成本
	MemoryCost     float64   `json:"memory_cost"`                                 // 内存成本
	StorageCost    float64   `json:"storage_cost"`                                // 存储成本
	TotalCost      float64   `json:"total_cost"`                                  // 总成本
	AvgCPUUsage    float64   `json:"avg_cpu_usage"`                               // 平均 CPU 使用
	AvgMemoryUsage float64   `json:"avg_memory_usage"`                            // 平均内存使用
	CreatedAt      time.Time `json:"created_at"`
}

// TableName 指定表名
func (CostSummary) TableName() string {
	return "cost_summaries"
}

// CostSuggestion 成本优化建议
// 存储成本优化建议
type CostSuggestion struct {
	ID              uint           `gorm:"primaryKey" json:"id"`
	ClusterID       uint           `gorm:"index" json:"cluster_id"`                    // 集群ID
	Namespace       string         `gorm:"size:100" json:"namespace"`                  // 命名空间
	ResourceType    string         `gorm:"size:50" json:"resource_type"`               // 资源类型
	ResourceName    string         `gorm:"size:200" json:"resource_name"`              // 资源名称
	SuggestionType  string         `gorm:"size:50" json:"suggestion_type"`             // 建议类型
	Severity        string         `gorm:"size:20" json:"severity"`                    // 严重程度
	Title           string         `gorm:"size:200" json:"title"`                      // 标题
	Description     string         `gorm:"type:text" json:"description"`               // 描述
	CurrentCost     float64        `json:"current_cost"`                               // 当前成本
	OptimizedCost   float64        `json:"optimized_cost"`                             // 优化后成本
	Savings         float64        `json:"savings"`                                    // 节省金额
	SavingsPercent  float64        `json:"savings_percent"`                            // 节省百分比
	CurrentConfig   string         `gorm:"type:text" json:"current_config"`            // 当前配置
	SuggestedConfig string         `gorm:"type:text" json:"suggested_config"`          // 建议配置
	Status          string         `gorm:"size:20;default:pending" json:"status"`      // 状态
	AppliedAt       *time.Time     `json:"applied_at"`                                 // 应用时间
	AppliedBy       *uint          `json:"applied_by"`                                 // 应用者
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
	DeletedAt       gorm.DeletedAt `gorm:"index" json:"-"`
}

// TableName 指定表名
func (CostSuggestion) TableName() string {
	return "cost_suggestions"
}

// CostConfig 成本配置（单价设置）
// 定义资源单价
type CostConfig struct {
	ID                uint      `gorm:"primaryKey" json:"id"`
	ClusterID         uint      `gorm:"uniqueIndex:idx_cluster_config" json:"cluster_id"` // 集群ID
	CPUPricePerCore   float64   `json:"cpu_price_per_core"`                               // CPU 单价
	MemoryPricePerGB  float64   `json:"memory_price_per_gb"`                              // 内存单价
	StoragePricePerGB float64   `json:"storage_price_per_gb"`                             // 存储单价
	Currency          string    `gorm:"size:10;default:CNY" json:"currency"`              // 货币单位
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

// TableName 指定表名
func (CostConfig) TableName() string {
	return "cost_configs"
}

// CostBudget 成本预算
// 定义成本预算
type CostBudget struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	ClusterID      uint      `gorm:"index" json:"cluster_id"`                    // 集群ID
	Namespace      string    `gorm:"size:100;index" json:"namespace"`            // 命名空间
	TeamName       string    `gorm:"size:100" json:"team_name"`                  // 团队名称
	MonthlyBudget  float64   `json:"monthly_budget"`                             // 月度预算
	AlertThreshold float64   `gorm:"default:80" json:"alert_threshold"`          // 告警阈值
	CurrentCost    float64   `json:"current_cost"`                               // 当前成本
	UsagePercent   float64   `json:"usage_percent"`                              // 使用百分比
	Status         string    `gorm:"size:20;default:normal" json:"status"`       // 状态
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// TableName 指定表名
func (CostBudget) TableName() string {
	return "cost_budgets"
}

// CostAlert 成本告警
// 记录成本告警
type CostAlert struct {
	ID             uint       `gorm:"primaryKey" json:"id"`
	ClusterID      uint       `gorm:"index" json:"cluster_id"`                   // 集群ID
	BudgetID       *uint      `gorm:"index" json:"budget_id"`                    // 预算ID
	AlertType      string     `gorm:"size:50" json:"alert_type"`                 // 告警类型
	Severity       string     `gorm:"size:20" json:"severity"`                   // 严重程度
	Title          string     `gorm:"size:200" json:"title"`                     // 标题
	Message        string     `gorm:"type:text" json:"message"`                  // 消息
	Threshold      float64    `json:"threshold"`                                 // 阈值
	ActualValue    float64    `json:"actual_value"`                              // 实际值
	Status         string     `gorm:"size:20;default:active" json:"status"`      // 状态
	AcknowledgedAt *time.Time `json:"acknowledged_at"`                           // 确认时间
	AcknowledgedBy *uint      `json:"acknowledged_by"`                           // 确认者
	ResolvedAt     *time.Time `json:"resolved_at"`                               // 解决时间
	CreatedAt      time.Time  `json:"created_at"`
}

// TableName 指定表名
func (CostAlert) TableName() string {
	return "cost_alerts"
}

// ResourceActivity 资源活跃度记录
// 记录资源活跃度
type ResourceActivity struct {
	ID             uint       `gorm:"primaryKey" json:"id"`
	ClusterID      uint       `gorm:"index:idx_cluster_time" json:"cluster_id"`  // 集群ID
	Namespace      string     `gorm:"size:100" json:"namespace"`                 // 命名空间
	ResourceType   string     `gorm:"size:50" json:"resource_type"`              // 资源类型
	ResourceName   string     `gorm:"size:200" json:"resource_name"`             // 资源名称
	RequestCount   int64      `json:"request_count"`                             // 请求数
	CPUUsageAvg    float64    `json:"cpu_usage_avg"`                             // 平均 CPU 使用
	MemoryUsageAvg float64    `json:"memory_usage_avg"`                          // 平均内存使用
	NetworkIn      int64      `json:"network_in"`                                // 入站流量
	NetworkOut     int64      `json:"network_out"`                               // 出站流量
	LastActiveAt   *time.Time `json:"last_active_at"`                            // 最后活跃时间
	IdleDays       int        `json:"idle_days"`                                 // 空闲天数
	IsZombie       bool       `json:"is_zombie"`                                 // 是否僵尸资源
	RecordedAt     time.Time  `gorm:"index:idx_cluster_time" json:"recorded_at"` // 记录时间
	CreatedAt      time.Time  `json:"created_at"`
}

// TableName 指定表名
func (ResourceActivity) TableName() string {
	return "resource_activities"
}
