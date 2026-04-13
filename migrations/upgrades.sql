-- 项目名称：devops
-- 文件名称：upgrades.sql
-- 作　　者：Jerion
-- 联系邮箱：416685476@qq.com
-- 功能描述：存量数据库升级补丁（仅对已有数据库执行，全新部署请使用 init_tables.sql）
-- 执行顺序：按章节顺序依次执行

-- ============================================
-- 1. 飞书相关补丁
-- ============================================

-- feishu_apps 补充 webhook 列（如列已存在可忽略报错）
ALTER TABLE `feishu_apps`
ADD COLUMN `webhook` varchar(500) DEFAULT '' COMMENT 'Webhook URL' AFTER `app_secret`;

-- 飞书消息发送记录表
CREATE TABLE IF NOT EXISTS `feishu_message_logs` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `created_at` datetime(3) DEFAULT CURRENT_TIMESTAMP(3),
  `msg_type` varchar(50) NOT NULL COMMENT '消息类型: text/post/interactive',
  `receive_id` varchar(100) NOT NULL COMMENT '接收者ID',
  `receive_id_type` varchar(50) NOT NULL COMMENT 'ID类型: chat_id/open_id/user_id',
  `content` text COMMENT '消息内容',
  `title` varchar(200) DEFAULT '' COMMENT '卡片标题',
  `source` varchar(50) DEFAULT '' COMMENT '来源: manual/oa_sync',
  `status` varchar(20) DEFAULT 'success' COMMENT '状态: success/failed',
  `error_msg` text COMMENT '错误信息',
  `app_id` bigint unsigned DEFAULT NULL COMMENT '使用的飞书应用ID',
  PRIMARY KEY (`id`),
  KEY `idx_fml_msg_type` (`msg_type`),
  KEY `idx_fml_source` (`source`),
  KEY `idx_fml_status` (`status`),
  KEY `idx_fml_app_id` (`app_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='飞书消息发送记录';

-- 飞书用户令牌表
CREATE TABLE IF NOT EXISTS `feishu_user_tokens` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `created_at` datetime(3) DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  `app_id` varchar(100) NOT NULL COMMENT '飞书 App ID',
  `access_token` text COMMENT '访问令牌',
  `refresh_token` text COMMENT '刷新令牌',
  `expires_at` datetime(3) DEFAULT NULL COMMENT '过期时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_fut_app_id` (`app_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='飞书用户OAuth令牌';


-- ============================================
-- 2. 流水线模板补丁（fix_pipeline_templates_columns.sql）
-- ============================================

-- 检查并添加 language 列
SET @col_exists = 0;
SELECT COUNT(*) INTO @col_exists
FROM information_schema.COLUMNS
WHERE TABLE_SCHEMA = DATABASE()
  AND TABLE_NAME = 'pipeline_templates'
  AND COLUMN_NAME = 'language';
SET @sql = IF(@col_exists = 0,
    'ALTER TABLE pipeline_templates ADD COLUMN language VARCHAR(50) COMMENT \'编程语言: java, go, nodejs, python\' AFTER category',
    'SELECT 1');
PREPARE stmt FROM @sql; EXECUTE stmt; DEALLOCATE PREPARE stmt;

-- 检查并添加 framework 列
SET @col_exists = 0;
SELECT COUNT(*) INTO @col_exists
FROM information_schema.COLUMNS
WHERE TABLE_SCHEMA = DATABASE()
  AND TABLE_NAME = 'pipeline_templates'
  AND COLUMN_NAME = 'framework';
SET @sql = IF(@col_exists = 0,
    'ALTER TABLE pipeline_templates ADD COLUMN framework VARCHAR(50) COMMENT \'框架: spring, gin, express, django\' AFTER language',
    'SELECT 1');
PREPARE stmt FROM @sql; EXECUTE stmt; DEALLOCATE PREPARE stmt;

-- 检查并添加 idx_language 索引
SET @index_exists = 0;
SELECT COUNT(*) INTO @index_exists
FROM information_schema.STATISTICS
WHERE TABLE_SCHEMA = DATABASE()
  AND TABLE_NAME = 'pipeline_templates'
  AND INDEX_NAME = 'idx_language';
SET @sql = IF(@index_exists = 0,
    'ALTER TABLE pipeline_templates ADD INDEX idx_language (language)',
    'SELECT 1');
PREPARE stmt FROM @sql; EXECUTE stmt; DEALLOCATE PREPARE stmt;

-- ============================================
-- 3. 负载均衡 hash_key 字段修复（fix_loadbalance_hash_key.sql）
-- ============================================

ALTER TABLE `traffic_loadbalance_config`
MODIFY COLUMN `hash_key` VARCHAR(20) DEFAULT NULL COMMENT '哈希键类型(header/cookie/source_ip/query_param)';

-- ============================================
-- 4. 制品仓库监控补丁（artifact_registry_monitoring.sql）
-- ============================================

-- 为 artifact_repositories 添加监控字段（如列已存在可忽略报错）
ALTER TABLE `artifact_repositories`
ADD COLUMN `connection_status` varchar(20) DEFAULT 'unknown' COMMENT '连接状态: connected/disconnected/checking/unknown' AFTER `enabled`,
ADD COLUMN `last_check_at` datetime(3) DEFAULT NULL COMMENT '最后检查时间' AFTER `connection_status`,
ADD COLUMN `last_error` text COMMENT '最后错误信息' AFTER `last_check_at`,
ADD COLUMN `enable_monitoring` tinyint(1) DEFAULT 1 COMMENT '是否启用监控' AFTER `last_error`,
ADD COLUMN `check_interval` int DEFAULT 300 COMMENT '检查间隔(秒)' AFTER `enable_monitoring`;

CREATE INDEX IF NOT EXISTS `idx_connection_status` ON `artifact_repositories`(`connection_status`);
CREATE INDEX IF NOT EXISTS `idx_enable_monitoring` ON `artifact_repositories`(`enable_monitoring`);

-- 创建统计视图
CREATE OR REPLACE VIEW `v_registry_connection_stats` AS
SELECT
  ar.id, ar.name, ar.type, ar.connection_status,
  ar.last_check_at, ar.enable_monitoring,
  COUNT(CASE WHEN arch.status = 'ok' THEN 1 END) AS success_count,
  COUNT(CASE WHEN arch.status = 'error' THEN 1 END) AS failed_count,
  ROUND(COUNT(CASE WHEN arch.status = 'ok' THEN 1 END) * 100.0 / NULLIF(COUNT(*), 0), 2) AS success_rate,
  AVG(arch.latency_ms) AS avg_response_time
FROM `artifact_repositories` ar
LEFT JOIN `artifact_registry_connection_history` arch
  ON ar.id = arch.registry_id
  AND arch.check_time >= DATE_SUB(NOW(), INTERVAL 7 DAY)
GROUP BY ar.id, ar.name, ar.type, ar.connection_status, ar.last_check_at, ar.enable_monitoring;

-- 触发器：连接状态变化时自动记录历史
DELIMITER $$

CREATE TRIGGER IF NOT EXISTS `after_registry_status_update`
AFTER UPDATE ON `artifact_repositories`
FOR EACH ROW
BEGIN
  IF OLD.connection_status != NEW.connection_status THEN
    INSERT INTO `artifact_registry_connection_history` (
      `registry_id`, `status`, `message`, `check_time`
    ) VALUES (
      NEW.id,
      CASE WHEN NEW.connection_status = 'connected' THEN 'ok' ELSE 'error' END,
      NEW.last_error,
      NEW.last_check_at
    );
  END IF;
END$$

-- 定期清理连接历史（保留最近 30 天）
CREATE EVENT IF NOT EXISTS `cleanup_registry_connection_history`
ON SCHEDULE EVERY 1 DAY
STARTS CURRENT_TIMESTAMP
DO
BEGIN
  DELETE FROM `artifact_registry_connection_history`
  WHERE `check_time` < DATE_SUB(NOW(), INTERVAL 30 DAY);
END$$

DELIMITER ;

-- ============================================
-- 5. 告警日志静默字段补丁（add_log_alert_silence_fields.sql）
-- ============================================

-- 为 log_alert_history 添加静默字段（如列已存在可忽略报错）
ALTER TABLE `log_alert_history`
ADD COLUMN `silenced` tinyint(1) DEFAULT 0 COMMENT '是否被静默';

ALTER TABLE `log_alert_history`
ADD COLUMN `silence_id` int unsigned DEFAULT NULL COMMENT '静默规则ID';

CREATE INDEX IF NOT EXISTS `idx_log_alert_history_silenced` ON `log_alert_history`(`silenced`);
CREATE INDEX IF NOT EXISTS `idx_log_alert_history_silence_id` ON `log_alert_history`(`silence_id`);

-- ============================================
-- 6. system_configs 补充 deleted_at 字段
-- ============================================

ALTER TABLE `system_configs`
ADD COLUMN `deleted_at` datetime(3) DEFAULT NULL COMMENT '软删除时间' AFTER `updated_at`;

CREATE INDEX IF NOT EXISTS `idx_sc_deleted_at` ON `system_configs`(`deleted_at`);

-- ============================================
-- 7. message_templates 补充 deleted_at 字段
-- ============================================

ALTER TABLE `message_templates`
ADD COLUMN `deleted_at` datetime(3) DEFAULT NULL COMMENT '软删除时间' AFTER `updated_at`;

CREATE INDEX IF NOT EXISTS `idx_mt_deleted_at` ON `message_templates`(`deleted_at`);

-- ============================================
-- 8. message_templates 列名与 Go 模型对齐
-- ============================================

-- 将 msg_type 重命名为 type
ALTER TABLE `message_templates`
  CHANGE COLUMN `msg_type` `type` varchar(50) NOT NULL DEFAULT 'text' COMMENT '模板类型: text/markdown/card';

-- 添加 is_active 字段
ALTER TABLE `message_templates`
  ADD COLUMN `is_active` tinyint(1) NOT NULL DEFAULT 1 COMMENT '是否激活' AFTER `description`;

-- 删除不再使用的字段（platform / title / variables）
ALTER TABLE `message_templates`
  DROP COLUMN IF EXISTS `platform`,
  DROP COLUMN IF EXISTS `title`,
  DROP COLUMN IF EXISTS `variables`;

-- 修正 created_by 允许 NULL（模型为 *uint）
ALTER TABLE `message_templates`
  MODIFY COLUMN `created_by` bigint unsigned DEFAULT NULL;
