package logs

import (
	"context"
	"encoding/json"
	"sort"
	"sync"
	"time"

	"devops/pkg/dto"
	"devops/pkg/logger"

	"github.com/gorilla/websocket"
)

// StreamService 日志流服务
type StreamService struct {
	adapter     *K8sLogAdapter
	connections sync.Map // map[string]*StreamConnection
}

// StreamConnection WebSocket 连接
type StreamConnection struct {
	ID        string
	Conn      *websocket.Conn
	Request   *dto.LogStreamRequest
	Cancel    context.CancelFunc
	IsPaused  bool
	mu        sync.Mutex
	sendCh    chan dto.LogStreamMessage
	closeCh   chan struct{}
	closeOnce sync.Once
}

// MultiPodStreamConnection 多 Pod 日志流连接
type MultiPodStreamConnection struct {
	ID         string
	Conn       *websocket.Conn
	Request    *dto.MultiPodLogStreamRequest
	Cancel     context.CancelFunc
	IsPaused   bool
	mu         sync.Mutex
	sendCh     chan dto.LogStreamMessage
	closeCh    chan struct{}
	closeOnce  sync.Once
	podCancels map[string]context.CancelFunc
}

// NewStreamService 创建日志流服务
func NewStreamService(adapter *K8sLogAdapter) *StreamService {
	return &StreamService{
		adapter: adapter,
	}
}

// NewStreamConnection 创建新的流连接
func NewStreamConnection(id string, conn *websocket.Conn, req *dto.LogStreamRequest) *StreamConnection {
	return &StreamConnection{
		ID:       id,
		Conn:     conn,
		Request:  req,
		IsPaused: false,
		sendCh:   make(chan dto.LogStreamMessage, 1000),
		closeCh:  make(chan struct{}),
	}
}

// HandleConnection 处理 WebSocket 连接
func (s *StreamService) HandleConnection(ctx context.Context, conn *StreamConnection) error {
	// 保存连接
	s.connections.Store(conn.ID, conn)
	defer s.connections.Delete(conn.ID)

	// 创建可取消的上下文
	streamCtx, cancel := context.WithCancel(ctx)
	conn.Cancel = cancel
	defer cancel()

	// 发送连接成功消息
	conn.Send(dto.LogStreamMessage{
		Type:      "connected",
		Timestamp: time.Now().Format(time.RFC3339),
		Content:   "日志流连接成功",
	})

	// 启动发送协程
	go conn.startSender()

	// 启动接收协程（处理客户端消息）
	go s.handleClientMessages(conn)

	// 启动日志流
	logCh := make(chan dto.LogEntry, 1000)

	go func() {
		defer close(logCh)
		err := s.adapter.StreamLogs(streamCtx, conn.Request, logCh)
		if err != nil {
			// 发送错误消息但不断开连接
			conn.Send(dto.LogStreamMessage{
				Type:      "error",
				Timestamp: time.Now().Format(time.RFC3339),
				Content:   err.Error(),
			})
		}
	}()

	// 处理日志
	for {
		select {
		case <-streamCtx.Done():
			conn.Send(dto.LogStreamMessage{
				Type:      "disconnected",
				Timestamp: time.Now().Format(time.RFC3339),
				Content:   "连接已关闭",
			})
			return nil

		case <-conn.closeCh:
			return nil

		case entry, ok := <-logCh:
			if !ok {
				// 日志流结束，发送完成消息
				conn.Send(dto.LogStreamMessage{
					Type:      "completed",
					Timestamp: time.Now().Format(time.RFC3339),
					Content:   "日志流已结束",
				})
				return nil
			}
			if !conn.IsPaused {
				conn.Send(dto.LogStreamMessage{
					Type:      "log",
					Timestamp: entry.Timestamp,
					Content:   entry.Content,
					PodName:   entry.PodName,
					Container: entry.Container,
					Level:     entry.Level,
				})
			}
		}
	}
}

// handleClientMessages 处理客户端消息
func (s *StreamService) handleClientMessages(conn *StreamConnection) {
	defer conn.Close()

	for {
		_, message, err := conn.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				logger.L().Error("WebSocket读取错误: %v", err)
			}
			return
		}

		var cmd ClientCommand
		if err := json.Unmarshal(message, &cmd); err != nil {
			continue
		}

		switch cmd.Action {
		case "pause":
			conn.Pause()
		case "resume":
			conn.Resume()
		case "close":
			conn.Cancel()
			return
		}
	}
}

// ClientCommand 客户端命令
type ClientCommand struct {
	Action string `json:"action"` // pause/resume/close
}

// Send 发送消息
func (c *StreamConnection) Send(msg dto.LogStreamMessage) {
	select {
	case c.sendCh <- msg:
	default:
		// 通道满了，丢弃消息
		logger.L().Warn("日志流通道已满，丢弃消息")
	}
}

// startSender 启动发送协程
func (c *StreamConnection) startSender() {
	for {
		select {
		case <-c.closeCh:
			return
		case msg := <-c.sendCh:
			c.mu.Lock()
			err := c.Conn.WriteJSON(msg)
			c.mu.Unlock()
			if err != nil {
				logger.L().Error("WebSocket发送错误: %v", err)
				return
			}
		}
	}
}

// Pause 暂停日志流
func (c *StreamConnection) Pause() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.IsPaused = true
}

// Resume 恢复日志流
func (c *StreamConnection) Resume() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.IsPaused = false
}

// Close 关闭连接
func (c *StreamConnection) Close() {
	c.closeOnce.Do(func() {
		close(c.closeCh)
		if c.Cancel != nil {
			c.Cancel()
		}
		c.Conn.Close()
	})
}

// GetConnection 获取连接
func (s *StreamService) GetConnection(id string) *StreamConnection {
	if conn, ok := s.connections.Load(id); ok {
		return conn.(*StreamConnection)
	}
	return nil
}

// CloseConnection 关闭指定连接
func (s *StreamService) CloseConnection(id string) {
	if conn := s.GetConnection(id); conn != nil {
		conn.Close()
		s.connections.Delete(id)
	}
}

// GetActiveConnections 获取活跃连接数
func (s *StreamService) GetActiveConnections() int {
	count := 0
	s.connections.Range(func(_, _ interface{}) bool {
		count++
		return true
	})
	return count
}

// NewMultiPodStreamConnection 创建多 Pod 流连接
func NewMultiPodStreamConnection(id string, conn *websocket.Conn, req *dto.MultiPodLogStreamRequest) *MultiPodStreamConnection {
	return &MultiPodStreamConnection{
		ID:         id,
		Conn:       conn,
		Request:    req,
		IsPaused:   false,
		sendCh:     make(chan dto.LogStreamMessage, 2000),
		closeCh:    make(chan struct{}),
		podCancels: make(map[string]context.CancelFunc),
	}
}

// HandleMultiPodConnection 处理多 Pod WebSocket 连接
func (s *StreamService) HandleMultiPodConnection(ctx context.Context, conn *MultiPodStreamConnection) error {
	// 保存连接
	s.connections.Store(conn.ID, conn)
	defer s.connections.Delete(conn.ID)

	// 创建可取消的上下文
	streamCtx, cancel := context.WithCancel(ctx)
	conn.Cancel = cancel
	defer cancel()

	// 发送连接成功消息
	conn.Send(dto.LogStreamMessage{
		Type:      "connected",
		Timestamp: time.Now().Format(time.RFC3339),
		Content:   "多 Pod 日志流连接成功",
	})

	// 启动发送协程
	go conn.startSender()

	// 启动接收协程（处理客户端消息）
	go s.handleMultiPodClientMessages(conn)

	// 聚合日志通道
	aggregatedCh := make(chan dto.LogEntry, 5000)
	errCh := make(chan error, len(conn.Request.Pods))

	// 为每个 Pod 启动日志流
	var wg sync.WaitGroup
	for _, podReq := range conn.Request.Pods {
		wg.Add(1)
		go func(pr dto.PodLogRequest) {
			defer wg.Done()
			podCtx, podCancel := context.WithCancel(streamCtx)
			conn.mu.Lock()
			conn.podCancels[pr.PodName] = podCancel
			conn.mu.Unlock()

			req := &dto.LogStreamRequest{
				ClusterID: conn.Request.ClusterID,
				Namespace: conn.Request.Namespace,
				PodName:   pr.PodName,
				Container: pr.Container,
				TailLines: conn.Request.TailLines,
				Follow:    true,
			}

			logCh := make(chan dto.LogEntry, 1000)
			go func() {
				err := s.adapter.StreamLogs(podCtx, req, logCh)
				if err != nil {
					select {
					case errCh <- err:
					default:
					}
				}
			}()

			for entry := range logCh {
				select {
				case aggregatedCh <- entry:
				case <-podCtx.Done():
					return
				}
			}
		}(podReq)
	}

	// 等待所有 Pod 流结束
	go func() {
		wg.Wait()
		close(aggregatedCh)
	}()

	// 日志排序缓冲区
	buffer := make([]dto.LogEntry, 0, 100)
	flushTicker := time.NewTicker(100 * time.Millisecond)
	defer flushTicker.Stop()

	// 处理聚合日志
	for {
		select {
		case <-streamCtx.Done():
			conn.Send(dto.LogStreamMessage{
				Type:      "disconnected",
				Timestamp: time.Now().Format(time.RFC3339),
				Content:   "连接已关闭",
			})
			return nil

		case <-conn.closeCh:
			return nil

		case err := <-errCh:
			conn.Send(dto.LogStreamMessage{
				Type:      "error",
				Timestamp: time.Now().Format(time.RFC3339),
				Content:   err.Error(),
			})

		case entry, ok := <-aggregatedCh:
			if !ok {
				// 刷新剩余缓冲区
				s.flushBuffer(conn, buffer)
				return nil
			}
			buffer = append(buffer, entry)
			// 缓冲区满时刷新
			if len(buffer) >= 50 {
				s.flushBuffer(conn, buffer)
				buffer = buffer[:0]
			}

		case <-flushTicker.C:
			// 定时刷新缓冲区
			if len(buffer) > 0 {
				s.flushBuffer(conn, buffer)
				buffer = buffer[:0]
			}
		}
	}
}

// flushBuffer 刷新日志缓冲区（按时间排序后发送）
func (s *StreamService) flushBuffer(conn *MultiPodStreamConnection, buffer []dto.LogEntry) {
	if len(buffer) == 0 || conn.IsPaused {
		return
	}

	// 按时间戳排序
	sort.Slice(buffer, func(i, j int) bool {
		return buffer[i].Timestamp < buffer[j].Timestamp
	})

	// 发送排序后的日志
	for _, entry := range buffer {
		conn.Send(dto.LogStreamMessage{
			Type:      "log",
			Timestamp: entry.Timestamp,
			Content:   entry.Content,
			PodName:   entry.PodName,
			Container: entry.Container,
			Level:     entry.Level,
		})
	}
}

// handleMultiPodClientMessages 处理多 Pod 客户端消息
func (s *StreamService) handleMultiPodClientMessages(conn *MultiPodStreamConnection) {
	defer conn.Close()

	for {
		_, message, err := conn.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				logger.L().Error("WebSocket读取错误: %v", err)
			}
			return
		}

		var cmd MultiPodClientCommand
		if err := json.Unmarshal(message, &cmd); err != nil {
			continue
		}

		switch cmd.Action {
		case "pause":
			conn.Pause()
		case "resume":
			conn.Resume()
		case "close":
			conn.Cancel()
			return
		case "add_pod":
			// 动态添加 Pod
			if cmd.PodName != "" {
				s.addPodToStream(conn, cmd.PodName, cmd.Container)
			}
		case "remove_pod":
			// 动态移除 Pod
			if cmd.PodName != "" {
				conn.RemovePod(cmd.PodName)
			}
		}
	}
}

// MultiPodClientCommand 多 Pod 客户端命令
type MultiPodClientCommand struct {
	Action    string `json:"action"`    // pause/resume/close/add_pod/remove_pod
	PodName   string `json:"podName"`   // 用于 add_pod/remove_pod
	Container string `json:"container"` // 用于 add_pod
}

// addPodToStream 动态添加 Pod 到流
func (s *StreamService) addPodToStream(conn *MultiPodStreamConnection, podName, container string) {
	conn.mu.Lock()
	if _, exists := conn.podCancels[podName]; exists {
		conn.mu.Unlock()
		return // 已存在
	}
	conn.mu.Unlock()

	// 创建新的 Pod 流
	podCtx, podCancel := context.WithCancel(context.Background())
	conn.mu.Lock()
	conn.podCancels[podName] = podCancel
	conn.mu.Unlock()

	go func() {
		req := &dto.LogStreamRequest{
			ClusterID: conn.Request.ClusterID,
			Namespace: conn.Request.Namespace,
			PodName:   podName,
			Container: container,
			TailLines: conn.Request.TailLines,
			Follow:    true,
		}

		logCh := make(chan dto.LogEntry, 1000)
		go func() {
			s.adapter.StreamLogs(podCtx, req, logCh)
		}()

		for entry := range logCh {
			if !conn.IsPaused {
				conn.Send(dto.LogStreamMessage{
					Type:      "log",
					Timestamp: entry.Timestamp,
					Content:   entry.Content,
					PodName:   entry.PodName,
					Container: entry.Container,
					Level:     entry.Level,
				})
			}
		}
	}()

	conn.Send(dto.LogStreamMessage{
		Type:      "pod_added",
		Timestamp: time.Now().Format(time.RFC3339),
		Content:   podName,
		PodName:   podName,
	})
}

// Send 发送消息
func (c *MultiPodStreamConnection) Send(msg dto.LogStreamMessage) {
	select {
	case c.sendCh <- msg:
	default:
		logger.L().Warn("多 Pod 日志流通道已满，丢弃消息")
	}
}

// startSender 启动发送协程
func (c *MultiPodStreamConnection) startSender() {
	for {
		select {
		case <-c.closeCh:
			return
		case msg := <-c.sendCh:
			c.mu.Lock()
			err := c.Conn.WriteJSON(msg)
			c.mu.Unlock()
			if err != nil {
				logger.L().Error("WebSocket发送错误: %v", err)
				return
			}
		}
	}
}

// Pause 暂停日志流
func (c *MultiPodStreamConnection) Pause() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.IsPaused = true
}

// Resume 恢复日志流
func (c *MultiPodStreamConnection) Resume() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.IsPaused = false
}

// RemovePod 移除 Pod
func (c *MultiPodStreamConnection) RemovePod(podName string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if cancel, exists := c.podCancels[podName]; exists {
		cancel()
		delete(c.podCancels, podName)
	}
}

// Close 关闭连接
func (c *MultiPodStreamConnection) Close() {
	c.closeOnce.Do(func() {
		close(c.closeCh)
		// 取消所有 Pod 流
		c.mu.Lock()
		for _, cancel := range c.podCancels {
			cancel()
		}
		c.mu.Unlock()
		if c.Cancel != nil {
			c.Cancel()
		}
		c.Conn.Close()
	})
}

// GetMultiPodConnection 获取多 Pod 连接
func (s *StreamService) GetMultiPodConnection(id string) *MultiPodStreamConnection {
	if conn, ok := s.connections.Load(id); ok {
		if mpConn, ok := conn.(*MultiPodStreamConnection); ok {
			return mpConn
		}
	}
	return nil
}
