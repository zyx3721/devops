package kubernetes

import (
	"context"
	"fmt"

	autoscalingv2 "k8s.io/api/autoscaling/v2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"devops/pkg/dto"
	apperrors "devops/pkg/errors"
)

// K8sHPAService HPA 管理服务
type K8sHPAService struct {
	clientMgr *K8sClientManager
}

// NewK8sHPAService 创建 HPA 服务
func NewK8sHPAService(clientMgr *K8sClientManager) *K8sHPAService {
	return &K8sHPAService{clientMgr: clientMgr}
}

// ListHPAs 获取 HPA 列表
func (s *K8sHPAService) ListHPAs(ctx context.Context, clusterID uint, namespace string) ([]dto.K8sHPA, error) {
	client, err := s.clientMgr.GetClient(ctx, clusterID)
	if err != nil {
		return nil, err
	}

	hpaList, err := client.AutoscalingV2().HorizontalPodAutoscalers(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, apperrors.Wrap(err, apperrors.ErrCodeInternalError, "获取HPA列表失败")
	}

	result := make([]dto.K8sHPA, len(hpaList.Items))
	for i, hpa := range hpaList.Items {
		metrics := s.formatMetrics(&hpa)
		result[i] = dto.K8sHPA{
			Name:            hpa.Name,
			Namespace:       hpa.Namespace,
			TargetKind:      hpa.Spec.ScaleTargetRef.Kind,
			TargetName:      hpa.Spec.ScaleTargetRef.Name,
			MinReplicas:     *hpa.Spec.MinReplicas,
			MaxReplicas:     hpa.Spec.MaxReplicas,
			CurrentReplicas: hpa.Status.CurrentReplicas,
			DesiredReplicas: hpa.Status.DesiredReplicas,
			Metrics:         metrics,
			CreatedAt:       hpa.CreationTimestamp.Format("2006-01-02 15:04:05"),
		}
	}
	return result, nil
}

// GetHPA 获取单个 HPA
func (s *K8sHPAService) GetHPA(ctx context.Context, clusterID uint, namespace, name string) (*dto.K8sHPA, error) {
	client, err := s.clientMgr.GetClient(ctx, clusterID)
	if err != nil {
		return nil, err
	}

	hpa, err := client.AutoscalingV2().HorizontalPodAutoscalers(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, apperrors.Wrap(err, apperrors.ErrCodeNotFound, "HPA不存在")
	}

	metrics := s.formatMetrics(hpa)
	return &dto.K8sHPA{
		Name:            hpa.Name,
		Namespace:       hpa.Namespace,
		TargetKind:      hpa.Spec.ScaleTargetRef.Kind,
		TargetName:      hpa.Spec.ScaleTargetRef.Name,
		MinReplicas:     *hpa.Spec.MinReplicas,
		MaxReplicas:     hpa.Spec.MaxReplicas,
		CurrentReplicas: hpa.Status.CurrentReplicas,
		DesiredReplicas: hpa.Status.DesiredReplicas,
		Metrics:         metrics,
		CreatedAt:       hpa.CreationTimestamp.Format("2006-01-02 15:04:05"),
	}, nil
}

// CreateHPA 创建 HPA
func (s *K8sHPAService) CreateHPA(ctx context.Context, clusterID uint, req *dto.CreateHPARequest) error {
	client, err := s.clientMgr.GetClient(ctx, clusterID)
	if err != nil {
		return err
	}

	minReplicas := req.MinReplicas
	hpa := &autoscalingv2.HorizontalPodAutoscaler{
		ObjectMeta: metav1.ObjectMeta{
			Name:      req.Name,
			Namespace: req.Namespace,
		},
		Spec: autoscalingv2.HorizontalPodAutoscalerSpec{
			ScaleTargetRef: autoscalingv2.CrossVersionObjectReference{
				APIVersion: "apps/v1",
				Kind:       req.TargetKind,
				Name:       req.TargetName,
			},
			MinReplicas: &minReplicas,
			MaxReplicas: req.MaxReplicas,
			Metrics:     []autoscalingv2.MetricSpec{},
		},
	}

	// 添加 CPU 指标
	if req.CPUTargetPercent != nil && *req.CPUTargetPercent > 0 {
		hpa.Spec.Metrics = append(hpa.Spec.Metrics, autoscalingv2.MetricSpec{
			Type: autoscalingv2.ResourceMetricSourceType,
			Resource: &autoscalingv2.ResourceMetricSource{
				Name: corev1.ResourceCPU,
				Target: autoscalingv2.MetricTarget{
					Type:               autoscalingv2.UtilizationMetricType,
					AverageUtilization: req.CPUTargetPercent,
				},
			},
		})
	}

	// 添加内存指标
	if req.MemTargetPercent != nil && *req.MemTargetPercent > 0 {
		hpa.Spec.Metrics = append(hpa.Spec.Metrics, autoscalingv2.MetricSpec{
			Type: autoscalingv2.ResourceMetricSourceType,
			Resource: &autoscalingv2.ResourceMetricSource{
				Name: corev1.ResourceMemory,
				Target: autoscalingv2.MetricTarget{
					Type:               autoscalingv2.UtilizationMetricType,
					AverageUtilization: req.MemTargetPercent,
				},
			},
		})
	}

	// 如果没有指定任何指标，默认使用 CPU 80%
	if len(hpa.Spec.Metrics) == 0 {
		defaultCPU := int32(80)
		hpa.Spec.Metrics = append(hpa.Spec.Metrics, autoscalingv2.MetricSpec{
			Type: autoscalingv2.ResourceMetricSourceType,
			Resource: &autoscalingv2.ResourceMetricSource{
				Name: corev1.ResourceCPU,
				Target: autoscalingv2.MetricTarget{
					Type:               autoscalingv2.UtilizationMetricType,
					AverageUtilization: &defaultCPU,
				},
			},
		})
	}

	_, err = client.AutoscalingV2().HorizontalPodAutoscalers(req.Namespace).Create(ctx, hpa, metav1.CreateOptions{})
	if err != nil {
		return apperrors.Wrap(err, apperrors.ErrCodeInternalError, "创建HPA失败")
	}
	return nil
}

// UpdateHPA 更新 HPA
func (s *K8sHPAService) UpdateHPA(ctx context.Context, clusterID uint, namespace, name string, req *dto.UpdateHPARequest) error {
	client, err := s.clientMgr.GetClient(ctx, clusterID)
	if err != nil {
		return err
	}

	hpa, err := client.AutoscalingV2().HorizontalPodAutoscalers(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return apperrors.Wrap(err, apperrors.ErrCodeNotFound, "HPA不存在")
	}

	hpa.Spec.MinReplicas = &req.MinReplicas
	hpa.Spec.MaxReplicas = req.MaxReplicas

	// 更新指标
	hpa.Spec.Metrics = []autoscalingv2.MetricSpec{}
	if req.CPUTargetPercent != nil && *req.CPUTargetPercent > 0 {
		hpa.Spec.Metrics = append(hpa.Spec.Metrics, autoscalingv2.MetricSpec{
			Type: autoscalingv2.ResourceMetricSourceType,
			Resource: &autoscalingv2.ResourceMetricSource{
				Name: corev1.ResourceCPU,
				Target: autoscalingv2.MetricTarget{
					Type:               autoscalingv2.UtilizationMetricType,
					AverageUtilization: req.CPUTargetPercent,
				},
			},
		})
	}
	if req.MemTargetPercent != nil && *req.MemTargetPercent > 0 {
		hpa.Spec.Metrics = append(hpa.Spec.Metrics, autoscalingv2.MetricSpec{
			Type: autoscalingv2.ResourceMetricSourceType,
			Resource: &autoscalingv2.ResourceMetricSource{
				Name: corev1.ResourceMemory,
				Target: autoscalingv2.MetricTarget{
					Type:               autoscalingv2.UtilizationMetricType,
					AverageUtilization: req.MemTargetPercent,
				},
			},
		})
	}

	_, err = client.AutoscalingV2().HorizontalPodAutoscalers(namespace).Update(ctx, hpa, metav1.UpdateOptions{})
	if err != nil {
		return apperrors.Wrap(err, apperrors.ErrCodeInternalError, "更新HPA失败")
	}
	return nil
}

// DeleteHPA 删除 HPA
func (s *K8sHPAService) DeleteHPA(ctx context.Context, clusterID uint, namespace, name string) error {
	client, err := s.clientMgr.GetClient(ctx, clusterID)
	if err != nil {
		return err
	}

	err = client.AutoscalingV2().HorizontalPodAutoscalers(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil {
		return apperrors.Wrap(err, apperrors.ErrCodeInternalError, "删除HPA失败")
	}
	return nil
}

// formatMetrics 格式化指标显示
func (s *K8sHPAService) formatMetrics(hpa *autoscalingv2.HorizontalPodAutoscaler) []string {
	var metrics []string
	for _, metric := range hpa.Spec.Metrics {
		if metric.Type == autoscalingv2.ResourceMetricSourceType && metric.Resource != nil {
			name := string(metric.Resource.Name)
			if metric.Resource.Target.AverageUtilization != nil {
				metrics = append(metrics, fmt.Sprintf("%s: %d%%", name, *metric.Resource.Target.AverageUtilization))
			}
		}
	}
	return metrics
}

// ==================== 资源配额服务 ====================

// ListResourceQuotas 获取资源配额列表
func (s *K8sHPAService) ListResourceQuotas(ctx context.Context, clusterID uint, namespace string) ([]dto.K8sResourceQuota, error) {
	client, err := s.clientMgr.GetClient(ctx, clusterID)
	if err != nil {
		return nil, err
	}

	quotaList, err := client.CoreV1().ResourceQuotas(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, apperrors.Wrap(err, apperrors.ErrCodeInternalError, "获取资源配额失败")
	}

	result := make([]dto.K8sResourceQuota, len(quotaList.Items))
	for i, quota := range quotaList.Items {
		hard := make(map[string]string)
		used := make(map[string]string)
		for k, v := range quota.Spec.Hard {
			hard[string(k)] = v.String()
		}
		for k, v := range quota.Status.Used {
			used[string(k)] = v.String()
		}
		result[i] = dto.K8sResourceQuota{
			Name:      quota.Name,
			Namespace: quota.Namespace,
			Hard:      hard,
			Used:      used,
			CreatedAt: quota.CreationTimestamp.Format("2006-01-02 15:04:05"),
		}
	}
	return result, nil
}

// CreateResourceQuota 创建资源配额
func (s *K8sHPAService) CreateResourceQuota(ctx context.Context, clusterID uint, req *dto.CreateResourceQuotaRequest) error {
	client, err := s.clientMgr.GetClient(ctx, clusterID)
	if err != nil {
		return err
	}

	hard := make(corev1.ResourceList)
	for k, v := range req.Hard {
		quantity, err := resource.ParseQuantity(v)
		if err != nil {
			return apperrors.Wrap(err, apperrors.ErrCodeInvalidParams, fmt.Sprintf("无效的资源值: %s=%s", k, v))
		}
		hard[corev1.ResourceName(k)] = quantity
	}

	quota := &corev1.ResourceQuota{
		ObjectMeta: metav1.ObjectMeta{
			Name:      req.Name,
			Namespace: req.Namespace,
		},
		Spec: corev1.ResourceQuotaSpec{
			Hard: hard,
		},
	}

	_, err = client.CoreV1().ResourceQuotas(req.Namespace).Create(ctx, quota, metav1.CreateOptions{})
	if err != nil {
		return apperrors.Wrap(err, apperrors.ErrCodeInternalError, "创建资源配额失败")
	}
	return nil
}

// DeleteResourceQuota 删除资源配额
func (s *K8sHPAService) DeleteResourceQuota(ctx context.Context, clusterID uint, namespace, name string) error {
	client, err := s.clientMgr.GetClient(ctx, clusterID)
	if err != nil {
		return err
	}

	err = client.CoreV1().ResourceQuotas(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil {
		return apperrors.Wrap(err, apperrors.ErrCodeInternalError, "删除资源配额失败")
	}
	return nil
}

// ListLimitRanges 获取 LimitRange 列表
func (s *K8sHPAService) ListLimitRanges(ctx context.Context, clusterID uint, namespace string) ([]dto.K8sLimitRange, error) {
	client, err := s.clientMgr.GetClient(ctx, clusterID)
	if err != nil {
		return nil, err
	}

	lrList, err := client.CoreV1().LimitRanges(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, apperrors.Wrap(err, apperrors.ErrCodeInternalError, "获取LimitRange失败")
	}

	result := make([]dto.K8sLimitRange, len(lrList.Items))
	for i, lr := range lrList.Items {
		limits := make([]dto.LimitRangeItem, len(lr.Spec.Limits))
		for j, limit := range lr.Spec.Limits {
			item := dto.LimitRangeItem{Type: string(limit.Type)}
			if limit.Default != nil {
				item.Default = resourceListToMap(limit.Default)
			}
			if limit.DefaultRequest != nil {
				item.DefaultRequest = resourceListToMap(limit.DefaultRequest)
			}
			if limit.Max != nil {
				item.Max = resourceListToMap(limit.Max)
			}
			if limit.Min != nil {
				item.Min = resourceListToMap(limit.Min)
			}
			limits[j] = item
		}
		result[i] = dto.K8sLimitRange{
			Name:      lr.Name,
			Namespace: lr.Namespace,
			Limits:    limits,
			CreatedAt: lr.CreationTimestamp.Format("2006-01-02 15:04:05"),
		}
	}
	return result, nil
}

func resourceListToMap(rl corev1.ResourceList) map[string]string {
	m := make(map[string]string)
	for k, v := range rl {
		m[string(k)] = v.String()
	}
	return m
}
