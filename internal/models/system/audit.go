// Package system 定义系统管理相关的数据模型
// 本文件包含审计日志相关的模型定义
package system

import (
	"encoding/json"
	"time"
)

// ==================== 审计日志模型 ====================

// AuditLog 操作审计日志
// 记录用户的所有操作
// 统一的审计日志模型，支持多租户、变更追踪、分布式追踪等高级特性
type AuditLog struct {
	ID           uint            `gorm:"primarykey" json:"id"`
	TenantID     *uint           `gorm:"index" json:"tenant_id"`                      // 租户ID（支持多租户）
	UserID       *uint           `gorm:"index" json:"user_id"`                        // 用户ID（可为空，如系统操作）
	Username     string          `gorm:"size:100;index" json:"username"`              // 用户名
	Action       string          `gorm:"size:50;not null;index" json:"action"`        // 操作: create/update/delete/login/logout/trigger/test
	ResourceType string          `gorm:"size:50;not null;index" json:"resource_type"` // 资源类型: jenkins/k8s/feishu/dingtalk/wechatwork/oa/user
	ResourceID   *uint           `gorm:"index" json:"resource_id"`                    // 资源ID（可为空）
	ResourceName string          `gorm:"size:200" json:"resource_name"`               // 资源名称
	OldValue     json.RawMessage `gorm:"type:json" json:"old_value"`                  // 变更前的值（JSON格式，用于审计追踪）
	NewValue     json.RawMessage `gorm:"type:json" json:"new_value"`                  // 变更后的值（JSON格式，用于审计追踪）
	IPAddress    string          `gorm:"size:45" json:"ip_address"`                   // IP地址（支持IPv6）
	UserAgent    string          `gorm:"size:500" json:"user_agent"`                  // User Agent
	RequestID    string          `gorm:"size:50;index" json:"request_id"`             // 请求ID（用于关联同一请求的多条日志）
	TraceID      string          `gorm:"size:50" json:"trace_id"`                     // 追踪ID（用于分布式追踪）
	Status       string          `gorm:"size:20;default:'success'" json:"status"`     // 状态: success/failed
	ErrorMessage string          `gorm:"type:text" json:"error_message"`              // 错误信息
	Duration     int64           `gorm:"comment:操作耗时(ms)" json:"duration"`            // 操作耗时（毫秒）
	CreatedAt    time.Time       `gorm:"index" json:"created_at"`                     // 创建时间
}

// TableName 指定表名
func (AuditLog) TableName() string {
	return "audit_logs"
}
