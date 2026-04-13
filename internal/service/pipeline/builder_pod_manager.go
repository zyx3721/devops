package pipeline

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/remotecommand"

	"gorm.io/gorm"

	"devops/internal/models"
	k8sservice "devops/internal/service/kubernetes"
	"devops/pkg/logger"
)

// BuilderPodConfig 构建 Pod 配置
type BuilderPodConfig struct {
	IdleTimeoutMinutes int    `json:"idle_timeout_minutes"`
	StorageType        string `json:"storage_type"`   // pvc, emptydir, hostpath
	PVCName            string `json:"pvc_name"`       // PVC 名称
	PVCSizeGi          int    `json:"pvc_size_gi"`    // PVC 大小（Gi）
	StorageClass       string `json:"storage_class"`  // StorageClass 名称
	AccessMode         string `json:"access_mode"`    // ReadWriteMany 或 ReadWriteOnce
	HostPath           string `json:"host_path"`      // HostPath 路径（仅 hostpath 模式）
	CPURequest         string `json:"cpu_request"`    // CPU 请求
	CPULimit           string `json:"cpu_limit"`      // CPU 限制
	MemoryRequest      string `json:"memory_request"` // 内存请求
	MemoryLimit        string `json:"memory_limit"`   // 内存限制
}

// BuilderPodManager 构建 Pod 管理器
type BuilderPodManager struct {
	db           *gorm.DB
	clientMgr    *k8sservice.K8sClientManager
	pods         sync.Map // key: "clusterID-namespace-image" -> *BuilderPod
	idleTimeout  time.Duration
	config       *BuilderPodConfig
	configMu     sync.RWMutex
	cleanupTimer *time.Ticker
	stopCh       chan struct{}
}

// BuilderPod 构建 Pod 信息
type BuilderPod struct {
	ClusterID  uint
	Namespace  string
	PodName    string
	Image      string
	LastUsedAt time.Time
	mu         sync.Mutex
}

// NewBuilderPodManager 创建构建 Pod 管理器
func NewBuilderPodManager(db *gorm.DB) *BuilderPodManager {
	m := &BuilderPodManager{
		db:          db,
		clientMgr:   k8sservice.NewK8sClientManager(db),
		idleTimeout: 30 * time.Minute, // 默认30分钟
		config: &BuilderPodConfig{
			IdleTimeoutMinutes: 30,
			StorageType:        "pvc",
			PVCName:            "devops-workspace-shared",
			PVCSizeGi:          10,
			AccessMode:         "ReadWriteMany",
			CPURequest:         "100m",
			CPULimit:           "2",
			MemoryRequest:      "256Mi",
			MemoryLimit:        "4Gi",
		},
		stopCh: make(chan struct{}),
	}

	// 启动时加载已有的 Builder Pod 到缓存
	go m.loadExistingPods()

	// 启动清理协程
	go m.cleanupLoop()

	return m
}

// loadExistingPods 加载已有的 Builder Pod 到缓存
func (m *BuilderPodManager) loadExistingPods() {
	ctx := context.Background()
	log := logger.L()

	// 检查 db 是否为 nil
	if m.db == nil {
		log.Warn("数据库连接未初始化，跳过加载已有 Builder Pod")
		return
	}

	// 获取所有已配置的集群
	var clusters []models.K8sCluster
	if err := m.db.Find(&clusters).Error; err != nil {
		log.WithError(err).Warn("获取集群列表失败")
		return
	}

	for _, cluster := range clusters {
		client, err := m.clientMgr.GetClient(ctx, cluster.ID)
		if err != nil {
			continue
		}

		// 查询带有 devops-builder 标签的 Pod
		pods, err := client.CoreV1().Pods("").List(ctx, metav1.ListOptions{
			LabelSelector: "app=devops-builder,managed-by=devops-platform",
		})
		if err != nil {
			continue
		}

		for _, pod := range pods.Items {
			if pod.Status.Phase != corev1.PodRunning {
				continue
			}

			image := ""
			if len(pod.Spec.Containers) > 0 {
				image = pod.Spec.Containers[0].Image
			}

			key := fmt.Sprintf("%d-%s-%s", cluster.ID, pod.Namespace, image)

			// 只有缓存中没有才添加
			if _, exists := m.pods.Load(key); !exists {
				builderPod := &BuilderPod{
					ClusterID:  cluster.ID,
					Namespace:  pod.Namespace,
					PodName:    pod.Name,
					Image:      image,
					LastUsedAt: pod.CreationTimestamp.Time,
				}
				m.pods.Store(key, builderPod)
				log.WithField("pod_name", pod.Name).WithField("image", image).Info("加载已有 Builder Pod 到缓存")
			}
		}
	}
}

// GetConfig 获取配置
func (m *BuilderPodManager) GetConfig() *BuilderPodConfig {
	m.configMu.RLock()
	defer m.configMu.RUnlock()
	cfg := *m.config
	cfg.IdleTimeoutMinutes = int(m.idleTimeout.Minutes())
	return &cfg
}

// SetConfig 设置配置
func (m *BuilderPodManager) SetConfig(cfg *BuilderPodConfig) {
	m.configMu.Lock()
	defer m.configMu.Unlock()
	m.config = cfg
	// 同步空闲超时时间
	if cfg.IdleTimeoutMinutes > 0 {
		m.idleTimeout = time.Duration(cfg.IdleTimeoutMinutes) * time.Minute
	}
}

// SetIdleTimeout 设置空闲超时时间
func (m *BuilderPodManager) SetIdleTimeout(timeout time.Duration) {
	m.idleTimeout = timeout
}

// GetOrCreatePod 获取或创建构建 Pod
func (m *BuilderPodManager) GetOrCreatePod(ctx context.Context, clusterID uint, namespace, image string) (*BuilderPod, error) {
	key := fmt.Sprintf("%d-%s-%s", clusterID, namespace, image)

	// 尝试获取已存在的 Pod（内存缓存）
	if val, ok := m.pods.Load(key); ok {
		pod := val.(*BuilderPod)
		pod.mu.Lock()
		pod.LastUsedAt = time.Now()
		pod.mu.Unlock()

		// 检查 Pod 是否还在运行
		if m.isPodRunning(ctx, clusterID, namespace, pod.PodName) {
			logger.L().WithField("pod_name", pod.PodName).Debug("复用已有 Builder Pod")
			return pod, nil
		}
		// Pod 不存在了，删除记录
		m.pods.Delete(key)
	}

	// 内存缓存没有，检查 K8s 集群中是否有相同镜像的 Pod
	existingPod := m.findExistingPodInK8s(ctx, clusterID, namespace, image)
	if existingPod != nil {
		m.pods.Store(key, existingPod)
		logger.L().WithField("pod_name", existingPod.PodName).Info("发现并复用 K8s 中已有的 Builder Pod")
		return existingPod, nil
	}

	// 创建新的 Pod
	pod, err := m.createBuilderPod(ctx, clusterID, namespace, image)
	if err != nil {
		return nil, err
	}

	m.pods.Store(key, pod)
	return pod, nil
}

// findExistingPodInK8s 在 K8s 集群中查找已有的 Builder Pod
func (m *BuilderPodManager) findExistingPodInK8s(ctx context.Context, clusterID uint, namespace, image string) *BuilderPod {
	client, err := m.clientMgr.GetClient(ctx, clusterID)
	if err != nil {
		return nil
	}

	// 查询带有 devops-builder 标签的 Pod
	pods, err := client.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: "app=devops-builder,managed-by=devops-platform",
	})
	if err != nil {
		return nil
	}

	for _, pod := range pods.Items {
		if pod.Status.Phase != corev1.PodRunning {
			continue
		}

		// 检查镜像是否匹配
		if len(pod.Spec.Containers) > 0 && pod.Spec.Containers[0].Image == image {
			return &BuilderPod{
				ClusterID:  clusterID,
				Namespace:  namespace,
				PodName:    pod.Name,
				Image:      image,
				LastUsedAt: time.Now(),
			}
		}
	}

	return nil
}

// createBuilderPod 创建构建 Pod
func (m *BuilderPodManager) createBuilderPod(ctx context.Context, clusterID uint, namespace, image string) (*BuilderPod, error) {
	log := logger.L().WithField("cluster_id", clusterID).WithField("namespace", namespace).WithField("image", image)
	log.Info("创建构建 Pod")

	client, err := m.clientMgr.GetClient(ctx, clusterID)
	if err != nil {
		return nil, fmt.Errorf("获取 K8s 客户端失败: %v", err)
	}

	// 获取配置
	m.configMu.RLock()
	cfg := m.config
	m.configMu.RUnlock()

	// 生成 Pod 名称
	podName := fmt.Sprintf("builder-%d", time.Now().UnixNano())

	// 构建 Volume 配置
	var volumes []corev1.Volume
	switch cfg.StorageType {
	case "pvc":
		pvcName := cfg.PVCName
		if pvcName == "" {
			pvcName = "devops-workspace-shared"
		}
		// 确保共享 PVC 存在
		if err := m.ensureSharedPVC(ctx, client, namespace, pvcName, cfg); err != nil {
			log.WithError(err).Warn("创建共享 PVC 失败，使用 EmptyDir")
			volumes = []corev1.Volume{{Name: "workspace", VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}}}
		} else {
			volumes = []corev1.Volume{{Name: "workspace", VolumeSource: corev1.VolumeSource{PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{ClaimName: pvcName}}}}
		}
	case "hostpath":
		hostPath := cfg.HostPath
		if hostPath == "" {
			hostPath = "/tmp/devops-workspace"
		}
		pathType := corev1.HostPathDirectoryOrCreate
		volumes = []corev1.Volume{{
			Name: "workspace",
			VolumeSource: corev1.VolumeSource{
				HostPath: &corev1.HostPathVolumeSource{
					Path: hostPath,
					Type: &pathType,
				},
			},
		}}
		log.WithField("host_path", hostPath).Info("使用 HostPath 存储")
	default: // emptydir
		volumes = []corev1.Volume{{Name: "workspace", VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}}}
	}

	// 解析资源配置
	cpuRequest := cfg.CPURequest
	if cpuRequest == "" {
		cpuRequest = "100m"
	}
	cpuLimit := cfg.CPULimit
	if cpuLimit == "" {
		cpuLimit = "2"
	}
	memRequest := cfg.MemoryRequest
	if memRequest == "" {
		memRequest = "256Mi"
	}
	memLimit := cfg.MemoryLimit
	if memLimit == "" {
		memLimit = "4Gi"
	}

	// 创建 Pod
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      podName,
			Namespace: namespace,
			Labels: map[string]string{
				"app":        "devops-builder",
				"managed-by": "devops-platform",
			},
		},
		Spec: corev1.PodSpec{
			RestartPolicy: corev1.RestartPolicyNever,
			Containers: []corev1.Container{
				{
					Name:       "builder",
					Image:      image,
					Command:    []string{"/bin/sh", "-c", "while true; do sleep 3600; done"},
					WorkingDir: "/workspace",
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse(cpuRequest),
							corev1.ResourceMemory: resource.MustParse(memRequest),
						},
						Limits: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse(cpuLimit),
							corev1.ResourceMemory: resource.MustParse(memLimit),
						},
					},
					VolumeMounts: []corev1.VolumeMount{{Name: "workspace", MountPath: "/workspace"}},
				},
			},
			Volumes: volumes,
		},
	}

	// 创建 Pod
	_, err = client.CoreV1().Pods(namespace).Create(ctx, pod, metav1.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf("创建 Pod 失败: %v", err)
	}

	// 等待 Pod 运行
	if err := m.waitPodRunning(ctx, client, namespace, podName); err != nil {
		// 清理失败的 Pod
		client.CoreV1().Pods(namespace).Delete(ctx, podName, metav1.DeleteOptions{})
		return nil, err
	}

	log.WithField("pod_name", podName).Info("构建 Pod 创建成功")

	return &BuilderPod{
		ClusterID:  clusterID,
		Namespace:  namespace,
		PodName:    podName,
		Image:      image,
		LastUsedAt: time.Now(),
	}, nil
}

// waitPodRunning 等待 Pod 运行
func (m *BuilderPodManager) waitPodRunning(ctx context.Context, client kubernetes.Interface, namespace, podName string) error {
	timeout := time.After(5 * time.Minute)
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timeout:
			return fmt.Errorf("等待 Pod 运行超时")
		case <-ticker.C:
			pod, err := client.CoreV1().Pods(namespace).Get(ctx, podName, metav1.GetOptions{})
			if err != nil {
				continue
			}
			if pod.Status.Phase == corev1.PodRunning {
				return nil
			}
			if pod.Status.Phase == corev1.PodFailed {
				return fmt.Errorf("Pod 启动失败")
			}
		}
	}
}

// ensureSharedPVC 确保共享 PVC 存在
func (m *BuilderPodManager) ensureSharedPVC(ctx context.Context, client kubernetes.Interface, namespace, pvcName string, cfg *BuilderPodConfig) error {
	// 检查 PVC 是否存在
	_, err := client.CoreV1().PersistentVolumeClaims(namespace).Get(ctx, pvcName, metav1.GetOptions{})
	if err == nil {
		return nil // PVC 已存在
	}

	if !errors.IsNotFound(err) {
		return err
	}

	// 解析配置
	pvcSize := cfg.PVCSizeGi
	if pvcSize <= 0 {
		pvcSize = 10
	}
	accessMode := corev1.ReadWriteMany
	if cfg.AccessMode == "ReadWriteOnce" {
		accessMode = corev1.ReadWriteOnce
	}

	// 创建 PVC
	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pvcName,
			Namespace: namespace,
			Labels: map[string]string{
				"app":        "devops-builder",
				"managed-by": "devops-platform",
			},
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{accessMode},
			Resources: corev1.VolumeResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: resource.MustParse(fmt.Sprintf("%dGi", pvcSize)),
				},
			},
		},
	}

	if cfg.StorageClass != "" {
		pvc.Spec.StorageClassName = &cfg.StorageClass
	}

	_, err = client.CoreV1().PersistentVolumeClaims(namespace).Create(ctx, pvc, metav1.CreateOptions{})
	if err != nil && !errors.IsAlreadyExists(err) {
		return fmt.Errorf("创建 PVC 失败: %v", err)
	}

	return nil
}

// isPodRunning 检查 Pod 是否运行中
func (m *BuilderPodManager) isPodRunning(ctx context.Context, clusterID uint, namespace, podName string) bool {
	client, err := m.clientMgr.GetClient(ctx, clusterID)
	if err != nil {
		return false
	}

	pod, err := client.CoreV1().Pods(namespace).Get(ctx, podName, metav1.GetOptions{})
	if err != nil {
		return false
	}

	return pod.Status.Phase == corev1.PodRunning
}

// ExecInPod 在 Pod 中执行命令
func (m *BuilderPodManager) ExecInPod(ctx context.Context, builderPod *BuilderPod, commands []string, workDir string, env map[string]string) (string, int, error) {
	builderPod.mu.Lock()
	builderPod.LastUsedAt = time.Now()
	builderPod.mu.Unlock()

	client, err := m.clientMgr.GetClient(ctx, builderPod.ClusterID)
	if err != nil {
		return "", 1, fmt.Errorf("获取 K8s 客户端失败: %v", err)
	}

	// 获取 REST config
	var cluster models.K8sCluster
	if err := m.db.First(&cluster, builderPod.ClusterID).Error; err != nil {
		return "", 1, fmt.Errorf("获取集群配置失败: %v", err)
	}

	restConfig, err := k8sservice.BuildRestConfig(&cluster)
	if err != nil {
		return "", 1, fmt.Errorf("构建 REST 配置失败: %v", err)
	}

	// 构建完整的 shell 脚本
	var script strings.Builder
	script.WriteString("set -e\n")

	// 设置工作目录（默认 /workspace）
	if workDir == "" || workDir == "/" {
		workDir = "/workspace"
	}
	script.WriteString(fmt.Sprintf("mkdir -p %s && cd %s\n", workDir, workDir))

	// 设置环境变量
	for k, v := range env {
		script.WriteString(fmt.Sprintf("export %s='%s'\n", k, v))
	}

	// 添加命令
	for _, cmd := range commands {
		script.WriteString(fmt.Sprintf("echo '+ %s'\n", cmd))
		script.WriteString(cmd + "\n")
	}

	// 创建 exec 请求
	req := client.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(builderPod.PodName).
		Namespace(builderPod.Namespace).
		SubResource("exec").
		VersionedParams(&corev1.PodExecOptions{
			Container: "builder",
			Command:   []string{"/bin/sh", "-c", script.String()},
			Stdin:     false,
			Stdout:    true,
			Stderr:    true,
			TTY:       false,
		}, scheme.ParameterCodec)

	exec, err := remotecommand.NewSPDYExecutor(restConfig, "POST", req.URL())
	if err != nil {
		return "", 1, fmt.Errorf("创建执行器失败: %v", err)
	}

	var stdout, stderr bytes.Buffer
	err = exec.StreamWithContext(ctx, remotecommand.StreamOptions{
		Stdout: &stdout,
		Stderr: &stderr,
	})

	logs := stdout.String()
	if stderr.Len() > 0 {
		logs += "\n" + stderr.String()
	}

	if err != nil {
		return logs, 1, err
	}

	return logs, 0, nil
}

// cleanupLoop 清理空闲 Pod
func (m *BuilderPodManager) cleanupLoop() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-m.stopCh:
			return
		case <-ticker.C:
			m.cleanupIdlePods()
		}
	}
}

// cleanupIdlePods 清理空闲的 Pod
func (m *BuilderPodManager) cleanupIdlePods() {
	ctx := context.Background()
	now := time.Now()

	// 获取当前配置的超时时间
	m.configMu.RLock()
	timeout := m.idleTimeout
	m.configMu.RUnlock()

	log := logger.L().WithField("idle_timeout", timeout)

	// 1. 清理内存缓存中的空闲 Pod
	m.pods.Range(func(key, value interface{}) bool {
		pod := value.(*BuilderPod)
		pod.mu.Lock()
		idle := now.Sub(pod.LastUsedAt)
		pod.mu.Unlock()

		if idle > timeout {
			log.WithField("pod_name", pod.PodName).WithField("idle", idle).Info("清理空闲构建 Pod")
			m.deletePod(ctx, pod)
			m.pods.Delete(key)
		}
		return true
	})

	// 检查 db 是否为 nil
	if m.db == nil {
		return
	}

	// 2. 清理 K8s 中不在缓存里的孤立 Pod（基于创建时间）
	var clusters []models.K8sCluster
	if err := m.db.Find(&clusters).Error; err != nil {
		return
	}

	for _, cluster := range clusters {
		client, err := m.clientMgr.GetClient(ctx, cluster.ID)
		if err != nil {
			continue
		}

		pods, err := client.CoreV1().Pods("").List(ctx, metav1.ListOptions{
			LabelSelector: "app=devops-builder,managed-by=devops-platform",
		})
		if err != nil {
			continue
		}

		for _, pod := range pods.Items {
			if pod.Status.Phase != corev1.PodRunning {
				continue
			}

			image := ""
			if len(pod.Spec.Containers) > 0 {
				image = pod.Spec.Containers[0].Image
			}

			key := fmt.Sprintf("%d-%s-%s", cluster.ID, pod.Namespace, image)

			// 如果不在缓存中，检查创建时间是否超过超时时间
			if _, exists := m.pods.Load(key); !exists {
				idle := now.Sub(pod.CreationTimestamp.Time)
				if idle > timeout {
					log.WithField("pod_name", pod.Name).WithField("idle", idle).Info("清理孤立的空闲构建 Pod")
					client.CoreV1().Pods(pod.Namespace).Delete(ctx, pod.Name, metav1.DeleteOptions{})
				}
			}
		}
	}
}

// deletePod 删除 Pod
func (m *BuilderPodManager) deletePod(ctx context.Context, pod *BuilderPod) {
	client, err := m.clientMgr.GetClient(ctx, pod.ClusterID)
	if err != nil {
		return
	}

	err = client.CoreV1().Pods(pod.Namespace).Delete(ctx, pod.PodName, metav1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		logger.L().WithError(err).WithField("pod_name", pod.PodName).Warn("删除 Pod 失败")
	}
}

// Stop 停止管理器
func (m *BuilderPodManager) Stop() {
	close(m.stopCh)

	// 清理所有 Pod
	ctx := context.Background()
	m.pods.Range(func(key, value interface{}) bool {
		pod := value.(*BuilderPod)
		m.deletePod(ctx, pod)
		return true
	})
}

// GetActivePods 获取活跃的构建 Pod 列表（从 K8s 集群实时查询）
func (m *BuilderPodManager) GetActivePods() []map[string]interface{} {
	var result []map[string]interface{}
	ctx := context.Background()

	// 检查 db 是否为 nil
	if m.db == nil {
		return result
	}

	// 获取所有已配置的集群
	var clusters []models.K8sCluster
	if err := m.db.Find(&clusters).Error; err != nil {
		return result
	}

	for _, cluster := range clusters {
		client, err := m.clientMgr.GetClient(ctx, cluster.ID)
		if err != nil {
			continue
		}

		// 查询带有 devops-builder 标签的 Pod
		pods, err := client.CoreV1().Pods("").List(ctx, metav1.ListOptions{
			LabelSelector: "app=devops-builder,managed-by=devops-platform",
		})
		if err != nil {
			continue
		}

		for _, pod := range pods.Items {
			if pod.Status.Phase != corev1.PodRunning {
				continue
			}

			// 计算空闲时间（从内存缓存获取，如果没有则用创建时间）
			var idleSeconds int
			var lastUsedAt time.Time
			key := fmt.Sprintf("%d-%s-%s", cluster.ID, pod.Namespace, pod.Spec.Containers[0].Image)
			if val, ok := m.pods.Load(key); ok {
				cachedPod := val.(*BuilderPod)
				cachedPod.mu.Lock()
				lastUsedAt = cachedPod.LastUsedAt
				cachedPod.mu.Unlock()
				idleSeconds = int(time.Since(lastUsedAt).Seconds())
			} else {
				lastUsedAt = pod.CreationTimestamp.Time
				idleSeconds = int(time.Since(pod.CreationTimestamp.Time).Seconds())
			}

			image := ""
			if len(pod.Spec.Containers) > 0 {
				image = pod.Spec.Containers[0].Image
			}

			result = append(result, map[string]interface{}{
				"cluster_id":   cluster.ID,
				"cluster_name": cluster.Name,
				"namespace":    pod.Namespace,
				"pod_name":     pod.Name,
				"image":        image,
				"last_used_at": lastUsedAt,
				"idle_seconds": idleSeconds,
				"status":       string(pod.Status.Phase),
			})
		}
	}

	return result
}

// DeletePodByName 根据名称删除 Pod
func (m *BuilderPodManager) DeletePodByName(ctx context.Context, clusterID uint, namespace, podName string) error {
	// 查找并删除
	var found bool
	m.pods.Range(func(key, value interface{}) bool {
		pod := value.(*BuilderPod)
		if pod.ClusterID == clusterID && pod.Namespace == namespace && pod.PodName == podName {
			m.deletePod(ctx, pod)
			m.pods.Delete(key)
			found = true
			return false
		}
		return true
	})

	if !found {
		// 即使不在缓存中，也尝试删除 K8s 中的 Pod
		client, err := m.clientMgr.GetClient(ctx, clusterID)
		if err != nil {
			return fmt.Errorf("获取 K8s 客户端失败: %v", err)
		}
		return client.CoreV1().Pods(namespace).Delete(ctx, podName, metav1.DeleteOptions{})
	}

	return nil
}
