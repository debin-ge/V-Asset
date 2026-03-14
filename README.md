# V-Asset

V-Asset 是一个面向视频解析、下载交付、资产沉淀和后台治理的完整平台，而不只是对 `yt-dlp` 的简单封装。

当前仓库已经形成一套清晰的分层：

- 用户前台负责链接提交、格式选择、历史查看和文件下载
- API Gateway 负责 HTTP / WebSocket 接入和统一鉴权
- Auth Service 负责用户认证、会话和 JWT 验证
- Media Service 负责解析、下载执行、进度发布和文件落盘
- Asset Service 负责历史、配额、统计、代理和 Cookie 治理
- Admin Service 负责管理员登录态和后台能力聚合
- Admin Frontend 负责代理、Cookie 和运营视角的管理界面

## 当前能力概览

- 多平台解析与下载，覆盖 YouTube、Bilibili、TikTok、Twitter/X、Instagram 等常见站点
- 用户登录、下载历史、使用统计和文件重新下载
- RabbitMQ 驱动的异步下载任务和 Redis 驱动的实时进度推送
- 代理源策略、任务级代理绑定和 Cookie 资产管理
- 独立的管理员会话模型与后台控制台

## 仓库结构

| 目录 | 角色 | 默认端口 / 路径 |
| --- | --- | --- |
| `frontend-service/` | 用户前台，Next.js Web 应用 | `3000` |
| `admin-frontend/` | 管理后台前端，Next.js Web 应用 | `3001`，部署后基路径为 `/admin-console` |
| `api-gateway/` | HTTP / WebSocket 网关，统一对外入口 | `8080` |
| `auth-service/` | 用户认证、JWT、会话和用户资料 | `9001` |
| `media-service/` | 视频解析、下载任务消费、进度发布 | `9002` |
| `asset-service/` | 历史、配额、统计、代理、Cookie | `9004` |
| `admin-service/` | 管理员登录态和后台聚合接口 | `9005` |
| `deploy/` | Nginx 与部署配置 | `80`（Docker Compose 默认入口） |
| `docs/` | 架构、用户和专题设计文档 | - |

## 关键架构特点

### 1. 两套鉴权模型并存

- 用户侧接口走 JWT，前台通过 `Authorization: Bearer <token>` 访问 `/api/v1/*`
- 管理后台走独立的 HttpOnly Cookie，会话名默认是 `vasset_admin_session`

这两套模型在代码中是明确分开的：

- `api-gateway/internal/middleware/auth.go`
- `api-gateway/internal/middleware/admin_session.go`

### 2. 下载链路是异步的

用户提交下载后，不会由 Gateway 直接执行下载，而是：

1. Gateway 校验用户并创建历史记录
2. Gateway 将任务投递到 RabbitMQ
3. Media Service 消费任务并执行下载
4. Media Service 通过 Redis PubSub 发布进度
5. Gateway 通过 WebSocket 将进度推送到前台
6. 下载完成后用户通过 Gateway 获取文件流

### 3. 治理能力集中在 Asset / Admin 侧

- Asset Service 持有下载历史、配额、统计、代理和 Cookie 的治理逻辑
- Admin Service 不是新的真源，而是为后台提供聚合编排和管理员会话能力

## 快速启动

### 方式一：使用 Docker Compose

在仓库根目录执行：

```bash
docker compose up -d --build
```

默认入口：

- 用户前台：`http://localhost/`
- 管理后台：`http://localhost/admin-console`
- 健康检查：`http://localhost/health`

### 方式二：按服务单独启动

各服务目录都提供了自己的 `README.md`、`Makefile` 或 `package.json` 脚本。常用命令：

```bash
# Go 服务
make proto
make build
make run
make test

# Next.js 前端
npm install
npm run dev
npm run lint
```

## 默认基础设施

本项目默认依赖以下组件：

- PostgreSQL：持久化用户、历史、配额、代理、Cookie 和下载状态
- Redis：缓存、用户 token 辅助校验、管理员会话和进度 PubSub
- RabbitMQ：下载任务异步投递与消费
- 本地磁盘：已下载文件存储
- yt-dlp：解析和下载执行引擎

## 主要文档

- [架构说明](docs/architecture.md)
- [用户使用说明](docs/user-guide.md)
- [API Gateway README](api-gateway/README.md)
- [Auth Service README](auth-service/README.md)
- [Media Service README](media-service/README.md)
- [Asset Service README](asset-service/README.md)
- [Admin Service README](admin-service/README.md)
- [Frontend README](frontend-service/README.md)
- [Admin Frontend README](admin-frontend/README.md)

## PRD 与专题设计

- [总 PRD](PRD/Backend_PRD.md)
- [后端 PRD 索引](PRD/Backend/README.md)
- [后台 PRD](PRD/Admin/Admin_PRD.md)
- [Web PRD](PRD/Web/Web_PRD.md)

## 当前阅读建议

如果你是第一次进入这个仓库，推荐阅读顺序：

1. `README.md`
2. `docs/architecture.md`
3. 你准备修改的服务 `README.md`
4. 对应服务的 `cmd/main.go` 与 `internal/` 目录

## 许可证

MIT
