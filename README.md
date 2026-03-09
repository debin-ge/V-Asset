# V-Asset

> 🎬 一站式视频资产下载与管理平台

V-Asset 面向“视频解析、下载交付、资产沉淀、后台治理”这一完整链路构建。它不是单一下载脚本的封装，而是一套可扩展的业务平台，适合承载用户下载、历史归档、配额控制、代理策略和后台运营管理等场景。

## ✨ 核心功能

- **多平台解析与下载**: 支持 YouTube、Bilibili、TikTok 等主流平台，提供元数据提取、格式选择与下载执行。
- **双端体验**: 同时提供用户前台与管理后台，覆盖用户使用和平台运营两类角色。
- **资产沉淀**: 下载历史、文件交付、配额统计与归档能力围绕同一套数据模型组织。
- **代理与 Cookie 治理**: 代理源策略、任务级代理绑定、Cookie 管理统一沉淀到后台能力中。
- **实时反馈**: 通过 WebSocket 推送任务进度，减少长任务场景下的等待不确定性。

## 🚀 核心优势

- **业务链路完整**: 从解析、下载、进度推送到文件交付和历史沉淀，链路闭环明确。
- **边界清晰**: Gateway 负责接入，Media 负责解析和下载，Asset 负责资产与代理治理，Admin 负责后台聚合。
- **适合持续演进**: 当前能力已经覆盖用户端、后台端、异步任务、代理策略和治理能力，便于后续继续扩展平台功能。

## 🏗️ 架构概览

系统采用前后端分离的微服务结构：

- `frontend-service` / `admin-frontend` 提供用户端与后台端界面
- `api-gateway` 作为统一 HTTP 入口
- `auth-service` 负责认证鉴权
- `media-service` 负责解析与下载执行
- `asset-service` 负责历史、配额、代理策略与 Cookie 管理
- `admin-service` 负责后台聚合与 Admin Session
- 基础设施使用 PostgreSQL、Redis、RabbitMQ

## 📁 仓库组成

- [frontend-service](frontend-service)
- [admin-frontend](admin-frontend)
- [api-gateway](api-gateway)
- [auth-service](auth-service)
- [media-service](media-service)
- [asset-service](asset-service)
- [admin-service](admin-service)


## 🛠️ 技术栈

- 前端: Next.js 16, React 19, TypeScript, TailwindCSS 4
- 后端: Go, Gin, gRPC, Protocol Buffers
- 基础设施: PostgreSQL, Redis, RabbitMQ
- 下载引擎: yt-dlp

## 📄 许可证

MIT License
