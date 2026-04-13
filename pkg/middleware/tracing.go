package middleware

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	"go.opentelemetry.io/otel/trace"
)

const (
	// CorrelationIDHeader 请求关联 ID 头
	CorrelationIDHeader = "X-Correlation-ID"
	// RequestIDHeader 请求 ID 头
	RequestIDHeader = "X-Request-ID"
	// TraceIDHeader 追踪 ID 头
	TraceIDHeader = "X-Trace-ID"
	// SpanIDHeader Span ID 头
	SpanIDHeader = "X-Span-ID"
)

// TracingConfig 追踪中间件配置
type TracingConfig struct {
	ServiceName    string
	ServiceVersion string
	SkipPaths      []string
}

// Tracing 分布式追踪中间件
func Tracing(config *TracingConfig) gin.HandlerFunc {
	if config == nil {
		config = &TracingConfig{
			ServiceName: "devops-platform",
		}
	}

	tracer := otel.Tracer(config.ServiceName)
	propagator := otel.GetTextMapPropagator()

	skipPaths := make(map[string]bool)
	for _, path := range config.SkipPaths {
		skipPaths[path] = true
	}

	return func(c *gin.Context) {
		// 跳过不需要追踪的路径
		if skipPaths[c.Request.URL.Path] {
			c.Next()
			return
		}

		// 从请求头提取上下文
		ctx := propagator.Extract(c.Request.Context(), propagation.HeaderCarrier(c.Request.Header))

		// 生成或获取 Correlation ID
		correlationID := c.GetHeader(CorrelationIDHeader)
		if correlationID == "" {
			correlationID = uuid.New().String()
		}

		// 生成 Request ID
		requestID := uuid.New().String()

		// 创建 Span
		spanName := fmt.Sprintf("%s %s", c.Request.Method, c.FullPath())
		ctx, span := tracer.Start(ctx, spanName,
			trace.WithSpanKind(trace.SpanKindServer),
			trace.WithAttributes(
				semconv.HTTPMethod(c.Request.Method),
				semconv.HTTPRoute(c.FullPath()),
				semconv.HTTPTarget(c.Request.URL.Path),
				semconv.HTTPScheme(c.Request.URL.Scheme),
				semconv.HTTPUserAgent(c.Request.UserAgent()),
				semconv.HTTPClientIP(c.ClientIP()),
				attribute.String("correlation_id", correlationID),
				attribute.String("request_id", requestID),
			),
		)
		defer span.End()

		// 获取 Trace ID 和 Span ID
		spanCtx := span.SpanContext()
		traceID := ""
		spanID := ""
		if spanCtx.HasTraceID() {
			traceID = spanCtx.TraceID().String()
		}
		if spanCtx.HasSpanID() {
			spanID = spanCtx.SpanID().String()
		}

		// 设置响应头
		c.Header(CorrelationIDHeader, correlationID)
		c.Header(RequestIDHeader, requestID)
		if traceID != "" {
			c.Header(TraceIDHeader, traceID)
		}
		if spanID != "" {
			c.Header(SpanIDHeader, spanID)
		}

		// 存储到上下文
		c.Set("correlation_id", correlationID)
		c.Set("request_id", requestID)
		c.Set("trace_id", traceID)
		c.Set("span_id", spanID)

		// 更新请求上下文
		c.Request = c.Request.WithContext(ctx)

		// 记录开始时间
		start := time.Now()

		// 处理请求
		c.Next()

		// 记录响应信息
		duration := time.Since(start)
		statusCode := c.Writer.Status()

		span.SetAttributes(
			semconv.HTTPStatusCode(statusCode),
			attribute.Int64("http.response_content_length", int64(c.Writer.Size())),
			attribute.Int64("duration_ms", duration.Milliseconds()),
		)

		// 记录错误
		if len(c.Errors) > 0 {
			for _, err := range c.Errors {
				span.RecordError(err.Err)
			}
		}

		// 根据状态码设置 Span 状态
		if statusCode >= 400 {
			span.SetAttributes(attribute.Bool("error", true))
		}
	}
}

// CorrelationID 获取关联 ID 中间件（简化版，不依赖 OpenTelemetry）
func CorrelationID() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 获取或生成 Correlation ID
		correlationID := c.GetHeader(CorrelationIDHeader)
		if correlationID == "" {
			correlationID = uuid.New().String()
		}

		// 生成 Request ID
		requestID := uuid.New().String()

		// 设置响应头
		c.Header(CorrelationIDHeader, correlationID)
		c.Header(RequestIDHeader, requestID)

		// 存储到上下文
		c.Set("correlation_id", correlationID)
		c.Set("request_id", requestID)

		c.Next()
	}
}

// GetCorrelationID 从上下文获取关联 ID
func GetCorrelationID(c *gin.Context) string {
	if id, exists := c.Get("correlation_id"); exists {
		return id.(string)
	}
	return ""
}

// GetRequestID 从上下文获取请求 ID
func GetRequestID(c *gin.Context) string {
	if id, exists := c.Get("request_id"); exists {
		return id.(string)
	}
	return ""
}

// GetTraceID 从上下文获取追踪 ID
func GetTraceID(c *gin.Context) string {
	if id, exists := c.Get("trace_id"); exists {
		return id.(string)
	}
	return ""
}

// GetSpanID 从上下文获取 Span ID
func GetSpanID(c *gin.Context) string {
	if id, exists := c.Get("span_id"); exists {
		return id.(string)
	}
	return ""
}

// InjectTraceContext 注入追踪上下文到 HTTP 请求头
func InjectTraceContext(c *gin.Context, headers map[string]string) {
	propagator := otel.GetTextMapPropagator()
	carrier := propagation.MapCarrier(headers)
	propagator.Inject(c.Request.Context(), carrier)

	// 同时注入 Correlation ID
	if correlationID := GetCorrelationID(c); correlationID != "" {
		headers[CorrelationIDHeader] = correlationID
	}
}
