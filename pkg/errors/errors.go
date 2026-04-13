package errors

import (
	"errors"
	"fmt"
	"net/http"
)

// AppError 应用错误（支持错误链）
type AppError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Details any    `json:"details,omitempty"`
	cause   error  // 原始错误，支持错误链
}

// Error 实现 error 接口
func (e *AppError) Error() string {
	if e.cause != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.cause)
	}
	return e.Message
}

// Unwrap 支持错误链，实现 errors.Unwrap
func (e *AppError) Unwrap() error {
	return e.cause
}

// Is 支持 errors.Is 比较
func (e *AppError) Is(target error) bool {
	t, ok := target.(*AppError)
	if !ok {
		return false
	}
	return e.Code == t.Code
}

// New 创建新的应用错误
func New(code int, message string) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
	}
}

// NewWithCause 创建带原因的应用错误
func NewWithCause(code int, message string, cause error) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
		cause:   cause,
	}
}

// WithDetails 添加错误详情
func (e *AppError) WithDetails(details any) *AppError {
	e.Details = details
	return e
}

// WithCause 添加原始错误
func (e *AppError) WithCause(cause error) *AppError {
	e.cause = cause
	return e
}

// Wrap 包装错误并添加上下文
func Wrap(err error, code int, message string) *AppError {
	if err == nil {
		return nil
	}
	return &AppError{
		Code:    code,
		Message: message,
		cause:   err,
	}
}

// WrapWithDetails 包装错误并添加详情
func WrapWithDetails(err error, code int, message string, details any) *AppError {
	if err == nil {
		return nil
	}
	return &AppError{
		Code:    code,
		Message: message,
		Details: details,
		cause:   err,
	}
}

// GetCode 从错误中获取错误码
func GetCode(err error) int {
	if err == nil {
		return Success
	}
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.Code
	}
	return ErrCodeInternalError
}

// GetMessage 从错误中获取消息
func GetMessage(err error) string {
	if err == nil {
		return ""
	}
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.Message
	}
	return "内部错误"
}

// IsAppError 判断是否为 AppError
func IsAppError(err error) bool {
	var appErr *AppError
	return errors.As(err, &appErr)
}

// AsAppError 尝试将错误转换为 AppError
func AsAppError(err error) (*AppError, bool) {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr, true
	}
	return nil, false
}

// 错误码定义
const (
	// 成功
	Success = 0

	// 通用错误 (1000-1999)
	ErrCodeInternalError    = 1000 // 内部错误
	ErrCodeInvalidParams    = 1001 // 参数错误
	ErrCodeMethodNotAllowed = 1002 // 方法不允许
	ErrCodeRequestTimeout   = 1003 // 请求超时

	// 认证授权错误 (2000-2999)
	ErrCodeUnauthorized = 2000 // 未授权
	ErrCodeForbidden    = 2001 // 禁止访问
	ErrCodeTokenExpired = 2002 // Token过期
	ErrCodeTokenInvalid = 2003 // Token无效

	// 业务错误 (3000-3999)
	ErrCodeNotFound      = 3000 // 资源不存在
	ErrCodeConflict      = 3001 // 资源冲突
	ErrCodeDuplicate     = 3002 // 重复创建
	ErrCodeBusiness      = 3003 // 业务错误
	ErrCodeStatusInvalid = 3004 // 状态无效

	// 飞书相关错误 (4000-4999)
	ErrCodeFeishuToken   = 4000 // 飞书Token错误
	ErrCodeFeishuAPI     = 4001 // 飞书API错误
	ErrCodeFeishuWebhook = 4002 // 飞书Webhook错误

	// Jenkins相关错误 (5000-5999)
	ErrCodeJenkinsConnect = 5000 // Jenkins连接错误
	ErrCodeJenkinsAPI     = 5001 // Jenkins API错误
	ErrCodeJenkinsBuild   = 5002 // Jenkins构建错误

	// K8s相关错误 (6000-6999)
	ErrCodeK8sConnect = 6000 // K8s连接错误
	ErrCodeK8sConfig  = 6001 // K8s配置错误
	ErrCodeK8sDeploy  = 6002 // K8s部署错误
	ErrCodeK8sPod     = 6003 // K8s Pod错误

	// Archery相关错误 (7000-7999)
	ErrCodeArcheryConnect  = 7000 // Archery连接错误
	ErrCodeArcheryAPI      = 7001 // Archery API错误
	ErrCodeArcheryWorkflow = 7002 // Archery工作流错误

	// Redis相关错误 (8000-8999)
	ErrCodeRedisConnect = 8000 // Redis连接错误
	ErrCodeRedisLock    = 8001 // Redis锁错误

	// 数据库相关错误 (9000-9999)
	ErrCodeDBConnect     = 9000 // 数据库连接错误
	ErrCodeDBQuery       = 9001 // 数据库查询错误
	ErrCodeDBTransaction = 9002 // 数据库事务错误
)

// 错误码到HTTP状态码的映射
var errorToHTTPStatus = map[int]int{
	ErrCodeInternalError:    http.StatusInternalServerError,
	ErrCodeInvalidParams:    http.StatusBadRequest,
	ErrCodeMethodNotAllowed: http.StatusMethodNotAllowed,
	ErrCodeRequestTimeout:   http.StatusRequestTimeout,
	ErrCodeUnauthorized:     http.StatusUnauthorized,
	ErrCodeForbidden:        http.StatusForbidden,
	ErrCodeTokenExpired:     http.StatusUnauthorized,
	ErrCodeTokenInvalid:     http.StatusUnauthorized,
	ErrCodeNotFound:         http.StatusNotFound,
	ErrCodeConflict:         http.StatusConflict,
	ErrCodeDuplicate:        http.StatusConflict,
	ErrCodeBusiness:         http.StatusBadRequest,
	ErrCodeStatusInvalid:    http.StatusBadRequest,
	ErrCodeFeishuToken:      http.StatusInternalServerError,
	ErrCodeFeishuAPI:        http.StatusInternalServerError,
	ErrCodeFeishuWebhook:    http.StatusInternalServerError,
	ErrCodeJenkinsConnect:   http.StatusServiceUnavailable,
	ErrCodeJenkinsAPI:       http.StatusInternalServerError,
	ErrCodeJenkinsBuild:     http.StatusInternalServerError,
	ErrCodeK8sConnect:       http.StatusServiceUnavailable,
	ErrCodeK8sConfig:        http.StatusInternalServerError,
	ErrCodeK8sDeploy:        http.StatusInternalServerError,
	ErrCodeK8sPod:           http.StatusInternalServerError,
	ErrCodeArcheryConnect:   http.StatusServiceUnavailable,
	ErrCodeArcheryAPI:       http.StatusInternalServerError,
	ErrCodeArcheryWorkflow:  http.StatusInternalServerError,
	ErrCodeRedisConnect:     http.StatusServiceUnavailable,
	ErrCodeRedisLock:        http.StatusInternalServerError,
	ErrCodeDBConnect:        http.StatusServiceUnavailable,
	ErrCodeDBQuery:          http.StatusInternalServerError,
	ErrCodeDBTransaction:    http.StatusInternalServerError,
}

// GetHTTPStatus 获取HTTP状态码
func GetHTTPStatus(code int) int {
	if status, ok := errorToHTTPStatus[code]; ok {
		return status
	}
	return http.StatusInternalServerError
}

// 预定义错误
var (
	ErrInternal        = New(ErrCodeInternalError, "内部错误")
	ErrInvalidParams   = New(ErrCodeInvalidParams, "参数错误")
	ErrUnauthorized    = New(ErrCodeUnauthorized, "未授权")
	ErrForbidden       = New(ErrCodeForbidden, "禁止访问")
	ErrNotFound        = New(ErrCodeNotFound, "资源不存在")
	ErrUserNotFound    = New(ErrCodeNotFound, "用户不存在")
	ErrPasswordInvalid = New(ErrCodeUnauthorized, "密码错误")
	ErrUserExists      = New(ErrCodeDuplicate, "用户已存在")
	ErrDuplicate       = New(ErrCodeDuplicate, "资源已存在")

	ErrFeishuToken = New(ErrCodeFeishuToken, "飞书Token获取失败")
	ErrFeishuAPI   = New(ErrCodeFeishuAPI, "飞书API调用失败")

	ErrJenkinsConnect = New(ErrCodeJenkinsConnect, "Jenkins连接失败")
	ErrJenkinsAPI     = New(ErrCodeJenkinsAPI, "Jenkins API调用失败")

	ErrK8sConnect = New(ErrCodeK8sConnect, "K8s连接失败")
	ErrK8sDeploy  = New(ErrCodeK8sDeploy, "K8s部署失败")

	ErrArcheryConnect = New(ErrCodeArcheryConnect, "Archery连接失败")
	ErrArcheryAPI     = New(ErrCodeArcheryAPI, "Archery API调用失败")

	ErrRedisLock = New(ErrCodeRedisLock, "Redis锁获取失败")

	ErrDBConnect    = New(ErrCodeDBConnect, "数据库连接失败")
	ErrDBQuery      = New(ErrCodeDBQuery, "数据库查询失败")
	ErrTokenInvalid = New(ErrCodeTokenInvalid, "Token无效")
)

// 错误码到友好消息的映射
var errorCodeMessages = map[int]string{
	ErrCodeInternalError:    "服务器内部错误，请稍后重试",
	ErrCodeInvalidParams:    "请求参数不正确，请检查后重试",
	ErrCodeMethodNotAllowed: "不支持的请求方法",
	ErrCodeRequestTimeout:   "请求超时，请稍后重试",
	ErrCodeUnauthorized:     "登录已过期，请重新登录",
	ErrCodeForbidden:        "您没有权限执行此操作",
	ErrCodeTokenExpired:     "登录已过期，请重新登录",
	ErrCodeTokenInvalid:     "登录凭证无效，请重新登录",
	ErrCodeNotFound:         "请求的资源不存在",
	ErrCodeConflict:         "操作冲突，资源可能已被修改",
	ErrCodeDuplicate:        "资源已存在，请勿重复创建",
	ErrCodeBusiness:         "业务处理失败",
	ErrCodeStatusInvalid:    "当前状态不允许此操作",
	ErrCodeFeishuToken:      "飞书认证失败，请检查配置",
	ErrCodeFeishuAPI:        "飞书服务调用失败，请稍后重试",
	ErrCodeFeishuWebhook:    "飞书消息发送失败",
	ErrCodeJenkinsConnect:   "Jenkins 服务连接失败，请检查配置",
	ErrCodeJenkinsAPI:       "Jenkins 操作失败，请稍后重试",
	ErrCodeJenkinsBuild:     "构建任务执行失败",
	ErrCodeK8sConnect:       "Kubernetes 集群连接失败，请检查配置",
	ErrCodeK8sConfig:        "Kubernetes 配置错误",
	ErrCodeK8sDeploy:        "部署失败，请检查配置和资源状态",
	ErrCodeK8sPod:           "Pod 操作失败",
	ErrCodeArcheryConnect:   "Archery 服务连接失败",
	ErrCodeArcheryAPI:       "Archery 操作失败",
	ErrCodeArcheryWorkflow:  "SQL 工单处理失败",
	ErrCodeRedisConnect:     "缓存服务连接失败",
	ErrCodeRedisLock:        "操作正在进行中，请稍后重试",
	ErrCodeDBConnect:        "数据库连接失败，请稍后重试",
	ErrCodeDBQuery:          "数据查询失败，请稍后重试",
	ErrCodeDBTransaction:    "数据操作失败，请稍后重试",
}

// GetFriendlyMessage 获取友好的错误消息
func GetFriendlyMessage(code int) string {
	if msg, ok := errorCodeMessages[code]; ok {
		return msg
	}
	return "操作失败，请稍后重试"
}

// FormatError 格式化错误消息，用于返回给前端
func FormatError(err error) (int, string) {
	if err == nil {
		return Success, ""
	}

	var appErr *AppError
	if errors.As(err, &appErr) {
		// 如果有自定义消息，使用自定义消息
		if appErr.Message != "" {
			return appErr.Code, appErr.Message
		}
		// 否则使用友好消息
		return appErr.Code, GetFriendlyMessage(appErr.Code)
	}

	// 未知错误
	return ErrCodeInternalError, "操作失败，请稍后重试"
}
