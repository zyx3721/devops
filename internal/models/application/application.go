// Package application 定义应用管理相关的数据模型
// 本文件包含应用管理相关的模型定义
package application

import (
	"time"
)

// ==================== 应用管理模型 ====================

// Application 应用模型
// 存储应用的基本信息和关联配置
type Application struct {
	ID                uint      `gorm:"primarykey" json:"id"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
	Name              string    `gorm:"size:100;not null;uniqueIndex" json:"name"` // 应用名称，唯一
	DisplayName       string    `gorm:"size:200" json:"display_name"`              // 显示名称
	Description       string    `gorm:"type:text" json:"description"`              // 描述
	GitRepo           string    `gorm:"size:500" json:"git_repo"`                  // Git 仓库地址
	Language          string    `gorm:"size:50" json:"language"`                   // 开发语言
	Framework         string    `gorm:"size:50" json:"framework"`                  // 框架
	Team              string    `gorm:"size:100;index" json:"team"`                // 所属团队
	Owner             string    `gorm:"size:100" json:"owner"`                     // 负责人
	Status            string    `gorm:"size:20;default:'active'" json:"status"`    // 状态
	JenkinsInstanceID *uint     `gorm:"index" json:"jenkins_instance_id"`          // Jenkins 实例ID
	JenkinsJobName    string    `gorm:"size:200" json:"jenkins_job_name"`          // Jenkins Job 名称
	K8sClusterID      *uint     `gorm:"index" json:"k8s_cluster_id"`               // K8s 集群ID
	K8sNamespace      string    `gorm:"size:100" json:"k8s_namespace"`             // K8s 命名空间
	K8sDeployment     string    `gorm:"size:200" json:"k8s_deployment"`            // K8s Deployment 名称
	NotifyPlatform    string    `gorm:"size:50" json:"notify_platform"`            // 通知平台
	NotifyAppID       *uint     `gorm:"index" json:"notify_app_id"`                // 通知应用ID
	NotifyReceiveID   string    `gorm:"size:200" json:"notify_receive_id"`         // 通知接收者ID
	NotifyReceiveType string    `gorm:"size:50" json:"notify_receive_type"`        // 接收者类型
	CreatedBy         *uint     `gorm:"index" json:"created_by"`
}

// TableName 指定表名
func (Application) TableName() string {
	return "applications"
}

// ApplicationEnv 应用环境配置
// 存储应用在不同环境下的配置
type ApplicationEnv struct {
	ID            uint      `gorm:"primarykey" json:"id"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
	ApplicationID uint      `gorm:"column:app_id;not null;index" json:"application_id"` // 应用ID
	EnvName       string    `gorm:"size:50;not null" json:"env_name"`                   // 环境名称
	Branch        string    `gorm:"size:100" json:"branch"`                             // Git 分支
	JenkinsJob    string    `gorm:"size:200" json:"jenkins_job"`                        // Jenkins Job
	K8sNamespace  string    `gorm:"size:100" json:"k8s_namespace"`                      // K8s 命名空间
	K8sDeployment string    `gorm:"size:200" json:"k8s_deployment"`                     // K8s Deployment
	Replicas      int       `gorm:"default:1" json:"replicas"`                          // 副本数
	Config        string    `gorm:"type:text" json:"config"`                            // 环境特定配置
}

// TableName 指定表名
func (ApplicationEnv) TableName() string {
	return "application_envs"
}
