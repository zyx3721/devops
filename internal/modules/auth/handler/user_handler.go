package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"devops/internal/config"
	"devops/internal/models"
	"devops/internal/service/user"
	"devops/pkg/dto"
	apperrors "devops/pkg/errors"
	"devops/pkg/ioc"
	"devops/pkg/logger"
	"devops/pkg/middleware"
	"devops/pkg/response"
	"devops/pkg/validator"
)

var userLog = logger.L().WithField("module", "user")

func init() {
	ioc.Api.RegisterContainer("UserHandler", &UserApiHandler{})
}

type UserApiHandler struct {
	handler *UserHandler
}

func (h *UserApiHandler) Init() error {
	cfg, _ := config.LoadConfig()
	db := cfg.GetDB()
	svc := user.NewUserService(db)
	h.handler = NewUserHandler(svc)

	root := cfg.Application.GinRootRouter().Group("users")
	root.Use(middleware.AuthMiddleware())
	h.Register(root)

	return nil
}

func (h *UserApiHandler) Register(r gin.IRouter) {
	r.GET("", h.handler.GetUserList)
	r.GET("/profile", h.handler.GetProfile)
	r.GET("/:id", h.handler.GetUserByID)
	r.POST("", middleware.RequireAdmin(), h.handler.CreateUser)
	r.PUT("/:id", middleware.RequireAdmin(), h.handler.UpdateUser)
	r.PUT("/:id/role", middleware.RequireAdmin(), h.handler.UpdateUserRole)
	r.PUT("/:id/status", middleware.RequireAdmin(), h.handler.UpdateUserStatus)
	r.DELETE("/:id", middleware.RequireAdmin(), h.handler.DeleteUser)
	r.POST("/change-password", h.handler.ChangePassword)
	r.POST("/:id/reset-password", middleware.RequireAdmin(), h.handler.ResetPassword)
}

type UserHandler struct {
	svc user.UserService
}

func NewUserHandler(svc user.UserService) *UserHandler {
	return &UserHandler{svc: svc}
}

// GetUserList godoc
// @Summary 获取用户列表
// @Description 分页获取用户列表，支持按用户名、角色、状态筛选
// @Tags 用户管理
// @Accept json
// @Produce json
// @Param username query string false "用户名"
// @Param role query string false "角色"
// @Param status query string false "状态"
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(20)
// @Success 200 {object} response.Response{data=response.PageData} "成功"
// @Failure 400 {object} response.Response "参数错误"
// @Failure 500 {object} response.Response "服务器错误"
// @Security BearerAuth
// @Router /users [get]
func (h *UserHandler) GetUserList(c *gin.Context) {
	var req dto.UserListRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 20
	}

	result, err := h.svc.GetUserList(c.Request.Context(), &req)
	if err != nil {
		response.InternalError(c, "查询用户列表失败")
		return
	}

	response.Success(c, result)
}

// GetUserByID godoc
// @Summary 获取用户详情
// @Description 根据ID获取用户详情
// @Tags 用户管理
// @Accept json
// @Produce json
// @Param id path int true "用户ID"
// @Success 200 {object} response.Response{data=dto.UserResponse} "成功"
// @Failure 400 {object} response.Response "参数错误"
// @Failure 404 {object} response.Response "用户不存在"
// @Security BearerAuth
// @Router /users/{id} [get]
func (h *UserHandler) GetUserByID(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, "ID格式错误")
		return
	}

	result, err := h.svc.GetUserByID(c.Request.Context(), uint(id))
	if err != nil {
		if err == apperrors.ErrNotFound {
			response.NotFound(c, "用户不存在")
			return
		}
		response.InternalError(c, "查询用户失败")
		return
	}

	response.Success(c, result)
}

// CreateUser godoc
// @Summary 创建用户
// @Description 创建新用户
// @Tags 用户管理
// @Accept json
// @Produce json
// @Param request body dto.CreateUserRequest true "创建用户请求"
// @Success 200 {object} response.Response{data=dto.UserResponse} "创建成功"
// @Failure 400 {object} response.Response "参数错误"
// @Failure 500 {object} response.Response "服务器错误"
// @Security BearerAuth
// @Router /users [post]
func (h *UserHandler) CreateUser(c *gin.Context) {
	var req dto.CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	// 使用验证器验证
	if ok, msg := validator.ValidateAndFormat(&req); !ok {
		userLog.Warn("创建用户参数错误: %s", msg)
		response.BadRequest(c, msg)
		return
	}

	result, err := h.svc.CreateUser(c.Request.Context(), &req)
	if err != nil {
		userLog.WithError(err).Error("创建用户失败: %s", req.Username)
		response.FromErrorWithDefault(c, err, "创建用户失败")
		return
	}

	userLog.Info("创建用户成功: %s (ID: %d)", req.Username, result.ID)
	response.SuccessWithMessage(c, "创建成功", result)
}

func (h *UserHandler) UpdateUser(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, "ID格式错误")
		return
	}

	var req dto.UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	result, err := h.svc.UpdateUser(c.Request.Context(), uint(id), &req)
	if err != nil {
		userLog.WithError(err).Error("更新用户失败: ID=%d", id)
		response.FromErrorWithDefault(c, err, "更新用户失败")
		return
	}

	userLog.Info("更新用户成功: %s (ID: %d)", result.Username, id)
	response.SuccessWithMessage(c, "更新成功", result)
}

func (h *UserHandler) UpdateUserRole(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, "ID格式错误")
		return
	}

	// 获取目标用户信息
	targetUser, err := h.svc.GetUserByID(c.Request.Context(), uint(id))
	if err != nil {
		if err == apperrors.ErrNotFound {
			response.NotFound(c, "用户不存在")
			return
		}
		response.InternalError(c, "查询用户失败")
		return
	}

	// 检查是否是受保护用户
	if models.IsProtectedUser(uint(id), targetUser.Role) {
		response.Forbidden(c, "无法修改超级管理员的角色")
		return
	}

	// 检查操作者权限
	operatorRole, _ := middleware.GetRole(c)
	if !models.CanManageRole(operatorRole, targetUser.Role) {
		response.Forbidden(c, "无权修改该用户的角色")
		return
	}

	var req dto.UpdateUserRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	// 检查是否试图设置为更高权限的角色
	if !models.CanManageRole(operatorRole, req.Role) {
		response.Forbidden(c, "无法设置高于自己权限的角色")
		return
	}

	result, err := h.svc.UpdateUserRole(c.Request.Context(), uint(id), &req)
	if err != nil {
		if err == apperrors.ErrNotFound {
			response.NotFound(c, "用户不存在")
			return
		}
		response.InternalError(c, "更新用户角色失败")
		return
	}

	response.SuccessWithMessage(c, "更新成功", result)
}

func (h *UserHandler) UpdateUserStatus(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, "ID格式错误")
		return
	}

	// 获取目标用户信息
	targetUser, err := h.svc.GetUserByID(c.Request.Context(), uint(id))
	if err != nil {
		if err == apperrors.ErrNotFound {
			response.NotFound(c, "用户不存在")
			return
		}
		response.InternalError(c, "查询用户失败")
		return
	}

	// 检查是否是受保护用户
	if models.IsProtectedUser(uint(id), targetUser.Role) {
		response.Forbidden(c, "无法修改超级管理员的状态")
		return
	}

	// 检查操作者权限
	operatorRole, _ := middleware.GetRole(c)
	if !models.CanManageRole(operatorRole, targetUser.Role) {
		response.Forbidden(c, "无权修改该用户的状态")
		return
	}

	var req dto.UpdateUserStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	result, err := h.svc.UpdateUserStatus(c.Request.Context(), uint(id), &req)
	if err != nil {
		if err == apperrors.ErrNotFound {
			response.NotFound(c, "用户不存在")
			return
		}
		response.InternalError(c, "更新用户状态失败")
		return
	}

	response.SuccessWithMessage(c, "更新成功", result)
}

func (h *UserHandler) DeleteUser(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, "ID格式错误")
		return
	}

	// 获取目标用户信息
	targetUser, err := h.svc.GetUserByID(c.Request.Context(), uint(id))
	if err != nil {
		if err == apperrors.ErrNotFound {
			response.NotFound(c, "用户不存在")
			return
		}
		response.InternalError(c, "查询用户失败")
		return
	}

	// 检查是否是受保护用户
	if models.IsProtectedUser(uint(id), targetUser.Role) {
		response.Forbidden(c, "无法删除超级管理员")
		return
	}

	// 检查操作者权限
	operatorRole, _ := middleware.GetRole(c)
	if !models.CanManageRole(operatorRole, targetUser.Role) {
		response.Forbidden(c, "无权删除该用户")
		return
	}

	// 不能删除自己
	operatorID, _ := middleware.GetUserID(c)
	if operatorID == uint(id) {
		response.Forbidden(c, "不能删除自己")
		return
	}

	if err := h.svc.DeleteUser(c.Request.Context(), uint(id)); err != nil {
		userLog.WithError(err).Error("删除用户失败: ID=%d", id)
		response.FromErrorWithDefault(c, err, "删除用户失败")
		return
	}

	userLog.Info("删除用户成功: ID=%d", id)
	response.OKWithMessage(c, "删除成功")
}

func (h *UserHandler) GetProfile(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		response.Unauthorized(c, "未登录")
		return
	}

	result, err := h.svc.GetProfile(c.Request.Context(), userID)
	if err != nil {
		if err == apperrors.ErrNotFound {
			response.NotFound(c, "用户不存在")
			return
		}
		response.InternalError(c, "获取个人资料失败")
		return
	}

	response.Success(c, result)
}

func (h *UserHandler) ChangePassword(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		response.Unauthorized(c, "未登录")
		return
	}

	var req dto.ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	if err := h.svc.ChangePassword(c.Request.Context(), userID, &req); err != nil {
		userLog.WithError(err).Error("修改密码失败: UserID=%d", userID)
		response.FromErrorWithDefault(c, err, "修改密码失败")
		return
	}

	userLog.Info("修改密码成功: UserID=%d", userID)
	response.OKWithMessage(c, "密码修改成功")
}

// ResetPassword 重置用户密码（管理员/超管使用）
func (h *UserHandler) ResetPassword(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, "ID格式错误")
		return
	}

	// 获取目标用户信息
	targetUser, err := h.svc.GetUserByID(c.Request.Context(), uint(id))
	if err != nil {
		if err == apperrors.ErrNotFound {
			response.NotFound(c, "用户不存在")
			return
		}
		response.InternalError(c, "查询用户失败")
		return
	}

	// 检查是否是受保护用户（超级管理员）
	if models.IsProtectedUser(uint(id), targetUser.Role) {
		response.Forbidden(c, "无法重置超级管理员的密码")
		return
	}

	// 检查操作者权限
	operatorRole, _ := middleware.GetRole(c)
	if !models.CanManageRole(operatorRole, targetUser.Role) {
		response.Forbidden(c, "无权重置该用户的密码")
		return
	}

	// 不能重置自己的密码（应该用修改密码功能）
	operatorID, _ := middleware.GetUserID(c)
	if operatorID == uint(id) {
		response.Forbidden(c, "不能重置自己的密码，请使用修改密码功能")
		return
	}

	var req dto.ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	if err := h.svc.ResetPassword(c.Request.Context(), uint(id), &req); err != nil {
		userLog.WithError(err).Error("重置密码失败: UserID=%d", id)
		response.FromErrorWithDefault(c, err, "重置密码失败")
		return
	}

	userLog.Info("重置密码成功: UserID=%d, 操作者=%d", id, operatorID)
	response.OKWithMessage(c, "密码重置成功")
}
