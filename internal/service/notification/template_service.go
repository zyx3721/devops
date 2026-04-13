package notification

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"text/template"

	"devops/internal/models/system"
	"devops/internal/modules/system/repository"
)

// TemplateService 模板服务
type TemplateService struct {
	repo *repository.MessageTemplateRepository
}

func NewTemplateService(repo *repository.MessageTemplateRepository) *TemplateService {
	return &TemplateService{repo: repo}
}

// Render 渲染模板
func (s *TemplateService) Render(ctx context.Context, templateName string, data interface{}) (string, error) {
	// 1. 获取模板
	tmplModel, err := s.repo.GetByName(ctx, templateName)
	if err != nil {
		return "", fmt.Errorf("failed to get template '%s': %w", templateName, err)
	}

	if !tmplModel.IsActive {
		return "", fmt.Errorf("template '%s' is inactive", templateName)
	}

	// 2. 解析模板
	// 使用 text/template 解析存储在 Content 中的字符串
	tmpl, err := template.New(templateName).Parse(tmplModel.Content)
	if err != nil {
		return "", fmt.Errorf("failed to parse template content: %w", err)
	}

	// 3. 执行渲染
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	// 4. 验证生成的 JSON 是否合法 (可选，但推荐)
	// 如果是 JSON 类型的模板，验证一下
	if tmplModel.Type == "json" || tmplModel.Type == "card" {
		var js map[string]interface{}
		if err := json.Unmarshal(buf.Bytes(), &js); err != nil {
			return "", fmt.Errorf("rendered content is not valid JSON: %w", err)
		}
	}

	return buf.String(), nil
}

// RenderByID 根据ID渲染模板
func (s *TemplateService) RenderByID(ctx context.Context, id uint, data interface{}) (string, error) {
	// 1. 获取模板
	tmplModel, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return "", fmt.Errorf("failed to get template id %d: %w", id, err)
	}

	if !tmplModel.IsActive {
		return "", fmt.Errorf("template '%s' (id: %d) is inactive", tmplModel.Name, id)
	}

	// 2. 解析模板
	tmpl, err := template.New(tmplModel.Name).Parse(tmplModel.Content)
	if err != nil {
		return "", fmt.Errorf("failed to parse template content: %w", err)
	}

	// 3. 执行渲染
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	// 4. 验证 JSON
	if tmplModel.Type == "json" || tmplModel.Type == "card" {
		var js map[string]interface{}
		if err := json.Unmarshal(buf.Bytes(), &js); err != nil {
			return "", fmt.Errorf("rendered content is not valid JSON: %w", err)
		}
	}

	return buf.String(), nil
}

// RenderContent 直接渲染模板内容（用于预览）
func (s *TemplateService) RenderContent(ctx context.Context, content string, data interface{}) (string, error) {
	// 1. 解析模板
	tmpl, err := template.New("preview").Parse(content)
	if err != nil {
		return "", fmt.Errorf("failed to parse template content: %w", err)
	}

	// 2. 执行渲染
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.String(), nil
}

// EnsureDefaultTemplates 确保默认模板存在
func (s *TemplateService) EnsureDefaultTemplates(ctx context.Context, defaults []system.MessageTemplate) error {
	for _, def := range defaults {
		_, err := s.repo.GetByName(ctx, def.Name)
		if err != nil {
			// 假设错误是"未找到"，则创建
			// 注意：这里应该更严谨地检查错误类型，但简化处理先假设是记录不存在
			if err := s.repo.Create(ctx, &def); err != nil {
				return fmt.Errorf("failed to create default template '%s': %w", def.Name, err)
			}
			fmt.Printf("Created default template: %s\n", def.Name)
		}
	}
	return nil
}
