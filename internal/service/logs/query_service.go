package logs

import (
	"context"
	"regexp"
	"strings"

	"devops/pkg/dto"
)

// QueryService 日志查询服务
type QueryService struct {
	adapter *K8sLogAdapter
}

// NewQueryService 创建日志查询服务
func NewQueryService(adapter *K8sLogAdapter) *QueryService {
	return &QueryService{adapter: adapter}
}

// Query 查询日志
func (s *QueryService) Query(ctx context.Context, req *dto.LogQueryRequest) (*dto.LogQueryResponse, error) {
	return s.adapter.GetLogs(ctx, req)
}

// SearchByKeyword 关键词搜索
func (s *QueryService) SearchByKeyword(ctx context.Context, req *dto.LogQueryRequest) (*dto.LogQueryResponse, error) {
	resp, err := s.adapter.GetLogs(ctx, req)
	if err != nil {
		return nil, err
	}

	if req.Keyword == "" {
		return resp, nil
	}

	keyword := strings.ToLower(req.Keyword)
	var filtered []dto.LogEntry
	for _, entry := range resp.Items {
		if strings.Contains(strings.ToLower(entry.Content), keyword) {
			filtered = append(filtered, entry)
		}
	}

	return &dto.LogQueryResponse{
		Total:   int64(len(filtered)),
		Items:   filtered,
		HasMore: false,
	}, nil
}

// SearchByRegex 正则搜索
func (s *QueryService) SearchByRegex(ctx context.Context, req *dto.LogQueryRequest) (*dto.LogQueryResponse, error) {
	resp, err := s.adapter.GetLogs(ctx, req)
	if err != nil {
		return nil, err
	}

	if req.Regex == "" {
		return resp, nil
	}

	pattern, err := regexp.Compile(req.Regex)
	if err != nil {
		return nil, err
	}

	var filtered []dto.LogEntry
	for _, entry := range resp.Items {
		if pattern.MatchString(entry.Content) {
			filtered = append(filtered, entry)
		}
	}

	return &dto.LogQueryResponse{
		Total:   int64(len(filtered)),
		Items:   filtered,
		HasMore: false,
	}, nil
}

// FilterByLevel 按日志级别过滤
func (s *QueryService) FilterByLevel(ctx context.Context, req *dto.LogQueryRequest) (*dto.LogQueryResponse, error) {
	resp, err := s.adapter.GetLogs(ctx, req)
	if err != nil {
		return nil, err
	}

	if req.Level == "" {
		return resp, nil
	}

	var filtered []dto.LogEntry
	for _, entry := range resp.Items {
		if strings.EqualFold(entry.Level, req.Level) {
			filtered = append(filtered, entry)
		}
	}

	return &dto.LogQueryResponse{
		Total:   int64(len(filtered)),
		Items:   filtered,
		HasMore: false,
	}, nil
}

// FilterByLevels 按多个日志级别过滤
func (s *QueryService) FilterByLevels(ctx context.Context, req *dto.LogQueryRequest, levels []string) (*dto.LogQueryResponse, error) {
	resp, err := s.adapter.GetLogs(ctx, req)
	if err != nil {
		return nil, err
	}

	if len(levels) == 0 {
		return resp, nil
	}

	levelSet := make(map[string]bool)
	for _, level := range levels {
		levelSet[strings.ToUpper(level)] = true
	}

	var filtered []dto.LogEntry
	for _, entry := range resp.Items {
		if levelSet[strings.ToUpper(entry.Level)] {
			filtered = append(filtered, entry)
		}
	}

	return &dto.LogQueryResponse{
		Total:   int64(len(filtered)),
		Items:   filtered,
		HasMore: false,
	}, nil
}

// GetContainers 获取容器列表
func (s *QueryService) GetContainers(ctx context.Context, clusterID int64, namespace, podName string) ([]string, error) {
	return s.adapter.GetContainers(ctx, clusterID, namespace, podName)
}

// HighlightKeyword 高亮关键词
func HighlightKeyword(content, keyword string) string {
	if keyword == "" {
		return content
	}
	// 使用 HTML 标签高亮
	re := regexp.MustCompile("(?i)" + regexp.QuoteMeta(keyword))
	return re.ReplaceAllString(content, "<mark>$0</mark>")
}

// HighlightRegex 高亮正则匹配
func HighlightRegex(content, pattern string) (string, error) {
	if pattern == "" {
		return content, nil
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		return content, err
	}
	return re.ReplaceAllString(content, "<mark>$0</mark>"), nil
}
