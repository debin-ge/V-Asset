-- Asset Service 数据库初始化脚本

-- 用户配额表
CREATE TABLE IF NOT EXISTS user_quotas (
    id          BIGSERIAL PRIMARY KEY,
    user_id     VARCHAR(36) NOT NULL UNIQUE,
    daily_limit INT NOT NULL DEFAULT 10,
    daily_used  INT NOT NULL DEFAULT 0,
    reset_at    TIMESTAMP NOT NULL,
    updated_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- 配额表索引
CREATE INDEX IF NOT EXISTS idx_user_quotas_user_id ON user_quotas(user_id);
CREATE INDEX IF NOT EXISTS idx_user_quotas_reset_at ON user_quotas(reset_at);
