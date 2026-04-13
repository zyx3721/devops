package dto

// ==================== 部署前置检查 DTO ====================

// DeployPreCheckRequest 部署前置检查请求
type DeployPreCheckRequest struct {
	ApplicationID uint   `json:"application_id" binding:"required"`
	EnvName       string `json:"env_name" binding:"required"`
	ImageTag      string `json:"image_tag"`
}

// DeployPreCheckResponse 部署前置检查响应
type DeployPreCheckResponse struct {
	CanDeploy bool           `json:"can_deploy"`
	Checks    []PreCheckItem `json:"checks"`
	Warnings  []string       `json:"warnings"`
	Errors    []string       `json:"errors"`
}

// PreCheckItem 检查项
type PreCheckItem struct {
	Name    string `json:"name"`
	Status  string `json:"status"` // passed, warning, failed, skipped
	Message string `json:"message"`
	Detail  string `json:"detail,omitempty"`
}

// ==================== 灰度发布 DTO ====================

// CanaryDeployRequest 灰度发布请求
type CanaryDeployRequest struct {
	ApplicationID     uint   `json:"application_id" binding:"required"`
	EnvName           string `json:"env_name" binding:"required"`
	ImageTag          string `json:"image_tag" binding:"required"`
	CanaryPercent     int    `json:"canary_percent" binding:"required,min=1,max=100"` // 灰度流量比例
	CanaryReplicas    int    `json:"canary_replicas"`                                 // 灰度副本数
	CanaryHeader      string `json:"canary_header,omitempty"`                         // 基于Header灰度 (如: X-Canary)
	CanaryHeaderValue string `json:"canary_header_value,omitempty"`                   // Header值 (如: always)
	CanaryCookie      string `json:"canary_cookie,omitempty"`                         // 基于Cookie灰度 (如: canary=always)
	Description       string `json:"description"`
}

// CanaryStatus 灰度状态
type CanaryStatus struct {
	DeployRecordID uint   `json:"deploy_record_id"`
	Status         string `json:"status"` // canary_running, canary_paused, promoting, completed, rolled_back
	CanaryReplicas int32  `json:"canary_replicas"`
	StableReplicas int32  `json:"stable_replicas"`
	CanaryReady    int32  `json:"canary_ready"`
	StableReady    int32  `json:"stable_ready"`
	CanaryImage    string `json:"canary_image"`
	StableImage    string `json:"stable_image"`
	StartedAt      string `json:"started_at"`
	CanaryHealthy  bool   `json:"canary_healthy"`
	ErrorRate      string `json:"error_rate,omitempty"`
	TrafficPercent int    `json:"traffic_percent"`         // 流量百分比
	CanaryHeader   string `json:"canary_header,omitempty"` // Header灰度配置
	CanaryCookie   string `json:"canary_cookie,omitempty"` // Cookie灰度配置
}

// CanaryPromoteRequest 灰度全量发布请求
type CanaryPromoteRequest struct {
	DeployRecordID uint `json:"deploy_record_id" binding:"required"`
}

// CanaryRollbackRequest 灰度回滚请求
type CanaryRollbackRequest struct {
	DeployRecordID uint   `json:"deploy_record_id" binding:"required"`
	Reason         string `json:"reason"`
}

// CanaryListItem 灰度列表项
type CanaryListItem struct {
	ID            uint   `json:"id"`
	CreatedAt     string `json:"created_at"`
	ApplicationID uint   `json:"application_id"`
	AppName       string `json:"app_name"`
	EnvName       string `json:"env_name"`
	ImageTag      string `json:"image_tag"`
	CanaryPercent int    `json:"canary_percent"`
	Status        string `json:"status"`
	Operator      string `json:"operator"`
	FinishedAt    string `json:"finished_at,omitempty"`
}

// CanaryAdjustRequest 调整灰度比例请求
type CanaryAdjustRequest struct {
	Percent int `json:"percent" binding:"required,min=1,max=100"`
}

// ==================== 自动回滚 DTO ====================

// AutoRollbackConfig 自动回滚配置
type AutoRollbackConfig struct {
	Enabled           bool `json:"enabled"`
	HealthCheckPeriod int  `json:"health_check_period"` // 健康检查周期（秒）
	FailureThreshold  int  `json:"failure_threshold"`   // 失败阈值
	SuccessThreshold  int  `json:"success_threshold"`   // 成功阈值（连续成功次数）
}

// RollbackTrigger 回滚触发条件
type RollbackTrigger struct {
	Type      string `json:"type"`      // pod_crash, health_check_fail, error_rate_high
	Threshold string `json:"threshold"` // 阈值
	Duration  int    `json:"duration"`  // 持续时间（秒）
}

// DeployHealthStatus 部署健康状态
type DeployHealthStatus struct {
	DeployRecordID   uint   `json:"deploy_record_id"`
	Status           string `json:"status"` // healthy, degraded, unhealthy
	ReadyReplicas    int32  `json:"ready_replicas"`
	DesiredReplicas  int32  `json:"desired_replicas"`
	UnavailableCount int32  `json:"unavailable_count"`
	RestartCount     int32  `json:"restart_count"`
	LastCheckTime    string `json:"last_check_time"`
	ConsecutiveFails int    `json:"consecutive_fails"`
	ShouldRollback   bool   `json:"should_rollback"`
	RollbackReason   string `json:"rollback_reason,omitempty"`
}
