# Pipeline Templates Migration Guide

## Issue
You're seeing this error:
```
Error 1054 (42S22): Unknown column 'language' in 'field list'
```

This means the `pipeline_templates` table exists in your database but is missing the required columns.

## Root Cause
The `pipeline_templates.sql` migration file was not executed when the table was initially created, or the table was created with an older schema.

## Solution

### Option 1: Quick Fix - Add Missing Columns (推荐)

执行专门的修复脚本来添加缺失的列：

```bash
# Windows (PowerShell) - 推荐
cd devops
Get-Content migrations/fix_pipeline_templates_columns.sql | mysql -u root -p devops

# Windows (CMD)
cd devops
mysql -u root -p devops < migrations/fix_pipeline_templates_columns.sql

# 或者使用简单版本（如果上面的不工作）
mysql -u root -p devops < migrations/fix_pipeline_templates_simple.sql
```

这个脚本会：
- 检查 `language` 列是否存在，不存在则添加
- 检查 `framework` 列是否存在，不存在则添加
- 添加必要的索引
- 显示修复后的表结构

### Option 2: Run the Full Migration

Execute the pipeline templates migration:

```bash
# Windows (PowerShell)
cd devops
Get-Content migrations/pipeline_templates.sql | mysql -u root -p devops

# Windows (CMD)
cd devops
mysql -u root -p devops < migrations/pipeline_templates.sql
```

**Note**: If the tables already exist, you may see errors like "Table already exists". This is normal - the migration uses `CREATE TABLE IF NOT EXISTS`.

### Option 2: Add Missing Columns Manually

If you want to preserve existing data and only add missing columns:

**方法 A: 使用 MySQL 命令行**

```bash
mysql -u root -p
```

然后执行：

```sql
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

-- 验证
DESCRIBE pipeline_templates;
```

**方法 B: 使用 SQL 文件**

```bash
cd devops
mysql -u root -p devops < migrations/fix_pipeline_templates_simple.sql
```

### Option 3: Drop and Recreate (Only if no important data)

**WARNING**: This will delete all existing template data!

```sql
USE devops;

-- Drop existing tables
DROP TABLE IF EXISTS pipeline_template_ratings;
DROP TABLE IF EXISTS pipeline_stage_templates;
DROP TABLE IF EXISTS pipeline_step_templates;
DROP TABLE IF EXISTS pipeline_templates;

-- Then run the migration
SOURCE migrations/pipeline_templates.sql;
```

## Verification

After running the migration, verify the table structure:

```sql
USE devops;
DESCRIBE pipeline_templates;
```

You should see these columns:
- id
- name
- description
- category
- **language** ← This should now exist
- **framework** ← This should now exist
- config_json
- icon_url
- is_builtin
- is_public
- usage_count
- rating
- rating_count
- created_by
- created_at
- updated_at

## Testing

After fixing the schema, restart your application and test:

```bash
# Restart the backend
cd devops
go run cmd/server/main.go
```

The error should be resolved and the template initialization should work correctly.

## Related Files

- Migration file: `devops/migrations/pipeline_templates.sql`
- Handler code: `devops/internal/modules/pipeline/handler/template_handler.go`
- README: `devops/migrations/README.md` (updated with execution order)

## Date
2026-01-18
