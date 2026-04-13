package executor

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"unicode/utf8"

	"devops/pkg/dto"
)

// sanitizeOutput 清理非 UTF-8 字符
func sanitizeOutput(s string) string {
	if utf8.ValidString(s) {
		return s
	}
	var builder strings.Builder
	for _, r := range s {
		if r == utf8.RuneError {
			builder.WriteRune('?')
		} else {
			builder.WriteRune(r)
		}
	}
	return builder.String()
}

// ShellExecutor Shell脚本执行器
type ShellExecutor struct{}

// NewShellExecutor 创建Shell执行器
func NewShellExecutor() *ShellExecutor {
	return &ShellExecutor{}
}

// Execute 执行Shell脚本
func (e *ShellExecutor) Execute(ctx context.Context, step *dto.Step, env map[string]string) (*StepResult, error) {
	script, ok := step.Config["script"].(string)
	if !ok || script == "" {
		return nil, fmt.Errorf("缺少script配置")
	}

	workDir, _ := step.Config["work_dir"].(string)

	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.CommandContext(ctx, "cmd", "/C", script)
	} else {
		cmd = exec.CommandContext(ctx, "sh", "-c", script)
	}

	if workDir != "" {
		cmd.Dir = workDir
	}

	// 设置环境变量
	cmd.Env = os.Environ()
	for k, v := range env {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	result := &StepResult{
		Logs:     sanitizeOutput(stdout.String() + stderr.String()),
		ExitCode: 0,
	}

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
		} else {
			result.ExitCode = 1
		}
		return result, fmt.Errorf("脚本执行失败: %v", err)
	}

	return result, nil
}

// Validate 验证配置
func (e *ShellExecutor) Validate(config map[string]interface{}) error {
	if _, ok := config["script"].(string); !ok {
		return fmt.Errorf("缺少script配置")
	}
	return nil
}

// GitExecutor Git克隆执行器
type GitExecutor struct{}

// NewGitExecutor 创建Git执行器
func NewGitExecutor() *GitExecutor {
	return &GitExecutor{}
}

// Execute 执行Git克隆
func (e *GitExecutor) Execute(ctx context.Context, step *dto.Step, env map[string]string) (*StepResult, error) {
	repo, _ := step.Config["repo"].(string)
	branch, _ := step.Config["branch"].(string)
	if branch == "" {
		branch = "main"
	}
	targetDir, _ := step.Config["target_dir"].(string)
	if targetDir == "" {
		targetDir = "."
	}

	// 从环境变量获取仓库地址
	if repo == "" {
		repo = env["GIT_REPO"]
	}
	if repo == "" {
		return nil, fmt.Errorf("缺少repo配置")
	}

	// 构建git clone命令
	args := []string{"clone", "--depth", "1", "-b", branch, repo, targetDir}
	cmd := exec.CommandContext(ctx, "git", args...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	result := &StepResult{
		Logs:     stdout.String() + stderr.String(),
		ExitCode: 0,
	}

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
		} else {
			result.ExitCode = 1
		}
		return result, fmt.Errorf("Git克隆失败: %v", err)
	}

	return result, nil
}

// Validate 验证配置
func (e *GitExecutor) Validate(config map[string]interface{}) error {
	return nil
}

// DockerBuildExecutor Docker构建执行器
type DockerBuildExecutor struct{}

// NewDockerBuildExecutor 创建Docker构建执行器
func NewDockerBuildExecutor() *DockerBuildExecutor {
	return &DockerBuildExecutor{}
}

// Execute 执行Docker构建
func (e *DockerBuildExecutor) Execute(ctx context.Context, step *dto.Step, env map[string]string) (*StepResult, error) {
	dockerfile, _ := step.Config["dockerfile"].(string)
	if dockerfile == "" {
		dockerfile = "Dockerfile"
	}
	context_, _ := step.Config["context"].(string)
	if context_ == "" {
		context_ = "."
	}
	imageName, _ := step.Config["image"].(string)
	if imageName == "" {
		imageName = env["IMAGE_NAME"]
	}
	if imageName == "" {
		return nil, fmt.Errorf("缺少image配置")
	}

	// 构建docker build命令
	args := []string{"build", "-f", dockerfile, "-t", imageName}

	// 添加构建参数
	if buildArgs, ok := step.Config["build_args"].(map[string]interface{}); ok {
		for k, v := range buildArgs {
			args = append(args, "--build-arg", fmt.Sprintf("%s=%v", k, v))
		}
	}

	args = append(args, context_)

	cmd := exec.CommandContext(ctx, "docker", args...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	result := &StepResult{
		Logs:     stdout.String() + stderr.String(),
		ExitCode: 0,
	}

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
		} else {
			result.ExitCode = 1
		}
		return result, fmt.Errorf("Docker构建失败: %v", err)
	}

	return result, nil
}

// Validate 验证配置
func (e *DockerBuildExecutor) Validate(config map[string]interface{}) error {
	return nil
}

// DockerPushExecutor Docker推送执行器
type DockerPushExecutor struct{}

// NewDockerPushExecutor 创建Docker推送执行器
func NewDockerPushExecutor() *DockerPushExecutor {
	return &DockerPushExecutor{}
}

// Execute 执行Docker推送
func (e *DockerPushExecutor) Execute(ctx context.Context, step *dto.Step, env map[string]string) (*StepResult, error) {
	imageName, _ := step.Config["image"].(string)
	if imageName == "" {
		imageName = env["IMAGE_NAME"]
	}
	if imageName == "" {
		return nil, fmt.Errorf("缺少image配置")
	}

	// 登录（如果有凭证）
	registry, _ := step.Config["registry"].(string)
	username, _ := step.Config["username"].(string)
	password, _ := step.Config["password"].(string)

	var logs strings.Builder

	if username != "" && password != "" {
		loginArgs := []string{"login"}
		if registry != "" {
			loginArgs = append(loginArgs, registry)
		}
		loginArgs = append(loginArgs, "-u", username, "-p", password)

		loginCmd := exec.CommandContext(ctx, "docker", loginArgs...)
		var loginOut bytes.Buffer
		loginCmd.Stdout = &loginOut
		loginCmd.Stderr = &loginOut
		if err := loginCmd.Run(); err != nil {
			return &StepResult{Logs: loginOut.String(), ExitCode: 1}, fmt.Errorf("Docker登录失败: %v", err)
		}
		logs.WriteString(loginOut.String())
		logs.WriteString("\n")
	}

	// 推送
	cmd := exec.CommandContext(ctx, "docker", "push", imageName)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	logs.WriteString(stdout.String())
	logs.WriteString(stderr.String())

	result := &StepResult{
		Logs:     logs.String(),
		ExitCode: 0,
	}

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
		} else {
			result.ExitCode = 1
		}
		return result, fmt.Errorf("Docker推送失败: %v", err)
	}

	return result, nil
}

// Validate 验证配置
func (e *DockerPushExecutor) Validate(config map[string]interface{}) error {
	return nil
}
