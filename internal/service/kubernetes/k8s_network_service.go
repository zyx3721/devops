package kubernetes

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"devops/pkg/dto"
	apperrors "devops/pkg/errors"
)

// K8sNetworkService 网络资源服务
type K8sNetworkService struct {
	clientMgr *K8sClientManager
}

// NewK8sNetworkService 创建网络资源服务
func NewK8sNetworkService(clientMgr *K8sClientManager) *K8sNetworkService {
	return &K8sNetworkService{clientMgr: clientMgr}
}

// GetServices 获取 Service 列表
func (s *K8sNetworkService) GetServices(ctx context.Context, clusterID uint, namespace string) ([]dto.K8sService, error) {
	client, err := s.clientMgr.GetClient(ctx, clusterID)
	if err != nil {
		return nil, err
	}

	svcList, err := client.CoreV1().Services(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, apperrors.Wrap(err, apperrors.ErrCodeInternalError, "获取Service失败")
	}

	result := make([]dto.K8sService, len(svcList.Items))
	for i, svc := range svcList.Items {
		ports := make([]dto.K8sServicePort, len(svc.Spec.Ports))
		for j, p := range svc.Spec.Ports {
			ports[j] = dto.K8sServicePort{
				Name:       p.Name,
				Port:       p.Port,
				TargetPort: p.TargetPort.String(),
				Protocol:   string(p.Protocol),
				NodePort:   p.NodePort,
			}
		}
		result[i] = dto.K8sService{
			Name:      svc.Name,
			Namespace: svc.Namespace,
			Type:      string(svc.Spec.Type),
			ClusterIP: svc.Spec.ClusterIP,
			Ports:     ports,
			Selector:  svc.Spec.Selector,
			CreatedAt: svc.CreationTimestamp.Format("2006-01-02 15:04:05"),
		}
	}
	return result, nil
}

// GetIngresses 获取 Ingress 列表
func (s *K8sNetworkService) GetIngresses(ctx context.Context, clusterID uint, namespace string) ([]dto.K8sIngress, error) {
	client, err := s.clientMgr.GetClient(ctx, clusterID)
	if err != nil {
		return nil, err
	}

	ingList, err := client.NetworkingV1().Ingresses(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, apperrors.Wrap(err, apperrors.ErrCodeInternalError, "获取Ingress失败")
	}

	result := make([]dto.K8sIngress, len(ingList.Items))
	for i, ing := range ingList.Items {
		hosts := []string{}
		rules := []dto.IngressRule{}

		for _, rule := range ing.Spec.Rules {
			if rule.Host != "" {
				hosts = append(hosts, rule.Host)
			}

			paths := []dto.IngressPath{}
			if rule.HTTP != nil {
				for _, path := range rule.HTTP.Paths {
					pathType := "Prefix"
					if path.PathType != nil {
						pathType = string(*path.PathType)
					}

					serviceName := ""
					var servicePort int32
					if path.Backend.Service != nil {
						serviceName = path.Backend.Service.Name
						if path.Backend.Service.Port.Number != 0 {
							servicePort = path.Backend.Service.Port.Number
						}
					}

					paths = append(paths, dto.IngressPath{
						Path:        path.Path,
						PathType:    pathType,
						ServiceName: serviceName,
						ServicePort: servicePort,
					})
				}
			}

			rules = append(rules, dto.IngressRule{
				Host:  rule.Host,
				Paths: paths,
			})
		}

		className := ""
		if ing.Spec.IngressClassName != nil {
			className = *ing.Spec.IngressClassName
		}

		result[i] = dto.K8sIngress{
			Name:      ing.Name,
			Namespace: ing.Namespace,
			ClassName: className,
			Hosts:     hosts,
			Rules:     rules,
			CreatedAt: ing.CreationTimestamp.Format("2006-01-02 15:04:05"),
		}
	}
	return result, nil
}

// GetEndpoints 获取 Endpoints 列表
func (s *K8sNetworkService) GetEndpoints(ctx context.Context, clusterID uint, namespace string) ([]dto.K8sEndpoint, error) {
	client, err := s.clientMgr.GetClient(ctx, clusterID)
	if err != nil {
		return nil, err
	}

	epList, err := client.CoreV1().Endpoints(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, apperrors.Wrap(err, apperrors.ErrCodeInternalError, "获取Endpoints失败")
	}

	result := make([]dto.K8sEndpoint, len(epList.Items))
	for i, ep := range epList.Items {
		addresses := []string{}
		ports := []string{}
		for _, subset := range ep.Subsets {
			for _, addr := range subset.Addresses {
				addresses = append(addresses, addr.IP)
			}
			for _, port := range subset.Ports {
				ports = append(ports, port.Name)
			}
		}
		result[i] = dto.K8sEndpoint{
			Name:      ep.Name,
			Namespace: ep.Namespace,
			Addresses: addresses,
			Ports:     ports,
			CreatedAt: ep.CreationTimestamp.Format("2006-01-02 15:04:05"),
		}
	}
	return result, nil
}

// GetServicePods 获取 Service 后端 Pods
func (s *K8sNetworkService) GetServicePods(ctx context.Context, clusterID uint, namespace, serviceName string) ([]dto.K8sPod, error) {
	client, err := s.clientMgr.GetClient(ctx, clusterID)
	if err != nil {
		return nil, err
	}

	// 获取 Service
	svc, err := client.CoreV1().Services(namespace).Get(ctx, serviceName, metav1.GetOptions{})
	if err != nil {
		return nil, apperrors.Wrap(err, apperrors.ErrCodeInternalError, "获取Service失败")
	}

	// 如果没有 selector，返回空列表
	if len(svc.Spec.Selector) == 0 {
		return []dto.K8sPod{}, nil
	}

	// 构建 label selector
	var labelSelector string
	for k, v := range svc.Spec.Selector {
		if labelSelector != "" {
			labelSelector += ","
		}
		labelSelector += k + "=" + v
	}

	// 获取匹配的 Pods
	podList, err := client.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		return nil, apperrors.Wrap(err, apperrors.ErrCodeInternalError, "获取Pod列表失败")
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
