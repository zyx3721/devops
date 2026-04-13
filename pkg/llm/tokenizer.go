package llm

import (
	"sync"
	"unicode"
	"unicode/utf8"
)

// TokenCounter Token计数器
type TokenCounter struct {
	mu          sync.RWMutex
	totalTokens int64
	requests    int64
}

// NewTokenCounter 创建Token计数器
func NewTokenCounter() *TokenCounter {
	return &TokenCounter{}
}

// Add 添加token使用量
func (tc *TokenCounter) Add(tokens int) {
	tc.mu.Lock()
	defer tc.mu.Unlock()
	tc.totalTokens += int64(tokens)
	tc.requests++
}

// GetTotal 获取总token数
func (tc *TokenCounter) GetTotal() int64 {
	tc.mu.RLock()
	defer tc.mu.RUnlock()
	return tc.totalTokens
}

// GetRequests 获取请求数
func (tc *TokenCounter) GetRequests() int64 {
	tc.mu.RLock()
	defer tc.mu.RUnlock()
	return tc.requests
}

// GetStats 获取统计信息
func (tc *TokenCounter) GetStats() TokenStats {
	tc.mu.RLock()
	defer tc.mu.RUnlock()
	return TokenStats{
		TotalTokens: tc.totalTokens,
		Requests:    tc.requests,
	}
}

// Reset 重置计数器
func (tc *TokenCounter) Reset() {
	tc.mu.Lock()
	defer tc.mu.Unlock()
	tc.totalTokens = 0
	tc.requests = 0
}

// TokenStats Token统计信息
type TokenStats struct {
	TotalTokens int64 `json:"total_tokens"`
	Requests    int64 `json:"requests"`
}

// EstimateTokens 估算文本的token数量
// 这是一个简化的估算方法，实际应使用tiktoken等库
func EstimateTokens(text string) int {
	if text == "" {
		return 0
	}

	// 统计字符类型
	var (
		asciiChars   int
		chineseChars int
		otherChars   int
	)

	for _, r := range text {
		if r <= 127 {
			asciiChars++
		} else if unicode.Is(unicode.Han, r) {
			chineseChars++
		} else {
			otherChars++
		}
	}

	// 估算规则：
	// - ASCII字符：约4字符/token
	// - 中文字符：约1.5字符/token
	// - 其他字符：约2字符/token
	tokens := (asciiChars + 3) / 4
	tokens += (chineseChars*2 + 2) / 3
	tokens += (otherChars + 1) / 2

	// 至少返回1
	if tokens < 1 && utf8.RuneCountInString(text) > 0 {
		tokens = 1
	}

	return tokens
}

// EstimateMessagesTokens 估算消息列表的token数量
func EstimateMessagesTokens(messages []ChatMessage) int {
	total := 0
	for _, msg := range messages {
		// 每条消息有额外的格式开销（约4 tokens）
		total += 4
		total += EstimateTokens(msg.Role)
		total += EstimateTokens(msg.Content)
		if msg.Name != "" {
			total += EstimateTokens(msg.Name)
		}
	}
	// 对话格式开销
	total += 3
	return total
}

// TokenUsageTracker Token使用追踪器
type TokenUsageTracker struct {
	mu      sync.RWMutex
	records []TokenUsageRecord
	maxSize int
}

// TokenUsageRecord Token使用记录
type TokenUsageRecord struct {
	ConversationID   string `json:"conversation_id"`
	MessageID        string `json:"message_id"`
	PromptTokens     int    `json:"prompt_tokens"`
	CompletionTokens int    `json:"completion_tokens"`
	TotalTokens      int    `json:"total_tokens"`
	Model            string `json:"model"`
	Timestamp        int64  `json:"timestamp"`
}

// NewTokenUsageTracker 创建Token使用追踪器
func NewTokenUsageTracker(maxSize int) *TokenUsageTracker {
	if maxSize <= 0 {
		maxSize = 1000
	}
	return &TokenUsageTracker{
		records: make([]TokenUsageRecord, 0, maxSize),
		maxSize: maxSize,
	}
}

// Record 记录token使用
func (t *TokenUsageTracker) Record(record TokenUsageRecord) {
	t.mu.Lock()
	defer t.mu.Unlock()

	// 如果超过最大容量，移除最旧的记录
	if len(t.records) >= t.maxSize {
		t.records = t.records[1:]
	}
	t.records = append(t.records, record)
}

// GetRecords 获取所有记录
func (t *TokenUsageTracker) GetRecords() []TokenUsageRecord {
	t.mu.RLock()
	defer t.mu.RUnlock()
	result := make([]TokenUsageRecord, len(t.records))
	copy(result, t.records)
	return result
}

// GetByConversation 获取会话的token使用记录
func (t *TokenUsageTracker) GetByConversation(conversationID string) []TokenUsageRecord {
	t.mu.RLock()
	defer t.mu.RUnlock()

	var result []TokenUsageRecord
	for _, r := range t.records {
		if r.ConversationID == conversationID {
			result = append(result, r)
		}
	}
	return result
}

// GetTotalUsage 获取总使用量
func (t *TokenUsageTracker) GetTotalUsage() TokenStats {
	t.mu.RLock()
	defer t.mu.RUnlock()

	var total int64
	for _, r := range t.records {
		total += int64(r.TotalTokens)
	}
	return TokenStats{
		TotalTokens: total,
		Requests:    int64(len(t.records)),
	}
}

// Clear 清空记录
func (t *TokenUsageTracker) Clear() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.records = t.records[:0]
}

// GlobalTokenCounter 全局Token计数器
var GlobalTokenCounter = NewTokenCounter()

// GlobalTokenTracker 全局Token追踪器
var GlobalTokenTracker = NewTokenUsageTracker(10000)
