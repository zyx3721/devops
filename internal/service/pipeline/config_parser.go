package pipeline

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"

	"devops/pkg/dto"
)

// ConfigParser 配置解析器
type ConfigParser struct{}

// NewConfigParser 创建配置解析器
func NewConfigParser() *ConfigParser {
	return &ConfigParser{}
}

// ParseYAML 解析 YAML 配置
func (p *ConfigParser) ParseYAML(yamlContent string) (*dto.PipelineYAMLConfig, error) {
	var config dto.PipelineYAMLConfig
	if err := yaml.Unmarshal([]byte(yamlContent), &config); err != nil {
		return nil, fmt.Errorf("YAML 解析失败: %v", err)
	}

	return &config, nil
}

// ParseJSON 解析 JSON 配置
func (p *ConfigParser) ParseJSON(jsonContent string) (*dto.PipelineYAMLConfig, error) {
	var config dto.PipelineYAMLConfig
	if err := json.Unmarshal([]byte(jsonContent), &config); err != nil {
		return nil, fmt.Errorf("JSON 解析失败: %v", err)
	}

	return &config, nil
}

// ValidateConfig 验证配置完整性
func (p *ConfigParser) ValidateConfig(config *dto.PipelineYAMLConfig) error {
	if config.Name == "" {
		return fmt.Errorf("流水线名称不能为空")
	}

	if len(config.Stages) == 0 {
		return fmt.Errorf("至少需要一个阶段")
	}

	stageNames := make(map[string]bool)
	for i, stage := range config.Stages {
		if stage.Name == "" {
			return fmt.Errorf("阶段 %d 名称不能为空", i+1)
		}

		if stageNames[stage.Name] {
			return fmt.Errorf("阶段名称重复: %s", stage.Name)
		}
		stageNames[stage.Name] = true

		if len(stage.Steps) == 0 {
			return fmt.Errorf("阶段 '%s' 至少需要一个步骤", stage.Name)
		}

		// 验证依赖
		for _, dep := range stage.Needs {
			if !stageNames[dep] {
				return fmt.Errorf("阶段 '%s' 依赖的阶段 '%s' 不存在或未在之前定义", stage.Name, dep)
			}
		}

		// 验证步骤
		for j, step := range stage.Steps {
			if step.Name == "" {
				return fmt.Errorf("阶段 '%s' 的步骤 %d 名称不能为空", stage.Name, j+1)
			}
			if step.Image == "" {
				return fmt.Errorf("步骤 '%s' 的镜像不能为空", step.Name)
			}
			if len(step.Commands) == 0 {
				return fmt.Errorf("步骤 '%s' 至少需要一个命令", step.Name)
			}
		}
	}

	return nil
}

// ExpandVariables 变量替换
func (p *ConfigParser) ExpandVariables(config *dto.PipelineYAMLConfig, env map[string]string) *dto.PipelineYAMLConfig {
	// 合并配置中的变量和运行时环境变量
	allVars := make(map[string]string)
	for k, v := range config.Variables {
		allVars[k] = v
	}
	for k, v := range env {
		allVars[k] = v
	}

	// 展开阶段和步骤中的变量
	for i := range config.Stages {
		for j := range config.Stages[i].Steps {
			step := &config.Stages[i].Steps[j]

			// 展开镜像名
			step.Image = p.expandString(step.Image, allVars)

			// 展开命令
			for k := range step.Commands {
				step.Commands[k] = p.expandString(step.Commands[k], allVars)
			}

			// 展开工作目录
			step.WorkDir = p.expandString(step.WorkDir, allVars)

			// 展开环境变量值
			for key, val := range step.Env {
				step.Env[key] = p.expandString(val, allVars)
			}
		}
	}

	return config
}

// expandString 展开字符串中的变量
func (p *ConfigParser) expandString(s string, vars map[string]string) string {
	result := s

	// 支持 $VAR 和 ${VAR} 两种格式
	for key, value := range vars {
		result = strings.ReplaceAll(result, "$"+key, value)
		result = strings.ReplaceAll(result, "${"+key+"}", value)
	}

	return result
}

// BuildExecutionPlan 生成执行计划
func (p *ConfigParser) BuildExecutionPlan(config *dto.PipelineYAMLConfig) (*ExecutionPlan, error) {
	plan := &ExecutionPlan{
		Name:   config.Name,
		Stages: make([]ExecutionStage, 0),
	}

	// 构建阶段依赖图
	stageMap := make(map[string]*dto.StageYAMLConfig)
	for i := range config.Stages {
		stageMap[config.Stages[i].Name] = &config.Stages[i]
	}

	// 拓扑排序确定执行顺序
	visited := make(map[string]bool)
	var sortedStages []string

	var visit func(name string) error
	visit = func(name string) error {
		if visited[name] {
			return nil
		}
		visited[name] = true

		stage := stageMap[name]
		if stage == nil {
			return fmt.Errorf("阶段 '%s' 不存在", name)
		}

		for _, dep := range stage.Needs {
			if err := visit(dep); err != nil {
				return err
			}
		}

		sortedStages = append(sortedStages, name)
		return nil
	}

	for _, stage := range config.Stages {
		if err := visit(stage.Name); err != nil {
			return nil, err
		}
	}

	// 构建执行阶段
	for _, stageName := range sortedStages {
		stage := stageMap[stageName]

		execStage := ExecutionStage{
			Name:      stage.Name,
			DependsOn: stage.Needs,
			Steps:     make([]ExecutionStep, 0),
		}

		// 处理矩阵构建
		if stage.Matrix != nil && len(stage.Matrix.Include) > 0 {
			expandedSteps := p.expandMatrix(stage.Steps, stage.Matrix)
			for _, step := range expandedSteps {
				execStage.Steps = append(execStage.Steps, ExecutionStep{
					ID:        step.ID,
					Name:      step.Name,
					Image:     step.Image,
					Commands:  step.Commands,
					WorkDir:   step.WorkDir,
					Env:       step.Env,
					Secrets:   step.Secrets,
					Resources: step.Resources,
					Timeout:   step.Timeout,
				})
			}
		} else {
			for _, step := range stage.Steps {
				execStage.Steps = append(execStage.Steps, ExecutionStep{
					ID:        step.ID,
					Name:      step.Name,
					Image:     step.Image,
					Commands:  step.Commands,
					WorkDir:   step.WorkDir,
					Env:       step.Env,
					Secrets:   step.Secrets,
					Resources: step.Resources,
					Timeout:   step.Timeout,
				})
			}
		}

		plan.Stages = append(plan.Stages, execStage)
	}

	return plan, nil
}

// expandMatrix 展开矩阵构建
func (p *ConfigParser) expandMatrix(steps []dto.ContainerStepConfig, matrix *dto.MatrixConfig) []dto.ContainerStepConfig {
	if matrix == nil || len(matrix.Include) == 0 {
		return steps
	}

	// 计算所有组合
	combinations := p.generateCombinations(matrix.Include)

	var expandedSteps []dto.ContainerStepConfig
	for _, combo := range combinations {
		for _, step := range steps {
			newStep := step
			newStep.ID = fmt.Sprintf("%s-%s", step.ID, p.comboSuffix(combo))
			newStep.Name = fmt.Sprintf("%s (%s)", step.Name, p.comboSuffix(combo))

			// 合并环境变量
			if newStep.Env == nil {
				newStep.Env = make(map[string]string)
			}
			for k, v := range combo {
				newStep.Env[k] = v
			}

			// 展开命令中的变量
			for i := range newStep.Commands {
				for k, v := range combo {
					newStep.Commands[i] = strings.ReplaceAll(newStep.Commands[i], "$"+k, v)
					newStep.Commands[i] = strings.ReplaceAll(newStep.Commands[i], "${"+k+"}", v)
				}
			}

			// 展开镜像名中的变量
			for k, v := range combo {
				newStep.Image = strings.ReplaceAll(newStep.Image, "$"+k, v)
				newStep.Image = strings.ReplaceAll(newStep.Image, "${"+k+"}", v)
			}

			expandedSteps = append(expandedSteps, newStep)
		}
	}

	return expandedSteps
}

// generateCombinations 生成所有组合
func (p *ConfigParser) generateCombinations(include map[string][]string) []map[string]string {
	if len(include) == 0 {
		return nil
	}

	keys := make([]string, 0, len(include))
	for k := range include {
		keys = append(keys, k)
	}

	var result []map[string]string
	var generate func(index int, current map[string]string)
	generate = func(index int, current map[string]string) {
		if index == len(keys) {
			combo := make(map[string]string)
			for k, v := range current {
				combo[k] = v
			}
			result = append(result, combo)
			return
		}

		key := keys[index]
		for _, value := range include[key] {
			current[key] = value
			generate(index+1, current)
		}
	}

	generate(0, make(map[string]string))
	return result
}

// comboSuffix 生成组合后缀
func (p *ConfigParser) comboSuffix(combo map[string]string) string {
	var parts []string
	for _, v := range combo {
		parts = append(parts, v)
	}
	return strings.Join(parts, "-")
}

// ConvertToLegacyConfig 转换为旧版配置格式
func (p *ConfigParser) ConvertToLegacyConfig(config *dto.PipelineYAMLConfig) ([]dto.Stage, []dto.Variable) {
	stages := make([]dto.Stage, 0)
	variables := make([]dto.Variable, 0)

	// 转换变量
	for k, v := range config.Variables {
		variables = append(variables, dto.Variable{
			Name:  k,
			Value: v,
		})
	}

	// 转换阶段
	for _, stage := range config.Stages {
		legacyStage := dto.Stage{
			ID:        strings.ToLower(strings.ReplaceAll(stage.Name, " ", "-")),
			Name:      stage.Name,
			DependsOn: stage.Needs,
			Steps:     make([]dto.Step, 0),
		}

		for _, step := range stage.Steps {
			legacyStep := dto.Step{
				ID:         step.ID,
				Name:       step.Name,
				Type:       "container", // 容器化步骤
				Timeout:    step.Timeout,
				RetryCount: step.RetryCount,
				Condition:  step.Condition,
				Config: map[string]interface{}{
					"image":    step.Image,
					"commands": step.Commands,
					"work_dir": step.WorkDir,
					"env":      step.Env,
					"secrets":  step.Secrets,
				},
			}

			if step.Resources != nil {
				legacyStep.Config["resources"] = step.Resources
			}

			legacyStage.Steps = append(legacyStage.Steps, legacyStep)
		}

		stages = append(stages, legacyStage)
	}

	return stages, variables
}

// GetBuiltinVariables 获取内置变量
func (p *ConfigParser) GetBuiltinVariables(pipelineID, runID uint, gitCommit, gitBranch, gitMessage string) map[string]string {
	return map[string]string{
		"CI":                  "true",
		"CI_PIPELINE_ID":      fmt.Sprintf("%d", pipelineID),
		"CI_PIPELINE_RUN_ID":  fmt.Sprintf("%d", runID),
		"CI_COMMIT_SHA":       gitCommit,
		"CI_COMMIT_BRANCH":    gitBranch,
		"CI_COMMIT_MESSAGE":   gitMessage,
		"CI_COMMIT_SHORT_SHA": p.shortSHA(gitCommit),
	}
}

// shortSHA 获取短 SHA
func (p *ConfigParser) shortSHA(sha string) string {
	if len(sha) > 8 {
		return sha[:8]
	}
	return sha
}

// ValidateImageName 验证镜像名称
func (p *ConfigParser) ValidateImageName(image string) error {
	// 简单的镜像名称验证
	pattern := `^[a-zA-Z0-9][a-zA-Z0-9._/-]*[a-zA-Z0-9](:[a-zA-Z0-9._-]+)?$`
	matched, err := regexp.MatchString(pattern, image)
	if err != nil {
		return err
	}
	if !matched {
		return fmt.Errorf("无效的镜像名称: %s", image)
	}
	return nil
}

// ExecutionPlan 执行计划
type ExecutionPlan struct {
	Name   string           `json:"name"`
	Stages []ExecutionStage `json:"stages"`
}

// ExecutionStage 执行阶段
type ExecutionStage struct {
	Name      string          `json:"name"`
	DependsOn []string        `json:"depends_on"`
	Steps     []ExecutionStep `json:"steps"`
}

// ExecutionStep 执行步骤
type ExecutionStep struct {
	ID        string                   `json:"id"`
	Name      string                   `json:"name"`
	Image     string                   `json:"image"`
	Commands  []string                 `json:"commands"`
	WorkDir   string                   `json:"work_dir"`
	Env       map[string]string        `json:"env"`
	Secrets   []string                 `json:"secrets"`
	Resources *dto.BuildResourceConfig `json:"resources"`
	Timeout   int                      `json:"timeout"`
}
