-- 回滚代理和 Cookie 管理表

DROP INDEX IF EXISTS idx_cookies_expire_at;
DROP INDEX IF EXISTS idx_cookies_frozen_until;
DROP INDEX IF EXISTS idx_cookies_status;
DROP INDEX IF EXISTS idx_cookies_platform;

DROP INDEX IF EXISTS idx_proxies_last_used_at;
DROP INDEX IF EXISTS idx_proxies_region;
DROP INDEX IF EXISTS idx_proxies_protocol;
DROP INDEX IF EXISTS idx_proxies_status;

DROP TABLE IF EXISTS cookies;
DROP TABLE IF EXISTS proxies;
