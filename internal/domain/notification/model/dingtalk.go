// Package model 定义消息通知领域的数据模型
package model

import "time"

// ==================== 钉钉模型 ====================

// DingtalkApp 钉钉应用模型
type DingtalkApp struct {
	ID          uint      `gorm:"primarykey" json:"id"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Name        string    `gorm:"size:100;not null" json:"name"`
	AppKey      string    `gorm:"column:app_key;size:100;not null" json:"app_key"`
	AppSecret   string    `gorm:"column:app_secret;size:200;not null" json:"app_secret"`
	AgentID     int64     `gorm:"column:agent_id" json:"agent_id"`
	Project     string    `gorm:"size:100" json:"project"`
	Status      string    `gorm:"size:20;default:'active';not null" json:"status"`
	Description string    `gorm:"type:text" json:"description"`
	IsDefault   bool      `gorm:"column:is_default;default:false" json:"is_default"`
	CreatedBy   *uint     `gorm:"column:created_by" json:"created_by"`
}

func (DingtalkApp) TableName() string { return "dingtalk_apps" }

// DingtalkBot 钉钉机器人模型
type DingtalkBot struct {
	ID          uint      `gorm:"primarykey" json:"id"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Name        string    `gorm:"size:100;not null" json:"name"`
	WebhookURL  string    `gorm:"size:500;not null" json:"webhook_url"`
	Secret      string    `gorm:"size:100" json:"secret"`
	Description string    `gorm:"type:text" json:"description"`
	Status      string    `gorm:"size:20;default:'active';not null" json:"status"`
	CreatedBy   *uint     `gorm:"index" json:"created_by"`
}

func (DingtalkBot) TableName() string { return "dingtalk_bots" }

// DingtalkMessageLog 钉钉消息发送记录
type DingtalkMessageLog struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	CreatedAt time.Time `json:"created_at"`
	MsgType   string    `gorm:"size:50;not null" json:"msg_type"`
	Target    string    `gorm:"size:200" json:"target"`
	Content   string    `gorm:"type:text" json:"content"`
	Title     string    `gorm:"size:200" json:"title"`
	Source    string    `gorm:"size:50" json:"source"`
	Status    string    `gorm:"size:20;default:'success'" json:"status"`
	ErrorMsg  string    `gorm:"type:text" json:"error_msg"`
	AppID     uint      `gorm:"column:app_id" json:"app_id"`
}

func (DingtalkMessageLog) TableName() string { return "dingtalk_message_logs" }
