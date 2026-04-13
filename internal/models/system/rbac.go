// Package system 定义系统管理相关的数据模型
// 本文件包含 RBAC 权限相关的模型定义
package system

import (
	"time"
)

// ==================== RBAC 权限模型 ====================

// Role 角色模型
// 定义系统角色
type Role struct {
	ID          uint      `gorm:"primarykey" json:"id"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Name        string    `gorm:"size:50;not null;uniqueIndex" json:"name"` // 角色名称，唯一
	DisplayName string    `gorm:"size:100" json:"display_name"`             // 显示名称
	Description string    `gorm:"type:text" json:"description"`             // 描述
	IsSystem    bool      `gorm:"default:false" json:"is_system"`           // 是否系统内置角色
	Status      string    `gorm:"size:20;default:'active'" json:"status"`   // 状态
}

// TableName 指定表名
func (Role) TableName() string {
	return "roles"
}

// Permission 权限模型
// 定义系统权限
type Permission struct {
	ID          uint      `gorm:"primarykey" json:"id"`
	CreatedAt   time.Time `json:"created_at"`
	Name        string    `gorm:"size:100;not null;uniqueIndex" json:"name"` // 权限名称，如 jenkins:read
	DisplayName string    `gorm:"size:100" json:"display_name"`              // 显示名称
	Resource    string    `gorm:"size:50;not null;index" json:"resource"`    // 资源: jenkins/k8s/feishu
	Action      string    `gorm:"size:50;not null" json:"action"`            // 操作: read/write/delete/admin
	Description string    `gorm:"type:text" json:"description"`              // 描述
}

// TableName 指定表名
func (Permission) TableName() string {
	return "permissions"
}

// RolePermission 角色权限关联
// 定义角色拥有的权限
type RolePermission struct {
	ID           uint `gorm:"primarykey" json:"id"`
	RoleID       uint `gorm:"not null;index" json:"role_id"`       // 角色ID
	PermissionID uint `gorm:"not null;index" json:"permission_id"` // 权限ID
}

// TableName 指定表名
func (RolePermission) TableName() string {
	return "role_permissions"
}

// UserRole 用户角色关联
// 定义用户拥有的角色（支持多角色）
type UserRole struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UserID    uint      `gorm:"not null;index" json:"user_id"` // 用户ID
	RoleID    uint      `gorm:"not null;index" json:"role_id"` // 角色ID
	Role      Role      `gorm:"foreignKey:RoleID" json:"role,omitempty"`
}

// TableName 指定表名
func (UserRole) TableName() string {
	return "user_roles"
}
