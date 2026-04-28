-- Proxy risk controls and observability.

ALTER TABLE proxies ADD COLUMN IF NOT EXISTS cooldown_until TIMESTAMP;
ALTER TABLE proxies ADD COLUMN IF NOT EXISTS consecutive_fail_count INT NOT NULL DEFAULT 0;
ALTER TABLE proxies ADD COLUMN IF NOT EXISTS risk_score INT NOT NULL DEFAULT 0;
ALTER TABLE proxies ADD COLUMN IF NOT EXISTS last_error_category VARCHAR(50);
ALTER TABLE proxies ADD COLUMN IF NOT EXISTS last_fail_at TIMESTAMP;
ALTER TABLE proxies ADD COLUMN IF NOT EXISTS max_concurrent INT NOT NULL DEFAULT 1;
ALTER TABLE proxies ADD COLUMN IF NOT EXISTS active_task_count INT NOT NULL DEFAULT 0;

ALTER TABLE task_proxy_bindings ADD COLUMN IF NOT EXISTS last_error_category VARCHAR(50);
ALTER TABLE task_proxy_bindings ADD COLUMN IF NOT EXISTS failure_count INT NOT NULL DEFAULT 0;
ALTER TABLE task_proxy_bindings ADD COLUMN IF NOT EXISTS released_at TIMESTAMP;
ALTER TABLE task_proxy_bindings ADD COLUMN IF NOT EXISTS expired_reason VARCHAR(100);
ALTER TABLE task_proxy_bindings ADD COLUMN IF NOT EXISTS binding_generation INT NOT NULL DEFAULT 1;

CREATE TABLE IF NOT EXISTS proxy_usage_events (
    id               BIGSERIAL PRIMARY KEY,
    task_id          VARCHAR(100),
    proxy_id         BIGINT REFERENCES proxies(id),
    proxy_lease_id   VARCHAR(100),
    source_type      VARCHAR(20),
    stage            VARCHAR(20) NOT NULL,
    platform         VARCHAR(50),
    success          BOOLEAN NOT NULL,
    error_category   VARCHAR(50),
    error_message    TEXT,
    created_at       TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS platform_risk_states (
    platform                  VARCHAR(50) PRIMARY KEY,
    cooldown_until            TIMESTAMP,
    rate_limit_level          INT NOT NULL DEFAULT 0,
    recent_bot_detected_count INT NOT NULL DEFAULT 0,
    recent_rate_limited_count INT NOT NULL DEFAULT 0,
    updated_at                TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS cookie_proxy_affinities (
    cookie_id      BIGINT NOT NULL REFERENCES cookies(id) ON DELETE CASCADE,
    platform       VARCHAR(50) NOT NULL,
    proxy_id       BIGINT REFERENCES proxies(id),
    source_type    VARCHAR(20),
    region         VARCHAR(50),
    last_used_at   TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    success_count  INT NOT NULL DEFAULT 0,
    fail_count     INT NOT NULL DEFAULT 0,
    PRIMARY KEY (cookie_id, platform)
);

CREATE INDEX IF NOT EXISTS idx_proxies_risk_selection
ON proxies(status, deleted_at, cooldown_until, risk_score, active_task_count, max_concurrent);

CREATE INDEX IF NOT EXISTS idx_proxies_cooldown_until
ON proxies(cooldown_until);

CREATE INDEX IF NOT EXISTS idx_proxy_usage_events_task
ON proxy_usage_events(task_id);

CREATE INDEX IF NOT EXISTS idx_proxy_usage_events_proxy
ON proxy_usage_events(proxy_id);

CREATE INDEX IF NOT EXISTS idx_proxy_usage_events_created_at
ON proxy_usage_events(created_at);

CREATE INDEX IF NOT EXISTS idx_proxy_usage_events_category
ON proxy_usage_events(error_category);

CREATE INDEX IF NOT EXISTS idx_platform_risk_states_cooldown
ON platform_risk_states(cooldown_until);
