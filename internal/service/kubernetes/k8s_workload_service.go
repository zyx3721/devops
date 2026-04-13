package kubernetes

import (
	"context"
	"io"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"devops/pkg/dto"
	apperrors "devops/pkg/errors"
)

// K8sWorkloadService 工作负载服务
type K8sWorkloadService struct {
	clientMgr *K8sClientManager
}

// NewK8sWorkloadService 创建工作负载服务
func NewK8sWorkloadService(clientMgr *K8sClientManager) *K8sWorkloadService {
	return &K8sWorkloadService{clientMgr: clientMgr}
}

// GetDeployments 获取 Deployment 列表
func (s *K8sWorkloadService) GetDeployments(ctx context.Context, clusterID uint, namespace string) ([]dto.K8sDeployment, error) {
	client, err := s.clientMgr.GetClient(ctx, clusterID)
	if err != nil {
		return nil, err
	}

	deployList, err := client.AppsV1().Deployments(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, apperrors.Wrap(err, apperrors.ErrCodeInternalError, "获取Deployment失败")
	}

	result := make([]dto.K8sDeployment, len(deployList.Items))
	for i, deploy := range deployList.Items {
		images := []string{}
		for _, c := range deploy.Spec.Template.Spec.Containers {
			images = append(images, c.Image)
		}
		replicas := int32(1)
		if deploy.Spec.Replicas != nil {
			replicas = *deploy.Spec.Replicas
		}
		result[i] = dto.K8sDeployment{
			Name:      deploy.Name,
			Namespace: deploy.Namespace,
			Replicas:  replicas,
			Ready:     deploy.Status.ReadyReplicas,
			Available: deploy.Status.AvailableReplicas,
			Images:    images,
			CreatedAt: deploy.CreationTimestamp.Format("2006-01-02 15:04:05"),
		}
	}
	return result, nil
}

// GetStatefulSets 获取 StatefulSet 列表
func (s *K8sWorkloadService) GetStatefulSets(ctx context.Context, clusterID uint, namespace string) ([]dto.K8sStatefulSet, error) {
	client, err := s.clientMgr.GetClient(ctx, clusterID)
	if err != nil {
		return nil, err
	}

	stsList, err := client.AppsV1().StatefulSets(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, apperrors.Wrap(err, apperrors.ErrCodeInternalError, "获取StatefulSet失败")
	}

	result := make([]dto.K8sStatefulSet, len(stsList.Items))
	for i, sts := range stsList.Items {
		replicas := int32(1)
		if sts.Spec.Replicas != nil {
			replicas = *sts.Spec.Replicas
		}
		result[i] = dto.K8sStatefulSet{
			Name:      sts.Name,
			Namespace: sts.Namespace,
			Replicas:  replicas,
			Ready:     sts.Status.ReadyReplicas,
			CreatedAt: sts.CreationTimestamp.Format("2006-01-02 15:04:05"),
		}
	}
	return result, nil
}

// GetDaemonSets 获取 DaemonSet 列表
func (s *K8sWorkloadService) GetDaemonSets(ctx context.Context, clusterID uint, namespace string) ([]dto.K8sDaemonSet, error) {
	client, err := s.clientMgr.GetClient(ctx, clusterID)
	if err != nil {
		return nil, err
	}

	dsList, err := client.AppsV1().DaemonSets(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, apperrors.Wrap(err, apperrors.ErrCodeInternalError, "获取DaemonSet失败")
	}

	result := make([]dto.K8sDaemonSet, len(dsList.Items))
	for i, ds := range dsList.Items {
		result[i] = dto.K8sDaemonSet{
			Name:      ds.Name,
			Namespace: ds.Namespace,
			Desired:   ds.Status.DesiredNumberScheduled,
			Ready:     ds.Status.NumberReady,
			CreatedAt: ds.CreationTimestamp.Format("2006-01-02 15:04:05"),
		}
	}
	return result, nil
}

// GetJobs 获取 Job 列表
func (s *K8sWorkloadService) GetJobs(ctx context.Context, clusterID uint, namespace string) ([]dto.K8sJob, error) {
	client, err := s.clientMgr.GetClient(ctx, clusterID)
	if err != nil {
		return nil, err
	}

	jobList, err := client.BatchV1().Jobs(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, apperrors.Wrap(err, apperrors.ErrCodeInternalError, "获取Job失败")
	}

	result := make([]dto.K8sJob, len(jobList.Items))
	for i, job := range jobList.Items {
		completions := int32(1)
		if job.Spec.Completions != nil {
			completions = *job.Spec.Completions
		}
		result[i] = dto.K8sJob{
			Name:        job.Name,
			Namespace:   job.Namespace,
			Completions: completions,
			Succeeded:   job.Status.Succeeded,
			Failed:      job.Status.Failed,
			CreatedAt:   job.CreationTimestamp.Format("2006-01-02 15:04:05"),
		}
	}
	return result, nil
}

// GetCronJobs 获取 CronJob 列表
func (s *K8sWorkloadService) GetCronJobs(ctx context.Context, clusterID uint, namespace string) ([]dto.K8sCronJob, error) {
	client, err := s.clientMgr.GetClient(ctx, clusterID)
	if err != nil {
		return nil, err
	}

	cjList, err := client.BatchV1().CronJobs(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, apperrors.Wrap(err, apperrors.ErrCodeInternalError, "获取CronJob失败")
	}

	result := make([]dto.K8sCronJob, len(cjList.Items))
	for i, cj := range cjList.Items {
		lastSchedule := ""
		if cj.Status.LastScheduleTime != nil {
			lastSchedule = cj.Status.LastScheduleTime.Format("2006-01-02 15:04:05")
		}
		result[i] = dto.K8sCronJob{
			Name:         cj.Name,
			Namespace:    cj.Namespace,
			Schedule:     cj.Spec.Schedule,
			Suspend:      cj.Spec.Suspend != nil && *cj.Spec.Suspend,
			LastSchedule: lastSchedule,
			CreatedAt:    cj.CreationTimestamp.Format("2006-01-02 15:04:05"),
		}
	}
	return result, nil
}

// GetPods 获取 Pod 列表
func (s *K8sWorkloadService) GetPods(ctx context.Context, clusterID uint, namespace string) ([]dto.K8sPod, error) {
	client, err := s.clientMgr.GetClient(ctx, clusterID)
	if err != nil {
		return nil, err
	}

	podList, err := client.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, apperrors.Wrap(err, apperrors.ErrCodeInternalError, "获取Pod失败")
	}

	result := make([]dto.K8sPod, len(podList.Items))
	for i, pod := range podList.Items {
		containers := make([]dto.K8sContainer, len(pod.Spec.Containers))
		for j, c := range pod.Spec.Containers {
			containers[j] = dto.K8sContainer{Name: c.Name, Image: c.Image}
		}

		restarts := int32(0)
		for _, cs := range pod.Status.ContainerStatuses {
			restarts += cs.RestartCount
		}

		result[i] = dto.K8sPod{
			Name:       pod.Name,
			Namespace:  pod.Namespace,
			Status:     string(pod.Status.Phase),
			Node:       pod.Spec.NodeName,
			IP:         pod.Status.PodIP,
			Restarts:   restarts,
			Containers: containers,
			CreatedAt:  pod.CreationTimestamp.Format("2006-01-02 15:04:05"),
		}
	}
	return result, nil
}

// GetPodLogs 获取 Pod 日志
func (s *K8sWorkloadService) GetPodLogs(ctx context.Context, clusterID uint, namespace, podName, container string, tailLines int64) (string, error) {
	client, err := s.clientMgr.GetClient(ctx, clusterID)
	if err != nil {
		return "", err
	}

	opts := &corev1.PodLogOptions{TailLines: &tailLines}
	if container != "" {
		opts.Container = container
	}

	req := client.CoreV1().Pods(namespace).GetLogs(podName, opts)
	stream, err := req.Stream(ctx)
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

// DeletePod 删除 Pod
func (s *K8sWorkloadService) DeletePod(ctx context.Context, clusterID uint, namespace, podName string) error {
	client, err := s.clientMgr.GetClient(ctx, clusterID)
	if err != nil {
		return err
	}

	if err := client.CoreV1().Pods(namespace).Delete(ctx, podName, metav1.DeleteOptions{}); err != nil {
		return apperrors.Wrap(err, apperrors.ErrCodeInternalError, "删除Pod失败")
	}
	return nil
}

// RestartDeployment 重启 Deployment
func (s *K8sWorkloadService) RestartDeployment(ctx context.Context, clusterID uint, namespace, deploymentName string) error {
	client, err := s.clientMgr.GetClient(ctx, clusterID)
	if err != nil {
		return err
	}

	deploy, err := client.AppsV1().Deployments(namespace).Get(ctx, deploymentName, metav1.GetOptions{})
	if err != nil {
		return apperrors.Wrap(err, apperrors.ErrCodeInternalError, "获取Deployment失败")
	}

	if deploy.Spec.Template.Annotations == nil {
		deploy.Spec.Template.Annotations = make(map[string]string)
	}
	deploy.Spec.Template.Annotations["kubectl.kubernetes.io/restartedAt"] = time.Now().Format(time.RFC3339)

	_, err = client.AppsV1().Deployments(namespace).Update(ctx, deploy, metav1.UpdateOptions{})
	if err != nil {
		return apperrors.Wrap(err, apperrors.ErrCodeInternalError, "重启Deployment失败")
	}
	return nil
}

// ScaleDeployment 调整 Deployment 副本数
func (s *K8sWorkloadService) ScaleDeployment(ctx context.Context, clusterID uint, namespace, deploymentName string, replicas int32) error {
	client, err := s.clientMgr.GetClient(ctx, clusterID)
	if err != nil {
		return err
	}

	scale, err := client.AppsV1().Deployments(namespace).GetScale(ctx, deploymentName, metav1.GetOptions{})
	if err != nil {
		return apperrors.Wrap(err, apperrors.ErrCodeInternalError, "获取Deployment失败")
	}

	scale.Spec.Replicas = replicas
	_, err = client.AppsV1().Deployments(namespace).UpdateScale(ctx, deploymentName, scale, metav1.UpdateOptions{})
	if err != nil {
		return apperrors.Wrap(err, apperrors.ErrCodeInternalError, "调整副本数失败")
	}
	return nil
}

// GetRelatedPods 获取工作负载关联的 Pods
func (s *K8sWorkloadService) GetRelatedPods(ctx context.Context, clusterID uint, ownerType, namespace, ownerName string) ([]dto.K8sPod, error) {
	client, err := s.clientMgr.GetClient(ctx, clusterID)
	if err != nil {
		return nil, err
	}

	// 获取所有 Pods
	podList, err := client.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, apperrors.Wrap(err, apperrors.ErrCodeInternalError, "获取Pod列表失败")
	}

	var result []dto.K8sPod
	for _, pod := range podList.Items {
		// 检查 OwnerReferences
		for _, owner := range pod.OwnerReferences {
			matched := false
			switch ownerType {
			case "deployment":
				// Deployment -> ReplicaSet -> Pod
				if owner.Kind == "ReplicaSet" {
					// 获取 ReplicaSet 检查其 owner
					rs, err := client.AppsV1().ReplicaSets(namespace).Get(ctx, owner.Name, metav1.GetOptions{})
					if err == nil {
						for _, rsOwner := range rs.OwnerReferences {
							if rsOwner.Kind == "Deployment" && rsOwner.Name == ownerName {
								matched = true
								break
							}
						}
					}
				}
			case "statefulset":
				if owner.Kind == "StatefulSet" && owner.Name == ownerName {
					matched = true
				}
			case "daemonset":
				if owner.Kind == "DaemonSet" && owner.Name == ownerName {
					matched = true
				}
			case "job":
				if owner.Kind == "Job" && owner.Name == ownerName {
					matched = true
				}
			case "cronjob":
				// CronJob -> Job -> Pod
				if owner.Kind == "Job" {
					job, err := client.BatchV1().Jobs(namespace).Get(ctx, owner.Name, metav1.GetOptions{})
					if err == nil {
						for _, jobOwner := range job.OwnerReferences {
							if jobOwner.Kind == "CronJob" && jobOwner.Name == ownerName {
								matched = true
								break
							}
						}
					}
				}
			case "replicaset":
				if owner.Kind == "ReplicaSet" && owner.Name == ownerName {
					matched = true
				}
			}

			if matched {
				containers := make([]dto.K8sContainer, len(pod.Spec.Containers))
				for j, c := range pod.Spec.Containers {
					containers[j] = dto.K8sContainer{Name: c.Name, Image: c.Image}
				}

				restarts := int32(0)
				for _, cs := range pod.Status.ContainerStatuses {
					restarts += cs.RestartCount
				}

				result = append(result, dto.K8sPod{
					Name:       pod.Name,
					Namespace:  pod.Namespace,
					Status:     string(pod.Status.Phase),
					Node:       pod.Spec.NodeName,
					IP:         pod.Status.PodIP,
					Restarts:   restarts,
					Containers: containers,
					CreatedAt:  pod.CreationTimestamp.Format("2006-01-02 15:04:05"),
				})
				break
			}
		}
	}

	return result, nil
}
