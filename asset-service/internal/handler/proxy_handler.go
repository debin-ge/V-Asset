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

// CreateProxy 创建代理
func (h *ProxyHandler) CreateProxy(ctx context.Context, req *pb.CreateProxyRequest) (*pb.CreateProxyResponse, error) {
	if req.Ip == "" || req.Port == 0 {
		return nil, status.Error(codes.InvalidArgument, "IP 和端口不能为空")
	}

	// 转换可空字段为指针
	var username, password, region *string
	if req.Username != "" {
		username = &req.Username
	}
	if req.Password != "" {
		password = &req.Password
	}
	if req.Region != "" {
		region = &req.Region
	}

	proxy := &models.Proxy{
		IP:       req.Ip,
		Port:     int(req.Port),
		Username: username,
		Password: password,
		Protocol: models.ProxyProtocol(req.Protocol),
		Region:   region,
	}

	id, passed, errMsg, err := h.proxyService.Create(ctx, proxy, req.CheckHealth)
	if err != nil {
		log.Printf("CreateProxy error: %v", err)
		return nil, status.Error(codes.Internal, "创建代理失败")
	}

	return &pb.CreateProxyResponse{
		Id:                id,
		HealthCheckPassed: passed,
		HealthCheckError:  errMsg,
	}, nil
}

// UpdateProxy 更新代理
func (h *ProxyHandler) UpdateProxy(ctx context.Context, req *pb.UpdateProxyRequest) (*pb.UpdateProxyResponse, error) {
	if req.Id == 0 {
		return nil, status.Error(codes.InvalidArgument, "代理 ID 不能为空")
	}

	// 转换可空字段为指针
	var username, password, region *string
	if req.Username != "" {
		username = &req.Username
	}
	if req.Password != "" {
		password = &req.Password
	}
	if req.Region != "" {
		region = &req.Region
	}

	proxy := &models.Proxy{
		ID:       req.Id,
		Username: username,
		Password: password,
		Protocol: models.ProxyProtocol(req.Protocol),
		Region:   region,
	}

	if err := h.proxyService.Update(ctx, proxy); err != nil {
		log.Printf("UpdateProxy error: %v", err)
		return nil, status.Error(codes.Internal, "更新代理失败")
	}

	return &pb.UpdateProxyResponse{Success: true}, nil
}

// DeleteProxy 删除代理
func (h *ProxyHandler) DeleteProxy(ctx context.Context, req *pb.DeleteProxyRequest) (*pb.DeleteProxyResponse, error) {
	if req.Id == 0 {
		return nil, status.Error(codes.InvalidArgument, "代理 ID 不能为空")
	}

	if err := h.proxyService.Delete(ctx, req.Id); err != nil {
		log.Printf("DeleteProxy error: %v", err)
		return nil, status.Error(codes.Internal, "删除代理失败")
	}

	return &pb.DeleteProxyResponse{Success: true}, nil
}

// GetProxy 获取代理
func (h *ProxyHandler) GetProxy(ctx context.Context, req *pb.GetProxyRequest) (*pb.GetProxyResponse, error) {
	if req.Id == 0 {
		return nil, status.Error(codes.InvalidArgument, "代理 ID 不能为空")
	}

	proxy, err := h.proxyService.GetByID(ctx, req.Id)
	if err != nil {
		log.Printf("GetProxy error: %v", err)
		return nil, status.Error(codes.Internal, "获取代理失败")
	}
	if proxy == nil {
		return nil, status.Error(codes.NotFound, "代理不存在")
	}

	return &pb.GetProxyResponse{
		Proxy: proxyToProto(proxy),
	}, nil
}

// ListProxies 列表代理
func (h *ProxyHandler) ListProxies(ctx context.Context, req *pb.ListProxiesRequest) (*pb.ListProxiesResponse, error) {
	filter := &models.ProxyFilter{
		Page:     int(req.Page),
		PageSize: int(req.PageSize),
	}

	if req.Status != 0 {
		s := models.ProxyStatus(req.Status)
		filter.Status = &s
	}
	if req.Protocol != "" {
		p := models.ProxyProtocol(req.Protocol)
		filter.Protocol = &p
	}
	if req.Region != "" {
		filter.Region = &req.Region
	}

	result, err := h.proxyService.List(ctx, filter)
	if err != nil {
		log.Printf("ListProxies error: %v", err)
		return nil, status.Error(codes.Internal, "查询代理列表失败")
	}

	items := make([]*pb.ProxyInfo, 0, len(result.Items))
	for _, p := range result.Items {
		items = append(items, proxyToProto(&p))
	}

	return &pb.ListProxiesResponse{
		Total:    result.Total,
		Page:     int32(result.Page),
		PageSize: int32(result.PageSize),
		Items:    items,
	}, nil
}

// CheckProxyHealth 检查代理健康
func (h *ProxyHandler) CheckProxyHealth(ctx context.Context, req *pb.CheckProxyHealthRequest) (*pb.CheckProxyHealthResponse, error) {
	if req.Id == 0 {
		return nil, status.Error(codes.InvalidArgument, "代理 ID 不能为空")
	}

	healthy, latency, err := h.proxyService.CheckHealthByID(ctx, req.Id)
	if err != nil {
		return &pb.CheckProxyHealthResponse{
			Healthy:   false,
			Error:     err.Error(),
			LatencyMs: latency,
		}, nil
	}

	return &pb.CheckProxyHealthResponse{
		Healthy:   healthy,
		LatencyMs: latency,
	}, nil
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

	proxyURL, proxyID, err := h.proxyService.GetAvailableProxy(ctx, protocol, region)
	if err != nil {
		log.Printf("GetAvailableProxy error: %v", err)
		return nil, status.Error(codes.NotFound, "没有可用的代理")
	}

	return &pb.GetAvailableProxyResponse{
		ProxyUrl: proxyURL,
		ProxyId:  proxyID,
	}, nil
}

// ReportProxyUsage 报告代理使用结果
func (h *ProxyHandler) ReportProxyUsage(ctx context.Context, req *pb.ReportProxyUsageRequest) (*pb.ReportProxyUsageResponse, error) {
	if req.ProxyId == 0 {
		return nil, status.Error(codes.InvalidArgument, "代理 ID 不能为空")
	}

	if err := h.proxyService.ReportUsage(ctx, req.ProxyId, req.Success); err != nil {
		log.Printf("ReportProxyUsage error: %v", err)
		return nil, status.Error(codes.Internal, "报告使用结果失败")
	}

	return &pb.ReportProxyUsageResponse{Success: true}, nil
}

// proxyToProto 将模型转换为 Proto
func proxyToProto(p *models.Proxy) *pb.ProxyInfo {
	info := &pb.ProxyInfo{
		Id:           p.ID,
		Ip:           p.IP,
		Port:         int32(p.Port),
		Protocol:     string(p.Protocol),
		Status:       int32(p.Status),
		SuccessCount: int32(p.SuccessCount),
		FailCount:    int32(p.FailCount),
		CreatedAt:    p.CreatedAt.Format("2006-01-02 15:04:05"),
		UpdatedAt:    p.UpdatedAt.Format("2006-01-02 15:04:05"),
	}

	// 转换指针类型字段
	if p.Username != nil {
		info.Username = *p.Username
	}
	if p.Password != nil {
		info.Password = *p.Password
	}
	if p.Region != nil {
		info.Region = *p.Region
	}
	if p.LastCheckAt != nil {
		info.LastCheckAt = p.LastCheckAt.Format("2006-01-02 15:04:05")
	}
	if p.LastCheckResult != nil {
		info.LastCheckResult = *p.LastCheckResult
	}
	if p.LastUsedAt != nil {
		info.LastUsedAt = p.LastUsedAt.Format("2006-01-02 15:04:05")
	}

	return info
}
