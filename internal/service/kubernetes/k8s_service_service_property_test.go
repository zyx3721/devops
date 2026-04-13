package kubernetes

import (
	"testing"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// Feature: k8s-resource-management, Property 5: 资源列表包含必需字段
// **Validates: Requirements 5.1**
func TestProperty_ServiceListContainsRequiredFields(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100

	properties := gopter.NewProperties(parameters)

	properties.Property("Service列表包含所有必需字段", prop.ForAll(
		func(services []*corev1.Service) bool {
			svc := &K8sServiceService{}
			result := make([]ServiceInfo, len(services))
			for i, s := range services {
				result[i] = svc.convertServiceInfo(s)
			}

			// 验证每个 ServiceInfo 都包含必需字段
			for _, info := range result {
				if info.Name == "" || info.Namespace == "" ||
					info.Type == "" || info.CreatedAt == "" {
					return false
				}
			}
			return true
		},
		genServiceList(),
	))

	properties.TestingRun(t)
}

// Feature: k8s-resource-management, Property 6: 资源详情显示
// **Validates: Requirements 5.2**
func TestProperty_ServiceDetailDisplay(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100

	properties := gopter.NewProperties(parameters)

	properties.Property("Service详情包含端口映射和选择器", prop.ForAll(
		func(service *corev1.Service) bool {
			svc := &K8sServiceService{}
			info := svc.convertServiceInfo(service)

			// 验证端口信息
			if len(service.Spec.Ports) != len(info.Ports) {
				return false
			}

			// 验证选择器
			if service.Spec.Selector != nil {
				if info.Selector == nil {
					return false
				}
				for k, v := range service.Spec.Selector {
					if info.Selector[k] != v {
						return false
					}
				}
			}

			return true
		},
		genService(),
	))

	properties.TestingRun(t)
}

// genServiceList 生成 Service 列表
func genServiceList() gopter.Gen {
	return gen.SliceOf(genService())
}

// genService 生成单个 Service
func genService() gopter.Gen {
	return gopter.CombineGens(
		gen.AlphaString(),                               // name
		gen.AlphaString(),                               // namespace
		genServiceType(),                                // type
		gen.SliceOf(genServicePort()),                   // ports
		gen.MapOf(gen.AlphaString(), gen.AlphaString()), // selector
		gen.MapOf(gen.AlphaString(), gen.AlphaString()), // labels
	).Map(func(values []interface{}) *corev1.Service {
		name := values[0].(string)
		namespace := values[1].(string)
		serviceType := values[2].(corev1.ServiceType)
		ports := values[3].([]corev1.ServicePort)
		selector := values[4].(map[string]string)
		labels := values[5].(map[string]string)

		// 确保名称和命名空间不为空
		if name == "" {
			name = "test-service"
		}
		if namespace == "" {
			namespace = "default"
		}

		service := &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:              name,
				Namespace:         namespace,
				Labels:            labels,
				CreationTimestamp: metav1.Now(),
			},
			Spec: corev1.ServiceSpec{
				Type:     serviceType,
				Ports:    ports,
				Selector: selector,
			},
		}

		// 根据类型设置 ClusterIP
		switch serviceType {
		case corev1.ServiceTypeClusterIP:
			service.Spec.ClusterIP = "10.96.0.1"
		case corev1.ServiceTypeNodePort:
			service.Spec.ClusterIP = "10.96.0.2"
		case corev1.ServiceTypeLoadBalancer:
			service.Spec.ClusterIP = "10.96.0.3"
			service.Status.LoadBalancer.Ingress = []corev1.LoadBalancerIngress{
				{IP: "203.0.113.1"},
			}
		case corev1.ServiceTypeExternalName:
			service.Spec.ExternalName = "example.com"
		}

		return service
	})
}

// genServiceType 生成 Service 类型
func genServiceType() gopter.Gen {
	return gen.OneConstOf(
		corev1.ServiceTypeClusterIP,
		corev1.ServiceTypeNodePort,
		corev1.ServiceTypeLoadBalancer,
		corev1.ServiceTypeExternalName,
	)
}

// genServicePort 生成 Service 端口
func genServicePort() gopter.Gen {
	return gopter.CombineGens(
		gen.AlphaString(),            // name
		genProtocol(),                // protocol
		gen.Int32Range(1, 65535),     // port
		genTargetPort(),              // targetPort
		gen.Int32Range(30000, 32767), // nodePort
	).Map(func(values []interface{}) corev1.ServicePort {
		name := values[0].(string)
		protocol := values[1].(corev1.Protocol)
		port := values[2].(int32)
		targetPort := values[3].(intstr.IntOrString)
		nodePort := values[4].(int32)

		// 确保端口名称不为空
		if name == "" {
			name = "http"
		}

		return corev1.ServicePort{
			Name:       name,
			Protocol:   protocol,
			Port:       port,
			TargetPort: targetPort,
			NodePort:   nodePort,
		}
	})
}

// genProtocol 生成协议
func genProtocol() gopter.Gen {
	return gen.OneConstOf(
		corev1.ProtocolTCP,
		corev1.ProtocolUDP,
		corev1.ProtocolSCTP,
	)
}

// genTargetPort 生成目标端口
func genTargetPort() gopter.Gen {
	return gen.OneGenOf(
		gen.Int32Range(1, 65535).Map(func(port int32) intstr.IntOrString {
			return intstr.FromInt(int(port))
		}),
		gen.AlphaString().Map(func(name string) intstr.IntOrString {
			if name == "" {
				name = "http"
			}
			return intstr.FromString(name)
		}),
	)
}
