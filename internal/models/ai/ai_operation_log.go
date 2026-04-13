package ai

import (
	"database/sql/driver"
	"encoding/json"
	"time"
)

// OperationAction 操作类型
type OperationAction string

const (
	ActionQueryLogs      OperationAction = "query_logs"      // 查询日志
	ActionQueryAlerts    OperationAction = "query_alerts"    // 查询告警
	ActionQueryMetrics   OperationAction = "query_metrics"   // 查询指标
	ActionRestartApp     OperationAction = "restart_app"     // 重启应用
	ActionScalePod       OperationAction = "scale_pod"       // 扩缩容
	ActionRollback       OperationAction = "rollback"        // 回滚部署
	ActionSilenceAlert   OperationAction = "silence_alert"   // 静默告警
	ActionQueryKnowledge OperationAction = "query_knowledge" // 查询知识库
)

// TargetType 目标类型
type TargetType string

const (
	TargetApplication TargetType = "application"
	TargetPod         TargetType = "pod"
	TargetDeployment  TargetType = "deployment"
	TargetAlert       TargetType = "alert"
	TargetCluster     TargetType = "cluster"
)

// JSONData 通用JSON数据类型
type JSONData map[string]any

// Scan 实现 sql.Scanner 接口
func (j *JSONData) Scan(value any) error {
	if value == nil {
		*j = nil
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(bytes, j)
}

// Value 实现 driver.Valuer 接口
func (j JSONData) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	return json.Marshal(j)
}

// AIOperationLog AI操作审计日志模型
type AIOperationLog struct {
	ID             uint            `gorm:"primaryKey" json:"id"`
	CreatedAt      time.Time       `gorm:"index" json:"created_at"`
	UserID         uint            `gorm:"not null;index" json:"user_id"`                     // 用户ID
	Username       string          `gorm:"size:100" json:"username"`                          // 用户名
	ConversationID *string         `gorm:"size:36;index" json:"conversation_id,omitempty"`    // 会话ID
	MessageID      *string         `gorm:"size:36" json:"message_id,omitempty"`               // 消息ID
	Action         OperationAction `gorm:"size:50;not null;index" json:"action"`              // 操作类型
	ActionName     string          `gorm:"size:100" json:"action_name"`                       // 操作名称
	TargetType     TargetType      `gorm:"size:50;index:idx_ai_op_target" json:"target_type"` // 目标类型
	TargetID       string          `gorm:"size:100;index:idx_ai_op_target" json:"target_id"`  // 目标ID
	TargetName     string          `gorm:"size:200" json:"target_name"`                       // 目标名称
	Params         JSONData        `gorm:"type:json" json:"params"`                           // 操作参数
	Result         JSONData        `gorm:"type:json" json:"result"`                           // 操作结果
	Success        bool            `gorm:"not null;default:false;index" json:"success"`       // 是否成功
	ErrorMsg       string          `gorm:"type:text" json:"error_msg,omitempty"`              // 错误信息
	DurationMs     int             `gorm:"default:0" json:"duration_ms"`                      // 执行耗时(毫秒)
	IPAddress      string          `gorm:"size:50" json:"ip_address"`                         // 客户端IP
}

// TableName 指定表名
func (AIOperationLog) TableName() string {
	return "ai_operation_logs"
}

// GetActionName 获取操作名称
func GetActionName(action OperationAction) string {
	names := map[OperationAction]string{
		ActionQueryLogs:      "查询日志",
		ActionQueryAlerts:    "查询告警",
		ActionQueryMetrics:   "查询指标",
		ActionRestartApp:     "重启应用",
		ActionScalePod:       "扩缩容",
		ActionRollback:       "回滚部署",
		ActionSilenceAlert:   "静默告警",
		ActionQueryKnowledge: "查询知识库",
	}
	if name, ok := names[action]; ok {
		return name
	}
	return string(action)
}

// IsDangerousAction 判断是否为危险操作
func IsDangerousAction(action OperationAction) bool {
	dangerous := map[OperationAction]bool{
		ActionRestartApp:   true,
		ActionScalePod:     true,
		ActionRollback:     true,
		ActionSilenceAlert: true,
	}
	return dangerous[action]
}
