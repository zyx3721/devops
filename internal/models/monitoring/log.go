// Package monitoring 定义监控告警相关的数据模型
// 本文件包含日志相关的模型定义
package monitoring

import (
	"database/sql/driver"
	"encoding/json"
	"time"
)

// ==================== 日志模型 ====================

// LogAlertRule 日志告警规则
// 定义基于日志内容的告警规则
type LogAlertRule struct {
	ID           int64     `json:"id" gorm:"primaryKey;autoIncrement"`
	Name         string    `json:"name" gorm:"size:100;not null" binding:"required,max=100"`                                  // 规则名称
	ClusterID    int64     `json:"cluster_id" gorm:"not null;index" binding:"required"`                                       // 集群ID
	Namespace    string    `json:"namespace" gorm:"size:100"`                                                                 // 命名空间
	MatchType    string    `json:"match_type" gorm:"size:20;not null" binding:"required,oneof=keyword regex level"`           // 匹配类型
	MatchValue   string    `json:"match_value" gorm:"size:500;not null" binding:"required,max=500"`                           // 匹配值
	Level        string    `json:"level" gorm:"size:20;not null;default:warning" binding:"oneof=info warning error critical"` // 级别
	Channels     JSONArray `json:"channels" gorm:"type:json"`                                                                 // 通知渠道
	Enabled      bool      `json:"enabled" gorm:"default:true;index"`                                                         // 是否启用
	AggregateMin int       `json:"aggregate_min" gorm:"default:5"`                                                            // 聚合时间(分钟)
	CreatedBy    int64     `json:"created_by" gorm:"index"`
	CreatedAt    time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt    time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

// TableName 指定表名
func (LogAlertRule) TableName() string {
	return "log_alert_rules"
}

// LogAlertHistory 日志告警历史
// 记录日志告警的触发情况
type LogAlertHistory struct {
	ID             int64      `json:"id" gorm:"primaryKey;autoIncrement"`
	RuleID         int64      `json:"rule_id" gorm:"not null;index"`               // 规则ID
	RuleName       string     `json:"rule_name" gorm:"size:100"`                   // 规则名称
	ClusterID      int64      `json:"cluster_id" gorm:"not null;index"`            // 集群ID
	Namespace      string     `json:"namespace" gorm:"size:100"`                   // 命名空间
	PodName        string     `json:"pod_name" gorm:"size:200"`                    // Pod 名称
	Container      string     `json:"container" gorm:"size:100"`                   // 容器名称
	MatchedContent string     `json:"matched_content" gorm:"type:text"`            // 匹配内容
	AlertCount     int        `json:"alert_count" gorm:"default:1"`                // 告警次数
	Status         string     `json:"status" gorm:"size:20;default:pending;index"` // 状态
	Silenced       bool       `json:"silenced" gorm:"default:false;index"`         // 是否被静默
	SilenceID      *uint      `json:"silence_id" gorm:"index"`                     // 静默规则ID
	SentAt         *time.Time `json:"sent_at"`                                     // 发送时间
	ErrorMsg       string     `json:"error_msg" gorm:"size:500"`                   // 错误信息
	CreatedAt      time.Time  `json:"created_at" gorm:"autoCreateTime;index"`
}

// TableName 指定表名
func (LogAlertHistory) TableName() string {
	return "log_alert_history"
}

// LogHighlightRule 日志染色规则
// 定义日志显示的高亮规则
type LogHighlightRule struct {
	ID         int64     `json:"id" gorm:"primaryKey;autoIncrement"`
	UserID     int64     `json:"user_id" gorm:"not null;index"`                                                   // 用户ID
	Name       string    `json:"name" gorm:"size:100;not null" binding:"required,max=100"`                        // 规则名称
	MatchType  string    `json:"match_type" gorm:"size:20;not null" binding:"required,oneof=keyword regex level"` // 匹配类型
	MatchValue string    `json:"match_value" gorm:"size:500;not null" binding:"required,max=500"`                 // 匹配值
	FgColor    string    `json:"fg_color" gorm:"size:20"`                                                         // 前景色
	BgColor    string    `json:"bg_color" gorm:"size:20"`                                                         // 背景色
	Priority   int       `json:"priority" gorm:"default:0;index"`                                                 // 优先级
	Enabled    bool      `json:"enabled" gorm:"default:true;index"`                                               // 是否启用
	IsPreset   bool      `json:"is_preset" gorm:"default:false"`                                                  // 是否预设
	CreatedAt  time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt  time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

// TableName 指定表名
func (LogHighlightRule) TableName() string {
	return "log_highlight_rules"
}

// ParseField 解析字段配置
type ParseField struct {
	Name     string `json:"name"`                // 字段名
	Type     string `json:"type"`                // 类型: string/int/float/timestamp
	JSONPath string `json:"json_path,omitempty"` // JSON 路径
}

// LogParseTemplate 日志解析模板
// 定义日志解析规则
type LogParseTemplate struct {
	ID          int64        `json:"id" gorm:"primaryKey;autoIncrement"`
	Name        string       `json:"name" gorm:"size:100;not null" binding:"required,max=100"`                    // 模板名称
	Description string       `json:"description" gorm:"size:500"`                                                 // 描述
	Type        string       `json:"type" gorm:"size:20;not null;index" binding:"required,oneof=json regex grok"` // 类型
	Pattern     string       `json:"pattern" gorm:"type:text"`                                                    // 解析模式
	Fields      []ParseField `json:"fields" gorm:"type:json;serializer:json"`                                     // 字段配置
	IsPreset    bool         `json:"is_preset" gorm:"default:false;index"`                                        // 是否预设
	Enabled     bool         `json:"enabled" gorm:"default:true;index"`                                           // 是否启用
	CreatedBy   int64        `json:"created_by"`
	CreatedAt   time.Time    `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt   time.Time    `json:"updated_at" gorm:"autoUpdateTime"`
}

// TableName 指定表名
func (LogParseTemplate) TableName() string {
	return "log_parse_templates"
}

// LogDataSource 日志数据源配置
// 定义日志数据源连接
type LogDataSource struct {
	ID              int64      `json:"id" gorm:"primaryKey;autoIncrement"`
	Name            string     `json:"name" gorm:"size:100;not null" binding:"required,max=100"`                           // 名称
	Type            string     `json:"type" gorm:"size:20;not null;index" binding:"required,oneof=k8s loki elasticsearch"` // 类型
	URL             string     `json:"url" gorm:"size:500"`                                                                // URL
	Username        string     `json:"username" gorm:"size:100"`                                                           // 用户名
	Password        string     `json:"-" gorm:"size:200"`                                                                  // 密码
	Config          JSONObject `json:"config" gorm:"type:json"`                                                            // 配置
	ClusterID       int64      `json:"cluster_id" gorm:"index"`                                                            // 集群ID
	Namespace       string     `json:"namespace" gorm:"size:100"`                                                          // 命名空间
	IsDefault       bool       `json:"is_default" gorm:"default:false"`                                                    // 是否默认
	Enabled         bool       `json:"enabled" gorm:"default:true;index"`                                                  // 是否启用
	LastCheckAt     *time.Time `json:"last_check_at"`                                                                      // 最后检查时间
	LastCheckStatus string     `json:"last_check_status" gorm:"size:20"`                                                   // 最后检查状态
	CreatedAt       time.Time  `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt       time.Time  `json:"updated_at" gorm:"autoUpdateTime"`
}

// TableName 指定表名
func (LogDataSource) TableName() string {
	return "log_datasources"
}

// LogBookmark 日志收藏书签
// 用户收藏的日志条目
type LogBookmark struct {
	ID             int64      `json:"id" gorm:"primaryKey;autoIncrement"`
	UserID         int64      `json:"user_id" gorm:"not null;index"`              // 用户ID
	ClusterID      int64      `json:"cluster_id" gorm:"not null;index"`           // 集群ID
	Namespace      string     `json:"namespace" gorm:"size:100"`                  // 命名空间
	PodName        string     `json:"pod_name" gorm:"size:200"`                   // Pod 名称
	Container      string     `json:"container" gorm:"size:100"`                  // 容器名称
	LogTimestamp   *time.Time `json:"log_timestamp"`                              // 日志时间戳
	Content        string     `json:"content" gorm:"type:text"`                   // 日志内容
	Note           string     `json:"note" gorm:"type:text"`                      // 备注
	ShareToken     string     `json:"share_token,omitempty" gorm:"size:64;index"` // 分享令牌
	ShareExpiresAt *time.Time `json:"share_expires_at,omitempty"`                 // 分享过期时间
	CreatedAt      time.Time  `json:"created_at" gorm:"autoCreateTime;index"`
	UpdatedAt      time.Time  `json:"updated_at" gorm:"autoUpdateTime"`
}

// TableName 指定表名
func (LogBookmark) TableName() string {
	return "log_bookmarks"
}

// LogSavedQuery 快捷查询
// 用户保存的日志查询条件
type LogSavedQuery struct {
	ID          int64      `json:"id" gorm:"primaryKey;autoIncrement"`
	UserID      int64      `json:"user_id" gorm:"not null;index"`                            // 用户ID
	Name        string     `json:"name" gorm:"size:100;not null" binding:"required,max=100"` // 名称
	Description string     `json:"description" gorm:"size:500"`                              // 描述
	QueryParams JSONObject `json:"query_params" gorm:"type:json;not null"`                   // 查询参数
	IsShared    bool       `json:"is_shared" gorm:"default:false;index"`                     // 是否共享
	UseCount    int        `json:"use_count" gorm:"default:0"`                               // 使用次数
	LastUsedAt  *time.Time `json:"last_used_at"`                                             // 最后使用时间
	CreatedAt   time.Time  `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt   time.Time  `json:"updated_at" gorm:"autoUpdateTime"`
}

// TableName 指定表名
func (LogSavedQuery) TableName() string {
	return "log_saved_queries"
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

// JSONObject 用于存储 JSON 对象
type JSONObject map[string]any
