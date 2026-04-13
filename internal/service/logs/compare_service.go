package logs

import (
	"context"
	"time"

	"devops/pkg/dto"
)

// CompareService 日志对比服务
type CompareService struct {
	adapter *K8sLogAdapter
}

// NewCompareService 创建对比服务
func NewCompareService(adapter *K8sLogAdapter) *CompareService {
	return &CompareService{
		adapter: adapter,
	}
}

// Compare 对比日志
func (s *CompareService) Compare(ctx context.Context, req *dto.LogCompareRequest) (*dto.LogCompareResponse, error) {
	var leftLogs, rightLogs []dto.LogEntry
	var err error

	if req.CompareType == "time_range" {
		// 时间段对比
		leftLogs, err = s.getLogsByTimeRange(ctx, req.ClusterID, req.Namespace, req.LeftPodName, req.Container, req.LeftStartTime, req.LeftEndTime, req.Keyword, req.Level)
		if err != nil {
			return nil, err
		}

		rightLogs, err = s.getLogsByTimeRange(ctx, req.ClusterID, req.Namespace, req.LeftPodName, req.Container, req.RightStartTime, req.RightEndTime, req.Keyword, req.Level)
		if err != nil {
			return nil, err
		}
	} else {
		// Pod 对比
		leftLogs, err = s.getLogsByPod(ctx, req.ClusterID, req.Namespace, req.LeftPodName, req.Container, req.Keyword, req.Level)
		if err != nil {
			return nil, err
		}

		rightLogs, err = s.getLogsByPod(ctx, req.ClusterID, req.Namespace, req.RightPodName, req.Container, req.Keyword, req.Level)
		if err != nil {
			return nil, err
		}
	}

	// 转换为对比行
	leftLines := s.toCompareLines(leftLogs, req.LeftPodName)
	rightLines := s.toCompareLines(rightLogs, req.RightPodName)

	// 计算差异
	s.calculateDiff(leftLines, rightLines)

	// 统计
	addedCount := 0
	removedCount := 0
	sameCount := 0

	for _, line := range rightLines {
		if line.DiffType == "added" {
			addedCount++
		}
	}
	for _, line := range leftLines {
		if line.DiffType == "removed" {
			removedCount++
		} else if line.DiffType == "same" {
			sameCount++
		}
	}

	return &dto.LogCompareResponse{
		LeftLines:    leftLines,
		RightLines:   rightLines,
		TotalLeft:    len(leftLines),
		TotalRight:   len(rightLines),
		AddedCount:   addedCount,
		RemovedCount: removedCount,
		SameCount:    sameCount,
	}, nil
}

func (s *CompareService) getLogsByTimeRange(ctx context.Context, clusterID int64, namespace, podName, container, startTime, endTime, keyword, level string) ([]dto.LogEntry, error) {
	logReq := &dto.LogStreamRequest{
		ClusterID: clusterID,
		Namespace: namespace,
		PodName:   podName,
		Container: container,
		SinceTime: startTime,
		Keyword:   keyword,
		Level:     level,
	}

	return s.getLogs(ctx, logReq, endTime)
}

func (s *CompareService) getLogsByPod(ctx context.Context, clusterID int64, namespace, podName, container, keyword, level string) ([]dto.LogEntry, error) {
	// 获取最近 1 小时的日志
	startTime := time.Now().Add(-time.Hour).Format(time.RFC3339)

	logReq := &dto.LogStreamRequest{
		ClusterID: clusterID,
		Namespace: namespace,
		PodName:   podName,
		Container: container,
		SinceTime: startTime,
		Keyword:   keyword,
		Level:     level,
		TailLines: 1000,
	}

	return s.getLogs(ctx, logReq, "")
}

func (s *CompareService) getLogs(ctx context.Context, req *dto.LogStreamRequest, endTime string) ([]dto.LogEntry, error) {
	logCh := make(chan dto.LogEntry, 5000)
	errCh := make(chan error, 1)

	timeoutCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	go func() {
		defer close(logCh)
		err := s.adapter.StreamLogs(timeoutCtx, req, logCh)
		if err != nil {
			errCh <- err
		}
	}()

	var logs []dto.LogEntry
	var endTimeT time.Time
	if endTime != "" {
		endTimeT, _ = time.Parse(time.RFC3339, endTime)
	}

	for {
		select {
		case entry, ok := <-logCh:
			if !ok {
				return logs, nil
			}
			// 检查结束时间
			if !endTimeT.IsZero() {
				entryTime, _ := time.Parse(time.RFC3339Nano, entry.Timestamp)
				if entryTime.After(endTimeT) {
					continue
				}
			}
			logs = append(logs, entry)
			// 限制数量
			if len(logs) >= 1000 {
				return logs, nil
			}
		case err := <-errCh:
			return nil, err
		case <-timeoutCtx.Done():
			return logs, nil
		}
	}
}

func (s *CompareService) toCompareLines(logs []dto.LogEntry, podName string) []dto.LogCompareLine {
	lines := make([]dto.LogCompareLine, len(logs))
	for i, log := range logs {
		lines[i] = dto.LogCompareLine{
			LineNumber: i + 1,
			Timestamp:  log.Timestamp,
			Content:    log.Content,
			Level:      log.Level,
			DiffType:   "same",
			PodName:    podName,
		}
	}
	return lines
}

func (s *CompareService) calculateDiff(leftLines, rightLines []dto.LogCompareLine) {
	// 简单的内容匹配算法
	rightContentMap := make(map[string]bool)
	for _, line := range rightLines {
		rightContentMap[line.Content] = true
	}

	leftContentMap := make(map[string]bool)
	for _, line := range leftLines {
		leftContentMap[line.Content] = true
	}

	// 标记左侧独有的为 removed
	for i := range leftLines {
		if !rightContentMap[leftLines[i].Content] {
			leftLines[i].DiffType = "removed"
		}
	}

	// 标记右侧独有的为 added
	for i := range rightLines {
		if !leftContentMap[rightLines[i].Content] {
			rightLines[i].DiffType = "added"
		}
	}
}
