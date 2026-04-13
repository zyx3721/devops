package kubernetes

import (
	"context"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	apperrors "devops/pkg/errors"
)

// K8sServiceService Service 资源管理服务
type K8sServiceService struct {
	clientMgr *K8sClientManager
}

// NewK8sServiceService 创建 Service 服务
func NewK8sServiceService(clientMgr *K8sClientManager) *K8sServiceService {
	return &K8sServiceService{clientMgr: clientMgr}
}

// ServiceInfo Service 信息
type ServiceInfo struct {
	Name       string            `json:"name"`
	Namespace  string            `json:"namespace"`
	Type       string            `json:"type"`
	ClusterIP  string            `json:"cluster_ip"`
	ExternalIP string            `json:"external_ip"`
	Ports      []ServicePort     `json:"ports"`
	Age        string            `json:"age"`
	Selector   map[string]string `json:"selector"`
	CreatedAt  string            `json:"created_at"`
}

// ServicePort Service 端口信息
type ServicePort struct {
	Name       string `json:"name"`
	Protocol   string `json:"protocol"`
	Port       int32  `json:"port"`
	TargetPort string `json:"target_port"`
	NodePort   int32  `json:"node_port,omitempty"`
}

// ServiceDetail Service 详情
type ServiceDetail struct {
	ServiceInfo
	Labels      map[string]string `json:"labels"`
	Annotations map[string]string `json:"annotations"`
	Endpoints   []EndpointInfo    `json:"endpoints"`
}

// EndpointInfo Endpoint 信息
type EndpointInfo struct {
	IP       string `json:"ip"`
	NodeName string `json:"node_name"`
	Ready    bool   `json:"ready"`
}

// ListServices 获取 Service 列表
func (s *K8sServiceService) ListServices(ctx context.Context, clusterID uint, namespace string) ([]ServiceInfo, error) {
	client, err := s.clientMgr.GetClient(ctx, clusterID)
	if err != nil {
		return nil, err
	}

	svcList, err := client.CoreV1().Services(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, apperrors.Wrap(err, apperrors.ErrCodeInternalError, "获取Service列表失败")
	}

	result := make([]ServiceInfo, len(svcList.Items))
	for i, svc := range svcList.Items {
		result[i] = s.convertServiceInfo(&svc)
	}
	return result, nil
}

// GetService 获取 Service 详情
func (s *K8sServiceService) GetService(ctx context.Context, clusterID uint, namespace, name string) (*ServiceDetail, error) {
	client, err := s.clientMgr.GetClient(ctx, clusterID)
	if err != nil {
		return nil, err
	}

	svc, err := client.CoreV1().Services(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, apperrors.Wrap(err, apperrors.ErrCodeNotFound, "Service不存在")
	}

	info := s.convertServiceInfo(svc)
	detail := &ServiceDetail{
		ServiceInfo: info,
		Labels:      svc.Labels,
		Annotations: svc.Annotations,
	}

	// 获取 Endpoints
	endpoints, err := s.GetEndpoints(ctx, clusterID, namespace, name)
	if err == nil {
		detail.Endpoints = endpoints
	}

	return detail, nil
}

// GetEndpoints 获取 Service 的 Endpoints
func (s *K8sServiceService) GetEndpoints(ctx context.Context, clusterID uint, namespace, name string) ([]EndpointInfo, error) {
	client, err := s.clientMgr.GetClient(ctx, clusterID)
	if err != nil {
		return nil, err
	}

	endpoints, err := client.CoreV1().Endpoints(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, apperrors.Wrap(err, apperrors.ErrCodeNotFound, "Endpoints不存在")
	}

	var result []EndpointInfo
	for _, subset := range endpoints.Subsets {
		// 处理就绪的地址
		for _, addr := range subset.Addresses {
			nodeName := ""
			if addr.NodeName != nil {
				nodeName = *addr.NodeName
			}
			result = append(result, EndpointInfo{
				IP:       addr.IP,
				NodeName: nodeName,
				Ready:    true,
			})
		}
		// 处理未就绪的地址
		for _, addr := range subset.NotReadyAddresses {
			nodeName := ""
			if addr.NodeName != nil {
				nodeName = *addr.NodeName
			}
			result = append(result, EndpointInfo{
				IP:       addr.IP,
				NodeName: nodeName,
				Ready:    false,
			})
		}
	}

	return result, nil
}

// convertServiceInfo 转换 Service 信息
func (s *K8sServiceService) convertServiceInfo(svc *corev1.Service) ServiceInfo {
	ports := make([]ServicePort, len(svc.Spec.Ports))
	for i, p := range svc.Spec.Ports {
		ports[i] = ServicePort{
			Name:       p.Name,
			Protocol:   string(p.Protocol),
			Port:       p.Port,
			TargetPort: p.TargetPort.String(),
			NodePort:   p.NodePort,
		}
	}

	externalIP := ""
	if len(svc.Status.LoadBalancer.Ingress) > 0 {
		if svc.Status.LoadBalancer.Ingress[0].IP != "" {
			externalIP = svc.Status.LoadBalancer.Ingress[0].IP
		} else if svc.Status.LoadBalancer.Ingress[0].Hostname != "" {
			externalIP = svc.Status.LoadBalancer.Ingress[0].Hostname
		}
	} else if len(svc.Spec.ExternalIPs) > 0 {
		externalIP = svc.Spec.ExternalIPs[0]
	}

	age := time.Since(svc.CreationTimestamp.Time)

	return ServiceInfo{
		Name:       svc.Name,
		Namespace:  svc.Namespace,
		Type:       string(svc.Spec.Type),
		ClusterIP:  svc.Spec.ClusterIP,
		ExternalIP: externalIP,
		Ports:      ports,
		Age:        formatDuration(age),
		Selector:   svc.Spec.Selector,
		CreatedAt:  svc.CreationTimestamp.Format("2006-01-02 15:04:05"),
	}
}
