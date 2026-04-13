package pipeline

import (
	"context"
	"crypto/sha256"
	"devops/internal/models"
	"devops/pkg/logger"
	"encoding/hex"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"gorm.io/gorm"
)

// CacheService 构建缓存服务
type CacheService struct {
	db *gorm.DB
}

// NewCacheService 创建缓存服务
func NewCacheService(db *gorm.DB) *CacheService {
	return &CacheService{db: db}
}

// CacheConfig 缓存配置
type CacheConfig struct {
	Key      string   `json:"key"`       // 缓存键模板
	Paths    []string `json:"paths"`     // 缓存路径
	Policy   string   `json:"policy"`    // pull, push, pull-push
	WhenExpr string   `json:"when_expr"` // 条件表达式
}

// SaveCache 保存缓存记录
func (s *CacheService) SaveCache(ctx context.Context, cache *models.BuildCache) error {
	log := logger.L().WithField("pipeline_id", cache.PipelineID).WithField("cache_key", cache.CacheKey)
	log.Info("保存构建缓存")

	// 检查是否已存在相同 key 的缓存
	var existing models.BuildCache
	err := s.db.WithContext(ctx).
		Where("pipeline_id = ? AND cache_key = ?", cache.PipelineID, cache.CacheKey).
		First(&existing).Error

	if err == nil {
		// 更新现有缓存
		cache.ID = existing.ID
		cache.HitCount = existing.HitCount
		cache.UpdatedAt = time.Now()
		if err := s.db.WithContext(ctx).Save(cache).Error; err != nil {
			log.WithField("error", err).Error("更新缓存记录失败")
			return fmt.Errorf("更新缓存记录失败: %w", err)
		}
		log.Info("缓存记录已更新")
		return nil
	}

	if err != gorm.ErrRecordNotFound {
		log.WithField("error", err).Error("查询缓存记录失败")
		return fmt.Errorf("查询缓存记录失败: %w", err)
	}

	// 创建新缓存
	cache.CreatedAt = time.Now()
	cache.UpdatedAt = time.Now()
	if err := s.db.WithContext(ctx).Create(cache).Error; err != nil {
		log.WithField("error", err).Error("创建缓存记录失败")
		return fmt.Errorf("创建缓存记录失败: %w", err)
	}

	log.Info("缓存记录已创建")
	return nil
}

// RestoreCache 恢复缓存
func (s *CacheService) RestoreCache(ctx context.Context, pipelineID uint, cacheKey string) (*models.BuildCache, error) {
	log := logger.L().WithField("pipeline_id", pipelineID).WithField("cache_key", cacheKey)
	log.Info("恢复构建缓存")

	var cache models.BuildCache
	err := s.db.WithContext(ctx).
		Where("pipeline_id = ? AND cache_key = ?", pipelineID, cacheKey).
		First(&cache).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			log.Info("缓存未命中")
			return nil, nil
		}
		log.WithField("error", err).Error("查询缓存失败")
		return nil, fmt.Errorf("查询缓存失败: %w", err)
	}

	// 检查缓存是否过期
	if cache.ExpiresAt != nil && cache.ExpiresAt.Before(time.Now()) {
		log.Info("缓存已过期")
		return nil, nil
	}

	// 更新命中次数和最后使用时间
	now := time.Now()
	s.db.WithContext(ctx).Model(&cache).Updates(map[string]interface{}{
		"hit_count":    gorm.Expr("hit_count + 1"),
		"last_used_at": now,
	})

	log.WithField("hit_count", cache.HitCount+1).Info("缓存命中")
	return &cache, nil
}

// InvalidateCache 使缓存失效
func (s *CacheService) InvalidateCache(ctx context.Context, pipelineID uint, cacheKey string) error {
	log := logger.L().WithField("pipeline_id", pipelineID).WithField("cache_key", cacheKey)
	log.Info("使缓存失效")

	result := s.db.WithContext(ctx).
		Where("pipeline_id = ? AND cache_key = ?", pipelineID, cacheKey).
		Delete(&models.BuildCache{})

	if result.Error != nil {
		log.WithField("error", result.Error).Error("删除缓存失败")
		return fmt.Errorf("删除缓存失败: %w", result.Error)
	}

	log.WithField("deleted", result.RowsAffected).Info("缓存已失效")
	return nil
}

// InvalidatePipelineCaches 使流水线所有缓存失效
func (s *CacheService) InvalidatePipelineCaches(ctx context.Context, pipelineID uint) error {
	log := logger.L().WithField("pipeline_id", pipelineID)
	log.Info("使流水线所有缓存失效")

	result := s.db.WithContext(ctx).
		Where("pipeline_id = ?", pipelineID).
		Delete(&models.BuildCache{})

	if result.Error != nil {
		log.WithField("error", result.Error).Error("删除缓存失败")
		return fmt.Errorf("删除缓存失败: %w", result.Error)
	}

	log.WithField("deleted", result.RowsAffected).Info("缓存已全部失效")
	return nil
}

// CleanupExpired 清理过期缓存
func (s *CacheService) CleanupExpired(ctx context.Context) (int64, error) {
	log := logger.L()
	log.Info("开始清理过期缓存")

	result := s.db.WithContext(ctx).
		Where("expires_at IS NOT NULL AND expires_at < ?", time.Now()).
		Delete(&models.BuildCache{})

	if result.Error != nil {
		log.WithField("error", result.Error).Error("清理过期缓存失败")
		return 0, fmt.Errorf("清理过期缓存失败: %w", result.Error)
	}

	log.WithField("deleted", result.RowsAffected).Info("过期缓存清理完成")
	return result.RowsAffected, nil
}

// CleanupUnused 清理长期未使用的缓存
func (s *CacheService) CleanupUnused(ctx context.Context, unusedDays int) (int64, error) {
	log := logger.L().WithField("unused_days", unusedDays)
	log.Info("开始清理未使用缓存")

	threshold := time.Now().AddDate(0, 0, -unusedDays)
	result := s.db.WithContext(ctx).
		Where("last_used_at IS NULL OR last_used_at < ?", threshold).
		Delete(&models.BuildCache{})

	if result.Error != nil {
		log.WithField("error", result.Error).Error("清理未使用缓存失败")
		return 0, fmt.Errorf("清理未使用缓存失败: %w", result.Error)
	}

	log.WithField("deleted", result.RowsAffected).Info("未使用缓存清理完成")
	return result.RowsAffected, nil
}

// ListCaches 列出缓存
func (s *CacheService) ListCaches(ctx context.Context, pipelineID uint, page, pageSize int) ([]models.BuildCache, int64, error) {
	var caches []models.BuildCache
	var total int64

	query := s.db.WithContext(ctx).Model(&models.BuildCache{})
	if pipelineID > 0 {
		query = query.Where("pipeline_id = ?", pipelineID)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("统计缓存数量失败: %w", err)
	}

	if err := query.Order("updated_at DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&caches).Error; err != nil {
		return nil, 0, fmt.Errorf("查询缓存列表失败: %w", err)
	}

	return caches, total, nil
}

// GetCacheStats 获取缓存统计
func (s *CacheService) GetCacheStats(ctx context.Context, pipelineID uint) (*CacheStats, error) {
	var stats CacheStats

	query := s.db.WithContext(ctx).Model(&models.BuildCache{})
	if pipelineID > 0 {
		query = query.Where("pipeline_id = ?", pipelineID)
	}

	// 总数和总大小
	if err := query.Select("COUNT(*) as total_count, COALESCE(SUM(size_bytes), 0) as total_size").
		Scan(&stats).Error; err != nil {
		return nil, fmt.Errorf("统计缓存失败: %w", err)
	}

	// 命中率
	var hitStats struct {
		TotalHits int64
	}
	if err := query.Select("COALESCE(SUM(hit_count), 0) as total_hits").
		Scan(&hitStats).Error; err != nil {
		return nil, fmt.Errorf("统计命中率失败: %w", err)
	}
	stats.TotalHits = hitStats.TotalHits

	return &stats, nil
}

// CacheStats 缓存统计
type CacheStats struct {
	TotalCount int64 `json:"total_count"`
	TotalSize  int64 `json:"total_size"`
	TotalHits  int64 `json:"total_hits"`
}

// ComputeCacheKey 计算缓存键
func (s *CacheService) ComputeCacheKey(keyTemplate string, variables map[string]string) string {
	key := keyTemplate

	// 替换变量
	for k, v := range variables {
		key = strings.ReplaceAll(key, fmt.Sprintf("${%s}", k), v)
		key = strings.ReplaceAll(key, fmt.Sprintf("$%s", k), v)
	}

	return key
}

// ComputeFileHash 计算文件哈希（用于缓存键）
func (s *CacheService) ComputeFileHash(files []string) string {
	h := sha256.New()
	for _, f := range files {
		h.Write([]byte(f))
	}
	return hex.EncodeToString(h.Sum(nil))[:16]
}

// ComputeContentHash 计算文件内容哈希
func (s *CacheService) ComputeContentHash(contents map[string][]byte) string {
	h := sha256.New()
	// 按文件名排序以确保一致性
	keys := make([]string, 0, len(contents))
	for k := range contents {
		keys = append(keys, k)
	}
	// 简单排序
	for i := 0; i < len(keys); i++ {
		for j := i + 1; j < len(keys); j++ {
			if keys[i] > keys[j] {
				keys[i], keys[j] = keys[j], keys[i]
			}
		}
	}
	for _, k := range keys {
		h.Write([]byte(k))
		h.Write(contents[k])
	}
	return hex.EncodeToString(h.Sum(nil))
}

// CheckCacheHit 检查缓存是否命中
func (s *CacheService) CheckCacheHit(ctx context.Context, pipelineID uint, cacheKey string) (*CacheHitResult, error) {
	cache, err := s.RestoreCache(ctx, pipelineID, cacheKey)
	if err != nil {
		return &CacheHitResult{Hit: false, Reason: "查询失败"}, err
	}
	if cache == nil {
		return &CacheHitResult{Hit: false, Reason: "缓存不存在"}, nil
	}
	if cache.ExpiresAt != nil && cache.ExpiresAt.Before(time.Now()) {
		return &CacheHitResult{Hit: false, Reason: "缓存已过期"}, nil
	}
	return &CacheHitResult{
		Hit:       true,
		Cache:     cache,
		HitCount:  cache.HitCount,
		CreatedAt: cache.CreatedAt,
	}, nil
}

// CacheHitResult 缓存命中结果
type CacheHitResult struct {
	Hit       bool               `json:"hit"`
	Reason    string             `json:"reason,omitempty"`
	Cache     *models.BuildCache `json:"cache,omitempty"`
	HitCount  int                `json:"hit_count"`
	CreatedAt time.Time          `json:"created_at,omitempty"`
}

// ComputeDependencyHash 计算依赖文件哈希（用于 npm/go/pip 等包管理器缓存）
func (s *CacheService) ComputeDependencyHash(lockFiles map[string]string) string {
	h := sha256.New()
	// 常见的依赖锁文件
	priorityOrder := []string{
		"package-lock.json", "yarn.lock", "pnpm-lock.yaml",
		"go.sum", "go.mod",
		"Pipfile.lock", "poetry.lock", "requirements.txt",
		"Gemfile.lock", "Cargo.lock", "composer.lock",
	}
	for _, file := range priorityOrder {
		if content, ok := lockFiles[file]; ok {
			h.Write([]byte(file))
			h.Write([]byte(content))
		}
	}
	// 处理其他文件
	for file, content := range lockFiles {
		found := false
		for _, pf := range priorityOrder {
			if file == pf {
				found = true
				break
			}
		}
		if !found {
			h.Write([]byte(file))
			h.Write([]byte(content))
		}
	}
	return hex.EncodeToString(h.Sum(nil))[:32]
}

// GenerateCacheKeyFromTemplate 从模板生成缓存键
func (s *CacheService) GenerateCacheKeyFromTemplate(template string, vars map[string]string) string {
	result := template
	// 替换内置变量
	builtinVars := map[string]func() string{
		"${date}":      func() string { return time.Now().Format("2006-01-02") },
		"${week}":      func() string { return fmt.Sprintf("%d", time.Now().YearDay()/7) },
		"${month}":     func() string { return time.Now().Format("2006-01") },
		"${timestamp}": func() string { return fmt.Sprintf("%d", time.Now().Unix()) },
	}
	for placeholder, fn := range builtinVars {
		result = strings.ReplaceAll(result, placeholder, fn())
	}
	// 替换用户变量
	for k, v := range vars {
		result = strings.ReplaceAll(result, fmt.Sprintf("${%s}", k), v)
		result = strings.ReplaceAll(result, fmt.Sprintf("$%s", k), v)
	}
	return result
}

// GetCachePolicy 获取缓存策略
func (s *CacheService) GetCachePolicy(policy string) CachePolicyType {
	switch strings.ToLower(policy) {
	case "pull":
		return CachePolicyPull
	case "push":
		return CachePolicyPush
	case "pull-push", "pullpush":
		return CachePolicyPullPush
	default:
		return CachePolicyPullPush
	}
}

// CachePolicyType 缓存策略类型
type CachePolicyType int

const (
	CachePolicyPull CachePolicyType = iota
	CachePolicyPush
	CachePolicyPullPush
)

// ShouldRestoreCache 是否应该恢复缓存
func (p CachePolicyType) ShouldRestoreCache() bool {
	return p == CachePolicyPull || p == CachePolicyPullPush
}

// ShouldSaveCache 是否应该保存缓存
func (p CachePolicyType) ShouldSaveCache() bool {
	return p == CachePolicyPush || p == CachePolicyPullPush
}

// GetCachePVCName 获取缓存 PVC 名称
func (s *CacheService) GetCachePVCName(pipelineID uint) string {
	return fmt.Sprintf("pipeline-%d-cache", pipelineID)
}

// GetCacheMountPath 获取缓存挂载路径
func (s *CacheService) GetCacheMountPath(cacheKey string) string {
	// 将缓存键转换为安全的路径
	safePath := strings.ReplaceAll(cacheKey, "/", "-")
	safePath = strings.ReplaceAll(safePath, ":", "-")
	return filepath.Join("/cache", safePath)
}

// GenerateCacheVolumeMounts 生成缓存卷挂载配置
func (s *CacheService) GenerateCacheVolumeMounts(caches []CacheConfig) []CacheVolumeMount {
	var mounts []CacheVolumeMount

	for i, cache := range caches {
		for _, path := range cache.Paths {
			mounts = append(mounts, CacheVolumeMount{
				Name:      fmt.Sprintf("cache-%d", i),
				MountPath: path,
				SubPath:   s.ComputeFileHash([]string{cache.Key, path}),
			})
		}
	}

	return mounts
}

// CacheVolumeMount 缓存卷挂载
type CacheVolumeMount struct {
	Name      string `json:"name"`
	MountPath string `json:"mount_path"`
	SubPath   string `json:"sub_path"`
}
