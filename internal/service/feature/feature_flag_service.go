package feature

import (
	"context"
	"crypto/md5"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"gorm.io/gorm"

	"devops/internal/models"
)

var (
	ErrFeatureFlagNotFound = errors.New("功能开关不存在")
)

// FeatureFlag 功能开关
type FeatureFlag struct {
	ID                uint            `gorm:"primarykey" json:"id"`
	Name              string          `gorm:"size:100;not null;uniqueIndex" json:"name"`
	DisplayName       string          `gorm:"size:200" json:"display_name,omitempty"`
	Description       string          `gorm:"type:text" json:"description,omitempty"`
	IsEnabled         bool            `gorm:"default:false" json:"is_enabled"`
	RolloutPercentage int             `gorm:"default:0" json:"rollout_percentage"`
	TenantWhitelist   models.JSONMap  `gorm:"type:json" json:"tenant_whitelist,omitempty"`
	TenantBlacklist   models.JSONMap  `gorm:"type:json" json:"tenant_blacklist,omitempty"`
	CreatedAt         time.Time       `json:"created_at"`
	UpdatedAt         time.Time       `json:"updated_at"`
}

func (FeatureFlag) TableName() string { return "feature_flags" }

// FeatureFlagService 功能开关服务接口
type FeatureFlagService interface {
	// 功能开关管理
	Create(ctx context.Context, req *CreateFeatureFlagRequest) (*FeatureFlag, error)
	Get(ctx context.Context, name string) (*FeatureFlag, error)
	Update(ctx context.Context, name string, req *UpdateFeatureFlagRequest) error
	Delete(ctx context.Context, name string) error
	List(ctx context.Context) ([]FeatureFlag, error)

	// 功能检查
	IsEnabled(ctx context.Context, name string, tenantID uint) bool
	IsEnabledForUser(ctx context.Context, name string, userID uint) bool

	// 批量检查
	GetEnabledFeatures(ctx context.Context, tenantID uint) ([]string, error)
}

// CreateFeatureFlagRequest 创建功能开关请求
type CreateFeatureFlagRequest struct {
	Name              string   `json:"name" binding:"required"`
	DisplayName       string   `json:"display_name,omitempty"`
	Description       string   `json:"description,omitempty"`
	IsEnabled         bool     `json:"is_enabled"`
	RolloutPercentage int      `json:"rollout_percentage"`
	TenantWhitelist   []uint   `json:"tenant_whitelist,omitempty"`
	TenantBlacklist   []uint   `json:"tenant_blacklist,omitempty"`
}

// UpdateFeatureFlagRequest 更新功能开关请求
type UpdateFeatureFlagRequest struct {
	DisplayName       string `json:"display_name,omitempty"`
	Description       string `json:"description,omitempty"`
	IsEnabled         *bool  `json:"is_enabled,omitempty"`
	RolloutPercentage *int   `json:"rollout_percentage,omitempty"`
	TenantWhitelist   []uint `json:"tenant_whitelist,omitempty"`
	TenantBlacklist   []uint `json:"tenant_blacklist,omitempty"`
}

// featureFlagServiceImpl 功能开关服务实现
type featureFlagServiceImpl struct {
	db    *gorm.DB
	cache sync.Map
}

// NewFeatureFlagService 创建功能开关服务
func NewFeatureFlagService(db *gorm.DB) FeatureFlagService {
	return &featureFlagServiceImpl{db: db}
}

func (s *featureFlagServiceImpl) Create(ctx context.Context, req *CreateFeatureFlagRequest) (*FeatureFlag, error) {
	flag := &FeatureFlag{
		Name:              req.Name,
		DisplayName:       req.DisplayName,
		Description:       req.Description,
		IsEnabled:         req.IsEnabled,
		RolloutPercentage: req.RolloutPercentage,
	}

	if len(req.TenantWhitelist) > 0 {
		flag.TenantWhitelist = map[string]interface{}{"ids": req.TenantWhitelist}
	}
	if len(req.TenantBlacklist) > 0 {
		flag.TenantBlacklist = map[string]interface{}{"ids": req.TenantBlacklist}
	}

	if err := s.db.WithContext(ctx).Create(flag).Error; err != nil {
		return nil, fmt.Errorf("创建功能开关失败: %w", err)
	}

	// 清除缓存
	s.cache.Delete(req.Name)

	return flag, nil
}

func (s *featureFlagServiceImpl) Get(ctx context.Context, name string) (*FeatureFlag, error) {
	// 先从缓存获取
	if cached, ok := s.cache.Load(name); ok {
		return cached.(*FeatureFlag), nil
	}

	var flag FeatureFlag
	if err := s.db.WithContext(ctx).Where("name = ?", name).First(&flag).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrFeatureFlagNotFound
		}
		return nil, fmt.Errorf("查询功能开关失败: %w", err)
	}

	// 缓存
	s.cache.Store(name, &flag)

	return &flag, nil
}

func (s *featureFlagServiceImpl) Update(ctx context.Context, name string, req *UpdateFeatureFlagRequest) error {
	updates := make(map[string]interface{})
	if req.DisplayName != "" {
		updates["display_name"] = req.DisplayName
	}
	if req.Description != "" {
		updates["description"] = req.Description
	}
	if req.IsEnabled != nil {
		updates["is_enabled"] = *req.IsEnabled
	}
	if req.RolloutPercentage != nil {
		updates["rollout_percentage"] = *req.RolloutPercentage
	}
	if req.TenantWhitelist != nil {
		updates["tenant_whitelist"] = map[string]interface{}{"ids": req.TenantWhitelist}
	}
	if req.TenantBlacklist != nil {
		updates["tenant_blacklist"] = map[string]interface{}{"ids": req.TenantBlacklist}
	}

	if len(updates) == 0 {
		return nil
	}

	result := s.db.WithContext(ctx).Model(&FeatureFlag{}).Where("name = ?", name).Updates(updates)
	if result.Error != nil {
		return fmt.Errorf("更新功能开关失败: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrFeatureFlagNotFound
	}

	// 清除缓存
	s.cache.Delete(name)

	return nil
}

func (s *featureFlagServiceImpl) Delete(ctx context.Context, name string) error {
	result := s.db.WithContext(ctx).Where("name = ?", name).Delete(&FeatureFlag{})
	if result.Error != nil {
		return fmt.Errorf("删除功能开关失败: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrFeatureFlagNotFound
	}

	// 清除缓存
	s.cache.Delete(name)

	return nil
}

func (s *featureFlagServiceImpl) List(ctx context.Context) ([]FeatureFlag, error) {
	var flags []FeatureFlag
	if err := s.db.WithContext(ctx).Order("name ASC").Find(&flags).Error; err != nil {
		return nil, fmt.Errorf("查询功能开关列表失败: %w", err)
	}
	return flags, nil
}

func (s *featureFlagServiceImpl) IsEnabled(ctx context.Context, name string, tenantID uint) bool {
	flag, err := s.Get(ctx, name)
	if err != nil {
		return false
	}

	// 检查是否全局启用
	if !flag.IsEnabled {
		return false
	}

	// 检查黑名单
	if flag.TenantBlacklist != nil {
		if ids, ok := flag.TenantBlacklist["ids"]; ok {
			if s.containsTenant(ids, tenantID) {
				return false
			}
		}
	}

	// 检查白名单
	if flag.TenantWhitelist != nil {
		if ids, ok := flag.TenantWhitelist["ids"]; ok {
			if s.containsTenant(ids, tenantID) {
				return true
			}
		}
	}

	// 检查灰度百分比
	if flag.RolloutPercentage > 0 && flag.RolloutPercentage < 100 {
		return s.isInRollout(name, tenantID, flag.RolloutPercentage)
	}

	return flag.RolloutPercentage == 100 || flag.IsEnabled
}

func (s *featureFlagServiceImpl) IsEnabledForUser(ctx context.Context, name string, userID uint) bool {
	flag, err := s.Get(ctx, name)
	if err != nil {
		return false
	}

	if !flag.IsEnabled {
		return false
	}

	// 检查灰度百分比
	if flag.RolloutPercentage > 0 && flag.RolloutPercentage < 100 {
		return s.isInRolloutByUser(name, userID, flag.RolloutPercentage)
	}

	return flag.RolloutPercentage == 100 || flag.IsEnabled
}

func (s *featureFlagServiceImpl) GetEnabledFeatures(ctx context.Context, tenantID uint) ([]string, error) {
	flags, err := s.List(ctx)
	if err != nil {
		return nil, err
	}

	var enabled []string
	for _, flag := range flags {
		if s.IsEnabled(ctx, flag.Name, tenantID) {
			enabled = append(enabled, flag.Name)
		}
	}

	return enabled, nil
}

// containsTenant 检查租户是否在列表中
func (s *featureFlagServiceImpl) containsTenant(ids interface{}, tenantID uint) bool {
	switch v := ids.(type) {
	case []interface{}:
		for _, id := range v {
			if uint(id.(float64)) == tenantID {
				return true
			}
		}
	case []uint:
		for _, id := range v {
			if id == tenantID {
				return true
			}
		}
	case string:
		var idList []uint
		if json.Unmarshal([]byte(v), &idList) == nil {
			for _, id := range idList {
				if id == tenantID {
					return true
				}
			}
		}
	}
	return false
}

// isInRollout 检查租户是否在灰度范围内
func (s *featureFlagServiceImpl) isInRollout(name string, tenantID uint, percentage int) bool {
	// 使用一致性哈希确保同一租户始终得到相同结果
	hash := md5.Sum([]byte(fmt.Sprintf("%s:%d", name, tenantID)))
	bucket := int(hash[0]) % 100
	return bucket < percentage
}

// isInRolloutByUser 检查用户是否在灰度范围内
func (s *featureFlagServiceImpl) isInRolloutByUser(name string, userID uint, percentage int) bool {
	hash := md5.Sum([]byte(fmt.Sprintf("%s:user:%d", name, userID)))
	bucket := int(hash[0]) % 100
	return bucket < percentage
}
