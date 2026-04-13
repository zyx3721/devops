package pipeline

import (
	"context"
	"devops/internal/models"
	"devops/pkg/logger"
	"encoding/base64"
	"fmt"
	"net/http"
	"time"

	"gorm.io/gorm"
)

// RegistryService 制品库服务
type RegistryService struct {
	db *gorm.DB
}

// NewRegistryService 创建制品库服务
func NewRegistryService(db *gorm.DB) *RegistryService {
	return &RegistryService{db: db}
}

// List 列表查询
func (s *RegistryService) List(ctx context.Context, page, pageSize int) ([]models.ArtifactRegistry, int64, error) {
	var registries []models.ArtifactRegistry
	var total int64

	query := s.db.WithContext(ctx).Model(&models.ArtifactRegistry{})
	query.Count(&total)

	if err := query.Order("is_default DESC, created_at DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&registries).Error; err != nil {
		return nil, 0, err
	}

	return registries, total, nil
}

// Get 获取详情
func (s *RegistryService) Get(ctx context.Context, id uint) (*models.ArtifactRegistry, error) {
	var registry models.ArtifactRegistry
	if err := s.db.WithContext(ctx).First(&registry, id).Error; err != nil {
		return nil, err
	}
	return &registry, nil
}

// Create 创建
func (s *RegistryService) Create(ctx context.Context, registry *models.ArtifactRegistry) error {
	log := logger.L().WithField("name", registry.Name)

	// 如果设为默认，取消其他默认
	if registry.IsDefault {
		s.db.WithContext(ctx).Model(&models.ArtifactRegistry{}).
			Where("is_default = ?", true).
			Update("is_default", false)
	}

	registry.Status = "unknown"
	registry.CreatedAt = time.Now()
	registry.UpdatedAt = time.Now()

	if err := s.db.WithContext(ctx).Create(registry).Error; err != nil {
		log.WithField("error", err).Error("创建制品库失败")
		return err
	}

	log.Info("创建制品库成功")
	return nil
}

// Update 更新
func (s *RegistryService) Update(ctx context.Context, id uint, registry *models.ArtifactRegistry) error {
	log := logger.L().WithField("id", id)

	var existing models.ArtifactRegistry
	if err := s.db.WithContext(ctx).First(&existing, id).Error; err != nil {
		return err
	}

	// 如果设为默认，取消其他默认
	if registry.IsDefault && !existing.IsDefault {
		s.db.WithContext(ctx).Model(&models.ArtifactRegistry{}).
			Where("is_default = ? AND id != ?", true, id).
			Update("is_default", false)
	}

	updates := map[string]interface{}{
		"name":        registry.Name,
		"type":        registry.Type,
		"url":         registry.URL,
		"username":    registry.Username,
		"description": registry.Description,
		"is_default":  registry.IsDefault,
		"updated_at":  time.Now(),
	}

	// 只有提供了新密码才更新
	if registry.Password != "" {
		updates["password"] = registry.Password
	}

	if err := s.db.WithContext(ctx).Model(&existing).Updates(updates).Error; err != nil {
		log.WithField("error", err).Error("更新制品库失败")
		return err
	}

	log.Info("更新制品库成功")
	return nil
}

// Delete 删除
func (s *RegistryService) Delete(ctx context.Context, id uint) error {
	return s.db.WithContext(ctx).Delete(&models.ArtifactRegistry{}, id).Error
}

// TestConnection 测试连接
func (s *RegistryService) TestConnection(ctx context.Context, id uint) (bool, string, error) {
	log := logger.L().WithField("id", id)

	var registry models.ArtifactRegistry
	if err := s.db.WithContext(ctx).First(&registry, id).Error; err != nil {
		return false, "", err
	}

	connected, errMsg := s.testRegistryConnection(&registry)

	// 更新状态
	status := "disconnected"
	if connected {
		status = "connected"
	}
	s.db.WithContext(ctx).Model(&registry).Update("status", status)

	log.WithField("connected", connected).Info("测试制品库连接")
	return connected, errMsg, nil
}

// testRegistryConnection 测试连接
func (s *RegistryService) testRegistryConnection(registry *models.ArtifactRegistry) (bool, string) {
	client := &http.Client{Timeout: 10 * time.Second}

	var testURL string
	switch registry.Type {
	case "harbor":
		testURL = fmt.Sprintf("%s/api/v2.0/ping", registry.URL)
	case "nexus":
		testURL = fmt.Sprintf("%s/service/rest/v1/status", registry.URL)
	case "dockerhub":
		testURL = "https://hub.docker.com/v2/"
	default:
		testURL = registry.URL
	}

	req, err := http.NewRequest("GET", testURL, nil)
	if err != nil {
		return false, err.Error()
	}

	// 添加认证
	if registry.Username != "" && registry.Password != "" {
		auth := base64.StdEncoding.EncodeToString([]byte(registry.Username + ":" + registry.Password))
		req.Header.Set("Authorization", "Basic "+auth)
	}

	resp, err := client.Do(req)
	if err != nil {
		return false, err.Error()
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return true, ""
	}

	return false, fmt.Sprintf("HTTP %d", resp.StatusCode)
}

// GetDefault 获取默认制品库
func (s *RegistryService) GetDefault(ctx context.Context) (*models.ArtifactRegistry, error) {
	var registry models.ArtifactRegistry
	if err := s.db.WithContext(ctx).Where("is_default = ?", true).First(&registry).Error; err != nil {
		return nil, err
	}
	return &registry, nil
}
