package ai

import (
	"database/sql/driver"
	"encoding/json"
	"time"

	"gorm.io/gorm"
)

// PageContext 页面上下文信息
type PageContext struct {
	Page        string          `json:"page"`                  // 当前页面标识
	Route       string          `json:"route"`                 // 当前路由
	Application *ApplicationCtx `json:"application,omitempty"` // 应用上下文
	Cluster     *ClusterCtx     `json:"cluster,omitempty"`     // 集群上下文
	Alert       *AlertCtx       `json:"alert,omitempty"`       // 告警上下文
	Deployment  *DeploymentCtx  `json:"deployment,omitempty"`  // 部署上下文
	Extra       map[string]any  `json:"extra,omitempty"`       // 额外上下文
}

// ApplicationCtx 应用上下文
type ApplicationCtx struct {
	ID          uint   `json:"id"`
	Name        string `json:"name"`
	Environment string `json:"environment"`
	Status      string `json:"status"`
}

// ClusterCtx 集群上下文
type ClusterCtx struct {
	ID   uint   `json:"id"`
	Name string `json:"name"`
}

// AlertCtx 告警上下文
type AlertCtx struct {
	ID      uint   `json:"id"`
	Level   string `json:"level"`
	Message string `json:"message"`
}

// DeploymentCtx 部署上下文
type DeploymentCtx struct {
	ID      uint   `json:"id"`
	Status  string `json:"status"`
	Version string `json:"version"`
}

// Scan 实现 sql.Scanner 接口
func (p *PageContext) Scan(value any) error {
	if value == nil {
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(bytes, p)
}

// Value 实现 driver.Valuer 接口
func (p PageContext) Value() (driver.Value, error) {
	return json.Marshal(p)
}

// AIConversation AI会话模型
type AIConversation struct {
	ID            string         `gorm:"primaryKey;size:36" json:"id"` // UUID
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
	UserID        uint           `gorm:"not null;index" json:"user_id"`  // 用户ID
	Title         string         `gorm:"size:255" json:"title"`          // 会话标题
	Context       *PageContext   `gorm:"type:json" json:"context"`       // 页面上下文
	MessageCount  int            `gorm:"default:0" json:"message_count"` // 消息数量
	LastMessageAt *time.Time     `gorm:"index" json:"last_message_at"`   // 最后消息时间

	// 关联
	Messages []AIMessage `gorm:"foreignKey:ConversationID;references:ID" json:"messages,omitempty"`
}

// TableName 指定表名
func (AIConversation) TableName() string {
	return "ai_conversations"
}

// BeforeCreate 创建前钩子
func (c *AIConversation) BeforeCreate(tx *gorm.DB) error {
	if c.ID == "" {
		c.ID = generateUUID()
	}
	return nil
}

// generateUUID 生成UUID (简单实现，实际使用时建议用 github.com/google/uuid)
func generateUUID() string {
	// 使用时间戳和随机数生成简单UUID
	return time.Now().Format("20060102150405") + randomString(22)
}

// randomString 生成随机字符串
func randomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[time.Now().UnixNano()%int64(len(letters))]
		time.Sleep(time.Nanosecond)
	}
	return string(b)
}
