package router

import (
	"github.com/gin-gonic/gin"

	"vasset/admin-service/internal/config"
	"vasset/admin-service/internal/handler"
	"vasset/admin-service/internal/middleware"
	"vasset/admin-service/internal/service"
)

type Dependencies struct {
	Config         *config.Config
	HealthHandler  *handler.HealthHandler
	AuthHandler    *handler.AuthHandler
	StatsHandler   *handler.StatsHandler
	ProxyHandler   *handler.ProxyHandler
	CookieHandler  *handler.CookieHandler
	SessionService *service.SessionService
}

func SetupRouter(deps *Dependencies) *gin.Engine {
	if deps.Config.Server.Mode == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(middleware.Recovery())
	r.Use(middleware.Logger())
	r.Use(middleware.CORS(&deps.Config.CORS))

	r.GET("/health", deps.HealthHandler.Health)

	publicV1 := r.Group("/api/v1/admin")
	{
		publicV1.POST("/auth/login", deps.AuthHandler.Login)
	}

	protectedV1 := r.Group("/api/v1/admin")
	protectedV1.Use(middleware.SessionAuth(deps.SessionService, deps.Config.Session.CookieName))
	protectedV1.Use(middleware.RequireAdmin())
	{
		protectedV1.POST("/auth/logout", deps.AuthHandler.Logout)
		protectedV1.GET("/auth/me", deps.AuthHandler.Me)

		protectedV1.GET("/stats/overview", deps.StatsHandler.Overview)
		protectedV1.GET("/stats/requests", deps.StatsHandler.RequestTrend)
		protectedV1.GET("/stats/users", deps.StatsHandler.Users)

		protectedV1.GET("/proxies/source/status", deps.ProxyHandler.GetSourceStatus)
		protectedV1.GET("/proxy-policies/current", deps.ProxyHandler.GetSourcePolicy)
		protectedV1.PUT("/proxy-policies/:id", deps.ProxyHandler.UpdateSourcePolicy)
		protectedV1.GET("/proxies", deps.ProxyHandler.List)
		protectedV1.POST("/proxies", deps.ProxyHandler.Create)
		protectedV1.PUT("/proxies/:id", deps.ProxyHandler.Update)
		protectedV1.PATCH("/proxies/:id/status", deps.ProxyHandler.UpdateStatus)
		protectedV1.DELETE("/proxies/:id", deps.ProxyHandler.Delete)

		protectedV1.GET("/cookies", deps.CookieHandler.List)
		protectedV1.POST("/cookies", deps.CookieHandler.Create)
		protectedV1.PUT("/cookies/:id", deps.CookieHandler.Update)
		protectedV1.DELETE("/cookies/:id", deps.CookieHandler.Delete)
		protectedV1.POST("/cookies/:id/freeze", deps.CookieHandler.Freeze)
	}

	return r
}
