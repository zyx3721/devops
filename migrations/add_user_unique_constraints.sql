-- 添加用户表唯一约束
-- 创建时间: 2026-01-13
-- 目的: 防止用户注册时的并发竞态条件

-- 为 username 列创建唯一索引（如果已存在会报错，可忽略）
CREATE UNIQUE INDEX idx_users_username ON users(username);

-- 为 email 列创建唯一索引（如果已存在会报错，可忽略）
CREATE UNIQUE INDEX idx_users_email ON users(email);

-- 回滚脚本（如需回滚，请手动执行以下语句）
-- DROP INDEX idx_users_username ON users;
-- DROP INDEX idx_users_email ON users;
