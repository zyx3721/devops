package logs

import (
	"context"
	"regexp"
	"sort"
	"strings"
	"time"

	"devops/pkg/dto"
)

// StatsService 日志统计服务
type StatsService struct {
	adapter *K8sLogAdapter
}

// NewStatsService 创建统计服务
func NewStatsService(adapter *K8sLogAdapter) *StatsService {
	return &StatsService{
		adapter: adapter,
	}
}

// GetStats 获取日志统计
func (s *StatsService) GetStats(ctx context.Context, req *dto.LogStatsRequest) (*dto.LogStatsResponse, error) {
	// 使用 GetLogs 方法获取日志（支持 namespace 级别查询）
	logReq := &dto.LogQueryRequest{
		ClusterID: req.ClusterID,
		Namespace: req.Namespace,
		StartTime: req.StartTime,
		EndTime:   req.EndTime,
		Page:      1,
		PageSize:  10000, // 获取足够多的日志用于统计
	}

	// 如果指定了 PodName，添加到查询条件
	if req.PodName != "" {
		logReq.PodNames = []string{req.PodName}
	}

	// 创建带超时的上下文
	timeoutCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	// 获取日志
	logResp, err := s.adapter.GetLogs(timeoutCtx, logReq)
	if err != nil {
		return nil, err
	}

	// 统计数据
	var totalCount int64
	levelCounts := make(map[string]int64)
	trendData := make(map[string]map[string]int64) // time -> level -> count
	errorPatterns := make(map[string]*errorPatternStat)

	// 解析结束时间
	var endTime time.Time
	if req.EndTime != "" {
		endTime, _ = time.Parse(time.RFC3339, req.EndTime)
	} else {
		endTime = time.Now()
	}

	// 确定时间间隔
	interval := time.Hour
	if req.Interval == "day" {
		interval = 24 * time.Hour
	}

	// 处理日志
	for _, entry := range logResp.Items {
		// 检查时间范围
		entryTime, err := time.Parse(time.RFC3339Nano, entry.Timestamp)
		if err != nil {
			entryTime, _ = time.Parse(time.RFC3339, entry.Timestamp)
		}
		if !endTime.IsZero() && entryTime.After(endTime) {
			continue
		}

		totalCount++

		// 级别统计
		level := strings.ToUpper(entry.Level)
		if level == "" {
			level = "UNKNOWN"
		}
		levelCounts[level]++

		// 趋势统计
		timeKey := s.getTimeKey(entryTime, interval)
		if trendData[timeKey] == nil {
			trendData[timeKey] = make(map[string]int64)
		}
		trendData[timeKey][level]++

		// 错误模式统计
		if level == "ERROR" || level == "FATAL" {
			pattern := s.extractErrorPattern(entry.Content)
			if stat, exists := errorPatterns[pattern]; exists {
				stat.count++
				stat.lastSample = entry.Content
			} else {
				errorPatterns[pattern] = &errorPatternStat{
					pattern:    pattern,
					count:      1,
					lastSample: entry.Content,
				}
			}
		}
	}
	// 构建趋势数据
	var trend []dto.TrendPoint
	for timeKey, levels := range trendData {
		for level, count := range levels {
			trend = append(trend, dto.TrendPoint{
				Time:  timeKey,
				Count: count,
				Level: level,
			})
		}
	}
	// 按时间排序
	sort.Slice(trend, func(i, j int) bool {
		return trend[i].Time < trend[j].Time
	})

	// 构建 Top 错误
	var topErrors []dto.ErrorStat
	for _, stat := range errorPatterns {
		topErrors = append(topErrors, dto.ErrorStat{
			Pattern: stat.pattern,
			Count:   stat.count,
			Sample:  stat.lastSample,
		})
	}
	// 按数量排序
	sort.Slice(topErrors, func(i, j int) bool {
		return topErrors[i].Count > topErrors[j].Count
	})
	// 只取 Top 10
	if len(topErrors) > 10 {
		topErrors = topErrors[:10]
	}

	return &dto.LogStatsResponse{
		TotalCount:  totalCount,
		LevelCounts: levelCounts,
		Trend:       trend,
		TopErrors:   topErrors,
	}, nil
}

type errorPatternStat struct {
	pattern    string
	count      int64
	lastSample string
}

// getTimeKey 获取时间键
func (s *StatsService) getTimeKey(t time.Time, interval time.Duration) string {
	if interval >= 24*time.Hour {
		return t.Format("2006-01-02")
	}
	return t.Format("2006-01-02 15:00")
}

// extractErrorPattern 提取错误模式
func (s *StatsService) extractErrorPattern(content string) string {
	// 移除数字、UUID、时间戳等变量部分
	pattern := content

	// 移除 UUID
	uuidRegex := regexp.MustCompile(`[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}`)
	pattern = uuidRegex.ReplaceAllString(pattern, "<UUID>")

	// 移除数字
	numRegex := regexp.MustCompile(`\b\d+\b`)
	pattern = numRegex.ReplaceAllString(pattern, "<NUM>")

	// 移除 IP 地址
	ipRegex := regexp.MustCompile(`\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}`)
	pattern = ipRegex.ReplaceAllString(pattern, "<IP>")

	// 截取前 100 个字符
	if len(pattern) > 100 {
		pattern = pattern[:100] + "..."
	}

	return pattern
}
