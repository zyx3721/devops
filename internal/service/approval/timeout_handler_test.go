package approval

import (
	"context"
	"testing"
	"time"

	"devops/internal/models"
)

// MockNodeInstanceRepo 模拟节点实例仓库
type MockNodeInstanceRepo struct {
	nodes         []models.ApprovalNodeInstance
	updatedStatus map[uint]string
}

func NewMockNodeInstanceRepo() *MockNodeInstanceRepo {
	return &MockNodeInstanceRepo{
		nodes:         make([]models.ApprovalNodeInstance, 0),
		updatedStatus: make(map[uint]string),
	}
}

func (m *MockNodeInstanceRepo) GetTimeoutNodes(ctx context.Context) ([]models.ApprovalNodeInstance, error) {
	now := time.Now()
	var result []models.ApprovalNodeInstance
	for _, node := range m.nodes {
		if node.Status == "active" && node.TimeoutAt != nil && node.TimeoutAt.Before(now) {
			result = append(result, node)
		}
	}
	return result, nil
}

func (m *MockNodeInstanceRepo) UpdateStatus(ctx context.Context, id uint, status string, finishedAt *time.Time) error {
	m.updatedStatus[id] = status
	return nil
}

func (m *MockNodeInstanceRepo) GetNearTimeoutNodes(ctx context.Context, reminderMinutes int) ([]models.ApprovalNodeInstance, error) {
	now := time.Now()
	reminderTime := now.Add(time.Duration(reminderMinutes) * time.Minute)
	var result []models.ApprovalNodeInstance
	for _, node := range m.nodes {
		if node.Status == "active" && node.TimeoutAt != nil {
			if node.TimeoutAt.After(now) && node.TimeoutAt.Before(reminderTime) {
				result = append(result, node)
			}
		}
	}
	return result, nil
}

func (m *MockNodeInstanceRepo) AddNode(node models.ApprovalNodeInstance) {
	m.nodes = append(m.nodes, node)
}

// MockInstanceRepo 模拟审批实例仓库
type MockInstanceRepo struct {
	instances     map[uint]*models.ApprovalInstance
	updatedStatus map[uint]string
}

func NewMockInstanceRepo() *MockInstanceRepo {
	return &MockInstanceRepo{
		instances:     make(map[uint]*models.ApprovalInstance),
		updatedStatus: make(map[uint]string),
	}
}

func (m *MockInstanceRepo) UpdateStatus(ctx context.Context, id uint, status string, finishedAt *time.Time) error {
	m.updatedStatus[id] = status
	return nil
}

func (m *MockInstanceRepo) GetWithNodeInstances(ctx context.Context, id uint) (*models.ApprovalInstance, error) {
	if instance, ok := m.instances[id]; ok {
		return instance, nil
	}
	return nil, nil
}

func (m *MockInstanceRepo) UpdateCurrentNode(ctx context.Context, id uint, nodeOrder int) error {
	if instance, ok := m.instances[id]; ok {
		instance.CurrentNodeOrder = nodeOrder
	}
	return nil
}

// TestGetTimeoutNodes 测试获取超时节点
func TestGetTimeoutNodes(t *testing.T) {
	repo := NewMockNodeInstanceRepo()

	// 添加已超时的节点
	pastTime := time.Now().Add(-10 * time.Minute)
	repo.AddNode(models.ApprovalNodeInstance{
		ID:            1,
		InstanceID:    1,
		Status:        "active",
		TimeoutAt:     &pastTime,
		TimeoutAction: "auto_reject",
	})

	// 添加未超时的节点
	futureTime := time.Now().Add(10 * time.Minute)
	repo.AddNode(models.ApprovalNodeInstance{
		ID:            2,
		InstanceID:    1,
		Status:        "active",
		TimeoutAt:     &futureTime,
		TimeoutAction: "auto_reject",
	})

	// 添加已完成的节点（不应被返回）
	repo.AddNode(models.ApprovalNodeInstance{
		ID:            3,
		InstanceID:    1,
		Status:        "approved",
		TimeoutAt:     &pastTime,
		TimeoutAction: "auto_reject",
	})

	nodes, err := repo.GetTimeoutNodes(context.Background())
	if err != nil {
		t.Fatalf("GetTimeoutNodes failed: %v", err)
	}

	if len(nodes) != 1 {
		t.Errorf("Expected 1 timeout node, got %d", len(nodes))
	}

	if len(nodes) > 0 && nodes[0].ID != 1 {
		t.Errorf("Expected node ID 1, got %d", nodes[0].ID)
	}
}

// TestGetNearTimeoutNodes 测试获取即将超时的节点
func TestGetNearTimeoutNodes(t *testing.T) {
	repo := NewMockNodeInstanceRepo()

	// 添加5分钟后超时的节点（应被返回）
	nearTime := time.Now().Add(5 * time.Minute)
	repo.AddNode(models.ApprovalNodeInstance{
		ID:         1,
		InstanceID: 1,
		Status:     "active",
		TimeoutAt:  &nearTime,
	})

	// 添加30分钟后超时的节点（不应被返回）
	farTime := time.Now().Add(30 * time.Minute)
	repo.AddNode(models.ApprovalNodeInstance{
		ID:         2,
		InstanceID: 1,
		Status:     "active",
		TimeoutAt:  &farTime,
	})

	// 添加已超时的节点（不应被返回）
	pastTime := time.Now().Add(-5 * time.Minute)
	repo.AddNode(models.ApprovalNodeInstance{
		ID:         3,
		InstanceID: 1,
		Status:     "active",
		TimeoutAt:  &pastTime,
	})

	nodes, err := repo.GetNearTimeoutNodes(context.Background(), 10)
	if err != nil {
		t.Fatalf("GetNearTimeoutNodes failed: %v", err)
	}

	if len(nodes) != 1 {
		t.Errorf("Expected 1 near-timeout node, got %d", len(nodes))
	}

	if len(nodes) > 0 && nodes[0].ID != 1 {
		t.Errorf("Expected node ID 1, got %d", nodes[0].ID)
	}
}

// TestTimeoutActionAutoReject 测试自动拒绝超时动作
func TestTimeoutActionAutoReject(t *testing.T) {
	nodeRepo := NewMockNodeInstanceRepo()
	instanceRepo := NewMockInstanceRepo()

	pastTime := time.Now().Add(-10 * time.Minute)
	nodeRepo.AddNode(models.ApprovalNodeInstance{
		ID:            1,
		InstanceID:    100,
		Status:        "active",
		TimeoutAt:     &pastTime,
		TimeoutAction: "auto_reject",
	})

	instanceRepo.instances[100] = &models.ApprovalInstance{
		ID:               100,
		Status:           "pending",
		CurrentNodeOrder: 1,
	}

	// 模拟超时处理
	ctx := context.Background()
	nodes, _ := nodeRepo.GetTimeoutNodes(ctx)

	for _, node := range nodes {
		if node.TimeoutAction == "auto_reject" {
			now := time.Now()
			nodeRepo.UpdateStatus(ctx, node.ID, "rejected", &now)
			instanceRepo.UpdateStatus(ctx, node.InstanceID, "rejected", &now)
		}
	}

	// 验证状态更新
	if nodeRepo.updatedStatus[1] != "rejected" {
		t.Errorf("Expected node status 'rejected', got '%s'", nodeRepo.updatedStatus[1])
	}

	if instanceRepo.updatedStatus[100] != "rejected" {
		t.Errorf("Expected instance status 'rejected', got '%s'", instanceRepo.updatedStatus[100])
	}
}

// TestTimeoutActionAutoApprove 测试自动通过超时动作
func TestTimeoutActionAutoApprove(t *testing.T) {
	nodeRepo := NewMockNodeInstanceRepo()
	instanceRepo := NewMockInstanceRepo()

	pastTime := time.Now().Add(-10 * time.Minute)
	nodeRepo.AddNode(models.ApprovalNodeInstance{
		ID:            1,
		InstanceID:    100,
		NodeOrder:     1,
		Status:        "active",
		TimeoutAt:     &pastTime,
		TimeoutAction: "auto_approve",
	})

	// 只有一个节点的实例
	instanceRepo.instances[100] = &models.ApprovalInstance{
		ID:               100,
		Status:           "pending",
		CurrentNodeOrder: 1,
		NodeInstances: []models.ApprovalNodeInstance{
			{ID: 1, NodeOrder: 1, Status: "active"},
		},
	}

	// 模拟超时处理
	ctx := context.Background()
	nodes, _ := nodeRepo.GetTimeoutNodes(ctx)

	for _, node := range nodes {
		if node.TimeoutAction == "auto_approve" {
			now := time.Now()
			nodeRepo.UpdateStatus(ctx, node.ID, "approved", &now)

			// 检查是否有下一个节点
			instance := instanceRepo.instances[node.InstanceID]
			hasNextNode := false
			for _, ni := range instance.NodeInstances {
				if ni.NodeOrder > node.NodeOrder {
					hasNextNode = true
					break
				}
			}

			if !hasNextNode {
				// 没有下一个节点，审批完成
				instanceRepo.UpdateStatus(ctx, node.InstanceID, "approved", &now)
			}
		}
	}

	// 验证状态更新
	if nodeRepo.updatedStatus[1] != "approved" {
		t.Errorf("Expected node status 'approved', got '%s'", nodeRepo.updatedStatus[1])
	}

	if instanceRepo.updatedStatus[100] != "approved" {
		t.Errorf("Expected instance status 'approved', got '%s'", instanceRepo.updatedStatus[100])
	}
}

// TestTimeoutActionAutoCancel 测试自动取消超时动作
func TestTimeoutActionAutoCancel(t *testing.T) {
	nodeRepo := NewMockNodeInstanceRepo()
	instanceRepo := NewMockInstanceRepo()

	pastTime := time.Now().Add(-10 * time.Minute)
	nodeRepo.AddNode(models.ApprovalNodeInstance{
		ID:            1,
		InstanceID:    100,
		Status:        "active",
		TimeoutAt:     &pastTime,
		TimeoutAction: "auto_cancel",
	})

	instanceRepo.instances[100] = &models.ApprovalInstance{
		ID:               100,
		Status:           "pending",
		CurrentNodeOrder: 1,
	}

	// 模拟超时处理
	ctx := context.Background()
	nodes, _ := nodeRepo.GetTimeoutNodes(ctx)

	for _, node := range nodes {
		if node.TimeoutAction == "auto_cancel" {
			now := time.Now()
			nodeRepo.UpdateStatus(ctx, node.ID, "timeout", &now)
			instanceRepo.UpdateStatus(ctx, node.InstanceID, "cancelled", &now)
		}
	}

	// 验证状态更新
	if nodeRepo.updatedStatus[1] != "timeout" {
		t.Errorf("Expected node status 'timeout', got '%s'", nodeRepo.updatedStatus[1])
	}

	if instanceRepo.updatedStatus[100] != "cancelled" {
		t.Errorf("Expected instance status 'cancelled', got '%s'", instanceRepo.updatedStatus[100])
	}
}
