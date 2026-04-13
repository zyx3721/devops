-- RBAC 权限系统表创建脚本

-- 1. 创建 roles 表
CREATE TABLE IF NOT EXISTS roles (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    name VARCHAR(50) NOT NULL,
    display_name VARCHAR(100),
    description TEXT,
    is_system TINYINT(1) DEFAULT 0,
    status VARCHAR(20) DEFAULT 'active',
    UNIQUE KEY idx_roles_name (name)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 2. 创建 permissions 表
CREATE TABLE IF NOT EXISTS permissions (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    name VARCHAR(100) NOT NULL,
    display_name VARCHAR(100),
    resource VARCHAR(50) NOT NULL,
    action VARCHAR(50) NOT NULL,
    description TEXT,
    UNIQUE KEY idx_permissions_name (name),
    KEY idx_permissions_resource (resource)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 3. 创建 role_permissions 表
CREATE TABLE IF NOT EXISTS role_permissions (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    role_id BIGINT UNSIGNED NOT NULL,
    permission_id BIGINT UNSIGNED NOT NULL,
    KEY idx_role_permissions_role (role_id),
    KEY idx_role_permissions_permission (permission_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 4. 创建 user_roles 表
CREATE TABLE IF NOT EXISTS user_roles (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    user_id BIGINT UNSIGNED NOT NULL,
    role_id BIGINT UNSIGNED NOT NULL,
    KEY idx_user_roles_user (user_id),
    KEY idx_user_roles_role (role_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 5. 插入默认角色
INSERT IGNORE INTO roles (name, display_name, description, is_system, status) VALUES
('admin', '管理员', '拥有所有权限', 1, 'active'),
('operator', '运维人员', '可以操作 Jenkins、K8s 等资源', 1, 'active'),
('developer', '开发人员', '可以查看资源和触发构建', 1, 'active'),
('viewer', '只读用户', '只能查看资源', 1, 'active'),
('owner', '所有者', '租户所有者，拥有全部权限', 1, 'active');

-- 6. 插入默认权限
INSERT IGNORE INTO permissions (name, display_name, resource, action, description) VALUES
-- Jenkins 权限
('jenkins:read', '查看 Jenkins', 'jenkins', 'read', '查看 Jenkins 实例和任务'),
('jenkins:write', '管理 Jenkins', 'jenkins', 'write', '创建和修改 Jenkins 配置'),
('jenkins:execute', '执行 Jenkins', 'jenkins', 'execute', '触发 Jenkins 构建'),
('jenkins:delete', '删除 Jenkins', 'jenkins', 'delete', '删除 Jenkins 实例'),
-- K8s 权限
('k8s:read', '查看 K8s', 'k8s', 'read', '查看 K8s 集群和资源'),
('k8s:write', '管理 K8s', 'k8s', 'write', '创建和修改 K8s 资源'),
('k8s:delete', '删除 K8s', 'k8s', 'delete', '删除 K8s 资源'),
-- 用户权限
('user:read', '查看用户', 'user', 'read', '查看用户列表'),
('user:write', '管理用户', 'user', 'write', '创建和修改用户'),
('user:delete', '删除用户', 'user', 'delete', '删除用户'),
-- 配额权限
('quota:read', '查看配额', 'quota', 'read', '查看配额和用量'),
('quota:manage', '管理配额', 'quota', 'manage', '管理套餐和配额'),
-- 审计权限
('audit:read', '查看审计', 'audit', 'read', '查看审计日志'),
('audit:export', '导出审计', 'audit', 'export', '导出审计日志');

-- 7. 为 admin 角色分配所有权限
INSERT IGNORE INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p WHERE r.name = 'admin';

-- 8. 为 operator 角色分配权限
INSERT IGNORE INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p 
WHERE r.name = 'operator' AND p.name IN (
    'jenkins:read', 'jenkins:write', 'jenkins:execute',
    'k8s:read', 'k8s:write',
    'user:read',
    'quota:read',
    'audit:read'
);

-- 9. 为 developer 角色分配权限
INSERT IGNORE INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p 
WHERE r.name = 'developer' AND p.name IN (
    'jenkins:read', 'jenkins:execute',
    'k8s:read',
    'user:read',
    'quota:read'
);

-- 10. 为 viewer 角色分配权限
INSERT IGNORE INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p 
WHERE r.name = 'viewer' AND p.name IN (
    'jenkins:read',
    'k8s:read',
    'user:read',
    'quota:read'
);

-- 11. 为 owner 角色分配所有权限
INSERT IGNORE INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p WHERE r.name = 'owner';

-- 12. 为管理员用户分配 admin 角色（如果用户存在）
INSERT IGNORE INTO user_roles (user_id, role_id)
SELECT u.id, r.id FROM users u, roles r 
WHERE u.id = 1 AND r.name = 'admin';

SELECT 'RBAC tables created successfully' AS status;
