# 全链路日志排障手册（request_id）

## 1. 目标

当出现“页面可访问但登录/注册失败”等问题时，通过同一个 `request_id` 在 `nginx -> api-gateway -> gRPC 服务` 进行跨容器检索，5 分钟内定位故障层。

## 2. 日志契约

- HTTP 请求头：支持并回传 `X-Request-ID`
- gRPC metadata：`x-request-id`
- 日志字段（核心）：
  - Nginx: `ts, request_id, method, uri, status, upstream_status, request_time, upstream_time`
  - Gateway: `service, layer=http, request_id, method, path, status, latency_ms, client_ip, user_id`
  - gRPC: `service, layer=grpc, request_id, grpc_method, grpc_code, latency_ms, peer_ip`
- 脱敏要求：禁止记录 `Authorization/Cookie/token/password` 明文值

## 3. 快速检索命令

先确认目标 `request_id`（示例：`rid-123`）：

```bash
RID="rid-123"
```

跨关键容器快速查一遍：

```bash
for s in youdlp-nginx youdlp-gateway youdlp-auth youdlp-media youdlp-asset youdlp-admin-service; do
  echo "========== $s =========="
  docker logs "$s" --since "30m" 2>&1 | grep "$RID" || true
done
```

按 compose service 名称查看（更贴近部署操作）：

```bash
docker compose logs nginx api-gateway auth-service media-service asset-service admin-service --since "30m" 2>&1 | grep "$RID"
```

指定北京时间时间窗（例如 `2026-03-24 00:00~01:00 +08:00`）：

```bash
docker compose logs nginx api-gateway auth-service media-service asset-service admin-service \
  --since "2026-03-24T00:00:00+08:00" \
  --until "2026-03-24T01:00:00+08:00" 2>&1 | grep "$RID"
```

## 4. 故障定位路径（按层）

1. 先看 Nginx
- 若 Nginx 无该 `request_id`：请求可能未到入口（DNS/CDN/证书/客户端）。
- 若 Nginx 有且 `status` 为 `4xx/5xx`：先按 Nginx `uri/status/upstream_status` 判断。

2. 再看 Gateway
- Nginx 有、Gateway 无：Nginx 到 Gateway 反代链路异常。
- Gateway 有且 `status` 为 `401/403`：优先鉴权、CORS、会话/Token。
- Gateway 有且 `status` 为 `5xx`：继续查后端 gRPC 服务。

3. 再看 gRPC 服务（auth/media/asset/admin）
- Gateway 有、某 gRPC 服务无该 `request_id`：多为调用未发出或 metadata 未透传。
- gRPC 有且 `grpc_code != OK`：基本可落到具体服务方法（`grpc_method`）。

## 5. 常见场景速查

- 登录/注册失败：
  - 先查 `auth-service` 的 `grpc_method`（如 `Login/Register`）和 `grpc_code`
  - 若 auth 无日志，回溯 Gateway 对应请求状态和错误
- 下载失败：
  - 链路通常涉及 `media-service` + `asset-service`
  - 先看 Gateway HTTP `status`，再看对应 gRPC `grpc_code`
- WebSocket 进度异常：
  - 先确认 Gateway WS 接入日志，再确认任务相关服务日志是否有同一 `request_id`

## 6. 值班 SOP（On-call）

1. 收集信息：用户时间点（明确时区）、接口路径、账号、是否可复现。
2. 锁定时间窗：统一换算到绝对时间（建议保留 `+08:00`）。
3. 找 `request_id`：优先从前端响应头或 Nginx access log 抽取。
4. 跨层检索：按 `nginx -> gateway -> grpc` 顺序执行检索命令。
5. 判定故障层：记录第一处异常状态码/错误码及上下游证据。
6. 输出结论：包含“影响范围、根因层、临时止血、后续修复项”。

## 7. 运行与容量建议

- Docker 日志滚动（已在 `docker-compose.yml` 关键服务启用）：
  - `driver: json-file`
  - `max-size: 100m`
  - `max-file: 7`
- 建议每周巡检：
  - 单机磁盘占用
  - 日志保留窗口是否满足回溯需求
  - 抽样验证 20 条请求跨层检索成功率
