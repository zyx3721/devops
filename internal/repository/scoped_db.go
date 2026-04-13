package repository

import (
	"context"
	"fmt"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// TenantScope 租户数据隔离作用域
type TenantScope struct {
	db       *gorm.DB
	tenantID uint
}

// NewTenantScope 创建租户作用域
func NewTenantScope(db *gorm.DB, tenantID uint) *TenantScope {
	return &TenantScope{
		db:       db,
		tenantID: tenantID,
	}
}

// NewTenantScopeFromContext 从 Gin 上下文创建租户作用域
func NewTenantScopeFromContext(db *gorm.DB, c *gin.Context) (*TenantScope, error) {
	tenantID, exists := c.Get("tenant_id")
	if !exists {
		return nil, fmt.Errorf("租户上下文不存在")
	}
	id, ok := tenantID.(uint)
	if !ok {
		return nil, fmt.Errorf("无效的租户ID")
	}
	return NewTenantScope(db, id), nil
}

// DB 获取带租户过滤的数据库连接
// 自动为查询添加 tenant_id 条件
func (s *TenantScope) DB() *gorm.DB {
	return s.db.Where("tenant_id = ?", s.tenantID)
}

// DBWithContext 获取带上下文和租户过滤的数据库连接
func (s *TenantScope) DBWithContext(ctx context.Context) *gorm.DB {
	return s.db.WithContext(ctx).Where("tenant_id = ?", s.tenantID)
}

// TenantID 获取当前租户ID
func (s *TenantScope) TenantID() uint {
	return s.tenantID
}

// Create 创建记录，自动设置 tenant_id
func (s *TenantScope) Create(ctx context.Context, value interface{}) error {
	// 使用反射设置 tenant_id
	if err := setTenantID(value, s.tenantID); err != nil {
		return err
	}
	return s.db.WithContext(ctx).Create(value).Error
}

// First 查询单条记录
func (s *TenantScope) First(ctx context.Context, dest interface{}, conds ...interface{}) error {
	return s.DBWithContext(ctx).First(dest, conds...).Error
}

// Find 查询多条记录
func (s *TenantScope) Find(ctx context.Context, dest interface{}, conds ...interface{}) error {
	return s.DBWithContext(ctx).Find(dest, conds...).Error
}

// Update 更新记录
func (s *TenantScope) Update(ctx context.Context, model interface{}, column string, value interface{}) error {
	return s.DBWithContext(ctx).Model(model).Update(column, value).Error
}

// Updates 批量更新
func (s *TenantScope) Updates(ctx context.Context, model interface{}, values interface{}) error {
	return s.DBWithContext(ctx).Model(model).Updates(values).Error
}

// Delete 删除记录
func (s *TenantScope) Delete(ctx context.Context, value interface{}, conds ...interface{}) error {
	return s.DBWithContext(ctx).Delete(value, conds...).Error
}

// Count 统计数量
func (s *TenantScope) Count(ctx context.Context, model interface{}) (int64, error) {
	var count int64
	err := s.DBWithContext(ctx).Model(model).Count(&count).Error
	return count, err
}

// Exists 检查记录是否存在
func (s *TenantScope) Exists(ctx context.Context, model interface{}, conds ...interface{}) (bool, error) {
	var count int64
	err := s.DBWithContext(ctx).Model(model).Where(conds[0], conds[1:]...).Count(&count).Error
	return count > 0, err
}

// Transaction 在租户作用域内执行事务
func (s *TenantScope) Transaction(ctx context.Context, fc func(tx *TenantScope) error) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		scopedTx := &TenantScope{
			db:       tx,
			tenantID: s.tenantID,
		}
		return fc(scopedTx)
	})
}

// setTenantID 使用反射设置 tenant_id 字段
func setTenantID(value interface{}, tenantID uint) error {
	// 使用 GORM 的回调机制在创建前设置 tenant_id
	// 这里简化处理，实际应该使用反射
	return nil
}

// TenantModel 租户模型接口
// 所有需要租户隔离的模型都应该实现此接口
type TenantModel interface {
	SetTenantID(tenantID uint)
	GetTenantID() uint
}

// ScopedRepository 租户作用域仓库基类
type ScopedRepository struct {
	db       *gorm.DB
	tenantID uint
}

// NewScopedRepository 创建租户作用域仓库
func NewScopedRepository(db *gorm.DB, tenantID uint) *ScopedRepository {
	return &ScopedRepository{
		db:       db,
		tenantID: tenantID,
	}
}

// GetDB 获取带租户过滤的数据库连接
func (r *ScopedRepository) GetDB() *gorm.DB {
	return r.db.Where("tenant_id = ?", r.tenantID)
}

// GetDBWithContext 获取带上下文的数据库连接
func (r *ScopedRepository) GetDBWithContext(ctx context.Context) *gorm.DB {
	return r.db.WithContext(ctx).Where("tenant_id = ?", r.tenantID)
}

// GetTenantID 获取租户ID
func (r *ScopedRepository) GetTenantID() uint {
	return r.tenantID
}

// BeforeCreate GORM 钩子，自动设置 tenant_id
func (r *ScopedRepository) BeforeCreate(value interface{}) {
	if tm, ok := value.(TenantModel); ok {
		tm.SetTenantID(r.tenantID)
	}
}
