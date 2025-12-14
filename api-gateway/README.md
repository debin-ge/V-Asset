# API Gateway Service

视频资产下载平台的统一 API 网关服务。

## 功能特性

- RESTful API 统一入口
- JWT Token 验证（Redis 缓存）
- gRPC 客户端集成（Auth、Parser、Asset）
- WebSocket 进度推送
- 文件流式下载代理
- 限流与熔断保护

## 快速开始

### 依赖服务

- Redis (localhost:6379)
- RabbitMQ (localhost:5672)
- Auth Service (localhost:9001)
- Parser Service (localhost:9002)
- Asset Service (localhost:9004)

### 编译运行

```bash
# 生成 protobuf 代码
make proto

# 下载依赖
make deps

# 编译
make build

# 运行
make run
```

### Docker

```bash
# 构建镜像
make docker-build

# 启动服务
make docker-up
```

## API 端点

### 公开接口

| 方法 | 路径 | 描述 |
|------|------|------|
| POST | /api/v1/auth/register | 用户注册 |
| POST | /api/v1/auth/login | 用户登录 |
| GET | /health | 健康检查 |

### 认证接口

| 方法 | 路径 | 描述 |
|------|------|------|
| GET | /api/v1/auth/profile | 获取用户信息 |
| POST | /api/v1/auth/logout | 登出 |
| POST | /api/v1/parse | 解析视频 URL |
| POST | /api/v1/download | 提交下载任务 |
| GET | /api/v1/user/history | 下载历史 |
| GET | /api/v1/user/quota | 配额信息 |
| GET | /api/v1/download/file | 文件下载 |
| WS | /api/v1/ws/progress | 进度推送 |

## 配置

配置文件位于 `config/dev.yaml`，支持以下环境变量覆盖：

- `REDIS_PASSWORD` - Redis 密码
- `RABBITMQ_URL` - RabbitMQ 连接 URL
