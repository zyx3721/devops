// Package repository 提供向后兼容的repository导入
// 这是一个临时的兼容层，用于在重构期间保持旧的import路径工作
package repository

import (
	// Auth模块
	authRepo "devops/internal/modules/auth/repository"
	// Application模块
	appRepo "devops/internal/modules/application/repository"
	// Approval模块
	approvalRepo "devops/internal/modules/approval/repository"
	// Infrastructure模块
	infraRepo "devops/internal/modules/infrastructure/repository"
	// Notification模块
	notificationRepo "devops/internal/domain/notification/repository"
	// Monitoring模块
	monitoringRepo "devops/internal/modules/monitoring/repository"
	// System模块
	systemRepo "devops/internal/modules/system/repository"
)

// Auth模块类型别名
type (
	RoleRepository           = authRepo.RoleRepository
	PermissionRepository     = authRepo.PermissionRepository
	RolePermissionRepository = authRepo.RolePermissionRepository
	UserRoleRepository       = authRepo.UserRoleRepository
	UserRepository           = authRepo.UserRepository
)

// Auth模块函数别名
var (
	NewRoleRepository           = authRepo.NewRoleRepository
	NewPermissionRepository     = authRepo.NewPermissionRepository
	NewRolePermissionRepository = authRepo.NewRolePermissionRepository
	NewUserRoleRepository       = authRepo.NewUserRoleRepository
	NewUserRepository           = authRepo.NewUserRepository
)

// Application模块类型别名
type (
	ApplicationRepository    = appRepo.ApplicationRepository
	ApplicationFilter        = appRepo.ApplicationFilter
	ApplicationEnvRepository = appRepo.ApplicationEnvRepository
	DeployRecordRepository   = appRepo.DeployRecordRepository
	DeployRecordFilter       = appRepo.DeployRecordFilter
	DeployStatsFilter        = appRepo.DeployStatsFilter
	DeployStats              = appRepo.DeployStats
	DeployLockRepository     = appRepo.DeployLockRepository
	ApprovalRecordRepository = appRepo.ApprovalRecordRepository
)

// Application模块函数别名
var (
	NewApplicationRepository    = appRepo.NewApplicationRepository
	NewApplicationEnvRepository = appRepo.NewApplicationEnvRepository
	NewDeployRecordRepository   = appRepo.NewDeployRecordRepository
	NewDeployLockRepository     = appRepo.NewDeployLockRepository
	NewApprovalRecordRepository = appRepo.NewApprovalRecordRepository
)

// Approval模块类型别名
type (
	ApprovalChainRepository        = approvalRepo.ApprovalChainRepository
	ApprovalInstanceRepository     = approvalRepo.ApprovalInstanceRepository
	ApprovalNodeRepository         = approvalRepo.ApprovalNodeRepository
	ApprovalNodeInstanceRepository = approvalRepo.ApprovalNodeInstanceRepository
	ApprovalActionRepository       = approvalRepo.ApprovalActionRepository
	ApprovalRuleRepository         = approvalRepo.ApprovalRuleRepository
	DeployWindowRepository         = approvalRepo.DeployWindowRepository
	ChainFilter                    = approvalRepo.ChainFilter
	InstanceFilter                 = approvalRepo.InstanceFilter
)

// Approval模块函数别名
var (
	NewApprovalChainRepository        = approvalRepo.NewApprovalChainRepository
	NewApprovalInstanceRepository     = approvalRepo.NewApprovalInstanceRepository
	NewApprovalNodeRepository         = approvalRepo.NewApprovalNodeRepository
	NewApprovalNodeInstanceRepository = approvalRepo.NewApprovalNodeInstanceRepository
	NewApprovalActionRepository       = approvalRepo.NewApprovalActionRepository
	NewApprovalRuleRepository         = approvalRepo.NewApprovalRuleRepository
	NewDeployWindowRepository         = approvalRepo.NewDeployWindowRepository
)

// Infrastructure模块类型别名
type (
	JenkinsInstanceRepository = infraRepo.JenkinsInstanceRepository
	K8sClusterRepository      = infraRepo.K8sClusterRepository
)

// Infrastructure模块函数别名
var (
	NewJenkinsInstanceRepository = infraRepo.NewJenkinsInstanceRepository
	NewK8sClusterRepository      = infraRepo.NewK8sClusterRepository
)

// Notification模块类型别名
type (
	FeishuRequestRepository        = notificationRepo.FeishuRequestRepository
	FeishuAppRepository            = notificationRepo.FeishuAppRepository
	FeishuBotRepository            = notificationRepo.FeishuBotRepository
	FeishuMessageLogRepository     = notificationRepo.FeishuMessageLogRepository
	FeishuUserTokenRepository      = notificationRepo.FeishuUserTokenRepository
	DingtalkAppRepository          = notificationRepo.DingtalkAppRepository
	DingtalkBotRepository          = notificationRepo.DingtalkBotRepository
	DingtalkMessageLogRepository   = notificationRepo.DingtalkMessageLogRepository
	WechatWorkAppRepository        = notificationRepo.WechatWorkAppRepository
	WechatWorkBotRepository        = notificationRepo.WechatWorkBotRepository
	WechatWorkMessageLogRepository = notificationRepo.WechatWorkMessageLogRepository
)

// Notification模块函数别名
var (
	NewFeishuRequestRepository        = notificationRepo.NewFeishuRequestRepository
	NewFeishuAppRepository            = notificationRepo.NewFeishuAppRepository
	NewFeishuBotRepository            = notificationRepo.NewFeishuBotRepository
	NewFeishuMessageLogRepository     = notificationRepo.NewFeishuMessageLogRepository
	NewFeishuUserTokenRepository      = notificationRepo.NewFeishuUserTokenRepository
	NewDingtalkAppRepository          = notificationRepo.NewDingtalkAppRepository
	NewDingtalkBotRepository          = notificationRepo.NewDingtalkBotRepository
	NewDingtalkMessageLogRepository   = notificationRepo.NewDingtalkMessageLogRepository
	NewWechatWorkAppRepository        = notificationRepo.NewWechatWorkAppRepository
	NewWechatWorkBotRepository        = notificationRepo.NewWechatWorkBotRepository
	NewWechatWorkMessageLogRepository = notificationRepo.NewWechatWorkMessageLogRepository
)

// Monitoring模块类型别名
type (
	AlertConfigRepository        = monitoringRepo.AlertConfigRepository
	AlertHistoryRepository       = monitoringRepo.AlertHistoryRepository
	AlertSilenceRepository       = monitoringRepo.AlertSilenceRepository
	AlertEscalationRepository    = monitoringRepo.AlertEscalationRepository
	AlertEscalationLogRepository = monitoringRepo.AlertEscalationLogRepository
	HealthCheckConfigRepository  = monitoringRepo.HealthCheckConfigRepository
	HealthCheckHistoryRepository = monitoringRepo.HealthCheckHistoryRepository
	CertInfo                     = monitoringRepo.CertInfo
	ListFilters                  = monitoringRepo.ListFilters
)

// Monitoring模块函数别名
var (
	NewAlertConfigRepository        = monitoringRepo.NewAlertConfigRepository
	NewAlertHistoryRepository       = monitoringRepo.NewAlertHistoryRepository
	NewAlertSilenceRepository       = monitoringRepo.NewAlertSilenceRepository
	NewAlertEscalationRepository    = monitoringRepo.NewAlertEscalationRepository
	NewAlertEscalationLogRepository = monitoringRepo.NewAlertEscalationLogRepository
	NewHealthCheckConfigRepository  = monitoringRepo.NewHealthCheckConfigRepository
	NewHealthCheckHistoryRepository = monitoringRepo.NewHealthCheckHistoryRepository
)

// System模块类型别名
type (
	AuditLogRepository       = systemRepo.AuditLogRepository
	AuditLogFilter           = systemRepo.AuditLogFilter
	OADataRepository         = systemRepo.OADataRepository
	OAAddressRepository      = systemRepo.OAAddressRepository
	OANotifyConfigRepository = systemRepo.OANotifyConfigRepository
	MessageTemplateRepository = systemRepo.MessageTemplateRepository
)

// System模块函数别名
var (
	NewAuditLogRepository       = systemRepo.NewAuditLogRepository
	NewOADataRepository         = systemRepo.NewOADataRepository
	NewOAAddressRepository      = systemRepo.NewOAAddressRepository
	NewOANotifyConfigRepository = systemRepo.NewOANotifyConfigRepository
	NewMessageTemplateRepository = systemRepo.NewMessageTemplateRepository
)
