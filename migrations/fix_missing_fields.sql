-- ============================================
-- 修复日志中发现的缺失字段补丁
-- 创建时间：2026-04-14
-- 作者：Claude Opus 4.6
--
-- 说明：根据 test.log 中的错误日志生成的字段补丁
-- 执行前请备份数据库！
-- ============================================

SET NAMES utf8mb4;
SET FOREIGN_KEY_CHECKS = 0;

-- ============================================
-- 1. 修复 health_check_configs 表
-- ============================================
ALTER TABLE `health_check_configs`
  ADD COLUMN IF NOT EXISTS `last_status` varchar(20) DEFAULT 'unknown' COMMENT '最后检查状态: healthy/unhealthy/unknown' AFTER `enabled`,
  ADD COLUMN IF NOT EXISTS `last_checked_at` datetime(3) DEFAULT NULL COMMENT '最后检查时间' AFTER `last_status`;

CREATE INDEX IF NOT EXISTS `idx_hcc_last_status` ON `health_check_configs`(`last_status`);

-- ============================================
-- 2. 修复 resource_costs 表
-- ============================================
ALTER TABLE `resource_costs`
  ADD COLUMN IF NOT EXISTS `total_cost` decimal(14,4) DEFAULT 0.0000 COMMENT '总成本' AFTER `storage_cost`,
  ADD COLUMN IF NOT EXISTS `cpu_request` decimal(10,2) DEFAULT 0.00 COMMENT 'CPU 请求量(核)' AFTER `cpu_usage`,
  ADD COLUMN IF NOT EXISTS `memory_request` decimal(10,2) DEFAULT 0.00 COMMENT '内存请求量(GB)' AFTER `memory_usage`;

-- 更新 total_cost 字段（如果已有数据）
UPDATE `resource_costs`
SET `total_cost` = COALESCE(`cpu_cost`, 0) + COALESCE(`memory_cost`, 0) + COALESCE(`storage_cost`, 0)
WHERE `total_cost` = 0;

-- ============================================
-- 3. 修复 cost_suggestions 表
-- ============================================
ALTER TABLE `cost_suggestions`
  ADD COLUMN IF NOT EXISTS `savings` decimal(14,4) DEFAULT 0.0000 COMMENT '预计节省金额' AFTER `suggestion`;

-- ============================================
-- 4. 修复 cost_budgets 表
-- ============================================
ALTER TABLE `cost_budgets`
  ADD COLUMN IF NOT EXISTS `monthly_budget` decimal(14,4) DEFAULT 0.0000 COMMENT '月度预算' AFTER `period`,
  ADD COLUMN IF NOT EXISTS `current_cost` decimal(14,4) DEFAULT 0.0000 COMMENT '当前花费' AFTER `monthly_budget`;

-- ============================================
-- 5. 修复 resource_activities 表
-- ============================================
ALTER TABLE `resource_activities`
  ADD COLUMN IF NOT EXISTS `is_zombie` tinyint(1) DEFAULT 0 COMMENT '是否为僵尸资源' AFTER `activity_type`,
  ADD COLUMN IF NOT EXISTS `last_active_at` datetime(3) DEFAULT NULL COMMENT '最后活跃时间' AFTER `is_zombie`;

CREATE INDEX IF NOT EXISTS `idx_ra_is_zombie` ON `resource_activities`(`is_zombie`);

-- ============================================
-- 6. 修复 image_scans 表
-- ============================================
ALTER TABLE `image_scans`
  ADD COLUMN IF NOT EXISTS `status` varchar(20) DEFAULT 'pending' COMMENT '扫描状态: pending/scanning/completed/failed' AFTER `scan_status`;

-- 如果已有 scan_status 字段，同步数据
UPDATE `image_scans`
SET `status` = `scan_status`
WHERE `status` = 'pending' AND `scan_status` IS NOT NULL;

CREATE INDEX IF NOT EXISTS `idx_is_status` ON `image_scans`(`status`);

-- ============================================
-- 7. 修复 config_checks 表
-- ============================================
ALTER TABLE `config_checks`
  ADD COLUMN IF NOT EXISTS `critical_count` int DEFAULT 0 COMMENT '严重问题数' AFTER `status`,
  ADD COLUMN IF NOT EXISTS `high_count` int DEFAULT 0 COMMENT '高危问题数' AFTER `critical_count`,
  ADD COLUMN IF NOT EXISTS `medium_count` int DEFAULT 0 COMMENT '中危问题数' AFTER `high_count`,
  ADD COLUMN IF NOT EXISTS `low_count` int DEFAULT 0 COMMENT '低危问题数' AFTER `medium_count`,
  ADD COLUMN IF NOT EXISTS `passed_count` int DEFAULT 0 COMMENT '通过数' AFTER `low_count`,
  ADD COLUMN IF NOT EXISTS `checked_at` datetime(3) DEFAULT NULL COMMENT '检查时间' AFTER `passed_count`;

CREATE INDEX IF NOT EXISTS `idx_cc_checked_at` ON `config_checks`(`checked_at`);

SET FOREIGN_KEY_CHECKS = 1;

-- ============================================
-- 修复完成
-- ============================================
