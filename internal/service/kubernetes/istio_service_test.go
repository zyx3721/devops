package kubernetes

import (
	"devops/internal/models"
	"testing"

	"github.com/stretchr/testify/assert"
)

// ============================================================================
// Task 3.2: buildMatchCondition 测试
// Requirements: 2.2
// ============================================================================

func TestIstioService_BuildMatchCondition(t *testing.T) {
	service := &IstioService{}

	tests := []struct {
		name     string
		operator string
		value    string
		wantKey  string
		wantVal  string
	}{
		{
			name:     "exact operator",
			operator: "exact",
			value:    "test-value",
			wantKey:  "exact",
			wantVal:  "test-value",
		},
		{
			name:     "prefix operator",
			operator: "prefix",
			value:    "/api/v1",
			wantKey:  "prefix",
			wantVal:  "/api/v1",
		},
		{
			name:     "regex operator",
			operator: "regex",
			value:    "^/api/.*",
			wantKey:  "regex",
			wantVal:  "^/api/.*",
		},
		{
			name:     "present operator",
			operator: "present",
			value:    "any",
			wantKey:  "regex",
			wantVal:  ".*",
		},
		{
			name:     "default operator (unknown)",
			operator: "unknown",
			value:    "test",
			wantKey:  "exact",
			wantVal:  "test",
		},
		{
			name:     "empty operator defaults to exact",
			operator: "",
			value:    "empty-op",
			wantKey:  "exact",
			wantVal:  "empty-op",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.buildMatchCondition(tt.operator, tt.value)
			assert.NotNil(t, result)
			assert.Contains(t, result, tt.wantKey)
			assert.Equal(t, tt.wantVal, result[tt.wantKey])
		})
	}
}

// ============================================================================
// Task 3.3: Property 2 - VirtualService Builder Validity
// Validates: Requirements 2.1
// ============================================================================

func TestIstioService_BuildVirtualService(t *testing.T) {
	service := &IstioService{}

	t.Run("empty rules creates default route", func(t *testing.T) {
		vs := service.buildVirtualService("test-vs", "default", "test-service", []models.TrafficRoutingRule{})

		assert.NotNil(t, vs)
		assert.Equal(t, "networking.istio.io/v1beta1", vs.Object["apiVersion"])
		assert.Equal(t, "VirtualService", vs.Object["kind"])

		metadata := vs.Object["metadata"].(map[string]interface{})
		assert.Equal(t, "test-vs", metadata["name"])
		assert.Equal(t, "default", metadata["namespace"])

		spec := vs.Object["spec"].(map[string]interface{})
		hosts := spec["hosts"].([]interface{})
		assert.Contains(t, hosts, "test-service")

		httpRoutes := spec["http"].([]interface{})
		assert.Len(t, httpRoutes, 1, "Should have default route")
	})

	t.Run("disabled rules are skipped", func(t *testing.T) {
		rules := []models.TrafficRoutingRule{
			{Name: "disabled-rule", Enabled: false, RouteType: "header"},
		}
		vs := service.buildVirtualService("test-vs", "default", "test-service", rules)

		spec := vs.Object["spec"].(map[string]interface{})
		httpRoutes := spec["http"].([]interface{})
		// Should only have default route since the rule is disabled
		assert.Len(t, httpRoutes, 1)
	})

	t.Run("weight routing with destinations", func(t *testing.T) {
		rules := []models.TrafficRoutingRule{
			{
				Name:      "weight-rule",
				Enabled:   true,
				RouteType: "weight",
				Destinations: models.JSONDestinations{
					{Subset: "v1", Weight: 80},
					{Subset: "v2", Weight: 20},
				},
			},
		}
		vs := service.buildVirtualService("test-vs", "default", "test-service", rules)

		spec := vs.Object["spec"].(map[string]interface{})
		httpRoutes := spec["http"].([]interface{})
		assert.Len(t, httpRoutes, 1)

		route := httpRoutes[0].(map[string]interface{})
		assert.Equal(t, "weight-rule", route["name"])

		routeDests := route["route"].([]interface{})
		assert.Len(t, routeDests, 2)
	})

	t.Run("header routing", func(t *testing.T) {
		rules := []models.TrafficRoutingRule{
			{
				Name:          "header-rule",
				Enabled:       true,
				RouteType:     "header",
				MatchKey:      "x-version",
				MatchOperator: "exact",
				MatchValue:    "v2",
				TargetSubset:  "v2",
			},
		}
		vs := service.buildVirtualService("test-vs", "default", "test-service", rules)

		spec := vs.Object["spec"].(map[string]interface{})
		httpRoutes := spec["http"].([]interface{})
		assert.Len(t, httpRoutes, 1)

		route := httpRoutes[0].(map[string]interface{})
		match := route["match"].([]interface{})
		assert.Len(t, match, 1)

		matchRule := match[0].(map[string]interface{})
		headers := matchRule["headers"].(map[string]interface{})
		assert.Contains(t, headers, "x-version")
	})

	t.Run("cookie routing", func(t *testing.T) {
		rules := []models.TrafficRoutingRule{
			{
				Name:          "cookie-rule",
				Enabled:       true,
				RouteType:     "cookie",
				MatchOperator: "regex",
				MatchValue:    ".*canary.*",
				TargetSubset:  "canary",
			},
		}
		vs := service.buildVirtualService("test-vs", "default", "test-service", rules)

		spec := vs.Object["spec"].(map[string]interface{})
		httpRoutes := spec["http"].([]interface{})
		route := httpRoutes[0].(map[string]interface{})
		match := route["match"].([]interface{})
		matchRule := match[0].(map[string]interface{})
		headers := matchRule["headers"].(map[string]interface{})
		assert.Contains(t, headers, "cookie")
	})

	t.Run("param routing", func(t *testing.T) {
		rules := []models.TrafficRoutingRule{
			{
				Name:          "param-rule",
				Enabled:       true,
				RouteType:     "param",
				MatchKey:      "version",
				MatchOperator: "exact",
				MatchValue:    "beta",
				TargetSubset:  "beta",
			},
		}
		vs := service.buildVirtualService("test-vs", "default", "test-service", rules)

		spec := vs.Object["spec"].(map[string]interface{})
		httpRoutes := spec["http"].([]interface{})
		route := httpRoutes[0].(map[string]interface{})
		match := route["match"].([]interface{})
		matchRule := match[0].(map[string]interface{})
		assert.Contains(t, matchRule, "queryParams")
	})

	t.Run("labels are set correctly", func(t *testing.T) {
		vs := service.buildVirtualService("my-vs", "prod", "my-service", []models.TrafficRoutingRule{})

		metadata := vs.Object["metadata"].(map[string]interface{})
		labels := metadata["labels"].(map[string]interface{})
		assert.Equal(t, "my-service", labels["app"])
		assert.Equal(t, "devops-platform", labels["managed-by"])
	})
}

// ============================================================================
// Task 3.4: 负载均衡策略映射测试
// Requirements: 2.4
// ============================================================================

func TestIstioService_BuildDestinationRule_LoadBalancing(t *testing.T) {
	service := &IstioService{}

	tests := []struct {
		name       string
		lbPolicy   string
		wantSimple string
		wantHash   bool
	}{
		{"round_robin", "round_robin", "ROUND_ROBIN", false},
		{"random", "random", "RANDOM", false},
		{"least_request", "least_request", "LEAST_REQUEST", false},
		{"passthrough", "passthrough", "PASSTHROUGH", false},
		{"consistent_hash", "consistent_hash", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &models.TrafficLoadBalanceConfig{
				LbPolicy:    tt.lbPolicy,
				HashKey:     "header",
				HashKeyName: "x-user-id",
				RingSize:    1024,
			}

			dr := service.buildDestinationRule("test-dr", "default", "test-service", config, nil)

			assert.NotNil(t, dr)
			assert.Equal(t, "networking.istio.io/v1beta1", dr.Object["apiVersion"])
			assert.Equal(t, "DestinationRule", dr.Object["kind"])

			spec := dr.Object["spec"].(map[string]interface{})
			trafficPolicy := spec["trafficPolicy"].(map[string]interface{})

			if tt.wantHash {
				lb := trafficPolicy["loadBalancer"].(map[string]interface{})
				assert.Contains(t, lb, "consistentHash")
			} else if tt.wantSimple != "" {
				lb := trafficPolicy["loadBalancer"].(map[string]interface{})
				assert.Equal(t, tt.wantSimple, lb["simple"])
			}
		})
	}
}

func TestIstioService_BuildDestinationRule_ConsistentHash(t *testing.T) {
	service := &IstioService{}

	tests := []struct {
		name        string
		hashKey     string
		hashKeyName string
		wantField   string
	}{
		{"header hash", "header", "x-user-id", "httpHeaderName"},
		{"cookie hash", "cookie", "session-id", "httpCookie"},
		{"source_ip hash", "source_ip", "", "useSourceIp"},
		{"query_param hash", "query_param", "user", "httpQueryParameterName"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &models.TrafficLoadBalanceConfig{
				LbPolicy:    "consistent_hash",
				HashKey:     tt.hashKey,
				HashKeyName: tt.hashKeyName,
				RingSize:    2048,
			}

			dr := service.buildDestinationRule("test-dr", "default", "test-service", config, nil)

			spec := dr.Object["spec"].(map[string]interface{})
			trafficPolicy := spec["trafficPolicy"].(map[string]interface{})
			lb := trafficPolicy["loadBalancer"].(map[string]interface{})
			consistentHash := lb["consistentHash"].(map[string]interface{})

			assert.Contains(t, consistentHash, tt.wantField)
			assert.Equal(t, 2048, consistentHash["minimumRingSize"])
		})
	}
}

// ============================================================================
// Task 3.5: Property 3 - DestinationRule Builder Validity
// Validates: Requirements 2.3, 2.5
// ============================================================================

func TestIstioService_BuildDestinationRule_CircuitBreaker(t *testing.T) {
	service := &IstioService{}

	t.Run("circuit breaker with error_count strategy", func(t *testing.T) {
		cbRules := []models.TrafficCircuitBreakerRule{
			{
				Name:            "cb-rule",
				Enabled:         true,
				Strategy:        "error_count",
				Threshold:       10,
				RecoveryTimeout: 60,
				StatInterval:    30,
			},
		}

		dr := service.buildDestinationRule("test-dr", "default", "test-service", nil, cbRules)

		spec := dr.Object["spec"].(map[string]interface{})
		trafficPolicy := spec["trafficPolicy"].(map[string]interface{})
		assert.Contains(t, trafficPolicy, "outlierDetection")

		outlier := trafficPolicy["outlierDetection"].(map[string]interface{})
		assert.Equal(t, "60s", outlier["baseEjectionTime"])
		assert.Equal(t, "30s", outlier["interval"])
		assert.Equal(t, 10, outlier["consecutive5xxErrors"])
	})

	t.Run("circuit breaker with error_ratio strategy", func(t *testing.T) {
		cbRules := []models.TrafficCircuitBreakerRule{
			{
				Name:            "cb-rule",
				Enabled:         true,
				Strategy:        "error_ratio",
				Threshold:       0.5,
				RecoveryTimeout: 30,
				StatInterval:    10,
			},
		}

		dr := service.buildDestinationRule("test-dr", "default", "test-service", nil, cbRules)

		spec := dr.Object["spec"].(map[string]interface{})
		trafficPolicy := spec["trafficPolicy"].(map[string]interface{})
		outlier := trafficPolicy["outlierDetection"].(map[string]interface{})
		// error_ratio uses consecutive5xxErrors as approximation
		assert.Equal(t, 5, outlier["consecutive5xxErrors"])
	})

	t.Run("circuit breaker with slow_request strategy", func(t *testing.T) {
		cbRules := []models.TrafficCircuitBreakerRule{
			{
				Name:            "cb-rule",
				Enabled:         true,
				Strategy:        "slow_request",
				Threshold:       0.3,
				RecoveryTimeout: 45,
				StatInterval:    15,
			},
		}

		dr := service.buildDestinationRule("test-dr", "default", "test-service", nil, cbRules)

		spec := dr.Object["spec"].(map[string]interface{})
		trafficPolicy := spec["trafficPolicy"].(map[string]interface{})
		outlier := trafficPolicy["outlierDetection"].(map[string]interface{})
		// slow_request uses consecutiveGatewayErrors
		assert.Equal(t, 5, outlier["consecutiveGatewayErrors"])
	})

	t.Run("disabled circuit breaker is skipped", func(t *testing.T) {
		cbRules := []models.TrafficCircuitBreakerRule{
			{
				Name:     "disabled-cb",
				Enabled:  false,
				Strategy: "error_count",
			},
		}

		dr := service.buildDestinationRule("test-dr", "default", "test-service", nil, cbRules)

		spec := dr.Object["spec"].(map[string]interface{})
		trafficPolicy := spec["trafficPolicy"].(map[string]interface{})
		assert.NotContains(t, trafficPolicy, "outlierDetection")
	})

	t.Run("only first enabled rule is used", func(t *testing.T) {
		cbRules := []models.TrafficCircuitBreakerRule{
			{Name: "first", Enabled: true, Strategy: "error_count", Threshold: 5, RecoveryTimeout: 30, StatInterval: 10},
			{Name: "second", Enabled: true, Strategy: "error_count", Threshold: 10, RecoveryTimeout: 60, StatInterval: 20},
		}

		dr := service.buildDestinationRule("test-dr", "default", "test-service", nil, cbRules)

		spec := dr.Object["spec"].(map[string]interface{})
		trafficPolicy := spec["trafficPolicy"].(map[string]interface{})
		outlier := trafficPolicy["outlierDetection"].(map[string]interface{})
		// Should use first rule's values
		assert.Equal(t, "30s", outlier["baseEjectionTime"])
		assert.Equal(t, 5, outlier["consecutive5xxErrors"])
	})
}

func TestIstioService_BuildDestinationRule_ConnectionPool(t *testing.T) {
	service := &IstioService{}

	t.Run("HTTP connection pool settings", func(t *testing.T) {
		config := &models.TrafficLoadBalanceConfig{
			LbPolicy:               "round_robin",
			HTTPMaxConnections:     100,
			HTTPMaxPendingRequests: 50,
			HTTPMaxRequestsPerConn: 10,
			HTTPMaxRetries:         3,
			HTTPIdleTimeout:        "30s",
		}

		dr := service.buildDestinationRule("test-dr", "default", "test-service", config, nil)

		spec := dr.Object["spec"].(map[string]interface{})
		trafficPolicy := spec["trafficPolicy"].(map[string]interface{})
		connPool := trafficPolicy["connectionPool"].(map[string]interface{})
		httpPool := connPool["http"].(map[string]interface{})

		assert.Equal(t, 50, httpPool["http1MaxPendingRequests"])
		assert.Equal(t, 10, httpPool["maxRequestsPerConnection"])
		assert.Equal(t, 3, httpPool["maxRetries"])
		assert.Equal(t, "30s", httpPool["idleTimeout"])
	})

	t.Run("TCP connection pool settings", func(t *testing.T) {
		config := &models.TrafficLoadBalanceConfig{
			LbPolicy:          "round_robin",
			TCPMaxConnections: 500,
			TCPConnectTimeout: "5s",
		}

		dr := service.buildDestinationRule("test-dr", "default", "test-service", config, nil)

		spec := dr.Object["spec"].(map[string]interface{})
		trafficPolicy := spec["trafficPolicy"].(map[string]interface{})
		connPool := trafficPolicy["connectionPool"].(map[string]interface{})
		tcpPool := connPool["tcp"].(map[string]interface{})

		assert.Equal(t, 500, tcpPool["maxConnections"])
		assert.Equal(t, "5s", tcpPool["connectTimeout"])
	})
}

func TestIstioService_BuildDestinationRule_Metadata(t *testing.T) {
	service := &IstioService{}

	dr := service.buildDestinationRule("my-dr", "production", "my-service", nil, nil)

	metadata := dr.Object["metadata"].(map[string]interface{})
	assert.Equal(t, "my-dr", metadata["name"])
	assert.Equal(t, "production", metadata["namespace"])

	labels := metadata["labels"].(map[string]interface{})
	assert.Equal(t, "my-service", labels["app"])
	assert.Equal(t, "devops-platform", labels["managed-by"])

	spec := dr.Object["spec"].(map[string]interface{})
	assert.Equal(t, "my-service", spec["host"])
}
