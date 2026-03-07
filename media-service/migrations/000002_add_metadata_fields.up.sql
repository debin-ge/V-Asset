-- 添加缺失的元数据字段到 download_history 表
ALTER TABLE download_history 
ADD COLUMN IF NOT EXISTS thumbnail VARCHAR(1000),
ADD COLUMN IF NOT EXISTS duration BIGINT,
ADD COLUMN IF NOT EXISTS author VARCHAR(200);

-- 为新字段添加索引（可选，用于优化查询）
CREATE INDEX IF NOT EXISTS idx_download_history_author ON download_history(author);
