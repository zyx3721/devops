// Package notification 定义消息通知相关的数据模型
// 本文件包含企业微信相关的模型定义
package notification

import (
	"time"
)

// ==================== 企业微信模型 ====================

// WechatWorkApp 企业微信应用模型
// 存储企业微信应用配置
type WechatWorkApp struct {
	ID          uint      `gorm:"primarykey" json:"id"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Name        string    `gorm:"size:100;not null" json:"name"`                     // 应用名称
	CorpID      string    `gorm:"column:corp_id;size:100;not null" json:"corp_id"`   // 企业ID
	AgentID     int64     `gorm:"column:agent_id;not null" json:"agent_id"`          // 应用 Agent ID
	Secret      string    `gorm:"column:secret;size:200;not null" json:"secret"`     // 应用密钥
	Project     string    `gorm:"size:100" json:"project"`                           // 关联项目
	Status      string    `gorm:"size:20;default:'active';not null" json:"status"`   // 状态
	Description string    `gorm:"type:text" json:"description"`                      // 描述
	IsDefault   bool      `gorm:"column:is_default;default:false" json:"is_default"` // 是否默认应用
	CreatedBy   *uint     `gorm:"column:created_by" json:"created_by"`
}

// TableName 指定表名
func (WechatWorkApp) TableName() string {
	return "wechat_work_apps"
}

// WechatWorkBot 企业微信机器人模型
// 存储企业微信 Webhook 机器人配置
type WechatWorkBot struct {
	ID          uint      `gorm:"primarykey" json:"id"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Name        string    `gorm:"size:100;not null" json:"name"`                   // 机器人名称
	WebhookURL  string    `gorm:"size:500;not null" json:"webhook_url"`            // Webhook URL
	Description string    `gorm:"type:text" json:"description"`                    // 描述
	Status      string    `gorm:"size:20;default:'active';not null" json:"status"` // 状态
	CreatedBy   *uint     `gorm:"index" json:"created_by"`
}

// TableName 指定表名
func (WechatWorkBot) TableName() string {
	return "wechat_work_bots"
}

// WechatWorkMessageLog 企业微信消息发送记录
// 记录所有通过企业微信发送的消息
type WechatWorkMessageLog struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	CreatedAt time.Time `json:"created_at"`
	MsgType   string    `gorm:"size:50;not null" json:"msg_type"`        // 消息类型
	ToUser    string    `gorm:"size:500" json:"to_user"`                 // 接收用户
	ToParty   string    `gorm:"size:500" json:"to_party"`                // 接收部门
	ToTag     string    `gorm:"size:500" json:"to_tag"`                  // 接收标签
	Content   string    `gorm:"type:text" json:"content"`                // 消息内容
	Title     string    `gorm:"size:200" json:"title"`                   // 标题
	Source    string    `gorm:"size:50" json:"source"`                   // 来源
	Status    string    `gorm:"size:20;default:'success'" json:"status"` // 状态
	ErrorMsg  string    `gorm:"type:text" json:"error_msg"`              // 错误信息
	AppID     uint      `gorm:"column:app_id" json:"app_id"`             // 使用的企业微信应用ID
}

// TableName 指定表名
func (WechatWorkMessageLog) TableName() string {
	return "wechat_work_message_logs"
}
