// Package deploy 定义部署流程相关的数据模型
// 本文件包含流水线相关的模型定义
package deploy

import (
	"time"
)

// ==================== 流水线模型 ====================

// Pipeline 流水线
// 定义 CI/CD 流水线配置
type Pipeline struct {
	ID                uint       `gorm:"primaryKey" json:"id"`
	Name              string     `gorm:"size:100;not null" json:"name"`                             // 流水线名称
	Description       string     `gorm:"type:text" json:"description"`                              // 描述
	ProjectID         *uint      `gorm:"index:idx_project" json:"project_id"`                       // 项目ID
	GitRepoID         *uint      `gorm:"index:idx_git_repo" json:"git_repo_id"`                     // Git 仓库ID
	GitBranch         string     `gorm:"size:100;default:main" json:"git_branch"`                   // Git 分支
	BuildClusterID    *uint      `json:"build_cluster_id"`                                          // 构建集群ID
	BuildNamespace    string     `gorm:"size:100;default:devops-build" json:"build_namespace"`      // 构建命名空间
	ConfigJSON        string     `gorm:"column:config_json;type:longtext;not null" json:"-"`        // 配置 JSON
	TriggerConfigJSON string     `gorm:"column:trigger_config_json;type:text" json:"-"`             // 触发配置 JSON
	TriggerConfig     string     `gorm:"column:trigger_config;type:text" json:"-"`                  // Webhook 触发配置
	Status            string     `gorm:"size:20;default:active;index:idx_status" json:"status"`     // 状态: active/disabled
	LastRunAt         *time.Time `json:"last_run_at"`                                               // 最后运行时间
	LastRunStatus     string     `gorm:"size:20" json:"last_run_status"`                            // 最后运行状态
	CreatedBy         *uint      `json:"created_by"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

// TableName 指定表名
func (Pipeline) TableName() string {
	return "pipelines"
}

// PipelineRun 流水线执行记录
// 记录每次流水线运行的详细信息
type PipelineRun struct {
	ID             uint       `gorm:"primaryKey" json:"id"`
	PipelineID     uint       `gorm:"not null;index:idx_pipeline" json:"pipeline_id"`  // 流水线ID
	PipelineName   string     `gorm:"size:100" json:"pipeline_name"`                   // 流水线名称
	Status         string     `gorm:"size:20;not null;index:idx_status" json:"status"` // 状态: pending/running/success/failed/cancelled
	TriggerType    string     `gorm:"size:20;not null" json:"trigger_type"`            // 触发类型: manual/scheduled/webhook
	TriggerBy      string     `gorm:"size:100" json:"trigger_by"`                      // 触发者
	ParametersJSON string     `gorm:"column:parameters_json;type:text" json:"-"`       // 参数 JSON
	GitCommit      string     `gorm:"size:100" json:"git_commit"`                      // Git 提交 SHA
	GitBranch      string     `gorm:"size:100" json:"git_branch"`                      // Git 分支
	GitMessage     string     `gorm:"type:text" json:"git_message"`                    // Git 提交信息
	WorkspaceID    *uint      `json:"workspace_id"`                                    // 工作空间ID
	StartedAt      *time.Time `json:"started_at"`                                      // 开始时间
	FinishedAt     *time.Time `json:"finished_at"`                                     // 完成时间
	Duration       int        `gorm:"default:0" json:"duration"`                       // 执行时长(秒)
	CreatedAt      time.Time  `gorm:"index:idx_created_at" json:"created_at"`
}

// TableName 指定表名
func (PipelineRun) TableName() string {
	return "pipeline_runs"
}

// StageRun 阶段执行记录
// 记录流水线中每个阶段的执行状态
type StageRun struct {
	ID            uint       `gorm:"primaryKey" json:"id"`
	PipelineRunID uint       `gorm:"not null;index:idx_pipeline_run" json:"pipeline_run_id"` // 流水线运行ID
	StageID       string     `gorm:"size:50;not null" json:"stage_id"`                       // 阶段ID
	StageName     string     `gorm:"size:100" json:"stage_name"`                             // 阶段名称
	Status        string     `gorm:"size:20;not null" json:"status"`                         // 状态
	StartedAt     *time.Time `json:"started_at"`                                             // 开始时间
	FinishedAt    *time.Time `json:"finished_at"`                                            // 完成时间
	CreatedAt     time.Time  `json:"created_at"`
}

// TableName 指定表名
func (StageRun) TableName() string {
	return "stage_runs"
}

// StepRun 步骤执行记录
// 记录流水线中每个步骤的执行状态
type StepRun struct {
	ID         uint       `gorm:"primaryKey" json:"id"`
	StageRunID uint       `gorm:"not null;index:idx_stage_run" json:"stage_run_id"` // 阶段运行ID
	StepID     string     `gorm:"size:50;not null" json:"step_id"`                  // 步骤ID
	StepName   string     `gorm:"size:100" json:"step_name"`                        // 步骤名称
	StepType   string     `gorm:"size:50" json:"step_type"`                         // 步骤类型
	BuildJobID *uint      `gorm:"index:idx_build_job" json:"build_job_id"`          // 构建任务ID
	Status     string     `gorm:"size:20;not null" json:"status"`                   // 状态
	Logs       string     `gorm:"type:longtext" json:"logs"`                        // 日志
	ExitCode   *int       `json:"exit_code"`                                        // 退出码
	StartedAt  *time.Time `json:"started_at"`                                       // 开始时间
	FinishedAt *time.Time `json:"finished_at"`                                      // 完成时间
	CreatedAt  time.Time  `json:"created_at"`
}

// TableName 指定表名
func (StepRun) TableName() string {
	return "step_runs"
}

// PipelineTemplate 流水线模板
// 预定义的流水线配置模板
type PipelineTemplate struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Name        string    `gorm:"size:100;not null" json:"name"`                        // 模板名称
	Description string    `gorm:"type:text" json:"description"`                         // 描述
	Category    string    `gorm:"size:50;index:idx_category" json:"category"`           // 分类: frontend/backend/fullstack
	ConfigJSON  string    `gorm:"column:config_json;type:longtext;not null" json:"-"`   // 配置 JSON
	IsBuiltin   bool      `gorm:"default:false" json:"is_builtin"`                      // 是否内置模板
	CreatedAt   time.Time `json:"created_at"`
}

// TableName 指定表名
func (PipelineTemplate) TableName() string {
	return "pipeline_templates"
}

// PipelineCredential 流水线凭证
// 存储流水线使用的各类凭证
type PipelineCredential struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	Name          string    `gorm:"size:100;not null;uniqueIndex:uk_name" json:"name"` // 凭证名称
	Type          string    `gorm:"size:50;not null" json:"type"`                      // 类型: username_password/ssh_key/docker_registry/kubeconfig
	Description   string    `gorm:"type:text" json:"description"`                      // 描述
	DataEncrypted string    `gorm:"column:data_encrypted;type:text;not null" json:"-"` // 加密数据
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// TableName 指定表名
func (PipelineCredential) TableName() string {
	return "pipeline_credentials"
}

// PipelineVariable 流水线环境变量
// 存储流水线使用的环境变量
type PipelineVariable struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	Name       string    `gorm:"size:100;not null" json:"name"`                        // 变量名
	Value      string    `gorm:"type:text;not null" json:"-"`                          // 变量值
	IsSecret   bool      `gorm:"default:false" json:"is_secret"`                       // 是否敏感
	Scope      string    `gorm:"size:20;default:global;index:idx_scope" json:"scope"`  // 作用域: global/pipeline
	PipelineID *uint     `gorm:"index:idx_pipeline" json:"pipeline_id"`                // 流水线ID
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// TableName 指定表名
func (PipelineVariable) TableName() string {
	return "pipeline_variables"
}

// GitRepository Git 仓库配置
// 存储 Git 仓库的连接配置
type GitRepository struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	Name          string    `gorm:"size:100;not null" json:"name"`                  // 仓库名称
	URL           string    `gorm:"size:500;not null" json:"url"`                   // 仓库 URL
	Provider      string    `gorm:"size:50" json:"provider"`                        // 提供商: github/gitlab/gitee/custom
	DefaultBranch string    `gorm:"size:100;default:main" json:"default_branch"`    // 默认分支
	CredentialID  *uint     `json:"credential_id"`                                  // 凭证ID
	WebhookSecret string    `gorm:"size:100" json:"-"`                              // Webhook 密钥
	WebhookURL    string    `gorm:"size:500" json:"webhook_url"`                    // Webhook URL
	Description   string    `gorm:"type:text" json:"description"`                   // 描述
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// TableName 指定表名
func (GitRepository) TableName() string {
	return "git_repositories"
}

// BuildJob 构建任务 (K8s Job)
// 记录 K8s 构建任务的执行状态
type BuildJob struct {
	ID            uint       `gorm:"primaryKey" json:"id"`
	PipelineRunID uint       `gorm:"not null;index:idx_pipeline_run" json:"pipeline_run_id"`       // 流水线运行ID
	StepID        string     `gorm:"size:50;not null" json:"step_id"`                              // 步骤ID
	StepName      string     `gorm:"size:100" json:"step_name"`                                    // 步骤名称
	JobName       string     `gorm:"size:100;not null" json:"job_name"`                            // Job 名称
	Namespace     string     `gorm:"size:100;not null" json:"namespace"`                           // 命名空间
	ClusterID     uint       `gorm:"not null;index:idx_cluster" json:"cluster_id"`                 // 集群ID
	Image         string     `gorm:"size:500;not null" json:"image"`                               // 镜像
	Commands      string     `gorm:"type:text" json:"-"`                                           // 命令 JSON
	WorkDir       string     `gorm:"size:200;default:/workspace" json:"work_dir"`                  // 工作目录
	EnvVars       string     `gorm:"type:text" json:"-"`                                           // 环境变量 JSON
	Resources     string     `gorm:"type:text" json:"-"`                                           // 资源配置 JSON
	Status        string     `gorm:"size:20;not null;default:pending;index:idx_status" json:"status"` // 状态
	PodName       string     `gorm:"size:100" json:"pod_name"`                                     // Pod 名称
	NodeName      string     `gorm:"size:100" json:"node_name"`                                    // 节点名称
	ExitCode      *int       `json:"exit_code"`                                                    // 退出码
	ErrorMessage  string     `gorm:"type:text" json:"error_message"`                               // 错误信息
	StartedAt     *time.Time `json:"started_at"`                                                   // 开始时间
	FinishedAt    *time.Time `json:"finished_at"`                                                  // 完成时间
	CreatedAt     time.Time  `json:"created_at"`
}

// TableName 指定表名
func (BuildJob) TableName() string {
	return "build_jobs"
}

// Artifact 构建制品
// 记录流水线产出的制品信息
type Artifact struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	PipelineRunID uint      `gorm:"not null;index:idx_pipeline_run" json:"pipeline_run_id"` // 流水线运行ID
	PipelineID    *uint     `gorm:"index:idx_pipeline" json:"pipeline_id"`                  // 流水线ID
	Name          string    `gorm:"size:200;not null" json:"name"`                          // 制品名称
	Type          string    `gorm:"size:50;not null;index:idx_type" json:"type"`            // 类型: docker_image/helm_chart/binary/archive
	Path          string    `gorm:"size:500;not null" json:"path"`                          // 路径
	Size          int64     `json:"size"`                                                   // 大小
	Checksum      string    `gorm:"size:100" json:"checksum"`                               // 校验和
	Metadata      string    `gorm:"type:text" json:"-"`                                     // 元数据 JSON
	GitCommit     string    `gorm:"size:100" json:"git_commit"`                             // Git 提交
	GitBranch     string    `gorm:"size:100" json:"git_branch"`                             // Git 分支
	CreatedAt     time.Time `json:"created_at"`
}

// TableName 指定表名
func (Artifact) TableName() string {
	return "artifacts"
}

// BuildCache 构建缓存
// 存储构建过程中的缓存信息
type BuildCache struct {
	ID          uint       `gorm:"primaryKey" json:"id"`
	PipelineID  uint       `gorm:"not null;uniqueIndex:uk_pipeline_key" json:"pipeline_id"` // 流水线ID
	CacheKey    string     `gorm:"size:200;not null;uniqueIndex:uk_pipeline_key" json:"cache_key"` // 缓存键
	StoragePath string     `gorm:"size:500;not null" json:"storage_path"`                   // 存储路径
	Size        int64      `json:"size"`                                                    // 大小
	HitCount    int        `gorm:"default:0" json:"hit_count"`                              // 命中次数
	LastUsedAt  *time.Time `json:"last_used_at"`                                            // 最后使用时间
	ExpiresAt   *time.Time `gorm:"index:idx_expires" json:"expires_at"`                     // 过期时间
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// TableName 指定表名
func (BuildCache) TableName() string {
	return "build_caches"
}

// BuildWorkspace 构建工作空间
// 管理构建过程中的工作空间
type BuildWorkspace struct {
	ID            uint       `gorm:"primaryKey" json:"id"`
	PipelineRunID uint       `gorm:"not null;index:idx_pipeline_run" json:"pipeline_run_id"`       // 流水线运行ID
	ClusterID     uint       `gorm:"not null" json:"cluster_id"`                                   // 集群ID
	Namespace     string     `gorm:"size:100;not null" json:"namespace"`                           // 命名空间
	PVCName       string     `gorm:"size:100;not null" json:"pvc_name"`                            // PVC 名称
	StorageSize   string     `gorm:"size:20;default:10Gi" json:"storage_size"`                     // 存储大小
	Status        string     `gorm:"size:20;not null;default:pending;index:idx_status" json:"status"` // 状态
	CreatedAt     time.Time  `json:"created_at"`
	DeletedAt     *time.Time `json:"deleted_at"`
}

// TableName 指定表名
func (BuildWorkspace) TableName() string {
	return "build_workspaces"
}

// WebhookLog Webhook 日志
// 记录 Webhook 请求的处理情况
type WebhookLog struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	GitRepoID   uint      `gorm:"not null;index:idx_git_repo" json:"git_repo_id"`  // Git 仓库ID
	Provider    string    `gorm:"size:50;not null" json:"provider"`                // 提供商
	Event       string    `gorm:"size:50;not null" json:"event"`                   // 事件类型
	Ref         string    `gorm:"size:200" json:"ref"`                             // 引用
	CommitSHA   string    `gorm:"size:100" json:"commit_sha"`                      // 提交 SHA
	Payload     string    `gorm:"type:longtext" json:"-"`                          // 请求体
	Status      string    `gorm:"size:20;not null;index:idx_status" json:"status"` // 状态
	PipelineRun uint      `gorm:"index:idx_pipeline_run" json:"pipeline_run"`      // 触发的流水线运行ID
	ErrorMsg    string    `gorm:"type:text" json:"error_msg"`                      // 错误信息
	ReceivedAt  time.Time `gorm:"index:idx_received_at" json:"received_at"`        // 接收时间
}

// TableName 指定表名
func (WebhookLog) TableName() string {
	return "webhook_logs"
}

// ArtifactRegistry 制品库配置
// 存储制品库的连接配置
type ArtifactRegistry struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Name        string    `gorm:"size:100;not null" json:"name"`                 // 名称
	Type        string    `gorm:"size:50;not null" json:"type"`                  // 类型: harbor/nexus/dockerhub/acr/ecr/gcr/custom
	URL         string    `gorm:"size:500;not null" json:"url"`                  // URL
	Username    string    `gorm:"size:100" json:"username"`                      // 用户名
	Password    string    `gorm:"size:500" json:"-"`                             // 密码
	Description string    `gorm:"type:text" json:"description"`                  // 描述
	IsDefault   bool      `gorm:"default:false" json:"is_default"`               // 是否默认
	Status      string    `gorm:"size:20;default:unknown" json:"status"`         // 状态
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// TableName 指定表名
func (ArtifactRegistry) TableName() string {
	return "artifact_registries"
}
