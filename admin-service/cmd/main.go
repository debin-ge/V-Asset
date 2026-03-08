package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"

	"vasset/admin-service/internal/client"
	"vasset/admin-service/internal/config"
	grpcserver "vasset/admin-service/internal/grpc"
	"vasset/admin-service/internal/service"
	pb "vasset/admin-service/proto"
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

	lis, err := net.Listen("tcp", net.JoinHostPort("", formatPort(cfg.Server.Port)))
	if err != nil {
		log.Fatalf("failed to listen on port %d: %v", cfg.Server.Port, err)
	}

	grpcSrv := grpc.NewServer()
	pb.RegisterAdminServiceServer(grpcSrv, grpcserver.NewAdminServer(authService, statsService, proxyService, cookieService))

	go func() {
		log.Printf("admin-service gRPC listening on :%d", cfg.Server.Port)
		if err := grpcSrv.Serve(lis); err != nil {
			log.Fatalf("failed to start grpc server: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	stopped := make(chan struct{})
	go func() {
		grpcSrv.GracefulStop()
		close(stopped)
	}()

	select {
	case <-shutdownCtx.Done():
		grpcSrv.Stop()
	case <-stopped:
	}
}

func formatPort(port int) string {
	return strconv.Itoa(port)
}
