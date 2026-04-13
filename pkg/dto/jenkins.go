package dto

// TriggerBuildRequest 触发构建请求
type TriggerBuildRequest struct {
	JobName            string `json:"job_name" binding:"required"`
	GitlabSourceBranch string `json:"gitlab_source_branch"`
	ChangeType         string `json:"change_type"`
	Branch             string `json:"branch"`
	DeployType         string `json:"deploy_type"`
	ImageVersion       string `json:"image_version"`
}

// TriggerBuildResponse 触发构建响应
type TriggerBuildResponse struct {
	Success     bool   `json:"success"`
	Message     string `json:"message"`
	JobName     string `json:"job_name"`
	BuildNumber int64  `json:"build_number"`
}

// JobInfoResponse Job信息响应
type JobInfoResponse struct {
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	URL         string `json:"url"`
	Buildable   bool   `json:"buildable"`
}

// BuildStatusResponse 构建状态响应
type BuildStatusResponse struct {
	JobName     string `json:"job_name"`
	BuildNumber int64  `json:"build_number"`
	Status      string `json:"status"`
	Timestamp   int64  `json:"timestamp"`
}

// BuildConsoleResponse 构建日志响应
type BuildConsoleResponse struct {
	JobName     string `json:"job_name"`
	BuildNumber int64  `json:"build_number"`
	Log         string `json:"log"`
}

// CreateJenkinsInstanceRequest 创建Jenkins实例请求
type CreateJenkinsInstanceRequest struct {
	Name        string `json:"name" binding:"required,min=1,max=100"`
	URL         string `json:"url" binding:"required,url,min=1,max=500"`
	Username    string `json:"username" binding:"max=100"`
	APIToken    string `json:"api_token" binding:"max=500"`
	Description string `json:"description"`
	Status      string `json:"status" binding:"required,oneof=active inactive"`
	IsDefault   bool   `json:"is_default"`
}

// UpdateJenkinsInstanceRequest 更新Jenkins实例请求
type UpdateJenkinsInstanceRequest struct {
	Name        string `json:"name" binding:"required,min=1,max=100"`
	URL         string `json:"url" binding:"required,url,min=1,max=500"`
	Username    string `json:"username" binding:"max=100"`
	APIToken    string `json:"api_token" binding:"max=500"`
	Description string `json:"description"`
	Status      string `json:"status" binding:"required,oneof=active inactive"`
	IsDefault   bool   `json:"is_default"`
}

// JenkinsInstanceResponse Jenkins实例响应
type JenkinsInstanceResponse struct {
	ID          uint   `json:"id"`
	Name        string `json:"name"`
	URL         string `json:"url"`
	Username    string `json:"username"`
	Description string `json:"description"`
	Status      string `json:"status"`
	IsDefault   bool   `json:"is_default"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

// JenkinsInstanceListRequest Jenkins实例列表请求
type JenkinsInstanceListRequest struct {
	Page     int    `form:"page,default=1"`
	PageSize int    `form:"page_size,default=20"`
	Keyword  string `form:"keyword"`
	Status   string `form:"status"`
}

// JenkinsInstanceListResponse Jenkins实例列表响应
type JenkinsInstanceListResponse struct {
	Total      int64                     `json:"total"`
	Page       int                       `json:"page"`
	PageSize   int                       `json:"page_size"`
	TotalPages int                       `json:"total_pages"`
	Items      []JenkinsInstanceResponse `json:"items"`
}

// FeishuAppSimple 飞书应用简单信息
type FeishuAppSimple struct {
	ID      uint   `json:"id"`
	Name    string `json:"name"`
	AppID   string `json:"app_id"`
	Project string `json:"project"`
}

// BindFeishuAppsRequest 绑定飞书应用请求
type BindFeishuAppsRequest struct {
	AppIDs []uint `json:"app_ids"`
}

// JenkinsJob Jenkins Job信息
type JenkinsJob struct {
	Name            string `json:"name"`
	URL             string `json:"url"`
	Color           string `json:"color"`
	Class           string `json:"class"`
	LastBuildNumber int64  `json:"last_build_number"`
	LastBuildResult string `json:"last_build_result"`
	LastBuildTime   string `json:"last_build_time"`
}

// JenkinsBuild Jenkins 构建信息
type JenkinsBuild struct {
	Number    int64  `json:"number"`
	Result    string `json:"result"`
	Building  bool   `json:"building"`
	Timestamp string `json:"timestamp"`
	Duration  int64  `json:"duration"`
	URL       string `json:"url"`
}
