-- 简单版本：直接添加缺失的列
-- 如果列已存在会报错，但不影响使用

USE devops;

-- 添加 language 列
ALTER TABLE pipeline_templates 
ADD COLUMN language VARCHAR(50) COMMENT '编程语言: java, go, nodejs, python' AFTER category;

-- 添加 framework 列
ALTER TABLE pipeline_templates 
ADD COLUMN framework VARCHAR(50) COMMENT '框架: spring, gin, express, django' AFTER language;

-- 添加索引
ALTER TABLE pipeline_templates 
ADD INDEX idx_language (language);

-- 查看表结构
DESCRIBE pipeline_templates;
