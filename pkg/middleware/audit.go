package middleware

import (
	"bytes"
	"encoding/json"
	"io"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"devops/internal/service/audit"
)

// AuditConfig 审计配置
type AuditConfig struct {
	// 需要审计的路径前缀
	IncludePaths []string
	// 排除的路径
	ExcludePaths []string
	// 需要审计的方法
	Methods []string
	// 是否记录请求体
	LogRequestBody bool
	// 是否记录响应体
	LogResponseBody bool
	// 最大请求体大小
	MaxBodySize int
}

// DefaultAuditConfig 默认审计配置
func DefaultAuditConfig() *AuditConfig {
	return &AuditConfig{
		IncludePaths:    []string{"/app/api/v1/"},
		ExcludePaths:    []string{"/app/api/v1/health", "/app/api/v1/metrics"},
		Methods:         []string{"POST", "PUT", "DELETE", "PATCH"},
		LogRequestBody:  true,
		LogResponseBody: false,
		MaxBodySize:     10240, // 10KB
	}
}

// responseWriter 自定义响应写入器
type responseWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w *responseWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

// AuditMiddleware 审计中间件
func AuditMiddleware(db *gorm.DB, config *AuditConfig) gin.HandlerFunc {
	if config == nil {
		config = DefaultAuditConfig()
	}

	auditService := audit.NewAuditService(db)

	return func(c *gin.Context) {
		// 检查是否需要审计
		if !shouldAudit(c, config) {
			c.Next()
			return
		}

		start := time.Now()

		// 读取请求体
		var requestBody interface{}
		if config.LogRequestBody && c.Request.Body != nil {
			bodyBytes, _ := io.ReadAll(c.Request.Body)
			c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

			if len(bodyBytes) > 0 && len(bodyBytes) <= config.MaxBodySize {
				// 尝试解析为 JSON
				var jsonBody map[string]interface{}
				if err := json.Unmarshal(bodyBytes, &jsonBody); err == nil {
					// 脱敏敏感字段
					sanitizeBody(jsonBody)
					requestBody = jsonBody
				}
			}
		}

		// 包装响应写入器
		var responseBody *bytes.Buffer
		if config.LogResponseBody {
			responseBody = &bytes.Buffer{}
			c.Writer = &responseWriter{ResponseWriter: c.Writer, body: responseBody}
		}

		// 处理请求
		c.Next()

		// 计算耗时
		duration := time.Since(start).Milliseconds()

		// 获取上下文信息
		userID, _ := GetUserID(c)
		username := ""
		if u, exists := c.Get("username"); exists {
			username = u.(string)
		}

		// 确定操作类型
		action := getActionFromMethod(c.Request.Method)

		// 确定资源类型
		resourceType, _ := parseResource(c.FullPath(), c.Params)

		// 确定状态
		status := audit.StatusSuccess
		var errorMsg string
		if c.Writer.Status() >= 400 {
			status = audit.StatusFailed
			if len(c.Errors) > 0 {
				errorMsg = c.Errors.Last().Error()
			}
		}

		// 构建审计条目
		entry := &audit.AuditEntry{
			UserID:       &userID,
			Username:     username,
			Action:       action,
			ResourceType: resourceType,
			NewValue:     requestBody,
			IPAddress:    c.ClientIP(),
			UserAgent:    c.Request.UserAgent(),
			RequestID:    GetRequestID(c),
			TraceID:      GetTraceID(c),
			Status:       status,
			ErrorMessage: errorMsg,
			Duration:     duration,
		}

		// 异步记录审计日志
		auditService.LogAsync(entry)
	}
}

// shouldAudit 判断是否需要审计
func shouldAudit(c *gin.Context, config *AuditConfig) bool {
	path := c.Request.URL.Path
	method := c.Request.Method

	// 检查方法
	methodMatch := false
	for _, m := range config.Methods {
		if m == method {
			methodMatch = true
			break
		}
	}
	if !methodMatch {
		return false
	}

	// 检查排除路径
	for _, p := range config.ExcludePaths {
		if strings.HasPrefix(path, p) {
			return false
		}
	}

	// 检查包含路径
	for _, p := range config.IncludePaths {
		if strings.HasPrefix(path, p) {
			return true
		}
	}

	return false
}

// getActionFromMethod 根据 HTTP 方法获取操作类型
func getActionFromMethod(method string) audit.AuditAction {
	switch method {
	case "POST":
		return audit.ActionCreate
	case "PUT", "PATCH":
		return audit.ActionUpdate
	case "DELETE":
		return audit.ActionDelete
	default:
		return audit.ActionRead
	}
}

// parseResource 解析资源类型和 ID
func parseResource(path string, params gin.Params) (string, uint) {
	// 从路径中提取资源类型
	parts := strings.Split(strings.Trim(path, "/"), "/")
	resourceType := ""
	var resourceID uint

	// 跳过 app/api/v1 前缀
	if len(parts) > 3 {
		resourceType = parts[3]
	}

	// 尝试从参数中获取 ID
	for _, p := range params {
		if strings.HasSuffix(p.Key, "_id") || p.Key == "id" {
			var id uint
			if _, err := parseUint(p.Value, &id); err == nil {
				resourceID = id
				break
			}
		}
	}

	return resourceType, resourceID
}

// parseUint 解析 uint
func parseUint(s string, v *uint) (string, error) {
	var n uint64
	for _, c := range s {
		if c < '0' || c > '9' {
			return s, nil
		}
		n = n*10 + uint64(c-'0')
	}
	*v = uint(n)
	return s, nil
}

// sanitizeBody 脱敏敏感字段
func sanitizeBody(body map[string]interface{}) {
	sensitiveFields := []string{
		"password", "secret", "token", "key", "credential",
		"api_key", "access_token", "refresh_token", "private_key",
	}

	for key := range body {
		lowerKey := strings.ToLower(key)
		for _, sensitive := range sensitiveFields {
			if strings.Contains(lowerKey, sensitive) {
				body[key] = "***REDACTED***"
				break
			}
		}
	}
}
