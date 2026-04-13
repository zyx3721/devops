// Package system 定义系统管理相关的数据模型
// 本文件包含权限常量和权限检查函数
package system

// ==================== 角色常量 ====================

const (
	RoleSuperAdmin = "super_admin" // 超级管理员，不可被修改/删除
	RoleAdmin      = "admin"       // 管理员
	RoleUser       = "user"        // 普通用户
	RoleGuest      = "guest"       // 访客（只读）
)

// ==================== 权限常量 ====================

const (
	// 用户管理权限
	PermUserView   = "user:view"
	PermUserCreate = "user:create"
	PermUserUpdate = "user:update"
	PermUserDelete = "user:delete"
	PermUserRole   = "user:role"   // 修改用户角色
	PermUserStatus = "user:status" // 修改用户状态

	// 应用管理权限
	PermAppView   = "app:view"
	PermAppCreate = "app:create"
	PermAppUpdate = "app:update"
	PermAppDelete = "app:delete"
	PermAppDeploy = "app:deploy"

	// 审批管理权限
	PermApprovalView   = "approval:view"
	PermApprovalCreate = "approval:create"
	PermApprovalUpdate = "approval:update"
	PermApprovalDelete = "approval:delete"

	// K8s 管理权限
	PermK8sView   = "k8s:view"
	PermK8sCreate = "k8s:create"
	PermK8sUpdate = "k8s:update"
	PermK8sDelete = "k8s:delete"
	PermK8sExec   = "k8s:exec" // 执行操作（重启、扩缩容等）

	// Jenkins 管理权限
	PermJenkinsView    = "jenkins:view"
	PermJenkinsCreate  = "jenkins:create"
	PermJenkinsUpdate  = "jenkins:update"
	PermJenkinsDelete  = "jenkins:delete"
	PermJenkinsTrigger = "jenkins:trigger"

	// 系统配置权限
	PermSystemView   = "system:view"
	PermSystemUpdate = "system:update"
)

// RolePermissions 角色默认权限映射
var RolePermissions = map[string][]string{
	RoleSuperAdmin: {
		// 超级管理员拥有所有权限
		PermUserView, PermUserCreate, PermUserUpdate, PermUserDelete, PermUserRole, PermUserStatus,
		PermAppView, PermAppCreate, PermAppUpdate, PermAppDelete, PermAppDeploy,
		PermApprovalView, PermApprovalCreate, PermApprovalUpdate, PermApprovalDelete,
		PermK8sView, PermK8sCreate, PermK8sUpdate, PermK8sDelete, PermK8sExec,
		PermJenkinsView, PermJenkinsCreate, PermJenkinsUpdate, PermJenkinsDelete, PermJenkinsTrigger,
		PermSystemView, PermSystemUpdate,
	},
	RoleAdmin: {
		// 管理员拥有大部分权限，但不能管理超级管理员
		PermUserView, PermUserCreate, PermUserUpdate, PermUserDelete, PermUserRole, PermUserStatus,
		PermAppView, PermAppCreate, PermAppUpdate, PermAppDelete, PermAppDeploy,
		PermApprovalView, PermApprovalCreate, PermApprovalUpdate, PermApprovalDelete,
		PermK8sView, PermK8sCreate, PermK8sUpdate, PermK8sDelete, PermK8sExec,
		PermJenkinsView, PermJenkinsCreate, PermJenkinsUpdate, PermJenkinsDelete, PermJenkinsTrigger,
		PermSystemView,
	},
	RoleUser: {
		// 普通用户：只有查看权限和基本操作，无任何管理权限
		PermAppView, PermAppDeploy,
		PermApprovalView,
		PermK8sView,
		PermJenkinsView, PermJenkinsTrigger,
	},
	RoleGuest: {
		// 访客：只有查看权限
		PermAppView,
		PermApprovalView,
		PermK8sView,
		PermJenkinsView,
	},
}

// ==================== 权限检查函数 ====================

// GetRoleLevel 获取角色等级（数字越小权限越高）
func GetRoleLevel(role string) int {
	switch role {
	case RoleSuperAdmin:
		return 0
	case RoleAdmin:
		return 1
	case RoleUser:
		return 2
	case RoleGuest:
		return 3
	default:
		return 99
	}
}

// CanManageRole 检查是否可以管理目标角色
func CanManageRole(operatorRole, targetRole string) bool {
	operatorLevel := GetRoleLevel(operatorRole)
	targetLevel := GetRoleLevel(targetRole)
	// 只能管理比自己等级低的角色
	return operatorLevel < targetLevel
}

// HasPermission 检查角色是否有某个权限
func HasPermission(role, permission string) bool {
	perms, ok := RolePermissions[role]
	if !ok {
		return false
	}
	for _, p := range perms {
		if p == permission {
			return true
		}
	}
	return false
}

// IsSuperAdmin 检查是否是超级管理员
func IsSuperAdmin(role string) bool {
	return role == RoleSuperAdmin
}

// IsProtectedUser 检查用户是否受保护（超级管理员）
func IsProtectedUser(userID uint, role string) bool {
	// ID=1 的 admin 用户是系统初始超级管理员，受保护
	return userID == 1 || role == RoleSuperAdmin
}
