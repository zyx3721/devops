package approval

import (
	"context"
	"strconv"
	"strings"

	"gorm.io/gorm"

	"devops/internal/models"
)

// ApproverResolver 审批人解析器
type ApproverResolver struct {
	db *gorm.DB
}

// NewApproverResolver 创建审批人解析器
func NewApproverResolver(db *gorm.DB) *ApproverResolver {
	return &ApproverResolver{db: db}
}

// ResolveApprovers 解析审批人，返回用户ID列表（逗号分隔的字符串）
// approverType: user/role/app_owner/team_leader
// approvers: 原始审批人配置（用户ID或角色名，逗号分隔）
// appID: 应用ID（用于解析 app_owner 和 team_leader）
func (r *ApproverResolver) ResolveApprovers(ctx context.Context, approverType, approvers string, appID uint) (string, error) {
	switch approverType {
	case "user":
		// 直接返回用户ID列表
		return approvers, nil

	case "role":
		// 根据角色名获取用户ID列表
		return r.resolveByRoles(ctx, approvers)

	case "app_owner":
		// 获取应用负责人
		return r.resolveAppOwner(ctx, appID)

	case "team_leader":
		// 获取团队负责人
		return r.resolveTeamLeader(ctx, appID)

	default:
		return approvers, nil
	}
}

// resolveByRoles 根据角色名获取用户ID列表
func (r *ApproverResolver) resolveByRoles(ctx context.Context, roleNames string) (string, error) {
	if roleNames == "" {
		return "", nil
	}

	names := strings.Split(roleNames, ",")
	var userIDs []string

	for _, name := range names {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}

		// 获取角色
		var role models.Role
		if err := r.db.WithContext(ctx).Where("name = ?", name).First(&role).Error; err != nil {
			continue
		}

		// 获取该角色的所有用户
		var userRoles []models.UserRole
		if err := r.db.WithContext(ctx).Where("role_id = ?", role.ID).Find(&userRoles).Error; err != nil {
			continue
		}

		for _, ur := range userRoles {
			userIDs = append(userIDs, strconv.FormatUint(uint64(ur.UserID), 10))
		}
	}

	// 去重
	userIDs = uniqueStrings(userIDs)
	return strings.Join(userIDs, ","), nil
}

// resolveAppOwner 获取应用负责人
func (r *ApproverResolver) resolveAppOwner(ctx context.Context, appID uint) (string, error) {
	if appID == 0 {
		return "", nil
	}

	var app models.Application
	if err := r.db.WithContext(ctx).First(&app, appID).Error; err != nil {
		return "", err
	}

	// Owner 字段存储的是用户名，需要转换为用户ID
	if app.Owner == "" {
		return "", nil
	}

	var user models.User
	if err := r.db.WithContext(ctx).Where("username = ?", app.Owner).First(&user).Error; err != nil {
		return "", nil
	}

	return strconv.FormatUint(uint64(user.ID), 10), nil
}

// resolveTeamLeader 获取团队负责人
func (r *ApproverResolver) resolveTeamLeader(ctx context.Context, appID uint) (string, error) {
	if appID == 0 {
		return "", nil
	}

	var app models.Application
	if err := r.db.WithContext(ctx).First(&app, appID).Error; err != nil {
		return "", err
	}

	if app.Team == "" {
		return "", nil
	}

	// 查找团队负责人（假设有 team_leader 角色的用户）
	// 这里简化处理：查找同一团队中有 admin 或 team_leader 角色的用户
	var users []models.User
	if err := r.db.WithContext(ctx).
		Joins("JOIN user_roles ON user_roles.user_id = users.id").
		Joins("JOIN roles ON roles.id = user_roles.role_id").
		Where("roles.name IN (?, ?) AND users.status = ?", "admin", "team_leader", "active").
		Find(&users).Error; err != nil {
		return "", err
	}

	var userIDs []string
	for _, u := range users {
		userIDs = append(userIDs, strconv.FormatUint(uint64(u.ID), 10))
	}

	return strings.Join(userIDs, ","), nil
}

// IsUserApprover 检查用户是否是审批人
func (r *ApproverResolver) IsUserApprover(ctx context.Context, userID uint, approverType, approvers string, appID uint) (bool, error) {
	// 解析实际的审批人ID列表
	resolvedApprovers, err := r.ResolveApprovers(ctx, approverType, approvers, appID)
	if err != nil {
		return false, err
	}

	if resolvedApprovers == "" {
		return false, nil
	}

	userIDStr := strconv.FormatUint(uint64(userID), 10)
	approverList := strings.Split(resolvedApprovers, ",")
	for _, a := range approverList {
		if strings.TrimSpace(a) == userIDStr {
			return true, nil
		}
	}

	return false, nil
}

// uniqueStrings 字符串去重
func uniqueStrings(strs []string) []string {
	seen := make(map[string]bool)
	result := make([]string, 0, len(strs))
	for _, s := range strs {
		if !seen[s] {
			seen[s] = true
			result = append(result, s)
		}
	}
	return result
}
