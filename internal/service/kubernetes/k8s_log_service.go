package kubernetes

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"regexp"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	apperrors "devops/pkg/errors"
)

// K8sLogService 日志服务
type K8sLogService struct {
	clientMgr *K8sClientManager
}

// NewK8sLogService 创建日志服务
func NewK8sLogService(clientMgr *K8sClientManager) *K8sLogService {
	return &K8sLogService{clientMgr: clientMgr}
}

// LogEntry 日志条目
type LogEntry struct {
	Timestamp string `json:"timestamp"`
	Content   string `json:"content"`
	Level     string `json:"level"`
	Pod       string `json:"pod,omitempty"`
	Container string `json:"container,omitempty"`
}

// LogQueryRequest 日志查询请求
type LogQueryRequest struct {
	ClusterID  uint     `json:"cluster_id"`
	Namespace  string   `json:"namespace"`
	PodNames   []string `json:"pod_names"` // 支持多Pod
	Container  string   `json:"container"`
	TailLines  int64    `json:"tail_lines"`
	SinceTime  string   `json:"since_time"` // RFC3339 格式
	UntilTime  string   `json:"until_time"`
	Keyword    string   `json:"keyword"` // 关键词搜索
	Level      string   `json:"level"`   // 日志级别过滤
	Timestamps bool     `json:"timestamps"`
}

// LogQueryResponse 日志查询响应
type LogQueryResponse struct {
	Logs       []LogEntry `json:"logs"`
	TotalLines int        `json:"total_lines"`
	HasMore    bool       `json:"has_more"`
}

// GetLogs 获取日志（支持多Pod聚合、搜索、过滤）
func (s *K8sLogService) GetLogs(ctx context.Context, req *LogQueryRequest) (*LogQueryResponse, error) {
	client, err := s.clientMgr.GetClient(ctx, req.ClusterID)
	if err != nil {
		return nil, err
	}

	var allLogs []LogEntry

	// 如果没有指定Pod，获取namespace下所有Pod
	podNames := req.PodNames
	if len(podNames) == 0 {
		pods, err := client.CoreV1().Pods(req.Namespace).List(ctx, metav1.ListOptions{})
		if err != nil {
			return nil, apperrors.Wrap(err, apperrors.ErrCodeInternalError, "获取Pod列表失败")
		}
		for _, pod := range pods.Items {
			podNames = append(podNames, pod.Name)
		}
	}

	// 获取每个Pod的日志
	for _, podName := range podNames {
		logs, err := s.getPodLogs(ctx, client, req, podName)
		if err != nil {
			continue // 跳过失败的Pod
		}
		allLogs = append(allLogs, logs...)
	}

	// 按时间排序
	sortLogsByTime(allLogs)

	// 应用过滤
	filteredLogs := s.filterLogs(allLogs, req)

	return &LogQueryResponse{
		Logs:       filteredLogs,
		TotalLines: len(filteredLogs),
		HasMore:    false,
	}, nil
}

// getPodLogs 获取单个Pod的日志
func (s *K8sLogService) getPodLogs(ctx context.Context, client interface{}, req *LogQueryRequest, podName string) ([]LogEntry, error) {
	k8sClient, err := s.clientMgr.GetClient(ctx, req.ClusterID)
	if err != nil {
		return nil, err
	}

	opts := &corev1.PodLogOptions{
		Timestamps: true,
	}
	if req.Container != "" {
		opts.Container = req.Container
	}
	if req.TailLines > 0 {
		opts.TailLines = &req.TailLines
	}
	if req.SinceTime != "" {
		t, err := time.Parse(time.RFC3339, req.SinceTime)
		if err == nil {
			sinceTime := metav1.NewTime(t)
			opts.SinceTime = &sinceTime
		}
	}

	stream, err := k8sClient.CoreV1().Pods(req.Namespace).GetLogs(podName, opts).Stream(ctx)
	if err != nil {
		return nil, err
	}
	defer stream.Close()

	var logs []LogEntry
	scanner := bufio.NewScanner(stream)
	// 增大buffer以处理长行
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	for scanner.Scan() {
		line := scanner.Text()
		entry := s.parseLine(line, podName, req.Container)
		logs = append(logs, entry)
	}

	return logs, nil
}

// parseLine 解析日志行
func (s *K8sLogService) parseLine(line, podName, container string) LogEntry {
	entry := LogEntry{
		Pod:       podName,
		Container: container,
	}

	// 尝试解析时间戳 (K8s格式: 2006-01-02T15:04:05.000000000Z)
	if len(line) > 30 && line[4] == '-' && line[7] == '-' {
		parts := strings.SplitN(line, " ", 2)
		if len(parts) == 2 {
			entry.Timestamp = parts[0]
			entry.Content = parts[1]
		} else {
			entry.Content = line
		}
	} else {
		entry.Content = line
		entry.Timestamp = time.Now().Format(time.RFC3339)
	}

	// 检测日志级别
	entry.Level = detectLogLevel(entry.Content)

	return entry
}

// detectLogLevel 检测日志级别
func detectLogLevel(content string) string {
	upper := strings.ToUpper(content)

	// 常见日志级别模式
	patterns := map[string]*regexp.Regexp{
		"ERROR": regexp.MustCompile(`\b(ERROR|ERR|FATAL|PANIC|CRITICAL)\b`),
		"WARN":  regexp.MustCompile(`\b(WARN|WARNING)\b`),
		"INFO":  regexp.MustCompile(`\b(INFO)\b`),
		"DEBUG": regexp.MustCompile(`\b(DEBUG|TRACE)\b`),
	}

	for level, pattern := range patterns {
		if pattern.MatchString(upper) {
			return level
		}
	}
	return "INFO"
}

// filterLogs 过滤日志
func (s *K8sLogService) filterLogs(logs []LogEntry, req *LogQueryRequest) []LogEntry {
	if req.Keyword == "" && req.Level == "" {
		return logs
	}

	var filtered []LogEntry
	keyword := strings.ToLower(req.Keyword)

	for _, log := range logs {
		// 关键词过滤
		if keyword != "" && !strings.Contains(strings.ToLower(log.Content), keyword) {
			continue
		}
		// 级别过滤
		if req.Level != "" && log.Level != req.Level {
			continue
		}
		filtered = append(filtered, log)
	}
	return filtered
}

// sortLogsByTime 按时间排序
func sortLogsByTime(logs []LogEntry) {
	// 简单的冒泡排序，实际可用更高效的排序
	for i := 0; i < len(logs)-1; i++ {
		for j := 0; j < len(logs)-i-1; j++ {
			if logs[j].Timestamp > logs[j+1].Timestamp {
				logs[j], logs[j+1] = logs[j+1], logs[j]
			}
		}
	}
}

// StreamLogsToChannel 流式日志到channel（用于WebSocket）
func (s *K8sLogService) StreamLogsToChannel(ctx context.Context, req *LogQueryRequest, ch chan<- LogEntry) error {
	client, err := s.clientMgr.GetClient(ctx, req.ClusterID)
	if err != nil {
		return err
	}

	opts := &corev1.PodLogOptions{
		Follow:     true,
		Timestamps: true,
	}
	if req.Container != "" {
		opts.Container = req.Container
	}
	tailLines := int64(100)
	if req.TailLines > 0 {
		tailLines = req.TailLines
	}
	opts.TailLines = &tailLines

	// 只支持单Pod流式
	podName := ""
	if len(req.PodNames) > 0 {
		podName = req.PodNames[0]
	} else {
		return fmt.Errorf("pod name required for streaming")
	}

	stream, err := client.CoreV1().Pods(req.Namespace).GetLogs(podName, opts).Stream(ctx)
	if err != nil {
		return apperrors.Wrap(err, apperrors.ErrCodeInternalError, "获取日志流失败")
	}
	defer stream.Close()

	reader := bufio.NewReader(stream)
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			line, err := reader.ReadString('\n')
			if err != nil {
				if err == io.EOF {
					time.Sleep(100 * time.Millisecond)
					continue
				}
				return err
			}
			entry := s.parseLine(strings.TrimSuffix(line, "\n"), podName, req.Container)

			// 应用过滤
			if req.Keyword != "" && !strings.Contains(strings.ToLower(entry.Content), strings.ToLower(req.Keyword)) {
				continue
			}
			if req.Level != "" && entry.Level != req.Level {
				continue
			}

			ch <- entry
		}
	}
}

// DownloadLogs 下载日志
func (s *K8sLogService) DownloadLogs(ctx context.Context, req *LogQueryRequest) ([]byte, error) {
	resp, err := s.GetLogs(ctx, req)
	if err != nil {
		return nil, err
	}

	var builder strings.Builder
	for _, log := range resp.Logs {
		if log.Pod != "" {
			builder.WriteString(fmt.Sprintf("[%s][%s][%s] %s\n", log.Timestamp, log.Pod, log.Level, log.Content))
		} else {
			builder.WriteString(fmt.Sprintf("[%s][%s] %s\n", log.Timestamp, log.Level, log.Content))
		}
	}

	return []byte(builder.String()), nil
}

// GetPodContainerList 获取Pod的容器列表
func (s *K8sLogService) GetPodContainerList(ctx context.Context, clusterID uint, namespace, podName string) ([]string, error) {
	client, err := s.clientMgr.GetClient(ctx, clusterID)
	if err != nil {
		return nil, err
	}

	pod, err := client.CoreV1().Pods(namespace).Get(ctx, podName, metav1.GetOptions{})
	if err != nil {
		return nil, apperrors.Wrap(err, apperrors.ErrCodeNotFound, "Pod不存在")
	}

	var containers []string
	for _, c := range pod.Spec.Containers {
		containers = append(containers, c.Name)
	}
	return containers, nil
}
