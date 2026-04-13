package logs

import (
	"context"
	"sort"
	"time"

	"devops/pkg/dto"
)

// ContextService 日志上下文服务
type ContextService struct {
	adapter *K8sLogAdapter
}

// NewContextService 创建上下文服务
func NewContextService(adapter *K8sLogAdapter) *ContextService {
	return &ContextService{
		adapter: adapter,
	}
}

// GetContext 获取日志上下文
func (s *ContextService) GetContext(ctx context.Context, req *dto.LogContextRequest) (*dto.LogContextResponse, error) {
	// 设置默认值
	if req.LinesBefore <= 0 {
		req.LinesBefore = 100
	}
	if req.LinesAfter <= 0 {
		req.LinesAfter = 100
	}

	// 解析目标时间戳
	targetTime, err := time.Parse(time.RFC3339Nano, req.Timestamp)
	if err != nil {
		targetTime, err = time.Parse(time.RFC3339, req.Timestamp)
		if err != nil {
			return nil, err
		}
	}

	// 计算时间范围（前后各扩展一些时间）
	beforeDuration := time.Duration(req.LinesBefore) * time.Second * 2
	afterDuration := time.Duration(req.LinesAfter) * time.Second * 2

	startTime := targetTime.Add(-beforeDuration)
	endTime := targetTime.Add(afterDuration)

	// 获取日志
	logReq := &dto.LogStreamRequest{
		ClusterID: req.ClusterID,
		Namespace: req.Namespace,
		PodName:   req.PodName,
		Container: req.Container,
		SinceTime: startTime.Format(time.RFC3339),
	}

	logCh := make(chan dto.LogEntry, 10000)
	errCh := make(chan error, 1)

	// 创建带超时的上下文
	timeoutCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	go func() {
		defer close(logCh)
		err := s.adapter.StreamLogs(timeoutCtx, logReq, logCh)
		if err != nil {
			errCh <- err
		}
	}()

	// 收集日志
	var allLogs []dto.LogEntry
	for {
		select {
		case entry, ok := <-logCh:
			if !ok {
				goto ProcessLogs
			}
			// 过滤时间范围
			entryTime, err := time.Parse(time.RFC3339Nano, entry.Timestamp)
			if err != nil {
				entryTime, _ = time.Parse(time.RFC3339, entry.Timestamp)
			}
			if entryTime.Before(endTime) {
				allLogs = append(allLogs, entry)
			}
		case err := <-errCh:
			return nil, err
		case <-timeoutCtx.Done():
			goto ProcessLogs
		}
	}

ProcessLogs:
	// 按时间戳排序
	sort.Slice(allLogs, func(i, j int) bool {
		return allLogs[i].Timestamp < allLogs[j].Timestamp
	})

	// 查找目标行
	targetIndex := -1
	for i, entry := range allLogs {
		if entry.Timestamp == req.Timestamp {
			targetIndex = i
			break
		}
		// 如果找不到精确匹配，找最接近的
		entryTime, _ := time.Parse(time.RFC3339Nano, entry.Timestamp)
		if entryTime.After(targetTime) && targetIndex == -1 {
			targetIndex = i
		}
	}

	if targetIndex == -1 && len(allLogs) > 0 {
		targetIndex = len(allLogs) / 2
	}

	// 构建响应
	resp := &dto.LogContextResponse{
		Before: []dto.LogEntry{},
		After:  []dto.LogEntry{},
	}

	if targetIndex >= 0 && targetIndex < len(allLogs) {
		resp.Current = allLogs[targetIndex]

		// 获取前面的行
		startIdx := targetIndex - req.LinesBefore
		if startIdx < 0 {
			startIdx = 0
		}
		resp.Before = allLogs[startIdx:targetIndex]
		resp.TotalBefore = len(resp.Before)

		// 获取后面的行
		endIdx := targetIndex + req.LinesAfter + 1
		if endIdx > len(allLogs) {
			endIdx = len(allLogs)
		}
		if targetIndex+1 < len(allLogs) {
			resp.After = allLogs[targetIndex+1 : endIdx]
		}
		resp.TotalAfter = len(resp.After)
	}

	return resp, nil
}
