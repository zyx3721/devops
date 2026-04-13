package kubernetes

import (
	"context"
	"sync"
	"time"

	"gorm.io/gorm"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"devops/internal/models"
	apperrors "devops/pkg/errors"
	"devops/pkg/logger"
)

// clientEntry 缓存的客户端条目
type clientEntry struct {
	client    *kubernetes.Clientset
	config    *rest.Config
	createdAt time.Time
}

// K8sClientManager 管理 K8s 客户端连接
type K8sClientManager struct {
	db      *gorm.DB
	clients sync.Map // map[uint]*clientEntry
	maxAge  time.Duration
}

// NewK8sClientManager 创建客户端管理器
func NewK8sClientManager(db *gorm.DB) *K8sClientManager {
	return &K8sClientManager{
		db:     db,
		maxAge: 30 * time.Minute, // 客户端最大缓存时间
	}
}

// GetClient 获取指定集群的 K8s 客户端
func (m *K8sClientManager) GetClient(ctx context.Context, clusterID uint) (*kubernetes.Clientset, error) {
	log := logger.L().WithField("clusterID", clusterID)

	// 尝试从缓存获取
	if entry, ok := m.clients.Load(clusterID); ok {
		e := entry.(*clientEntry)
		// 检查是否过期
		if time.Since(e.createdAt) < m.maxAge {
			// 验证连接是否有效
			if m.isClientHealthy(ctx, e.client) {
				return e.client, nil
			}
			log.Warn("K8s客户端连接失效，重新创建")
		} else {
			log.Info("K8s客户端缓存过期，重新创建")
		}
		// 删除失效的缓存
		m.clients.Delete(clusterID)
	}

	// 从数据库获取集群配置
	var cluster models.K8sCluster
	if err := m.db.First(&cluster, clusterID).Error; err != nil {
		return nil, apperrors.Wrap(err, apperrors.ErrCodeNotFound, "集群不存在")
	}

	// 创建客户端
	config, err := clientcmd.RESTConfigFromKubeConfig([]byte(cluster.Kubeconfig))
	if err != nil {
		return nil, apperrors.Wrap(err, apperrors.ErrCodeInternalError, "解析kubeconfig失败")
	}

	// 设置超时
	config.Timeout = 10 * time.Second

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, apperrors.Wrap(err, apperrors.ErrCodeInternalError, "创建k8s客户端失败")
	}

	// 缓存客户端
	entry := &clientEntry{
		client:    clientset,
		config:    config,
		createdAt: time.Now(),
	}

	// 使用 LoadOrStore 防止并发创建多个客户端
	actual, loaded := m.clients.LoadOrStore(clusterID, entry)
	if loaded {
		return actual.(*clientEntry).client, nil
	}

	log.Info("创建新的K8s客户端连接")
	return clientset, nil
}

// isClientHealthy 检查客户端连接是否健康
func (m *K8sClientManager) isClientHealthy(ctx context.Context, client *kubernetes.Clientset) bool {
	// 使用短超时的 context
	checkCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	// 尝试获取 server version 来验证连接
	_, err := client.Discovery().ServerVersion()
	if err != nil {
		// 检查是否是 context 超时
		if checkCtx.Err() != nil {
			return false
		}
		return false
	}
	return true
}

// GetClientWithRetry 获取客户端，失败时自动重试
func (m *K8sClientManager) GetClientWithRetry(ctx context.Context, clusterID uint) (*kubernetes.Clientset, error) {
	var lastErr error
	for i := 0; i < 2; i++ {
		client, err := m.GetClient(ctx, clusterID)
		if err == nil {
			return client, nil
		}
		lastErr = err
		// 清除缓存后重试
		m.InvalidateClient(clusterID)
	}
	return nil, lastErr
}

// GetConfig 获取指定集群的 REST 配置
func (m *K8sClientManager) GetConfig(ctx context.Context, clusterID uint) (*rest.Config, error) {
	// 先尝试从缓存获取
	if entry, ok := m.clients.Load(clusterID); ok {
		e := entry.(*clientEntry)
		if time.Since(e.createdAt) < m.maxAge {
			return e.config, nil
		}
	}

	// 从数据库获取集群配置
	var cluster models.K8sCluster
	if err := m.db.First(&cluster, clusterID).Error; err != nil {
		return nil, apperrors.Wrap(err, apperrors.ErrCodeNotFound, "集群不存在")
	}

	// 创建配置
	config, err := clientcmd.RESTConfigFromKubeConfig([]byte(cluster.Kubeconfig))
	if err != nil {
		return nil, apperrors.Wrap(err, apperrors.ErrCodeInternalError, "解析kubeconfig失败")
	}

	config.Timeout = 10 * time.Second
	return config, nil
}

// GetRestConfig 获取指定集群的 REST 配置（别名）
func (m *K8sClientManager) GetRestConfig(ctx context.Context, clusterID uint) (*rest.Config, error) {
	return m.GetConfig(ctx, clusterID)
}

// InvalidateClient 使指定集群的客户端缓存失效
func (m *K8sClientManager) InvalidateClient(clusterID uint) {
	m.clients.Delete(clusterID)
	logger.L().Info("K8s客户端缓存已清除: clusterID=%d", clusterID)
}

// InvalidateAllClients 清除所有客户端缓存
func (m *K8sClientManager) InvalidateAllClients() {
	m.clients.Range(func(key, value interface{}) bool {
		m.clients.Delete(key)
		return true
	})
	logger.L().Info("所有K8s客户端缓存已清除")
}

// GetDB 获取数据库连接
func (m *K8sClientManager) GetDB() *gorm.DB {
	return m.db
}
