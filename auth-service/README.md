# Auth Service

`auth-service` 是 V-Asset 的用户认证服务，负责用户账户、登录、JWT、Refresh Token 和用户会话管理。

## 当前职责

- 用户注册和登录
- Access Token / Refresh Token 生成
- Token 验证
- Token 刷新
- 用户登出
- 用户资料查询与修改
- 密码修改
- 平台用户数统计
- 用户 session 清理

## 当前实现特点

### 1. Token 验证是 session-backed 的

当前 `VerifyToken` 不只是“JWT 能解析就算合法”，还要求：

- token 对应的 session 仍然存在
- session 未过期
- token 所属用户与 session 用户一致

这意味着：

- 用户登出后，access token 不会继续在 session 删除后长期有效
- 刷新 token 时，会同步轮换 session 里保存的 `token_hash`

关键代码：

- `internal/service/token_service.go`
- `internal/repository/session_repository.go`

### 2. Auth Service 只提供 gRPC

浏览器不会直接访问这个服务；用户侧所有 HTTP 请求都先到 `api-gateway`。

## 核心 RPC

定义位于 `proto/auth.proto`。

常用方法包括：

- `Register`
- `Login`
- `VerifyToken`
- `RefreshToken`
- `Logout`
- `GetUserInfo`
- `UpdateProfile`
- `ChangePassword`
- `GetPlatformUserStats`

## 运行依赖

- PostgreSQL
- Redis

默认端口：`9001`

## 数据职责

Auth Service 负责的数据主要包括：

- 用户账号
- 用户密码哈希
- 登录失败状态
- 用户 session

Redis 主要用于：

- token claims 缓存
- 登录失败计数

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

## 关键目录

```text
auth-service/
├── cmd/main.go
├── internal/
│   ├── config/
│   ├── database/
│   ├── handler/
│   ├── models/
│   ├── repository/
│   ├── service/
│   └── utils/
├── migrations/
├── proto/
└── config/
```

## 配置重点

配置文件：`config/dev.yaml`

重点配置：

- `database.*`
- `redis.*`
- `jwt.secret`
- `jwt.access_token_ttl`
- `jwt.refresh_token_ttl`
- `session.max_sessions`
- `session.cleanup_interval`
- `password.*`

常见环境变量：

- `DB_HOST`
- `DB_PORT`
- `DB_USER`
- `DB_PASSWORD`
- `DB_NAME`
- `REDIS_ADDR`
- `REDIS_PASSWORD`
- `JWT_SECRET`

## 本地调试示例

```bash
grpcurl -plaintext -d '{
  "email": "test@example.com",
  "password": "Test1234",
  "nickname": "tester"
}' localhost:9001 auth.AuthService/Register
```

```bash
grpcurl -plaintext -d '{
  "email": "test@example.com",
  "password": "Test1234"
}' localhost:9001 auth.AuthService/Login
```

## 代码阅读建议

推荐阅读顺序：

1. `cmd/main.go`
2. `internal/handler/grpc_server.go`
3. `internal/service/auth_service.go`
4. `internal/service/token_service.go`
5. `internal/repository/`

## 安全提示

- 生产环境必须替换 `JWT_SECRET`
- `refresh_token_ttl` 不应短于 `access_token_ttl`
- 网关和 Auth Service 必须使用同一套鉴权约定，尤其是 token 刷新和登出行为
