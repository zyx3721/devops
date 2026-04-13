package response

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	apperrors "devops/pkg/errors"
)

// Response 标准响应结构
type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// PageData 分页数据结构
type PageData struct {
	List     interface{} `json:"list"`
	Total    int64       `json:"total"`
	Page     int         `json:"page"`
	PageSize int         `json:"page_size"`
}

// Success 成功响应
func Success(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Code:    0,
		Message: "success",
		Data:    data,
	})
}

// SuccessWithMessage 带消息的成功响应
func SuccessWithMessage(c *gin.Context, message string, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Code:    0,
		Message: message,
		Data:    data,
	})
}

// OK 无数据的成功响应
func OK(c *gin.Context) {
	c.JSON(http.StatusOK, Response{
		Code:    0,
		Message: "success",
	})
}

// OKWithMessage 带消息的无数据成功响应
func OKWithMessage(c *gin.Context, message string) {
	c.JSON(http.StatusOK, Response{
		Code:    0,
		Message: message,
	})
}

// Error 错误响应
func Error(c *gin.Context, code int, message string) {
	httpStatus := getHTTPStatus(code)
	c.JSON(httpStatus, Response{
		Code:    code,
		Message: message,
	})
}

// ErrorWithData 带数据的错误响应
func ErrorWithData(c *gin.Context, code int, message string, data interface{}) {
	httpStatus := getHTTPStatus(code)
	c.JSON(httpStatus, Response{
		Code:    code,
		Message: message,
		Data:    data,
	})
}

// BadRequest 参数错误响应
func BadRequest(c *gin.Context, message string) {
	Error(c, apperrors.ErrCodeInvalidParams, message)
}

// BadRequestWithDetail 带详情的参数错误响应
func BadRequestWithDetail(c *gin.Context, message string, detail string) {
	if detail != "" {
		message = message + "：" + detail
	}
	Error(c, apperrors.ErrCodeInvalidParams, message)
}

// Unauthorized 未授权响应
func Unauthorized(c *gin.Context, message string) {
	Error(c, apperrors.ErrCodeUnauthorized, message)
}

// Forbidden 禁止访问响应
func Forbidden(c *gin.Context, message string) {
	Error(c, apperrors.ErrCodeForbidden, message)
}

// NotFound 资源不存在响应
func NotFound(c *gin.Context, message string) {
	Error(c, apperrors.ErrCodeNotFound, message)
}

// Conflict 冲突响应
func Conflict(c *gin.Context, message string) {
	c.JSON(http.StatusConflict, Response{
		Code:    apperrors.ErrCodeConflict,
		Message: message,
	})
}

// InternalError 内部错误响应
func InternalError(c *gin.Context, message string) {
	Error(c, apperrors.ErrCodeInternalError, message)
}

// ValidationError 参数校验错误响应
func ValidationError(c *gin.Context, errors interface{}) {
	c.JSON(http.StatusBadRequest, Response{
		Code:    apperrors.ErrCodeInvalidParams,
		Message: "参数校验失败",
		Data:    errors,
	})
}

// Page 分页响应
func Page(c *gin.Context, list interface{}, total int64, page, pageSize int) {
	c.JSON(http.StatusOK, Response{
		Code:    0,
		Message: "success",
		Data: PageData{
			List:     list,
			Total:    total,
			Page:     page,
			PageSize: pageSize,
		},
	})
}

// FromError 从 error 构建响应
func FromError(c *gin.Context, err error) {
	if err == nil {
		OK(c)
		return
	}

	var appErr *apperrors.AppError
	if errors.As(err, &appErr) {
		Error(c, appErr.Code, appErr.Message)
		return
	}

	// 未知错误，返回内部错误
	InternalError(c, "服务器内部错误，请稍后重试")
}

// FromErrorWithDefault 从 error 构建响应，带默认消息
func FromErrorWithDefault(c *gin.Context, err error, defaultMsg string) {
	if err == nil {
		OK(c)
		return
	}

	var appErr *apperrors.AppError
	if errors.As(err, &appErr) {
		Error(c, appErr.Code, appErr.Message)
		return
	}

	// 未知错误，使用默认消息
	InternalError(c, defaultMsg)
}

// ===== 常用业务错误响应 =====

// ParamError 参数错误
func ParamError(c *gin.Context, detail string) {
	msg := "请求参数不正确"
	if detail != "" {
		msg = msg + "：" + detail
	}
	Error(c, apperrors.ErrCodeInvalidParams, msg)
}

// ParamIDError ID参数错误
func ParamIDError(c *gin.Context, name string) {
	Error(c, apperrors.ErrCodeInvalidParams, name+"格式不正确")
}

// ResourceNotFound 资源不存在
func ResourceNotFound(c *gin.Context, resource string) {
	Error(c, apperrors.ErrCodeNotFound, resource+"不存在")
}

// OperationFailed 操作失败
func OperationFailed(c *gin.Context, operation string, err error) {
	msg := operation + "失败"
	if err != nil {
		// 检查是否是 AppError
		var appErr *apperrors.AppError
		if errors.As(err, &appErr) {
			Error(c, appErr.Code, msg+"："+appErr.Message)
			return
		}
	}
	Error(c, apperrors.ErrCodeInternalError, msg+"，请稍后重试")
}

// K8sError K8s 操作错误
func K8sError(c *gin.Context, operation string, err error) {
	msg := operation + "失败"
	if err != nil {
		errMsg := err.Error()
		// 简化常见的 K8s 错误消息
		if contains(errMsg, "not found") {
			msg = operation + "失败：资源不存在"
		} else if contains(errMsg, "already exists") {
			msg = operation + "失败：资源已存在"
		} else if contains(errMsg, "forbidden") {
			msg = operation + "失败：权限不足"
		} else if contains(errMsg, "connection refused") {
			msg = operation + "失败：集群连接失败"
		} else if contains(errMsg, "timeout") {
			msg = operation + "失败：请求超时"
		}
	}
	Error(c, apperrors.ErrCodeK8sDeploy, msg)
}

// DBError 数据库操作错误
func DBError(c *gin.Context, operation string) {
	Error(c, apperrors.ErrCodeDBQuery, operation+"失败，请稍后重试")
}

// getHTTPStatus 根据错误码获取 HTTP 状态码
func getHTTPStatus(code int) int {
	status := apperrors.GetHTTPStatus(code)
	// GetHTTPStatus 返回错误码而不是 HTTP 状态码时的兼容处理
	if status >= 1000 {
		return http.StatusInternalServerError
	}
	return status
}

// contains 检查字符串是否包含子串（不区分大小写）
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsLower(s, substr))
}

func containsLower(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if equalFoldAt(s, i, substr) {
			return true
		}
	}
	return false
}

func equalFoldAt(s string, i int, substr string) bool {
	for j := 0; j < len(substr); j++ {
		c1 := s[i+j]
		c2 := substr[j]
		if c1 != c2 && toLower(c1) != toLower(c2) {
			return false
		}
	}
	return true
}

func toLower(c byte) byte {
	if c >= 'A' && c <= 'Z' {
		return c + 32
	}
	return c
}
