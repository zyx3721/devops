package pipeline

import (
	"container/heap"
	"context"
	"fmt"
	"sync"
	"time"

	"devops/pkg/logger"
)

// ConcurrencyService Job 并发控制服务
type ConcurrencyService struct {
	maxConcurrent int
	currentCount  int
	queue         *JobQueue
	mu            sync.Mutex
	cond          *sync.Cond
	running       map[uint]bool
	metrics       *ConcurrencyMetrics
}

// ConcurrencyMetrics 并发指标
type ConcurrencyMetrics struct {
	mu              sync.RWMutex
	TotalQueued     int64
	TotalProcessed  int64
	TotalTimeout    int64
	CurrentRunning  int
	CurrentQueued   int
	AvgWaitTime     float64
	MaxWaitTime     time.Duration
	totalWaitTime   time.Duration
	waitTimeCount   int64
}

// JobItem 队列中的任务项
type JobItem struct {
	RunID     uint
	Priority  int // 优先级，数值越小优先级越高
	QueuedAt  time.Time
	Timeout   time.Duration
	index     int
}

// JobQueue 优先级队列
type JobQueue []*JobItem

func (pq JobQueue) Len() int { return len(pq) }

func (pq JobQueue) Less(i, j int) bool {
	// 优先级相同时，先入队的优先
	if pq[i].Priority == pq[j].Priority {
		return pq[i].QueuedAt.Before(pq[j].QueuedAt)
	}
	return pq[i].Priority < pq[j].Priority
}

func (pq JobQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].index = i
	pq[j].index = j
}

func (pq *JobQueue) Push(x interface{}) {
	n := len(*pq)
	item := x.(*JobItem)
	item.index = n
	*pq = append(*pq, item)
}

func (pq *JobQueue) Pop() interface{} {
	old := *pq
	n := len(old)
	item := old[n-1]
	old[n-1] = nil
	item.index = -1
	*pq = old[0 : n-1]
	return item
}

// NewConcurrencyService 创建并发控制服务
func NewConcurrencyService(maxConcurrent int) *ConcurrencyService {
	if maxConcurrent <= 0 {
		maxConcurrent = 10 // 默认最大并发数
	}

	s := &ConcurrencyService{
		maxConcurrent: maxConcurrent,
		queue:         &JobQueue{},
		running:       make(map[uint]bool),
		metrics:       &ConcurrencyMetrics{},
	}
	s.cond = sync.NewCond(&s.mu)
	heap.Init(s.queue)

	return s
}

// Acquire 获取执行许可
func (s *ConcurrencyService) Acquire(ctx context.Context, runID uint, priority int, timeout time.Duration) error {
	log := logger.L().WithField("run_id", runID).WithField("priority", priority)
	log.Info("请求执行许可")

	s.mu.Lock()

	// 检查是否已在运行
	if s.running[runID] {
		s.mu.Unlock()
		return fmt.Errorf("任务 %d 已在运行中", runID)
	}

	// 如果有空闲槽位，直接获取
	if s.currentCount < s.maxConcurrent {
		s.currentCount++
		s.running[runID] = true
		s.metrics.CurrentRunning = s.currentCount
		s.mu.Unlock()
		log.Info("直接获取执行许可")
		return nil
	}

	// 加入等待队列
	item := &JobItem{
		RunID:    runID,
		Priority: priority,
		QueuedAt: time.Now(),
		Timeout:  timeout,
	}
	heap.Push(s.queue, item)
	s.metrics.TotalQueued++
	s.metrics.CurrentQueued = s.queue.Len()

	log.WithField("queue_size", s.queue.Len()).Info("加入等待队列")

	// 设置超时
	var timeoutCh <-chan time.Time
	if timeout > 0 {
		timeoutCh = time.After(timeout)
	}

	// 等待获取许可
	for {
		s.mu.Unlock()

		select {
		case <-ctx.Done():
			s.removeFromQueue(runID)
			return ctx.Err()
		case <-timeoutCh:
			s.removeFromQueue(runID)
			s.metrics.mu.Lock()
			s.metrics.TotalTimeout++
			s.metrics.mu.Unlock()
			return fmt.Errorf("等待执行许可超时")
		default:
		}

		s.mu.Lock()

		// 检查是否轮到自己
		if s.currentCount < s.maxConcurrent && s.queue.Len() > 0 {
			next := (*s.queue)[0]
			if next.RunID == runID {
				heap.Pop(s.queue)
				s.currentCount++
				s.running[runID] = true
				s.metrics.CurrentRunning = s.currentCount
				s.metrics.CurrentQueued = s.queue.Len()

				// 记录等待时间
				waitTime := time.Since(item.QueuedAt)
				s.recordWaitTime(waitTime)

				s.mu.Unlock()
				log.WithField("wait_time", waitTime).Info("获取执行许可")
				return nil
			}
		}

		// 等待条件变量
		s.cond.Wait()
	}
}

// Release 释放执行许可
func (s *ConcurrencyService) Release(runID uint) {
	log := logger.L().WithField("run_id", runID)

	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running[runID] {
		log.Warn("任务未在运行中")
		return
	}

	delete(s.running, runID)
	s.currentCount--
	s.metrics.CurrentRunning = s.currentCount
	s.metrics.TotalProcessed++

	log.Info("释放执行许可")

	// 通知等待的任务
	s.cond.Broadcast()
}

// removeFromQueue 从队列中移除任务
func (s *ConcurrencyService) removeFromQueue(runID uint) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, item := range *s.queue {
		if item.RunID == runID {
			heap.Remove(s.queue, i)
			s.metrics.CurrentQueued = s.queue.Len()
			break
		}
	}
}

// recordWaitTime 记录等待时间
func (s *ConcurrencyService) recordWaitTime(waitTime time.Duration) {
	s.metrics.mu.Lock()
	defer s.metrics.mu.Unlock()

	s.metrics.totalWaitTime += waitTime
	s.metrics.waitTimeCount++
	s.metrics.AvgWaitTime = float64(s.metrics.totalWaitTime) / float64(s.metrics.waitTimeCount) / float64(time.Millisecond)

	if waitTime > s.metrics.MaxWaitTime {
		s.metrics.MaxWaitTime = waitTime
	}
}

// GetMetrics 获取指标
func (s *ConcurrencyService) GetMetrics() *ConcurrencyMetrics {
	s.metrics.mu.RLock()
	defer s.metrics.mu.RUnlock()

	return &ConcurrencyMetrics{
		TotalQueued:    s.metrics.TotalQueued,
		TotalProcessed: s.metrics.TotalProcessed,
		TotalTimeout:   s.metrics.TotalTimeout,
		CurrentRunning: s.metrics.CurrentRunning,
		CurrentQueued:  s.metrics.CurrentQueued,
		AvgWaitTime:    s.metrics.AvgWaitTime,
		MaxWaitTime:    s.metrics.MaxWaitTime,
	}
}

// GetQueueStatus 获取队列状态
func (s *ConcurrencyService) GetQueueStatus() *QueueStatus {
	s.mu.Lock()
	defer s.mu.Unlock()

	items := make([]QueueItemStatus, 0, s.queue.Len())
	for _, item := range *s.queue {
		items = append(items, QueueItemStatus{
			RunID:    item.RunID,
			Priority: item.Priority,
			QueuedAt: item.QueuedAt,
			WaitTime: time.Since(item.QueuedAt),
		})
	}

	return &QueueStatus{
		MaxConcurrent:  s.maxConcurrent,
		CurrentRunning: s.currentCount,
		QueueLength:    s.queue.Len(),
		Items:          items,
	}
}

// QueueStatus 队列状态
type QueueStatus struct {
	MaxConcurrent  int               `json:"max_concurrent"`
	CurrentRunning int               `json:"current_running"`
	QueueLength    int               `json:"queue_length"`
	Items          []QueueItemStatus `json:"items"`
}

// QueueItemStatus 队列项状态
type QueueItemStatus struct {
	RunID    uint          `json:"run_id"`
	Priority int           `json:"priority"`
	QueuedAt time.Time     `json:"queued_at"`
	WaitTime time.Duration `json:"wait_time"`
}

// SetMaxConcurrent 设置最大并发数
func (s *ConcurrencyService) SetMaxConcurrent(max int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if max > 0 {
		s.maxConcurrent = max
		// 如果增加了并发数，通知等待的任务
		s.cond.Broadcast()
	}
}

// IsRunning 检查任务是否在运行
func (s *ConcurrencyService) IsRunning(runID uint) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.running[runID]
}

// GetRunningCount 获取当前运行数
func (s *ConcurrencyService) GetRunningCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.currentCount
}

// GetQueueLength 获取队列长度
func (s *ConcurrencyService) GetQueueLength() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.queue.Len()
}
