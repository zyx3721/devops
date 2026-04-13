package pipeline

import (
	"context"
	"devops/internal/models"
	"devops/pkg/logger"
	"fmt"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"gorm.io/gorm"
)

var (
	// 构建任务计数器
	buildJobsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "pipeline_build_jobs_total",
			Help: "Total number of build jobs",
		},
		[]string{"pipeline_id", "status"},
	)

	// 构建任务耗时直方图
	buildJobDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "pipeline_build_job_duration_seconds",
			Help:    "Build job duration in seconds",
			Buckets: []float64{30, 60, 120, 300, 600, 1200, 1800, 3600},
		},
		[]string{"pipeline_id"},
	)

	// 当前运行中的任务数
	buildJobsRunning = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "pipeline_build_jobs_running",
			Help: "Number of currently running build jobs",
		},
		[]string{"pipeline_id"},
	)

	// 构建队列长度
	buildQueueLength = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "pipeline_build_queue_length",
			Help: "Number of pending build jobs in queue",
		},
	)

	// 构建成功率
	buildSuccessRate = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "pipeline_build_success_rate",
			Help: "Build success rate (0-1)",
		},
		[]string{"pipeline_id"},
	)

	// 制品数量
	artifactsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "pipeline_artifacts_total",
			Help: "Total number of artifacts created",
		},
		[]string{"pipeline_id", "type"},
	)

	// 制品大小
	artifactsSize = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "pipeline_artifacts_size_bytes",
			Help: "Total size of artifacts in bytes",
		},
		[]string{"pipeline_id"},
	)

	// 缓存命中率
	cacheHitRate = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "pipeline_cache_hit_rate",
			Help: "Cache hit rate (0-1)",
		},
		[]string{"pipeline_id"},
	)

	// Webhook 触发计数
	webhookTriggersTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "pipeline_webhook_triggers_total",
			Help: "Total number of webhook triggers",
		},
		[]string{"provider", "event", "status"},
	)
)

// MetricsCollector 指标收集器
type MetricsCollector struct {
	db       *gorm.DB
	mu       sync.RWMutex
	stopChan chan struct{}
}

// NewMetricsCollector 创建指标收集器
func NewMetricsCollector(db *gorm.DB) *MetricsCollector {
	return &MetricsCollector{
		db:       db,
		stopChan: make(chan struct{}),
	}
}

// Start 启动指标收集
func (c *MetricsCollector) Start(ctx context.Context) {
	log := logger.L().WithField("component", "pipeline_metrics")
	log.Info("启动流水线指标收集器")

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	// 立即收集一次
	c.collect(ctx)

	for {
		select {
		case <-ticker.C:
			c.collect(ctx)
		case <-c.stopChan:
			log.Info("停止流水线指标收集器")
			return
		case <-ctx.Done():
			log.Info("上下文取消，停止指标收集器")
			return
		}
	}
}

// Stop 停止指标收集
func (c *MetricsCollector) Stop() {
	close(c.stopChan)
}

// collect 收集指标
func (c *MetricsCollector) collect(ctx context.Context) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.collectQueueMetrics(ctx)
	c.collectRunningMetrics(ctx)
	c.collectSuccessRateMetrics(ctx)
	c.collectArtifactMetrics(ctx)
}

// collectQueueMetrics 收集队列指标
func (c *MetricsCollector) collectQueueMetrics(ctx context.Context) {
	var count int64
	c.db.WithContext(ctx).Model(&models.PipelineRun{}).
		Where("status = ?", "pending").
		Count(&count)

	buildQueueLength.Set(float64(count))
}

// collectRunningMetrics 收集运行中任务指标
func (c *MetricsCollector) collectRunningMetrics(ctx context.Context) {
	type Result struct {
		PipelineID uint
		Count      int64
	}

	var results []Result
	c.db.WithContext(ctx).Model(&models.PipelineRun{}).
		Select("pipeline_id, COUNT(*) as count").
		Where("status = ?", "running").
		Group("pipeline_id").
		Scan(&results)

	// 重置所有
	buildJobsRunning.Reset()

	for _, r := range results {
		buildJobsRunning.WithLabelValues(uintToStr(r.PipelineID)).Set(float64(r.Count))
	}
}

// collectSuccessRateMetrics 收集成功率指标
func (c *MetricsCollector) collectSuccessRateMetrics(ctx context.Context) {
	// 获取最近 24 小时的数据
	since := time.Now().Add(-24 * time.Hour)

	type Result struct {
		PipelineID uint
		Total      int64
		Success    int64
	}

	var results []Result
	c.db.WithContext(ctx).Model(&models.PipelineRun{}).
		Select("pipeline_id, COUNT(*) as total, SUM(CASE WHEN status = 'success' THEN 1 ELSE 0 END) as success").
		Where("created_at > ? AND status IN ('success', 'failed')", since).
		Group("pipeline_id").
		Scan(&results)

	for _, r := range results {
		if r.Total > 0 {
			rate := float64(r.Success) / float64(r.Total)
			buildSuccessRate.WithLabelValues(uintToStr(r.PipelineID)).Set(rate)
		}
	}
}

// collectArtifactMetrics 收集制品指标
func (c *MetricsCollector) collectArtifactMetrics(ctx context.Context) {
	type Result struct {
		PipelineID uint
		TotalSize  int64
	}

	var results []Result
	c.db.WithContext(ctx).Model(&models.Artifact{}).
		Select("pipeline_id, SUM(size) as total_size").
		Group("pipeline_id").
		Scan(&results)

	for _, r := range results {
		artifactsSize.WithLabelValues(uintToStr(r.PipelineID)).Set(float64(r.TotalSize))
	}
}

// RecordBuildJob 记录构建任务
func RecordBuildJob(pipelineID uint, status string, duration float64) {
	buildJobsTotal.WithLabelValues(uintToStr(pipelineID), status).Inc()
	if duration > 0 {
		buildJobDuration.WithLabelValues(uintToStr(pipelineID)).Observe(duration)
	}
}

// RecordArtifact 记录制品
func RecordArtifact(pipelineID uint, artifactType string) {
	artifactsTotal.WithLabelValues(uintToStr(pipelineID), artifactType).Inc()
}

// RecordWebhookTrigger 记录 Webhook 触发
func RecordWebhookTrigger(provider, event, status string) {
	webhookTriggersTotal.WithLabelValues(provider, event, status).Inc()
}

// uintToStr 转换 uint 为字符串
func uintToStr(n uint) string {
	return fmt.Sprintf("%d", n)
}
