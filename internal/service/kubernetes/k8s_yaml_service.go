package kubernetes

import (
	"context"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"

	apperrors "devops/pkg/errors"
)

// K8sYAMLService YAML 操作服务
type K8sYAMLService struct {
	clientMgr *K8sClientManager
}

// NewK8sYAMLService 创建 YAML 操作服务
func NewK8sYAMLService(clientMgr *K8sClientManager) *K8sYAMLService {
	return &K8sYAMLService{clientMgr: clientMgr}
}

// GetResourceYAML 获取资源的 YAML
func (s *K8sYAMLService) GetResourceYAML(ctx context.Context, clusterID uint, resourceType, namespace, name string) (string, error) {
	client, err := s.clientMgr.GetClient(ctx, clusterID)
	if err != nil {
		return "", err
	}

	var obj interface{}
	switch strings.ToLower(resourceType) {
	case "deployment", "deployments":
		obj, err = client.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
	case "statefulset", "statefulsets":
		obj, err = client.AppsV1().StatefulSets(namespace).Get(ctx, name, metav1.GetOptions{})
	case "daemonset", "daemonsets":
		obj, err = client.AppsV1().DaemonSets(namespace).Get(ctx, name, metav1.GetOptions{})
	case "service", "services":
		obj, err = client.CoreV1().Services(namespace).Get(ctx, name, metav1.GetOptions{})
	case "configmap", "configmaps":
		obj, err = client.CoreV1().ConfigMaps(namespace).Get(ctx, name, metav1.GetOptions{})
	case "secret", "secrets":
		obj, err = client.CoreV1().Secrets(namespace).Get(ctx, name, metav1.GetOptions{})
	case "ingress", "ingresses":
		obj, err = client.NetworkingV1().Ingresses(namespace).Get(ctx, name, metav1.GetOptions{})
	case "job", "jobs":
		obj, err = client.BatchV1().Jobs(namespace).Get(ctx, name, metav1.GetOptions{})
	case "cronjob", "cronjobs":
		obj, err = client.BatchV1().CronJobs(namespace).Get(ctx, name, metav1.GetOptions{})
	case "pvc", "pvcs", "persistentvolumeclaim", "persistentvolumeclaims":
		obj, err = client.CoreV1().PersistentVolumeClaims(namespace).Get(ctx, name, metav1.GetOptions{})
	case "pv", "pvs", "persistentvolume", "persistentvolumes":
		obj, err = client.CoreV1().PersistentVolumes().Get(ctx, name, metav1.GetOptions{})
	case "namespace", "namespaces":
		obj, err = client.CoreV1().Namespaces().Get(ctx, name, metav1.GetOptions{})
	case "node", "nodes":
		obj, err = client.CoreV1().Nodes().Get(ctx, name, metav1.GetOptions{})
	case "pod", "pods":
		obj, err = client.CoreV1().Pods(namespace).Get(ctx, name, metav1.GetOptions{})
	case "storageclass", "storageclasses":
		obj, err = client.StorageV1().StorageClasses().Get(ctx, name, metav1.GetOptions{})
	case "serviceaccount", "serviceaccounts":
		obj, err = client.CoreV1().ServiceAccounts(namespace).Get(ctx, name, metav1.GetOptions{})
	case "endpoints":
		obj, err = client.CoreV1().Endpoints(namespace).Get(ctx, name, metav1.GetOptions{})
	default:
		return "", apperrors.New(apperrors.ErrCodeInvalidParams, "不支持的资源类型: "+resourceType)
	}

	if err != nil {
		return "", apperrors.Wrap(err, apperrors.ErrCodeInternalError, "获取资源失败")
	}

	s.cleanObjectMeta(obj)

	yamlBytes, err := yaml.Marshal(obj)
	if err != nil {
		return "", apperrors.Wrap(err, apperrors.ErrCodeInternalError, "序列化YAML失败")
	}

	return string(yamlBytes), nil
}

// ApplyResourceYAML 应用 YAML 创建或更新资源
func (s *K8sYAMLService) ApplyResourceYAML(ctx context.Context, clusterID uint, yamlContent string) error {
	return s.ApplyYAMLSimple(ctx, clusterID, yamlContent)
}

// DeleteResource 删除资源
func (s *K8sYAMLService) DeleteResource(ctx context.Context, clusterID uint, resourceType, namespace, name string) error {
	client, err := s.clientMgr.GetClient(ctx, clusterID)
	if err != nil {
		return err
	}

	switch strings.ToLower(resourceType) {
	case "deployment", "deployments":
		err = client.AppsV1().Deployments(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	case "statefulset", "statefulsets":
		err = client.AppsV1().StatefulSets(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	case "daemonset", "daemonsets":
		err = client.AppsV1().DaemonSets(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	case "service", "services":
		err = client.CoreV1().Services(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	case "configmap", "configmaps":
		err = client.CoreV1().ConfigMaps(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	case "secret", "secrets":
		err = client.CoreV1().Secrets(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	case "ingress", "ingresses":
		err = client.NetworkingV1().Ingresses(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	case "job", "jobs":
		propagation := metav1.DeletePropagationBackground
		err = client.BatchV1().Jobs(namespace).Delete(ctx, name, metav1.DeleteOptions{PropagationPolicy: &propagation})
	case "cronjob", "cronjobs":
		err = client.BatchV1().CronJobs(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	case "pvc", "pvcs", "persistentvolumeclaim", "persistentvolumeclaims":
		err = client.CoreV1().PersistentVolumeClaims(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	case "pv", "pvs", "persistentvolume", "persistentvolumes":
		err = client.CoreV1().PersistentVolumes().Delete(ctx, name, metav1.DeleteOptions{})
	case "namespace", "namespaces":
		err = client.CoreV1().Namespaces().Delete(ctx, name, metav1.DeleteOptions{})
	case "node", "nodes":
		err = client.CoreV1().Nodes().Delete(ctx, name, metav1.DeleteOptions{})
	case "pod", "pods":
		err = client.CoreV1().Pods(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	case "storageclass", "storageclasses":
		err = client.StorageV1().StorageClasses().Delete(ctx, name, metav1.DeleteOptions{})
	case "serviceaccount", "serviceaccounts":
		err = client.CoreV1().ServiceAccounts(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	case "endpoints":
		err = client.CoreV1().Endpoints(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	default:
		return apperrors.New(apperrors.ErrCodeInvalidParams, "不支持的资源类型: "+resourceType)
	}

	if err != nil {
		return apperrors.Wrap(err, apperrors.ErrCodeInternalError, "删除资源失败")
	}
	return nil
}

// cleanObjectMeta 清理对象的元数据并设置 TypeMeta
func (s *K8sYAMLService) cleanObjectMeta(obj interface{}) {
	switch o := obj.(type) {
	case *appsv1.Deployment:
		o.TypeMeta = metav1.TypeMeta{APIVersion: "apps/v1", Kind: "Deployment"}
		o.ManagedFields = nil
		o.ResourceVersion = ""
		o.UID = ""
		o.Generation = 0
		o.CreationTimestamp = metav1.Time{}
		o.Status = appsv1.DeploymentStatus{}
	case *appsv1.StatefulSet:
		o.TypeMeta = metav1.TypeMeta{APIVersion: "apps/v1", Kind: "StatefulSet"}
		o.ManagedFields = nil
		o.ResourceVersion = ""
		o.UID = ""
		o.Generation = 0
		o.CreationTimestamp = metav1.Time{}
		o.Status = appsv1.StatefulSetStatus{}
	case *appsv1.DaemonSet:
		o.TypeMeta = metav1.TypeMeta{APIVersion: "apps/v1", Kind: "DaemonSet"}
		o.ManagedFields = nil
		o.ResourceVersion = ""
		o.UID = ""
		o.Generation = 0
		o.CreationTimestamp = metav1.Time{}
		o.Status = appsv1.DaemonSetStatus{}
	case *corev1.Service:
		o.TypeMeta = metav1.TypeMeta{APIVersion: "v1", Kind: "Service"}
		o.ManagedFields = nil
		o.ResourceVersion = ""
		o.UID = ""
		o.CreationTimestamp = metav1.Time{}
		o.Status = corev1.ServiceStatus{}
	case *corev1.ConfigMap:
		o.TypeMeta = metav1.TypeMeta{APIVersion: "v1", Kind: "ConfigMap"}
		o.ManagedFields = nil
		o.ResourceVersion = ""
		o.UID = ""
		o.CreationTimestamp = metav1.Time{}
	case *corev1.Secret:
		o.TypeMeta = metav1.TypeMeta{APIVersion: "v1", Kind: "Secret"}
		o.ManagedFields = nil
		o.ResourceVersion = ""
		o.UID = ""
		o.CreationTimestamp = metav1.Time{}
	case *networkingv1.Ingress:
		o.TypeMeta = metav1.TypeMeta{APIVersion: "networking.k8s.io/v1", Kind: "Ingress"}
		o.ManagedFields = nil
		o.ResourceVersion = ""
		o.UID = ""
		o.Generation = 0
		o.CreationTimestamp = metav1.Time{}
		o.Status = networkingv1.IngressStatus{}
	case *batchv1.Job:
		o.TypeMeta = metav1.TypeMeta{APIVersion: "batch/v1", Kind: "Job"}
		o.ManagedFields = nil
		o.ResourceVersion = ""
		o.UID = ""
		o.CreationTimestamp = metav1.Time{}
		o.Status = batchv1.JobStatus{}
	case *batchv1.CronJob:
		o.TypeMeta = metav1.TypeMeta{APIVersion: "batch/v1", Kind: "CronJob"}
		o.ManagedFields = nil
		o.ResourceVersion = ""
		o.UID = ""
		o.Generation = 0
		o.CreationTimestamp = metav1.Time{}
		o.Status = batchv1.CronJobStatus{}
	case *corev1.PersistentVolumeClaim:
		o.TypeMeta = metav1.TypeMeta{APIVersion: "v1", Kind: "PersistentVolumeClaim"}
		o.ManagedFields = nil
		o.ResourceVersion = ""
		o.UID = ""
		o.CreationTimestamp = metav1.Time{}
		o.Status = corev1.PersistentVolumeClaimStatus{}
	case *corev1.PersistentVolume:
		o.TypeMeta = metav1.TypeMeta{APIVersion: "v1", Kind: "PersistentVolume"}
		o.ManagedFields = nil
		o.ResourceVersion = ""
		o.UID = ""
		o.CreationTimestamp = metav1.Time{}
		o.Status = corev1.PersistentVolumeStatus{}
	case *corev1.Namespace:
		o.TypeMeta = metav1.TypeMeta{APIVersion: "v1", Kind: "Namespace"}
		o.ManagedFields = nil
		o.ResourceVersion = ""
		o.UID = ""
		o.CreationTimestamp = metav1.Time{}
		o.Status = corev1.NamespaceStatus{}
	case *corev1.Node:
		o.TypeMeta = metav1.TypeMeta{APIVersion: "v1", Kind: "Node"}
		o.ManagedFields = nil
		o.ResourceVersion = ""
		o.UID = ""
		o.CreationTimestamp = metav1.Time{}
		o.Status = corev1.NodeStatus{}
	case *corev1.Pod:
		o.TypeMeta = metav1.TypeMeta{APIVersion: "v1", Kind: "Pod"}
		o.ManagedFields = nil
		o.ResourceVersion = ""
		o.UID = ""
		o.CreationTimestamp = metav1.Time{}
		o.Status = corev1.PodStatus{}
	case *storagev1.StorageClass:
		o.TypeMeta = metav1.TypeMeta{APIVersion: "storage.k8s.io/v1", Kind: "StorageClass"}
		o.ManagedFields = nil
		o.ResourceVersion = ""
		o.UID = ""
		o.CreationTimestamp = metav1.Time{}
	case *corev1.ServiceAccount:
		o.TypeMeta = metav1.TypeMeta{APIVersion: "v1", Kind: "ServiceAccount"}
		o.ManagedFields = nil
		o.ResourceVersion = ""
		o.UID = ""
		o.CreationTimestamp = metav1.Time{}
	case *corev1.Endpoints:
		o.TypeMeta = metav1.TypeMeta{APIVersion: "v1", Kind: "Endpoints"}
		o.ManagedFields = nil
		o.ResourceVersion = ""
		o.UID = ""
		o.CreationTimestamp = metav1.Time{}
	}
}

// GetResourceDetail 获取资源详情
func (s *K8sYAMLService) GetResourceDetail(ctx context.Context, clusterID uint, resourceType, namespace, name string) (interface{}, error) {
	client, err := s.clientMgr.GetClient(ctx, clusterID)
	if err != nil {
		return nil, err
	}

	switch strings.ToLower(resourceType) {
	case "deployment", "deployments":
		deploy, err := client.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return nil, apperrors.Wrap(err, apperrors.ErrCodeInternalError, "获取Deployment失败")
		}
		return map[string]interface{}{
			"name":        deploy.Name,
			"namespace":   deploy.Namespace,
			"uid":         string(deploy.UID),
			"labels":      deploy.Labels,
			"annotations": deploy.Annotations,
			"replicas":    deploy.Spec.Replicas,
			"ready":       deploy.Status.ReadyReplicas,
			"strategy":    string(deploy.Spec.Strategy.Type),
			"containers":  s.extractContainers(deploy.Spec.Template.Spec.Containers),
			"created_at":  deploy.CreationTimestamp.Format("2006-01-02 15:04:05"),
			"spec": map[string]interface{}{
				"template": map[string]interface{}{
					"metadata": map[string]interface{}{
						"labels": deploy.Spec.Template.Labels,
					},
				},
			},
		}, nil

	case "statefulset", "statefulsets":
		sts, err := client.AppsV1().StatefulSets(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return nil, apperrors.Wrap(err, apperrors.ErrCodeInternalError, "获取StatefulSet失败")
		}
		return map[string]interface{}{
			"name":        sts.Name,
			"namespace":   sts.Namespace,
			"uid":         string(sts.UID),
			"labels":      sts.Labels,
			"annotations": sts.Annotations,
			"replicas":    sts.Spec.Replicas,
			"ready":       sts.Status.ReadyReplicas,
			"containers":  s.extractContainers(sts.Spec.Template.Spec.Containers),
			"created_at":  sts.CreationTimestamp.Format("2006-01-02 15:04:05"),
			"spec": map[string]interface{}{
				"template": map[string]interface{}{
					"metadata": map[string]interface{}{
						"labels": sts.Spec.Template.Labels,
					},
				},
			},
		}, nil

	case "daemonset", "daemonsets":
		ds, err := client.AppsV1().DaemonSets(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return nil, apperrors.Wrap(err, apperrors.ErrCodeInternalError, "获取DaemonSet失败")
		}
		return map[string]interface{}{
			"name":        ds.Name,
			"namespace":   ds.Namespace,
			"uid":         string(ds.UID),
			"labels":      ds.Labels,
			"annotations": ds.Annotations,
			"desired":     ds.Status.DesiredNumberScheduled,
			"ready":       ds.Status.NumberReady,
			"containers":  s.extractContainers(ds.Spec.Template.Spec.Containers),
			"created_at":  ds.CreationTimestamp.Format("2006-01-02 15:04:05"),
			"spec": map[string]interface{}{
				"template": map[string]interface{}{
					"metadata": map[string]interface{}{
						"labels": ds.Spec.Template.Labels,
					},
				},
			},
		}, nil

	case "pod", "pods":
		pod, err := client.CoreV1().Pods(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return nil, apperrors.Wrap(err, apperrors.ErrCodeInternalError, "获取Pod失败")
		}
		return map[string]interface{}{
			"name":             pod.Name,
			"namespace":        pod.Namespace,
			"uid":              string(pod.UID),
			"labels":           pod.Labels,
			"annotations":      pod.Annotations,
			"status":           string(pod.Status.Phase),
			"ip":               pod.Status.PodIP,
			"node":             pod.Spec.NodeName,
			"restarts":         s.getPodRestarts(pod),
			"containers":       s.extractContainers(pod.Spec.Containers),
			"owner_references": s.extractOwnerReferences(pod.OwnerReferences),
			"created_at":       pod.CreationTimestamp.Format("2006-01-02 15:04:05"),
		}, nil

	case "service", "services":
		svc, err := client.CoreV1().Services(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return nil, apperrors.Wrap(err, apperrors.ErrCodeInternalError, "获取Service失败")
		}
		return map[string]interface{}{
			"name":        svc.Name,
			"namespace":   svc.Namespace,
			"uid":         string(svc.UID),
			"labels":      svc.Labels,
			"annotations": svc.Annotations,
			"type":        string(svc.Spec.Type),
			"cluster_ip":  svc.Spec.ClusterIP,
			"ports":       s.extractServicePorts(svc.Spec.Ports),
			"selector":    svc.Spec.Selector,
			"created_at":  svc.CreationTimestamp.Format("2006-01-02 15:04:05"),
		}, nil

	case "node", "nodes":
		node, err := client.CoreV1().Nodes().Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return nil, apperrors.Wrap(err, apperrors.ErrCodeInternalError, "获取Node失败")
		}
		status := "Unknown"
		for _, cond := range node.Status.Conditions {
			if cond.Type == corev1.NodeReady {
				if cond.Status == corev1.ConditionTrue {
					status = "Ready"
				} else {
					status = "NotReady"
				}
				break
			}
		}
		internalIP := ""
		for _, addr := range node.Status.Addresses {
			if addr.Type == corev1.NodeInternalIP {
				internalIP = addr.Address
				break
			}
		}
		return map[string]interface{}{
			"name":               node.Name,
			"uid":                string(node.UID),
			"labels":             node.Labels,
			"annotations":        node.Annotations,
			"status":             status,
			"schedulable":        !node.Spec.Unschedulable,
			"internal_ip":        internalIP,
			"cpu_capacity":       node.Status.Capacity.Cpu().String(),
			"memory_capacity":    node.Status.Capacity.Memory().String(),
			"cpu_allocatable":    node.Status.Allocatable.Cpu().String(),
			"memory_allocatable": node.Status.Allocatable.Memory().String(),
			"kubelet_version":    node.Status.NodeInfo.KubeletVersion,
			"created_at":         node.CreationTimestamp.Format("2006-01-02 15:04:05"),
		}, nil

	default:
		// 对于其他类型，返回基本信息
		return map[string]interface{}{
			"name":      name,
			"namespace": namespace,
			"type":      resourceType,
		}, nil
	}
}

func (s *K8sYAMLService) extractContainers(containers []corev1.Container) []map[string]interface{} {
	result := make([]map[string]interface{}, len(containers))
	for i, c := range containers {
		resources := map[string]interface{}{}
		if c.Resources.Requests != nil || c.Resources.Limits != nil {
			if c.Resources.Requests != nil {
				resources["requests"] = map[string]string{
					"cpu":    c.Resources.Requests.Cpu().String(),
					"memory": c.Resources.Requests.Memory().String(),
				}
			}
			if c.Resources.Limits != nil {
				resources["limits"] = map[string]string{
					"cpu":    c.Resources.Limits.Cpu().String(),
					"memory": c.Resources.Limits.Memory().String(),
				}
			}
		}
		result[i] = map[string]interface{}{
			"name":      c.Name,
			"image":     c.Image,
			"resources": resources,
		}
	}
	return result
}

func (s *K8sYAMLService) extractServicePorts(ports []corev1.ServicePort) []map[string]interface{} {
	result := make([]map[string]interface{}, len(ports))
	for i, p := range ports {
		result[i] = map[string]interface{}{
			"name":        p.Name,
			"port":        p.Port,
			"target_port": p.TargetPort.String(),
			"protocol":    string(p.Protocol),
			"node_port":   p.NodePort,
		}
	}
	return result
}

func (s *K8sYAMLService) extractOwnerReferences(refs []metav1.OwnerReference) []map[string]interface{} {
	result := make([]map[string]interface{}, len(refs))
	for i, r := range refs {
		result[i] = map[string]interface{}{
			"kind": r.Kind,
			"name": r.Name,
			"uid":  string(r.UID),
		}
	}
	return result
}

func (s *K8sYAMLService) getPodRestarts(pod *corev1.Pod) int32 {
	var restarts int32
	for _, cs := range pod.Status.ContainerStatuses {
		restarts += cs.RestartCount
	}
	return restarts
}
