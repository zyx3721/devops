package kubernetes

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	metricsv1beta1 "k8s.io/metrics/pkg/apis/metrics/v1beta1"
	metricsclient "k8s.io/metrics/pkg/client/clientset/versioned"

	"gorm.io/gorm"

	"devops/internal/models"
	"devops/pkg/dto"
	apperrors "devops/pkg/errors"
	"devops/pkg/logger"
)

// K8sMetricsService K8s 资源监控服务
type K8sMetricsService interface {
	GetPodMetrics(ctx context.Context, clusterID uint, namespace, podName string) (*dto.PodMetricsResponse, error)
	GetPodListMetrics(ctx context.Context, clusterID uint, namespace string) (*dto.PodMetricsListResponse, error)
	GetNodeMetrics(ctx context.Context, clusterID uint) (*dto.NodeMetricsListResponse, error)
	IsMetricsServerAvailable(ctx context.Context, clusterID uint) (bool, error)
}

type k8sMetricsService struct {
	db  *gorm.DB
	log *logger.Logger
}

// NewK8sMetricsService 创建 K8s 资源监控服务
func NewK8sMetricsService(db *gorm.DB) K8sMetricsService {
	return &k8sMetricsService{
		db:  db,
		log: logger.NewLogger("info"),
	}
}

// getMetricsClient 获取 metrics 客户端
func (s *k8sMetricsService) getMetricsClient(clusterID uint) (metricsclient.Interface, error) {
	var cluster models.K8sCluster
	if err := s.db.First(&cluster, clusterID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, apperrors.ErrNotFound
		}
		return nil, apperrors.Wrap(err, apperrors.ErrCodeInternalError, "查询K8s集群失败")
	}

	config, err := clientcmd.RESTConfigFromKubeConfig([]byte(cluster.Kubeconfig))
	if err != nil {
		return nil, apperrors.Wrap(err, apperrors.ErrCodeInternalError, "解析kubeconfig失败")
	}

	client, err := metricsclient.NewForConfig(config)
	if err != nil {
		return nil, apperrors.Wrap(err, apperrors.ErrCodeInternalError, "创建metrics客户端失败")
	}

	return client, nil
}

// getK8sClient 获取 K8s 客户端
func (s *k8sMetricsService) getK8sClient(clusterID uint) (kubernetes.Interface, error) {
	var cluster models.K8sCluster
	if err := s.db.First(&cluster, clusterID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, apperrors.ErrNotFound
		}
		return nil, apperrors.Wrap(err, apperrors.ErrCodeInternalError, "查询K8s集群失败")
	}

	config, err := clientcmd.RESTConfigFromKubeConfig([]byte(cluster.Kubeconfig))
	if err != nil {
		return nil, apperrors.Wrap(err, apperrors.ErrCodeInternalError, "解析kubeconfig失败")
	}

	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, apperrors.Wrap(err, apperrors.ErrCodeInternalError, "创建k8s客户端失败")
	}

	return client, nil
}

// IsMetricsServerAvailable 检查 metrics-server 是否可用
func (s *k8sMetricsService) IsMetricsServerAvailable(ctx context.Context, clusterID uint) (bool, error) {
	client, err := s.getMetricsClient(clusterID)
	if err != nil {
		return false, err
	}

	// 尝试获取节点指标来检测 metrics-server 是否可用
	_, err = client.MetricsV1beta1().NodeMetricses().List(ctx, metav1.ListOptions{Limit: 1})
	if err != nil {
		s.log.WithField("cluster_id", clusterID).WithError(err).Warn("metrics-server 不可用")
		return false, nil
	}

	return true, nil
}

// GetPodMetrics 获取单个 Pod 的资源指标
func (s *k8sMetricsService) GetPodMetrics(ctx context.Context, clusterID uint, namespace, podName string) (*dto.PodMetricsResponse, error) {
	log := s.log.WithField("cluster_id", clusterID).WithField("namespace", namespace).WithField("pod", podName)

	metricsClient, err := s.getMetricsClient(clusterID)
	if err != nil {
		return nil, err
	}

	// 获取 Pod 指标
	podMetrics, err := metricsClient.MetricsV1beta1().PodMetricses(namespace).Get(ctx, podName, metav1.GetOptions{})
	if err != nil {
		log.WithError(err).Warn("获取 Pod 指标失败")
		return &dto.PodMetricsResponse{
			PodName:   podName,
			Namespace: namespace,
			Available: false,
			Message:   "metrics-server 不可用或 Pod 指标未就绪",
		}, nil
	}

	// 获取 Pod 详情以获取资源限制
	k8sClient, err := s.getK8sClient(clusterID)
	if err != nil {
		return nil, err
	}

	pod, err := k8sClient.CoreV1().Pods(namespace).Get(ctx, podName, metav1.GetOptions{})
	if err != nil {
		return nil, apperrors.Wrap(err, apperrors.ErrCodeInternalError, "获取Pod详情失败")
	}

	// 构建响应
	response := &dto.PodMetricsResponse{
		PodName:    podName,
		Namespace:  namespace,
		Available:  true,
		Containers: make([]dto.ContainerMetrics, 0, len(podMetrics.Containers)),
	}

	for _, containerMetrics := range podMetrics.Containers {
		cm := dto.ContainerMetrics{
			Name:     containerMetrics.Name,
			CPUUsage: containerMetrics.Usage.Cpu().MilliValue(),
			MemUsage: containerMetrics.Usage.Memory().Value(),
		}

		// 查找对应容器的资源限制
		for _, container := range pod.Spec.Containers {
			if container.Name == containerMetrics.Name {
				if cpuLimit := container.Resources.Limits.Cpu(); cpuLimit != nil {
					cm.CPULimit = cpuLimit.MilliValue()
					if cm.CPULimit > 0 {
						cm.CPUPercent = float64(cm.CPUUsage) / float64(cm.CPULimit) * 100
					}
				}
				if memLimit := container.Resources.Limits.Memory(); memLimit != nil {
					cm.MemLimit = memLimit.Value()
					if cm.MemLimit > 0 {
						cm.MemPercent = float64(cm.MemUsage) / float64(cm.MemLimit) * 100
					}
				}
				break
			}
		}

		response.Containers = append(response.Containers, cm)
	}

	// 计算 Pod 总资源使用
	for _, cm := range response.Containers {
		response.TotalCPU += cm.CPUUsage
		response.TotalMem += cm.MemUsage
	}

	return response, nil
}

// GetPodListMetrics 获取命名空间下所有 Pod 的资源指标
func (s *k8sMetricsService) GetPodListMetrics(ctx context.Context, clusterID uint, namespace string) (*dto.PodMetricsListResponse, error) {
	log := s.log.WithField("cluster_id", clusterID).WithField("namespace", namespace)

	metricsClient, err := s.getMetricsClient(clusterID)
	if err != nil {
		return nil, err
	}

	// 获取所有 Pod 指标
	var podMetricsList *metricsv1beta1.PodMetricsList
	if namespace == "" {
		podMetricsList, err = metricsClient.MetricsV1beta1().PodMetricses("").List(ctx, metav1.ListOptions{})
	} else {
		podMetricsList, err = metricsClient.MetricsV1beta1().PodMetricses(namespace).List(ctx, metav1.ListOptions{})
	}

	if err != nil {
		log.WithError(err).Warn("获取 Pod 指标列表失败")
		return &dto.PodMetricsListResponse{
			Available: false,
			Message:   "metrics-server 不可用",
			Items:     []dto.PodMetricsSummary{},
		}, nil
	}

	response := &dto.PodMetricsListResponse{
		Available: true,
		Items:     make([]dto.PodMetricsSummary, 0, len(podMetricsList.Items)),
	}

	for _, pm := range podMetricsList.Items {
		summary := dto.PodMetricsSummary{
			PodName:   pm.Name,
			Namespace: pm.Namespace,
		}

		for _, cm := range pm.Containers {
			summary.CPUUsage += cm.Usage.Cpu().MilliValue()
			summary.MemUsage += cm.Usage.Memory().Value()
		}

		response.Items = append(response.Items, summary)
	}

	return response, nil
}

// GetNodeMetrics 获取节点资源指标
func (s *k8sMetricsService) GetNodeMetrics(ctx context.Context, clusterID uint) (*dto.NodeMetricsListResponse, error) {
	log := s.log.WithField("cluster_id", clusterID)

	metricsClient, err := s.getMetricsClient(clusterID)
	if err != nil {
		return nil, err
	}

	k8sClient, err := s.getK8sClient(clusterID)
	if err != nil {
		return nil, err
	}

	// 获取节点指标
	nodeMetricsList, err := metricsClient.MetricsV1beta1().NodeMetricses().List(ctx, metav1.ListOptions{})
	if err != nil {
		log.WithError(err).Warn("获取节点指标失败")
		return &dto.NodeMetricsListResponse{
			Available: false,
			Message:   "metrics-server 不可用",
			Items:     []dto.NodeMetrics{},
		}, nil
	}

	// 获取节点详情
	nodeList, err := k8sClient.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, apperrors.Wrap(err, apperrors.ErrCodeInternalError, "获取节点列表失败")
	}

	// 构建节点容量映射
	nodeCapacity := make(map[string]struct {
		CPUCapacity int64
		MemCapacity int64
	})
	for _, node := range nodeList.Items {
		nodeCapacity[node.Name] = struct {
			CPUCapacity int64
			MemCapacity int64
		}{
			CPUCapacity: node.Status.Capacity.Cpu().MilliValue(),
			MemCapacity: node.Status.Capacity.Memory().Value(),
		}
	}

	response := &dto.NodeMetricsListResponse{
		Available: true,
		Items:     make([]dto.NodeMetrics, 0, len(nodeMetricsList.Items)),
	}

	for _, nm := range nodeMetricsList.Items {
		nodeMetrics := dto.NodeMetrics{
			NodeName: nm.Name,
			CPUUsage: nm.Usage.Cpu().MilliValue(),
			MemUsage: nm.Usage.Memory().Value(),
		}

		if cap, ok := nodeCapacity[nm.Name]; ok {
			nodeMetrics.CPUCapacity = cap.CPUCapacity
			nodeMetrics.MemCapacity = cap.MemCapacity
			if cap.CPUCapacity > 0 {
				nodeMetrics.CPUPercent = float64(nodeMetrics.CPUUsage) / float64(cap.CPUCapacity) * 100
			}
			if cap.MemCapacity > 0 {
				nodeMetrics.MemPercent = float64(nodeMetrics.MemUsage) / float64(cap.MemCapacity) * 100
			}
		}

		response.Items = append(response.Items, nodeMetrics)
	}

	return response, nil
}

// FormatCPU 格式化 CPU 使用量 (毫核 -> 核)
func FormatCPU(milliCores int64) string {
	if milliCores < 1000 {
		return fmt.Sprintf("%dm", milliCores)
	}
	return fmt.Sprintf("%.2f", float64(milliCores)/1000)
}

// FormatMemory 格式化内存使用量 (字节 -> 人类可读)
func FormatMemory(bytes int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)

	switch {
	case bytes >= GB:
		return fmt.Sprintf("%.2fGi", float64(bytes)/float64(GB))
	case bytes >= MB:
		return fmt.Sprintf("%.2fMi", float64(bytes)/float64(MB))
	case bytes >= KB:
		return fmt.Sprintf("%.2fKi", float64(bytes)/float64(KB))
	default:
		return fmt.Sprintf("%dB", bytes)
	}
}
