-- 创建 users 表
CREATE TABLE IF NOT EXISTS users (
    id            VARCHAR(36) PRIMARY KEY,  -- UUID 格式
    email         VARCHAR(255) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    nickname      VARCHAR(100),
    avatar_url    VARCHAR(500),
    role          INT NOT NULL DEFAULT 1,  -- 1:普通用户,2:VIP,99:管理员
    status        INT NOT NULL DEFAULT 1,  -- 1:正常,0:禁用
    created_at    TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at    TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_login_at TIMESTAMP
);

-- 创建索引
CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
CREATE INDEX IF NOT EXISTS idx_users_status ON users(status);
CREATE INDEX IF NOT EXISTS idx_users_role ON users(role);

-- 创建 user_sessions 表
CREATE TABLE IF NOT EXISTS user_sessions (
    id            BIGSERIAL PRIMARY KEY,
    user_id       VARCHAR(36) NOT NULL,
    refresh_token VARCHAR(255) NOT NULL UNIQUE,
    token_hash    VARCHAR(64) NOT NULL,
    device_info   TEXT,
    ip_address    INET,
    expires_at    TIMESTAMP NOT NULL,
    last_used_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_at    TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- 创建索引
CREATE INDEX IF NOT EXISTS idx_sessions_user_id ON user_sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_sessions_refresh_token ON user_sessions(refresh_token);
CREATE INDEX IF NOT EXISTS idx_sessions_expires_at ON user_sessions(expires_at);

-- 创建更新时间触发器函数
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

-- 为 users 表添加更新时间触发器
CREATE TRIGGER update_users_updated_at BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
