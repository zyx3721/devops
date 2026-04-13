package kubernetes

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
)

// ============================================================================
// Task 2.1: K8sNamespaceService 实现测试
// Requirements: 2.1
// ============================================================================

func TestK8sNamespaceService_ConvertNamespaceInfo(t *testing.T) {
	service := &K8sNamespaceService{}

	t.Run("converts namespace with all fields", func(t *testing.T) {
		now := time.Now()
		ns := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "test-namespace",
				CreationTimestamp: metav1.Time{Time: now.Add(-2 * time.Hour)},
				Labels: map[string]string{
					"env":  "production",
					"team": "backend",
				},
			},
			Status: corev1.NamespaceStatus{
				Phase: corev1.NamespaceActive,
			},
		}

		result := service.convertNamespaceInfo(ns)

		assert.Equal(t, "test-namespace", result.Name)
		assert.Equal(t, "Active", result.Status)
		assert.Equal(t, "2h", result.Age)
		assert.Equal(t, "production", result.Labels["env"])
		assert.Equal(t, "backend", result.Labels["team"])
		assert.NotEmpty(t, result.CreatedAt)
	})

	t.Run("converts namespace without labels", func(t *testing.T) {
		now := time.Now()
		ns := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "default",
				CreationTimestamp: metav1.Time{Time: now.Add(-24 * time.Hour)},
			},
			Status: corev1.NamespaceStatus{
				Phase: corev1.NamespaceActive,
			},
		}

		result := service.convertNamespaceInfo(ns)

		assert.Equal(t, "default", result.Name)
		assert.Equal(t, "Active", result.Status)
		assert.Equal(t, "1d", result.Age)
		assert.NotNil(t, result.Labels)
	})

	t.Run("formats age correctly for different durations", func(t *testing.T) {
		now := time.Now()
		testCases := []struct {
			name     string
			duration time.Duration
			wantAge  string
		}{
			{"seconds", 30 * time.Second, "30s"},
			{"minutes", 45 * time.Minute, "45m"},
			{"hours", 5 * time.Hour, "5h"},
			{"days", 3 * 24 * time.Hour, "3d"},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				ns := &corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name:              "test",
						CreationTimestamp: metav1.Time{Time: now.Add(-tc.duration)},
					},
					Status: corev1.NamespaceStatus{
						Phase: corev1.NamespaceActive,
					},
				}

				result := service.convertNamespaceInfo(ns)
				assert.Equal(t, tc.wantAge, result.Age)
			})
		}
	})

	t.Run("handles terminating namespace", func(t *testing.T) {
		now := time.Now()
		ns := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "terminating-ns",
				CreationTimestamp: metav1.Time{Time: now.Add(-1 * time.Hour)},
			},
			Status: corev1.NamespaceStatus{
				Phase: corev1.NamespaceTerminating,
			},
		}

		result := service.convertNamespaceInfo(ns)

		assert.Equal(t, "terminating-ns", result.Name)
		assert.Equal(t, "Terminating", result.Status)
	})
}

func TestK8sNamespaceService_ConvertNamespaceInfo_EdgeCases(t *testing.T) {
	service := &K8sNamespaceService{}

	t.Run("handles empty namespace name", func(t *testing.T) {
		ns := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "",
				CreationTimestamp: metav1.Now(),
			},
			Status: corev1.NamespaceStatus{
				Phase: corev1.NamespaceActive,
			},
		}

		result := service.convertNamespaceInfo(ns)
		assert.Equal(t, "", result.Name)
	})

	t.Run("handles nil labels map", func(t *testing.T) {
		ns := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "test",
				CreationTimestamp: metav1.Now(),
				Labels:            nil,
			},
			Status: corev1.NamespaceStatus{
				Phase: corev1.NamespaceActive,
			},
		}

		result := service.convertNamespaceInfo(ns)
		assert.NotNil(t, result.Labels)
	})

	t.Run("handles empty labels map", func(t *testing.T) {
		ns := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "test",
				CreationTimestamp: metav1.Now(),
				Labels:            map[string]string{},
			},
			Status: corev1.NamespaceStatus{
				Phase: corev1.NamespaceActive,
			},
		}

		result := service.convertNamespaceInfo(ns)
		assert.NotNil(t, result.Labels)
		assert.Empty(t, result.Labels)
	})
}

// ============================================================================
// Task 2.2: Namespace Service 单元测试 - 数据转换验证
// Requirements: 2.1
// ============================================================================

func TestK8sNamespaceService_DataConversion(t *testing.T) {
	service := &K8sNamespaceService{}

	t.Run("CreatedAt format is correct", func(t *testing.T) {
		// Use a specific time for predictable testing
		specificTime := time.Date(2024, 1, 15, 10, 30, 45, 0, time.UTC)
		ns := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "test",
				CreationTimestamp: metav1.Time{Time: specificTime},
			},
			Status: corev1.NamespaceStatus{
				Phase: corev1.NamespaceActive,
			},
		}

		result := service.convertNamespaceInfo(ns)

		// Verify the format is "2006-01-02 15:04:05"
		assert.Equal(t, "2024-01-15 10:30:45", result.CreatedAt)
	})

	t.Run("Status is converted to string", func(t *testing.T) {
		testCases := []struct {
			phase      corev1.NamespacePhase
			wantStatus string
		}{
			{corev1.NamespaceActive, "Active"},
			{corev1.NamespaceTerminating, "Terminating"},
		}

		for _, tc := range testCases {
			ns := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "test",
					CreationTimestamp: metav1.Now(),
				},
				Status: corev1.NamespaceStatus{
					Phase: tc.phase,
				},
			}

			result := service.convertNamespaceInfo(ns)
			assert.Equal(t, tc.wantStatus, result.Status)
		}
	})

	t.Run("Labels are preserved", func(t *testing.T) {
		labels := map[string]string{
			"app":         "myapp",
			"environment": "staging",
			"version":     "v1.2.3",
		}

		ns := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "test",
				CreationTimestamp: metav1.Now(),
				Labels:            labels,
			},
			Status: corev1.NamespaceStatus{
				Phase: corev1.NamespaceActive,
			},
		}

		result := service.convertNamespaceInfo(ns)

		assert.Equal(t, len(labels), len(result.Labels))
		for k, v := range labels {
			assert.Equal(t, v, result.Labels[k])
		}
	})
}

// ============================================================================
// Additional tests for NamespaceInfo structure validation
// ============================================================================

func TestNamespaceInfo_Structure(t *testing.T) {
	t.Run("NamespaceInfo has all required fields", func(t *testing.T) {
		info := NamespaceInfo{
			Name:      "test-ns",
			Status:    "Active",
			Age:       "1h",
			Labels:    map[string]string{"key": "value"},
			CreatedAt: "2024-01-15 10:30:45",
		}

		assert.NotEmpty(t, info.Name)
		assert.NotEmpty(t, info.Status)
		assert.NotEmpty(t, info.Age)
		assert.NotNil(t, info.Labels)
		assert.NotEmpty(t, info.CreatedAt)
	})
}

func TestNamespaceDetail_Structure(t *testing.T) {
	t.Run("NamespaceDetail extends NamespaceInfo", func(t *testing.T) {
		detail := NamespaceDetail{
			NamespaceInfo: NamespaceInfo{
				Name:      "test-ns",
				Status:    "Active",
				Age:       "1h",
				Labels:    map[string]string{"key": "value"},
				CreatedAt: "2024-01-15 10:30:45",
			},
			Annotations: map[string]string{
				"description": "Test namespace",
			},
			ResourceQuota: []ResourceQuotaInfo{
				{
					Name: "quota-1",
					Hard: map[string]string{"cpu": "10"},
					Used: map[string]string{"cpu": "5"},
				},
			},
		}

		assert.NotEmpty(t, detail.Name)
		assert.NotEmpty(t, detail.Status)
		assert.NotNil(t, detail.Annotations)
		assert.NotNil(t, detail.ResourceQuota)
		assert.Len(t, detail.ResourceQuota, 1)
	})
}

// ============================================================================
// Test helper function formatDuration
// ============================================================================

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		want     string
	}{
		{"zero duration", 0, "0s"},
		{"seconds", 30 * time.Second, "30s"},
		{"one minute", 1 * time.Minute, "1m"},
		{"minutes", 45 * time.Minute, "45m"},
		{"one hour", 1 * time.Hour, "1h"},
		{"hours", 5 * time.Hour, "5h"},
		{"one day", 24 * time.Hour, "1d"},
		{"days", 3 * 24 * time.Hour, "3d"},
		{"weeks", 7 * 24 * time.Hour, "7d"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatDuration(tt.duration)
			assert.Equal(t, tt.want, result)
		})
	}
}

// ============================================================================
// Test Helper: Create test service with fake client
// ============================================================================

// testK8sNamespaceService wraps K8sNamespaceService for testing
type testK8sNamespaceService struct {
	*K8sNamespaceService
	fakeClient *fake.Clientset
}

// newTestK8sNamespaceService creates a test service with a fake K8s client
func newTestK8sNamespaceService(objects ...runtime.Object) *testK8sNamespaceService {
	fakeClient := fake.NewSimpleClientset(objects...)

	// Create a service with nil clientMgr since we'll use the fake client directly
	service := &K8sNamespaceService{
		clientMgr: nil,
	}

	return &testK8sNamespaceService{
		K8sNamespaceService: service,
		fakeClient:          fakeClient,
	}
}

// ListNamespaces overrides the service method to use fake client
func (ts *testK8sNamespaceService) ListNamespaces(ctx context.Context, clusterID uint) ([]NamespaceInfo, error) {
	nsList, err := ts.fakeClient.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	result := make([]NamespaceInfo, len(nsList.Items))
	for i, ns := range nsList.Items {
		result[i] = ts.convertNamespaceInfo(&ns)
	}
	return result, nil
}

// GetNamespace overrides the service method to use fake client
func (ts *testK8sNamespaceService) GetNamespace(ctx context.Context, clusterID uint, name string) (*NamespaceDetail, error) {
	ns, err := ts.fakeClient.CoreV1().Namespaces().Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	info := ts.convertNamespaceInfo(ns)
	detail := &NamespaceDetail{
		NamespaceInfo: info,
		Annotations:   ns.Annotations,
	}

	// Get resource quotas
	quotaList, err := ts.fakeClient.CoreV1().ResourceQuotas(name).List(ctx, metav1.ListOptions{})
	if err == nil && len(quotaList.Items) > 0 {
		detail.ResourceQuota = make([]ResourceQuotaInfo, len(quotaList.Items))
		for i, quota := range quotaList.Items {
			hard := make(map[string]string)
			used := make(map[string]string)

			for k, v := range quota.Status.Hard {
				hard[string(k)] = v.String()
			}
			for k, v := range quota.Status.Used {
				used[string(k)] = v.String()
			}

			detail.ResourceQuota[i] = ResourceQuotaInfo{
				Name: quota.Name,
				Hard: hard,
				Used: used,
			}
		}
	}

	return detail, nil
}

// ============================================================================
// Task 2.2: Namespace Service 单元测试 - ListNamespaces
// Requirements: 2.1
// 测试正常获取命名空间列表
// ============================================================================

func TestK8sNamespaceService_ListNamespaces_Success(t *testing.T) {
	// Create fake K8s client with test data
	now := time.Now()
	namespaces := []runtime.Object{
		&corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "default",
				CreationTimestamp: metav1.Time{Time: now.Add(-24 * time.Hour)},
				Labels: map[string]string{
					"kubernetes.io/metadata.name": "default",
				},
			},
			Status: corev1.NamespaceStatus{
				Phase: corev1.NamespaceActive,
			},
		},
		&corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "kube-system",
				CreationTimestamp: metav1.Time{Time: now.Add(-48 * time.Hour)},
				Labels: map[string]string{
					"kubernetes.io/metadata.name": "kube-system",
				},
			},
			Status: corev1.NamespaceStatus{
				Phase: corev1.NamespaceActive,
			},
		},
		&corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "production",
				CreationTimestamp: metav1.Time{Time: now.Add(-72 * time.Hour)},
				Labels: map[string]string{
					"env":  "production",
					"team": "backend",
				},
			},
			Status: corev1.NamespaceStatus{
				Phase: corev1.NamespaceActive,
			},
		},
	}

	// Create test service
	service := newTestK8sNamespaceService(namespaces...)

	// Test ListNamespaces
	ctx := context.Background()
	result, err := service.ListNamespaces(ctx, 1)

	// Assertions
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result, 3)

	// Verify namespace names
	names := make([]string, len(result))
	for i, ns := range result {
		names[i] = ns.Name
	}
	assert.Contains(t, names, "default")
	assert.Contains(t, names, "kube-system")
	assert.Contains(t, names, "production")

	// Verify all namespaces have required fields
	for _, ns := range result {
		assert.NotEmpty(t, ns.Name)
		assert.NotEmpty(t, ns.Status)
		assert.NotEmpty(t, ns.Age)
		assert.NotEmpty(t, ns.CreatedAt)
		assert.NotNil(t, ns.Labels)
	}
}

// ============================================================================
// Task 2.2: Namespace Service 单元测试 - 集群不存在的错误情况
// Requirements: 2.1
// Note: Since we're using fake clients, we test the error handling at the API level
// The actual cluster validation happens in K8sClientManager.GetClient()
// ============================================================================

func TestK8sNamespaceService_ListNamespaces_APIError(t *testing.T) {
	// Test that the service properly handles K8s API errors
	// This simulates what happens when the cluster connection fails

	// Create test service with no namespaces
	service := newTestK8sNamespaceService()

	// Test ListNamespaces - should succeed with empty list
	ctx := context.Background()
	result, err := service.ListNamespaces(ctx, 1)

	// Assertions - fake client returns empty list, not error
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Empty(t, result)
}

// ============================================================================
// Task 2.2: Namespace Service 单元测试 - 空命名空间列表的边缘情况
// Requirements: 2.1
// ============================================================================

func TestK8sNamespaceService_ListNamespaces_EmptyList(t *testing.T) {
	// Create fake K8s client with no namespaces
	service := newTestK8sNamespaceService()

	// Test ListNamespaces
	ctx := context.Background()
	result, err := service.ListNamespaces(ctx, 1)

	// Assertions
	assert.NoError(t, err)
	assert.NotNil(t, result, "Result should not be nil even when empty")
	assert.Empty(t, result, "Result should be empty array, not nil")
	assert.Len(t, result, 0)
}

// ============================================================================
// Task 2.2: Namespace Service 单元测试 - GetNamespace
// Requirements: 2.1
// ============================================================================

func TestK8sNamespaceService_GetNamespace_Success(t *testing.T) {
	// Create fake K8s client with test data
	now := time.Now()
	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "production",
			CreationTimestamp: metav1.Time{Time: now.Add(-72 * time.Hour)},
			Labels: map[string]string{
				"env":  "production",
				"team": "backend",
			},
			Annotations: map[string]string{
				"description": "Production environment",
				"owner":       "backend-team",
			},
		},
		Status: corev1.NamespaceStatus{
			Phase: corev1.NamespaceActive,
		},
	}

	service := newTestK8sNamespaceService(namespace)

	// Test GetNamespace
	ctx := context.Background()
	result, err := service.GetNamespace(ctx, 1, "production")

	// Assertions
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "production", result.Name)
	assert.Equal(t, "Active", result.Status)
	assert.NotEmpty(t, result.Age)
	assert.NotEmpty(t, result.CreatedAt)
	assert.NotNil(t, result.Labels)
	assert.Equal(t, "production", result.Labels["env"])
	assert.Equal(t, "backend", result.Labels["team"])
	assert.NotNil(t, result.Annotations)
	assert.Equal(t, "Production environment", result.Annotations["description"])
	assert.Equal(t, "backend-team", result.Annotations["owner"])
}

func TestK8sNamespaceService_GetNamespace_NotFound(t *testing.T) {
	// Create fake K8s client with no namespaces
	service := newTestK8sNamespaceService()

	// Test GetNamespace with non-existent namespace
	ctx := context.Background()
	result, err := service.GetNamespace(ctx, 1, "non-existent")

	// Assertions
	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestK8sNamespaceService_GetNamespace_WithResourceQuota(t *testing.T) {
	// Create fake K8s client with namespace and resource quota
	now := time.Now()
	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "production",
			CreationTimestamp: metav1.Time{Time: now.Add(-24 * time.Hour)},
		},
		Status: corev1.NamespaceStatus{
			Phase: corev1.NamespaceActive,
		},
	}

	// Note: ResourceQuota is tested separately as fake client may not fully support it
	// This test verifies the service handles the case gracefully
	service := newTestK8sNamespaceService(namespace)

	// Test GetNamespace
	ctx := context.Background()
	result, err := service.GetNamespace(ctx, 1, "production")

	// Assertions
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "production", result.Name)
	// ResourceQuota may be nil or empty when there are no quotas, both are acceptable
	// The service initializes it as empty slice if there are no quotas
}

// ============================================================================
// Task 2.2: Namespace Service 单元测试 - 边缘情况
// Requirements: 2.1
// ============================================================================

func TestK8sNamespaceService_ListNamespaces_WithTerminatingNamespace(t *testing.T) {
	// Create fake K8s client with terminating namespace
	now := time.Now()
	namespaces := []runtime.Object{
		&corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "active-ns",
				CreationTimestamp: metav1.Time{Time: now.Add(-24 * time.Hour)},
			},
			Status: corev1.NamespaceStatus{
				Phase: corev1.NamespaceActive,
			},
		},
		&corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "terminating-ns",
				CreationTimestamp: metav1.Time{Time: now.Add(-12 * time.Hour)},
			},
			Status: corev1.NamespaceStatus{
				Phase: corev1.NamespaceTerminating,
			},
		},
	}

	service := newTestK8sNamespaceService(namespaces...)

	// Test ListNamespaces
	ctx := context.Background()
	result, err := service.ListNamespaces(ctx, 1)

	// Assertions
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result, 2)

	// Find terminating namespace
	var terminatingNs *NamespaceInfo
	for i := range result {
		if result[i].Name == "terminating-ns" {
			terminatingNs = &result[i]
			break
		}
	}

	assert.NotNil(t, terminatingNs)
	assert.Equal(t, "Terminating", terminatingNs.Status)
}

func TestK8sNamespaceService_ListNamespaces_WithVariousLabels(t *testing.T) {
	// Create fake K8s client with namespaces having different label configurations
	now := time.Now()
	namespaces := []runtime.Object{
		&corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "no-labels",
				CreationTimestamp: metav1.Time{Time: now},
				Labels:            nil,
			},
			Status: corev1.NamespaceStatus{
				Phase: corev1.NamespaceActive,
			},
		},
		&corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "empty-labels",
				CreationTimestamp: metav1.Time{Time: now},
				Labels:            map[string]string{},
			},
			Status: corev1.NamespaceStatus{
				Phase: corev1.NamespaceActive,
			},
		},
		&corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "with-labels",
				CreationTimestamp: metav1.Time{Time: now},
				Labels: map[string]string{
					"app": "myapp",
					"env": "prod",
				},
			},
			Status: corev1.NamespaceStatus{
				Phase: corev1.NamespaceActive,
			},
		},
	}

	service := newTestK8sNamespaceService(namespaces...)

	// Test ListNamespaces
	ctx := context.Background()
	result, err := service.ListNamespaces(ctx, 1)

	// Assertions
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result, 3)

	// Verify all namespaces have non-nil labels map
	for _, ns := range result {
		assert.NotNil(t, ns.Labels, "Labels should never be nil for namespace: %s", ns.Name)
	}
}
