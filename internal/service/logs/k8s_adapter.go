package logs

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"regexp"
	"strings"
	"time"

	"devops/internal/service/kubernetes"
	"devops/pkg/dto"
	apperrors "devops/pkg/errors"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sclient "k8s.io/client-go/kubernetes"
)

// K8sLogAdapter K8s 日志适配器
type K8sLogAdapter struct {
	clientMgr *kubernetes.K8sClientManager
}

// NewK8sLogAdapter 创建 K8s 日志适配器
func NewK8sLogAdapter(clientMgr *kubernetes.K8sClientManager) *K8sLogAdapter {
	return &K8sLogAdapter{clientMgr: clientMgr}
}

// LogAdapter 日志适配器接口
type LogAdapter interface {
	// GetLogs 获取日志
	GetLogs(ctx context.Context, req *dto.LogQueryRequest) (*dto.LogQueryResponse, error)
	// StreamLogs 流式获取日志
	StreamLogs(ctx context.Context, req *dto.LogStreamRequest, ch chan<- dto.LogEntry) error
	// GetContainers 获取 Pod 容器列表
	GetContainers(ctx context.Context, clusterID int64, namespace, podName string) ([]string, error)
}

// GetLogs 获取日志
func (a *K8sLogAdapter) GetLogs(ctx context.Context, req *dto.LogQueryRequest) (*dto.LogQueryResponse, error) {
	client, err := a.clientMgr.GetClient(ctx, uint(req.ClusterID))
	if err != nil {
		return nil, apperrors.Wrap(err, apperrors.ErrCodeInternalError, "获取K8s客户端失败")
	}

	var allLogs []dto.LogEntry
	podNames := req.PodNames

	// 如果没有指定 Pod，获取 namespace 下所有 Pod
	if len(podNames) == 0 {
		pods, err := client.CoreV1().Pods(req.Namespace).List(ctx, metav1.ListOptions{})
		if err != nil {
			return nil, apperrors.Wrap(err, apperrors.ErrCodeInternalError, "获取Pod列表失败")
		}
		for _, pod := range pods.Items {
			podNames = append(podNames, pod.Name)
		}
	}

	// 获取每个 Pod 的日志
	for _, podName := range podNames {
		containers := req.Containers
		if len(containers) == 0 {
			// 获取 Pod 的所有容器
			containerList, err := a.GetContainers(ctx, req.ClusterID, req.Namespace, podName)
			if err != nil {
				continue
			}
			containers = containerList
		}

		for _, container := range containers {
			logs, err := a.getPodContainerLogs(ctx, client, req, podName, container)
			if err != nil {
				continue
			}
			allLogs = append(allLogs, logs...)
		}
	}

	// 按时间排序
	sortLogsByTimestamp(allLogs, req.Order == "desc")

	// 应用过滤
	filteredLogs := a.filterLogs(allLogs, req)

	// 分页
	total := int64(len(filteredLogs))
	page := req.Page
	if page < 1 {
		page = 1
	}
	pageSize := req.PageSize
	if pageSize < 1 {
		pageSize = 100
	}

	start := (page - 1) * pageSize
	end := start + pageSize
	if start > int(total) {
		start = int(total)
	}
	if end > int(total) {
		end = int(total)
	}

	return &dto.LogQueryResponse{
		Total:   total,
		Items:   filteredLogs[start:end],
		HasMore: end < int(total),
	}, nil
}

// getPodContainerLogs 获取单个容器的日志
func (a *K8sLogAdapter) getPodContainerLogs(ctx context.Context, client *k8sclient.Clientset, req *dto.LogQueryRequest, podName, container string) ([]dto.LogEntry, error) {
	opts := &corev1.PodLogOptions{
		Container:  container,
		Timestamps: true,
	}

	// 设置时间范围
	if req.StartTime != "" {
		t, err := time.Parse(time.RFC3339, req.StartTime)
		if err == nil {
			sinceTime := metav1.NewTime(t)
			opts.SinceTime = &sinceTime
		}
	}

	stream, err := client.CoreV1().Pods(req.Namespace).GetLogs(podName, opts).Stream(ctx)
	if err != nil {
		return nil, err
	}
	defer stream.Close()

	var logs []dto.LogEntry
	scanner := bufio.NewScanner(stream)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		entry := a.parseLine(line, podName, container, lineNum)

		// 时间范围过滤
		if req.EndTime != "" {
			endTime, err := time.Parse(time.RFC3339, req.EndTime)
			if err == nil && entry.Timestamp != "" {
				logTime, err := time.Parse(time.RFC3339Nano, entry.Timestamp)
				if err == nil && logTime.After(endTime) {
					continue
				}
			}
		}

		logs = append(logs, entry)
	}

	return logs, nil
}

// StreamLogs 流式获取日志
func (a *K8sLogAdapter) StreamLogs(ctx context.Context, req *dto.LogStreamRequest, ch chan<- dto.LogEntry) error {
	client, err := a.clientMgr.GetClient(ctx, uint(req.ClusterID))
	if err != nil {
		return apperrors.Wrap(err, apperrors.ErrCodeInternalError, "获取K8s客户端失败")
	}

	podName := req.PodName
	if podName == "" && len(req.PodNames) > 0 {
		podName = req.PodNames[0]
	}
	if podName == "" {
		return fmt.Errorf("pod name required for streaming")
	}

	// 检查 Pod 状态
	pod, err := client.CoreV1().Pods(req.Namespace).Get(ctx, podName, metav1.GetOptions{})
	if err != nil {
		return apperrors.Wrap(err, apperrors.ErrCodeNotFound, "Pod不存在: "+podName)
	}

	// 对于已完成的 Pod，只获取历史日志，不 follow
	follow := req.Follow
	if pod.Status.Phase == corev1.PodSucceeded || pod.Status.Phase == corev1.PodFailed {
		follow = false
	}

	container := req.Container
	if container == "" {
		containers, err := a.GetContainers(ctx, req.ClusterID, req.Namespace, podName)
		if err != nil {
			return fmt.Errorf("获取容器列表失败: %v", err)
		}
		if len(containers) > 0 {
			container = containers[0]
		}
	}

	if container == "" {
		return fmt.Errorf("container name required for pod %s", podName)
	}

	opts := &corev1.PodLogOptions{
		Container:  container,
		Follow:     follow,
		Timestamps: true,
	}

	tailLines := req.TailLines
	if tailLines <= 0 {
		tailLines = 100
	}
	opts.TailLines = &tailLines

	if req.SinceTime != "" {
		t, err := time.Parse(time.RFC3339, req.SinceTime)
		if err == nil {
			sinceTime := metav1.NewTime(t)
			opts.SinceTime = &sinceTime
		}
	}

	stream, err := client.CoreV1().Pods(req.Namespace).GetLogs(podName, opts).Stream(ctx)
	if err != nil {
		return apperrors.Wrap(err, apperrors.ErrCodeInternalError, fmt.Sprintf("获取日志流失败: pod=%s, container=%s, err=%v", podName, container, err))
	}
	defer stream.Close()

	reader := bufio.NewReader(stream)
	lineNum := 0
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			line, err := reader.ReadString('\n')
			if err != nil {
				if err == io.EOF {
					if !req.Follow {
						return nil
					}
					time.Sleep(100 * time.Millisecond)
					continue
				}
				return err
			}

			lineNum++
			entry := a.parseLine(strings.TrimSuffix(line, "\n"), podName, container, lineNum)

			// 应用过滤
			if !a.matchFilter(entry, req) {
				continue
			}

			ch <- entry
		}
	}
}

// GetContainers 获取 Pod 容器列表
func (a *K8sLogAdapter) GetContainers(ctx context.Context, clusterID int64, namespace, podName string) ([]string, error) {
	client, err := a.clientMgr.GetClient(ctx, uint(clusterID))
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

// parseLine 解析日志行
func (a *K8sLogAdapter) parseLine(line, podName, container string, lineNum int) dto.LogEntry {
	entry := dto.LogEntry{
		ID:        fmt.Sprintf("%s-%s-%d", podName, container, lineNum),
		PodName:   podName,
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
			entry.Timestamp = time.Now().Format(time.RFC3339Nano)
		}
	} else {
		entry.Content = line
		entry.Timestamp = time.Now().Format(time.RFC3339Nano)
	}

	// 检测日志级别
	entry.Level = detectLogLevel(entry.Content)

	return entry
}

// filterLogs 过滤日志
func (a *K8sLogAdapter) filterLogs(logs []dto.LogEntry, req *dto.LogQueryRequest) []dto.LogEntry {
	if req.Keyword == "" && req.Regex == "" && req.Level == "" {
		return logs
	}

	var filtered []dto.LogEntry
	keyword := strings.ToLower(req.Keyword)

	var regexPattern *regexp.Regexp
	if req.Regex != "" {
		regexPattern, _ = regexp.Compile(req.Regex)
	}

	for _, log := range logs {
		// 关键词过滤
		if keyword != "" && !strings.Contains(strings.ToLower(log.Content), keyword) {
			continue
		}
		// 正则过滤
		if regexPattern != nil && !regexPattern.MatchString(log.Content) {
			continue
		}
		// 级别过滤
		if req.Level != "" && !strings.EqualFold(log.Level, req.Level) {
			continue
		}
		filtered = append(filtered, log)
	}
	return filtered
}

// matchFilter 检查日志是否匹配过滤条件
func (a *K8sLogAdapter) matchFilter(entry dto.LogEntry, req *dto.LogStreamRequest) bool {
	// 关键词过滤
	if req.Keyword != "" {
		if req.UseRegex {
			matched, _ := regexp.MatchString(req.Keyword, entry.Content)
			if !matched {
				return false
			}
		} else {
			if !strings.Contains(strings.ToLower(entry.Content), strings.ToLower(req.Keyword)) {
				return false
			}
		}
	}
	// 级别过滤
	if req.Level != "" && !strings.EqualFold(entry.Level, req.Level) {
		return false
	}
	return true
}

// detectLogLevel 检测日志级别
func detectLogLevel(content string) string {
	upper := strings.ToUpper(content)

	patterns := map[string]*regexp.Regexp{
		"FATAL": regexp.MustCompile(`\b(FATAL|PANIC|CRITICAL)\b`),
		"ERROR": regexp.MustCompile(`\b(ERROR|ERR)\b`),
		"WARN":  regexp.MustCompile(`\b(WARN|WARNING)\b`),
		"INFO":  regexp.MustCompile(`\b(INFO)\b`),
		"DEBUG": regexp.MustCompile(`\b(DEBUG|TRACE)\b`),
	}

	// 按优先级检测
	for _, level := range []string{"FATAL", "ERROR", "WARN", "INFO", "DEBUG"} {
		if patterns[level].MatchString(upper) {
			return level
		}
	}
	return "INFO"
}

// sortLogsByTimestamp 按时间戳排序
func sortLogsByTimestamp(logs []dto.LogEntry, desc bool) {
	for i := 0; i < len(logs)-1; i++ {
		for j := 0; j < len(logs)-i-1; j++ {
			shouldSwap := logs[j].Timestamp > logs[j+1].Timestamp
			if desc {
				shouldSwap = logs[j].Timestamp < logs[j+1].Timestamp
			}
			if shouldSwap {
				logs[j], logs[j+1] = logs[j+1], logs[j]
			}
		}
	}
}
