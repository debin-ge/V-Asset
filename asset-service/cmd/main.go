package main

import (
	"database/sql"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	_ "github.com/lib/pq"
	"google.golang.org/grpc"

	"vasset/asset-service/internal/config"
	"vasset/asset-service/internal/database"
	"vasset/asset-service/internal/handler"
	"vasset/asset-service/internal/repository"
	"vasset/asset-service/internal/service"
	pb "vasset/asset-service/proto"
)

func main() {
	// 1. 加载配置
	cfg, err := config.LoadConfig("config/dev.yaml")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	log.Printf("Starting Asset Service on port %d...", cfg.Server.Port)

	log.Printf("Starting Asset Service on port %d...", cfg.Server.Port)

	// 2. 运行数据库迁移
	if err := database.RunMigrations(cfg.Database.GetURL()); err != nil {
		log.Fatalf("Failed to run database migrations: %v", err)
	}
	log.Println("✓ Database migrations completed")

	// 3. 连接 PostgreSQL
	db, err := initDatabase(&cfg.Database)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()
	log.Println("✓ Connected to PostgreSQL")

	// 3. 初始化 Repository
	historyRepo := repository.NewHistoryRepository(db)
	quotaRepo := repository.NewQuotaRepository(db)
	proxyRepo := repository.NewProxyRepository(db)
	cookieRepo := repository.NewCookieRepository(db)

	// 4. 初始化 Service
	historyService := service.NewHistoryService(historyRepo)
	quotaService := service.NewQuotaService(quotaRepo, &cfg.Quota)
	statsService := service.NewStatsService(historyRepo)
	proxyService := service.NewProxyService(proxyRepo, cfg)
	cookieService := service.NewCookieService(cookieRepo, cfg)

	// 5. 初始化 Handler
	proxyHandler := handler.NewProxyHandler(proxyService)
	cookieHandler := handler.NewCookieHandler(cookieService)

	// 6. 初始化 gRPC 服务器
	grpcServer := handler.NewGRPCServer(historyService, quotaService, statsService, proxyHandler, cookieHandler, cfg)

	// 6. 启动 gRPC 服务
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.Server.Port))
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	s := grpc.NewServer()
	pb.RegisterAssetServiceServer(s, grpcServer)

	// 7. 优雅关闭
	go func() {
		log.Printf("✓ gRPC server listening on :%d", cfg.Server.Port)
		if err := s.Serve(lis); err != nil {
			log.Fatalf("Failed to serve: %v", err)
		}
	}()

	// 等待中断信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")
	s.GracefulStop()
	log.Println("Server stopped")
}

// initDatabase 初始化数据库连接
func initDatabase(cfg *config.DatabaseConfig) (*sql.DB, error) {
	db, err := sql.Open("postgres", cfg.GetDSN())
	if err != nil {
		return nil, err
	}

	// 配置连接池
	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.ConnMaxLifetime)

	// 测试连接
	if err := db.Ping(); err != nil {
		return nil, err
	}

	return db, nil
}
