package logs

import (
	"encoding/json"
	"regexp"
	"strconv"
	"strings"
	"time"

	"devops/internal/models"
	"devops/pkg/dto"
)

// ParserService 日志解析服务
type ParserService struct {
	templates []models.LogParseTemplate
}

// NewParserService 创建日志解析服务
func NewParserService() *ParserService {
	return &ParserService{}
}

// SetTemplates 设置解析模板
func (s *ParserService) SetTemplates(templates []models.LogParseTemplate) {
	s.templates = templates
}

// Parse 解析日志
func (s *ParserService) Parse(content string) (map[string]interface{}, error) {
	// 尝试 JSON 解析
	if parsed, err := s.ParseJSON(content); err == nil && len(parsed) > 0 {
		return parsed, nil
	}

	// 尝试使用模板解析
	for _, template := range s.templates {
		if !template.Enabled {
			continue
		}
		parsed, err := s.ParseWithTemplate(content, &template)
		if err == nil && len(parsed) > 0 {
			return parsed, nil
		}
	}

	return nil, nil
}

// ParseJSON 解析 JSON 日志
func (s *ParserService) ParseJSON(content string) (map[string]interface{}, error) {
	// 查找 JSON 开始位置
	start := strings.Index(content, "{")
	if start == -1 {
		return nil, nil
	}

	jsonStr := content[start:]
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return nil, err
	}

	return result, nil
}

// ParseWithTemplate 使用模板解析日志
func (s *ParserService) ParseWithTemplate(content string, template *models.LogParseTemplate) (map[string]interface{}, error) {
	switch template.Type {
	case "json":
		return s.parseJSONWithFields(content, template.Fields)
	case "regex":
		return s.parseRegex(content, template.Pattern, template.Fields)
	case "grok":
		return s.parseGrok(content, template.Pattern, template.Fields)
	default:
		return nil, nil
	}
}

// parseJSONWithFields 使用字段映射解析 JSON
func (s *ParserService) parseJSONWithFields(content string, fields []models.ParseField) (map[string]interface{}, error) {
	parsed, err := s.ParseJSON(content)
	if err != nil {
		return nil, err
	}

	result := make(map[string]interface{})
	for _, field := range fields {
		if field.JSONPath != "" {
			value := getJSONPath(parsed, field.JSONPath)
			if value != nil {
				result[field.Name] = convertValue(value, field.Type)
			}
		}
	}

	return result, nil
}

// parseRegex 使用正则解析日志
func (s *ParserService) parseRegex(content, pattern string, fields []models.ParseField) (map[string]interface{}, error) {
	if pattern == "" {
		return nil, nil
	}

	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, err
	}

	match := re.FindStringSubmatch(content)
	if match == nil {
		return nil, nil
	}

	names := re.SubexpNames()
	result := make(map[string]interface{})

	for i, name := range names {
		if i > 0 && name != "" && i < len(match) {
			// 查找字段类型
			fieldType := "string"
			for _, field := range fields {
				if field.Name == name {
					fieldType = field.Type
					break
				}
			}
			result[name] = convertValue(match[i], fieldType)
		}
	}

	return result, nil
}

// parseGrok 使用 Grok 模式解析日志
func (s *ParserService) parseGrok(content, pattern string, fields []models.ParseField) (map[string]interface{}, error) {
	// 将 Grok 模式转换为正则表达式
	regexPattern := grokToRegex(pattern)
	return s.parseRegex(content, regexPattern, fields)
}

// grokToRegex 将 Grok 模式转换为正则表达式
func grokToRegex(pattern string) string {
	// 常用 Grok 模式
	grokPatterns := map[string]string{
		"TIMESTAMP_ISO8601": `\d{4}-\d{2}-\d{2}[T ]\d{2}:\d{2}:\d{2}(?:\.\d+)?(?:Z|[+-]\d{2}:?\d{2})?`,
		"LOGLEVEL":          `(?:DEBUG|INFO|WARN(?:ING)?|ERROR|FATAL|CRITICAL)`,
		"GREEDYDATA":        `.*`,
		"DATA":              `.*?`,
		"WORD":              `\w+`,
		"NUMBER":            `\d+(?:\.\d+)?`,
		"INT":               `\d+`,
		"IP":                `\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}`,
		"HOSTNAME":          `[a-zA-Z0-9._-]+`,
		"PATH":              `(?:/[^\s]*)+`,
		"URI":               `[a-zA-Z][a-zA-Z0-9+.-]*://[^\s]+`,
		"QUOTEDSTRING":      `"(?:[^"\\]|\\.)*"`,
	}

	result := pattern
	// 替换 %{PATTERN:name} 格式
	re := regexp.MustCompile(`%\{(\w+)(?::(\w+))?\}`)
	result = re.ReplaceAllStringFunc(result, func(match string) string {
		parts := re.FindStringSubmatch(match)
		patternName := parts[1]
		fieldName := parts[2]

		regex, ok := grokPatterns[patternName]
		if !ok {
			regex = `.*?`
		}

		if fieldName != "" {
			return `(?P<` + fieldName + `>` + regex + `)`
		}
		return `(?:` + regex + `)`
	})

	return result
}

// getJSONPath 获取 JSON 路径值
func getJSONPath(data map[string]interface{}, path string) interface{} {
	// 简单的 JSONPath 实现，支持 $.field.subfield 格式
	path = strings.TrimPrefix(path, "$.")
	parts := strings.Split(path, ".")

	var current interface{} = data
	for _, part := range parts {
		if m, ok := current.(map[string]interface{}); ok {
			current = m[part]
		} else {
			return nil
		}
	}
	return current
}

// convertValue 转换值类型
func convertValue(value interface{}, fieldType string) interface{} {
	str, ok := value.(string)
	if !ok {
		return value
	}

	switch fieldType {
	case "int":
		if v, err := strconv.ParseInt(str, 10, 64); err == nil {
			return v
		}
	case "float":
		if v, err := strconv.ParseFloat(str, 64); err == nil {
			return v
		}
	case "timestamp":
		// 尝试多种时间格式
		formats := []string{
			time.RFC3339,
			time.RFC3339Nano,
			"2006-01-02 15:04:05",
			"2006-01-02 15:04:05.000",
			"2006/01/02 15:04:05",
		}
		for _, format := range formats {
			if t, err := time.Parse(format, str); err == nil {
				return t.Format(time.RFC3339)
			}
		}
	}
	return str
}

// DetectLevel 检测日志级别
func (s *ParserService) DetectLevel(content string) string {
	return detectLogLevel(content)
}

// TestTemplate 测试解析模板
func (s *ParserService) TestTemplate(req *dto.ParseTestRequest) *dto.ParseTestResponse {
	template := &models.LogParseTemplate{
		Type:    req.Type,
		Pattern: req.Pattern,
	}

	parsed, err := s.ParseWithTemplate(req.LogContent, template)
	if err != nil {
		return &dto.ParseTestResponse{
			Success: false,
			Error:   err.Error(),
		}
	}

	if len(parsed) == 0 {
		return &dto.ParseTestResponse{
			Success: false,
			Error:   "未能解析出任何字段",
		}
	}

	return &dto.ParseTestResponse{
		Success: true,
		Parsed:  parsed,
	}
}
