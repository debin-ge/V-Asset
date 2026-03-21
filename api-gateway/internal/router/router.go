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
		deps.RedisClient,
		deps.Config.GRPC.Timeout,
	)
	parseHandler := handler.NewParseHandler(
		deps.GRPCClients.MediaClient,
		deps.Config.GRPC.Timeout,
	)
	downloadHandler := handler.NewDownloadHandler(
		deps.GRPCClients.AssetClient,
		deps.GRPCClients.MediaClient,
		deps.MQPublisher,
		deps.Config.GRPC.Timeout,
		deps.Config.Billing.Enabled,
	)
	historyHandler := handler.NewHistoryHandler(
		deps.GRPCClients.AssetClient,
		deps.Config.GRPC.Timeout,
	)
	billingHandler := handler.NewBillingHandler(
		deps.GRPCClients.AssetClient,
		deps.Config.GRPC.Timeout,
	)
	fileHandler := handler.NewFileHandler(
		deps.GRPCClients.AssetClient,
		handler.NewRedisDownloadTicketStore(deps.RedisClient),
		deps.Config.GRPC.Timeout,
		deps.Config.FileDownload.BufferSize,
		deps.Config.Billing.Enabled,
	)
	healthHandler := handler.NewHealthHandler(
		deps.GRPCClients,
		deps.RedisClient,
		deps.MQPublisher,
		deps.WSManager,
		"1.0.0",
	)
	wsHandler := handler.NewWebSocketHandler(deps.WSManager)
	adminAuthHandler := handler.NewAdminAuthHandler(
		deps.GRPCClients.AdminClient,
		deps.Config.GRPC.Timeout,
		&deps.Config.AdminSession,
	)
	adminStatsHandler := handler.NewAdminStatsHandler(
		deps.GRPCClients.AdminClient,
		deps.Config.GRPC.Timeout,
	)
	adminProxyHandler := handler.NewAdminProxyHandler(
		deps.GRPCClients.AdminClient,
		deps.Config.GRPC.Timeout,
	)
	adminCookieHandler := handler.NewAdminCookieHandler(
		deps.GRPCClients.AdminClient,
		deps.Config.GRPC.Timeout,
	)
	adminBillingHandler := handler.NewAdminBillingHandler(
		deps.GRPCClients.AdminClient,
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
		publicV1.POST("/admin/auth/login", adminAuthHandler.Login)
		publicV1.GET("/download/file/browser", fileHandler.DownloadFileByTicket)
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
		protectedV1.GET("/user/account", billingHandler.GetAccount)
		protectedV1.GET("/user/billing/ledger", billingHandler.ListStatements)
		protectedV1.POST("/user/billing/estimate", billingHandler.Estimate)
		protectedV1.DELETE("/user/history/:id", historyHandler.DeleteHistory)

		// 文件下载
		protectedV1.POST("/download/file-ticket", fileHandler.CreateDownloadTicket)
		protectedV1.GET("/download/file", fileHandler.DownloadFile)

	}

	adminV1 := r.Group("/api/v1/admin")
	adminV1.Use(middleware.AdminSession(deps.GRPCClients.AdminClient, &deps.Config.AdminSession, deps.Config.GRPC.Timeout))
	adminV1.Use(middleware.RateLimit(rateLimiter))
	{
		adminV1.POST("/auth/logout", adminAuthHandler.Logout)
		adminV1.GET("/auth/me", adminAuthHandler.Me)

		adminV1.GET("/stats/overview", adminStatsHandler.Overview)
		adminV1.GET("/stats/requests", adminStatsHandler.RequestTrend)
		adminV1.GET("/stats/users", adminStatsHandler.Users)

		adminV1.GET("/proxies/source/status", adminProxyHandler.GetSourceStatus)
		adminV1.GET("/proxy-policies/current", adminProxyHandler.GetSourcePolicy)
		adminV1.PUT("/proxy-policies/:id", adminProxyHandler.UpdateSourcePolicy)
		adminV1.GET("/proxies", adminProxyHandler.List)
		adminV1.POST("/proxies", adminProxyHandler.Create)
		adminV1.PUT("/proxies/:id", adminProxyHandler.Update)
		adminV1.PATCH("/proxies/:id/status", adminProxyHandler.UpdateStatus)
		adminV1.DELETE("/proxies/:id", adminProxyHandler.Delete)

		adminV1.GET("/cookies", adminCookieHandler.List)
		adminV1.GET("/cookies/:id", adminCookieHandler.Get)
		adminV1.POST("/cookies", adminCookieHandler.Create)
		adminV1.PUT("/cookies/:id", adminCookieHandler.Update)
		adminV1.DELETE("/cookies/:id", adminCookieHandler.Delete)
		adminV1.POST("/cookies/:id/freeze", adminCookieHandler.Freeze)

		adminV1.GET("/billing/accounts", adminBillingHandler.ListAccounts)
		adminV1.GET("/billing/accounts/:userId", adminBillingHandler.GetAccountDetail)
		adminV1.POST("/billing/accounts/:userId/adjustments", adminBillingHandler.AdjustBalance)
		adminV1.GET("/billing/shortfalls", adminBillingHandler.ListShortfalls)
		adminV1.POST("/billing/shortfalls/:orderNo/reconcile", adminBillingHandler.ReconcileShortfall)
		adminV1.GET("/billing/ledger", adminBillingHandler.ListLedger)
		adminV1.GET("/billing/usage-records", adminBillingHandler.ListUsageRecords)
		adminV1.GET("/billing/pricing", adminBillingHandler.GetPricing)
		adminV1.PUT("/billing/pricing", adminBillingHandler.UpdatePricing)
	}

	// ==================== WebSocket 路由 ====================
	// WebSocket 进度推送 (浏览器通过子协议传递 bearer token)
	r.GET("/api/v1/ws/progress", wsHandler.Progress)

	return r
}
