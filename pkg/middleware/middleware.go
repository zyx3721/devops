package middleware

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	apperrors "devops/pkg/errors"
)

// CrsMiddleware 跨域中间件
func CrsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		c.Writer.Header().Set("Access-Control-Max-Age", "86400")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusOK)
			return
		}

		c.Next()
	}
}

// Success 成功响应
func Success(data any, c *gin.Context) {
	c.JSON(http.StatusOK, data)
}

// Failed 失败响应
func Failed(err error, c *gin.Context) {
	httpCode := http.StatusInternalServerError
	if v, ok := err.(*ApiException); ok {
		if v.HttpCode != 0 {
			httpCode = v.HttpCode
		}
	} else {
		err = ErrServerInternal("%s", err.Error())
	}

	c.JSON(httpCode, err)
	c.Abort()
}

func NewApiException(code int, message string) *ApiException {
	return &ApiException{
		Code:    code,
		Message: message,
	}
}

// ApiException 业务异常
type ApiException struct {
	Code     int    `json:"code"`
	Message  string `json:"message"`
	HttpCode int    `json:"-"`
}

func (e *ApiException) Error() string {
	return e.Message
}

func (e *ApiException) String() string {
	dj, _ := json.MarshalIndent(e, "", "  ")
	return string(dj)
}

func (e *ApiException) WithMessage(msg string) *ApiException {
	e.Message = msg
	return e
}

func (e *ApiException) WithHttpCode(httpCode int) *ApiException {
	e.HttpCode = httpCode
	return e
}

func ErrServerInternal(format string, a ...any) *ApiException {
	return &ApiException{
		Code:    50000,
		Message: fmt.Sprintf(format, a...),
	}
}

func ErrNotFound(format string, a ...any) *ApiException {
	return &ApiException{
		Code:    404,
		Message: fmt.Sprintf(format, a...),
	}
}

func ErrValidateFailed(format string, a ...any) *ApiException {
	return &ApiException{
		Code:    400,
		Message: fmt.Sprintf(format, a...),
	}
}

// Error 统一错误响应
func Error(c *gin.Context, err error) {
	if appErr, ok := err.(*apperrors.AppError); ok {
		httpCode := apperrors.GetHTTPStatus(appErr.Code)
		c.JSON(httpCode, gin.H{
			"code":    appErr.Code,
			"message": appErr.Message,
			"data":    appErr.Details,
		})
	} else {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    apperrors.ErrCodeInternalError,
			"message": err.Error(),
		})
	}
}

// ErrorWithMessage 带自定义消息的错误响应
func ErrorWithMessage(c *gin.Context, err error, message string) {
	if appErr, ok := err.(*apperrors.AppError); ok {
		httpCode := apperrors.GetHTTPStatus(appErr.Code)
		c.JSON(httpCode, gin.H{
			"code":    appErr.Code,
			"message": message,
			"data":    appErr.Details,
		})
	} else {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    apperrors.ErrCodeInternalError,
			"message": message,
		})
	}
}
