package kubernetes

import (
	"context"
	"fmt"
	"sync"

	"gorm.io/gorm"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"devops/internal/models"
	"devops/pkg/dto"
	"devops/pkg/logger"
)

// K8sOverviewService 集群概览服务
type K8sOverviewService struct {
	clientMgr *K8sClientManager
	db        *gorm.DB
}

// NewK8sOverviewService 创建概览服务
func NewK8sOverviewService(clientMgr *K8sClientManager, db *gorm.DB) *K8sOverviewService {
	return &K8sOverviewService{clientMgr: clientMgr, db: db}
}

// GetClusterOverview 获取单个集群概览
func (s *K8sOverviewService) GetClusterOverview(ctx context.Context, clusterID uint) (*dto.ClusterOverview, error) {
	// 获取集群信息
	var cluster models.K8sCluster
	if err := s.db.First(&cluster, clusterID).Error; err != nil {
		return nil, err
	}

	client, err := s.clientMgr.GetClient(ctx, clusterID)
	if err != nil {
		return &dto.ClusterOverview{
			ClusterID:   clusterID,
			ClusterName: cluster.Name,
			Status:      "disconnected",
		}, nil
	}

	overview := &dto.ClusterOverview{
		ClusterID:   clusterID,
		ClusterName: cluster.Name,
		Status:      "connected",
	}

	// 获取节点信息
	nodes, err := client.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err == nil {
		overview.NodeTotal = len(nodes.Items)
		var cpuCapacity, memCapacity int64
		var podCapacity int
		for _, node := range nodes.Items {
			// 检查节点状态
			for _, cond := range node.Status.Conditions {
				if cond.Type == corev1.NodeReady && cond.Status == corev1.ConditionTrue {
					overview.NodeReady++
					break
				}
			}
			// 累计资源
			if cpu := node.Status.Capacity.Cpu(); cpu != nil {
				cpuCapacity += cpu.MilliValue()
			}
			if mem := node.Status.Capacity.Memory(); mem != nil {
				memCapacity += mem.Value()
			}
			if pods := node.Status.Capacity.Pods(); pods != nil {
				podCapacity += int(pods.Value())
			}
		}
		overview.CPUCapacity = formatCPU(cpuCapacity)
		overview.MemoryCapacity = formatMemory(memCapacity)
		overview.PodCapacity = podCapacity
	} else {
		logger.L().WithField("cluster", cluster.Name).WithError(err).Error("获取节点列表失败")
	}

	// 获取 Pod 数量
	pods, err := client.CoreV1().Pods("").List(ctx, metav1.ListOptions{})
	if err == nil {
		overview.PodUsed = len(pods.Items)
	} else {
		logger.L().WithField("cluster", cluster.Name).WithError(err).Error("获取Pod列表失败")
	}

	// 获取 Deployment 统计
	deployments, err := client.AppsV1().Deployments("").List(ctx, metav1.ListOptions{})
	if err == nil {
		overview.DeploymentTotal = len(deployments.Items)
		for _, d := range deployments.Items {
			if d.Status.ReadyReplicas == *d.Spec.Replicas {
				overview.DeploymentReady++
			}
		}
	} else {
		logger.L().WithField("cluster", cluster.Name).WithError(err).Error("获取Deployment列表失败")
	}

	// 获取 StatefulSet 统计
	statefulsets, err := client.AppsV1().StatefulSets("").List(ctx, metav1.ListOptions{})
	if err == nil {
		overview.StatefulSetTotal = len(statefulsets.Items)
		for _, s := range statefulsets.Items {
			if s.Status.ReadyReplicas == *s.Spec.Replicas {
				overview.StatefulSetReady++
			}
		}
	} else {
		logger.L().WithField("cluster", cluster.Name).WithError(err).Error("获取StatefulSet列表失败")
	}

	// 获取 DaemonSet 统计
	daemonsets, err := client.AppsV1().DaemonSets("").List(ctx, metav1.ListOptions{})
	if err == nil {
		overview.DaemonSetTotal = len(daemonsets.Items)
		for _, d := range daemonsets.Items {
			if d.Status.NumberReady == d.Status.DesiredNumberScheduled {
				overview.DaemonSetReady++
			}
		}
	} else {
		logger.L().WithField("cluster", cluster.Name).WithError(err).Error("获取DaemonSet列表失败")
	}

	return overview, nil
}

// GetMultiClusterOverview 获取多集群概览
func (s *K8sOverviewService) GetMultiClusterOverview(ctx context.Context) (*dto.MultiClusterOverview, error) {
	var clusters []models.K8sCluster
	if err := s.db.Where("status = ?", "active").Find(&clusters).Error; err != nil {
		return nil, err
	}

	result := &dto.MultiClusterOverview{
		Clusters: make([]dto.ClusterOverview, 0, len(clusters)),
		Summary: dto.ClusterSummary{
			TotalClusters: len(clusters),
		},
	}

	var wg sync.WaitGroup
	var mu sync.Mutex
	overviews := make([]dto.ClusterOverview, len(clusters))

	for i, cluster := range clusters {
		wg.Add(1)
		go func(idx int, c models.K8sCluster) {
			defer wg.Done()
			overview, err := s.GetClusterOverview(ctx, c.ID)
			if err != nil {
				logger.L().WithField("cluster", c.Name).WithError(err).Error("获取集群概览失败")
			}
			if overview != nil {
				overviews[idx] = *overview
			} else {
				overviews[idx] = dto.ClusterOverview{
					ClusterID:   c.ID,
					ClusterName: c.Name,
					Status:      "unknown",
				}
			}
		}(i, cluster)
	}
	wg.Wait()

	// 汇总统计
	for _, o := range overviews {
		mu.Lock()
		result.Clusters = append(result.Clusters, o)
		if o.Status == "connected" {
			result.Summary.HealthyClusters++
		}
		result.Summary.TotalNodes += o.NodeTotal
		result.Summary.TotalPods += o.PodUsed
		result.Summary.TotalDeployments += o.DeploymentTotal
		mu.Unlock()
	}

	return result, nil
}

func formatCPU(milliCPU int64) string {
	if milliCPU >= 1000 {
		return fmt.Sprintf("%.1f cores", float64(milliCPU)/1000)
	}
	return fmt.Sprintf("%dm", milliCPU)
}

func formatMemory(bytes int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)
	if bytes >= GB {
		return fmt.Sprintf("%.1f Gi", float64(bytes)/float64(GB))
	}
	if bytes >= MB {
		return fmt.Sprintf("%.1f Mi", float64(bytes)/float64(MB))
	}
	return fmt.Sprintf("%.1f Ki", float64(bytes)/float64(KB))
}
