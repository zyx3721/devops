// Package notification 定义消息通知相关的数据模型
// 本文件包含飞书相关的模型定义
package notification

import (
	"time"

	"gorm.io/gorm"
)

// ==================== 飞书模型 ====================

// FeishuBot 飞书机器人模型
// 存储飞书 Webhook 机器人配置
type FeishuBot struct {
	ID                uint      `gorm:"primarykey" json:"id"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
	Name              string    `gorm:"size:100;not null" json:"name"`                   // 机器人名称
	WebhookURL        string    `gorm:"size:500;not null" json:"webhook_url"`            // Webhook URL
	Project           string    `gorm:"size:100;default:''" json:"project"`              // 关联项目
	Secret            string    `gorm:"size:100" json:"secret"`                          // 签名密钥
	Description       string    `gorm:"type:text" json:"description"`                    // 描述
	Status            string    `gorm:"size:20;default:'active';not null" json:"status"` // 状态
	MessageTemplateID *uint     `gorm:"index" json:"message_template_id"`                // 消息模板ID
	CreatedBy         *uint     `gorm:"index" json:"created_by"`
}

// TableName 指定表名
func (FeishuBot) TableName() string {
	return "feishu_bots"
}

// FeishuApp 飞书应用模型
// 存储飞书开放平台应用配置
type FeishuApp struct {
	ID          uint      `gorm:"primarykey" json:"id"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Name        string    `gorm:"size:100;not null" json:"name"`                         // 应用名称
	AppID       string    `gorm:"column:app_id;size:100;not null" json:"app_id"`         // 飞书 App ID
	AppSecret   string    `gorm:"column:app_secret;size:200;not null" json:"app_secret"` // 飞书 App Secret
	Webhook     string    `gorm:"size:500" json:"webhook"`                               // Webhook URL
	Project     string    `gorm:"size:100;not null" json:"project"`                      // 关联项目
	Status      string    `gorm:"size:20;not null" json:"status"`                        // 状态
	Description string    `gorm:"type:text" json:"description"`                          // 描述
	IsDefault   bool      `gorm:"column:is_default;default:false" json:"is_default"`     // 是否默认应用
	CreatedBy   *uint     `gorm:"column:created_by" json:"created_by"`
}

// TableName 指定表名
func (FeishuApp) TableName() string {
	return "feishu_apps"
}

// FeishuRequest 飞书请求存储模型
// 存储飞书交互请求的状态
type FeishuRequest struct {
	gorm.Model
	RequestID       string `gorm:"size:100;not null;uniqueIndex" json:"request_id"` // 请求ID
	OriginalRequest string `gorm:"type:text" json:"original_request"`               // 原始请求 JSON
	DisabledActions string `gorm:"type:text" json:"disabled_actions"`               // 已禁用的操作 JSON
	ActionCounts    string `gorm:"type:text" json:"action_counts"`                  // 操作计数 JSON
}

// TableName 指定表名
func (FeishuRequest) TableName() string {
	return "feishu_requests"
}

// FeishuUserToken 飞书用户令牌模型
// 存储飞书用户 OAuth 令牌
type FeishuUserToken struct {
	ID           uint      `gorm:"primarykey" json:"id"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	AppID        string    `gorm:"size:100;not null;uniqueIndex" json:"app_id"` // 飞书 App ID
	AccessToken  string    `gorm:"type:text" json:"access_token"`               // 访问令牌
	RefreshToken string    `gorm:"type:text" json:"refresh_token"`              // 刷新令牌
	ExpiresAt    time.Time `json:"expires_at"`                                  // 过期时间
}

// TableName 指定表名
func (FeishuUserToken) TableName() string {
	return "feishu_user_tokens"
}

// FeishuMessageLog 飞书消息发送记录
// 记录所有通过飞书发送的消息
type FeishuMessageLog struct {
	ID            uint      `gorm:"primarykey" json:"id"`
	CreatedAt     time.Time `json:"created_at"`
	MsgType       string    `gorm:"size:50;not null" json:"msg_type"`        // 消息类型: text/post/interactive
	ReceiveID     string    `gorm:"size:100;not null" json:"receive_id"`     // 接收者ID
	ReceiveIDType string    `gorm:"size:50;not null" json:"receive_id_type"` // ID类型: chat_id/open_id/user_id
	Content       string    `gorm:"type:text" json:"content"`                // 消息内容
	Title         string    `gorm:"size:200" json:"title"`                   // 卡片标题
	Source        string    `gorm:"size:50" json:"source"`                   // 来源: manual/oa_sync
	Status        string    `gorm:"size:20;default:'success'" json:"status"` // 状态: success/failed
	ErrorMsg      string    `gorm:"type:text" json:"error_msg"`              // 错误信息
	AppID         uint      `gorm:"column:app_id" json:"app_id"`             // 使用的飞书应用ID
}

// TableName 指定表名
func (FeishuMessageLog) TableName() string {
	return "feishu_message_logs"
}
