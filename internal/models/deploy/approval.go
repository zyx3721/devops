// Package deploy 定义部署流程相关的数据模型
// 本文件包含审批流程相关的模型定义
package deploy

import (
	"time"

	"gorm.io/gorm"
)

// ==================== 审批规则模型 ====================

// ApprovalRule 审批规则
// 定义哪些环境需要审批
type ApprovalRule struct {
	ID             uint      `gorm:"primarykey" json:"id"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
	AppID          uint      `gorm:"index;default:0" json:"app_id"`     // 应用ID，0表示全局规则
	Env            string    `gorm:"size:50;not null" json:"env"`       // 环境: dev/test/staging/prod/*
	NeedApproval   bool      `gorm:"default:true" json:"need_approval"` // 是否需要审批
	Approvers      string    `gorm:"size:500" json:"approvers"`         // 审批人ID列表，逗号分隔
	TimeoutMinutes int       `gorm:"default:30" json:"timeout_minutes"` // 超时时间(分钟)
	Enabled        bool      `gorm:"default:true" json:"enabled"`       // 是否启用
	CreatedBy      uint      `gorm:"default:0" json:"created_by"`
}

// TableName 指定表名
func (ApprovalRule) TableName() string {
	return "approval_rules"
}

// ApprovalRecord 审批记录
// 记录每次审批操作
type ApprovalRecord struct {
	ID           uint      `gorm:"primarykey" json:"id"`
	CreatedAt    time.Time `json:"created_at"`
	RecordID     uint      `gorm:"not null;index" json:"record_id"` // 关联的部署记录ID
	ApproverID   uint      `gorm:"not null" json:"approver_id"`     // 审批人ID
	ApproverName string    `gorm:"size:100" json:"approver_name"`   // 审批人名称
	Action       string    `gorm:"size:20;not null" json:"action"`  // 操作: approve/reject
	Comment      string    `gorm:"type:text" json:"comment"`        // 审批意见
}

// TableName 指定表名
func (ApprovalRecord) TableName() string {
	return "approval_records"
}

// ==================== 多级审批链模型 ====================

// ApprovalChain 审批链
// 定义多级审批流程
type ApprovalChain struct {
	ID             uint           `gorm:"primarykey" json:"id"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"-"`
	Name           string         `gorm:"size:100;not null" json:"name"`                       // 审批链名称
	Description    string         `gorm:"size:500" json:"description"`                         // 描述
	AppID          uint           `gorm:"index;default:0" json:"app_id"`                       // 应用ID，0表示全局
	Env            string         `gorm:"size:50;default:'*'" json:"env"`                      // 环境，*表示所有
	Priority       int            `gorm:"default:0" json:"priority"`                           // 优先级，数值越大优先级越高
	TimeoutMinutes int            `gorm:"default:60" json:"timeout_minutes"`                   // 超时时间(分钟)
	TimeoutAction  string         `gorm:"size:20;default:'auto_cancel'" json:"timeout_action"` // 超时动作: auto_approve/auto_reject/auto_cancel
	AllowEmergency bool           `gorm:"default:true" json:"allow_emergency"`                 // 是否允许紧急跳过
	Enabled        bool           `gorm:"default:true" json:"enabled"`                         // 是否启用
	CreatedBy      uint           `json:"created_by"`
	Nodes          []ApprovalNode `gorm:"foreignKey:ChainID" json:"nodes,omitempty"` // 审批节点列表
}

// TableName 指定表名
func (ApprovalChain) TableName() string {
	return "approval_chains"
}

// ApprovalNode 审批节点
// 定义审批链中的每个节点
type ApprovalNode struct {
	ID             uint      `gorm:"primarykey" json:"id"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
	ChainID        uint      `gorm:"not null;index" json:"chain_id"`                      // 审批链ID
	Name           string    `gorm:"size:100;not null" json:"name"`                       // 节点名称
	NodeOrder      int       `gorm:"not null" json:"node_order"`                          // 节点顺序，从1开始
	ApproveMode    string    `gorm:"size:20;default:'any'" json:"approve_mode"`           // 审批模式: any/all/count
	ApproveCount   int       `gorm:"default:1" json:"approve_count"`                      // 需要的审批人数(mode=count时)
	ApproverType   string    `gorm:"size:20;default:'user'" json:"approver_type"`         // 审批人类型: user/role/app_owner/team_leader
	Approvers      string    `gorm:"size:500" json:"approvers"`                           // 审批人ID或角色名，逗号分隔
	TimeoutMinutes int       `gorm:"default:0" json:"timeout_minutes"`                    // 超时时间，0表示继承链配置
	TimeoutAction  string    `gorm:"size:20;default:'auto_reject'" json:"timeout_action"` // 超时动作
	RejectOnAny    bool      `gorm:"default:true" json:"reject_on_any"`                   // 任一人拒绝是否立即拒绝
}

// TableName 指定表名
func (ApprovalNode) TableName() string {
	return "approval_nodes"
}

// ApprovalInstance 审批实例
// 记录每次审批流程的执行状态
type ApprovalInstance struct {
	ID               uint                   `gorm:"primarykey" json:"id"`
	CreatedAt        time.Time              `json:"created_at"`
	UpdatedAt        time.Time              `json:"updated_at"`
	RecordID         uint                   `gorm:"not null;uniqueIndex" json:"record_id"`         // 部署记录ID
	ChainID          uint                   `gorm:"not null;index" json:"chain_id"`                // 审批链ID
	ChainName        string                 `gorm:"size:100" json:"chain_name"`                    // 审批链名称
	Status           string                 `gorm:"size:20;default:'pending';index" json:"status"` // 状态: pending/approved/rejected/cancelled
	CurrentNodeOrder int                    `gorm:"default:1" json:"current_node_order"`           // 当前节点顺序
	StartedAt        *time.Time             `json:"started_at"`                                    // 开始时间
	FinishedAt       *time.Time             `json:"finished_at"`                                   // 完成时间
	CancelReason     string                 `gorm:"size:500" json:"cancel_reason"`                 // 取消原因
	NodeInstances    []ApprovalNodeInstance `gorm:"foreignKey:InstanceID" json:"node_instances,omitempty"`
}

// TableName 指定表名
func (ApprovalInstance) TableName() string {
	return "approval_instances"
}

// ApprovalNodeInstance 节点实例
// 记录每个审批节点的执行状态
type ApprovalNodeInstance struct {
	ID            uint             `gorm:"primarykey" json:"id"`
	CreatedAt     time.Time        `json:"created_at"`
	UpdatedAt     time.Time        `json:"updated_at"`
	InstanceID    uint             `gorm:"not null;index" json:"instance_id"`                 // 审批实例ID
	NodeID        uint             `gorm:"not null" json:"node_id"`                           // 审批节点ID
	NodeName      string           `gorm:"size:100" json:"node_name"`                         // 节点名称
	NodeOrder     int              `gorm:"not null" json:"node_order"`                        // 节点顺序
	ApproveMode   string           `gorm:"size:20;default:'any'" json:"approve_mode"`         // 审批模式
	ApproveCount  int              `gorm:"default:1" json:"approve_count"`                    // 需要的审批人数
	ApproverType  string           `gorm:"size:20;default:'user'" json:"approver_type"`       // 审批人类型
	Approvers     string           `gorm:"size:500" json:"approvers"`                         // 实际审批人ID列表
	Status        string           `gorm:"size:20;default:'pending';index" json:"status"`     // 状态: pending/active/approved/rejected/timeout
	ApprovedCount int              `gorm:"default:0" json:"approved_count"`                   // 已通过人数
	RejectedCount int              `gorm:"default:0" json:"rejected_count"`                   // 已拒绝人数
	RejectOnAny   bool             `gorm:"default:true" json:"reject_on_any"`                 // 任一人拒绝是否立即拒绝
	TimeoutAction string           `gorm:"size:20;default:'auto_reject'" json:"timeout_action"` // 超时动作
	ActivatedAt   *time.Time       `json:"activated_at"`                                      // 激活时间
	FinishedAt    *time.Time       `json:"finished_at"`                                       // 完成时间
	TimeoutAt     *time.Time       `gorm:"index" json:"timeout_at"`                           // 超时时间
	Actions       []ApprovalAction `gorm:"foreignKey:NodeInstanceID" json:"actions,omitempty"`
}

// TableName 指定表名
func (ApprovalNodeInstance) TableName() string {
	return "approval_node_instances"
}

// ApprovalAction 审批动作
// 记录每个审批人的操作
type ApprovalAction struct {
	ID             uint      `gorm:"primarykey" json:"id"`
	CreatedAt      time.Time `json:"created_at"`
	NodeInstanceID uint      `gorm:"not null;index" json:"node_instance_id"` // 节点实例ID
	UserID         uint      `gorm:"not null;index" json:"user_id"`          // 用户ID
	UserName       string    `gorm:"size:100" json:"user_name"`              // 用户名
	Action         string    `gorm:"size:20;not null" json:"action"`         // 操作: approve/reject/transfer
	Comment        string    `gorm:"type:text" json:"comment"`               // 审批意见
	TransferTo     *uint     `json:"transfer_to"`                            // 转交目标用户ID
	TransferToName string    `gorm:"size:100" json:"transfer_to_name"`       // 转交目标用户名
}

// TableName 指定表名
func (ApprovalAction) TableName() string {
	return "approval_actions"
}
