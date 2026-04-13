// Package monitoring 定义监控告警相关的数据模型
// 本文件包含健康检查相关的模型定义
package monitoring

import (
	"time"
)

// ==================== 健康检查模型 ====================

// HealthCheckConfig 健康检查配置
// 定义健康检查规则
type HealthCheckConfig struct {
	ID            uint       `gorm:"primarykey" json:"id"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
	Name          string     `gorm:"size:100;not null" json:"name"`      // 配置名称
	Type          string     `gorm:"size:50;not null;index" json:"type"` // 类型: jenkins/k8s/oa/custom/ssl_cert
	TargetID      uint       `gorm:"index" json:"target_id"`             // 目标资源ID
	TargetName    string     `gorm:"size:200" json:"target_name"`        // 目标名称
	URL           string     `gorm:"size:500" json:"url"`                // 自定义检查URL
	Interval      int        `gorm:"default:300" json:"interval"`        // 检查间隔(秒)
	Timeout       int        `gorm:"default:10" json:"timeout"`          // 超时时间(秒)
	RetryCount    int        `gorm:"default:3" json:"retry_count"`       // 重试次数
	Enabled       bool       `gorm:"default:true" json:"enabled"`        // 是否启用
	AlertEnabled  bool       `gorm:"default:true" json:"alert_enabled"`  // 是否启用告警
	AlertPlatform string     `gorm:"size:50" json:"alert_platform"`      // 告警平台
	AlertBotID    *uint      `gorm:"index" json:"alert_bot_id"`          // 告警机器人ID
	LastCheckAt   *time.Time `json:"last_check_at"`                      // 最后检查时间
	LastStatus    string     `gorm:"size:20" json:"last_status"`         // 最后状态: healthy/unhealthy/unknown
	LastError     string     `gorm:"type:text" json:"last_error"`        // 最后错误信息
	CreatedBy     *uint      `gorm:"index" json:"created_by"`

	// SSL证书相关字段
	CertExpiryDate    *time.Time `json:"cert_expiry_date"`                   // 证书过期时间
	CertDaysRemaining *int       `json:"cert_days_remaining"`                // 证书剩余天数
	CertIssuer        string     `gorm:"size:500" json:"cert_issuer"`        // 证书颁发者
	CertSubject       string     `gorm:"size:500" json:"cert_subject"`       // 证书主题
	CertSerialNumber  string     `gorm:"size:100" json:"cert_serial_number"` // 证书序列号

	// 告警阈值配置
	CriticalDays int `gorm:"default:7" json:"critical_days"` // 严重告警阈值（天）
	WarningDays  int `gorm:"default:30" json:"warning_days"` // 警告告警阈值（天）
	NoticeDays   int `gorm:"default:60" json:"notice_days"`  // 提醒告警阈值（天）

	// 告警状态
	LastAlertLevel string     `gorm:"size:20" json:"last_alert_level"` // 最后告警级别
	LastAlertAt    *time.Time `json:"last_alert_at"`                   // 最后告警时间
}

// TableName 指定表名
func (HealthCheckConfig) TableName() string {
	return "health_check_configs"
}

// HealthCheckHistory 健康检查历史
// 记录每次健康检查的结果
type HealthCheckHistory struct {
	ID             uint      `gorm:"primarykey" json:"id"`
	CreatedAt      time.Time `json:"created_at"`
	ConfigID       uint      `gorm:"not null;index" json:"config_id"`   // 配置ID
	ConfigName     string    `gorm:"size:100" json:"config_name"`       // 配置名称
	Type           string    `gorm:"size:50;index" json:"type"`         // 检查类型
	TargetName     string    `gorm:"size:200" json:"target_name"`       // 目标名称
	Status         string    `gorm:"size:20;not null" json:"status"`    // 状态: healthy/unhealthy
	ResponseTimeMs int64     `gorm:"default:0" json:"response_time_ms"` // 响应时间(毫秒)
	ErrorMsg       string    `gorm:"type:text" json:"error_msg"`        // 错误信息
	AlertSent      bool      `gorm:"default:false" json:"alert_sent"`   // 是否已发送告警

	// SSL证书检查结果
	CertDaysRemaining *int       `json:"cert_days_remaining"`        // 检查时的证书剩余天数
	CertExpiryDate    *time.Time `json:"cert_expiry_date"`           // 检查时的证书过期时间
	AlertLevel        string     `gorm:"size:20" json:"alert_level"` // 告警级别
}

// TableName 指定表名
func (HealthCheckHistory) TableName() string {
	return "health_check_histories"
}
