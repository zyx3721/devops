// Package traffic 定义流量治理相关的数据模型
// 本文件包含规则模板相关模型定义
package traffic

import (
	"time"
)

// TrafficRuleTemplate 流量治理规则模板
// 用于预定义常用的流量治理规则配置，方便快速应用
type TrafficRuleTemplate struct {
	ID          uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	Name        string    `gorm:"size:100;not null;uniqueIndex;comment:模板名称" json:"name"`
	Description string    `gorm:"size:500;comment:模板描述" json:"description"`
	Category    string    `gorm:"size:50;index;comment:模板分类" json:"category"` // ratelimit, circuitbreaker, routing, loadbalance, timeout, mirror, fault
	RuleType    string    `gorm:"size:50;not null;index;comment:规则类型" json:"rule_type"`
	Content     string    `gorm:"type:json;not null;comment:规则内容" json:"content"`
	IsBuiltin   bool      `gorm:"default:false;comment:是否内置模板" json:"is_builtin"`
	IsPublic    bool      `gorm:"default:true;comment:是否公开" json:"is_public"`
	UsageCount  int       `gorm:"default:0;comment:使用次数" json:"usage_count"`
	CreatedBy   string    `gorm:"size:100;comment:创建人" json:"created_by"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (TrafficRuleTemplate) TableName() string { return "traffic_rule_templates" }

// BuiltinTemplates 内置模板列表
// 提供常用的流量治理规则模板
var BuiltinTemplates = []TrafficRuleTemplate{
	// 限流模板
	{
		Name:        "api-qps-100",
		Description: "API 接口 QPS 限流 100/秒",
		Category:    "ratelimit",
		RuleType:    "ratelimit",
		Content:     `{"strategy":"qps","threshold":100,"burst":10,"control_behavior":"reject","rejected_code":429,"rejected_message":"请求过于频繁，请稍后重试"}`,
		IsBuiltin:   true,
		IsPublic:    true,
	},
	{
		Name:        "api-qps-1000",
		Description: "API 接口 QPS 限流 1000/秒",
		Category:    "ratelimit",
		RuleType:    "ratelimit",
		Content:     `{"strategy":"qps","threshold":1000,"burst":100,"control_behavior":"reject","rejected_code":429,"rejected_message":"请求过于频繁，请稍后重试"}`,
		IsBuiltin:   true,
		IsPublic:    true,
	},
	{
		Name:        "api-warmup",
		Description: "API 接口预热限流（冷启动保护）",
		Category:    "ratelimit",
		RuleType:    "ratelimit",
		Content:     `{"strategy":"qps","threshold":500,"burst":50,"control_behavior":"warm_up","warm_up_period":30,"rejected_code":429}`,
		IsBuiltin:   true,
		IsPublic:    true,
	},
	// 熔断模板
	{
		Name:        "error-ratio-50",
		Description: "错误率熔断 50%",
		Category:    "circuitbreaker",
		RuleType:    "circuitbreaker",
		Content:     `{"strategy":"error_ratio","threshold":0.5,"stat_interval":10,"min_request_amount":10,"recovery_timeout":30,"fallback_strategy":"return_error"}`,
		IsBuiltin:   true,
		IsPublic:    true,
	},
	{
		Name:        "slow-request-1s",
		Description: "慢调用熔断（RT > 1秒）",
		Category:    "circuitbreaker",
		RuleType:    "circuitbreaker",
		Content:     `{"strategy":"slow_request","slow_rt_threshold":1000,"threshold":0.5,"stat_interval":10,"min_request_amount":5,"recovery_timeout":30}`,
		IsBuiltin:   true,
		IsPublic:    true,
	},
	{
		Name:        "error-count-10",
		Description: "连续错误熔断（10次）",
		Category:    "circuitbreaker",
		RuleType:    "circuitbreaker",
		Content:     `{"strategy":"error_count","threshold":10,"stat_interval":60,"recovery_timeout":60,"fallback_strategy":"return_error"}`,
		IsBuiltin:   true,
		IsPublic:    true,
	},
	// 路由模板
	{
		Name:        "canary-10-percent",
		Description: "金丝雀发布 10% 流量",
		Category:    "routing",
		RuleType:    "routing",
		Content:     `{"route_type":"weight","destinations":[{"subset":"stable","weight":90},{"subset":"canary","weight":10}]}`,
		IsBuiltin:   true,
		IsPublic:    true,
	},
	{
		Name:        "header-based-routing",
		Description: "基于 Header 的路由（测试用户）",
		Category:    "routing",
		RuleType:    "routing",
		Content:     `{"route_type":"header","match_key":"x-user-type","match_operator":"exact","match_value":"test","target_subset":"canary"}`,
		IsBuiltin:   true,
		IsPublic:    true,
	},
	{
		Name:        "blue-green-switch",
		Description: "蓝绿部署切换",
		Category:    "routing",
		RuleType:    "routing",
		Content:     `{"route_type":"weight","destinations":[{"subset":"blue","weight":0},{"subset":"green","weight":100}]}`,
		IsBuiltin:   true,
		IsPublic:    true,
	},
	// 负载均衡模板
	{
		Name:        "round-robin",
		Description: "轮询负载均衡",
		Category:    "loadbalance",
		RuleType:    "loadbalance",
		Content:     `{"lb_policy":"round_robin","http_max_connections":1024,"http_max_pending_requests":1024}`,
		IsBuiltin:   true,
		IsPublic:    true,
	},
	{
		Name:        "least-request",
		Description: "最少请求负载均衡",
		Category:    "loadbalance",
		RuleType:    "loadbalance",
		Content:     `{"lb_policy":"least_request","choice_count":2,"http_max_connections":1024}`,
		IsBuiltin:   true,
		IsPublic:    true,
	},
	{
		Name:        "consistent-hash-cookie",
		Description: "一致性哈希（基于 Cookie）",
		Category:    "loadbalance",
		RuleType:    "loadbalance",
		Content:     `{"lb_policy":"consistent_hash","hash_key":"cookie","hash_key_name":"JSESSIONID","ring_size":1024}`,
		IsBuiltin:   true,
		IsPublic:    true,
	},
	// 超时重试模板
	{
		Name:        "timeout-30s-retry-3",
		Description: "30秒超时 + 3次重试",
		Category:    "timeout",
		RuleType:    "timeout",
		Content:     `{"timeout":"30s","retries":3,"per_try_timeout":"10s","retry_on":["5xx","reset","connect-failure"]}`,
		IsBuiltin:   true,
		IsPublic:    true,
	},
	{
		Name:        "timeout-5s-no-retry",
		Description: "5秒超时（无重试）",
		Category:    "timeout",
		RuleType:    "timeout",
		Content:     `{"timeout":"5s","retries":0}`,
		IsBuiltin:   true,
		IsPublic:    true,
	},
	// 故障注入模板
	{
		Name:        "delay-injection-5s",
		Description: "延迟注入 5秒（10%流量）",
		Category:    "fault",
		RuleType:    "fault",
		Content:     `{"type":"delay","delay_duration":"5s","percentage":10,"enabled":false}`,
		IsBuiltin:   true,
		IsPublic:    true,
	},
	{
		Name:        "abort-injection-500",
		Description: "中断注入 HTTP 500（5%流量）",
		Category:    "fault",
		RuleType:    "fault",
		Content:     `{"type":"abort","abort_code":500,"abort_message":"Service Unavailable","percentage":5,"enabled":false}`,
		IsBuiltin:   true,
		IsPublic:    true,
	},
}
