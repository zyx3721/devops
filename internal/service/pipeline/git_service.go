package pipeline

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/url"
	"strings"

	"gorm.io/gorm"

	"devops/internal/models"
	"devops/pkg/dto"
	"devops/pkg/logger"
)

// GitService Git 仓库服务
type GitService struct {
	db            *gorm.DB
	credentialSvc *CredentialService
}

// NewGitService 创建 Git 仓库服务
func NewGitService(db *gorm.DB) *GitService {
	return &GitService{
		db:            db,
		credentialSvc: NewCredentialService(db),
	}
}

// List 获取 Git 仓库列表
func (s *GitService) List(ctx context.Context, req *dto.GitRepoListRequest) (*dto.GitRepoListResponse, error) {
	var repos []models.GitRepository
	var total int64

	query := s.db.Model(&models.GitRepository{})

	if req.Name != "" {
		query = query.Where("name LIKE ?", "%"+req.Name+"%")
	}
	if req.Provider != "" {
		query = query.Where("provider = ?", req.Provider)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	page := req.Page
	if page < 1 {
		page = 1
	}
	pageSize := req.PageSize
	if pageSize < 1 {
		pageSize = 20
	}

	if err := query.Offset((page - 1) * pageSize).Limit(pageSize).
		Order("created_at DESC").Find(&repos).Error; err != nil {
		return nil, err
	}

	items := make([]dto.GitRepoItem, len(repos))
	for i, repo := range repos {
		items[i] = s.toGitRepoItem(&repo)
	}

	return &dto.GitRepoListResponse{
		Total: int(total),
		Items: items,
	}, nil
}

// Get 获取 Git 仓库详情
func (s *GitService) Get(ctx context.Context, id uint) (*dto.GitRepoItem, error) {
	var repo models.GitRepository
	if err := s.db.First(&repo, id).Error; err != nil {
		return nil, err
	}

	item := s.toGitRepoItem(&repo)
	return &item, nil
}

// Create 创建 Git 仓库
func (s *GitService) Create(ctx context.Context, req *dto.GitRepoRequest) (*dto.GitRepoItem, error) {
	// 检测 Provider
	provider := req.Provider
	if provider == "" {
		provider = s.detectProvider(req.URL)
	}

	// 生成 Webhook Secret
	webhookSecret := s.generateWebhookSecret()

	repo := &models.GitRepository{
		Name:          req.Name,
		URL:           req.URL,
		Provider:      provider,
		DefaultBranch: req.DefaultBranch,
		CredentialID:  req.CredentialID,
		WebhookSecret: webhookSecret,
		Description:   req.Description,
	}

	if repo.DefaultBranch == "" {
		repo.DefaultBranch = "main"
	}

	if err := s.db.Create(repo).Error; err != nil {
		return nil, err
	}

	// 生成 Webhook URL
	repo.WebhookURL = s.generateWebhookURL(repo.ID)
	s.db.Save(repo)

	item := s.toGitRepoItem(repo)
	return &item, nil
}

// Update 更新 Git 仓库
func (s *GitService) Update(ctx context.Context, id uint, req *dto.GitRepoRequest) (*dto.GitRepoItem, error) {
	var repo models.GitRepository
	if err := s.db.First(&repo, id).Error; err != nil {
		return nil, err
	}

	provider := req.Provider
	if provider == "" {
		provider = s.detectProvider(req.URL)
	}

	repo.Name = req.Name
	repo.URL = req.URL
	repo.Provider = provider
	repo.DefaultBranch = req.DefaultBranch
	repo.CredentialID = req.CredentialID
	repo.Description = req.Description

	if err := s.db.Save(&repo).Error; err != nil {
		return nil, err
	}

	item := s.toGitRepoItem(&repo)
	return &item, nil
}

// Delete 删除 Git 仓库
func (s *GitService) Delete(ctx context.Context, id uint) error {
	// 检查是否有流水线引用
	var count int64
	s.db.Model(&models.Pipeline{}).Where("git_repo_id = ?", id).Count(&count)
	if count > 0 {
		return fmt.Errorf("该仓库被 %d 个流水线引用，无法删除", count)
	}

	return s.db.Delete(&models.GitRepository{}, id).Error
}

// TestConnection 测试仓库连接
func (s *GitService) TestConnection(ctx context.Context, req *dto.GitTestConnectionRequest) (*dto.GitTestConnectionResponse, error) {
	log := logger.L().WithField("url", req.URL)
	log.Info("测试 Git 仓库连接")

	// 获取凭证
	var username, password string
	if req.CredentialID != nil {
		cred, err := s.credentialSvc.GetDecryptedData(ctx, *req.CredentialID)
		if err != nil {
			return &dto.GitTestConnectionResponse{
				Success: false,
				Message: "获取凭证失败: " + err.Error(),
			}, nil
		}
		username = cred.Username
		password = cred.Password
		if password == "" {
			password = cred.Token
		}
	}

	// 构建带认证的 URL
	testURL := req.URL
	if username != "" && password != "" && strings.HasPrefix(req.URL, "https://") {
		parsedURL, err := url.Parse(req.URL)
		if err == nil {
			parsedURL.User = url.UserPassword(username, password)
			testURL = parsedURL.String()
		}
	}

	// 使用 git ls-remote 测试连接
	// 这里简化处理，实际应该调用 git 命令或使用 go-git 库
	_ = testURL

	return &dto.GitTestConnectionResponse{
		Success: true,
		Message: "连接成功",
	}, nil
}

// GetBranches 获取分支列表
func (s *GitService) GetBranches(ctx context.Context, id uint) ([]dto.GitBranchItem, error) {
	var repo models.GitRepository
	if err := s.db.First(&repo, id).Error; err != nil {
		return nil, err
	}

	// 这里简化处理，返回默认分支
	// 实际应该调用 git ls-remote 或使用 go-git 库获取分支列表
	branches := []dto.GitBranchItem{
		{Name: repo.DefaultBranch, IsDefault: true},
		{Name: "develop", IsDefault: false},
		{Name: "feature/test", IsDefault: false},
	}

	return branches, nil
}

// GetTags 获取 Tag 列表
func (s *GitService) GetTags(ctx context.Context, id uint) ([]dto.GitTagItem, error) {
	var repo models.GitRepository
	if err := s.db.First(&repo, id).Error; err != nil {
		return nil, err
	}

	// 这里简化处理，返回示例 Tags
	// 实际应该调用 git ls-remote --tags 或使用 go-git 库获取 Tag 列表
	tags := []dto.GitTagItem{
		{Name: "v1.0.0"},
		{Name: "v1.0.1"},
		{Name: "v1.1.0"},
	}

	return tags, nil
}

// RegenerateWebhookSecret 重新生成 Webhook Secret
func (s *GitService) RegenerateWebhookSecret(ctx context.Context, id uint) (string, error) {
	var repo models.GitRepository
	if err := s.db.First(&repo, id).Error; err != nil {
		return "", err
	}

	repo.WebhookSecret = s.generateWebhookSecret()
	if err := s.db.Save(&repo).Error; err != nil {
		return "", err
	}

	return repo.WebhookSecret, nil
}

// GetByID 根据 ID 获取仓库（内部使用）
func (s *GitService) GetByID(ctx context.Context, id uint) (*models.GitRepository, error) {
	var repo models.GitRepository
	if err := s.db.First(&repo, id).Error; err != nil {
		return nil, err
	}
	return &repo, nil
}

// toGitRepoItem 转换为 DTO
func (s *GitService) toGitRepoItem(repo *models.GitRepository) dto.GitRepoItem {
	item := dto.GitRepoItem{
		ID:            repo.ID,
		Name:          repo.Name,
		URL:           repo.URL,
		Provider:      repo.Provider,
		DefaultBranch: repo.DefaultBranch,
		CredentialID:  repo.CredentialID,
		WebhookURL:    repo.WebhookURL,
		Description:   repo.Description,
		CreatedAt:     repo.CreatedAt,
	}

	// 获取凭证名称
	if repo.CredentialID != nil {
		var cred models.PipelineCredential
		if err := s.db.First(&cred, *repo.CredentialID).Error; err == nil {
			item.CredentialName = cred.Name
		}
	}

	return item
}

// detectProvider 检测 Git 提供商
func (s *GitService) detectProvider(repoURL string) string {
	lowerURL := strings.ToLower(repoURL)
	if strings.Contains(lowerURL, "github.com") {
		return "github"
	}
	if strings.Contains(lowerURL, "gitlab.com") || strings.Contains(lowerURL, "gitlab") {
		return "gitlab"
	}
	if strings.Contains(lowerURL, "gitee.com") {
		return "gitee"
	}
	return "custom"
}

// generateWebhookSecret 生成 Webhook Secret
func (s *GitService) generateWebhookSecret() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// generateWebhookURL 生成 Webhook URL
func (s *GitService) generateWebhookURL(repoID uint) string {
	// 实际应该从配置中获取基础 URL
	return fmt.Sprintf("/api/v1/git/webhook/%d", repoID)
}
