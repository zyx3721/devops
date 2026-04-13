// Package deploy 定义部署流程相关的数据模型
// 本文件包含部署记录相关的模型定义
package deploy

import (
	"time"

	"gorm.io/gorm"
)

// ==================== 部署记录模型 ====================

// DeployRecord 部署记录（含审批流程）
// 记录每次部署的详细信息
type DeployRecord struct {
	ID            uint      `gorm:"primarykey" json:"id"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
	ApplicationID uint      `gorm:"not null;index" json:"application_id"`          // 应用ID
	AppName       string    `gorm:"size:100;index" json:"app_name"`                 // 应用名称
	EnvName       string    `gorm:"size:50;index" json:"env_name"`                  // 环境名称
	Version       string    `gorm:"size:100" json:"version"`                        // 版本号
	Branch        string    `gorm:"size:100" json:"branch"`                         // Git 分支
	CommitID      string    `gorm:"size:100" json:"commit_id"`                      // Git Commit ID
	ImageTag      string    `gorm:"size:200" json:"image_tag"`                      // 镜像标签
	DeployType    string    `gorm:"size:50" json:"deploy_type"`                     // 部署类型: deploy/rollback/restart/scale
	DeployMethod  string    `gorm:"size:50;default:'jenkins'" json:"deploy_method"` // 部署方式: jenkins/k8s
	Status        string    `gorm:"size:20;index" json:"status"`                    // 状态: pending/approved/rejected/running/success/failed/cancelled
	Description   string    `gorm:"type:text" json:"description"`                   // 发布说明
	// Jenkins 相关
	JenkinsBuild int    `gorm:"default:0" json:"jenkins_build"` // Jenkins 构建号
	JenkinsURL   string `gorm:"size:500" json:"jenkins_url"`    // Jenkins 构建 URL
	// 审批相关
	NeedApproval    bool       `gorm:"default:false" json:"need_approval"` // 是否需要审批
	ApprovalChainID *uint      `gorm:"index" json:"approval_chain_id"`     // 审批链ID
	ApproverID      *uint      `json:"approver_id"`                        // 审批人ID
	ApproverName    string     `gorm:"size:100" json:"approver_name"`      // 审批人名称
	ApprovedAt      *time.Time `json:"approved_at"`                        // 审批时间
	RejectReason    string     `gorm:"type:text" json:"reject_reason"`     // 拒绝原因
	// 执行相关
	Duration     int        `gorm:"default:0" json:"duration"`  // 执行时长(秒)
	ErrorMsg     string     `gorm:"type:text" json:"error_msg"` // 错误信息
	Operator     string     `gorm:"size:100" json:"operator"`   // 操作人
	OperatorID   uint       `gorm:"index" json:"operator_id"`   // 操作人ID
	StartedAt    *time.Time `json:"started_at"`                 // 开始时间
	FinishedAt   *time.Time `json:"finished_at"`                // 完成时间
	RollbackFrom *uint      `json:"rollback_from"`              // 回滚来源记录ID
}

// TableName 指定表名
func (DeployRecord) TableName() string {
	return "deploy_records"
}

// DeployLock 发布锁
// 防止同一应用同一环境并发部署
type DeployLock struct {
	ID            uint       `gorm:"primarykey" json:"id"`
	CreatedAt     time.Time  `json:"created_at"`
	ApplicationID uint       `gorm:"not null;index" json:"application_id"`   // 应用ID
	EnvName       string     `gorm:"size:50;not null" json:"env_name"`       // 环境名称
	RecordID      uint       `gorm:"not null;index" json:"record_id"`        // 关联的部署记录ID
	LockedBy      uint       `gorm:"not null" json:"locked_by"`              // 锁定者ID
	LockedByName  string     `gorm:"size:100" json:"locked_by_name"`         // 锁定者名称
	ExpiresAt     time.Time  `gorm:"not null" json:"expires_at"`             // 过期时间
	Status        string     `gorm:"size:20;default:'active'" json:"status"` // 状态: active/released/expired
	ReleasedAt    *time.Time `json:"released_at"`                            // 释放时间
	ReleasedBy    *uint      `json:"released_by"`                            // 释放者ID
	ReleaseReason string     `gorm:"size:200" json:"release_reason"`         // 释放原因
}

// TableName 指定表名
func (DeployLock) TableName() string {
	return "deploy_locks"
}

// DeployWindow 发布窗口
// 定义允许发布的时间窗口
type DeployWindow struct {
	ID             uint      `gorm:"primarykey" json:"id"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
	AppID          uint      `gorm:"index;default:0" json:"app_id"`               // 应用ID，0表示全局
	Env            string    `gorm:"size:50;not null" json:"env"`                 // 环境
	Weekdays       string    `gorm:"size:50;default:'1,2,3,4,5'" json:"weekdays"` // 允许的星期: 1-7
	StartTime      string    `gorm:"size:10;default:'10:00'" json:"start_time"`   // 开始时间
	EndTime        string    `gorm:"size:10;default:'18:00'" json:"end_time"`     // 结束时间
	AllowEmergency bool      `gorm:"default:true" json:"allow_emergency"`         // 是否允许紧急发布
	Enabled        bool      `gorm:"default:true" json:"enabled"`                 // 是否启用
	CreatedBy      uint      `gorm:"default:0" json:"created_by"`
}

// TableName 指定表名
func (DeployWindow) TableName() string {
	return "deploy_windows"
}

// Task 任务模型
// 存储异步任务信息
type Task struct {
	gorm.Model
	Name        string    `gorm:"size:100;not null" json:"name"`                    // 任务名称
	Description string    `gorm:"type:text" json:"description"`                     // 描述
	Status      string    `gorm:"size:20;default:'pending';not null" json:"status"` // 状态
	CreatedBy   uint      `gorm:"not null" json:"created_by"`                       // 创建者ID
	StartTime   time.Time `json:"start_time"`                                       // 开始时间
	EndTime     time.Time `json:"end_time"`                                         // 结束时间
	JenkinsJob  string    `gorm:"size:100" json:"jenkins_job"`                      // Jenkins Job
	Parameters  string    `gorm:"type:text" json:"parameters"`                      // 参数 JSON
}

// TableName 指定表名
func (Task) TableName() string {
	return "tasks"
}
