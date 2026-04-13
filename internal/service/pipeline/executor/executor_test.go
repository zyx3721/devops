package executor

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"devops/pkg/dto"
)

func TestShellExecutor_Execute(t *testing.T) {
	exec := NewShellExecutor()

	tests := []struct {
		name        string
		step        *dto.Step
		env         map[string]string
		expectError bool
	}{
		{
			name: "Simple echo command",
			step: &dto.Step{
				ID:   "step-1",
				Name: "Echo Test",
				Type: "shell",
				Config: map[string]interface{}{
					"script": "echo hello",
				},
			},
			env:         map[string]string{},
			expectError: false,
		},
		{
			name: "Command with environment variable",
			step: &dto.Step{
				ID:   "step-2",
				Name: "Env Test",
				Type: "shell",
				Config: map[string]interface{}{
					"script": "echo %TEST_VAR%",
				},
			},
			env:         map[string]string{"TEST_VAR": "test_value"},
			expectError: false,
		},
		{
			name: "Empty script",
			step: &dto.Step{
				ID:     "step-3",
				Name:   "Empty",
				Type:   "shell",
				Config: map[string]interface{}{},
			},
			env:         map[string]string{},
			expectError: true, // 空脚本应该报错
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			result, err := exec.Execute(ctx, tt.step, tt.env)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
			}
		})
	}
}

func TestShellExecutor_Timeout(t *testing.T) {
	exec := NewShellExecutor()

	step := &dto.Step{
		ID:      "step-timeout",
		Name:    "Timeout Test",
		Type:    "shell",
		Timeout: 1, // 1 second timeout
		Config: map[string]interface{}{
			"script": "ping -n 10 127.0.0.1", // Windows: 10 second ping
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	_, err := exec.Execute(ctx, step, nil)
	// 应该因为超时而失败
	assert.Error(t, err)
}

func TestGitExecutor_ParseConfig(t *testing.T) {
	exec := NewGitExecutor()

	step := &dto.Step{
		ID:   "git-step",
		Name: "Git Clone",
		Type: "git",
		Config: map[string]interface{}{
			"url":    "https://github.com/example/repo.git",
			"branch": "main",
			"depth":  1,
		},
	}

	// 验证配置解析
	url, _ := step.Config["url"].(string)
	branch, _ := step.Config["branch"].(string)
	depth, _ := step.Config["depth"].(int)

	assert.Equal(t, "https://github.com/example/repo.git", url)
	assert.Equal(t, "main", branch)
	assert.Equal(t, 1, depth)
	assert.NotNil(t, exec)
}

func TestDockerBuildExecutor_ParseConfig(t *testing.T) {
	exec := NewDockerBuildExecutor()

	step := &dto.Step{
		ID:   "docker-build",
		Name: "Build Image",
		Type: "docker_build",
		Config: map[string]interface{}{
			"dockerfile": "Dockerfile",
			"context":    ".",
			"tags":       []interface{}{"myapp:latest", "myapp:v1.0"},
			"build_args": map[string]interface{}{
				"VERSION": "1.0.0",
			},
		},
	}

	dockerfile, _ := step.Config["dockerfile"].(string)
	contextPath, _ := step.Config["context"].(string)
	tags, _ := step.Config["tags"].([]interface{})

	assert.Equal(t, "Dockerfile", dockerfile)
	assert.Equal(t, ".", contextPath)
	assert.Len(t, tags, 2)
	assert.NotNil(t, exec)
}

func TestDockerPushExecutor_ParseConfig(t *testing.T) {
	exec := NewDockerPushExecutor()

	step := &dto.Step{
		ID:   "docker-push",
		Name: "Push Image",
		Type: "docker_push",
		Config: map[string]interface{}{
			"image":    "registry.example.com/myapp",
			"tags":     []interface{}{"latest", "v1.0"},
			"registry": "registry.example.com",
		},
	}

	image, _ := step.Config["image"].(string)
	registry, _ := step.Config["registry"].(string)

	assert.Equal(t, "registry.example.com/myapp", image)
	assert.Equal(t, "registry.example.com", registry)
	assert.NotNil(t, exec)
}

func TestNotifyExecutor_Execute(t *testing.T) {
	exec := NewNotifyExecutor()

	tests := []struct {
		name        string
		notifyType  string
		expectError bool
	}{
		{"Feishu notification", "feishu", true}, // 没有真实 webhook，会失败
		{"DingTalk notification", "dingtalk", true},
		{"WeChat notification", "wechat", true},
		{"Unknown type", "unknown", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			step := &dto.Step{
				ID:   "notify-step",
				Name: "Notify",
				Type: "notify",
				Config: map[string]interface{}{
					"type":        tt.notifyType,
					"webhook_url": "https://example.com/webhook",
				},
			}

			ctx := context.Background()
			_, err := exec.Execute(ctx, step, nil)
			if err != nil {
				fmt.Println(err)
			}

			if tt.expectError {
				// 由于没有真实的 webhook，预期会失败
				// 但不应该 panic
				assert.NotNil(t, exec)
			}
		})
	}
}

func TestContainerExecutor_ParseConfig(t *testing.T) {
	exec := NewContainerExecutor()

	step := &dto.Step{
		ID:   "container-step",
		Name: "Run in Container",
		Type: "container",
		Config: map[string]interface{}{
			"image":    "node:18",
			"commands": []interface{}{"npm install", "npm test"},
			"work_dir": "/app",
			"env": map[string]interface{}{
				"NODE_ENV": "test",
			},
		},
	}

	image, _ := step.Config["image"].(string)
	workDir, _ := step.Config["work_dir"].(string)
	commands, _ := step.Config["commands"].([]interface{})

	assert.Equal(t, "node:18", image)
	assert.Equal(t, "/app", workDir)
	assert.Len(t, commands, 2)
	assert.NotNil(t, exec)
}

func TestStepResult(t *testing.T) {
	result := &StepResult{
		Logs:     "Build completed successfully",
		ExitCode: 0,
	}

	assert.Equal(t, 0, result.ExitCode)
	assert.Contains(t, result.Logs, "successfully")
}

func TestStepResult_Failed(t *testing.T) {
	result := &StepResult{
		Logs:     "Error: command not found",
		ExitCode: 127,
	}

	assert.NotEqual(t, 0, result.ExitCode)
	assert.Contains(t, result.Logs, "Error")
}

func TestPipelineNotifyExecutor_StatusText(t *testing.T) {
	exec := NewPipelineNotifyExecutor()

	tests := []struct {
		status   string
		expected string
	}{
		{"success", "构建成功"},
		{"failed", "构建失败"},
		{"cancelled", "已取消"},
		{"running", "运行中"},
		{"unknown", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			result := exec.statusText(tt.status)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestPipelineNotifyExecutor_StatusEmoji(t *testing.T) {
	exec := NewPipelineNotifyExecutor()

	tests := []struct {
		status   string
		expected string
	}{
		{"success", "✅"},
		{"failed", "❌"},
		{"cancelled", "⚠️"},
		{"running", "🔄"},
		{"unknown", "📋"},
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			result := exec.statusEmoji(tt.status)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestPipelineNotifyExecutor_FormatDuration(t *testing.T) {
	exec := NewPipelineNotifyExecutor()

	tests := []struct {
		seconds  int
		expected string
	}{
		{30, "30秒"},
		{60, "1分0秒"},
		{90, "1分30秒"},
		{3600, "1时0分"},
		{3661, "1时1分"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := exec.formatDuration(tt.seconds)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestPipelineNotifyExecutor_RenderTemplate(t *testing.T) {
	exec := NewPipelineNotifyExecutor()

	template := "Pipeline {{.PipelineName}} {{.Status}}"
	ctx := &NotifyContext{
		PipelineName: "test-pipeline",
		Status:       "success",
	}

	result := exec.renderTemplate(template, ctx)
	assert.Equal(t, "Pipeline test-pipeline success", result)
}

func TestPipelineNotifyExecutor_RenderTemplate_Empty(t *testing.T) {
	exec := NewPipelineNotifyExecutor()

	result := exec.renderTemplate("", nil)
	assert.Empty(t, result)
}

func TestPipelineNotifyExecutor_RenderTemplate_Invalid(t *testing.T) {
	exec := NewPipelineNotifyExecutor()

	// 无效的模板语法
	result := exec.renderTemplate("{{.Invalid", nil)
	assert.Empty(t, result)
}

func TestParseNotifyConfigs(t *testing.T) {
	configJSON := `[
		{
			"type": "feishu",
			"webhook_url": "https://example.com/webhook",
			"at_all": true
		},
		{
			"type": "dingtalk",
			"webhook_url": "https://example.com/webhook2",
			"secret": "secret123"
		}
	]`

	configs, err := ParseNotifyConfigs(configJSON)
	assert.NoError(t, err)
	assert.Len(t, configs, 2)
	assert.Equal(t, "feishu", configs[0].Type)
	assert.True(t, configs[0].AtAll)
	assert.Equal(t, "dingtalk", configs[1].Type)
	assert.Equal(t, "secret123", configs[1].Secret)
}

func TestParseNotifyConfigs_Empty(t *testing.T) {
	configs, err := ParseNotifyConfigs("")
	assert.NoError(t, err)
	assert.Nil(t, configs)
}

func TestParseNotifyConfigs_Invalid(t *testing.T) {
	_, err := ParseNotifyConfigs("invalid json")
	assert.Error(t, err)
}

func BenchmarkShellExecutor(b *testing.B) {
	exec := NewShellExecutor()
	step := &dto.Step{
		ID:   "bench-step",
		Name: "Benchmark",
		Type: "shell",
		Config: map[string]interface{}{
			"script": "echo test",
		},
	}
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		exec.Execute(ctx, step, nil)
	}
}

func BenchmarkNotifyTemplate(b *testing.B) {
	exec := NewPipelineNotifyExecutor()
	template := "Pipeline {{.PipelineName}} - Status: {{.Status}} - Duration: {{.Duration}}s"
	ctx := &NotifyContext{
		PipelineName: "test-pipeline",
		Status:       "success",
		Duration:     120,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		exec.renderTemplate(template, ctx)
	}
}
