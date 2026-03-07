-- Proxy 任务语义接口升级
-- 1. 扩展手动代理池字段
-- 2. 新增来源策略表
-- 3. 新增任务级代理绑定表

ALTER TABLE proxies ADD COLUMN IF NOT EXISTS host VARCHAR(255);
ALTER TABLE proxies ADD COLUMN IF NOT EXISTS priority INT NOT NULL DEFAULT 0;
ALTER TABLE proxies ADD COLUMN IF NOT EXISTS platform_tags VARCHAR(255);
ALTER TABLE proxies ADD COLUMN IF NOT EXISTS remark VARCHAR(255);
ALTER TABLE proxies ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMP;

CREATE INDEX IF NOT EXISTS idx_proxies_priority ON proxies(priority);
CREATE INDEX IF NOT EXISTS idx_proxies_deleted_at ON proxies(deleted_at);

CREATE TABLE IF NOT EXISTS proxy_source_policies (
    id                          BIGSERIAL PRIMARY KEY,
    scope_type                  VARCHAR(20) NOT NULL DEFAULT 'global',
    scope_value                 VARCHAR(50),
    primary_source              VARCHAR(20) NOT NULL,
    fallback_source             VARCHAR(20),
    fallback_enabled            BOOLEAN NOT NULL DEFAULT TRUE,
    dynamic_timeout_ms          INT NOT NULL DEFAULT 3000,
    dynamic_retry_count         INT NOT NULL DEFAULT 2,
    dynamic_circuit_breaker_sec INT NOT NULL DEFAULT 60,
    min_lease_ttl_sec           INT NOT NULL DEFAULT 600,
    manual_selection_strategy   VARCHAR(30) NOT NULL DEFAULT 'lru',
    status                      SMALLINT NOT NULL DEFAULT 0,
    created_at                  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at                  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_proxy_source_policies_scope
ON proxy_source_policies(scope_type, COALESCE(scope_value, ''));

CREATE INDEX IF NOT EXISTS idx_proxy_source_policies_status
ON proxy_source_policies(status);

CREATE TABLE IF NOT EXISTS task_proxy_bindings (
    id                  BIGSERIAL PRIMARY KEY,
    task_id             VARCHAR(100) NOT NULL,
    source_type         VARCHAR(20) NOT NULL,
    source_policy_id    BIGINT,
    proxy_id            BIGINT,
    proxy_lease_id      VARCHAR(100),
    proxy_url_snapshot  TEXT NOT NULL,
    protocol            VARCHAR(10) NOT NULL,
    region              VARCHAR(50),
    platform            VARCHAR(50),
    expire_at           TIMESTAMP,
    bind_status         VARCHAR(20) NOT NULL DEFAULT 'bound',
    is_degraded         BOOLEAN NOT NULL DEFAULT FALSE,
    degrade_reason      VARCHAR(255),
    last_report_stage   VARCHAR(20),
    last_report_success BOOLEAN,
    last_report_at      TIMESTAMP,
    created_at          TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at          TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT uq_task_proxy_bindings_task UNIQUE(task_id),
    CONSTRAINT fk_task_proxy_bindings_proxy FOREIGN KEY (proxy_id) REFERENCES proxies(id),
    CONSTRAINT fk_task_proxy_bindings_policy FOREIGN KEY (source_policy_id) REFERENCES proxy_source_policies(id)
);

CREATE INDEX IF NOT EXISTS idx_task_proxy_bindings_source_type
ON task_proxy_bindings(source_type);

CREATE INDEX IF NOT EXISTS idx_task_proxy_bindings_proxy_id
ON task_proxy_bindings(proxy_id);

CREATE INDEX IF NOT EXISTS idx_task_proxy_bindings_proxy_lease_id
ON task_proxy_bindings(proxy_lease_id);

CREATE INDEX IF NOT EXISTS idx_task_proxy_bindings_status
ON task_proxy_bindings(bind_status);

INSERT INTO proxy_source_policies (
    scope_type,
    scope_value,
    primary_source,
    fallback_source,
    fallback_enabled,
    dynamic_timeout_ms,
    dynamic_retry_count,
    dynamic_circuit_breaker_sec,
    min_lease_ttl_sec,
    manual_selection_strategy,
    status
)
SELECT
    'global',
    NULL,
    'dynamic_api',
    'manual_pool',
    TRUE,
    3000,
    2,
    60,
    600,
    'lru',
    0
WHERE NOT EXISTS (
    SELECT 1
    FROM proxy_source_policies
    WHERE scope_type = 'global'
      AND scope_value IS NULL
);
