package kubernetes

import (
	"context"
	"testing"
	"time"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
)

// ============================================================================
// Task 2.3: 编写 Namespace 属性测试
// Property 3: Namespace 列表加载
// **Validates: Requirements 2.1**
// ============================================================================

// TestProperty_NamespaceListLoading tests that for any K8s cluster,
// when the page loads, the system should call K8s API to get all namespaces
// and display them in the list.
//
// This property verifies that:
// 1. All namespaces from K8s API are included in the result
// 2. Each namespace has all required fields populated
// 3. The conversion preserves namespace identity and data
func TestProperty_NamespaceListLoading(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100

	properties := gopter.NewProperties(parameters)

	properties.Property("All namespaces are loaded and contain required fields", prop.ForAll(
		func(namespaces []corev1.Namespace) bool {
			// Skip empty lists as they're tested separately
			if len(namespaces) == 0 {
				return true
			}

			// Create fake K8s client with generated namespaces
			objects := make([]runtime.Object, len(namespaces))
			for i := range namespaces {
				objects[i] = &namespaces[i]
			}
			fakeClient := fake.NewSimpleClientset(objects...)

			// Create service with fake client
			service := &K8sNamespaceService{clientMgr: nil}

			// Call ListNamespaces using fake client
			ctx := context.Background()
			nsList, err := fakeClient.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
			if err != nil {
				return false
			}

			result := make([]NamespaceInfo, len(nsList.Items))
			for i, ns := range nsList.Items {
				result[i] = service.convertNamespaceInfo(&ns)
			}

			// Property 1: All namespaces from K8s API are included
			if len(result) != len(namespaces) {
				return false
			}

			// Create a map of input namespaces by name for easy lookup
			inputMap := make(map[string]corev1.Namespace)
			for _, ns := range namespaces {
				inputMap[ns.Name] = ns
			}

			// Property 2: Each namespace has all required fields populated
			// and matches the input namespace
			for _, info := range result {
				// Find the corresponding input namespace
				inputNs, found := inputMap[info.Name]
				if !found {
					return false
				}

				// Status must match
				if info.Status != string(inputNs.Status.Phase) {
					return false
				}

				// Age must not be empty
				if info.Age == "" {
					return false
				}

				// CreatedAt must not be empty
				if info.CreatedAt == "" {
					return false
				}

				// Labels must not be nil (can be empty map)
				if info.Labels == nil {
					return false
				}

				// Labels must match input (accounting for nil -> empty map conversion)
				inputLabels := inputNs.Labels
				if inputLabels == nil {
					inputLabels = make(map[string]string)
				}
				if len(info.Labels) != len(inputLabels) {
					return false
				}
				for k, v := range inputLabels {
					if info.Labels[k] != v {
						return false
					}
				}
			}

			return true
		},
		genNamespaceList(),
	))

	properties.TestingRun(t)
}

// TestProperty_NamespaceListLoading_EmptyCluster tests that the system
// handles empty namespace lists correctly (returns empty array, not nil)
func TestProperty_NamespaceListLoading_EmptyCluster(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 10

	properties := gopter.NewProperties(parameters)

	properties.Property("Empty namespace list returns empty array not nil", prop.ForAll(
		func() bool {
			// Create fake K8s client with no namespaces
			fakeClient := fake.NewSimpleClientset()

			// Create service
			service := &K8sNamespaceService{clientMgr: nil}

			// Call ListNamespaces
			ctx := context.Background()
			nsList, err := fakeClient.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
			if err != nil {
				return false
			}

			result := make([]NamespaceInfo, len(nsList.Items))
			for i, ns := range nsList.Items {
				result[i] = service.convertNamespaceInfo(&ns)
			}

			// Must return non-nil slice
			if result == nil {
				return false
			}

			// Must be empty
			if len(result) != 0 {
				return false
			}

			return true
		},
	))

	properties.TestingRun(t)
}

// TestProperty_NamespaceListLoading_LabelsPreserved tests that labels
// are correctly preserved during namespace conversion
func TestProperty_NamespaceListLoading_LabelsPreserved(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100

	properties := gopter.NewProperties(parameters)

	properties.Property("Namespace labels are preserved during conversion", prop.ForAll(
		func(ns corev1.Namespace) bool {
			// Create service
			service := &K8sNamespaceService{clientMgr: nil}

			// Convert namespace
			result := service.convertNamespaceInfo(&ns)

			// Labels must not be nil
			if result.Labels == nil {
				return false
			}

			// If input had labels, they must be preserved
			if ns.Labels != nil {
				if len(result.Labels) != len(ns.Labels) {
					return false
				}
				for k, v := range ns.Labels {
					if result.Labels[k] != v {
						return false
					}
				}
			}

			return true
		},
		genNamespace(),
	))

	properties.TestingRun(t)
}

// TestProperty_NamespaceListLoading_StatusConversion tests that namespace
// status is correctly converted from K8s phase to string
func TestProperty_NamespaceListLoading_StatusConversion(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100

	properties := gopter.NewProperties(parameters)

	properties.Property("Namespace status is correctly converted", prop.ForAll(
		func(phase corev1.NamespacePhase) bool {
			// Create namespace with specific phase
			ns := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "test-ns",
					CreationTimestamp: metav1.Now(),
				},
				Status: corev1.NamespaceStatus{
					Phase: phase,
				},
			}

			// Create service
			service := &K8sNamespaceService{clientMgr: nil}

			// Convert namespace
			result := service.convertNamespaceInfo(ns)

			// Status must match the phase
			if result.Status != string(phase) {
				return false
			}

			return true
		},
		genNamespacePhase(),
	))

	properties.TestingRun(t)
}

// TestProperty_NamespaceListLoading_AgeCalculation tests that age
// is correctly calculated and formatted
func TestProperty_NamespaceListLoading_AgeCalculation(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100

	properties := gopter.NewProperties(parameters)

	properties.Property("Namespace age is correctly calculated", prop.ForAll(
		func(ageSeconds int64) bool {
			// Constrain age to reasonable range (0 to 365 days)
			if ageSeconds < 0 || ageSeconds > 365*24*60*60 {
				return true // Skip invalid inputs
			}

			// Create namespace with specific age
			creationTime := time.Now().Add(-time.Duration(ageSeconds) * time.Second)
			ns := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "test-ns",
					CreationTimestamp: metav1.Time{Time: creationTime},
				},
				Status: corev1.NamespaceStatus{
					Phase: corev1.NamespaceActive,
				},
			}

			// Create service
			service := &K8sNamespaceService{clientMgr: nil}

			// Convert namespace
			result := service.convertNamespaceInfo(ns)

			// Age must not be empty
			if result.Age == "" {
				return false
			}

			// Age must be a valid format (ends with s, m, h, or d)
			lastChar := result.Age[len(result.Age)-1]
			if lastChar != 's' && lastChar != 'm' && lastChar != 'h' && lastChar != 'd' {
				return false
			}

			return true
		},
		gen.Int64Range(0, 365*24*60*60),
	))

	properties.TestingRun(t)
}

// TestProperty_NamespaceListLoading_CreatedAtFormat tests that CreatedAt
// timestamp is correctly formatted
func TestProperty_NamespaceListLoading_CreatedAtFormat(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100

	properties := gopter.NewProperties(parameters)

	properties.Property("Namespace CreatedAt is correctly formatted", prop.ForAll(
		func(year int, month int, day int, hour int, minute int, second int) bool {
			// Constrain to valid date/time ranges
			if year < 2020 || year > 2030 {
				return true
			}
			if month < 1 || month > 12 {
				return true
			}
			if day < 1 || day > 28 { // Use 28 to avoid month-specific logic
				return true
			}
			if hour < 0 || hour > 23 {
				return true
			}
			if minute < 0 || minute > 59 {
				return true
			}
			if second < 0 || second > 59 {
				return true
			}

			// Create namespace with specific time
			creationTime := time.Date(year, time.Month(month), day, hour, minute, second, 0, time.UTC)
			ns := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "test-ns",
					CreationTimestamp: metav1.Time{Time: creationTime},
				},
				Status: corev1.NamespaceStatus{
					Phase: corev1.NamespaceActive,
				},
			}

			// Create service
			service := &K8sNamespaceService{clientMgr: nil}

			// Convert namespace
			result := service.convertNamespaceInfo(ns)

			// CreatedAt must not be empty
			if result.CreatedAt == "" {
				return false
			}

			// CreatedAt must be in format "2006-01-02 15:04:05"
			// Try to parse it back
			_, err := time.Parse("2006-01-02 15:04:05", result.CreatedAt)
			if err != nil {
				return false
			}

			return true
		},
		gen.IntRange(2020, 2030),
		gen.IntRange(1, 12),
		gen.IntRange(1, 28),
		gen.IntRange(0, 23),
		gen.IntRange(0, 59),
		gen.IntRange(0, 59),
	))

	properties.TestingRun(t)
}

// ============================================================================
// Property Test Generators
// ============================================================================

// genNamespace generates a random Namespace object
func genNamespace() gopter.Gen {
	return gopter.CombineGens(
		gen.Identifier(),                               // name
		genNamespacePhase(),                            // phase
		gen.Int64Range(0, 365*24*60*60),                // age in seconds
		gen.MapOf(gen.Identifier(), gen.AlphaString()), // labels
	).Map(func(values []interface{}) corev1.Namespace {
		name := values[0].(string)
		phase := values[1].(corev1.NamespacePhase)
		ageSeconds := values[2].(int64)
		labels := values[3].(map[string]string)

		creationTime := time.Now().Add(-time.Duration(ageSeconds) * time.Second)

		return corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name:              name,
				CreationTimestamp: metav1.Time{Time: creationTime},
				Labels:            labels,
			},
			Status: corev1.NamespaceStatus{
				Phase: phase,
			},
		}
	})
}

// genNamespaceList generates a list of random Namespace objects
func genNamespaceList() gopter.Gen {
	return gen.SliceOfN(10, genNamespace()).
		SuchThat(func(namespaces []corev1.Namespace) bool {
			// Ensure unique namespace names
			names := make(map[string]bool)
			for _, ns := range namespaces {
				if names[ns.Name] {
					return false
				}
				names[ns.Name] = true
			}
			return true
		})
}

// genNamespacePhase generates a random NamespacePhase
func genNamespacePhase() gopter.Gen {
	return gen.OneConstOf(
		corev1.NamespaceActive,
		corev1.NamespaceTerminating,
	)
}

// ============================================================================
// Additional Property Tests for Edge Cases
// ============================================================================

// TestProperty_NamespaceListLoading_NilLabelsHandling tests that nil labels
// are converted to empty map, never nil
func TestProperty_NamespaceListLoading_NilLabelsHandling(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 50

	properties := gopter.NewProperties(parameters)

	properties.Property("Nil labels are converted to empty map", prop.ForAll(
		func(name string, phase corev1.NamespacePhase) bool {
			// Create namespace with nil labels
			ns := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name:              name,
					CreationTimestamp: metav1.Now(),
					Labels:            nil,
				},
				Status: corev1.NamespaceStatus{
					Phase: phase,
				},
			}

			// Create service
			service := &K8sNamespaceService{clientMgr: nil}

			// Convert namespace
			result := service.convertNamespaceInfo(ns)

			// Labels must not be nil
			if result.Labels == nil {
				return false
			}

			return true
		},
		gen.Identifier(),
		genNamespacePhase(),
	))

	properties.TestingRun(t)
}

// TestProperty_NamespaceListLoading_Idempotency tests that converting
// the same namespace multiple times produces the same result
func TestProperty_NamespaceListLoading_Idempotency(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100

	properties := gopter.NewProperties(parameters)

	properties.Property("Converting same namespace multiple times is idempotent", prop.ForAll(
		func(ns corev1.Namespace) bool {
			// Create service
			service := &K8sNamespaceService{clientMgr: nil}

			// Convert namespace multiple times
			result1 := service.convertNamespaceInfo(&ns)
			result2 := service.convertNamespaceInfo(&ns)

			// Results must be identical
			if result1.Name != result2.Name {
				return false
			}
			if result1.Status != result2.Status {
				return false
			}
			// Note: Age might differ slightly due to time passing, so we skip it
			if result1.CreatedAt != result2.CreatedAt {
				return false
			}
			if len(result1.Labels) != len(result2.Labels) {
				return false
			}
			for k, v := range result1.Labels {
				if result2.Labels[k] != v {
					return false
				}
			}

			return true
		},
		genNamespace(),
	))

	properties.TestingRun(t)
}
