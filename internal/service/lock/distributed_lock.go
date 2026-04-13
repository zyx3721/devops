package lock

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
)

var (
	ErrLockNotAcquired = errors.New("无法获取锁")
	ErrLockNotHeld     = errors.New("未持有锁")
	ErrLockExpired     = errors.New("锁已过期")
)

// DistributedLock 分布式锁接口
type DistributedLock interface {
	// 获取锁
	Acquire(ctx context.Context, key string, ttl time.Duration) (string, error)
	// 尝试获取锁（非阻塞）
	TryAcquire(ctx context.Context, key string, ttl time.Duration) (string, bool, error)
	// 释放锁
	Release(ctx context.Context, key string, token string) error
	// 续期
	Extend(ctx context.Context, key string, token string, ttl time.Duration) error
	// 检查锁状态
	IsLocked(ctx context.Context, key string) (bool, error)
}

// RedisLock Redis 实现的分布式锁
type RedisLock struct {
	client *redis.Client
	prefix string
}

// NewRedisLock 创建 Redis 分布式锁
func NewRedisLock(client *redis.Client, prefix string) DistributedLock {
	if prefix == "" {
		prefix = "lock:"
	}
	return &RedisLock{
		client: client,
		prefix: prefix,
	}
}

func (l *RedisLock) Acquire(ctx context.Context, key string, ttl time.Duration) (string, error) {
	token := uuid.New().String()
	fullKey := l.prefix + key

	// 尝试获取锁，最多重试 10 次
	for i := 0; i < 10; i++ {
		ok, err := l.client.SetNX(ctx, fullKey, token, ttl).Result()
		if err != nil {
			return "", fmt.Errorf("获取锁失败: %w", err)
		}
		if ok {
			return token, nil
		}

		// 等待一段时间后重试
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(100 * time.Millisecond):
		}
	}

	return "", ErrLockNotAcquired
}

func (l *RedisLock) TryAcquire(ctx context.Context, key string, ttl time.Duration) (string, bool, error) {
	token := uuid.New().String()
	fullKey := l.prefix + key

	ok, err := l.client.SetNX(ctx, fullKey, token, ttl).Result()
	if err != nil {
		return "", false, fmt.Errorf("获取锁失败: %w", err)
	}

	return token, ok, nil
}

func (l *RedisLock) Release(ctx context.Context, key string, token string) error {
	fullKey := l.prefix + key

	// 使用 Lua 脚本确保只释放自己持有的锁
	script := `
		if redis.call("get", KEYS[1]) == ARGV[1] then
			return redis.call("del", KEYS[1])
		else
			return 0
		end
	`

	result, err := l.client.Eval(ctx, script, []string{fullKey}, token).Int()
	if err != nil {
		return fmt.Errorf("释放锁失败: %w", err)
	}

	if result == 0 {
		return ErrLockNotHeld
	}

	return nil
}

func (l *RedisLock) Extend(ctx context.Context, key string, token string, ttl time.Duration) error {
	fullKey := l.prefix + key

	// 使用 Lua 脚本确保只续期自己持有的锁
	script := `
		if redis.call("get", KEYS[1]) == ARGV[1] then
			return redis.call("pexpire", KEYS[1], ARGV[2])
		else
			return 0
		end
	`

	result, err := l.client.Eval(ctx, script, []string{fullKey}, token, ttl.Milliseconds()).Int()
	if err != nil {
		return fmt.Errorf("续期失败: %w", err)
	}

	if result == 0 {
		return ErrLockNotHeld
	}

	return nil
}

func (l *RedisLock) IsLocked(ctx context.Context, key string) (bool, error) {
	fullKey := l.prefix + key
	exists, err := l.client.Exists(ctx, fullKey).Result()
	if err != nil {
		return false, fmt.Errorf("检查锁状态失败: %w", err)
	}
	return exists > 0, nil
}

// MemoryLock 内存实现的分布式锁（单机使用）
type MemoryLock struct {
	prefix string
	locks  sync.Map
	mu     sync.Mutex
}

type lockEntry struct {
	token     string
	expiresAt time.Time
}

// NewMemoryLock 创建内存锁
func NewMemoryLock(prefix string) DistributedLock {
	if prefix == "" {
		prefix = "lock:"
	}
	ml := &MemoryLock{
		prefix: prefix,
	}
	// 启动后台清理过期锁的 goroutine
	go ml.cleanupExpiredLocks()
	return ml
}

// cleanupExpiredLocks 定期清理过期的锁
func (l *MemoryLock) cleanupExpiredLocks() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		now := time.Now()
		l.locks.Range(func(key, value any) bool {
			entry := value.(*lockEntry)
			if now.After(entry.expiresAt) {
				l.locks.Delete(key)
			}
			return true
		})
	}
}

func (l *MemoryLock) Acquire(ctx context.Context, key string, ttl time.Duration) (string, error) {
	token := uuid.New().String()
	fullKey := l.prefix + key

	for i := 0; i < 10; i++ {
		if l.trySet(fullKey, token, ttl) {
			return token, nil
		}

		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(100 * time.Millisecond):
		}
	}

	return "", ErrLockNotAcquired
}

func (l *MemoryLock) TryAcquire(ctx context.Context, key string, ttl time.Duration) (string, bool, error) {
	token := uuid.New().String()
	fullKey := l.prefix + key

	if l.trySet(fullKey, token, ttl) {
		return token, true, nil
	}

	return "", false, nil
}

func (l *MemoryLock) Release(ctx context.Context, key string, token string) error {
	fullKey := l.prefix + key

	l.mu.Lock()
	defer l.mu.Unlock()

	val, ok := l.locks.Load(fullKey)
	if !ok {
		return ErrLockNotHeld
	}

	entry := val.(*lockEntry)
	if entry.token != token {
		return ErrLockNotHeld
	}

	l.locks.Delete(fullKey)
	return nil
}

func (l *MemoryLock) Extend(ctx context.Context, key string, token string, ttl time.Duration) error {
	fullKey := l.prefix + key

	l.mu.Lock()
	defer l.mu.Unlock()

	val, ok := l.locks.Load(fullKey)
	if !ok {
		return ErrLockNotHeld
	}

	entry := val.(*lockEntry)
	if entry.token != token {
		return ErrLockNotHeld
	}

	entry.expiresAt = time.Now().Add(ttl)
	return nil
}

func (l *MemoryLock) IsLocked(ctx context.Context, key string) (bool, error) {
	fullKey := l.prefix + key

	val, ok := l.locks.Load(fullKey)
	if !ok {
		return false, nil
	}

	entry := val.(*lockEntry)
	if time.Now().After(entry.expiresAt) {
		l.locks.Delete(fullKey)
		return false, nil
	}

	return true, nil
}

func (l *MemoryLock) trySet(key, token string, ttl time.Duration) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	// 检查是否已存在且未过期
	if val, ok := l.locks.Load(key); ok {
		entry := val.(*lockEntry)
		if time.Now().Before(entry.expiresAt) {
			return false
		}
	}

	// 设置新锁
	l.locks.Store(key, &lockEntry{
		token:     token,
		expiresAt: time.Now().Add(ttl),
	})

	return true
}

// LockManager 锁管理器
type LockManager struct {
	lock DistributedLock
}

// NewLockManager 创建锁管理器
func NewLockManager(redisClient *redis.Client) *LockManager {
	var lock DistributedLock
	if redisClient != nil {
		lock = NewRedisLock(redisClient, "devops:lock:")
	} else {
		lock = NewMemoryLock("devops:lock:")
	}
	return &LockManager{lock: lock}
}

// WithLock 在锁保护下执行函数
func (m *LockManager) WithLock(ctx context.Context, key string, ttl time.Duration, fn func() error) error {
	token, err := m.lock.Acquire(ctx, key, ttl)
	if err != nil {
		return err
	}
	defer m.lock.Release(ctx, key, token)

	return fn()
}

// TryWithLock 尝试在锁保护下执行函数
func (m *LockManager) TryWithLock(ctx context.Context, key string, ttl time.Duration, fn func() error) (bool, error) {
	token, acquired, err := m.lock.TryAcquire(ctx, key, ttl)
	if err != nil {
		return false, err
	}
	if !acquired {
		return false, nil
	}
	defer m.lock.Release(ctx, key, token)

	return true, fn()
}
