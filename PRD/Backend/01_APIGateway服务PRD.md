# API Gateway服务 - 详细设计PRD

| 文档属性 | 详细信息 |
|:---|:---:|
| **服务名称** | API Gateway Service |
| **版本** | V1.0.0 |
| **端口** | 8080 (HTTP), 8081 (Health Check) |
| **依赖服务** | Auth, Parser, Asset, RabbitMQ, Redis |

---

## 1. 服务概述

### 1.1 职责定义
API Gateway作为系统的统一接入层,负责:
- **路由转发**: 将HTTP请求路由到对应的后端微服务
- **认证鉴权**: 拦截所有请求,验证JWT Token
- **协议转换**: HTTP/JSON → gRPC/Protobuf
- **实时推送**: 管理WebSocket连接,推送下载进度
- **文件代理**: 下载文件流式传输
- **限流熔断**: 保护后端服务不被压垮

### 1.2 核心特性
- 统一的RESTful API入口
- JWT Token验证中间件
- WebSocket长连接管理
- 跨域CORS支持
- 请求日志与追踪

---

## 2. 架构设计

### 2.1 服务架构图

```mermaid
graph TB
    subgraph "API Gateway Service"
        subgraph "HTTP Server (Gin)"
            Router[路由器<br/>gin.Engine]
            
            subgraph "中间件链 Middleware Chain"
                CORS[CORS中间件]
                Logger[日志中间件]
                Recovery[恢复中间件]
                Auth[JWT认证中间件]
                RateLimit[限流中间件]
            end
            
            subgraph "处理器层 Handlers"
                AuthHandler[认证处理器]
                ParseHandler[解析处理器]
                DownloadHandler[下载处理器]
                HistoryHandler[历史处理器]
                FileHandler[文件下载处理器]
            end
        end
        
        subgraph "WebSocket Manager"
            WSServer[WebSocket服务器]
            ConnPool[连接池]
            SubManager[Redis订阅管理器]
        end
        
        subgraph "gRPC Client Pool"
            AuthClient[Auth Service Client]
            ParserClient[Parser Service Client]
            AssetClient[Asset Service Client]
        end
        
        subgraph "MQ Publisher"
            RabbitMQPub[RabbitMQ发布器]
        end
    end
    
    Router --> CORS
    CORS --> Logger
    Logger --> Recovery
    Recovery --> Auth
    Auth --> RateLimit
    RateLimit --> AuthHandler
    RateLimit --> ParseHandler
    RateLimit --> DownloadHandler
    RateLimit --> HistoryHandler
    RateLimit --> FileHandler
    
    AuthHandler --> AuthClient
    ParseHandler --> ParserClient
    DownloadHandler --> RabbitMQPub
    HistoryHandler --> AssetClient
    FileHandler --> AssetClient
    
    WSServer --> SubManager
    SubManager -.->|Subscribe| RedisPS[(Redis PubSub)]
    RabbitMQPub --> MQ[(RabbitMQ)]
```

### 2.2 请求处理流程

```mermaid
sequenceDiagram
    participant Client as 客户端
    participant Router as Gin路由器
    participant MW as 中间件链
    participant Handler as 业务处理器
    participant gRPC as gRPC客户端
    participant Backend as 后端服务
    
    Client->>Router: HTTP请求
    Router->>MW: CORS中间件
    MW->>MW: 日志中间件
    MW->>MW: 恢复中间件
    
    alt 需要认证的接口
        MW->>MW: JWT认证中间件
        alt Token无效
            MW-->>Client: 401 Unauthorized
        end
    end
    
    MW->>MW: 限流中间件
    alt 超过限流
        MW-->>Client: 429 Too Many Requests
    end
    
    MW->>Handler: 转发到处理器
    Handler->>gRPC: gRPC调用
    gRPC->>Backend: 请求后端服务
    Backend-->>gRPC: 响应
    gRPC-->>Handler: 返回结果
    Handler-->>Router: HTTP响应
    Router-->>Client: JSON响应
```

---

## 3. 功能模块

### 3.1 路由管理

#### 路由表设计
```go
// 公开接口(无需认证)
公开路由组 /api/v1
├── POST   /auth/register        # 用户注册
├── POST   /auth/login           # 用户登录
├── GET    /health               # 健康检查
└── GET    /version              # 版本信息

// 认证接口(需要JWT Token)
认证路由组 /api/v1 [JWT中间件]
├── GET    /auth/profile         # 获取用户信息
├── POST   /auth/logout          # 登出
├── POST   /parse                # 解析URL
├── POST   /download             # 提交下载任务
├── GET    /user/history         # 下载历史
├── GET    /user/quota           # 配额信息
├── GET    /download/file        # 下载文件
└── WS     /ws/progress          # WebSocket进度推送
```

#### 路由配置代码示例
```go
func SetupRoutes(r *gin.Engine, deps *Dependencies) {
    // 公开路由
    public := r.Group("/api/v1")
    {
        public.POST("/auth/register", handlers.Register(deps.AuthClient))
        public.POST("/auth/login", handlers.Login(deps.AuthClient))
        public.GET("/health", handlers.HealthCheck)
    }
    
    // 认证路由
    protected := r.Group("/api/v1")
    protected.Use(middleware.JWTAuth(deps.AuthClient))
    {
        protected.GET("/auth/profile", handlers.GetProfile(deps.AuthClient))
        protected.POST("/parse", handlers.ParseURL(deps.ParserClient))
        protected.POST("/download", handlers.SubmitDownload(deps.MQPublisher))
        protected.GET("/user/history", handlers.GetHistory(deps.AssetClient))
        protected.GET("/download/file", handlers.DownloadFile(deps.AssetClient))
    }
    
    // WebSocket路由
    r.GET("/api/v1/ws/progress", handlers.WebSocketProgress(deps.RedisClient))
}
```

### 3.2 认证鉴权中间件

#### JWT验证流程
```mermaid
flowchart TD
    Start[请求到达] --> ExtractToken{提取Token}
    ExtractToken -->|无Token| Return401[返回401]
    ExtractToken -->|有Token| CheckCache{Redis缓存检查}
    
    CheckCache -->|缓存命中| ValidateExp{检查过期}
    CheckCache -->|缓存未命中| CallAuth[调用Auth Service]
    
    CallAuth -->|验证成功| CacheToken[写入Redis缓存]
    CallAuth -->|验证失败| Return401
    
    CacheToken --> SetContext[设置用户上下文]
    ValidateExp -->|已过期| Return401
    ValidateExp -->|未过期| SetContext
    
    SetContext --> Next[继续处理]
    Return401 --> End[结束]
    Next --> End
```

#### 中间件代码实现
```go
func JWTAuth(authClient pb.AuthServiceClient) gin.HandlerFunc {
    return func(c *gin.Context) {
        // 1. 提取Token
        authHeader := c.GetHeader("Authorization")
        if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
            c.JSON(401, gin.H{"error": "missing or invalid token"})
            c.Abort()
            return
        }
        
        token := strings.TrimPrefix(authHeader, "Bearer ")
        
        // 2. 检查Redis缓存
        cacheKey := fmt.Sprintf("auth:token:%s", hashToken(token))
        userID, err := redisClient.Get(ctx, cacheKey).Int64()
        
        if err == nil {
            // 缓存命中
            c.Set("user_id", userID)
            c.Next()
            return
        }
        
        // 3. 调用Auth Service验证
        resp, err := authClient.VerifyToken(ctx, &pb.VerifyTokenRequest{
            Token: token,
        })
        
        if err != nil || !resp.Valid {
            c.JSON(401, gin.H{"error": "invalid token"})
            c.Abort()
            return
        }
        
        // 4. 写入缓存(TTL=5分钟)
        redisClient.Set(ctx, cacheKey, resp.UserId, 5*time.Minute)
        
        // 5. 设置上下文
        c.Set("user_id", resp.UserId)
        c.Set("user_email", resp.Email)
        c.Next()
    }
}
```

### 3.3 限流与熔断

#### 限流策略
```go
// 全局限流: 每秒最多200个请求
globalLimiter := rate.NewLimiter(200, 400)

// 用户限流: 每个用户每秒最多10个请求
userLimiters := sync.Map{}

func RateLimitMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        // 全局限流
        if !globalLimiter.Allow() {
            c.JSON(429, gin.H{"error": "global rate limit exceeded"})
            c.Abort()
            return
        }
        
        // 用户限流
        userID, exists := c.Get("user_id")
        if exists {
            limiter := getUserLimiter(userID.(int64))
            if !limiter.Allow() {
                c.JSON(429, gin.H{"error": "user rate limit exceeded"})
                c.Abort()
                return
            }
        }
        
        c.Next()
    }
}
```

#### 熔断配置
```go
// 使用hystrix-go实现熔断
hystrix.ConfigureCommand("auth_service", hystrix.CommandConfig{
    Timeout:                5000,  // 超时5秒
    MaxConcurrentRequests:  100,   // 最大并发100
    ErrorPercentThreshold:  50,    // 错误率>50%触发熔断
    RequestVolumeThreshold: 20,    // 至少20个请求才计算错误率
    SleepWindow:            30000, // 熔断30秒后尝试恢复
})
```

### 3.4 WebSocket管理

#### 连接管理
```go
type WebSocketManager struct {
    connections sync.Map // map[taskID]*websocket.Conn
    rdb         *redis.Client
}

func (m *WebSocketManager) HandleConnection(c *gin.Context) {
    // 升级HTTP到WebSocket
    conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
    if err != nil {
        return
    }
    defer conn.Close()
    
    // 获取任务ID
    taskID := c.Query("task_id")
    m.connections.Store(taskID, conn)
    
    // 订阅Redis频道
    pubsub := m.rdb.Subscribe(ctx, fmt.Sprintf("progress:%s", taskID))
    defer pubsub.Close()
    
    // 监听进度消息
    ch := pubsub.Channel()
    for msg := range ch {
        conn.WriteJSON(msg.Payload)
    }
}
```

### 3.5 文件下载代理

#### 文件下载流程
```mermaid
sequenceDiagram
    participant Client as 客户端
    participant GW as API Gateway
    participant Asset as Asset Service
    participant Disk as 本地磁盘
    
    Client->>GW: GET /api/v1/download/file?history_id=123
    GW->>GW: 验证JWT Token
    GW->>Asset: gRPC: GetFileInfo(history_id)
    Asset->>Asset: 检查权限(user_id匹配)
    Asset-->>GW: {file_path, file_name, file_size}
    
    alt 权限不足
        GW-->>Client: 403 Forbidden
    end
    
    GW->>Disk: os.Open(file_path)
    Disk-->>GW: File Stream
    GW->>Client: HTTP响应头<br/>Content-Disposition: attachment<br/>Content-Length: xxx
    GW->>Client: 流式传输文件内容
```

#### 文件下载代码实现
```go
func DownloadFile(assetClient pb.AssetServiceClient) gin.HandlerFunc {
    return func(c *gin.Context) {
        historyID := c.Query("history_id")
        userID := c.GetInt64("user_id")
        
        // 1. 调用Asset Service获取文件信息
        resp, err := assetClient.GetFileInfo(ctx, &pb.GetFileInfoRequest{
            HistoryId: historyID,
            UserId:    userID,
        })
        
        if err != nil {
            c.JSON(403, gin.H{"error": "permission denied"})
            return
        }
        
        // 2. 打开本地文件
        file, err := os.Open(resp.FilePath)
        if err != nil {
            c.JSON(404, gin.H{"error": "file not found"})
            return
        }
        defer file.Close()
        
        // 3. 设置响应头
        c.Header("Content-Description", "File Transfer")
        c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", resp.FileName))
        c.Header("Content-Type", "application/octet-stream")
        c.Header("Content-Length", fmt.Sprintf("%d", resp.FileSize))
        
        // 4. 流式传输
        io.Copy(c.Writer, file)
    }
}
```

---

## 4. 接口设计

### 4.1 RESTful API定义

#### A. 用户认证接口

**用户注册**
```http
POST /api/v1/auth/register
Content-Type: application/json

Request Body:
{
  "email": "user@example.com",
  "password": "SecurePass123",
  "nickname": "张三"
}

Response: 201 Created
{
  "code": 0,
  "message": "success",
  "data": {
    "user_id": 1001,
    "email": "user@example.com",
    "nickname": "张三"
  }
}
```

**用户登录**
```http
POST /api/v1/auth/login
Content-Type: application/json

Request Body:
{
  "email": "user@example.com",
  "password": "SecurePass123"
}

Response: 200 OK
{
  "code": 0,
  "message": "success",
  "data": {
    "token": "eyJhbGciOiJIUzI1NiIs...",
    "refresh_token": "refresh_token_xxx",
    "expires_in": 86400,
    "user": {
      "user_id": 1001,
      "email": "user@example.com",
      "nickname": "张三"
    }
  }
}
```

#### B. URL解析接口

**解析视频URL**
```http
POST /api/v1/parse
Content-Type: application/json
Authorization: Bearer <token>

Request Body:
{
  "url": "https://www.youtube.com/watch?v=dQw4w9WgXcQ"
}

Response: 200 OK
{
  "code": 0,
  "message": "success",
  "data": {
    "video_id": "dQw4w9WgXcQ",
    "platform": "youtube",
    "title": "Rick Astley - Never Gonna Give You Up",
    "duration": 213,
    "thumbnail": "https://i.ytimg.com/vi/dQw4w9WgXcQ/maxresdefault.jpg",
    "author": "Rick Astley",
    "formats": [
      {
        "format_id": "137",
        "quality": "1080p",
        "ext": "mp4",
        "filesize": 104857600
      },
      {
        "format_id": "136",
        "quality": "720p",
        "ext": "mp4",
        "filesize": 52428800
      }
    ]
  }
}
```

#### C. 下载任务接口

**提交下载任务**
```http
POST /api/v1/download
Content-Type: application/json
Authorization: Bearer <token>

Request Body:
{
  "url": "https://www.youtube.com/watch?v=dQw4w9WgXcQ",
  "mode": "quick_download",  // 或 "archive"
  "quality": "1080p",
  "format": "mp4"
}

Response: 202 Accepted
{
  "code": 0,
  "message": "task submitted",
  "data": {
    "task_id": "550e8400-e29b-41d4-a716-446655440000",
    "history_id": 5001,
    "estimated_time": 120  // 预计耗时(秒)
  }
}
```

#### D. 下载历史接口

**获取下载历史**
```http
GET /api/v1/user/history?page=1&page_size=20&status=completed
Authorization: Bearer <token>

Response: 200 OK
{
  "code": 0,
  "message": "success",
  "data": {
    "total": 156,
    "page": 1,
    "page_size": 20,
    "items": [
      {
        "history_id": 5001,
        "url": "https://www.youtube.com/watch?v=xxx",
        "title": "视频标题",
        "file_size": 104857600,
        "status": "completed",
        "mode": "archive",
        "created_at": "2025-12-03T10:30:00Z",
        "completed_at": "2025-12-03T10:32:15Z"
      }
    ]
  }
}
```

#### E. 文件下载接口

**下载文件**
```http
GET /api/v1/download/file?history_id=5001
Authorization: Bearer <token>

Response: 200 OK
Content-Type: application/octet-stream
Content-Disposition: attachment; filename="video_title.mp4"
Content-Length: 104857600

<文件二进制流>
```

### 4.2 WebSocket协议

**连接建立**
```javascript
const ws = new WebSocket('wss://api.vasset.com/api/v1/ws/progress?task_id=xxx&token=xxx');

ws.onmessage = (event) => {
  const progress = JSON.parse(event.data);
  console.log(progress);
};
```

**进度消息格式**
```json
{
  "task_id": "550e8400-e29b-41d4-a716-446655440000",
  "status": "downloading",
  "percent": 65.5,
  "downloaded_bytes": 68681728,
  "total_bytes": 104857600,
  "speed": "2.5 MB/s",
  "eta": "15s",
  "message": "正在下载..."
}
```

**完成消息**
```json
{
  "task_id": "550e8400-e29b-41d4-a716-446655440000",
  "status": "completed",
  "percent": 100,
  "history_id": 5001,
  "file_size": 104857600,
  "message": "下载完成"
}
```

**错误消息**
```json
{
  "task_id": "550e8400-e29b-41d4-a716-446655440000",
  "status": "failed",
  "percent": 45.2,
  "error_code": 50001,
  "message": "网络连接超时"
}
```

---

## 5. gRPC客户端集成

### 5.1 客户端连接池
```go
type GRPCClients struct {
    AuthClient   pb.AuthServiceClient
    ParserClient pb.ParserServiceClient
    AssetClient  pb.AssetServiceClient
}

func NewGRPCClients(cfg *config.Config) (*GRPCClients, error) {
    // Auth Service连接
    authConn, err := grpc.Dial(
        cfg.AuthServiceAddr,
        grpc.WithTransportCredentials(insecure.NewCredentials()),
        grpc.WithDefaultCallOptions(
            grpc.MaxCallRecvMsgSize(10*1024*1024),
            grpc.MaxCallSendMsgSize(10*1024*1024),
        ),
    )
    
    // 其他服务连接...
    
    return &GRPCClients{
        AuthClient:   pb.NewAuthServiceClient(authConn),
        ParserClient: pb.NewParserServiceClient(parserConn),
        AssetClient:  pb.NewAssetServiceClient(assetConn),
    }, nil
}
```

### 5.2 调用超时控制
```go
func (h *Handler) callAuthService(userID int64) error {
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    
    resp, err := h.authClient.GetUserInfo(ctx, &pb.GetUserInfoRequest{
        UserId: userID,
    })
    
    if err != nil {
        if code := status.Code(err); code == codes.DeadlineExceeded {
            return errors.New("auth service timeout")
        }
        return err
    }
    
    return nil
}
```

---

## 6. 配置管理

### 6.1 配置文件示例 (config/dev.yaml)
```yaml
server:
  port: 8080
  mode: debug  # debug, release
  read_timeout: 30s
  write_timeout: 30s
  max_header_bytes: 1048576  # 1MB

grpc:
  auth_service: "localhost:9001"
  parser_service: "localhost:9002"
  asset_service: "localhost:9004"
  timeout: 5s

redis:
  addr: "localhost:6379"
  password: ""
  db: 0
  pool_size: 100

rabbitmq:
  url: "amqp://guest:guest@localhost:5672/"
  exchange: "vasset.download.exchange"
  queue: "vasset.download.queue"
  routing_key: "download.quick"

cors:
  allowed_origins:
    - "http://localhost:3000"
    - "https://vasset.com"
  allowed_methods:
    - "GET"
    - "POST"
    - "PUT"
    - "DELETE"
  allowed_headers:
    - "Authorization"
    - "Content-Type"
  max_age: 86400

rate_limit:
  global_rps: 200
  user_rps: 10
  burst: 20

file_download:
  max_concurrent: 50  # 最大并发下载连接数
  buffer_size: 32768  # 32KB缓冲
```

---

## 7. 非功能性需求

### 7.1 性能指标
- **API响应时间(P95)**: < 200ms (不含文件下载)
- **WebSocket消息延迟**: < 100ms
- **文件下载吞吐**: 受限于磁盘IO和网络带宽
- **并发连接数**: 10000+

### 7.2 容错与降级
- **熔断**: 后端服务错误率>50%时触发熔断
- **降级**: 非核心接口(如历史记录)可降级为缓存数据
- **重试**: gRPC调用失败重试3次,指数退避

### 7.3 监控指标
```yaml
metrics:
  - http_requests_total (按路由、方法、状态码)
  - http_request_duration_seconds (P50/P95/P99)
  - websocket_connections_active
  - grpc_client_requests_total
  - grpc_client_request_duration_seconds
  - rate_limit_rejected_total
```

---

## 8. 部署与运维

### 8.1 健康检查
```http
GET /health

Response:
{
  "status": "healthy",
  "version": "1.0.0",
  "uptime": 86400,
  "dependencies": {
    "auth_service": "healthy",
    "parser_service": "healthy",
    "asset_service": "healthy",
    "redis": "healthy",
    "rabbitmq": "healthy"
  }
}
```

### 8.2 日志格式
```json
{
  "timestamp": "2025-12-03T10:30:15Z",
  "level": "info",
  "service": "api-gateway",
  "trace_id": "abc123",
  "method": "POST",
  "path": "/api/v1/download",
  "status": 202,
  "latency_ms": 45,
  "user_id": 1001,
  "ip": "192.168.1.100"
}
```

---

**下一步**: 查看其他微服务PRD文档以了解完整的系统设计。
