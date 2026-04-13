-- CronHPA table for scheduled HPA scaling rules.

CREATE TABLE IF NOT EXISTS `cron_hpa` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `cluster_id` bigint unsigned NOT NULL,
  `name` varchar(100) NOT NULL,
  `namespace` varchar(100) NOT NULL,
  `target_kind` varchar(50) NOT NULL,
  `target_name` varchar(100) NOT NULL,
  `enabled` tinyint(1) DEFAULT 1,
  `schedules` json NOT NULL,
  `created_by` varchar(100) DEFAULT '',
  `created_at` datetime(3) DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` datetime(3) DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_cron_hpa_cluster_ns_name` (`cluster_id`, `namespace`, `name`),
  KEY `idx_cron_hpa_cluster_enabled` (`cluster_id`, `enabled`),
  KEY `idx_cron_hpa_namespace` (`namespace`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='CronHPA';
