package monitoring

import "time"

// AlertEvent 标准化告警事件
// 所有外部来源的告警（Prometheus, Grafana, AWS等）都将转换为此结构
type AlertEvent struct {
	Fingerprint string            `json:"fingerprint"` // 告警指纹，用于去重
	Title       string            `json:"title"`       // 告警标题
	Content     string            `json:"content"`     // 告警详情
	Level       string            `json:"level"`       // 告警级别: info, warning, error, critical
	Status      string            `json:"status"`      // 状态: firing, resolved
	Source      string            `json:"source"`      // 来源: prometheus, grafana, etc.
	SourceID    string            `json:"source_id"`   // 来源系统中的ID
	SourceURL   string            `json:"source_url"`  // 查看详情的链接
	Labels      map[string]string `json:"labels"`      // 标签，用于匹配路由和静默规则
	StartsAt    time.Time         `json:"starts_at"`   // 开始时间
	EndsAt      *time.Time        `json:"ends_at"`     // 结束时间
	RawData     interface{}       `json:"raw_data"`    // 原始数据 payload
}
