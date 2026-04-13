package dto

import "time"

// ==================== 安全概览 ====================

// SecurityOverviewResponse 安全概览响应
type SecurityOverviewResponse struct {
	SecurityScore int                  `json:"security_score"` // 安全评分 0-100
	RiskLevel     string               `json:"risk_level"`     // 风险等级
	VulnSummary   VulnSummary          `json:"vuln_summary"`   // 漏洞统计
	ConfigSummary ConfigCheckSummary   `json:"config_summary"` // 配置检查统计
	RecentScans   []ImageScanItem      `json:"recent_scans"`   // 最近扫描
	RecentChecks  []ConfigCheckItem    `json:"recent_checks"`  // 最近检查
	TrendData     []SecurityTrendPoint `json:"trend_data"`     // 趋势数据
}

type VulnSummary struct {
	Critical int `json:"critical"`
	High     int `json:"high"`
	Medium   int `json:"medium"`
	Low      int `json:"low"`
	Total    int `json:"total"`
}

type ConfigCheckSummary struct {
	Critical int `json:"critical"`
	High     int `json:"high"`
	Medium   int `json:"medium"`
	Low      int `json:"low"`
	Passed   int `json:"passed"`
	Total    int `json:"total"`
}

type SecurityTrendPoint struct {
	Date       string `json:"date"`
	VulnCount  int    `json:"vuln_count"`
	IssueCount int    `json:"issue_count"`
}

// ==================== 镜像扫描 ====================

// ScanImageRequest 扫描镜像请求
type ScanImageRequest struct {
	Image      string `json:"image" binding:"required"`
	RegistryID uint   `json:"registry_id"`
}

// ImageScanItem 镜像扫描项
type ImageScanItem struct {
	ID            uint       `json:"id"`
	Image         string     `json:"image"`
	Status        string     `json:"status"`
	RiskLevel     string     `json:"risk_level"`
	CriticalCount int        `json:"critical_count"`
	HighCount     int        `json:"high_count"`
	MediumCount   int        `json:"medium_count"`
	LowCount      int        `json:"low_count"`
	ScannedAt     *time.Time `json:"scanned_at"`
	CreatedAt     time.Time  `json:"created_at"`
}

// ScanResultResponse 扫描结果响应
type ScanResultResponse struct {
	ID              uint            `json:"id"`
	Image           string          `json:"image"`
	Status          string          `json:"status"`
	RiskLevel       string          `json:"risk_level"`
	VulnSummary     VulnSummary     `json:"vuln_summary"`
	Vulnerabilities []Vulnerability `json:"vulnerabilities"`
	ScannedAt       *time.Time      `json:"scanned_at"`
	ErrorMessage    string          `json:"error_message,omitempty"`
}

// Vulnerability 漏洞信息
type Vulnerability struct {
	VulnID       string   `json:"vuln_id"`
	PkgName      string   `json:"pkg_name"`
	InstalledVer string   `json:"installed_ver"`
	FixedVer     string   `json:"fixed_ver"`
	Severity     string   `json:"severity"`
	Title        string   `json:"title"`
	Description  string   `json:"description"`
	References   []string `json:"references"`
}

// ScanHistoryRequest 扫描历史请求
type ScanHistoryRequest struct {
	Image    string `form:"image"`
	Status   string `form:"status"`
	Page     int    `form:"page"`
	PageSize int    `form:"page_size"`
}

// ScanHistoryResponse 扫描历史响应
type ScanHistoryResponse struct {
	Total int             `json:"total"`
	Items []ImageScanItem `json:"items"`
}

// ==================== 镜像仓库 ====================

// ImageRegistryRequest 镜像仓库请求
type ImageRegistryRequest struct {
	ID        uint   `json:"id"`
	Name      string `json:"name" binding:"required"`
	Type      string `json:"type" binding:"required"`
	URL       string `json:"url" binding:"required"`
	Username  string `json:"username"`
	Password  string `json:"password"`
	IsDefault bool   `json:"is_default"`
}

// ImageRegistryItem 镜像仓库项
type ImageRegistryItem struct {
	ID        uint      `json:"id"`
	Name      string    `json:"name"`
	Type      string    `json:"type"`
	URL       string    `json:"url"`
	Username  string    `json:"username"`
	IsDefault bool      `json:"is_default"`
	CreatedAt time.Time `json:"created_at"`
}

// RegistryImageItem 仓库镜像项
type RegistryImageItem struct {
	Name      string   `json:"name"`
	Tags      []string `json:"tags"`
	Size      int64    `json:"size"`
	UpdatedAt string   `json:"updated_at"`
}

// ==================== 配置检查 ====================

// ConfigCheckRequest 配置检查请求
type ConfigCheckRequest struct {
	ClusterID uint   `json:"cluster_id" binding:"required"`
	Namespace string `json:"namespace"`
	RuleIDs   []uint `json:"rule_ids"`
}

// ConfigCheckItem 配置检查项
type ConfigCheckItem struct {
	ID            uint       `json:"id"`
	ClusterID     uint       `json:"cluster_id"`
	ClusterName   string     `json:"cluster_name"`
	Namespace     string     `json:"namespace"`
	Status        string     `json:"status"`
	CriticalCount int        `json:"critical_count"`
	HighCount     int        `json:"high_count"`
	MediumCount   int        `json:"medium_count"`
	LowCount      int        `json:"low_count"`
	PassedCount   int        `json:"passed_count"`
	CheckedAt     *time.Time `json:"checked_at"`
}

// ConfigCheckHistoryResponse 配置检查历史响应
type ConfigCheckHistoryResponse struct {
	Total int               `json:"total"`
	Items []ConfigCheckItem `json:"items"`
}

// ConfigCheckResultResponse 配置检查结果响应
type ConfigCheckResultResponse struct {
	ID          uint               `json:"id"`
	ClusterID   uint               `json:"cluster_id"`
	ClusterName string             `json:"cluster_name"`
	Namespace   string             `json:"namespace"`
	Status      string             `json:"status"`
	Summary     ConfigCheckSummary `json:"summary"`
	Issues      []ConfigIssue      `json:"issues"`
	CheckedAt   *time.Time         `json:"checked_at"`
}

// ConfigIssue 配置问题
type ConfigIssue struct {
	RuleID       uint   `json:"rule_id"`
	RuleName     string `json:"rule_name"`
	Severity     string `json:"severity"`
	ResourceKind string `json:"resource_kind"`
	ResourceName string `json:"resource_name"`
	Namespace    string `json:"namespace"`
	Message      string `json:"message"`
	Remediation  string `json:"remediation"`
}

// ==================== 合规规则 ====================

// ComplianceRuleRequest 合规规则请求
type ComplianceRuleRequest struct {
	ID            uint   `json:"id"`
	Name          string `json:"name" binding:"required"`
	Description   string `json:"description"`
	Severity      string `json:"severity" binding:"required"`
	Category      string `json:"category" binding:"required"`
	Enabled       bool   `json:"enabled"`
	ConditionJSON string `json:"condition_json" binding:"required"`
	Remediation   string `json:"remediation"`
}

// ComplianceRuleItem 合规规则项
type ComplianceRuleItem struct {
	ID          uint      `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Severity    string    `json:"severity"`
	Category    string    `json:"category"`
	CheckType   string    `json:"check_type"`
	Enabled     bool      `json:"enabled"`
	Remediation string    `json:"remediation"`
	CreatedAt   time.Time `json:"created_at"`
}

// ==================== 审计日志 ====================

// AuditLogRequest 审计日志查询请求
type AuditLogRequest struct {
	UserID       uint   `form:"user_id"`
	Action       string `form:"action"`
	ResourceType string `form:"resource_type"`
	ClusterID    uint   `form:"cluster_id"`
	StartTime    string `form:"start_time"`
	EndTime      string `form:"end_time"`
	Page         int    `form:"page"`
	PageSize     int    `form:"page_size"`
}

// AuditLogItem 审计日志项
type AuditLogItem struct {
	ID           uint      `json:"id"`
	UserID       *uint     `json:"user_id"`
	Username     string    `json:"username"`
	Action       string    `json:"action"`
	ResourceType string    `json:"resource_type"`
	ResourceName string    `json:"resource_name"`
	Namespace    string    `json:"namespace"`
	ClusterID    *uint     `json:"cluster_id"`
	ClusterName  string    `json:"cluster_name"`
	Detail       string    `json:"detail"`
	Result       string    `json:"result"`
	ClientIP     string    `json:"client_ip"`
	CreatedAt    time.Time `json:"created_at"`
}

// AuditLogResponse 审计日志响应
type AuditLogResponse struct {
	Total int            `json:"total"`
	Items []AuditLogItem `json:"items"`
}

// ==================== 安全报告 ====================

// GenerateReportRequest 生成报告请求
type GenerateReportRequest struct {
	Name      string `json:"name" binding:"required"`
	ClusterID uint   `json:"cluster_id"`
	Type      string `json:"type"` // manual, scheduled
}

// SecurityReportItem 安全报告项
type SecurityReportItem struct {
	ID          uint       `json:"id"`
	Name        string     `json:"name"`
	Type        string     `json:"type"`
	ClusterID   *uint      `json:"cluster_id"`
	Status      string     `json:"status"`
	GeneratedAt *time.Time `json:"generated_at"`
	CreatedAt   time.Time  `json:"created_at"`
}
