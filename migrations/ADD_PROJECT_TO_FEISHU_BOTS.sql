-- 给 feishu_bots 表添加 project 字段
-- 用于标识机器人对应的项目

ALTER TABLE `feishu_bots` 
ADD COLUMN `project` VARCHAR(100) DEFAULT '' COMMENT '关联项目' AFTER `webhook_url`;

-- 验证
DESCRIBE feishu_bots;
