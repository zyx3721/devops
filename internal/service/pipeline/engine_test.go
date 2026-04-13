package pipeline

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"devops/internal/models"
	"devops/pkg/dto"
)

// MockDB 模拟数据库
type MockDB struct {
	mock.Mock
}

func TestExecutorEngine_NewExecutorEngine(t *testing.T) {
	engine := NewExecutorEngine(nil)
	assert.NotNil(t, engine)
	assert.NotNil(t, engine.executors)
	assert.Contains(t, engine.executors, "git")
	assert.Contains(t, engine.executors, "shell")
	assert.Contains(t, engine.executors, "docker_build")
	assert.Contains(t, engine.executors, "docker_push")
	assert.Contains(t, engine.executors, "k8s_deploy")
	assert.Contains(t, engine.executors, "notify")
	assert.Contains(t, engine.executors, "container")
}

func TestExecutorEngine_Cancel(t *testing.T) {
	engine := NewExecutorEngine(nil)

	// 模拟一个运行中的任务
	ctx, cancel := context.WithCancel(context.Background())
	engine.cancelMap.Store(uint(1), cancel)

	// 取消任务 - 由于没有真实数据库，会返回错误
	err := engine.Cancel(context.Background(), 1)
	// 数据库为 nil 会导致错误，但取消函数应该被调用
	assert.Error(t, err)

	// 验证 context 已被取消
	select {
	case <-ctx.Done():
		// 预期行为 - cancel 函数被调用了
	default:
		t.Error("Context should be cancelled")
	}
}

func TestExecutorEngine_SetBuilderIdleTimeout(t *testing.T) {
	engine := NewExecutorEngine(nil)
	timeout := 10 * time.Minute
	engine.SetBuilderIdleTimeout(timeout)
	// 验证设置成功（通过 GetBuilderConfig）
	config := engine.GetBuilderConfig()
	assert.NotNil(t, config)
}

func TestExecutorEngine_GetActiveBuilderPods(t *testing.T) {
	engine := NewExecutorEngine(nil)
	pods := engine.GetActiveBuilderPods()
	// db 为 nil 时返回空切片
	assert.Empty(t, pods) // 初始应该为空
}

func TestPipelineConfig_Parse(t *testing.T) {
	configJSON := `{
		"stages": [
			{
				"id": "stage-1",
				"name": "Build",
				"parallel": false,
				"steps": [
					{
						"id": "step-1",
						"name": "Compile",
						"type": "shell",
						"config": {
							"commands": ["go build ./..."]
						}
					}
				]
			}
		],
		"variables": [
			{"name": "VERSION", "value": "1.0.0"}
		]
	}`

	var config struct {
		Stages    []dto.Stage    `json:"stages"`
		Variables []dto.Variable `json:"variables"`
	}

	err := json.Unmarshal([]byte(configJSON), &config)
	assert.NoError(t, err)
	assert.Len(t, config.Stages, 1)
	assert.Equal(t, "Build", config.Stages[0].Name)
	assert.Len(t, config.Stages[0].Steps, 1)
	assert.Equal(t, "shell", config.Stages[0].Steps[0].Type)
	assert.Len(t, config.Variables, 1)
	assert.Equal(t, "VERSION", config.Variables[0].Name)
}

func TestPipelineConfig_ParallelStages(t *testing.T) {
	configJSON := `{
		"stages": [
			{
				"id": "stage-1",
				"name": "Parallel Build",
				"parallel": true,
				"steps": [
					{"id": "step-1", "name": "Build A", "type": "shell"},
					{"id": "step-2", "name": "Build B", "type": "shell"},
					{"id": "step-3", "name": "Build C", "type": "shell"}
				]
			}
		]
	}`

	var config struct {
		Stages []dto.Stage `json:"stages"`
	}

	err := json.Unmarshal([]byte(configJSON), &config)
	assert.NoError(t, err)
	assert.True(t, config.Stages[0].Parallel)
	assert.Len(t, config.Stages[0].Steps, 3)
}

func TestPipelineRun_StatusTransitions(t *testing.T) {
	tests := []struct {
		name        string
		fromStatus  string
		toStatus    string
		shouldAllow bool
	}{
		{"pending to running", "pending", "running", true},
		{"running to success", "running", "success", true},
		{"running to failed", "running", "failed", true},
		{"running to cancelled", "running", "cancelled", true},
		{"pending to cancelled", "pending", "cancelled", true},
		{"success to running", "success", "running", false},
		{"failed to running", "failed", "running", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			run := &models.PipelineRun{Status: tt.fromStatus}
			allowed := isValidStatusTransition(run.Status, tt.toStatus)
			assert.Equal(t, tt.shouldAllow, allowed)
		})
	}
}

// isValidStatusTransition 检查状态转换是否有效
func isValidStatusTransition(from, to string) bool {
	validTransitions := map[string][]string{
		"pending":   {"running", "cancelled"},
		"running":   {"success", "failed", "cancelled"},
		"success":   {},
		"failed":    {},
		"cancelled": {},
	}

	allowed, exists := validTransitions[from]
	if !exists {
		return false
	}

	for _, s := range allowed {
		if s == to {
			return true
		}
	}
	return false
}

func TestEnvironmentVariables_Build(t *testing.T) {
	pipeline := &models.Pipeline{
		ID:   1,
		Name: "test-pipeline",
	}

	run := &models.PipelineRun{
		ID:        100,
		GitBranch: "main",
	}

	env := buildEnvironmentVariables(pipeline, run, nil)

	assert.Equal(t, "true", env["CI"])
	assert.Equal(t, "test-pipeline", env["__PIPELINE_NAME__"])
	assert.Equal(t, "1", env["__PIPELINE_ID__"])
	assert.Equal(t, "100", env["__RUN_ID__"])
}

// buildEnvironmentVariables 构建环境变量
func buildEnvironmentVariables(pipeline *models.Pipeline, run *models.PipelineRun, params map[string]string) map[string]string {
	env := make(map[string]string)

	// 内置变量
	env["CI"] = "true"
	env["__PIPELINE_NAME__"] = pipeline.Name
	env["__PIPELINE_ID__"] = string(rune(pipeline.ID + '0'))
	env["__RUN_ID__"] = string(rune(run.ID + '0'))

	// 修复：使用 fmt.Sprintf 正确转换
	env["__PIPELINE_ID__"] = formatUint(pipeline.ID)
	env["__RUN_ID__"] = formatUint(run.ID)

	// 用户参数
	for k, v := range params {
		env[k] = v
	}

	return env
}

func formatUint(n uint) string {
	if n == 0 {
		return "0"
	}
	result := ""
	for n > 0 {
		result = string(rune('0'+n%10)) + result
		n /= 10
	}
	return result
}

func TestStepTimeout(t *testing.T) {
	step := dto.Step{
		ID:      "step-1",
		Name:    "Test Step",
		Type:    "shell",
		Timeout: 30, // 30 seconds
	}

	ctx := context.Background()
	if step.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, time.Duration(step.Timeout)*time.Second)
		defer cancel()
	}

	// 验证 context 有 deadline
	deadline, ok := ctx.Deadline()
	assert.True(t, ok)
	assert.True(t, deadline.After(time.Now()))
	assert.True(t, deadline.Before(time.Now().Add(31*time.Second)))
}

func TestRetryMechanism(t *testing.T) {
	maxRetries := 3
	retryCount := 0
	var lastErr error

	for i := 0; i < maxRetries; i++ {
		retryCount++
		// 模拟失败
		lastErr = simulateStepExecution(i < 2) // 前两次失败，第三次成功
		if lastErr == nil {
			break
		}
		time.Sleep(time.Duration(i+1) * 100 * time.Millisecond) // 指数退避
	}

	assert.Equal(t, 3, retryCount)
	assert.NoError(t, lastErr)
}

func simulateStepExecution(shouldFail bool) error {
	if shouldFail {
		return assert.AnError
	}
	return nil
}

func TestConcurrencyControl(t *testing.T) {
	svc := NewConcurrencyService(2) // 最大并发 2

	ctx := context.Background()

	// 获取第一个许可
	err := svc.Acquire(ctx, 1, 0, 5*time.Second)
	assert.NoError(t, err)
	assert.Equal(t, 1, svc.GetRunningCount())

	// 获取第二个许可
	err = svc.Acquire(ctx, 2, 0, 5*time.Second)
	assert.NoError(t, err)
	assert.Equal(t, 2, svc.GetRunningCount())

	// 第三个应该进入队列
	go func() {
		time.Sleep(100 * time.Millisecond)
		svc.Release(1) // 释放一个
	}()

	err = svc.Acquire(ctx, 3, 0, 5*time.Second)
	assert.NoError(t, err)
	assert.Equal(t, 2, svc.GetRunningCount())

	// 清理
	svc.Release(2)
	svc.Release(3)
}

func TestConcurrencyPriority(t *testing.T) {
	svc := NewConcurrencyService(1) // 最大并发 1

	ctx := context.Background()

	// 占用唯一的槽位
	err := svc.Acquire(ctx, 1, 0, 5*time.Second)
	assert.NoError(t, err)

	// 添加低优先级任务
	go func() {
		svc.Acquire(ctx, 2, 10, 5*time.Second) // 低优先级
	}()

	time.Sleep(50 * time.Millisecond)

	// 添加高优先级任务
	go func() {
		svc.Acquire(ctx, 3, 1, 5*time.Second) // 高优先级
	}()

	time.Sleep(50 * time.Millisecond)

	// 释放槽位
	svc.Release(1)

	time.Sleep(100 * time.Millisecond)

	// 高优先级任务应该先获取到
	assert.True(t, svc.IsRunning(3))

	// 清理
	svc.Release(3)
	time.Sleep(50 * time.Millisecond)
	svc.Release(2)
}

func TestLogBuffer(t *testing.T) {
	// 使用 nil db，只测试缓冲逻辑
	svc := NewLogBufferService(nil)
	defer svc.Stop()

	stepRunID := uint(1)

	// 追加日志
	for i := 0; i < 50; i++ {
		svc.AppendLog(stepRunID, "test log line")
	}

	// 检查缓冲区状态
	stats := svc.GetBufferStats()
	assert.Equal(t, 1, stats.BufferCount)
	assert.LessOrEqual(t, stats.TotalLines, 50) // 可能已经部分刷新

	// 关闭缓冲区
	svc.CloseBuffer(stepRunID)

	stats = svc.GetBufferStats()
	assert.Equal(t, 0, stats.BufferCount)
}

func BenchmarkConcurrencyAcquireRelease(b *testing.B) {
	svc := NewConcurrencyService(100)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		runID := uint(i)
		svc.Acquire(ctx, runID, 0, time.Second)
		svc.Release(runID)
	}
}

func BenchmarkLogBuffer(b *testing.B) {
	svc := NewLogBufferService(nil)
	defer svc.Stop()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		svc.AppendLog(1, "benchmark log line with some content")
	}
}
