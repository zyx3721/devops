// Package infrastructure 定义基础设施相关的数据模型
// 本文件包含 Kubernetes 集群相关的模型定义
package infrastructure

import (
	"time"

	"gorm.io/gorm"
)

// ==================== K8s 集群模型 ====================

// K8sCluster K8s集群模型
// 存储 Kubernetes 集群的连接配置
type K8sCluster struct {
	gorm.Model
	Name            string `gorm:"size:100;not null" json:"name"`                        // 集群名称
	Kubeconfig      string `gorm:"type:text;not null" json:"kubeconfig"`                 // kubeconfig 配置内容
	Namespace       string `gorm:"size:100;default:'default';not null" json:"namespace"` // 默认命名空间
	Registry        string `gorm:"size:500" json:"registry"`                             // 镜像仓库地址
	Repository      string `gorm:"size:200" json:"repository"`                           // 镜像仓库名称
	Description     string `gorm:"type:text" json:"description"`                         // 描述
	Status          string `gorm:"size:20;default:'active';not null" json:"status"`      // 状态
	IsDefault       bool   `gorm:"default:false" json:"is_default"`                      // 是否默认集群
	InsecureSkipTLS bool   `gorm:"default:false" json:"insecure_skip_tls"`               // 跳过 TLS 证书验证
	CheckTimeout    int    `gorm:"default:180;not null" json:"check_timeout"`            // 健康检查超时时间(秒)
	CreatedBy       *uint  `gorm:"index" json:"created_by"`
	UpdatedBy       *uint  `gorm:"index" json:"updated_by"`
}

// TableName 指定表名
func (K8sCluster) TableName() string {
	return "k8s_clusters"
}

// K8sClusterFeishuApp K8s集群与飞书应用关联表
// 用于配置 K8s 部署通知发送到哪个飞书应用
type K8sClusterFeishuApp struct {
	ID           uint      `gorm:"primarykey" json:"id"`
	CreatedAt    time.Time `json:"created_at"`
	K8sClusterID uint      `gorm:"not null;index" json:"k8s_cluster_id"` // K8s集群ID
	FeishuAppID  uint      `gorm:"not null;index" json:"feishu_app_id"`  // 飞书应用ID
	FeishuApp    FeishuApp `gorm:"foreignKey:FeishuAppID" json:"feishu_app,omitempty"`
}

// TableName 指定表名
func (K8sClusterFeishuApp) TableName() string {
	return "k8s_cluster_feishu_apps"
}

// K8sClusterDingtalkApp K8s集群与钉钉应用关联表
// 用于配置 K8s 部署通知发送到哪个钉钉应用
type K8sClusterDingtalkApp struct {
	ID            uint      `gorm:"primarykey" json:"id"`
	CreatedAt     time.Time `json:"created_at"`
	K8sClusterID  uint      `gorm:"not null;index" json:"k8s_cluster_id"`  // K8s集群ID
	DingtalkAppID uint      `gorm:"not null;index" json:"dingtalk_app_id"` // 钉钉应用ID
}

// TableName 指定表名
func (K8sClusterDingtalkApp) TableName() string {
	return "k8s_cluster_dingtalk_apps"
}

// K8sClusterWechatWorkApp K8s集群与企业微信应用关联表
// 用于配置 K8s 部署通知发送到哪个企业微信应用
type K8sClusterWechatWorkApp struct {
	ID              uint      `gorm:"primarykey" json:"id"`
	CreatedAt       time.Time `json:"created_at"`
	K8sClusterID    uint      `gorm:"not null;index" json:"k8s_cluster_id"`     // K8s集群ID
	WechatWorkAppID uint      `gorm:"not null;index" json:"wechat_work_app_id"` // 企业微信应用ID
}

// TableName 指定表名
func (K8sClusterWechatWorkApp) TableName() string {
	return "k8s_cluster_wechat_work_apps"
}
