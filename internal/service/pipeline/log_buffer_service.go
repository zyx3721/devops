package pipeline

import (
	"context"
	"sync"
	"time"

	"gorm.io/gorm"

	"devops/internal/models"
	"devops/pkg/logger"
)

// LogBufferService 日志缓冲服务
type LogBufferService struct {
	db            *gorm.DB
	buffers       map[uint]*LogBuffer
	mu            sync.RWMutex
	flushInterval time.Duration
	maxBufferSize int
	stopCh        chan struct{}
	wg            sync.WaitGroup
}

// LogBuffer 单个步骤的日志缓冲
type LogBuffer struct {
	StepRunID uint
	Lines     []string
	mu        sync.Mutex
	lastFlush time.Time
}

// NewLogBufferService 创建日志缓冲服务
func NewLogBufferService(db *gorm.DB) *LogBufferService {
	s := &LogBufferService{
		db:            db,
		buffers:       make(map[uint]*LogBuffer),
		flushInterval: 2 * time.Second, // 默认2秒刷新一次
		maxBufferSize: 100,             // 默认缓冲100行
		stopCh:        make(chan struct{}),
	}

	// 启动后台刷新协程
	s.wg.Add(1)
	go s.backgroundFlush()

	return s
}

// SetFlushInterval 设置刷新间隔
func (s *LogBufferService) SetFlushInterval(interval time.Duration) {
	s.flushInterval = interval
}

// SetMaxBufferSize 设置最大缓冲大小
func (s *LogBufferService) SetMaxBufferSize(size int) {
	s.maxBufferSize = size
}

// AppendLog 追加日志
func (s *LogBufferService) AppendLog(stepRunID uint, line string) {
	s.mu.Lock()
	buffer, exists := s.buffers[stepRunID]
	if !exists {
		buffer = &LogBuffer{
			StepRunID: stepRunID,
			Lines:     make([]string, 0, s.maxBufferSize),
			lastFlush: time.Now(),
		}
		s.buffers[stepRunID] = buffer
	}
	s.mu.Unlock()

	buffer.mu.Lock()
	buffer.Lines = append(buffer.Lines, line)
	shouldFlush := len(buffer.Lines) >= s.maxBufferSize
	buffer.mu.Unlock()

	// 如果缓冲区满了，立即刷新
	if shouldFlush {
		s.FlushBuffer(stepRunID)
	}
}

// AppendLogs 批量追加日志
func (s *LogBufferService) AppendLogs(stepRunID uint, lines []string) {
	s.mu.Lock()
	buffer, exists := s.buffers[stepRunID]
	if !exists {
		buffer = &LogBuffer{
			StepRunID: stepRunID,
			Lines:     make([]string, 0, s.maxBufferSize),
			lastFlush: time.Now(),
		}
		s.buffers[stepRunID] = buffer
	}
	s.mu.Unlock()

	buffer.mu.Lock()
	buffer.Lines = append(buffer.Lines, lines...)
	shouldFlush := len(buffer.Lines) >= s.maxBufferSize
	buffer.mu.Unlock()

	if shouldFlush {
		s.FlushBuffer(stepRunID)
	}
}

// FlushBuffer 刷新指定步骤的缓冲区
func (s *LogBufferService) FlushBuffer(stepRunID uint) {
	s.mu.RLock()
	buffer, exists := s.buffers[stepRunID]
	s.mu.RUnlock()

	if !exists {
		return
	}

	buffer.mu.Lock()
	if len(buffer.Lines) == 0 {
		buffer.mu.Unlock()
		return
	}

	// 复制并清空缓冲区
	lines := make([]string, len(buffer.Lines))
	copy(lines, buffer.Lines)
	buffer.Lines = buffer.Lines[:0]
	buffer.lastFlush = time.Now()
	buffer.mu.Unlock()

	// 批量写入数据库
	s.writeToDB(stepRunID, lines)
}

// FlushAll 刷新所有缓冲区
func (s *LogBufferService) FlushAll() {
	s.mu.RLock()
	stepRunIDs := make([]uint, 0, len(s.buffers))
	for id := range s.buffers {
		stepRunIDs = append(stepRunIDs, id)
	}
	s.mu.RUnlock()

	for _, id := range stepRunIDs {
		s.FlushBuffer(id)
	}
}

// CloseBuffer 关闭并刷新指定步骤的缓冲区
func (s *LogBufferService) CloseBuffer(stepRunID uint) {
	// 先刷新
	s.FlushBuffer(stepRunID)

	// 移除缓冲区
	s.mu.Lock()
	delete(s.buffers, stepRunID)
	s.mu.Unlock()
}

// Stop 停止服务
func (s *LogBufferService) Stop() {
	close(s.stopCh)
	s.wg.Wait()

	// 刷新所有剩余缓冲
	s.FlushAll()
}

// backgroundFlush 后台定时刷新
func (s *LogBufferService) backgroundFlush() {
	defer s.wg.Done()

	ticker := time.NewTicker(s.flushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopCh:
			return
		case <-ticker.C:
			s.flushExpired()
		}
	}
}

// flushExpired 刷新过期的缓冲区
func (s *LogBufferService) flushExpired() {
	s.mu.RLock()
	expiredIDs := make([]uint, 0)
	now := time.Now()

	for id, buffer := range s.buffers {
		buffer.mu.Lock()
		if len(buffer.Lines) > 0 && now.Sub(buffer.lastFlush) >= s.flushInterval {
			expiredIDs = append(expiredIDs, id)
		}
		buffer.mu.Unlock()
	}
	s.mu.RUnlock()

	for _, id := range expiredIDs {
		s.FlushBuffer(id)
	}
}

// writeToDB 写入数据库
func (s *LogBufferService) writeToDB(stepRunID uint, lines []string) {
	if len(lines) == 0 {
		return
	}

	// 检查 db 是否为 nil
	if s.db == nil {
		return
	}

	log := logger.L().WithField("step_run_id", stepRunID).WithField("lines", len(lines))

	// 合并日志行
	var content string
	for _, line := range lines {
		content += line + "\n"
	}

	// 追加到数据库
	err := s.db.Model(&models.StepRun{}).
		Where("id = ?", stepRunID).
		Update("logs", gorm.Expr("CONCAT(COALESCE(logs, ''), ?)", content)).Error

	if err != nil {
		log.WithField("error", err).Error("写入日志失败")
	} else {
		log.Debug("日志写入成功")
	}
}

// GetBufferStats 获取缓冲区统计
func (s *LogBufferService) GetBufferStats() *BufferStats {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stats := &BufferStats{
		BufferCount: len(s.buffers),
		Buffers:     make([]BufferInfo, 0, len(s.buffers)),
	}

	for id, buffer := range s.buffers {
		buffer.mu.Lock()
		stats.TotalLines += len(buffer.Lines)
		stats.Buffers = append(stats.Buffers, BufferInfo{
			StepRunID:     id,
			LineCount:     len(buffer.Lines),
			LastFlushTime: buffer.lastFlush,
		})
		buffer.mu.Unlock()
	}

	return stats
}

// BufferStats 缓冲区统计
type BufferStats struct {
	BufferCount int          `json:"buffer_count"`
	TotalLines  int          `json:"total_lines"`
	Buffers     []BufferInfo `json:"buffers"`
}

// BufferInfo 单个缓冲区信息
type BufferInfo struct {
	StepRunID     uint      `json:"step_run_id"`
	LineCount     int       `json:"line_count"`
	LastFlushTime time.Time `json:"last_flush_time"`
}

// LogBufferWriter 实现 io.Writer 接口的日志写入器
type LogBufferWriter struct {
	service   *LogBufferService
	stepRunID uint
	buffer    []byte
}

// NewLogBufferWriter 创建日志写入器
func NewLogBufferWriter(service *LogBufferService, stepRunID uint) *LogBufferWriter {
	return &LogBufferWriter{
		service:   service,
		stepRunID: stepRunID,
		buffer:    make([]byte, 0, 4096),
	}
}

// Write 实现 io.Writer 接口
func (w *LogBufferWriter) Write(p []byte) (n int, err error) {
	w.buffer = append(w.buffer, p...)

	// 按行分割
	for {
		idx := -1
		for i, b := range w.buffer {
			if b == '\n' {
				idx = i
				break
			}
		}

		if idx == -1 {
			break
		}

		line := string(w.buffer[:idx])
		w.buffer = w.buffer[idx+1:]
		w.service.AppendLog(w.stepRunID, line)
	}

	return len(p), nil
}

// Flush 刷新剩余内容
func (w *LogBufferWriter) Flush() {
	if len(w.buffer) > 0 {
		w.service.AppendLog(w.stepRunID, string(w.buffer))
		w.buffer = w.buffer[:0]
	}
	w.service.FlushBuffer(w.stepRunID)
}

// Close 关闭写入器
func (w *LogBufferWriter) Close() error {
	w.Flush()
	w.service.CloseBuffer(w.stepRunID)
	return nil
}

// BatchLogWriter 批量日志写入器（用于高频写入场景）
type BatchLogWriter struct {
	service     *LogBufferService
	stepRunID   uint
	batchSize   int
	lines       []string
	mu          sync.Mutex
	flushTicker *time.Ticker
	stopCh      chan struct{}
	closed      bool
}

// NewBatchLogWriter 创建批量日志写入器
func NewBatchLogWriter(service *LogBufferService, stepRunID uint, batchSize int) *BatchLogWriter {
	w := &BatchLogWriter{
		service:     service,
		stepRunID:   stepRunID,
		batchSize:   batchSize,
		lines:       make([]string, 0, batchSize),
		flushTicker: time.NewTicker(500 * time.Millisecond),
		stopCh:      make(chan struct{}),
	}

	go w.autoFlush()
	return w
}

// WriteLine 写入一行日志
func (w *BatchLogWriter) WriteLine(line string) {
	w.mu.Lock()
	w.lines = append(w.lines, line)
	shouldFlush := len(w.lines) >= w.batchSize
	w.mu.Unlock()

	if shouldFlush {
		w.Flush()
	}
}

// Flush 刷新缓冲
func (w *BatchLogWriter) Flush() {
	w.mu.Lock()
	if len(w.lines) == 0 {
		w.mu.Unlock()
		return
	}

	lines := make([]string, len(w.lines))
	copy(lines, w.lines)
	w.lines = w.lines[:0]
	w.mu.Unlock()

	w.service.AppendLogs(w.stepRunID, lines)
}

// autoFlush 自动刷新
func (w *BatchLogWriter) autoFlush() {
	for {
		select {
		case <-w.stopCh:
			return
		case <-w.flushTicker.C:
			w.Flush()
		}
	}
}

// Close 关闭写入器（防止重复关闭）
func (w *BatchLogWriter) Close() error {
	w.mu.Lock()
	if w.closed {
		w.mu.Unlock()
		return nil
	}
	w.closed = true
	w.mu.Unlock()

	close(w.stopCh)
	w.flushTicker.Stop()
	w.Flush()
	w.service.CloseBuffer(w.stepRunID)
	return nil
}

// CreateLogBufferServiceWithContext 创建带上下文的日志缓冲服务
func CreateLogBufferServiceWithContext(ctx context.Context, db *gorm.DB) *LogBufferService {
	s := NewLogBufferService(db)

	// 监听上下文取消
	go func() {
		<-ctx.Done()
		s.Stop()
	}()

	return s
}
