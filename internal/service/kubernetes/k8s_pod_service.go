package kubernetes

import (
	"context"
	"fmt"
	"io"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"devops/pkg/dto"
	apperrors "devops/pkg/errors"
)

// K8sPodService Pod 管理服务
type K8sPodService struct {
	clientMgr *K8sClientManager
}

// NewK8sPodService 创建 Pod 服务
func NewK8sPodService(clientMgr *K8sClientManager) *K8sPodService {
	return &K8sPodService{clientMgr: clientMgr}
}

// PodInfo Pod 详细信息
type PodInfo struct {
	Name       string          `json:"name"`
	Namespace  string          `json:"namespace"`
	Status     string          `json:"status"`
	Ready      string          `json:"ready"`
	Restarts   int32           `json:"restarts"`
	Age        string          `json:"age"`
	IP         string          `json:"ip"`
	Node       string          `json:"node"`
	Containers []ContainerInfo `json:"containers"`
	Labels     map[string]string `json:"labels"`
	CreatedAt  string          `json:"created_at"`
}

// ContainerInfo 容器信息
type ContainerInfo struct {
	Name         string `json:"name"`
	Image        string `json:"image"`
	Ready        bool   `json:"ready"`
	State        string `json:"state"`
	RestartCount int32  `json:"restart_count"`
}

// ListPods 获取 Pod 列表
func (s *K8sPodService) ListPods(ctx context.Context, clusterID uint, namespace string, labelSelector string) ([]PodInfo, error) {
	client, err := s.clientMgr.GetClient(ctx, clusterID)
	if err != nil {
		return nil, err
	}

	opts := metav1.ListOptions{}
	if labelSelector != "" {
		opts.LabelSelector = labelSelector
	}

	podList, err := client.CoreV1().Pods(namespace).List(ctx, opts)
	if err != nil {
		return nil, apperrors.Wrap(err, apperrors.ErrCodeInternalError, "获取Pod列表失败")
	}

	result := make([]PodInfo, len(podList.Items))
	for i, pod := range podList.Items {
		result[i] = s.convertPodInfo(&pod)
	}
	return result, nil
}

// GetPod 获取 Pod 详情
func (s *K8sPodService) GetPod(ctx context.Context, clusterID uint, namespace, name string) (*PodInfo, error) {
	client, err := s.clientMgr.GetClient(ctx, clusterID)
	if err != nil {
		return nil, err
	}

	pod, err := client.CoreV1().Pods(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, apperrors.Wrap(err, apperrors.ErrCodeNotFound, "Pod不存在")
	}

	info := s.convertPodInfo(pod)
	return &info, nil
}

// DeletePod 删除 Pod
func (s *K8sPodService) DeletePod(ctx context.Context, clusterID uint, namespace, name string) error {
	client, err := s.clientMgr.GetClient(ctx, clusterID)
	if err != nil {
		return err
	}

	if err := client.CoreV1().Pods(namespace).Delete(ctx, name, metav1.DeleteOptions{}); err != nil {
		return apperrors.Wrap(err, apperrors.ErrCodeInternalError, "删除Pod失败")
	}
	return nil
}

// GetPodContainers 获取 Pod 的容器列表
func (s *K8sPodService) GetPodContainers(ctx context.Context, clusterID uint, namespace, name string) ([]dto.K8sContainer, error) {
	client, err := s.clientMgr.GetClient(ctx, clusterID)
	if err != nil {
		return nil, err
	}

	pod, err := client.CoreV1().Pods(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, apperrors.Wrap(err, apperrors.ErrCodeNotFound, "Pod不存在")
	}

	containers := make([]dto.K8sContainer, len(pod.Spec.Containers))
	for i, c := range pod.Spec.Containers {
		containers[i] = dto.K8sContainer{
			Name:  c.Name,
			Image: c.Image,
		}
	}
	return containers, nil
}

// convertPodInfo 转换 Pod 信息
func (s *K8sPodService) convertPodInfo(pod *corev1.Pod) PodInfo {
	containers := make([]ContainerInfo, len(pod.Spec.Containers))
	totalRestarts := int32(0)
	readyCount := 0

	statusMap := make(map[string]corev1.ContainerStatus)
	for _, cs := range pod.Status.ContainerStatuses {
		statusMap[cs.Name] = cs
	}

	for i, c := range pod.Spec.Containers {
		cs, ok := statusMap[c.Name]
		state := "Unknown"
		ready := false
		restartCount := int32(0)

		if ok {
			ready = cs.Ready
			restartCount = cs.RestartCount
			totalRestarts += restartCount

			if cs.State.Running != nil {
				state = "Running"
			} else if cs.State.Waiting != nil {
				state = cs.State.Waiting.Reason
			} else if cs.State.Terminated != nil {
				state = cs.State.Terminated.Reason
			}

			if ready {
				readyCount++
			}
		}

		containers[i] = ContainerInfo{
			Name:         c.Name,
			Image:        c.Image,
			Ready:        ready,
			State:        state,
			RestartCount: restartCount,
		}
	}

	age := time.Since(pod.CreationTimestamp.Time)
	ageStr := formatDuration(age)

	return PodInfo{
		Name:       pod.Name,
		Namespace:  pod.Namespace,
		Status:     string(pod.Status.Phase),
		Ready:      fmt.Sprintf("%d/%d", readyCount, len(pod.Spec.Containers)),
		Restarts:   totalRestarts,
		Age:        ageStr,
		IP:         pod.Status.PodIP,
		Node:       pod.Spec.NodeName,
		Containers: containers,
		Labels:     pod.Labels,
		CreatedAt:  pod.CreationTimestamp.Format("2006-01-02 15:04:05"),
	}
}

// formatDuration 格式化时间间隔
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%dh", int(d.Hours()))
	}
	return fmt.Sprintf("%dd", int(d.Hours()/24))
}

// LogRequest 日志请求参数
type LogRequest struct {
	ClusterID  uint   `json:"cluster_id"`
	Namespace  string `json:"namespace"`
	PodName    string `json:"pod_name"`
	Container  string `json:"container"`
	TailLines  int64  `json:"tail_lines"`
	Follow     bool   `json:"follow"`
	Timestamps bool   `json:"timestamps"`
	SinceTime  string `json:"since_time"`
}

// GetLogs 获取 Pod 日志
func (s *K8sPodService) GetLogs(ctx context.Context, req *LogRequest) (string, error) {
	client, err := s.clientMgr.GetClient(ctx, req.ClusterID)
	if err != nil {
		return "", err
	}

	opts := &corev1.PodLogOptions{
		Timestamps: req.Timestamps,
	}
	if req.Container != "" {
		opts.Container = req.Container
	}
	if req.TailLines > 0 {
		opts.TailLines = &req.TailLines
	}

	stream, err := client.CoreV1().Pods(req.Namespace).GetLogs(req.PodName, opts).Stream(ctx)
	if err != nil {
		return "", apperrors.Wrap(err, apperrors.ErrCodeInternalError, "获取日志失败")
	}
	defer stream.Close()

	logs, err := io.ReadAll(stream)
	if err != nil {
		return "", apperrors.Wrap(err, apperrors.ErrCodeInternalError, "读取日志失败")
	}

	return string(logs), nil
}

// StreamLogs 流式获取日志（用于 WebSocket）
func (s *K8sPodService) StreamLogs(ctx context.Context, req *LogRequest, writer io.Writer) error {
	client, err := s.clientMgr.GetClient(ctx, req.ClusterID)
	if err != nil {
		return err
	}

	opts := &corev1.PodLogOptions{
		Follow:     true,
		Timestamps: req.Timestamps,
	}
	if req.Container != "" {
		opts.Container = req.Container
	}
	if req.TailLines > 0 {
		opts.TailLines = &req.TailLines
	}

	stream, err := client.CoreV1().Pods(req.Namespace).GetLogs(req.PodName, opts).Stream(ctx)
	if err != nil {
		return apperrors.Wrap(err, apperrors.ErrCodeInternalError, "获取日志流失败")
	}
	defer stream.Close()

	buf := make([]byte, 4096)
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			n, err := stream.Read(buf)
			if err != nil {
				if err == io.EOF {
					return nil
				}
				return err
			}
			if n > 0 {
				if _, err := writer.Write(buf[:n]); err != nil {
					return err
				}
			}
		}
	}
}
