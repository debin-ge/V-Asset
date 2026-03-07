# V-Asset

> 🎬 一站式视频资产下载与管理平台

V-Asset 是一个现代化的视频下载平台，支持多平台视频解析与下载，提供优雅的用户界面和高效的后端服务。

## ✨ 功能特性

- 🔗 **多平台支持** - 支持 YouTube、Bilibili、TikTok 等主流视频平台
- 📊 **视频解析** - 快速解析视频元数据，支持多格式多清晰度选择
- ⬇️ **高速下载** - 基于 yt-dlp 的稳定下载引擎，支持代理加速
- 📁 **资产管理** - 完整的下载历史和资产管理功能
- 🔐 **用户系统** - 安全的用户认证与授权机制
- 🎨 **现代 UI** - 基于 Next.js 15 + TailwindCSS 4 的精美前端界面

## 🏗️ 技术架构

```
┌─────────────────────────────────────────────────────────┐
│                     Frontend (Next.js)                    │
│                      localhost:3000                       │
└─────────────────────────┬───────────────────────────────┘
                          │
┌─────────────────────────▼───────────────────────────────┐
│                     API Gateway                          │
│                      localhost:8080                      │
└─────┬─────────┬─────────┬──────────────────────────────┘
      │         │         │
┌─────▼───┐ ┌───▼───┐ ┌───▼───┐
│  Auth   │ │Media  │ │Asset  │
│ :9001   │ │ :9002 │ │ :9004 │
└─────────┘ └───────┘ └───────┘
      │         │         │
┌─────▼─────────▼─────────▼──────────────────────────────┐
│         PostgreSQL │ Redis │ RabbitMQ                   │
└────────────────────────────────────────────────────────┘
```

## 📦 服务列表

| 服务 | 端口 | 技术栈 | 描述 |
|------|------|--------|------|
| **Frontend** | 3000 | Next.js 16, React 19, TailwindCSS 4 | 前端 Web 应用 |
| **API Gateway** | 8080 | Go, HTTP/REST | 统一 API 入口，路由分发 |
| **Auth Service** | 9001 | Go, gRPC | 用户认证与授权 |
| **Media Service** | 9002 | Go, gRPC, yt-dlp | 视频 URL 解析与下载执行 |
| **Asset Service** | 9004 | Go, gRPC | 资产与历史管理 |
| **PostgreSQL** | 5432 | PostgreSQL 15 | 主数据库 |
| **Redis** | 6379 | Redis 7 | 缓存与会话 |
| **RabbitMQ** | 5672/15672 | RabbitMQ 3 | 消息队列 |

## 📁 项目结构

```
V-Asset/
├── frontend-service/     # 前端服务 (Next.js)
├── api-gateway/          # API 网关
├── auth-service/         # 认证服务
├── media-service/        # 解析+下载融合服务
├── asset-service/        # 资产服务
├── PRD/                  # 产品需求文档
├── docker-compose.yml    # Docker Compose 配置
└── README.md
```

## 🚀 快速启动

### 前置要求

- Docker & Docker Compose
- Node.js 18+ (本地开发)
- Go 1.21+ (本地开发)

### 一键启动

```bash
# 1. 克隆项目
git clone <repository-url>
cd V-Asset

# 2. 复制环境变量配置
cp .env.example .env

# 3. 修改 .env 中的敏感配置 (如 JWT_SECRET, DB_PASSWORD)
#    如果启用动态代理，还需要配置 PROXY_API_ENDPOINT / PROXY_API_KEY

# 4. 启动所有服务
docker-compose up -d

# 5. 查看服务状态
docker-compose ps

# 6. 查看日志
docker-compose logs -f
```

## 📖 常用命令

```bash
# 停止所有服务
docker-compose down

# 重建并启动
docker-compose up -d --build

# 只启动基础设施 (数据库、缓存、消息队列)
docker-compose up -d postgres redis rabbitmq

# 重启单个服务
docker-compose restart api-gateway

# 查看特定服务日志
docker-compose logs -f frontend-service

# 清理所有数据 (⚠️ 危险!)
docker-compose down -v
```

## 🌐 访问地址

| 服务 | 地址 | 说明 |
|------|------|------|
| **前端应用** | http://localhost:3000 | Web 用户界面 |
| **API Gateway** | http://localhost:8080 | 后端 API 接口 |
| **RabbitMQ 管理界面** | http://localhost:15672 | 账号: guest/guest |

## 代理配置

下载链路现在使用动态代理租约，不再在后台逐条录入代理地址。启动前请在根目录 `.env` 中配置：

```bash
PROXY_API_ENDPOINT=https://your-proxy-provider.example.com/get
PROXY_API_KEY=your-api-key
```

这两个变量会注入到 `asset-service`，由它在每次解析前实时请求代理，并把同一租约透传给下载任务。

## 💻 开发模式

### 启动基础设施

```bash
# 只启动基础设施服务
docker-compose up -d postgres redis rabbitmq
```

### 启动后端服务

```bash
# 在各服务目录下运行
cd auth-service && go run cmd/server/main.go
cd media-service && go run cmd/main.go
cd asset-service && go run cmd/server/main.go
cd api-gateway && go run cmd/server/main.go
```

### 启动前端服务

```bash
cd frontend-service

# 安装依赖
npm install

# 开发模式启动
npm run dev
```

## 📚 API 文档

API 文档请参考 [PRD/Backend_PRD.md](./PRD/Backend_PRD.md)

## 🛠️ 技术栈

### 前端
- **框架**: Next.js 16 (App Router)
- **语言**: TypeScript 5
- **样式**: TailwindCSS 4
- **UI 组件**: Radix UI + shadcn/ui
- **动画**: Framer Motion
- **状态管理**: React Hook Form + Zod
- **图表**: Recharts

### 后端
- **语言**: Go 1.21+
- **服务间通信**: gRPC + Protocol Buffers
- **HTTP 框架**: Gin
- **数据库**: PostgreSQL 15
- **缓存**: Redis 7
- **消息队列**: RabbitMQ 3
- **视频解析**: yt-dlp

## 📄 许可证

MIT License
