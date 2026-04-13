package executor

import (
	"context"

	"devops/pkg/dto"
)

// StepExecutor 步骤执行器接口
type StepExecutor interface {
	Execute(ctx context.Context, step *dto.Step, env map[string]string) (*StepResult, error)
	Validate(config map[string]interface{}) error
}

// StepResult 步骤执行结果
type StepResult struct {
	Logs     string
	ExitCode int
	Output   map[string]string // 输出变量
}
