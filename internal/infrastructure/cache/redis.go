package cache

import (
	"context"
	"time"

	"devops/internal/config"
	"devops/pkg/logger"
	"devops/pkg/utils"

	"github.com/go-redis/redis/v8"
)

// InitRedis 初始化 Redis 连接（带重试）
func InitRedis(cfg *config.Config) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:         cfg.RedisAddr,
		Password:     cfg.RedisPassword,
		DB:           cfg.RedisDB,
		PoolSize:     cfg.RedisPoolSize,
		MinIdleConns: cfg.RedisMinIdleConns,
	})

	// 带重试的连接测试
	err := utils.RetryWithBackoffSimple("Redis", func() error {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return client.Ping(ctx).Err()
	})

	if err != nil {
		client.Close()
		return nil, err
	}

	logger.L().Info("[Redis] Connected successfully to %s", cfg.RedisAddr)
	return client, nil
}
