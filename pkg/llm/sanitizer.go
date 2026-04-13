package llm

import (
	"regexp"
	"strings"
)

// Sanitizer 敏感信息过滤器
type Sanitizer struct {
	patterns []*sensitivePattern
}

// sensitivePattern 敏感信息模式
type sensitivePattern struct {
	name    string
	regex   *regexp.Regexp
	replace string
}

// NewSanitizer 创建敏感信息过滤器
func NewSanitizer() *Sanitizer {
	s := &Sanitizer{
		patterns: make([]*sensitivePattern, 0),
	}
	s.initPatterns()
	return s
}

// initPatterns 初始化敏感信息模式
func (s *Sanitizer) initPatterns() {
	patterns := []struct {
		name    string
		pattern string
		replace string
	}{
		// 密码相关
		{"password", `(?i)(password|passwd|pwd)\s*[=:]\s*['"]?[^\s'"]+['"]?`, "$1=***"},
		{"password_json", `(?i)"(password|passwd|pwd)"\s*:\s*"[^"]*"`, `"$1":"***"`},

		// API Key / Token
		{"api_key", `(?i)(api[_-]?key|apikey)\s*[=:]\s*['"]?[A-Za-z0-9_\-]{16,}['"]?`, "$1=***"},
		{"api_key_json", `(?i)"(api[_-]?key|apikey)"\s*:\s*"[^"]*"`, `"$1":"***"`},
		{"bearer_token", `(?i)Bearer\s+[A-Za-z0-9_\-\.]+`, "Bearer ***"},
		{"token", `(?i)(token|access_token|refresh_token|auth_token)\s*[=:]\s*['"]?[A-Za-z0-9_\-\.]{16,}['"]?`, "$1=***"},
		{"token_json", `(?i)"(token|access_token|refresh_token|auth_token)"\s*:\s*"[^"]*"`, `"$1":"***"`},

		// 密钥相关
		{"secret", `(?i)(secret|secret_key|app_secret|client_secret)\s*[=:]\s*['"]?[A-Za-z0-9_\-]{16,}['"]?`, "$1=***"},
		{"secret_json", `(?i)"(secret|secret_key|app_secret|client_secret)"\s*:\s*"[^"]*"`, `"$1":"***"`},
		{"private_key", `(?i)-----BEGIN\s+(RSA\s+)?PRIVATE\s+KEY-----[\s\S]*?-----END\s+(RSA\s+)?PRIVATE\s+KEY-----`, "[PRIVATE_KEY_REDACTED]"},

		// AWS 凭证
		{"aws_access_key", `(?i)(aws_access_key_id|aws_secret_access_key)\s*[=:]\s*['"]?[A-Za-z0-9/+=]{16,}['"]?`, "$1=***"},
		{"aws_key_pattern", `AKIA[0-9A-Z]{16}`, "***AWS_KEY***"},

		// 数据库连接字符串
		{"db_connection", `(?i)(mysql|postgres|mongodb|redis)://[^@]+@`, "$1://***:***@"},
		{"jdbc_password", `(?i)jdbc:[^?]+\?.*password=[^&]+`, "[JDBC_URL_REDACTED]"},

		// 邮箱（可选，根据需求）
		// {"email", `[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`, "[EMAIL_REDACTED]"},

		// IP地址（内网IP，可选）
		// {"internal_ip", `\b(10\.\d{1,3}\.\d{1,3}\.\d{1,3}|172\.(1[6-9]|2\d|3[01])\.\d{1,3}\.\d{1,3}|192\.168\.\d{1,3}\.\d{1,3})\b`, "[INTERNAL_IP]"},

		// 手机号（中国）
		{"phone", `\b1[3-9]\d{9}\b`, "[PHONE_REDACTED]"},

		// 身份证号
		{"id_card", `\b\d{17}[\dXx]\b`, "[ID_CARD_REDACTED]"},

		// 银行卡号
		{"bank_card", `\b\d{16,19}\b`, "[CARD_REDACTED]"},

		// Kubernetes secrets
		{"k8s_secret", `(?i)kind:\s*Secret[\s\S]*?data:[\s\S]*?[a-zA-Z0-9_-]+:\s*[A-Za-z0-9+/=]+`, "[K8S_SECRET_REDACTED]"},

		// 环境变量中的敏感值
		{"env_sensitive", `(?i)(DB_PASSWORD|MYSQL_PASSWORD|REDIS_PASSWORD|JWT_SECRET|ENCRYPTION_KEY)\s*=\s*[^\s]+`, "$1=***"},
	}

	for _, p := range patterns {
		regex, err := regexp.Compile(p.pattern)
		if err != nil {
			continue // 跳过无效的正则
		}
		s.patterns = append(s.patterns, &sensitivePattern{
			name:    p.name,
			regex:   regex,
			replace: p.replace,
		})
	}
}

// Sanitize 过滤敏感信息
func (s *Sanitizer) Sanitize(text string) string {
	result := text
	for _, p := range s.patterns {
		result = p.regex.ReplaceAllString(result, p.replace)
	}
	return result
}

// SanitizeMessages 过滤消息列表中的敏感信息
func (s *Sanitizer) SanitizeMessages(messages []ChatMessage) []ChatMessage {
	sanitized := make([]ChatMessage, len(messages))
	for i, msg := range messages {
		sanitized[i] = ChatMessage{
			Role:       msg.Role,
			Content:    s.Sanitize(msg.Content),
			Name:       msg.Name,
			ToolCallID: msg.ToolCallID,
			ToolCalls:  msg.ToolCalls,
		}
	}
	return sanitized
}

// ContainsSensitiveInfo 检查是否包含敏感信息
func (s *Sanitizer) ContainsSensitiveInfo(text string) bool {
	for _, p := range s.patterns {
		if p.regex.MatchString(text) {
			return true
		}
	}
	return false
}

// GetSensitivePatterns 获取匹配到的敏感信息类型
func (s *Sanitizer) GetSensitivePatterns(text string) []string {
	var matched []string
	for _, p := range s.patterns {
		if p.regex.MatchString(text) {
			matched = append(matched, p.name)
		}
	}
	return matched
}

// MaskString 掩码字符串（保留首尾字符）
func MaskString(s string, visibleChars int) string {
	if len(s) <= visibleChars*2 {
		return strings.Repeat("*", len(s))
	}
	return s[:visibleChars] + strings.Repeat("*", len(s)-visibleChars*2) + s[len(s)-visibleChars:]
}

// DefaultSanitizer 默认敏感信息过滤器
var DefaultSanitizer = NewSanitizer()

// Sanitize 使用默认过滤器过滤敏感信息
func Sanitize(text string) string {
	return DefaultSanitizer.Sanitize(text)
}

// SanitizeMessages 使用默认过滤器过滤消息
func SanitizeMessages(messages []ChatMessage) []ChatMessage {
	return DefaultSanitizer.SanitizeMessages(messages)
}
