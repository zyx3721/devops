package database

import (
	"devops/internal/config"
	"devops/pkg/logger"
	"devops/pkg/utils"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// InitMySQL 初始化 MySQL 连接（带重试）
func InitMySQL(cfg *config.Config) (*gorm.DB, error) {
	var db *gorm.DB

	err := utils.RetryWithBackoffSimple("MySQL", func() error {
		var connErr error
		db, connErr = gorm.Open(mysql.Open(cfg.DSN()), &gorm.Config{})
		if connErr != nil {
			return connErr
		}
		// 测试连接
		sqlDB, err := db.DB()
		if err != nil {
			return err
		}
		return sqlDB.Ping()
	})

	if err != nil {
		return nil, err
	}

	// 配置连接池
	sqlDB, _ := db.DB()
	if sqlDB != nil {
		sqlDB.SetMaxIdleConns(cfg.MySQLMaxIdleConns)
		sqlDB.SetMaxOpenConns(cfg.MySQLMaxOpenConns)
		sqlDB.SetConnMaxLifetime(cfg.MySQLConnMaxLifetime)
	}

	if cfg.Debug {
		db = db.Debug()
	}

	logger.L().Info("[MySQL] Connected successfully to %s:%d/%s", cfg.MySQLHost, cfg.MySQLPort, cfg.MySQLDatabase)
	return db, nil
}

// AutoMigrate 自动迁移数据库表
func AutoMigrate(db *gorm.DB) {
	// 注意：这里只做基础表的自动迁移
	// 复杂的表结构（如制品、流水线等）应该使用 migrations/*.sql 文件
	// 避免在代码中重复定义模型，统一使用 internal/models 中的定义
}
