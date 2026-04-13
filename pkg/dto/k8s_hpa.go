package dto

// ==================== HPA 相关 DTO ====================

// K8sHPA HPA 信息
type K8sHPA struct {
	Name            string   `json:"name"`
	Namespace       string   `json:"namespace"`
	TargetKind      string   `json:"target_kind"` // Deployment/StatefulSet
	TargetName      string   `json:"target_name"`
	MinReplicas     int32    `json:"min_replicas"`
	MaxReplicas     int32    `json:"max_replicas"`
	CurrentReplicas int32    `json:"current_replicas"`
	DesiredReplicas int32    `json:"desired_replicas"`
	Metrics         []string `json:"metrics"` // CPU: 50%, Memory: 80%
	CreatedAt       string   `json:"created_at"`
}

// CreateHPARequest 创建 HPA 请求
type CreateHPARequest struct {
	Name             string `json:"name" binding:"required"`
	Namespace        string `json:"namespace" binding:"required"`
	TargetKind       string `json:"target_kind" binding:"required,oneof=Deployment StatefulSet"`
	TargetName       string `json:"target_name" binding:"required"`
	MinReplicas      int32  `json:"min_replicas" binding:"required,min=1"`
	MaxReplicas      int32  `json:"max_replicas" binding:"required,min=1"`
	CPUTargetPercent *int32 `json:"cpu_target_percent"` // CPU 目标使用率
	MemTargetPercent *int32 `json:"mem_target_percent"` // 内存目标使用率
}

// UpdateHPARequest 更新 HPA 请求
type UpdateHPARequest struct {
	MinReplicas      int32  `json:"min_replicas" binding:"required,min=1"`
	MaxReplicas      int32  `json:"max_replicas" binding:"required,min=1"`
	CPUTargetPercent *int32 `json:"cpu_target_percent"`
	MemTargetPercent *int32 `json:"mem_target_percent"`
}

// ==================== CronHPA 相关 DTO ====================

// K8sCronHPA CronHPA 信息
type K8sCronHPA struct {
	Name       string         `json:"name"`
	Namespace  string         `json:"namespace"`
	TargetKind string         `json:"target_kind"` // Deployment/StatefulSet
	TargetName string         `json:"target_name"`
	Enabled    bool           `json:"enabled"`
	Schedules  []CronSchedule `json:"schedules"`
	CreatedAt  string         `json:"created_at"`
}

// CronSchedule 定时调度规则
type CronSchedule struct {
	Name        string `json:"name"`         // 规则名称，如 "工作时间"
	Cron        string `json:"cron"`         // cron 表达式
	MinReplicas int32  `json:"min_replicas"` // HPA最小副本数
	MaxReplicas int32  `json:"max_replicas"` // HPA最大副本数
}

// CreateCronHPARequest 创建 CronHPA 请求
type CreateCronHPARequest struct {
	Name       string         `json:"name" binding:"required"`
	Namespace  string         `json:"namespace" binding:"required"`
	TargetKind string         `json:"target_kind" binding:"required,oneof=Deployment StatefulSet"`
	TargetName string         `json:"target_name" binding:"required"`
	Enabled    bool           `json:"enabled"`
	Schedules  []CronSchedule `json:"schedules" binding:"required,min=1,dive"`
}

// UpdateCronHPARequest 更新 CronHPA 请求
type UpdateCronHPARequest struct {
	Enabled   bool           `json:"enabled"`
	Schedules []CronSchedule `json:"schedules" binding:"required,min=1,dive"`
}

// ==================== 资源配额相关 DTO ====================

// K8sResourceQuota 资源配额信息
type K8sResourceQuota struct {
	Name      string            `json:"name"`
	Namespace string            `json:"namespace"`
	Hard      map[string]string `json:"hard"` // 配额限制
	Used      map[string]string `json:"used"` // 已使用
	CreatedAt string            `json:"created_at"`
}

// CreateResourceQuotaRequest 创建资源配额请求
type CreateResourceQuotaRequest struct {
	Name      string            `json:"name" binding:"required"`
	Namespace string            `json:"namespace" binding:"required"`
	Hard      map[string]string `json:"hard" binding:"required"` // cpu, memory, pods, services 等
}

// K8sLimitRange LimitRange 信息
type K8sLimitRange struct {
	Name      string           `json:"name"`
	Namespace string           `json:"namespace"`
	Limits    []LimitRangeItem `json:"limits"`
	CreatedAt string           `json:"created_at"`
}

// LimitRangeItem 限制项
type LimitRangeItem struct {
	Type           string            `json:"type"` // Container, Pod, PersistentVolumeClaim
	Default        map[string]string `json:"default,omitempty"`
	DefaultRequest map[string]string `json:"default_request,omitempty"`
	Max            map[string]string `json:"max,omitempty"`
	Min            map[string]string `json:"min,omitempty"`
}

// ==================== 集群概览 DTO ====================

// ClusterOverview 集群概览
type ClusterOverview struct {
	ClusterID   uint   `json:"cluster_id"`
	ClusterName string `json:"cluster_name"`
	Status      string `json:"status"`
	// 节点统计
	NodeTotal int `json:"node_total"`
	NodeReady int `json:"node_ready"`
	// 资源统计
	CPUCapacity    string `json:"cpu_capacity"`
	CPUUsed        string `json:"cpu_used"`
	MemoryCapacity string `json:"memory_capacity"`
	MemoryUsed     string `json:"memory_used"`
	PodCapacity    int    `json:"pod_capacity"`
	PodUsed        int    `json:"pod_used"`
	// 工作负载统计
	DeploymentTotal  int `json:"deployment_total"`
	DeploymentReady  int `json:"deployment_ready"`
	StatefulSetTotal int `json:"statefulset_total"`
	StatefulSetReady int `json:"statefulset_ready"`
	DaemonSetTotal   int `json:"daemonset_total"`
	DaemonSetReady   int `json:"daemonset_ready"`
}

// MultiClusterOverview 多集群概览
type MultiClusterOverview struct {
	Clusters []ClusterOverview `json:"clusters"`
	Summary  ClusterSummary    `json:"summary"`
}

// ClusterSummary 集群汇总
type ClusterSummary struct {
	TotalClusters    int `json:"total_clusters"`
	HealthyClusters  int `json:"healthy_clusters"`
	TotalNodes       int `json:"total_nodes"`
	TotalPods        int `json:"total_pods"`
	TotalDeployments int `json:"total_deployments"`
}
