package ai

import (
	"context"
	"fmt"
	"strings"

	"gorm.io/gorm"

	"devops/internal/models/ai"
	"devops/internal/repository"
)

// KnowledgeService 知识库服务
type KnowledgeService struct {
	repo *repository.AIKnowledgeRepository
	db   *gorm.DB
}

// NewKnowledgeService 创建知识库服务
func NewKnowledgeService(db *gorm.DB) *KnowledgeService {
	return &KnowledgeService{
		repo: repository.NewAIKnowledgeRepository(db),
		db:   db,
	}
}

// Search 搜索知识
func (s *KnowledgeService) Search(ctx context.Context, query string, limit int) ([]ai.KnowledgeItem, error) {
	if limit <= 0 {
		limit = 5
	}
	return s.repo.Search(ctx, query, limit)
}

// GetByID 根据ID获取知识
func (s *KnowledgeService) GetByID(ctx context.Context, id uint) (*ai.AIKnowledge, error) {
	knowledge, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// 增加查看次数
	_ = s.repo.IncrementViewCount(ctx, id)

	return knowledge, nil
}

// List 获取知识列表
func (s *KnowledgeService) List(ctx context.Context, filter repository.AIKnowledgeFilter, page, pageSize int) ([]ai.AIKnowledge, int64, error) {
	return s.repo.List(ctx, filter, page, pageSize)
}

// GetByCategory 按分类获取知识
func (s *KnowledgeService) GetByCategory(ctx context.Context, category ai.KnowledgeCategory) ([]ai.AIKnowledge, error) {
	return s.repo.GetByCategory(ctx, category)
}

// GetAllCategories 获取所有分类
func (s *KnowledgeService) GetAllCategories(ctx context.Context) ([]string, error) {
	return s.repo.GetAllCategories(ctx)
}

// AddDocument 添加文档
func (s *KnowledgeService) AddDocument(ctx context.Context, doc ai.Document, userID uint) (*ai.AIKnowledge, error) {
	// 验证分类
	category := ai.KnowledgeCategory(doc.Category)
	if !isValidCategory(category) {
		category = ai.CategoryGeneral
	}

	knowledge := &ai.AIKnowledge{
		Title:     doc.Title,
		Content:   doc.Content,
		Category:  category,
		Tags:      doc.Tags,
		IsActive:  true,
		CreatedBy: &userID,
		UpdatedBy: &userID,
	}

	if err := s.repo.Create(ctx, knowledge); err != nil {
		return nil, fmt.Errorf("create knowledge: %w", err)
	}

	return knowledge, nil
}

// UpdateDocument 更新文档
func (s *KnowledgeService) UpdateDocument(ctx context.Context, id uint, doc ai.Document, userID uint) (*ai.AIKnowledge, error) {
	knowledge, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get knowledge: %w", err)
	}

	// 更新字段
	if doc.Title != "" {
		knowledge.Title = doc.Title
	}
	if doc.Content != "" {
		knowledge.Content = doc.Content
	}
	if doc.Category != "" {
		category := ai.KnowledgeCategory(doc.Category)
		if isValidCategory(category) {
			knowledge.Category = category
		}
	}
	if doc.Tags != nil {
		knowledge.Tags = doc.Tags
	}
	knowledge.UpdatedBy = &userID

	if err := s.repo.Update(ctx, knowledge); err != nil {
		return nil, fmt.Errorf("update knowledge: %w", err)
	}

	return knowledge, nil
}

// Delete 删除文档
func (s *KnowledgeService) Delete(ctx context.Context, id uint) error {
	return s.repo.Delete(ctx, id)
}

// SetActive 设置文档状态
func (s *KnowledgeService) SetActive(ctx context.Context, id uint, active bool) error {
	knowledge, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	knowledge.IsActive = active
	return s.repo.Update(ctx, knowledge)
}

// ImportMarkdown 导入Markdown文档
func (s *KnowledgeService) ImportMarkdown(ctx context.Context, title, content, category string, tags []string, userID uint) (*ai.AIKnowledge, error) {
	doc := ai.Document{
		Title:    title,
		Content:  content,
		Category: category,
		Tags:     tags,
	}
	return s.AddDocument(ctx, doc, userID)
}

// GetKnowledgeForContext 获取与上下文相关的知识
func (s *KnowledgeService) GetKnowledgeForContext(ctx context.Context, pageCtx *ai.PageContext) ([]ai.KnowledgeItem, error) {
	if pageCtx == nil {
		return nil, nil
	}

	// 根据页面类型确定相关分类
	var categories []ai.KnowledgeCategory
	switch {
	case strings.Contains(pageCtx.Page, "application") || pageCtx.Application != nil:
		categories = append(categories, ai.CategoryApplication)
	case strings.Contains(pageCtx.Page, "traffic"):
		categories = append(categories, ai.CategoryTraffic)
	case strings.Contains(pageCtx.Page, "approval"):
		categories = append(categories, ai.CategoryApproval)
	case strings.Contains(pageCtx.Page, "k8s") || strings.Contains(pageCtx.Page, "cluster") || pageCtx.Cluster != nil:
		categories = append(categories, ai.CategoryK8s)
	case strings.Contains(pageCtx.Page, "alert") || strings.Contains(pageCtx.Page, "monitor") || pageCtx.Alert != nil:
		categories = append(categories, ai.CategoryMonitoring)
	case strings.Contains(pageCtx.Page, "pipeline") || strings.Contains(pageCtx.Page, "cicd"):
		categories = append(categories, ai.CategoryCICD)
	}

	if len(categories) == 0 {
		return nil, nil
	}

	// 获取相关知识
	var results []ai.KnowledgeItem
	for _, cat := range categories {
		items, err := s.repo.GetByCategory(ctx, cat)
		if err != nil {
			continue
		}
		for _, item := range items {
			results = append(results, ai.KnowledgeItem{
				ID:       item.ID,
				Title:    item.Title,
				Content:  item.Content,
				Category: item.Category,
				Score:    1.0,
			})
		}
	}

	return results, nil
}

// BuildKnowledgeContext 构建知识上下文（用于System Prompt）
func (s *KnowledgeService) BuildKnowledgeContext(ctx context.Context, pageCtx *ai.PageContext, maxLength int) (string, error) {
	items, err := s.GetKnowledgeForContext(ctx, pageCtx)
	if err != nil {
		return "", err
	}

	if len(items) == 0 {
		return "", nil
	}

	var builder strings.Builder
	currentLength := 0

	for _, item := range items {
		section := fmt.Sprintf("### %s\n%s\n\n", item.Title, item.Content)
		if currentLength+len(section) > maxLength {
			break
		}
		builder.WriteString(section)
		currentLength += len(section)
	}

	return builder.String(), nil
}

// isValidCategory 检查分类是否有效
func isValidCategory(category ai.KnowledgeCategory) bool {
	validCategories := []ai.KnowledgeCategory{
		ai.CategoryApplication,
		ai.CategoryTraffic,
		ai.CategoryApproval,
		ai.CategoryK8s,
		ai.CategoryMonitoring,
		ai.CategoryCICD,
		ai.CategoryGeneral,
	}

	for _, c := range validCategories {
		if c == category {
			return true
		}
	}
	return false
}

// GetCategoryDisplayName 获取分类显示名称
func GetCategoryDisplayName(category ai.KnowledgeCategory) string {
	names := map[ai.KnowledgeCategory]string{
		ai.CategoryApplication: "应用管理",
		ai.CategoryTraffic:     "流量治理",
		ai.CategoryApproval:    "审批流程",
		ai.CategoryK8s:         "K8s管理",
		ai.CategoryMonitoring:  "监控告警",
		ai.CategoryCICD:        "CI/CD流水线",
		ai.CategoryGeneral:     "通用",
	}

	if name, ok := names[category]; ok {
		return name
	}
	return string(category)
}
