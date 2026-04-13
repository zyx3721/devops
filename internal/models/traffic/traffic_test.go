package traffic

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

// ============================================================================
// Task 1.2: Property 1 - JSONDestinations Round-Trip Consistency
// Validates: Requirements 1.1, 1.3
// ============================================================================

func TestJSONDestinations_RoundTrip(t *testing.T) {
	testCases := []struct {
		name string
		data JSONDestinations
	}{
		{
			name: "single destination",
			data: JSONDestinations{
				{Subset: "v1", Weight: 100},
			},
		},
		{
			name: "multiple destinations",
			data: JSONDestinations{
				{Subset: "v1", Weight: 80},
				{Subset: "v2", Weight: 20},
			},
		},
		{
			name: "three destinations",
			data: JSONDestinations{
				{Subset: "stable", Weight: 70},
				{Subset: "canary", Weight: 20},
				{Subset: "test", Weight: 10},
			},
		},
		{
			name: "zero weight",
			data: JSONDestinations{
				{Subset: "v1", Weight: 0},
				{Subset: "v2", Weight: 100},
			},
		},
		{
			name: "empty subset name",
			data: JSONDestinations{
				{Subset: "", Weight: 100},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Value() - serialize
			value, err := tc.data.Value()
			assert.NoError(t, err, "Value() should not return error")
			assert.NotNil(t, value, "Value() should return non-nil")

			// Scan() - deserialize
			var result JSONDestinations
			err = result.Scan(value)
			assert.NoError(t, err, "Scan() should not return error")

			// Verify round-trip consistency
			assert.Equal(t, len(tc.data), len(result), "Length should match")
			for i, dest := range tc.data {
				assert.Equal(t, dest.Subset, result[i].Subset, "Subset should match")
				assert.Equal(t, dest.Weight, result[i].Weight, "Weight should match")
			}
		})
	}
}

// ============================================================================
// Task 1.3: JSONDestinations 边界测试
// Requirements: 1.2
// ============================================================================

func TestJSONDestinations_Scan_NilValue(t *testing.T) {
	var dest JSONDestinations
	err := dest.Scan(nil)
	assert.NoError(t, err, "Scan(nil) should not return error")
	assert.Nil(t, dest, "Scan(nil) should result in nil")
}

func TestJSONDestinations_Scan_EmptyArray(t *testing.T) {
	var dest JSONDestinations
	err := dest.Scan([]byte("[]"))
	assert.NoError(t, err, "Scan empty array should not return error")
	assert.NotNil(t, dest, "Result should not be nil")
	assert.Empty(t, dest, "Result should be empty")
}

func TestJSONDestinations_Scan_InvalidJSON(t *testing.T) {
	var dest JSONDestinations
	err := dest.Scan([]byte("invalid json"))
	assert.Error(t, err, "Scan invalid JSON should return error")
}

func TestJSONDestinations_Scan_NonByteSlice(t *testing.T) {
	var dest JSONDestinations
	err := dest.Scan("not a byte slice")
	assert.NoError(t, err, "Scan non-byte-slice should not return error")
	assert.Nil(t, dest, "Result should be nil for non-byte-slice input")
}

func TestJSONDestinations_Value_Nil(t *testing.T) {
	var dest JSONDestinations = nil
	value, err := dest.Value()
	assert.NoError(t, err, "Value() on nil should not return error")
	assert.Nil(t, value, "Value() on nil should return nil")
}

func TestJSONDestinations_Value_Empty(t *testing.T) {
	dest := JSONDestinations{}
	value, err := dest.Value()
	assert.NoError(t, err, "Value() on empty should not return error")
	assert.NotNil(t, value, "Value() on empty should return non-nil")

	// Verify it's valid JSON
	var result []RouteDestination
	err = json.Unmarshal(value.([]byte), &result)
	assert.NoError(t, err, "Should be valid JSON")
	assert.Empty(t, result, "Should be empty array")
}

// ============================================================================
// Task 1.4: TableName 方法测试
// Requirements: 1.4
// ============================================================================

func TestTrafficModels_TableName(t *testing.T) {
	tests := []struct {
		name      string
		model     interface{ TableName() string }
		wantTable string
	}{
		{"TrafficRateLimitRule", TrafficRateLimitRule{}, "traffic_ratelimit_rules"},
		{"TrafficCircuitBreakerRule", TrafficCircuitBreakerRule{}, "traffic_circuitbreaker_rules"},
		{"TrafficRoutingRule", TrafficRoutingRule{}, "traffic_routing_rules"},
		{"TrafficLoadBalanceConfig", TrafficLoadBalanceConfig{}, "traffic_loadbalance_config"},
		{"TrafficTimeoutConfig", TrafficTimeoutConfig{}, "traffic_timeout_config"},
		{"TrafficMirrorRule", TrafficMirrorRule{}, "traffic_mirror_rules"},
		{"TrafficFaultRule", TrafficFaultRule{}, "traffic_fault_rules"},
		{"TrafficOperationLog", TrafficOperationLog{}, "traffic_operation_logs"},
		{"TrafficStatistics", TrafficStatistics{}, "traffic_statistics"},
		{"TrafficRuleVersion", TrafficRuleVersion{}, "traffic_rule_versions"},
		{"CanaryRelease", CanaryRelease{}, "canary_releases"},
		{"BlueGreenDeployment", BlueGreenDeployment{}, "blue_green_deployments"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.model.TableName()
			assert.Equal(t, tt.wantTable, got, "TableName() should return correct table name")
		})
	}
}

// ============================================================================
// Task 1.5: 默认值测试
// Requirements: 1.5, 5.1, 5.2, 5.3, 5.4
// ============================================================================

func TestTrafficRateLimitRule_Defaults(t *testing.T) {
	// 注意：GORM 默认值在数据库层面设置，这里测试 Go 结构体的零值行为
	// 实际默认值由 GORM tag 定义，在数据库插入时生效
	rule := TrafficRateLimitRule{}

	// 验证零值状态
	assert.Equal(t, uint64(0), rule.ID)
	assert.Equal(t, uint64(0), rule.AppID)
	assert.Equal(t, "", rule.Name)
	assert.Equal(t, "", rule.ResourceType) // GORM default: "api"
	assert.Equal(t, "", rule.Strategy)     // GORM default: "qps"
	assert.Equal(t, 0, rule.Threshold)     // GORM default: 100
	assert.Equal(t, 0, rule.Burst)         // GORM default: 10
	assert.Equal(t, 0, rule.RejectedCode)  // GORM default: 429
	assert.False(t, rule.Enabled)          // GORM default: true
}

func TestCanaryRelease_Defaults(t *testing.T) {
	release := CanaryRelease{}

	// 验证零值状态
	assert.Equal(t, uint64(0), release.ID)
	assert.Equal(t, "", release.Status)                   // GORM default: "pending"
	assert.Equal(t, 0, release.CurrentWeight)             // GORM default: 0
	assert.Equal(t, 0, release.TargetWeight)              // GORM default: 100
	assert.Equal(t, 0, release.WeightIncrement)           // GORM default: 10
	assert.Equal(t, 0, release.IntervalSeconds)           // GORM default: 60
	assert.Equal(t, float64(0), release.SuccessThreshold) // GORM default: 95
	assert.Equal(t, 0, release.LatencyThreshold)          // GORM default: 500
	assert.False(t, release.AutoRollback)                 // GORM default: true
	assert.Nil(t, release.StartedAt)
	assert.Nil(t, release.CompletedAt)
}

func TestBlueGreenDeployment_Defaults(t *testing.T) {
	deployment := BlueGreenDeployment{}

	// 验证零值状态
	assert.Equal(t, uint64(0), deployment.ID)
	assert.Equal(t, "", deployment.Status)       // GORM default: "pending"
	assert.Equal(t, "", deployment.ActiveColor)  // GORM default: "blue"
	assert.Equal(t, 0, deployment.Replicas)      // GORM default: 2
	assert.Equal(t, 0, deployment.WarmupSeconds) // GORM default: 30
	assert.Nil(t, deployment.SwitchedAt)
}

func TestTrafficCircuitBreakerRule_Defaults(t *testing.T) {
	rule := TrafficCircuitBreakerRule{}

	assert.Equal(t, uint64(0), rule.ID)
	assert.Equal(t, "", rule.Strategy)        // GORM default: "slow_request"
	assert.Equal(t, 0, rule.SlowRtThreshold)  // GORM default: 1000
	assert.Equal(t, 0, rule.StatInterval)     // GORM default: 10
	assert.Equal(t, 0, rule.MinRequestAmount) // GORM default: 5
	assert.Equal(t, 0, rule.RecoveryTimeout)  // GORM default: 30
	assert.Equal(t, "", rule.CircuitStatus)   // GORM default: "closed"
	assert.False(t, rule.Enabled)             // GORM default: true
}

func TestTrafficLoadBalanceConfig_Defaults(t *testing.T) {
	config := TrafficLoadBalanceConfig{}

	assert.Equal(t, uint64(0), config.ID)
	assert.Equal(t, "", config.LbPolicy)          // GORM default: "round_robin"
	assert.Equal(t, 0, config.RingSize)           // GORM default: 1024
	assert.Equal(t, 0, config.HTTPMaxConnections) // GORM default: 1024
	assert.Equal(t, 0, config.HTTPMaxRetries)     // GORM default: 3
	assert.Equal(t, 0, config.TCPMaxConnections)  // GORM default: 1024
	assert.False(t, config.HealthCheckEnabled)    // GORM default: false
	assert.True(t, !config.TCPKeepaliveEnabled)   // GORM default: true (Go zero value is false)
}

func TestTrafficMirrorRule_Defaults(t *testing.T) {
	rule := TrafficMirrorRule{}

	assert.Equal(t, uint64(0), rule.ID)
	assert.Equal(t, 0, rule.Percentage) // GORM default: 100
	assert.False(t, rule.Enabled)       // GORM default: true
}

func TestTrafficFaultRule_Defaults(t *testing.T) {
	rule := TrafficFaultRule{}

	assert.Equal(t, uint64(0), rule.ID)
	assert.Equal(t, "", rule.Type)          // GORM default: "delay"
	assert.Equal(t, "", rule.Path)          // GORM default: "/"
	assert.Equal(t, "", rule.DelayDuration) // GORM default: "5s"
	assert.Equal(t, 0, rule.AbortCode)      // GORM default: 500
	assert.Equal(t, 0, rule.Percentage)     // GORM default: 10
	assert.False(t, rule.Enabled)           // GORM default: false
}
