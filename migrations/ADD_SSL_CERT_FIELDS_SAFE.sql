-- ============================================
-- SSL 证书检查功能 - 添加字段脚本（兼容版本）
-- 执行时间: 2026-01-17
-- 说明: 为现有的 health_check_configs 和 health_check_histories 表添加 SSL 证书相关字段
-- 注意: 如果字段已存在会报错，可以忽略错误继续执行
-- ============================================

USE my_devops;

-- ============================================
-- 1. 为 health_check_configs 表添加 SSL 证书字段
-- ============================================

-- 证书信息字段
ALTER TABLE health_check_configs ADD COLUMN cert_expiry_date DATETIME COMMENT '证书过期时间';
ALTER TABLE health_check_configs ADD COLUMN cert_days_remaining INT COMMENT '证书剩余天数';
ALTER TABLE health_check_configs ADD COLUMN cert_issuer VARCHAR(500) COMMENT '证书颁发者';
ALTER TABLE health_check_configs ADD COLUMN cert_subject VARCHAR(500) COMMENT '证书主题（CN）';
ALTER TABLE health_check_configs ADD COLUMN cert_serial_number VARCHAR(100) COMMENT '证书序列号';

-- 告警阈值配置字段
ALTER TABLE health_check_configs ADD COLUMN critical_days INT DEFAULT 7 COMMENT '严重告警阈值（天）';
ALTER TABLE health_check_configs ADD COLUMN warning_days INT DEFAULT 30 COMMENT '警告告警阈值（天）';
ALTER TABLE health_check_configs ADD COLUMN notice_days INT DEFAULT 60 COMMENT '提醒告警阈值（天）';

-- 告警状态字段
ALTER TABLE health_check_configs ADD COLUMN last_alert_level VARCHAR(20) DEFAULT 'normal' COMMENT '最后告警级别: expired/critical/warning/notice/normal';
ALTER TABLE health_check_configs ADD COLUMN last_alert_at DATETIME COMMENT '最后告警时间';

-- ============================================
-- 2. 为 health_check_histories 表添加 SSL 证书字段
-- ============================================

ALTER TABLE health_check_histories ADD COLUMN cert_days_remaining INT COMMENT '证书剩余天数';
ALTER TABLE health_check_histories ADD COLUMN cert_expiry_date DATETIME COMMENT '证书过期时间';
ALTER TABLE health_check_histories ADD COLUMN alert_level VARCHAR(20) COMMENT '告警级别';

-- ============================================
-- 3. 添加索引以优化查询性能
-- ============================================

-- 为 SSL 证书类型和告警级别添加索引
ALTER TABLE health_check_configs ADD INDEX idx_type_alert_level (type, last_alert_level);

-- 为证书剩余天数添加索引（用于排序和筛选）
ALTER TABLE health_check_configs ADD INDEX idx_cert_days_remaining (cert_days_remaining);

-- 为证书过期时间添加索引
ALTER TABLE health_check_configs ADD INDEX idx_cert_expiry_date (cert_expiry_date);

-- 为最后告警时间添加索引
ALTER TABLE health_check_configs ADD INDEX idx_last_alert_at (last_alert_at);

-- ============================================
-- 4. 验证字段是否添加成功
-- ============================================

SELECT '✅ SSL 证书字段添加完成！' AS status;

-- 查看 health_check_configs 表结构
SELECT 
    COLUMN_NAME, 
    COLUMN_TYPE, 
    IS_NULLABLE, 
    COLUMN_DEFAULT, 
    COLUMN_COMMENT 
FROM INFORMATION_SCHEMA.COLUMNS 
WHERE TABLE_SCHEMA = 'my_devops' 
  AND TABLE_NAME = 'health_check_configs' 
  AND (COLUMN_NAME LIKE 'cert%' OR COLUMN_NAME LIKE '%alert%' OR COLUMN_NAME LIKE '%days')
ORDER BY ORDINAL_POSITION;

-- 查看 health_check_histories 表结构
SELECT 
    COLUMN_NAME, 
    COLUMN_TYPE, 
    IS_NULLABLE, 
    COLUMN_DEFAULT, 
    COLUMN_COMMENT 
FROM INFORMATION_SCHEMA.COLUMNS 
WHERE TABLE_SCHEMA = 'my_devops' 
  AND TABLE_NAME = 'health_check_histories' 
  AND (COLUMN_NAME LIKE 'cert%' OR COLUMN_NAME LIKE 'alert%')
ORDER BY ORDINAL_POSITION;

-- 查看索引
SELECT 
    INDEX_NAME, 
    COLUMN_NAME, 
    SEQ_IN_INDEX 
FROM INFORMATION_SCHEMA.STATISTICS 
WHERE TABLE_SCHEMA = 'my_devops' 
  AND TABLE_NAME = 'health_check_configs' 
  AND INDEX_NAME LIKE 'idx_%'
ORDER BY INDEX_NAME, SEQ_IN_INDEX;
