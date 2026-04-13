package response

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	apperrors "devops/pkg/errors"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func setupTestContext() (*gin.Context, *httptest.ResponseRecorder) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	return c, w
}

// TestResponseFormatConsistency 测试响应格式一致性
// Property 1: Response Format Consistency
func TestResponseFormatConsistency(t *testing.T) {
	tests := []struct {
		name     string
		action   func(c *gin.Context)
		wantCode int
	}{
		{
			name: "Success response",
			action: func(c *gin.Context) {
				Success(c, map[string]string{"key": "value"})
			},
			wantCode: 0,
		},
		{
			name: "Error response",
			action: func(c *gin.Context) {
				Error(c, apperrors.ErrCodeInvalidParams, "参数错误")
			},
			wantCode: apperrors.ErrCodeInvalidParams,
		},
		{
			name: "Page response",
			action: func(c *gin.Context) {
				Page(c, []string{"a", "b"}, 100, 1, 10)
			},
			wantCode: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, w := setupTestContext()
			tt.action(c)

			var resp Response
			err := json.Unmarshal(w.Body.Bytes(), &resp)
			assert.NoError(t, err, "响应应该是有效的 JSON")
			assert.Equal(t, tt.wantCode, resp.Code, "响应码应该匹配")
			assert.NotEmpty(t, resp.Message, "响应消息不应为空")
		})
	}
}

// TestSuccessResponseCode 测试成功响应码
// Property 2: Success Response Code
func TestSuccessResponseCode(t *testing.T) {
	testCases := []struct {
		name   string
		action func(c *gin.Context)
	}{
		{"Success", func(c *gin.Context) { Success(c, nil) }},
		{"SuccessWithMessage", func(c *gin.Context) { SuccessWithMessage(c, "ok", nil) }},
		{"OK", func(c *gin.Context) { OK(c) }},
		{"OKWithMessage", func(c *gin.Context) { OKWithMessage(c, "done") }},
		{"Page", func(c *gin.Context) { Page(c, []int{}, 0, 1, 10) }},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			c, w := setupTestContext()
			tc.action(c)

			var resp Response
			err := json.Unmarshal(w.Body.Bytes(), &resp)
			assert.NoError(t, err)
			assert.Equal(t, 0, resp.Code, "成功响应的 code 应该是 0")
			assert.Equal(t, http.StatusOK, w.Code, "HTTP 状态码应该是 200")
		})
	}
}

// TestErrorResponseCode 测试错误响应码
// Property 3: Error Response Code
func TestErrorResponseCode(t *testing.T) {
	errorCodes := []int{
		apperrors.ErrCodeInvalidParams,
		apperrors.ErrCodeUnauthorized,
		apperrors.ErrCodeForbidden,
		apperrors.ErrCodeNotFound,
		apperrors.ErrCodeInternalError,
	}

	for _, code := range errorCodes {
		t.Run("ErrorCode_"+string(rune(code)), func(t *testing.T) {
			c, w := setupTestContext()
			Error(c, code, "测试错误")

			var resp Response
			err := json.Unmarshal(w.Body.Bytes(), &resp)
			assert.NoError(t, err)
			assert.Equal(t, code, resp.Code, "错误响应的 code 应该匹配传入的错误码")
			assert.NotEqual(t, 0, resp.Code, "错误响应的 code 不应该是 0")
		})
	}
}

// TestPageResponseStructure 测试分页响应结构
// Property 4: Page Response Structure
func TestPageResponseStructure(t *testing.T) {
	c, w := setupTestContext()
	
	testList := []string{"item1", "item2", "item3"}
	Page(c, testList, 100, 2, 20)

	var resp Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, 0, resp.Code)

	// 验证 data 结构
	data, ok := resp.Data.(map[string]interface{})
	assert.True(t, ok, "data 应该是一个对象")
	
	// 验证必需字段
	assert.Contains(t, data, "list", "应该包含 list 字段")
	assert.Contains(t, data, "total", "应该包含 total 字段")
	assert.Contains(t, data, "page", "应该包含 page 字段")
	assert.Contains(t, data, "page_size", "应该包含 page_size 字段")

	// 验证字段值
	assert.Equal(t, float64(100), data["total"])
	assert.Equal(t, float64(2), data["page"])
	assert.Equal(t, float64(20), data["page_size"])
}

// TestFromError 测试从错误构建响应
func TestFromError(t *testing.T) {
	t.Run("nil error", func(t *testing.T) {
		c, w := setupTestContext()
		FromError(c, nil)

		var resp Response
		json.Unmarshal(w.Body.Bytes(), &resp)
		assert.Equal(t, 0, resp.Code)
	})

	t.Run("AppError", func(t *testing.T) {
		c, w := setupTestContext()
		appErr := apperrors.New(apperrors.ErrCodeNotFound, "资源不存在")
		FromError(c, appErr)

		var resp Response
		json.Unmarshal(w.Body.Bytes(), &resp)
		assert.Equal(t, apperrors.ErrCodeNotFound, resp.Code)
		assert.Equal(t, "资源不存在", resp.Message)
	})
}

// TestConvenienceMethods 测试便捷方法
func TestConvenienceMethods(t *testing.T) {
	t.Run("BadRequest", func(t *testing.T) {
		c, w := setupTestContext()
		BadRequest(c, "参数错误")

		var resp Response
		json.Unmarshal(w.Body.Bytes(), &resp)
		assert.Equal(t, apperrors.ErrCodeInvalidParams, resp.Code)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Unauthorized", func(t *testing.T) {
		c, w := setupTestContext()
		Unauthorized(c, "未授权")

		var resp Response
		json.Unmarshal(w.Body.Bytes(), &resp)
		assert.Equal(t, apperrors.ErrCodeUnauthorized, resp.Code)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("NotFound", func(t *testing.T) {
		c, w := setupTestContext()
		NotFound(c, "资源不存在")

		var resp Response
		json.Unmarshal(w.Body.Bytes(), &resp)
		assert.Equal(t, apperrors.ErrCodeNotFound, resp.Code)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("InternalError", func(t *testing.T) {
		c, w := setupTestContext()
		InternalError(c, "内部错误")

		var resp Response
		json.Unmarshal(w.Body.Bytes(), &resp)
		assert.Equal(t, apperrors.ErrCodeInternalError, resp.Code)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}
