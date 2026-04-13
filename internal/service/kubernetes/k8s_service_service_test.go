package kubernetes

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// TestConvertServiceInfo 测试 Service 信息转换
func TestConvertServiceInfo(t *testing.T) {
	svc := &K8sServiceService{}

	// 创建测试 Service
	now := metav1.Now()
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "test-service",
			Namespace:         "default",
			CreationTimestamp: now,
		},
		Spec: corev1.ServiceSpec{
			Type:      corev1.ServiceTypeClusterIP,
			ClusterIP: "10.96.0.1",
			Ports: []corev1.ServicePort{
				{
					Name:       "http",
					Protocol:   corev1.ProtocolTCP,
					Port:       80,
					TargetPort: intstr.FromInt(8080),
				},
				{
					Name:       "https",
					Protocol:   corev1.ProtocolTCP,
					Port:       443,
					TargetPort: intstr.FromString("https"),
				},
			},
			Selector: map[string]string{
				"app": "test",
			},
		},
	}

	// 执行转换
	info := svc.convertServiceInfo(service)

	// 验证基本信息
	assert.Equal(t, "test-service", info.Name)
	assert.Equal(t, "default", info.Namespace)
	assert.Equal(t, "ClusterIP", info.Type)
	assert.Equal(t, "10.96.0.1", info.ClusterIP)
	assert.Equal(t, "", info.ExternalIP)
	assert.NotEmpty(t, info.Age)
	assert.NotEmpty(t, info.CreatedAt)

	// 验证端口信息
	assert.Equal(t, 2, len(info.Ports))
	assert.Equal(t, "http", info.Ports[0].Name)
	assert.Equal(t, "TCP", info.Ports[0].Protocol)
	assert.Equal(t, int32(80), info.Ports[0].Port)
	assert.Equal(t, "8080", info.Ports[0].TargetPort)
	assert.Equal(t, int32(0), info.Ports[0].NodePort)

	assert.Equal(t, "https", info.Ports[1].Name)
	assert.Equal(t, "TCP", info.Ports[1].Protocol)
	assert.Equal(t, int32(443), info.Ports[1].Port)
	assert.Equal(t, "https", info.Ports[1].TargetPort)

	// 验证选择器
	assert.Equal(t, "test", info.Selector["app"])
}

// TestConvertServiceInfo_NodePort 测试 NodePort 类型的 Service
func TestConvertServiceInfo_NodePort(t *testing.T) {
	svc := &K8sServiceService{}

	now := metav1.Now()
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "nodeport-service",
			Namespace:         "default",
			CreationTimestamp: now,
		},
		Spec: corev1.ServiceSpec{
			Type:      corev1.ServiceTypeNodePort,
			ClusterIP: "10.96.0.2",
			Ports: []corev1.ServicePort{
				{
					Name:       "http",
					Protocol:   corev1.ProtocolTCP,
					Port:       80,
					TargetPort: intstr.FromInt(8080),
					NodePort:   30080,
				},
			},
			Selector: map[string]string{
				"app": "web",
			},
		},
	}

	info := svc.convertServiceInfo(service)

	assert.Equal(t, "NodePort", info.Type)
	assert.Equal(t, "10.96.0.2", info.ClusterIP)
	assert.Equal(t, 1, len(info.Ports))
	assert.Equal(t, int32(30080), info.Ports[0].NodePort)
}

// TestConvertServiceInfo_LoadBalancer 测试 LoadBalancer 类型的 Service
func TestConvertServiceInfo_LoadBalancer(t *testing.T) {
	svc := &K8sServiceService{}

	now := metav1.Now()
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "lb-service",
			Namespace:         "default",
			CreationTimestamp: now,
		},
		Spec: corev1.ServiceSpec{
			Type:      corev1.ServiceTypeLoadBalancer,
			ClusterIP: "10.96.0.3",
			Ports: []corev1.ServicePort{
				{
					Name:       "http",
					Protocol:   corev1.ProtocolTCP,
					Port:       80,
					TargetPort: intstr.FromInt(8080),
				},
			},
		},
		Status: corev1.ServiceStatus{
			LoadBalancer: corev1.LoadBalancerStatus{
				Ingress: []corev1.LoadBalancerIngress{
					{
						IP: "203.0.113.1",
					},
				},
			},
		},
	}

	info := svc.convertServiceInfo(service)

	assert.Equal(t, "LoadBalancer", info.Type)
	assert.Equal(t, "203.0.113.1", info.ExternalIP)
}

// TestConvertServiceInfo_LoadBalancerWithHostname 测试带主机名的 LoadBalancer
func TestConvertServiceInfo_LoadBalancerWithHostname(t *testing.T) {
	svc := &K8sServiceService{}

	now := metav1.Now()
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "lb-hostname-service",
			Namespace:         "default",
			CreationTimestamp: now,
		},
		Spec: corev1.ServiceSpec{
			Type:      corev1.ServiceTypeLoadBalancer,
			ClusterIP: "10.96.0.4",
			Ports: []corev1.ServicePort{
				{
					Name:       "http",
					Protocol:   corev1.ProtocolTCP,
					Port:       80,
					TargetPort: intstr.FromInt(8080),
				},
			},
		},
		Status: corev1.ServiceStatus{
			LoadBalancer: corev1.LoadBalancerStatus{
				Ingress: []corev1.LoadBalancerIngress{
					{
						Hostname: "lb.example.com",
					},
				},
			},
		},
	}

	info := svc.convertServiceInfo(service)

	assert.Equal(t, "LoadBalancer", info.Type)
	assert.Equal(t, "lb.example.com", info.ExternalIP)
}

// TestConvertServiceInfo_ExternalIPs 测试带外部 IP 的 Service
func TestConvertServiceInfo_ExternalIPs(t *testing.T) {
	svc := &K8sServiceService{}

	now := metav1.Now()
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "external-ip-service",
			Namespace:         "default",
			CreationTimestamp: now,
		},
		Spec: corev1.ServiceSpec{
			Type:      corev1.ServiceTypeClusterIP,
			ClusterIP: "10.96.0.5",
			ExternalIPs: []string{
				"203.0.113.2",
				"203.0.113.3",
			},
			Ports: []corev1.ServicePort{
				{
					Name:       "http",
					Protocol:   corev1.ProtocolTCP,
					Port:       80,
					TargetPort: intstr.FromInt(8080),
				},
			},
		},
	}

	info := svc.convertServiceInfo(service)

	assert.Equal(t, "ClusterIP", info.Type)
	assert.Equal(t, "203.0.113.2", info.ExternalIP) // 取第一个外部 IP
}

// TestConvertServiceInfo_ExternalName 测试 ExternalName 类型的 Service
func TestConvertServiceInfo_ExternalName(t *testing.T) {
	svc := &K8sServiceService{}

	now := metav1.Now()
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "external-name-service",
			Namespace:         "default",
			CreationTimestamp: now,
		},
		Spec: corev1.ServiceSpec{
			Type:         corev1.ServiceTypeExternalName,
			ExternalName: "example.com",
		},
	}

	info := svc.convertServiceInfo(service)

	assert.Equal(t, "ExternalName", info.Type)
	assert.Equal(t, "", info.ClusterIP) // ExternalName 类型没有 ClusterIP
}

// TestConvertServiceInfo_MultiplePorts 测试多端口 Service
func TestConvertServiceInfo_MultiplePorts(t *testing.T) {
	svc := &K8sServiceService{}

	now := metav1.Now()
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "multi-port-service",
			Namespace:         "default",
			CreationTimestamp: now,
		},
		Spec: corev1.ServiceSpec{
			Type:      corev1.ServiceTypeClusterIP,
			ClusterIP: "10.96.0.6",
			Ports: []corev1.ServicePort{
				{
					Name:       "http",
					Protocol:   corev1.ProtocolTCP,
					Port:       80,
					TargetPort: intstr.FromInt(8080),
				},
				{
					Name:       "https",
					Protocol:   corev1.ProtocolTCP,
					Port:       443,
					TargetPort: intstr.FromInt(8443),
				},
				{
					Name:       "metrics",
					Protocol:   corev1.ProtocolTCP,
					Port:       9090,
					TargetPort: intstr.FromString("metrics"),
				},
			},
		},
	}

	info := svc.convertServiceInfo(service)

	assert.Equal(t, 3, len(info.Ports))
	assert.Equal(t, "http", info.Ports[0].Name)
	assert.Equal(t, int32(80), info.Ports[0].Port)
	assert.Equal(t, "8080", info.Ports[0].TargetPort)

	assert.Equal(t, "https", info.Ports[1].Name)
	assert.Equal(t, int32(443), info.Ports[1].Port)
	assert.Equal(t, "8443", info.Ports[1].TargetPort)

	assert.Equal(t, "metrics", info.Ports[2].Name)
	assert.Equal(t, int32(9090), info.Ports[2].Port)
	assert.Equal(t, "metrics", info.Ports[2].TargetPort)
}

// TestConvertServiceInfo_NoSelector 测试没有选择器的 Service
func TestConvertServiceInfo_NoSelector(t *testing.T) {
	svc := &K8sServiceService{}

	now := metav1.Now()
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "no-selector-service",
			Namespace:         "default",
			CreationTimestamp: now,
		},
		Spec: corev1.ServiceSpec{
			Type:      corev1.ServiceTypeClusterIP,
			ClusterIP: "10.96.0.7",
			Ports: []corev1.ServicePort{
				{
					Name:       "http",
					Protocol:   corev1.ProtocolTCP,
					Port:       80,
					TargetPort: intstr.FromInt(8080),
				},
			},
			Selector: nil, // 没有选择器
		},
	}

	info := svc.convertServiceInfo(service)

	// Selector 可以是 nil 或空 map
	if info.Selector != nil {
		assert.Equal(t, 0, len(info.Selector))
	}
}

// TestConvertServiceInfo_NoPorts 测试没有端口的 Service
func TestConvertServiceInfo_NoPorts(t *testing.T) {
	svc := &K8sServiceService{}

	now := metav1.Now()
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "no-ports-service",
			Namespace:         "default",
			CreationTimestamp: now,
		},
		Spec: corev1.ServiceSpec{
			Type:      corev1.ServiceTypeClusterIP,
			ClusterIP: "10.96.0.8",
			Ports:     []corev1.ServicePort{}, // 没有端口
		},
	}

	info := svc.convertServiceInfo(service)

	assert.Equal(t, 0, len(info.Ports))
}

// TestConvertServiceInfo_UDPProtocol 测试 UDP 协议的 Service
func TestConvertServiceInfo_UDPProtocol(t *testing.T) {
	svc := &K8sServiceService{}

	now := metav1.Now()
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "udp-service",
			Namespace:         "default",
			CreationTimestamp: now,
		},
		Spec: corev1.ServiceSpec{
			Type:      corev1.ServiceTypeClusterIP,
			ClusterIP: "10.96.0.9",
			Ports: []corev1.ServicePort{
				{
					Name:       "dns",
					Protocol:   corev1.ProtocolUDP,
					Port:       53,
					TargetPort: intstr.FromInt(53),
				},
			},
		},
	}

	info := svc.convertServiceInfo(service)

	assert.Equal(t, 1, len(info.Ports))
	assert.Equal(t, "UDP", info.Ports[0].Protocol)
	assert.Equal(t, int32(53), info.Ports[0].Port)
}

// TestConvertServiceInfo_HeadlessService 测试 Headless Service
func TestConvertServiceInfo_HeadlessService(t *testing.T) {
	svc := &K8sServiceService{}

	now := metav1.Now()
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "headless-service",
			Namespace:         "default",
			CreationTimestamp: now,
		},
		Spec: corev1.ServiceSpec{
			Type:      corev1.ServiceTypeClusterIP,
			ClusterIP: "None", // Headless Service
			Ports: []corev1.ServicePort{
				{
					Name:       "http",
					Protocol:   corev1.ProtocolTCP,
					Port:       80,
					TargetPort: intstr.FromInt(8080),
				},
			},
			Selector: map[string]string{
				"app": "stateful",
			},
		},
	}

	info := svc.convertServiceInfo(service)

	assert.Equal(t, "ClusterIP", info.Type)
	assert.Equal(t, "None", info.ClusterIP)
}
