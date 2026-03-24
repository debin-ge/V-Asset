# API Gateway

`api-gateway` 是 YouDLP 的统一 HTTP / WebSocket 入口，负责把浏览器请求转换为对后端 gRPC 服务和消息队列的调用。

## 这个服务负责什么

- 暴露用户端 REST API
- 暴露管理后台 REST API
- 区分用户 JWT 与管理员 Session 两套鉴权模型
- 将请求转发到 `auth-service`、`media-service`、`asset-service`、`admin-service`
- 发布下载任务到 RabbitMQ
- 将 Redis 中的进度消息通过 WebSocket 转发给前端
- 提供文件下载出口
- 提供健康检查、就绪检查和版本信息

它不负责：

- 真正执行视频解析或下载
- 保存用户、历史、代理、Cookie 等业务真源数据

## 运行依赖

- Redis
- RabbitMQ
- Auth Service
- Media Service
- Asset Service
- Admin Service

默认端口：`8080`

## 路由概览

### 公开接口

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| `GET` | `/health` | 健康检查 |
| `GET` | `/ready` | 就绪检查 |
| `GET` | `/live` | 存活检查 |
| `GET` | `/version` | 版本信息 |
| `POST` | `/api/v1/auth/register` | 用户注册 |
| `POST` | `/api/v1/auth/login` | 用户登录 |
| `POST` | `/api/v1/admin/auth/login` | 管理员登录 |

### 用户接口（JWT 保护）

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| `POST` | `/api/v1/auth/logout` | 用户登出 |
| `GET` | `/api/v1/auth/profile` | 获取用户资料 |
| `PUT` | `/api/v1/auth/profile` | 修改用户资料 |
| `PUT` | `/api/v1/auth/password` | 修改密码 |
| `POST` | `/api/v1/parse` | 解析视频链接 |
| `POST` | `/api/v1/download` | 提交下载任务 |
| `GET` | `/api/v1/download/file` | 下载已完成文件 |
| `GET` | `/api/v1/user/history` | 获取历史记录 |
| `DELETE` | `/api/v1/user/history/:id` | 删除历史记录 |
| `GET` | `/api/v1/user/quota` | 获取用户配额 |
| `GET` | `/api/v1/user/stats` | 获取用户统计 |

### 管理后台接口（Admin Session 保护）

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| `POST` | `/api/v1/admin/auth/logout` | 管理员登出 |
| `GET` | `/api/v1/admin/auth/me` | 当前管理员信息 |
| `GET` | `/api/v1/admin/stats/overview` | 概览数据 |
| `GET` | `/api/v1/admin/stats/requests` | 请求趋势 |
| `GET` | `/api/v1/admin/stats/users` | 用户统计 |
| `GET` | `/api/v1/admin/proxies/source/status` | 代理源状态 |
| `GET` | `/api/v1/admin/proxy-policies/current` | 当前代理策略 |
| `PUT` | `/api/v1/admin/proxy-policies/:id` | 更新代理策略 |
| `GET` | `/api/v1/admin/proxies` | 代理列表 |
| `POST` | `/api/v1/admin/proxies` | 创建代理 |
| `PUT` | `/api/v1/admin/proxies/:id` | 更新代理 |
| `PATCH` | `/api/v1/admin/proxies/:id/status` | 更新代理状态 |
| `DELETE` | `/api/v1/admin/proxies/:id` | 删除代理 |
| `GET` | `/api/v1/admin/cookies` | Cookie 列表 |
| `GET` | `/api/v1/admin/cookies/:id` | Cookie 详情 |
| `POST` | `/api/v1/admin/cookies` | 创建 Cookie |
| `PUT` | `/api/v1/admin/cookies/:id` | 更新 Cookie |
| `DELETE` | `/api/v1/admin/cookies/:id` | 删除 Cookie |
| `POST` | `/api/v1/admin/cookies/:id/freeze` | 冻结 Cookie |

### WebSocket

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| `WS` | `/api/v1/ws/progress` | 任务进度推送 |

当前 WebSocket 认证方式：

- 不再使用 query 参数传 token
- 浏览器通过 `Sec-WebSocket-Protocol: bearer, <token>` 传递用户 token

## 鉴权模型

### 用户端

- 依赖 `Authorization: Bearer <token>`
- Gateway 先查 Redis，再调用 `auth-service` 进行 token 验证
- Token 必须同时满足 JWT 合法和后端 session 仍然存在

### 管理端

- 依赖管理员会话 Cookie
- 默认 Cookie 名：`youdlp_admin_session`
- 由 `admin-service` 校验并通过 Gateway 中间件注入管理员信息

## 下载与文件交付

用户提交下载后，Gateway 会：

1. 调用 `asset-service` 创建历史记录并检查配额
2. 把任务发布到 RabbitMQ
3. 返回 `task_id` 与 `history_id`

文件下载阶段：

- 前端使用带 `Authorization` 头的请求访问 `/api/v1/download/file`
- Gateway 从 `asset-service` 获取文件信息并流式输出
- 当前实现不再把 bearer token 放进下载 URL

## 关键代码入口

- 路由注册：`internal/router/router.go`
- 用户鉴权：`internal/middleware/auth.go`
- 管理员鉴权：`internal/middleware/admin_session.go`
- 通用 token 校验：`internal/middleware/token_auth.go`
- 下载提交：`internal/handler/download.go`
- 文件下载：`internal/handler/file.go`
- WebSocket 管理：`internal/ws/manager.go`

## 本地开发

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

- `grpc.*`：后端服务地址
- `admin_session.*`：管理员 Cookie 名、TTL、SameSite、Secure
- `redis.*`：缓存与进度订阅 Redis
- `rabbitmq.*`：下载任务投递配置
- `cors.*`：跨域白名单
- `file_download.buffer_size`：文件流 buffer 大小

常见环境变量覆盖：

- `AUTH_SERVICE_ADDR`
- `MEDIA_SERVICE_ADDR`
- `ASSET_SERVICE_ADDR`
- `ADMIN_SERVICE_ADDR`
- `REDIS_ADDR`
- `REDIS_PASSWORD`
- `RABBITMQ_URL`
- `ADMIN_SESSION_COOKIE_NAME`
- `ADMIN_SESSION_COOKIE_DOMAIN`
- `ADMIN_SESSION_SAME_SITE`
- `ADMIN_SESSION_SECURE`

## 当前注意点

- 如果浏览器需要跨域访问 Gateway，`cors.allowed_origins` 必须显式配置
- 如果只通过同域 Nginx 反代访问，可减少大量本地跨域问题
- 下载和 WebSocket 的前端接入方式已经基于当前安全模型收紧，文档和接入代码应保持一致
