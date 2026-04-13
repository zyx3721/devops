package kubernetes

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/remotecommand"

	apperrors "devops/pkg/errors"
	"devops/pkg/logger"
)

// K8sTerminalService 终端服务
type K8sTerminalService struct {
	clientMgr *K8sClientManager
}

// NewK8sTerminalService 创建终端服务
func NewK8sTerminalService(clientMgr *K8sClientManager) *K8sTerminalService {
	return &K8sTerminalService{clientMgr: clientMgr}
}

// TerminalSession 终端会话
type TerminalSession struct {
	ID        string
	ClusterID uint
	Namespace string
	Pod       string
	Container string
	Shell     string
	Cols      uint16
	Rows      uint16
	ws        *websocket.Conn
	sizeChan  chan remotecommand.TerminalSize
	stdinR    *io.PipeReader
	stdinW    *io.PipeWriter
	doneChan  chan struct{}
	mu        sync.Mutex
	closed    bool
}

// TerminalMessage WebSocket 消息
type TerminalMessage struct {
	Type string `json:"type"` // input, output, resize, ping
	Data string `json:"data,omitempty"`
	Cols uint16 `json:"cols,omitempty"`
	Rows uint16 `json:"rows,omitempty"`
}

// NewTerminalSession 创建终端会话
func (s *K8sTerminalService) NewTerminalSession(clusterID uint, namespace, pod, container, shell string) *TerminalSession {
	if shell == "" {
		shell = "/bin/sh"
	}
	stdinR, stdinW := io.Pipe()
	return &TerminalSession{
		ID:        fmt.Sprintf("%d-%s-%s-%s", clusterID, namespace, pod, container),
		ClusterID: clusterID,
		Namespace: namespace,
		Pod:       pod,
		Container: container,
		Shell:     shell,
		Cols:      80,
		Rows:      24,
		sizeChan:  make(chan remotecommand.TerminalSize, 1),
		stdinR:    stdinR,
		stdinW:    stdinW,
		doneChan:  make(chan struct{}),
	}
}

// HandleTerminal 处理终端 WebSocket 连接
func (s *K8sTerminalService) HandleTerminal(ctx context.Context, session *TerminalSession, ws *websocket.Conn) error {
	session.ws = ws

	client, err := s.clientMgr.GetClient(ctx, session.ClusterID)
	if err != nil {
		logger.L().Error("获取 K8s 客户端失败: %v", err)
		s.sendError(ws, "获取 K8s 客户端失败: "+err.Error())
		return err
	}

	restConfig, err := s.clientMgr.GetConfig(ctx, session.ClusterID)
	if err != nil {
		logger.L().Error("获取 K8s 配置失败: %v", err)
		s.sendError(ws, "获取 K8s 配置失败: "+err.Error())
		return err
	}

	// 构建 exec 请求
	req := client.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(session.Pod).
		Namespace(session.Namespace).
		SubResource("exec").
		VersionedParams(&corev1.PodExecOptions{
			Container: session.Container,
			Command:   []string{session.Shell},
			Stdin:     true,
			Stdout:    true,
			Stderr:    true,
			TTY:       true,
		}, scheme.ParameterCodec)

	logger.L().Info("创建终端连接: cluster=%d, ns=%s, pod=%s, container=%s, shell=%s, url=%s",
		session.ClusterID, session.Namespace, session.Pod, session.Container, session.Shell, req.URL().String())

	exec, err := remotecommand.NewSPDYExecutor(restConfig, "POST", req.URL())
	if err != nil {
		logger.L().Error("创建终端执行器失败: %v", err)
		s.sendError(ws, "创建终端执行器失败: "+err.Error())
		return apperrors.Wrap(err, apperrors.ErrCodeInternalError, "创建终端执行器失败")
	}

	// 启动 WebSocket 读取协程
	go session.readLoop()

	// 发送初始终端大小
	select {
	case session.sizeChan <- remotecommand.TerminalSize{Width: session.Cols, Height: session.Rows}:
	default:
	}

	// 创建一个带超时的 context
	execCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	// 监听 doneChan，当 WebSocket 关闭时取消 exec
	go func() {
		select {
		case <-session.doneChan:
			cancel()
		case <-execCtx.Done():
		}
	}()

	logger.L().Info("开始执行终端流...")

	// 执行终端
	err = exec.StreamWithContext(execCtx, remotecommand.StreamOptions{
		Stdin:             session.stdinR,
		Stdout:            session,
		Stderr:            session,
		Tty:               true,
		TerminalSizeQueue: session,
	})

	if err != nil {
		logger.L().Error("终端执行错误: %v", err)
		s.sendError(ws, "终端执行错误: "+err.Error())
	}

	session.Close()
	return err
}

// sendError 发送错误消息到 WebSocket
func (s *K8sTerminalService) sendError(ws *websocket.Conn, errMsg string) {
	msg := TerminalMessage{
		Type: "output",
		Data: "\r\n\x1b[31m错误: " + errMsg + "\x1b[0m\r\n",
	}
	ws.WriteJSON(msg)
}

// Write 实现 io.Writer 接口，向 WebSocket 写入输出
func (s *TerminalSession) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return 0, io.EOF
	}

	msg := TerminalMessage{
		Type: "output",
		Data: string(p),
	}
	if err := s.ws.WriteJSON(msg); err != nil {
		return 0, err
	}
	return len(p), nil
}

// Next 实现 TerminalSizeQueue 接口
func (s *TerminalSession) Next() *remotecommand.TerminalSize {
	select {
	case size := <-s.sizeChan:
		return &size
	case <-s.doneChan:
		return nil
	}
}

// readLoop 读取 WebSocket 消息的循环
func (s *TerminalSession) readLoop() {
	defer func() {
		s.Close()
	}()

	// 设置读取超时
	s.ws.SetReadDeadline(time.Time{}) // 无超时

	for {
		select {
		case <-s.doneChan:
			return
		default:
		}

		_, message, err := s.ws.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure, websocket.CloseNormalClosure) {
				logger.L().Error("WebSocket read error: %v", err)
			}
			return
		}

		var msg TerminalMessage
		if err := json.Unmarshal(message, &msg); err != nil {
			// 如果不是 JSON，当作原始输入处理
			s.stdinW.Write(message)
			continue
		}

		switch msg.Type {
		case "input":
			if msg.Data != "" {
				s.stdinW.Write([]byte(msg.Data))
			}
		case "resize":
			s.Cols = msg.Cols
			s.Rows = msg.Rows
			select {
			case s.sizeChan <- remotecommand.TerminalSize{Width: msg.Cols, Height: msg.Rows}:
			default:
				// 如果 channel 满了，清空后重新发送
				select {
				case <-s.sizeChan:
				default:
				}
				s.sizeChan <- remotecommand.TerminalSize{Width: msg.Cols, Height: msg.Rows}
			}
		case "ping":
			s.mu.Lock()
			s.ws.WriteJSON(TerminalMessage{Type: "pong"})
			s.mu.Unlock()
		}
	}
}

// Close 关闭会话
func (s *TerminalSession) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return
	}
	s.closed = true

	select {
	case <-s.doneChan:
	default:
		close(s.doneChan)
	}

	if s.stdinW != nil {
		s.stdinW.Close()
	}
	if s.stdinR != nil {
		s.stdinR.Close()
	}
	if s.ws != nil {
		s.ws.Close()
	}
}
