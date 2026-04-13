package resilience

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// Handler 限流熔断处理器
type Handler struct {
}

// NewHandler 创建处理器
func NewHandler() *Handler {
	return &Handler{}
}

// RegisterRoutes 注册路由
func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	g := r.Group("/resilience")
	{
		// 限流配置
		g.GET("/ratelimit/configs", h.GetRateLimitConfigs)
		g.POST("/ratelimit/configs", h.CreateRateLimitConfig)
		g.PUT("/ratelimit/configs/:id", h.UpdateRateLimitConfig)
		g.DELETE("/ratelimit/configs/:id", h.DeleteRateLimitConfig)
		g.GET("/ratelimit/stats", h.GetRateLimitStats)
		g.POST("/ratelimit/reset", h.ResetRateLimit)

		// 熔断器
		g.GET("/circuit/breakers", h.GetCircuitBreakers)
		g.GET("/circuit/breakers/:name/stats", h.GetCircuitBreakerStats)
		g.POST("/circuit/breakers/:name/reset", h.ResetCircuitBreaker)

		// 综合概览
		g.GET("/overview", h.GetOverview)
	}
}

// RateLimitConfigResponse 限流配置响应
type RateLimitConfigResponse struct {
	ID             int    `json:"id"`
	Name           string `json:"name"`
	Endpoint       string `json:"endpoint"`
	LimitType      string `json:"limit_type"` // ip, user, global
	RequestsPerMin int    `json:"requests_per_min"`
	BurstSize      int    `json:"burst_size"`
	WindowSeconds  int    `json:"window_seconds"`
	Enabled        bool   `json:"enabled"`
	Description    string `json:"description"`
	CreatedAt      string `json:"created_at"`
	UpdatedAt      string `json:"updated_at"`
}

// GetRateLimitConfigs 获取限流配置列表
func (h *Handler) GetRateLimitConfigs(c *gin.Context) {
	// 返回默认配置 + 自定义配置
	configs := []RateLimitConfigResponse{
		{ID: 1, Name: "api_read", Endpoint: "/api/v1/*", LimitType: "global", RequestsPerMin: 1000, BurstSize: 100, WindowSeconds: 60, Enabled: true, Description: "API 读取限流"},
		{ID: 2, Name: "api_write", Endpoint: "/api/v1/*", LimitType: "user", RequestsPerMin: 300, BurstSize: 50, WindowSeconds: 60, Enabled: true, Description: "API 写入限流"},
		{ID: 3, Name: "auth_login", Endpoint: "/api/v1/auth/login", LimitType: "ip", RequestsPerMin: 10, BurstSize: 5, WindowSeconds: 60, Enabled: true, Description: "登录限流"},
		{ID: 4, Name: "webhook", Endpoint: "/api/v1/webhook/*", LimitType: "global", RequestsPerMin: 100, BurstSize: 20, WindowSeconds: 60, Enabled: true, Description: "Webhook 限流"},
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"items": configs,
			"total": len(configs),
		},
	})
}

// CreateRateLimitConfigRequest 创建限流配置请求
type CreateRateLimitConfigRequest struct {
	Name           string `json:"name" binding:"required"`
	Endpoint       string `json:"endpoint" binding:"required"`
	LimitType      string `json:"limit_type" binding:"required"`
	RequestsPerMin int    `json:"requests_per_min" binding:"required,min=1"`
	BurstSize      int    `json:"burst_size" binding:"required,min=1"`
	WindowSeconds  int    `json:"window_seconds" binding:"required,min=1"`
	Description    string `json:"description"`
}

// CreateRateLimitConfig 创建限流配置
func (h *Handler) CreateRateLimitConfig(c *gin.Context) {
	var req CreateRateLimitConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误: " + err.Error()})
		return
	}

	// TODO: 保存到数据库
	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "创建成功",
	})
}

// UpdateRateLimitConfig 更新限流配置
func (h *Handler) UpdateRateLimitConfig(c *gin.Context) {
	id := c.Param("id")
	var req CreateRateLimitConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误: " + err.Error()})
		return
	}

	// TODO: 更新数据库
	_ = id
	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "更新成功",
	})
}

// DeleteRateLimitConfig 删除限流配置
func (h *Handler) DeleteRateLimitConfig(c *gin.Context) {
	id := c.Param("id")
	// TODO: 从数据库删除
	_ = id
	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "删除成功",
	})
}

// RateLimitStatsResponse 限流统计响应
type RateLimitStatsResponse struct {
	Key            string    `json:"key"`
	Endpoint       string    `json:"endpoint"`
	TotalRequests  int64     `json:"total_requests"`
	AllowedCount   int64     `json:"allowed_count"`
	RejectedCount  int64     `json:"rejected_count"`
	CurrentCount   int       `json:"current_count"`
	Limit          int       `json:"limit"`
	Remaining      int       `json:"remaining"`
	ResetAt        time.Time `json:"reset_at"`
	LastAccessTime time.Time `json:"last_access_time"`
}

// GetRateLimitStats 获取限流统计
func (h *Handler) GetRateLimitStats(c *gin.Context) {
	// 模拟统计数据
	stats := []RateLimitStatsResponse{
		{Key: "ip:192.168.1.100", Endpoint: "/api/v1/users", TotalRequests: 1500, AllowedCount: 1480, RejectedCount: 20, CurrentCount: 45, Limit: 100, Remaining: 55, ResetAt: time.Now().Add(30 * time.Second)},
		{Key: "user:1", Endpoint: "/api/v1/deploy", TotalRequests: 320, AllowedCount: 300, RejectedCount: 20, CurrentCount: 28, Limit: 50, Remaining: 22, ResetAt: time.Now().Add(45 * time.Second)},
		{Key: "global:webhook", Endpoint: "/api/v1/webhook", TotalRequests: 850, AllowedCount: 850, RejectedCount: 0, CurrentCount: 12, Limit: 100, Remaining: 88, ResetAt: time.Now().Add(50 * time.Second)},
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": stats,
	})
}

// ResetRateLimitRequest 重置限流请求
type ResetRateLimitRequest struct {
	Key string `json:"key" binding:"required"`
}

// ResetRateLimit 重置限流
func (h *Handler) ResetRateLimit(c *gin.Context) {
	var req ResetRateLimitRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误: " + err.Error()})
		return
	}

	// TODO: 实现限流重置逻辑
	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "重置成功",
	})
}

// CircuitBreakerResponse 熔断器响应
type CircuitBreakerResponse struct {
	Name             string    `json:"name"`
	State            string    `json:"state"`
	Requests         int64     `json:"requests"`
	Successes        int64     `json:"successes"`
	Failures         int64     `json:"failures"`
	ConsecutiveFails int64     `json:"consecutive_fails"`
	SuccessRate      float64   `json:"success_rate"`
	LastFailure      time.Time `json:"last_failure,omitempty"`
	Config           struct {
		MaxRequests      int `json:"max_requests"`
		FailureThreshold int `json:"failure_threshold"`
		SuccessThreshold int `json:"success_threshold"`
		TimeoutSeconds   int `json:"timeout_seconds"`
		IntervalSeconds  int `json:"interval_seconds"`
	} `json:"config"`
}

// GetCircuitBreakers 获取所有熔断器状态
func (h *Handler) GetCircuitBreakers(c *gin.Context) {
	// 返回示例数据
	breakers := []CircuitBreakerResponse{
		{Name: "jenkins-api", State: "closed", Requests: 5420, Successes: 5400, Failures: 20, ConsecutiveFails: 0, SuccessRate: 99.63},
		{Name: "k8s-api", State: "closed", Requests: 12350, Successes: 12340, Failures: 10, ConsecutiveFails: 0, SuccessRate: 99.92},
		{Name: "feishu-notify", State: "half-open", Requests: 890, Successes: 850, Failures: 40, ConsecutiveFails: 3, SuccessRate: 95.51},
		{Name: "dingtalk-notify", State: "open", Requests: 450, Successes: 380, Failures: 70, ConsecutiveFails: 8, SuccessRate: 84.44},
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": breakers,
	})
}

// GetCircuitBreakerStats 获取单个熔断器统计
func (h *Handler) GetCircuitBreakerStats(c *gin.Context) {
	name := c.Param("name")

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": CircuitBreakerResponse{
			Name:  name,
			State: "closed",
		},
	})
}

// ResetCircuitBreaker 重置熔断器
func (h *Handler) ResetCircuitBreaker(c *gin.Context) {
	// TODO: 实现熔断器重置逻辑
	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "熔断器已重置",
	})
}

// OverviewResponse 概览响应
type OverviewResponse struct {
	RateLimit struct {
		TotalConfigs   int     `json:"total_configs"`
		EnabledConfigs int     `json:"enabled_configs"`
		TotalRequests  int64   `json:"total_requests"`
		RejectedCount  int64   `json:"rejected_count"`
		RejectionRate  float64 `json:"rejection_rate"`
	} `json:"rate_limit"`
	CircuitBreaker struct {
		TotalBreakers    int     `json:"total_breakers"`
		OpenBreakers     int     `json:"open_breakers"`
		HalfOpenBreakers int     `json:"half_open_breakers"`
		ClosedBreakers   int     `json:"closed_breakers"`
		AvgSuccessRate   float64 `json:"avg_success_rate"`
	} `json:"circuit_breaker"`
	RecentEvents []ResilienceEvent `json:"recent_events"`
}

// ResilienceEvent 弹性事件
type ResilienceEvent struct {
	Time        time.Time `json:"time"`
	Type        string    `json:"type"` // rate_limit, circuit_open, circuit_close
	Target      string    `json:"target"`
	Description string    `json:"description"`
}

// GetOverview 获取概览
func (h *Handler) GetOverview(c *gin.Context) {
	overview := OverviewResponse{}

	// 限流统计
	overview.RateLimit.TotalConfigs = 4
	overview.RateLimit.EnabledConfigs = 4
	overview.RateLimit.TotalRequests = 25680
	overview.RateLimit.RejectedCount = 156
	overview.RateLimit.RejectionRate = 0.61

	// 熔断器统计
	overview.CircuitBreaker.TotalBreakers = 4
	overview.CircuitBreaker.OpenBreakers = 1
	overview.CircuitBreaker.HalfOpenBreakers = 1
	overview.CircuitBreaker.ClosedBreakers = 2
	overview.CircuitBreaker.AvgSuccessRate = 94.88

	// 最近事件
	overview.RecentEvents = []ResilienceEvent{
		{Time: time.Now().Add(-5 * time.Minute), Type: "circuit_open", Target: "dingtalk-notify", Description: "熔断器打开：连续失败 8 次"},
		{Time: time.Now().Add(-15 * time.Minute), Type: "rate_limit", Target: "ip:192.168.1.100", Description: "触发限流：超过 100 次/分钟"},
		{Time: time.Now().Add(-30 * time.Minute), Type: "circuit_close", Target: "feishu-notify", Description: "熔断器恢复：成功率恢复正常"},
		{Time: time.Now().Add(-1 * time.Hour), Type: "rate_limit", Target: "user:5", Description: "触发限流：超过 50 次/分钟"},
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": overview,
	})
}

// GetStateColor 获取状态颜色
func GetStateColor(state string) string {
	switch state {
	case "closed":
		return "success"
	case "half-open":
		return "warning"
	case "open":
		return "error"
	default:
		return "default"
	}
}

// ParseInt 解析整数
func ParseInt(s string, defaultVal int) int {
	if v, err := strconv.Atoi(s); err == nil {
		return v
	}
	return defaultVal
}
