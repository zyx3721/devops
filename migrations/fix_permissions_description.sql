-- Fix missing description column in the permissions table.
-- Safe to rerun on partially initialized databases.

SET @permissions_description_exists := (
  SELECT COUNT(*)
  FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE()
    AND TABLE_NAME = 'permissions'
    AND COLUMN_NAME = 'description'
);
SET @permissions_description_sql := IF(
  @permissions_description_exists = 0,
  'ALTER TABLE `permissions` ADD COLUMN `description` TEXT AFTER `action`',
  'SELECT 1'
);
PREPARE permissions_description_stmt FROM @permissions_description_sql;
EXECUTE permissions_description_stmt;
DEALLOCATE PREPARE permissions_description_stmt;

SET @permissions_resource_index_exists := (
  SELECT COUNT(*)
  FROM information_schema.STATISTICS
  WHERE TABLE_SCHEMA = DATABASE()
    AND TABLE_NAME = 'permissions'
    AND INDEX_NAME = 'idx_permissions_resource'
);
SET @permissions_resource_index_sql := IF(
  @permissions_resource_index_exists = 0,
  'ALTER TABLE `permissions` ADD KEY `idx_permissions_resource` (`resource`)',
  'SELECT 1'
);
PREPARE permissions_resource_index_stmt FROM @permissions_resource_index_sql;
EXECUTE permissions_resource_index_stmt;
DEALLOCATE PREPARE permissions_resource_index_stmt;
