package approval

import (
	"context"
	"log"
	"sync"
	"time"

	"devops/internal/models"
	"devops/internal/repository"
)

// TimeoutHandler 超时处理器
type TimeoutHandler struct {
	nodeInstanceRepo *repository.ApprovalNodeInstanceRepository
	instanceRepo     *repository.ApprovalInstanceRepository
	nodeExecutor     *NodeExecutor
	stopChan         chan struct{}
	interval         time.Duration
	stopped          bool
	mu               sync.Mutex
}

// NewTimeoutHandler 创建超时处理器
func NewTimeoutHandler(
	nodeInstanceRepo *repository.ApprovalNodeInstanceRepository,
	instanceRepo *repository.ApprovalInstanceRepository,
	nodeExecutor *NodeExecutor,
) *TimeoutHandler {
	return &TimeoutHandler{
		nodeInstanceRepo: nodeInstanceRepo,
		instanceRepo:     instanceRepo,
		nodeExecutor:     nodeExecutor,
		stopChan:         make(chan struct{}),
		interval:         time.Minute, // 每分钟检查一次
	}
}

// Start 启动超时检查定时任务
func (h *TimeoutHandler) Start() {
	go func() {
		ticker := time.NewTicker(h.interval)
		defer ticker.Stop()

		log.Println("[TimeoutHandler] 超时检查器已启动")

		for {
			select {
			case <-ticker.C:
				h.checkTimeouts()
			case <-h.stopChan:
				log.Println("[TimeoutHandler] 超时检查器已停止")
				return
			}
		}
	}()
}

// Stop 停止超时检查（防止重复关闭 channel）
func (h *TimeoutHandler) Stop() {
	h.mu.Lock()
	defer h.mu.Unlock()
	if !h.stopped {
		h.stopped = true
		close(h.stopChan)
	}
}

// checkTimeouts 检查并处理超时节点
func (h *TimeoutHandler) checkTimeouts() {
	ctx := context.Background()

	// 获取所有超时的节点实例
	timeoutNodes, err := h.nodeInstanceRepo.GetTimeoutNodes(ctx)
	if err != nil {
		log.Printf("[TimeoutHandler] 获取超时节点失败: %v", err)
		return
	}

	if len(timeoutNodes) == 0 {
		return
	}

	log.Printf("[TimeoutHandler] 发现 %d 个超时节点", len(timeoutNodes))

	for _, node := range timeoutNodes {
		if err := h.handleTimeout(ctx, &node); err != nil {
			log.Printf("[TimeoutHandler] 处理超时节点 %d 失败: %v", node.ID, err)
		}
	}
}

// handleTimeout 处理单个超时节点
func (h *TimeoutHandler) handleTimeout(ctx context.Context, node *models.ApprovalNodeInstance) error {
	log.Printf("[TimeoutHandler] 处理超时节点: ID=%d, NodeName=%s, TimeoutAction=%s",
		node.ID, node.NodeName, node.TimeoutAction)

	now := time.Now()

	switch node.TimeoutAction {
	case "auto_approve":
		// 自动通过
		if err := h.nodeInstanceRepo.UpdateStatus(ctx, node.ID, "approved", &now); err != nil {
			return err
		}
		// 推进到下一个节点
		return h.advanceAfterTimeout(ctx, node.InstanceID)

	case "auto_reject":
		// 自动拒绝
		if err := h.nodeInstanceRepo.UpdateStatus(ctx, node.ID, "rejected", &now); err != nil {
			return err
		}
		// 拒绝整个实例
		return h.instanceRepo.UpdateStatus(ctx, node.InstanceID, "rejected", &now)

	case "auto_cancel":
		// 自动取消
		if err := h.nodeInstanceRepo.UpdateStatus(ctx, node.ID, "timeout", &now); err != nil {
			return err
		}
		return h.instanceRepo.UpdateStatus(ctx, node.InstanceID, "cancelled", &now)

	default:
		// 默认标记为超时，不做其他处理
		return h.nodeInstanceRepo.UpdateStatus(ctx, node.ID, "timeout", &now)
	}
}

// advanceAfterTimeout 超时自动通过后推进到下一节点
func (h *TimeoutHandler) advanceAfterTimeout(ctx context.Context, instanceID uint) error {
	instance, err := h.instanceRepo.GetWithNodeInstances(ctx, instanceID)
	if err != nil {
		return err
	}

	// 找到下一个节点
	nextOrder := instance.CurrentNodeOrder + 1
	var nextNodeInstance *models.ApprovalNodeInstance
	for i := range instance.NodeInstances {
		if instance.NodeInstances[i].NodeOrder == nextOrder {
			nextNodeInstance = &instance.NodeInstances[i]
			break
		}
	}

	if nextNodeInstance == nil {
		// 没有下一个节点，审批完成
		now := time.Now()
		return h.instanceRepo.UpdateStatus(ctx, instanceID, "approved", &now)
	}

	// 更新当前节点顺序
	if err := h.instanceRepo.UpdateCurrentNode(ctx, instanceID, nextOrder); err != nil {
		return err
	}

	// 激活下一个节点
	timeoutAt := time.Now().Add(60 * time.Minute)
	return h.nodeInstanceRepo.Activate(ctx, nextNodeInstance.ID, &timeoutAt)
}

// SendTimeoutReminder 发送超时提醒（在超时前发送）
func (h *TimeoutHandler) SendTimeoutReminder(ctx context.Context, reminderMinutes int) error {
	// 获取即将超时的节点（在 reminderMinutes 分钟内超时）
	nodes, err := h.nodeInstanceRepo.GetNearTimeoutNodes(ctx, reminderMinutes)
	if err != nil {
		return err
	}

	for _, node := range nodes {
		// TODO: 调用通知服务发送提醒
		log.Printf("[TimeoutHandler] 发送超时提醒: NodeInstanceID=%d, NodeName=%s, TimeoutAt=%v",
			node.ID, node.NodeName, node.TimeoutAt)
	}

	return nil
}
