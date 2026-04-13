-- 修复 traffic_loadbalance_config 表的 hash_key 字段
-- 将 ENUM 改为 VARCHAR，允许为空

ALTER TABLE traffic_loadbalance_config 
MODIFY COLUMN hash_key VARCHAR(20) DEFAULT NULL COMMENT '哈希键类型(header/cookie/source_ip/query_param)';
