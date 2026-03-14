# Frontend Service

`frontend-service` 是 V-Asset 的用户前台，基于 Next.js App Router 构建。

它负责用户侧的核心体验：

- 登录和注册
- 视频链接解析
- 格式选择
- 下载任务提交
- WebSocket 进度展示
- 用户中心中的历史与统计

## 技术栈

- Next.js 16
- React 19
- TypeScript
- Tailwind CSS 4
- Axios
- Sonner

## 页面与功能

### 主要页面

- `/`：首页，负责链接输入、解析和下载
- `/user`：用户中心，包含 Profile / History / Stats

### 关键能力

- 通过 `useDownload` 统一管理解析、下载和进度状态
- 通过 `lib/api/*` 封装后端接口
- 通过 `lib/ws-client.ts` 订阅下载进度
- 下载完成后自动触发浏览器文件下载

## 与后端的接入方式

### 1. 浏览器 API 调用

前台默认通过 Axios 调用 `/api/v1/*`。

当前用户鉴权方式：

- token 保存在浏览器本地存储
- 请求头自动附带 `Authorization: Bearer <token>`

相关代码：

- `lib/api-client.ts`
- `hooks/use-auth.ts`

### 2. Next Route Handler 代理

前台还提供了一个服务端代理层：

- `app/api/v1/[...path]/route.ts`

这个代理会把请求转发到：

- `API_GATEWAY_INTERNAL_URL`
- 或 `NEXT_PUBLIC_API_BASE_URL`
- 默认为 `http://localhost:8080`

它适合：

- SSR / Route Handler 场景
- 同域反代部署
- 避免前端直接硬编码网关地址

### 3. WebSocket 进度连接

当前 WebSocket 连接地址为：

- `/api/v1/ws/progress?task_id=<task_id>`

鉴权不是通过 query token，而是通过 WebSocket subprotocol：

```text
["bearer", token]
```

## 环境变量

常见环境变量：

- `NEXT_PUBLIC_API_BASE_URL`
- `API_GATEWAY_INTERNAL_URL`
- `NEXT_PUBLIC_WS_URL`
- `NEXT_PUBLIC_APP_VERSION`
- `NEXT_PUBLIC_PARSE_TIMEOUT_MS`

## 本地开发

```bash
npm install
npm run dev
```

其他常用命令：

```bash
npm run build
npm run start
npm run lint
```

默认端口：`3000`

## 目录概览

```text
frontend-service/
├── app/
│   ├── api/
│   ├── user/
│   └── page.tsx
├── components/
│   ├── auth/
│   ├── common/
│   ├── home/
│   ├── ui/
│   └── user/
├── hooks/
├── lib/
│   ├── api/
│   └── ws-client.ts
└── next.config.ts
```

## 当前注意点

- 下载文件现在通过带鉴权头的请求获取，不再把 token 放进 URL
- WebSocket 认证同样不再走 query 参数
- 仓库里还存在一个旧的 `app/admin` 页面目录，但当前正式后台控制台已经独立到 `admin-frontend`

## 推荐阅读顺序

1. `app/page.tsx`
2. `hooks/use-download.ts`
3. `lib/api/parse.ts`
4. `lib/api/download.ts`
5. `lib/ws-client.ts`
