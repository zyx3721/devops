// Package notification 定义消息通知相关的数据模型
// 本文件包含钉钉相关的模型定义
package notification

import (
	"time"
)

// ==================== 钉钉模型 ====================

// DingtalkApp 钉钉应用模型
// 存储钉钉开放平台应用配置
type DingtalkApp struct {
	ID          uint      `gorm:"primarykey" json:"id"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Name        string    `gorm:"size:100;not null" json:"name"`                         // 应用名称
	AppKey      string    `gorm:"column:app_key;size:100;not null" json:"app_key"`       // 钉钉 App Key
	AppSecret   string    `gorm:"column:app_secret;size:200;not null" json:"app_secret"` // 钉钉 App Secret
	AgentID     int64     `gorm:"column:agent_id" json:"agent_id"`                       // 钉钉 Agent ID
	Project     string    `gorm:"size:100" json:"project"`                               // 关联项目
	Status      string    `gorm:"size:20;default:'active';not null" json:"status"`       // 状态
	Description string    `gorm:"type:text" json:"description"`                          // 描述
	IsDefault   bool      `gorm:"column:is_default;default:false" json:"is_default"`     // 是否默认应用
	CreatedBy   *uint     `gorm:"column:created_by" json:"created_by"`
}

// TableName 指定表名
func (DingtalkApp) TableName() string {
	return "dingtalk_apps"
}

// DingtalkBot 钉钉机器人模型
// 存储钉钉 Webhook 机器人配置
type DingtalkBot struct {
	ID          uint      `gorm:"primarykey" json:"id"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Name        string    `gorm:"size:100;not null" json:"name"`                   // 机器人名称
	WebhookURL  string    `gorm:"size:500;not null" json:"webhook_url"`            // Webhook URL
	Secret      string    `gorm:"size:100" json:"secret"`                          // 加签密钥
	Description string    `gorm:"type:text" json:"description"`                    // 描述
	Status      string    `gorm:"size:20;default:'active';not null" json:"status"` // 状态
	CreatedBy   *uint     `gorm:"index" json:"created_by"`
}

// TableName 指定表名
func (DingtalkBot) TableName() string {
	return "dingtalk_bots"
}

// DingtalkMessageLog 钉钉消息发送记录
// 记录所有通过钉钉发送的消息
type DingtalkMessageLog struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	CreatedAt time.Time `json:"created_at"`
	MsgType   string    `gorm:"size:50;not null" json:"msg_type"`        // 消息类型
	Target    string    `gorm:"size:200" json:"target"`                  // 发送目标
	Content   string    `gorm:"type:text" json:"content"`                // 消息内容
	Title     string    `gorm:"size:200" json:"title"`                   // 标题
	Source    string    `gorm:"size:50" json:"source"`                   // 来源
	Status    string    `gorm:"size:20;default:'success'" json:"status"` // 状态
	ErrorMsg  string    `gorm:"type:text" json:"error_msg"`              // 错误信息
	AppID     uint      `gorm:"column:app_id" json:"app_id"`             // 使用的钉钉应用ID
}

// TableName 指定表名
func (DingtalkMessageLog) TableName() string {
	return "dingtalk_message_logs"
}
