// Package system 定义系统管理相关的数据模型
// 本文件包含安全相关的模型定义
package system

import (
	"time"
)

// ==================== 安全模型 ====================

// ImageRegistry 镜像仓库配置
type ImageRegistry struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Name      string    `gorm:"size:100;not null" json:"name"`
	Type      string    `gorm:"size:50;not null" json:"type"` // harbor, dockerhub, acr, ecr
	URL       string    `gorm:"size:500;not null" json:"url"`
	Username  string    `gorm:"size:100" json:"username"`
	Password  string    `gorm:"size:500" json:"-"` // 加密存储，不返回
	IsDefault bool      `gorm:"default:false" json:"is_default"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TableName 指定表名
func (ImageRegistry) TableName() string {
	return "image_registries"
}

// ImageScan 镜像扫描记录
type ImageScan struct {
	ID            uint       `gorm:"primaryKey" json:"id"`
	Image         string     `gorm:"size:500;not null;index:idx_image" json:"image"`
	RegistryID    *uint      `json:"registry_id"`
	Status        string     `gorm:"size:20;not null;index:idx_status" json:"status"` // scanning, completed, failed
	RiskLevel     string     `gorm:"size:20" json:"risk_level"`                       // critical, high, medium, low, none
	CriticalCount int        `gorm:"default:0" json:"critical_count"`
	HighCount     int        `gorm:"default:0" json:"high_count"`
	MediumCount   int        `gorm:"default:0" json:"medium_count"`
	LowCount      int        `gorm:"default:0" json:"low_count"`
	ResultJSON    string     `gorm:"type:longtext" json:"-"`
	ErrorMessage  string     `gorm:"type:text" json:"error_message"`
	ScannedAt     *time.Time `gorm:"index:idx_scanned_at" json:"scanned_at"`
	CreatedAt     time.Time  `json:"created_at"`
}

// TableName 指定表名
func (ImageScan) TableName() string {
	return "image_scans"
}

// ComplianceRule 合规规则
type ComplianceRule struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	Name          string    `gorm:"size:100;not null" json:"name"`
	Description   string    `gorm:"type:text" json:"description"`
	Severity      string    `gorm:"size:20;not null" json:"severity"` // critical, high, medium, low
	Category      string    `gorm:"size:50;not null;index:idx_category" json:"category"`
	CheckType     string    `gorm:"size:20;not null" json:"check_type"` // builtin, custom
	Enabled       bool      `gorm:"default:true;index:idx_enabled" json:"enabled"`
	ConditionJSON string    `gorm:"column:condition_json;type:text;not null" json:"condition_json"`
	Remediation   string    `gorm:"type:text" json:"remediation"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// TableName 指定表名
func (ComplianceRule) TableName() string {
	return "compliance_rules"
}

// ConfigCheck 配置检查记录
type ConfigCheck struct {
	ID            uint       `gorm:"primaryKey" json:"id"`
	ClusterID     uint       `gorm:"not null;index:idx_cluster" json:"cluster_id"`
	Namespace     string     `gorm:"size:100" json:"namespace"`
	Status        string     `gorm:"size:20;not null" json:"status"` // running, completed, failed
	CriticalCount int        `gorm:"default:0" json:"critical_count"`
	HighCount     int        `gorm:"default:0" json:"high_count"`
	MediumCount   int        `gorm:"default:0" json:"medium_count"`
	LowCount      int        `gorm:"default:0" json:"low_count"`
	PassedCount   int        `gorm:"default:0" json:"passed_count"`
	ResultJSON    string     `gorm:"type:longtext" json:"-"`
	CheckedAt     *time.Time `gorm:"index:idx_checked_at" json:"checked_at"`
	CreatedAt     time.Time  `json:"created_at"`
}

// TableName 指定表名
func (ConfigCheck) TableName() string {
	return "config_checks"
}

// SecurityAuditLog 安全审计日志
type SecurityAuditLog struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	UserID       *uint     `gorm:"index:idx_user" json:"user_id"`
	Username     string    `gorm:"size:100" json:"username"`
	Action       string    `gorm:"size:50;not null;index:idx_action" json:"action"`
	ResourceType string    `gorm:"size:50;index:idx_resource" json:"resource_type"`
	ResourceName string    `gorm:"size:200" json:"resource_name"`
	Namespace    string    `gorm:"size:100" json:"namespace"`
	ClusterID    *uint     `gorm:"index:idx_cluster" json:"cluster_id"`
	ClusterName  string    `gorm:"size:100" json:"cluster_name"`
	Detail       string    `gorm:"type:text" json:"detail"`
	Result       string    `gorm:"size:20" json:"result"` // success, failed
	ClientIP     string    `gorm:"size:50" json:"client_ip"`
	CreatedAt    time.Time `gorm:"index:idx_created_at" json:"created_at"`
}

// TableName 指定表名
func (SecurityAuditLog) TableName() string {
	return "security_audit_logs"
}

// SecurityReport 安全报告
type SecurityReport struct {
	ID          uint       `gorm:"primaryKey" json:"id"`
	Name        string     `gorm:"size:200;not null" json:"name"`
	Type        string     `gorm:"size:50;not null" json:"type"` // manual, scheduled
	ClusterID   *uint      `gorm:"index:idx_cluster" json:"cluster_id"`
	Status      string     `gorm:"size:20;not null" json:"status"` // generating, completed, failed
	FilePath    string     `gorm:"size:500" json:"file_path"`
	SummaryJSON string     `gorm:"type:text" json:"-"`
	GeneratedAt *time.Time `gorm:"index:idx_generated_at" json:"generated_at"`
	CreatedAt   time.Time  `json:"created_at"`
}

// TableName 指定表名
func (SecurityReport) TableName() string {
	return "security_reports"
}
