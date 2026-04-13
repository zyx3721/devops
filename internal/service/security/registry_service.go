package security

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"gorm.io/gorm"

	"devops/internal/models"
	"devops/pkg/dto"
	"devops/pkg/logger"
)

// RegistryService 镜像仓库服务
type RegistryService struct {
	db *gorm.DB
}

// NewRegistryService 创建镜像仓库服务
func NewRegistryService(db *gorm.DB) *RegistryService {
	return &RegistryService{db: db}
}

// List 获取仓库列表
func (s *RegistryService) List(ctx context.Context) ([]dto.ImageRegistryItem, error) {
	var registries []models.ImageRegistry
	if err := s.db.Order("created_at DESC").Find(&registries).Error; err != nil {
		return nil, err
	}

	items := make([]dto.ImageRegistryItem, 0, len(registries))
	for _, r := range registries {
		items = append(items, dto.ImageRegistryItem{
			ID:        r.ID,
			Name:      r.Name,
			Type:      r.Type,
			URL:       r.URL,
			Username:  r.Username,
			IsDefault: r.IsDefault,
			CreatedAt: r.CreatedAt,
		})
	}

	return items, nil
}

// Create 创建仓库
func (s *RegistryService) Create(ctx context.Context, req *dto.ImageRegistryRequest) error {
	log := logger.L().WithField("name", req.Name)

	// 如果设为默认，取消其他默认
	if req.IsDefault {
		s.db.Model(&models.ImageRegistry{}).Where("is_default = ?", true).Update("is_default", false)
	}

	registry := &models.ImageRegistry{
		Name:      req.Name,
		Type:      req.Type,
		URL:       req.URL,
		Username:  req.Username,
		Password:  req.Password, // TODO: 加密存储
		IsDefault: req.IsDefault,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := s.db.Create(registry).Error; err != nil {
		log.WithField("error", err).Error("创建仓库失败")
		return err
	}

	log.Info("创建仓库成功")
	return nil
}

// Update 更新仓库
func (s *RegistryService) Update(ctx context.Context, req *dto.ImageRegistryRequest) error {
	log := logger.L().WithField("id", req.ID)

	var registry models.ImageRegistry
	if err := s.db.First(&registry, req.ID).Error; err != nil {
		return err
	}

	// 如果设为默认，取消其他默认
	if req.IsDefault && !registry.IsDefault {
		s.db.Model(&models.ImageRegistry{}).Where("is_default = ?", true).Update("is_default", false)
	}

	registry.Name = req.Name
	registry.Type = req.Type
	registry.URL = req.URL
	registry.Username = req.Username
	if req.Password != "" {
		registry.Password = req.Password // TODO: 加密存储
	}
	registry.IsDefault = req.IsDefault
	registry.UpdatedAt = time.Now()

	if err := s.db.Save(&registry).Error; err != nil {
		log.WithField("error", err).Error("更新仓库失败")
		return err
	}

	log.Info("更新仓库成功")
	return nil
}

// Delete 删除仓库
func (s *RegistryService) Delete(ctx context.Context, id uint) error {
	return s.db.Delete(&models.ImageRegistry{}, id).Error
}

// Get 获取仓库详情
func (s *RegistryService) Get(ctx context.Context, id uint) (*models.ImageRegistry, error) {
	var registry models.ImageRegistry
	if err := s.db.First(&registry, id).Error; err != nil {
		return nil, err
	}
	return &registry, nil
}

// TestConnection 测试连接
func (s *RegistryService) TestConnection(ctx context.Context, req *dto.ImageRegistryRequest) error {
	log := logger.L().WithField("url", req.URL)

	client := &http.Client{Timeout: 10 * time.Second}

	// 根据仓库类型构建测试URL
	var testURL string
	switch req.Type {
	case "harbor":
		testURL = strings.TrimSuffix(req.URL, "/") + "/api/v2.0/ping"
	case "dockerhub":
		testURL = "https://hub.docker.com/v2/"
	default:
		testURL = strings.TrimSuffix(req.URL, "/") + "/v2/"
	}

	httpReq, err := http.NewRequestWithContext(ctx, "GET", testURL, nil)
	if err != nil {
		return err
	}

	// 添加认证
	if req.Username != "" && req.Password != "" {
		auth := base64.StdEncoding.EncodeToString([]byte(req.Username + ":" + req.Password))
		httpReq.Header.Set("Authorization", "Basic "+auth)
	}

	resp, err := client.Do(httpReq)
	if err != nil {
		log.WithField("error", err).Error("连接仓库失败")
		return fmt.Errorf("连接失败: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return fmt.Errorf("认证失败，请检查用户名和密码")
	}

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("连接失败: %s", string(body))
	}

	log.Info("仓库连接测试成功")
	return nil
}

// ListImages 列出仓库镜像
func (s *RegistryService) ListImages(ctx context.Context, registryID uint) ([]dto.RegistryImageItem, error) {
	registry, err := s.Get(ctx, registryID)
	if err != nil {
		return nil, err
	}

	client := &http.Client{Timeout: 30 * time.Second}

	// 构建请求URL
	var catalogURL string
	switch registry.Type {
	case "harbor":
		catalogURL = strings.TrimSuffix(registry.URL, "/") + "/v2/_catalog"
	default:
		catalogURL = strings.TrimSuffix(registry.URL, "/") + "/v2/_catalog"
	}

	req, err := http.NewRequestWithContext(ctx, "GET", catalogURL, nil)
	if err != nil {
		return nil, err
	}

	// 添加认证
	if registry.Username != "" && registry.Password != "" {
		auth := base64.StdEncoding.EncodeToString([]byte(registry.Username + ":" + registry.Password))
		req.Header.Set("Authorization", "Basic "+auth)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("获取镜像列表失败: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("获取镜像列表失败: %s", string(body))
	}

	var catalog struct {
		Repositories []string `json:"repositories"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&catalog); err != nil {
		return nil, err
	}

	items := make([]dto.RegistryImageItem, 0, len(catalog.Repositories))
	for _, repo := range catalog.Repositories {
		items = append(items, dto.RegistryImageItem{
			Name: repo,
			Tags: []string{}, // 可以进一步获取tags
		})
	}

	return items, nil
}
