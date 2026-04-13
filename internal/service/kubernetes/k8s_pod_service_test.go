package kubernetes

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TestConvertPodInfo 测试 Pod 信息转换
func TestConvertPodInfo(t *testing.T) {
	svc := &K8sPodService{}

	// 创建测试 Pod
	now := metav1.Now()
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "test-pod",
			Namespace:         "default",
			CreationTimestamp: now,
			Labels: map[string]string{
				"app": "test",
			},
		},
		Spec: corev1.PodSpec{
			NodeName: "node-1",
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
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
			PodIP: "10.0.0.1",
			ContainerStatuses: []corev1.ContainerStatus{
				{
					Name:         "nginx",
					Ready:        true,
					RestartCount: 0,
					State: corev1.ContainerState{
						Running: &corev1.ContainerStateRunning{
							StartedAt: now,
						},
					},
				},
				{
					Name:         "sidecar",
					Ready:        true,
					RestartCount: 2,
					State: corev1.ContainerState{
						Running: &corev1.ContainerStateRunning{
							StartedAt: now,
						},
					},
				},
			},
		},
	}

	// 执行转换
	info := svc.convertPodInfo(pod)

	// 验证基本信息
	assert.Equal(t, "test-pod", info.Name)
	assert.Equal(t, "default", info.Namespace)
	assert.Equal(t, "Running", info.Status)
	assert.Equal(t, "2/2", info.Ready)
	assert.Equal(t, int32(2), info.Restarts) // 总重启次数
	assert.Equal(t, "10.0.0.1", info.IP)
	assert.Equal(t, "node-1", info.Node)
	assert.NotEmpty(t, info.Age)
	assert.NotEmpty(t, info.CreatedAt)

	// 验证容器信息
	assert.Equal(t, 2, len(info.Containers))
	assert.Equal(t, "nginx", info.Containers[0].Name)
	assert.Equal(t, "nginx:1.19", info.Containers[0].Image)
	assert.True(t, info.Containers[0].Ready)
	assert.Equal(t, "Running", info.Containers[0].State)
	assert.Equal(t, int32(0), info.Containers[0].RestartCount)

	assert.Equal(t, "sidecar", info.Containers[1].Name)
	assert.Equal(t, "busybox:latest", info.Containers[1].Image)
	assert.True(t, info.Containers[1].Ready)
	assert.Equal(t, "Running", info.Containers[1].State)
	assert.Equal(t, int32(2), info.Containers[1].RestartCount)

	// 验证标签
	assert.Equal(t, "test", info.Labels["app"])
}

// TestConvertPodInfo_PendingPod 测试 Pending 状态的 Pod
func TestConvertPodInfo_PendingPod(t *testing.T) {
	svc := &K8sPodService{}

	now := metav1.Now()
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "pending-pod",
			Namespace:         "default",
			CreationTimestamp: now,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "nginx",
					Image: "nginx:1.19",
				},
			},
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodPending,
			ContainerStatuses: []corev1.ContainerStatus{
				{
					Name:  "nginx",
					Ready: false,
					State: corev1.ContainerState{
						Waiting: &corev1.ContainerStateWaiting{
							Reason: "ContainerCreating",
						},
					},
				},
			},
		},
	}

	info := svc.convertPodInfo(pod)

	assert.Equal(t, "Pending", info.Status)
	assert.Equal(t, "0/1", info.Ready)
	assert.False(t, info.Containers[0].Ready)
	assert.Equal(t, "ContainerCreating", info.Containers[0].State)
}

// TestConvertPodInfo_FailedPod 测试 Failed 状态的 Pod
func TestConvertPodInfo_FailedPod(t *testing.T) {
	svc := &K8sPodService{}

	now := metav1.Now()
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "failed-pod",
			Namespace:         "default",
			CreationTimestamp: now,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "nginx",
					Image: "nginx:1.19",
				},
			},
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodFailed,
			ContainerStatuses: []corev1.ContainerStatus{
				{
					Name:         "nginx",
					Ready:        false,
					RestartCount: 5,
					State: corev1.ContainerState{
						Terminated: &corev1.ContainerStateTerminated{
							Reason:   "Error",
							ExitCode: 1,
						},
					},
				},
			},
		},
	}

	info := svc.convertPodInfo(pod)

	assert.Equal(t, "Failed", info.Status)
	assert.Equal(t, "0/1", info.Ready)
	assert.Equal(t, int32(5), info.Restarts)
	assert.False(t, info.Containers[0].Ready)
	assert.Equal(t, "Error", info.Containers[0].State)
	assert.Equal(t, int32(5), info.Containers[0].RestartCount)
}

// TestConvertPodInfo_EmptyContainerStatuses 测试没有容器状态的 Pod
func TestConvertPodInfo_EmptyContainerStatuses(t *testing.T) {
	svc := &K8sPodService{}

	now := metav1.Now()
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "no-status-pod",
			Namespace:         "default",
			CreationTimestamp: now,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "nginx",
					Image: "nginx:1.19",
				},
			},
		},
		Status: corev1.PodStatus{
			Phase:             corev1.PodPending,
			ContainerStatuses: []corev1.ContainerStatus{}, // 空状态
		},
	}

	info := svc.convertPodInfo(pod)

	assert.Equal(t, "Pending", info.Status)
	assert.Equal(t, "0/1", info.Ready)
	assert.Equal(t, 1, len(info.Containers))
	assert.Equal(t, "nginx", info.Containers[0].Name)
	assert.False(t, info.Containers[0].Ready)
	assert.Equal(t, "Unknown", info.Containers[0].State)
}

// TestConvertPodInfo_MultipleContainers 测试多容器 Pod
func TestConvertPodInfo_MultipleContainers(t *testing.T) {
	svc := &K8sPodService{}

	now := metav1.Now()
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "multi-container-pod",
			Namespace:         "default",
			CreationTimestamp: now,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{Name: "app", Image: "app:1.0"},
				{Name: "sidecar", Image: "sidecar:1.0"},
				{Name: "init", Image: "init:1.0"},
			},
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
			ContainerStatuses: []corev1.ContainerStatus{
				{
					Name:         "app",
					Ready:        true,
					RestartCount: 0,
					State: corev1.ContainerState{
						Running: &corev1.ContainerStateRunning{},
					},
				},
				{
					Name:         "sidecar",
					Ready:        true,
					RestartCount: 1,
					State: corev1.ContainerState{
						Running: &corev1.ContainerStateRunning{},
					},
				},
				{
					Name:         "init",
					Ready:        false,
					RestartCount: 3,
					State: corev1.ContainerState{
						Waiting: &corev1.ContainerStateWaiting{
							Reason: "CrashLoopBackOff",
						},
					},
				},
			},
		},
	}

	info := svc.convertPodInfo(pod)

	assert.Equal(t, "2/3", info.Ready)       // 2 个容器就绪，共 3 个
	assert.Equal(t, int32(4), info.Restarts) // 总重启次数 0+1+3=4
	assert.Equal(t, 3, len(info.Containers))

	// 验证每个容器
	assert.Equal(t, "app", info.Containers[0].Name)
	assert.True(t, info.Containers[0].Ready)
	assert.Equal(t, int32(0), info.Containers[0].RestartCount)

	assert.Equal(t, "sidecar", info.Containers[1].Name)
	assert.True(t, info.Containers[1].Ready)
	assert.Equal(t, int32(1), info.Containers[1].RestartCount)

	assert.Equal(t, "init", info.Containers[2].Name)
	assert.False(t, info.Containers[2].Ready)
	assert.Equal(t, int32(3), info.Containers[2].RestartCount)
	assert.Equal(t, "CrashLoopBackOff", info.Containers[2].State)
}

// TestConvertPodInfo_NoLabels 测试没有标签的 Pod
func TestConvertPodInfo_NoLabels(t *testing.T) {
	svc := &K8sPodService{}

	now := metav1.Now()
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "no-labels-pod",
			Namespace:         "default",
			CreationTimestamp: now,
			Labels:            nil, // 没有标签
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{Name: "nginx", Image: "nginx:1.19"},
			},
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
		},
	}

	info := svc.convertPodInfo(pod)

	// Labels 可以是 nil 或空 map
	if info.Labels != nil {
		assert.Equal(t, 0, len(info.Labels))
	}
}

// TestConvertPodInfo_NoNodeName 测试没有节点名称的 Pod
func TestConvertPodInfo_NoNodeName(t *testing.T) {
	svc := &K8sPodService{}

	now := metav1.Now()
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "no-node-pod",
			Namespace:         "default",
			CreationTimestamp: now,
		},
		Spec: corev1.PodSpec{
			NodeName: "", // 没有节点名称
			Containers: []corev1.Container{
				{Name: "nginx", Image: "nginx:1.19"},
			},
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodPending,
		},
	}

	info := svc.convertPodInfo(pod)

	assert.Equal(t, "", info.Node)
	assert.Equal(t, "Pending", info.Status)
}

// TestConvertPodInfo_NoIP 测试没有 IP 的 Pod
func TestConvertPodInfo_NoIP(t *testing.T) {
	svc := &K8sPodService{}

	now := metav1.Now()
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "no-ip-pod",
			Namespace:         "default",
			CreationTimestamp: now,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{Name: "nginx", Image: "nginx:1.19"},
			},
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodPending,
			PodIP: "", // 没有 IP
		},
	}

	info := svc.convertPodInfo(pod)

	assert.Equal(t, "", info.IP)
	assert.Equal(t, "Pending", info.Status)
}
