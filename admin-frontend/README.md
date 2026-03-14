# Admin Frontend

`admin-frontend` 是 V-Asset 的管理后台前端，基于 Next.js App Router 构建。

当前它是正式的后台控制台入口，对应的部署基路径为：

```text
/admin-console
```

## 当前页面

- `/login`：管理员登录页
- `/dashboard`：概览、请求趋势、用户统计、代理概况
- `/proxies`：代理管理页
- `/cookies`：Cookie 管理页

在默认 Nginx 部署下，这些页面实际访问路径类似：

- `/admin-console/login`
- `/admin-console/dashboard`
- `/admin-console/proxies`
- `/admin-console/cookies`

## 技术栈

- Next.js 16
- React 19
- TypeScript
- Tailwind CSS 4
- Axios
- Sonner

## 鉴权模型

后台不使用浏览器本地 JWT，而是依赖管理员 Session Cookie。

当前特点：

- Axios 开启 `withCredentials: true`
- 登录成功后由 Gateway 写入 HttpOnly Cookie
- 前端通过 `/api/v1/admin/auth/me` 判断当前是否有有效管理员会话
- `ProtectedRoute` 在未登录时跳转到 `/login`

相关代码：

- `lib/api-client.ts`
- `hooks/use-auth.ts`
- `components/auth/ProtectedRoute.tsx`

## 与后端的接入方式

### Next Route Handler 代理

后台前端并不直接把所有请求发到外部网关域名，而是通过：

- `app/api/v1/admin/[...path]/route.ts`

将请求代理到：

- `API_GATEWAY_INTERNAL_URL`
- 或 `NEXT_PUBLIC_API_BASE_URL`
- 默认为 `http://localhost:8080`

这个代理会转发：

- `cookie`
- `content-type`
- `user-agent`
- `set-cookie`

因此它适合在同域部署下稳定传递管理员会话。

## 环境变量

常见环境变量：

- `NEXT_PUBLIC_API_BASE_URL`
- `API_GATEWAY_INTERNAL_URL`

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

默认开发端口：`3001`

## 目录概览

```text
admin-frontend/
├── app/
│   ├── api/
│   ├── cookies/
│   ├── dashboard/
│   ├── login/
│   └── proxies/
├── components/
│   ├── auth/
│   ├── cookies/
│   ├── dashboard/
│   ├── layout/
│   ├── proxies/
│   └── ui/
├── hooks/
├── lib/
│   └── api/
└── next.config.ts
```

## 当前注意点

- `next.config.ts` 中已配置 `basePath: "/admin-console"`
- 后台 Cookie 列表页默认展示元数据；需要查看原始内容时再请求详情接口
- 后台登录和用户前台登录是两套不同会话模型，不要混用

## 推荐阅读顺序

1. `app/login/page.tsx`
2. `hooks/use-auth.ts`
3. `components/auth/ProtectedRoute.tsx`
4. `app/dashboard/page.tsx`
5. `app/proxies/page.tsx`
6. `app/cookies/page.tsx`
