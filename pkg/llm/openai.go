package llm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// OpenAIClient OpenAI兼容的LLM客户端
type OpenAIClient struct {
	config     Config
	httpClient *http.Client
}

// NewOpenAIClient 创建OpenAI客户端
func NewOpenAIClient(cfg Config) (*OpenAIClient, error) {
	// 对于流式响应，不设置整体超时，而是依赖连接和读取超时
	transport := &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 10,
		IdleConnTimeout:     90 * time.Second,
	}

	return &OpenAIClient{
		config: cfg,
		httpClient: &http.Client{
			Transport: transport,
			// 不设置 Timeout，流式响应可能需要很长时间
		},
	}, nil
}

// openAIRequest OpenAI API请求格式
type openAIRequest struct {
	Model       string        `json:"model"`
	Messages    []ChatMessage `json:"messages"`
	Tools       []Tool        `json:"tools,omitempty"`
	Temperature float64       `json:"temperature,omitempty"`
	MaxTokens   int           `json:"max_tokens,omitempty"`
	Stream      bool          `json:"stream"`
}

// openAIResponse OpenAI API响应格式
type openAIResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index        int         `json:"index"`
		Message      ChatMessage `json:"message"`
		Delta        ChatMessage `json:"delta"`
		FinishReason string      `json:"finish_reason"`
	} `json:"choices"`
	Usage Usage `json:"usage"`
}

// openAIStreamResponse SSE流式响应格式
type openAIStreamResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index int `json:"index"`
		Delta struct {
			Role      string     `json:"role,omitempty"`
			Content   string     `json:"content,omitempty"`
			ToolCalls []ToolCall `json:"tool_calls,omitempty"`
		} `json:"delta"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage *Usage `json:"usage,omitempty"`
}

// Chat 发送流式聊天请求
func (c *OpenAIClient) Chat(ctx context.Context, req *ChatRequest) (<-chan StreamChunk, error) {
	ch := make(chan StreamChunk, 100)

	go func() {
		defer close(ch)

		// 构建请求
		apiReq := openAIRequest{
			Model:       c.getModel(req.Model),
			Messages:    req.Messages,
			Tools:       req.Tools,
			Temperature: c.getTemperature(req.Temperature),
			MaxTokens:   c.getMaxTokens(req.MaxTokens),
			Stream:      true,
		}

		body, err := json.Marshal(apiReq)
		if err != nil {
			ch <- StreamChunk{Type: ChunkTypeError, Error: fmt.Sprintf("marshal request: %v", err)}
			return
		}

		httpReq, err := http.NewRequestWithContext(ctx, "POST", c.getChatURL(), bytes.NewReader(body))
		if err != nil {
			ch <- StreamChunk{Type: ChunkTypeError, Error: fmt.Sprintf("create request: %v", err)}
			return
		}

		c.setHeaders(httpReq)

		resp, err := c.httpClient.Do(httpReq)
		if err != nil {
			ch <- StreamChunk{Type: ChunkTypeError, Error: fmt.Sprintf("send request: %v", err)}
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			bodyBytes, _ := io.ReadAll(resp.Body)
			ch <- StreamChunk{Type: ChunkTypeError, Error: fmt.Sprintf("API error %d: %s", resp.StatusCode, string(bodyBytes))}
			return
		}

		// 解析SSE流
		c.parseSSEStream(resp.Body, ch)
	}()

	return ch, nil
}

// parseSSEStream 解析SSE流
func (c *OpenAIClient) parseSSEStream(body io.Reader, ch chan<- StreamChunk) {
	scanner := bufio.NewScanner(body)
	// 增加 buffer 大小，防止长行被截断
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	var totalUsage Usage
	hasContent := false

	for scanner.Scan() {
		line := scanner.Text()

		// 跳过空行
		if line == "" {
			continue
		}

		// 解析SSE数据
		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")

		// 检查结束标记
		if data == "[DONE]" {
			ch <- StreamChunk{Type: ChunkTypeDone, Usage: &totalUsage}
			return
		}

		// 解析JSON
		var streamResp openAIStreamResponse
		if err := json.Unmarshal([]byte(data), &streamResp); err != nil {
			continue // 跳过无法解析的行
		}

		// 更新usage
		if streamResp.Usage != nil {
			totalUsage = *streamResp.Usage
		}

		// 处理choices
		for _, choice := range streamResp.Choices {
			// 内容
			if choice.Delta.Content != "" {
				hasContent = true
				ch <- StreamChunk{Type: ChunkTypeContent, Content: choice.Delta.Content}
			}

			// 工具调用
			for _, tc := range choice.Delta.ToolCalls {
				ch <- StreamChunk{Type: ChunkTypeToolCall, ToolCall: &tc}
			}

			// 完成 - 支持更多的结束原因
			if choice.FinishReason != "" {
				ch <- StreamChunk{Type: ChunkTypeDone, Usage: &totalUsage}
				return
			}
		}
	}

	// 检查扫描错误
	if err := scanner.Err(); err != nil {
		ch <- StreamChunk{Type: ChunkTypeError, Error: fmt.Sprintf("read stream: %v", err)}
		return
	}

	// 如果流正常结束但没有收到明确的结束标记，也发送 Done
	if hasContent {
		ch <- StreamChunk{Type: ChunkTypeDone, Usage: &totalUsage}
	}
}

// ChatSync 同步聊天请求
func (c *OpenAIClient) ChatSync(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	// 构建请求
	apiReq := openAIRequest{
		Model:       c.getModel(req.Model),
		Messages:    req.Messages,
		Tools:       req.Tools,
		Temperature: c.getTemperature(req.Temperature),
		MaxTokens:   c.getMaxTokens(req.MaxTokens),
		Stream:      false,
	}

	body, err := json.Marshal(apiReq)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.getChatURL(), bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	c.setHeaders(httpReq)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var apiResp openAIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	if len(apiResp.Choices) == 0 {
		return nil, fmt.Errorf("no choices in response")
	}

	return &ChatResponse{
		ID:        apiResp.ID,
		Model:     apiResp.Model,
		Message:   apiResp.Choices[0].Message,
		Usage:     apiResp.Usage,
		ToolCalls: apiResp.Choices[0].Message.ToolCalls,
	}, nil
}

// CountTokens 估算token数量（简单实现）
func (c *OpenAIClient) CountTokens(text string) int {
	// 简单估算：英文约4字符/token，中文约2字符/token
	// 这是一个粗略估算，实际应使用tiktoken等库
	chars := len(text)
	// 假设混合内容，平均3字符/token
	return (chars + 2) / 3
}

// getChatURL 获取聊天API URL
func (c *OpenAIClient) getChatURL() string {
	baseURL := strings.TrimSuffix(c.config.APIURL, "/")
	return baseURL + "/chat/completions"
}

// setHeaders 设置请求头
func (c *OpenAIClient) setHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")

	if c.config.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.config.APIKey)
	}
}

// getModel 获取模型名称
func (c *OpenAIClient) getModel(reqModel string) string {
	if reqModel != "" {
		return reqModel
	}
	if c.config.Model != "" {
		return c.config.Model
	}
	return "gpt-3.5-turbo"
}

// getTemperature 获取温度参数
func (c *OpenAIClient) getTemperature(reqTemp float64) float64 {
	if reqTemp > 0 {
		return reqTemp
	}
	if c.config.Temperature > 0 {
		return c.config.Temperature
	}
	return 0.7
}

// getMaxTokens 获取最大token数
func (c *OpenAIClient) getMaxTokens(reqMax int) int {
	maxTokens := reqMax
	if maxTokens <= 0 {
		maxTokens = c.config.MaxTokens
	}
	if maxTokens <= 0 {
		maxTokens = 4096
	}

	// 根据不同 provider 限制 max_tokens
	switch c.config.Provider {
	case "deepseek":
		// DeepSeek 限制 max_tokens 在 1-8192
		if maxTokens > 8192 {
			maxTokens = 8192
		}
	case "qwen":
		// 通义千问限制
		if maxTokens > 6000 {
			maxTokens = 6000
		}
	}

	return maxTokens
}
