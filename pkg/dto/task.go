package dto

// CreateTaskRequest 创建任务请求
type CreateTaskRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
	JenkinsJob  string `json:"jenkins_job"`
	Parameters  string `json:"parameters"`
}

// UpdateTaskRequest 更新任务请求
type UpdateTaskRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Status      string `json:"status"`
	JenkinsJob  string `json:"jenkins_job"`
	Parameters  string `json:"parameters"`
}

// GetTaskListRequest 获取任务列表请求
type GetTaskListRequest struct {
	Page     int    `form:"page,default=1"`
	PageSize int    `form:"page_size,default=20"`
	Status   string `form:"status"`
}

// TaskResponse 任务响应
type TaskResponse struct {
	ID          uint   `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Status      string `json:"status"`
	CreatedBy   uint   `json:"created_by"`
	StartTime   string `json:"start_time"`
	EndTime     string `json:"end_time"`
	JenkinsJob  string `json:"jenkins_job"`
	Parameters  string `json:"parameters"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

// TaskListResponse 任务列表响应
type TaskListResponse struct {
	Total      int64          `json:"total"`
	Page       int            `json:"page"`
	PageSize   int            `json:"page_size"`
	TotalPages int            `json:"total_pages"`
	Items      []TaskResponse `json:"items"`
}
