package errors

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestErrorWrappingPreservesChain 测试错误包装保留错误链
// Property 5: Error Wrapping Preserves Chain
func TestErrorWrappingPreservesChain(t *testing.T) {
	originalErr := fmt.Errorf("original error")
	wrappedErr := Wrap(originalErr, ErrCodeInternalError, "wrapped message")

	// Unwrap 应该返回原始错误
	unwrapped := wrappedErr.Unwrap()
	assert.Equal(t, originalErr, unwrapped, "Unwrap 应该返回原始错误")

	// errors.Unwrap 也应该工作
	unwrapped2 := errors.Unwrap(wrappedErr)
	assert.Equal(t, originalErr, unwrapped2, "errors.Unwrap 应该返回原始错误")
}

// TestErrorCodeExtraction 测试错误码提取
// Property 6: Error Code Extraction
func TestErrorCodeExtraction(t *testing.T) {
	t.Run("AppError returns its code", func(t *testing.T) {
		appErr := New(ErrCodeNotFound, "not found")
		code := GetCode(appErr)
		assert.Equal(t, ErrCodeNotFound, code)
	})

	t.Run("Wrapped AppError returns its code", func(t *testing.T) {
		originalErr := fmt.Errorf("db error")
		wrappedErr := Wrap(originalErr, ErrCodeDBQuery, "query failed")
		code := GetCode(wrappedErr)
		assert.Equal(t, ErrCodeDBQuery, code)
	})

	t.Run("Non-AppError returns ErrCodeInternalError", func(t *testing.T) {
		regularErr := fmt.Errorf("regular error")
		code := GetCode(regularErr)
		assert.Equal(t, ErrCodeInternalError, code)
	})

	t.Run("Nil error returns Success", func(t *testing.T) {
		code := GetCode(nil)
		assert.Equal(t, Success, code)
	})
}

// TestErrorMessage 测试错误消息
func TestErrorMessage(t *testing.T) {
	t.Run("AppError returns its message", func(t *testing.T) {
		appErr := New(ErrCodeNotFound, "资源不存在")
		msg := GetMessage(appErr)
		assert.Equal(t, "资源不存在", msg)
	})

	t.Run("Non-AppError returns default message", func(t *testing.T) {
		regularErr := fmt.Errorf("regular error")
		msg := GetMessage(regularErr)
		assert.Equal(t, "内部错误", msg)
	})

	t.Run("Nil error returns empty string", func(t *testing.T) {
		msg := GetMessage(nil)
		assert.Equal(t, "", msg)
	})
}

// TestErrorIs 测试 errors.Is 支持
func TestErrorIs(t *testing.T) {
	t.Run("Same code matches", func(t *testing.T) {
		err1 := New(ErrCodeNotFound, "not found 1")
		err2 := New(ErrCodeNotFound, "not found 2")
		assert.True(t, errors.Is(err1, err2), "相同错误码的错误应该匹配")
	})

	t.Run("Different code does not match", func(t *testing.T) {
		err1 := New(ErrCodeNotFound, "not found")
		err2 := New(ErrCodeInternalError, "internal error")
		assert.False(t, errors.Is(err1, err2), "不同错误码的错误不应该匹配")
	})

	t.Run("Wrapped error matches", func(t *testing.T) {
		originalErr := fmt.Errorf("original")
		wrappedErr := Wrap(originalErr, ErrCodeNotFound, "wrapped")
		targetErr := New(ErrCodeNotFound, "target")
		assert.True(t, errors.Is(wrappedErr, targetErr), "包装后的错误应该匹配相同错误码")
	})
}

// TestErrorAs 测试 errors.As 支持
func TestErrorAs(t *testing.T) {
	t.Run("AppError can be extracted", func(t *testing.T) {
		appErr := New(ErrCodeNotFound, "not found")
		var target *AppError
		assert.True(t, errors.As(appErr, &target))
		assert.Equal(t, ErrCodeNotFound, target.Code)
	})

	t.Run("Wrapped error can be extracted", func(t *testing.T) {
		originalErr := fmt.Errorf("original")
		wrappedErr := Wrap(originalErr, ErrCodeDBQuery, "query failed")
		var target *AppError
		assert.True(t, errors.As(wrappedErr, &target))
		assert.Equal(t, ErrCodeDBQuery, target.Code)
	})
}

// TestWrapNil 测试包装 nil 错误
func TestWrapNil(t *testing.T) {
	result := Wrap(nil, ErrCodeInternalError, "message")
	assert.Nil(t, result, "包装 nil 应该返回 nil")
}

// TestWrapWithDetails 测试带详情的包装
func TestWrapWithDetails(t *testing.T) {
	originalErr := fmt.Errorf("original")
	details := map[string]string{"field": "value"}
	wrappedErr := WrapWithDetails(originalErr, ErrCodeInvalidParams, "validation failed", details)

	assert.Equal(t, ErrCodeInvalidParams, wrappedErr.Code)
	assert.Equal(t, "validation failed", wrappedErr.Message)
	assert.Equal(t, details, wrappedErr.Details)
	assert.Equal(t, originalErr, wrappedErr.Unwrap())
}

// TestNewWithCause 测试带原因创建错误
func TestNewWithCause(t *testing.T) {
	cause := fmt.Errorf("root cause")
	appErr := NewWithCause(ErrCodeDBConnect, "connection failed", cause)

	assert.Equal(t, ErrCodeDBConnect, appErr.Code)
	assert.Equal(t, "connection failed", appErr.Message)
	assert.Equal(t, cause, appErr.Unwrap())
}

// TestWithCause 测试添加原因
func TestWithCause(t *testing.T) {
	appErr := New(ErrCodeInternalError, "error")
	cause := fmt.Errorf("cause")
	appErr.WithCause(cause)

	assert.Equal(t, cause, appErr.Unwrap())
}

// TestErrorString 测试错误字符串
func TestErrorString(t *testing.T) {
	t.Run("Without cause", func(t *testing.T) {
		appErr := New(ErrCodeNotFound, "not found")
		assert.Equal(t, "not found", appErr.Error())
	})

	t.Run("With cause", func(t *testing.T) {
		cause := fmt.Errorf("root cause")
		appErr := NewWithCause(ErrCodeDBConnect, "connection failed", cause)
		assert.Contains(t, appErr.Error(), "connection failed")
		assert.Contains(t, appErr.Error(), "root cause")
	})
}

// TestIsAppError 测试 IsAppError
func TestIsAppError(t *testing.T) {
	t.Run("AppError returns true", func(t *testing.T) {
		appErr := New(ErrCodeNotFound, "not found")
		assert.True(t, IsAppError(appErr))
	})

	t.Run("Regular error returns false", func(t *testing.T) {
		regularErr := fmt.Errorf("regular error")
		assert.False(t, IsAppError(regularErr))
	})
}

// TestAsAppError 测试 AsAppError
func TestAsAppError(t *testing.T) {
	t.Run("AppError conversion succeeds", func(t *testing.T) {
		appErr := New(ErrCodeNotFound, "not found")
		result, ok := AsAppError(appErr)
		assert.True(t, ok)
		assert.Equal(t, ErrCodeNotFound, result.Code)
	})

	t.Run("Regular error conversion fails", func(t *testing.T) {
		regularErr := fmt.Errorf("regular error")
		result, ok := AsAppError(regularErr)
		assert.False(t, ok)
		assert.Nil(t, result)
	})
}

// TestGetHTTPStatus 测试 HTTP 状态码映射
func TestGetHTTPStatus(t *testing.T) {
	tests := []struct {
		code       int
		wantStatus int
	}{
		{ErrCodeInvalidParams, 400},
		{ErrCodeUnauthorized, 401},
		{ErrCodeForbidden, 403},
		{ErrCodeNotFound, 404},
		{ErrCodeInternalError, 500},
		{ErrCodeDBConnect, 503},
		{9999, 500}, // 未知错误码
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("Code_%d", tt.code), func(t *testing.T) {
			status := GetHTTPStatus(tt.code)
			assert.Equal(t, tt.wantStatus, status)
		})
	}
}

// ============================================================================
// Task 7.2: Property 4 - Error Wrapping Preserves Chain (增强测试)
// Validates: Requirements 4.1
// ============================================================================

// TestErrorWrappingPreservesChain_Property 属性测试：错误包装保留错误链
func TestErrorWrappingPreservesChain_Property(t *testing.T) {
	// 测试多层包装
	t.Run("multiple wrapping preserves chain", func(t *testing.T) {
		original := fmt.Errorf("original error")
		wrapped1 := Wrap(original, ErrCodeDBQuery, "layer 1")
		wrapped2 := Wrap(wrapped1, ErrCodeInternalError, "layer 2")

		// 第一层 Unwrap 应该返回 wrapped1
		unwrapped := wrapped2.Unwrap()
		assert.Equal(t, wrapped1, unwrapped)

		// 继续 Unwrap 应该返回 original
		if appErr, ok := unwrapped.(*AppError); ok {
			assert.Equal(t, original, appErr.Unwrap())
		}
	})

	// 测试不同错误码的包装
	t.Run("wrapping with different codes", func(t *testing.T) {
		codes := []int{
			ErrCodeInvalidParams,
			ErrCodeUnauthorized,
			ErrCodeForbidden,
			ErrCodeNotFound,
			ErrCodeInternalError,
			ErrCodeDBConnect,
			ErrCodeDBQuery,
		}

		for _, code := range codes {
			original := fmt.Errorf("test error for code %d", code)
			wrapped := Wrap(original, code, "wrapped")

			assert.Equal(t, original, wrapped.Unwrap(),
				"Unwrap should return original for code %d", code)
			assert.Equal(t, code, wrapped.Code,
				"Code should be preserved for code %d", code)
		}
	})

	// 测试空消息包装
	t.Run("wrapping with empty message", func(t *testing.T) {
		original := fmt.Errorf("original")
		wrapped := Wrap(original, ErrCodeInternalError, "")

		assert.Equal(t, original, wrapped.Unwrap())
		assert.Equal(t, "", wrapped.Message)
	})

	// 测试包装 AppError
	t.Run("wrapping AppError", func(t *testing.T) {
		original := New(ErrCodeNotFound, "not found")
		wrapped := Wrap(original, ErrCodeInternalError, "wrapped")

		assert.Equal(t, original, wrapped.Unwrap())
		assert.Equal(t, ErrCodeInternalError, wrapped.Code)
	})
}

// TestErrorChainIntegrity 测试错误链完整性
func TestErrorChainIntegrity(t *testing.T) {
	t.Run("errors.Is works through chain", func(t *testing.T) {
		original := New(ErrCodeNotFound, "original")
		wrapped := Wrap(original, ErrCodeInternalError, "wrapped")

		// wrapped 应该匹配 ErrCodeInternalError
		target := New(ErrCodeInternalError, "target")
		assert.True(t, errors.Is(wrapped, target))
	})

	t.Run("errors.As works through chain", func(t *testing.T) {
		original := fmt.Errorf("standard error")
		wrapped := Wrap(original, ErrCodeDBQuery, "db error")

		var appErr *AppError
		assert.True(t, errors.As(wrapped, &appErr))
		assert.Equal(t, ErrCodeDBQuery, appErr.Code)
	})
}

// TestWrapNilVariants 测试 nil 包装的各种情况
func TestWrapNilVariants(t *testing.T) {
	t.Run("Wrap nil returns nil", func(t *testing.T) {
		result := Wrap(nil, ErrCodeInternalError, "message")
		assert.Nil(t, result)
	})

	t.Run("WrapWithDetails nil returns nil", func(t *testing.T) {
		result := WrapWithDetails(nil, ErrCodeInternalError, "message", nil)
		assert.Nil(t, result)
	})
}

// TestErrorCodeConsistency 测试错误码一致性
func TestErrorCodeConsistency(t *testing.T) {
	// 验证 GetCode 对各种输入的一致性
	testCases := []struct {
		name     string
		err      error
		wantCode int
	}{
		{"nil error", nil, Success},
		{"standard error", fmt.Errorf("test"), ErrCodeInternalError},
		{"AppError NotFound", New(ErrCodeNotFound, "not found"), ErrCodeNotFound},
		{"AppError Unauthorized", New(ErrCodeUnauthorized, "unauthorized"), ErrCodeUnauthorized},
		{"Wrapped error", Wrap(fmt.Errorf("test"), ErrCodeDBQuery, "db"), ErrCodeDBQuery},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			code := GetCode(tc.err)
			assert.Equal(t, tc.wantCode, code)
		})
	}
}
