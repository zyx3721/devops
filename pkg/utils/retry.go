package utils

import (
	"context"
	"math"
	"time"

	"devops/pkg/logger"
)

// RetryConfig 重试配置
type RetryConfig struct {
	MaxRetries     int           // 最大重试次数
	InitialBackoff time.Duration // 初始等待时间
	MaxBackoff     time.Duration // 最大等待时间
	Multiplier     float64       // 退避倍数
}

// DefaultRetryConfig 默认重试配置
// 效果: 2s -> 4s -> 8s -> 16s -> 30s，总共尝试约 1 分钟
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxRetries:     5,
		InitialBackoff: 2 * time.Second,
		MaxBackoff:     30 * time.Second,
		Multiplier:     2.0,
	}
}

// RetryWithBackoff 带指数退避的重试函数
// operation: 要执行的操作，返回 error 表示需要重试
// name: 操作名称，用于日志
func RetryWithBackoff(ctx context.Context, cfg RetryConfig, name string, operation func() error) error {
	var lastErr error
	backoff := cfg.InitialBackoff

	for attempt := 1; attempt <= cfg.MaxRetries; attempt++ {
		lastErr = operation()
		if lastErr == nil {
			if attempt > 1 {
				logger.L().Info("[%s] Connection succeeded after %d attempts", name, attempt)
			}
			return nil
		}

		if attempt == cfg.MaxRetries {
			break
		}

		logger.L().Warn("[%s] Connection failed, retrying in %v... (Attempt %d/%d): %v",
			name, backoff, attempt, cfg.MaxRetries, lastErr)

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(backoff):
		}

		// 计算下次退避时间
		backoff = time.Duration(float64(backoff) * cfg.Multiplier)
		if backoff > cfg.MaxBackoff {
			backoff = cfg.MaxBackoff
		}
	}

	logger.L().Error("[%s] Connection failed after %d attempts: %v", name, cfg.MaxRetries, lastErr)
	return lastErr
}

// RetryWithBackoffSimple 简化版重试函数，使用默认配置
func RetryWithBackoffSimple(name string, operation func() error) error {
	return RetryWithBackoff(context.Background(), DefaultRetryConfig(), name, operation)
}

// CalculateTotalWaitTime 计算总等待时间（用于文档/调试）
func CalculateTotalWaitTime(cfg RetryConfig) time.Duration {
	var total time.Duration
	backoff := cfg.InitialBackoff

	for i := 1; i < cfg.MaxRetries; i++ {
		total += backoff
		backoff = time.Duration(math.Min(float64(backoff)*cfg.Multiplier, float64(cfg.MaxBackoff)))
	}

	return total
}
