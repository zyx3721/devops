-- 流水线模板相关表
-- 用于支持流水线模板市场和可视化编排功能

-- 流水线模板表
CREATE TABLE IF NOT EXISTS pipeline_templates (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(100) NOT NULL COMMENT '模板名称',
    description VARCHAR(500) COMMENT '模板描述',
    category VARCHAR(50) COMMENT '模板分类: build, deploy, test, release',
    language VARCHAR(50) COMMENT '编程语言: java, go, nodejs, python',
    framework VARCHAR(50) COMMENT '框架: spring, gin, express, django',
    config_json JSON NOT NULL COMMENT '流水线配置',
    icon_url VARCHAR(500) COMMENT '图标URL',
    is_builtin BOOLEAN DEFAULT FALSE COMMENT '是否内置模板',
    is_public BOOLEAN DEFAULT TRUE COMMENT '是否公开',
    usage_count INT DEFAULT 0 COMMENT '使用次数',
    rating DECIMAL(3,2) DEFAULT 0 COMMENT '评分',
    rating_count INT DEFAULT 0 COMMENT '评分人数',
    created_by VARCHAR(100) COMMENT '创建人',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY uk_name (name),
    KEY idx_category (category),
    KEY idx_language (language)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='流水线模板表';

-- 模板评分表
CREATE TABLE IF NOT EXISTS pipeline_template_ratings (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    template_id BIGINT UNSIGNED NOT NULL COMMENT '模板ID',
    user_id INT UNSIGNED NOT NULL COMMENT '用户ID',
    rating TINYINT NOT NULL COMMENT '评分(1-5)',
    comment VARCHAR(500) COMMENT '评价',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    KEY idx_template_id (template_id),
    KEY idx_user_id (user_id),
    UNIQUE KEY uk_template_user (template_id, user_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='模板评分表';

-- 阶段模板表（用于拖拽式设计）
CREATE TABLE IF NOT EXISTS pipeline_stage_templates (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(100) NOT NULL COMMENT '阶段名称',
    description VARCHAR(500) COMMENT '阶段描述',
    category VARCHAR(50) COMMENT '分类: source, build, test, deploy, notify',
    icon_name VARCHAR(50) COMMENT '图标名称',
    color VARCHAR(20) COMMENT '颜色',
    config_json JSON COMMENT '默认配置',
    is_builtin BOOLEAN DEFAULT TRUE COMMENT '是否内置',
    sort_order INT DEFAULT 0 COMMENT '排序',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    KEY idx_category (category)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='阶段模板表';

-- 步骤模板表（用于拖拽式设计）
CREATE TABLE IF NOT EXISTS pipeline_step_templates (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(100) NOT NULL COMMENT '步骤名称',
    description VARCHAR(500) COMMENT '步骤描述',
    step_type VARCHAR(50) NOT NULL COMMENT '步骤类型: git, shell, docker_build, k8s_deploy',
    category VARCHAR(50) COMMENT '分类',
    icon_name VARCHAR(50) COMMENT '图标名称',
    config_schema JSON COMMENT '配置Schema',
    default_json JSON COMMENT '默认配置',
    is_builtin BOOLEAN DEFAULT TRUE COMMENT '是否内置',
    sort_order INT DEFAULT 0 COMMENT '排序',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    KEY idx_step_type (step_type),
    KEY idx_category (category)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='步骤模板表';
