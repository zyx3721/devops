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

// sanitizeUTF8 清理非 UTF-8 字符
func sanitizeUTF8(s string) string {
	if utf8.ValidString(s) {
		return s
	}
	// 替换无效的 UTF-8 字符
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

// ContainerExecutor 容器执行器（本地模拟，用于开发测试）
// 在没有 K8s 集群时，直接在本地执行命令
type ContainerExecutor struct{}

// NewContainerExecutor 创建容器执行器
func NewContainerExecutor() *ContainerExecutor {
	return &ContainerExecutor{}
}

// Execute 执行容器步骤
func (e *ContainerExecutor) Execute(ctx context.Context, step *dto.Step, env map[string]string) (*StepResult, error) {
	// 从配置中获取命令
	commands, _ := step.Config["commands"].([]interface{})
	if len(commands) == 0 {
		return &StepResult{
			Logs:     "没有要执行的命令",
			ExitCode: 0,
		}, nil
	}

	// 获取工作目录
	workDir, _ := step.Config["work_dir"].(string)
	if workDir == "" {
		workDir = "."
	}

	// 获取镜像（仅记录，本地执行不使用）
	image, _ := step.Config["image"].(string)

	var logs strings.Builder
	logs.WriteString(fmt.Sprintf("=== 步骤: %s ===\n", step.Name))
	if image != "" {
		logs.WriteString(fmt.Sprintf("镜像: %s (本地模式，直接执行命令)\n", image))
	}
	logs.WriteString(fmt.Sprintf("工作目录: %s\n", workDir))
	logs.WriteString("---\n")

	// 执行每个命令
	for i, cmd := range commands {
		cmdStr, ok := cmd.(string)
		if !ok {
			continue
		}

		logs.WriteString(fmt.Sprintf("\n[%d] $ %s\n", i+1, cmdStr))

		var execCmd *exec.Cmd
		if runtime.GOOS == "windows" {
			execCmd = exec.CommandContext(ctx, "cmd", "/C", cmdStr)
		} else {
			execCmd = exec.CommandContext(ctx, "sh", "-c", cmdStr)
		}

		if workDir != "" && workDir != "." {
			execCmd.Dir = workDir
		}

		// 设置环境变量
		execCmd.Env = os.Environ()
		for k, v := range env {
			execCmd.Env = append(execCmd.Env, fmt.Sprintf("%s=%s", k, v))
		}

		// 添加步骤配置中的环境变量
		if envConfig, ok := step.Config["env"].(map[string]interface{}); ok {
			for k, v := range envConfig {
				execCmd.Env = append(execCmd.Env, fmt.Sprintf("%s=%v", k, v))
			}
		}

		var stdout, stderr bytes.Buffer
		execCmd.Stdout = &stdout
		execCmd.Stderr = &stderr

		err := execCmd.Run()
		logs.WriteString(sanitizeUTF8(stdout.String()))
		if stderr.Len() > 0 {
			logs.WriteString(sanitizeUTF8(stderr.String()))
		}

		if err != nil {
			exitCode := 1
			if exitErr, ok := err.(*exec.ExitError); ok {
				exitCode = exitErr.ExitCode()
			}
			logs.WriteString(fmt.Sprintf("\n命令执行失败: %v\n", err))
			return &StepResult{
				Logs:     logs.String(),
				ExitCode: exitCode,
			}, fmt.Errorf("命令执行失败: %v", err)
		}
	}

	logs.WriteString("\n=== 步骤执行完成 ===\n")

	return &StepResult{
		Logs:     logs.String(),
		ExitCode: 0,
	}, nil
}

// Validate 验证配置
func (e *ContainerExecutor) Validate(config map[string]interface{}) error {
	return nil
}
