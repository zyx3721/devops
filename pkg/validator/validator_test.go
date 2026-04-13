package validator

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestUser 测试用结构体
type TestUser struct {
	Username string `json:"username" validate:"required,min=3,max=20" label:"用户名"`
	Email    string `json:"email" validate:"required,email" label:"邮箱"`
	Age      int    `json:"age" validate:"gte=0,lte=150" label:"年龄"`
	Phone    string `json:"phone" validate:"omitempty,mobile" label:"手机号"`
	Password string `json:"password" validate:"required,min=6" label:"密码"`
}

// TestValidationErrorFormat 测试验证错误格式
// Property 7: Validation Error Format
func TestValidationErrorFormat(t *testing.T) {
	user := TestUser{
		Username: "",
		Email:    "invalid",
		Age:      200,
		Password: "123",
	}

	errors := Validate(user)
	assert.NotEmpty(t, errors, "应该有验证错误")

	for _, err := range errors {
		assert.NotEmpty(t, err.Field, "错误应该包含字段名")
		assert.NotEmpty(t, err.Message, "错误应该包含消息")
		// 验证消息是中文
		assert.True(t, containsChinese(err.Message), "错误消息应该是中文: %s", err.Message)
	}
}

// TestRequiredFieldValidation 测试必填字段验证
// Property 8: Required Field Validation
func TestRequiredFieldValidation(t *testing.T) {
	testCases := []struct {
		name     string
		user     TestUser
		hasError bool
	}{
		{
			name: "empty username",
			user: TestUser{
				Username: "",
				Email:    "test@example.com",
				Password: "123456",
			},
			hasError: true,
		},
		{
			name: "empty email",
			user: TestUser{
				Username: "testuser",
				Email:    "",
				Password: "123456",
			},
			hasError: true,
		},
		{
			name: "empty password",
			user: TestUser{
				Username: "testuser",
				Email:    "test@example.com",
				Password: "",
			},
			hasError: true,
		},
		{
			name: "all required fields filled",
			user: TestUser{
				Username: "testuser",
				Email:    "test@example.com",
				Password: "123456",
			},
			hasError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			errors := Validate(tc.user)
			if tc.hasError {
				assert.NotEmpty(t, errors, "应该有验证错误")
			} else {
				assert.Empty(t, errors, "不应该有验证错误")
			}
		})
	}
}

// TestMinMaxValidation 测试最小最大值验证
func TestMinMaxValidation(t *testing.T) {
	t.Run("username too short", func(t *testing.T) {
		user := TestUser{
			Username: "ab",
			Email:    "test@example.com",
			Password: "123456",
		}
		errors := Validate(user)
		assert.NotEmpty(t, errors)
		assert.Equal(t, "用户名", errors[0].Field)
	})

	t.Run("password too short", func(t *testing.T) {
		user := TestUser{
			Username: "testuser",
			Email:    "test@example.com",
			Password: "12345",
		}
		errors := Validate(user)
		assert.NotEmpty(t, errors)
		assert.Equal(t, "密码", errors[0].Field)
	})

	t.Run("age out of range", func(t *testing.T) {
		user := TestUser{
			Username: "testuser",
			Email:    "test@example.com",
			Password: "123456",
			Age:      200,
		}
		errors := Validate(user)
		assert.NotEmpty(t, errors)
		assert.Equal(t, "年龄", errors[0].Field)
	})
}

// TestEmailValidation 测试邮箱验证
func TestEmailValidation(t *testing.T) {
	testCases := []struct {
		email    string
		hasError bool
	}{
		{"test@example.com", false},
		{"user.name@domain.org", false},
		{"invalid", true},
		{"@example.com", true},
		{"test@", true},
	}

	for _, tc := range testCases {
		t.Run(tc.email, func(t *testing.T) {
			user := TestUser{
				Username: "testuser",
				Email:    tc.email,
				Password: "123456",
			}
			errors := Validate(user)
			if tc.hasError {
				assert.NotEmpty(t, errors)
			} else {
				assert.Empty(t, errors)
			}
		})
	}
}

// TestMobileValidation 测试手机号验证
func TestMobileValidation(t *testing.T) {
	testCases := []struct {
		phone    string
		hasError bool
	}{
		{"13800138000", false},
		{"15912345678", false},
		{"12345678901", true}, // 不以1开头
		{"1380013800", true},  // 长度不对
		{"", false},           // 空值允许（omitempty）
	}

	for _, tc := range testCases {
		t.Run(tc.phone, func(t *testing.T) {
			user := TestUser{
				Username: "testuser",
				Email:    "test@example.com",
				Password: "123456",
				Phone:    tc.phone,
			}
			errors := Validate(user)
			if tc.hasError {
				assert.NotEmpty(t, errors)
			} else {
				assert.Empty(t, errors)
			}
		})
	}
}

// TestValidateVar 测试单变量验证
func TestValidateVar(t *testing.T) {
	t.Run("valid email", func(t *testing.T) {
		err := ValidateVar("test@example.com", "email")
		assert.NoError(t, err)
	})

	t.Run("invalid email", func(t *testing.T) {
		err := ValidateVar("invalid", "email")
		assert.Error(t, err)
	})

	t.Run("valid required", func(t *testing.T) {
		err := ValidateVar("value", "required")
		assert.NoError(t, err)
	})

	t.Run("invalid required", func(t *testing.T) {
		err := ValidateVar("", "required")
		assert.Error(t, err)
	})
}

// TestValidateAndFormat 测试验证并格式化
func TestValidateAndFormat(t *testing.T) {
	t.Run("valid struct", func(t *testing.T) {
		user := TestUser{
			Username: "testuser",
			Email:    "test@example.com",
			Password: "123456",
		}
		ok, msg := ValidateAndFormat(user)
		assert.True(t, ok)
		assert.Empty(t, msg)
	})

	t.Run("invalid struct", func(t *testing.T) {
		user := TestUser{
			Username: "",
			Email:    "test@example.com",
			Password: "123456",
		}
		ok, msg := ValidateAndFormat(user)
		assert.False(t, ok)
		assert.NotEmpty(t, msg)
	})
}

// TestValidateAndFormatAll 测试验证并返回所有错误
func TestValidateAndFormatAll(t *testing.T) {
	user := TestUser{
		Username: "",
		Email:    "invalid",
		Password: "123",
	}
	ok, messages := ValidateAndFormatAll(user)
	assert.False(t, ok)
	assert.True(t, len(messages) >= 3, "应该有多个错误")
}

// containsChinese 检查字符串是否包含中文
func containsChinese(s string) bool {
	for _, r := range s {
		if r >= 0x4e00 && r <= 0x9fff {
			return true
		}
	}
	return false
}
