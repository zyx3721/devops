package dto

import "time"

// ==================== 流水线 ====================

// PipelineListRequest 流水线列表请求
type PipelineListRequest struct {
	Name      string `form:"name"`
	ProjectID uint   `form:"project_id"`
	Status    string `form:"status"`
	Page      int    `form:"page"`
	PageSize  int    `form:"page_size"`
}

// PipelineListResponse 流水线列表响应
type PipelineListResponse struct {
	Total int            `json:"total"`
	Items []PipelineItem `json:"items"`
}

// PipelineItem 流水线项
type PipelineItem struct {
	ID            uint       `json:"id"`
	Name          string     `json:"name"`
	Description   string     `json:"description"`
	ProjectID     *uint      `json:"project_id"`
	GitRepoID     *uint      `json:"git_repo_id"`
	GitRepoURL    string     `json:"git_repo_url"`
	GitBranch     string     `json:"git_branch"`
	Status        string     `json:"status"`
	LastRunAt     *time.Time `json:"last_run_at"`
	LastRunStatus string     `json:"last_run_status"`
	CreatedBy     *uint      `json:"created_by"`
	CreatedAt     time.Time  `json:"created_at"`
}

// PipelineRequest 流水线请求
type PipelineRequest struct {
	ID             uint          `json:"id"`
	Name           string        `json:"name" binding:"required"`
	Description    string        `json:"description"`
	ProjectID      *uint         `json:"project_id"`
	GitRepoID      *uint         `json:"git_repo_id"`
	GitBranch      string        `json:"git_branch"`
	BuildClusterID *uint         `json:"build_cluster_id"`
	BuildNamespace string        `json:"build_namespace"`
	Stages         []Stage       `json:"stages" binding:"required"`
	Variables      []Variable    `json:"variables"`
	TriggerConfig  TriggerConfig `json:"trigger_config"`
}

// PipelineDetailResponse 流水线详情响应
type PipelineDetailResponse struct {
	ID             uint          `json:"id"`
	Name           string        `json:"name"`
	Description    string        `json:"description"`
	ProjectID      *uint         `json:"project_id"`
	GitRepoID      *uint         `json:"git_repo_id"`
	GitBranch      string        `json:"git_branch"`
	BuildClusterID *uint         `json:"build_cluster_id"`
	BuildNamespace string        `json:"build_namespace"`
	Stages         []Stage       `json:"stages"`
	Variables      []Variable    `json:"variables"`
	TriggerConfig  TriggerConfig `json:"trigger_config"`
	Status         string        `json:"status"`
	LastRunAt      *time.Time    `json:"last_run_at"`
	LastRunStatus  string        `json:"last_run_status"`
	CreatedBy      *uint         `json:"created_by"`
	CreatedAt      time.Time     `json:"created_at"`
	UpdatedAt      time.Time     `json:"updated_at"`
}

// Stage 阶段
type Stage struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	Steps     []Step   `json:"steps"`
	Parallel  bool     `json:"parallel"`
	DependsOn []string `json:"depends_on"`
}

// Step 步骤
type Step struct {
	ID         string                 `json:"id"`
	Name       string                 `json:"name"`
	Type       string                 `json:"type"` // git, docker_build, docker_push, shell, k8s_deploy, scan, notify
	Config     map[string]interface{} `json:"config"`
	Timeout    int                    `json:"timeout"`
	RetryCount int                    `json:"retry_count"`
	Condition  string                 `json:"condition"` // always, on_success, on_failure
}

// Variable 变量
type Variable struct {
	Name     string `json:"name"`
	Value    string `json:"value"`
	IsSecret bool   `json:"is_secret"`
}

// TriggerConfig 触发器配置
type TriggerConfig struct {
	Manual    bool            `json:"manual"`
	Scheduled *ScheduleConfig `json:"scheduled"`
	Webhook   *WebhookConfig  `json:"webhook"`
}

// ScheduleConfig 定时配置
type ScheduleConfig struct {
	Enabled  bool   `json:"enabled"`
	Cron     string `json:"cron"`
	Timezone string `json:"timezone"`
}

// WebhookConfig Webhook配置
type WebhookConfig struct {
	Enabled      bool     `json:"enabled"`
	Secret       string   `json:"secret"`
	BranchFilter []string `json:"branch_filter"`
	URL          string   `json:"url"` // 生成的Webhook URL
}

// ==================== 流水线执行 ====================

// RunPipelineRequest 运行流水线请求
type RunPipelineRequest struct {
	Parameters map[string]string `json:"parameters"`
	Branch     string            `json:"branch"`
}

// PipelineRunListRequest 执行历史请求
type PipelineRunListRequest struct {
	PipelineID uint   `form:"pipeline_id"`
	Status     string `form:"status"`
	Page       int    `form:"page"`
	PageSize   int    `form:"page_size"`
}

// PipelineRunListResponse 执行历史响应
type PipelineRunListResponse struct {
	Total int               `json:"total"`
	Items []PipelineRunItem `json:"items"`
}

// PipelineRunItem 执行记录项
type PipelineRunItem struct {
	ID           uint       `json:"id"`
	PipelineID   uint       `json:"pipeline_id"`
	PipelineName string     `json:"pipeline_name"`
	Status       string     `json:"status"`
	TriggerType  string     `json:"trigger_type"`
	TriggerBy    string     `json:"trigger_by"`
	StartedAt    *time.Time `json:"started_at"`
	FinishedAt   *time.Time `json:"finished_at"`
	Duration     int        `json:"duration"`
	CreatedAt    time.Time  `json:"created_at"`
}

// PipelineRunDetailResponse 执行详情响应
type PipelineRunDetailResponse struct {
	ID           uint              `json:"id"`
	PipelineID   uint              `json:"pipeline_id"`
	PipelineName string            `json:"pipeline_name"`
	Status       string            `json:"status"`
	TriggerType  string            `json:"trigger_type"`
	TriggerBy    string            `json:"trigger_by"`
	Parameters   map[string]string `json:"parameters"`
	StageRuns    []StageRunItem    `json:"stage_runs"`
	StartedAt    *time.Time        `json:"started_at"`
	FinishedAt   *time.Time        `json:"finished_at"`
	Duration     int               `json:"duration"`
	CreatedAt    time.Time         `json:"created_at"`
}

// StageRunItem 阶段执行项
type StageRunItem struct {
	ID         uint          `json:"id"`
	StageID    string        `json:"stage_id"`
	StageName  string        `json:"stage_name"`
	Status     string        `json:"status"`
	StepRuns   []StepRunItem `json:"step_runs"`
	StartedAt  *time.Time    `json:"started_at"`
	FinishedAt *time.Time    `json:"finished_at"`
}

// StepRunItem 步骤执行项
type StepRunItem struct {
	ID         uint       `json:"id"`
	StepID     string     `json:"step_id"`
	StepName   string     `json:"step_name"`
	StepType   string     `json:"step_type"`
	Status     string     `json:"status"`
	Logs       string     `json:"logs"`
	ExitCode   *int       `json:"exit_code"`
	StartedAt  *time.Time `json:"started_at"`
	FinishedAt *time.Time `json:"finished_at"`
}

// StepLogsResponse 步骤日志响应
type StepLogsResponse struct {
	StepID   string `json:"step_id"`
	StepName string `json:"step_name"`
	Logs     string `json:"logs"`
	Status   string `json:"status"`
}

// ==================== 模板 ====================

// PipelineTemplateItem 模板项
type PipelineTemplateItem struct {
	ID          uint      `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Category    string    `json:"category"`
	IsBuiltin   bool      `json:"is_builtin"`
	CreatedAt   time.Time `json:"created_at"`
}

// PipelineTemplateDetailResponse 模板详情响应
type PipelineTemplateDetailResponse struct {
	ID          uint      `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Category    string    `json:"category"`
	Stages      []Stage   `json:"stages"`
	IsBuiltin   bool      `json:"is_builtin"`
	CreatedAt   time.Time `json:"created_at"`
}

// CreateFromTemplateRequest 从模板创建请求
type CreateFromTemplateRequest struct {
	TemplateID  uint   `json:"template_id" binding:"required"`
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
	ProjectID   *uint  `json:"project_id"`
}

// ==================== 凭证 ====================

// CredentialRequest 凭证请求
type CredentialRequest struct {
	ID          uint   `json:"id"`
	Name        string `json:"name" binding:"required"`
	Type        string `json:"type" binding:"required"`
	Description string `json:"description"`
	Data        string `json:"data" binding:"required"` // JSON格式的凭证数据
}

// CredentialItem 凭证项
type CredentialItem struct {
	ID          uint      `json:"id"`
	Name        string    `json:"name"`
	Type        string    `json:"type"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// ==================== 环境变量 ====================

// VariableRequest 变量请求
type VariableRequest struct {
	ID         uint   `json:"id"`
	Name       string `json:"name" binding:"required"`
	Value      string `json:"value" binding:"required"`
	IsSecret   bool   `json:"is_secret"`
	Scope      string `json:"scope"` // global, pipeline
	PipelineID *uint  `json:"pipeline_id"`
}

// VariableItem 变量项
type VariableItem struct {
	ID         uint      `json:"id"`
	Name       string    `json:"name"`
	Value      string    `json:"value"` // 敏感变量返回 ******
	IsSecret   bool      `json:"is_secret"`
	Scope      string    `json:"scope"`
	PipelineID *uint     `json:"pipeline_id"`
	CreatedAt  time.Time `json:"created_at"`
}

// ==================== Git 仓库 ====================

// GitRepoRequest Git 仓库请求
type GitRepoRequest struct {
	ID            uint   `json:"id"`
	Name          string `json:"name" binding:"required"`
	URL           string `json:"url" binding:"required"`
	Provider      string `json:"provider"` // github, gitlab, gitee, custom
	DefaultBranch string `json:"default_branch"`
	CredentialID  *uint  `json:"credential_id"`
	Description   string `json:"description"`
}

// GitRepoItem Git 仓库项
type GitRepoItem struct {
	ID             uint      `json:"id"`
	Name           string    `json:"name"`
	URL            string    `json:"url"`
	Provider       string    `json:"provider"`
	DefaultBranch  string    `json:"default_branch"`
	CredentialID   *uint     `json:"credential_id"`
	CredentialName string    `json:"credential_name"`
	WebhookURL     string    `json:"webhook_url"`
	Description    string    `json:"description"`
	CreatedAt      time.Time `json:"created_at"`
}

// GitRepoListRequest Git 仓库列表请求
type GitRepoListRequest struct {
	Name     string `form:"name"`
	Provider string `form:"provider"`
	Page     int    `form:"page"`
	PageSize int    `form:"page_size"`
}

// GitRepoListResponse Git 仓库列表响应
type GitRepoListResponse struct {
	Total int           `json:"total"`
	Items []GitRepoItem `json:"items"`
}

// GitBranchItem Git 分支项
type GitBranchItem struct {
	Name      string `json:"name"`
	IsDefault bool   `json:"is_default"`
	CommitSHA string `json:"commit_sha"`
}

// GitTagItem Git Tag 项
type GitTagItem struct {
	Name      string `json:"name"`
	CommitSHA string `json:"commit_sha"`
}

// GitTestConnectionRequest 测试连接请求
type GitTestConnectionRequest struct {
	URL          string `json:"url" binding:"required"`
	CredentialID *uint  `json:"credential_id"`
}

// GitTestConnectionResponse 测试连接响应
type GitTestConnectionResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// ==================== 构建任务 ====================

// BuildJobItem 构建任务项
type BuildJobItem struct {
	ID            uint       `json:"id"`
	PipelineRunID uint       `json:"pipeline_run_id"`
	StepID        string     `json:"step_id"`
	StepName      string     `json:"step_name"`
	JobName       string     `json:"job_name"`
	Namespace     string     `json:"namespace"`
	ClusterID     uint       `json:"cluster_id"`
	Image         string     `json:"image"`
	Status        string     `json:"status"`
	PodName       string     `json:"pod_name"`
	NodeName      string     `json:"node_name"`
	ExitCode      *int       `json:"exit_code"`
	ErrorMessage  string     `json:"error_message"`
	StartedAt     *time.Time `json:"started_at"`
	FinishedAt    *time.Time `json:"finished_at"`
	CreatedAt     time.Time  `json:"created_at"`
}

// BuildJobListRequest 构建任务列表请求
type BuildJobListRequest struct {
	PipelineRunID uint   `form:"pipeline_run_id"`
	Status        string `form:"status"`
	Page          int    `form:"page"`
	PageSize      int    `form:"page_size"`
}

// BuildJobListResponse 构建任务列表响应
type BuildJobListResponse struct {
	Total int            `json:"total"`
	Items []BuildJobItem `json:"items"`
}

// BuildJobConfig 构建任务配置
type BuildJobConfig struct {
	PipelineRunID uint                 `json:"pipeline_run_id"`
	StepID        string               `json:"step_id"`
	StepName      string               `json:"step_name"`
	ClusterID     uint                 `json:"cluster_id"`
	Namespace     string               `json:"namespace"`
	Image         string               `json:"image"`
	Commands      []string             `json:"commands"`
	WorkDir       string               `json:"work_dir"`
	EnvVars       map[string]string    `json:"env_vars"`
	Secrets       []string             `json:"secrets"`
	Resources     *BuildResourceConfig `json:"resources"`
	Timeout       int                  `json:"timeout"`
	GitURL        string               `json:"git_url"`
	GitBranch     string               `json:"git_branch"`
	GitCredential *uint                `json:"git_credential"`
	WorkspacePVC  string               `json:"workspace_pvc"`
	CachePVC      string               `json:"cache_pvc"`   // 缓存 PVC 名称
	CachePaths    []string             `json:"cache_paths"` // 缓存挂载路径列表
}

// BuildResourceConfig 构建资源配置
type BuildResourceConfig struct {
	CPURequest    string `json:"cpu_request"`
	CPULimit      string `json:"cpu_limit"`
	MemoryRequest string `json:"memory_request"`
	MemoryLimit   string `json:"memory_limit"`
}

// ==================== 制品 ====================

// ArtifactItem 制品项
type ArtifactItem struct {
	ID            uint      `json:"id"`
	PipelineRunID uint      `json:"pipeline_run_id"`
	PipelineID    *uint     `json:"pipeline_id"`
	PipelineName  string    `json:"pipeline_name"`
	Name          string    `json:"name"`
	Type          string    `json:"type"`
	Path          string    `json:"path"`
	Size          int64     `json:"size"`
	SizeHuman     string    `json:"size_human"`
	Checksum      string    `json:"checksum"`
	GitCommit     string    `json:"git_commit"`
	GitBranch     string    `json:"git_branch"`
	CreatedAt     time.Time `json:"created_at"`
}

// ArtifactListRequest 制品列表请求
type ArtifactListRequest struct {
	PipelineID    uint   `form:"pipeline_id"`
	PipelineRunID uint   `form:"pipeline_run_id"`
	Type          string `form:"type"`
	Page          int    `form:"page"`
	PageSize      int    `form:"page_size"`
}

// ArtifactListResponse 制品列表响应
type ArtifactListResponse struct {
	Total int            `json:"total"`
	Items []ArtifactItem `json:"items"`
}

// ArtifactCreateRequest 创建制品请求
type ArtifactCreateRequest struct {
	PipelineRunID uint   `json:"pipeline_run_id" binding:"required"`
	Name          string `json:"name" binding:"required"`
	Type          string `json:"type" binding:"required"`
	Path          string `json:"path" binding:"required"`
	Size          int64  `json:"size"`
	Checksum      string `json:"checksum"`
	Metadata      string `json:"metadata"`
	GitCommit     string `json:"git_commit"`
	GitBranch     string `json:"git_branch"`
}

// ==================== 构建缓存 ====================

// BuildCacheItem 构建缓存项
type BuildCacheItem struct {
	ID          uint       `json:"id"`
	PipelineID  uint       `json:"pipeline_id"`
	CacheKey    string     `json:"cache_key"`
	StoragePath string     `json:"storage_path"`
	Size        int64      `json:"size"`
	SizeHuman   string     `json:"size_human"`
	HitCount    int        `json:"hit_count"`
	LastUsedAt  *time.Time `json:"last_used_at"`
	ExpiresAt   *time.Time `json:"expires_at"`
	CreatedAt   time.Time  `json:"created_at"`
}

// BuildCacheListRequest 构建缓存列表请求
type BuildCacheListRequest struct {
	PipelineID uint `form:"pipeline_id"`
	Page       int  `form:"page"`
	PageSize   int  `form:"page_size"`
}

// BuildCacheListResponse 构建缓存列表响应
type BuildCacheListResponse struct {
	Total int              `json:"total"`
	Items []BuildCacheItem `json:"items"`
}

// ==================== 容器化步骤配置 ====================

// ContainerStepConfig 容器化步骤配置
type ContainerStepConfig struct {
	ID         string               `json:"id" yaml:"id"`
	Name       string               `json:"name" yaml:"name"`
	Image      string               `json:"image" yaml:"image"`
	Commands   []string             `json:"commands" yaml:"commands"`
	WorkDir    string               `json:"work_dir" yaml:"work_dir"`
	Env        map[string]string    `json:"env" yaml:"env"`
	Secrets    []string             `json:"secrets" yaml:"secrets"`
	Resources  *BuildResourceConfig `json:"resources" yaml:"resources"`
	Timeout    int                  `json:"timeout" yaml:"timeout"`
	RetryCount int                  `json:"retry_count" yaml:"retry_count"`
	Condition  string               `json:"condition" yaml:"condition"`
}

// PipelineYAMLConfig YAML 格式流水线配置
type PipelineYAMLConfig struct {
	Name      string             `json:"name" yaml:"name"`
	Trigger   *TriggerYAMLConfig `json:"trigger" yaml:"trigger"`
	Variables map[string]string  `json:"variables" yaml:"variables"`
	Cache     *CacheConfig       `json:"cache" yaml:"cache"`
	Stages    []StageYAMLConfig  `json:"stages" yaml:"stages"`
}

// TriggerYAMLConfig YAML 触发器配置
type TriggerYAMLConfig struct {
	Branches []string `json:"branches" yaml:"branches"`
	Tags     []string `json:"tags" yaml:"tags"`
	Events   []string `json:"events" yaml:"events"` // push, pull_request, tag
	Cron     string   `json:"cron" yaml:"cron"`
}

// StageYAMLConfig YAML 阶段配置
type StageYAMLConfig struct {
	Name   string                `json:"name" yaml:"name"`
	Needs  []string              `json:"needs" yaml:"needs"`
	Matrix *MatrixConfig         `json:"matrix" yaml:"matrix"`
	Steps  []ContainerStepConfig `json:"steps" yaml:"steps"`
}

// CacheConfig 缓存配置
type CacheConfig struct {
	Key   string   `json:"key" yaml:"key"`
	Paths []string `json:"paths" yaml:"paths"`
}

// MatrixConfig 矩阵构建配置
type MatrixConfig struct {
	Include map[string][]string `json:"include" yaml:"include"`
}

// ==================== 扩展流水线请求 ====================

// PipelineCreateRequest 创建流水线请求（扩展）
type PipelineCreateRequest struct {
	Name           string        `json:"name" binding:"required"`
	Description    string        `json:"description"`
	ProjectID      *uint         `json:"project_id"`
	GitRepoID      *uint         `json:"git_repo_id"`
	GitBranch      string        `json:"git_branch"`
	BuildClusterID *uint         `json:"build_cluster_id"`
	BuildNamespace string        `json:"build_namespace"`
	Stages         []Stage       `json:"stages"`
	Variables      []Variable    `json:"variables"`
	TriggerConfig  TriggerConfig `json:"trigger_config"`
	YAMLConfig     string        `json:"yaml_config"` // YAML 格式配置
}

// PipelineDetailExtResponse 流水线详情响应（扩展）
type PipelineDetailExtResponse struct {
	ID               uint          `json:"id"`
	Name             string        `json:"name"`
	Description      string        `json:"description"`
	ProjectID        *uint         `json:"project_id"`
	GitRepoID        *uint         `json:"git_repo_id"`
	GitRepoName      string        `json:"git_repo_name"`
	GitRepoURL       string        `json:"git_repo_url"`
	GitBranch        string        `json:"git_branch"`
	BuildClusterID   *uint         `json:"build_cluster_id"`
	BuildClusterName string        `json:"build_cluster_name"`
	BuildNamespace   string        `json:"build_namespace"`
	Stages           []Stage       `json:"stages"`
	Variables        []Variable    `json:"variables"`
	TriggerConfig    TriggerConfig `json:"trigger_config"`
	Status           string        `json:"status"`
	LastRunAt        *time.Time    `json:"last_run_at"`
	LastRunStatus    string        `json:"last_run_status"`
	CreatedBy        *uint         `json:"created_by"`
	CreatedAt        time.Time     `json:"created_at"`
	UpdatedAt        time.Time     `json:"updated_at"`
}

// PipelineRunDetailExtResponse 执行详情响应（扩展）
type PipelineRunDetailExtResponse struct {
	ID           uint              `json:"id"`
	PipelineID   uint              `json:"pipeline_id"`
	PipelineName string            `json:"pipeline_name"`
	Status       string            `json:"status"`
	TriggerType  string            `json:"trigger_type"`
	TriggerBy    string            `json:"trigger_by"`
	Parameters   map[string]string `json:"parameters"`
	GitCommit    string            `json:"git_commit"`
	GitBranch    string            `json:"git_branch"`
	GitMessage   string            `json:"git_message"`
	StageRuns    []StageRunItem    `json:"stage_runs"`
	BuildJobs    []BuildJobItem    `json:"build_jobs"`
	Artifacts    []ArtifactItem    `json:"artifacts"`
	StartedAt    *time.Time        `json:"started_at"`
	FinishedAt   *time.Time        `json:"finished_at"`
	Duration     int               `json:"duration"`
	CreatedAt    time.Time         `json:"created_at"`
}

// ==================== 流水线统计 ====================

// PipelineStatsRequest 流水线统计请求
type PipelineStatsRequest struct {
	StartDate string `form:"start_date"` // YYYY-MM-DD
	EndDate   string `form:"end_date"`   // YYYY-MM-DD
}

// PipelineStatsResponse 流水线统计响应
type PipelineStatsResponse struct {
	Overview           PipelineStatsOverview     `json:"overview"`
	Trend              []PipelineStatsTrendItem  `json:"trend"`
	StatusDistribution map[string]int            `json:"statusDistribution"`
	DurationTrend      []PipelineDurationItem    `json:"durationTrend"`
	Rank               []PipelineRankItem        `json:"rank"`
	RecentFailed       []PipelineRecentFailedRun `json:"recentFailed"`
}

// PipelineStatsOverview 统计概览
type PipelineStatsOverview struct {
	Total       int     `json:"total"`
	Success     int     `json:"success"`
	Failed      int     `json:"failed"`
	SuccessRate float64 `json:"successRate"`
}

// PipelineStatsTrendItem 趋势项
type PipelineStatsTrendItem struct {
	Date    string `json:"date"`
	Success int    `json:"success"`
	Failed  int    `json:"failed"`
}

// PipelineDurationItem 耗时项
type PipelineDurationItem struct {
	Date     string `json:"date"`
	Duration int    `json:"duration"` // 秒
}

// PipelineRankItem 排行项
type PipelineRankItem struct {
	ID          uint    `json:"id"`
	Name        string  `json:"name"`
	Total       int     `json:"total"`
	SuccessRate float64 `json:"successRate"`
	AvgDuration string  `json:"avgDuration"`
}

// PipelineRecentFailedRun 最近失败的执行
type PipelineRecentFailedRun struct {
	ID           uint      `json:"id"`
	PipelineID   uint      `json:"pipeline_id"`
	PipelineName string    `json:"pipeline_name"`
	RunNumber    int       `json:"run_number"`
	Status       string    `json:"status"`
	ErrorMessage string    `json:"error_message"`
	CreatedAt    time.Time `json:"created_at"`
}
