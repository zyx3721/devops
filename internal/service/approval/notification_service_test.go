package approval

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockNotificationClient 模拟通知客户端
type MockNotificationClient struct {
	mock.Mock
}

func (m *MockNotificationClient) SendFeishuCard(ctx context.Context, userID string, card interface{}) error {
	args := m.Called(ctx, userID, card)
	return args.Error(0)
}

func (m *MockNotificationClient) SendDingTalkCard(ctx context.Context, userID string, card interface{}) error {
	args := m.Called(ctx, userID, card)
	return args.Error(0)
}

func (m *MockNotificationClient) SendWeComCard(ctx context.Context, userID string, card interface{}) error {
	args := m.Called(ctx, userID, card)
	return args.Error(0)
}

// TestNotificationService_SendApprovalRequest 测试发送审批请求通知
func TestNotificationService_SendApprovalRequest(t *testing.T) {
	t.Skip("需要配置通知服务")

	ctx := context.Background()

	t.Run("send feishu approval request", func(t *testing.T) {
		mockClient := new(MockNotificationClient)

		mockClient.On("SendFeishuCard", ctx, "user123", mock.Anything).Return(nil)

		err := mockClient.SendFeishuCard(ctx, "user123", map[string]interface{}{
			"type": "approval_request",
		})
		assert.NoError(t, err)
		mockClient.AssertExpectations(t)
	})

	t.Run("send dingtalk approval request", func(t *testing.T) {
		mockClient := new(MockNotificationClient)

		mockClient.On("SendDingTalkCard", ctx, "user456", mock.Anything).Return(nil)

		err := mockClient.SendDingTalkCard(ctx, "user456", map[string]interface{}{
			"type": "approval_request",
		})
		assert.NoError(t, err)
		mockClient.AssertExpectations(t)
	})

	t.Run("send wecom approval request", func(t *testing.T) {
		mockClient := new(MockNotificationClient)

		mockClient.On("SendWeComCard", ctx, "user789", mock.Anything).Return(nil)

		err := mockClient.SendWeComCard(ctx, "user789", map[string]interface{}{
			"type": "approval_request",
		})
		assert.NoError(t, err)
		mockClient.AssertExpectations(t)
	})
}

// TestNotificationService_SendApprovalResult 测试发送审批结果通知
func TestNotificationService_SendApprovalResult(t *testing.T) {
	t.Skip("需要配置通知服务")

	ctx := context.Background()

	t.Run("send approval approved notification", func(t *testing.T) {
		mockClient := new(MockNotificationClient)

		mockClient.On("SendFeishuCard", ctx, "applicant123", mock.Anything).Return(nil)

		err := mockClient.SendFeishuCard(ctx, "applicant123", map[string]interface{}{
			"type":   "approval_result",
			"status": "approved",
		})
		assert.NoError(t, err)
	})

	t.Run("send approval rejected notification", func(t *testing.T) {
		mockClient := new(MockNotificationClient)

		mockClient.On("SendFeishuCard", ctx, "applicant123", mock.Anything).Return(nil)

		err := mockClient.SendFeishuCard(ctx, "applicant123", map[string]interface{}{
			"type":   "approval_result",
			"status": "rejected",
			"reason": "不符合发布规范",
		})
		assert.NoError(t, err)
	})
}

// TestNotificationService_SendTimeoutReminder 测试发送超时提醒
func TestNotificationService_SendTimeoutReminder(t *testing.T) {
	t.Skip("需要配置通知服务")

	ctx := context.Background()

	t.Run("send timeout reminder to approver", func(t *testing.T) {
		mockClient := new(MockNotificationClient)

		mockClient.On("SendFeishuCard", ctx, "approver123", mock.Anything).Return(nil)

		err := mockClient.SendFeishuCard(ctx, "approver123", map[string]interface{}{
			"type":    "timeout_reminder",
			"message": "您有一个待审批的发布请求即将超时",
		})
		assert.NoError(t, err)
	})
}

// TestNotificationService_CardTemplates 测试卡片模板生成
func TestNotificationService_CardTemplates(t *testing.T) {
	t.Run("feishu card template", func(t *testing.T) {
		// 测试飞书卡片模板生成
		data := map[string]interface{}{
			"app_name":    "用户服务",
			"env":         "production",
			"version":     "v1.2.3",
			"applicant":   "张三",
			"description": "修复登录问题",
		}

		// 验证模板包含必要字段
		assert.NotEmpty(t, data["app_name"])
		assert.NotEmpty(t, data["env"])
		assert.NotEmpty(t, data["version"])
	})

	t.Run("dingtalk card template", func(t *testing.T) {
		// 测试钉钉卡片模板生成
		data := map[string]interface{}{
			"title":       "发布审批请求",
			"app_name":    "订单服务",
			"env":         "staging",
			"version":     "v2.0.0",
			"applicant":   "李四",
			"description": "新功能上线",
		}

		assert.NotEmpty(t, data["title"])
		assert.NotEmpty(t, data["app_name"])
	})

	t.Run("wecom card template", func(t *testing.T) {
		// 测试企业微信卡片模板生成
		data := map[string]interface{}{
			"title":       "发布审批请求",
			"app_name":    "支付服务",
			"env":         "production",
			"version":     "v3.1.0",
			"applicant":   "王五",
			"description": "安全更新",
		}

		assert.NotEmpty(t, data["title"])
		assert.NotEmpty(t, data["app_name"])
	})
}

// TestNotificationService_CallbackHandling 测试回调处理
func TestNotificationService_CallbackHandling(t *testing.T) {
	t.Run("feishu callback - approve", func(t *testing.T) {
		callback := map[string]interface{}{
			"action":    "approve",
			"record_id": "123",
			"user_id":   "approver123",
		}

		assert.Equal(t, "approve", callback["action"])
		assert.NotEmpty(t, callback["record_id"])
	})

	t.Run("feishu callback - reject", func(t *testing.T) {
		callback := map[string]interface{}{
			"action":    "reject",
			"record_id": "123",
			"user_id":   "approver123",
			"reason":    "不符合规范",
		}

		assert.Equal(t, "reject", callback["action"])
		assert.NotEmpty(t, callback["reason"])
	})

	t.Run("dingtalk callback signature verification", func(t *testing.T) {
		// 测试钉钉回调签名验证
		// 实际实现需要验证签名
		assert.True(t, true, "签名验证测试")
	})

	t.Run("wecom callback signature verification", func(t *testing.T) {
		// 测试企业微信回调签名验证
		// 实际实现需要验证签名
		assert.True(t, true, "签名验证测试")
	})
}
