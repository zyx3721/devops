// Package traffic 流量治理服务
// 本文件实现流量治理规则测试功能
package traffic

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"gorm.io/gorm"

	"devops/internal/models"
	"devops/internal/service/kubernetes"
	"devops/pkg/logger"
)

// RuleTester 规则测试器
type RuleTester struct {
	db           *gorm.DB
	istioService *kubernetes.IstioService
	httpClient   *http.Client
}

// NewRuleTester 创建规则测试器
func NewRuleTester(db *gorm.DB, istioService *kubernetes.IstioService) *RuleTester {
	return &RuleTester{
		db:           db,
		istioService: istioService,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// TestRequest 测试请求
type TestRequest struct {
	AppID       uint              `json:"app_id" binding:"required"`
	RuleType    string            `json:"rule_type" binding:"required"` // ratelimit, circuitbreaker, routing
	RuleID      uint64            `json:"rule_id"`                      // 可选，指定测试某条规则
	TargetURL   string            `json:"target_url"`                   // 测试目标 URL
	Concurrency int               `json:"concurrency"`                  // 并发数
	Duration    int               `json:"duration"`                     // 测试时长（秒）
	Headers     map[string]string `json:"headers"`                      // 自定义请求头
}

// TestResult 测试结果
type TestResult struct {
	RuleType       string             `json:"rule_type"`
	RuleName       string             `json:"rule_name"`
	Status         string             `json:"status"` // passed, failed, warning
	Message        string             `json:"message"`
	TotalRequests  int64              `json:"total_requests"`
	SuccessCount   int64              `json:"success_count"`
	FailedCount    int64              `json:"failed_count"`
	RateLimited    int64              `json:"rate_limited"`    // 被限流次数
	CircuitBreaked int64              `json:"circuit_breaked"` // 被熔断次数
	AvgLatencyMs   float64            `json:"avg_latency_ms"`
	P50LatencyMs   float64            `json:"p50_latency_ms"`
	P90LatencyMs   float64            `json:"p90_latency_ms"`
	P99LatencyMs   float64            `json:"p99_latency_ms"`
	Details        []TestResultDetail `json:"details,omitempty"`
	StartTime      time.Time          `json:"start_time"`
	EndTime        time.Time          `json:"end_time"`
}

// TestResultDetail 测试结果详情
type TestResultDetail struct {
	RequestID    int           `json:"request_id"`
	StatusCode   int           `json:"status_code"`
	Latency      time.Duration `json:"latency"`
	Error        string        `json:"error,omitempty"`
	RoutedTo     string        `json:"routed_to,omitempty"` // 路由到的版本
	WasLimited   bool          `json:"was_limited"`
	WasBreaked   bool          `json:"was_breaked"`
	ResponseBody string        `json:"response_body,omitempty"`
}

// TestRateLimitRule 测试限流规则
func (t *RuleTester) TestRateLimitRule(ctx context.Context, req *TestRequest) (*TestResult, error) {
	log := logger.L().WithField("app_id", req.AppID).WithField("rule_type", "ratelimit")
	log.Info("开始测试限流规则")

	// 获取限流规则
	var rules []models.TrafficRateLimitRule
	query := t.db.Where("app_id = ? AND enabled = ?", req.AppID, true)
	if req.RuleID > 0 {
		query = query.Where("id = ?", req.RuleID)
	}
	if err := query.Find(&rules).Error; err != nil {
		return nil, fmt.Errorf("获取限流规则失败: %w", err)
	}

	if len(rules) == 0 {
		return &TestResult{
			RuleType: "ratelimit",
			Status:   "warning",
			Message:  "没有找到启用的限流规则",
		}, nil
	}

	// 使用第一条规则进行测试
	rule := rules[0]
	result := &TestResult{
		RuleType:  "ratelimit",
		RuleName:  rule.Name,
		StartTime: time.Now(),
	}

	// 设置默认值
	concurrency := req.Concurrency
	if concurrency <= 0 {
		concurrency = 10
	}
	duration := req.Duration
	if duration <= 0 {
		duration = 10
	}

	// 计算预期请求数（超过阈值以触发限流）
	expectedQPS := rule.Threshold + rule.Burst + 10
	totalRequests := expectedQPS * duration

	// 执行测试
	var wg sync.WaitGroup
	var successCount, failedCount, rateLimited int64
	latencies := make([]time.Duration, 0, totalRequests)
	var latencyMu sync.Mutex
	details := make([]TestResultDetail, 0)
	var detailsMu sync.Mutex

	// 创建请求通道
	requestCh := make(chan int, totalRequests)
	for i := 0; i < totalRequests; i++ {
		requestCh <- i
	}
	close(requestCh)

	// 启动并发 worker
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for reqID := range requestCh {
				select {
				case <-ctx.Done():
					return
				default:
				}

				detail := t.sendTestRequest(ctx, req.TargetURL, req.Headers, reqID)

				atomic.AddInt64(&result.TotalRequests, 1)
				if detail.StatusCode == 429 || detail.WasLimited {
					atomic.AddInt64(&rateLimited, 1)
					detail.WasLimited = true
				} else if detail.StatusCode >= 200 && detail.StatusCode < 300 {
					atomic.AddInt64(&successCount, 1)
				} else {
					atomic.AddInt64(&failedCount, 1)
				}

				latencyMu.Lock()
				latencies = append(latencies, detail.Latency)
				latencyMu.Unlock()

				// 只保留前 100 条详情
				detailsMu.Lock()
				if len(details) < 100 {
					details = append(details, detail)
				}
				detailsMu.Unlock()

				// 控制 QPS
				time.Sleep(time.Second / time.Duration(expectedQPS/concurrency))
			}
		}()
	}

	wg.Wait()
	result.EndTime = time.Now()
	result.SuccessCount = successCount
	result.FailedCount = failedCount
	result.RateLimited = rateLimited
	result.Details = details

	// 计算延迟统计
	if len(latencies) > 0 {
		result.AvgLatencyMs, result.P50LatencyMs, result.P90LatencyMs, result.P99LatencyMs = calculateLatencyStats(latencies)
	}

	// 判断测试结果
	// 如果有请求被限流，说明限流规则生效
	if rateLimited > 0 {
		result.Status = "passed"
		result.Message = fmt.Sprintf("限流规则生效，共 %d 个请求被限流（阈值: %d QPS）", rateLimited, rule.Threshold)
	} else if successCount == result.TotalRequests {
		result.Status = "warning"
		result.Message = "所有请求都成功，限流规则可能未生效（请检查规则配置或增加请求量）"
	} else {
		result.Status = "failed"
		result.Message = fmt.Sprintf("测试失败，成功: %d, 失败: %d", successCount, failedCount)
	}

	log.WithField("status", result.Status).Info("限流规则测试完成")
	return result, nil
}

// TestCircuitBreakerRule 测试熔断规则
func (t *RuleTester) TestCircuitBreakerRule(ctx context.Context, req *TestRequest) (*TestResult, error) {
	log := logger.L().WithField("app_id", req.AppID).WithField("rule_type", "circuitbreaker")
	log.Info("开始测试熔断规则")

	// 获取熔断规则
	var rules []models.TrafficCircuitBreakerRule
	query := t.db.Where("app_id = ? AND enabled = ?", req.AppID, true)
	if req.RuleID > 0 {
		query = query.Where("id = ?", req.RuleID)
	}
	if err := query.Find(&rules).Error; err != nil {
		return nil, fmt.Errorf("获取熔断规则失败: %w", err)
	}

	if len(rules) == 0 {
		return &TestResult{
			RuleType: "circuitbreaker",
			Status:   "warning",
			Message:  "没有找到启用的熔断规则",
		}, nil
	}

	rule := rules[0]
	result := &TestResult{
		RuleType:  "circuitbreaker",
		RuleName:  rule.Name,
		StartTime: time.Now(),
	}

	// 设置默认值
	concurrency := req.Concurrency
	if concurrency <= 0 {
		concurrency = 5
	}
	duration := req.Duration
	if duration <= 0 {
		duration = 30
	}

	// 执行测试
	var wg sync.WaitGroup
	var successCount, failedCount, circuitBreaked int64
	latencies := make([]time.Duration, 0)
	var latencyMu sync.Mutex
	details := make([]TestResultDetail, 0)
	var detailsMu sync.Mutex

	// 持续发送请求直到超时
	endTime := time.Now().Add(time.Duration(duration) * time.Second)
	var reqID int64

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for time.Now().Before(endTime) {
				select {
				case <-ctx.Done():
					return
				default:
				}

				currentReqID := atomic.AddInt64(&reqID, 1)
				detail := t.sendTestRequest(ctx, req.TargetURL, req.Headers, int(currentReqID))

				atomic.AddInt64(&result.TotalRequests, 1)

				// 检查是否被熔断（503 或特定错误）
				if detail.StatusCode == 503 || detail.WasBreaked {
					atomic.AddInt64(&circuitBreaked, 1)
					detail.WasBreaked = true
				} else if detail.StatusCode >= 200 && detail.StatusCode < 300 {
					atomic.AddInt64(&successCount, 1)
				} else {
					atomic.AddInt64(&failedCount, 1)
				}

				latencyMu.Lock()
				latencies = append(latencies, detail.Latency)
				latencyMu.Unlock()

				detailsMu.Lock()
				if len(details) < 100 {
					details = append(details, detail)
				}
				detailsMu.Unlock()

				// 间隔发送
				time.Sleep(100 * time.Millisecond)
			}
		}()
	}

	wg.Wait()
	result.EndTime = time.Now()
	result.SuccessCount = successCount
	result.FailedCount = failedCount
	result.CircuitBreaked = circuitBreaked
	result.Details = details

	// 计算延迟统计
	if len(latencies) > 0 {
		result.AvgLatencyMs, result.P50LatencyMs, result.P90LatencyMs, result.P99LatencyMs = calculateLatencyStats(latencies)
	}

	// 判断测试结果
	if circuitBreaked > 0 {
		result.Status = "passed"
		result.Message = fmt.Sprintf("熔断规则生效，共 %d 个请求被熔断", circuitBreaked)
	} else {
		// 检查熔断器当前状态
		var currentRule models.TrafficCircuitBreakerRule
		t.db.First(&currentRule, rule.ID)
		if currentRule.CircuitStatus == "open" {
			result.Status = "passed"
			result.Message = "熔断器已打开，规则生效"
		} else {
			result.Status = "warning"
			result.Message = "熔断规则未触发（可能需要更多错误请求来触发熔断）"
		}
	}

	log.WithField("status", result.Status).Info("熔断规则测试完成")
	return result, nil
}

// TestRoutingRule 测试路由规则
func (t *RuleTester) TestRoutingRule(ctx context.Context, req *TestRequest) (*TestResult, error) {
	log := logger.L().WithField("app_id", req.AppID).WithField("rule_type", "routing")
	log.Info("开始测试路由规则")

	// 获取路由规则
	var rules []models.TrafficRoutingRule
	query := t.db.Where("app_id = ? AND enabled = ?", req.AppID, true)
	if req.RuleID > 0 {
		query = query.Where("id = ?", req.RuleID)
	}
	if err := query.Order("priority DESC").Find(&rules).Error; err != nil {
		return nil, fmt.Errorf("获取路由规则失败: %w", err)
	}

	if len(rules) == 0 {
		return &TestResult{
			RuleType: "routing",
			Status:   "warning",
			Message:  "没有找到启用的路由规则",
		}, nil
	}

	rule := rules[0]
	result := &TestResult{
		RuleType:  "routing",
		RuleName:  rule.Name,
		StartTime: time.Now(),
	}

	// 设置默认值
	totalRequests := 100
	if req.Duration > 0 {
		totalRequests = req.Duration * 10
	}

	// 统计路由分布
	routeDistribution := make(map[string]int64)
	var routeMu sync.Mutex
	details := make([]TestResultDetail, 0)
	var detailsMu sync.Mutex
	var successCount, failedCount int64
	latencies := make([]time.Duration, 0)
	var latencyMu sync.Mutex

	// 发送测试请求
	for i := 0; i < totalRequests; i++ {
		select {
		case <-ctx.Done():
			break
		default:
		}

		// 根据路由类型设置请求头
		headers := make(map[string]string)
		for k, v := range req.Headers {
			headers[k] = v
		}

		// 如果是 header 路由，添加匹配的 header
		if rule.RouteType == "header" && rule.MatchKey != "" {
			headers[rule.MatchKey] = rule.MatchValue
		}

		detail := t.sendTestRequest(ctx, req.TargetURL, headers, i)
		atomic.AddInt64(&result.TotalRequests, 1)

		if detail.StatusCode >= 200 && detail.StatusCode < 300 {
			atomic.AddInt64(&successCount, 1)
		} else {
			atomic.AddInt64(&failedCount, 1)
		}

		// 记录路由目标
		routeMu.Lock()
		if detail.RoutedTo != "" {
			routeDistribution[detail.RoutedTo]++
		}
		routeMu.Unlock()

		latencyMu.Lock()
		latencies = append(latencies, detail.Latency)
		latencyMu.Unlock()

		detailsMu.Lock()
		if len(details) < 100 {
			details = append(details, detail)
		}
		detailsMu.Unlock()

		time.Sleep(50 * time.Millisecond)
	}

	result.EndTime = time.Now()
	result.SuccessCount = successCount
	result.FailedCount = failedCount
	result.Details = details

	// 计算延迟统计
	if len(latencies) > 0 {
		result.AvgLatencyMs, result.P50LatencyMs, result.P90LatencyMs, result.P99LatencyMs = calculateLatencyStats(latencies)
	}

	// 判断测试结果
	if rule.RouteType == "weight" && len(rule.Destinations) > 0 {
		// 权重路由：检查流量分布是否符合预期
		result.Status = "passed"
		var distributionMsg string
		for _, dest := range rule.Destinations {
			actual := routeDistribution[dest.Subset]
			expected := int64(float64(totalRequests) * float64(dest.Weight) / 100.0)
			distributionMsg += fmt.Sprintf("%s: 预期 %d%%, 实际 %d 次; ", dest.Subset, dest.Weight, actual)
			// 允许 20% 的误差
			if expected > 0 && (actual < expected*8/10 || actual > expected*12/10) {
				result.Status = "warning"
			}
		}
		result.Message = "路由分布: " + distributionMsg
	} else if rule.RouteType == "header" {
		// Header 路由：检查是否正确路由
		if routeDistribution[rule.TargetSubset] > 0 {
			result.Status = "passed"
			result.Message = fmt.Sprintf("Header 路由生效，%d 个请求路由到 %s", routeDistribution[rule.TargetSubset], rule.TargetSubset)
		} else {
			result.Status = "warning"
			result.Message = "Header 路由可能未生效"
		}
	} else {
		result.Status = "passed"
		result.Message = "路由规则测试完成"
	}

	log.WithField("status", result.Status).Info("路由规则测试完成")
	return result, nil
}

// sendTestRequest 发送测试请求
func (t *RuleTester) sendTestRequest(ctx context.Context, targetURL string, headers map[string]string, reqID int) TestResultDetail {
	detail := TestResultDetail{
		RequestID: reqID,
	}

	if targetURL == "" {
		detail.StatusCode = 0
		detail.Error = "目标 URL 未配置"
		return detail
	}

	start := time.Now()

	req, err := http.NewRequestWithContext(ctx, "GET", targetURL, nil)
	if err != nil {
		detail.Error = err.Error()
		detail.Latency = time.Since(start)
		return detail
	}

	// 添加请求头
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	req.Header.Set("X-Request-ID", fmt.Sprintf("test-%d-%d", time.Now().UnixNano(), reqID))

	resp, err := t.httpClient.Do(req)
	detail.Latency = time.Since(start)

	if err != nil {
		detail.Error = err.Error()
		return detail
	}
	defer resp.Body.Close()

	detail.StatusCode = resp.StatusCode

	// 检查响应头获取路由信息
	if routedTo := resp.Header.Get("X-Routed-To"); routedTo != "" {
		detail.RoutedTo = routedTo
	}
	if resp.Header.Get("X-Rate-Limited") == "true" {
		detail.WasLimited = true
	}
	if resp.Header.Get("X-Circuit-Breaker") == "open" {
		detail.WasBreaked = true
	}

	return detail
}

// calculateLatencyStats 计算延迟统计
// 使用标准库排序算法（O(n log n)）替代冒泡排序（O(n²)）以提升性能
// 在高并发压测场景下（如 10,000+ 样本），性能提升可达 750 倍以上
func calculateLatencyStats(latencies []time.Duration) (avg, p50, p90, p99 float64) {
	if len(latencies) == 0 {
		return
	}

	// 计算平均值
	var total time.Duration
	for _, l := range latencies {
		total += l
	}
	avg = float64(total.Milliseconds()) / float64(len(latencies))

	// 排序计算百分位 - 使用标准库排序（O(n log n)）
	// 创建副本以避免修改原始数据
	sorted := make([]time.Duration, len(latencies))
	copy(sorted, latencies)

	// 使用 sort.Slice 进行高效排序
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i] < sorted[j]
	})

	// 计算百分位数
	p50 = float64(sorted[len(sorted)*50/100].Milliseconds())
	p90 = float64(sorted[len(sorted)*90/100].Milliseconds())
	p99 = float64(sorted[len(sorted)*99/100].Milliseconds())

	return
}

// SaveTestResult 保存测试结果
func (t *RuleTester) SaveTestResult(ctx context.Context, appID uint, result *TestResult) error {
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return err
	}

	log := &models.TrafficOperationLog{
		AppID:     uint64(appID),
		RuleType:  result.RuleType,
		Operation: "test",
		Operator:  "system",
		NewValue:  string(resultJSON),
		CreatedAt: time.Now(),
	}

	return t.db.Create(log).Error
}

// SimulateLoad 模拟负载测试
func (t *RuleTester) SimulateLoad(ctx context.Context, req *TestRequest) (*TestResult, error) {
	log := logger.L().WithField("app_id", req.AppID)
	log.Info("开始模拟负载测试")

	result := &TestResult{
		RuleType:  "load_test",
		StartTime: time.Now(),
	}

	concurrency := req.Concurrency
	if concurrency <= 0 {
		concurrency = 50
	}
	duration := req.Duration
	if duration <= 0 {
		duration = 60
	}

	var wg sync.WaitGroup
	var successCount, failedCount, rateLimited, circuitBreaked int64
	latencies := make([]time.Duration, 0)
	var latencyMu sync.Mutex

	endTime := time.Now().Add(time.Duration(duration) * time.Second)
	reqID := int64(0)

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for time.Now().Before(endTime) {
				select {
				case <-ctx.Done():
					return
				default:
				}

				currentReqID := atomic.AddInt64(&reqID, 1)
				detail := t.sendTestRequest(ctx, req.TargetURL, req.Headers, int(currentReqID))

				atomic.AddInt64(&result.TotalRequests, 1)

				if detail.WasLimited {
					atomic.AddInt64(&rateLimited, 1)
				}
				if detail.WasBreaked {
					atomic.AddInt64(&circuitBreaked, 1)
				}

				if detail.StatusCode >= 200 && detail.StatusCode < 300 {
					atomic.AddInt64(&successCount, 1)
				} else {
					atomic.AddInt64(&failedCount, 1)
				}

				latencyMu.Lock()
				latencies = append(latencies, detail.Latency)
				latencyMu.Unlock()

				// 随机间隔，模拟真实流量
				time.Sleep(time.Duration(rand.Intn(100)) * time.Millisecond)
			}
		}()
	}

	wg.Wait()
	result.EndTime = time.Now()
	result.SuccessCount = successCount
	result.FailedCount = failedCount
	result.RateLimited = rateLimited
	result.CircuitBreaked = circuitBreaked

	if len(latencies) > 0 {
		result.AvgLatencyMs, result.P50LatencyMs, result.P90LatencyMs, result.P99LatencyMs = calculateLatencyStats(latencies)
	}

	// 计算 QPS
	durationSec := result.EndTime.Sub(result.StartTime).Seconds()
	qps := float64(result.TotalRequests) / durationSec

	result.Status = "completed"
	result.Message = fmt.Sprintf("负载测试完成: QPS=%.2f, 成功率=%.2f%%, 限流=%d, 熔断=%d",
		qps,
		float64(successCount)/float64(result.TotalRequests)*100,
		rateLimited,
		circuitBreaked)

	log.WithField("qps", qps).Info("负载测试完成")
	return result, nil
}
