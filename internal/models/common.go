package models

import (
	"database/sql/driver"
	"encoding/json"
	"time"
)

// JSONMap 通用 JSON Map 类型
type JSONMap map[string]any

func (m *JSONMap) Scan(value any) error {
	if value == nil {
		*m = make(map[string]any)
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(bytes, m)
}

func (m JSONMap) Value() (driver.Value, error) {
	if m == nil {
		return json.Marshal(map[string]any{})
	}
	return json.Marshal(m)
}

// EncryptionKey 加密密钥
type EncryptionKey struct {
	ID           uint       `gorm:"primarykey" json:"id"`
	KeyID        string     `gorm:"size:100;uniqueIndex" json:"key_id"`
	EncryptedKey []byte     `gorm:"type:blob" json:"-"`
	Algorithm    string     `gorm:"size:20;default:'AES-256-GCM'" json:"algorithm"`
	Status       string     `gorm:"size:20;default:'active'" json:"status"` // active, rotating, retired
	Version      int        `gorm:"default:1" json:"version"`
	CreatedAt    time.Time  `json:"created_at"`
	RotatedAt    *time.Time `json:"rotated_at,omitempty"`
}

func (EncryptionKey) TableName() string { return "encryption_keys" }
