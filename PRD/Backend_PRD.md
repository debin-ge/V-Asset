# 视频资产中台 (V-Asset) - 后端技术规格说明书 (MVP 修正版)

| 文档属性 | 详细信息 |
|:---|:---|
| **项目名称** | V-Asset Backend System |
| **版本** | V1.1.0 (Local Storage & Simplified Proxy) |
| **适用阶段** | MVP (Microservices Architecture) |
| **技术栈** | Go (Gin/gRPC), Redis, RabbitMQ, PostgreSQL |
| **核心变更** | **存储**：S3 -\> **本地磁盘**；**代理**：内部服务 -\> **直接调用第三方 API** |

-----

## 1\. 系统架构设计 (System Architecture)

### 1.1 架构图 (Microservices Topology)

由于移除了 S3 和 Proxy Pool Service，架构变得更精简。

```mermaid
graph TD
    Client[Web Client / Browser]
    
    subgraph "接入层 (API Gateway)"
        Gateway[**API Gateway Service**\n(Go/Gin)]
        WS_Handler[WebSocket Manager]
    end

    subgraph "基础设施 (Infrastructure)"
        Redis[(**Redis Cluster**)\n缓存/配额/PubSub]
        MQ[(**RabbitMQ**)\n任务队列]
        DB[(**PostgreSQL**)\n持久化存储]
        Disk[**Local Disk**\n视频文件存储]
    end

    subgraph "业务服务层 (Microservices)"
        Auth[**Auth Service**\n用户认证/鉴权]
        Parser[**Parser Service**\n元数据解析]
        Downloader[**Downloader Service**\n视频下载/处理/代理调用/文件管理]
        Asset[**Asset Service**\n历史记录/用户数据]
    end

    %% 链路关系
    Client -->|HTTPS| Gateway
    Client -->|WSS| WS_Handler
    
    Gateway -->|gRPC| Auth
    Gateway -->|gRPC| Parser
    Gateway -->|gRPC| Asset
    Gateway -->|Async/MQ| Downloader
    
    Downloader -->|Call 3rd Party API| ThirdParty[第三方 IP 服务商 API]
    Downloader -->|Write File| Disk
    Downloader -->|Pub Progress| Redis
    
    Redis -->|Sub Progress| WS_Handler
    
    Auth --> DB
    Asset --> DB
```

### 1.2 服务职责调整

| 服务名称 | 端口 | 核心职责 (变更点) |
|:---|:---|:---|
| **Downloader Service** | - | 消费队列，执行 `yt-dlp` 下载，**调用第三方 IP API 获取代理**，**存储文件到本地磁盘**，**管理本地文件生命周期**。 |
| **API Gateway** | 8080 | 增加 **`/download/file` 路由**，负责校验 Token 并从本地磁盘流式传输文件给用户。 |
| **Asset Service** | 9003 | **记录本地文件的存储路径** (`local_path`)，而非 S3 Key。 |
| **Proxy Service** | **已移除** | 职责并入 Downloader Service。 |

-----

## 2\. 核心业务流程与后端逻辑

### 2.1 任务执行 (Downloader Worker) 逻辑修正

*Worker 监听 RabbitMQ 队列，主要步骤如下：*

1.  **获取任务**：Pop `task_id`。
2.  **获取代理 IP**：
      * Downloader Service 调用**第三方 IP 服务商提供的 API** 获取一个高匿代理 IP（`http://user:pass@ip:port`）。
      * 若 API 调用失败，记录日志并返回错误，触发重试。
3.  **创建存储路径**：根据下载模式 (`quick_download` 或 `archive`) 和 `user_id` 在本地磁盘创建文件路径。
      * `quick_download`: `/data/tmp/{UUID}/`
      * `archive`: `/data/archive/{user_id}/{YYYYMMDD}/`
4.  **执行 yt-dlp (Shell 调用)**：
      * `yt-dlp --proxy "{ThirdPartyProxy}" --output "{local_path}" {URL}`
5.  **MD5 计算**：下载完成后，计算本地文件的 Hash。
6.  **文件交付与清理**：
      * **Archive 模式**：更新 `Download_History` 表中的 `file_path`，状态 `completed`。文件**永久保留**。
      * **Quick Download 模式**：更新 `Download_History` 表中的 `file_path`，状态 `pending_cleanup`，同时记录一个 `expire_at` 时间戳（例如：`now + 24h`）。
      * **定时清理 (Go Routine)**：Downloader Service 内部需运行一个定时任务（例如每小时一次），扫描 DB 中 `pending_cleanup` 且 `expire_at < now` 的记录，**物理删除**对应的本地文件，并更新 DB 状态为 `expired`。

### 2.2 下载交付流程 (Local Streaming)

由于不再使用 S3 Presigned URL，下载链接必须由后端服务自行生成和校验。

1.  **前端点击下载**：请求 `GET /api/v1/download/file?history_id={id}`。
2.  **API Gateway 校验**：
      * 验证 JWT Token，获取 `user_id`。
      * 调用 Asset Service 校验权限：`history_id` 是否属于 `user_id`。
3.  **Asset Service 权限校验**:
      * 查询 DB，获取 `file_path` 和 `file_name`。
      * 若记录不存在或 `user_id` 不匹配，返回 403。
4.  **API Gateway 文件传输**:
      * 使用 Go 的 `os.Open(file_path)` 读取本地文件。
      * 设置 `Content-Disposition` Header，使用 `file_name` 作为下载文件名。
      * 使用 `io.Copy()` 或类似的流式传输方法将文件内容写入 HTTP Response Body。
      * **关键**: 需设置并发限制，防止单个 Downloader Service 节点被大量下载请求拖垮带宽和内存。

-----

## 3\. 数据库设计 (Database Schema)

### 3.1 Download\_History (历史记录表) 字段修正

| 字段 | 类型 | 说明 |
|:---|:---|:---|
| `s3_key` | **已移除** | |
| `file_path` | Varchar | **本地磁盘绝对路径**（例如：`/data/archive/123/video.mp4`） |
| `file_size` | BigInt | 字节 |
| `file_hash` | Char(32) | MD5 校验码 |
| `status` | Int | 1:成功, 0:失败, 2:已过期 **(新增)**, 3:待清理 **(新增)** |
| `expire_at` | DateTime | **新增**：仅用于 Quick Download 模式，记录文件应被清理的时间戳 |

-----

## 4\. 接口详细定义 (API Specifications)

### 4.1 核心交付接口

#### A. 文件下载接口 (GET)

  * **Endpoint**: `GET /api/v1/download/file`
  * **Query**: `history_id={ID}`
  * **Headers**: `Authorization: Bearer <token>`
  * **Response**: **File Stream** (`Content-Type: application/octet-stream`)
  * **Backend Logic**: 权限校验、路径获取、本地文件流式传输（参见 2.2）。

#### B. 获取历史记录 (GET)

  * **Endpoint**: `GET /api/v1/user/history`
  * **Response**: 历史记录列表。
  * **Backend Logic**:
      * **无需再生成 Presigned URL。**
      * 返回结果中，每条记录只需包含 `history_id`。前端使用此 ID 构造 A 接口的下载链接。

-----

## 5\. 基础设施与非功能需求 (Infrastructure & NFR) 修正

### 5.1 动态 IP 接入 (Proxy Integration)

  * **实现方式**: Downloader Service 使用一个轻量级的 Go Client 封装对第三方 API 的调用。
  * **配置**: 配置文件中保存第三方服务的 `API Key` 和 `Endpoint`。
  * **故障转移**: Downloader Service 需配置合理的超时和错误处理机制。如果第三方 API 在 5 秒内无响应，应放弃本次任务并标记重试。

### 5.2 本地存储与生命周期管理 (Local Storage & TTL)

  * **存储路径**: 统一使用 `/data/vasset/` 作为根目录，并区分 `/data/vasset/tmp/` (极速下载) 和 `/data/vasset/archive/` (归档)。
  * **监控需求**: 需要配置主机监控系统 (Prometheus/Grafana) 监控 `/data` 挂载点的磁盘使用率。
      * **报警阈值**: 磁盘使用率达到 85% 时发出警告。
  * **文件清理任务**:
      * **服务**: 由 **Downloader Service** 负责执行。
      * **逻辑**: 定时任务 `CleanupRoutine()`，防止文件堆积耗尽磁盘。
      * **风险**: 若 Downloader Service 宕机，清理任务将暂停，可能导致磁盘溢出。需配置 H/A 或使用外部 Job Scheduler 确保其持续运行。

### 5.3 并发控制 (Concurrency)

  * **文件传输限流**: API Gateway 上的 `/download/file` 接口必须实现限流器 (Rate Limiter)，限制单个用户或全局的下载带宽，防止带宽资源被少数用户独占。
      * *例如：全局最大并发下载连接数 50 个。*