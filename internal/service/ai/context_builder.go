package ai

import (
	"fmt"
	"strings"

	"devops/internal/models/ai"
)

// ContextBuilder 上下文构建器
type ContextBuilder struct{}

// NewContextBuilder 创建上下文构建器
func NewContextBuilder() *ContextBuilder {
	return &ContextBuilder{}
}

// BuildContextString 构建上下文字符串
func (b *ContextBuilder) BuildContextString(ctx *ai.PageContext) string {
	if ctx == nil {
		return "当前没有特定的页面上下文。"
	}

	var parts []string

	// 页面信息
	if ctx.Page != "" {
		parts = append(parts, fmt.Sprintf("当前页面: %s", b.getPageDisplayName(ctx.Page)))
	}

	// 应用上下文
	if ctx.Application != nil {
		app := ctx.Application
		parts = append(parts, fmt.Sprintf("当前应用: %s", app.Name))
		if app.Environment != "" {
			parts = append(parts, fmt.Sprintf("环境: %s", app.Environment))
		}
		if app.Status != "" {
			parts = append(parts, fmt.Sprintf("状态: %s", app.Status))
		}
	}

	// 集群上下文
	if ctx.Cluster != nil {
		parts = append(parts, fmt.Sprintf("当前集群: %s", ctx.Cluster.Name))
	}

	// 告警上下文
	if ctx.Alert != nil {
		alert := ctx.Alert
		parts = append(parts, fmt.Sprintf("当前告警: %s", alert.Message))
		if alert.Level != "" {
			parts = append(parts, fmt.Sprintf("告警级别: %s", alert.Level))
		}
	}

	// 部署上下文
	if ctx.Deployment != nil {
		deploy := ctx.Deployment
		parts = append(parts, fmt.Sprintf("当前部署: 版本 %s", deploy.Version))
		if deploy.Status != "" {
			parts = append(parts, fmt.Sprintf("部署状态: %s", deploy.Status))
		}
	}

	if len(parts) == 0 {
		return "当前没有特定的页面上下文。"
	}

	return strings.Join(parts, "\n")
}

// getPageDisplayName 获取页面显示名称
func (b *ContextBuilder) getPageDisplayName(page string) string {
	pageNames := map[string]string{
		"dashboard":        "仪表盘",
		"application":      "应用管理",
		"application-list": "应用列表",
		"application-detail": "应用详情",
		"deploy":           "部署中心",
		"deploy-list":      "部署记录",
		"deploy-detail":    "部署详情",
		"traffic":          "流量治理",
		"traffic-ratelimit": "限流配置",
		"traffic-circuit":  "熔断配置",
		"traffic-routing":  "路由配置",
		"approval":         "审批管理",
		"approval-list":    "审批列表",
		"approval-chain":   "审批链配置",
		"k8s":              "K8s管理",
		"k8s-cluster":      "集群管理",
		"k8s-workload":     "工作负载",
		"k8s-pod":          "Pod管理",
		"alert":            "告警中心",
		"alert-list":       "告警列表",
		"alert-config":     "告警配置",
		"monitor":          "监控中心",
		"pipeline":         "流水线",
		"pipeline-list":    "流水线列表",
		"pipeline-detail":  "流水线详情",
		"logs":             "日志中心",
		"cost":             "成本分析",
		"security":         "安全中心",
		"admin":            "系统管理",
	}

	if name, ok := pageNames[page]; ok {
		return name
	}
	return page
}

// BuildFromRoute 从路由构建上下文
func (b *ContextBuilder) BuildFromRoute(route string) *ai.PageContext {
	ctx := &ai.PageContext{
		Route: route,
	}

	// 解析路由确定页面类型
	parts := strings.Split(strings.Trim(route, "/"), "/")
	if len(parts) > 0 {
		ctx.Page = parts[0]
		if len(parts) > 1 {
			ctx.Page = parts[0] + "-" + parts[1]
		}
	}

	return ctx
}

// MergeContext 合并上下文
func (b *ContextBuilder) MergeContext(base, update *ai.PageContext) *ai.PageContext {
	if base == nil {
		return update
	}
	if update == nil {
		return base
	}

	result := &ai.PageContext{
		Page:  update.Page,
		Route: update.Route,
	}

	// 合并应用上下文
	if update.Application != nil {
		result.Application = update.Application
	} else {
		result.Application = base.Application
	}

	// 合并集群上下文
	if update.Cluster != nil {
		result.Cluster = update.Cluster
	} else {
		result.Cluster = base.Cluster
	}

	// 合并告警上下文
	if update.Alert != nil {
		result.Alert = update.Alert
	} else {
		result.Alert = base.Alert
	}

	// 合并部署上下文
	if update.Deployment != nil {
		result.Deployment = update.Deployment
	} else {
		result.Deployment = base.Deployment
	}

	// 合并额外上下文
	if update.Extra != nil {
		result.Extra = update.Extra
	} else {
		result.Extra = base.Extra
	}

	return result
}

// ExtractKeywords 从上下文提取关键词（用于知识库搜索）
func (b *ContextBuilder) ExtractKeywords(ctx *ai.PageContext) []string {
	if ctx == nil {
		return nil
	}

	var keywords []string

	// 页面关键词
	if ctx.Page != "" {
		keywords = append(keywords, ctx.Page)
	}

	// 应用名称
	if ctx.Application != nil && ctx.Application.Name != "" {
		keywords = append(keywords, ctx.Application.Name)
	}

	// 集群名称
	if ctx.Cluster != nil && ctx.Cluster.Name != "" {
		keywords = append(keywords, ctx.Cluster.Name)
	}

	return keywords
}

// GetContextCategory 获取上下文对应的知识分类
func (b *ContextBuilder) GetContextCategory(ctx *ai.PageContext) ai.KnowledgeCategory {
	if ctx == nil {
		return ai.CategoryGeneral
	}

	page := strings.ToLower(ctx.Page)

	switch {
	case strings.Contains(page, "application") || strings.Contains(page, "deploy"):
		return ai.CategoryApplication
	case strings.Contains(page, "traffic"):
		return ai.CategoryTraffic
	case strings.Contains(page, "approval"):
		return ai.CategoryApproval
	case strings.Contains(page, "k8s") || strings.Contains(page, "cluster"):
		return ai.CategoryK8s
	case strings.Contains(page, "alert") || strings.Contains(page, "monitor"):
		return ai.CategoryMonitoring
	case strings.Contains(page, "pipeline") || strings.Contains(page, "cicd"):
		return ai.CategoryCICD
	default:
		return ai.CategoryGeneral
	}
}
