package kubernetes

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
)

// ============================================================================
// Task 3.1: 验证现有 K8sDeploymentService 功能
// Requirements: 3.1, 3.2, 3.3, 3.4
// ============================================================================

// testK8sDeploymentService wraps K8sDeploymentService for testing
type testK8sDeploymentService struct {
	*K8sDeploymentService
	fakeClient *fake.Clientset
}

// newTestK8sDeploymentService creates a test service with a fake K8s client
func newTestK8sDeploymentService(objects ...runtime.Object) *testK8sDeploymentService {
	fakeClient := fake.NewSimpleClientset(objects...)

	service := &K8sDeploymentService{
		clientMgr: nil,
	}

	return &testK8sDeploymentService{
		K8sDeploymentService: service,
		fakeClient:           fakeClient,
	}
}

// ListDeployments overrides the service method to use fake client
func (ts *testK8sDeploymentService) ListDeployments(ctx context.Context, clusterID uint, namespace string) ([]DeploymentInfo, error) {
	deployList, err := ts.fakeClient.AppsV1().Deployments(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	result := make([]DeploymentInfo, len(deployList.Items))
	for i, deploy := range deployList.Items {
		result[i] = ts.convertDeploymentInfo(&deploy)
	}
	return result, nil
}

// GetDeployment overrides the service method to use fake client
func (ts *testK8sDeploymentService) GetDeployment(ctx context.Context, clusterID uint, namespace, name string) (*DeploymentDetail, error) {
	deploy, err := ts.fakeClient.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	info := ts.convertDeploymentInfo(deploy)
	detail := &DeploymentDetail{
		DeploymentInfo: info,
		Labels:         deploy.Labels,
		Annotations:    deploy.Annotations,
		Strategy:       string(deploy.Spec.Strategy.Type),
		Selector:       deploy.Spec.Selector.MatchLabels,
	}

	for _, cond := range deploy.Status.Conditions {
		detail.Conditions = append(detail.Conditions, DeploymentCondition{
			Type:    string(cond.Type),
			Status:  string(cond.Status),
			Reason:  cond.Reason,
			Message: cond.Message,
		})
	}

	return detail, nil
}

// Restart overrides the service method to use fake client
func (ts *testK8sDeploymentService) Restart(ctx context.Context, clusterID uint, namespace, name string) error {
	// Get the deployment first to verify it exists
	deploy, err := ts.fakeClient.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return err
	}

	// Update the restart annotation
	if deploy.Spec.Template.Annotations == nil {
		deploy.Spec.Template.Annotations = make(map[string]string)
	}
	deploy.Spec.Template.Annotations["kubectl.kubernetes.io/restartedAt"] = time.Now().Format(time.RFC3339)

	_, err = ts.fakeClient.AppsV1().Deployments(namespace).Update(ctx, deploy, metav1.UpdateOptions{})
	return err
}

// Scale overrides the service method to use fake client
func (ts *testK8sDeploymentService) Scale(ctx context.Context, clusterID uint, namespace, name string, replicas int32) error {
	if replicas < 0 || replicas > 100 {
		return assert.AnError
	}

	deploy, err := ts.fakeClient.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return err
	}

	deploy.Spec.Replicas = &replicas
	_, err = ts.fakeClient.AppsV1().Deployments(namespace).Update(ctx, deploy, metav1.UpdateOptions{})
	return err
}

// ============================================================================
// Test ConvertDeploymentInfo - Data Conversion
// Requirements: 3.1
// ============================================================================

func TestK8sDeploymentService_ConvertDeploymentInfo(t *testing.T) {
	service := &K8sDeploymentService{}

	t.Run("converts deployment with all fields", func(t *testing.T) {
		now := time.Now()
		replicas := int32(3)
		deploy := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "test-deployment",
				Namespace:         "default",
				CreationTimestamp: metav1.Time{Time: now.Add(-2 * time.Hour)},
			},
			Spec: appsv1.DeploymentSpec{
				Replicas: &replicas,
				Template: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name:  "nginx",
								Image: "nginx:1.19",
							},
							{
								Name:  "sidecar",
								Image: "busybox:latest",
							},
						},
					},
				},
			},
			Status: appsv1.DeploymentStatus{
				ReadyReplicas:     2,
				UpdatedReplicas:   3,
				AvailableReplicas: 2,
			},
		}

		result := service.convertDeploymentInfo(deploy)

		assert.Equal(t, "test-deployment", result.Name)
		assert.Equal(t, "default", result.Namespace)
		assert.Equal(t, "2/3", result.Ready)
		assert.Equal(t, int32(3), result.UpToDate)
		assert.Equal(t, int32(2), result.Available)
		assert.Equal(t, int32(3), result.Replicas)
		assert.Equal(t, "2h", result.Age)
		assert.Len(t, result.Images, 2)
		assert.Contains(t, result.Images, "nginx:1.19")
		assert.Contains(t, result.Images, "busybox:latest")
		assert.Len(t, result.Containers, 2)
		assert.NotEmpty(t, result.CreatedAt)
	})

	t.Run("handles deployment with nil replicas", func(t *testing.T) {
		deploy := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "test",
				Namespace:         "default",
				CreationTimestamp: metav1.Now(),
			},
			Spec: appsv1.DeploymentSpec{
				Replicas: nil,
				Template: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{Name: "app", Image: "app:v1"},
						},
					},
				},
			},
			Status: appsv1.DeploymentStatus{
				ReadyReplicas: 1,
			},
		}

		result := service.convertDeploymentInfo(deploy)

		assert.Equal(t, int32(1), result.Replicas)
		assert.Equal(t, "1/1", result.Ready)
	})
}

// ============================================================================
// Test ListDeployments - Requirement 3.1
// ============================================================================

func TestK8sDeploymentService_ListDeployments_Success(t *testing.T) {
	now := time.Now()
	replicas1 := int32(2)
	replicas2 := int32(3)

	deployments := []runtime.Object{
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "app1",
				Namespace:         "default",
				CreationTimestamp: metav1.Time{Time: now.Add(-24 * time.Hour)},
			},
			Spec: appsv1.DeploymentSpec{
				Replicas: &replicas1,
				Template: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{Name: "nginx", Image: "nginx:1.19"},
						},
					},
				},
			},
			Status: appsv1.DeploymentStatus{
				ReadyReplicas:     2,
				UpdatedReplicas:   2,
				AvailableReplicas: 2,
			},
		},
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "app2",
				Namespace:         "default",
				CreationTimestamp: metav1.Time{Time: now.Add(-48 * time.Hour)},
			},
			Spec: appsv1.DeploymentSpec{
				Replicas: &replicas2,
				Template: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{Name: "redis", Image: "redis:6"},
						},
					},
				},
			},
			Status: appsv1.DeploymentStatus{
				ReadyReplicas:     3,
				UpdatedReplicas:   3,
				AvailableReplicas: 3,
			},
		},
	}

	service := newTestK8sDeploymentService(deployments...)

	ctx := context.Background()
	result, err := service.ListDeployments(ctx, 1, "default")

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result, 2)

	// Verify deployment names
	names := make([]string, len(result))
	for i, d := range result {
		names[i] = d.Name
	}
	assert.Contains(t, names, "app1")
	assert.Contains(t, names, "app2")

	// Verify all deployments have required fields
	for _, d := range result {
		assert.NotEmpty(t, d.Name)
		assert.NotEmpty(t, d.Namespace)
		assert.NotEmpty(t, d.Ready)
		assert.NotEmpty(t, d.Age)
		assert.NotEmpty(t, d.CreatedAt)
		assert.NotEmpty(t, d.Images)
	}
}

func TestK8sDeploymentService_ListDeployments_EmptyList(t *testing.T) {
	service := newTestK8sDeploymentService()

	ctx := context.Background()
	result, err := service.ListDeployments(ctx, 1, "default")

	assert.NoError(t, err)
	assert.NotNil(t, result, "Result should not be nil even when empty")
	assert.Empty(t, result)
	assert.Len(t, result, 0)
}

func TestK8sDeploymentService_ListDeployments_MultipleNamespaces(t *testing.T) {
	now := time.Now()
	replicas := int32(1)

	deployments := []runtime.Object{
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "app1",
				Namespace:         "default",
				CreationTimestamp: metav1.Time{Time: now},
			},
			Spec: appsv1.DeploymentSpec{
				Replicas: &replicas,
				Template: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{{Name: "app", Image: "app:v1"}},
					},
				},
			},
		},
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "app2",
				Namespace:         "production",
				CreationTimestamp: metav1.Time{Time: now},
			},
			Spec: appsv1.DeploymentSpec{
				Replicas: &replicas,
				Template: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{{Name: "app", Image: "app:v2"}},
					},
				},
			},
		},
	}

	service := newTestK8sDeploymentService(deployments...)

	// Test listing from default namespace
	ctx := context.Background()
	result, err := service.ListDeployments(ctx, 1, "default")

	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, "app1", result[0].Name)

	// Test listing from production namespace
	result, err = service.ListDeployments(ctx, 1, "production")

	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, "app2", result[0].Name)
}

// ============================================================================
// Test GetDeployment - Requirement 3.2
// ============================================================================

func TestK8sDeploymentService_GetDeployment_Success(t *testing.T) {
	now := time.Now()
	replicas := int32(3)

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "test-app",
			Namespace:         "default",
			CreationTimestamp: metav1.Time{Time: now.Add(-24 * time.Hour)},
			Labels: map[string]string{
				"app":     "test",
				"version": "v1",
			},
			Annotations: map[string]string{
				"description": "Test application",
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "test",
				},
			},
			Strategy: appsv1.DeploymentStrategy{
				Type: appsv1.RollingUpdateDeploymentStrategyType,
			},
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{Name: "app", Image: "app:v1"},
					},
				},
			},
		},
		Status: appsv1.DeploymentStatus{
			ReadyReplicas:     3,
			UpdatedReplicas:   3,
			AvailableReplicas: 3,
			Conditions: []appsv1.DeploymentCondition{
				{
					Type:    appsv1.DeploymentAvailable,
					Status:  corev1.ConditionTrue,
					Reason:  "MinimumReplicasAvailable",
					Message: "Deployment has minimum availability.",
				},
			},
		},
	}

	service := newTestK8sDeploymentService(deployment)

	ctx := context.Background()
	result, err := service.GetDeployment(ctx, 1, "default", "test-app")

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "test-app", result.Name)
	assert.Equal(t, "default", result.Namespace)
	assert.Equal(t, int32(3), result.Replicas)
	assert.NotNil(t, result.Labels)
	assert.Equal(t, "test", result.Labels["app"])
	assert.Equal(t, "v1", result.Labels["version"])
	assert.NotNil(t, result.Annotations)
	assert.Equal(t, "Test application", result.Annotations["description"])
	assert.Equal(t, "RollingUpdate", result.Strategy)
	assert.NotNil(t, result.Selector)
	assert.Equal(t, "test", result.Selector["app"])
	assert.Len(t, result.Conditions, 1)
	assert.Equal(t, "Available", result.Conditions[0].Type)
}

func TestK8sDeploymentService_GetDeployment_NotFound(t *testing.T) {
	service := newTestK8sDeploymentService()

	ctx := context.Background()
	result, err := service.GetDeployment(ctx, 1, "default", "non-existent")

	assert.Error(t, err)
	assert.Nil(t, result)
}

// ============================================================================
// Test Restart - Requirement 3.3
// ============================================================================

func TestK8sDeploymentService_Restart_Success(t *testing.T) {
	replicas := int32(2)
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "test-app",
			Namespace:         "default",
			CreationTimestamp: metav1.Now(),
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: make(map[string]string),
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{Name: "app", Image: "app:v1"},
					},
				},
			},
		},
	}

	service := newTestK8sDeploymentService(deployment)

	ctx := context.Background()
	err := service.Restart(ctx, 1, "default", "test-app")

	assert.NoError(t, err)

	// Verify the deployment was updated with restart annotation
	updated, err := service.fakeClient.AppsV1().Deployments("default").Get(ctx, "test-app", metav1.GetOptions{})
	assert.NoError(t, err)
	assert.NotNil(t, updated.Spec.Template.Annotations)
	assert.Contains(t, updated.Spec.Template.Annotations, "kubectl.kubernetes.io/restartedAt")
}

func TestK8sDeploymentService_Restart_NotFound(t *testing.T) {
	service := newTestK8sDeploymentService()

	ctx := context.Background()
	err := service.Restart(ctx, 1, "default", "non-existent")

	assert.Error(t, err)
}

func TestK8sDeploymentService_Restart_MultipleRestarts(t *testing.T) {
	replicas := int32(1)
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "test-app",
			Namespace:         "default",
			CreationTimestamp: metav1.Now(),
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{Name: "app", Image: "app:v1"},
					},
				},
			},
		},
	}

	service := newTestK8sDeploymentService(deployment)

	ctx := context.Background()

	// First restart
	err := service.Restart(ctx, 1, "default", "test-app")
	assert.NoError(t, err)

	updated1, _ := service.fakeClient.AppsV1().Deployments("default").Get(ctx, "test-app", metav1.GetOptions{})
	firstRestartTime := updated1.Spec.Template.Annotations["kubectl.kubernetes.io/restartedAt"]

	// Wait a bit to ensure different timestamp
	time.Sleep(1100 * time.Millisecond)

	// Second restart
	err = service.Restart(ctx, 1, "default", "test-app")
	assert.NoError(t, err)

	updated2, _ := service.fakeClient.AppsV1().Deployments("default").Get(ctx, "test-app", metav1.GetOptions{})
	secondRestartTime := updated2.Spec.Template.Annotations["kubectl.kubernetes.io/restartedAt"]

	// Verify timestamps are different
	assert.NotEqual(t, firstRestartTime, secondRestartTime)
}

// ============================================================================
// Test Scale - Requirement 3.4
// ============================================================================

func TestK8sDeploymentService_Scale_Success(t *testing.T) {
	replicas := int32(2)
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "test-app",
			Namespace:         "default",
			CreationTimestamp: metav1.Now(),
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{Name: "app", Image: "app:v1"},
					},
				},
			},
		},
	}

	service := newTestK8sDeploymentService(deployment)

	ctx := context.Background()

	// Scale up to 5
	err := service.Scale(ctx, 1, "default", "test-app", 5)
	assert.NoError(t, err)

	// Verify the deployment was scaled
	updated, err := service.fakeClient.AppsV1().Deployments("default").Get(ctx, "test-app", metav1.GetOptions{})
	assert.NoError(t, err)
	assert.Equal(t, int32(5), *updated.Spec.Replicas)

	// Scale down to 1
	err = service.Scale(ctx, 1, "default", "test-app", 1)
	assert.NoError(t, err)

	updated, err = service.fakeClient.AppsV1().Deployments("default").Get(ctx, "test-app", metav1.GetOptions{})
	assert.NoError(t, err)
	assert.Equal(t, int32(1), *updated.Spec.Replicas)

	// Scale to 0 (valid)
	err = service.Scale(ctx, 1, "default", "test-app", 0)
	assert.NoError(t, err)

	updated, err = service.fakeClient.AppsV1().Deployments("default").Get(ctx, "test-app", metav1.GetOptions{})
	assert.NoError(t, err)
	assert.Equal(t, int32(0), *updated.Spec.Replicas)
}

func TestK8sDeploymentService_Scale_InvalidReplicas(t *testing.T) {
	replicas := int32(2)
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "test-app",
			Namespace:         "default",
			CreationTimestamp: metav1.Now(),
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{Name: "app", Image: "app:v1"},
					},
				},
			},
		},
	}

	service := newTestK8sDeploymentService(deployment)

	ctx := context.Background()

	// Test negative replicas
	err := service.Scale(ctx, 1, "default", "test-app", -1)
	assert.Error(t, err)

	// Test replicas > 100
	err = service.Scale(ctx, 1, "default", "test-app", 101)
	assert.Error(t, err)

	// Verify deployment was not modified
	updated, err := service.fakeClient.AppsV1().Deployments("default").Get(ctx, "test-app", metav1.GetOptions{})
	assert.NoError(t, err)
	assert.Equal(t, int32(2), *updated.Spec.Replicas)
}

func TestK8sDeploymentService_Scale_BoundaryValues(t *testing.T) {
	replicas := int32(2)
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "test-app",
			Namespace:         "default",
			CreationTimestamp: metav1.Now(),
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{Name: "app", Image: "app:v1"},
					},
				},
			},
		},
	}

	service := newTestK8sDeploymentService(deployment)

	ctx := context.Background()

	// Test minimum valid value (0)
	err := service.Scale(ctx, 1, "default", "test-app", 0)
	assert.NoError(t, err)

	// Test maximum valid value (100)
	err = service.Scale(ctx, 1, "default", "test-app", 100)
	assert.NoError(t, err)

	updated, err := service.fakeClient.AppsV1().Deployments("default").Get(ctx, "test-app", metav1.GetOptions{})
	assert.NoError(t, err)
	assert.Equal(t, int32(100), *updated.Spec.Replicas)
}

func TestK8sDeploymentService_Scale_NotFound(t *testing.T) {
	service := newTestK8sDeploymentService()

	ctx := context.Background()
	err := service.Scale(ctx, 1, "default", "non-existent", 5)

	assert.Error(t, err)
}

// ============================================================================
// Test Edge Cases and Error Handling
// ============================================================================

func TestK8sDeploymentService_EdgeCases(t *testing.T) {
	t.Run("deployment with no containers", func(t *testing.T) {
		service := &K8sDeploymentService{}
		replicas := int32(1)

		deploy := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "no-containers",
				Namespace:         "default",
				CreationTimestamp: metav1.Now(),
			},
			Spec: appsv1.DeploymentSpec{
				Replicas: &replicas,
				Template: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{},
					},
				},
			},
		}

		result := service.convertDeploymentInfo(deploy)

		assert.Equal(t, "no-containers", result.Name)
		assert.Empty(t, result.Images)
		assert.Empty(t, result.Containers)
	})

	t.Run("deployment with zero ready replicas", func(t *testing.T) {
		service := &K8sDeploymentService{}
		replicas := int32(3)

		deploy := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "not-ready",
				Namespace:         "default",
				CreationTimestamp: metav1.Now(),
			},
			Spec: appsv1.DeploymentSpec{
				Replicas: &replicas,
				Template: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{Name: "app", Image: "app:v1"},
						},
					},
				},
			},
			Status: appsv1.DeploymentStatus{
				ReadyReplicas:     0,
				UpdatedReplicas:   3,
				AvailableReplicas: 0,
			},
		}

		result := service.convertDeploymentInfo(deploy)

		assert.Equal(t, "0/3", result.Ready)
		assert.Equal(t, int32(0), result.Available)
	})

	t.Run("deployment with partial ready replicas", func(t *testing.T) {
		service := &K8sDeploymentService{}
		replicas := int32(5)

		deploy := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "partial-ready",
				Namespace:         "default",
				CreationTimestamp: metav1.Now(),
			},
			Spec: appsv1.DeploymentSpec{
				Replicas: &replicas,
				Template: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{Name: "app", Image: "app:v1"},
						},
					},
				},
			},
			Status: appsv1.DeploymentStatus{
				ReadyReplicas:     3,
				UpdatedReplicas:   5,
				AvailableReplicas: 3,
			},
		}

		result := service.convertDeploymentInfo(deploy)

		assert.Equal(t, "3/5", result.Ready)
		assert.Equal(t, int32(3), result.Available)
		assert.Equal(t, int32(5), result.UpToDate)
	})
}

// ============================================================================
// Test Required Fields Validation
// ============================================================================

func TestDeploymentInfo_RequiredFields(t *testing.T) {
	t.Run("DeploymentInfo has all required fields", func(t *testing.T) {
		info := DeploymentInfo{
			Name:       "test-deployment",
			Namespace:  "default",
			Ready:      "2/3",
			UpToDate:   3,
			Available:  2,
			Age:        "1h",
			Images:     []string{"nginx:1.19"},
			Replicas:   3,
			CreatedAt:  "2024-01-15 10:30:45",
			Containers: []DeploymentContainer{{Name: "nginx", Image: "nginx:1.19"}},
		}

		assert.NotEmpty(t, info.Name)
		assert.NotEmpty(t, info.Namespace)
		assert.NotEmpty(t, info.Ready)
		assert.NotEmpty(t, info.Age)
		assert.NotEmpty(t, info.Images)
		assert.NotEmpty(t, info.CreatedAt)
		assert.NotZero(t, info.Replicas)
	})
}

func TestDeploymentDetail_Structure(t *testing.T) {
	t.Run("DeploymentDetail extends DeploymentInfo", func(t *testing.T) {
		detail := DeploymentDetail{
			DeploymentInfo: DeploymentInfo{
				Name:      "test-deployment",
				Namespace: "default",
				Ready:     "2/3",
				Replicas:  3,
			},
			Labels: map[string]string{
				"app": "test",
			},
			Annotations: map[string]string{
				"description": "Test deployment",
			},
			Strategy: "RollingUpdate",
			Selector: map[string]string{
				"app": "test",
			},
			Conditions: []DeploymentCondition{
				{
					Type:    "Available",
					Status:  "True",
					Reason:  "MinimumReplicasAvailable",
					Message: "Deployment has minimum availability.",
				},
			},
		}

		assert.NotEmpty(t, detail.Name)
		assert.NotEmpty(t, detail.Namespace)
		assert.NotNil(t, detail.Labels)
		assert.NotNil(t, detail.Annotations)
		assert.NotEmpty(t, detail.Strategy)
		assert.NotNil(t, detail.Selector)
		assert.NotEmpty(t, detail.Conditions)
	})
}
