// Package system 定义系统管理相关的数据模型
// 本文件包含用户相关的模型定义
package system

import (
	"database/sql"

	"gorm.io/gorm"
)

// ==================== 用户模型 ====================

// User 用户模型
// 存储系统用户的基本信息，包括认证和状态
type User struct {
	gorm.Model
	Username    string       `gorm:"uniqueIndex;size:50;not null" json:"username"`    // 用户名，唯一
	Password    string       `gorm:"size:100;not null" json:"-"`                      // 密码，JSON序列化时隐藏
	Email       string       `gorm:"size:100;not null" json:"email"`                  // 邮箱
	Phone       string       `gorm:"size:20" json:"phone"`                            // 手机号
	Role        string       `gorm:"size:20;default:'user';not null" json:"role"`     // 角色: admin/user
	Status      string       `gorm:"size:20;default:'active';not null" json:"status"` // 状态: active/inactive
	LastLoginAt sql.NullTime `json:"last_login_at"`                                   // 最后登录时间
}

// TableName 指定表名
func (User) TableName() string {
	return "users"
}
