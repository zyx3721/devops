package pipeline

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLogService_SanitizeLogs(t *testing.T) {
	svc := NewLogService(nil)

	tests := []struct {
		name     string
		input    string
		expected string
		desc     string
	}{
		{
			name:     "AWS Access Key",
			input:    "Using AWS AKIAIOSFODNN7EXAMPLE",
			expected: "Using AWS ******AWS_KEY******",
			desc:     "Should mask AWS Access Key ID",
		},
		{
			name:     "AWS Secret Key",
			input:    "aws_secret_access_key=wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
			expected: "aws_secret_access_key=******",
			desc:     "Should mask AWS Secret Access Key",
		},
		{
			name:     "GitHub Token ghp format",
			input:    "Found ghp_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
			expected: "Found ******GITHUB_TOKEN******",
			desc:     "Should mask GitHub personal access token",
		},
		{
			name:     "GitLab Token",
			input:    "Found glpat-xxxxxxxxxxxxxxxxxxxx",
			expected: "Found ******GITLAB_TOKEN******",
			desc:     "Should mask GitLab personal access token",
		},
		{
			name:     "JWT Token",
			input:    "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.dozjgNryP4J3jVmNHl0w5N_XgL0n3I9PlFUP0THsR8U",
			expected: "Bearer ******JWT_TOKEN******",
			desc:     "Should mask JWT token",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := svc.SanitizeLogs(tt.input)
			assert.Equal(t, tt.expected, result, tt.desc)
		})
	}
}

func TestLogService_SanitizeLogs_MoreCases(t *testing.T) {
	svc := NewLogService(nil)

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"Private Key", "-----BEGIN RSA PRIVATE KEY-----\nMIIE...\n-----END RSA PRIVATE KEY-----", "******PRIVATE_KEY******"},
		{"MySQL Connection", "mysql://root:secretpassword@localhost:3306/db", "mysql://***:******@localhost:3306/db"},
		{"Slack Webhook", "Webhook: https://hooks.slack.com/services/T00000000/B00000000/XXXXXXXX", "Webhook: ******SLACK_WEBHOOK******"},
		{"Bearer Token", "Authorization: Bearer abc123xyz789token", "Authorization: Bearer ******"},
		{"Password key=value", "password=mysecretpassword123", "password=******"},
		{"No sensitive data", "Building project... Done!", "Building project... Done!"},
		{"Empty string", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := svc.SanitizeLogs(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestLogService_AddCustomRule(t *testing.T) {
	svc := NewLogService(nil)
	err := svc.AddCustomRule("CustomToken", `custom_token_[A-Za-z0-9]{20}`, "******CUSTOM******")
	assert.NoError(t, err)

	// 使用不包含敏感关键字的输入
	input := "Found: custom_token_abcdefghij1234567890"
	result := svc.SanitizeLogs(input)
	assert.Equal(t, "Found: ******CUSTOM******", result)
}

func TestLogService_AddCustomRule_InvalidPattern(t *testing.T) {
	svc := NewLogService(nil)
	err := svc.AddCustomRule("Invalid", `[invalid`, "replacement")
	assert.Error(t, err)
}

func TestLogService_AddSensitiveKey(t *testing.T) {
	svc := NewLogService(nil)
	svc.AddSensitiveKey("my_custom_secret")

	input := "my_custom_secret=verysecretvalue"
	result := svc.SanitizeLogs(input)
	assert.Equal(t, "my_custom_secret=******", result)
}

func TestLogService_SearchLogs(t *testing.T) {
	svc := NewLogService(nil)
	logs := "Line 1: Starting build\nLine 2: ERROR: failed\nLine 3: Done"

	results := svc.SearchLogs(nil, logs, "error")
	assert.Len(t, results, 1)
	assert.Contains(t, results[0], "ERROR")
}

func TestLogService_HighlightErrors(t *testing.T) {
	svc := NewLogService(nil)
	result := svc.HighlightErrors("Build error occurred")
	assert.Contains(t, result, "[ERROR]error[/ERROR]")
}

func BenchmarkSanitizeLogs(b *testing.B) {
	svc := NewLogService(nil)
	input := "password=secret123 AWS_ACCESS_KEY_ID=AKIAIOSFODNN7EXAMPLE"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		svc.SanitizeLogs(input)
	}
}
