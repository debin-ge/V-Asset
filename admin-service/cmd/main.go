package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/redis/go-redis/v9"

	"vasset/admin-service/internal/client"
	"vasset/admin-service/internal/config"
	"vasset/admin-service/internal/handler"
	"vasset/admin-service/internal/router"
	"vasset/admin-service/internal/service"
)

func main() {
	cfg, err := config.LoadConfig("config/dev.yaml")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	redisClient := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
		PoolSize: cfg.Redis.PoolSize,
	})
	defer redisClient.Close()

	if err := redisClient.Ping(context.Background()).Err(); err != nil {
		log.Printf("Warning: failed to connect to redis: %v", err)
	} else {
		log.Printf("Connected to redis: %s", cfg.Redis.Addr)
	}

	grpcClients, err := client.NewGRPCClients(&cfg.GRPC)
	if err != nil {
		log.Fatalf("Failed to connect grpc clients: %v", err)
	}
	defer grpcClients.Close()

	sessionService := service.NewSessionService(redisClient, cfg.Session.TTL)
	authService := service.NewAuthService(grpcClients.AuthClient, sessionService)
	statsService := service.NewStatsService(grpcClients.AuthClient, grpcClients.AssetClient)
	proxyService := service.NewProxyService(grpcClients.AssetClient)
	cookieService := service.NewCookieService(grpcClients.AssetClient)

	deps := &router.Dependencies{
		Config:         cfg,
		HealthHandler:  handler.NewHealthHandler(),
		AuthHandler:    handler.NewAuthHandler(authService, cfg.Session.CookieName, cfg.Session.Secure, cfg.Session.CookieDomain, cfg.Session.SameSite),
		StatsHandler:   handler.NewStatsHandler(statsService),
		ProxyHandler:   handler.NewProxyHandler(proxyService),
		CookieHandler:  handler.NewCookieHandler(cookieService),
		SessionService: sessionService,
	}

	r := router.SetupRouter(deps)
	srv := &http.Server{
		Addr:           fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:        r,
		ReadTimeout:    cfg.Server.ReadTimeout,
		WriteTimeout:   cfg.Server.WriteTimeout,
		MaxHeaderBytes: cfg.Server.MaxHeaderBytes,
	}

	go func() {
		log.Printf("admin-service listening on :%d", cfg.Server.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("failed to start server: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("server forced to shutdown: %v", err)
	}
}
