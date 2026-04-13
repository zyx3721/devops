package kubernetes

import (
	"context"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	apperrors "devops/pkg/errors"
)

// NamespaceService 命名空间服务接口
type NamespaceService interface {
	ListNamespaces(ctx context.Context, clusterID uint) ([]NamespaceInfo, error)
	GetNamespace(ctx context.Context, clusterID uint, name string) (*NamespaceDetail, error)
}

// K8sNamespaceService Namespace 管理服务
type K8sNamespaceService struct {
	clientMgr *K8sClientManager
}

// NewK8sNamespaceService 创建 Namespace 服务
func NewK8sNamespaceService(clientMgr *K8sClientManager) *K8sNamespaceService {
	return &K8sNamespaceService{clientMgr: clientMgr}
}

// NamespaceInfo Namespace 信息
type NamespaceInfo struct {
	Name      string            `json:"name"`
	Status    string            `json:"status"`
	Age       string            `json:"age"`
	Labels    map[string]string `json:"labels"`
	CreatedAt string            `json:"created_at"`
}

// ResourceQuotaInfo 资源配额信息
type ResourceQuotaInfo struct {
	Name string            `json:"name"`
	Hard map[string]string `json:"hard"`
	Used map[string]string `json:"used"`
}

// NamespaceDetail Namespace 详情
type NamespaceDetail struct {
	NamespaceInfo
	Annotations   map[string]string   `json:"annotations"`
	ResourceQuota []ResourceQuotaInfo `json:"resource_quota,omitempty"`
}

// ListNamespaces 获取命名空间列表
func (s *K8sNamespaceService) ListNamespaces(ctx context.Context, clusterID uint) ([]NamespaceInfo, error) {
	client, err := s.clientMgr.GetClient(ctx, clusterID)
	if err != nil {
		return nil, err
	}

	nsList, err := client.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, apperrors.Wrap(err, apperrors.ErrCodeInternalError, "获取Namespace列表失败")
	}

	result := make([]NamespaceInfo, len(nsList.Items))
	for i, ns := range nsList.Items {
		result[i] = s.convertNamespaceInfo(&ns)
	}
	return result, nil
}

// GetNamespace 获取命名空间详情
func (s *K8sNamespaceService) GetNamespace(ctx context.Context, clusterID uint, name string) (*NamespaceDetail, error) {
	client, err := s.clientMgr.GetClient(ctx, clusterID)
	if err != nil {
		return nil, err
	}

	ns, err := client.CoreV1().Namespaces().Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, apperrors.Wrap(err, apperrors.ErrCodeNotFound, "Namespace不存在")
	}

	info := s.convertNamespaceInfo(ns)
	detail := &NamespaceDetail{
		NamespaceInfo: info,
		Annotations:   ns.Annotations,
	}

	// 获取资源配额
	quotaList, err := client.CoreV1().ResourceQuotas(name).List(ctx, metav1.ListOptions{})
	if err == nil && len(quotaList.Items) > 0 {
		detail.ResourceQuota = make([]ResourceQuotaInfo, len(quotaList.Items))
		for i, quota := range quotaList.Items {
			hard := make(map[string]string)
			used := make(map[string]string)

			for k, v := range quota.Status.Hard {
				hard[string(k)] = v.String()
			}
			for k, v := range quota.Status.Used {
				used[string(k)] = v.String()
			}

			detail.ResourceQuota[i] = ResourceQuotaInfo{
				Name: quota.Name,
				Hard: hard,
				Used: used,
			}
		}
	}

	return detail, nil
}

// convertNamespaceInfo 转换 Namespace 信息
func (s *K8sNamespaceService) convertNamespaceInfo(ns *corev1.Namespace) NamespaceInfo {
	age := time.Since(ns.CreationTimestamp.Time)

	// Ensure labels is never nil
	labels := ns.Labels
	if labels == nil {
		labels = make(map[string]string)
	}

	return NamespaceInfo{
		Name:      ns.Name,
		Status:    string(ns.Status.Phase),
		Age:       formatDuration(age),
		Labels:    labels,
		CreatedAt: ns.CreationTimestamp.Format("2006-01-02 15:04:05"),
	}
}
