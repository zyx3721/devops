// Package system 定义系统管理相关的数据模型
// 本文件包含 OA 系统相关的模型定义
package system

import (
	"time"

	"gorm.io/gorm"
)

// ==================== OA 模型 ====================

// OAData OA数据模型
// 存储从 OA 系统同步的数据
type OAData struct {
	ID           uint      `gorm:"primarykey" json:"id"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	UniqueID     string    `gorm:"size:100;not null;uniqueIndex" json:"unique_id"` // 唯一标识
	Source       string    `gorm:"size:100;index" json:"source"`                   // 来源 OA 地址名称
	IPAddress    string    `gorm:"size:50" json:"ip_address"`                      // IP 地址
	UserAgent    string    `gorm:"size:500" json:"user_agent"`                     // User Agent
	OriginalData string    `gorm:"type:text" json:"original_data"`                 // 原始数据 JSON
}

// TableName 指定表名
func (OAData) TableName() string {
	return "oa_data"
}

// OAAddress OA地址配置模型
// 存储 OA 系统的地址配置
type OAAddress struct {
	ID          uint      `gorm:"primarykey" json:"id"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Name        string    `gorm:"size:100;not null" json:"name"`                   // 名称
	URL         string    `gorm:"size:500;not null" json:"url"`                    // URL
	Type        string    `gorm:"size:50;default:'webhook'" json:"type"`           // 类型: webhook/callback/api
	Description string    `gorm:"type:text" json:"description"`                    // 描述
	Status      string    `gorm:"size:20;default:'active';not null" json:"status"` // 状态
	IsDefault   bool      `gorm:"default:false" json:"is_default"`                 // 是否默认
	CreatedBy   *uint     `gorm:"index" json:"created_by"`
}

// TableName 指定表名
func (OAAddress) TableName() string {
	return "oa_addresses"
}

// OANotifyConfig OA通知配置模型
// 配置 OA 数据同步后的通知
type OANotifyConfig struct {
	ID            uint      `gorm:"primarykey" json:"id"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
	Name          string    `gorm:"size:100;not null" json:"name"`                   // 配置名称
	AppID         uint      `gorm:"column:app_id" json:"app_id"`                     // 飞书应用ID
	ReceiveID     string    `gorm:"size:100;not null" json:"receive_id"`             // 接收者ID
	ReceiveIDType string    `gorm:"size:50;not null" json:"receive_id_type"`         // ID类型: chat_id/open_id/user_id
	Description   string    `gorm:"type:text" json:"description"`                    // 描述
	Status        string    `gorm:"size:20;default:'active';not null" json:"status"` // 状态
	IsDefault     bool      `gorm:"default:false" json:"is_default"`                 // 是否默认
}

// TableName 指定表名
func (OANotifyConfig) TableName() string {
	return "oa_notify_configs"
}

// SystemConfig 系统配置模型
// 存储系统级别的配置
type SystemConfig struct {
	gorm.Model
	Key         string `gorm:"size:100;not null;uniqueIndex" json:"key"` // 配置键
	Value       string `gorm:"type:text" json:"value"`                   // 配置值
	Description string `gorm:"type:text" json:"description"`             // 描述
}

// TableName 指定表名
func (SystemConfig) TableName() string {
	return "system_configs"
}

// MessageTemplate 消息模板模型
// 存储消息模板
type MessageTemplate struct {
	gorm.Model
	Name        string `gorm:"size:100;not null" json:"name"`     // 模板名称
	Type        string `gorm:"size:50;not null" json:"type"`      // 模板类型
	Content     string `gorm:"type:text;not null" json:"content"` // 模板内容
	Description string `gorm:"type:text" json:"description"`      // 描述
	IsActive    bool   `gorm:"default:true" json:"is_active"`     // 是否激活
	CreatedBy   *uint  `gorm:"index" json:"created_by"`
}

// TableName 指定表名
func (MessageTemplate) TableName() string {
	return "message_templates"
}
