-- 回滚：删除添加的元数据字段
ALTER TABLE download_history 
DROP COLUMN IF EXISTS thumbnail,
DROP COLUMN IF EXISTS duration,
DROP COLUMN IF EXISTS author;
