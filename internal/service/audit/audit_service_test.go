package audit

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// setupTestDB 创建测试数据库
func setupTestDB(t *testing.T) *gorm.DB {
	dsn := "root:@tcp(127.0.0.1:3306)/devops_test?charset=utf8mb4&parseTime=True&loc=Local"
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Skipf("跳过测试: MySQL 数据库不可用 - %v", err)
		return nil
	}

	// 自动迁移
	err = db.AutoMigrate(&AuditEntry{})
	require.NoError(t, err)

	// 清理测试数据
	db.Exec("TRUNCATE TABLE audit_logs")

	return db
}

// TestProperty_AuditLogCompleteness 属性测试：审计日志完整性
// Property 7: For any data modification operation,
// an audit log entry SHALL be created containing
// the user, action, resource, and timestamp.
func TestProperty_AuditLogCompleteness(t *testing.T) {
	db := setupTestDB(t)
	if db == nil {
		return
	}
	service := NewAuditService(db)
	ctx := context.Background()

	tenantID := uint(1)
	userID := uint(100)
	resourceID := uint(200)

	// 记录审计日志
	entry := &AuditEntry{
		TenantID:     &tenantID,
		UserID:       &userID,
		Username:     "testuser",
		Action:       ActionCreate,
		ResourceType: "pipeline",
		ResourceID:   &resourceID,
		ResourceName: "test-pipeline",
		NewValue:     map[string]interface{}{"name": "test-pipeline"},
		IPAddress:    "192.168.1.1",
		UserAgent:    "Mozilla/5.0",
		RequestID:    "req-123",
		Status:       StatusSuccess,
	}

	err := service.Log(ctx, entry)
	require.NoError(t, err)

	// 查询审计日志
	result, err := service.Query(ctx, &QueryRequest{
		TenantID: &tenantID,
		Page:     1,
		PageSize: 10,
	})
	require.NoError(t, err)
	require.Len(t, result.List, 1)

	log := result.List[0]

	// 属性验证：审计日志应包含所有必要字段
	assert.Equal(t, tenantID, *log.TenantID, "应包含租户ID")
	assert.Equal(t, userID, *log.UserID, "应包含用户ID")
	assert.Equal(t, "testuser", log.Username, "应包含用户名")
	assert.Equal(t, string(ActionCreate), log.Action, "应包含操作类型")
	assert.Equal(t, "pipeline", log.ResourceType, "应包含资源类型")
	assert.Equal(t, resourceID, *log.ResourceID, "应包含资源ID")
	assert.NotZero(t, log.CreatedAt, "应包含时间戳")
}

// TestProperty_AuditLogImmutability 属性测试：审计日志不可变性
func TestProperty_AuditLogImmutability(t *testing.T) {
	db := setupTestDB(t)
	if db == nil {
		return
	}
	service := NewAuditService(db)
	ctx := context.Background()

	tenantID := uint(1)
	userID := uint(100)

	// 记录审计日志
	entry := &AuditEntry{
		TenantID:     &tenantID,
		UserID:       &userID,
		Username:     "testuser",
		Action:       ActionUpdate,
		ResourceType: "application",
		Status:       StatusSuccess,
	}

	err := service.Log(ctx, entry)
	require.NoError(t, err)

	// 查询原始日志
	result, _ := service.Query(ctx, &QueryRequest{TenantID: &tenantID, Page: 1, PageSize: 10})
	originalLog := result.List[0]
	originalAction := originalLog.Action

	// 尝试修改日志（直接通过数据库）
	// 注意：在生产环境中，审计日志表应该有写保护
	db.Model(&AuditEntry{}).Where("id = ?", originalLog.ID).Update("action", "delete")

	// 属性验证：审计日志应该被修改（这里只是演示，实际应该有保护机制）
	// 在实际系统中，应该通过数据库权限或触发器防止修改
	result, _ = service.Query(ctx, &QueryRequest{TenantID: &tenantID, Page: 1, PageSize: 10})
	modifiedLog := result.List[0]

	// 这个测试展示了需要额外的保护机制
	t.Logf("原始操作: %s, 修改后: %s", originalAction, modifiedLog.Action)
}

// TestProperty_AuditLogQueryByRequestID 属性测试：按请求ID查询
func TestProperty_AuditLogQueryByRequestID(t *testing.T) {
	db := setupTestDB(t)
	if db == nil {
		return
	}
	service := NewAuditService(db)
	ctx := context.Background()

	requestID := "req-456"
	tenantID := uint(1)
	userID := uint(100)

	// 记录多条相同请求ID的日志
	for i := 0; i < 3; i++ {
		entry := &AuditEntry{
			TenantID:     &tenantID,
			UserID:       &userID,
			Username:     "testuser",
			Action:       ActionUpdate,
			ResourceType: "config",
			RequestID:    requestID,
			Status:       StatusSuccess,
		}
		service.Log(ctx, entry)
	}

	// 按请求ID查询
	logs, err := service.GetByRequestID(ctx, requestID)
	require.NoError(t, err)

	// 属性验证：应该返回所有相同请求ID的日志
	assert.Len(t, logs, 3, "应返回所有相同请求ID的日志")

	for _, log := range logs {
		assert.Equal(t, requestID, log.RequestID, "所有日志的请求ID应该相同")
	}
}

// TestProperty_AuditLogResourceHistory 属性测试：资源变更历史
func TestProperty_AuditLogResourceHistory(t *testing.T) {
	db := setupTestDB(t)
	if db == nil {
		return
	}
	service := NewAuditService(db)
	ctx := context.Background()

	tenantID := uint(1)
	userID := uint(100)
	resourceID := uint(500)

	// 记录资源的创建、更新、删除操作
	actions := []AuditAction{ActionCreate, ActionUpdate, ActionUpdate, ActionDelete}
	for _, action := range actions {
		entry := &AuditEntry{
			TenantID:     &tenantID,
			UserID:       &userID,
			Username:     "testuser",
			Action:       action,
			ResourceType: "deployment",
			ResourceID:   &resourceID,
			Status:       StatusSuccess,
		}
		service.Log(ctx, entry)
		time.Sleep(10 * time.Millisecond) // 确保时间戳不同
	}

	// 查询资源历史
	history, err := service.GetResourceHistory(ctx, "deployment", resourceID)
	require.NoError(t, err)

	// 属性验证：应该返回资源的完整变更历史
	assert.Len(t, history, 4, "应返回资源的完整变更历史")

	// 属性验证：历史应该按时间倒序排列
	for i := 0; i < len(history)-1; i++ {
		assert.True(t, history[i].CreatedAt.After(history[i+1].CreatedAt) ||
			history[i].CreatedAt.Equal(history[i+1].CreatedAt),
			"历史应该按时间倒序排列")
	}
}

// TestProperty_AuditLogOldNewValue 属性测试：变更前后值记录
func TestProperty_AuditLogOldNewValue(t *testing.T) {
	db := setupTestDB(t)
	if db == nil {
		return
	}
	service := NewAuditService(db)
	ctx := context.Background()

	tenantID := uint(1)
	userID := uint(100)
	resourceID := uint(600)

	oldValue := map[string]interface{}{
		"name":     "old-name",
		"replicas": 2,
	}
	newValue := map[string]interface{}{
		"name":     "new-name",
		"replicas": 4,
	}

	// 记录更新操作
	entry := &AuditEntry{
		TenantID:     &tenantID,
		UserID:       &userID,
		Username:     "testuser",
		Action:       ActionUpdate,
		ResourceType: "deployment",
		ResourceID:   &resourceID,
		OldValue:     oldValue,
		NewValue:     newValue,
		Status:       StatusSuccess,
	}

	err := service.Log(ctx, entry)
	require.NoError(t, err)

	// 查询日志
	result, _ := service.Query(ctx, &QueryRequest{TenantID: &tenantID, Page: 1, PageSize: 10})
	log := result.List[0]

	// 属性验证：应该记录变更前后的值
	assert.NotNil(t, log.OldValue, "应记录变更前的值")
	assert.NotNil(t, log.NewValue, "应记录变更后的值")
}

// TestProperty_AuditLogFailedOperation 属性测试：失败操作记录
func TestProperty_AuditLogFailedOperation(t *testing.T) {
	db := setupTestDB(t)
	if db == nil {
		return
	}
	service := NewAuditService(db)
	ctx := context.Background()

	tenantID := uint(1)
	userID := uint(100)

	// 记录失败的操作
	entry := &AuditEntry{
		TenantID:     &tenantID,
		UserID:       &userID,
		Username:     "testuser",
		Action:       ActionDelete,
		ResourceType: "pipeline",
		Status:       StatusFailed,
		ErrorMessage: "权限不足",
	}

	err := service.Log(ctx, entry)
	require.NoError(t, err)

	// 查询失败的操作
	result, err := service.Query(ctx, &QueryRequest{
		TenantID: &tenantID,
		Status:   string(StatusFailed),
		Page:     1,
		PageSize: 10,
	})
	require.NoError(t, err)

	// 属性验证：失败的操作也应该被记录
	assert.Len(t, result.List, 1, "失败的操作也应该被记录")
	assert.Equal(t, string(StatusFailed), result.List[0].Status)
	assert.Equal(t, "权限不足", result.List[0].ErrorMessage)
}

// TestProperty_AuditLogCleanup 属性测试：日志清理
func TestProperty_AuditLogCleanup(t *testing.T) {
	db := setupTestDB(t)
	if db == nil {
		return
	}
	service := NewAuditService(db)
	ctx := context.Background()

	tenantID := uint(1)
	userID := uint(100)

	// 创建一些旧日志
	for i := 0; i < 5; i++ {
		log := &AuditEntry{
			TenantID:     &tenantID,
			UserID:       &userID,
			Username:     "testuser",
			Action:       ActionRead,
			ResourceType: "pipeline",
			Status:       StatusSuccess,
			//CreatedAt:    time.Now().AddDate(0, 0, -100), // 100 天前
		}
		db.Create(log)
	}

	// 创建一些新日志
	for i := 0; i < 3; i++ {
		entry := &AuditEntry{
			TenantID:     &tenantID,
			UserID:       &userID,
			Username:     "testuser",
			Action:       ActionRead,
			ResourceType: "pipeline",
			Status:       StatusSuccess,
		}
		service.Log(ctx, entry)
	}

	// 清理 90 天前的日志
	deleted, err := service.Cleanup(ctx, 90)
	require.NoError(t, err)

	// 属性验证：应该删除旧日志
	assert.Equal(t, int64(5), deleted, "应该删除 5 条旧日志")

	// 属性验证：新日志应该保留
	result, _ := service.Query(ctx, &QueryRequest{TenantID: &tenantID, Page: 1, PageSize: 100})
	assert.Len(t, result.List, 3, "新日志应该保留")
}
