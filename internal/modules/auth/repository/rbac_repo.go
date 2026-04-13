package repository

import (
	"context"

	"gorm.io/gorm"

	"devops/internal/models"
)

// RoleRepository 角色仓库
type RoleRepository struct {
	db *gorm.DB
}

func NewRoleRepository(db *gorm.DB) *RoleRepository {
	return &RoleRepository{db: db}
}

func (r *RoleRepository) Create(ctx context.Context, role *models.Role) error {
	return r.db.WithContext(ctx).Create(role).Error
}

func (r *RoleRepository) Update(ctx context.Context, role *models.Role) error {
	return r.db.WithContext(ctx).Save(role).Error
}

func (r *RoleRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&models.Role{}, id).Error
}

func (r *RoleRepository) GetByID(ctx context.Context, id uint) (*models.Role, error) {
	var role models.Role
	if err := r.db.WithContext(ctx).First(&role, id).Error; err != nil {
		return nil, err
	}
	return &role, nil
}

func (r *RoleRepository) GetByName(ctx context.Context, name string) (*models.Role, error) {
	var role models.Role
	if err := r.db.WithContext(ctx).Where("name = ?", name).First(&role).Error; err != nil {
		return nil, err
	}
	return &role, nil
}

func (r *RoleRepository) List(ctx context.Context, page, pageSize int) ([]models.Role, int64, error) {
	var roles []models.Role
	var total int64

	if err := r.db.WithContext(ctx).Model(&models.Role{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := r.db.WithContext(ctx).Order("id ASC").Offset(offset).Limit(pageSize).Find(&roles).Error; err != nil {
		return nil, 0, err
	}

	return roles, total, nil
}

func (r *RoleRepository) GetAll(ctx context.Context) ([]models.Role, error) {
	var roles []models.Role
	if err := r.db.WithContext(ctx).Where("status = ?", "active").Find(&roles).Error; err != nil {
		return nil, err
	}
	return roles, nil
}

// PermissionRepository 权限仓库
type PermissionRepository struct {
	db *gorm.DB
}

func NewPermissionRepository(db *gorm.DB) *PermissionRepository {
	return &PermissionRepository{db: db}
}

func (r *PermissionRepository) Create(ctx context.Context, perm *models.Permission) error {
	return r.db.WithContext(ctx).Create(perm).Error
}

func (r *PermissionRepository) GetByID(ctx context.Context, id uint) (*models.Permission, error) {
	var perm models.Permission
	if err := r.db.WithContext(ctx).First(&perm, id).Error; err != nil {
		return nil, err
	}
	return &perm, nil
}

func (r *PermissionRepository) List(ctx context.Context) ([]models.Permission, error) {
	var perms []models.Permission
	if err := r.db.WithContext(ctx).Order("resource, action").Find(&perms).Error; err != nil {
		return nil, err
	}
	return perms, nil
}

func (r *PermissionRepository) GetByResource(ctx context.Context, resource string) ([]models.Permission, error) {
	var perms []models.Permission
	if err := r.db.WithContext(ctx).Where("resource = ?", resource).Find(&perms).Error; err != nil {
		return nil, err
	}
	return perms, nil
}

// RolePermissionRepository 角色权限关联仓库
type RolePermissionRepository struct {
	db *gorm.DB
}

func NewRolePermissionRepository(db *gorm.DB) *RolePermissionRepository {
	return &RolePermissionRepository{db: db}
}

func (r *RolePermissionRepository) SetRolePermissions(ctx context.Context, roleID uint, permissionIDs []uint) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 删除旧的权限
		if err := tx.Where("role_id = ?", roleID).Delete(&models.RolePermission{}).Error; err != nil {
			return err
		}
		// 添加新的权限
		for _, permID := range permissionIDs {
			rp := &models.RolePermission{RoleID: roleID, PermissionID: permID}
			if err := tx.Create(rp).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *RolePermissionRepository) GetRolePermissions(ctx context.Context, roleID uint) ([]models.Permission, error) {
	var perms []models.Permission
	if err := r.db.WithContext(ctx).
		Joins("JOIN role_permissions ON role_permissions.permission_id = permissions.id").
		Where("role_permissions.role_id = ?", roleID).
		Find(&perms).Error; err != nil {
		return nil, err
	}
	return perms, nil
}

func (r *RolePermissionRepository) GetRolePermissionIDs(ctx context.Context, roleID uint) ([]uint, error) {
	var ids []uint
	if err := r.db.WithContext(ctx).Model(&models.RolePermission{}).
		Where("role_id = ?", roleID).
		Pluck("permission_id", &ids).Error; err != nil {
		return nil, err
	}
	return ids, nil
}

// UserRoleRepository 用户角色关联仓库
type UserRoleRepository struct {
	db *gorm.DB
}

func NewUserRoleRepository(db *gorm.DB) *UserRoleRepository {
	return &UserRoleRepository{db: db}
}

func (r *UserRoleRepository) SetUserRoles(ctx context.Context, userID uint, roleIDs []uint) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 删除旧的角色
		if err := tx.Where("user_id = ?", userID).Delete(&models.UserRole{}).Error; err != nil {
			return err
		}
		// 添加新的角色
		for _, roleID := range roleIDs {
			ur := &models.UserRole{UserID: userID, RoleID: roleID}
			if err := tx.Create(ur).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *UserRoleRepository) GetUserRoles(ctx context.Context, userID uint) ([]models.Role, error) {
	var roles []models.Role
	if err := r.db.WithContext(ctx).
		Joins("JOIN user_roles ON user_roles.role_id = roles.id").
		Where("user_roles.user_id = ?", userID).
		Find(&roles).Error; err != nil {
		return nil, err
	}
	return roles, nil
}

func (r *UserRoleRepository) GetUserRoleIDs(ctx context.Context, userID uint) ([]uint, error) {
	var ids []uint
	if err := r.db.WithContext(ctx).Model(&models.UserRole{}).
		Where("user_id = ?", userID).
		Pluck("role_id", &ids).Error; err != nil {
		return nil, err
	}
	return ids, nil
}

func (r *UserRoleRepository) GetUserPermissions(ctx context.Context, userID uint) ([]models.Permission, error) {
	var perms []models.Permission
	if err := r.db.WithContext(ctx).
		Distinct().
		Joins("JOIN role_permissions ON role_permissions.permission_id = permissions.id").
		Joins("JOIN user_roles ON user_roles.role_id = role_permissions.role_id").
		Where("user_roles.user_id = ?", userID).
		Find(&perms).Error; err != nil {
		return nil, err
	}
	return perms, nil
}

func (r *UserRoleRepository) HasPermission(ctx context.Context, userID uint, resource, action string) (bool, error) {
	var count int64
	permName := resource + ":" + action
	if err := r.db.WithContext(ctx).Model(&models.Permission{}).
		Joins("JOIN role_permissions ON role_permissions.permission_id = permissions.id").
		Joins("JOIN user_roles ON user_roles.role_id = role_permissions.role_id").
		Where("user_roles.user_id = ? AND (permissions.name = ? OR permissions.name = ?)", userID, permName, resource+":admin").
		Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}
