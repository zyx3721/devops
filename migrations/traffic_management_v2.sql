-- 流量治理功能数据库表 V2
-- 参考 Istio、Sentinel、APISIX、Kong 等主流方案

-- =====================================================
-- 1. 限流规则表 (Rate Limit Rules)
-- =====================================================
DROP TABLE IF EXISTS traffic_ratelimit_rules;
CREATE TABLE traffic_ratelimit_rules (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    app_id BIGINT NOT NULL COMMENT '应用ID',
    name VARCHAR(100) NOT NULL COMMENT '规则名称',
    description VARCHAR(500) COMMENT '规则描述',
    
    -- 资源配置
    resource_type ENUM('api', 'service', 'method') DEFAULT 'api' COMMENT '资源类型',
    resource VARCHAR(500) NOT NULL COMMENT '资源标识(接口路径/服务名/方法名)',
    method VARCHAR(10) COMMENT '请求方法(GET/POST等)',
    
    -- 限流策略
    strategy ENUM('qps', 'concurrent', 'token_bucket', 'leaky_bucket') DEFAULT 'qps' COMMENT '限流策略',
    threshold INT NOT NULL DEFAULT 100 COMMENT '阈值',
    burst INT DEFAULT 10 COMMENT '突发容量(令牌桶)',
    queue_size INT DEFAULT 100 COMMENT '队列大小(漏桶)',
    
    -- 超限行为
    control_behavior ENUM('reject', 'warm_up', 'queue', 'warm_up_queue') DEFAULT 'reject' COMMENT '超限行为',
    warm_up_period INT DEFAULT 10 COMMENT '预热时长(秒)',
    max_queue_time INT DEFAULT 500 COMMENT '最大排队时间(毫秒)',
    
    -- 限流维度
    limit_dimensions JSON COMMENT '限流维度["ip","user","api_key","header"]',
    limit_header VARCHAR(100) COMMENT '限流Header名',
    
    -- 响应配置
    rejected_code INT DEFAULT 429 COMMENT '拒绝状态码',
    rejected_message VARCHAR(500) DEFAULT 'Too Many Requests' COMMENT '拒绝消息',
    
    enabled TINYINT(1) DEFAULT 1 COMMENT '是否启用',
    priority INT DEFAULT 100 COMMENT '优先级',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    
    INDEX idx_app_id (app_id),
    INDEX idx_resource (resource(100)),
    INDEX idx_enabled (enabled)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='限流规则表';

-- =====================================================
-- 2. 熔断规则表 (Circuit Breaker Rules)
-- =====================================================
DROP TABLE IF EXISTS traffic_circuitbreaker_rules;
CREATE TABLE traffic_circuitbreaker_rules (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    app_id BIGINT NOT NULL COMMENT '应用ID',
    name VARCHAR(100) NOT NULL COMMENT '规则名称',
    resource VARCHAR(500) NOT NULL COMMENT '资源标识',
    
    -- 熔断策略
    strategy ENUM('slow_request', 'error_ratio', 'error_count') DEFAULT 'slow_request' COMMENT '熔断策略',
    slow_rt_threshold INT DEFAULT 1000 COMMENT '慢调用RT阈值(毫秒)',
    threshold DECIMAL(5,2) NOT NULL COMMENT '阈值(比例为百分比,异常数为整数)',
    
    -- 统计配置
    stat_interval INT DEFAULT 10 COMMENT '统计窗口(秒)',
    min_request_amount INT DEFAULT 5 COMMENT '最小请求数',
    
    -- 恢复配置
    recovery_timeout INT DEFAULT 30 COMMENT '熔断时长(秒)',
    probe_num INT DEFAULT 3 COMMENT '半开探测请求数',
    
    -- 降级配置
    fallback_strategy ENUM('return_error', 'return_default', 'call_fallback') DEFAULT 'return_error' COMMENT '降级策略',
    fallback_value TEXT COMMENT '降级返回值(JSON)',
    fallback_service VARCHAR(200) COMMENT '降级服务地址',
    
    -- 状态
    circuit_status ENUM('closed', 'open', 'half_open') DEFAULT 'closed' COMMENT '熔断状态',
    last_open_time TIMESTAMP NULL COMMENT '上次熔断时间',
    
    enabled TINYINT(1) DEFAULT 1 COMMENT '是否启用',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    
    INDEX idx_app_id (app_id),
    INDEX idx_resource (resource(100)),
    INDEX idx_status (circuit_status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='熔断规则表';

-- =====================================================
-- 3. 流量路由规则表 (Traffic Routing Rules)
-- =====================================================
DROP TABLE IF EXISTS traffic_routing_rules;
CREATE TABLE traffic_routing_rules (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    app_id BIGINT NOT NULL COMMENT '应用ID',
    name VARCHAR(100) NOT NULL COMMENT '规则名称',
    description VARCHAR(500) COMMENT '规则描述',
    priority INT DEFAULT 100 COMMENT '优先级(数字越小优先级越高)',
    
    -- 路由类型
    route_type ENUM('weight', 'header', 'cookie', 'param') DEFAULT 'weight' COMMENT '路由类型',
    
    -- 权重路由配置
    destinations JSON COMMENT '目标配置[{"subset":"v1","weight":90},{"subset":"v2","weight":10}]',
    
    -- 条件路由配置
    match_key VARCHAR(100) COMMENT '匹配键(header名/cookie名/参数名)',
    match_operator ENUM('exact', 'prefix', 'regex', 'present') DEFAULT 'exact' COMMENT '匹配方式',
    match_value VARCHAR(500) COMMENT '匹配值',
    target_subset VARCHAR(100) COMMENT '目标子集/版本',
    
    enabled TINYINT(1) DEFAULT 1 COMMENT '是否启用',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    
    INDEX idx_app_id (app_id),
    INDEX idx_priority (priority),
    INDEX idx_enabled (enabled)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='流量路由规则表';

-- =====================================================
-- 4. 负载均衡配置表 (Load Balance Config)
-- =====================================================
DROP TABLE IF EXISTS traffic_loadbalance_config;
CREATE TABLE traffic_loadbalance_config (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    app_id BIGINT NOT NULL UNIQUE COMMENT '应用ID',
    
    -- 负载均衡策略
    lb_policy ENUM('round_robin', 'random', 'least_request', 'consistent_hash', 'passthrough') DEFAULT 'round_robin' COMMENT '负载均衡算法',
    hash_key ENUM('header', 'cookie', 'source_ip', 'query_param') COMMENT '哈希键类型',
    hash_key_name VARCHAR(100) COMMENT '哈希键名称',
    ring_size INT DEFAULT 1024 COMMENT '一致性哈希环大小',
    choice_count INT DEFAULT 2 COMMENT '最少请求选择数量',
    warmup_duration VARCHAR(20) DEFAULT '60s' COMMENT '预热时间',
    
    -- 健康检查
    health_check_enabled TINYINT(1) DEFAULT 0 COMMENT '是否启用健康检查',
    health_check_path VARCHAR(200) DEFAULT '/health' COMMENT '健康检查路径',
    health_check_interval VARCHAR(20) DEFAULT '10s' COMMENT '检查间隔',
    health_check_timeout VARCHAR(20) DEFAULT '5s' COMMENT '检查超时',
    healthy_threshold INT DEFAULT 2 COMMENT '健康阈值',
    unhealthy_threshold INT DEFAULT 3 COMMENT '不健康阈值',
    
    -- HTTP连接池
    http_max_connections INT DEFAULT 1024 COMMENT 'HTTP最大连接数',
    http_max_requests_per_conn INT DEFAULT 0 COMMENT '每连接最大请求数',
    http_max_pending_requests INT DEFAULT 1024 COMMENT '最大等待请求数',
    http_max_retries INT DEFAULT 3 COMMENT '最大重试次数',
    http_idle_timeout VARCHAR(20) DEFAULT '1h' COMMENT 'HTTP空闲超时',
    
    -- TCP连接池
    tcp_max_connections INT DEFAULT 1024 COMMENT 'TCP最大连接数',
    tcp_connect_timeout VARCHAR(20) DEFAULT '10s' COMMENT 'TCP连接超时',
    tcp_keepalive_enabled TINYINT(1) DEFAULT 1 COMMENT 'TCP Keepalive',
    tcp_keepalive_interval VARCHAR(20) DEFAULT '60s' COMMENT 'Keepalive间隔',
    
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    
    INDEX idx_app_id (app_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='负载均衡配置表';

-- =====================================================
-- 5. 超时重试配置表 (Timeout Retry Config)
-- =====================================================
DROP TABLE IF EXISTS traffic_timeout_config;
CREATE TABLE traffic_timeout_config (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    app_id BIGINT NOT NULL UNIQUE COMMENT '应用ID',
    
    timeout VARCHAR(20) DEFAULT '30s' COMMENT '请求超时',
    retries INT DEFAULT 3 COMMENT '重试次数',
    per_try_timeout VARCHAR(20) DEFAULT '10s' COMMENT '单次重试超时',
    retry_on JSON COMMENT '重试条件,默认["5xx"]',
    
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    
    INDEX idx_app_id (app_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='超时重试配置表';

-- =====================================================
-- 6. 流量镜像规则表 (Traffic Mirror Rules)
-- =====================================================
DROP TABLE IF EXISTS traffic_mirror_rules;
CREATE TABLE traffic_mirror_rules (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    app_id BIGINT NOT NULL COMMENT '应用ID',
    
    target_service VARCHAR(200) NOT NULL COMMENT '目标服务',
    target_subset VARCHAR(100) COMMENT '目标子集',
    percentage INT DEFAULT 100 COMMENT '镜像比例(1-100)',
    
    enabled TINYINT(1) DEFAULT 1 COMMENT '是否启用',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    
    INDEX idx_app_id (app_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='流量镜像规则表';

-- =====================================================
-- 7. 故障注入规则表 (Fault Injection Rules)
-- =====================================================
DROP TABLE IF EXISTS traffic_fault_rules;
CREATE TABLE traffic_fault_rules (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    app_id BIGINT NOT NULL COMMENT '应用ID',
    
    type ENUM('delay', 'abort') DEFAULT 'delay' COMMENT '故障类型',
    path VARCHAR(500) DEFAULT '/' COMMENT '接口路径',
    
    -- 延迟注入
    delay_duration VARCHAR(20) DEFAULT '5s' COMMENT '延迟时间',
    
    -- 中断注入
    abort_code INT DEFAULT 500 COMMENT 'HTTP状态码',
    abort_message VARCHAR(500) COMMENT '错误消息',
    
    percentage INT DEFAULT 10 COMMENT '影响比例(1-100)',
    enabled TINYINT(1) DEFAULT 0 COMMENT '是否启用',
    
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    
    INDEX idx_app_id (app_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='故障注入规则表';

-- =====================================================
-- 8. 流量治理操作日志表
-- =====================================================
DROP TABLE IF EXISTS traffic_operation_logs;
CREATE TABLE traffic_operation_logs (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    app_id BIGINT NOT NULL COMMENT '应用ID',
    rule_type VARCHAR(50) NOT NULL COMMENT '规则类型',
    rule_id BIGINT COMMENT '规则ID',
    operation VARCHAR(50) NOT NULL COMMENT '操作类型(create/update/delete/enable/disable)',
    operator VARCHAR(100) COMMENT '操作人',
    old_value JSON COMMENT '旧值',
    new_value JSON COMMENT '新值',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    INDEX idx_app_id (app_id),
    INDEX idx_rule_type (rule_type),
    INDEX idx_created_at (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='流量治理操作日志表';
