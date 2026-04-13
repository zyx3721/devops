// Package model 定义消息通知领域的数据模型
package model

import "time"

// ==================== 企业微信模型 ====================

// WechatWorkApp 企业微信应用模型
type WechatWorkApp struct {
	ID          uint      `gorm:"primarykey" json:"id"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Name        string    `gorm:"size:100;not null" json:"name"`
	CorpID      string    `gorm:"column:corp_id;size:100;not null" json:"corp_id"`
	AgentID     int64     `gorm:"column:agent_id;not null" json:"agent_id"`
	Secret      string    `gorm:"column:secret;size:200;not null" json:"secret"`
	Project     string    `gorm:"size:100" json:"project"`
	Status      string    `gorm:"size:20;default:'active';not null" json:"status"`
	Description string    `gorm:"type:text" json:"description"`
	IsDefault   bool      `gorm:"column:is_default;default:false" json:"is_default"`
	CreatedBy   *uint     `gorm:"column:created_by" json:"created_by"`
}

func (WechatWorkApp) TableName() string { return "wechat_work_apps" }

// WechatWorkBot 企业微信机器人模型
type WechatWorkBot struct {
	ID          uint      `gorm:"primarykey" json:"id"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Name        string    `gorm:"size:100;not null" json:"name"`
	WebhookURL  string    `gorm:"size:500;not null" json:"webhook_url"`
	Description string    `gorm:"type:text" json:"description"`
	Status      string    `gorm:"size:20;default:'active';not null" json:"status"`
	CreatedBy   *uint     `gorm:"index" json:"created_by"`
}

func (WechatWorkBot) TableName() string { return "wechat_work_bots" }

// WechatWorkMessageLog 企业微信消息发送记录
type WechatWorkMessageLog struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	CreatedAt time.Time `json:"created_at"`
	MsgType   string    `gorm:"size:50;not null" json:"msg_type"`
	ToUser    string    `gorm:"size:500" json:"to_user"`
	ToParty   string    `gorm:"size:500" json:"to_party"`
	ToTag     string    `gorm:"size:500" json:"to_tag"`
	Content   string    `gorm:"type:text" json:"content"`
	Title     string    `gorm:"size:200" json:"title"`
	Source    string    `gorm:"size:50" json:"source"`
	Status    string    `gorm:"size:20;default:'success'" json:"status"`
	ErrorMsg  string    `gorm:"type:text" json:"error_msg"`
	AppID     uint      `gorm:"column:app_id" json:"app_id"`
}

func (WechatWorkMessageLog) TableName() string { return "wechat_work_message_logs" }
