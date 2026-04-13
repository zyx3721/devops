package dto

// PageRequest 分页请求
type PageRequest struct {
	Page     int `json:"page" form:"page"`
	PageSize int `json:"pageSize" form:"pageSize"`
}

// PageResponse 分页响应
type PageResponse struct {
	Total      int64 `json:"total"`
	Page       int   `json:"page"`
	PageSize   int   `json:"pageSize"`
	TotalPages int   `json:"totalPages"`
}

// Response 通用响应
type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// ConnectionTestResult 连接测试结果
type ConnectionTestResult struct {
	Connected      bool   `json:"connected"`
	Version        string `json:"version,omitempty"`
	ServerVersion  string `json:"server_version,omitempty"`
	NodeCount      int    `json:"node_count,omitempty"`
	ResponseTimeMs int64  `json:"response_time_ms"`
	Error          string `json:"error,omitempty"`
}
