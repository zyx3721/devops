package kubernetes

import (
	"context"
	"devops/internal/models"
	"fmt"

	"gorm.io/gorm"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
)

// IstioService Istio 流量治理服务
type IstioService struct {
	clientManager *K8sClientManager
	db            *gorm.DB
}

// NewIstioService 创建 Istio 服务
func NewIstioService(clientManager *K8sClientManager, db *gorm.DB) *IstioService {
	return &IstioService{
		clientManager: clientManager,
		db:            db,
	}
}

// Istio CRD GVR 定义
var (
	virtualServiceGVR = schema.GroupVersionResource{
		Group:    "networking.istio.io",
		Version:  "v1beta1",
		Resource: "virtualservices",
	}
	destinationRuleGVR = schema.GroupVersionResource{
		Group:    "networking.istio.io",
		Version:  "v1beta1",
		Resource: "destinationrules",
	}
	gatewayGVR = schema.GroupVersionResource{
		Group:    "networking.istio.io",
		Version:  "v1beta1",
		Resource: "gateways",
	}
	serviceEntryGVR = schema.GroupVersionResource{
		Group:    "networking.istio.io",
		Version:  "v1beta1",
		Resource: "serviceentries",
	}
)

// IstioStatus Istio 状态
type IstioStatus struct {
	Installed      bool   `json:"installed"`
	Version        string `json:"version"`
	InjectionReady bool   `json:"injection_ready"`
	Message        string `json:"message"`
}

// CheckIstioStatus 检查 Istio 安装状态
func (s *IstioService) CheckIstioStatus(ctx context.Context, clusterID uint) (*IstioStatus, error) {
	client, err := s.clientManager.GetClient(ctx, clusterID)
	if err != nil {
		return nil, err
	}

	status := &IstioStatus{
		Installed:      false,
		InjectionReady: false,
	}

	// 检查 istio-system 命名空间
	_, err = client.CoreV1().Namespaces().Get(ctx, "istio-system", metav1.GetOptions{})
	if err != nil {
		status.Message = "Istio 未安装：istio-system 命名空间不存在"
		return status, nil
	}

	// 检查 istiod deployment
	istiod, err := client.AppsV1().Deployments("istio-system").Get(ctx, "istiod", metav1.GetOptions{})
	if err != nil {
		status.Message = "Istio 未安装：istiod 不存在"
		return status, nil
	}

	status.Installed = true

	// 检查 istiod 是否就绪
	if istiod.Status.ReadyReplicas > 0 {
		status.InjectionReady = true
		status.Message = "Istio 已安装并就绪"
	} else {
		status.Message = "Istio 已安装但 istiod 未就绪"
	}

	// 获取版本
	if istiod.Spec.Template.Spec.Containers != nil && len(istiod.Spec.Template.Spec.Containers) > 0 {
		image := istiod.Spec.Template.Spec.Containers[0].Image
		status.Version = image
	}

	return status, nil
}

// getDynamicClient 获取动态客户端
func (s *IstioService) getDynamicClient(ctx context.Context, clusterID uint) (dynamic.Interface, error) {
	config, err := s.clientManager.GetRestConfig(ctx, clusterID)
	if err != nil {
		return nil, err
	}
	return dynamic.NewForConfig(config)
}

// ========== VirtualService 管理 ==========

// SyncRoutingRules 同步流量路由规则到 VirtualService
func (s *IstioService) SyncRoutingRules(ctx context.Context, clusterID uint, namespace string, app *models.Application, rules []models.TrafficRoutingRule) error {
	dynamicClient, err := s.getDynamicClient(ctx, clusterID)
	if err != nil {
		return fmt.Errorf("获取动态客户端失败: %w", err)
	}

	vsName := fmt.Sprintf("%s-vs", app.Name)

	// 构建 VirtualService
	vs := s.buildVirtualService(vsName, namespace, app.Name, rules)

	// 尝试更新，如果不存在则创建
	_, err = dynamicClient.Resource(virtualServiceGVR).Namespace(namespace).Get(ctx, vsName, metav1.GetOptions{})
	if err != nil {
		// 创建
		_, err = dynamicClient.Resource(virtualServiceGVR).Namespace(namespace).Create(ctx, vs, metav1.CreateOptions{})
		if err != nil {
			return fmt.Errorf("创建 VirtualService 失败: %w", err)
		}
	} else {
		// 更新
		_, err = dynamicClient.Resource(virtualServiceGVR).Namespace(namespace).Update(ctx, vs, metav1.UpdateOptions{})
		if err != nil {
			return fmt.Errorf("更新 VirtualService 失败: %w", err)
		}
	}

	return nil
}

// buildVirtualService 构建 VirtualService 对象
func (s *IstioService) buildVirtualService(name, namespace, serviceName string, rules []models.TrafficRoutingRule) *unstructured.Unstructured {
	httpRoutes := make([]interface{}, 0)

	for _, rule := range rules {
		if !rule.Enabled {
			continue
		}

		route := map[string]interface{}{
			"name": rule.Name,
		}

		// 根据路由类型构建匹配条件
		switch rule.RouteType {
		case "header":
			route["match"] = []interface{}{
				map[string]interface{}{
					"headers": map[string]interface{}{
						rule.MatchKey: s.buildMatchCondition(rule.MatchOperator, rule.MatchValue),
					},
				},
			}
		case "cookie":
			route["match"] = []interface{}{
				map[string]interface{}{
					"headers": map[string]interface{}{
						"cookie": s.buildMatchCondition(rule.MatchOperator, rule.MatchValue),
					},
				},
			}
		case "param":
			route["match"] = []interface{}{
				map[string]interface{}{
					"queryParams": map[string]interface{}{
						rule.MatchKey: s.buildMatchCondition(rule.MatchOperator, rule.MatchValue),
					},
				},
			}
		}

		// 构建目标
		if rule.RouteType == "weight" && len(rule.Destinations) > 0 {
			routeDests := make([]interface{}, 0)
			for _, dest := range rule.Destinations {
				routeDest := map[string]interface{}{
					"destination": map[string]interface{}{
						"host":   serviceName,
						"subset": dest.Subset,
					},
					"weight": dest.Weight,
				}
				routeDests = append(routeDests, routeDest)
			}
			route["route"] = routeDests
		} else if rule.TargetSubset != "" {
			route["route"] = []interface{}{
				map[string]interface{}{
					"destination": map[string]interface{}{
						"host":   serviceName,
						"subset": rule.TargetSubset,
					},
				},
			}
		}

		httpRoutes = append(httpRoutes, route)
	}

	// 添加默认路由
	if len(httpRoutes) == 0 {
		httpRoutes = append(httpRoutes, map[string]interface{}{
			"route": []interface{}{
				map[string]interface{}{
					"destination": map[string]interface{}{
						"host": serviceName,
					},
				},
			},
		})
	}

	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "networking.istio.io/v1beta1",
			"kind":       "VirtualService",
			"metadata": map[string]interface{}{
				"name":      name,
				"namespace": namespace,
				"labels": map[string]interface{}{
					"app":        serviceName,
					"managed-by": "devops-platform",
				},
			},
			"spec": map[string]interface{}{
				"hosts": []interface{}{serviceName},
				"http":  httpRoutes,
			},
		},
	}
}

func (s *IstioService) buildMatchCondition(operator, value string) map[string]interface{} {
	switch operator {
	case "exact":
		return map[string]interface{}{"exact": value}
	case "prefix":
		return map[string]interface{}{"prefix": value}
	case "regex":
		return map[string]interface{}{"regex": value}
	case "present":
		return map[string]interface{}{"regex": ".*"}
	default:
		return map[string]interface{}{"exact": value}
	}
}

// ========== DestinationRule 管理 ==========

// SyncDestinationRule 同步负载均衡和熔断规则到 DestinationRule
func (s *IstioService) SyncDestinationRule(ctx context.Context, clusterID uint, namespace string, app *models.Application,
	lbConfig *models.TrafficLoadBalanceConfig, cbRules []models.TrafficCircuitBreakerRule) error {

	dynamicClient, err := s.getDynamicClient(ctx, clusterID)
	if err != nil {
		return fmt.Errorf("获取动态客户端失败: %w", err)
	}

	drName := fmt.Sprintf("%s-dr", app.Name)

	// 构建 DestinationRule
	dr := s.buildDestinationRule(drName, namespace, app.Name, lbConfig, cbRules)

	// 尝试更新，如果不存在则创建
	_, err = dynamicClient.Resource(destinationRuleGVR).Namespace(namespace).Get(ctx, drName, metav1.GetOptions{})
	if err != nil {
		// 创建
		_, err = dynamicClient.Resource(destinationRuleGVR).Namespace(namespace).Create(ctx, dr, metav1.CreateOptions{})
		if err != nil {
			return fmt.Errorf("创建 DestinationRule 失败: %w", err)
		}
	} else {
		// 更新
		_, err = dynamicClient.Resource(destinationRuleGVR).Namespace(namespace).Update(ctx, dr, metav1.UpdateOptions{})
		if err != nil {
			return fmt.Errorf("更新 DestinationRule 失败: %w", err)
		}
	}

	return nil
}

// buildDestinationRule 构建 DestinationRule 对象
func (s *IstioService) buildDestinationRule(name, namespace, serviceName string,
	lbConfig *models.TrafficLoadBalanceConfig, cbRules []models.TrafficCircuitBreakerRule) *unstructured.Unstructured {

	trafficPolicy := map[string]interface{}{}

	// 负载均衡配置
	if lbConfig != nil {
		lb := map[string]interface{}{}

		switch lbConfig.LbPolicy {
		case "round_robin":
			lb["simple"] = "ROUND_ROBIN"
		case "random":
			lb["simple"] = "RANDOM"
		case "least_request":
			lb["simple"] = "LEAST_REQUEST"
		case "passthrough":
			lb["simple"] = "PASSTHROUGH"
		case "consistent_hash":
			consistentHash := map[string]interface{}{}
			switch lbConfig.HashKey {
			case "header":
				consistentHash["httpHeaderName"] = lbConfig.HashKeyName
			case "cookie":
				consistentHash["httpCookie"] = map[string]interface{}{
					"name": lbConfig.HashKeyName,
					"ttl":  "0s",
				}
			case "source_ip":
				consistentHash["useSourceIp"] = true
			case "query_param":
				consistentHash["httpQueryParameterName"] = lbConfig.HashKeyName
			}
			if lbConfig.RingSize > 0 {
				consistentHash["minimumRingSize"] = lbConfig.RingSize
			}
			lb["consistentHash"] = consistentHash
		}

		if len(lb) > 0 {
			trafficPolicy["loadBalancer"] = lb
		}

		// 连接池配置
		connectionPool := map[string]interface{}{}

		if lbConfig.HTTPMaxConnections > 0 || lbConfig.HTTPMaxPendingRequests > 0 || lbConfig.HTTPMaxRetries > 0 {
			httpPool := map[string]interface{}{}
			if lbConfig.HTTPMaxConnections > 0 {
				httpPool["h2UpgradePolicy"] = "UPGRADE"
			}
			if lbConfig.HTTPMaxPendingRequests > 0 {
				httpPool["http1MaxPendingRequests"] = lbConfig.HTTPMaxPendingRequests
			}
			if lbConfig.HTTPMaxRequestsPerConn > 0 {
				httpPool["maxRequestsPerConnection"] = lbConfig.HTTPMaxRequestsPerConn
			}
			if lbConfig.HTTPMaxRetries > 0 {
				httpPool["maxRetries"] = lbConfig.HTTPMaxRetries
			}
			if lbConfig.HTTPIdleTimeout != "" {
				httpPool["idleTimeout"] = lbConfig.HTTPIdleTimeout
			}
			connectionPool["http"] = httpPool
		}

		if lbConfig.TCPMaxConnections > 0 {
			tcpPool := map[string]interface{}{
				"maxConnections": lbConfig.TCPMaxConnections,
			}
			if lbConfig.TCPConnectTimeout != "" {
				tcpPool["connectTimeout"] = lbConfig.TCPConnectTimeout
			}
			connectionPool["tcp"] = tcpPool
		}

		if len(connectionPool) > 0 {
			trafficPolicy["connectionPool"] = connectionPool
		}
	}

	// 熔断配置 (outlierDetection)
	if len(cbRules) > 0 {
		for _, rule := range cbRules {
			if !rule.Enabled {
				continue
			}

			outlierDetection := map[string]interface{}{
				"baseEjectionTime":         fmt.Sprintf("%ds", rule.RecoveryTimeout),
				"consecutiveGatewayErrors": 5,
				"interval":                 fmt.Sprintf("%ds", rule.StatInterval),
				"maxEjectionPercent":       100,
			}

			switch rule.Strategy {
			case "error_count":
				outlierDetection["consecutive5xxErrors"] = int(rule.Threshold)
			case "error_ratio":
				// Istio 不直接支持错误率，使用连续错误数近似
				outlierDetection["consecutive5xxErrors"] = 5
			case "slow_request":
				// Istio 不直接支持慢请求熔断，使用网关错误
				outlierDetection["consecutiveGatewayErrors"] = 5
			}

			trafficPolicy["outlierDetection"] = outlierDetection
			break // 只使用第一个启用的规则
		}
	}

	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "networking.istio.io/v1beta1",
			"kind":       "DestinationRule",
			"metadata": map[string]interface{}{
				"name":      name,
				"namespace": namespace,
				"labels": map[string]interface{}{
					"app":        serviceName,
					"managed-by": "devops-platform",
				},
			},
			"spec": map[string]interface{}{
				"host":          serviceName,
				"trafficPolicy": trafficPolicy,
			},
		},
	}
}

// ========== 超时重试配置 ==========

// SyncTimeoutRetry 同步超时重试配置到 VirtualService
func (s *IstioService) SyncTimeoutRetry(ctx context.Context, clusterID uint, namespace string, app *models.Application,
	config *models.TrafficTimeoutConfig) error {

	dynamicClient, err := s.getDynamicClient(ctx, clusterID)
	if err != nil {
		return fmt.Errorf("获取动态客户端失败: %w", err)
	}

	vsName := fmt.Sprintf("%s-vs", app.Name)

	// 获取现有的 VirtualService
	existing, err := dynamicClient.Resource(virtualServiceGVR).Namespace(namespace).Get(ctx, vsName, metav1.GetOptions{})
	if err != nil {
		// 如果不存在，创建一个基础的
		existing = &unstructured.Unstructured{
			Object: map[string]interface{}{
				"apiVersion": "networking.istio.io/v1beta1",
				"kind":       "VirtualService",
				"metadata": map[string]interface{}{
					"name":      vsName,
					"namespace": namespace,
					"labels": map[string]interface{}{
						"app":        app.Name,
						"managed-by": "devops-platform",
					},
				},
				"spec": map[string]interface{}{
					"hosts": []interface{}{app.Name},
					"http": []interface{}{
						map[string]interface{}{
							"route": []interface{}{
								map[string]interface{}{
									"destination": map[string]interface{}{
										"host": app.Name,
									},
								},
							},
						},
					},
				},
			},
		}
	}

	// 更新超时和重试配置
	spec := existing.Object["spec"].(map[string]interface{})
	httpRoutes := spec["http"].([]interface{})

	if len(httpRoutes) > 0 {
		route := httpRoutes[0].(map[string]interface{})

		// 设置超时
		if config.Timeout != "" {
			route["timeout"] = config.Timeout
		}

		// 设置重试
		if config.Retries > 0 {
			retryPolicy := map[string]interface{}{
				"attempts": config.Retries,
			}
			if config.PerTryTimeout != "" {
				retryPolicy["perTryTimeout"] = config.PerTryTimeout
			}
			if len(config.RetryOn) > 0 {
				retryPolicy["retryOn"] = config.RetryOn[0] // Istio 使用逗号分隔的字符串
			}
			route["retries"] = retryPolicy
		}

		httpRoutes[0] = route
		spec["http"] = httpRoutes
	}

	// 更新或创建
	_, err = dynamicClient.Resource(virtualServiceGVR).Namespace(namespace).Get(ctx, vsName, metav1.GetOptions{})
	if err != nil {
		_, err = dynamicClient.Resource(virtualServiceGVR).Namespace(namespace).Create(ctx, existing, metav1.CreateOptions{})
	} else {
		_, err = dynamicClient.Resource(virtualServiceGVR).Namespace(namespace).Update(ctx, existing, metav1.UpdateOptions{})
	}

	return err
}

// ========== 流量镜像配置 ==========

// SyncMirrorRule 同步流量镜像规则
func (s *IstioService) SyncMirrorRule(ctx context.Context, clusterID uint, namespace string, app *models.Application,
	rule *models.TrafficMirrorRule) error {

	if rule == nil || !rule.Enabled {
		return nil
	}

	dynamicClient, err := s.getDynamicClient(ctx, clusterID)
	if err != nil {
		return fmt.Errorf("获取动态客户端失败: %w", err)
	}

	vsName := fmt.Sprintf("%s-vs", app.Name)

	// 获取现有的 VirtualService
	existing, err := dynamicClient.Resource(virtualServiceGVR).Namespace(namespace).Get(ctx, vsName, metav1.GetOptions{})
	if err != nil {
		// 创建基础 VirtualService
		existing = &unstructured.Unstructured{
			Object: map[string]interface{}{
				"apiVersion": "networking.istio.io/v1beta1",
				"kind":       "VirtualService",
				"metadata": map[string]interface{}{
					"name":      vsName,
					"namespace": namespace,
				},
				"spec": map[string]interface{}{
					"hosts": []interface{}{app.Name},
					"http": []interface{}{
						map[string]interface{}{
							"route": []interface{}{
								map[string]interface{}{
									"destination": map[string]interface{}{
										"host": app.Name,
									},
								},
							},
						},
					},
				},
			},
		}
	}

	// 添加镜像配置
	spec := existing.Object["spec"].(map[string]interface{})
	httpRoutes := spec["http"].([]interface{})

	if len(httpRoutes) > 0 {
		route := httpRoutes[0].(map[string]interface{})

		mirror := map[string]interface{}{
			"host": rule.TargetService,
		}
		if rule.TargetSubset != "" {
			mirror["subset"] = rule.TargetSubset
		}
		route["mirror"] = mirror

		// 镜像百分比
		if rule.Percentage > 0 && rule.Percentage < 100 {
			route["mirrorPercentage"] = map[string]interface{}{
				"value": float64(rule.Percentage),
			}
		}

		httpRoutes[0] = route
		spec["http"] = httpRoutes
	}

	// 更新或创建
	_, err = dynamicClient.Resource(virtualServiceGVR).Namespace(namespace).Get(ctx, vsName, metav1.GetOptions{})
	if err != nil {
		_, err = dynamicClient.Resource(virtualServiceGVR).Namespace(namespace).Create(ctx, existing, metav1.CreateOptions{})
	} else {
		_, err = dynamicClient.Resource(virtualServiceGVR).Namespace(namespace).Update(ctx, existing, metav1.UpdateOptions{})
	}

	return err
}

// ========== 故障注入配置 ==========

// SyncFaultRule 同步故障注入规则
func (s *IstioService) SyncFaultRule(ctx context.Context, clusterID uint, namespace string, app *models.Application,
	rule *models.TrafficFaultRule) error {

	if rule == nil || !rule.Enabled {
		return nil
	}

	dynamicClient, err := s.getDynamicClient(ctx, clusterID)
	if err != nil {
		return fmt.Errorf("获取动态客户端失败: %w", err)
	}

	vsName := fmt.Sprintf("%s-vs", app.Name)

	// 获取现有的 VirtualService
	existing, err := dynamicClient.Resource(virtualServiceGVR).Namespace(namespace).Get(ctx, vsName, metav1.GetOptions{})
	if err != nil {
		// 创建基础 VirtualService
		existing = &unstructured.Unstructured{
			Object: map[string]interface{}{
				"apiVersion": "networking.istio.io/v1beta1",
				"kind":       "VirtualService",
				"metadata": map[string]interface{}{
					"name":      vsName,
					"namespace": namespace,
				},
				"spec": map[string]interface{}{
					"hosts": []interface{}{app.Name},
					"http": []interface{}{
						map[string]interface{}{
							"route": []interface{}{
								map[string]interface{}{
									"destination": map[string]interface{}{
										"host": app.Name,
									},
								},
							},
						},
					},
				},
			},
		}
	}

	// 添加故障注入配置
	spec := existing.Object["spec"].(map[string]interface{})
	httpRoutes := spec["http"].([]interface{})

	if len(httpRoutes) > 0 {
		route := httpRoutes[0].(map[string]interface{})

		fault := map[string]interface{}{}

		switch rule.Type {
		case "delay":
			fault["delay"] = map[string]interface{}{
				"percentage": map[string]interface{}{
					"value": float64(rule.Percentage),
				},
				"fixedDelay": rule.DelayDuration,
			}
		case "abort":
			fault["abort"] = map[string]interface{}{
				"percentage": map[string]interface{}{
					"value": float64(rule.Percentage),
				},
				"httpStatus": rule.AbortCode,
			}
		}

		// 添加路径匹配
		if rule.Path != "" && rule.Path != "/" {
			route["match"] = []interface{}{
				map[string]interface{}{
					"uri": map[string]interface{}{
						"prefix": rule.Path,
					},
				},
			}
		}

		route["fault"] = fault
		httpRoutes[0] = route
		spec["http"] = httpRoutes
	}

	// 更新或创建
	_, err = dynamicClient.Resource(virtualServiceGVR).Namespace(namespace).Get(ctx, vsName, metav1.GetOptions{})
	if err != nil {
		_, err = dynamicClient.Resource(virtualServiceGVR).Namespace(namespace).Create(ctx, existing, metav1.CreateOptions{})
	} else {
		_, err = dynamicClient.Resource(virtualServiceGVR).Namespace(namespace).Update(ctx, existing, metav1.UpdateOptions{})
	}

	return err
}

// ========== 删除规则 ==========

// DeleteVirtualService 删除 VirtualService
func (s *IstioService) DeleteVirtualService(ctx context.Context, clusterID uint, namespace, name string) error {
	dynamicClient, err := s.getDynamicClient(ctx, clusterID)
	if err != nil {
		return err
	}
	return dynamicClient.Resource(virtualServiceGVR).Namespace(namespace).Delete(ctx, name, metav1.DeleteOptions{})
}

// DeleteDestinationRule 删除 DestinationRule
func (s *IstioService) DeleteDestinationRule(ctx context.Context, clusterID uint, namespace, name string) error {
	dynamicClient, err := s.getDynamicClient(ctx, clusterID)
	if err != nil {
		return err
	}
	return dynamicClient.Resource(destinationRuleGVR).Namespace(namespace).Delete(ctx, name, metav1.DeleteOptions{})
}

// ========== 获取规则 ==========

// GetVirtualService 获取 VirtualService
func (s *IstioService) GetVirtualService(ctx context.Context, clusterID uint, namespace, name string) (*unstructured.Unstructured, error) {
	dynamicClient, err := s.getDynamicClient(ctx, clusterID)
	if err != nil {
		return nil, err
	}
	return dynamicClient.Resource(virtualServiceGVR).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
}

// GetDestinationRule 获取 DestinationRule
func (s *IstioService) GetDestinationRule(ctx context.Context, clusterID uint, namespace, name string) (*unstructured.Unstructured, error) {
	dynamicClient, err := s.getDynamicClient(ctx, clusterID)
	if err != nil {
		return nil, err
	}
	return dynamicClient.Resource(destinationRuleGVR).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
}

// ListVirtualServices 列出命名空间下的所有 VirtualService
func (s *IstioService) ListVirtualServices(ctx context.Context, clusterID uint, namespace string) (*unstructured.UnstructuredList, error) {
	dynamicClient, err := s.getDynamicClient(ctx, clusterID)
	if err != nil {
		return nil, err
	}
	return dynamicClient.Resource(virtualServiceGVR).Namespace(namespace).List(ctx, metav1.ListOptions{})
}

// ListDestinationRules 列出命名空间下的所有 DestinationRule
func (s *IstioService) ListDestinationRules(ctx context.Context, clusterID uint, namespace string) (*unstructured.UnstructuredList, error) {
	dynamicClient, err := s.getDynamicClient(ctx, clusterID)
	if err != nil {
		return nil, err
	}
	return dynamicClient.Resource(destinationRuleGVR).Namespace(namespace).List(ctx, metav1.ListOptions{})
}

// ========== Gateway 管理 ==========

// GatewayConfig Gateway 配置
type GatewayConfig struct {
	Name      string   `json:"name"`
	Namespace string   `json:"namespace"`
	Hosts     []string `json:"hosts"`
	Port      int      `json:"port"`
	Protocol  string   `json:"protocol"` // HTTP, HTTPS, GRPC, TCP
	TLSMode   string   `json:"tls_mode"` // PASSTHROUGH, SIMPLE, MUTUAL
	CredName  string   `json:"cred_name"`
}

// CreateGateway 创建 Gateway
func (s *IstioService) CreateGateway(ctx context.Context, clusterID uint, config *GatewayConfig) error {
	dynamicClient, err := s.getDynamicClient(ctx, clusterID)
	if err != nil {
		return fmt.Errorf("获取动态客户端失败: %w", err)
	}

	servers := []interface{}{
		map[string]interface{}{
			"port": map[string]interface{}{
				"number":   config.Port,
				"name":     fmt.Sprintf("%s-%d", config.Protocol, config.Port),
				"protocol": config.Protocol,
			},
			"hosts": config.Hosts,
		},
	}

	// 添加 TLS 配置
	if config.TLSMode != "" && config.TLSMode != "PASSTHROUGH" {
		server := servers[0].(map[string]interface{})
		server["tls"] = map[string]interface{}{
			"mode":           config.TLSMode,
			"credentialName": config.CredName,
		}
	}

	gateway := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "networking.istio.io/v1beta1",
			"kind":       "Gateway",
			"metadata": map[string]interface{}{
				"name":      config.Name,
				"namespace": config.Namespace,
				"labels": map[string]interface{}{
					"managed-by": "devops-platform",
				},
			},
			"spec": map[string]interface{}{
				"selector": map[string]interface{}{
					"istio": "ingressgateway",
				},
				"servers": servers,
			},
		},
	}

	_, err = dynamicClient.Resource(gatewayGVR).Namespace(config.Namespace).Create(ctx, gateway, metav1.CreateOptions{})
	if err != nil {
		_, err = dynamicClient.Resource(gatewayGVR).Namespace(config.Namespace).Update(ctx, gateway, metav1.UpdateOptions{})
	}
	return err
}

// DeleteGateway 删除 Gateway
func (s *IstioService) DeleteGateway(ctx context.Context, clusterID uint, namespace, name string) error {
	dynamicClient, err := s.getDynamicClient(ctx, clusterID)
	if err != nil {
		return err
	}
	return dynamicClient.Resource(gatewayGVR).Namespace(namespace).Delete(ctx, name, metav1.DeleteOptions{})
}

// GetGateway 获取 Gateway
func (s *IstioService) GetGateway(ctx context.Context, clusterID uint, namespace, name string) (*unstructured.Unstructured, error) {
	dynamicClient, err := s.getDynamicClient(ctx, clusterID)
	if err != nil {
		return nil, err
	}
	return dynamicClient.Resource(gatewayGVR).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
}

// ListGateways 列出 Gateway
func (s *IstioService) ListGateways(ctx context.Context, clusterID uint, namespace string) (*unstructured.UnstructuredList, error) {
	dynamicClient, err := s.getDynamicClient(ctx, clusterID)
	if err != nil {
		return nil, err
	}
	return dynamicClient.Resource(gatewayGVR).Namespace(namespace).List(ctx, metav1.ListOptions{})
}

// ========== ServiceEntry 管理 ==========

// ServiceEntryConfig ServiceEntry 配置
type ServiceEntryConfig struct {
	Name       string                 `json:"name"`
	Namespace  string                 `json:"namespace"`
	Hosts      []string               `json:"hosts"`
	Ports      []ServiceEntryPort     `json:"ports"`
	Location   string                 `json:"location"`   // MESH_EXTERNAL, MESH_INTERNAL
	Resolution string                 `json:"resolution"` // NONE, STATIC, DNS
	Endpoints  []ServiceEntryEndpoint `json:"endpoints"`
}

type ServiceEntryPort struct {
	Number   int    `json:"number"`
	Name     string `json:"name"`
	Protocol string `json:"protocol"`
}

type ServiceEntryEndpoint struct {
	Address string            `json:"address"`
	Ports   map[string]int    `json:"ports"`
	Labels  map[string]string `json:"labels"`
}

// CreateServiceEntry 创建 ServiceEntry
func (s *IstioService) CreateServiceEntry(ctx context.Context, clusterID uint, config *ServiceEntryConfig) error {
	dynamicClient, err := s.getDynamicClient(ctx, clusterID)
	if err != nil {
		return fmt.Errorf("获取动态客户端失败: %w", err)
	}

	ports := make([]interface{}, 0)
	for _, p := range config.Ports {
		ports = append(ports, map[string]interface{}{
			"number":   p.Number,
			"name":     p.Name,
			"protocol": p.Protocol,
		})
	}

	spec := map[string]interface{}{
		"hosts":      config.Hosts,
		"ports":      ports,
		"location":   config.Location,
		"resolution": config.Resolution,
	}

	if len(config.Endpoints) > 0 {
		endpoints := make([]interface{}, 0)
		for _, ep := range config.Endpoints {
			endpoint := map[string]interface{}{
				"address": ep.Address,
			}
			if len(ep.Ports) > 0 {
				endpoint["ports"] = ep.Ports
			}
			if len(ep.Labels) > 0 {
				endpoint["labels"] = ep.Labels
			}
			endpoints = append(endpoints, endpoint)
		}
		spec["endpoints"] = endpoints
	}

	serviceEntry := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "networking.istio.io/v1beta1",
			"kind":       "ServiceEntry",
			"metadata": map[string]interface{}{
				"name":      config.Name,
				"namespace": config.Namespace,
				"labels": map[string]interface{}{
					"managed-by": "devops-platform",
				},
			},
			"spec": spec,
		},
	}

	_, err = dynamicClient.Resource(serviceEntryGVR).Namespace(config.Namespace).Create(ctx, serviceEntry, metav1.CreateOptions{})
	if err != nil {
		_, err = dynamicClient.Resource(serviceEntryGVR).Namespace(config.Namespace).Update(ctx, serviceEntry, metav1.UpdateOptions{})
	}
	return err
}

// DeleteServiceEntry 删除 ServiceEntry
func (s *IstioService) DeleteServiceEntry(ctx context.Context, clusterID uint, namespace, name string) error {
	dynamicClient, err := s.getDynamicClient(ctx, clusterID)
	if err != nil {
		return err
	}
	return dynamicClient.Resource(serviceEntryGVR).Namespace(namespace).Delete(ctx, name, metav1.DeleteOptions{})
}

// GetServiceEntry 获取 ServiceEntry
func (s *IstioService) GetServiceEntry(ctx context.Context, clusterID uint, namespace, name string) (*unstructured.Unstructured, error) {
	dynamicClient, err := s.getDynamicClient(ctx, clusterID)
	if err != nil {
		return nil, err
	}
	return dynamicClient.Resource(serviceEntryGVR).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
}

// ListServiceEntries 列出 ServiceEntry
func (s *IstioService) ListServiceEntries(ctx context.Context, clusterID uint, namespace string) (*unstructured.UnstructuredList, error) {
	dynamicClient, err := s.getDynamicClient(ctx, clusterID)
	if err != nil {
		return nil, err
	}
	return dynamicClient.Resource(serviceEntryGVR).Namespace(namespace).List(ctx, metav1.ListOptions{})
}

// ========== 流量治理统计 ==========

// TrafficStats 流量统计数据
type TrafficStats struct {
	AppID               uint64             `json:"app_id"`
	TotalRequests       int64              `json:"total_requests"`
	SuccessRequests     int64              `json:"success_requests"`
	FailedRequests      int64              `json:"failed_requests"`
	RateLimitedCount    int64              `json:"rate_limited_count"`
	CircuitBreakerOpen  bool               `json:"circuit_breaker_open"`
	AvgLatencyMs        float64            `json:"avg_latency_ms"`
	P99LatencyMs        float64            `json:"p99_latency_ms"`
	TrafficDistribution map[string]float64 `json:"traffic_distribution"`
}

// GetTrafficStats 获取流量统计（从 Prometheus 查询）
func (s *IstioService) GetTrafficStats(ctx context.Context, appID uint64, namespace, serviceName string) (*TrafficStats, error) {
	stats := &TrafficStats{
		AppID:               appID,
		TrafficDistribution: make(map[string]float64),
	}

	// 从数据库获取熔断状态
	var cbRules []models.TrafficCircuitBreakerRule
	s.db.Where("app_id = ? AND enabled = ?", appID, true).Find(&cbRules)
	for _, rule := range cbRules {
		if rule.CircuitStatus == "open" {
			stats.CircuitBreakerOpen = true
			break
		}
	}

	// 从数据库获取限流统计
	var rateLimitLogs []models.TrafficOperationLog
	s.db.Where("app_id = ? AND rule_type = ? AND operation = ?", appID, "ratelimit", "rejected").
		Order("created_at DESC").Limit(1000).Find(&rateLimitLogs)
	stats.RateLimitedCount = int64(len(rateLimitLogs))

	// 获取路由流量分布
	var routingRules []models.TrafficRoutingRule
	s.db.Where("app_id = ? AND enabled = ?", appID, true).Find(&routingRules)
	for _, rule := range routingRules {
		if rule.RouteType == "weight" && len(rule.Destinations) > 0 {
			for _, dest := range rule.Destinations {
				stats.TrafficDistribution[dest.Subset] = float64(dest.Weight)
			}
		}
	}

	return stats, nil
}

// RecordRateLimitEvent 记录限流事件
func (s *IstioService) RecordRateLimitEvent(appID, ruleID uint64, operator string) error {
	log := models.TrafficOperationLog{
		AppID:     appID,
		RuleType:  "ratelimit",
		RuleID:    ruleID,
		Operation: "rejected",
		Operator:  operator,
	}
	return s.db.Create(&log).Error
}

// UpdateCircuitBreakerStatus 更新熔断状态
func (s *IstioService) UpdateCircuitBreakerStatus(ruleID uint64, status string) error {
	updates := map[string]interface{}{
		"circuit_status": status,
	}
	if status == "open" {
		updates["last_open_time"] = s.db.NowFunc()
	}
	return s.db.Model(&models.TrafficCircuitBreakerRule{}).Where("id = ?", ruleID).Updates(updates).Error
}

// GetCircuitBreakerStatus 获取熔断状态
func (s *IstioService) GetCircuitBreakerStatus(appID uint64) ([]map[string]interface{}, error) {
	var rules []models.TrafficCircuitBreakerRule
	s.db.Where("app_id = ?", appID).Find(&rules)

	result := make([]map[string]interface{}, 0)
	for _, rule := range rules {
		result = append(result, map[string]interface{}{
			"id":             rule.ID,
			"name":           rule.Name,
			"resource":       rule.Resource,
			"status":         rule.CircuitStatus,
			"enabled":        rule.Enabled,
			"last_open_time": rule.LastOpenTime,
		})
	}
	return result, nil
}
