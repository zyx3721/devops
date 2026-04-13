-- ============================================
-- RBAC 权限初始化 SQL（增量更新，保留现有数据）
-- ============================================

-- ============================================
-- 1. 补充角色（如果不存在则插入）
-- ============================================
SET @permissions_description_exists := (
  SELECT COUNT(*)
  FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE()
    AND TABLE_NAME = 'permissions'
    AND COLUMN_NAME = 'description'
);
SET @permissions_description_sql := IF(
  @permissions_description_exists = 0,
  'ALTER TABLE `permissions` ADD COLUMN `description` TEXT AFTER `action`',
  'SELECT 1'
);
PREPARE permissions_description_stmt FROM @permissions_description_sql;
EXECUTE permissions_description_stmt;
DEALLOCATE PREPARE permissions_description_stmt;

SET @permissions_resource_index_exists := (
  SELECT COUNT(*)
  FROM information_schema.STATISTICS
  WHERE TABLE_SCHEMA = DATABASE()
    AND TABLE_NAME = 'permissions'
    AND INDEX_NAME = 'idx_permissions_resource'
);
SET @permissions_resource_index_sql := IF(
  @permissions_resource_index_exists = 0,
  'ALTER TABLE `permissions` ADD KEY `idx_permissions_resource` (`resource`)',
  'SELECT 1'
);
PREPARE permissions_resource_index_stmt FROM @permissions_resource_index_sql;
EXECUTE permissions_resource_index_stmt;
DEALLOCATE PREPARE permissions_resource_index_stmt;

INSERT IGNORE INTO roles (name, display_name, description, is_system, created_at, updated_at) VALUES
('super_admin', '超级管理员', '拥有所有权限，不可被修改或删除', 1, NOW(), NOW()),
('admin', '管理员', '拥有大部分管理权限', 1, NOW(), NOW()),
('user', '普通用户', '查看和基本操作权限', 1, NOW(), NOW()),
('guest', '访客', '只有查看权限', 1, NOW(), NOW());

-- ============================================
-- 2. 补充权限（如果不存在则插入）
-- ============================================
INSERT IGNORE INTO permissions (name, display_name, resource, action, description, created_at) VALUES
-- 用户管理
('user:view', '查看用户', 'user', 'view', '查看用户列表和详情', NOW()),
('user:create', '创建用户', 'user', 'create', '创建新用户', NOW()),
('user:update', '更新用户', 'user', 'update', '更新用户信息', NOW()),
('user:delete', '删除用户', 'user', 'delete', '删除用户', NOW()),
('user:role', '修改角色', 'user', 'role', '修改用户角色', NOW()),
('user:status', '修改状态', 'user', 'status', '启用/禁用用户', NOW()),
-- 应用管理
('app:view', '查看应用', 'app', 'view', '查看应用', NOW()),
('app:create', '创建应用', 'app', 'create', '创建应用', NOW()),
('app:update', '更新应用', 'app', 'update', '更新应用', NOW()),
('app:delete', '删除应用', 'app', 'delete', '删除应用', NOW()),
('app:deploy', '发布应用', 'app', 'deploy', '发布部署', NOW()),
-- 审批管理
('approval:view', '查看审批', 'approval', 'view', '查看审批', NOW()),
('approval:create', '创建审批', 'approval', 'create', '创建审批规则', NOW()),
('approval:update', '更新审批', 'approval', 'update', '更新审批配置', NOW()),
('approval:delete', '删除审批', 'approval', 'delete', '删除审批规则', NOW()),
-- K8s管理
('k8s:view', '查看K8s', 'k8s', 'view', '查看K8s资源', NOW()),
('k8s:create', '创建K8s', 'k8s', 'create', '创建K8s配置', NOW()),
('k8s:update', '更新K8s', 'k8s', 'update', '更新K8s配置', NOW()),
('k8s:delete', '删除K8s', 'k8s', 'delete', '删除K8s配置', NOW()),
('k8s:exec', 'K8s操作', 'k8s', 'exec', '重启/扩缩容等', NOW()),
-- Jenkins管理
('jenkins:view', '查看Jenkins', 'jenkins', 'view', '查看Jenkins', NOW()),
('jenkins:create', '创建Jenkins', 'jenkins', 'create', '创建Jenkins', NOW()),
('jenkins:update', '更新Jenkins', 'jenkins', 'update', '更新Jenkins', NOW()),
('jenkins:delete', '删除Jenkins', 'jenkins', 'delete', '删除Jenkins', NOW()),
('jenkins:trigger', '触发构建', 'jenkins', 'trigger', '触发构建', NOW()),
-- 系统配置
('system:view', '查看系统配置', 'system', 'view', '查看系统配置', NOW()),
('system:update', '更新系统配置', 'system', 'update', '更新系统配置', NOW()),
-- 告警管理
('alert:view', '查看告警', 'alert', 'view', '查看告警', NOW()),
('alert:create', '创建告警', 'alert', 'create', '创建告警配置', NOW()),
('alert:update', '更新告警', 'alert', 'update', '更新告警配置', NOW()),
('alert:delete', '删除告警', 'alert', 'delete', '删除告警配置', NOW());

-- ============================================
-- 3. 重建角色权限关联
-- ============================================
DELETE FROM role_permissions;

-- 超级管理员 - 所有权限
INSERT INTO role_permissions (role_id, permission_id, created_at)
SELECT r.id, p.id, NOW() FROM roles r, permissions p WHERE r.name = 'super_admin';

-- 管理员 - 除系统配置更新外的所有权限
INSERT INTO role_permissions (role_id, permission_id, created_at)
SELECT r.id, p.id, NOW() FROM roles r, permissions p 
WHERE r.name = 'admin' AND p.name != 'system:update';

-- 普通用户 - 查看 + 发布 + 触发构建
INSERT INTO role_permissions (role_id, permission_id, created_at)
SELECT r.id, p.id, NOW() FROM roles r, permissions p 
WHERE r.name = 'user' AND p.name IN (
    'app:view', 'app:deploy',
    'approval:view',
    'k8s:view',
    'jenkins:view', 'jenkins:trigger',
    'alert:view'
);

-- 访客 - 只有查看权限
INSERT INTO role_permissions (role_id, permission_id, created_at)
SELECT r.id, p.id, NOW() FROM roles r, permissions p 
WHERE r.name = 'guest' AND p.action = 'view';

-- ============================================
-- 4. 更新用户角色
-- ============================================
UPDATE users SET role = 'super_admin' WHERE id = 1;
UPDATE users SET role = 'user' WHERE username = 'test';
UPDATE users SET role = 'guest' WHERE role IS NULL OR role = '';
