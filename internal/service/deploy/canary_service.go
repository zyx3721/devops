package deploy

import (
	"context"
	"encoding/json"
	"strconv"
	"time"

	"gorm.io/gorm"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"devops/internal/models"
	"devops/internal/service/kubernetes"
	"devops/pkg/dto"
	apperrors "devops/pkg/errors"
)

// CanaryService 灰度发布服务 (基于 Nginx Ingress Canary)
type CanaryService struct {
	db        *gorm.DB
	clientMgr *kubernetes.K8sClientManager
}

// NewCanaryService 创建灰度发布服务
func NewCanaryService(db *gorm.DB, clientMgr *kubernetes.K8sClientManager) *CanaryService {
	return &CanaryService{db: db, clientMgr: clientMgr}
}

// CanaryMetadata 灰度发布元数据
type CanaryMetadata struct {
	CanaryPercent   int    `json:"canary_percent"`
	CanaryHeader    string `json:"canary_header,omitempty"`
	CanaryHeaderVal string `json:"canary_header_value,omitempty"`
	CanaryCookie    string `json:"canary_cookie,omitempty"`
	StableImage     string `json:"stable_image"`
	CanaryImage     string `json:"canary_image"`
	StartedAt       string `json:"started_at"`
	IngressName     string `json:"ingress_name"`
	ServiceName     string `json:"service_name"`
}

// StartCanary 开始灰度发布 (Nginx Ingress Canary)
// 流程:
// 1. 创建灰度 Deployment (app-canary)
// 2. 创建灰度 Service (app-canary)
// 3. 创建灰度 Ingress (app-canary) 带 canary annotations
func (s *CanaryService) StartCanary(ctx context.Context, req *dto.CanaryDeployRequest) (*dto.CanaryStatus, error) {
	// 获取应用信息
	var app models.Application
	if err := s.db.First(&app, req.ApplicationID).Error; err != nil {
		return nil, apperrors.Wrap(err, apperrors.ErrCodeNotFound, "应用不存在")
	}

	if app.K8sClusterID == nil {
		return nil, apperrors.New(apperrors.ErrCodeInvalidParams, "应用未配置K8s集群")
	}

	client, err := s.clientMgr.GetClient(ctx, *app.K8sClusterID)
	if err != nil {
		return nil, err
	}

	namespace := app.K8sNamespace
	deployName := app.K8sDeployment
	canaryDeployName := deployName + "-canary"
	canaryServiceName := deployName + "-canary"
	canaryIngressName := deployName + "-canary"

	// 获取原 Deployment
	stableDeploy, err := client.AppsV1().Deployments(namespace).Get(ctx, deployName, metav1.GetOptions{})
	if err != nil {
		return nil, apperrors.Wrap(err, apperrors.ErrCodeNotFound, "Deployment不存在")
	}

	// 获取原镜像
	stableImage := ""
	if len(stableDeploy.Spec.Template.Spec.Containers) > 0 {
		stableImage = stableDeploy.Spec.Template.Spec.Containers[0].Image
	}

	// 获取原 Service
	stableService, err := client.CoreV1().Services(namespace).Get(ctx, deployName, metav1.GetOptions{})
	if err != nil {
		// 尝试用应用名查找
		stableService, err = client.CoreV1().Services(namespace).Get(ctx, app.Name, metav1.GetOptions{})
		if err != nil {
			return nil, apperrors.Wrap(err, apperrors.ErrCodeNotFound, "Service不存在，请确保应用有对应的Service")
		}
	}

	// 获取原 Ingress
	stableIngress, err := client.NetworkingV1().Ingresses(namespace).Get(ctx, deployName, metav1.GetOptions{})
	if err != nil {
		stableIngress, err = client.NetworkingV1().Ingresses(namespace).Get(ctx, app.Name, metav1.GetOptions{})
		if err != nil {
			return nil, apperrors.Wrap(err, apperrors.ErrCodeNotFound, "Ingress不存在，Nginx Ingress Canary需要原Ingress")
		}
	}

	// 计算灰度副本数 (至少1个)
	canaryReplicas := int32(1)
	if req.CanaryReplicas > 0 {
		canaryReplicas = int32(req.CanaryReplicas)
	}

	// 创建部署记录
	metadata := CanaryMetadata{
		CanaryPercent:   req.CanaryPercent,
		CanaryHeader:    req.CanaryHeader,
		CanaryHeaderVal: req.CanaryHeaderValue,
		CanaryCookie:    req.CanaryCookie,
		StableImage:     stableImage,
		CanaryImage:     req.ImageTag,
		StartedAt:       time.Now().Format("2006-01-02 15:04:05"),
		IngressName:     stableIngress.Name,
		ServiceName:     stableService.Name,
	}
	metadataJSON, _ := json.Marshal(metadata)

	record := &models.DeployRecord{
		ApplicationID: req.ApplicationID,
		AppName:       app.Name,
		EnvName:       req.EnvName,
		ImageTag:      req.ImageTag,
		DeployType:    "canary",
		Status:        "canary_running",
		Description:   string(metadataJSON),
	}

	if err := s.db.Create(record).Error; err != nil {
		return nil, apperrors.Wrap(err, apperrors.ErrCodeInternalError, "创建部署记录失败")
	}

	// 1. 创建灰度 Deployment
	canaryDeploy := s.buildCanaryDeployment(stableDeploy, canaryDeployName, req.ImageTag, canaryReplicas)
	_, err = client.AppsV1().Deployments(namespace).Create(ctx, canaryDeploy, metav1.CreateOptions{})
	if err != nil && !errors.IsAlreadyExists(err) {
		s.updateRecordFailed(record, err.Error())
		return nil, apperrors.Wrap(err, apperrors.ErrCodeInternalError, "创建灰度Deployment失败")
	}

	// 2. 创建灰度 Service
	canaryService := s.buildCanaryService(stableService, canaryServiceName, canaryDeployName)
	_, err = client.CoreV1().Services(namespace).Create(ctx, canaryService, metav1.CreateOptions{})
	if err != nil && !errors.IsAlreadyExists(err) {
		// 回滚: 删除已创建的 Deployment
		client.AppsV1().Deployments(namespace).Delete(ctx, canaryDeployName, metav1.DeleteOptions{})
		s.updateRecordFailed(record, err.Error())
		return nil, apperrors.Wrap(err, apperrors.ErrCodeInternalError, "创建灰度Service失败")
	}

	// 3. 创建灰度 Ingress (带 Canary annotations)
	canaryIngress := s.buildCanaryIngress(stableIngress, canaryIngressName, canaryServiceName, req)
	_, err = client.NetworkingV1().Ingresses(namespace).Create(ctx, canaryIngress, metav1.CreateOptions{})
	if err != nil && !errors.IsAlreadyExists(err) {
		// 回滚
		client.CoreV1().Services(namespace).Delete(ctx, canaryServiceName, metav1.DeleteOptions{})
		client.AppsV1().Deployments(namespace).Delete(ctx, canaryDeployName, metav1.DeleteOptions{})
		s.updateRecordFailed(record, err.Error())
		return nil, apperrors.Wrap(err, apperrors.ErrCodeInternalError, "创建灰度Ingress失败")
	}

	return &dto.CanaryStatus{
		DeployRecordID: record.ID,
		Status:         "canary_running",
		CanaryReplicas: canaryReplicas,
		StableReplicas: *stableDeploy.Spec.Replicas,
		CanaryImage:    req.ImageTag,
		StableImage:    stableImage,
		StartedAt:      metadata.StartedAt,
		TrafficPercent: req.CanaryPercent,
	}, nil
}

// buildCanaryDeployment 构建灰度 Deployment
func (s *CanaryService) buildCanaryDeployment(stable *appsv1.Deployment, name, image string, replicas int32) *appsv1.Deployment {
	canary := stable.DeepCopy()
	canary.Name = name
	canary.ResourceVersion = ""
	canary.UID = ""
	canary.Spec.Replicas = &replicas

	// 更新标签
	canaryLabels := map[string]string{
		"app":     stable.Labels["app"],
		"version": "canary",
	}
	canary.Labels = canaryLabels
	canary.Spec.Selector = &metav1.LabelSelector{
		MatchLabels: map[string]string{
			"app":     stable.Labels["app"],
			"version": "canary",
		},
	}
	canary.Spec.Template.Labels = map[string]string{
		"app":     stable.Labels["app"],
		"version": "canary",
	}

	// 更新镜像
	if len(canary.Spec.Template.Spec.Containers) > 0 {
		canary.Spec.Template.Spec.Containers[0].Image = image
	}

	return canary
}

// buildCanaryService 构建灰度 Service
func (s *CanaryService) buildCanaryService(stable *corev1.Service, name, deployName string) *corev1.Service {
	canary := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: stable.Namespace,
			Labels: map[string]string{
				"app":     stable.Labels["app"],
				"version": "canary",
			},
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"app":     stable.Labels["app"],
				"version": "canary",
			},
			Ports: stable.Spec.Ports,
			Type:  stable.Spec.Type,
		},
	}
	return canary
}

// buildCanaryIngress 构建灰度 Ingress (带 Nginx Canary annotations)
func (s *CanaryService) buildCanaryIngress(stable *networkingv1.Ingress, name, serviceName string, req *dto.CanaryDeployRequest) *networkingv1.Ingress {
	annotations := map[string]string{
		"nginx.ingress.kubernetes.io/canary":        "true",
		"nginx.ingress.kubernetes.io/canary-weight": strconv.Itoa(req.CanaryPercent),
	}

	// 基于 Header 的灰度
	if req.CanaryHeader != "" {
		annotations["nginx.ingress.kubernetes.io/canary-by-header"] = req.CanaryHeader
		if req.CanaryHeaderValue != "" {
			annotations["nginx.ingress.kubernetes.io/canary-by-header-value"] = req.CanaryHeaderValue
		}
	}

	// 基于 Cookie 的灰度
	if req.CanaryCookie != "" {
		annotations["nginx.ingress.kubernetes.io/canary-by-cookie"] = req.CanaryCookie
	}

	canary := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   stable.Namespace,
			Annotations: annotations,
			Labels: map[string]string{
				"app":     stable.Labels["app"],
				"version": "canary",
			},
		},
		Spec: *stable.Spec.DeepCopy(),
	}

	// 更新后端 Service 为灰度 Service
	pathType := networkingv1.PathTypePrefix
	for i := range canary.Spec.Rules {
		for j := range canary.Spec.Rules[i].HTTP.Paths {
			canary.Spec.Rules[i].HTTP.Paths[j].Backend = networkingv1.IngressBackend{
				Service: &networkingv1.IngressServiceBackend{
					Name: serviceName,
					Port: networkingv1.ServiceBackendPort{
						Number: stable.Spec.Rules[i].HTTP.Paths[j].Backend.Service.Port.Number,
					},
				},
			}
			canary.Spec.Rules[i].HTTP.Paths[j].PathType = &pathType
		}
	}

	return canary
}

// GetCanaryStatus 获取灰度状态
func (s *CanaryService) GetCanaryStatus(ctx context.Context, recordID uint) (*dto.CanaryStatus, error) {
	var record models.DeployRecord
	if err := s.db.First(&record, recordID).Error; err != nil {
		return nil, apperrors.Wrap(err, apperrors.ErrCodeNotFound, "部署记录不存在")
	}

	var app models.Application
	if err := s.db.First(&app, record.ApplicationID).Error; err != nil {
		return nil, apperrors.Wrap(err, apperrors.ErrCodeNotFound, "应用不存在")
	}

	// 解析元数据
	var metadata CanaryMetadata
	json.Unmarshal([]byte(record.Description), &metadata)

	status := &dto.CanaryStatus{
		DeployRecordID: recordID,
		Status:         record.Status,
		CanaryImage:    record.ImageTag,
		StableImage:    metadata.StableImage,
		StartedAt:      metadata.StartedAt,
		TrafficPercent: metadata.CanaryPercent,
		CanaryHeader:   metadata.CanaryHeader,
		CanaryCookie:   metadata.CanaryCookie,
	}

	if app.K8sClusterID == nil {
		return status, nil
	}

	client, err := s.clientMgr.GetClient(ctx, *app.K8sClusterID)
	if err != nil {
		return status, nil
	}

	namespace := app.K8sNamespace
	deployName := app.K8sDeployment
	canaryDeployName := deployName + "-canary"

	// 获取稳定版本状态
	stableDeploy, err := client.AppsV1().Deployments(namespace).Get(ctx, deployName, metav1.GetOptions{})
	if err == nil {
		if stableDeploy.Spec.Replicas != nil {
			status.StableReplicas = *stableDeploy.Spec.Replicas
		}
		status.StableReady = stableDeploy.Status.ReadyReplicas
	}

	// 获取灰度版本状态
	canaryDeploy, err := client.AppsV1().Deployments(namespace).Get(ctx, canaryDeployName, metav1.GetOptions{})
	if err == nil {
		if canaryDeploy.Spec.Replicas != nil {
			status.CanaryReplicas = *canaryDeploy.Spec.Replicas
		}
		status.CanaryReady = canaryDeploy.Status.ReadyReplicas
		status.CanaryHealthy = status.CanaryReady == status.CanaryReplicas && status.CanaryReplicas > 0
	}

	return status, nil
}

// AdjustCanary 调整灰度比例 (更新 Ingress annotation)
func (s *CanaryService) AdjustCanary(ctx context.Context, recordID uint, newPercent int) error {
	var record models.DeployRecord
	if err := s.db.First(&record, recordID).Error; err != nil {
		return apperrors.Wrap(err, apperrors.ErrCodeNotFound, "部署记录不存在")
	}

	if record.Status != "canary_running" {
		return apperrors.New(apperrors.ErrCodeInvalidParams, "当前状态不允许调整灰度比例")
	}

	var app models.Application
	if err := s.db.First(&app, record.ApplicationID).Error; err != nil {
		return apperrors.Wrap(err, apperrors.ErrCodeNotFound, "应用不存在")
	}

	if app.K8sClusterID == nil {
		return apperrors.New(apperrors.ErrCodeInvalidParams, "应用未配置K8s集群")
	}

	client, err := s.clientMgr.GetClient(ctx, *app.K8sClusterID)
	if err != nil {
		return err
	}

	namespace := app.K8sNamespace
	canaryIngressName := app.K8sDeployment + "-canary"

	// 更新灰度 Ingress 的权重
	canaryIngress, err := client.NetworkingV1().Ingresses(namespace).Get(ctx, canaryIngressName, metav1.GetOptions{})
	if err != nil {
		return apperrors.Wrap(err, apperrors.ErrCodeNotFound, "灰度Ingress不存在")
	}

	if canaryIngress.Annotations == nil {
		canaryIngress.Annotations = make(map[string]string)
	}
	canaryIngress.Annotations["nginx.ingress.kubernetes.io/canary-weight"] = strconv.Itoa(newPercent)

	_, err = client.NetworkingV1().Ingresses(namespace).Update(ctx, canaryIngress, metav1.UpdateOptions{})
	if err != nil {
		return apperrors.Wrap(err, apperrors.ErrCodeInternalError, "更新灰度Ingress失败")
	}

	// 更新元数据
	var metadata CanaryMetadata
	if err := json.Unmarshal([]byte(record.Description), &metadata); err == nil {
		metadata.CanaryPercent = newPercent
		metadataJSON, _ := json.Marshal(metadata)
		s.db.Model(&record).Update("description", string(metadataJSON))
	}

	return nil
}

// Promote 全量发布
func (s *CanaryService) Promote(ctx context.Context, recordID uint) error {
	var record models.DeployRecord
	if err := s.db.First(&record, recordID).Error; err != nil {
		return apperrors.Wrap(err, apperrors.ErrCodeNotFound, "部署记录不存在")
	}

	if record.Status != "canary_running" {
		return apperrors.New(apperrors.ErrCodeInvalidParams, "当前状态不允许全量发布")
	}

	var app models.Application
	if err := s.db.First(&app, record.ApplicationID).Error; err != nil {
		return apperrors.Wrap(err, apperrors.ErrCodeNotFound, "应用不存在")
	}

	if app.K8sClusterID == nil {
		return apperrors.New(apperrors.ErrCodeInvalidParams, "应用未配置K8s集群")
	}

	client, err := s.clientMgr.GetClient(ctx, *app.K8sClusterID)
	if err != nil {
		return err
	}

	namespace := app.K8sNamespace
	deployName := app.K8sDeployment
	canaryDeployName := deployName + "-canary"
	canaryServiceName := deployName + "-canary"
	canaryIngressName := deployName + "-canary"

	// 1. 更新稳定版本的镜像
	stableDeploy, err := client.AppsV1().Deployments(namespace).Get(ctx, deployName, metav1.GetOptions{})
	if err != nil {
		return apperrors.Wrap(err, apperrors.ErrCodeNotFound, "Deployment不存在")
	}

	if len(stableDeploy.Spec.Template.Spec.Containers) > 0 {
		stableDeploy.Spec.Template.Spec.Containers[0].Image = record.ImageTag
	}

	_, err = client.AppsV1().Deployments(namespace).Update(ctx, stableDeploy, metav1.UpdateOptions{})
	if err != nil {
		return apperrors.Wrap(err, apperrors.ErrCodeInternalError, "更新Deployment失败")
	}

	// 2. 删除灰度资源 (Ingress -> Service -> Deployment)
	client.NetworkingV1().Ingresses(namespace).Delete(ctx, canaryIngressName, metav1.DeleteOptions{})
	client.CoreV1().Services(namespace).Delete(ctx, canaryServiceName, metav1.DeleteOptions{})
	client.AppsV1().Deployments(namespace).Delete(ctx, canaryDeployName, metav1.DeleteOptions{})

	// 更新记录状态
	now := time.Now()
	s.db.Model(&record).Updates(map[string]interface{}{
		"status":      "success",
		"finished_at": &now,
	})

	return nil
}

// Rollback 回滚灰度
func (s *CanaryService) Rollback(ctx context.Context, recordID uint, reason string) error {
	var record models.DeployRecord
	if err := s.db.First(&record, recordID).Error; err != nil {
		return apperrors.Wrap(err, apperrors.ErrCodeNotFound, "部署记录不存在")
	}

	if record.Status != "canary_running" {
		return apperrors.New(apperrors.ErrCodeInvalidParams, "当前状态不允许回滚")
	}

	var app models.Application
	if err := s.db.First(&app, record.ApplicationID).Error; err != nil {
		return apperrors.Wrap(err, apperrors.ErrCodeNotFound, "应用不存在")
	}

	if app.K8sClusterID == nil {
		return apperrors.New(apperrors.ErrCodeInvalidParams, "应用未配置K8s集群")
	}

	client, err := s.clientMgr.GetClient(ctx, *app.K8sClusterID)
	if err != nil {
		return err
	}

	namespace := app.K8sNamespace
	deployName := app.K8sDeployment
	canaryDeployName := deployName + "-canary"
	canaryServiceName := deployName + "-canary"
	canaryIngressName := deployName + "-canary"

	// 删除灰度资源 (Ingress -> Service -> Deployment)
	client.NetworkingV1().Ingresses(namespace).Delete(ctx, canaryIngressName, metav1.DeleteOptions{})
	client.CoreV1().Services(namespace).Delete(ctx, canaryServiceName, metav1.DeleteOptions{})
	client.AppsV1().Deployments(namespace).Delete(ctx, canaryDeployName, metav1.DeleteOptions{})

	// 更新记录状态
	now := time.Now()
	s.db.Model(&record).Updates(map[string]interface{}{
		"status":        "rolled_back",
		"finished_at":   &now,
		"reject_reason": reason,
	})

	return nil
}

// ListCanary 获取灰度发布列表
func (s *CanaryService) ListCanary(ctx context.Context, page, pageSize int, appID uint, envName, status string) ([]dto.CanaryListItem, int64, error) {
	var total int64
	query := s.db.Model(&models.DeployRecord{}).Where("deploy_type = ?", "canary")

	if appID > 0 {
		query = query.Where("application_id = ?", appID)
	}
	if envName != "" {
		query = query.Where("env_name = ?", envName)
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}

	query.Count(&total)

	var records []models.DeployRecord
	offset := (page - 1) * pageSize
	if err := query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&records).Error; err != nil {
		return nil, 0, apperrors.Wrap(err, apperrors.ErrCodeDBQuery, "查询灰度列表失败")
	}

	result := make([]dto.CanaryListItem, len(records))
	for i, r := range records {
		var metadata CanaryMetadata
		canaryPercent := 10
		if err := json.Unmarshal([]byte(r.Description), &metadata); err == nil {
			canaryPercent = metadata.CanaryPercent
		}

		result[i] = dto.CanaryListItem{
			ID:            r.ID,
			CreatedAt:     r.CreatedAt.Format("2006-01-02 15:04:05"),
			ApplicationID: r.ApplicationID,
			AppName:       r.AppName,
			EnvName:       r.EnvName,
			ImageTag:      r.ImageTag,
			CanaryPercent: canaryPercent,
			Status:        r.Status,
			Operator:      r.Operator,
		}
		if r.FinishedAt != nil {
			result[i].FinishedAt = r.FinishedAt.Format("2006-01-02 15:04:05")
		}
	}

	return result, total, nil
}

// updateRecordFailed 更新记录为失败状态
func (s *CanaryService) updateRecordFailed(record *models.DeployRecord, errMsg string) {
	s.db.Model(record).Updates(map[string]interface{}{
		"status":    "failed",
		"error_msg": errMsg,
	})
}
