package errors

import (
	"fmt"
	"strings"
)

// ErrorHelper 错误辅助工具
type ErrorHelper struct{}

// NewHelper 创建错误辅助工具
func NewHelper() *ErrorHelper {
	return &ErrorHelper{}
}

// FormatK8sError 格式化 K8s 错误消息
func (h *ErrorHelper) FormatK8sError(err error, operation string) *AppError {
	if err == nil {
		return nil
	}

	errMsg := err.Error()
	errLower := strings.ToLower(errMsg)

	// 资源不存在
	if strings.Contains(errLower, "not found") {
		return New(ErrCodeNotFound, fmt.Sprintf("%s失败：资源不存在", operation))
	}

	// 资源已存在
	if strings.Contains(errLower, "already exists") {
		return New(ErrCodeDuplicate, fmt.Sprintf("%s失败：资源已存在", operation))
	}

	// 权限不足
	if strings.Contains(errLower, "forbidden") {
		return New(ErrCodeForbidden, fmt.Sprintf("%s失败：权限不足，请检查集群权限配置", operation))
	}

	// 连接失败
	if strings.Contains(errLower, "connection refused") || strings.Contains(errLower, "dial tcp") {
		return New(ErrCodeK8sConnect, fmt.Sprintf("%s失败：无法连接到集群，请检查集群配置", operation))
	}

	// 超时
	if strings.Contains(errLower, "timeout") || strings.Contains(errLower, "deadline exceeded") {
		return New(ErrCodeRequestTimeout, fmt.Sprintf("%s失败：请求超时，请稍后重试", operation))
	}

	// 配置错误
	if strings.Contains(errLower, "invalid") || strings.Contains(errLower, "malformed") {
		return New(ErrCodeK8sConfig, fmt.Sprintf("%s失败：配置格式错误，请检查 YAML 配置", operation))
	}

	// 资源不足
	if strings.Contains(errLower, "insufficient") || strings.Contains(errLower, "exceeded quota") {
		return New(ErrCodeK8sDeploy, fmt.Sprintf("%s失败：集群资源不足", operation))
	}

	// Pod 相关错误
	if strings.Contains(errLower, "pod") {
		if strings.Contains(errLower, "crashloopbackoff") {
			return New(ErrCodeK8sPod, fmt.Sprintf("%s失败：Pod 启动失败，请检查容器配置和日志", operation))
		}
		if strings.Contains(errLower, "imagepullbackoff") || strings.Contains(errLower, "errimagepull") {
			return New(ErrCodeK8sPod, fmt.Sprintf("%s失败：镜像拉取失败，请检查镜像地址和权限", operation))
		}
	}

	// 默认错误
	return Wrap(err, ErrCodeK8sDeploy, fmt.Sprintf("%s失败", operation))
}

// FormatDBError 格式化数据库错误消息
func (h *ErrorHelper) FormatDBError(err error, operation string) *AppError {
	if err == nil {
		return nil
	}

	errMsg := err.Error()
	errLower := strings.ToLower(errMsg)

	// 记录不存在
	if strings.Contains(errLower, "record not found") {
		return New(ErrCodeNotFound, fmt.Sprintf("%s失败：记录不存在", operation))
	}

	// 重复键
	if strings.Contains(errLower, "duplicate") || strings.Contains(errLower, "unique constraint") {
		if strings.Contains(errLower, "username") {
			return New(ErrCodeDuplicate, "用户名已存在")
		}
		if strings.Contains(errLower, "email") {
			return New(ErrCodeDuplicate, "邮箱已被使用")
		}
		if strings.Contains(errLower, "name") {
			return New(ErrCodeDuplicate, "名称已存在")
		}
		return New(ErrCodeDuplicate, fmt.Sprintf("%s失败：数据已存在", operation))
	}

	// 外键约束
	if strings.Contains(errLower, "foreign key") {
		return New(ErrCodeConflict, fmt.Sprintf("%s失败：存在关联数据，请先删除关联项", operation))
	}

	// 连接错误
	if strings.Contains(errLower, "connection") {
		return New(ErrCodeDBConnect, "数据库连接失败，请稍后重试")
	}

	// 事务错误
	if strings.Contains(errLower, "transaction") {
		return New(ErrCodeDBTransaction, fmt.Sprintf("%s失败：事务处理错误", operation))
	}

	// 默认错误
	return Wrap(err, ErrCodeDBQuery, fmt.Sprintf("%s失败，请稍后重试", operation))
}

// FormatJenkinsError 格式化 Jenkins 错误消息
func (h *ErrorHelper) FormatJenkinsError(err error, operation string) *AppError {
	if err == nil {
		return nil
	}

	errMsg := err.Error()
	errLower := strings.ToLower(errMsg)

	// 连接失败
	if strings.Contains(errLower, "connection refused") || strings.Contains(errLower, "dial tcp") {
		return New(ErrCodeJenkinsConnect, "Jenkins 服务连接失败，请检查配置")
	}

	// 认证失败
	if strings.Contains(errLower, "401") || strings.Contains(errLower, "unauthorized") {
		return New(ErrCodeJenkinsAPI, "Jenkins 认证失败，请检查用户名和密码")
	}

	// 权限不足
	if strings.Contains(errLower, "403") || strings.Contains(errLower, "forbidden") {
		return New(ErrCodeJenkinsAPI, "Jenkins 权限不足，请检查用户权限")
	}

	// 任务不存在
	if strings.Contains(errLower, "404") || strings.Contains(errLower, "not found") {
		return New(ErrCodeNotFound, fmt.Sprintf("%s失败：Jenkins 任务不存在", operation))
	}

	// 构建失败
	if strings.Contains(errLower, "build") && strings.Contains(errLower, "failed") {
		return New(ErrCodeJenkinsBuild, "构建失败，请查看 Jenkins 日志")
	}

	// 默认错误
	return Wrap(err, ErrCodeJenkinsAPI, fmt.Sprintf("%s失败", operation))
}

// FormatValidationError 格式化参数校验错误
func (h *ErrorHelper) FormatValidationError(field, tag string) string {
	messages := map[string]string{
		"required": "不能为空",
		"email":    "格式不正确",
		"min":      "长度不足",
		"max":      "长度超出限制",
		"oneof":    "值不在允许范围内",
		"url":      "URL 格式不正确",
		"numeric":  "必须是数字",
		"alpha":    "只能包含字母",
		"alphanum": "只能包含字母和数字",
	}

	if msg, ok := messages[tag]; ok {
		return fmt.Sprintf("%s%s", field, msg)
	}
	return fmt.Sprintf("%s验证失败", field)
}

// SimplifyError 简化错误消息（移除技术细节）
func (h *ErrorHelper) SimplifyError(err error) string {
	if err == nil {
		return ""
	}

	errMsg := err.Error()

	// 移除常见的技术前缀
	prefixes := []string{
		"rpc error: code = ",
		"Error from server ",
		"error: ",
		"failed to ",
	}

	for _, prefix := range prefixes {
		if strings.HasPrefix(errMsg, prefix) {
			errMsg = strings.TrimPrefix(errMsg, prefix)
			break
		}
	}

	// 截断过长的错误消息
	if len(errMsg) > 200 {
		errMsg = errMsg[:200] + "..."
	}

	return errMsg
}

// WrapWithContext 包装错误并添加上下文信息
func (h *ErrorHelper) WrapWithContext(err error, code int, operation, resource string) *AppError {
	if err == nil {
		return nil
	}

	message := fmt.Sprintf("%s %s 失败", operation, resource)
	return Wrap(err, code, message)
}

// IsRetryable 判断错误是否可重试
func (h *ErrorHelper) IsRetryable(err error) bool {
	if err == nil {
		return false
	}

	errLower := strings.ToLower(err.Error())

	retryableKeywords := []string{
		"timeout",
		"deadline exceeded",
		"connection refused",
		"connection reset",
		"temporary failure",
		"try again",
		"too many requests",
	}

	for _, keyword := range retryableKeywords {
		if strings.Contains(errLower, keyword) {
			return true
		}
	}

	return false
}

// GetUserFriendlyMessage 获取用户友好的错误消息
func (h *ErrorHelper) GetUserFriendlyMessage(err error) string {
	if err == nil {
		return ""
	}

	// 如果是 AppError，返回其消息
	if appErr, ok := AsAppError(err); ok {
		return appErr.Message
	}

	// 否则简化错误消息
	return h.SimplifyError(err)
}

// 全局错误辅助工具实例
var Helper = NewHelper()

// 便捷函数
func FormatK8sError(err error, operation string) *AppError {
	return Helper.FormatK8sError(err, operation)
}

func FormatDBError(err error, operation string) *AppError {
	return Helper.FormatDBError(err, operation)
}

func FormatJenkinsError(err error, operation string) *AppError {
	return Helper.FormatJenkinsError(err, operation)
}

func SimplifyError(err error) string {
	return Helper.SimplifyError(err)
}

func IsRetryable(err error) bool {
	return Helper.IsRetryable(err)
}
