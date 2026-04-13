-- ============================================================
-- 数据库迁移脚本 - 2026-01-31 SSL证书检查功能
-- ============================================================
-- 包含内容:
-- 1. health_check_configs 表添加证书相关字段
-- 2. health_check_histories 表添加证书检查结果字段
-- 3. 添加必要的索引以优化查询性能
-- ============================================================
-- 需求: 1.1, 1.2, 1.3, 8.1, 8.2, 8.3, 8.5
-- ============================================================

-- ============================================================
-- 第一部分: health_check_configs 表扩展
-- ============================================================
-- 目的: 添加SSL证书相关字段，支持证书监控和告警

-- 添加证书信息字段
ALTER TABLE health_check_configs 
ADD COLUMN cert_expiry_date DATETIME(3) NULL COMMENT '证书过期时间';

ALTER TABLE health_check_configs 
ADD COLUMN cert_days_remaining INT NULL COMMENT '证书剩余天数';

ALTER TABLE health_check_configs 
ADD COLUMN cert_issuer VARCHAR(500) DEFAULT '' COMMENT '证书颁发者';

ALTER TABLE health_check_configs 
ADD COLUMN cert_subject VARCHAR(500) DEFAULT '' COMMENT '证书主题';

ALTER TABLE health_check_configs 
ADD COLUMN cert_serial_number VARCHAR(100) DEFAULT '' COMMENT '证书序列号';

-- 添加告警阈值字段
ALTER TABLE health_check_configs 
ADD COLUMN critical_days INT DEFAULT 7 COMMENT '严重告警阈值（天）';

ALTER TABLE health_check_configs 
ADD COLUMN warning_days INT DEFAULT 30 COMMENT '警告告警阈值（天）';

ALTER TABLE health_check_configs 
ADD COLUMN notice_days INT DEFAULT 60 COMMENT '提醒告警阈值（天）';

-- 添加告警状态字段
ALTER TABLE health_check_configs 
ADD COLUMN last_alert_level VARCHAR(20) DEFAULT '' COMMENT '最后告警级别: expired/critical/warning/notice/normal';

ALTER TABLE health_check_configs 
ADD COLUMN last_alert_at DATETIME(3) NULL COMMENT '最后告警时间';

-- ============================================================
-- 第二部分: health_check_histories 表扩展
-- ============================================================
-- 目的: 记录每次证书检查的结果

-- 添加证书检查结果字段
ALTER TABLE health_check_histories 
ADD COLUMN cert_days_remaining INT NULL COMMENT '检查时的证书剩余天数';

ALTER TABLE health_check_histories 
ADD COLUMN cert_expiry_date DATETIME(3) NULL COMMENT '检查时的证书过期时间';

ALTER TABLE health_check_histories 
ADD COLUMN alert_level VARCHAR(20) DEFAULT '' COMMENT '告警级别: expired/critical/warning/notice/normal';

-- ============================================================
-- 第三部分: 添加索引以优化查询性能
-- ============================================================
-- 目的: 提高证书查询和筛选的性能

-- health_check_configs 表索引
CREATE INDEX idx_hc_type_enabled ON health_check_configs(type, enabled);
CREATE INDEX idx_hc_cert_days_remaining ON health_check_configs(cert_days_remaining);
CREATE INDEX idx_hc_last_alert_level ON health_check_configs(last_alert_level);

-- health_check_histories 表索引
CREATE INDEX idx_hc_history_alert_level ON health_check_histories(alert_level);
CREATE INDEX idx_hc_history_config_created ON health_check_histories(config_id, created_at);

-- ============================================================
-- 回滚脚本（如需回滚，请手动执行以下语句）
-- ============================================================
/*
-- 回滚 health_check_configs 表
DROP INDEX idx_hc_type_enabled ON health_check_configs;
DROP INDEX idx_hc_cert_days_remaining ON health_check_configs;
DROP INDEX idx_hc_last_alert_level ON health_check_configs;

ALTER TABLE health_check_configs DROP COLUMN cert_expiry_date;
ALTER TABLE health_check_configs DROP COLUMN cert_days_remaining;
ALTER TABLE health_check_configs DROP COLUMN cert_issuer;
ALTER TABLE health_check_configs DROP COLUMN cert_subject;
ALTER TABLE health_check_configs DROP COLUMN cert_serial_number;
ALTER TABLE health_check_configs DROP COLUMN critical_days;
ALTER TABLE health_check_configs DROP COLUMN warning_days;
ALTER TABLE health_check_configs DROP COLUMN notice_days;
ALTER TABLE health_check_configs DROP COLUMN last_alert_level;
ALTER TABLE health_check_configs DROP COLUMN last_alert_at;

-- 回滚 health_check_histories 表
DROP INDEX idx_hc_history_alert_level ON health_check_histories;
DROP INDEX idx_hc_history_config_created ON health_check_histories;

ALTER TABLE health_check_histories DROP COLUMN cert_days_remaining;
ALTER TABLE health_check_histories DROP COLUMN cert_expiry_date;
ALTER TABLE health_check_histories DROP COLUMN alert_level;
*/

-- ============================================================
-- 执行完成
-- ============================================================
SELECT '数据库迁移完成 - SSL证书检查功能' AS status;
