// Package traffic 定义流量治理相关的数据模型
// 本文件包含流量治理核心模型定义
package traffic

import (
	"database/sql/driver"
	"encoding/json"
	"time"
)

// JSONDestinations 路由目标配置
type JSONDestinations []RouteDestination

// RouteDestination 路由目标
type RouteDestination struct {
	Subset string `json:"subset"` // 子集名称
	Weight int    `json:"weight"` // 权重
}

// Scan 实现 sql.Scanner 接口
func (j *JSONDestinations) Scan(value any) error {
	if value == nil {
		*j = nil
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(bytes, j)
}

// Value 实现 driver.Valuer 接口
func (j JSONDestinations) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	return json.Marshal(j)
}

// JSONArray 用于存储 JSON 数组
type JSONArray []string

// Scan 实现 sql.Scanner 接口
func (j *JSONArray) Scan(value any) error {
	if value == nil {
		*j = nil
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(bytes, j)
}


// Value 实现 driver.Valuer 接口
func (j JSONArray) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	return json.Marshal(j)
}

// TrafficRateLimitRule 限流规则表
type TrafficRateLimitRule struct {
	ID              uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	AppID           uint64    `gorm:"index;not null;comment:应用ID" json:"app_id"`
	Name            string    `gorm:"size:100;not null;comment:规则名称" json:"name"`
	Description     string    `gorm:"size:500;comment:规则描述" json:"description"`
	ResourceType    string    `gorm:"size:20;default:api;comment:资源类型" json:"resource_type"`
	Resource        string    `gorm:"size:500;not null;comment:资源标识" json:"resource"`
	Method          string    `gorm:"size:10;comment:请求方法" json:"method"`
	Strategy        string    `gorm:"size:20;default:qps;comment:限流策略" json:"strategy"`
	Threshold       int       `gorm:"default:100;comment:阈值" json:"threshold"`
	Burst           int       `gorm:"default:10;comment:突发容量" json:"burst"`
	QueueSize       int       `gorm:"default:100;comment:队列大小" json:"queue_size"`
	ControlBehavior string    `gorm:"size:20;default:reject;comment:超限行为" json:"control_behavior"`
	WarmUpPeriod    int       `gorm:"default:10;comment:预热时长" json:"warm_up_period"`
	MaxQueueTime    int       `gorm:"default:500;comment:最大排队时间" json:"max_queue_time"`
	LimitDimensions JSONArray `gorm:"type:json;comment:限流维度" json:"limit_dimensions"`
	LimitHeader     string    `gorm:"size:100;comment:限流Header名" json:"limit_header"`
	RejectedCode    int       `gorm:"default:429;comment:拒绝状态码" json:"rejected_code"`
	RejectedMessage string    `gorm:"size:500;default:Too Many Requests;comment:拒绝消息" json:"rejected_message"`
	Enabled         bool      `gorm:"default:true;comment:是否启用" json:"enabled"`
	Priority        int       `gorm:"default:100;comment:优先级" json:"priority"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

func (TrafficRateLimitRule) TableName() string { return "traffic_ratelimit_rules" }

// TrafficCircuitBreakerRule 熔断规则表
type TrafficCircuitBreakerRule struct {
	ID               uint64     `gorm:"primaryKey;autoIncrement" json:"id"`
	AppID            uint64     `gorm:"index;not null;comment:应用ID" json:"app_id"`
	Name             string     `gorm:"size:100;not null;comment:规则名称" json:"name"`
	Resource         string     `gorm:"size:500;not null;comment:资源标识" json:"resource"`
	Strategy         string     `gorm:"size:20;default:slow_request;comment:熔断策略" json:"strategy"`
	SlowRtThreshold  int        `gorm:"default:1000;comment:慢调用RT阈值" json:"slow_rt_threshold"`
	Threshold        float64    `gorm:"not null;comment:阈值" json:"threshold"`
	StatInterval     int        `gorm:"default:10;comment:统计窗口" json:"stat_interval"`
	MinRequestAmount int        `gorm:"default:5;comment:最小请求数" json:"min_request_amount"`
	RecoveryTimeout  int        `gorm:"default:30;comment:熔断时长" json:"recovery_timeout"`
	ProbeNum         int        `gorm:"default:3;comment:半开探测请求数" json:"probe_num"`
	FallbackStrategy string     `gorm:"size:20;default:return_error;comment:降级策略" json:"fallback_strategy"`
	FallbackValue    string     `gorm:"type:text;comment:降级返回值" json:"fallback_value"`
	FallbackService  string     `gorm:"size:200;comment:降级服务地址" json:"fallback_service"`
	CircuitStatus    string     `gorm:"size:20;default:closed;comment:熔断状态" json:"circuit_status"`
	LastOpenTime     *time.Time `gorm:"comment:上次熔断时间" json:"last_open_time"`
	Enabled          bool       `gorm:"default:true;comment:是否启用" json:"enabled"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

func (TrafficCircuitBreakerRule) TableName() string { return "traffic_circuitbreaker_rules" }

// TrafficRoutingRule 流量路由规则表
type TrafficRoutingRule struct {
	ID            uint64           `gorm:"primaryKey;autoIncrement" json:"id"`
	AppID         uint64           `gorm:"index;not null;comment:应用ID" json:"app_id"`
	Name          string           `gorm:"size:100;not null;comment:规则名称" json:"name"`
	Description   string           `gorm:"size:500;comment:规则描述" json:"description"`
	Priority      int              `gorm:"default:100;comment:优先级" json:"priority"`
	RouteType     string           `gorm:"size:20;default:weight;comment:路由类型" json:"route_type"`
	Destinations  JSONDestinations `gorm:"type:json;comment:目标配置" json:"destinations"`
	MatchKey      string           `gorm:"size:100;comment:匹配键" json:"match_key"`
	MatchOperator string           `gorm:"size:20;default:exact;comment:匹配方式" json:"match_operator"`
	MatchValue    string           `gorm:"size:500;comment:匹配值" json:"match_value"`
	TargetSubset  string           `gorm:"size:100;comment:目标子集" json:"target_subset"`
	Enabled       bool             `gorm:"default:true;comment:是否启用" json:"enabled"`
	CreatedAt     time.Time        `json:"created_at"`
	UpdatedAt     time.Time        `json:"updated_at"`
}

func (TrafficRoutingRule) TableName() string { return "traffic_routing_rules" }

// TrafficLoadBalanceConfig 负载均衡配置表
type TrafficLoadBalanceConfig struct {
	ID                     uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	AppID                  uint64    `gorm:"uniqueIndex;not null;comment:应用ID" json:"app_id"`
	LbPolicy               string    `gorm:"size:20;default:round_robin;comment:负载均衡算法" json:"lb_policy"`
	HashKey                string    `gorm:"size:20;comment:哈希键类型" json:"hash_key"`
	HashKeyName            string    `gorm:"size:100;comment:哈希键名称" json:"hash_key_name"`
	RingSize               int       `gorm:"default:1024;comment:一致性哈希环大小" json:"ring_size"`
	ChoiceCount            int       `gorm:"default:2;comment:最少请求选择数量" json:"choice_count"`
	WarmupDuration         string    `gorm:"size:20;default:60s;comment:预热时间" json:"warmup_duration"`
	HealthCheckEnabled     bool      `gorm:"default:false;comment:是否启用健康检查" json:"health_check_enabled"`
	HealthCheckPath        string    `gorm:"size:200;default:/health;comment:健康检查路径" json:"health_check_path"`
	HealthCheckInterval    string    `gorm:"size:20;default:10s;comment:检查间隔" json:"health_check_interval"`
	HealthCheckTimeout     string    `gorm:"size:20;default:5s;comment:检查超时" json:"health_check_timeout"`
	HealthyThreshold       int       `gorm:"default:2;comment:健康阈值" json:"healthy_threshold"`
	UnhealthyThreshold     int       `gorm:"default:3;comment:不健康阈值" json:"unhealthy_threshold"`
	HTTPMaxConnections     int       `gorm:"default:1024;comment:HTTP最大连接数" json:"http_max_connections"`
	HTTPMaxRequestsPerConn int       `gorm:"default:0;comment:每连接最大请求数" json:"http_max_requests_per_conn"`
	HTTPMaxPendingRequests int       `gorm:"default:1024;comment:最大等待请求数" json:"http_max_pending_requests"`
	HTTPMaxRetries         int       `gorm:"default:3;comment:最大重试次数" json:"http_max_retries"`
	HTTPIdleTimeout        string    `gorm:"size:20;default:1h;comment:HTTP空闲超时" json:"http_idle_timeout"`
	TCPMaxConnections      int       `gorm:"default:1024;comment:TCP最大连接数" json:"tcp_max_connections"`
	TCPConnectTimeout      string    `gorm:"size:20;default:10s;comment:TCP连接超时" json:"tcp_connect_timeout"`
	TCPKeepaliveEnabled    bool      `gorm:"default:true;comment:TCP Keepalive" json:"tcp_keepalive_enabled"`
	TCPKeepaliveInterval   string    `gorm:"size:20;default:60s;comment:Keepalive间隔" json:"tcp_keepalive_interval"`
	CreatedAt              time.Time `json:"created_at"`
	UpdatedAt              time.Time `json:"updated_at"`
}

func (TrafficLoadBalanceConfig) TableName() string { return "traffic_loadbalance_config" }

// TrafficTimeoutConfig 超时重试配置表
type TrafficTimeoutConfig struct {
	ID            uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	AppID         uint64    `gorm:"uniqueIndex;not null;comment:应用ID" json:"app_id"`
	Timeout       string    `gorm:"size:20;default:30s;comment:请求超时" json:"timeout"`
	Retries       int       `gorm:"default:3;comment:重试次数" json:"retries"`
	PerTryTimeout string    `gorm:"size:20;default:10s;comment:单次重试超时" json:"per_try_timeout"`
	RetryOn       JSONArray `gorm:"type:json;default:[\"5xx\"];comment:重试条件" json:"retry_on"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func (TrafficTimeoutConfig) TableName() string { return "traffic_timeout_config" }

// TrafficMirrorRule 流量镜像规则表
type TrafficMirrorRule struct {
	ID            uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	AppID         uint64    `gorm:"index;not null;comment:应用ID" json:"app_id"`
	TargetService string    `gorm:"size:200;not null;comment:目标服务" json:"target_service"`
	TargetSubset  string    `gorm:"size:100;comment:目标子集" json:"target_subset"`
	Percentage    int       `gorm:"default:100;comment:镜像比例" json:"percentage"`
	Enabled       bool      `gorm:"default:true;comment:是否启用" json:"enabled"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func (TrafficMirrorRule) TableName() string { return "traffic_mirror_rules" }

// TrafficFaultRule 故障注入规则表
type TrafficFaultRule struct {
	ID            uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	AppID         uint64    `gorm:"index;not null;comment:应用ID" json:"app_id"`
	Type          string    `gorm:"size:20;default:delay;comment:故障类型" json:"type"`
	Path          string    `gorm:"size:500;default:/;comment:接口路径" json:"path"`
	DelayDuration string    `gorm:"size:20;default:5s;comment:延迟时间" json:"delay_duration"`
	AbortCode     int       `gorm:"default:500;comment:HTTP状态码" json:"abort_code"`
	AbortMessage  string    `gorm:"size:500;comment:错误消息" json:"abort_message"`
	Percentage    int       `gorm:"default:10;comment:影响比例" json:"percentage"`
	Enabled       bool      `gorm:"default:false;comment:是否启用" json:"enabled"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func (TrafficFaultRule) TableName() string { return "traffic_fault_rules" }

// TrafficOperationLog 流量治理操作日志表
type TrafficOperationLog struct {
	ID        uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	AppID     uint64    `gorm:"index;not null;comment:应用ID" json:"app_id"`
	RuleType  string    `gorm:"size:50;not null;comment:规则类型" json:"rule_type"`
	RuleID    uint64    `gorm:"comment:规则ID" json:"rule_id"`
	Operation string    `gorm:"size:50;not null;comment:操作类型" json:"operation"`
	Operator  string    `gorm:"size:100;comment:操作人" json:"operator"`
	OldValue  string    `gorm:"type:json;comment:旧值" json:"old_value"`
	NewValue  string    `gorm:"type:json;comment:新值" json:"new_value"`
	CreatedAt time.Time `gorm:"index" json:"created_at"`
}

func (TrafficOperationLog) TableName() string { return "traffic_operation_logs" }

// TrafficStatistics 流量统计表
type TrafficStatistics struct {
	ID                uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	AppID             uint64    `gorm:"index;not null;comment:应用ID" json:"app_id"`
	Timestamp         time.Time `gorm:"index;not null;comment:统计时间" json:"timestamp"`
	TotalRequests     int64     `gorm:"default:0;comment:总请求数" json:"total_requests"`
	SuccessRequests   int64     `gorm:"default:0;comment:成功请求数" json:"success_requests"`
	FailedRequests    int64     `gorm:"default:0;comment:失败请求数" json:"failed_requests"`
	RateLimitedCount  int64     `gorm:"default:0;comment:限流次数" json:"rate_limited_count"`
	CircuitBreakCount int64     `gorm:"default:0;comment:熔断次数" json:"circuit_break_count"`
	AvgLatencyMs      float64   `gorm:"default:0;comment:平均延迟(ms)" json:"avg_latency_ms"`
	P50LatencyMs      float64   `gorm:"default:0;comment:P50延迟(ms)" json:"p50_latency_ms"`
	P90LatencyMs      float64   `gorm:"default:0;comment:P90延迟(ms)" json:"p90_latency_ms"`
	P99LatencyMs      float64   `gorm:"default:0;comment:P99延迟(ms)" json:"p99_latency_ms"`
	CreatedAt         time.Time `json:"created_at"`
}

func (TrafficStatistics) TableName() string { return "traffic_statistics" }

// TrafficRuleVersion 规则版本表（用于回滚）
type TrafficRuleVersion struct {
	ID          uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	AppID       uint64    `gorm:"index;not null;comment:应用ID" json:"app_id"`
	RuleType    string    `gorm:"size:50;not null;comment:规则类型" json:"rule_type"`
	RuleID      uint64    `gorm:"index;not null;comment:规则ID" json:"rule_id"`
	Version     int       `gorm:"not null;comment:版本号" json:"version"`
	Content     string    `gorm:"type:json;not null;comment:规则内容" json:"content"`
	Operator    string    `gorm:"size:100;comment:操作人" json:"operator"`
	Description string    `gorm:"size:500;comment:版本描述" json:"description"`
	CreatedAt   time.Time `gorm:"index" json:"created_at"`
}

func (TrafficRuleVersion) TableName() string { return "traffic_rule_versions" }

// CanaryRelease 金丝雀发布配置
type CanaryRelease struct {
	ID               uint64     `gorm:"primaryKey;autoIncrement" json:"id"`
	AppID            uint64     `gorm:"index;not null;comment:应用ID" json:"app_id"`
	Name             string     `gorm:"size:100;not null;comment:发布名称" json:"name"`
	EnvName          string     `gorm:"size:50;comment:环境名称" json:"env_name"`
	Status           string     `gorm:"size:20;default:pending;comment:状态" json:"status"` // pending, running, paused, completed, rollback, canary_running, success, rolled_back
	StableVersion    string     `gorm:"size:100;comment:稳定版本" json:"stable_version"`
	CanaryVersion    string     `gorm:"size:100;comment:金丝雀版本" json:"canary_version"`
	CurrentWeight    int        `gorm:"default:0;comment:当前金丝雀权重" json:"current_weight"`
	TargetWeight     int        `gorm:"default:100;comment:目标权重" json:"target_weight"`
	WeightIncrement  int        `gorm:"default:10;comment:权重增量" json:"weight_increment"`
	IntervalSeconds  int        `gorm:"default:60;comment:增量间隔(秒)" json:"interval_seconds"`
	SuccessThreshold float64    `gorm:"default:95;comment:成功率阈值" json:"success_threshold"`
	LatencyThreshold int        `gorm:"default:500;comment:延迟阈值(ms)" json:"latency_threshold"`
	AutoRollback     bool       `gorm:"default:true;comment:自动回滚" json:"auto_rollback"`
	StartedAt        *time.Time `gorm:"comment:开始时间" json:"started_at"`
	CompletedAt      *time.Time `gorm:"comment:完成时间" json:"completed_at"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

func (CanaryRelease) TableName() string { return "canary_releases" }

// BlueGreenDeployment 蓝绿部署配置
type BlueGreenDeployment struct {
	ID            uint64     `gorm:"primaryKey;autoIncrement" json:"id"`
	AppID         uint64     `gorm:"index;not null;comment:应用ID" json:"app_id"`
	Name          string     `gorm:"size:100;not null;comment:部署名称" json:"name"`
	EnvName       string     `gorm:"size:50;comment:环境名称" json:"env_name"`
	Status        string     `gorm:"size:20;default:pending;comment:状态" json:"status"` // pending, blue_active, green_active, switching, switched, rolled_back, completed
	BlueVersion   string     `gorm:"size:100;comment:蓝版本" json:"blue_version"`
	GreenVersion  string     `gorm:"size:100;comment:绿版本" json:"green_version"`
	ActiveColor   string     `gorm:"size:10;default:blue;comment:当前活跃" json:"active_color"`
	Replicas      int        `gorm:"default:2;comment:副本数" json:"replicas"`
	WarmupSeconds int        `gorm:"default:30;comment:预热时间(秒)" json:"warmup_seconds"`
	SwitchedAt    *time.Time `gorm:"comment:切换时间" json:"switched_at"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

func (BlueGreenDeployment) TableName() string { return "blue_green_deployments" }

// ==================== 应用级流量规则 ====================

// AppRateLimitRule 应用限流规则
type AppRateLimitRule struct {
	ID                uint      `gorm:"primarykey" json:"id"`
	AppID             uint      `gorm:"index;not null" json:"app_id"`
	Path              string    `gorm:"size:500;not null" json:"path"`
	Method            string    `gorm:"size:20" json:"method"`
	RequestsPerSecond int       `gorm:"default:100" json:"requests_per_second"`
	Burst             int       `gorm:"default:10" json:"burst"`
	Enabled           bool      `gorm:"default:true" json:"enabled"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

func (AppRateLimitRule) TableName() string { return "app_ratelimit_rules" }

// AppMirrorRule 应用流量镜像规则
type AppMirrorRule struct {
	ID            uint      `gorm:"primarykey" json:"id"`
	AppID         uint      `gorm:"index;not null" json:"app_id"`
	TargetService string    `gorm:"size:200;not null" json:"target_service"`
	TargetSubset  string    `gorm:"size:100" json:"target_subset"`
	Percentage    int       `gorm:"default:100" json:"percentage"`
	Enabled       bool      `gorm:"default:true" json:"enabled"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func (AppMirrorRule) TableName() string { return "app_mirror_rules" }

// AppFaultRule 应用故障注入规则
type AppFaultRule struct {
	ID            uint      `gorm:"primarykey" json:"id"`
	AppID         uint      `gorm:"index;not null" json:"app_id"`
	Type          string    `gorm:"size:20;not null" json:"type"` // delay, abort
	Path          string    `gorm:"size:500" json:"path"`
	DelayDuration string    `gorm:"size:20" json:"delay_duration"`
	AbortCode     int       `json:"abort_code"`
	Percentage    int       `gorm:"default:10" json:"percentage"`
	Enabled       bool      `gorm:"default:false" json:"enabled"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func (AppFaultRule) TableName() string { return "app_fault_rules" }
