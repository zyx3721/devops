package ai

import (
	"database/sql/driver"
	"encoding/json"
	"time"

	"gorm.io/gorm"
)

// KnowledgeCategory 知识分类
type KnowledgeCategory string

const (
	CategoryApplication KnowledgeCategory = "application" // 应用管理
	CategoryTraffic     KnowledgeCategory = "traffic"     // 流量治理
	CategoryApproval    KnowledgeCategory = "approval"    // 审批流程
	CategoryK8s         KnowledgeCategory = "k8s"         // K8s管理
	CategoryMonitoring  KnowledgeCategory = "monitoring"  // 监控告警
	CategoryCICD        KnowledgeCategory = "cicd"        // CI/CD流水线
	CategoryGeneral     KnowledgeCategory = "general"     // 通用
)

// StringSlice 字符串切片类型，用于JSON存储
type StringSlice []string

// Scan 实现 sql.Scanner 接口
func (s *StringSlice) Scan(value any) error {
	if value == nil {
		*s = nil
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(bytes, s)
}

// Value 实现 driver.Valuer 接口
func (s StringSlice) Value() (driver.Value, error) {
	if s == nil {
		return nil, nil
	}
	return json.Marshal(s)
}

// AIKnowledge AI知识库模型
type AIKnowledge struct {
	ID        uint              `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time         `json:"created_at"`
	UpdatedAt time.Time         `json:"updated_at"`
	DeletedAt gorm.DeletedAt    `gorm:"index" json:"deleted_at,omitempty"`
	Title     string            `gorm:"size:255;not null" json:"title"`         // 知识标题
	Content   string            `gorm:"type:text;not null" json:"content"`      // 知识内容(Markdown)
	Category  KnowledgeCategory `gorm:"size:50;not null;index" json:"category"` // 分类
	Tags      StringSlice       `gorm:"type:json" json:"tags"`                  // 标签列表
	Embedding []byte            `gorm:"type:blob" json:"-"`                     // 向量嵌入
	IsActive  bool              `gorm:"default:true;index" json:"is_active"`    // 是否启用
	ViewCount int               `gorm:"default:0" json:"view_count"`            // 查看次数
	CreatedBy *uint             `gorm:"index" json:"created_by"`                // 创建人ID
	UpdatedBy *uint             `json:"updated_by"`                             // 更新人ID
}

// TableName 指定表名
func (AIKnowledge) TableName() string {
	return "ai_knowledge"
}

// KnowledgeItem 知识搜索结果项
type KnowledgeItem struct {
	ID       uint              `json:"id"`
	Title    string            `json:"title"`
	Content  string            `json:"content"`
	Category KnowledgeCategory `json:"category"`
	Score    float64           `json:"score"` // 相关性分数
}

// Document 文档导入结构
type Document struct {
	Title    string   `json:"title"`
	Content  string   `json:"content"`
	Category string   `json:"category"`
	Tags     []string `json:"tags"`
}
