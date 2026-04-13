package kubernetes

import (
	"context"
	"fmt"
	"sort"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	apperrors "devops/pkg/errors"
)

// K8sDeploymentService Deployment 管理服务
type K8sDeploymentService struct {
	clientMgr *K8sClientManager
}

// NewK8sDeploymentService 创建 Deployment 服务
func NewK8sDeploymentService(clientMgr *K8sClientManager) *K8sDeploymentService {
	return &K8sDeploymentService{clientMgr: clientMgr}
}

// DeploymentInfo Deployment 信息
type DeploymentInfo struct {
	Name       string                  `json:"name"`
	Namespace  string                  `json:"namespace"`
	Ready      string                  `json:"ready"`
	UpToDate   int32                   `json:"up_to_date"`
	Available  int32                   `json:"available"`
	Age        string                  `json:"age"`
	Images     []string                `json:"images"`
	Replicas   int32                   `json:"replicas"`
	CreatedAt  string                  `json:"created_at"`
	Containers []DeploymentContainer   `json:"containers,omitempty"`
}

// DeploymentContainer Deployment 容器信息
type DeploymentContainer struct {
	Name  string `json:"name"`
	Image string `json:"image"`
}

// DeploymentDetail Deployment 详情
type DeploymentDetail struct {
	DeploymentInfo
	Labels      map[string]string     `json:"labels"`
	Annotations map[string]string     `json:"annotations"`
	Strategy    string                `json:"strategy"`
	Selector    map[string]string     `json:"selector"`
	Conditions  []DeploymentCondition `json:"conditions"`
}

// DeploymentCondition Deployment 状态条件
type DeploymentCondition struct {
	Type    string `json:"type"`
	Status  string `json:"status"`
	Reason  string `json:"reason"`
	Message string `json:"message"`
}

// RevisionInfo 版本信息
type RevisionInfo struct {
	Revision    int64     `json:"revision"`
	Image       string    `json:"image"`
	CreatedAt   time.Time `json:"created_at"`
	ChangeCause string    `json:"change_cause"`
}

// ListDeployments 获取 Deployment 列表
func (s *K8sDeploymentService) ListDeployments(ctx context.Context, clusterID uint, namespace string) ([]DeploymentInfo, error) {
	client, err := s.clientMgr.GetClient(ctx, clusterID)
	if err != nil {
		return nil, err
	}

	deployList, err := client.AppsV1().Deployments(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, apperrors.Wrap(err, apperrors.ErrCodeInternalError, "获取Deployment列表失败")
	}

	result := make([]DeploymentInfo, len(deployList.Items))
	for i, deploy := range deployList.Items {
		result[i] = s.convertDeploymentInfo(&deploy)
	}
	return result, nil
}

// GetDeployment 获取 Deployment 详情
func (s *K8sDeploymentService) GetDeployment(ctx context.Context, clusterID uint, namespace, name string) (*DeploymentDetail, error) {
	client, err := s.clientMgr.GetClient(ctx, clusterID)
	if err != nil {
		return nil, err
	}

	deploy, err := client.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, apperrors.Wrap(err, apperrors.ErrCodeNotFound, "Deployment不存在")
	}

	info := s.convertDeploymentInfo(deploy)
	detail := &DeploymentDetail{
		DeploymentInfo: info,
		Labels:         deploy.Labels,
		Annotations:    deploy.Annotations,
		Strategy:       string(deploy.Spec.Strategy.Type),
		Selector:       deploy.Spec.Selector.MatchLabels,
	}

	for _, cond := range deploy.Status.Conditions {
		detail.Conditions = append(detail.Conditions, DeploymentCondition{
			Type:    string(cond.Type),
			Status:  string(cond.Status),
			Reason:  cond.Reason,
			Message: cond.Message,
		})
	}

	return detail, nil
}

// UpdateImage 更新 Deployment 镜像
func (s *K8sDeploymentService) UpdateImage(ctx context.Context, clusterID uint, namespace, name, container, image string) error {
	client, err := s.clientMgr.GetClient(ctx, clusterID)
	if err != nil {
		return err
	}

	// 使用 patch 更新镜像
	patch := fmt.Sprintf(`{"spec":{"template":{"spec":{"containers":[{"name":"%s","image":"%s"}]}}}}`, container, image)
	_, err = client.AppsV1().Deployments(namespace).Patch(ctx, name, types.StrategicMergePatchType, []byte(patch), metav1.PatchOptions{})
	if err != nil {
		return apperrors.Wrap(err, apperrors.ErrCodeInternalError, "更新镜像失败")
	}

	return nil
}

// Scale 扩缩容
func (s *K8sDeploymentService) Scale(ctx context.Context, clusterID uint, namespace, name string, replicas int32) error {
	if replicas < 0 || replicas > 100 {
		return apperrors.New(apperrors.ErrCodeInvalidParams, "副本数必须在 0-100 之间")
	}

	client, err := s.clientMgr.GetClient(ctx, clusterID)
	if err != nil {
		return err
	}

	scale, err := client.AppsV1().Deployments(namespace).GetScale(ctx, name, metav1.GetOptions{})
	if err != nil {
		return apperrors.Wrap(err, apperrors.ErrCodeInternalError, "获取Deployment失败")
	}

	scale.Spec.Replicas = replicas
	_, err = client.AppsV1().Deployments(namespace).UpdateScale(ctx, name, scale, metav1.UpdateOptions{})
	if err != nil {
		return apperrors.Wrap(err, apperrors.ErrCodeInternalError, "扩缩容失败")
	}

	return nil
}

// Restart 重启 Deployment
func (s *K8sDeploymentService) Restart(ctx context.Context, clusterID uint, namespace, name string) error {
	client, err := s.clientMgr.GetClient(ctx, clusterID)
	if err != nil {
		return err
	}

	// 通过更新 annotation 触发滚动更新
	patch := fmt.Sprintf(`{"spec":{"template":{"metadata":{"annotations":{"kubectl.kubernetes.io/restartedAt":"%s"}}}}}`, time.Now().Format(time.RFC3339))
	_, err = client.AppsV1().Deployments(namespace).Patch(ctx, name, types.StrategicMergePatchType, []byte(patch), metav1.PatchOptions{})
	if err != nil {
		return apperrors.Wrap(err, apperrors.ErrCodeInternalError, "重启Deployment失败")
	}

	return nil
}

// GetRevisionHistory 获取版本历史
func (s *K8sDeploymentService) GetRevisionHistory(ctx context.Context, clusterID uint, namespace, name string) ([]RevisionInfo, error) {
	client, err := s.clientMgr.GetClient(ctx, clusterID)
	if err != nil {
		return nil, err
	}

	// 获取 Deployment
	deploy, err := client.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, apperrors.Wrap(err, apperrors.ErrCodeNotFound, "Deployment不存在")
	}

	// 获取关联的 ReplicaSets
	selector, err := metav1.LabelSelectorAsSelector(deploy.Spec.Selector)
	if err != nil {
		return nil, err
	}

	rsList, err := client.AppsV1().ReplicaSets(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: selector.String(),
	})
	if err != nil {
		return nil, apperrors.Wrap(err, apperrors.ErrCodeInternalError, "获取ReplicaSet列表失败")
	}

	// 过滤属于该 Deployment 的 ReplicaSets
	var revisions []RevisionInfo
	for _, rs := range rsList.Items {
		// 检查 OwnerReference
		for _, owner := range rs.OwnerReferences {
			if owner.Kind == "Deployment" && owner.Name == name {
				revision := getRevision(&rs)
				image := ""
				if len(rs.Spec.Template.Spec.Containers) > 0 {
					image = rs.Spec.Template.Spec.Containers[0].Image
				}
				changeCause := ""
				if rs.Annotations != nil {
					changeCause = rs.Annotations["kubernetes.io/change-cause"]
				}
				revisions = append(revisions, RevisionInfo{
					Revision:    revision,
					Image:       image,
					CreatedAt:   rs.CreationTimestamp.Time,
					ChangeCause: changeCause,
				})
				break
			}
		}
	}

	// 按版本号排序
	sort.Slice(revisions, func(i, j int) bool {
		return revisions[i].Revision > revisions[j].Revision
	})

	return revisions, nil
}

// Rollback 回滚到指定版本
func (s *K8sDeploymentService) Rollback(ctx context.Context, clusterID uint, namespace, name string, revision int64) error {
	client, err := s.clientMgr.GetClient(ctx, clusterID)
	if err != nil {
		return err
	}

	// 获取 Deployment
	deploy, err := client.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return apperrors.Wrap(err, apperrors.ErrCodeNotFound, "Deployment不存在")
	}

	// 获取目标版本的 ReplicaSet
	selector, err := metav1.LabelSelectorAsSelector(deploy.Spec.Selector)
	if err != nil {
		return err
	}

	rsList, err := client.AppsV1().ReplicaSets(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: selector.String(),
	})
	if err != nil {
		return apperrors.Wrap(err, apperrors.ErrCodeInternalError, "获取ReplicaSet列表失败")
	}

	var targetRS *appsv1.ReplicaSet
	for i := range rsList.Items {
		rs := &rsList.Items[i]
		if getRevision(rs) == revision {
			targetRS = rs
			break
		}
	}

	if targetRS == nil {
		return apperrors.New(apperrors.ErrCodeNotFound, "目标版本不存在")
	}

	// 更新 Deployment 的 template 为目标版本的 template
	deploy.Spec.Template = targetRS.Spec.Template
	_, err = client.AppsV1().Deployments(namespace).Update(ctx, deploy, metav1.UpdateOptions{})
	if err != nil {
		return apperrors.Wrap(err, apperrors.ErrCodeInternalError, "回滚失败")
	}

	return nil
}

// GetUpdateProgress 获取更新进度
func (s *K8sDeploymentService) GetUpdateProgress(ctx context.Context, clusterID uint, namespace, name string) (map[string]interface{}, error) {
	client, err := s.clientMgr.GetClient(ctx, clusterID)
	if err != nil {
		return nil, err
	}

	deploy, err := client.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, apperrors.Wrap(err, apperrors.ErrCodeNotFound, "Deployment不存在")
	}

	replicas := int32(1)
	if deploy.Spec.Replicas != nil {
		replicas = *deploy.Spec.Replicas
	}

	progress := map[string]interface{}{
		"replicas":           replicas,
		"updated_replicas":   deploy.Status.UpdatedReplicas,
		"ready_replicas":     deploy.Status.ReadyReplicas,
		"available_replicas": deploy.Status.AvailableReplicas,
		"unavailable":        deploy.Status.UnavailableReplicas,
	}

	// 判断是否完成
	if deploy.Status.UpdatedReplicas == replicas &&
		deploy.Status.ReadyReplicas == replicas &&
		deploy.Status.AvailableReplicas == replicas {
		progress["status"] = "completed"
	} else {
		progress["status"] = "updating"
	}

	return progress, nil
}

// convertDeploymentInfo 转换 Deployment 信息
func (s *K8sDeploymentService) convertDeploymentInfo(deploy *appsv1.Deployment) DeploymentInfo {
	images := []string{}
	containers := []DeploymentContainer{}
	for _, c := range deploy.Spec.Template.Spec.Containers {
		images = append(images, c.Image)
		containers = append(containers, DeploymentContainer{
			Name:  c.Name,
			Image: c.Image,
		})
	}

	replicas := int32(1)
	if deploy.Spec.Replicas != nil {
		replicas = *deploy.Spec.Replicas
	}

	age := time.Since(deploy.CreationTimestamp.Time)

	return DeploymentInfo{
		Name:       deploy.Name,
		Namespace:  deploy.Namespace,
		Ready:      fmt.Sprintf("%d/%d", deploy.Status.ReadyReplicas, replicas),
		UpToDate:   deploy.Status.UpdatedReplicas,
		Available:  deploy.Status.AvailableReplicas,
		Age:        formatDuration(age),
		Images:     images,
		Replicas:   replicas,
		CreatedAt:  deploy.CreationTimestamp.Format("2006-01-02 15:04:05"),
		Containers: containers,
	}
}

// getRevision 获取 ReplicaSet 的版本号
func getRevision(rs *appsv1.ReplicaSet) int64 {
	if rs.Annotations == nil {
		return 0
	}
	revStr := rs.Annotations["deployment.kubernetes.io/revision"]
	var rev int64
	fmt.Sscanf(revStr, "%d", &rev)
	return rev
}
