package cost

import (
	"sync"
	"time"

	"gorm.io/gorm"

	"devops/internal/service/kubernetes"
	"devops/pkg/logger"
)

// CostCache 成本数据缓存
type CostCache struct {
	data      interface{}
	expiredAt time.Time
}

// CostService 成本服务
type CostService struct {
	db        *gorm.DB
	log       *logger.Logger
	cache     sync.Map // 缓存
	clientMgr *kubernetes.K8sClientManager
}

// NewCostService 创建成本服务
func NewCostService(db *gorm.DB) *CostService {
	return &CostService{
		db:        db,
		log:       logger.NewLogger("CostService"),
		clientMgr: kubernetes.NewK8sClientManager(db),
	}
}

// GetDB 获取数据库连接
func (s *CostService) GetDB() *gorm.DB {
	return s.db
}

// getCache 获取缓存
func (s *CostService) getCache(key string) (interface{}, bool) {
	if v, ok := s.cache.Load(key); ok {
		cache := v.(*CostCache)
		if time.Now().Before(cache.expiredAt) {
			return cache.data, true
		}
		s.cache.Delete(key)
	}
	return nil, false
}

// setCache 设置缓存
func (s *CostService) setCache(key string, data interface{}, ttl time.Duration) {
	s.cache.Store(key, &CostCache{
		data:      data,
		expiredAt: time.Now().Add(ttl),
	})
}

// InvalidateCache 清除缓存
func (s *CostService) InvalidateCache(prefix string) {
	s.cache.Range(func(key, _ interface{}) bool {
		if k, ok := key.(string); ok && (prefix == "" || len(k) >= len(prefix) && k[:len(prefix)] == prefix) {
			s.cache.Delete(key)
		}
		return true
	})
}
