DROP INDEX IF EXISTS idx_task_proxy_bindings_status;
DROP INDEX IF EXISTS idx_task_proxy_bindings_proxy_lease_id;
DROP INDEX IF EXISTS idx_task_proxy_bindings_proxy_id;
DROP INDEX IF EXISTS idx_task_proxy_bindings_source_type;

DROP TABLE IF EXISTS task_proxy_bindings;

DROP INDEX IF EXISTS idx_proxy_source_policies_status;
DROP INDEX IF EXISTS idx_proxy_source_policies_scope;

DROP TABLE IF EXISTS proxy_source_policies;

DROP INDEX IF EXISTS idx_proxies_deleted_at;
DROP INDEX IF EXISTS idx_proxies_priority;

ALTER TABLE proxies DROP COLUMN IF EXISTS deleted_at;
ALTER TABLE proxies DROP COLUMN IF EXISTS remark;
ALTER TABLE proxies DROP COLUMN IF EXISTS platform_tags;
ALTER TABLE proxies DROP COLUMN IF EXISTS priority;
ALTER TABLE proxies DROP COLUMN IF EXISTS host;
