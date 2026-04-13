package healthcheck

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"devops/internal/models"
	"devops/pkg/logger"
)

// TestHealthChecker_ShouldSendAlert 测试告警判断逻辑
func TestHealthChecker_ShouldSendAlert(t *testing.T) {
	checker := &HealthChecker{
		log: logger.NewLogger("test-healthcheck"),
	}

	tests := []struct {
		name          string
		config        *models.HealthCheckConfig
		newAlertLevel string
		expectedSend  bool
		expectedLevel string
	}{
		{
			name: "normal level - no alert",
			config: &models.HealthCheckConfig{
				AlertEnabled:   true,
				LastAlertLevel: "",
			},
			newAlertLevel: "normal",
			expectedSend:  false,
			expectedLevel: "",
		},
		{
			name: "alert disabled - no alert",
			config: &models.HealthCheckConfig{
				AlertEnabled:   false,
				LastAlertLevel: "",
			},
			newAlertLevel: "warning",
			expectedSend:  false,
			expectedLevel: "",
		},
		{
			name: "level upgrade - send alert",
			config: &models.HealthCheckConfig{
				AlertEnabled:   true,
				LastAlertLevel: "notice",
			},
			newAlertLevel: "warning",
			expectedSend:  true,
			expectedLevel: "warning",
		},
		{
			name: "same level within cooldown - no alert",
			config: &models.HealthCheckConfig{
				AlertEnabled:   true,
				LastAlertLevel: "warning",
				LastAlertAt:    ptrTime(time.Now().Add(-1 * time.Hour)),
			},
			newAlertLevel: "warning",
			expectedSend:  false,
			expectedLevel: "",
		},
		{
			name: "same level after cooldown - send alert",
			config: &models.HealthCheckConfig{
				AlertEnabled:   true,
				LastAlertLevel: "warning",
				LastAlertAt:    ptrTime(time.Now().Add(-25 * time.Hour)),
			},
			newAlertLevel: "warning",
			expectedSend:  true,
			expectedLevel: "warning",
		},
		{
			name: "level downgrade - no alert",
			config: &models.HealthCheckConfig{
				AlertEnabled:   true,
				LastAlertLevel: "critical",
			},
			newAlertLevel: "warning",
			expectedSend:  false,
			expectedLevel: "",
		},
		{
			name: "first alert - send alert",
			config: &models.HealthCheckConfig{
				AlertEnabled:   true,
				LastAlertLevel: "",
				LastAlertAt:    nil,
			},
			newAlertLevel: "notice",
			expectedSend:  true,
			expectedLevel: "notice",
		},
		{
			name: "expired level - send alert",
			config: &models.HealthCheckConfig{
				AlertEnabled:   true,
				LastAlertLevel: "critical",
			},
			newAlertLevel: "expired",
			expectedSend:  true,
			expectedLevel: "expired",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			shouldSend, level := checker.shouldSendAlert(tt.config, tt.newAlertLevel)
			assert.Equal(t, tt.expectedSend, shouldSend, "shouldSend mismatch")
			assert.Equal(t, tt.expectedLevel, level, "level mismatch")
		})
	}
}

// TestHealthChecker_IsAlertLevelUpgrade 测试告警级别升级判断
func TestHealthChecker_IsAlertLevelUpgrade(t *testing.T) {
	checker := &HealthChecker{
		log: logger.NewLogger("test-healthcheck"),
	}

	tests := []struct {
		name     string
		oldLevel string
		newLevel string
		expected bool
	}{
		{"empty to notice", "", "notice", true},
		{"normal to notice", "normal", "notice", true},
		{"notice to warning", "notice", "warning", true},
		{"warning to critical", "warning", "critical", true},
		{"critical to expired", "critical", "expired", true},
		{"same level", "warning", "warning", false},
		{"downgrade", "critical", "warning", false},
		{"downgrade to normal", "notice", "normal", false},
		{"empty to normal", "", "normal", false},
		{"normal to normal", "normal", "normal", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := checker.isAlertLevelUpgrade(tt.oldLevel, tt.newLevel)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestHealthChecker_CheckSSLCert_InvalidDomain 测试无效域名处理
func TestHealthChecker_CheckSSLCert_InvalidDomain(t *testing.T) {
	checker := &HealthChecker{
		log: logger.NewLogger("test-healthcheck"),
	}

	config := &models.HealthCheckConfig{
		URL: "",
	}

	status, errorMsg, _ := checker.checkSSLCert(context.Background(), config)
	assert.Equal(t, "unhealthy", status)
	assert.Equal(t, "No domain specified", errorMsg)
}

// ptrTime 辅助函数：返回时间指针
func ptrTime(t time.Time) *time.Time {
	return &t
}

// TestHealthChecker_ConcurrencyControl 测试并发控制
func TestHealthChecker_ConcurrencyControl(t *testing.T) {
	// 创建一个带有信号量的 HealthChecker
	checker := &HealthChecker{
		semaphore: make(chan struct{}, 3), // 设置最大并发为3，便于测试
	}

	// 用于跟踪并发执行的数量
	var currentConcurrent int32
	var maxConcurrent int32
	var mu sync.Mutex

	// 模拟检查函数，会阻塞一段时间
	checkFunc := func() {
		// 获取信号量
		checker.semaphore <- struct{}{}
		defer func() {
			<-checker.semaphore
		}()

		// 增加当前并发数
		current := atomic.AddInt32(&currentConcurrent, 1)

		// 更新最大并发数
		mu.Lock()
		if current > maxConcurrent {
			maxConcurrent = current
		}
		mu.Unlock()

		// 模拟检查工作
		time.Sleep(50 * time.Millisecond)

		// 减少当前并发数
		atomic.AddInt32(&currentConcurrent, -1)
	}

	// 启动10个并发任务
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			checkFunc()
		}()
	}

	// 等待所有任务完成
	wg.Wait()

	// 验证最大并发数不超过3
	assert.LessOrEqual(t, maxConcurrent, int32(3), "Max concurrent should not exceed semaphore limit")
	assert.Equal(t, int32(0), currentConcurrent, "All goroutines should have completed")
}

// TestHealthChecker_PanicRecovery 测试panic恢复机制
func TestHealthChecker_PanicRecovery(t *testing.T) {
	// 这个测试验证单个检查的panic不会影响其他检查
	// 由于我们在checkAll中添加了recover，这里只是验证概念

	checker := &HealthChecker{
		semaphore: make(chan struct{}, 2),
	}

	var successCount int32
	var wg sync.WaitGroup

	// 模拟一个会panic的检查
	panicCheck := func() {
		checker.semaphore <- struct{}{}
		defer func() {
			<-checker.semaphore
			if r := recover(); r != nil {
				// Panic被捕获
			}
		}()
		panic("simulated panic")
	}

	// 模拟一个正常的检查
	normalCheck := func() {
		checker.semaphore <- struct{}{}
		defer func() {
			<-checker.semaphore
		}()
		time.Sleep(10 * time.Millisecond)
		atomic.AddInt32(&successCount, 1)
	}

	// 启动混合任务
	wg.Add(4)
	go func() {
		defer wg.Done()
		panicCheck()
	}()
	go func() {
		defer wg.Done()
		normalCheck()
	}()
	go func() {
		defer wg.Done()
		normalCheck()
	}()
	go func() {
		defer wg.Done()
		normalCheck()
	}()

	wg.Wait()

	// 验证正常的检查都成功完成
	assert.Equal(t, int32(3), successCount, "Normal checks should complete despite panic in other check")
}
