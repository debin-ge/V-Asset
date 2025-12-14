# Auth Service

V-Asset 认证鉴权服务,提供用户注册、登录、Token 验证等功能。

## 功能特性

- ✅ 用户注册和登录
- ✅ JWT Token 认证(Access Token + Refresh Token)
- ✅ bcrypt 密码加密
- ✅ Redis 缓存优化
- ✅ 登录失败限制(5次锁定30分钟)
- ✅ 会话管理(最多5个并发会话)
- ✅ 自动清理过期会话

## 技术栈

- **语言**: Go 1.21
- **框架**: gRPC
- **数据库**: PostgreSQL 15
- **缓存**: Redis 7
- **认证**: JWT (golang-jwt/jwt)
- **密码**: bcrypt

## 快速开始

### 1. 启动服务

```bash
# 使用 Docker Compose 启动所有服务
docker-compose up -d

# 查看日志
docker-compose logs -f auth-service
```

### 2. 数据库迁移

数据库表会在 PostgreSQL 容器启动时自动创建(通过 `migrations/001_init.sql`)。

如果需要手动执行迁移:

```bash
docker exec -i auth-postgres psql -U vasset -d vasset < migrations/001_init.sql
```

### 3. 测试接口

安装 grpcurl:
```bash
brew install grpcurl
```

#### 注册用户
```bash
grpcurl -plaintext -d '{
  "email": "test@example.com",
  "password": "Test1234",
  "nickname": "测试用户"
}' localhost:9001 auth.AuthService/Register
```

#### 登录
```bash
grpcurl -plaintext -d '{
  "email": "test@example.com",
  "password": "Test1234"
}' localhost:9001 auth.AuthService/Login
```

#### 验证 Token
```bash
grpcurl -plaintext -d '{
  "token": "<your_access_token>"
}' localhost:9001 auth.AuthService/VerifyToken
```

#### 刷新 Token
```bash
grpcurl -plaintext -d '{
  "refresh_token": "<your_refresh_token>"
}' localhost:9001 auth.AuthService/RefreshToken
```

## 项目结构

```
auth-service/
├── cmd/
│   └── main.go              # 主程序入口
├── internal/
│   ├── config/              # 配置管理
│   ├── handler/             # gRPC 处理器
│   ├── models/              # 数据模型
│   ├── repository/          # 数据访问层
│   ├── service/             # 业务逻辑层
│   └── utils/               # 工具函数
├── proto/                   # Proto 定义
├── migrations/              # 数据库迁移
├── config/                  # 配置文件
├── Dockerfile
├── docker-compose.yml
└── Makefile
```

## 配置说明

配置文件位于 `config/dev.yaml`:

```yaml
server:
  port: 9001                 # gRPC 服务端口

database:
  host: localhost
  port: 5432
  user: vasset
  password: password         # 可通过环境变量 DB_PASSWORD 覆盖
  dbname: vasset

redis:
  addr: localhost:6379
  password: ""               # 可通过环境变量 REDIS_PASSWORD 覆盖

jwt:
  secret: "..."              # 可通过环境变量 JWT_SECRET 覆盖
  access_token_ttl: 86400    # Access Token 有效期(24小时)
  refresh_token_ttl: 604800  # Refresh Token 有效期(7天)

password:
  bcrypt_cost: 10
  min_length: 8
  require_uppercase: true
  require_lowercase: true
  require_number: true
```

## 开发指南

### 本地开发

```bash
# 安装依赖
go mod download

# 生成 Proto 代码
make proto

# 编译
make build

# 运行
make run
```

### 修改 Proto 定义

1. 编辑 `proto/auth.proto`
2. 重新生成代码: `make proto`
3. 更新对应的处理器代码

## API 文档

### gRPC 服务

服务定义在 `proto/auth.proto`,包含以下 RPC 方法:

- `Register`: 用户注册
- `Login`: 用户登录
- `VerifyToken`: Token 验证
- `RefreshToken`: Token 刷新
- `Logout`: 用户登出
- `GetUserInfo`: 获取用户信息

详细的请求/响应格式请查看 Proto 定义文件。

## 安全说明

### 生产环境配置

⚠️ **重要**: 在生产环境中,必须修改以下配置:

1. **JWT Secret**: 使用强随机字符串
   ```bash
   export JWT_SECRET=$(openssl rand -base64 32)
   ```

2. **数据库密码**: 使用强密码
   ```bash
   export DB_PASSWORD=<strong_password>
   ```

3. **Redis 密码**: 配置 Redis 密码
   ```bash
   export REDIS_PASSWORD=<redis_password>
   ```

### 密码策略

- 最短 8 字符
- 至少包含 1 个大写字母
- 至少包含 1 个小写字母
- 至少包含 1 个数字

### Token 安全

- Access Token 有效期: 24 小时
- Refresh Token 有效期: 7 天
- Token 使用 HS256 签名算法
- Token 验证结果缓存 5 分钟(Redis)

### 防暴力破解

- 登录失败 5 次后锁定账号 30 分钟
- 使用 Redis 记录登录失败次数

## 监控

### 日志

服务使用结构化日志,输出格式:

```json
{
  "timestamp": "2025-12-03T10:30:00Z",
  "level": "info",
  "service": "auth-service",
  "method": "Login",
  "user_id": 1001,
  "email": "user@example.com",
  "success": true
}
```

### 健康检查

```bash
# 检查服务是否运行
docker ps | grep auth-service

# 检查数据库连接
docker exec auth-postgres pg_isready -U vasset

# 检查 Redis 连接
docker exec auth-redis redis-cli ping
```

## 故障排查

### 服务无法启动

1. 检查端口是否被占用: `lsof -i :9001`
2. 检查数据库连接: `docker logs auth-postgres`
3. 检查 Redis 连接: `docker logs auth-redis`

### Token 验证失败

1. 检查 JWT Secret 是否一致
2. 检查 Token 是否过期
3. 检查 Redis 连接是否正常

### 登录失败

1. 检查用户是否存在
2. 检查密码是否正确
3. 检查账号是否被锁定(登录失败次数)

## License

MIT
