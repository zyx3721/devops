package metrics

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	// HTTP 请求指标
	httpRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "devops_http_requests_total",
			Help: "HTTP 请求总数",
		},
		[]string{"method", "path", "status"},
	)

	httpRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "devops_http_request_duration_seconds",
			Help:    "HTTP 请求延迟",
			Buckets: []float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
		},
		[]string{"method", "path"},
	)

	httpRequestsInFlight = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "devops_http_requests_in_flight",
			Help: "当前正在处理的 HTTP 请求数",
		},
	)

	// 业务指标
	pipelineExecutionsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "devops_pipeline_executions_total",
			Help: "流水线执行总数",
		},
		[]string{"tenant_id", "status"},
	)

	pipelineExecutionDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "devops_pipeline_execution_duration_seconds",
			Help:    "流水线执行时长",
			Buckets: []float64{10, 30, 60, 120, 300, 600, 1200, 1800, 3600},
		},
		[]string{"tenant_id"},
	)

	deploymentsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "devops_deployments_total",
			Help: "部署总数",
		},
		[]string{"tenant_id", "environment", "status"},
	)

	activeTenantsGauge = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "devops_active_tenants",
			Help: "活跃租户数",
		},
	)

	activePipelinesGauge = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "devops_active_pipelines",
			Help: "活跃流水线数",
		},
		[]string{"tenant_id"},
	)

	// 资源使用指标
	buildMinutesUsed = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "devops_build_minutes_used_total",
			Help: "已使用的构建分钟数",
		},
		[]string{"tenant_id"},
	)

	storageUsedBytes = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "devops_storage_used_bytes",
			Help: "已使用的存储空间（字节）",
		},
		[]string{"tenant_id"},
	)

	// 错误指标
	errorsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "devops_errors_total",
			Help: "错误总数",
		},
		[]string{"type", "code"},
	)

	// 外部服务指标
	externalServiceRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "devops_external_service_requests_total",
			Help: "外部服务请求总数",
		},
		[]string{"service", "status"},
	)

	externalServiceRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "devops_external_service_request_duration_seconds",
			Help:    "外部服务请求延迟",
			Buckets: []float64{.01, .05, .1, .25, .5, 1, 2.5, 5, 10},
		},
		[]string{"service"},
	)
)

// MetricsMiddleware Prometheus 指标中间件
func MetricsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.FullPath()
		if path == "" {
			path = "unknown"
		}

		httpRequestsInFlight.Inc()
		defer httpRequestsInFlight.Dec()

		c.Next()

		status := strconv.Itoa(c.Writer.Status())
		duration := time.Since(start).Seconds()

		httpRequestsTotal.WithLabelValues(c.Request.Method, path, status).Inc()
		httpRequestDuration.WithLabelValues(c.Request.Method, path).Observe(duration)
	}
}

// MetricsHandler 返回 Prometheus 指标处理器
func MetricsHandler() gin.HandlerFunc {
	h := promhttp.Handler()
	return func(c *gin.Context) {
		h.ServeHTTP(c.Writer, c.Request)
	}
}

// RecordPipelineExecution 记录流水线执行
func RecordPipelineExecution(tenantID uint, status string, duration time.Duration) {
	tid := strconv.FormatUint(uint64(tenantID), 10)
	pipelineExecutionsTotal.WithLabelValues(tid, status).Inc()
	pipelineExecutionDuration.WithLabelValues(tid).Observe(duration.Seconds())
}

// RecordDeployment 记录部署
func RecordDeployment(tenantID uint, environment, status string) {
	tid := strconv.FormatUint(uint64(tenantID), 10)
	deploymentsTotal.WithLabelValues(tid, environment, status).Inc()
}

// SetActiveTenants 设置活跃租户数
func SetActiveTenants(count int) {
	activeTenantsGauge.Set(float64(count))
}

// SetActivePipelines 设置活跃流水线数
func SetActivePipelines(tenantID uint, count int) {
	tid := strconv.FormatUint(uint64(tenantID), 10)
	activePipelinesGauge.WithLabelValues(tid).Set(float64(count))
}

// RecordBuildMinutes 记录构建分钟数
func RecordBuildMinutes(tenantID uint, minutes float64) {
	tid := strconv.FormatUint(uint64(tenantID), 10)
	buildMinutesUsed.WithLabelValues(tid).Add(minutes)
}

// SetStorageUsed 设置存储使用量
func SetStorageUsed(tenantID uint, bytes float64) {
	tid := strconv.FormatUint(uint64(tenantID), 10)
	storageUsedBytes.WithLabelValues(tid).Set(bytes)
}

// RecordError 记录错误
func RecordError(errorType, code string) {
	errorsTotal.WithLabelValues(errorType, code).Inc()
}

// RecordExternalServiceRequest 记录外部服务请求
func RecordExternalServiceRequest(service, status string, duration time.Duration) {
	externalServiceRequestsTotal.WithLabelValues(service, status).Inc()
	externalServiceRequestDuration.WithLabelValues(service).Observe(duration.Seconds())
}

// Collector 自定义收集器接口
type Collector interface {
	Collect() error
}

// BusinessMetricsCollector 业务指标收集器
type BusinessMetricsCollector struct {
	collectors []Collector
}

// NewBusinessMetricsCollector 创建业务指标收集器
func NewBusinessMetricsCollector() *BusinessMetricsCollector {
	return &BusinessMetricsCollector{
		collectors: make([]Collector, 0),
	}
}

// Register 注册收集器
func (c *BusinessMetricsCollector) Register(collector Collector) {
	c.collectors = append(c.collectors, collector)
}

// CollectAll 收集所有指标
func (c *BusinessMetricsCollector) CollectAll() {
	for _, collector := range c.collectors {
		collector.Collect()
	}
}
