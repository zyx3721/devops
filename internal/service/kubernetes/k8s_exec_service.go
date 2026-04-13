package kubernetes

import (
	"context"
	"fmt"
	"io"
	"sync"

	"github.com/gorilla/websocket"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/remotecommand"

	"devops/pkg/logger"
)

// K8sExecService Pod 执行服务
type K8sExecService struct {
	clientMgr *K8sClientManager
}

// NewK8sExecService 创建执行服务
func NewK8sExecService(clientMgr *K8sClientManager) *K8sExecService {
	return &K8sExecService{clientMgr: clientMgr}
}

// ExecRequest 执行请求
type ExecRequest struct {
	ClusterID uint   `json:"cluster_id"`
	Namespace string `json:"namespace"`
	PodName   string `json:"pod_name"`
	Container string `json:"container"`
	Command   string `json:"command"`
}

// WebTerminal WebSocket 终端
type WebTerminal struct {
	conn   *websocket.Conn
	sizeCh chan remotecommand.TerminalSize
	doneCh chan struct{}
	mu     sync.Mutex
}

// NewWebTerminal 创建 Web 终端
func NewWebTerminal(conn *websocket.Conn) *WebTerminal {
	return &WebTerminal{
		conn:   conn,
		sizeCh: make(chan remotecommand.TerminalSize, 1),
		doneCh: make(chan struct{}),
	}
}

// Read 实现 io.Reader
func (t *WebTerminal) Read(p []byte) (n int, err error) {
	_, message, err := t.conn.ReadMessage()
	if err != nil {
		return 0, err
	}

	// 处理终端大小调整消息
	if len(message) > 0 && message[0] == 1 {
		// 格式: 1 + rows(2bytes) + cols(2bytes)
		if len(message) >= 5 {
			rows := uint16(message[1])<<8 | uint16(message[2])
			cols := uint16(message[3])<<8 | uint16(message[4])
			select {
			case t.sizeCh <- remotecommand.TerminalSize{Width: cols, Height: rows}:
			default:
			}
		}
		return 0, nil
	}

	copy(p, message)
	return len(message), nil
}

// Write 实现 io.Writer
func (t *WebTerminal) Write(p []byte) (n int, err error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	err = t.conn.WriteMessage(websocket.BinaryMessage, p)
	if err != nil {
		return 0, err
	}
	return len(p), nil
}

// Next 实现 remotecommand.TerminalSizeQueue
func (t *WebTerminal) Next() *remotecommand.TerminalSize {
	select {
	case size := <-t.sizeCh:
		return &size
	case <-t.doneCh:
		return nil
	}
}

// Close 关闭终端
func (t *WebTerminal) Close() {
	close(t.doneCh)
}

// ExecInPodWithWebSocket 在 Pod 中执行命令（WebSocket 交互式）
func (s *K8sExecService) ExecInPodWithWebSocket(ctx context.Context, req *ExecRequest, conn *websocket.Conn) error {
	log := logger.L().WithField("pod", req.PodName).WithField("namespace", req.Namespace)

	client, err := s.clientMgr.GetClient(ctx, req.ClusterID)
	if err != nil {
		return fmt.Errorf("获取集群客户端失败: %w", err)
	}

	restConfig, err := s.clientMgr.GetRestConfig(ctx, req.ClusterID)
	if err != nil {
		return fmt.Errorf("获取集群配置失败: %w", err)
	}

	// 检查 Pod 是否存在
	pod, err := client.CoreV1().Pods(req.Namespace).Get(ctx, req.PodName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("Pod 不存在: %w", err)
	}

	// 确定容器
	container := req.Container
	if container == "" && len(pod.Spec.Containers) > 0 {
		container = pod.Spec.Containers[0].Name
	}

	// 确定 shell
	shell := req.Command
	if shell == "" {
		shell = "/bin/sh"
	}

	// 创建 exec 请求
	execReq := client.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(req.PodName).
		Namespace(req.Namespace).
		SubResource("exec").
		VersionedParams(&corev1.PodExecOptions{
			Container: container,
			Command:   []string{shell},
			Stdin:     true,
			Stdout:    true,
			Stderr:    true,
			TTY:       true,
		}, scheme.ParameterCodec)

	log.WithField("url", execReq.URL().String()).Info("创建 WebShell 连接")

	// 创建 SPDY 执行器
	exec, err := remotecommand.NewSPDYExecutor(restConfig, "POST", execReq.URL())
	if err != nil {
		return fmt.Errorf("创建执行器失败: %w", err)
	}

	// 创建 Web 终端
	terminal := NewWebTerminal(conn)
	defer terminal.Close()

	// 执行
	err = exec.StreamWithContext(ctx, remotecommand.StreamOptions{
		Stdin:             terminal,
		Stdout:            terminal,
		Stderr:            terminal,
		Tty:               true,
		TerminalSizeQueue: terminal,
	})

	if err != nil {
		log.WithField("error", err).Error("WebShell 执行错误")
		return err
	}

	return nil
}

// GetPodShells 获取 Pod 可用的 shell
func (s *K8sExecService) GetPodShells(ctx context.Context, clusterID uint, namespace, podName, container string) ([]string, error) {
	client, err := s.clientMgr.GetClient(ctx, clusterID)
	if err != nil {
		return nil, err
	}

	restConfig, err := s.clientMgr.GetRestConfig(ctx, clusterID)
	if err != nil {
		return nil, err
	}

	shells := []string{"/bin/sh", "/bin/bash", "/bin/zsh"}
	available := []string{}

	for _, shell := range shells {
		execReq := client.CoreV1().RESTClient().Post().
			Resource("pods").
			Name(podName).
			Namespace(namespace).
			SubResource("exec").
			VersionedParams(&corev1.PodExecOptions{
				Container: container,
				Command:   []string{"which", shell},
				Stdout:    true,
				Stderr:    true,
			}, scheme.ParameterCodec)

		exec, err := remotecommand.NewSPDYExecutor(restConfig, "POST", execReq.URL())
		if err != nil {
			continue
		}

		err = exec.StreamWithContext(ctx, remotecommand.StreamOptions{
			Stdout: io.Discard,
			Stderr: io.Discard,
		})
		if err == nil {
			available = append(available, shell)
		}
	}

	if len(available) == 0 {
		available = []string{"/bin/sh"}
	}

	return available, nil
}
