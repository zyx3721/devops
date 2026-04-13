package dto

import "time"

// =====================================================
// 日志流请求/响应
// =====================================================

// LogStreamRequest WebSocket 日志流请求
type LogStreamRequest struct {
	ClusterID int64    `json:"cluster_id" binding:"required"`
	Namespace string   `json:"namespace" binding:"required"`
	PodName   string   `json:"pod_name"`
	PodNames  []string `json:"pod_names"`
	Container string   `json:"container"`
	Follow    bool     `json:"follow"`
	TailLines int64    `json:"tail_lines"`
	SinceTime string   `json:"since_time"`
	Keyword   string   `json:"keyword"`
	Level     string   `json:"level"`
	UseRegex  bool     `json:"use_regex"`
}

// LogStreamMessage 日志流消息
type LogStreamMessage struct {
	Type      string `json:"type"` // log/error/connected/disconnected
	Timestamp string `json:"timestamp"`
	Content   string `json:"content"`
	PodName   string `json:"pod_name"`
	Container string `json:"container"`
	Level     string `json:"level"`
}

// =====================================================
// 日志查询请求/响应
// =====================================================

// LogQueryRequest 日志查询请求
type LogQueryRequest struct {
	ClusterID  int64    `form:"cluster_id" json:"cluster_id" binding:"required"`
	Namespace  string   `form:"namespace" json:"namespace" binding:"required"`
	PodNames   []string `form:"pod_names" json:"pod_names"`
	Containers []string `form:"containers" json:"containers"`
	Keyword    string   `form:"keyword" json:"keyword"`
	Regex      string   `form:"regex" json:"regex"`
	Level      string   `form:"level" json:"level"`
	StartTime  string   `form:"start_time" json:"start_time"`
	EndTime    string   `form:"end_time" json:"end_time"`
	Page       int      `form:"page" json:"page"`
	PageSize   int      `form:"page_size" json:"page_size"`
	Order      string   `form:"order" json:"order"` // asc/desc
}

// LogEntry 日志条目
type LogEntry struct {
	ID        string                 `json:"id"`
	Timestamp string                 `json:"timestamp"`
	Content   string                 `json:"content"`
	PodName   string                 `json:"pod_name"`
	Container string                 `json:"container"`
	Level     string                 `json:"level"`
	Parsed    map[string]interface{} `json:"parsed,omitempty"`
}

// LogQueryResponse 日志查询响应
type LogQueryResponse struct {
	Total   int64      `json:"total"`
	Items   []LogEntry `json:"items"`
	HasMore bool       `json:"has_more"`
}

// =====================================================
// 日志上下文请求/响应
// =====================================================

// LogContextRequest 日志上下文请求
type LogContextRequest struct {
	ClusterID   int64  `form:"cluster_id" json:"cluster_id" binding:"required"`
	Namespace   string `form:"namespace" json:"namespace" binding:"required"`
	PodName     string `form:"pod_name" json:"pod_name" binding:"required"`
	Container   string `form:"container" json:"container"`
	Timestamp   string `form:"timestamp" json:"timestamp" binding:"required"`
	LinesBefore int    `form:"lines_before" json:"lines_before"`
	LinesAfter  int    `form:"lines_after" json:"lines_after"`
}

// LogContextResponse 日志上下文响应
type LogContextResponse struct {
	Before      []LogEntry `json:"before"`
	Current     LogEntry   `json:"current"`
	After       []LogEntry `json:"after"`
	TotalBefore int        `json:"total_before"`
	TotalAfter  int        `json:"total_after"`
}

// =====================================================
// 日志导出请求/响应
// =====================================================

// LogExportRequest 日志导出请求
type LogExportRequest struct {
	ClusterID int64    `json:"cluster_id" binding:"required"`
	Namespace string   `json:"namespace" binding:"required"`
	PodNames  []string `json:"pod_names"`
	StartTime string   `json:"start_time"`
	EndTime   string   `json:"end_time"`
	Format    string   `json:"format" binding:"required,oneof=txt json csv"` // txt/json/csv
	Keyword   string   `json:"keyword"`
	Level     string   `json:"level"`
}

// LogExportResponse 日志导出响应
type LogExportResponse struct {
	TaskID   string `json:"task_id"`
	Status   string `json:"status"` // pending/processing/completed/failed
	Progress int    `json:"progress"`
	URL      string `json:"url,omitempty"`
	Error    string `json:"error,omitempty"`
}

// =====================================================
// 告警规则请求/响应
// =====================================================

// LogAlertRuleRequest 告警规则请求
type LogAlertRuleRequest struct {
	Name         string   `json:"name" binding:"required,max=100"`
	ClusterID    int64    `json:"cluster_id" binding:"required"`
	Namespace    string   `json:"namespace"`
	MatchType    string   `json:"match_type" binding:"required,oneof=keyword regex level"`
	MatchValue   string   `json:"match_value" binding:"required,max=500"`
	Level        string   `json:"level" binding:"oneof=info warning error critical"`
	Channels     []string `json:"channels"`
	Enabled      bool     `json:"enabled"`
	AggregateMin int      `json:"aggregate_min"`
}

// LogAlertRuleResponse 告警规则响应
type LogAlertRuleResponse struct {
	ID           int64     `json:"id"`
	Name         string    `json:"name"`
	ClusterID    int64     `json:"cluster_id"`
	ClusterName  string    `json:"cluster_name,omitempty"`
	Namespace    string    `json:"namespace"`
	MatchType    string    `json:"match_type"`
	MatchValue   string    `json:"match_value"`
	Level        string    `json:"level"`
	Channels     []string  `json:"channels"`
	Enabled      bool      `json:"enabled"`
	AggregateMin int       `json:"aggregate_min"`
	CreatedBy    int64     `json:"created_by"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// LogAlertHistoryResponse 告警历史响应
type LogAlertHistoryResponse struct {
	ID             int64      `json:"id"`
	RuleID         int64      `json:"rule_id"`
	RuleName       string     `json:"rule_name"`
	ClusterID      int64      `json:"cluster_id"`
	Namespace      string     `json:"namespace"`
	PodName        string     `json:"pod_name"`
	Container      string     `json:"container"`
	MatchedContent string     `json:"matched_content"`
	AlertCount     int        `json:"alert_count"`
	Status         string     `json:"status"`
	SentAt         *time.Time `json:"sent_at"`
	ErrorMsg       string     `json:"error_msg,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
}

// =====================================================
// 染色规则请求/响应
// =====================================================

// HighlightRuleRequest 染色规则请求
type HighlightRuleRequest struct {
	Name       string `json:"name" binding:"required,max=100"`
	MatchType  string `json:"match_type" binding:"required,oneof=keyword regex level"`
	MatchValue string `json:"match_value" binding:"required,max=500"`
	FgColor    string `json:"fg_color"`
	BgColor    string `json:"bg_color"`
	Priority   int    `json:"priority"`
	Enabled    bool   `json:"enabled"`
}

// HighlightRuleResponse 染色规则响应
type HighlightRuleResponse struct {
	ID         int64     `json:"id"`
	UserID     int64     `json:"user_id"`
	Name       string    `json:"name"`
	MatchType  string    `json:"match_type"`
	MatchValue string    `json:"match_value"`
	FgColor    string    `json:"fg_color"`
	BgColor    string    `json:"bg_color"`
	Priority   int       `json:"priority"`
	Enabled    bool      `json:"enabled"`
	IsPreset   bool      `json:"is_preset"`
	CreatedAt  time.Time `json:"created_at"`
}

// =====================================================
// 解析模板请求/响应
// =====================================================

// ParseFieldDTO 解析字段
type ParseFieldDTO struct {
	Name     string `json:"name" binding:"required"`
	Type     string `json:"type" binding:"required,oneof=string int float timestamp"`
	JSONPath string `json:"json_path,omitempty"`
}

// ParseTemplateRequest 解析模板请求
type ParseTemplateRequest struct {
	Name        string          `json:"name" binding:"required,max=100"`
	Description string          `json:"description"`
	Type        string          `json:"type" binding:"required,oneof=json regex grok"`
	Pattern     string          `json:"pattern"`
	Fields      []ParseFieldDTO `json:"fields"`
	Enabled     bool            `json:"enabled"`
}

// ParseTemplateResponse 解析模板响应
type ParseTemplateResponse struct {
	ID          int64           `json:"id"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Type        string          `json:"type"`
	Pattern     string          `json:"pattern"`
	Fields      []ParseFieldDTO `json:"fields"`
	IsPreset    bool            `json:"is_preset"`
	Enabled     bool            `json:"enabled"`
	CreatedBy   int64           `json:"created_by"`
	CreatedAt   time.Time       `json:"created_at"`
}

// ParseTestRequest 解析测试请求
type ParseTestRequest struct {
	TemplateID int64  `json:"template_id"`
	Type       string `json:"type" binding:"required,oneof=json regex grok"`
	Pattern    string `json:"pattern"`
	LogContent string `json:"log_content" binding:"required"`
}

// ParseTestResponse 解析测试响应
type ParseTestResponse struct {
	Success bool                   `json:"success"`
	Parsed  map[string]interface{} `json:"parsed,omitempty"`
	Error   string                 `json:"error,omitempty"`
}

// =====================================================
// 数据源请求/响应
// =====================================================

// DataSourceRequest 数据源请求
type DataSourceRequest struct {
	Name      string                 `json:"name" binding:"required,max=100"`
	Type      string                 `json:"type" binding:"required,oneof=k8s loki elasticsearch"`
	URL       string                 `json:"url"`
	Username  string                 `json:"username"`
	Password  string                 `json:"password"`
	Config    map[string]interface{} `json:"config"`
	ClusterID int64                  `json:"cluster_id"`
	Namespace string                 `json:"namespace"`
	IsDefault bool                   `json:"is_default"`
	Enabled   bool                   `json:"enabled"`
}

// DataSourceResponse 数据源响应
type DataSourceResponse struct {
	ID              int64                  `json:"id"`
	Name            string                 `json:"name"`
	Type            string                 `json:"type"`
	URL             string                 `json:"url"`
	Username        string                 `json:"username,omitempty"`
	Config          map[string]interface{} `json:"config,omitempty"`
	ClusterID       int64                  `json:"cluster_id"`
	ClusterName     string                 `json:"cluster_name,omitempty"`
	Namespace       string                 `json:"namespace"`
	IsDefault       bool                   `json:"is_default"`
	Enabled         bool                   `json:"enabled"`
	LastCheckAt     *time.Time             `json:"last_check_at"`
	LastCheckStatus string                 `json:"last_check_status"`
	CreatedAt       time.Time              `json:"created_at"`
}

// DataSourceTestResponse 数据源测试响应
type DataSourceTestResponse struct {
	Success bool   `json:"success"`
	Latency int64  `json:"latency"` // 毫秒
	Version string `json:"version,omitempty"`
	Error   string `json:"error,omitempty"`
}

// =====================================================
// 统计分析请求/响应
// =====================================================

// LogStatsRequest 日志统计请求
type LogStatsRequest struct {
	ClusterID int64  `form:"cluster_id" json:"cluster_id" binding:"required"`
	Namespace string `form:"namespace" json:"namespace"`
	PodName   string `form:"pod_name" json:"pod_name"`
	StartTime string `form:"start_time" json:"start_time"`
	EndTime   string `form:"end_time" json:"end_time"`
	Interval  string `form:"interval" json:"interval"` // hour/day
}

// TrendPoint 趋势数据点
type TrendPoint struct {
	Time  string `json:"time"`
	Count int64  `json:"count"`
	Level string `json:"level,omitempty"`
}

// ErrorStat 错误统计
type ErrorStat struct {
	Pattern string `json:"pattern"`
	Count   int64  `json:"count"`
	Sample  string `json:"sample"`
}

// LogStatsResponse 日志统计响应
type LogStatsResponse struct {
	TotalCount  int64            `json:"total_count"`
	LevelCounts map[string]int64 `json:"level_counts"`
	Trend       []TrendPoint     `json:"trend"`
	TopErrors   []ErrorStat      `json:"top_errors"`
}

// =====================================================
// 收藏书签请求/响应
// =====================================================

// BookmarkRequest 书签请求
type BookmarkRequest struct {
	ClusterID    int64  `json:"cluster_id" binding:"required"`
	Namespace    string `json:"namespace"`
	PodName      string `json:"pod_name"`
	Container    string `json:"container"`
	LogTimestamp string `json:"log_timestamp"`
	Content      string `json:"content" binding:"required"`
	Note         string `json:"note"`
}

// BookmarkResponse 书签响应
type BookmarkResponse struct {
	ID             int64      `json:"id"`
	UserID         int64      `json:"user_id"`
	ClusterID      int64      `json:"cluster_id"`
	ClusterName    string     `json:"cluster_name,omitempty"`
	Namespace      string     `json:"namespace"`
	PodName        string     `json:"pod_name"`
	Container      string     `json:"container"`
	LogTimestamp   *time.Time `json:"log_timestamp"`
	Content        string     `json:"content"`
	Note           string     `json:"note"`
	ShareURL       string     `json:"share_url,omitempty"`
	ShareExpiresAt *time.Time `json:"share_expires_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
}

// BookmarkShareRequest 书签分享请求
type BookmarkShareRequest struct {
	ExpiresInDays int `json:"expires_in_days"` // 过期天数，0表示永不过期
}

// =====================================================
// 快捷查询请求/响应
// =====================================================

// SavedQueryRequest 快捷查询请求
type SavedQueryRequest struct {
	Name        string                 `json:"name" binding:"required,max=100"`
	Description string                 `json:"description"`
	QueryParams map[string]interface{} `json:"query_params" binding:"required"`
	IsShared    bool                   `json:"is_shared"`
}

// SavedQueryResponse 快捷查询响应
type SavedQueryResponse struct {
	ID          int64                  `json:"id"`
	UserID      int64                  `json:"user_id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	QueryParams map[string]interface{} `json:"query_params"`
	IsShared    bool                   `json:"is_shared"`
	UseCount    int                    `json:"use_count"`
	LastUsedAt  *time.Time             `json:"last_used_at"`
	CreatedAt   time.Time              `json:"created_at"`
}

// =====================================================
// 多 Pod 日志流请求/响应
// =====================================================

// PodLogRequest 单个 Pod 日志请求
type PodLogRequest struct {
	PodName   string `json:"pod_name" binding:"required"`
	Container string `json:"container"`
}

// MultiPodLogStreamRequest 多 Pod 日志流请求
type MultiPodLogStreamRequest struct {
	ClusterID int64           `json:"cluster_id" binding:"required"`
	Namespace string          `json:"namespace" binding:"required"`
	Pods      []PodLogRequest `json:"pods" binding:"required,min=1,max=10"`
	TailLines int64           `json:"tail_lines"`
	Follow    bool            `json:"follow"`
	Keyword   string          `json:"keyword"`
	Level     string          `json:"level"`
	UseRegex  bool            `json:"use_regex"`
}

// =====================================================
// 日志对比请求/响应
// =====================================================

// LogCompareRequest 日志对比请求
type LogCompareRequest struct {
	ClusterID int64  `json:"cluster_id" binding:"required"`
	Namespace string `json:"namespace" binding:"required"`
	// 对比类型: time_range / pod
	CompareType string `json:"compare_type" binding:"required,oneof=time_range pod"`
	// 时间段对比
	LeftStartTime  string `json:"left_start_time"`
	LeftEndTime    string `json:"left_end_time"`
	RightStartTime string `json:"right_start_time"`
	RightEndTime   string `json:"right_end_time"`
	// Pod 对比
	LeftPodName  string `json:"left_pod_name"`
	RightPodName string `json:"right_pod_name"`
	// 通用参数
	Container string `json:"container"`
	Keyword   string `json:"keyword"`
	Level     string `json:"level"`
}

// LogCompareLine 对比行
type LogCompareLine struct {
	LineNumber int    `json:"line_number"`
	Timestamp  string `json:"timestamp"`
	Content    string `json:"content"`
	Level      string `json:"level"`
	DiffType   string `json:"diff_type"` // same/added/removed/modified
	PodName    string `json:"pod_name,omitempty"`
}

// LogCompareResponse 日志对比响应
type LogCompareResponse struct {
	LeftLines    []LogCompareLine `json:"left_lines"`
	RightLines   []LogCompareLine `json:"right_lines"`
	TotalLeft    int              `json:"total_left"`
	TotalRight   int              `json:"total_right"`
	AddedCount   int              `json:"added_count"`
	RemovedCount int              `json:"removed_count"`
	SameCount    int              `json:"same_count"`
}
