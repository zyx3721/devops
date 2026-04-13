// Package pipeline 流水线相关模型
// 本文件定义构建优化相关模型
package pipeline

import (
	"time"
)

// BuildCache 构建缓存
type BuildCache struct {
	ID          uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	PipelineID  uint64    `gorm:"index;not null;comment:流水线ID" json:"pipeline_id"`
	CacheKey    string    `gorm:"size:255;not null;uniqueIndex;comment:缓存键" json:"cache_key"`
	CacheType   string    `gorm:"size:50;index;comment:缓存类型" json:"cache_type"` // maven, npm, go, pip, docker_layer
	CachePath   string    `gorm:"size:500;comment:缓存路径" json:"cache_path"`
	SizeBytes   int64     `gorm:"default:0;comment:缓存大小(字节)" json:"size_bytes"`
	HitCount    int       `gorm:"default:0;comment:命中次数" json:"hit_count"`
	LastHitAt   time.Time `gorm:"comment:最后命中时间" json:"last_hit_at"`
	ExpireAt    time.Time `gorm:"index;comment:过期时间" json:"expire_at"`
	StorageType string    `gorm:"size:50;default:local;comment:存储类型" json:"storage_type"` // local, s3, oss
	StorageURL  string    `gorm:"size:500;comment:存储URL" json:"storage_url"`
	Checksum    string    `gorm:"size:64;comment:校验和" json:"checksum"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (BuildCache) TableName() string { return "build_caches" }

// BuildResourceQuota 构建资源配额
type BuildResourceQuota struct {
	ID            uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	Name          string    `gorm:"size:100;not null;uniqueIndex;comment:配额名称" json:"name"`
	Description   string    `gorm:"size:500;comment:描述" json:"description"`
	ProjectID     *uint64   `gorm:"index;comment:项目ID(空表示全局)" json:"project_id"`
	MaxCPU        string    `gorm:"size:20;default:2;comment:最大CPU" json:"max_cpu"`       // 如 "2" 或 "2000m"
	MaxMemory     string    `gorm:"size:20;default:4Gi;comment:最大内存" json:"max_memory"`   // 如 "4Gi"
	MaxStorage    string    `gorm:"size:20;default:10Gi;comment:最大存储" json:"max_storage"` // 如 "10Gi"
	MaxConcurrent int       `gorm:"default:5;comment:最大并发构建数" json:"max_concurrent"`
	MaxDuration   int       `gorm:"default:3600;comment:最大构建时长(秒)" json:"max_duration"`
	Priority      int       `gorm:"default:0;comment:优先级" json:"priority"`
	IsDefault     bool      `gorm:"default:false;comment:是否默认配额" json:"is_default"`
	Enabled       bool      `gorm:"default:true;comment:是否启用" json:"enabled"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func (BuildResourceQuota) TableName() string { return "build_resource_quotas" }

// BuildResourceUsage 构建资源使用记录
type BuildResourceUsage struct {
	ID          uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	PipelineID  uint64    `gorm:"index;not null;comment:流水线ID" json:"pipeline_id"`
	RunID       uint64    `gorm:"index;not null;comment:执行ID" json:"run_id"`
	QuotaID     uint64    `gorm:"index;comment:配额ID" json:"quota_id"`
	CPUUsed     string    `gorm:"size:20;comment:CPU使用量" json:"cpu_used"`
	MemoryUsed  string    `gorm:"size:20;comment:内存使用量" json:"memory_used"`
	StorageUsed string    `gorm:"size:20;comment:存储使用量" json:"storage_used"`
	DurationSec int       `gorm:"comment:构建时长(秒)" json:"duration_sec"`
	CacheHit    bool      `gorm:"default:false;comment:是否命中缓存" json:"cache_hit"`
	CacheSaved  int64     `gorm:"default:0;comment:缓存节省时间(秒)" json:"cache_saved"`
	StartedAt   time.Time `gorm:"comment:开始时间" json:"started_at"`
	CompletedAt time.Time `gorm:"comment:完成时间" json:"completed_at"`
	CreatedAt   time.Time `json:"created_at"`
}

func (BuildResourceUsage) TableName() string { return "build_resource_usages" }

// ParallelBuildConfig 并行构建配置
type ParallelBuildConfig struct {
	ID              uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	PipelineID      uint64    `gorm:"uniqueIndex;not null;comment:流水线ID" json:"pipeline_id"`
	Enabled         bool      `gorm:"default:true;comment:是否启用并行构建" json:"enabled"`
	MaxParallel     int       `gorm:"default:3;comment:最大并行数" json:"max_parallel"`
	FailFast        bool      `gorm:"default:true;comment:快速失败" json:"fail_fast"`
	ParallelStages  string    `gorm:"type:json;comment:可并行的阶段" json:"parallel_stages"` // JSON数组
	DependencyGraph string    `gorm:"type:json;comment:依赖图" json:"dependency_graph"`   // JSON对象
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

func (ParallelBuildConfig) TableName() string { return "parallel_build_configs" }
