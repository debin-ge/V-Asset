package client

import (
	"context"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"vasset/admin-service/internal/config"
	pb "vasset/admin-service/proto"
)

type GRPCClients struct {
	AuthClient  pb.AuthServiceClient
	AssetClient pb.AssetServiceClient

	authConn  *grpc.ClientConn
	assetConn *grpc.ClientConn
}

func NewGRPCClients(cfg *config.GRPCConfig) (*GRPCClients, error) {
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}

	authConn, err := grpc.NewClient(cfg.AuthService, opts...)
	if err != nil {
		return nil, err
	}

	assetConn, err := grpc.NewClient(cfg.AssetService, opts...)
	if err != nil {
		authConn.Close()
		return nil, err
	}

	return &GRPCClients{
		AuthClient:  pb.NewAuthServiceClient(authConn),
		AssetClient: pb.NewAssetServiceClient(assetConn),
		authConn:    authConn,
		assetConn:   assetConn,
	}, nil
}

func (c *GRPCClients) Close() {
	if c.authConn != nil {
		c.authConn.Close()
	}
	if c.assetConn != nil {
		c.assetConn.Close()
	}
}

func WithTimeout(timeout time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), timeout)
}
