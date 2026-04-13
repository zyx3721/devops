package middleware

import (
	"context"
	"net/http"
	"runtime/debug"
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	apperrors "devops/pkg/errors"
	"devops/pkg/logger"
	"devops/pkg/response"
)

// ErrorRecovery 错误恢复中间件
// 捕获 panic 并返回标准错误响应
func ErrorRecovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				// 获取堆栈信息
				stack := string(debug.Stack())

				// 记录详细日志
				logger.L().WithFields(map[string]interface{}{
					"error":      err,
					"path":       c.Request.URL.Path,
					"method":     c.Request.Method,
					"client_ip":  c.ClientIP(),
					"user_agent": c.Request.UserAgent(),
				}).Error("panic recovered:\n%s", stack)

				// 返回通用错误响应，不暴露内部错误详情
				c.AbortWithStatusJSON(http.StatusInternalServerError, response.Response{
					Code:    apperrors.ErrCodeInternalError,
					Message: "服务器内部错误",
				})
			}
		}()
		c.Next()
	}
}

// ErrorHandler 统一错误处理中间件
// 处理 handler 中设置的错误
func ErrorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		// 检查是否有错误
		if len(c.Errors) > 0 {
			err := c.Errors.Last().Err
			response.FromError(c, err)
			return
		}
	}
}

// RequestTimeout 请求超时中间件
func RequestTimeout(timeout time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), timeout)
		defer cancel()

		c.Request = c.Request.WithContext(ctx)

		finished := make(chan struct{}, 1)
		go func() {
			c.Next()
			finished <- struct{}{}
		}()

		select {
		case <-finished:
			// 正常完成
		case <-ctx.Done():
			// 超时
			c.AbortWithStatusJSON(http.StatusRequestTimeout, response.Response{
				Code:    apperrors.ErrCodeRequestTimeout,
				Message: "请求超时，请稍后重试",
			})
		}
	}
}

// RateLimiter 简单的请求限流器
type RateLimiter struct {
	mu       sync.Mutex
	requests map[string][]time.Time
	limit    int           // 限制次数
	window   time.Duration // 时间窗口
}

// NewRateLimiter 创建限流器
func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	rl := &RateLimiter{
		requests: make(map[string][]time.Time),
		limit:    limit,
		window:   window,
	}
	// 定期清理过期记录
	go rl.cleanup()
	return rl
}

func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(time.Minute)
	for range ticker.C {
		rl.mu.Lock()
		now := time.Now()
		for key, times := range rl.requests {
			var valid []time.Time
			for _, t := range times {
				if now.Sub(t) < rl.window {
					valid = append(valid, t)
				}
			}
			if len(valid) == 0 {
				delete(rl.requests, key)
			} else {
				rl.requests[key] = valid
			}
		}
		rl.mu.Unlock()
	}
}

// Allow 检查是否允许请求
func (rl *RateLimiter) Allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	times := rl.requests[key]

	// 清理过期记录
	var valid []time.Time
	for _, t := range times {
		if now.Sub(t) < rl.window {
			valid = append(valid, t)
		}
	}

	if len(valid) >= rl.limit {
		return false
	}

	valid = append(valid, now)
	rl.requests[key] = valid
	return true
}

// RateLimit 请求限流中间件
func RateLimit(limiter *RateLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := c.ClientIP()
		if !limiter.Allow(key) {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, response.Response{
				Code:    429,
				Message: "请求过于频繁，请稍后重试",
			})
			return
		}
		c.Next()
	}
}

// RequestLogger 请求日志中间件（增强版）
func RequestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()

		fields := map[string]interface{}{
			"status":     status,
			"method":     c.Request.Method,
			"path":       path,
			"query":      query,
			"ip":         c.ClientIP(),
			"latency":    latency.String(),
			"latency_ms": latency.Milliseconds(),
			"user_agent": c.Request.UserAgent(),
		}

		// 添加用户信息（如果有）
		if userID, exists := c.Get("user_id"); exists {
			fields["user_id"] = userID
		}
		if username, exists := c.Get("username"); exists {
			fields["username"] = username
		}

		// 根据状态码选择日志级别
		log := logger.L().WithFields(fields)
		if status >= 500 {
			log.Error("Request failed")
		} else if status >= 400 {
			log.Warn("Request error")
		} else if latency > 3*time.Second {
			log.Warn("Slow request")
		}
	}
}

// SecureHeaders 安全响应头中间件
func SecureHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Next()
	}
}

// NotFoundHandler 404 处理
func NotFoundHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusNotFound, response.Response{
			Code:    apperrors.ErrCodeNotFound,
			Message: "接口不存在",
		})
	}
}

// MethodNotAllowedHandler 405 处理
func MethodNotAllowedHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusMethodNotAllowed, response.Response{
			Code:    apperrors.ErrCodeMethodNotAllowed,
			Message: "请求方法不允许",
		})
	}
}
