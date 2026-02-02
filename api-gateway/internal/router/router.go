package router

import (
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"

	"vasset/api-gateway/internal/client"
	"vasset/api-gateway/internal/config"
	"vasset/api-gateway/internal/handler"
	"vasset/api-gateway/internal/middleware"
)

// Dependencies 路由依赖
type Dependencies struct {
	Config      *config.Config
	GRPCClients *client.GRPCClients
	RedisClient *redis.Client
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
		deps.GRPCClients.ProxyClient,
		deps.Config.GRPC.Timeout,
	)
	historyHandler := handler.NewHistoryHandler(
		deps.GRPCClients.AssetClient,
		deps.Config.GRPC.Timeout,
	)
	healthHandler := handler.NewHealthHandler(
		deps.GRPCClients,
		deps.RedisClient,
		"1.0.0",
	)
	streamHandler := handler.NewStreamHandler(
		deps.GRPCClients.ProxyClient,
	)
	progressHandler := handler.NewProgressHandler(
		deps.GRPCClients.ProxyClient,
		deps.Config.GRPC.Timeout,
	)

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

		// 用户数据
		protectedV1.GET("/user/history", historyHandler.GetHistory)
		protectedV1.GET("/user/quota", historyHandler.GetQuota)
		protectedV1.GET("/user/stats", historyHandler.GetUserStats)
		protectedV1.DELETE("/user/history/:id", historyHandler.DeleteHistory)

		// 流式下载 (直接转发第三方流)
		protectedV1.GET("/stream", streamHandler.StreamDownload)

		// 进度查询
		protectedV1.GET("/progress/:task_id", progressHandler.GetProgress)
	}

	return r
}
