# Downloader Service - 详细设计PRD (核心服务)

| 文档属性 | 详细信息 |
|:---|:---:|
| **服务名称** | Downloader Service (下载调度服务) ⭐核心服务 |
| **版本** | V1.0.0 |
| **端口** | 9003 (gRPC, 内部使用) |
| **依赖服务** | PostgreSQL, RabbitMQ, Redis, 第三方代理API |

---

## 1. 服务概述

### 1.1 职责定义
Downloader Service是系统的核心服务,负责:
- **任务调度**: 从RabbitMQ消费下载任务
- **代理管理**: 调用第三方API获取代理IP
- **视频下载**: 执行yt-dlp下载视频文件
- **文件管理**: 本地磁盘文件存储和生命周期管理
- **进度推送**: 通过Redis PubSub实时推送下载进度
- **定时清理**: 清理过期的临时文件

### 1.2 核心特性
- Worker池并发下载
- 动态代理IP集成
- 双模式下载(极速/归档)
- 文件自动清理机制
- 实时进度推送
- 下载任务重试

---

## 2. 架构设计

### 2.1 服务架构图

```mermaid
graph TB
    subgraph "Downloader Service"
        subgraph "Worker池 Worker Pool"
            Worker1[Worker 1]
            Worker2[Worker 2]
            Worker3[Worker N]
        end
        
        subgraph "任务管理 Task Manager"
            Consumer[MQ消费者]
            Dispatcher[任务分发器]
            Retry[重试管理器]
        end
        
        subgraph "下载引擎 Download Engine"
            ProxyManager[代理管理器]
            YTDLPExecutor[yt-dlp执行器]
            ProgressTracker[进度跟踪器]
        end
        
        subgraph "文件管理 File Manager"
            FileWriter[文件写入器]
            PathGenerator[路径生成器]
            MD5Calculator[MD5计算器]
        end
        
        subgraph "清理任务 Cleanup"
            CleanupScheduler[定时调度器]
            FileScanner[文件扫描器]
            FileCleaner[文件清理器]
        end
    end
    
    subgraph "外部依赖"
        MQ[(RabbitMQ<br/>任务队列)]
        DB[(PostgreSQL<br/>下载历史)]
        Redis[(Redis<br/>进度PubSub)]
        ProxyAPI[第三方代理API]
        Storage[本地磁盘<br/>/data/vasset/]
    end
    
    MQ --> Consumer
    Consumer --> Dispatcher
    Dispatcher --> Worker1
    Dispatcher --> Worker2
    Dispatcher --> Worker3
    
    Worker1 --> ProxyManager
    Worker2 --> ProxyManager
    Worker3 --> ProxyManager
    
    ProxyManager --> ProxyAPI
    Worker1 --> YTDLPExecutor
    YTDLPExecutor --> ProgressTracker
    ProgressTracker --> Redis
    
    YTDLPExecutor --> FileWriter
    FileWriter --> PathGenerator
    FileWriter --> Storage
    FileWriter --> MD5Calculator
    FileWriter --> DB
    
    Worker1 -.->|失败| Retry
    Retry -.->|重新入队| MQ
    
    CleanupScheduler --> FileScanner
    FileScanner --> DB
    FileScanner --> FileCleaner
    FileCleaner --> Storage
    FileCleaner --> DB
```

### 2.2 任务执行流程

```mermaid
sequenceDiagram
    participant MQ as RabbitMQ
    participant Worker as Worker
    participant Proxy as 代理管理器
    participant API as 第三方代理API
    participant YTDLP as yt-dlp
    participant Redis as Redis
    participant Storage as 本地磁盘
    participant DB as PostgreSQL
    
    MQ->>Worker: 消费任务
    Worker->>DB: UPDATE status=processing
    
    Worker->>Proxy: 获取代理IP
    Proxy->>API: HTTP: GetProxy()
    API-->>Proxy: http://user:pass@ip:port
    
    Worker->>Worker: 生成文件路径
    Worker->>YTDLP: 执行下载命令
    
    loop 下载中
        YTDLP->>Redis: PUBLISH progress:task_id
        YTDLP->>Storage: 写入文件块
    end
    
    YTDLP-->>Worker: 下载完成
    
    Worker->>Storage: 计算MD5
    Worker->>DB: UPDATE file_path, file_hash, status=completed
    Worker->>Redis: PUBLISH progress:task_id (100%)
    Worker->>MQ: ACK消息
```

---

## 3. 功能模块

### 3.1 任务消费

#### MQ消费者
```go
type TaskConsumer struct {
    conn    *amqp.Connection
    channel *amqp.Channel
    workers *WorkerPool
}

func (c *TaskConsumer) Start(ctx context.Context) error {
    msgs, err := c.channel.Consume(
        "vasset.download.queue", // 队列名
        "",                      // 消费者名
        false,                   // 手动ACK
        false,                   // 非独占
        false,                   // no-local
        false,                   // no-wait
        nil,                     // args
    )
    
    for {
        select {
        case msg := <-msgs:
            task := parseTask(msg.Body)
            c.workers.Submit(task, func(err error) {
                if err != nil {
                    msg.Nack(false, true) // 重新入队
                } else {
                    msg.Ack(false)
                }
            })
        case <-ctx.Done():
            return ctx.Err()
        }
    }
}
```

#### 任务数据结构
```go
type DownloadTask struct {
    TaskID   string    `json:"task_id"`
    UserID   int64     `json:"user_id"`
    URL      string    `json:"url"`
    Mode     string    `json:"mode"`      // quick_download, archive
    Quality  string    `json:"quality"`   // 1080p, 720p, etc.
    Format   string    `json:"format"`    // mp4, webm
    Metadata Metadata  `json:"metadata"`
}

type Metadata struct {
    Title    string `json:"title"`
    Duration int64  `json:"duration"`
    Platform string `json:"platform"`
}
```

### 3.2 Worker池管理

#### Worker池实现
```go
type WorkerPool struct {
    size      int
    tasks     chan *DownloadTask
    semaphore chan struct{}
}

func NewWorkerPool(size int) *WorkerPool {
    pool := &WorkerPool{
        size:      size,
        tasks:     make(chan *DownloadTask, size*2),
        semaphore: make(chan struct{}, size),
    }
    
    // 启动workers
    for i := 0; i < size; i++ {
        go pool.worker(i)
    }
    
    return pool
}

func (p *WorkerPool) worker(id int) {
    for task := range p.tasks {
        p.semaphore <- struct{}{}  // 获取信号量
        
        logger.Info("worker processing task",
            zap.Int("worker_id", id),
            zap.String("task_id", task.TaskID),
        )
        
        if err := p.processTask(task); err != nil {
            logger.Error("task failed", zap.Error(err))
        }
        
        <-p.semaphore  // 释放信号量
    }
}

func (p *WorkerPool) Submit(task *DownloadTask, callback func(error)) {
    go func() {
        p.tasks <- task
        // 等待完成并回调
    }()
}
```

### 3.3 代理IP管理

#### 第三方API集成
```go
type ProxyProvider struct {
    apiKey   string
    endpoint string
    client   *http.Client
}

type ProxyResponse struct {
    IP       string `json:"ip"`
    Port     int    `json:"port"`
    Username string `json:"username"`
    Password string `json:"password"`
    ExpireAt string `json:"expire_at"`
}

func (p *ProxyProvider) GetProxy(ctx context.Context) (string, error) {
    req, _ := http.NewRequestWithContext(ctx, "GET", p.endpoint, nil)
    req.Header.Set("Authorization", "Bearer "+p.apiKey)
    
    resp, err := p.client.Do(req)
    if err != nil {
        return "", fmt.Errorf("proxy API request failed: %w", err)
    }
    defer resp.Body.Close()
    
    if resp.StatusCode != 200 {
        return "", fmt.Errorf("proxy API returned %d", resp.StatusCode)
    }
    
    var proxyResp ProxyResponse
    if err := json.NewDecoder(resp.Body).Decode(&proxyResp); err != nil {
        return "", err
    }
    
    // 格式化代理URL
    proxyURL := fmt.Sprintf("http://%s:%s@%s:%d",
        proxyResp.Username,
        proxyResp.Password,
        proxyResp.IP,
        proxyResp.Port,
    )
    
    logger.Info("got proxy IP", zap.String("ip", proxyResp.IP))
    return proxyURL, nil
}
```

#### 代理重试策略
```go
func (w *Worker) getProxyWithRetry(ctx context.Context) (string, error) {
    var lastErr error
    
    for i := 0; i < 3; i++ {
        proxy, err := w.proxyProvider.GetProxy(ctx)
        if err == nil {
            return proxy, nil
        }
        
        lastErr = err
        logger.Warn("failed to get proxy, retrying",
            zap.Int("attempt", i+1),
            zap.Error(err),
        )
        
        time.Sleep(time.Duration(math.Pow(2, float64(i))) * time.Second)
    }
    
    return "", fmt.Errorf("failed to get proxy after 3 attempts: %w", lastErr)
}
```

### 3.4 yt-dlp下载执行

#### 下载命令构建
```go
func (e *YTDLPExecutor) buildCommand(task *DownloadTask, proxyURL, outputPath string) *exec.Cmd {
    args := []string{
        "--proxy", proxyURL,
        "--output", outputPath,
        "--format", task.Quality,
        "--merge-output-format", task.Format,
        "--progress",                           // 输出进度
        "--newline",                           // 每次进度新行
        "--no-playlist",
        "--concurrent-fragments", "3",         // 并发分片
        task.URL,
    }
    
    cmd := exec.Command("yt-dlp", args...)
    return cmd
}
```

#### 进度解析
```go
func (e *YTDLPExecutor) executeWithProgress(cmd *exec.Cmd, taskID string) error {
    stdoutPipe, _ := cmd.StdoutPipe()
    stderrPipe, _ := cmd.StderrPipe()
    
    if err := cmd.Start(); err != nil {
        return err
    }
    
    // 解析进度输出
    scanner := bufio.NewScanner(stdoutPipe)
    for scanner.Scan() {
        line := scanner.Text()
        progress := parseProgress(line)
        
        if progress != nil {
            e.publishProgress(taskID, progress)
        }
    }
    
    return cmd.Wait()
}

func parseProgress(line string) *Progress {
    // yt-dlp输出格式: [download]  45.2% of 100.00MiB at 2.50MiB/s ETA 00:22
    re := regexp.MustCompile(`\[download\]\s+(\d+\.?\d*)%.*?(\d+\.?\d*\w+/s)?.*?ETA\s+(\d+:\d+)`)
    matches := re.FindStringSubmatch(line)
    
    if len(matches) < 2 {
        return nil
    }
    
    percent, _ := strconv.ParseFloat(matches[1], 64)
    
    return &Progress{
        Percent: percent,
        Speed:   matches[2],
        ETA:     matches[3],
    }
}
```

### 3.5 文件存储管理

#### 路径生成策略
```go
type PathGenerator struct {
    basePath string
}

func (g *PathGenerator) GeneratePath(task *DownloadTask) (string, error) {
    // 清理文件名
    safeTitle := sanitizeFilename(task.Metadata.Title)
    
    var filePath string
    
    switch task.Mode {
    case "quick_download":
        // 临时文件: /data/vasset/tmp/{task_id}/video.mp4
        filePath = filepath.Join(
            g.basePath, "tmp", task.TaskID,
            fmt.Sprintf("%s.%s", safeTitle, task.Format),
        )
        
    case "archive":
        // 归档文件: /data/vasset/archive/{user_id}/{YYYYMMDD}/video_{timestamp}.mp4
        date := time.Now().Format("20060102")
        timestamp := time.Now().Unix()
        
        filePath = filepath.Join(
            g.basePath, "archive",
            fmt.Sprintf("%d", task.UserID),
            date,
            fmt.Sprintf("%s_%d.%s", safeTitle, timestamp, task.Format),
        )
    }
    
    // 确保目录存在
    dir := filepath.Dir(filePath)
    if err := os.MkdirAll(dir, 0755); err != nil {
        return "", err
    }
    
    return filePath, nil
}

func sanitizeFilename(name string) string {
    // 移除非法字符
    reg := regexp.MustCompile(`[<>:"/\\|?*]`)
    clean := reg.ReplaceAllString(name, "_")
    
    // 限制长度
    if len(clean) > 200 {
        clean = clean[:200]
    }
    
    return clean
}
```

#### MD5计算
```go
func calculateMD5(filePath string) (string, error) {
    file, err := os.Open(filePath)
    if err != nil {
        return "", err
    }
    defer file.Close()
    
    hash := md5.New()
    if _, err := io.Copy(hash, file); err != nil {
        return "", err
    }
    
    return fmt.Sprintf("%x", hash.Sum(nil)), nil
}
```

### 3.6 进度推送

#### Redis PubSub发布
```go
type ProgressPublisher struct {
    redis *redis.Client
}

type ProgressMessage struct {
    TaskID          string  `json:"task_id"`
    Status          string  `json:"status"`           // downloading, completed, failed
    Percent         float64 `json:"percent"`
    DownloadedBytes int64   `json:"downloaded_bytes"`
    TotalBytes      int64   `json:"total_bytes"`
    Speed           string  `json:"speed"`
    ETA             string  `json:"eta"`
    Message         string  `json:"message"`
}

func (p *ProgressPublisher) Publish(msg *ProgressMessage) error {
    channel := fmt.Sprintf("progress:%s", msg.TaskID)
    
    data, err := json.Marshal(msg)
    if err != nil {
        return err
    }
    
    return p.redis.Publish(context.Background(), channel, data).Err()
}
```

### 3.7 定时清理任务

#### 清理调度器
```go
type CleanupScheduler struct {
    db       *gorm.DB
    interval time.Duration
}

func (s *CleanupScheduler) Start(ctx context.Context) {
    ticker := time.NewTicker(s.interval)
    defer ticker.Stop()
    
    for {
        select {
        case <-ticker.C:
            s.cleanup()
        case <-ctx.Done():
            return
        }
    }
}

func (s *CleanupScheduler) cleanup() {
    logger.Info("starting file cleanup task")
    
    // 查询需要清理的记录
    var records []DownloadHistory
    s.db.Where("status = ? AND expire_at < ?", StatusPendingCleanup, time.Now()).
        Find(&records)
    
    logger.Info("found expired files", zap.Int("count", len(records)))
    
    for _, record := range records {
        // 删除物理文件
        if err := os.Remove(record.FilePath); err != nil {
            logger.Error("failed to delete file",
                zap.String("path", record.FilePath),
                zap.Error(err),
            )
            continue
        }
        
        // 更新数据库状态
        s.db.Model(&record).Updates(map[string]interface{}{
            "status":     StatusExpired,
            "updated_at": time.Now(),
        })
        
        logger.Info("cleaned up file",
            zap.Int64("history_id", record.ID),
            zap.String("path", record.FilePath),
        )
    }
}
```

---

## 4. 数据库设计

### 4.1 Download_History表

```sql
CREATE TABLE download_history (
    id            BIGSERIAL PRIMARY KEY,
    task_id       VARCHAR(64) NOT NULL UNIQUE,
    user_id       BIGINT NOT NULL,
    url           TEXT NOT NULL,
    platform      VARCHAR(50),
    title         VARCHAR(500),
    mode          VARCHAR(20) NOT NULL,     -- quick_download, archive
    quality       VARCHAR(20),              -- 1080p, 720p, etc.
    file_path     VARCHAR(1000),            -- 本地文件路径
    file_name     VARCHAR(500),
    file_size     BIGINT,                   -- 字节
    file_hash     CHAR(32),                 -- MD5
    status        INT NOT NULL DEFAULT 0,   -- 状态机(见下方)
    error_message TEXT,                     -- 错误信息
    retry_count   INT DEFAULT 0,
    expire_at     TIMESTAMP,                -- 清理时间(仅quick_download)
    created_at    TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    started_at    TIMESTAMP,
    completed_at  TIMESTAMP,
    
    FOREIGN KEY (user_id) REFERENCES users(id),
    INDEX idx_task_id (task_id),
    INDEX idx_user_id (user_id),
    INDEX idx_status (status),
    INDEX idx_expire_at (expire_at),
    INDEX idx_created_at (created_at)
);
```

#### 状态机设计
```go
const (
    StatusPending         = 0  // 待处理
    StatusProcessing      = 1  // 下载中
    StatusCompleted       = 2  // 完成
    StatusFailed          = 3  // 失败
    StatusPendingCleanup  = 4  // 待清理(quick_download完成后)
    StatusExpired         = 5  // 已过期(已清理)
)
```

#### 状态转换
```mermaid
stateDiagram-v2
    [*] --> Pending: 任务创建
    Pending --> Processing: Worker开始处理
    Processing --> Completed: 下载成功
    Processing --> Failed: 下载失败
    Failed --> Pending: 重试
    Completed --> PendingCleanup: quick_download模式
    Completed --> [*]: archive模式
    PendingCleanup --> Expired: 定时清理
    Expired --> [*]
```

### 4.2 索引优化

```sql
-- 任务查询索引
CREATE INDEX idx_download_history_task_id ON download_history(task_id);

-- 用户历史查询索引
CREATE INDEX idx_download_history_user_status ON download_history(user_id, status, created_at DESC);

-- 清理任务查询索引
CREATE INDEX idx_download_history_cleanup ON download_history(status, expire_at) WHERE status = 4;

-- 统计查询索引
CREATE INDEX idx_download_history_stats ON download_history(created_at, status);
```

---

## 5. 配置管理

### 5.1 配置文件 (config/dev.yaml)
```yaml
server:
  port: 9003
  
database:
  host: localhost
  port: 5432
  user: vasset
  password: password
  dbname: vasset
  max_open_conns: 50

rabbitmq:
  url: "amqp://guest:guest@localhost:5672/"
  queue: "vasset.download.queue"
  prefetch_count: 5             # 预取消息数
  
redis:
  addr: localhost:6379
  password: ""
  db: 0

worker:
  pool_size: 10                 # Worker池大小
  max_concurrent: 10            # 最大并发下载数
  
ytdlp:
  binary_path: "/usr/local/bin/yt-dlp"
  timeout: 1800                 # 下载超时30分钟
  concurrent_fragments: 3
  
proxy:
  provider: "your-proxy-provider"
  api_endpoint: "https://api.proxy-provider.com/get"
  api_key: "your-api-key"
  timeout: 5                    # 获取代理超时5秒
  retry_count: 3
  
storage:
  base_path: "/data/vasset"
  tmp_ttl: 86400                # 临时文件TTL 24小时
  
cleanup:
  enabled: true
  interval: 3600                # 每小时执行一次清理
  batch_size: 100               # 每次清理100个文件
  
retry:
  max_attempts: 3
  initial_interval: 60          # 首次重试等待60秒
  max_interval: 3600            # 最大重试间隔1小时
```

---

## 6. 非功能性需求

### 6.1 性能指标
- **并发下载数**: 10个任务
- **单任务下载速度**: 受限于代理IP带宽
- **平均任务完成时间**: 取决于文件大小和网络情况
- **MD5计算性能**: 100MB文件 < 2秒

### 6.2 可靠性

#### 任务重试机制
```go
type RetryPolicy struct {
    maxAttempts     int
    initialInterval time.Duration
    maxInterval     time.Duration
}

func (p *RetryPolicy) ShouldRetry(task *DownloadTask) bool {
    return task.RetryCount < p.maxAttempts
}

func (p *RetryPolicy) GetRetryDelay(attempt int) time.Duration {
    delay := p.initialInterval * time.Duration(math.Pow(2, float64(attempt)))
    if delay > p.maxInterval {
        delay = p.maxInterval
    }
    return delay
}
```

#### 错误处理
```go
var (
    ErrProxyUnavailable   = errors.New("proxy unavailable")
    ErrDownloadTimeout    = errors.New("download timeout")
    ErrVideoNotFound      = errors.New("video not found")
    ErrInsufficientSpace  = errors.New("insufficient disk space")
)

func (w *Worker) handleError(task *DownloadTask, err error) {
    switch {
    case errors.Is(err, ErrProxyUnavailable):
        // 代理问题,可重试
        w.retryTask(task)
        
    case errors.Is(err, ErrVideoNotFound):
        // 视频不存在,不重试
        w.markFailed(task, err.Error())
        
    case errors.Is(err, ErrInsufficientSpace):
        // 磁盘空间不足,停止Worker
        logger.Fatal("insufficient disk space")
        
    default:
        if w.retryPolicy.ShouldRetry(task) {
            w.retryTask(task)
        } else {
            w.markFailed(task, err.Error())
        }
    }
}
```

### 6.3 磁盘管理

#### 磁盘空间检查
```go
func checkDiskSpace(path string) error {
    var stat syscall.Statfs_t
    if err := syscall.Statfs(path, &stat); err != nil {
        return err
    }
    
    // 可用空间(字节)
    available := stat.Bavail * uint64(stat.Bsize)
    total := stat.Blocks * uint64(stat.Bsize)
    usagePercent := float64(total-available) / float64(total) * 100
    
    logger.Info("disk usage",
        zap.Float64("usage_percent", usagePercent),
        zap.Uint64("available_gb", available/1024/1024/1024),
    )
    
    if usagePercent > 90 {
        return ErrInsufficientSpace
    }
    
    return nil
}
```

---

## 7. 监控与告警

### 7.1 监控指标
```yaml
metrics:
  business:
    - downloader_tasks_total (按状态分类)
    - downloader_tasks_in_progress
    - downloader_tasks_completed_total
    - downloader_tasks_failed_total
    - downloader_retry_total
    - downloader_cleanup_deleted_files_total
    
  performance:
    - downloader_task_duration_seconds (P50/P95/P99)
    - downloader_download_speed_mbps
    - downloader_file_size_bytes (直方图)
    - downloader_md5_duration_seconds
    
  system:
    - downloader_worker_pool_active
    - downloader_disk_usage_percent
    - downloader_disk_available_gb
```

### 7.2 告警规则
| 告警项 | 阈值 | 级别 |
|:---|:---|:---|
| 下载失败率 | > 30% | P1 |
| 队列堆积 | > 1000 | P2 |
| Worker池繁忙率 | > 90% | P3 |
| 磁盘使用率 | > 85% | P1 |
| 代理API失败率 | > 20% | P2 |
| 平均下载时间 | > 10分钟 | P3 |

---

## 8. 部署与运维

### 8.1 环境要求
- **操作系统**: Linux (Ubuntu 20.04+)
- **yt-dlp**: 2023.11+
- **ffmpeg**: 4.4+ (视频合并需要)
- **磁盘**: SSD,至少500GB

### 8.2 yt-dlp安装
```bash
# 安装yt-dlp
curl -L https://github.com/yt-dlp/yt-dlp/releases/latest/download/yt-dlp -o /usr/local/bin/yt-dlp
chmod +x /usr/local/bin/yt-dlp

# 安装ffmpeg
apt-get install -y ffmpeg

# 验证安装
yt-dlp --version
ffmpeg -version
```

### 8.3 日志格式
```json
{
  "timestamp": "2025-12-03T10:30:00Z",
  "level": "info",
  "service": "downloader",
  "worker_id": 3,
  "task_id": "uuid-xxx",
  "user_id": 1001,
  "url": "https://youtube.com/watch?v=xxx",
  "status": "completed",
  "duration_seconds": 125,
  "file_size_mb": 98.5
}
```

---

**关键提示**: Downloader Service是系统的核心,其稳定性直接影响用户体验。建议在生产环境部署多个Worker实例,并配置充足的磁盘空间监控。
