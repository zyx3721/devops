package ai

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	"devops/internal/repository"
)

// ==================== 查询类工具 ====================

// QueryLogsTool 查询日志工具
type QueryLogsTool struct {
	db *gorm.DB
}

func (t *QueryLogsTool) Name() string { return "query_logs" }

func (t *QueryLogsTool) Description() string {
	return "查询应用日志，支持按应用名、时间范围、关键词过滤"
}

func (t *QueryLogsTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"app_name": map[string]interface{}{
				"type":        "string",
				"description": "应用名称",
			},
			"namespace": map[string]interface{}{
				"type":        "string",
				"description": "K8s命名空间",
			},
			"keyword": map[string]interface{}{
				"type":        "string",
				"description": "搜索关键词",
			},
			"level": map[string]interface{}{
				"type":        "string",
				"description": "日志级别: error, warn, info",
				"enum":        []string{"error", "warn", "info"},
			},
			"start_time": map[string]interface{}{
				"type":        "string",
				"description": "开始时间，格式: 2006-01-02 15:04:05",
			},
			"end_time": map[string]interface{}{
				"type":        "string",
				"description": "结束时间，格式: 2006-01-02 15:04:05",
			},
			"limit": map[string]interface{}{
				"type":        "integer",
				"description": "返回条数限制，默认100",
			},
		},
		"required": []string{"app_name"},
	}
}

func (t *QueryLogsTool) Execute(ctx context.Context, userID uint, params map[string]interface{}) (interface{}, error) {
	appName, _ := params["app_name"].(string)
	if appName == "" {
		return nil, fmt.Errorf("app_name is required")
	}

	// 这里应该调用实际的日志查询服务
	// 目前返回模拟数据
	result := map[string]interface{}{
		"app_name":    appName,
		"total_count": 0,
		"logs":        []interface{}{},
		"message":     fmt.Sprintf("查询应用 %s 的日志", appName),
	}

	return result, nil
}

func (t *QueryLogsTool) RequiredPermissions() []string {
	return []string{"app:view", "logs:view"}
}

func (t *QueryLogsTool) IsDangerous() bool { return false }

// QueryAlertsTool 查询告警工具
type QueryAlertsTool struct {
	db *gorm.DB
}

func (t *QueryAlertsTool) Name() string { return "query_alerts" }

func (t *QueryAlertsTool) Description() string {
	return "查询告警信息，支持按状态、级别、时间范围过滤"
}

func (t *QueryAlertsTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"status": map[string]interface{}{
				"type":        "string",
				"description": "告警状态: firing, resolved",
				"enum":        []string{"firing", "resolved", "all"},
			},
			"severity": map[string]interface{}{
				"type":        "string",
				"description": "告警级别: critical, warning, info",
				"enum":        []string{"critical", "warning", "info"},
			},
			"app_name": map[string]interface{}{
				"type":        "string",
				"description": "应用名称",
			},
			"start_time": map[string]interface{}{
				"type":        "string",
				"description": "开始时间",
			},
			"limit": map[string]interface{}{
				"type":        "integer",
				"description": "返回条数限制",
			},
		},
	}
}

func (t *QueryAlertsTool) Execute(ctx context.Context, userID uint, params map[string]interface{}) (interface{}, error) {
	status, _ := params["status"].(string)
	if status == "" {
		status = "firing"
	}

	// 这里应该调用实际的告警查询服务
	result := map[string]interface{}{
		"status":      status,
		"total_count": 0,
		"alerts":      []interface{}{},
		"message":     fmt.Sprintf("查询状态为 %s 的告警", status),
	}

	return result, nil
}

func (t *QueryAlertsTool) RequiredPermissions() []string {
	return []string{"alert:view"}
}

func (t *QueryAlertsTool) IsDangerous() bool { return false }

// QueryMetricsTool 查询监控指标工具
type QueryMetricsTool struct {
	db *gorm.DB
}

func (t *QueryMetricsTool) Name() string { return "query_metrics" }

func (t *QueryMetricsTool) Description() string {
	return "查询应用监控指标，如CPU、内存、请求量等"
}

func (t *QueryMetricsTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"app_name": map[string]interface{}{
				"type":        "string",
				"description": "应用名称",
			},
			"metric_type": map[string]interface{}{
				"type":        "string",
				"description": "指标类型: cpu, memory, requests, latency",
				"enum":        []string{"cpu", "memory", "requests", "latency", "errors"},
			},
			"time_range": map[string]interface{}{
				"type":        "string",
				"description": "时间范围: 1h, 6h, 24h, 7d",
				"enum":        []string{"1h", "6h", "24h", "7d"},
			},
		},
		"required": []string{"app_name", "metric_type"},
	}
}

func (t *QueryMetricsTool) Execute(ctx context.Context, userID uint, params map[string]interface{}) (interface{}, error) {
	appName, _ := params["app_name"].(string)
	metricType, _ := params["metric_type"].(string)

	if appName == "" || metricType == "" {
		return nil, fmt.Errorf("app_name and metric_type are required")
	}

	// 这里应该调用实际的监控服务
	result := map[string]interface{}{
		"app_name":    appName,
		"metric_type": metricType,
		"data_points": []interface{}{},
		"summary": map[string]interface{}{
			"current": 0,
			"avg":     0,
			"max":     0,
			"min":     0,
		},
		"message": fmt.Sprintf("查询应用 %s 的 %s 指标", appName, metricType),
	}

	return result, nil
}

func (t *QueryMetricsTool) RequiredPermissions() []string {
	return []string{"app:view", "metrics:view"}
}

func (t *QueryMetricsTool) IsDangerous() bool { return false }

// QueryKnowledgeTool 查询知识库工具
type QueryKnowledgeTool struct {
	db            *gorm.DB
	knowledgeRepo *repository.AIKnowledgeRepository
}

func NewQueryKnowledgeTool(db *gorm.DB) *QueryKnowledgeTool {
	return &QueryKnowledgeTool{
		db:            db,
		knowledgeRepo: repository.NewAIKnowledgeRepository(db),
	}
}

func (t *QueryKnowledgeTool) Name() string { return "query_knowledge" }

func (t *QueryKnowledgeTool) Description() string {
	return "查询系统使用文档和知识库，获取功能说明和操作指南"
}

func (t *QueryKnowledgeTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"query": map[string]interface{}{
				"type":        "string",
				"description": "搜索关键词",
			},
			"category": map[string]interface{}{
				"type":        "string",
				"description": "知识分类: application, traffic, approval, k8s, monitoring, cicd",
				"enum":        []string{"application", "traffic", "approval", "k8s", "monitoring", "cicd", "general"},
			},
		},
		"required": []string{"query"},
	}
}

func (t *QueryKnowledgeTool) Execute(ctx context.Context, userID uint, params map[string]interface{}) (interface{}, error) {
	query, _ := params["query"].(string)
	if query == "" {
		return nil, fmt.Errorf("query is required")
	}

	items, err := t.knowledgeRepo.Search(ctx, query, 5)
	if err != nil {
		return nil, fmt.Errorf("search knowledge: %w", err)
	}

	return map[string]interface{}{
		"query":   query,
		"results": items,
		"count":   len(items),
	}, nil
}

func (t *QueryKnowledgeTool) RequiredPermissions() []string {
	return []string{} // 知识库查询不需要特殊权限
}

func (t *QueryKnowledgeTool) IsDangerous() bool { return false }

// GetCurrentTime 获取当前时间的辅助函数
func GetCurrentTime() time.Time {
	return time.Now()
}
