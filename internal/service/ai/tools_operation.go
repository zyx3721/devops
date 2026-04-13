package ai

import (
	"context"
	"fmt"

	"gorm.io/gorm"
)

// ==================== 操作类工具 ====================

// RestartAppTool 重启应用工具
type RestartAppTool struct {
	db *gorm.DB
}

func (t *RestartAppTool) Name() string { return "restart_app" }

func (t *RestartAppTool) Description() string {
	return "重启应用的所有Pod，用于解决应用异常或更新配置"
}

func (t *RestartAppTool) Parameters() map[string]interface{} {
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
			"env": map[string]interface{}{
				"type":        "string",
				"description": "环境: dev, test, staging, prod",
			},
			"reason": map[string]interface{}{
				"type":        "string",
				"description": "重启原因",
			},
		},
		"required": []string{"app_name", "env"},
	}
}

func (t *RestartAppTool) Execute(ctx context.Context, userID uint, params map[string]interface{}) (interface{}, error) {
	appName, _ := params["app_name"].(string)
	env, _ := params["env"].(string)
	reason, _ := params["reason"].(string)

	if appName == "" || env == "" {
		return nil, fmt.Errorf("app_name and env are required")
	}

	// 这里应该调用实际的K8s服务执行重启
	// 目前返回模拟结果
	result := map[string]interface{}{
		"app_name":   appName,
		"env":        env,
		"reason":     reason,
		"status":     "success",
		"message":    fmt.Sprintf("应用 %s 在 %s 环境的重启操作已提交", appName, env),
		"pods_count": 0,
	}

	return result, nil
}

func (t *RestartAppTool) RequiredPermissions() []string {
	return []string{"app:deploy", "k8s:exec"}
}

func (t *RestartAppTool) IsDangerous() bool { return true }

// ScalePodTool 扩缩容工具
type ScalePodTool struct {
	db *gorm.DB
}

func (t *ScalePodTool) Name() string { return "scale_pod" }

func (t *ScalePodTool) Description() string {
	return "调整应用的Pod副本数量，用于扩容或缩容"
}

func (t *ScalePodTool) Parameters() map[string]interface{} {
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
			"env": map[string]interface{}{
				"type":        "string",
				"description": "环境",
			},
			"replicas": map[string]interface{}{
				"type":        "integer",
				"description": "目标副本数",
				"minimum":     0,
				"maximum":     100,
			},
			"reason": map[string]interface{}{
				"type":        "string",
				"description": "扩缩容原因",
			},
		},
		"required": []string{"app_name", "replicas"},
	}
}

func (t *ScalePodTool) Execute(ctx context.Context, userID uint, params map[string]interface{}) (interface{}, error) {
	appName, _ := params["app_name"].(string)
	replicas, _ := params["replicas"].(float64)
	reason, _ := params["reason"].(string)

	if appName == "" {
		return nil, fmt.Errorf("app_name is required")
	}

	replicasInt := int(replicas)
	if replicasInt < 0 || replicasInt > 100 {
		return nil, fmt.Errorf("replicas must be between 0 and 100")
	}

	// 这里应该调用实际的K8s服务执行扩缩容
	result := map[string]interface{}{
		"app_name":     appName,
		"replicas":     replicasInt,
		"reason":       reason,
		"status":       "success",
		"message":      fmt.Sprintf("应用 %s 的副本数已调整为 %d", appName, replicasInt),
		"old_replicas": 0,
	}

	return result, nil
}

func (t *ScalePodTool) RequiredPermissions() []string {
	return []string{"app:deploy", "k8s:update"}
}

func (t *ScalePodTool) IsDangerous() bool { return true }

// RollbackTool 回滚工具
type RollbackTool struct {
	db *gorm.DB
}

func (t *RollbackTool) Name() string { return "rollback" }

func (t *RollbackTool) Description() string {
	return "回滚应用到指定版本，用于快速恢复故障"
}

func (t *RollbackTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"app_name": map[string]interface{}{
				"type":        "string",
				"description": "应用名称",
			},
			"env": map[string]interface{}{
				"type":        "string",
				"description": "环境",
			},
			"version": map[string]interface{}{
				"type":        "string",
				"description": "目标版本号，不填则回滚到上一个版本",
			},
			"deploy_record_id": map[string]interface{}{
				"type":        "integer",
				"description": "部署记录ID，用于回滚到指定部署",
			},
			"reason": map[string]interface{}{
				"type":        "string",
				"description": "回滚原因",
			},
		},
		"required": []string{"app_name", "env"},
	}
}

func (t *RollbackTool) Execute(ctx context.Context, userID uint, params map[string]interface{}) (interface{}, error) {
	appName, _ := params["app_name"].(string)
	env, _ := params["env"].(string)
	version, _ := params["version"].(string)
	reason, _ := params["reason"].(string)

	if appName == "" || env == "" {
		return nil, fmt.Errorf("app_name and env are required")
	}

	// 这里应该调用实际的部署服务执行回滚
	result := map[string]interface{}{
		"app_name":       appName,
		"env":            env,
		"target_version": version,
		"reason":         reason,
		"status":         "success",
		"message":        fmt.Sprintf("应用 %s 在 %s 环境的回滚操作已提交", appName, env),
	}

	return result, nil
}

func (t *RollbackTool) RequiredPermissions() []string {
	return []string{"app:deploy", "deploy:rollback"}
}

func (t *RollbackTool) IsDangerous() bool { return true }

// SilenceAlertTool 静默告警工具
type SilenceAlertTool struct {
	db *gorm.DB
}

func (t *SilenceAlertTool) Name() string { return "silence_alert" }

func (t *SilenceAlertTool) Description() string {
	return "静默指定告警，在指定时间内不再发送通知"
}

func (t *SilenceAlertTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"alert_id": map[string]interface{}{
				"type":        "integer",
				"description": "告警ID",
			},
			"alert_name": map[string]interface{}{
				"type":        "string",
				"description": "告警名称（与alert_id二选一）",
			},
			"duration": map[string]interface{}{
				"type":        "string",
				"description": "静默时长: 1h, 2h, 4h, 8h, 24h",
				"enum":        []string{"1h", "2h", "4h", "8h", "24h"},
			},
			"reason": map[string]interface{}{
				"type":        "string",
				"description": "静默原因",
			},
		},
		"required": []string{"duration", "reason"},
	}
}

func (t *SilenceAlertTool) Execute(ctx context.Context, userID uint, params map[string]interface{}) (interface{}, error) {
	alertID, _ := params["alert_id"].(float64)
	alertName, _ := params["alert_name"].(string)
	duration, _ := params["duration"].(string)
	reason, _ := params["reason"].(string)

	if alertID == 0 && alertName == "" {
		return nil, fmt.Errorf("alert_id or alert_name is required")
	}

	if duration == "" {
		duration = "1h"
	}

	// 这里应该调用实际的告警服务执行静默
	result := map[string]interface{}{
		"alert_id":   int(alertID),
		"alert_name": alertName,
		"duration":   duration,
		"reason":     reason,
		"status":     "success",
		"message":    fmt.Sprintf("告警已静默 %s", duration),
	}

	return result, nil
}

func (t *SilenceAlertTool) RequiredPermissions() []string {
	return []string{"alert:update", "alert:silence"}
}

func (t *SilenceAlertTool) IsDangerous() bool { return true }
