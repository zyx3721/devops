package dto

// CreateK8sClusterRequest 创建K8s集群请求
type CreateK8sClusterRequest struct {
	Name        string `json:"name" binding:"required,min=1,max=100"`
	Kubeconfig  string `json:"kubeconfig" binding:"required"`
	Description string `json:"description"`
	Status      string `json:"status" binding:"required,oneof=active inactive"`
	IsDefault   bool   `json:"is_default"`
}

// UpdateK8sClusterRequest 更新K8s集群请求
type UpdateK8sClusterRequest struct {
	Name        string `json:"name" binding:"required,min=1,max=100"`
	Kubeconfig  string `json:"kubeconfig" binding:"required"`
	Description string `json:"description"`
	Status      string `json:"status" binding:"required,oneof=active inactive"`
	IsDefault   bool   `json:"is_default"`
}

// K8sClusterResponse K8s集群响应
type K8sClusterResponse struct {
	ID          uint   `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Status      string `json:"status"`
	IsDefault   bool   `json:"is_default"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

// K8sClusterListRequest K8s集群列表请求
type K8sClusterListRequest struct {
	Page     int    `form:"page,default=1"`
	PageSize int    `form:"page_size,default=20"`
	Keyword  string `form:"keyword"`
	Status   string `form:"status"`
}

// K8sClusterListResponse K8s集群列表响应
type K8sClusterListResponse struct {
	Total      int64                `json:"total"`
	Page       int                  `json:"page"`
	PageSize   int                  `json:"page_size"`
	TotalPages int                  `json:"total_pages"`
	Items      []K8sClusterResponse `json:"items"`
}

// K8s 资源相关 DTO

// CreateNamespaceRequest 创建命名空间请求
type CreateNamespaceRequest struct {
	Name   string            `json:"name" binding:"required"`
	Labels map[string]string `json:"labels"`
}

// K8sNamespace 命名空间
type K8sNamespace struct {
	Name      string `json:"name"`
	Status    string `json:"status"`
	CreatedAt string `json:"created_at"`
}

// K8sDeployment Deployment
type K8sDeployment struct {
	Name      string   `json:"name"`
	Namespace string   `json:"namespace"`
	Replicas  int32    `json:"replicas"`
	Ready     int32    `json:"ready"`
	Available int32    `json:"available"`
	Images    []string `json:"images"`
	CreatedAt string   `json:"created_at"`
}

// K8sContainer 容器
type K8sContainer struct {
	Name  string `json:"name"`
	Image string `json:"image"`
}

// K8sPod Pod
type K8sPod struct {
	Name       string         `json:"name"`
	Namespace  string         `json:"namespace"`
	Status     string         `json:"status"`
	Node       string         `json:"node"`
	IP         string         `json:"ip"`
	Restarts   int32          `json:"restarts"`
	Containers []K8sContainer `json:"containers"`
	CreatedAt  string         `json:"created_at"`
}

// K8sServicePort Service端口
type K8sServicePort struct {
	Name       string `json:"name"`
	Port       int32  `json:"port"`
	TargetPort string `json:"target_port"`
	Protocol   string `json:"protocol"`
	NodePort   int32  `json:"node_port,omitempty"`
}

// K8sService Service
type K8sService struct {
	Name      string            `json:"name"`
	Namespace string            `json:"namespace"`
	Type      string            `json:"type"`
	ClusterIP string            `json:"cluster_ip"`
	Ports     []K8sServicePort  `json:"ports"`
	Selector  map[string]string `json:"selector"`
	CreatedAt string            `json:"created_at"`
}

// K8sConfigMap ConfigMap
type K8sConfigMap struct {
	Name      string   `json:"name"`
	Namespace string   `json:"namespace"`
	Keys      []string `json:"keys"`
	CreatedAt string   `json:"created_at"`
}

// K8sSecret Secret
type K8sSecret struct {
	Name      string   `json:"name"`
	Namespace string   `json:"namespace"`
	Type      string   `json:"type"`
	Keys      []string `json:"keys"`
	CreatedAt string   `json:"created_at"`
}

// ScaleDeploymentRequest 调整副本数请求
type ScaleDeploymentRequest struct {
	Replicas int32 `json:"replicas" binding:"required,min=0"`
}

// K8sStatefulSet StatefulSet
type K8sStatefulSet struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	Replicas  int32  `json:"replicas"`
	Ready     int32  `json:"ready"`
	CreatedAt string `json:"created_at"`
}

// K8sDaemonSet DaemonSet
type K8sDaemonSet struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	Desired   int32  `json:"desired"`
	Ready     int32  `json:"ready"`
	CreatedAt string `json:"created_at"`
}

// K8sJob Job
type K8sJob struct {
	Name        string `json:"name"`
	Namespace   string `json:"namespace"`
	Completions int32  `json:"completions"`
	Succeeded   int32  `json:"succeeded"`
	Failed      int32  `json:"failed"`
	CreatedAt   string `json:"created_at"`
}

// K8sCronJob CronJob
type K8sCronJob struct {
	Name         string `json:"name"`
	Namespace    string `json:"namespace"`
	Schedule     string `json:"schedule"`
	Suspend      bool   `json:"suspend"`
	LastSchedule string `json:"last_schedule"`
	CreatedAt    string `json:"created_at"`
}

// K8sIngress Ingress
type K8sIngress struct {
	Name      string        `json:"name"`
	Namespace string        `json:"namespace"`
	ClassName string        `json:"class_name"`
	Hosts     []string      `json:"hosts"`
	Rules     []IngressRule `json:"rules"`
	CreatedAt string        `json:"created_at"`
}

// IngressRule Ingress 规则
type IngressRule struct {
	Host  string        `json:"host"`
	Paths []IngressPath `json:"paths"`
}

// IngressPath Ingress 路径
type IngressPath struct {
	Path        string `json:"path"`
	PathType    string `json:"path_type"`
	ServiceName string `json:"service_name"`
	ServicePort int32  `json:"service_port"`
}

// K8sPVC PersistentVolumeClaim
type K8sPVC struct {
	Name         string `json:"name"`
	Namespace    string `json:"namespace"`
	Status       string `json:"status"`
	Capacity     string `json:"capacity"`
	StorageClass string `json:"storage_class"`
	AccessModes  string `json:"access_modes"`
	CreatedAt    string `json:"created_at"`
}

// ApplyResourceRequest 应用资源请求
type ApplyResourceRequest struct {
	YAML string `json:"yaml" binding:"required"`
}

// K8sNode 节点
type K8sNode struct {
	Name              string            `json:"name"`
	Status            string            `json:"status"`
	Roles             []string          `json:"roles"`
	InternalIP        string            `json:"internal_ip"`
	Hostname          string            `json:"hostname"`
	CPUCapacity       string            `json:"cpu_capacity"`
	MemoryCapacity    string            `json:"memory_capacity"`
	CPUAllocatable    string            `json:"cpu_allocatable"`
	MemoryAllocatable string            `json:"memory_allocatable"`
	PodCapacity       string            `json:"pod_capacity"`
	Schedulable       bool              `json:"schedulable"`
	Taints            []K8sNodeTaint    `json:"taints"`
	Labels            map[string]string `json:"labels"`
	KubeletVersion    string            `json:"kubelet_version"`
	ContainerRuntime  string            `json:"container_runtime"`
	OSImage           string            `json:"os_image"`
	KernelVersion     string            `json:"kernel_version"`
	Architecture      string            `json:"architecture"`
	CreatedAt         string            `json:"created_at"`
}

// K8sNodeTaint 节点污点
type K8sNodeTaint struct {
	Key    string `json:"key"`
	Value  string `json:"value"`
	Effect string `json:"effect"`
}

// K8sNodeDetail 节点详情
type K8sNodeDetail struct {
	Name              string             `json:"name"`
	Labels            map[string]string  `json:"labels"`
	Annotations       map[string]string  `json:"annotations"`
	Taints            []K8sNodeTaint     `json:"taints"`
	Conditions        []K8sNodeCondition `json:"conditions"`
	Pods              []K8sNodePod       `json:"pods"`
	PodCount          int                `json:"pod_count"`
	Schedulable       bool               `json:"schedulable"`
	CPUCapacity       string             `json:"cpu_capacity"`
	MemoryCapacity    string             `json:"memory_capacity"`
	CPUAllocatable    string             `json:"cpu_allocatable"`
	MemoryAllocatable string             `json:"memory_allocatable"`
	CreatedAt         string             `json:"created_at"`
}

// K8sNodeCondition 节点条件
type K8sNodeCondition struct {
	Type    string `json:"type"`
	Status  string `json:"status"`
	Reason  string `json:"reason"`
	Message string `json:"message"`
}

// K8sNodePod 节点上的 Pod
type K8sNodePod struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	Status    string `json:"status"`
	IP        string `json:"ip"`
}

// K8sPV PersistentVolume
type K8sPV struct {
	Name          string `json:"name"`
	Status        string `json:"status"`
	Capacity      string `json:"capacity"`
	AccessModes   string `json:"access_modes"`
	ReclaimPolicy string `json:"reclaim_policy"`
	StorageClass  string `json:"storage_class"`
	ClaimRef      string `json:"claim_ref"`
	VolumeMode    string `json:"volume_mode"`
	CreatedAt     string `json:"created_at"`
}

// K8sStorageClass StorageClass
type K8sStorageClass struct {
	Name              string `json:"name"`
	Provisioner       string `json:"provisioner"`
	ReclaimPolicy     string `json:"reclaim_policy"`
	VolumeBindingMode string `json:"volume_binding_mode"`
	AllowExpansion    bool   `json:"allow_expansion"`
	IsDefault         bool   `json:"is_default"`
	CreatedAt         string `json:"created_at"`
}

// K8sEvent 事件
type K8sEvent struct {
	Name          string `json:"name"`
	Namespace     string `json:"namespace"`
	Type          string `json:"type"`
	Reason        string `json:"reason"`
	Message       string `json:"message"`
	Object        string `json:"object"`
	Count         int32  `json:"count"`
	LastTimestamp string `json:"last_timestamp"`
}

// K8sEndpoint Endpoint
type K8sEndpoint struct {
	Name      string   `json:"name"`
	Namespace string   `json:"namespace"`
	Addresses []string `json:"addresses"`
	Ports     []string `json:"ports"`
	CreatedAt string   `json:"created_at"`
}

// K8sServiceAccount ServiceAccount
type K8sServiceAccount struct {
	Name      string   `json:"name"`
	Namespace string   `json:"namespace"`
	Secrets   []string `json:"secrets"`
	CreatedAt string   `json:"created_at"`
}

// AddNodeTaintRequest 添加节点污点请求
type AddNodeTaintRequest struct {
	Key    string `json:"key" binding:"required"`
	Value  string `json:"value"`
	Effect string `json:"effect" binding:"required,oneof=NoSchedule PreferNoSchedule NoExecute"`
}

// UpdateNodeLabelsRequest 更新节点标签请求
type UpdateNodeLabelsRequest struct {
	Labels map[string]string `json:"labels" binding:"required"`
}

// ==================== Pod 资源监控 ====================

// PodMetricsResponse Pod 资源指标响应
type PodMetricsResponse struct {
	PodName    string             `json:"pod_name"`
	Namespace  string             `json:"namespace"`
	Available  bool               `json:"available"`
	Message    string             `json:"message,omitempty"`
	TotalCPU   int64              `json:"total_cpu"` // 毫核
	TotalMem   int64              `json:"total_mem"` // 字节
	Containers []ContainerMetrics `json:"containers"`
}

// ContainerMetrics 容器资源指标
type ContainerMetrics struct {
	Name       string  `json:"name"`
	CPUUsage   int64   `json:"cpu_usage"`   // 毫核
	CPULimit   int64   `json:"cpu_limit"`   // 毫核
	CPUPercent float64 `json:"cpu_percent"` // 百分比
	MemUsage   int64   `json:"mem_usage"`   // 字节
	MemLimit   int64   `json:"mem_limit"`   // 字节
	MemPercent float64 `json:"mem_percent"` // 百分比
}

// PodMetricsListResponse Pod 资源指标列表响应
type PodMetricsListResponse struct {
	Available bool                `json:"available"`
	Message   string              `json:"message,omitempty"`
	Items     []PodMetricsSummary `json:"items"`
}

// PodMetricsSummary Pod 资源指标摘要
type PodMetricsSummary struct {
	PodName   string `json:"pod_name"`
	Namespace string `json:"namespace"`
	CPUUsage  int64  `json:"cpu_usage"` // 毫核
	MemUsage  int64  `json:"mem_usage"` // 字节
}

// NodeMetricsListResponse 节点资源指标列表响应
type NodeMetricsListResponse struct {
	Available bool          `json:"available"`
	Message   string        `json:"message,omitempty"`
	Items     []NodeMetrics `json:"items"`
}

// NodeMetrics 节点资源指标
type NodeMetrics struct {
	NodeName    string  `json:"node_name"`
	CPUUsage    int64   `json:"cpu_usage"`    // 毫核
	CPUCapacity int64   `json:"cpu_capacity"` // 毫核
	CPUPercent  float64 `json:"cpu_percent"`  // 百分比
	MemUsage    int64   `json:"mem_usage"`    // 字节
	MemCapacity int64   `json:"mem_capacity"` // 字节
	MemPercent  float64 `json:"mem_percent"`  // 百分比
}
