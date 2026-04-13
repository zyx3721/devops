-- 修复 pipeline_templates 表缺失的列
-- 执行日期: 2026-01-18
-- 问题: Error 1054 (42S22): Unknown column 'language' in 'field list'

USE devops;

-- 检查并添加 language 列
SET @col_exists = 0;
SELECT COUNT(*) INTO @col_exists 
FROM information_schema.COLUMNS 
WHERE TABLE_SCHEMA = 'devops' 
  AND TABLE_NAME = 'pipeline_templates' 
  AND COLUMN_NAME = 'language';

SET @sql = IF(@col_exists = 0,
    'ALTER TABLE pipeline_templates ADD COLUMN language VARCHAR(50) COMMENT ''编程语言: java, go, nodejs, python'' AFTER category',
    'SELECT ''Column language already exists'' AS message');
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

-- 检查并添加 framework 列
SET @col_exists = 0;
SELECT COUNT(*) INTO @col_exists 
FROM information_schema.COLUMNS 
WHERE TABLE_SCHEMA = 'devops' 
  AND TABLE_NAME = 'pipeline_templates' 
  AND COLUMN_NAME = 'framework';

SET @sql = IF(@col_exists = 0,
    'ALTER TABLE pipeline_templates ADD COLUMN framework VARCHAR(50) COMMENT ''框架: spring, gin, express, django'' AFTER language',
    'SELECT ''Column framework already exists'' AS message');
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

-- 添加索引（如果不存在）
SET @index_exists = 0;
SELECT COUNT(*) INTO @index_exists 
FROM information_schema.STATISTICS 
WHERE TABLE_SCHEMA = 'devops' 
  AND TABLE_NAME = 'pipeline_templates' 
  AND INDEX_NAME = 'idx_language';

SET @sql = IF(@index_exists = 0,
    'ALTER TABLE pipeline_templates ADD INDEX idx_language (language)',
    'SELECT ''Index idx_language already exists'' AS message');
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

-- 验证表结构
SELECT 'Pipeline templates table structure after fix:' AS message;
DESCRIBE pipeline_templates;

SELECT 'Fix completed successfully!' AS message;
