package pipeline

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"gorm.io/gorm"

	"devops/internal/models"
	"devops/internal/service/pipeline/executor"
	"devops/pkg/dto"
	"devops/pkg/logger"
)

// ExecutorEngine 执行引擎
type ExecutorEngine struct {
	db            *gorm.DB
	executors     map[string]executor.StepExecutor
	k8sBuildExec  *executor.K8sBuildExecutor
	builderPodMgr *BuilderPodManager
	cancelMap     sync.Map // runID -> cancel func
}

// NewExecutorEngine 创建执行引擎
func NewExecutorEngine(db *gorm.DB) *ExecutorEngine {
	e := &ExecutorEngine{
		db:            db,
		executors:     make(map[string]executor.StepExecutor),
		k8sBuildExec:  executor.NewK8sBuildExecutor(db),
		builderPodMgr: NewBuilderPodManager(db),
	}

	// 注册执行器（本地模式）
	e.executors["git"] = executor.NewGitExecutor()
	e.executors["shell"] = executor.NewShellExecutor()
	e.executors["docker_build"] = executor.NewDockerBuildExecutor()
	e.executors["docker_push"] = executor.NewDockerPushExecutor()
	e.executors["k8s_deploy"] = executor.NewK8sDeployExecutor(db)
	e.executors["notify"] = executor.NewNotifyExecutor()
	e.executors["container"] = executor.NewContainerExecutor()

	return e
}

// SetBuilderIdleTimeout 设置构建 Pod 空闲超时时间
func (e *ExecutorEngine) SetBuilderIdleTimeout(timeout time.Duration) {
	e.builderPodMgr.SetIdleTimeout(timeout)
}

// GetActiveBuilderPods 获取活跃的构建 Pod 列表
func (e *ExecutorEngine) GetActiveBuilderPods() []map[string]interface{} {
	return e.builderPodMgr.GetActivePods()
}

// StopBuilderPodManager 停止构建 Pod 管理器
func (e *ExecutorEngine) StopBuilderPodManager() {
	e.builderPodMgr.Stop()
}

// GetBuilderConfig 获取构建器配置
func (e *ExecutorEngine) GetBuilderConfig() *BuilderPodConfig {
	return e.builderPodMgr.GetConfig()
}

// SetBuilderConfig 设置构建器配置
func (e *ExecutorEngine) SetBuilderConfig(cfg *BuilderPodConfig) {
	e.builderPodMgr.SetConfig(cfg)
}

// Execute 执行流水线
func (e *ExecutorEngine) Execute(ctx context.Context, runID uint) error {
	log := logger.L().WithField("run_id", runID)
	log.Info("开始执行流水线")

	// 创建可取消的context
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
	var config struct {
		Stages    []dto.Stage    `json:"stages"`
		Variables []dto.Variable `json:"variables"`
	}
	if err := json.Unmarshal([]byte(pipeline.ConfigJSON), &config); err != nil {
		return err
	}

	// 更新状态为运行中
	now := time.Now()
	run.Status = "running"
	run.StartedAt = &now
	e.db.Save(&run)

	// 更新流水线最后执行时间
	pipeline.LastRunAt = &now
	pipeline.LastRunStatus = "running"
	e.db.Save(&pipeline)

	// 构建环境变量
	env := make(map[string]string)
	for _, v := range config.Variables {
		env[v.Name] = v.Value
	}

	// 解析运行参数
	var params map[string]string
	if run.ParametersJSON != "" {
		json.Unmarshal([]byte(run.ParametersJSON), &params)
		for k, v := range params {
			env[k] = v
		}
	}

	// 添加构建集群配置到环境变量（内部使用）
	if pipeline.BuildClusterID != nil && *pipeline.BuildClusterID > 0 {
		env["__BUILD_CLUSTER_ID__"] = fmt.Sprintf("%d", *pipeline.BuildClusterID)
		env["__BUILD_NAMESPACE__"] = pipeline.BuildNamespace
		if env["__BUILD_NAMESPACE__"] == "" {
			env["__BUILD_NAMESPACE__"] = "devops-build"
		}
		log.WithField("cluster_id", *pipeline.BuildClusterID).WithField("namespace", pipeline.BuildNamespace).Info("使用 K8s 集群执行构建")
	}

	// 添加 Git 仓库信息
	if pipeline.GitRepoID != nil && *pipeline.GitRepoID > 0 {
		var gitRepo models.GitRepository
		if err := e.db.First(&gitRepo, *pipeline.GitRepoID).Error; err == nil {
			env["__GIT_REPO_URL__"] = gitRepo.URL
			// 优先使用执行时指定的分支，其次是流水线配置的分支，最后是仓库默认分支
			if run.GitBranch != "" {
				env["__GIT_BRANCH__"] = run.GitBranch
			} else if pipeline.GitBranch != "" {
				env["__GIT_BRANCH__"] = pipeline.GitBranch
			} else {
				env["__GIT_BRANCH__"] = gitRepo.DefaultBranch
			}
		}
	}

	// 添加流水线名称（用于工作目录隔离）
	env["__PIPELINE_NAME__"] = pipeline.Name
	env["__PIPELINE_ID__"] = fmt.Sprintf("%d", pipeline.ID)
	env["__RUN_ID__"] = fmt.Sprintf("%d", runID)

	// 执行阶段
	var finalStatus = "success"
stageLoop:
	for _, stage := range config.Stages {
		select {
		case <-ctx.Done():
			finalStatus = "cancelled"
			break stageLoop
		default:
		}

		status := e.executeStage(ctx, runID, stage, env)
		if status == "failed" {
			finalStatus = "failed"
			break stageLoop
		} else if status == "cancelled" {
			finalStatus = "cancelled"
			break stageLoop
		}
	}

	// 更新最终状态
	finishedAt := time.Now()
	run.Status = finalStatus
	run.FinishedAt = &finishedAt
	if run.StartedAt != nil {
		run.Duration = int(finishedAt.Sub(*run.StartedAt).Seconds())
	}
	e.db.Save(&run)

	// 更新流水线状态
	pipeline.LastRunStatus = finalStatus
	e.db.Save(&pipeline)

	log.WithField("status", finalStatus).Info("流水线执行完成")
	return nil
}

// executeStage 执行阶段
func (e *ExecutorEngine) executeStage(ctx context.Context, runID uint, stage dto.Stage, env map[string]string) string {
	log := logger.L().WithField("run_id", runID).WithField("stage", stage.Name)
	log.Info("开始执行阶段")

	// 创建阶段执行记录
	now := time.Now()
	stageRun := &models.StageRun{
		PipelineRunID: runID,
		StageID:       stage.ID,
		StageName:     stage.Name,
		Status:        "running",
		StartedAt:     &now,
		CreatedAt:     time.Now(),
	}
	e.db.Create(stageRun)

	var finalStatus = "success"

	if stage.Parallel {
		// 并行执行步骤
		var wg sync.WaitGroup
		var mu sync.Mutex
		for _, step := range stage.Steps {
			wg.Add(1)
			go func(s dto.Step) {
				defer wg.Done()
				status := e.executeStep(ctx, stageRun.ID, s, env)
				mu.Lock()
				if status == "failed" && finalStatus == "success" {
					finalStatus = "failed"
				}
				mu.Unlock()
			}(step)
		}
		wg.Wait()
	} else {
		// 串行执行步骤
	stepLoop:
		for _, step := range stage.Steps {
			select {
			case <-ctx.Done():
				finalStatus = "cancelled"
				break stepLoop
			default:
			}

			status := e.executeStep(ctx, stageRun.ID, step, env)
			if status == "failed" {
				finalStatus = "failed"
				break stepLoop
			}
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

// executeStep 执行步骤
func (e *ExecutorEngine) executeStep(ctx context.Context, stageRunID uint, step dto.Step, env map[string]string) string {
	log := logger.L().WithField("stage_run_id", stageRunID).WithField("step", step.Name)
	log.Info("开始执行步骤")

	// 创建步骤执行记录
	now := time.Now()
	stepRun := &models.StepRun{
		StageRunID: stageRunID,
		StepID:     step.ID,
		StepName:   step.Name,
		StepType:   step.Type,
		Status:     "running",
		StartedAt:  &now,
		CreatedAt:  time.Now(),
	}
	e.db.Create(stepRun)

	// 设置超时
	stepCtx := ctx
	if step.Timeout > 0 {
		var cancel context.CancelFunc
		stepCtx, cancel = context.WithTimeout(ctx, time.Duration(step.Timeout)*time.Second)
		defer cancel()
	}

	var result *executor.StepResult
	var err error

	// 检查是否配置了 K8s 集群（从环境变量获取）
	clusterIDStr := env["__BUILD_CLUSTER_ID__"]
	namespace := env["__BUILD_NAMESPACE__"]

	if clusterIDStr != "" && namespace != "" && step.Type == "container" {
		// 使用 K8s Job 执行
		result, err = e.executeStepInK8s(stepCtx, stepRun, step, env, clusterIDStr, namespace)
	} else {
		// 本地执行
		exec, ok := e.executors[step.Type]
		if !ok {
			stepRun.Status = "failed"
			stepRun.Logs = "未知的步骤类型: " + step.Type
			finishedAt := time.Now()
			stepRun.FinishedAt = &finishedAt
			e.db.Save(stepRun)
			return "failed"
		}
		result, err = exec.Execute(stepCtx, &step, env)
	}

	finishedAt := time.Now()
	stepRun.FinishedAt = &finishedAt

	if err != nil {
		stepRun.Status = "failed"
		stepRun.Logs = err.Error()
		if result != nil {
			stepRun.Logs = result.Logs + "\n" + err.Error()
			stepRun.ExitCode = &result.ExitCode
		}
	} else {
		stepRun.Status = "success"
		if result != nil {
			stepRun.Logs = result.Logs
			stepRun.ExitCode = &result.ExitCode
		}
	}

	e.db.Save(stepRun)

	log.WithField("status", stepRun.Status).Info("步骤执行完成")
	return stepRun.Status
}

// executeStepInK8s 在 K8s 中执行步骤（使用持久化 Builder Pod）
func (e *ExecutorEngine) executeStepInK8s(ctx context.Context, _ *models.StepRun, step dto.Step, env map[string]string, clusterIDStr, namespace string) (*executor.StepResult, error) {
	log := logger.L().WithField("step", step.Name).WithField("namespace", namespace)
	log.Info("在 K8s 集群中执行步骤（Builder Pod 模式）")

	// 解析集群 ID
	var clusterID uint
	fmt.Sscanf(clusterIDStr, "%d", &clusterID)

	// 构建命令
	commands := make([]string, 0)
	if cmds, ok := step.Config["commands"].([]interface{}); ok {
		for _, cmd := range cmds {
			if cmdStr, ok := cmd.(string); ok {
				commands = append(commands, cmdStr)
			}
		}
	}

	// 获取镜像
	image, _ := step.Config["image"].(string)
	if image == "" {
		image = "alpine:latest"
	}

	// 获取流水线名称，构建隔离的工作目录
	pipelineName := env["__PIPELINE_NAME__"]
	if pipelineName == "" {
		pipelineName = "default"
	}
	// 基础工作目录：/workspace/{pipeline_name}
	baseWorkDir := fmt.Sprintf("/workspace/%s", pipelineName)

	// 获取用户配置的工作目录（相对于基础目录）
	userWorkDir, _ := step.Config["work_dir"].(string)
	var workDir string
	if userWorkDir == "" || userWorkDir == "/" || userWorkDir == "/workspace" {
		workDir = baseWorkDir
	} else if strings.HasPrefix(userWorkDir, "/") {
		// 绝对路径，追加到基础目录
		workDir = baseWorkDir + userWorkDir
	} else {
		// 相对路径
		workDir = baseWorkDir + "/" + userWorkDir
	}

	// 构建环境变量（排除内部变量）
	envVars := make(map[string]string)
	for k, v := range env {
		if !strings.HasPrefix(k, "__") {
			envVars[k] = v
		}
	}

	// 添加步骤配置中的环境变量
	if stepEnv, ok := step.Config["env"].(map[string]interface{}); ok {
		for k, v := range stepEnv {
			envVars[k] = fmt.Sprintf("%v", v)
		}
	}

	// 添加内置环境变量
	envVars["CI"] = "true"
	envVars["CI_STEP_ID"] = step.ID
	envVars["CI_STEP_NAME"] = step.Name
	envVars["CI_WORKSPACE"] = baseWorkDir

	// 获取或创建 Builder Pod
	builderPod, err := e.builderPodMgr.GetOrCreatePod(ctx, clusterID, namespace, image)
	if err != nil {
		return &executor.StepResult{
			Logs:     fmt.Sprintf("获取/创建 Builder Pod 失败: %v", err),
			ExitCode: 1,
		}, err
	}

	log.WithField("pod_name", builderPod.PodName).Info("使用 Builder Pod 执行命令")

	// 如果配置了 Git 仓库，检查是否需要克隆代码
	gitURL := env["__GIT_REPO_URL__"]
	gitBranch := env["__GIT_BRANCH__"]
	if gitURL != "" && gitBranch != "" {
		// 使用 alpine/git 镜像的 Pod 来检查和克隆代码（确保有 git 命令且能访问共享 PVC）
		gitPod, gitErr := e.builderPodMgr.GetOrCreatePod(ctx, clusterID, namespace, "alpine/git:latest")
		if gitErr != nil {
			return &executor.StepResult{
				Logs:     fmt.Sprintf("获取 Git Pod 失败: %v", gitErr),
				ExitCode: 1,
			}, gitErr
		}

		// 检查基础工作目录是否已有代码
		checkCmd := []string{
			fmt.Sprintf("if [ -d '%s/.git' ]; then echo 'HAS_CODE'; else echo 'NEED_CLONE'; fi", baseWorkDir),
		}
		checkResult, _, _ := e.builderPodMgr.ExecInPod(ctx, gitPod, checkCmd, "/", nil)

		if strings.Contains(checkResult, "NEED_CLONE") {
			log.Info("工作目录为空或无 .git，开始克隆代码")

			// 先验证分支/tag 是否存在（设置 60 秒超时）
			gitCtx, gitCancel := context.WithTimeout(ctx, 60*time.Second)
			defer gitCancel()

			// 使用 git ls-remote 验证分支/tag 是否存在
			verifyCmd := []string{
				fmt.Sprintf("git ls-remote --exit-code --heads --tags '%s' '%s' || git ls-remote --exit-code --heads --tags '%s' 'refs/heads/%s' || git ls-remote --exit-code --heads --tags '%s' 'refs/tags/%s'",
					gitURL, gitBranch, gitURL, gitBranch, gitURL, gitBranch),
			}
			verifyLogs, verifyCode, verifyErr := e.builderPodMgr.ExecInPod(gitCtx, gitPod, verifyCmd, "/", nil)
			if verifyErr != nil || verifyCode != 0 {
				return &executor.StepResult{
					Logs:     fmt.Sprintf("Git 分支/Tag '%s' 不存在或无法访问:\n%s", gitBranch, verifyLogs),
					ExitCode: 1,
				}, fmt.Errorf("git ref '%s' not found", gitBranch)
			}

			// 克隆代码（设置 5 分钟超时）
			cloneCtx, cloneCancel := context.WithTimeout(ctx, 5*time.Minute)
			defer cloneCancel()

			gitCommands := []string{
				fmt.Sprintf("rm -rf '%s' 2>/dev/null || true", baseWorkDir),
				fmt.Sprintf("mkdir -p '%s'", baseWorkDir),
				fmt.Sprintf("git clone --depth=1 -b '%s' '%s' '%s'", gitBranch, gitURL, baseWorkDir),
			}
			gitLogs, exitCode, err := e.builderPodMgr.ExecInPod(cloneCtx, gitPod, gitCommands, "/", nil)
			if err != nil || exitCode != 0 {
				errMsg := "git clone failed"
				if cloneCtx.Err() == context.DeadlineExceeded {
					errMsg = "git clone timeout (5 minutes)"
				}
				return &executor.StepResult{
					Logs:     fmt.Sprintf("Git 克隆失败:\n%s\n%v", gitLogs, err),
					ExitCode: exitCode,
				}, fmt.Errorf("%s", errMsg)
			}
			log.Info("Git 代码克隆完成")
		} else {
			log.Info("工作目录已有代码，跳过克隆")
		}
	}

	// 确保工作目录存在
	mkdirCmd := []string{fmt.Sprintf("mkdir -p %s", workDir)}
	e.builderPodMgr.ExecInPod(ctx, builderPod, mkdirCmd, "/", nil)

	// 执行构建命令
	logs, exitCode, err := e.builderPodMgr.ExecInPod(ctx, builderPod, commands, workDir, envVars)

	// 构建日志
	var logBuilder strings.Builder
	fmt.Fprintf(&logBuilder, "=== 步骤: %s ===\n", step.Name)
	fmt.Fprintf(&logBuilder, "镜像: %s\n", image)
	fmt.Fprintf(&logBuilder, "Pod: %s\n", builderPod.PodName)
	fmt.Fprintf(&logBuilder, "工作目录: %s\n", workDir)
	logBuilder.WriteString("---\n")
	logBuilder.WriteString(logs)
	logBuilder.WriteString("\n=== 步骤执行完成 ===\n")

	if err != nil {
		return &executor.StepResult{
			Logs:     logBuilder.String(),
			ExitCode: exitCode,
		}, err
	}

	return &executor.StepResult{
		Logs:     logBuilder.String(),
		ExitCode: exitCode,
	}, nil
}

// Cancel 取消执行
func (e *ExecutorEngine) Cancel(ctx context.Context, runID uint) error {
	if cancel, ok := e.cancelMap.Load(runID); ok {
		cancel.(context.CancelFunc)()
	}

	// 检查 db 是否为 nil
	if e.db == nil {
		return fmt.Errorf("数据库连接未初始化")
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

// Retry 重试执行
func (e *ExecutorEngine) Retry(ctx context.Context, runID uint, fromStage string) error {
	// 获取原执行记录
	var run models.PipelineRun
	if err := e.db.First(&run, runID).Error; err != nil {
		return err
	}

	// 创建新的执行记录
	newRun := &models.PipelineRun{
		PipelineID:     run.PipelineID,
		PipelineName:   run.PipelineName,
		Status:         "pending",
		TriggerType:    "retry",
		TriggerBy:      run.TriggerBy,
		ParametersJSON: run.ParametersJSON,
		CreatedAt:      time.Now(),
	}

	if err := e.db.Create(newRun).Error; err != nil {
		return err
	}

	// 异步执行（使用传入的 context，但创建新的可取消 context）
	ctx, cancel := context.WithCancel(ctx)
	go func() {
		defer cancel()
		_ = e.Execute(ctx, newRun.ID)
	}()

	return nil
}

// DeleteBuilderPod 删除指定的构建 Pod
func (e *ExecutorEngine) DeleteBuilderPod(ctx context.Context, clusterID uint, namespace, podName string) error {
	return e.builderPodMgr.DeletePodByName(ctx, clusterID, namespace, podName)
}
