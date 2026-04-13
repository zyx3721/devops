package kubernetes

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"devops/pkg/dto"
	apperrors "devops/pkg/errors"
)

// K8sStorageService 存储资源服务
type K8sStorageService struct {
	clientMgr *K8sClientManager
}

// NewK8sStorageService 创建存储资源服务
func NewK8sStorageService(clientMgr *K8sClientManager) *K8sStorageService {
	return &K8sStorageService{clientMgr: clientMgr}
}

// GetPVCs 获取 PVC 列表
func (s *K8sStorageService) GetPVCs(ctx context.Context, clusterID uint, namespace string) ([]dto.K8sPVC, error) {
	client, err := s.clientMgr.GetClient(ctx, clusterID)
	if err != nil {
		return nil, err
	}

	pvcList, err := client.CoreV1().PersistentVolumeClaims(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, apperrors.Wrap(err, apperrors.ErrCodeInternalError, "获取PVC失败")
	}

	result := make([]dto.K8sPVC, len(pvcList.Items))
	for i, pvc := range pvcList.Items {
		capacity := ""
		if pvc.Status.Capacity != nil {
			if storage, ok := pvc.Status.Capacity[corev1.ResourceStorage]; ok {
				capacity = storage.String()
			}
		}
		storageClass := ""
		if pvc.Spec.StorageClassName != nil {
			storageClass = *pvc.Spec.StorageClassName
		}
		accessModes := ""
		for j, mode := range pvc.Spec.AccessModes {
			if j > 0 {
				accessModes += ","
			}
			accessModes += string(mode)
		}
		result[i] = dto.K8sPVC{
			Name:         pvc.Name,
			Namespace:    pvc.Namespace,
			Status:       string(pvc.Status.Phase),
			Capacity:     capacity,
			StorageClass: storageClass,
			AccessModes:  accessModes,
			CreatedAt:    pvc.CreationTimestamp.Format("2006-01-02 15:04:05"),
		}
	}
	return result, nil
}

// GetPVs 获取 PV 列表
func (s *K8sStorageService) GetPVs(ctx context.Context, clusterID uint) ([]dto.K8sPV, error) {
	client, err := s.clientMgr.GetClient(ctx, clusterID)
	if err != nil {
		return nil, err
	}

	pvList, err := client.CoreV1().PersistentVolumes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, apperrors.Wrap(err, apperrors.ErrCodeInternalError, "获取PV失败")
	}

	result := make([]dto.K8sPV, len(pvList.Items))
	for i, pv := range pvList.Items {
		capacity := ""
		if storage, ok := pv.Spec.Capacity[corev1.ResourceStorage]; ok {
			capacity = storage.String()
		}
		accessModes := ""
		for j, mode := range pv.Spec.AccessModes {
			if j > 0 {
				accessModes += ","
			}
			accessModes += string(mode)
		}
		claimRef := ""
		if pv.Spec.ClaimRef != nil {
			claimRef = pv.Spec.ClaimRef.Namespace + "/" + pv.Spec.ClaimRef.Name
		}
		storageClass := pv.Spec.StorageClassName
		volumeMode := ""
		if pv.Spec.VolumeMode != nil {
			volumeMode = string(*pv.Spec.VolumeMode)
		}
		result[i] = dto.K8sPV{
			Name:            pv.Name,
			Status:          string(pv.Status.Phase),
			Capacity:        capacity,
			AccessModes:     accessModes,
			ReclaimPolicy:   string(pv.Spec.PersistentVolumeReclaimPolicy),
			StorageClass:    storageClass,
			ClaimRef:        claimRef,
			VolumeMode:      volumeMode,
			CreatedAt:       pv.CreationTimestamp.Format("2006-01-02 15:04:05"),
		}
	}
	return result, nil
}

// GetStorageClasses 获取 StorageClass 列表
func (s *K8sStorageService) GetStorageClasses(ctx context.Context, clusterID uint) ([]dto.K8sStorageClass, error) {
	client, err := s.clientMgr.GetClient(ctx, clusterID)
	if err != nil {
		return nil, err
	}

	scList, err := client.StorageV1().StorageClasses().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, apperrors.Wrap(err, apperrors.ErrCodeInternalError, "获取StorageClass失败")
	}

	result := make([]dto.K8sStorageClass, len(scList.Items))
	for i, sc := range scList.Items {
		isDefault := false
		if sc.Annotations != nil {
			if v, ok := sc.Annotations["storageclass.kubernetes.io/is-default-class"]; ok && v == "true" {
				isDefault = true
			}
		}
		reclaimPolicy := ""
		if sc.ReclaimPolicy != nil {
			reclaimPolicy = string(*sc.ReclaimPolicy)
		}
		volumeBindingMode := ""
		if sc.VolumeBindingMode != nil {
			volumeBindingMode = string(*sc.VolumeBindingMode)
		}
		result[i] = dto.K8sStorageClass{
			Name:              sc.Name,
			Provisioner:       sc.Provisioner,
			ReclaimPolicy:     reclaimPolicy,
			VolumeBindingMode: volumeBindingMode,
			AllowExpansion:    sc.AllowVolumeExpansion != nil && *sc.AllowVolumeExpansion,
			IsDefault:         isDefault,
			CreatedAt:         sc.CreationTimestamp.Format("2006-01-02 15:04:05"),
		}
	}
	return result, nil
}
