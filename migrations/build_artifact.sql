-- 构建优化和制品管理相关表

-- ========== 构建优化 ==========

-- 构建缓存表
CREATE TABLE IF NOT EXISTS build_caches (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    pipeline_id BIGINT UNSIGNED NOT NULL COMMENT '流水线ID',
    cache_key VARCHAR(255) NOT NULL COMMENT '缓存键',
    cache_type VARCHAR(50) COMMENT '缓存类型: maven, npm, go, pip, docker_layer',
    cache_path VARCHAR(500) COMMENT '缓存路径',
    size_bytes BIGINT DEFAULT 0 COMMENT '缓存大小(字节)',
    hit_count INT DEFAULT 0 COMMENT '命中次数',
    last_hit_at TIMESTAMP COMMENT '最后命中时间',
    expire_at TIMESTAMP COMMENT '过期时间',
    storage_type VARCHAR(50) DEFAULT 'local' COMMENT '存储类型: local, s3, oss',
    storage_url VARCHAR(500) COMMENT '存储URL',
    checksum VARCHAR(64) COMMENT '校验和',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY uk_cache_key (cache_key),
    KEY idx_pipeline_id (pipeline_id),
    KEY idx_expire_at (expire_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='构建缓存表';

-- 构建资源配额表
CREATE TABLE IF NOT EXISTS build_resource_quotas (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(100) NOT NULL COMMENT '配额名称',
    description VARCHAR(500) COMMENT '描述',
    project_id BIGINT UNSIGNED COMMENT '项目ID(空表示全局)',
    max_cpu VARCHAR(20) DEFAULT '2' COMMENT '最大CPU',
    max_memory VARCHAR(20) DEFAULT '4Gi' COMMENT '最大内存',
    max_storage VARCHAR(20) DEFAULT '10Gi' COMMENT '最大存储',
    max_concurrent INT DEFAULT 5 COMMENT '最大并发构建数',
    max_duration INT DEFAULT 3600 COMMENT '最大构建时长(秒)',
    priority INT DEFAULT 0 COMMENT '优先级',
    is_default BOOLEAN DEFAULT FALSE COMMENT '是否默认配额',
    enabled BOOLEAN DEFAULT TRUE COMMENT '是否启用',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY uk_name (name),
    KEY idx_project_id (project_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='构建资源配额表';

-- 构建资源使用记录表
CREATE TABLE IF NOT EXISTS build_resource_usages (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    pipeline_id BIGINT UNSIGNED NOT NULL COMMENT '流水线ID',
    run_id BIGINT UNSIGNED NOT NULL COMMENT '执行ID',
    quota_id BIGINT UNSIGNED COMMENT '配额ID',
    cpu_used VARCHAR(20) COMMENT 'CPU使用量',
    memory_used VARCHAR(20) COMMENT '内存使用量',
    storage_used VARCHAR(20) COMMENT '存储使用量',
    duration_sec INT COMMENT '构建时长(秒)',
    cache_hit BOOLEAN DEFAULT FALSE COMMENT '是否命中缓存',
    cache_saved BIGINT DEFAULT 0 COMMENT '缓存节省时间(秒)',
    started_at TIMESTAMP COMMENT '开始时间',
    completed_at TIMESTAMP COMMENT '完成时间',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    KEY idx_pipeline_id (pipeline_id),
    KEY idx_run_id (run_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='构建资源使用记录表';

-- 并行构建配置表
CREATE TABLE IF NOT EXISTS parallel_build_configs (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    pipeline_id BIGINT UNSIGNED NOT NULL COMMENT '流水线ID',
    enabled BOOLEAN DEFAULT TRUE COMMENT '是否启用并行构建',
    max_parallel INT DEFAULT 3 COMMENT '最大并行数',
    fail_fast BOOLEAN DEFAULT TRUE COMMENT '快速失败',
    parallel_stages JSON COMMENT '可并行的阶段',
    dependency_graph JSON COMMENT '依赖图',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY uk_pipeline_id (pipeline_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='并行构建配置表';

-- ========== 制品管理 ==========

-- 制品仓库表
CREATE TABLE IF NOT EXISTS artifact_repositories (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(100) NOT NULL COMMENT '仓库名称',
    description VARCHAR(500) COMMENT '描述',
    type VARCHAR(50) NOT NULL COMMENT '仓库类型: docker, maven, npm, pypi, generic',
    url VARCHAR(500) NOT NULL COMMENT '仓库地址',
    username VARCHAR(100) COMMENT '用户名',
    password VARCHAR(500) COMMENT '密码(加密)',
    is_default BOOLEAN DEFAULT FALSE COMMENT '是否默认仓库',
    is_public BOOLEAN DEFAULT FALSE COMMENT '是否公开',
    enabled BOOLEAN DEFAULT TRUE COMMENT '是否启用',
    created_by VARCHAR(100) COMMENT '创建人',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY uk_name (name),
    KEY idx_type (type)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='制品仓库表';

-- 制品表
CREATE TABLE IF NOT EXISTS artifacts (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    repository_id BIGINT UNSIGNED NOT NULL COMMENT '仓库ID',
    name VARCHAR(200) NOT NULL COMMENT '制品名称',
    group_id VARCHAR(200) COMMENT '组ID(Maven)',
    artifact_id VARCHAR(200) COMMENT '制品ID(Maven)',
    type VARCHAR(50) COMMENT '制品类型: jar, war, docker, npm, wheel',
    description VARCHAR(500) COMMENT '描述',
    latest_version VARCHAR(100) COMMENT '最新版本',
    download_cnt BIGINT DEFAULT 0 COMMENT '下载次数',
    tags VARCHAR(500) COMMENT '标签(逗号分隔)',
    created_by VARCHAR(100) COMMENT '创建人',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    KEY idx_repository_id (repository_id),
    KEY idx_name (name),
    KEY idx_group_id (group_id),
    KEY idx_artifact_id (artifact_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='制品表';

-- 制品版本表
CREATE TABLE IF NOT EXISTS artifact_versions (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    artifact_id BIGINT UNSIGNED NOT NULL COMMENT '制品ID',
    version VARCHAR(100) NOT NULL COMMENT '版本号',
    size_bytes BIGINT DEFAULT 0 COMMENT '大小(字节)',
    checksum VARCHAR(64) COMMENT 'SHA256校验和',
    download_url VARCHAR(500) COMMENT '下载地址',
    metadata JSON COMMENT '元数据',
    pipeline_id BIGINT UNSIGNED COMMENT '来源流水线ID',
    run_id BIGINT UNSIGNED COMMENT '来源执行ID',
    git_commit VARCHAR(64) COMMENT 'Git提交',
    git_branch VARCHAR(100) COMMENT 'Git分支',
    build_number INT COMMENT '构建号',
    download_cnt BIGINT DEFAULT 0 COMMENT '下载次数',
    scan_status VARCHAR(20) DEFAULT 'pending' COMMENT '扫描状态: pending, scanning, passed, failed',
    scan_result JSON COMMENT '扫描结果',
    is_release BOOLEAN DEFAULT FALSE COMMENT '是否正式版本',
    released_at TIMESTAMP COMMENT '发布时间',
    released_by VARCHAR(100) COMMENT '发布人',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    KEY idx_artifact_id (artifact_id),
    KEY idx_version (version),
    KEY idx_pipeline_id (pipeline_id),
    KEY idx_run_id (run_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='制品版本表';

-- 制品扫描结果表
CREATE TABLE IF NOT EXISTS artifact_scan_results (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    version_id BIGINT UNSIGNED NOT NULL COMMENT '版本ID',
    scan_type VARCHAR(50) NOT NULL COMMENT '扫描类型: vulnerability, license, quality',
    scanner VARCHAR(50) COMMENT '扫描器: trivy, sonarqube',
    status VARCHAR(20) COMMENT '状态: passed, failed, warning',
    critical_count INT DEFAULT 0 COMMENT '严重漏洞数',
    high_count INT DEFAULT 0 COMMENT '高危漏洞数',
    medium_count INT DEFAULT 0 COMMENT '中危漏洞数',
    low_count INT DEFAULT 0 COMMENT '低危漏洞数',
    details JSON COMMENT '详细结果',
    report_url VARCHAR(500) COMMENT '报告URL',
    scanned_at TIMESTAMP COMMENT '扫描时间',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    KEY idx_version_id (version_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='制品扫描结果表';

-- 制品晋级记录表
CREATE TABLE IF NOT EXISTS artifact_promotions (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    version_id BIGINT UNSIGNED NOT NULL COMMENT '版本ID',
    from_repo_id BIGINT UNSIGNED COMMENT '源仓库ID',
    to_repo_id BIGINT UNSIGNED COMMENT '目标仓库ID',
    from_env VARCHAR(50) COMMENT '源环境: dev, test, staging',
    to_env VARCHAR(50) COMMENT '目标环境: test, staging, prod',
    status VARCHAR(20) COMMENT '状态: pending, approved, rejected, completed',
    approval_id BIGINT UNSIGNED COMMENT '审批ID',
    promoted_by VARCHAR(100) COMMENT '晋级人',
    promoted_at TIMESTAMP COMMENT '晋级时间',
    comment VARCHAR(500) COMMENT '备注',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    KEY idx_version_id (version_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='制品晋级记录表';
