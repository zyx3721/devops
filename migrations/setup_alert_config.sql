-- -1. 确保 alert_configs 表有新字段
-- 如果字段已存在，这些语句可能会报错，可以忽略 Duplicate column name 错误
-- 或者使用存储过程判断，但简单起见直接尝试添加
SET @dbname = DATABASE();
SET @tablename = "alert_configs";
SET @columnname = "template_id";
SET @preparedStatement = (SELECT IF(
  (
    SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS
    WHERE
      (table_name = @tablename)
      AND (table_schema = @dbname)
      AND (column_name = @columnname)
  ) > 0,
  "SELECT 1",
  "ALTER TABLE alert_configs ADD COLUMN template_id bigint(20) unsigned DEFAULT NULL"
));
PREPARE alterIfNotExists FROM @preparedStatement;
EXECUTE alterIfNotExists;
DEALLOCATE PREPARE alterIfNotExists;

SET @columnname = "channels";
SET @preparedStatement = (SELECT IF(
  (
    SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS
    WHERE
      (table_name = @tablename)
      AND (table_schema = @dbname)
      AND (column_name = @columnname)
  ) > 0,
  "SELECT 1",
  "ALTER TABLE alert_configs ADD COLUMN channels text DEFAULT NULL"
));
PREPARE alterIfNotExists FROM @preparedStatement;
EXECUTE alterIfNotExists;
DEALLOCATE PREPARE alterIfNotExists;


-- 0. 如果表不存在，先创建表
CREATE TABLE IF NOT EXISTS `sys_message_templates` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `deleted_at` datetime(3) DEFAULT NULL,
  `name` varchar(100) NOT NULL,
  `template_type` varchar(20) DEFAULT 'text',
  `title` varchar(200) DEFAULT NULL,
  `content` text,
  `variables` text,
  `description` varchar(255) DEFAULT NULL,
  `created_by` bigint(20) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_name` (`name`),
  KEY `idx_sys_message_templates_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 1. 确保有模板
INSERT IGNORE INTO sys_message_templates (created_at, updated_at, name, template_type, title, content, variables, description, created_by)
VALUES (NOW(), NOW(), 'COST_BUDGET_WARNING', 'card', '成本预算告警', 
'{"config": {"wide_screen_mode": true}, "header": {"template": "red", "title": {"content": "⚠️ 成本超支预警", "tag": "plain_text"}}, "elements": [{"tag": "div", "fields": [{"is_short": true, "text": {"tag": "lark_md", "content": "**项目：**\n{{.Project}}"}}, {"is_short": true, "text": {"tag": "lark_md", "content": "**当前成本：**\n¥{{.CurrentCost}}"}}, {"is_short": true, "text": {"tag": "lark_md", "content": "**预算：**\n¥{{.Budget}}"}}, {"is_short": true, "text": {"tag": "lark_md", "content": "**使用率：**\n{{.UsageRate}}%"}}]}, {"tag": "hr"}, {"tag": "div", "text": {"tag": "lark_md", "content": "{{.Message}}"}}]}', 
'["Project", "CurrentCost", "Budget", "UsageRate", "Message"]', '成本告警模板', 1);

-- 2. 获取模板ID (假设是刚插入的或者已存在的)
SET @tmpl_id = (SELECT id FROM sys_message_templates WHERE name = 'COST_BUDGET_WARNING' LIMIT 1);

-- 3. 创建告警配置
-- 请将下面的 webhook_url 替换为您真实的飞书 Webhook 地址
INSERT IGNORE INTO alert_configs (created_at, updated_at, name, type, enabled, platform, bot_id, template_id, channels, conditions, description, created_by)
VALUES (NOW(), NOW(), 'CPU_HIGH_ALERT', 'test', 1, 'feishu', 0, @tmpl_id, 
'[{"type":"webhook","url":"https://open.feishu.cn/open-apis/bot/v2/hook/xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"}]', 
'{}', 'CPU High Alert Rule', 1);
