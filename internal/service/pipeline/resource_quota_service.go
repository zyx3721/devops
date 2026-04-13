// Package pipeline 流水线服务
// 本文件实现构建资源配额管理服务
package pipeline

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	"devops/internal/models/pipeline"
	"devops/pkg/logger"
)

// ResourceQuotaService 资源配额服务
type ResourceQuotaService struct {
	db *gorm.DB
}

// NewResourceQuotaService 创建资源配额服务
func NewResourceQuotaService(db *gorm.DB) *ResourceQuotaService {
	return &ResourceQuotaService{db: db}
}

// CreateQuota 创建配额
func (s *ResourceQuotaService) CreateQuota(ctx context.Context, quota *pipeline.BuildResourceQuota) error {
	// 如果设为默认，取消其他默认配额
	if quota.IsDefault {
		s.db.Model(&pipeline.BuildResourceQuota{}).
			Where("is_default = ?", true).
			Update("is_default", false)
	}
	return s.db.Create(quota).Error
}

// UpdateQuota 更新配额
func (s *ResourceQuotaService) UpdateQuota(ctx context.Context, quota *pipeline.BuildResourceQuota) error {
	if quota.IsDefault {
		s.db.Model(&pipeline.BuildResourceQuota{}).
			Where("is_default = ? AND id != ?", true, quota.ID).
			Update("is_default", false)
	}
	return s.db.Save(quota).Error
}

// DeleteQuota 删除配额
func (s *ResourceQuotaService) DeleteQuota(ctx context.Context, quotaID uint64) error {
	return s.db.Delete(&pipeline.BuildResourceQuota{}, quotaID).Error
}

// GetQuota 获取配额
func (s *ResourceQuotaService) GetQuota(ctx context.Context, quotaID uint64) (*pipeline.BuildResourceQuota, error) {
	var quota pipeline.BuildResourceQuota
	err := s.db.First(&quota, quotaID).Error
	return &quota, err
}

// ListQuotas 获取配额列表
func (s *ResourceQuotaService) ListQuotas(ctx context.Context, projectID *uint64) ([]pipeline.BuildResourceQuota, error) {
	var quotas []pipeline.BuildResourceQuota
	query := s.db.Model(&pipeline.BuildResourceQuota{})

	if projectID != nil {
		query = query.Where("project_id = ? OR project_id IS NULL", *projectID)
	}

	err := query.Order("priority DESC, created_at DESC").Find(&quotas).Error
	return quotas, err
}

// GetEffectiveQuota 获取生效的配额
func (s *ResourceQuotaService) GetEffectiveQuota(ctx context.Context, projectID *uint64) (*pipeline.BuildResourceQuota, error) {
	var quota pipeline.BuildResourceQuota

	// 优先查找项目级配额
	if projectID != nil {
		err := s.db.Where("project_id = ? AND enabled = ?", *projectID, true).
			Order("priority DESC").
			First(&quota).Error
		if err == nil {
			return &quota, nil
		}
	}

	// 查找默认配额
	err := s.db.Where("is_default = ? AND enabled = ?", true, true).First(&quota).Error
	if err == nil {
		return &quota, nil
	}

	// 返回系统默认值
	return &pipeline.BuildResourceQuota{
		Name:          "system_default",
		MaxCPU:        "2",
		MaxMemory:     "4Gi",
		MaxStorage:    "10Gi",
		MaxConcurrent: 5,
		MaxDuration:   3600,
	}, nil
}

// CheckQuota 检查配额是否允许构建
func (s *ResourceQuotaService) CheckQuota(ctx context.Context, pipelineID uint64, projectID *uint64) (*QuotaCheckResult, error) {
	log := logger.L().WithField("pipeline_id", pipelineID)

	quota, err := s.GetEffectiveQuota(ctx, projectID)
	if err != nil {
		return nil, err
	}

	// 检查当前并发数
	var runningCount int64
	s.db.Model(&pipeline.BuildResourceUsage{}).
		Where("pipeline_id = ? AND completed_at IS NULL", pipelineID).
		Count(&runningCount)

	result := &QuotaCheckResult{
		Allowed:        true,
		Quota:          quota,
		CurrentRunning: int(runningCount),
	}

	if int(runningCount) >= quota.MaxConcurrent {
		result.Allowed = false
		result.Reason = fmt.Sprintf("已达到最大并发数限制 (%d/%d)", runningCount, quota.MaxConcurrent)
		log.Warn(result.Reason)
	}

	return result, nil
}

// RecordUsage 记录资源使用
func (s *ResourceQuotaService) RecordUsage(ctx context.Context, usage *pipeline.BuildResourceUsage) error {
	return s.db.Create(usage).Error
}

// CompleteUsage 完成资源使用记录
func (s *ResourceQuotaService) CompleteUsage(ctx context.Context, runID uint64, cpuUsed, memoryUsed, storageUsed string, cacheHit bool, cacheSaved int64) error {
	return s.db.Model(&pipeline.BuildResourceUsage{}).
		Where("run_id = ?", runID).
		Updates(map[string]any{
			"cpu_used":     cpuUsed,
			"memory_used":  memoryUsed,
			"storage_used": storageUsed,
			"cache_hit":    cacheHit,
			"cache_saved":  cacheSaved,
			"completed_at": time.Now(),
			"duration_sec": gorm.Expr("TIMESTAMPDIFF(SECOND, started_at, ?)", time.Now()),
		}).Error
}

// GetUsageStats 获取使用统计
func (s *ResourceQuotaService) GetUsageStats(ctx context.Context, pipelineID uint64, startTime, endTime time.Time) (*UsageStats, error) {
	stats := &UsageStats{}

	query := s.db.Model(&pipeline.BuildResourceUsage{}).
		Where("pipeline_id = ? AND started_at BETWEEN ? AND ?", pipelineID, startTime, endTime)

	// 总构建次数
	query.Count(&stats.TotalBuilds)

	// 平均构建时长
	query.Select("COALESCE(AVG(duration_sec), 0)").Scan(&stats.AvgDurationSec)

	// 缓存命中率
	var cacheHits int64
	query.Where("cache_hit = ?", true).Count(&cacheHits)
	if stats.TotalBuilds > 0 {
		stats.CacheHitRate = float64(cacheHits) / float64(stats.TotalBuilds) * 100
	}

	// 缓存节省时间
	query.Select("COALESCE(SUM(cache_saved), 0)").Scan(&stats.TotalCacheSaved)

	return stats, nil
}

// QuotaCheckResult 配额检查结果
type QuotaCheckResult struct {
	Allowed        bool                         `json:"allowed"`
	Reason         string                       `json:"reason,omitempty"`
	Quota          *pipeline.BuildResourceQuota `json:"quota"`
	CurrentRunning int                          `json:"current_running"`
}

// UsageStats 使用统计
type UsageStats struct {
	TotalBuilds     int64   `json:"total_builds"`
	AvgDurationSec  float64 `json:"avg_duration_sec"`
	CacheHitRate    float64 `json:"cache_hit_rate"`
	TotalCacheSaved int64   `json:"total_cache_saved"`
}
