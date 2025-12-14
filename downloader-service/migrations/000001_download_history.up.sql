-- 创建 download_history 表
CREATE TABLE IF NOT EXISTS download_history (
    id            BIGSERIAL PRIMARY KEY,
    task_id       VARCHAR(64) NOT NULL UNIQUE,
    user_id       VARCHAR(36) NOT NULL,
    url           TEXT NOT NULL,
    platform      VARCHAR(50),
    title         VARCHAR(500),
    mode          VARCHAR(20) NOT NULL,     -- quick_download, archive
    quality       VARCHAR(20),              -- 1080p, 720p, etc.
    file_path     VARCHAR(1000),            -- 本地文件路径
    file_name     VARCHAR(500),
    file_size     BIGINT,                   -- 字节
    file_hash     CHAR(32),                 -- MD5
    status        INT NOT NULL DEFAULT 0,   -- 状态机: 0待处理,1下载中,2完成,3失败,4待清理,5已过期
    error_message TEXT,                     -- 错误信息
    retry_count   INT DEFAULT 0,
    expire_at     TIMESTAMP,                -- 清理时间(仅quick_download)
    created_at    TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    started_at    TIMESTAMP,
    completed_at  TIMESTAMP
);

-- 索引
CREATE INDEX IF NOT EXISTS idx_download_history_task_id ON download_history(task_id);
CREATE INDEX IF NOT EXISTS idx_download_history_user_id ON download_history(user_id);
CREATE INDEX IF NOT EXISTS idx_download_history_status ON download_history(status);
CREATE INDEX IF NOT EXISTS idx_download_history_expire_at ON download_history(expire_at);
CREATE INDEX IF NOT EXISTS idx_download_history_created_at ON download_history(created_at DESC);

-- 复合索引: 用户历史查询
CREATE INDEX IF NOT EXISTS idx_download_history_user_status ON download_history(user_id, status, created_at DESC);

-- 部分索引: 清理任务查询
CREATE INDEX IF NOT EXISTS idx_download_history_cleanup ON download_history(status, expire_at) WHERE status = 4;

-- 统计查询索引
CREATE INDEX IF NOT EXISTS idx_download_history_stats ON download_history(created_at, status);

-- 补充索引 (来自 asset-service 的优化)
CREATE INDEX IF NOT EXISTS idx_history_user_platform ON download_history(user_id, platform);
