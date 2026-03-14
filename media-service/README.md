# Media Service

`media-service` 是 V-Asset 的解析与下载执行服务。

它既承担同步的“链接解析”能力，也承担异步的“下载任务消费与执行”能力。

## 当前职责

- 校验和解析视频链接
- 基于 yt-dlp 获取元数据与格式信息
- 缓存解析结果
- 消费 RabbitMQ 下载任务
- 下载文件并写入本地磁盘
- 发布 Redis 进度消息
- 与 `asset-service` 协作获取代理和 Cookie
- 回写下载状态与完成结果
- 周期清理临时文件
- 周期检测 yt-dlp 更新

## 两条核心链路

### 1. 解析链路

同步 gRPC 请求：

```text
api-gateway -> media-service ParseURL / ValidateURL
```

处理过程：

1. 识别平台
2. 查询 Redis 缓存
3. 未命中时调用 yt-dlp
4. 标准化格式信息并返回

### 2. 下载链路

异步任务消费：

```text
api-gateway -> RabbitMQ
media-service task consumer -> worker pool -> yt-dlp download
```

处理过程：

1. 消费下载消息
2. 更新下载状态
3. 必要时从 `asset-service` 获取代理和 Cookie
4. 执行下载并写入本地磁盘
5. 将进度通过 Redis PubSub 发布
6. 更新下载结果和历史状态

## 当前实现特点

### 1. 解析和下载共用一个服务

这不是单一 Parser 服务，而是“解析 + 下载融合服务”。

优点是：

- 解析阶段和下载阶段可以共享平台识别、yt-dlp 能力和部分配置
- 代理 / Cookie 策略可以在执行侧统一落地

### 2. 下载任务已按最大重试次数控制

当前任务消费者会读取 `retry.max_attempts`：

- 未达到上限时重新发布任务
- 达到上限后确认终态失败

这避免了永久失败任务无限重入队。

### 3. 进度推送走 Redis PubSub

Media Service 不直接与浏览器通信，而是：

- 发布 `progress:<task_id>` 消息到 Redis
- 由 `api-gateway` 的 WebSocket 管理器转发给前端

## 运行依赖

- PostgreSQL
- Redis
- RabbitMQ
- yt-dlp
- 可选：Asset Service（用于代理 / Cookie / 历史同步）

默认端口：`9002`

## 关键目录

```text
media-service/
├── cmd/main.go
├── internal/
│   ├── adapter/
│   ├── cache/
│   ├── config/
│   ├── detector/
│   ├── download/
│   │   ├── cleanup/
│   │   ├── client/
│   │   ├── config/
│   │   ├── database/
│   │   ├── repository/
│   │   ├── scheduler/
│   │   ├── storage/
│   │   ├── worker/
│   │   └── ytdlp/
│   ├── handler/
│   ├── platformpolicy/
│   ├── service/
│   └── ytdlp/
├── proto/
└── config/
```

## 启动方式

```bash
make proto
make deps
make build
make run
```

测试：

```bash
go test ./...
```

## 配置重点

配置文件：`config/dev.yaml`

重点配置项：

- `redis.*`
- `database.*`
- `rabbitmq.*`
- `worker.*`
- `retry.*`
- `ytdlp.*`
- `storage.*`
- `cleanup.*`
- `asset_service.*`

常见环境变量：

- `DB_HOST`
- `DB_PORT`
- `DB_USER`
- `DB_PASSWORD`
- `DB_NAME`
- `REDIS_ADDR`
- `REDIS_PASSWORD`
- `RABBITMQ_URL`
- `ASSET_SERVICE_ADDR`

## 本地调试

### 解析接口

```bash
grpcurl -plaintext -d '{"url":"https://www.youtube.com/watch?v=dQw4w9WgXcQ"}' \
  localhost:9002 media.MediaService/ParseURL
```

### 运行前确认

- `yt-dlp` 已安装且路径可用
- Redis 可连接
- RabbitMQ 可连接
- PostgreSQL 可连接

## 代码阅读建议

推荐阅读顺序：

1. `cmd/main.go`
2. `internal/handler/grpc_server.go`
3. `internal/service/`
4. `internal/download/worker/task_consumer.go`
5. `internal/download/worker/pool.go`
6. `internal/download/ytdlp/`

## 常见问题

### yt-dlp 不可用

请先确认：

```bash
which yt-dlp
```

必要时在配置中显式指定：

```yaml
ytdlp:
  binary_path: "/usr/local/bin/yt-dlp"
```

### 下载任务没有被消费

优先检查：

- RabbitMQ 是否可用
- `rabbitmq.queue` 是否与 Gateway 发布配置一致
- worker pool 是否正常启动
