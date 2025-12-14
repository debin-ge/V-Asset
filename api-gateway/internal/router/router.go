package router

import (
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"

	"vasset/api-gateway/internal/client"
	"vasset/api-gateway/internal/config"
	"vasset/api-gateway/internal/handler"
	"vasset/api-gateway/internal/middleware"
	"vasset/api-gateway/internal/mq"
	"vasset/api-gateway/internal/ws"
)

// Dependencies 路由依赖
type Dependencies struct {
	Config      *config.Config
	GRPCClients *client.GRPCClients
	RedisClient *redis.Client
	MQPublisher *mq.Publisher
	WSManager   *ws.Manager
}

// SetupRouter 设置路由
func SetupRouter(deps *Dependencies) *gin.Engine {
	// 设置 Gin 模式
	if deps.Config.Server.Mode == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()

	// 全局中间件
	r.Use(middleware.Recovery())
	r.Use(middleware.Logger())
	r.Use(middleware.CORS(&deps.Config.CORS))

	// 创建限流器
	rateLimiter := middleware.NewRateLimiter(&deps.Config.RateLimit)

	// 创建处理器
	authHandler := handler.NewAuthHandler(
		deps.GRPCClients.AuthClient,
		deps.Config.GRPC.Timeout,
	)
	parseHandler := handler.NewParseHandler(
		deps.GRPCClients.ParserClient,
		deps.Config.GRPC.Timeout,
	)
	downloadHandler := handler.NewDownloadHandler(
		deps.GRPCClients.AssetClient,
		deps.GRPCClients.ParserClient,
		deps.MQPublisher,
		deps.Config.GRPC.Timeout,
	)
	historyHandler := handler.NewHistoryHandler(
		deps.GRPCClients.AssetClient,
		deps.Config.GRPC.Timeout,
	)
	fileHandler := handler.NewFileHandler(
		deps.GRPCClients.AssetClient,
		deps.Config.GRPC.Timeout,
		deps.Config.FileDownload.BufferSize,
	)
	healthHandler := handler.NewHealthHandler(
		deps.GRPCClients,
		deps.RedisClient,
		deps.MQPublisher,
		deps.WSManager,
		"1.0.0",
	)
	wsHandler := handler.NewWebSocketHandler(deps.WSManager)

	// ==================== 公开路由 ====================
	// 健康检查 (无需认证)
	r.GET("/health", healthHandler.HealthCheck)
	r.GET("/version", healthHandler.Version)
	r.GET("/ready", healthHandler.Ready)
	r.GET("/live", healthHandler.Live)

	// API v1 公开路由
	publicV1 := r.Group("/api/v1")
	publicV1.Use(middleware.IPRateLimit(rateLimiter))
	{
		// 认证接口
		publicV1.POST("/auth/register", authHandler.Register)
		publicV1.POST("/auth/login", authHandler.Login)
	}

	// ==================== 认证路由 ====================
	protectedV1 := r.Group("/api/v1")
	protectedV1.Use(middleware.JWTAuth(deps.GRPCClients.AuthClient, deps.RedisClient))
	protectedV1.Use(middleware.RateLimit(rateLimiter))
	{
		// 认证-用户
		protectedV1.POST("/auth/logout", authHandler.Logout)
		protectedV1.GET("/auth/profile", authHandler.GetProfile)
		protectedV1.PUT("/auth/profile", authHandler.UpdateProfile)
		protectedV1.PUT("/auth/password", authHandler.ChangePassword)

		// 解析
		protectedV1.POST("/parse", parseHandler.ParseURL)

		// 下载
		protectedV1.POST("/download", downloadHandler.SubmitDownload)

		// 用户数据
		protectedV1.GET("/user/history", historyHandler.GetHistory)
		protectedV1.GET("/user/quota", historyHandler.GetQuota)
		protectedV1.GET("/user/stats", historyHandler.GetUserStats)
		protectedV1.DELETE("/user/history/:id", historyHandler.DeleteHistory)

		// 文件下载
		protectedV1.GET("/download/file", fileHandler.DownloadFile)
	}

	// ==================== WebSocket 路由 ====================
	// WebSocket 进度推送 (Token 通过查询参数验证)
	r.GET("/api/v1/ws/progress", wsHandler.Progress)

	return r
}
