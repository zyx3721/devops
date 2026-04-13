package pipeline

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"regexp"
	"strings"
	"time"

	"gorm.io/gorm"

	"devops/internal/models"
	"devops/pkg/logger"
)

// NotifyTemplateService 通知模板服务
type NotifyTemplateService struct {
	db *gorm.DB
}

// NewNotifyTemplateService 创建通知模板服务
func NewNotifyTemplateService(db *gorm.DB) *NotifyTemplateService {
	return &NotifyTemplateService{db: db}
}

// NotifyTemplate 通知模板
type NotifyTemplate struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	TenantID    uint      `json:"tenant_id" gorm:"index"`
	Name        string    `json:"name" gorm:"size:100;not null"`
	Type        string    `json:"type" gorm:"size:50;not null"`
	Category    string    `json:"category" gorm:"size:50"`
	Subject     string    `json:"subject" gorm:"size:255"`
	Content     string    `json:"content" gorm:"type:text"`
	Variables   string    `json:"variables" gorm:"type:text"`
	IsDefault   bool      `json:"is_default" gorm:"default:false"`
	IsSystem    bool      `json:"is_system" gorm:"default:false"`
	Description string    `json:"description" gorm:"size:500"`
	CreatedBy   uint      `json:"created_by"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// TemplateVariable 模板变量定义
type TemplateVariable struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Example     string `json:"example"`
	Required    bool   `json:"required"`
}

// CreateTemplateRequest 创建模板请求
type CreateTemplateRequest struct {
	Name        string `json:"name" binding:"required"`
	Type        string `json:"type" binding:"required"`
	Category    string `json:"category"`
	Subject     string `json:"subject"`
	Content     string `json:"content" binding:"required"`
	Description string `json:"description"`
	IsDefault   bool   `json:"is_default"`
}

// UpdateTemplateRequest 更新模板请求
type UpdateTemplateRequest struct {
	ID          uint   `json:"id"`
	Name        string `json:"name"`
	Subject     string `json:"subject"`
	Content     string `json:"content"`
	Description string `json:"description"`
	IsDefault   bool   `json:"is_default"`
}

// Create 创建通知模板
func (s *NotifyTemplateService) Create(ctx context.Context, tenantID, userID uint, req *CreateTemplateRequest) (*NotifyTemplate, error) {
	log := logger.L().WithField("tenant_id", tenantID).WithField("name", req.Name)
	log.Info("创建通知模板")

	if err := s.ValidateTemplate(req.Content); err != nil {
		return nil, fmt.Errorf("模板语法错误: %w", err)
	}

	variables := s.GetAvailableVariables(req.Category)
	variablesJSON, _ := json.Marshal(variables)

	tmpl := &NotifyTemplate{
		TenantID:    tenantID,
		Name:        req.Name,
		Type:        req.Type,
		Category:    req.Category,
		Subject:     req.Subject,
		Content:     req.Content,
		Variables:   string(variablesJSON),
		IsDefault:   req.IsDefault,
		Description: req.Description,
		CreatedBy:   userID,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if req.IsDefault {
		s.db.WithContext(ctx).Model(&NotifyTemplate{}).
			Where("tenant_id = ? AND type = ? AND category = ? AND is_default = ?", tenantID, req.Type, req.Category, true).
			Update("is_default", false)
	}

	if err := s.db.WithContext(ctx).Create(tmpl).Error; err != nil {
		log.WithField("error", err).Error("创建模板失败")
		return nil, fmt.Errorf("创建模板失败: %w", err)
	}

	log.WithField("id", tmpl.ID).Info("模板创建成功")
	return tmpl, nil
}

// Update 更新通知模板
func (s *NotifyTemplateService) Update(ctx context.Context, tenantID uint, req *UpdateTemplateRequest) error {
	log := logger.L().WithField("id", req.ID)
	log.Info("更新通知模板")

	var tmpl NotifyTemplate
	if err := s.db.WithContext(ctx).Where("id = ? AND tenant_id = ?", req.ID, tenantID).First(&tmpl).Error; err != nil {
		return fmt.Errorf("模板不存在: %w", err)
	}

	if tmpl.IsSystem {
		return fmt.Errorf("系统模板不可修改")
	}

	if req.Content != "" {
		if err := s.ValidateTemplate(req.Content); err != nil {
			return fmt.Errorf("模板语法错误: %w", err)
		}
	}

	updates := map[string]interface{}{
		"updated_at": time.Now(),
	}
	if req.Name != "" {
		updates["name"] = req.Name
	}
	if req.Subject != "" {
		updates["subject"] = req.Subject
	}
	if req.Content != "" {
		updates["content"] = req.Content
	}
	if req.Description != "" {
		updates["description"] = req.Description
	}

	if req.IsDefault && !tmpl.IsDefault {
		s.db.WithContext(ctx).Model(&NotifyTemplate{}).
			Where("tenant_id = ? AND type = ? AND category = ? AND is_default = ?", tenantID, tmpl.Type, tmpl.Category, true).
			Update("is_default", false)
		updates["is_default"] = true
	}

	if err := s.db.WithContext(ctx).Model(&tmpl).Updates(updates).Error; err != nil {
		log.WithField("error", err).Error("更新模板失败")
		return fmt.Errorf("更新模板失败: %w", err)
	}

	log.Info("模板更新成功")
	return nil
}

// Delete 删除通知模板
func (s *NotifyTemplateService) Delete(ctx context.Context, tenantID, id uint) error {
	log := logger.L().WithField("id", id)
	log.Info("删除通知模板")

	var tmpl NotifyTemplate
	if err := s.db.WithContext(ctx).Where("id = ? AND tenant_id = ?", id, tenantID).First(&tmpl).Error; err != nil {
		return fmt.Errorf("模板不存在: %w", err)
	}

	if tmpl.IsSystem {
		return fmt.Errorf("系统模板不可删除")
	}

	if err := s.db.WithContext(ctx).Delete(&tmpl).Error; err != nil {
		log.WithField("error", err).Error("删除模板失败")
		return fmt.Errorf("删除模板失败: %w", err)
	}

	log.Info("模板删除成功")
	return nil
}

// Get 获取模板详情
func (s *NotifyTemplateService) Get(ctx context.Context, tenantID, id uint) (*NotifyTemplate, error) {
	var tmpl NotifyTemplate
	if err := s.db.WithContext(ctx).Where("id = ? AND (tenant_id = ? OR is_system = ?)", id, tenantID, true).First(&tmpl).Error; err != nil {
		return nil, fmt.Errorf("模板不存在: %w", err)
	}
	return &tmpl, nil
}

// List 获取模板列表
func (s *NotifyTemplateService) List(ctx context.Context, tenantID uint, notifyType, category string) ([]NotifyTemplate, error) {
	var templates []NotifyTemplate

	query := s.db.WithContext(ctx).Where("tenant_id = ? OR is_system = ?", tenantID, true)
	if notifyType != "" {
		query = query.Where("type = ?", notifyType)
	}
	if category != "" {
		query = query.Where("category = ?", category)
	}

	if err := query.Order("is_system DESC, is_default DESC, created_at DESC").Find(&templates).Error; err != nil {
		return nil, fmt.Errorf("查询模板列表失败: %w", err)
	}

	return templates, nil
}

// GetDefault 获取默认模板
func (s *NotifyTemplateService) GetDefault(ctx context.Context, tenantID uint, notifyType, category string) (*NotifyTemplate, error) {
	var tmpl NotifyTemplate

	err := s.db.WithContext(ctx).
		Where("tenant_id = ? AND type = ? AND category = ? AND is_default = ?", tenantID, notifyType, category, true).
		First(&tmpl).Error

	if err == nil {
		return &tmpl, nil
	}

	err = s.db.WithContext(ctx).
		Where("is_system = ? AND type = ? AND category = ? AND is_default = ?", true, notifyType, category, true).
		First(&tmpl).Error

	if err != nil {
		return nil, fmt.Errorf("未找到默认模板")
	}

	return &tmpl, nil
}

// ValidateTemplate 验证模板语法
func (s *NotifyTemplateService) ValidateTemplate(content string) error {
	_, err := template.New("validate").Funcs(s.getTemplateFuncs()).Parse(content)
	return err
}

// RenderTemplate 渲染模板
func (s *NotifyTemplateService) RenderTemplate(content string, data map[string]interface{}) (string, error) {
	tmpl, err := template.New("render").Funcs(s.getTemplateFuncs()).Parse(content)
	if err != nil {
		return "", fmt.Errorf("解析模板失败: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("渲染模板失败: %w", err)
	}

	return buf.String(), nil
}

// PreviewTemplate 预览模板
func (s *NotifyTemplateService) PreviewTemplate(ctx context.Context, content, category string) (string, error) {
	data := s.GetSampleData(category)
	return s.RenderTemplate(content, data)
}

// getTemplateFuncs 获取模板函数
func (s *NotifyTemplateService) getTemplateFuncs() template.FuncMap {
	return template.FuncMap{
		"formatTime": func(t time.Time) string {
			return t.Format("2006-01-02 15:04:05")
		},
		"formatDuration": func(seconds int) string {
			if seconds < 60 {
				return fmt.Sprintf("%d秒", seconds)
			}
			if seconds < 3600 {
				return fmt.Sprintf("%d分%d秒", seconds/60, seconds%60)
			}
			return fmt.Sprintf("%d时%d分", seconds/3600, (seconds%3600)/60)
		},
		"statusEmoji": func(status string) string {
			switch status {
			case "success":
				return "✅"
			case "failed":
				return "❌"
			case "cancelled":
				return "⚠️"
			case "running":
				return "🔄"
			default:
				return "📋"
			}
		},
		"statusText": func(status string) string {
			switch status {
			case "success":
				return "成功"
			case "failed":
				return "失败"
			case "cancelled":
				return "已取消"
			case "running":
				return "运行中"
			case "pending":
				return "等待中"
			default:
				return status
			}
		},
		"statusColor": func(status string) string {
			switch status {
			case "success":
				return "green"
			case "failed":
				return "red"
			case "cancelled":
				return "orange"
			default:
				return "blue"
			}
		},
		"truncate": func(s string, length int) string {
			if len(s) <= length {
				return s
			}
			return s[:length] + "..."
		},
		"shortCommit": func(commit string) string {
			if len(commit) > 8 {
				return commit[:8]
			}
			return commit
		},
		"join": func(arr []string, sep string) string {
			return strings.Join(arr, sep)
		},
		"default": func(defaultVal, val interface{}) interface{} {
			if val == nil || val == "" {
				return defaultVal
			}
			return val
		},
		"upper": strings.ToUpper,
		"lower": strings.ToLower,
		"replace": func(old, new, s string) string {
			return strings.ReplaceAll(s, old, new)
		},
		"contains": strings.Contains,
		"hasPrefix": strings.HasPrefix,
		"hasSuffix": strings.HasSuffix,
	}
}

// GetAvailableVariables 获取可用变量
func (s *NotifyTemplateService) GetAvailableVariables(category string) []TemplateVariable {
	common := []TemplateVariable{
		{Name: "TenantName", Description: "租户名称", Example: "MyCompany", Required: false},
		{Name: "Timestamp", Description: "时间戳", Example: "2026-01-11 10:30:00", Required: true},
		{Name: "URL", Description: "详情页链接", Example: "https://devops.example.com/...", Required: false},
	}

	switch category {
	case "pipeline":
		return append(common, []TemplateVariable{
			{Name: "PipelineName", Description: "流水线名称", Example: "frontend-build", Required: true},
			{Name: "PipelineID", Description: "流水线ID", Example: "123", Required: true},
			{Name: "RunID", Description: "执行ID", Example: "456", Required: true},
			{Name: "Status", Description: "执行状态", Example: "success/failed/cancelled", Required: true},
			{Name: "TriggerBy", Description: "触发人", Example: "admin", Required: true},
			{Name: "TriggerType", Description: "触发类型", Example: "manual/webhook/schedule", Required: true},
			{Name: "GitBranch", Description: "Git分支", Example: "main", Required: false},
			{Name: "GitCommit", Description: "Git提交", Example: "abc123def", Required: false},
			{Name: "GitMessage", Description: "提交信息", Example: "fix: bug fix", Required: false},
			{Name: "Duration", Description: "执行耗时(秒)", Example: "120", Required: false},
			{Name: "StartedAt", Description: "开始时间", Example: "2026-01-11 10:00:00", Required: false},
			{Name: "FinishedAt", Description: "结束时间", Example: "2026-01-11 10:02:00", Required: false},
			{Name: "FailedStage", Description: "失败阶段", Example: "build", Required: false},
			{Name: "FailedStep", Description: "失败步骤", Example: "npm install", Required: false},
			{Name: "ErrorMessage", Description: "错误信息", Example: "npm ERR! ...", Required: false},
		}...)
	case "deploy":
		return append(common, []TemplateVariable{
			{Name: "ApplicationName", Description: "应用名称", Example: "my-app", Required: true},
			{Name: "Environment", Description: "部署环境", Example: "production", Required: true},
			{Name: "Version", Description: "部署版本", Example: "v1.2.3", Required: true},
			{Name: "Status", Description: "部署状态", Example: "success/failed", Required: true},
			{Name: "DeployBy", Description: "部署人", Example: "admin", Required: true},
			{Name: "ApprovedBy", Description: "审批人", Example: "manager", Required: false},
			{Name: "Replicas", Description: "副本数", Example: "3", Required: false},
			{Name: "Image", Description: "镜像地址", Example: "registry/app:v1.2.3", Required: false},
		}...)
	case "alert":
		return append(common, []TemplateVariable{
			{Name: "AlertName", Description: "告警名称", Example: "HighCPUUsage", Required: true},
			{Name: "Severity", Description: "告警级别", Example: "critical/warning/info", Required: true},
			{Name: "Resource", Description: "资源名称", Example: "pod/my-app-xxx", Required: true},
			{Name: "Namespace", Description: "命名空间", Example: "production", Required: false},
			{Name: "Cluster", Description: "集群名称", Example: "prod-cluster", Required: false},
			{Name: "Value", Description: "当前值", Example: "95%", Required: false},
			{Name: "Threshold", Description: "阈值", Example: "80%", Required: false},
			{Name: "Message", Description: "告警信息", Example: "CPU usage exceeded 80%", Required: true},
		}...)
	default:
		return common
	}
}

// GetSampleData 获取示例数据
func (s *NotifyTemplateService) GetSampleData(category string) map[string]interface{} {
	common := map[string]interface{}{
		"TenantName": "示例公司",
		"Timestamp":  time.Now().Format("2006-01-02 15:04:05"),
		"URL":        "https://devops.example.com/detail",
	}

	switch category {
	case "pipeline":
		return mergeMaps(common, map[string]interface{}{
			"PipelineName": "frontend-build",
			"PipelineID":   123,
			"RunID":        456,
			"Status":       "success",
			"TriggerBy":    "admin",
			"TriggerType":  "manual",
			"GitBranch":    "main",
			"GitCommit":    "abc123def456",
			"GitMessage":   "feat: add new feature",
			"Duration":     120,
			"StartedAt":    time.Now().Add(-2 * time.Minute).Format("2006-01-02 15:04:05"),
			"FinishedAt":   time.Now().Format("2006-01-02 15:04:05"),
		})
	case "deploy":
		return mergeMaps(common, map[string]interface{}{
			"ApplicationName": "my-app",
			"Environment":     "production",
			"Version":         "v1.2.3",
			"Status":          "success",
			"DeployBy":        "admin",
			"ApprovedBy":      "manager",
			"Replicas":        3,
			"Image":           "registry.example.com/my-app:v1.2.3",
		})
	case "alert":
		return mergeMaps(common, map[string]interface{}{
			"AlertName": "HighCPUUsage",
			"Severity":  "warning",
			"Resource":  "pod/my-app-xxx",
			"Namespace": "production",
			"Cluster":   "prod-cluster",
			"Value":     "85%",
			"Threshold": "80%",
			"Message":   "CPU usage exceeded threshold",
		})
	default:
		return common
	}
}

// InitSystemTemplates 初始化系统模板
func (s *NotifyTemplateService) InitSystemTemplates(ctx context.Context) error {
	log := logger.L()
	log.Info("初始化系统通知模板")

	templates := []NotifyTemplate{
		{
			Name:      "流水线通知-飞书",
			Type:      "feishu",
			Category:  "pipeline",
			Subject:   "流水线执行通知",
			Content:   s.getFeishuPipelineTemplate(),
			IsDefault: true,
			IsSystem:  true,
		},
		{
			Name:      "流水线通知-钉钉",
			Type:      "dingtalk",
			Category:  "pipeline",
			Subject:   "流水线执行通知",
			Content:   s.getDingtalkPipelineTemplate(),
			IsDefault: true,
			IsSystem:  true,
		},
		{
			Name:      "流水线通知-企业微信",
			Type:      "wechat",
			Category:  "pipeline",
			Subject:   "流水线执行通知",
			Content:   s.getWechatPipelineTemplate(),
			IsDefault: true,
			IsSystem:  true,
		},
		{
			Name:      "部署通知-飞书",
			Type:      "feishu",
			Category:  "deploy",
			Subject:   "部署通知",
			Content:   s.getFeishuDeployTemplate(),
			IsDefault: true,
			IsSystem:  true,
		},
	}

	for _, tmpl := range templates {
		var existing NotifyTemplate
		err := s.db.WithContext(ctx).Where("is_system = ? AND type = ? AND category = ?", true, tmpl.Type, tmpl.Category).First(&existing).Error
		if err == gorm.ErrRecordNotFound {
			variables := s.GetAvailableVariables(tmpl.Category)
			variablesJSON, _ := json.Marshal(variables)
			tmpl.Variables = string(variablesJSON)
			tmpl.CreatedAt = time.Now()
			tmpl.UpdatedAt = time.Now()
			if err := s.db.WithContext(ctx).Create(&tmpl).Error; err != nil {
				log.WithField("error", err).WithField("name", tmpl.Name).Error("创建系统模板失败")
			}
		}
	}

	log.Info("系统模板初始化完成")
	return nil
}

func (s *NotifyTemplateService) getFeishuPipelineTemplate() string {
	return `**执行ID**: {{.RunID}}
**触发人**: {{.TriggerBy}}
**触发方式**: {{.TriggerType}}
{{if .GitBranch}}**分支**: {{.GitBranch}}{{end}}
{{if .GitCommit}}**提交**: {{shortCommit .GitCommit}}{{end}}
{{if .GitMessage}}**提交信息**: {{truncate .GitMessage 50}}{{end}}
{{if .Duration}}**耗时**: {{formatDuration .Duration}}{{end}}`
}

func (s *NotifyTemplateService) getDingtalkPipelineTemplate() string {
	return `### 流水线执行通知

- **执行ID**: {{.RunID}}
- **触发人**: {{.TriggerBy}}
- **触发方式**: {{.TriggerType}}
{{if .GitBranch}}- **分支**: {{.GitBranch}}{{end}}
{{if .GitCommit}}- **提交**: {{shortCommit .GitCommit}}{{end}}
{{if .Duration}}- **耗时**: {{formatDuration .Duration}}{{end}}

[查看详情]({{.URL}})`
}

func (s *NotifyTemplateService) getWechatPipelineTemplate() string {
	return `## 流水线执行通知

> 执行ID: {{.RunID}}
> 触发人: {{.TriggerBy}}
{{if .GitBranch}}> 分支: {{.GitBranch}}{{end}}
{{if .Duration}}> 耗时: {{formatDuration .Duration}}{{end}}

[查看详情]({{.URL}})`
}

func (s *NotifyTemplateService) getFeishuDeployTemplate() string {
	return `**应用**: {{.ApplicationName}}
**环境**: {{.Environment}}
**版本**: {{.Version}}
**部署人**: {{.DeployBy}}
{{if .ApprovedBy}}**审批人**: {{.ApprovedBy}}{{end}}
{{if .Replicas}}**副本数**: {{.Replicas}}{{end}}`
}

// ExtractVariables 从模板中提取变量
func (s *NotifyTemplateService) ExtractVariables(content string) []string {
	re := regexp.MustCompile(`\{\{\.(\w+)\}\}`)
	matches := re.FindAllStringSubmatch(content, -1)

	varMap := make(map[string]bool)
	for _, match := range matches {
		if len(match) > 1 {
			varMap[match[1]] = true
		}
	}

	vars := make([]string, 0, len(varMap))
	for v := range varMap {
		vars = append(vars, v)
	}
	return vars
}

func mergeMaps(m1, m2 map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	for k, v := range m1 {
		result[k] = v
	}
	for k, v := range m2 {
		result[k] = v
	}
	return result
}

// 确保 models 包被使用
var _ = models.Pipeline{}
