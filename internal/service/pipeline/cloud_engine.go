package pipeline

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"gorm.io/gorm"

	"devops/internal/models"
	"devops/internal/service/pipeline/executor"
	"devops/pkg/dto"
	"devops/pkg/logger"
)

// CloudNativeEngine 云原生执行引擎
type CloudNativeEngine struct {
	db              *gorm.DB
	buildExecutor   *executor.K8sBuildExecutor
	workspaceSvc    *WorkspaceService
	logSvc          *LogService
	configParser    *ConfigParser
	gitSvc          *GitService
	legacyExecutors map[string]executor.StepExecutor
	cancelMap       sync.Map
	runningJobs     sync.Map // runID -> []buildJobID
}

// NewCloudNativeEngine 创建云原生执行引擎
func NewCloudNativeEngine(db *gorm.DB) *CloudNativeEngine {
	e := &CloudNativeEngine{
		db:              db,
		buildExecutor:   executor.NewK8sBuildExecutor(db),
		workspaceSvc:    NewWorkspaceService(db),
		logSvc:          NewLogService(db),
		configParser:    NewConfigParser(),
		gitSvc:          NewGitService(db),
		legacyExecutors: make(map[string]executor.StepExecutor),
	}

	// 注册传统执行器（用于非容器化步骤）
	e.legacyExecutors["git"] = executor.NewGitExecutor()
	e.legacyExecutors["shell"] = executor.NewShellExecutor()
	e.legacyExecutors["docker_build"] = executor.NewDockerBuildExecutor()
	e.legacyExecutors["docker_push"] = executor.NewDockerPushExecutor()
	e.legacyExecutors["k8s_deploy"] = executor.NewK8sDeployExecutor(db)
	e.legacyExecutors["notify"] = executor.NewNotifyExecutor()

	return e
}

// Execute 执行流水线
func (e *CloudNativeEngine) Execute(ctx context.Context, runID uint) error {
	log := logger.L().WithField("run_id", runID)
	log.Info("开始执行流水线（云原生模式）")

	// 创建可取消的 context
	ctx, cancel := context.WithCancel(ctx)
	e.cancelMap.Store(runID, cancel)
	defer e.cancelMap.Delete(runID)

	// 获取执行记录
	var run models.PipelineRun
	if err := e.db.First(&run, runID).Error; err != nil {
		return err
	}

	// 获取流水线配置
	var pipeline models.Pipeline
	if err := e.db.First(&pipeline, run.PipelineID).Error; err != nil {
		return err
	}

	// 解析配置
	config, err := e.parseConfig(&pipeline)
	if err != nil {
		e.updateRunStatus(&run, "failed", fmt.Sprintf("配置解析失败: %v", err))
		return err
	}

	// 更新状态为运行中
	now := time.Now()
	run.Status = "running"
	run.StartedAt = &now
	e.db.Save(&run)

	pipeline.LastRunAt = &now
	pipeline.LastRunStatus = "running"
	e.db.Save(&pipeline)

	// 构建环境变量
	env := e.buildEnv(&pipeline, &run, config)

	// 创建工作空间（如果配置了构建集群）
	var workspace *models.BuildWorkspace
	if pipeline.BuildClusterID != nil && *pipeline.BuildClusterID > 0 {
		workspace, err = e.workspaceSvc.CreateWorkspace(ctx, runID, *pipeline.BuildClusterID, pipeline.BuildNamespace, "10Gi")
		if err != nil {
			log.WithError(err).Warn("创建工作空间失败，将使用 EmptyDir")
		} else {
			run.WorkspaceID = &workspace.ID
			e.db.Save(&run)

			// 等待工作空间就绪
			if err := e.workspaceSvc.WaitForWorkspaceReady(ctx, workspace, 2*time.Minute); err != nil {
				log.WithError(err).Warn("等待工作空间就绪超时")
			}
		}
	}

	// 执行阶段
	finalStatus := e.executeStages(ctx, &run, &pipeline, config.Stages, env, workspace)

	// 清理工作空间
	if workspace != nil {
		go func() {
			time.Sleep(5 * time.Minute) // 延迟清理，便于调试
			e.workspaceSvc.CleanupWorkspace(context.Background(), workspace)
		}()
	}

	// 更新最终状态
	finishedAt := time.Now()
	run.Status = finalStatus
	run.FinishedAt = &finishedAt
	if run.StartedAt != nil {
		run.Duration = int(finishedAt.Sub(*run.StartedAt).Seconds())
	}
	e.db.Save(&run)

	pipeline.LastRunStatus = finalStatus
	e.db.Save(&pipeline)

	log.WithField("status", finalStatus).Info("流水线执行完成")
	return nil
}

// parseConfig 解析流水线配置
func (e *CloudNativeEngine) parseConfig(pipeline *models.Pipeline) (*dto.PipelineYAMLConfig, error) {
	// 尝试解析为新格式
	var config dto.PipelineYAMLConfig
	if err := json.Unmarshal([]byte(pipeline.ConfigJSON), &config); err == nil && len(config.Stages) > 0 {
		return &config, nil
	}

	// 解析为旧格式并转换
	var legacyConfig struct {
		Stages    []dto.Stage    `json:"stages"`
		Variables []dto.Variable `json:"variables"`
	}
	if err := json.Unmarshal([]byte(pipeline.ConfigJSON), &legacyConfig); err != nil {
		return nil, err
	}

	// 转换为新格式
	config.Name = pipeline.Name
	config.Variables = make(map[string]string)
	for _, v := range legacyConfig.Variables {
		config.Variables[v.Name] = v.Value
	}

	for _, stage := range legacyConfig.Stages {
		yamlStage := dto.StageYAMLConfig{
			Name:  stage.Name,
			Needs: stage.DependsOn,
			Steps: make([]dto.ContainerStepConfig, 0),
		}

		for _, step := range stage.Steps {
			containerStep := e.convertToContainerStep(&step)
			yamlStage.Steps = append(yamlStage.Steps, containerStep)
		}

		config.Stages = append(config.Stages, yamlStage)
	}

	return &config, nil
}

// convertToContainerStep 转换为容器化步骤
func (e *CloudNativeEngine) convertToContainerStep(step *dto.Step) dto.ContainerStepConfig {
	containerStep := dto.ContainerStepConfig{
		ID:         step.ID,
		Name:       step.Name,
		Timeout:    step.Timeout,
		RetryCount: step.RetryCount,
		Condition:  step.Condition,
	}

	// 根据步骤类型设置默认镜像和命令
	switch step.Type {
	case "container":
		// 已经是容器化步骤
		if img, ok := step.Config["image"].(string); ok {
			containerStep.Image = img
		}
		if cmds, ok := step.Config["commands"].([]interface{}); ok {
			for _, cmd := range cmds {
				if s, ok := cmd.(string); ok {
					containerStep.Commands = append(containerStep.Commands, s)
				}
			}
		}
		if workDir, ok := step.Config["work_dir"].(string); ok {
			containerStep.WorkDir = workDir
		}
		if envMap, ok := step.Config["env"].(map[string]interface{}); ok {
			containerStep.Env = make(map[string]string)
			for k, v := range envMap {
				if s, ok := v.(string); ok {
					containerStep.Env[k] = s
				}
			}
		}
	case "shell":
		containerStep.Image = "alpine:latest"
		if script, ok := step.Config["script"].(string); ok {
			containerStep.Commands = []string{script}
		}
	case "git":
		containerStep.Image = "alpine/git:latest"
		branch := "main"
		if b, ok := step.Config["branch"].(string); ok {
			branch = b
		}
		containerStep.Commands = []string{fmt.Sprintf("git clone --depth=1 -b %s $GIT_URL /workspace", branch)}
	case "docker_build":
		containerStep.Image = "gcr.io/kaniko-project/executor:latest"
		dockerfile := "Dockerfile"
		if df, ok := step.Config["dockerfile"].(string); ok {
			dockerfile = df
		}
		containerStep.Commands = []string{
			fmt.Sprintf("/kaniko/executor --dockerfile=%s --destination=$IMAGE_NAME --context=/workspace", dockerfile),
		}
	default:
		// 保持原有类型，使用传统执行器
		containerStep.Image = ""
	}

	if containerStep.WorkDir == "" {
		containerStep.WorkDir = "/workspace"
	}

	return containerStep
}

// buildEnv 构建环境变量
func (e *CloudNativeEngine) buildEnv(pipeline *models.Pipeline, run *models.PipelineRun, config *dto.PipelineYAMLConfig) map[string]string {
	env := make(map[string]string)

	// 添加配置中的变量
	for k, v := range config.Variables {
		env[k] = v
	}

	// 添加运行参数
	if run.ParametersJSON != "" {
		var params map[string]string
		json.Unmarshal([]byte(run.ParametersJSON), &params)
		for k, v := range params {
			env[k] = v
		}
	}

	// 添加内置变量
	builtinVars := e.configParser.GetBuiltinVariables(
		pipeline.ID,
		run.ID,
		run.GitCommit,
		run.GitBranch,
		run.GitMessage,
	)
	for k, v := range builtinVars {
		env[k] = v
	}

	// 添加 Git 仓库 URL
	if pipeline.GitRepoID != nil {
		repo, err := e.gitSvc.GetByID(context.Background(), *pipeline.GitRepoID)
		if err == nil {
			env["GIT_URL"] = repo.URL
			env["GIT_BRANCH"] = pipeline.GitBranch
		}
	}

	return env
}

// executeStages 执行所有阶段
func (e *CloudNativeEngine) executeStages(ctx context.Context, run *models.PipelineRun, pipeline *models.Pipeline, stages []dto.StageYAMLConfig, env map[string]string, workspace *models.BuildWorkspace) string {
	log := logger.L().WithField("run_id", run.ID)

	// 构建执行计划
	plan, err := e.configParser.BuildExecutionPlan(&dto.PipelineYAMLConfig{Stages: stages})
	if err != nil {
		log.WithError(err).Error("构建执行计划失败")
		return "failed"
	}

	// 阶段状态跟踪
	stageStatus := make(map[string]string)
	var finalStatus = "success"

	for _, stage := range plan.Stages {
		select {
		case <-ctx.Done():
			return "cancelled"
		default:
		}

		// 检查依赖阶段是否成功
		canRun := true
		for _, dep := range stage.DependsOn {
			if stageStatus[dep] != "success" {
				canRun = false
				break
			}
		}

		if !canRun {
			log.WithField("stage", stage.Name).Warn("依赖阶段未成功，跳过")
			stageStatus[stage.Name] = "skipped"
			continue
		}

		status := e.executeStage(ctx, run, pipeline, &stage, env, workspace)
		stageStatus[stage.Name] = status

		if status == "failed" {
			finalStatus = "failed"
			break
		} else if status == "cancelled" {
			finalStatus = "cancelled"
			break
		}
	}

	return finalStatus
}

// executeStage 执行单个阶段
func (e *CloudNativeEngine) executeStage(ctx context.Context, run *models.PipelineRun, pipeline *models.Pipeline, stage *ExecutionStage, env map[string]string, workspace *models.BuildWorkspace) string {
	log := logger.L().WithField("run_id", run.ID).WithField("stage", stage.Name)
	log.Info("开始执行阶段")

	// 创建阶段执行记录
	now := time.Now()
	stageRun := &models.StageRun{
		PipelineRunID: run.ID,
		StageID:       stage.Name,
		StageName:     stage.Name,
		Status:        "running",
		StartedAt:     &now,
		CreatedAt:     time.Now(),
	}
	e.db.Create(stageRun)

	var finalStatus = "success"

	// 执行步骤
	for _, step := range stage.Steps {
		select {
		case <-ctx.Done():
			finalStatus = "cancelled"
			break
		default:
		}

		if finalStatus != "success" {
			break
		}

		status := e.executeStep(ctx, run, pipeline, stageRun.ID, &step, env, workspace)
		if status == "failed" {
			finalStatus = "failed"
			break
		} else if status == "cancelled" {
			finalStatus = "cancelled"
			break
		}
	}

	// 更新阶段状态
	finishedAt := time.Now()
	stageRun.Status = finalStatus
	stageRun.FinishedAt = &finishedAt
	e.db.Save(stageRun)

	log.WithField("status", finalStatus).Info("阶段执行完成")
	return finalStatus
}

// executeStep 执行单个步骤
func (e *CloudNativeEngine) executeStep(ctx context.Context, run *models.PipelineRun, pipeline *models.Pipeline, stageRunID uint, step *ExecutionStep, env map[string]string, workspace *models.BuildWorkspace) string {
	log := logger.L().WithField("run_id", run.ID).WithField("step", step.Name)
	log.Info("开始执行步骤")

	// 创建步骤执行记录
	now := time.Now()
	stepRun := &models.StepRun{
		StageRunID: stageRunID,
		StepID:     step.ID,
		StepName:   step.Name,
		StepType:   "container",
		Status:     "running",
		StartedAt:  &now,
		CreatedAt:  time.Now(),
	}
	e.db.Create(stepRun)

	var status string
	var logs string

	// 判断是否使用容器化执行
	if step.Image != "" && pipeline.BuildClusterID != nil && *pipeline.BuildClusterID > 0 {
		// 容器化执行
		status, logs = e.executeContainerStep(ctx, run, pipeline, stepRun, step, env, workspace)
	} else {
		// 传统执行（本地执行器）
		status, logs = e.executeLegacyStep(ctx, stepRun, step, env)
	}

	// 更新步骤状态
	finishedAt := time.Now()
	stepRun.Status = status
	stepRun.Logs = logs
	stepRun.FinishedAt = &finishedAt
	if status == "success" {
		exitCode := 0
		stepRun.ExitCode = &exitCode
	} else if status == "failed" {
		exitCode := 1
		stepRun.ExitCode = &exitCode
	}
	e.db.Save(stepRun)

	log.WithField("status", status).Info("步骤执行完成")
	return status
}

// executeContainerStep 容器化执行步骤
func (e *CloudNativeEngine) executeContainerStep(ctx context.Context, run *models.PipelineRun, pipeline *models.Pipeline, stepRun *models.StepRun, step *ExecutionStep, env map[string]string, workspace *models.BuildWorkspace) (string, string) {
	log := logger.L().WithField("step", step.Name)

	// 合并环境变量
	stepEnv := make(map[string]string)
	for k, v := range env {
		stepEnv[k] = v
	}
	for k, v := range step.Env {
		stepEnv[k] = v
	}

	// 构建 Job 配置
	jobConfig := &dto.BuildJobConfig{
		PipelineRunID: run.ID,
		StepID:        step.ID,
		StepName:      step.Name,
		ClusterID:     *pipeline.BuildClusterID,
		Namespace:     pipeline.BuildNamespace,
		Image:         step.Image,
		Commands:      step.Commands,
		WorkDir:       step.WorkDir,
		EnvVars:       stepEnv,
		Resources:     step.Resources,
		Timeout:       step.Timeout,
	}

	// 设置 Git 信息
	if pipeline.GitRepoID != nil {
		repo, err := e.gitSvc.GetByID(ctx, *pipeline.GitRepoID)
		if err == nil {
			jobConfig.GitURL = repo.URL
			jobConfig.GitBranch = pipeline.GitBranch
			if repo.CredentialID != nil {
				jobConfig.GitCredential = repo.CredentialID
			}
		}
	}

	// 设置工作空间
	if workspace != nil {
		jobConfig.WorkspacePVC = workspace.PVCName
	}

	// 创建构建 Job
	buildJob, err := e.buildExecutor.CreateBuildJob(ctx, jobConfig)
	if err != nil {
		log.WithError(err).Error("创建构建 Job 失败")
		return "failed", fmt.Sprintf("创建构建 Job 失败: %v", err)
	}

	// 关联构建任务
	stepRun.BuildJobID = &buildJob.ID
	e.db.Save(stepRun)

	// 监控 Job 状态
	err = e.buildExecutor.WatchJobStatus(ctx, buildJob)

	// 获取日志
	logs, _ := e.buildExecutor.GetJobLogs(ctx, buildJob)

	if err != nil {
		return "failed", logs
	}

	return "success", logs
}

// executeLegacyStep 传统方式执行步骤
func (e *CloudNativeEngine) executeLegacyStep(ctx context.Context, stepRun *models.StepRun, step *ExecutionStep, env map[string]string) (string, string) {
	// 构建旧格式步骤
	legacyStep := &dto.Step{
		ID:      step.ID,
		Name:    step.Name,
		Type:    "shell",
		Timeout: step.Timeout,
		Config: map[string]interface{}{
			"script": joinCommands(step.Commands),
		},
	}

	// 获取执行器
	exec, ok := e.legacyExecutors[legacyStep.Type]
	if !ok {
		exec = e.legacyExecutors["shell"]
	}

	// 设置超时
	stepCtx := ctx
	if step.Timeout > 0 {
		var cancel context.CancelFunc
		stepCtx, cancel = context.WithTimeout(ctx, time.Duration(step.Timeout)*time.Second)
		defer cancel()
	}

	// 执行
	result, err := exec.Execute(stepCtx, legacyStep, env)

	if err != nil {
		logs := err.Error()
		if result != nil {
			logs = result.Logs + "\n" + err.Error()
		}
		return "failed", logs
	}

	if result != nil {
		return "success", result.Logs
	}

	return "success", ""
}

// Cancel 取消执行
func (e *CloudNativeEngine) Cancel(ctx context.Context, runID uint) error {
	// 取消 context
	if cancel, ok := e.cancelMap.Load(runID); ok {
		cancel.(context.CancelFunc)()
	}

	// 取消所有构建 Job
	var buildJobs []models.BuildJob
	e.db.Where("pipeline_run_id = ? AND status IN (?)", runID, []string{"pending", "running"}).Find(&buildJobs)

	for _, job := range buildJobs {
		e.buildExecutor.CancelJob(ctx, &job)
	}

	// 更新状态
	var run models.PipelineRun
	if err := e.db.First(&run, runID).Error; err != nil {
		return err
	}

	if run.Status == "running" || run.Status == "pending" {
		now := time.Now()
		run.Status = "cancelled"
		run.FinishedAt = &now
		if run.StartedAt != nil {
			run.Duration = int(now.Sub(*run.StartedAt).Seconds())
		}
		e.db.Save(&run)
	}

	return nil
}

// updateRunStatus 更新执行状态
func (e *CloudNativeEngine) updateRunStatus(run *models.PipelineRun, status, message string) {
	now := time.Now()
	run.Status = status
	run.FinishedAt = &now
	if run.StartedAt != nil {
		run.Duration = int(now.Sub(*run.StartedAt).Seconds())
	}
	e.db.Save(run)
}

// joinCommands 合并命令
func joinCommands(commands []string) string {
	if len(commands) == 0 {
		return ""
	}
	result := ""
	for i, cmd := range commands {
		if i > 0 {
			result += " && "
		}
		result += cmd
	}
	return result
}
