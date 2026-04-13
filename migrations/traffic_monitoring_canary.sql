-- 流量监控和灰度发布相关表
-- 执行时间: 2026-01-12

-- 流量统计表
CREATE TABLE IF NOT EXISTS traffic_statistics (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    app_id BIGINT UNSIGNED NOT NULL COMMENT '应用ID',
    timestamp DATETIME NOT NULL COMMENT '统计时间',
    total_requests BIGINT DEFAULT 0 COMMENT '总请求数',
    success_requests BIGINT DEFAULT 0 COMMENT '成功请求数',
    failed_requests BIGINT DEFAULT 0 COMMENT '失败请求数',
    rate_limited_count BIGINT DEFAULT 0 COMMENT '限流次数',
    circuit_break_count BIGINT DEFAULT 0 COMMENT '熔断次数',
    avg_latency_ms DOUBLE DEFAULT 0 COMMENT '平均延迟(ms)',
    p50_latency_ms DOUBLE DEFAULT 0 COMMENT 'P50延迟(ms)',
    p90_latency_ms DOUBLE DEFAULT 0 COMMENT 'P90延迟(ms)',
    p99_latency_ms DOUBLE DEFAULT 0 COMMENT 'P99延迟(ms)',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_app_timestamp (app_id, timestamp)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='流量统计表';

-- 规则版本表（用于回滚）
CREATE TABLE IF NOT EXISTS traffic_rule_versions (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    app_id BIGINT UNSIGNED NOT NULL COMMENT '应用ID',
    rule_type VARCHAR(50) NOT NULL COMMENT '规则类型',
    rule_id BIGINT UNSIGNED NOT NULL COMMENT '规则ID',
    version INT NOT NULL COMMENT '版本号',
    content JSON NOT NULL COMMENT '规则内容',
    operator VARCHAR(100) COMMENT '操作人',
    description VARCHAR(500) COMMENT '版本描述',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_app_id (app_id),
    INDEX idx_rule (rule_type, rule_id),
    INDEX idx_created_at (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='规则版本表';

-- 金丝雀发布配置表
CREATE TABLE IF NOT EXISTS canary_releases (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    app_id BIGINT UNSIGNED NOT NULL COMMENT '应用ID',
    name VARCHAR(100) NOT NULL COMMENT '发布名称',
    status VARCHAR(20) DEFAULT 'pending' COMMENT '状态: pending/running/paused/completed/rollback/canary_running/success/rolled_back',
    stable_version VARCHAR(100) COMMENT '稳定版本',
    canary_version VARCHAR(100) COMMENT '金丝雀版本',
    current_weight INT DEFAULT 0 COMMENT '当前金丝雀权重',
    target_weight INT DEFAULT 100 COMMENT '目标权重',
    weight_increment INT DEFAULT 10 COMMENT '权重增量',
    interval_seconds INT DEFAULT 60 COMMENT '增量间隔(秒)',
    success_threshold DOUBLE DEFAULT 95 COMMENT '成功率阈值',
    latency_threshold INT DEFAULT 500 COMMENT '延迟阈值(ms)',
    auto_rollback BOOLEAN DEFAULT TRUE COMMENT '自动回滚',
    started_at DATETIME COMMENT '开始时间',
    completed_at DATETIME COMMENT '完成时间',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_app_id (app_id),
    INDEX idx_status (status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='金丝雀发布配置表';

-- 添加 env_name 列（如果不存在）
SET @exist := (SELECT COUNT(*) FROM information_schema.columns WHERE table_schema = DATABASE() AND table_name = 'canary_releases' AND column_name = 'env_name');
SET @sql := IF(@exist = 0, 'ALTER TABLE canary_releases ADD COLUMN env_name VARCHAR(50) DEFAULT \'\' COMMENT \'环境名称\' AFTER name, ADD INDEX idx_env_name (env_name)', 'SELECT 1');
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

-- 蓝绿部署配置表
CREATE TABLE IF NOT EXISTS blue_green_deployments (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    app_id BIGINT UNSIGNED NOT NULL COMMENT '应用ID',
    name VARCHAR(100) NOT NULL COMMENT '部署名称',
    status VARCHAR(20) DEFAULT 'pending' COMMENT '状态: pending/blue_active/green_active/switching/switched/rolled_back/completed',
    blue_version VARCHAR(100) COMMENT '蓝版本',
    green_version VARCHAR(100) COMMENT '绿版本',
    active_color VARCHAR(10) DEFAULT 'blue' COMMENT '当前活跃: blue/green',
    replicas INT DEFAULT 2 COMMENT '副本数',
    warmup_seconds INT DEFAULT 30 COMMENT '预热时间(秒)',
    switched_at DATETIME COMMENT '切换时间',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_app_id (app_id),
    INDEX idx_status (status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='蓝绿部署配置表';

-- 添加 env_name 列（如果不存在）
SET @exist := (SELECT COUNT(*) FROM information_schema.columns WHERE table_schema = DATABASE() AND table_name = 'blue_green_deployments' AND column_name = 'env_name');
SET @sql := IF(@exist = 0, 'ALTER TABLE blue_green_deployments ADD COLUMN env_name VARCHAR(50) DEFAULT \'\' COMMENT \'环境名称\' AFTER name, ADD INDEX idx_env_name (env_name)', 'SELECT 1');
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;
