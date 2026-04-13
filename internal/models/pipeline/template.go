// Package pipeline 流水线相关模型
// 本文件定义流水线模板相关模型
package pipeline

import (
	"time"
)

// PipelineTemplate 流水线模板
type PipelineTemplate struct {
	ID          uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	Name        string    `gorm:"size:100;not null;uniqueIndex;comment:模板名称" json:"name"`
	Description string    `gorm:"size:500;comment:模板描述" json:"description"`
	Category    string    `gorm:"size:50;index;comment:模板分类" json:"category"` // build, deploy, test, release
	Language    string    `gorm:"size:50;index;comment:编程语言" json:"language"` // java, go, nodejs, python, etc.
	Framework   string    `gorm:"size:50;comment:框架" json:"framework"`         // spring, gin, express, django, etc.
	ConfigJSON  string    `gorm:"type:json;not null;comment:流水线配置" json:"config_json"`
	IconURL     string    `gorm:"size:500;comment:图标URL" json:"icon_url"`
	IsBuiltin   bool      `gorm:"default:false;comment:是否内置模板" json:"is_builtin"`
	IsPublic    bool      `gorm:"default:true;comment:是否公开" json:"is_public"`
	UsageCount  int       `gorm:"default:0;comment:使用次数" json:"usage_count"`
	Rating      float64   `gorm:"default:0;comment:评分" json:"rating"`
	RatingCount int       `gorm:"default:0;comment:评分人数" json:"rating_count"`
	CreatedBy   string    `gorm:"size:100;comment:创建人" json:"created_by"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (PipelineTemplate) TableName() string { return "pipeline_templates" }

// PipelineTemplateRating 模板评分
type PipelineTemplateRating struct {
	ID         uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	TemplateID uint64    `gorm:"index;not null;comment:模板ID" json:"template_id"`
	UserID     uint      `gorm:"index;not null;comment:用户ID" json:"user_id"`
	Rating     int       `gorm:"not null;comment:评分(1-5)" json:"rating"`
	Comment    string    `gorm:"size:500;comment:评价" json:"comment"`
	CreatedAt  time.Time `json:"created_at"`
}

func (PipelineTemplateRating) TableName() string { return "pipeline_template_ratings" }

// PipelineStageTemplate 阶段模板（用于拖拽式设计）
type PipelineStageTemplate struct {
	ID          uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	Name        string    `gorm:"size:100;not null;comment:阶段名称" json:"name"`
	Description string    `gorm:"size:500;comment:阶段描述" json:"description"`
	Category    string    `gorm:"size:50;index;comment:分类" json:"category"` // source, build, test, deploy, notify
	IconName    string    `gorm:"size:50;comment:图标名称" json:"icon_name"`
	Color       string    `gorm:"size:20;comment:颜色" json:"color"`
	ConfigJSON  string    `gorm:"type:json;comment:默认配置" json:"config_json"`
	IsBuiltin   bool      `gorm:"default:true;comment:是否内置" json:"is_builtin"`
	SortOrder   int       `gorm:"default:0;comment:排序" json:"sort_order"`
	CreatedAt   time.Time `json:"created_at"`
}

func (PipelineStageTemplate) TableName() string { return "pipeline_stage_templates" }

// PipelineStepTemplate 步骤模板（用于拖拽式设计）
type PipelineStepTemplate struct {
	ID           uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	Name         string    `gorm:"size:100;not null;comment:步骤名称" json:"name"`
	Description  string    `gorm:"size:500;comment:步骤描述" json:"description"`
	StepType     string    `gorm:"size:50;not null;index;comment:步骤类型" json:"step_type"` // git, shell, docker_build, k8s_deploy, etc.
	Category     string    `gorm:"size:50;index;comment:分类" json:"category"`
	IconName     string    `gorm:"size:50;comment:图标名称" json:"icon_name"`
	ConfigSchema string    `gorm:"type:json;comment:配置Schema" json:"config_schema"` // JSON Schema 定义配置项
	DefaultJSON  string    `gorm:"type:json;comment:默认配置" json:"default_json"`
	IsBuiltin    bool      `gorm:"default:true;comment:是否内置" json:"is_builtin"`
	SortOrder    int       `gorm:"default:0;comment:排序" json:"sort_order"`
	CreatedAt    time.Time `json:"created_at"`
}

func (PipelineStepTemplate) TableName() string { return "pipeline_step_templates" }

// BuiltinPipelineTemplates 内置流水线模板
var BuiltinPipelineTemplates = []PipelineTemplate{
	{
		Name:        "java-maven-k8s",
		Description: "Java Maven 项目构建并部署到 K8s",
		Category:    "build",
		Language:    "java",
		Framework:   "spring",
		ConfigJSON: `{
			"stages": [
				{
					"id": "checkout",
					"name": "代码检出",
					"steps": [
						{"id": "git", "name": "Git Clone", "type": "git", "config": {}}
					]
				},
				{
					"id": "build",
					"name": "编译构建",
					"steps": [
						{"id": "maven", "name": "Maven Build", "type": "container", "config": {"image": "maven:3.8-openjdk-17", "commands": ["mvn clean package -DskipTests"]}}
					]
				},
				{
					"id": "test",
					"name": "单元测试",
					"steps": [
						{"id": "test", "name": "Maven Test", "type": "container", "config": {"image": "maven:3.8-openjdk-17", "commands": ["mvn test"]}}
					]
				},
				{
					"id": "docker",
					"name": "镜像构建",
					"steps": [
						{"id": "docker_build", "name": "Docker Build", "type": "docker_build", "config": {}},
						{"id": "docker_push", "name": "Docker Push", "type": "docker_push", "config": {}}
					]
				},
				{
					"id": "deploy",
					"name": "部署",
					"steps": [
						{"id": "k8s_deploy", "name": "K8s Deploy", "type": "k8s_deploy", "config": {}}
					]
				}
			],
			"variables": [
				{"name": "MAVEN_OPTS", "value": "-Xmx512m"},
				{"name": "DOCKER_REGISTRY", "value": ""}
			]
		}`,
		IsBuiltin: true,
		IsPublic:  true,
	},
	{
		Name:        "go-k8s",
		Description: "Go 项目构建并部署到 K8s",
		Category:    "build",
		Language:    "go",
		Framework:   "gin",
		ConfigJSON: `{
			"stages": [
				{
					"id": "checkout",
					"name": "代码检出",
					"steps": [
						{"id": "git", "name": "Git Clone", "type": "git", "config": {}}
					]
				},
				{
					"id": "build",
					"name": "编译构建",
					"steps": [
						{"id": "go_build", "name": "Go Build", "type": "container", "config": {"image": "golang:1.21-alpine", "commands": ["go mod download", "CGO_ENABLED=0 go build -o app ./cmd/server"]}}
					]
				},
				{
					"id": "test",
					"name": "单元测试",
					"steps": [
						{"id": "go_test", "name": "Go Test", "type": "container", "config": {"image": "golang:1.21-alpine", "commands": ["go test -v ./..."]}}
					]
				},
				{
					"id": "docker",
					"name": "镜像构建",
					"steps": [
						{"id": "docker_build", "name": "Docker Build", "type": "docker_build", "config": {}},
						{"id": "docker_push", "name": "Docker Push", "type": "docker_push", "config": {}}
					]
				},
				{
					"id": "deploy",
					"name": "部署",
					"steps": [
						{"id": "k8s_deploy", "name": "K8s Deploy", "type": "k8s_deploy", "config": {}}
					]
				}
			],
			"variables": [
				{"name": "GOPROXY", "value": "https://goproxy.cn,direct"},
				{"name": "DOCKER_REGISTRY", "value": ""}
			]
		}`,
		IsBuiltin: true,
		IsPublic:  true,
	},
	{
		Name:        "nodejs-k8s",
		Description: "Node.js 项目构建并部署到 K8s",
		Category:    "build",
		Language:    "nodejs",
		Framework:   "express",
		ConfigJSON: `{
			"stages": [
				{
					"id": "checkout",
					"name": "代码检出",
					"steps": [
						{"id": "git", "name": "Git Clone", "type": "git", "config": {}}
					]
				},
				{
					"id": "install",
					"name": "安装依赖",
					"steps": [
						{"id": "npm_install", "name": "NPM Install", "type": "container", "config": {"image": "node:18-alpine", "commands": ["npm ci"]}}
					]
				},
				{
					"id": "build",
					"name": "编译构建",
					"steps": [
						{"id": "npm_build", "name": "NPM Build", "type": "container", "config": {"image": "node:18-alpine", "commands": ["npm run build"]}}
					]
				},
				{
					"id": "test",
					"name": "单元测试",
					"steps": [
						{"id": "npm_test", "name": "NPM Test", "type": "container", "config": {"image": "node:18-alpine", "commands": ["npm test"]}}
					]
				},
				{
					"id": "docker",
					"name": "镜像构建",
					"steps": [
						{"id": "docker_build", "name": "Docker Build", "type": "docker_build", "config": {}},
						{"id": "docker_push", "name": "Docker Push", "type": "docker_push", "config": {}}
					]
				},
				{
					"id": "deploy",
					"name": "部署",
					"steps": [
						{"id": "k8s_deploy", "name": "K8s Deploy", "type": "k8s_deploy", "config": {}}
					]
				}
			],
			"variables": [
				{"name": "NPM_REGISTRY", "value": "https://registry.npmmirror.com"},
				{"name": "DOCKER_REGISTRY", "value": ""}
			]
		}`,
		IsBuiltin: true,
		IsPublic:  true,
	},
	{
		Name:        "python-k8s",
		Description: "Python 项目构建并部署到 K8s",
		Category:    "build",
		Language:    "python",
		Framework:   "django",
		ConfigJSON: `{
			"stages": [
				{
					"id": "checkout",
					"name": "代码检出",
					"steps": [
						{"id": "git", "name": "Git Clone", "type": "git", "config": {}}
					]
				},
				{
					"id": "install",
					"name": "安装依赖",
					"steps": [
						{"id": "pip_install", "name": "Pip Install", "type": "container", "config": {"image": "python:3.11-slim", "commands": ["pip install -r requirements.txt"]}}
					]
				},
				{
					"id": "test",
					"name": "单元测试",
					"steps": [
						{"id": "pytest", "name": "Pytest", "type": "container", "config": {"image": "python:3.11-slim", "commands": ["pytest"]}}
					]
				},
				{
					"id": "docker",
					"name": "镜像构建",
					"steps": [
						{"id": "docker_build", "name": "Docker Build", "type": "docker_build", "config": {}},
						{"id": "docker_push", "name": "Docker Push", "type": "docker_push", "config": {}}
					]
				},
				{
					"id": "deploy",
					"name": "部署",
					"steps": [
						{"id": "k8s_deploy", "name": "K8s Deploy", "type": "k8s_deploy", "config": {}}
					]
				}
			],
			"variables": [
				{"name": "PIP_INDEX_URL", "value": "https://pypi.tuna.tsinghua.edu.cn/simple"},
				{"name": "DOCKER_REGISTRY", "value": ""}
			]
		}`,
		IsBuiltin: true,
		IsPublic:  true,
	},
	{
		Name:        "frontend-cdn",
		Description: "前端项目构建并部署到 CDN",
		Category:    "build",
		Language:    "nodejs",
		Framework:   "vue",
		ConfigJSON: `{
			"stages": [
				{
					"id": "checkout",
					"name": "代码检出",
					"steps": [
						{"id": "git", "name": "Git Clone", "type": "git", "config": {}}
					]
				},
				{
					"id": "install",
					"name": "安装依赖",
					"steps": [
						{"id": "npm_install", "name": "NPM Install", "type": "container", "config": {"image": "node:18-alpine", "commands": ["npm ci"]}}
					]
				},
				{
					"id": "build",
					"name": "编译构建",
					"steps": [
						{"id": "npm_build", "name": "NPM Build", "type": "container", "config": {"image": "node:18-alpine", "commands": ["npm run build"]}}
					]
				},
				{
					"id": "deploy",
					"name": "部署CDN",
					"steps": [
						{"id": "upload", "name": "Upload to CDN", "type": "shell", "config": {"commands": ["echo 'Upload to CDN'"]}}
					]
				}
			],
			"variables": [
				{"name": "NPM_REGISTRY", "value": "https://registry.npmmirror.com"},
				{"name": "CDN_BUCKET", "value": ""}
			]
		}`,
		IsBuiltin: true,
		IsPublic:  true,
	},
}

// BuiltinStageTemplates 内置阶段模板
var BuiltinStageTemplates = []PipelineStageTemplate{
	{Name: "代码检出", Description: "从 Git 仓库检出代码", Category: "source", IconName: "git", Color: "#f05032", SortOrder: 1, IsBuiltin: true},
	{Name: "编译构建", Description: "编译构建项目", Category: "build", IconName: "build", Color: "#4caf50", SortOrder: 2, IsBuiltin: true},
	{Name: "单元测试", Description: "运行单元测试", Category: "test", IconName: "test", Color: "#2196f3", SortOrder: 3, IsBuiltin: true},
	{Name: "代码扫描", Description: "代码质量扫描", Category: "test", IconName: "scan", Color: "#9c27b0", SortOrder: 4, IsBuiltin: true},
	{Name: "镜像构建", Description: "构建 Docker 镜像", Category: "build", IconName: "docker", Color: "#2496ed", SortOrder: 5, IsBuiltin: true},
	{Name: "镜像推送", Description: "推送镜像到仓库", Category: "build", IconName: "upload", Color: "#ff9800", SortOrder: 6, IsBuiltin: true},
	{Name: "部署", Description: "部署到目标环境", Category: "deploy", IconName: "deploy", Color: "#e91e63", SortOrder: 7, IsBuiltin: true},
	{Name: "通知", Description: "发送通知消息", Category: "notify", IconName: "notify", Color: "#607d8b", SortOrder: 8, IsBuiltin: true},
}

// BuiltinStepTemplates 内置步骤模板
var BuiltinStepTemplates = []PipelineStepTemplate{
	{
		Name:        "Git Clone",
		Description: "从 Git 仓库克隆代码",
		StepType:    "git",
		Category:    "source",
		IconName:    "git",
		ConfigSchema: `{
			"type": "object",
			"properties": {
				"branch": {"type": "string", "title": "分支", "default": "main"},
				"depth": {"type": "integer", "title": "克隆深度", "default": 1}
			}
		}`,
		IsBuiltin: true,
		SortOrder: 1,
	},
	{
		Name:        "Shell 命令",
		Description: "执行 Shell 命令",
		StepType:    "shell",
		Category:    "build",
		IconName:    "terminal",
		ConfigSchema: `{
			"type": "object",
			"properties": {
				"commands": {"type": "array", "items": {"type": "string"}, "title": "命令列表"}
			},
			"required": ["commands"]
		}`,
		IsBuiltin: true,
		SortOrder: 2,
	},
	{
		Name:        "容器执行",
		Description: "在容器中执行命令",
		StepType:    "container",
		Category:    "build",
		IconName:    "container",
		ConfigSchema: `{
			"type": "object",
			"properties": {
				"image": {"type": "string", "title": "镜像"},
				"commands": {"type": "array", "items": {"type": "string"}, "title": "命令列表"},
				"work_dir": {"type": "string", "title": "工作目录"},
				"env": {"type": "object", "title": "环境变量"}
			},
			"required": ["image", "commands"]
		}`,
		IsBuiltin: true,
		SortOrder: 3,
	},
	{
		Name:        "Docker Build",
		Description: "构建 Docker 镜像",
		StepType:    "docker_build",
		Category:    "build",
		IconName:    "docker",
		ConfigSchema: `{
			"type": "object",
			"properties": {
				"dockerfile": {"type": "string", "title": "Dockerfile 路径", "default": "Dockerfile"},
				"context": {"type": "string", "title": "构建上下文", "default": "."},
				"tags": {"type": "array", "items": {"type": "string"}, "title": "镜像标签"}
			}
		}`,
		IsBuiltin: true,
		SortOrder: 4,
	},
	{
		Name:        "Docker Push",
		Description: "推送 Docker 镜像",
		StepType:    "docker_push",
		Category:    "build",
		IconName:    "upload",
		ConfigSchema: `{
			"type": "object",
			"properties": {
				"registry": {"type": "string", "title": "镜像仓库"},
				"image": {"type": "string", "title": "镜像名称"},
				"tag": {"type": "string", "title": "镜像标签"}
			}
		}`,
		IsBuiltin: true,
		SortOrder: 5,
	},
	{
		Name:        "K8s 部署",
		Description: "部署到 Kubernetes",
		StepType:    "k8s_deploy",
		Category:    "deploy",
		IconName:    "kubernetes",
		ConfigSchema: `{
			"type": "object",
			"properties": {
				"cluster_id": {"type": "integer", "title": "集群ID"},
				"namespace": {"type": "string", "title": "命名空间"},
				"deployment": {"type": "string", "title": "Deployment名称"},
				"image": {"type": "string", "title": "镜像"}
			}
		}`,
		IsBuiltin: true,
		SortOrder: 6,
	},
	{
		Name:        "通知",
		Description: "发送通知消息",
		StepType:    "notify",
		Category:    "notify",
		IconName:    "notify",
		ConfigSchema: `{
			"type": "object",
			"properties": {
				"type": {"type": "string", "title": "通知类型", "enum": ["feishu", "dingtalk", "wechat", "email"]},
				"webhook": {"type": "string", "title": "Webhook URL"},
				"message": {"type": "string", "title": "消息内容"}
			}
		}`,
		IsBuiltin: true,
		SortOrder: 7,
	},
}
