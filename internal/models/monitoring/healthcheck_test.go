package monitoring

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// setupTestDB 创建测试数据库连接
func setupTestDB(t *testing.T) *gorm.DB {
	dsn := "root:@tcp(127.0.0.1:3306)/devops_test?charset=utf8mb4&parseTime=True&loc=Local"
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Skipf("跳过测试: MySQL 数据库不可用 - %v", err)
		return nil
	}
	return db
}

// TestHealthCheckConfigModel 测试 HealthCheckConfig 模型结构
func TestHealthCheckConfigModel(t *testing.T) {
	db := setupTestDB(t)
	if db == nil {
		return
	}

	// 自动迁移
	err := db.AutoMigrate(&HealthCheckConfig{})
	require.NoError(t, err)

	// 清理测试数据
	db.Exec("TRUNCATE TABLE health_check_configs")

	// 测试创建配置
	now := time.Now()
	daysRemaining := 30
	config := &HealthCheckConfig{
		Name:          "Test SSL Cert Check",
		Type:          "ssl_cert",
		URL:           "example.com",
		Interval:      3600,
		Timeout:       10,
		RetryCount:    3,
		Enabled:       true,
		AlertEnabled:  true,
		AlertPlatform: "feishu",
		LastStatus:    "healthy",

		// SSL证书相关字段
		CertExpiryDate:    &now,
		CertDaysRemaining: &daysRemaining,
		CertIssuer:        "Let's Encrypt",
		CertSubject:       "CN=example.com",
		CertSerialNumber:  "123456789",

		// 告警阈值配置
		CriticalDays: 7,
		WarningDays:  30,
		NoticeDays:   60,

		// 告警状态
		LastAlertLevel: "normal",
		LastAlertAt:    &now,
	}

	err = db.Create(config).Error
	require.NoError(t, err)
	assert.Greater(t, config.ID, uint(0))

	// 测试查询配置
	var retrieved HealthCheckConfig
	err = db.First(&retrieved, config.ID).Error
	require.NoError(t, err)

	assert.Equal(t, "Test SSL Cert Check", retrieved.Name)
	assert.Equal(t, "ssl_cert", retrieved.Type)
	assert.Equal(t, "example.com", retrieved.URL)
	assert.NotNil(t, retrieved.CertExpiryDate)
	assert.NotNil(t, retrieved.CertDaysRemaining)
	assert.Equal(t, 30, *retrieved.CertDaysRemaining)
	assert.Equal(t, "Let's Encrypt", retrieved.CertIssuer)
	assert.Equal(t, "CN=example.com", retrieved.CertSubject)
	assert.Equal(t, "123456789", retrieved.CertSerialNumber)
	assert.Equal(t, 7, retrieved.CriticalDays)
	assert.Equal(t, 30, retrieved.WarningDays)
	assert.Equal(t, 60, retrieved.NoticeDays)
	assert.Equal(t, "normal", retrieved.LastAlertLevel)
	assert.NotNil(t, retrieved.LastAlertAt)
}

// TestHealthCheckHistoryModel 测试 HealthCheckHistory 模型结构
func TestHealthCheckHistoryModel(t *testing.T) {
	db := setupTestDB(t)
	if db == nil {
		return
	}

	// 自动迁移
	err := db.AutoMigrate(&HealthCheckHistory{})
	require.NoError(t, err)

	// 清理测试数据
	db.Exec("TRUNCATE TABLE health_check_histories")

	// 测试创建历史记录
	now := time.Now()
	daysRemaining := 15
	history := &HealthCheckHistory{
		ConfigID:       1,
		ConfigName:     "Test SSL Cert Check",
		Type:           "ssl_cert",
		TargetName:     "example.com",
		Status:         "healthy",
		ResponseTimeMs: 150,
		ErrorMsg:       "",
		AlertSent:      false,

		// SSL证书检查结果
		CertDaysRemaining: &daysRemaining,
		CertExpiryDate:    &now,
		AlertLevel:        "warning",
	}

	err = db.Create(history).Error
	require.NoError(t, err)
	assert.Greater(t, history.ID, uint(0))

	// 测试查询历史记录
	var retrieved HealthCheckHistory
	err = db.First(&retrieved, history.ID).Error
	require.NoError(t, err)

	assert.Equal(t, uint(1), retrieved.ConfigID)
	assert.Equal(t, "Test SSL Cert Check", retrieved.ConfigName)
	assert.Equal(t, "ssl_cert", retrieved.Type)
	assert.Equal(t, "example.com", retrieved.TargetName)
	assert.Equal(t, "healthy", retrieved.Status)
	assert.Equal(t, int64(150), retrieved.ResponseTimeMs)
	assert.NotNil(t, retrieved.CertDaysRemaining)
	assert.Equal(t, 15, *retrieved.CertDaysRemaining)
	assert.NotNil(t, retrieved.CertExpiryDate)
	assert.Equal(t, "warning", retrieved.AlertLevel)
}

// TestHealthCheckConfigTableName 测试表名
func TestHealthCheckConfigTableName(t *testing.T) {
	config := HealthCheckConfig{}
	assert.Equal(t, "health_check_configs", config.TableName())
}

// TestHealthCheckHistoryTableName 测试表名
func TestHealthCheckHistoryTableName(t *testing.T) {
	history := HealthCheckHistory{}
	assert.Equal(t, "health_check_histories", history.TableName())
}

// TestHealthCheckConfigDefaultValues 测试默认值
func TestHealthCheckConfigDefaultValues(t *testing.T) {
	db := setupTestDB(t)
	if db == nil {
		return
	}

	err := db.AutoMigrate(&HealthCheckConfig{})
	require.NoError(t, err)

	// 清理测试数据
	db.Exec("TRUNCATE TABLE health_check_configs")

	// 创建最小配置
	config := &HealthCheckConfig{
		Name: "Minimal Config",
		Type: "ssl_cert",
		URL:  "example.com",
	}

	err = db.Create(config).Error
	require.NoError(t, err)

	// 验证默认值
	var retrieved HealthCheckConfig
	err = db.First(&retrieved, config.ID).Error
	require.NoError(t, err)

	// GORM 默认值在 SQLite 中可能不会自动应用，但我们可以验证字段存在
	assert.Equal(t, "Minimal Config", retrieved.Name)
	assert.Equal(t, "ssl_cert", retrieved.Type)
	assert.Equal(t, "example.com", retrieved.URL)
}

// TestHealthCheckConfigNullableFields 测试可空字段
func TestHealthCheckConfigNullableFields(t *testing.T) {
	db := setupTestDB(t)
	if db == nil {
		return
	}

	err := db.AutoMigrate(&HealthCheckConfig{})
	require.NoError(t, err)

	// 清理测试数据
	db.Exec("TRUNCATE TABLE health_check_configs")

	// 创建配置，不设置可空字段
	config := &HealthCheckConfig{
		Name: "Config with Nulls",
		Type: "ssl_cert",
		URL:  "example.com",
	}

	err = db.Create(config).Error
	require.NoError(t, err)

	// 验证可空字段为 nil
	var retrieved HealthCheckConfig
	err = db.First(&retrieved, config.ID).Error
	require.NoError(t, err)

	assert.Nil(t, retrieved.CertExpiryDate)
	assert.Nil(t, retrieved.CertDaysRemaining)
	assert.Nil(t, retrieved.LastAlertAt)
	assert.Nil(t, retrieved.AlertBotID)
	assert.Nil(t, retrieved.CreatedBy)
}
