# Downloader Service

V-Asset 平台的视频下载调度服务，负责任务调度、代理管理、视频下载、文件管理、进度推送和定时清理。

## 功能特性

- **Worker池并发下载** - 支持多 Worker 并行处理下载任务
- **动态代理IP集成** - 支持第三方代理 API 集成
- **双模式下载** - 极速下载(临时文件)和归档下载(永久保存)
- **文件自动清理** - 定时清理过期的临时文件
- **实时进度推送** - 通过 Redis PubSub 推送下载进度
- **下载任务重试** - 失败任务自动重试机制

## 项目结构

```
downloader-service/
├── cmd/
│   └── main.go                 # 服务入口
├── config/
│   └── dev.yaml                # 开发环境配置
├── internal/
│   ├── cleanup/                # 清理模块
│   │   └── scheduler.go        # 定时清理调度器
│   ├── config/                 # 配置管理
│   │   └── config.go
│   ├── handler/                # gRPC 处理器
│   │   └── grpc_handler.go
│   ├── models/                 # 数据模型
│   │   └── download_history.go
│   ├── proxy/                  # 代理管理
│   │   └── provider.go
│   ├── repository/             # 数据访问层
│   │   └── download_repository.go
│   ├── service/                # 业务逻辑层
│   │   └── downloader_service.go
│   ├── storage/                # 存储管理
│   │   ├── file_manager.go
│   │   └── path_generator.go
│   ├── worker/                 # Worker池
│   │   ├── errors.go
│   │   ├── pool.go
│   │   ├── progress.go
│   │   └── task_consumer.go
│   └── ytdlp/                  # yt-dlp执行器
│       └── executor.go
├── migrations/
│   └── 001_download_history.sql
├── proto/
│   └── downloader.proto        # gRPC 接口定义
├── Dockerfile
├── docker-compose.yml
├── Makefile
├── go.mod
└── README.md
```

## 快速开始

### 前置要求

- Go 1.21+
- PostgreSQL 15+
- Redis 7+
- RabbitMQ 3+
- yt-dlp (下载用)
- ffmpeg (视频合并用)

### 本地开发

```bash
# 安装依赖
make deps

# 生成 protobuf 代码
make proto

# 运行服务
make run
```

### Docker 部署

```bash
# 启动所有服务(包括依赖)
make docker-up

# 停止
make docker-down
```

## 配置说明

配置文件: `config/dev.yaml`

| 配置项 | 说明 |
|--------|------|
| server.port | gRPC 服务端口 (9003) |
| worker.pool_size | Worker 池大小 |
| worker.max_concurrent | 最大并发下载数 |
| storage.base_path | 文件存储根路径 |
| storage.tmp_ttl | 临时文件过期时间(秒) |
| cleanup.interval | 清理任务间隔(秒) |

## gRPC 接口

| 方法 | 说明 |
|------|------|
| GetTaskStatus | 获取任务状态 |
| GetDownloadHistory | 获取用户下载历史 |
| CancelTask | 取消下载任务 |

## 任务状态

| 状态码 | 说明 |
|--------|------|
| 0 | pending - 待处理 |
| 1 | processing - 下载中 |
| 2 | completed - 完成 |
| 3 | failed - 失败 |
| 4 | pending_cleanup - 待清理 |
| 5 | expired - 已过期 |

## 环境变量

| 变量名 | 说明 |
|--------|------|
| DB_PASSWORD | 数据库密码 |
| REDIS_PASSWORD | Redis 密码 |
| RABBITMQ_URL | RabbitMQ 连接地址 |
| PROXY_API_KEY | 代理 API 密钥 |
