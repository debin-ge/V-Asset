# Admin Service

`admin-service` 是 V-Asset 管理后台的聚合后端。

它不直接持有用户、代理或 Cookie 的真源数据，而是为管理后台提供：

- 管理员登录态
- 概览统计
- 代理治理入口
- Cookie 治理入口

## 当前职责

- 调用 `auth-service` 完成管理员登录验证
- 将管理员会话存入 Redis
- 校验并返回当前管理员信息
- 聚合 `auth-service` 与 `asset-service` 的统计数据
- 透传并编排代理管理能力
- 透传并编排 Cookie 管理能力

## 管理员身份模型

当前实现中，管理员登录逻辑是：

1. 通过 `auth-service` 执行普通用户登录
2. 校验返回用户的 `role`
3. 只有 `role == 99` 才允许创建管理员会话
4. 会话内容序列化后写入 Redis

因此这个服务的重点不是“再做一次密码体系”，而是：

- 把用户体系里的管理员账户转换成后台 Session
- 为后台提供专用鉴权入口

关键代码：

- `internal/service/auth_service.go`
- `internal/service/session_service.go`

## 运行依赖

- Redis
- Auth Service
- Asset Service

默认端口：`9005`

## 核心 RPC

定义位于 `proto/admin.proto`。

当前主要方法包括：

- `Login`
- `Logout`
- `GetCurrentUser`
- `GetOverview`
- `GetRequestTrend`
- `GetUserStats`
- `GetProxySourceStatus`
- `GetProxySourcePolicy`
- `UpdateProxySourcePolicy`
- `ListProxies`
- `CreateProxy`
- `UpdateProxy`
- `UpdateProxyStatus`
- `DeleteProxy`
- `ListCookies`
- `GetCookie`
- `CreateCookie`
- `UpdateCookie`
- `DeleteCookie`
- `FreezeCookie`

## 启动方式

```bash
make proto
make build
make run
```

测试：

```bash
go test ./...
```

## 配置重点

配置文件：`config/dev.yaml`

重点配置：

- `grpc.auth_service`
- `grpc.asset_service`
- `redis.*`
- `session.ttl`
- `session.cookie_name`
- `session.same_site`
- `session.secure`

常见环境变量：

- `AUTH_SERVICE_ADDR`
- `ASSET_SERVICE_ADDR`
- `REDIS_ADDR`
- `REDIS_PASSWORD`
- `SESSION_SECURE`
- `SESSION_SAME_SITE`
- `SESSION_COOKIE_DOMAIN`

## 关键目录

```text
admin-service/
├── cmd/main.go
├── internal/
│   ├── client/
│   ├── config/
│   ├── grpc/
│   ├── models/
│   └── service/
├── proto/
└── config/
```

## 代码阅读建议

推荐顺序：

1. `cmd/main.go`
2. `internal/grpc/admin_server.go`
3. `internal/service/auth_service.go`
4. `internal/service/session_service.go`
5. `internal/service/stats_service.go`
6. `internal/service/proxy_service.go`
7. `internal/service/cookie_service.go`

## 与 Gateway 的关系

浏览器不会直接访问这个服务。

访问路径始终是：

```text
admin-frontend -> api-gateway /api/v1/admin/* -> admin-service
```

Gateway 会负责：

- 设置和清理管理员 Cookie
- 在受保护路由上校验管理员 Session
- 将后台请求转发到本服务
