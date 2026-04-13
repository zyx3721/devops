-- 为日志告警历史表添加静默相关字段
-- 执行时间: 2026-01-13

-- 添加 silenced 字段（如果已存在会报错，可忽略）
ALTER TABLE log_alert_history 
ADD COLUMN silenced BOOLEAN DEFAULT FALSE COMMENT '是否被静默';

-- 添加 silence_id 字段（如果已存在会报错，可忽略）
ALTER TABLE log_alert_history 
ADD COLUMN silence_id INT UNSIGNED NULL COMMENT '静默规则ID';

-- 为 silenced 字段添加索引
CREATE INDEX idx_log_alert_history_silenced ON log_alert_history(silenced);

-- 为 silence_id 字段添加索引
CREATE INDEX idx_log_alert_history_silence_id ON log_alert_history(silence_id);

-- 添加外键约束（可选，如果需要严格的引用完整性）
-- ALTER TABLE log_alert_history 
-- ADD CONSTRAINT fk_log_alert_history_silence 
-- FOREIGN KEY (silence_id) REFERENCES alert_silences(id) ON DELETE SET NULL;
