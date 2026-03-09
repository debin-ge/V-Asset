package grpcserver

import (
	"context"
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"vasset/admin-service/internal/models"
	"vasset/admin-service/internal/service"
	pb "vasset/admin-service/proto"
)

type AdminServer struct {
	pb.UnimplementedAdminServiceServer
	authService   *service.AuthService
	statsService  *service.StatsService
	proxyService  *service.ProxyService
	cookieService *service.CookieService
}

func NewAdminServer(
	authService *service.AuthService,
	statsService *service.StatsService,
	proxyService *service.ProxyService,
	cookieService *service.CookieService,
) *AdminServer {
	return &AdminServer{
		authService:   authService,
		statsService:  statsService,
		proxyService:  proxyService,
		cookieService: cookieService,
	}
}

func (s *AdminServer) Login(ctx context.Context, req *pb.AdminLoginRequest) (*pb.AdminLoginResponse, error) {
	session, err := s.authService.Login(ctx, req.GetEmail(), req.GetPassword(), req.GetUserAgent(), req.GetIpAddress())
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, err.Error())
	}

	return &pb.AdminLoginResponse{
		SessionId: session.SessionID,
		User:      adminUserToProto(session.User),
	}, nil
}

func (s *AdminServer) Logout(ctx context.Context, req *pb.AdminLogoutRequest) (*pb.AdminOperationResponse, error) {
	if req.GetSessionId() == "" {
		return nil, status.Error(codes.InvalidArgument, "missing session id")
	}
	if err := s.authService.Logout(ctx, req.GetSessionId()); err != nil {
		return nil, mapServiceError(err)
	}
	return &pb.AdminOperationResponse{Success: true}, nil
}

func (s *AdminServer) GetCurrentUser(ctx context.Context, req *pb.AdminSessionRequest) (*pb.AdminCurrentUserResponse, error) {
	if req.GetSessionId() == "" {
		return nil, status.Error(codes.InvalidArgument, "missing session id")
	}

	user, err := s.authService.GetCurrentUser(ctx, req.GetSessionId())
	if err != nil {
		return nil, mapServiceError(err)
	}

	return &pb.AdminCurrentUserResponse{User: adminUserToProto(*user)}, nil
}

func (s *AdminServer) GetOverview(ctx context.Context, _ *pb.AdminEmpty) (*pb.AdminOverviewResponse, error) {
	resp, err := s.statsService.GetOverview(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &pb.AdminOverviewResponse{
		TotalUsers:          resp.TotalUsers,
		DailyActiveUsers:    resp.DailyActiveUsers,
		WeeklyActiveUsers:   resp.WeeklyActiveUsers,
		TotalDownloads:      resp.TotalDownloads,
		DownloadsToday:      resp.DownloadsToday,
		SuccessDownloads:    resp.SuccessDownloads,
		FailedDownloads:     resp.FailedDownloads,
		ActiveManualProxies: resp.ActiveManualProxies,
		TotalManualProxies:  resp.TotalManualProxies,
	}, nil
}

func (s *AdminServer) GetRequestTrend(ctx context.Context, req *pb.AdminRequestTrendRequest) (*pb.AdminRequestTrendResponse, error) {
	resp, err := s.statsService.GetRequestTrend(ctx, req.GetGranularity(), req.GetLimit())
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	points := make([]*pb.AdminTrendPoint, 0, len(resp.Points))
	for _, point := range resp.Points {
		points = append(points, &pb.AdminTrendPoint{Label: point.Label, Count: point.Count})
	}

	return &pb.AdminRequestTrendResponse{
		Granularity: resp.Granularity,
		Points:      points,
	}, nil
}

func (s *AdminServer) GetUserStats(ctx context.Context, _ *pb.AdminEmpty) (*pb.AdminUserStatsResponse, error) {
	resp, err := s.statsService.GetUserStats(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &pb.AdminUserStatsResponse{
		TotalUsers:        resp.TotalUsers,
		DailyActiveUsers:  resp.DailyActiveUsers,
		WeeklyActiveUsers: resp.WeeklyActiveUsers,
	}, nil
}

func (s *AdminServer) GetProxySourceStatus(ctx context.Context, _ *pb.AdminEmpty) (*pb.AdminProxySourceStatusResponse, error) {
	resp, err := s.proxyService.GetSourceStatus(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &pb.AdminProxySourceStatusResponse{
		Healthy:       resp.Healthy,
		Mode:          resp.Mode,
		Message:       resp.Message,
		ProxyUrl:      resp.ProxyURL,
		ProxyLeaseId:  resp.ProxyLeaseID,
		ProxyExpireAt: resp.ProxyExpireAt,
		CheckedAt:     resp.CheckedAt,
	}, nil
}

func (s *AdminServer) GetProxySourcePolicy(ctx context.Context, _ *pb.AdminEmpty) (*pb.AdminProxySourcePolicyResponse, error) {
	resp, err := s.proxyService.GetSourcePolicy(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return proxyPolicyToProto(resp), nil
}

func (s *AdminServer) UpdateProxySourcePolicy(ctx context.Context, req *pb.AdminUpdateProxySourcePolicyRequest) (*pb.AdminOperationResponse, error) {
	err := s.proxyService.UpdateSourcePolicy(ctx, req.GetId(), models.UpdateProxySourcePolicyRequest{
		PrimarySource:            req.GetPrimarySource(),
		FallbackSource:           req.GetFallbackSource(),
		FallbackEnabled:          req.GetFallbackEnabled(),
		DynamicTimeoutMS:         req.GetDynamicTimeoutMs(),
		DynamicRetryCount:        req.GetDynamicRetryCount(),
		DynamicCircuitBreakerSec: req.GetDynamicCircuitBreakerSec(),
		MinLeaseTTLSec:           req.GetMinLeaseTtlSec(),
		ManualSelectionStrategy:  req.GetManualSelectionStrategy(),
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &pb.AdminOperationResponse{Success: true}, nil
}

func (s *AdminServer) ListProxies(ctx context.Context, req *pb.AdminListProxiesRequest) (*pb.AdminListProxiesResponse, error) {
	modelReq := models.ListProxiesRequest{
		Search:   req.GetSearch(),
		Protocol: req.GetProtocol(),
		Region:   req.GetRegion(),
	}
	if req.GetHasStatus() {
		status := req.GetStatus()
		modelReq.Status = &status
	}

	resp, err := s.proxyService.List(ctx, modelReq)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	items := make([]*pb.AdminProxyInfo, 0, len(resp.Items))
	for _, item := range resp.Items {
		items = append(items, proxyInfoToProto(item))
	}
	return &pb.AdminListProxiesResponse{Items: items}, nil
}

func (s *AdminServer) CreateProxy(ctx context.Context, req *pb.AdminCreateProxyRequest) (*pb.AdminCreateResourceResponse, error) {
	id, err := s.proxyService.Create(ctx, models.CreateProxyRequest{
		Host:         req.GetHost(),
		Port:         req.GetPort(),
		Protocol:     req.GetProtocol(),
		Username:     req.GetUsername(),
		Password:     req.GetPassword(),
		Region:       req.GetRegion(),
		Priority:     req.GetPriority(),
		PlatformTags: req.GetPlatformTags(),
		Remark:       req.GetRemark(),
		Status:       req.GetStatus(),
	})
	if err != nil {
		return nil, mapDownstreamError(err)
	}
	return &pb.AdminCreateResourceResponse{Id: id}, nil
}

func (s *AdminServer) UpdateProxy(ctx context.Context, req *pb.AdminUpdateProxyRequest) (*pb.AdminOperationResponse, error) {
	err := s.proxyService.Update(ctx, req.GetId(), models.UpdateProxyRequest{
		Host:         req.GetHost(),
		Port:         req.GetPort(),
		Protocol:     req.GetProtocol(),
		Username:     req.GetUsername(),
		Password:     req.GetPassword(),
		Region:       req.GetRegion(),
		Priority:     req.GetPriority(),
		PlatformTags: req.GetPlatformTags(),
		Remark:       req.GetRemark(),
	})
	if err != nil {
		return nil, mapDownstreamError(err)
	}
	return &pb.AdminOperationResponse{Success: true}, nil
}

func (s *AdminServer) UpdateProxyStatus(ctx context.Context, req *pb.AdminUpdateProxyStatusRequest) (*pb.AdminOperationResponse, error) {
	if err := s.proxyService.UpdateStatus(ctx, req.GetId(), req.GetStatus()); err != nil {
		return nil, mapDownstreamError(err)
	}
	return &pb.AdminOperationResponse{Success: true}, nil
}

func (s *AdminServer) DeleteProxy(ctx context.Context, req *pb.AdminDeleteRequest) (*pb.AdminOperationResponse, error) {
	if err := s.proxyService.Delete(ctx, req.GetId()); err != nil {
		return nil, mapDownstreamError(err)
	}
	return &pb.AdminOperationResponse{Success: true}, nil
}

func (s *AdminServer) ListCookies(ctx context.Context, req *pb.AdminListCookiesRequest) (*pb.AdminListCookiesResponse, error) {
	resp, err := s.cookieService.List(ctx, models.ListCookiesRequest{
		Platform: req.GetPlatform(),
		Status:   req.GetStatus(),
		Page:     int(req.GetPage()),
		PageSize: int(req.GetPageSize()),
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	items := make([]*pb.AdminCookieInfo, 0, len(resp.Items))
	for _, item := range resp.Items {
		items = append(items, cookieInfoToProto(item))
	}

	return &pb.AdminListCookiesResponse{
		Total:    resp.Total,
		Page:     int32(resp.Page),
		PageSize: int32(resp.PageSize),
		Items:    items,
	}, nil
}

func (s *AdminServer) CreateCookie(ctx context.Context, req *pb.AdminCreateCookieRequest) (*pb.AdminCreateResourceResponse, error) {
	id, err := s.cookieService.Create(ctx, models.CreateCookieRequest{
		Platform:      req.GetPlatform(),
		Name:          req.GetName(),
		Content:       req.GetContent(),
		ExpireAt:      req.GetExpireAt(),
		FreezeSeconds: req.GetFreezeSeconds(),
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &pb.AdminCreateResourceResponse{Id: id}, nil
}

func (s *AdminServer) UpdateCookie(ctx context.Context, req *pb.AdminUpdateCookieRequest) (*pb.AdminOperationResponse, error) {
	err := s.cookieService.Update(ctx, req.GetId(), models.UpdateCookieRequest{
		Name:          req.GetName(),
		Content:       req.GetContent(),
		ExpireAt:      req.GetExpireAt(),
		FreezeSeconds: req.GetFreezeSeconds(),
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &pb.AdminOperationResponse{Success: true}, nil
}

func (s *AdminServer) DeleteCookie(ctx context.Context, req *pb.AdminDeleteRequest) (*pb.AdminOperationResponse, error) {
	if err := s.cookieService.Delete(ctx, req.GetId()); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &pb.AdminOperationResponse{Success: true}, nil
}

func (s *AdminServer) FreezeCookie(ctx context.Context, req *pb.AdminFreezeCookieRequest) (*pb.AdminFreezeCookieResponse, error) {
	resp, err := s.cookieService.Freeze(ctx, req.GetId(), req.GetFreezeSeconds())
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &pb.AdminFreezeCookieResponse{Success: resp.Success, FrozenUntil: resp.FrozenUntil}, nil
}

func adminUserToProto(user models.AdminUser) *pb.AdminUser {
	return &pb.AdminUser{
		UserId:    user.UserID,
		Email:     user.Email,
		Nickname:  user.Nickname,
		AvatarUrl: user.AvatarURL,
		Role:      user.Role,
		CreatedAt: user.CreatedAt,
	}
}

func proxyPolicyToProto(policy *models.ProxySourcePolicy) *pb.AdminProxySourcePolicyResponse {
	return &pb.AdminProxySourcePolicyResponse{
		Id:                       policy.ID,
		ScopeType:                policy.ScopeType,
		ScopeValue:               policy.ScopeValue,
		PrimarySource:            policy.PrimarySource,
		FallbackSource:           policy.FallbackSource,
		FallbackEnabled:          policy.FallbackEnabled,
		DynamicTimeoutMs:         policy.DynamicTimeoutMS,
		DynamicRetryCount:        policy.DynamicRetryCount,
		DynamicCircuitBreakerSec: policy.DynamicCircuitBreakerSec,
		MinLeaseTtlSec:           policy.MinLeaseTTLSec,
		ManualSelectionStrategy:  policy.ManualSelectionStrategy,
	}
}

func proxyInfoToProto(item models.ProxyInfo) *pb.AdminProxyInfo {
	return &pb.AdminProxyInfo{
		Id:           item.ID,
		Host:         item.Host,
		Port:         item.Port,
		Protocol:     item.Protocol,
		Username:     item.Username,
		Region:       item.Region,
		Priority:     item.Priority,
		PlatformTags: item.PlatformTags,
		Remark:       item.Remark,
		Status:       item.Status,
		LastUsedAt:   item.LastUsedAt,
		SuccessCount: item.SuccessCount,
		FailCount:    item.FailCount,
		CreatedAt:    item.CreatedAt,
		UpdatedAt:    item.UpdatedAt,
	}
}

func cookieInfoToProto(item models.CookieInfo) *pb.AdminCookieInfo {
	return &pb.AdminCookieInfo{
		Id:            item.ID,
		Platform:      item.Platform,
		Name:          item.Name,
		Content:       item.Content,
		Status:        item.Status,
		ExpireAt:      item.ExpireAt,
		FrozenUntil:   item.FrozenUntil,
		FreezeSeconds: item.FreezeSeconds,
		LastUsedAt:    item.LastUsedAt,
		UseCount:      item.UseCount,
		SuccessCount:  item.SuccessCount,
		FailCount:     item.FailCount,
		CreatedAt:     item.CreatedAt,
		UpdatedAt:     item.UpdatedAt,
	}
}

func mapServiceError(err error) error {
	if errors.Is(err, context.DeadlineExceeded) {
		return status.Error(codes.DeadlineExceeded, err.Error())
	}
	return status.Error(codes.Unauthenticated, err.Error())
}

func mapDownstreamError(err error) error {
	if st, ok := status.FromError(err); ok {
		return status.Error(st.Code(), st.Message())
	}
	return status.Error(codes.Internal, err.Error())
}
