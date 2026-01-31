package client

import (
	"context"
	"log"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"vasset/api-gateway/internal/config"
	pb "vasset/api-gateway/proto"
)

// GRPCClients gRPC 客户端集合
type GRPCClients struct {
	AuthClient  pb.AuthServiceClient
	ProxyClient pb.ProxyServiceClient
	AssetClient pb.AssetServiceClient

	authConn  *grpc.ClientConn
	proxyConn *grpc.ClientConn
	assetConn *grpc.ClientConn
}

// NewGRPCClients 创建 gRPC 客户端
func NewGRPCClients(cfg *config.GRPCConfig) (*GRPCClients, error) {
	// 通用 gRPC 选项
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(10*1024*1024), // 10MB
			grpc.MaxCallSendMsgSize(10*1024*1024), // 10MB
		),
	}

	// 连接 Auth Service
	authConn, err := grpc.NewClient(cfg.AuthService, opts...)
	if err != nil {
		return nil, err
	}
	log.Printf("✓ Connected to Auth Service: %s", cfg.AuthService)

	// 连接 Proxy Service
	proxyConn, err := grpc.NewClient(cfg.ProxyService, opts...)
	if err != nil {
		authConn.Close()
		return nil, err
	}
	log.Printf("✓ Connected to Proxy Service: %s", cfg.ProxyService)

	// 连接 Asset Service
	assetConn, err := grpc.NewClient(cfg.AssetService, opts...)
	if err != nil {
		authConn.Close()
		proxyConn.Close()
		return nil, err
	}
	log.Printf("✓ Connected to Asset Service: %s", cfg.AssetService)

	return &GRPCClients{
		AuthClient:  pb.NewAuthServiceClient(authConn),
		ProxyClient: pb.NewProxyServiceClient(proxyConn),
		AssetClient: pb.NewAssetServiceClient(assetConn),
		authConn:    authConn,
		proxyConn:   proxyConn,
		assetConn:   assetConn,
	}, nil
}

// Close 关闭所有连接
func (c *GRPCClients) Close() {
	if c.authConn != nil {
		c.authConn.Close()
	}
	if c.proxyConn != nil {
		c.proxyConn.Close()
	}
	if c.assetConn != nil {
		c.assetConn.Close()
	}
}

// WithTimeout 创建带超时的上下文
func WithTimeout(timeout time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), timeout)
}
