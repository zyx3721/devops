package kubernetes

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"devops/pkg/dto"
	apperrors "devops/pkg/errors"
)

// K8sConfigService 配置资源服务
type K8sConfigService struct {
	clientMgr *K8sClientManager
}

// NewK8sConfigService 创建配置资源服务
func NewK8sConfigService(clientMgr *K8sClientManager) *K8sConfigService {
	return &K8sConfigService{clientMgr: clientMgr}
}

// GetNamespaces 获取命名空间列表
func (s *K8sConfigService) GetNamespaces(ctx context.Context, clusterID uint) ([]dto.K8sNamespace, error) {
	client, err := s.clientMgr.GetClient(ctx, clusterID)
	if err != nil {
		return nil, err
	}

	nsList, err := client.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, apperrors.Wrap(err, apperrors.ErrCodeInternalError, "获取命名空间失败")
	}

	result := make([]dto.K8sNamespace, len(nsList.Items))
	for i, ns := range nsList.Items {
		result[i] = dto.K8sNamespace{
			Name:      ns.Name,
			Status:    string(ns.Status.Phase),
			CreatedAt: ns.CreationTimestamp.Format("2006-01-02 15:04:05"),
		}
	}
	return result, nil
}

// CreateNamespace 创建命名空间
func (s *K8sConfigService) CreateNamespace(ctx context.Context, clusterID uint, name string, labels map[string]string) error {
	client, err := s.clientMgr.GetClient(ctx, clusterID)
	if err != nil {
		return err
	}

	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: labels,
		},
	}

	_, err = client.CoreV1().Namespaces().Create(ctx, ns, metav1.CreateOptions{})
	if err != nil {
		return apperrors.Wrap(err, apperrors.ErrCodeInternalError, "创建命名空间失败")
	}
	return nil
}

// DeleteNamespace 删除命名空间
func (s *K8sConfigService) DeleteNamespace(ctx context.Context, clusterID uint, name string) error {
	client, err := s.clientMgr.GetClient(ctx, clusterID)
	if err != nil {
		return err
	}

	err = client.CoreV1().Namespaces().Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil {
		return apperrors.Wrap(err, apperrors.ErrCodeInternalError, "删除命名空间失败")
	}
	return nil
}

// GetConfigMaps 获取 ConfigMap 列表
func (s *K8sConfigService) GetConfigMaps(ctx context.Context, clusterID uint, namespace string) ([]dto.K8sConfigMap, error) {
	client, err := s.clientMgr.GetClient(ctx, clusterID)
	if err != nil {
		return nil, err
	}

	cmList, err := client.CoreV1().ConfigMaps(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, apperrors.Wrap(err, apperrors.ErrCodeInternalError, "获取ConfigMap失败")
	}

	result := make([]dto.K8sConfigMap, len(cmList.Items))
	for i, cm := range cmList.Items {
		keys := make([]string, 0, len(cm.Data))
		for k := range cm.Data {
			keys = append(keys, k)
		}
		result[i] = dto.K8sConfigMap{
			Name:      cm.Name,
			Namespace: cm.Namespace,
			Keys:      keys,
			CreatedAt: cm.CreationTimestamp.Format("2006-01-02 15:04:05"),
		}
	}
	return result, nil
}

// GetSecrets 获取 Secret 列表
func (s *K8sConfigService) GetSecrets(ctx context.Context, clusterID uint, namespace string) ([]dto.K8sSecret, error) {
	client, err := s.clientMgr.GetClient(ctx, clusterID)
	if err != nil {
		return nil, err
	}

	secretList, err := client.CoreV1().Secrets(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, apperrors.Wrap(err, apperrors.ErrCodeInternalError, "获取Secret失败")
	}

	result := make([]dto.K8sSecret, len(secretList.Items))
	for i, secret := range secretList.Items {
		keys := make([]string, 0, len(secret.Data))
		for k := range secret.Data {
			keys = append(keys, k)
		}
		result[i] = dto.K8sSecret{
			Name:      secret.Name,
			Namespace: secret.Namespace,
			Type:      string(secret.Type),
			Keys:      keys,
			CreatedAt: secret.CreationTimestamp.Format("2006-01-02 15:04:05"),
		}
	}
	return result, nil
}

// GetServiceAccounts 获取 ServiceAccount 列表
func (s *K8sConfigService) GetServiceAccounts(ctx context.Context, clusterID uint, namespace string) ([]dto.K8sServiceAccount, error) {
	client, err := s.clientMgr.GetClient(ctx, clusterID)
	if err != nil {
		return nil, err
	}

	saList, err := client.CoreV1().ServiceAccounts(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, apperrors.Wrap(err, apperrors.ErrCodeInternalError, "获取ServiceAccount失败")
	}

	result := make([]dto.K8sServiceAccount, len(saList.Items))
	for i, sa := range saList.Items {
		secrets := make([]string, len(sa.Secrets))
		for j, s := range sa.Secrets {
			secrets[j] = s.Name
		}
		result[i] = dto.K8sServiceAccount{
			Name:      sa.Name,
			Namespace: sa.Namespace,
			Secrets:   secrets,
			CreatedAt: sa.CreationTimestamp.Format("2006-01-02 15:04:05"),
		}
	}
	return result, nil
}
