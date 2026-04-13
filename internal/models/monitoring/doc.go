// Package monitoring 定义监控告警相关的数据模型
//
// 本包包含与系统监控和告警相关的所有数据模型，包括：
//   - 告警：告警规则、告警历史、静默规则
//   - 健康检查：检查配置、检查历史、状态记录
//   - 日志：日志查询、日志模板、书签
//   - 成本：资源成本、预算配置、成本分析
//
// 使用示例:
//
//	import "devops/internal/models/monitoring"
//
//	// 创建告警规则
//	rule := &monitoring.AlertRule{
//	    Name:      "CPU使用率告警",
//	    Threshold: 80,
//	}
package monitoring
