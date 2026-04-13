// Package infrastructure 定义基础设施相关的数据模型
// 本文件包含 Jenkins 相关的模型定义
package infrastructure

import (
	"time"

	"gorm.io/gorm"
)

// ==================== Jenkins 模型 ====================

// JenkinsInstance Jenkins实例模型
// 存储 Jenkins 服务器的连接配置
type JenkinsInstance struct {
	gorm.Model
	Name        string `gorm:"size:100;not null" json:"name"`                   // 实例名称
	URL         string `gorm:"size:500;not null" json:"url"`                    // Jenkins URL
	Username    string `gorm:"size:100" json:"username"`                        // 用户名
	APIToken    string `gorm:"size:500" json:"api_token"`                       // API Token
	Description string `gorm:"type:text" json:"description"`                    // 描述
	Status      string `gorm:"size:20;default:'active';not null" json:"status"` // 状态
	IsDefault   bool   `gorm:"default:false" json:"is_default"`                 // 是否默认实例
	CreatedBy   *uint  `gorm:"index" json:"created_by"`                         // 创建者ID
	UpdatedBy   *uint  `gorm:"index" json:"updated_by"`                         // 更新者ID
}

// TableName 指定表名
func (JenkinsInstance) TableName() string {
	return "jenkins_instances"
}

// JenkinsFeishuApp Jenkins实例与飞书应用关联表
// 用于配置 Jenkins 构建通知发送到哪个飞书应用
type JenkinsFeishuApp struct {
	ID                uint      `gorm:"primarykey" json:"id"`
	CreatedAt         time.Time `json:"created_at"`
	JenkinsInstanceID uint      `gorm:"not null;index" json:"jenkins_instance_id"` // Jenkins实例ID
	FeishuAppID       uint      `gorm:"not null;index" json:"feishu_app_id"`       // 飞书应用ID
	FeishuApp         FeishuApp `gorm:"foreignKey:FeishuAppID" json:"feishu_app,omitempty"`
}

// TableName 指定表名
func (JenkinsFeishuApp) TableName() string {
	return "jenkins_feishu_apps"
}

// JenkinsDingtalkApp Jenkins实例与钉钉应用关联表
// 用于配置 Jenkins 构建通知发送到哪个钉钉应用
type JenkinsDingtalkApp struct {
	ID                uint      `gorm:"primarykey" json:"id"`
	CreatedAt         time.Time `json:"created_at"`
	JenkinsInstanceID uint      `gorm:"not null;index" json:"jenkins_instance_id"` // Jenkins实例ID
	DingtalkAppID     uint      `gorm:"not null;index" json:"dingtalk_app_id"`     // 钉钉应用ID
}

// TableName 指定表名
func (JenkinsDingtalkApp) TableName() string {
	return "jenkins_dingtalk_apps"
}

// JenkinsWechatWorkApp Jenkins实例与企业微信应用关联表
// 用于配置 Jenkins 构建通知发送到哪个企业微信应用
type JenkinsWechatWorkApp struct {
	ID                uint      `gorm:"primarykey" json:"id"`
	CreatedAt         time.Time `json:"created_at"`
	JenkinsInstanceID uint      `gorm:"not null;index" json:"jenkins_instance_id"` // Jenkins实例ID
	WechatWorkAppID   uint      `gorm:"not null;index" json:"wechat_work_app_id"`  // 企业微信应用ID
}

// TableName 指定表名
func (JenkinsWechatWorkApp) TableName() string {
	return "jenkins_wechat_work_apps"
}
