-- 制品仓库集成状态监控功能
-- 为 artifact_repositories 表添加连接状态监控字段

-- 添加连接状态监控字段
ALTER TABLE artifact_repositories 
ADD COLUMN connection_status VARCHAR(20) DEFAULT 'unknown' COMMENT '连接状态: connected, disconnected, checking, unknown' AFTER enabled,
ADD COLUMN last_check_at TIMESTAMP NULL COMMENT '最后检查时间' AFTER connection_status,
ADD COLUMN last_error TEXT COMMENT '最后错误信息' AFTER last_check_at,
ADD COLUMN enable_monitoring BOOLEAN DEFAULT TRUE COMMENT '是否启用监控' AFTER last_error,
ADD COLUMN check_interval INT DEFAULT 300 COMMENT '检查间隔(秒)' AFTER enable_monitoring;

-- 创建制品仓库连接历史记录表
CREATE TABLE IF NOT EXISTS artifact_registry_connection_history (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    registry_id BIGINT UNSIGNED NOT NULL COMMENT '制品仓库ID',
    status VARCHAR(20) NOT NULL COMMENT '连接状态: success, failed',
    response_time INT COMMENT '响应时间(毫秒)',
    error_message TEXT COMMENT '错误信息',
    checked_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP COMMENT '检查时间',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    KEY idx_registry_id (registry_id),
    KEY idx_checked_at (checked_at),
    KEY idx_status (status),
    CONSTRAINT fk_registry_history FOREIGN KEY (registry_id) 
        REFERENCES artifact_repositories(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='制品仓库连接历史记录表';

-- 创建索引以优化查询性能
CREATE INDEX idx_connection_status ON artifact_repositories(connection_status);
CREATE INDEX idx_last_check_at ON artifact_repositories(last_check_at);
CREATE INDEX idx_enable_monitoring ON artifact_repositories(enable_monitoring);

-- 插入示例数据（可选）
-- INSERT INTO artifact_repositories (name, description, type, url, is_default, connection_status, enable_monitoring) 
-- VALUES 
-- ('Harbor 生产环境', 'Harbor 生产环境制品仓库', 'harbor', 'https://harbor.example.com', TRUE, 'unknown', TRUE),
-- ('Nexus 开发环境', 'Nexus 开发环境制品仓库', 'nexus', 'https://nexus-dev.example.com', FALSE, 'unknown', TRUE);

-- 创建定期清理历史记录的事件（保留最近 30 天的记录）
-- 注意：需要确保 MySQL 的事件调度器已启用 (SET GLOBAL event_scheduler = ON;)
DELIMITER $$

CREATE EVENT IF NOT EXISTS cleanup_registry_connection_history
ON SCHEDULE EVERY 1 DAY
STARTS CURRENT_TIMESTAMP
DO
BEGIN
    DELETE FROM artifact_registry_connection_history 
    WHERE checked_at < DATE_SUB(NOW(), INTERVAL 30 DAY);
END$$

DELIMITER ;

-- 创建触发器：当连接状态变化时自动记录历史
DELIMITER $$

CREATE TRIGGER after_registry_status_update
AFTER UPDATE ON artifact_repositories
FOR EACH ROW
BEGIN
    -- 只有当连接状态发生变化时才记录
    IF OLD.connection_status != NEW.connection_status THEN
        INSERT INTO artifact_registry_connection_history (
            registry_id, 
            status, 
            error_message,
            checked_at
        ) VALUES (
            NEW.id,
            CASE 
                WHEN NEW.connection_status = 'connected' THEN 'success'
                ELSE 'failed'
            END,
            NEW.last_error,
            NEW.last_check_at
        );
    END IF;
END$$

DELIMITER ;

-- 添加注释说明
ALTER TABLE artifact_repositories 
MODIFY COLUMN connection_status VARCHAR(20) DEFAULT 'unknown' 
COMMENT '连接状态: connected(已连接), disconnected(连接失败), checking(检查中), unknown(未知)';

-- 创建视图：最近的连接状态统计
CREATE OR REPLACE VIEW v_registry_connection_stats AS
SELECT 
    ar.id,
    ar.name,
    ar.type,
    ar.connection_status,
    ar.last_check_at,
    ar.enable_monitoring,
    COUNT(CASE WHEN arch.status = 'success' THEN 1 END) as success_count,
    COUNT(CASE WHEN arch.status = 'failed' THEN 1 END) as failed_count,
    ROUND(
        COUNT(CASE WHEN arch.status = 'success' THEN 1 END) * 100.0 / 
        NULLIF(COUNT(*), 0), 
        2
    ) as success_rate,
    AVG(arch.response_time) as avg_response_time
FROM artifact_repositories ar
LEFT JOIN artifact_registry_connection_history arch 
    ON ar.id = arch.registry_id 
    AND arch.checked_at >= DATE_SUB(NOW(), INTERVAL 7 DAY)
GROUP BY ar.id, ar.name, ar.type, ar.connection_status, ar.last_check_at, ar.enable_monitoring;

-- 添加说明文档
/*
使用说明：

1. 连接状态字段说明：
   - connection_status: 当前连接状态
     * connected: 连接成功
     * disconnected: 连接失败
     * checking: 正在检查
     * unknown: 未知状态（初始状态）
   
   - last_check_at: 最后一次检查的时间
   - last_error: 最后一次连接失败的错误信息
   - enable_monitoring: 是否启用自动监控
   - check_interval: 自动检查的时间间隔（秒）

2. 连接历史记录表：
   - 记录每次连接测试的结果
   - 包含响应时间、错误信息等详细数据
   - 自动清理 30 天前的历史记录

3. 触发器：
   - 当连接状态发生变化时，自动记录到历史表
   - 便于追踪状态变化和问题排查

4. 统计视图：
   - v_registry_connection_stats: 提供最近 7 天的连接统计
   - 包含成功率、平均响应时间等指标

5. API 端点建议：
   - GET /artifact-registries - 获取所有仓库及其状态
   - POST /artifact-registries/:id/test - 测试单个仓库连接
   - POST /artifact-registries/refresh-status - 批量刷新所有仓库状态
   - GET /artifact-registries/:id/history - 获取连接历史记录
   - GET /artifact-registries/stats - 获取连接统计信息

6. 定期任务建议：
   - 创建后台任务，每 5 分钟检查一次启用监控的仓库
   - 当状态从 connected 变为 disconnected 时发送告警通知
   - 定期清理过期的历史记录
*/
