// Package artifact 制品管理相关模型
package artifact

import (
	"time"
)

// Repository 制品仓库
type Repository struct {
	ID               uint64     `gorm:"primaryKey;autoIncrement" json:"id"`
	Name             string     `gorm:"size:100;not null;uniqueIndex;comment:仓库名称" json:"name"`
	Description      string     `gorm:"size:500;comment:描述" json:"description"`
	Type             string     `gorm:"size:50;not null;index;comment:仓库类型" json:"type"` // docker, maven, npm, pypi, generic
	URL              string     `gorm:"size:500;not null;comment:仓库地址" json:"url"`
	Username         string     `gorm:"size:100;comment:用户名" json:"username"`
	Password         string     `gorm:"size:500;comment:密码(加密)" json:"-"`
	IsDefault        bool       `gorm:"default:false;comment:是否默认仓库" json:"is_default"`
	IsPublic         bool       `gorm:"default:false;comment:是否公开" json:"is_public"`
	Enabled          bool       `gorm:"default:true;comment:是否启用" json:"enabled"`
	ConnectionStatus string     `gorm:"size:20;default:unknown;index;comment:连接状态" json:"connection_status"` // connected, disconnected, checking, unknown
	LastCheckAt      *time.Time `gorm:"index;comment:最后检查时间" json:"last_check_at"`
	LastError        string     `gorm:"type:text;comment:最后错误信息" json:"last_error,omitempty"`
	EnableMonitoring bool       `gorm:"default:true;index;comment:是否启用监控" json:"enable_monitoring"`
	CheckInterval    int        `gorm:"default:300;comment:检查间隔(秒)" json:"check_interval"`
	CreatedBy        string     `gorm:"size:100;comment:创建人" json:"created_by"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

func (Repository) TableName() string { return "artifact_repositories" }

// Artifact 制品
type Artifact struct {
	ID           uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	RepositoryID uint64    `gorm:"index;not null;comment:仓库ID" json:"repository_id"`
	Name         string    `gorm:"size:200;not null;index;comment:制品名称" json:"name"`
	GroupID      string    `gorm:"size:200;index;comment:组ID(Maven)" json:"group_id"`
	ArtifactID   string    `gorm:"size:200;index;comment:制品ID(Maven)" json:"artifact_id"`
	Type         string    `gorm:"size:50;index;comment:制品类型" json:"type"` // jar, war, docker, npm, wheel
	Description  string    `gorm:"size:500;comment:描述" json:"description"`
	LatestVer    string    `gorm:"size:100;comment:最新版本" json:"latest_version"`
	DownloadCnt  int64     `gorm:"default:0;comment:下载次数" json:"download_count"`
	Tags         string    `gorm:"size:500;comment:标签(逗号分隔)" json:"tags"`
	CreatedBy    string    `gorm:"size:100;comment:创建人" json:"created_by"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func (Artifact) TableName() string { return "artifacts" }

// ArtifactVersion 制品版本
type ArtifactVersion struct {
	ID          uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	ArtifactID  uint64    `gorm:"index;not null;comment:制品ID" json:"artifact_id"`
	Version     string    `gorm:"size:100;not null;index;comment:版本号" json:"version"`
	SizeBytes   int64     `gorm:"default:0;comment:大小(字节)" json:"size_bytes"`
	Checksum    string    `gorm:"size:64;comment:SHA256校验和" json:"checksum"`
	DownloadURL string    `gorm:"size:500;comment:下载地址" json:"download_url"`
	Metadata    string    `gorm:"type:json;comment:元数据" json:"metadata"`
	PipelineID  *uint64   `gorm:"index;comment:来源流水线ID" json:"pipeline_id"`
	RunID       *uint64   `gorm:"index;comment:来源执行ID" json:"run_id"`
	GitCommit   string    `gorm:"size:64;comment:Git提交" json:"git_commit"`
	GitBranch   string    `gorm:"size:100;comment:Git分支" json:"git_branch"`
	BuildNumber int       `gorm:"comment:构建号" json:"build_number"`
	DownloadCnt int64     `gorm:"default:0;comment:下载次数" json:"download_count"`
	ScanStatus  string    `gorm:"size:20;default:pending;comment:扫描状态" json:"scan_status"` // pending, scanning, passed, failed
	ScanResult  string    `gorm:"type:json;comment:扫描结果" json:"scan_result"`
	IsRelease   bool      `gorm:"default:false;comment:是否正式版本" json:"is_release"`
	ReleasedAt  time.Time `gorm:"comment:发布时间" json:"released_at"`
	ReleasedBy  string    `gorm:"size:100;comment:发布人" json:"released_by"`
	CreatedAt   time.Time `json:"created_at"`
}

func (ArtifactVersion) TableName() string { return "artifact_versions" }

// ArtifactScanResult 制品扫描结果
type ArtifactScanResult struct {
	ID            uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	VersionID     uint64    `gorm:"index;not null;comment:版本ID" json:"version_id"`
	ScanType      string    `gorm:"size:50;not null;comment:扫描类型" json:"scan_type"` // vulnerability, license, quality
	Scanner       string    `gorm:"size:50;comment:扫描器" json:"scanner"`             // trivy, sonarqube, etc.
	Status        string    `gorm:"size:20;comment:状态" json:"status"`               // passed, failed, warning
	CriticalCount int       `gorm:"default:0;comment:严重漏洞数" json:"critical_count"`
	HighCount     int       `gorm:"default:0;comment:高危漏洞数" json:"high_count"`
	MediumCount   int       `gorm:"default:0;comment:中危漏洞数" json:"medium_count"`
	LowCount      int       `gorm:"default:0;comment:低危漏洞数" json:"low_count"`
	Details       string    `gorm:"type:json;comment:详细结果" json:"details"`
	ReportURL     string    `gorm:"size:500;comment:报告URL" json:"report_url"`
	ScannedAt     time.Time `gorm:"comment:扫描时间" json:"scanned_at"`
	CreatedAt     time.Time `json:"created_at"`
}

func (ArtifactScanResult) TableName() string { return "artifact_scan_results" }

// ArtifactPromotion 制品晋级记录
type ArtifactPromotion struct {
	ID         uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	VersionID  uint64    `gorm:"index;not null;comment:版本ID" json:"version_id"`
	FromRepoID uint64    `gorm:"comment:源仓库ID" json:"from_repo_id"`
	ToRepoID   uint64    `gorm:"comment:目标仓库ID" json:"to_repo_id"`
	FromEnv    string    `gorm:"size:50;comment:源环境" json:"from_env"` // dev, test, staging
	ToEnv      string    `gorm:"size:50;comment:目标环境" json:"to_env"`  // test, staging, prod
	Status     string    `gorm:"size:20;comment:状态" json:"status"`    // pending, approved, rejected, completed
	ApprovalID *uint64   `gorm:"comment:审批ID" json:"approval_id"`
	PromotedBy string    `gorm:"size:100;comment:晋级人" json:"promoted_by"`
	PromotedAt time.Time `gorm:"comment:晋级时间" json:"promoted_at"`
	Comment    string    `gorm:"size:500;comment:备注" json:"comment"`
	CreatedAt  time.Time `json:"created_at"`
}

func (ArtifactPromotion) TableName() string { return "artifact_promotions" }
