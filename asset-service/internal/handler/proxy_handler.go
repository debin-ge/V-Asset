package handler

import (
	"context"
	"log"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"vasset/asset-service/internal/models"
	"vasset/asset-service/internal/service"
	pb "vasset/asset-service/proto"
)

// ProxyHandler 代理 gRPC 处理器
type ProxyHandler struct {
	proxyService *service.ProxyService
}

// NewProxyHandler 创建代理处理器
func NewProxyHandler(proxyService *service.ProxyService) *ProxyHandler {
	return &ProxyHandler{proxyService: proxyService}
}

// GetAvailableProxy 获取可用代理
func (h *ProxyHandler) GetAvailableProxy(ctx context.Context, req *pb.GetAvailableProxyRequest) (*pb.GetAvailableProxyResponse, error) {
	var protocol *models.ProxyProtocol
	var region *string

	if req.Protocol != "" {
		p := models.ProxyProtocol(req.Protocol)
		protocol = &p
	}
	if req.Region != "" {
		region = &req.Region
	}

	proxyURL, leaseID, expireAt, err := h.proxyService.GetAvailableProxy(ctx, protocol, region)
	if err != nil {
		log.Printf("GetAvailableProxy error: %v", err)
		return nil, status.Error(codes.NotFound, "没有可用的代理")
	}

	return &pb.GetAvailableProxyResponse{
		ProxyUrl:     proxyURL,
		ProxyLeaseId: leaseID,
		ExpireAt:     expireAt,
	}, nil
}

// ReportProxyUsage 报告代理使用结果
func (h *ProxyHandler) ReportProxyUsage(ctx context.Context, req *pb.ReportProxyUsageRequest) (*pb.ReportProxyUsageResponse, error) {
	if req.ProxyLeaseId == "" {
		return nil, status.Error(codes.InvalidArgument, "代理租约 ID 不能为空")
	}

	if err := h.proxyService.ReportUsage(ctx, req.ProxyLeaseId, req.Success); err != nil {
		log.Printf("ReportProxyUsage error: %v", err)
		return nil, status.Error(codes.Internal, "报告使用结果失败")
	}

	return &pb.ReportProxyUsageResponse{Success: true}, nil
}
