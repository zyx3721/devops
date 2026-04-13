-- ==========================================
-- 审计模块优化 - 模型统一迁移脚本
-- ==========================================
-- 创建时间: 2026-01-13
-- 目的: 统一审计日志模型，添加缺失字段，优化性能
-- 影响: audit_logs 表结构变更
-- ==========================================

-- 1. 添加租户ID字段（支持多租户）
ALTER TABLE audit_logs 
ADD COLUMN tenant_id INT UNSIGNED NULL COMMENT '租户ID' AFTER id;

-- 2. 添加旧值/新值字段（JSON格式，用于变更追踪）
ALTER TABLE audit_logs 
ADD COLUMN old_value JSON NULL COMMENT '变更前的值' AFTER resource_name;

ALTER TABLE audit_logs 
ADD COLUMN new_value JSON NULL COMMENT '变更后的值' AFTER old_value;

-- 3. 添加请求ID和追踪ID（用于分布式追踪）
ALTER TABLE audit_logs 
ADD COLUMN request_id VARCHAR(50) NULL COMMENT '请求ID' AFTER user_agent;

ALTER TABLE audit_logs 
ADD COLUMN trace_id VARCHAR(50) NULL COMMENT '追踪ID' AFTER request_id;

-- 4. 添加操作耗时字段
ALTER TABLE audit_logs 
ADD COLUMN duration BIGINT NULL COMMENT '操作耗时(ms)' AFTER trace_id;

-- 5. 重命名字段（统一命名规范）
ALTER TABLE audit_logs 
CHANGE COLUMN resource resource_type VARCHAR(50) NOT NULL COMMENT '资源类型';

ALTER TABLE audit_logs 
CHANGE COLUMN error_msg error_message TEXT NULL COMMENT '错误信息';

-- 6. 修改字段类型（支持 NULL，用于系统操作）
ALTER TABLE audit_logs 
MODIFY COLUMN user_id INT UNSIGNED NULL COMMENT '用户ID（可为空）';

ALTER TABLE audit_logs 
MODIFY COLUMN resource_id INT UNSIGNED NULL COMMENT '资源ID（可为空）';

-- 7. 添加索引（优化查询性能）
ALTER TABLE audit_logs 
ADD INDEX idx_tenant_id (tenant_id);

ALTER TABLE audit_logs 
ADD INDEX idx_request_id (request_id);

-- 8. 删除旧字段（如果存在 detail 字段）
-- 注意: 如果表中不存在 detail 字段，此语句会报错，可以注释掉
-- ALTER TABLE audit_logs DROP COLUMN IF EXISTS detail;

-- ==========================================
-- 迁移完成
-- ==========================================
-- 新增字段: tenant_id, old_value, new_value, request_id, trace_id, duration
-- 重命名字段: resource → resource_type, error_msg → error_message
-- 修改类型: user_id, resource_id 改为可 NULL
-- 新增索引: idx_tenant_id, idx_request_id
-- ==========================================
