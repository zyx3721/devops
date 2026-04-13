package user

import (
	"context"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"devops/internal/models"
	"devops/pkg/dto"
	apperrors "devops/pkg/errors"
	"devops/pkg/logger"
)

type UserService interface {
	GetUserList(ctx context.Context, req *dto.UserListRequest) (*dto.UserListResponse, error)
	GetUserByID(ctx context.Context, userID uint) (*dto.UserResponse, error)
	CreateUser(ctx context.Context, req *dto.CreateUserRequest) (*dto.UserResponse, error)
	UpdateUser(ctx context.Context, userID uint, req *dto.UpdateUserRequest) (*dto.UserResponse, error)
	UpdateUserRole(ctx context.Context, userID uint, req *dto.UpdateUserRoleRequest) (*dto.UserResponse, error)
	UpdateUserStatus(ctx context.Context, userID uint, req *dto.UpdateUserStatusRequest) (*dto.UserResponse, error)
	DeleteUser(ctx context.Context, userID uint) error
	GetProfile(ctx context.Context, userID uint) (*dto.UserResponse, error)
	ChangePassword(ctx context.Context, userID uint, req *dto.ChangePasswordRequest) error
	ResetPassword(ctx context.Context, userID uint, req *dto.ResetPasswordRequest) error
}

type userService struct {
	db  *gorm.DB
	log *logger.Logger
}

func NewUserService(db *gorm.DB) UserService {
	return &userService{db: db, log: logger.NewLogger("info")}
}

func (s *userService) GetUserList(ctx context.Context, req *dto.UserListRequest) (*dto.UserListResponse, error) {
	query := s.db.Model(&models.User{})

	if req.Keyword != "" {
		keyword := "%" + req.Keyword + "%"
		query = query.Where("username LIKE ? OR email LIKE ? OR phone LIKE ?", keyword, keyword, keyword)
	}
	if req.Role != "" {
		query = query.Where("role = ?", req.Role)
	}
	if req.Status != "" {
		query = query.Where("status = ?", req.Status)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, apperrors.Wrap(err, apperrors.ErrCodeInternalError, "查询用户总数失败")
	}

	var users []models.User
	offset := (req.Page - 1) * req.PageSize
	if err := query.Offset(offset).Limit(req.PageSize).Order("created_at DESC").Find(&users).Error; err != nil {
		return nil, apperrors.Wrap(err, apperrors.ErrCodeInternalError, "查询用户列表失败")
	}

	items := make([]dto.UserResponse, len(users))
	for i, user := range users {
		items[i] = s.buildUserResponse(&user)
	}

	totalPages := int(total) / req.PageSize
	if int(total)%req.PageSize != 0 {
		totalPages++
	}

	return &dto.UserListResponse{Items: items, Total: total, Page: req.Page, PageSize: req.PageSize, TotalPages: totalPages}, nil
}

func (s *userService) GetUserByID(ctx context.Context, userID uint) (*dto.UserResponse, error) {
	var user models.User
	if err := s.db.First(&user, userID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, apperrors.ErrNotFound
		}
		return nil, apperrors.Wrap(err, apperrors.ErrCodeInternalError, "查询用户失败")
	}
	resp := s.buildUserResponse(&user)
	return &resp, nil
}

func (s *userService) CreateUser(ctx context.Context, req *dto.CreateUserRequest) (*dto.UserResponse, error) {
	// 检查用户名是否存在（包括软删除的记录）
	var existing models.User
	if err := s.db.Unscoped().Where("username = ?", req.Username).First(&existing).Error; err == nil {
		return nil, apperrors.New(apperrors.ErrCodeDuplicate, "用户名已存在")
	}

	// 检查邮箱是否存在
	if req.Email != "" {
		if err := s.db.Unscoped().Where("email = ?", req.Email).First(&existing).Error; err == nil {
			return nil, apperrors.New(apperrors.ErrCodeDuplicate, "邮箱已被使用")
		}
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, apperrors.New(apperrors.ErrCodeInternalError, "密码加密失败")
	}

	user := &models.User{
		Username: req.Username, Password: string(hashedPassword), Email: req.Email,
		Phone: req.Phone, Role: req.Role, Status: req.Status,
	}

	if err := s.db.Create(&user).Error; err != nil {
		// 处理数据库唯一约束错误
		if strings.Contains(err.Error(), "Duplicate entry") {
			if strings.Contains(err.Error(), "username") {
				return nil, apperrors.New(apperrors.ErrCodeDuplicate, "用户名已存在")
			}
			if strings.Contains(err.Error(), "email") {
				return nil, apperrors.New(apperrors.ErrCodeDuplicate, "邮箱已被使用")
			}
			return nil, apperrors.New(apperrors.ErrCodeDuplicate, "数据重复")
		}
		return nil, apperrors.New(apperrors.ErrCodeInternalError, "创建用户失败: "+err.Error())
	}

	resp := s.buildUserResponse(user)
	return &resp, nil
}

func (s *userService) UpdateUser(ctx context.Context, userID uint, req *dto.UpdateUserRequest) (*dto.UserResponse, error) {
	var user models.User
	if err := s.db.First(&user, userID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, apperrors.ErrNotFound
		}
		return nil, apperrors.Wrap(err, apperrors.ErrCodeInternalError, "查询用户失败")
	}

	if req.Email != "" && req.Email != user.Email {
		var existing models.User
		if err := s.db.Where("email = ? AND id != ?", req.Email, userID).First(&existing).Error; err == nil {
			return nil, apperrors.New(apperrors.ErrCodeInvalidParams, "邮箱已被使用")
		}
	}

	updates := make(map[string]any)
	if req.Email != "" {
		updates["email"] = req.Email
		user.Email = req.Email
	}
	if req.Phone != "" {
		updates["phone"] = req.Phone
		user.Phone = req.Phone
	}
	if req.Status != "" {
		updates["status"] = req.Status
		user.Status = req.Status
	}

	if len(updates) > 0 {
		if err := s.db.Model(&user).Updates(updates).Error; err != nil {
			return nil, apperrors.Wrap(err, apperrors.ErrCodeInternalError, "更新用户失败")
		}
	}

	resp := s.buildUserResponse(&user)
	return &resp, nil
}

func (s *userService) UpdateUserRole(ctx context.Context, userID uint, req *dto.UpdateUserRoleRequest) (*dto.UserResponse, error) {
	var user models.User
	if err := s.db.First(&user, userID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, apperrors.ErrNotFound
		}
		return nil, apperrors.Wrap(err, apperrors.ErrCodeInternalError, "查询用户失败")
	}

	user.Role = req.Role
	if err := s.db.Save(&user).Error; err != nil {
		return nil, apperrors.Wrap(err, apperrors.ErrCodeInternalError, "更新用户角色失败")
	}

	resp := s.buildUserResponse(&user)
	return &resp, nil
}

func (s *userService) UpdateUserStatus(ctx context.Context, userID uint, req *dto.UpdateUserStatusRequest) (*dto.UserResponse, error) {
	var user models.User
	if err := s.db.First(&user, userID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, apperrors.ErrNotFound
		}
		return nil, apperrors.Wrap(err, apperrors.ErrCodeInternalError, "查询用户失败")
	}

	user.Status = req.Status
	if err := s.db.Save(&user).Error; err != nil {
		return nil, apperrors.Wrap(err, apperrors.ErrCodeInternalError, "更新用户状态失败")
	}

	resp := s.buildUserResponse(&user)
	return &resp, nil
}

func (s *userService) DeleteUser(ctx context.Context, userID uint) error {
	var user models.User
	if err := s.db.First(&user, userID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return apperrors.ErrNotFound
		}
		return apperrors.Wrap(err, apperrors.ErrCodeInternalError, "查询用户失败")
	}

	if err := s.db.Delete(&user).Error; err != nil {
		return apperrors.Wrap(err, apperrors.ErrCodeInternalError, "删除用户失败")
	}
	return nil
}

func (s *userService) GetProfile(ctx context.Context, userID uint) (*dto.UserResponse, error) {
	var user models.User
	if err := s.db.First(&user, userID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, apperrors.ErrNotFound
		}
		return nil, apperrors.Wrap(err, apperrors.ErrCodeInternalError, "查询用户失败")
	}
	resp := s.buildUserResponse(&user)
	return &resp, nil
}

func (s *userService) ChangePassword(ctx context.Context, userID uint, req *dto.ChangePasswordRequest) error {
	var user models.User
	if err := s.db.First(&user, userID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return apperrors.ErrNotFound
		}
		return apperrors.Wrap(err, apperrors.ErrCodeInternalError, "查询用户失败")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.OldPassword)); err != nil {
		return apperrors.New(apperrors.ErrCodeInvalidParams, "原密码错误")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return apperrors.Wrap(err, apperrors.ErrCodeInternalError, "密码加密失败")
	}

	if err := s.db.Model(&user).Update("password", string(hashedPassword)).Error; err != nil {
		return apperrors.Wrap(err, apperrors.ErrCodeInternalError, "更新密码失败")
	}
	return nil
}

func (s *userService) ResetPassword(ctx context.Context, userID uint, req *dto.ResetPasswordRequest) error {
	var user models.User
	if err := s.db.First(&user, userID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return apperrors.ErrNotFound
		}
		return apperrors.Wrap(err, apperrors.ErrCodeInternalError, "查询用户失败")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return apperrors.Wrap(err, apperrors.ErrCodeInternalError, "密码加密失败")
	}

	if err := s.db.Model(&user).Update("password", string(hashedPassword)).Error; err != nil {
		return apperrors.Wrap(err, apperrors.ErrCodeInternalError, "重置密码失败")
	}
	return nil
}

func (s *userService) UpdateLastLogin(ctx context.Context, userID uint) error {
	now := time.Now()
	return s.db.Model(&models.User{}).Where("id = ?", userID).Update("last_login_at", &now).Error
}

func (s *userService) buildUserResponse(user *models.User) dto.UserResponse {
	lastLoginAt := ""
	if user.LastLoginAt.Valid {
		lastLoginAt = user.LastLoginAt.Time.Format("2006-01-02 15:04:05")
	}
	return dto.UserResponse{
		ID: user.ID, Username: user.Username, Email: user.Email, Phone: user.Phone,
		Role: user.Role, Status: user.Status, LastLoginAt: lastLoginAt,
		CreatedAt: user.CreatedAt.Format("2006-01-02 15:04:05"), UpdatedAt: user.UpdatedAt.Format("2006-01-02 15:04:05"),
	}
}
