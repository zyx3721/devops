// Package model 定义消息通知领域的数据模型
package model

import (
	"time"

	"gorm.io/gorm"
)

// ==================== 飞书模型 ====================

// FeishuBot 飞书机器人模型
type FeishuBot struct {
	ID                uint      `gorm:"primarykey" json:"id"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
	Name              string    `gorm:"size:100;not null" json:"name"`
	WebhookURL        string    `gorm:"size:500;not null" json:"webhook_url"`
	Project           string    `gorm:"size:100;default:''" json:"project"`
	Secret            string    `gorm:"size:100" json:"secret"`
	Description       string    `gorm:"type:text" json:"description"`
	Status            string    `gorm:"size:20;default:'active';not null" json:"status"`
	MessageTemplateID *uint     `gorm:"index" json:"message_template_id"`
	CreatedBy         *uint     `gorm:"index" json:"created_by"`
}

func (FeishuBot) TableName() string { return "feishu_bots" }

// FeishuApp 飞书应用模型
type FeishuApp struct {
	ID          uint      `gorm:"primarykey" json:"id"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Name        string    `gorm:"size:100;not null" json:"name"`
	AppID       string    `gorm:"column:app_id;size:100;not null" json:"app_id"`
	AppSecret   string    `gorm:"column:app_secret;size:200;not null" json:"app_secret"`
	Webhook     string    `gorm:"size:500" json:"webhook"`
	Project     string    `gorm:"size:100;not null" json:"project"`
	Status      string    `gorm:"size:20;not null" json:"status"`
	Description string    `gorm:"type:text" json:"description"`
	IsDefault   bool      `gorm:"column:is_default;default:false" json:"is_default"`
	CreatedBy   *uint     `gorm:"column:created_by" json:"created_by"`
}

func (FeishuApp) TableName() string { return "feishu_apps" }

// FeishuRequest 飞书请求存储模型
type FeishuRequest struct {
	gorm.Model
	RequestID       string `gorm:"size:100;not null;uniqueIndex" json:"request_id"`
	OriginalRequest string `gorm:"type:text" json:"original_request"`
	DisabledActions string `gorm:"type:text" json:"disabled_actions"`
	ActionCounts    string `gorm:"type:text" json:"action_counts"`
}

func (FeishuRequest) TableName() string { return "feishu_requests" }

// FeishuUserToken 飞书用户令牌模型
type FeishuUserToken struct {
	ID           uint      `gorm:"primarykey" json:"id"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	AppID        string    `gorm:"size:100;not null;uniqueIndex" json:"app_id"`
	AccessToken  string    `gorm:"type:text" json:"access_token"`
	RefreshToken string    `gorm:"type:text" json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
}

func (FeishuUserToken) TableName() string { return "feishu_user_tokens" }

// FeishuMessageLog 飞书消息发送记录
type FeishuMessageLog struct {
	ID            uint      `gorm:"primarykey" json:"id"`
	CreatedAt     time.Time `json:"created_at"`
	MsgType       string    `gorm:"size:50;not null" json:"msg_type"`
	ReceiveID     string    `gorm:"size:100;not null" json:"receive_id"`
	ReceiveIDType string    `gorm:"size:50;not null" json:"receive_id_type"`
	Content       string    `gorm:"type:text" json:"content"`
	Title         string    `gorm:"size:200" json:"title"`
	Source        string    `gorm:"size:50" json:"source"`
	Status        string    `gorm:"size:20;default:'success'" json:"status"`
	ErrorMsg      string    `gorm:"type:text" json:"error_msg"`
	AppID         uint      `gorm:"column:app_id" json:"app_id"`
}

func (FeishuMessageLog) TableName() string { return "feishu_message_logs" }
