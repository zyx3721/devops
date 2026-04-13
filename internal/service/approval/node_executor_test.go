package approval

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"devops/internal/models"
)

// ============================================================================
// Task 3.5: Property 2 - 审批模式完成判断正确性
// Validates: Requirements 4.2, 4.3, 4.4
// ============================================================================

func TestCheckComplete_AnyMode(t *testing.T) {
	// Property 2.1: any 模式 - 任一人通过即完成
	executor := &NodeExecutor{}

	tests := []struct {
		name          string
		nodeInstance  *models.ApprovalNodeInstance
		wantCompleted bool
		wantStatus    string
	}{
		{
			name: "any模式_一人通过_应完成",
			nodeInstance: &models.ApprovalNodeInstance{
				ApproveMode:   "any",
				Approvers:     "1,2,3",
				ApprovedCount: 1,
				RejectedCount: 0,
			},
			wantCompleted: true,
			wantStatus:    "approved",
		},
		{
			name: "any模式_无人通过_未完成",
			nodeInstance: &models.ApprovalNodeInstance{
				ApproveMode:   "any",
				Approvers:     "1,2,3",
				ApprovedCount: 0,
				RejectedCount: 0,
			},
			wantCompleted: false,
			wantStatus:    "",
		},
		{
			name: "any模式_有人拒绝但无人通过_未完成",
			nodeInstance: &models.ApprovalNodeInstance{
				ApproveMode:   "any",
				Approvers:     "1,2,3",
				ApprovedCount: 0,
				RejectedCount: 2,
			},
			wantCompleted: false,
			wantStatus:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			completed, status := executor.CheckComplete(tt.nodeInstance)
			assert.Equal(t, tt.wantCompleted, completed)
			assert.Equal(t, tt.wantStatus, status)
		})
	}
}

func TestCheckComplete_AllMode(t *testing.T) {
	// Property 2.2: all 模式 - 所有人通过才完成
	executor := &NodeExecutor{}

	tests := []struct {
		name          string
		nodeInstance  *models.ApprovalNodeInstance
		wantCompleted bool
		wantStatus    string
	}{
		{
			name: "all模式_全部通过_应完成",
			nodeInstance: &models.ApprovalNodeInstance{
				ApproveMode:   "all",
				Approvers:     "1,2,3",
				ApprovedCount: 3,
				RejectedCount: 0,
			},
			wantCompleted: true,
			wantStatus:    "approved",
		},
		{
			name: "all模式_部分通过_未完成",
			nodeInstance: &models.ApprovalNodeInstance{
				ApproveMode:   "all",
				Approvers:     "1,2,3",
				ApprovedCount: 2,
				RejectedCount: 0,
			},
			wantCompleted: false,
			wantStatus:    "",
		},
		{
			name: "all模式_全部操作有拒绝_应拒绝",
			nodeInstance: &models.ApprovalNodeInstance{
				ApproveMode:   "all",
				Approvers:     "1,2,3",
				ApprovedCount: 2,
				RejectedCount: 1,
			},
			wantCompleted: true,
			wantStatus:    "rejected",
		},
		{
			name: "all模式_单人审批通过_应完成",
			nodeInstance: &models.ApprovalNodeInstance{
				ApproveMode:   "all",
				Approvers:     "1",
				ApprovedCount: 1,
				RejectedCount: 0,
			},
			wantCompleted: true,
			wantStatus:    "approved",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			completed, status := executor.CheckComplete(tt.nodeInstance)
			assert.Equal(t, tt.wantCompleted, completed)
			assert.Equal(t, tt.wantStatus, status)
		})
	}
}

func TestCheckComplete_CountMode(t *testing.T) {
	// Property 2.3: count 模式 - 指定人数通过即完成
	executor := &NodeExecutor{}

	tests := []struct {
		name          string
		nodeInstance  *models.ApprovalNodeInstance
		wantCompleted bool
		wantStatus    string
	}{
		{
			name: "count模式_达到指定人数_应完成",
			nodeInstance: &models.ApprovalNodeInstance{
				ApproveMode:   "count",
				Approvers:     "1,2,3,4,5",
				ApproveCount:  3,
				ApprovedCount: 3,
				RejectedCount: 0,
			},
			wantCompleted: true,
			wantStatus:    "approved",
		},
		{
			name: "count模式_超过指定人数_应完成",
			nodeInstance: &models.ApprovalNodeInstance{
				ApproveMode:   "count",
				Approvers:     "1,2,3,4,5",
				ApproveCount:  2,
				ApprovedCount: 4,
				RejectedCount: 0,
			},
			wantCompleted: true,
			wantStatus:    "approved",
		},
		{
			name: "count模式_未达到指定人数_未完成",
			nodeInstance: &models.ApprovalNodeInstance{
				ApproveMode:   "count",
				Approvers:     "1,2,3,4,5",
				ApproveCount:  3,
				ApprovedCount: 2,
				RejectedCount: 0,
			},
			wantCompleted: false,
			wantStatus:    "",
		},
		{
			name: "count模式_剩余人数不足以达标_应拒绝",
			nodeInstance: &models.ApprovalNodeInstance{
				ApproveMode:   "count",
				Approvers:     "1,2,3,4,5",
				ApproveCount:  4,
				ApprovedCount: 1,
				RejectedCount: 3,
			},
			wantCompleted: true,
			wantStatus:    "rejected",
		},
		{
			name: "count模式_刚好可能达标_未完成",
			nodeInstance: &models.ApprovalNodeInstance{
				ApproveMode:   "count",
				Approvers:     "1,2,3,4,5",
				ApproveCount:  3,
				ApprovedCount: 1,
				RejectedCount: 2,
			},
			wantCompleted: false,
			wantStatus:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			completed, status := executor.CheckComplete(tt.nodeInstance)
			assert.Equal(t, tt.wantCompleted, completed)
			assert.Equal(t, tt.wantStatus, status)
		})
	}
}

// ============================================================================
// Task 3.6: Property 3 - 辅助函数测试
// Validates: Requirements 1.4, 5.3
// ============================================================================

func TestIsApprover(t *testing.T) {
	executor := &NodeExecutor{}

	tests := []struct {
		name      string
		approvers string
		userID    uint
		expected  bool
	}{
		{
			name:      "用户在审批人列表中",
			approvers: "1,2,3",
			userID:    2,
			expected:  true,
		},
		{
			name:      "用户不在审批人列表中",
			approvers: "1,2,3",
			userID:    99,
			expected:  false,
		},
		{
			name:      "单个审批人匹配",
			approvers: "1",
			userID:    1,
			expected:  true,
		},
		{
			name:      "单个审批人不匹配",
			approvers: "1",
			userID:    2,
			expected:  false,
		},
		{
			name:      "用户ID为0",
			approvers: "0,1,2",
			userID:    0,
			expected:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := executor.isApprover(tt.approvers, tt.userID)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestReplaceApprover(t *testing.T) {
	executor := &NodeExecutor{}

	tests := []struct {
		name       string
		approvers  string
		fromUserID uint
		toUserID   uint
		expected   string
	}{
		{
			name:       "替换中间的审批人",
			approvers:  "1,2,3",
			fromUserID: 2,
			toUserID:   99,
			expected:   "1,99,3",
		},
		{
			name:       "替换第一个审批人",
			approvers:  "1,2,3",
			fromUserID: 1,
			toUserID:   99,
			expected:   "99,2,3",
		},
		{
			name:       "替换最后一个审批人",
			approvers:  "1,2,3",
			fromUserID: 3,
			toUserID:   99,
			expected:   "1,2,99",
		},
		{
			name:       "单个审批人替换",
			approvers:  "1",
			fromUserID: 1,
			toUserID:   99,
			expected:   "99",
		},
		{
			name:       "审批人不存在不变",
			approvers:  "1,2,3",
			fromUserID: 100,
			toUserID:   99,
			expected:   "1,2,3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := executor.replaceApprover(tt.approvers, tt.fromUserID, tt.toUserID)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// ============================================================================
// 边界条件测试
// ============================================================================

func TestCheckComplete_EdgeCases(t *testing.T) {
	executor := &NodeExecutor{}

	tests := []struct {
		name          string
		nodeInstance  *models.ApprovalNodeInstance
		wantCompleted bool
		wantStatus    string
	}{
		{
			name: "count模式_需要人数等于总人数_全部通过",
			nodeInstance: &models.ApprovalNodeInstance{
				ApproveMode:   "count",
				Approvers:     "1,2,3",
				ApproveCount:  3,
				ApprovedCount: 3,
				RejectedCount: 0,
			},
			wantCompleted: true,
			wantStatus:    "approved",
		},
		{
			name: "count模式_需要1人_等同any",
			nodeInstance: &models.ApprovalNodeInstance{
				ApproveMode:   "count",
				Approvers:     "1,2,3",
				ApproveCount:  1,
				ApprovedCount: 1,
				RejectedCount: 0,
			},
			wantCompleted: true,
			wantStatus:    "approved",
		},
		{
			name: "all模式_只有一人且拒绝",
			nodeInstance: &models.ApprovalNodeInstance{
				ApproveMode:   "all",
				Approvers:     "1",
				ApprovedCount: 0,
				RejectedCount: 1,
			},
			wantCompleted: true,
			wantStatus:    "rejected",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			completed, status := executor.CheckComplete(tt.nodeInstance)
			assert.Equal(t, tt.wantCompleted, completed)
			assert.Equal(t, tt.wantStatus, status)
		})
	}
}
