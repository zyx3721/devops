// Package infrastructure 定义基础设施相关的数据模型
// 本文件包含定时 HPA 相关的模型定义
package infrastructure

import (
	"time"

	"gorm.io/datatypes"
)

// CronHPA 定时水平自动伸缩配置
// 用于配置基于时间的 Pod 自动伸缩规则
type CronHPA struct {
	ID         uint           `gorm:"primarykey" json:"id"`
	ClusterID  uint           `gorm:"not null;index" json:"cluster_id"`            // 关联的 K8s 集群ID
	Name       string         `gorm:"size:100;not null" json:"name"`               // 配置名称
	Namespace  string         `gorm:"size:100;not null;index" json:"namespace"`    // 命名空间
	TargetKind string         `gorm:"size:50;not null" json:"target_kind"`         // 目标类型: Deployment/StatefulSet
	TargetName string         `gorm:"size:100;not null" json:"target_name"`        // 目标名称
	Enabled    bool           `gorm:"default:true" json:"enabled"`                 // 是否启用
	Schedules  datatypes.JSON `gorm:"type:json" json:"schedules"`                  // JSON 存储调度规则
	CreatedBy  string         `gorm:"size:100" json:"created_by"`                  // 创建者
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
}

// TableName 指定表名
func (CronHPA) TableName() string {
	return "cron_hpa"
}
