package pipeline

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"gorm.io/gorm"

	"devops/internal/models"
	"devops/pkg/dto"
	"devops/pkg/logger"
)

const builderConfigKey = "builder_pod_config"

// RunService 流水线执行服务
type RunService struct {
	db     *gorm.DB
	engine *ExecutorEngine
}

// NewRunService 创建执行服务
func NewRunService(db *gorm.DB) *RunService {
	svc := &RunService{
		db:     db,
		engine: NewExecutorEngine(db),
	}
	// 从数据库加载配置
	svc.loadBuilderConfig()
	return svc
}

// loadBuilderConfig 从数据库加载构建器配置
func (s *RunService) loadBuilderConfig() {
	var sysConfig models.SystemConfig
	if err := s.db.Where("`key` = ?", builderConfigKey).First(&sysConfig).Error; err == nil {
		var cfg BuilderConfig
		if json.Unmarshal([]byte(sysConfig.Value), &cfg) == nil {
			s.engine.SetBuilderConfig(&cfg)
			if cfg.IdleTimeoutMinutes > 0 {
				s.engine.SetBuilderIdleTimeout(time.Duration(cfg.IdleTimeoutMinutes) * time.Minute)
			}
			logger.L().Info("已从数据库加载构建器配置")
		}
	}
}

// Run 运行流水线
func (s *RunService) Run(ctx context.Context, pipelineID uint, req *dto.RunPipelineRequest, triggerType, triggerBy string) (*dto.PipelineRunItem, error) {
	log := logger.L().WithField("pipeline_id", pipelineID)

	// 获取流水线
	var pipeline models.Pipeline
	if err := s.db.First(&pipeline, pipelineID).Error; err != nil {
		return nil, err
	}

	if pipeline.Status != "active" {
		return nil, &ValidationError{Message: "流水线已禁用"}
	}

	// 确定使用的分支
	branch := req.Branch
	if branch == "" {
		branch = pipeline.GitBranch
	}
	if branch == "" {
		branch = "main"
	}

	// 创建执行记录
	parametersJSON, _ := json.Marshal(req.Parameters)
	run := &models.PipelineRun{
		PipelineID:     pipelineID,
		PipelineName:   pipeline.Name,
		Status:         "pending",
		TriggerType:    triggerType,
		TriggerBy:      triggerBy,
		GitBranch:      branch,
		ParametersJSON: string(parametersJSON),
		CreatedAt:      time.Now(),
	}

	if err := s.db.Create(run).Error; err != nil {
		log.WithField("error", err).Error("创建执行记录失败")
		return nil, err
	}

	// 异步执行
	go s.engine.Execute(context.Background(), run.ID)

	log.WithField("run_id", run.ID).Info("流水线开始执行")

	return &dto.PipelineRunItem{
		ID:           run.ID,
		PipelineID:   run.PipelineID,
		PipelineName: run.PipelineName,
		Status:       run.Status,
		TriggerType:  run.TriggerType,
		TriggerBy:    run.TriggerBy,
		CreatedAt:    run.CreatedAt,
	}, nil
}

// Cancel 取消执行
func (s *RunService) Cancel(ctx context.Context, runID uint) error {
	return s.engine.Cancel(ctx, runID)
}

// Retry 重试执行
func (s *RunService) Retry(ctx context.Context, runID uint, fromStage string) error {
	return s.engine.Retry(ctx, runID, fromStage)
}

// ListRuns 获取执行历史
func (s *RunService) ListRuns(ctx context.Context, req *dto.PipelineRunListRequest) (*dto.PipelineRunListResponse, error) {
	var runs []models.PipelineRun
	var total int64

	query := s.db.Model(&models.PipelineRun{})

	if req.PipelineID > 0 {
		query = query.Where("pipeline_id = ?", req.PipelineID)
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

	query.Order("created_at DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&runs)

	items := make([]dto.PipelineRunItem, 0, len(runs))
	for _, r := range runs {
		items = append(items, dto.PipelineRunItem{
			ID:           r.ID,
			PipelineID:   r.PipelineID,
			PipelineName: r.PipelineName,
			Status:       r.Status,
			TriggerType:  r.TriggerType,
			TriggerBy:    r.TriggerBy,
			StartedAt:    r.StartedAt,
			FinishedAt:   r.FinishedAt,
			Duration:     r.Duration,
			CreatedAt:    r.CreatedAt,
		})
	}

	return &dto.PipelineRunListResponse{
		Total: int(total),
		Items: items,
	}, nil
}

// GetRun 获取执行详情
func (s *RunService) GetRun(ctx context.Context, runID uint) (*dto.PipelineRunDetailResponse, error) {
	var run models.PipelineRun
	if err := s.db.First(&run, runID).Error; err != nil {
		return nil, err
	}

	result := &dto.PipelineRunDetailResponse{
		ID:           run.ID,
		PipelineID:   run.PipelineID,
		PipelineName: run.PipelineName,
		Status:       run.Status,
		TriggerType:  run.TriggerType,
		TriggerBy:    run.TriggerBy,
		StartedAt:    run.StartedAt,
		FinishedAt:   run.FinishedAt,
		Duration:     run.Duration,
		CreatedAt:    run.CreatedAt,
	}

	// 解析参数
	if run.ParametersJSON != "" {
		json.Unmarshal([]byte(run.ParametersJSON), &result.Parameters)
	}

	// 获取阶段执行记录
	var stageRuns []models.StageRun
	s.db.Where("pipeline_run_id = ?", runID).Order("id").Find(&stageRuns)

	result.StageRuns = make([]dto.StageRunItem, 0, len(stageRuns))
	for _, sr := range stageRuns {
		stageItem := dto.StageRunItem{
			ID:         sr.ID,
			StageID:    sr.StageID,
			StageName:  sr.StageName,
			Status:     sr.Status,
			StartedAt:  sr.StartedAt,
			FinishedAt: sr.FinishedAt,
		}

		// 获取步骤执行记录
		var stepRuns []models.StepRun
		s.db.Where("stage_run_id = ?", sr.ID).Order("id").Find(&stepRuns)

		stageItem.StepRuns = make([]dto.StepRunItem, 0, len(stepRuns))
		for _, step := range stepRuns {
			stageItem.StepRuns = append(stageItem.StepRuns, dto.StepRunItem{
				ID:         step.ID,
				StepID:     step.StepID,
				StepName:   step.StepName,
				StepType:   step.StepType,
				Status:     step.Status,
				Logs:       step.Logs,
				ExitCode:   step.ExitCode,
				StartedAt:  step.StartedAt,
				FinishedAt: step.FinishedAt,
			})
		}

		result.StageRuns = append(result.StageRuns, stageItem)
	}

	return result, nil
}

// GetStepLogs 获取步骤日志
func (s *RunService) GetStepLogs(ctx context.Context, stepRunID uint) (*dto.StepLogsResponse, error) {
	var stepRun models.StepRun
	if err := s.db.First(&stepRun, stepRunID).Error; err != nil {
		return nil, err
	}

	return &dto.StepLogsResponse{
		StepID:   stepRun.StepID,
		StepName: stepRun.StepName,
		Logs:     stepRun.Logs,
		Status:   stepRun.Status,
	}, nil
}

// GetActiveBuilderPods 获取活跃的构建 Pod 列表
func (s *RunService) GetActiveBuilderPods() []map[string]interface{} {
	return s.engine.GetActiveBuilderPods()
}

// SetBuilderIdleTimeout 设置构建 Pod 空闲超时时间
func (s *RunService) SetBuilderIdleTimeout(timeout time.Duration) {
	s.engine.SetBuilderIdleTimeout(timeout)
}

// BuilderConfig 构建器配置（别名）
type BuilderConfig = BuilderPodConfig

// GetBuilderConfig 获取构建器配置
func (s *RunService) GetBuilderConfig() *BuilderConfig {
	// 先从数据库加载
	var sysConfig models.SystemConfig
	if err := s.db.Where("`key` = ?", builderConfigKey).First(&sysConfig).Error; err == nil {
		var cfg BuilderConfig
		if json.Unmarshal([]byte(sysConfig.Value), &cfg) == nil {
			// 同步到内存
			s.engine.SetBuilderConfig(&cfg)
			return &cfg
		}
	}
	return s.engine.GetBuilderConfig()
}

// SetBuilderConfig 设置构建器配置
func (s *RunService) SetBuilderConfig(cfg *BuilderConfig) {
	// 保存到内存
	s.engine.SetBuilderConfig(cfg)

	// 持久化到数据库
	cfgJSON, _ := json.Marshal(cfg)
	var sysConfig models.SystemConfig
	err := s.db.Where("`key` = ?", builderConfigKey).First(&sysConfig).Error
	if err != nil {
		// 不存在，创建
		sysConfig = models.SystemConfig{
			Key:         builderConfigKey,
			Value:       string(cfgJSON),
			Description: "构建 Pod 配置",
		}
		if createErr := s.db.Create(&sysConfig).Error; createErr != nil {
			logger.L().WithError(createErr).Error("保存构建器配置失败")
		} else {
			logger.L().Info("构建器配置已保存到数据库")
		}
	} else {
		// 存在，更新
		if updateErr := s.db.Model(&sysConfig).Update("value", string(cfgJSON)).Error; updateErr != nil {
			logger.L().WithError(updateErr).Error("更新构建器配置失败")
		} else {
			logger.L().Info("构建器配置已更新")
		}
	}
}

// DeleteBuilderPod 删除指定的构建 Pod
func (s *RunService) DeleteBuilderPod(ctx context.Context, clusterID uint, namespace, podName string) error {
	return s.engine.DeleteBuilderPod(ctx, clusterID, namespace, podName)
}

// GetStats 获取流水线执行统计
func (s *RunService) GetStats(ctx context.Context, req *dto.PipelineStatsRequest) (*dto.PipelineStatsResponse, error) {
	log := logger.L().WithField("method", "GetStats")

	// 解析日期范围
	startDate, err := time.Parse("2006-01-02", req.StartDate)
	if err != nil {
		startDate = time.Now().AddDate(0, 0, -7) // 默认最近7天
	}
	endDate, err := time.Parse("2006-01-02", req.EndDate)
	if err != nil {
		endDate = time.Now()
	}
	endDate = endDate.Add(24*time.Hour - time.Second) // 包含当天

	log.WithField("start", startDate).WithField("end", endDate).Info("获取流水线统计")

	result := &dto.PipelineStatsResponse{
		StatusDistribution: make(map[string]int),
	}

	// 1. 概览统计
	var total, success, failed int64
	s.db.Model(&models.PipelineRun{}).
		Where("created_at BETWEEN ? AND ?", startDate, endDate).
		Count(&total)
	s.db.Model(&models.PipelineRun{}).
		Where("created_at BETWEEN ? AND ? AND status = ?", startDate, endDate, "success").
		Count(&success)
	s.db.Model(&models.PipelineRun{}).
		Where("created_at BETWEEN ? AND ? AND status = ?", startDate, endDate, "failed").
		Count(&failed)

	successRate := float64(0)
	if total > 0 {
		successRate = float64(success) / float64(total) * 100
	}
	result.Overview = dto.PipelineStatsOverview{
		Total:       int(total),
		Success:     int(success),
		Failed:      int(failed),
		SuccessRate: successRate,
	}

	// 2. 状态分布
	type StatusCount struct {
		Status string
		Count  int
	}
	var statusCounts []StatusCount
	s.db.Model(&models.PipelineRun{}).
		Select("status, COUNT(*) as count").
		Where("created_at BETWEEN ? AND ?", startDate, endDate).
		Group("status").
		Scan(&statusCounts)
	for _, sc := range statusCounts {
		result.StatusDistribution[sc.Status] = sc.Count
	}

	// 3. 每日趋势
	type DailyStats struct {
		Date    string
		Success int
		Failed  int
	}
	var dailyStats []DailyStats
	s.db.Model(&models.PipelineRun{}).
		Select("DATE(created_at) as date, "+
			"SUM(CASE WHEN status = 'success' THEN 1 ELSE 0 END) as success, "+
			"SUM(CASE WHEN status = 'failed' THEN 1 ELSE 0 END) as failed").
		Where("created_at BETWEEN ? AND ?", startDate, endDate).
		Group("DATE(created_at)").
		Order("date").
		Scan(&dailyStats)

	for _, ds := range dailyStats {
		result.Trend = append(result.Trend, dto.PipelineStatsTrendItem{
			Date:    ds.Date,
			Success: ds.Success,
			Failed:  ds.Failed,
		})
	}

	// 4. 平均耗时趋势
	type DailyDuration struct {
		Date        string
		AvgDuration float64
	}
	var dailyDurations []DailyDuration
	s.db.Model(&models.PipelineRun{}).
		Select("DATE(created_at) as date, AVG(duration) as avg_duration").
		Where("created_at BETWEEN ? AND ? AND duration > 0", startDate, endDate).
		Group("DATE(created_at)").
		Order("date").
		Scan(&dailyDurations)

	for _, dd := range dailyDurations {
		result.DurationTrend = append(result.DurationTrend, dto.PipelineDurationItem{
			Date:     dd.Date,
			Duration: int(dd.AvgDuration),
		})
	}

	// 5. 流水线排行
	type PipelineStats struct {
		PipelineID   uint
		PipelineName string
		Total        int
		Success      int
		AvgDuration  float64
	}
	var pipelineStats []PipelineStats
	s.db.Model(&models.PipelineRun{}).
		Select("pipeline_id, pipeline_name, COUNT(*) as total, "+
			"SUM(CASE WHEN status = 'success' THEN 1 ELSE 0 END) as success, "+
			"AVG(duration) as avg_duration").
		Where("created_at BETWEEN ? AND ?", startDate, endDate).
		Group("pipeline_id, pipeline_name").
		Order("total DESC").
		Limit(10).
		Scan(&pipelineStats)

	for _, ps := range pipelineStats {
		rate := float64(0)
		if ps.Total > 0 {
			rate = float64(ps.Success) / float64(ps.Total) * 100
		}
		avgDur := formatDuration(int(ps.AvgDuration))
		result.Rank = append(result.Rank, dto.PipelineRankItem{
			ID:          ps.PipelineID,
			Name:        ps.PipelineName,
			Total:       ps.Total,
			SuccessRate: rate,
			AvgDuration: avgDur,
		})
	}

	// 6. 最近失败的执行
	var failedRuns []models.PipelineRun
	s.db.Where("status = ? AND created_at BETWEEN ? AND ?", "failed", startDate, endDate).
		Order("created_at DESC").
		Limit(10).
		Find(&failedRuns)

	for _, run := range failedRuns {
		// 获取错误信息（从最后一个失败的步骤）
		var stepRun models.StepRun
		var errorMsg string
		if err := s.db.Joins("JOIN stage_runs ON step_runs.stage_run_id = stage_runs.id").
			Where("stage_runs.pipeline_run_id = ? AND step_runs.status = ?", run.ID, "failed").
			Order("step_runs.id DESC").
			First(&stepRun).Error; err == nil {
			if len(stepRun.Logs) > 200 {
				errorMsg = stepRun.Logs[len(stepRun.Logs)-200:]
			} else {
				errorMsg = stepRun.Logs
			}
		}

		result.RecentFailed = append(result.RecentFailed, dto.PipelineRecentFailedRun{
			ID:           run.ID,
			PipelineID:   run.PipelineID,
			PipelineName: run.PipelineName,
			RunNumber:    int(run.ID),
			Status:       run.Status,
			ErrorMessage: errorMsg,
			CreatedAt:    run.CreatedAt,
		})
	}

	return result, nil
}

// formatDuration 格式化时长
func formatDuration(seconds int) string {
	if seconds < 60 {
		return fmt.Sprintf("%ds", seconds)
	}
	m := seconds / 60
	s := seconds % 60
	if m < 60 {
		return fmt.Sprintf("%dm %ds", m, s)
	}
	h := m / 60
	m = m % 60
	return fmt.Sprintf("%dh %dm", h, m)
}
