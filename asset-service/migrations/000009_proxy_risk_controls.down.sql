DROP INDEX IF EXISTS idx_platform_risk_states_cooldown;
DROP INDEX IF EXISTS idx_proxy_usage_events_category;
DROP INDEX IF EXISTS idx_proxy_usage_events_created_at;
DROP INDEX IF EXISTS idx_proxy_usage_events_proxy;
DROP INDEX IF EXISTS idx_proxy_usage_events_task;
DROP INDEX IF EXISTS idx_proxies_cooldown_until;
DROP INDEX IF EXISTS idx_proxies_risk_selection;

DROP TABLE IF EXISTS cookie_proxy_affinities;
DROP TABLE IF EXISTS platform_risk_states;
DROP TABLE IF EXISTS proxy_usage_events;

ALTER TABLE task_proxy_bindings DROP COLUMN IF EXISTS binding_generation;
ALTER TABLE task_proxy_bindings DROP COLUMN IF EXISTS expired_reason;
ALTER TABLE task_proxy_bindings DROP COLUMN IF EXISTS released_at;
ALTER TABLE task_proxy_bindings DROP COLUMN IF EXISTS failure_count;
ALTER TABLE task_proxy_bindings DROP COLUMN IF EXISTS last_error_category;

ALTER TABLE proxies DROP COLUMN IF EXISTS active_task_count;
ALTER TABLE proxies DROP COLUMN IF EXISTS max_concurrent;
ALTER TABLE proxies DROP COLUMN IF EXISTS last_fail_at;
ALTER TABLE proxies DROP COLUMN IF EXISTS last_error_category;
ALTER TABLE proxies DROP COLUMN IF EXISTS risk_score;
ALTER TABLE proxies DROP COLUMN IF EXISTS consecutive_fail_count;
ALTER TABLE proxies DROP COLUMN IF EXISTS cooldown_until;
