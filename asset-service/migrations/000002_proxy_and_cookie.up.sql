-- 代理和 Cookie 管理表

-- 代理表
CREATE TABLE IF NOT EXISTS proxies (
    id                BIGSERIAL PRIMARY KEY,
    ip                VARCHAR(45) NOT NULL,
    port              INT NOT NULL,
    username          VARCHAR(100),
    password          VARCHAR(100),
    protocol          VARCHAR(10) NOT NULL DEFAULT 'http',
    region            VARCHAR(50),
    status            SMALLINT NOT NULL DEFAULT 0,
    last_check_at     TIMESTAMP,
    last_check_result VARCHAR(500),
    success_count     INT NOT NULL DEFAULT 0,
    fail_count        INT NOT NULL DEFAULT 0,
    last_used_at      TIMESTAMP,
    created_at        TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at        TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    UNIQUE(ip, port)
);

-- Cookie 表
CREATE TABLE IF NOT EXISTS cookies (
    id              BIGSERIAL PRIMARY KEY,
    platform        VARCHAR(50) NOT NULL,
    name            VARCHAR(100) NOT NULL,
    content         TEXT NOT NULL,
    status          SMALLINT NOT NULL DEFAULT 0,
    expire_at       TIMESTAMP,
    frozen_until    TIMESTAMP,
    freeze_seconds  INT NOT NULL DEFAULT 0,
    last_used_at    TIMESTAMP,
    use_count       INT NOT NULL DEFAULT 0,
    success_count   INT NOT NULL DEFAULT 0,
    fail_count      INT NOT NULL DEFAULT 0,
    created_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- 索引
CREATE INDEX IF NOT EXISTS idx_proxies_status ON proxies(status);
CREATE INDEX IF NOT EXISTS idx_proxies_protocol ON proxies(protocol);
CREATE INDEX IF NOT EXISTS idx_proxies_region ON proxies(region);
CREATE INDEX IF NOT EXISTS idx_proxies_last_used_at ON proxies(last_used_at);

CREATE INDEX IF NOT EXISTS idx_cookies_platform ON cookies(platform);
CREATE INDEX IF NOT EXISTS idx_cookies_status ON cookies(status);
CREATE INDEX IF NOT EXISTS idx_cookies_frozen_until ON cookies(frozen_until);
CREATE INDEX IF NOT EXISTS idx_cookies_expire_at ON cookies(expire_at);
