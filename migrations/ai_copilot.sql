-- AI Copilot 功能建表 SQL
-- 创建时间: 2026-01-15
-- 功能: 智能运维助手相关表

-- ============================================
-- AI 会话相关表
-- ============================================

-- 1. AI 会话表
CREATE TABLE IF NOT EXISTS `ai_conversations` (
  `id` varchar(36) NOT NULL COMMENT '会话UUID',
  `created_at` datetime(3) DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  `deleted_at` datetime(3) DEFAULT NULL,
  `user_id` bigint unsigned NOT NULL COMMENT '用户ID',
  `title` varchar(255) DEFAULT '' COMMENT '会话标题',
  `context` json DEFAULT NULL COMMENT '页面上下文JSON',
  `message_count` int DEFAULT 0 COMMENT '消息数量',
  `last_message_at` datetime(3) DEFAULT NULL COMMENT '最后消息时间',
  PRIMARY KEY (`id`),
  KEY `idx_ai_conv_user_id` (`user_id`),
  KEY `idx_ai_conv_deleted_at` (`deleted_at`),
  KEY `idx_ai_conv_last_message` (`last_message_at`),
  CONSTRAINT `fk_ai_conv_user` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='AI会话表';

-- 2. AI 消息表
CREATE TABLE IF NOT EXISTS `ai_messages` (
  `id` varchar(36) NOT NULL COMMENT '消息UUID',
  `created_at` datetime(3) DEFAULT CURRENT_TIMESTAMP(3),
  `conversation_id` varchar(36) NOT NULL COMMENT '会话ID',
  `role` enum('user','assistant','system','tool') NOT NULL COMMENT '消息角色',
  `content` text NOT NULL COMMENT '消息内容',
  `tool_calls` json DEFAULT NULL COMMENT '工具调用信息JSON',
  `tool_call_id` varchar(100) DEFAULT '' COMMENT '工具调用ID',
  `token_count` int DEFAULT 0 COMMENT 'Token数量',
  `status` varchar(20) DEFAULT 'complete' COMMENT '状态: pending/streaming/complete/error',
  `error_msg` text COMMENT '错误信息',
  `feedback_rating` varchar(20) DEFAULT NULL COMMENT '反馈评分: like/dislike',
  `feedback_comment` text COMMENT '反馈评论',
  `feedback_at` datetime(3) DEFAULT NULL COMMENT '反馈时间',
  PRIMARY KEY (`id`),
  KEY `idx_ai_msg_conversation` (`conversation_id`),
  KEY `idx_ai_msg_created` (`created_at`),
  KEY `idx_ai_msg_role` (`role`),
  KEY `idx_ai_msg_feedback` (`feedback_rating`),
  CONSTRAINT `fk_ai_msg_conv` FOREIGN KEY (`conversation_id`) REFERENCES `ai_conversations` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='AI消息表';

-- ============================================
-- AI 知识库相关表
-- ============================================

-- 3. AI 知识库表
CREATE TABLE IF NOT EXISTS `ai_knowledge` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `created_at` datetime(3) DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  `deleted_at` datetime(3) DEFAULT NULL,
  `title` varchar(255) NOT NULL COMMENT '知识标题',
  `content` text NOT NULL COMMENT '知识内容(Markdown)',
  `category` varchar(50) NOT NULL COMMENT '分类: application/traffic/approval/k8s/monitoring/cicd',
  `tags` json DEFAULT NULL COMMENT '标签列表JSON',
  `embedding` blob DEFAULT NULL COMMENT '向量嵌入(可选,用于语义搜索)',
  `is_active` tinyint(1) DEFAULT 1 COMMENT '是否启用',
  `view_count` int DEFAULT 0 COMMENT '查看次数',
  `created_by` bigint unsigned DEFAULT NULL COMMENT '创建人ID',
  `updated_by` bigint unsigned DEFAULT NULL COMMENT '更新人ID',
  PRIMARY KEY (`id`),
  KEY `idx_ai_knowledge_category` (`category`),
  KEY `idx_ai_knowledge_deleted_at` (`deleted_at`),
  KEY `idx_ai_knowledge_active` (`is_active`),
  KEY `idx_ai_knowledge_created_by` (`created_by`),
  FULLTEXT INDEX `idx_ai_knowledge_fulltext` (`title`, `content`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='AI知识库表';

-- ============================================
-- AI 操作审计相关表
-- ============================================

-- 4. AI 操作日志表
CREATE TABLE IF NOT EXISTS `ai_operation_logs` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `created_at` datetime(3) DEFAULT CURRENT_TIMESTAMP(3),
  `user_id` bigint unsigned NOT NULL COMMENT '用户ID',
  `username` varchar(100) DEFAULT '' COMMENT '用户名',
  `conversation_id` varchar(36) DEFAULT NULL COMMENT '会话ID',
  `message_id` varchar(36) DEFAULT NULL COMMENT '消息ID',
  `action` varchar(50) NOT NULL COMMENT '操作类型: restart_app/scale_pod/rollback/silence_alert等',
  `action_name` varchar(100) DEFAULT '' COMMENT '操作名称',
  `target_type` varchar(50) DEFAULT '' COMMENT '目标类型: application/pod/deployment/alert',
  `target_id` varchar(100) DEFAULT '' COMMENT '目标ID',
  `target_name` varchar(200) DEFAULT '' COMMENT '目标名称',
  `params` json DEFAULT NULL COMMENT '操作参数JSON',
  `result` json DEFAULT NULL COMMENT '操作结果JSON',
  `success` tinyint(1) NOT NULL DEFAULT 0 COMMENT '是否成功',
  `error_msg` text COMMENT '错误信息',
  `duration_ms` int DEFAULT 0 COMMENT '执行耗时(毫秒)',
  `ip_address` varchar(50) DEFAULT '' COMMENT '客户端IP',
  PRIMARY KEY (`id`),
  KEY `idx_ai_op_user_id` (`user_id`),
  KEY `idx_ai_op_conversation` (`conversation_id`),
  KEY `idx_ai_op_action` (`action`),
  KEY `idx_ai_op_target` (`target_type`, `target_id`),
  KEY `idx_ai_op_success` (`success`),
  KEY `idx_ai_op_created` (`created_at`),
  CONSTRAINT `fk_ai_op_user` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='AI操作审计日志表';

-- ============================================
-- AI LLM 配置相关表
-- ============================================

-- 5. AI LLM 配置表
CREATE TABLE IF NOT EXISTS `ai_llm_configs` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `created_at` datetime(3) DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  `deleted_at` datetime(3) DEFAULT NULL,
  `name` varchar(50) NOT NULL COMMENT '配置名称',
  `provider` varchar(50) NOT NULL COMMENT '提供商: openai/azure/qwen/zhipu/ollama',
  `api_url` varchar(255) NOT NULL COMMENT 'API地址',
  `api_key_encrypted` varchar(512) NOT NULL COMMENT '加密的API密钥',
  `model_name` varchar(100) NOT NULL COMMENT '模型名称',
  `max_tokens` int DEFAULT 4096 COMMENT '最大Token数',
  `temperature` decimal(3,2) DEFAULT 0.70 COMMENT '温度参数',
  `timeout_seconds` int DEFAULT 60 COMMENT '请求超时时间(秒)',
  `is_default` tinyint(1) DEFAULT 0 COMMENT '是否默认配置',
  `is_active` tinyint(1) DEFAULT 1 COMMENT '是否启用',
  `description` varchar(500) DEFAULT '' COMMENT '描述',
  `created_by` bigint unsigned DEFAULT NULL COMMENT '创建人ID',
  `updated_by` bigint unsigned DEFAULT NULL COMMENT '更新人ID',
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_ai_llm_name` (`name`),
  KEY `idx_ai_llm_provider` (`provider`),
  KEY `idx_ai_llm_default` (`is_default`),
  KEY `idx_ai_llm_active` (`is_active`),
  KEY `idx_ai_llm_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='AI LLM配置表';

-- ============================================
-- AI 用户反馈表
-- ============================================

-- 6. AI 消息反馈表
CREATE TABLE IF NOT EXISTS `ai_message_feedbacks` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `created_at` datetime(3) DEFAULT CURRENT_TIMESTAMP(3),
  `user_id` bigint unsigned NOT NULL COMMENT '用户ID',
  `message_id` varchar(36) NOT NULL COMMENT '消息ID',
  `conversation_id` varchar(36) NOT NULL COMMENT '会话ID',
  `feedback_type` enum('like','dislike') NOT NULL COMMENT '反馈类型',
  `comment` text COMMENT '反馈评论',
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_ai_feedback_user_msg` (`user_id`, `message_id`),
  KEY `idx_ai_feedback_message` (`message_id`),
  KEY `idx_ai_feedback_conversation` (`conversation_id`),
  KEY `idx_ai_feedback_type` (`feedback_type`),
  CONSTRAINT `fk_ai_feedback_user` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE,
  CONSTRAINT `fk_ai_feedback_msg` FOREIGN KEY (`message_id`) REFERENCES `ai_messages` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='AI消息反馈表';
