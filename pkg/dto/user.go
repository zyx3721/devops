package dto

// UserListRequest 用户列表请求
type UserListRequest struct {
	Page     int    `form:"page,default=1"`
	PageSize int    `form:"page_size,default=20"`
	Keyword  string `form:"keyword"`
	Role     string `form:"role"`
	Status   string `form:"status"`
}

// UserListResponse 用户列表响应
type UserListResponse struct {
	Items      []UserResponse `json:"items"`
	Total      int64          `json:"total"`
	Page       int            `json:"page"`
	PageSize   int            `json:"page_size"`
	TotalPages int            `json:"total_pages"`
}

// CreateUserRequest 创建用户请求
type CreateUserRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
	Email    string `json:"email" binding:"required,email"`
	Phone    string `json:"phone"`
	Role     string `json:"role" binding:"required,oneof=admin user viewer"`
	Status   string `json:"status" binding:"required,oneof=active inactive"`
}

// UpdateUserRequest 更新用户请求
type UpdateUserRequest struct {
	Email  string `json:"email"`
	Phone  string `json:"phone"`
	Status string `json:"status"`
}

// UpdateUserRoleRequest 更新用户角色请求
type UpdateUserRoleRequest struct {
	Role string `json:"role" binding:"required,oneof=admin user viewer"`
}

// UpdateUserStatusRequest 更新用户状态请求
type UpdateUserStatusRequest struct {
	Status string `json:"status" binding:"required,oneof=active inactive"`
}

// ChangePasswordRequest 修改密码请求
type ChangePasswordRequest struct {
	OldPassword string `json:"old_password" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=6"`
}

// ResetPasswordRequest 重置密码请求（管理员使用）
type ResetPasswordRequest struct {
	NewPassword string `json:"new_password" binding:"required,min=6"`
}
