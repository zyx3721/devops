// Package models 定义数据库模型
//
// 本包包含 DevOps 平台的所有数据库模型定义，按功能领域拆分为多个子包：
//
// 子包结构:
//   - notification/  - 消息通知模型（飞书、钉钉、企业微信）
//   - infrastructure/ - 基础设施模型（Jenkins、K8s、CronHPA）
//   - deploy/        - 部署流程模型（部署记录、审批、流水线）
//   - monitoring/    - 监控告警模型（告警、健康检查、日志、成本）
//   - traffic/       - 流量治理模型（限流、熔断、路由、负载均衡）
//   - system/        - 系统管理模型（用户、RBAC、权限、审计）
//   - application/   - 应用管理模型（应用、环境配置）
//
// 向后兼容:
//
//	本包提供类型别名，允许继续使用 models.TypeName 的方式访问类型。
//	新代码建议直接导入子包使用。
//
// 使用示例:
//
//	// 方式1: 使用类型别名（向后兼容）
//	import "devops/internal/models"
//	user := &models.User{Username: "admin"}
//
//	// 方式2: 直接使用子包（推荐）
//	import "devops/internal/models/system"
//	user := &system.User{Username: "admin"}
package models

import (
	"devops/internal/domain/notification/model"
	"devops/internal/models/ai"
	"devops/internal/models/application"
	"devops/internal/models/deploy"
	"devops/internal/models/infrastructure"
	"devops/internal/models/monitoring"
	"devops/internal/models/system"
	"devops/internal/models/traffic"
)

// ==================== 通知领域类型别名 ====================

// FeishuBot 飞书机器人 (别名)
type FeishuBot = model.FeishuBot

// FeishuApp 飞书应用 (别名)
type FeishuApp = model.FeishuApp

// FeishuRequest 飞书请求 (别名)
type FeishuRequest = model.FeishuRequest

// FeishuUserToken 飞书用户令牌 (别名)
type FeishuUserToken = model.FeishuUserToken

// FeishuMessageLog 飞书消息日志 (别名)
type FeishuMessageLog = model.FeishuMessageLog

// DingtalkApp 钉钉应用 (别名)
type DingtalkApp = model.DingtalkApp

// DingtalkBot 钉钉机器人 (别名)
type DingtalkBot = model.DingtalkBot

// DingtalkMessageLog 钉钉消息日志 (别名)
type DingtalkMessageLog = model.DingtalkMessageLog

// WechatWorkApp 企业微信应用 (别名)
type WechatWorkApp = model.WechatWorkApp

// WechatWorkBot 企业微信机器人 (别名)
type WechatWorkBot = model.WechatWorkBot

// WechatWorkMessageLog 企业微信消息日志 (别名)
type WechatWorkMessageLog = model.WechatWorkMessageLog

// ==================== 基础设施领域类型别名 ====================
type JenkinsInstance = infrastructure.JenkinsInstance

// JenkinsFeishuApp Jenkins飞书应用关联 (别名)
type JenkinsFeishuApp = infrastructure.JenkinsFeishuApp

// JenkinsDingtalkApp Jenkins钉钉应用关联 (别名)
type JenkinsDingtalkApp = infrastructure.JenkinsDingtalkApp

// JenkinsWechatWorkApp Jenkins企业微信应用关联 (别名)
type JenkinsWechatWorkApp = infrastructure.JenkinsWechatWorkApp

// K8sCluster K8s集群 (别名)
type K8sCluster = infrastructure.K8sCluster

// K8sClusterFeishuApp K8s集群飞书应用关联 (别名)
type K8sClusterFeishuApp = infrastructure.K8sClusterFeishuApp

// K8sClusterDingtalkApp K8s集群钉钉应用关联 (别名)
type K8sClusterDingtalkApp = infrastructure.K8sClusterDingtalkApp

// K8sClusterWechatWorkApp K8s集群企业微信应用关联 (别名)
type K8sClusterWechatWorkApp = infrastructure.K8sClusterWechatWorkApp

// CronHPA 定时HPA (别名)
type CronHPA = infrastructure.CronHPA

// ==================== 部署领域类型别名 ====================

// DeployRecord 部署记录 (别名)
type DeployRecord = deploy.DeployRecord

// DeployLock 部署锁 (别名)
type DeployLock = deploy.DeployLock

// DeployWindow 部署窗口 (别名)
type DeployWindow = deploy.DeployWindow

// Task 任务 (别名)
type Task = deploy.Task

// ApprovalRule 审批规则 (别名)
type ApprovalRule = deploy.ApprovalRule

// ApprovalRecord 审批记录 (别名)
type ApprovalRecord = deploy.ApprovalRecord

// ApprovalChain 审批链 (别名)
type ApprovalChain = deploy.ApprovalChain

// ApprovalNode 审批节点 (别名)
type ApprovalNode = deploy.ApprovalNode

// ApprovalInstance 审批实例 (别名)
type ApprovalInstance = deploy.ApprovalInstance

// ApprovalNodeInstance 审批节点实例 (别名)
type ApprovalNodeInstance = deploy.ApprovalNodeInstance

// ApprovalAction 审批动作 (别名)
type ApprovalAction = deploy.ApprovalAction

// Pipeline 流水线 (别名)
type Pipeline = deploy.Pipeline

// PipelineRun 流水线运行记录 (别名)
type PipelineRun = deploy.PipelineRun

// StageRun 阶段运行记录 (别名)
type StageRun = deploy.StageRun

// StepRun 步骤运行记录 (别名)
type StepRun = deploy.StepRun

// PipelineTemplate 流水线模板 (别名)
type PipelineTemplate = deploy.PipelineTemplate

// PipelineCredential 流水线凭证 (别名)
type PipelineCredential = deploy.PipelineCredential

// PipelineVariable 流水线变量 (别名)
type PipelineVariable = deploy.PipelineVariable

// GitRepository Git仓库 (别名)
type GitRepository = deploy.GitRepository

// BuildJob 构建任务 (别名)
type BuildJob = deploy.BuildJob

// Artifact 构建制品 (别名)
type Artifact = deploy.Artifact

// BuildCache 构建缓存 (别名)
type BuildCache = deploy.BuildCache

// BuildWorkspace 构建工作空间 (别名)
type BuildWorkspace = deploy.BuildWorkspace

// WebhookLog Webhook日志 (别名)
type WebhookLog = deploy.WebhookLog

// ArtifactRegistry 制品库 (别名)
type ArtifactRegistry = deploy.ArtifactRegistry

// ==================== 监控领域类型别名 ====================

// AlertConfig 告警配置 (别名)
type AlertConfig = monitoring.AlertConfig

// AlertHistory 告警历史 (别名)
type AlertHistory = monitoring.AlertHistory

// AlertSilence 告警静默规则 (别名)
type AlertSilence = monitoring.AlertSilence

// AlertEscalation 告警升级规则 (别名)
type AlertEscalation = monitoring.AlertEscalation

// AlertEscalationLog 告警升级记录 (别名)
type AlertEscalationLog = monitoring.AlertEscalationLog

// HealthCheckConfig 健康检查配置 (别名)
type HealthCheckConfig = monitoring.HealthCheckConfig

// HealthCheckHistory 健康检查历史 (别名)
type HealthCheckHistory = monitoring.HealthCheckHistory

// LogAlertRule 日志告警规则 (别名)
type LogAlertRule = monitoring.LogAlertRule

// LogAlertHistory 日志告警历史 (别名)
type LogAlertHistory = monitoring.LogAlertHistory

// LogHighlightRule 日志染色规则 (别名)
type LogHighlightRule = monitoring.LogHighlightRule

// LogParseTemplate 日志解析模板 (别名)
type LogParseTemplate = monitoring.LogParseTemplate

// LogDataSource 日志数据源 (别名)
type LogDataSource = monitoring.LogDataSource

// LogBookmark 日志书签 (别名)
type LogBookmark = monitoring.LogBookmark

// LogSavedQuery 日志快捷查询 (别名)
type LogSavedQuery = monitoring.LogSavedQuery

// JSONObject JSON对象 (别名)
type JSONObject = monitoring.JSONObject

// ParseField 解析字段 (别名)
type ParseField = monitoring.ParseField

// ResourceCost 资源成本 (别名)
type ResourceCost = monitoring.ResourceCost

// CostSummary 成本汇总 (别名)
type CostSummary = monitoring.CostSummary

// CostSuggestion 成本优化建议 (别名)
type CostSuggestion = monitoring.CostSuggestion

// CostConfig 成本配置 (别名)
type CostConfig = monitoring.CostConfig

// CostBudget 成本预算 (别名)
type CostBudget = monitoring.CostBudget

// CostAlert 成本告警 (别名)
type CostAlert = monitoring.CostAlert

// ResourceActivity 资源活跃度 (别名)
type ResourceActivity = monitoring.ResourceActivity

// ==================== 流量治理领域类型别名 ====================

// JSONDestinations 路由目标配置 (别名)
type JSONDestinations = traffic.JSONDestinations

// RouteDestination 路由目标 (别名)
type RouteDestination = traffic.RouteDestination

// JSONArray JSON数组 (别名) - 使用 monitoring 包的定义以兼容日志服务
type JSONArray = monitoring.JSONArray

// TrafficRateLimitRule 限流规则 (别名)
type TrafficRateLimitRule = traffic.TrafficRateLimitRule

// TrafficCircuitBreakerRule 熔断规则 (别名)
type TrafficCircuitBreakerRule = traffic.TrafficCircuitBreakerRule

// TrafficRoutingRule 路由规则 (别名)
type TrafficRoutingRule = traffic.TrafficRoutingRule

// TrafficLoadBalanceConfig 负载均衡配置 (别名)
type TrafficLoadBalanceConfig = traffic.TrafficLoadBalanceConfig

// TrafficTimeoutConfig 超时配置 (别名)
type TrafficTimeoutConfig = traffic.TrafficTimeoutConfig

// TrafficMirrorRule 流量镜像规则 (别名)
type TrafficMirrorRule = traffic.TrafficMirrorRule

// TrafficFaultRule 故障注入规则 (别名)
type TrafficFaultRule = traffic.TrafficFaultRule

// TrafficOperationLog 流量操作日志 (别名)
type TrafficOperationLog = traffic.TrafficOperationLog

// TrafficStatistics 流量统计 (别名)
type TrafficStatistics = traffic.TrafficStatistics

// TrafficRuleVersion 规则版本 (别名)
type TrafficRuleVersion = traffic.TrafficRuleVersion

// TrafficRuleTemplate 规则模板 (别名)
type TrafficRuleTemplate = traffic.TrafficRuleTemplate

// CanaryRelease 金丝雀发布 (别名)
type CanaryRelease = traffic.CanaryRelease

// BlueGreenDeployment 蓝绿部署 (别名)
type BlueGreenDeployment = traffic.BlueGreenDeployment

// AppRateLimitRule 应用限流规则 (别名)
type AppRateLimitRule = traffic.AppRateLimitRule

// AppMirrorRule 应用镜像规则 (别名)
type AppMirrorRule = traffic.AppMirrorRule

// AppFaultRule 应用故障注入规则 (别名)
type AppFaultRule = traffic.AppFaultRule

// ==================== 系统管理领域类型别名 ====================

// User 用户 (别名)
type User = system.User

// Role 角色 (别名)
type Role = system.Role

// Permission 权限 (别名)
type Permission = system.Permission

// RolePermission 角色权限关联 (别名)
type RolePermission = system.RolePermission

// UserRole 用户角色关联 (别名)
type UserRole = system.UserRole

// AuditLog 审计日志 (别名)
type AuditLog = system.AuditLog

// OAData OA数据 (别名)
type OAData = system.OAData

// OAAddress OA地址 (别名)
type OAAddress = system.OAAddress

// OANotifyConfig OA通知配置 (别名)
type OANotifyConfig = system.OANotifyConfig

// SystemConfig 系统配置 (别名)
type SystemConfig = system.SystemConfig

// MessageTemplate 消息模板 (别名)
type MessageTemplate = system.MessageTemplate

// ImageRegistry 镜像仓库 (别名)
type ImageRegistry = system.ImageRegistry

// ImageScan 镜像扫描 (别名)
type ImageScan = system.ImageScan

// ComplianceRule 合规规则 (别名)
type ComplianceRule = system.ComplianceRule

// ConfigCheck 配置检查 (别名)
type ConfigCheck = system.ConfigCheck

// SecurityAuditLog 安全审计日志 (别名)
type SecurityAuditLog = system.SecurityAuditLog

// SecurityReport 安全报告 (别名)
type SecurityReport = system.SecurityReport

// ==================== 应用管理领域类型别名 ====================

// Application 应用 (别名)
type Application = application.Application

// ApplicationEnv 应用环境 (别名)
type ApplicationEnv = application.ApplicationEnv

// ==================== 权限常量和函数别名 ====================

// 角色常量
const (
	RoleSuperAdmin = system.RoleSuperAdmin
	RoleAdmin      = system.RoleAdmin
	RoleUser       = system.RoleUser
	RoleGuest      = system.RoleGuest
)

// 权限常量
const (
	PermUserView   = system.PermUserView
	PermUserCreate = system.PermUserCreate
	PermUserUpdate = system.PermUserUpdate
	PermUserDelete = system.PermUserDelete
	PermUserRole   = system.PermUserRole
	PermUserStatus = system.PermUserStatus

	PermAppView   = system.PermAppView
	PermAppCreate = system.PermAppCreate
	PermAppUpdate = system.PermAppUpdate
	PermAppDelete = system.PermAppDelete
	PermAppDeploy = system.PermAppDeploy

	PermApprovalView   = system.PermApprovalView
	PermApprovalCreate = system.PermApprovalCreate
	PermApprovalUpdate = system.PermApprovalUpdate
	PermApprovalDelete = system.PermApprovalDelete

	PermK8sView   = system.PermK8sView
	PermK8sCreate = system.PermK8sCreate
	PermK8sUpdate = system.PermK8sUpdate
	PermK8sDelete = system.PermK8sDelete
	PermK8sExec   = system.PermK8sExec

	PermJenkinsView    = system.PermJenkinsView
	PermJenkinsCreate  = system.PermJenkinsCreate
	PermJenkinsUpdate  = system.PermJenkinsUpdate
	PermJenkinsDelete  = system.PermJenkinsDelete
	PermJenkinsTrigger = system.PermJenkinsTrigger

	PermSystemView   = system.PermSystemView
	PermSystemUpdate = system.PermSystemUpdate
)

// RolePermissions 角色权限映射 (别名)
var RolePermissions = system.RolePermissions

// GetRoleLevel 获取角色等级 (别名)
var GetRoleLevel = system.GetRoleLevel

// CanManageRole 检查是否可以管理目标角色 (别名)
var CanManageRole = system.CanManageRole

// HasPermission 检查角色是否有某个权限 (别名)
var HasPermission = system.HasPermission

// IsSuperAdmin 检查是否是超级管理员 (别名)
var IsSuperAdmin = system.IsSuperAdmin

// IsProtectedUser 检查用户是否受保护 (别名)
var IsProtectedUser = system.IsProtectedUser

// ==================== AI Copilot 领域类型别名 ====================

// AIConversation AI会话 (别名)
type AIConversation = ai.AIConversation

// AIMessage AI消息 (别名)
type AIMessage = ai.AIMessage

// AIKnowledge AI知识库 (别名)
type AIKnowledge = ai.AIKnowledge

// AIOperationLog AI操作日志 (别名)
type AIOperationLog = ai.AIOperationLog

// AILLMConfig AI LLM配置 (别名)
type AILLMConfig = ai.AILLMConfig

// AIMessageFeedback AI消息反馈 (别名)
type AIMessageFeedback = ai.AIMessageFeedback

// PageContext 页面上下文 (别名)
type PageContext = ai.PageContext

// MessageRole 消息角色 (别名)
type MessageRole = ai.MessageRole

// MessageStatus 消息状态 (别名)
type MessageStatus = ai.MessageStatus

// KnowledgeCategory 知识分类 (别名)
type KnowledgeCategory = ai.KnowledgeCategory

// OperationAction 操作类型 (别名)
type OperationAction = ai.OperationAction

// LLMProvider LLM提供商 (别名)
type LLMProvider = ai.LLMProvider

// AI 消息角色常量
const (
	RoleUserAI      = ai.RoleUser
	RoleAssistantAI = ai.RoleAssistant
	RoleSystemAI    = ai.RoleSystem
	RoleToolAI      = ai.RoleTool
)

// AI 消息状态常量
const (
	StatusPending   = ai.StatusPending
	StatusStreaming = ai.StatusStreaming
	StatusComplete  = ai.StatusComplete
	StatusError     = ai.StatusError
)

// AI 知识分类常量
const (
	CategoryApplication = ai.CategoryApplication
	CategoryTraffic     = ai.CategoryTraffic
	CategoryApproval    = ai.CategoryApproval
	CategoryK8s         = ai.CategoryK8s
	CategoryMonitoring  = ai.CategoryMonitoring
	CategoryCICD        = ai.CategoryCICD
	CategoryGeneral     = ai.CategoryGeneral
)

// AI 操作类型常量
const (
	ActionQueryLogs      = ai.ActionQueryLogs
	ActionQueryAlerts    = ai.ActionQueryAlerts
	ActionQueryMetrics   = ai.ActionQueryMetrics
	ActionRestartApp     = ai.ActionRestartApp
	ActionScalePod       = ai.ActionScalePod
	ActionRollback       = ai.ActionRollback
	ActionSilenceAlert   = ai.ActionSilenceAlert
	ActionQueryKnowledge = ai.ActionQueryKnowledge
)

// AI LLM提供商常量
const (
	ProviderOpenAI = ai.ProviderOpenAI
	ProviderAzure  = ai.ProviderAzure
	ProviderQwen   = ai.ProviderQwen
	ProviderZhipu  = ai.ProviderZhipu
	ProviderOllama = ai.ProviderOllama
)

// GetActionName 获取操作名称 (别名)
var GetActionName = ai.GetActionName

// IsDangerousAction 判断是否为危险操作 (别名)
var IsDangerousAction = ai.IsDangerousAction
