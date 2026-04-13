package pipeline

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	"devops/internal/models"
	"devops/pkg/dto"
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

// List 获取制品列表
func (s *ArtifactService) List(ctx context.Context, req *dto.ArtifactListRequest) (*dto.ArtifactListResponse, error) {
	var artifacts []models.Artifact
	var total int64

	query := s.db.Model(&models.Artifact{})

	if req.PipelineID > 0 {
		query = query.Where("pipeline_id = ?", req.PipelineID)
	}
	if req.PipelineRunID > 0 {
		query = query.Where("pipeline_run_id = ?", req.PipelineRunID)
	}
	if req.Type != "" {
		query = query.Where("type = ?", req.Type)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	page := req.Page
	if page < 1 {
		page = 1
	}
	pageSize := req.PageSize
	if pageSize < 1 {
		pageSize = 20
	}

	if err := query.Offset((page - 1) * pageSize).Limit(pageSize).
		Order("created_at DESC").Find(&artifacts).Error; err != nil {
		return nil, err
	}

	items := make([]dto.ArtifactItem, len(artifacts))
	for i, artifact := range artifacts {
		items[i] = s.toArtifactItem(&artifact)
	}

	return &dto.ArtifactListResponse{
		Total: int(total),
		Items: items,
	}, nil
}

// Get 获取制品详情
func (s *ArtifactService) Get(ctx context.Context, id uint) (*dto.ArtifactItem, error) {
	var artifact models.Artifact
	if err := s.db.First(&artifact, id).Error; err != nil {
		return nil, err
	}

	item := s.toArtifactItem(&artifact)
	return &item, nil
}

// Create 创建制品记录
func (s *ArtifactService) Create(ctx context.Context, req *dto.ArtifactCreateRequest) (*dto.ArtifactItem, error) {
	// 获取流水线 ID
	var pipelineID *uint
	if req.PipelineRunID > 0 {
		var run models.PipelineRun
		if err := s.db.First(&run, req.PipelineRunID).Error; err == nil {
			pipelineID = &run.PipelineID
		}
	}

	artifact := &models.Artifact{
		PipelineRunID: req.PipelineRunID,
		PipelineID:    pipelineID,
		Name:          req.Name,
		Type:          req.Type,
		Path:          req.Path,
		Size:          req.Size,
		Checksum:      req.Checksum,
		Metadata:      req.Metadata,
		GitCommit:     req.GitCommit,
		GitBranch:     req.GitBranch,
		CreatedAt:     time.Now(),
	}

	if err := s.db.Create(artifact).Error; err != nil {
		return nil, err
	}

	item := s.toArtifactItem(artifact)
	return &item, nil
}

// Delete 删除制品
func (s *ArtifactService) Delete(ctx context.Context, id uint) error {
	return s.db.Delete(&models.Artifact{}, id).Error
}

// DeleteByPipelineRun 删除流水线执行的所有制品
func (s *ArtifactService) DeleteByPipelineRun(ctx context.Context, pipelineRunID uint) error {
	return s.db.Where("pipeline_run_id = ?", pipelineRunID).Delete(&models.Artifact{}).Error
}

// CleanupExpired 清理过期制品
func (s *ArtifactService) CleanupExpired(ctx context.Context, maxAge time.Duration, maxCount int) error {
	log := logger.L()
	log.Info("开始清理过期制品")

	// 按时间清理
	cutoff := time.Now().Add(-maxAge)
	result := s.db.Where("created_at < ?", cutoff).Delete(&models.Artifact{})
	if result.Error != nil {
		return result.Error
	}
	log.WithField("count", result.RowsAffected).Info("按时间清理制品完成")

	// 按数量清理（每个流水线保留最新的 maxCount 个）
	if maxCount > 0 {
		var pipelines []uint
		s.db.Model(&models.Artifact{}).Distinct("pipeline_id").Where("pipeline_id IS NOT NULL").Pluck("pipeline_id", &pipelines)

		for _, pipelineID := range pipelines {
			var artifacts []models.Artifact
			s.db.Where("pipeline_id = ?", pipelineID).Order("created_at DESC").Offset(maxCount).Find(&artifacts)

			for _, artifact := range artifacts {
				s.db.Delete(&artifact)
			}
		}
	}

	return nil
}

// GetByPipelineRun 获取流水线执行的所有制品
func (s *ArtifactService) GetByPipelineRun(ctx context.Context, pipelineRunID uint) ([]dto.ArtifactItem, error) {
	var artifacts []models.Artifact
	if err := s.db.Where("pipeline_run_id = ?", pipelineRunID).Order("created_at DESC").Find(&artifacts).Error; err != nil {
		return nil, err
	}

	items := make([]dto.ArtifactItem, len(artifacts))
	for i, artifact := range artifacts {
		items[i] = s.toArtifactItem(&artifact)
	}

	return items, nil
}

// GetLatestByPipeline 获取流水线最新的制品
func (s *ArtifactService) GetLatestByPipeline(ctx context.Context, pipelineID uint, artifactType string) (*dto.ArtifactItem, error) {
	var artifact models.Artifact
	query := s.db.Where("pipeline_id = ?", pipelineID)
	if artifactType != "" {
		query = query.Where("type = ?", artifactType)
	}

	if err := query.Order("created_at DESC").First(&artifact).Error; err != nil {
		return nil, err
	}

	item := s.toArtifactItem(&artifact)
	return &item, nil
}

// toArtifactItem 转换为 DTO
func (s *ArtifactService) toArtifactItem(artifact *models.Artifact) dto.ArtifactItem {
	item := dto.ArtifactItem{
		ID:            artifact.ID,
		PipelineRunID: artifact.PipelineRunID,
		PipelineID:    artifact.PipelineID,
		Name:          artifact.Name,
		Type:          artifact.Type,
		Path:          artifact.Path,
		Size:          artifact.Size,
		SizeHuman:     formatSize(artifact.Size),
		Checksum:      artifact.Checksum,
		GitCommit:     artifact.GitCommit,
		GitBranch:     artifact.GitBranch,
		CreatedAt:     artifact.CreatedAt,
	}

	// 获取流水线名称
	if artifact.PipelineID != nil {
		var pipeline models.Pipeline
		if err := s.db.First(&pipeline, *artifact.PipelineID).Error; err == nil {
			item.PipelineName = pipeline.Name
		}
	}

	return item
}

// formatSize 格式化文件大小
func formatSize(size int64) string {
	if size < 1024 {
		return fmt.Sprintf("%d B", size)
	}
	if size < 1024*1024 {
		return fmt.Sprintf("%.2f KB", float64(size)/1024)
	}
	if size < 1024*1024*1024 {
		return fmt.Sprintf("%.2f MB", float64(size)/(1024*1024))
	}
	return fmt.Sprintf("%.2f GB", float64(size)/(1024*1024*1024))
}
