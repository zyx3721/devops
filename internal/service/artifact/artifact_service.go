// Package artifact 制品管理服务
package artifact

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	"devops/internal/models/artifact"
	"devops/pkg/logger"
)

// ArtifactService 制品服务
type ArtifactService struct {
	db *gorm.DB
}

// NewArtifactService 创建制品服务
func NewArtifactService(db *gorm.DB) *ArtifactService {
	return &ArtifactService{db: db}
}

// ========== 仓库管理 ==========

// CreateRepository 创建仓库
func (s *ArtifactService) CreateRepository(ctx context.Context, repo *artifact.Repository) error {
	if repo.IsDefault {
		// 取消同类型其他默认仓库
		s.db.Model(&artifact.Repository{}).
			Where("type = ? AND is_default = ?", repo.Type, true).
			Update("is_default", false)
	}
	return s.db.Create(repo).Error
}

// UpdateRepository 更新仓库
func (s *ArtifactService) UpdateRepository(ctx context.Context, repo *artifact.Repository) error {
	if repo.IsDefault {
		s.db.Model(&artifact.Repository{}).
			Where("type = ? AND is_default = ? AND id != ?", repo.Type, true, repo.ID).
			Update("is_default", false)
	}
	return s.db.Save(repo).Error
}

// DeleteRepository 删除仓库
func (s *ArtifactService) DeleteRepository(ctx context.Context, repoID uint64) error {
	// 检查是否有制品
	var count int64
	s.db.Model(&artifact.Artifact{}).Where("repository_id = ?", repoID).Count(&count)
	if count > 0 {
		return fmt.Errorf("仓库下存在 %d 个制品，无法删除", count)
	}
	return s.db.Delete(&artifact.Repository{}, repoID).Error
}

// GetRepository 获取仓库
func (s *ArtifactService) GetRepository(ctx context.Context, repoID uint64) (*artifact.Repository, error) {
	var repo artifact.Repository
	err := s.db.First(&repo, repoID).Error
	return &repo, err
}

// ListRepositories 获取仓库列表
func (s *ArtifactService) ListRepositories(ctx context.Context, repoType string) ([]artifact.Repository, error) {
	var repos []artifact.Repository
	query := s.db.Model(&artifact.Repository{})
	if repoType != "" {
		query = query.Where("type = ?", repoType)
	}
	err := query.Order("is_default DESC, created_at DESC").Find(&repos).Error
	return repos, err
}

// GetDefaultRepository 获取默认仓库
func (s *ArtifactService) GetDefaultRepository(ctx context.Context, repoType string) (*artifact.Repository, error) {
	var repo artifact.Repository
	err := s.db.Where("type = ? AND is_default = ? AND enabled = ?", repoType, true, true).First(&repo).Error
	return &repo, err
}

// ========== 制品管理 ==========

// CreateArtifact 创建制品
func (s *ArtifactService) CreateArtifact(ctx context.Context, art *artifact.Artifact) error {
	return s.db.Create(art).Error
}

// UpdateArtifact 更新制品
func (s *ArtifactService) UpdateArtifact(ctx context.Context, art *artifact.Artifact) error {
	return s.db.Save(art).Error
}

// DeleteArtifact 删除制品
func (s *ArtifactService) DeleteArtifact(ctx context.Context, artifactID uint64) error {
	// 删除所有版本
	s.db.Where("artifact_id = ?", artifactID).Delete(&artifact.ArtifactVersion{})
	return s.db.Delete(&artifact.Artifact{}, artifactID).Error
}

// GetArtifact 获取制品
func (s *ArtifactService) GetArtifact(ctx context.Context, artifactID uint64) (*artifact.Artifact, error) {
	var art artifact.Artifact
	err := s.db.First(&art, artifactID).Error
	return &art, err
}

// ListArtifacts 获取制品列表
func (s *ArtifactService) ListArtifacts(ctx context.Context, repoID uint64, keyword string, page, pageSize int) (*ArtifactListResult, error) {
	var arts []artifact.Artifact
	query := s.db.Model(&artifact.Artifact{})

	if repoID > 0 {
		query = query.Where("repository_id = ?", repoID)
	}
	if keyword != "" {
		query = query.Where("name LIKE ? OR description LIKE ?", "%"+keyword+"%", "%"+keyword+"%")
	}

	var total int64
	query.Count(&total)

	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}

	err := query.Offset((page - 1) * pageSize).Limit(pageSize).
		Order("updated_at DESC").Find(&arts).Error

	return &ArtifactListResult{
		Items: arts,
		Total: total,
		Page:  page,
		Size:  pageSize,
	}, err
}

// SearchArtifacts 搜索制品
func (s *ArtifactService) SearchArtifacts(ctx context.Context, keyword string, artType string) ([]artifact.Artifact, error) {
	var arts []artifact.Artifact
	query := s.db.Model(&artifact.Artifact{}).
		Where("name LIKE ? OR group_id LIKE ? OR artifact_id LIKE ?",
			"%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%")

	if artType != "" {
		query = query.Where("type = ?", artType)
	}

	err := query.Limit(50).Order("download_cnt DESC").Find(&arts).Error
	return arts, err
}

// ========== 版本管理 ==========

// CreateVersion 创建版本
func (s *ArtifactService) CreateVersion(ctx context.Context, ver *artifact.ArtifactVersion) error {
	log := logger.L().WithField("artifact_id", ver.ArtifactID).WithField("version", ver.Version)

	if err := s.db.Create(ver).Error; err != nil {
		return err
	}

	// 更新制品最新版本
	s.db.Model(&artifact.Artifact{}).Where("id = ?", ver.ArtifactID).
		Update("latest_version", ver.Version)

	log.Info("创建制品版本")
	return nil
}

// GetVersion 获取版本
func (s *ArtifactService) GetVersion(ctx context.Context, versionID uint64) (*artifact.ArtifactVersion, error) {
	var ver artifact.ArtifactVersion
	err := s.db.First(&ver, versionID).Error
	return &ver, err
}

// GetVersionByName 根据版本号获取版本
func (s *ArtifactService) GetVersionByName(ctx context.Context, artifactID uint64, version string) (*artifact.ArtifactVersion, error) {
	var ver artifact.ArtifactVersion
	err := s.db.Where("artifact_id = ? AND version = ?", artifactID, version).First(&ver).Error
	return &ver, err
}

// ListVersions 获取版本列表
func (s *ArtifactService) ListVersions(ctx context.Context, artifactID uint64, page, pageSize int) (*VersionListResult, error) {
	var vers []artifact.ArtifactVersion
	query := s.db.Model(&artifact.ArtifactVersion{}).Where("artifact_id = ?", artifactID)

	var total int64
	query.Count(&total)

	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}

	err := query.Offset((page - 1) * pageSize).Limit(pageSize).
		Order("created_at DESC").Find(&vers).Error

	return &VersionListResult{
		Items: vers,
		Total: total,
		Page:  page,
		Size:  pageSize,
	}, err
}

// DeleteVersion 删除版本
func (s *ArtifactService) DeleteVersion(ctx context.Context, versionID uint64) error {
	// 删除扫描结果
	s.db.Where("version_id = ?", versionID).Delete(&artifact.ArtifactScanResult{})
	return s.db.Delete(&artifact.ArtifactVersion{}, versionID).Error
}

// ReleaseVersion 发布版本
func (s *ArtifactService) ReleaseVersion(ctx context.Context, versionID uint64, releasedBy string) error {
	return s.db.Model(&artifact.ArtifactVersion{}).Where("id = ?", versionID).
		Updates(map[string]any{
			"is_release":  true,
			"released_at": time.Now(),
			"released_by": releasedBy,
		}).Error
}

// IncrementDownload 增加下载次数
func (s *ArtifactService) IncrementDownload(ctx context.Context, versionID uint64) error {
	// 更新版本下载次数
	s.db.Model(&artifact.ArtifactVersion{}).Where("id = ?", versionID).
		UpdateColumn("download_cnt", gorm.Expr("download_cnt + 1"))

	// 更新制品下载次数
	var ver artifact.ArtifactVersion
	s.db.First(&ver, versionID)
	s.db.Model(&artifact.Artifact{}).Where("id = ?", ver.ArtifactID).
		UpdateColumn("download_cnt", gorm.Expr("download_cnt + 1"))

	return nil
}

// ArtifactListResult 制品列表结果
type ArtifactListResult struct {
	Items []artifact.Artifact `json:"items"`
	Total int64               `json:"total"`
	Page  int                 `json:"page"`
	Size  int                 `json:"size"`
}

// VersionListResult 版本列表结果
type VersionListResult struct {
	Items []artifact.ArtifactVersion `json:"items"`
	Total int64                      `json:"total"`
	Page  int                        `json:"page"`
	Size  int                        `json:"size"`
}

// ========== 连接监控 ==========

// ConnectionTestResult 连接测试结果
type ConnectionTestResult struct {
	Connected    bool   `json:"connected"`
	ResponseTime int    `json:"response_time"` // 毫秒
	Error        string `json:"error,omitempty"`
}

// ConnectionHistory 连接历史记录
type ConnectionHistory struct {
	ID           uint64    `json:"id"`
	RegistryID   uint64    `json:"registry_id"`
	Status       string    `json:"status"`
	ResponseTime int       `json:"response_time,omitempty"`
	ErrorMessage string    `json:"error_message,omitempty"`
	CheckedAt    time.Time `json:"checked_at"`
}

// TestConnection 测试仓库连接
func (s *ArtifactService) TestConnection(ctx context.Context, repoID uint64) (*ConnectionTestResult, error) {
	repo, err := s.GetRepository(ctx, repoID)
	if err != nil {
		return nil, err
	}

	startTime := time.Now()
	result := &ConnectionTestResult{}

	// 这里应该根据不同的仓库类型进行实际的连接测试
	// 简化实现：模拟连接测试
	// TODO: 实现真实的连接测试逻辑
	connected, testErr := s.testRepositoryConnection(repo)

	responseTime := int(time.Since(startTime).Milliseconds())
	result.ResponseTime = responseTime
	result.Connected = connected

	// 更新仓库状态
	status := "connected"
	var lastError string
	if !connected {
		status = "disconnected"
		if testErr != nil {
			lastError = testErr.Error()
			result.Error = lastError
		}
	}

	// 更新数据库
	s.db.Model(&artifact.Repository{}).Where("id = ?", repoID).Updates(map[string]interface{}{
		"connection_status": status,
		"last_check_at":     time.Now(),
		"last_error":        lastError,
	})

	// 记录历史
	history := map[string]interface{}{
		"registry_id":   repoID,
		"status":        map[bool]string{true: "success", false: "failed"}[connected],
		"response_time": responseTime,
		"error_message": lastError,
		"checked_at":    time.Now(),
	}
	s.db.Table("artifact_registry_connection_history").Create(history)

	return result, nil
}

// testRepositoryConnection 测试仓库连接（实际实现）
func (s *ArtifactService) testRepositoryConnection(repo *artifact.Repository) (bool, error) {
	// TODO: 根据不同的仓库类型实现真实的连接测试
	// 这里是简化实现，实际应该：
	// 1. Harbor: 调用 /api/v2.0/health 或 /api/v2.0/systeminfo
	// 2. Nexus: 调用 /service/rest/v1/status
	// 3. Docker Hub: 调用 /v2/ 端点
	// 4. 其他: 根据具体类型实现

	// 简化实现：检查 URL 是否可访问
	if repo.URL == "" {
		return false, fmt.Errorf("仓库地址为空")
	}

	// 模拟连接测试
	// 实际应该使用 HTTP 客户端进行真实测试

	logger.GetLogger().Info("Testing connection to repository: %s (%s)", repo.Name, repo.URL)

	// 这里返回成功，实际应该进行真实的网络请求
	return true, nil
}

// RefreshAllStatus 刷新所有仓库状态
func (s *ArtifactService) RefreshAllStatus(ctx context.Context) error {
	var repos []artifact.Repository
	if err := s.db.Where("enabled = ? AND enable_monitoring = ?", true, true).Find(&repos).Error; err != nil {
		return err
	}

	for _, repo := range repos {
		// 异步测试连接，避免阻塞
		go func(r artifact.Repository) {
			_, err := s.TestConnection(ctx, r.ID)
			if err != nil {
				logger.GetLogger().Error("Failed to test connection for repository %s: %v", r.Name, err)
			}
		}(repo)
	}

	return nil
}

// GetConnectionHistory 获取连接历史记录
func (s *ArtifactService) GetConnectionHistory(ctx context.Context, repoID uint64, page, pageSize int) (map[string]interface{}, error) {
	var histories []ConnectionHistory
	var total int64

	offset := (page - 1) * pageSize

	// 查询总数
	s.db.Table("artifact_registry_connection_history").
		Where("registry_id = ?", repoID).
		Count(&total)

	// 查询历史记录
	err := s.db.Table("artifact_registry_connection_history").
		Where("registry_id = ?", repoID).
		Order("checked_at DESC").
		Limit(pageSize).
		Offset(offset).
		Find(&histories).Error

	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"items":     histories,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	}, nil
}
