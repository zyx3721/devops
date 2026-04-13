package pipeline

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"time"

	"gorm.io/gorm"

	"devops/internal/models"
	"devops/pkg/dto"
	"devops/pkg/logger"
)

// 流水线名称正则：只允许英文字母、数字、下划线、横线
var pipelineNameRegex = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_-]*$`)

// validatePipelineName 校验流水线名称
func validatePipelineName(name string) error {
	if name == "" {
		return fmt.Errorf("流水线名称不能为空")
	}
	if len(name) < 2 || len(name) > 64 {
		return fmt.Errorf("流水线名称长度必须在 2-64 个字符之间")
	}
	if !pipelineNameRegex.MatchString(name) {
		return fmt.Errorf("流水线名称只能包含英文字母、数字、下划线和横线，且必须以字母开头")
	}
	return nil
}

// PipelineService 流水线服务
type PipelineService struct {
	db *gorm.DB
}

// NewPipelineService 创建流水线服务
func NewPipelineService(db *gorm.DB) *PipelineService {
	return &PipelineService{db: db}
}

// GetDB 获取数据库连接
func (s *PipelineService) GetDB() *gorm.DB {
	return s.db
}

// List 获取流水线列表
func (s *PipelineService) List(ctx context.Context, req *dto.PipelineListRequest) (*dto.PipelineListResponse, error) {
	var pipelines []models.Pipeline
	var total int64

	query := s.db.Model(&models.Pipeline{})

	if req.Name != "" {
		query = query.Where("name LIKE ?", "%"+req.Name+"%")
	}
	if req.ProjectID > 0 {
		query = query.Where("project_id = ?", req.ProjectID)
	}
	if req.Status != "" {
		query = query.Where("status = ?", req.Status)
	}

	query.Count(&total)

	page := req.Page
	if page <= 0 {
		page = 1
	}
	pageSize := req.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}

	query.Order("updated_at DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&pipelines)

	items := make([]dto.PipelineItem, 0, len(pipelines))
	for _, p := range pipelines {
		item := dto.PipelineItem{
			ID:            p.ID,
			Name:          p.Name,
			Description:   p.Description,
			ProjectID:     p.ProjectID,
			GitRepoID:     p.GitRepoID,
			GitBranch:     p.GitBranch,
			Status:        p.Status,
			LastRunAt:     p.LastRunAt,
			LastRunStatus: p.LastRunStatus,
			CreatedBy:     p.CreatedBy,
			CreatedAt:     p.CreatedAt,
		}
		// 获取 Git 仓库 URL
		if p.GitRepoID != nil && *p.GitRepoID > 0 {
			var gitRepo models.GitRepository
			if s.db.First(&gitRepo, *p.GitRepoID).Error == nil {
				item.GitRepoURL = gitRepo.URL
			}
		}
		items = append(items, item)
	}

	return &dto.PipelineListResponse{
		Total: int(total),
		Items: items,
	}, nil
}

// Get 获取流水线详情
func (s *PipelineService) Get(ctx context.Context, id uint) (*dto.PipelineDetailExtResponse, error) {
	var pipeline models.Pipeline
	if err := s.db.First(&pipeline, id).Error; err != nil {
		return nil, err
	}

	result := &dto.PipelineDetailExtResponse{
		ID:             pipeline.ID,
		Name:           pipeline.Name,
		Description:    pipeline.Description,
		ProjectID:      pipeline.ProjectID,
		GitRepoID:      pipeline.GitRepoID,
		GitBranch:      pipeline.GitBranch,
		BuildClusterID: pipeline.BuildClusterID,
		BuildNamespace: pipeline.BuildNamespace,
		Status:         pipeline.Status,
		LastRunAt:      pipeline.LastRunAt,
		LastRunStatus:  pipeline.LastRunStatus,
		CreatedBy:      pipeline.CreatedBy,
		CreatedAt:      pipeline.CreatedAt,
		UpdatedAt:      pipeline.UpdatedAt,
	}

	// 获取 Git 仓库信息
	if pipeline.GitRepoID != nil && *pipeline.GitRepoID > 0 {
		var gitRepo models.GitRepository
		if s.db.First(&gitRepo, *pipeline.GitRepoID).Error == nil {
			result.GitRepoName = gitRepo.Name
			result.GitRepoURL = gitRepo.URL
		}
	}

	// 获取集群名称
	if pipeline.BuildClusterID != nil && *pipeline.BuildClusterID > 0 {
		var cluster models.K8sCluster
		if s.db.First(&cluster, *pipeline.BuildClusterID).Error == nil {
			result.BuildClusterName = cluster.Name
		}
	}

	// 解析配置
	if pipeline.ConfigJSON != "" {
		var config struct {
			Stages    []dto.Stage    `json:"stages"`
			Variables []dto.Variable `json:"variables"`
		}
		if err := json.Unmarshal([]byte(pipeline.ConfigJSON), &config); err == nil {
			result.Stages = config.Stages
			result.Variables = config.Variables
		}
	}

	// 解析触发器配置
	if pipeline.TriggerConfigJSON != "" {
		var triggerConfig dto.TriggerConfig
		if err := json.Unmarshal([]byte(pipeline.TriggerConfigJSON), &triggerConfig); err == nil {
			result.TriggerConfig = triggerConfig
		}
	}

	return result, nil
}

// Create 创建流水线
func (s *PipelineService) Create(ctx context.Context, req *dto.PipelineRequest, userID uint) error {
	log := logger.L().WithField("name", req.Name)

	// 校验流水线名称
	if err := validatePipelineName(req.Name); err != nil {
		return err
	}

	// 序列化配置
	config := struct {
		Stages    []dto.Stage    `json:"stages"`
		Variables []dto.Variable `json:"variables"`
	}{
		Stages:    req.Stages,
		Variables: req.Variables,
	}
	configJSON, err := json.Marshal(config)
	if err != nil {
		return err
	}

	triggerConfigJSON, err := json.Marshal(req.TriggerConfig)
	if err != nil {
		return err
	}

	pipeline := &models.Pipeline{
		Name:              req.Name,
		Description:       req.Description,
		ProjectID:         req.ProjectID,
		GitRepoID:         req.GitRepoID,
		GitBranch:         req.GitBranch,
		BuildClusterID:    req.BuildClusterID,
		BuildNamespace:    req.BuildNamespace,
		ConfigJSON:        string(configJSON),
		TriggerConfigJSON: string(triggerConfigJSON),
		Status:            "active",
		CreatedBy:         &userID,
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}

	// 设置默认值
	if pipeline.GitBranch == "" {
		pipeline.GitBranch = "main"
	}
	if pipeline.BuildNamespace == "" {
		pipeline.BuildNamespace = "devops-build"
	}

	if err := s.db.Create(pipeline).Error; err != nil {
		log.WithField("error", err).Error("创建流水线失败")
		return err
	}

	log.Info("创建流水线成功")
	return nil
}

// Update 更新流水线
func (s *PipelineService) Update(ctx context.Context, req *dto.PipelineRequest) error {
	log := logger.L().WithField("id", req.ID)

	// 校验流水线名称
	if err := validatePipelineName(req.Name); err != nil {
		return err
	}

	var pipeline models.Pipeline
	if err := s.db.First(&pipeline, req.ID).Error; err != nil {
		return err
	}

	// 序列化配置
	config := struct {
		Stages    []dto.Stage    `json:"stages"`
		Variables []dto.Variable `json:"variables"`
	}{
		Stages:    req.Stages,
		Variables: req.Variables,
	}
	configJSON, err := json.Marshal(config)
	if err != nil {
		return err
	}

	triggerConfigJSON, err := json.Marshal(req.TriggerConfig)
	if err != nil {
		return err
	}

	pipeline.Name = req.Name
	pipeline.Description = req.Description
	pipeline.ProjectID = req.ProjectID
	pipeline.GitRepoID = req.GitRepoID
	pipeline.GitBranch = req.GitBranch
	pipeline.BuildClusterID = req.BuildClusterID
	pipeline.BuildNamespace = req.BuildNamespace
	pipeline.ConfigJSON = string(configJSON)
	pipeline.TriggerConfigJSON = string(triggerConfigJSON)
	pipeline.UpdatedAt = time.Now()

	// 设置默认值
	if pipeline.GitBranch == "" {
		pipeline.GitBranch = "main"
	}
	if pipeline.BuildNamespace == "" {
		pipeline.BuildNamespace = "devops-build"
	}

	if err := s.db.Save(&pipeline).Error; err != nil {
		log.WithField("error", err).Error("更新流水线失败")
		return err
	}

	log.Info("更新流水线成功")
	return nil
}

// Delete 删除流水线
func (s *PipelineService) Delete(ctx context.Context, id uint) error {
	// 删除相关的执行记录
	s.db.Where("pipeline_id = ?", id).Delete(&models.PipelineRun{})

	return s.db.Delete(&models.Pipeline{}, id).Error
}

// ToggleStatus 切换状态
func (s *PipelineService) ToggleStatus(ctx context.Context, id uint) error {
	var pipeline models.Pipeline
	if err := s.db.First(&pipeline, id).Error; err != nil {
		return err
	}

	if pipeline.Status == "active" {
		pipeline.Status = "disabled"
	} else {
		pipeline.Status = "active"
	}
	pipeline.UpdatedAt = time.Now()

	return s.db.Save(&pipeline).Error
}

// Validate 验证流水线配置
func (s *PipelineService) Validate(ctx context.Context, req *dto.PipelineRequest) error {
	// 检查阶段
	if len(req.Stages) == 0 {
		return &ValidationError{Message: "至少需要一个阶段"}
	}

	stageIDs := make(map[string]bool)
	for _, stage := range req.Stages {
		if stage.ID == "" {
			return &ValidationError{Message: "阶段ID不能为空"}
		}
		if stageIDs[stage.ID] {
			return &ValidationError{Message: "阶段ID重复: " + stage.ID}
		}
		stageIDs[stage.ID] = true

		// 检查依赖
		for _, dep := range stage.DependsOn {
			if !stageIDs[dep] {
				return &ValidationError{Message: "依赖的阶段不存在: " + dep}
			}
		}

		// 检查步骤
		if len(stage.Steps) == 0 {
			return &ValidationError{Message: "阶段至少需要一个步骤: " + stage.Name}
		}
	}

	return nil
}

// ValidationError 验证错误
type ValidationError struct {
	Message string
}

func (e *ValidationError) Error() string {
	return e.Message
}
