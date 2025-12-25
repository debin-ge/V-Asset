-- 移除 cookies 表的 status 字段，状态改为由 expire_at 和 frozen_until 动态计算

-- 删除 status 字段上的索引
DROP INDEX IF EXISTS idx_cookies_status;

-- 移除 status 字段
ALTER TABLE cookies DROP COLUMN IF EXISTS status;
