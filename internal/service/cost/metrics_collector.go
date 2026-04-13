package cost

import (
	"context"
	"encoding/json"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"devops/pkg/logger"
)

// MetricsCollector 从 metrics-server 采集资源使用率
type MetricsCollector struct {
	log *logger.Logger
}

// PodMetrics Pod 指标
type PodMetrics struct {
	Metadata struct {
		Name      string `json:"name"`
		Namespace string `json:"namespace"`
	} `json:"metadata"`
	Containers []ContainerMetrics `json:"containers"`
}

// ContainerMetrics 容器指标
type ContainerMetrics struct {
	Name  string `json:"name"`
	Usage struct {
		CPU    string `json:"cpu"`
		Memory string `json:"memory"`
	} `json:"usage"`
}

// PodMetricsList Pod 指标列表
type PodMetricsList struct {
	Items []PodMetrics `json:"items"`
}

// NewMetricsCollector 创建指标采集器
func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{
		log: logger.NewLogger("MetricsCollector"),
	}
}

// GetPodMetrics 获取 Pod 指标
func (m *MetricsCollector) GetPodMetrics(ctx context.Context, client *kubernetes.Clientset, config *rest.Config, namespace string) (map[string]*PodMetrics, error) {
	// 使用 REST 客户端调用 metrics API
	result := make(map[string]*PodMetrics)

	// 构建 metrics API 路径
	path := "/apis/metrics.k8s.io/v1beta1/pods"
	if namespace != "" {
		path = fmt.Sprintf("/apis/metrics.k8s.io/v1beta1/namespaces/%s/pods", namespace)
	}

	// 调用 metrics API
	data, err := client.RESTClient().Get().AbsPath(path).DoRaw(ctx)
	if err != nil {
		m.log.WithField("error", err.Error()).Warn("获取 Pod 指标失败，metrics-server 可能未安装")
		return result, nil // 返回空结果，不报错
	}

	var metricsList PodMetricsList
	if err := json.Unmarshal(data, &metricsList); err != nil {
		m.log.WithField("error", err.Error()).Warn("解析 Pod 指标失败")
		return result, nil
	}

	for i := range metricsList.Items {
		pod := &metricsList.Items[i]
		key := fmt.Sprintf("%s/%s", pod.Metadata.Namespace, pod.Metadata.Name)
		result[key] = pod
	}

	m.log.WithField("count", len(result)).Debug("获取 Pod 指标成功")
	return result, nil
}

// ParseCPU 解析 CPU 值（返回核数）
func (m *MetricsCollector) ParseCPU(cpuStr string) float64 {
	if cpuStr == "" {
		return 0
	}

	var value float64
	var unit string
	fmt.Sscanf(cpuStr, "%f%s", &value, &unit)

	switch unit {
	case "n": // 纳核
		return value / 1e9
	case "u": // 微核
		return value / 1e6
	case "m": // 毫核
		return value / 1000
	default: // 核
		return value
	}
}

// ParseMemory 解析内存值（返回 GB）
func (m *MetricsCollector) ParseMemory(memStr string) float64 {
	if memStr == "" {
		return 0
	}

	var value float64
	var unit string
	fmt.Sscanf(memStr, "%f%s", &value, &unit)

	switch unit {
	case "Ki":
		return value / (1024 * 1024)
	case "Mi":
		return value / 1024
	case "Gi":
		return value
	case "Ti":
		return value * 1024
	default: // 字节
		return value / (1024 * 1024 * 1024)
	}
}

// GetWorkloadMetrics 获取工作负载的聚合指标
func (m *MetricsCollector) GetWorkloadMetrics(ctx context.Context, client *kubernetes.Clientset, config *rest.Config) (map[string]*WorkloadMetrics, error) {
	result := make(map[string]*WorkloadMetrics)

	// 获取所有 Pod 指标
	podMetrics, err := m.GetPodMetrics(ctx, client, config, "")
	if err != nil {
		return result, err
	}

	// 获取所有 Pod 以关联到工作负载
	pods, err := client.CoreV1().Pods("").List(ctx, metav1.ListOptions{})
	if err != nil {
		m.log.WithField("error", err.Error()).Warn("获取 Pod 列表失败")
		return result, nil
	}

	for _, pod := range pods.Items {
		// 获取 Pod 的 owner
		var ownerKind, ownerName string
		for _, owner := range pod.OwnerReferences {
			if owner.Controller != nil && *owner.Controller {
				ownerKind = owner.Kind
				ownerName = owner.Name
				break
			}
		}

		// ReplicaSet 需要进一步查找 Deployment
		if ownerKind == "ReplicaSet" {
			rs, err := client.AppsV1().ReplicaSets(pod.Namespace).Get(ctx, ownerName, metav1.GetOptions{})
			if err == nil {
				for _, rsOwner := range rs.OwnerReferences {
					if rsOwner.Controller != nil && *rsOwner.Controller && rsOwner.Kind == "Deployment" {
						ownerKind = "Deployment"
						ownerName = rsOwner.Name
						break
					}
				}
			}
		}

		if ownerKind == "" || ownerName == "" {
			continue
		}

		// 获取 Pod 指标
		podKey := fmt.Sprintf("%s/%s", pod.Namespace, pod.Name)
		metrics, ok := podMetrics[podKey]
		if !ok {
			continue
		}

		// 聚合到工作负载
		workloadKey := fmt.Sprintf("%s/%s/%s", pod.Namespace, ownerKind, ownerName)
		if _, exists := result[workloadKey]; !exists {
			result[workloadKey] = &WorkloadMetrics{
				Namespace:   pod.Namespace,
				Kind:        ownerKind,
				Name:        ownerName,
				PodCount:    0,
				CPUUsage:    0,
				MemoryUsage: 0,
			}
		}

		wm := result[workloadKey]
		wm.PodCount++

		// 累加容器指标
		for _, container := range metrics.Containers {
			wm.CPUUsage += m.ParseCPU(container.Usage.CPU)
			wm.MemoryUsage += m.ParseMemory(container.Usage.Memory)
		}
	}

	return result, nil
}

// WorkloadMetrics 工作负载指标
type WorkloadMetrics struct {
	Namespace   string
	Kind        string
	Name        string
	PodCount    int
	CPUUsage    float64 // 核
	MemoryUsage float64 // GB
}

// NodeMetrics 节点指标
type NodeMetrics struct {
	NodeName    string
	CPUUsage    float64 // 核
	MemoryUsage float64 // GB
}

// NodeMetricsList 节点指标列表
type NodeMetricsList struct {
	Items []NodeMetricsItem `json:"items"`
}

// NodeMetricsItem 节点指标项
type NodeMetricsItem struct {
	Metadata struct {
		Name string `json:"name"`
	} `json:"metadata"`
	Usage struct {
		CPU    string `json:"cpu"`
		Memory string `json:"memory"`
	} `json:"usage"`
}

// GetNodeMetrics 获取节点指标
func (m *MetricsCollector) GetNodeMetrics(ctx context.Context, config *rest.Config) (map[string]*NodeMetrics, error) {
	result := make(map[string]*NodeMetrics)

	// 创建 REST 客户端
	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		m.log.WithField("error", err.Error()).Warn("创建K8s客户端失败")
		return result, nil
	}

	// 调用 metrics API 获取节点指标
	path := "/apis/metrics.k8s.io/v1beta1/nodes"
	data, err := client.RESTClient().Get().AbsPath(path).DoRaw(ctx)
	if err != nil {
		m.log.WithField("error", err.Error()).Warn("获取节点指标失败，metrics-server 可能未安装")
		return result, nil
	}

	var metricsList NodeMetricsList
	if err := json.Unmarshal(data, &metricsList); err != nil {
		m.log.WithField("error", err.Error()).Warn("解析节点指标失败")
		return result, nil
	}

	for _, item := range metricsList.Items {
		result[item.Metadata.Name] = &NodeMetrics{
			NodeName:    item.Metadata.Name,
			CPUUsage:    m.ParseCPU(item.Usage.CPU),
			MemoryUsage: m.ParseMemory(item.Usage.Memory),
		}
	}

	m.log.WithField("count", len(result)).Debug("获取节点指标成功")
	return result, nil
}
