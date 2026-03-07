package handler

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"time"

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

// AcquireProxyForTask 按任务获取或复用代理绑定
func (h *ProxyHandler) AcquireProxyForTask(ctx context.Context, req *pb.AcquireProxyForTaskRequest) (*pb.AcquireProxyForTaskResponse, error) {
	if req.TaskId == "" {
		return nil, status.Error(codes.InvalidArgument, "任务 ID 不能为空")
	}

	var protocol *models.ProxyProtocol
	var region *string
	var platform *string

	if req.Protocol != "" {
		p := models.ProxyProtocol(req.Protocol)
		protocol = &p
	}
	if req.Region != "" {
		region = &req.Region
	}
	if req.Platform != "" {
		platform = &req.Platform
	}

	binding, err := h.proxyService.AcquireProxyForTask(ctx, req.TaskId, protocol, region, platform)
	if err != nil {
		log.Printf("AcquireProxyForTask error: %v", err)
		return nil, status.Error(codes.NotFound, "没有可用的代理")
	}

	resp := &pb.AcquireProxyForTaskResponse{
		ProxyUrl:      binding.ProxyURLSnapshot,
		SourceType:    string(binding.SourceType),
		IsDegraded:    binding.IsDegraded,
		DegradeReason: "",
	}
	if binding.ProxyLeaseID != nil {
		resp.ProxyLeaseId = *binding.ProxyLeaseID
	}
	if binding.ProxyID != nil {
		resp.ProxyId = *binding.ProxyID
	}
	if binding.DegradeReason != nil {
		resp.DegradeReason = *binding.DegradeReason
	}
	if binding.ExpireAt != nil {
		resp.ExpireAt = binding.ExpireAt.Format(time.RFC3339)
	}

	return resp, nil
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
	if req.TaskId == "" && req.ProxyLeaseId == "" {
		return nil, status.Error(codes.InvalidArgument, "任务 ID 和代理租约 ID 不能同时为空")
	}

	if err := h.proxyService.ReportUsage(ctx, req.TaskId, req.ProxyLeaseId, req.Stage, req.Success); err != nil {
		log.Printf("ReportProxyUsage error: %v", err)
		return nil, status.Error(codes.Internal, "报告使用结果失败")
	}

	return &pb.ReportProxyUsageResponse{Success: true}, nil
}

// GetProxySourcePolicy 获取当前生效策略
func (h *ProxyHandler) GetProxySourcePolicy(ctx context.Context, req *pb.GetProxySourcePolicyRequest) (*pb.GetProxySourcePolicyResponse, error) {
	policy, err := h.proxyService.GetSourcePolicy(ctx)
	if err != nil {
		log.Printf("GetProxySourcePolicy error: %v", err)
		return nil, status.Error(codes.Internal, "获取代理策略失败")
	}
	if policy == nil {
		return nil, status.Error(codes.NotFound, "代理策略不存在")
	}

	resp := &pb.GetProxySourcePolicyResponse{
		Id:                       policy.ID,
		ScopeType:                policy.ScopeType,
		PrimarySource:            string(policy.PrimarySource),
		FallbackEnabled:          policy.FallbackEnabled,
		DynamicTimeoutMs:         int32(policy.DynamicTimeoutMS),
		DynamicRetryCount:        int32(policy.DynamicRetryCount),
		DynamicCircuitBreakerSec: int32(policy.DynamicCircuitBreakerSec),
		MinLeaseTtlSec:           int32(policy.MinLeaseTTLSec),
		ManualSelectionStrategy:  policy.ManualSelectionStrategy,
	}
	if policy.ScopeValue != nil {
		resp.ScopeValue = *policy.ScopeValue
	}
	if policy.FallbackSource != nil {
		resp.FallbackSource = *policy.FallbackSource
	}
	return resp, nil
}

// UpdateProxySourcePolicy 更新全局策略
func (h *ProxyHandler) UpdateProxySourcePolicy(ctx context.Context, req *pb.UpdateProxySourcePolicyRequest) (*pb.UpdateProxySourcePolicyResponse, error) {
	var fallbackSource *string
	if req.FallbackSource != "" {
		fallbackSource = &req.FallbackSource
	}

	if err := h.proxyService.UpdateSourcePolicy(
		ctx,
		req.Id,
		req.PrimarySource,
		fallbackSource,
		req.FallbackEnabled,
		int(req.DynamicTimeoutMs),
		int(req.DynamicRetryCount),
		int(req.DynamicCircuitBreakerSec),
		int(req.MinLeaseTtlSec),
		req.ManualSelectionStrategy,
	); err != nil {
		log.Printf("UpdateProxySourcePolicy error: %v", err)
		if errors.Is(err, sql.ErrNoRows) {
			return nil, status.Error(codes.NotFound, "代理策略不存在")
		}
		return nil, status.Error(codes.Internal, "更新代理策略失败")
	}

	return &pb.UpdateProxySourcePolicyResponse{Success: true}, nil
}

// ListProxies 列出手动代理
func (h *ProxyHandler) ListProxies(ctx context.Context, req *pb.ListProxiesRequest) (*pb.ListProxiesResponse, error) {
	var search *string
	var protocol *string
	var region *string
	var proxyStatus *models.ProxyStatus

	if req.Search != "" {
		search = &req.Search
	}
	if req.Protocol != "" {
		protocol = &req.Protocol
	}
	if req.Region != "" {
		region = &req.Region
	}
	if req.Status >= 0 {
		parsed := models.ProxyStatus(req.Status)
		proxyStatus = &parsed
	}

	items, err := h.proxyService.ListProxies(ctx, search, protocol, region, proxyStatus)
	if err != nil {
		log.Printf("ListProxies error: %v", err)
		return nil, status.Error(codes.Internal, "获取代理列表失败")
	}

	respItems := make([]*pb.ProxyInfo, 0, len(items))
	for _, item := range items {
		respItems = append(respItems, toProxyInfo(item))
	}

	return &pb.ListProxiesResponse{Items: respItems}, nil
}

// CreateProxy 创建手动代理
func (h *ProxyHandler) CreateProxy(ctx context.Context, req *pb.CreateProxyRequest) (*pb.CreateProxyResponse, error) {
	host := req.Host
	proxy := &models.Proxy{
		Host:         &host,
		IP:           req.Host,
		Port:         int(req.Port),
		Protocol:     models.ProxyProtocol(req.Protocol),
		Priority:     int(req.Priority),
		Status:       models.ProxyStatus(req.Status),
		Region:       nil,
		PlatformTags: nil,
		Remark:       nil,
	}
	if req.Region != "" {
		proxy.Region = &req.Region
	}
	if req.PlatformTags != "" {
		proxy.PlatformTags = &req.PlatformTags
	}
	if req.Remark != "" {
		proxy.Remark = &req.Remark
	}
	if req.Username != "" {
		proxy.Username = &req.Username
	}
	if req.Password != "" {
		proxy.Password = &req.Password
	}

	id, err := h.proxyService.CreateProxy(ctx, proxy)
	if err != nil {
		log.Printf("CreateProxy error: %v", err)
		return nil, status.Error(codes.Internal, "创建代理失败")
	}

	return &pb.CreateProxyResponse{Id: id}, nil
}

// UpdateProxy 更新手动代理
func (h *ProxyHandler) UpdateProxy(ctx context.Context, req *pb.UpdateProxyRequest) (*pb.UpdateProxyResponse, error) {
	host := req.Host
	proxy := &models.Proxy{
		ID:       req.Id,
		Host:     &host,
		IP:       req.Host,
		Port:     int(req.Port),
		Protocol: models.ProxyProtocol(req.Protocol),
		Priority: int(req.Priority),
	}
	if req.Region != "" {
		proxy.Region = &req.Region
	}
	if req.PlatformTags != "" {
		proxy.PlatformTags = &req.PlatformTags
	}
	if req.Remark != "" {
		proxy.Remark = &req.Remark
	}
	if req.Username != "" {
		proxy.Username = &req.Username
	}
	if req.Password != "" {
		proxy.Password = &req.Password
	}

	if err := h.proxyService.UpdateProxy(ctx, proxy); err != nil {
		log.Printf("UpdateProxy error: %v", err)
		if h.proxyService.IsNotFoundError(err) {
			return nil, status.Error(codes.NotFound, "代理不存在")
		}
		return nil, status.Error(codes.Internal, "更新代理失败")
	}

	return &pb.UpdateProxyResponse{Success: true}, nil
}

// UpdateProxyStatus 更新代理状态
func (h *ProxyHandler) UpdateProxyStatus(ctx context.Context, req *pb.UpdateProxyStatusRequest) (*pb.UpdateProxyStatusResponse, error) {
	if err := h.proxyService.UpdateProxyStatus(ctx, req.Id, models.ProxyStatus(req.Status)); err != nil {
		log.Printf("UpdateProxyStatus error: %v", err)
		if h.proxyService.IsNotFoundError(err) {
			return nil, status.Error(codes.NotFound, "代理不存在")
		}
		return nil, status.Error(codes.Internal, "更新代理状态失败")
	}

	return &pb.UpdateProxyStatusResponse{Success: true}, nil
}

// DeleteProxy 删除代理
func (h *ProxyHandler) DeleteProxy(ctx context.Context, req *pb.DeleteProxyRequest) (*pb.DeleteProxyResponse, error) {
	if err := h.proxyService.DeleteProxy(ctx, req.Id); err != nil {
		log.Printf("DeleteProxy error: %v", err)
		if h.proxyService.IsNotFoundError(err) {
			return nil, status.Error(codes.NotFound, "代理不存在")
		}
		return nil, status.Error(codes.Internal, "删除代理失败")
	}

	return &pb.DeleteProxyResponse{Success: true}, nil
}

func toProxyInfo(item *models.Proxy) *pb.ProxyInfo {
	resp := &pb.ProxyInfo{
		Id:           item.ID,
		Host:         item.IP,
		Port:         int32(item.Port),
		Protocol:     string(item.Protocol),
		Priority:     int32(item.Priority),
		Status:       int32(item.Status),
		SuccessCount: int32(item.SuccessCount),
		FailCount:    int32(item.FailCount),
		CreatedAt:    item.CreatedAt.Format(time.RFC3339),
		UpdatedAt:    item.UpdatedAt.Format(time.RFC3339),
	}
	if item.Host != nil && *item.Host != "" {
		resp.Host = *item.Host
	}
	if item.Region != nil {
		resp.Region = *item.Region
	}
	if item.PlatformTags != nil {
		resp.PlatformTags = *item.PlatformTags
	}
	if item.Remark != nil {
		resp.Remark = *item.Remark
	}
	if item.Username != nil {
		resp.Username = *item.Username
	}
	if item.LastUsedAt != nil {
		resp.LastUsedAt = item.LastUsedAt.Format(time.RFC3339)
	}
	return resp
}
