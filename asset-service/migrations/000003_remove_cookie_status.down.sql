-- 恢复 cookies 表的 status 字段

-- 添加 status 字段
ALTER TABLE cookies ADD COLUMN status SMALLINT NOT NULL DEFAULT 0;

-- 重新创建索引
CREATE INDEX IF NOT EXISTS idx_cookies_status ON cookies(status);
