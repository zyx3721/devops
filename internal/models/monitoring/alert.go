// Package monitoring 定义监控告警相关的数据模型
// 本文件包含告警相关的模型定义
package monitoring

import (
	"time"
)

// ==================== 告警配置模型 ====================

// AlertConfig 告警配置
// 定义告警规则
type AlertConfig struct {
	ID          uint      `gorm:"primarykey" json:"id"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Name        string    `gorm:"size:100;not null" json:"name"`      // 配置名称
	Type        string    `gorm:"size:50;not null;index" json:"type"` // 类型: jenkins_build/k8s_pod/health_check
	Enabled     bool      `gorm:"default:true" json:"enabled"`        // 是否启用
	Platform    string    `gorm:"size:50;not null" json:"platform"`   // 通知平台: feishu/dingtalk/wechatwork
	BotID       uint      `gorm:"index" json:"bot_id"`                // 机器人ID
	TemplateID  *uint     `gorm:"index" json:"template_id"`           // 关联的告警模板ID
	Channels    string    `gorm:"type:text" json:"channels"`          // 通知渠道配置 JSON (包含 Webhook 等)
	Conditions  string    `gorm:"type:text" json:"conditions"`        // 触发条件 JSON
	Description string    `gorm:"type:text" json:"description"`       // 描述
	CreatedBy   *uint     `gorm:"index" json:"created_by"`
}

// TableName 指定表名
func (AlertConfig) TableName() string {
	return "alert_configs"
}

// AlertHistory 告警历史
// 记录每次告警的详细信息
type AlertHistory struct {
	ID             uint       `gorm:"primarykey" json:"id"`
	CreatedAt      time.Time  `json:"created_at"`
	AlertConfigID  uint       `gorm:"index" json:"alert_config_id"`                // 告警配置ID
	Type           string     `gorm:"size:50;not null;index" json:"type"`          // 告警类型
	Title          string     `gorm:"size:200" json:"title"`                       // 标题
	Content        string     `gorm:"type:text" json:"content"`                    // 内容
	Level          string     `gorm:"size:20;default:'warning'" json:"level"`      // 级别: info/warning/error/critical
	Status         string     `gorm:"size:20;default:'sent'" json:"status"`        // 发送状态: sent/failed
	AckStatus      string     `gorm:"size:20;default:'pending'" json:"ack_status"` // 确认状态: pending/acked/resolved
	AckBy          *uint      `json:"ack_by"`                                      // 确认人ID
	AckAt          *time.Time `json:"ack_at"`                                      // 确认时间
	ResolvedBy     *uint      `json:"resolved_by"`                                 // 解决人ID
	ResolvedAt     *time.Time `json:"resolved_at"`                                 // 解决时间
	ResolveComment string     `gorm:"type:text" json:"resolve_comment"`            // 解决备注
	Silenced       bool       `gorm:"default:false" json:"silenced"`               // 是否被静默
	SilenceID      *uint      `json:"silence_id"`                                  // 静默规则ID
	Escalated      bool       `gorm:"default:false" json:"escalated"`              // 是否已升级
	EscalationID   *uint      `json:"escalation_id"`                               // 升级规则ID
	ErrorMsg       string     `gorm:"type:text" json:"error_msg"`                  // 错误信息
	SourceID       string     `gorm:"size:100" json:"source_id"`                   // 来源ID
	SourceURL      string     `gorm:"size:500" json:"source_url"`                  // 来源URL
}

// TableName 指定表名
func (AlertHistory) TableName() string {
	return "alert_histories"
}

// AlertSilence 告警静默规则
// 定义告警静默条件
type AlertSilence struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Name      string    `gorm:"size:100;not null" json:"name"`                   // 规则名称
	Type      string    `gorm:"size:50;not null;index" json:"type"`              // 类型: jenkins_build/k8s_pod/health_check/all
	Matchers  string    `gorm:"type:text" json:"matchers"`                       // 匹配条件 JSON
	StartTime time.Time `gorm:"not null" json:"start_time"`                      // 开始时间
	EndTime   time.Time `gorm:"not null" json:"end_time"`                        // 结束时间
	Reason    string    `gorm:"size:500" json:"reason"`                          // 静默原因
	Status    string    `gorm:"size:20;default:'active';not null" json:"status"` // 状态: active/expired/cancelled
	CreatedBy *uint     `gorm:"index" json:"created_by"`
}

// TableName 指定表名
func (AlertSilence) TableName() string {
	return "alert_silences"
}

// AlertEscalation 告警升级规则
// 定义告警升级策略
type AlertEscalation struct {
	ID            uint      `gorm:"primarykey" json:"id"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
	Name          string    `gorm:"size:100;not null" json:"name"`            // 规则名称
	AlertConfigID *uint     `gorm:"index" json:"alert_config_id"`             // 告警配置ID，NULL表示全局
	Level         string    `gorm:"size:20;not null" json:"level"`            // 级别: warning/error/critical
	DelayMinutes  int       `gorm:"default:30;not null" json:"delay_minutes"` // 延迟时间(分钟)
	Platform      string    `gorm:"size:50;not null" json:"platform"`         // 通知平台
	BotID         *uint     `gorm:"index" json:"bot_id"`                      // 机器人ID
	NotifyUsers   string    `gorm:"type:text" json:"notify_users"`            // 通知用户 JSON
	Enabled       bool      `gorm:"default:true" json:"enabled"`              // 是否启用
	Description   string    `gorm:"type:text" json:"description"`             // 描述
	CreatedBy     *uint     `gorm:"index" json:"created_by"`
}

// TableName 指定表名
func (AlertEscalation) TableName() string {
	return "alert_escalations"
}

// AlertEscalationLog 告警升级记录
// 记录告警升级的执行情况
type AlertEscalationLog struct {
	ID             uint      `gorm:"primarykey" json:"id"`
	CreatedAt      time.Time `json:"created_at"`
	AlertHistoryID uint      `gorm:"not null;index" json:"alert_history_id"` // 告警历史ID
	EscalationID   uint      `gorm:"not null;index" json:"escalation_id"`    // 升级规则ID
	Platform       string    `gorm:"size:50;not null" json:"platform"`       // 通知平台
	BotID          *uint     `json:"bot_id"`                                 // 机器人ID
	Status         string    `gorm:"size:20;default:'sent'" json:"status"`   // 状态: sent/failed
	ErrorMsg       string    `gorm:"type:text" json:"error_msg"`             // 错误信息
}

// TableName 指定表名
func (AlertEscalationLog) TableName() string {
	return "alert_escalation_logs"
}
