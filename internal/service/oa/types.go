package oa

// APIResponse API响应结构
type APIResponse struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// StoredJSON 存储的JSON数据
type StoredJSON struct {
	ID           string                 `json:"id"`
	ReceivedAt   string                 `json:"received_at"`
	IPAddress    string                 `json:"ip_address"`
	UserAgent    string                 `json:"user_agent"`
	OriginalData map[string]interface{} `json:"original_data"`
}
