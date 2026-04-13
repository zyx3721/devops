-- 修复 applications 表结构，使其与 Go 模型匹配
-- 执行时间: 2026-01-12
-- 说明: 逐条执行，如果字段已存在会报错，忽略即可

-- 如果有 repo_url 列，先重命名为 git_repo
-- ALTER TABLE `applications` CHANGE COLUMN `repo_url` `git_repo` VARCHAR(500) DEFAULT '';

-- 添加缺失的字段
ALTER TABLE `applications` ADD COLUMN `git_repo` VARCHAR(500) DEFAULT '' AFTER `description`;
ALTER TABLE `applications` ADD COLUMN `jenkins_instance_id` BIGINT UNSIGNED DEFAULT NULL AFTER `status`;
ALTER TABLE `applications` ADD COLUMN `jenkins_job_name` VARCHAR(200) DEFAULT '' AFTER `jenkins_instance_id`;
ALTER TABLE `applications` ADD COLUMN `k8s_cluster_id` BIGINT UNSIGNED DEFAULT NULL AFTER `jenkins_job_name`;
ALTER TABLE `applications` ADD COLUMN `k8s_namespace` VARCHAR(100) DEFAULT '' AFTER `k8s_cluster_id`;
ALTER TABLE `applications` ADD COLUMN `k8s_deployment` VARCHAR(200) DEFAULT '' AFTER `k8s_namespace`;
ALTER TABLE `applications` ADD COLUMN `notify_platform` VARCHAR(50) DEFAULT '' AFTER `k8s_deployment`;
ALTER TABLE `applications` ADD COLUMN `notify_app_id` BIGINT UNSIGNED DEFAULT NULL AFTER `notify_platform`;
ALTER TABLE `applications` ADD COLUMN `notify_receive_id` VARCHAR(200) DEFAULT '' AFTER `notify_app_id`;
ALTER TABLE `applications` ADD COLUMN `notify_receive_type` VARCHAR(50) DEFAULT '' AFTER `notify_receive_id`;

-- 添加索引
ALTER TABLE `applications` ADD INDEX `idx_jenkins_instance` (`jenkins_instance_id`);
ALTER TABLE `applications` ADD INDEX `idx_k8s_cluster` (`k8s_cluster_id`);
ALTER TABLE `applications` ADD INDEX `idx_created_by` (`created_by`);
