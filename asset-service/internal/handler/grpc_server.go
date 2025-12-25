package handler

import (
	"context"
	"log"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"vasset/asset-service/internal/config"
	"vasset/asset-service/internal/models"
	"vasset/asset-service/internal/service"
	pb "vasset/asset-service/proto"
)

// GRPCServer gRPC 服务器
type GRPCServer struct {
	pb.UnimplementedAssetServiceServer
	historyService *service.HistoryService
	quotaService   *service.QuotaService
	statsService   *service.StatsService
	proxyHandler   *ProxyHandler
	cookieHandler  *CookieHandler
	cfg            *config.Config
}

// NewGRPCServer 创建 gRPC 服务器
func NewGRPCServer(
	historyService *service.HistoryService,
	quotaService *service.QuotaService,
	statsService *service.StatsService,
	proxyHandler *ProxyHandler,
	cookieHandler *CookieHandler,
	cfg *config.Config,
) *GRPCServer {
	return &GRPCServer{
		historyService: historyService,
		quotaService:   quotaService,
		statsService:   statsService,
		proxyHandler:   proxyHandler,
		cookieHandler:  cookieHandler,
		cfg:            cfg,
	}
}

// GetHistory 获取下载历史
func (s *GRPCServer) GetHistory(ctx context.Context, req *pb.GetHistoryRequest) (*pb.GetHistoryResponse, error) {
	// 参数验证
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "用户ID不能为空")
	}

	// 构建过滤条件
	filter := &models.HistoryFilter{
		UserID:    req.UserId,
		Page:      int(req.Page),
		PageSize:  int(req.PageSize),
		SortBy:    req.SortBy,
		SortOrder: req.SortOrder,
	}

	// 处理可选过滤条件
	if req.Status != 0 {
		status := models.HistoryStatus(req.Status)
		filter.Status = &status
	}

	if req.Platform != "" {
		filter.Platform = &req.Platform
	}

	if req.StartDate != "" {
		if t, err := time.Parse("2006-01-02", req.StartDate); err == nil {
			filter.StartDate = &t
		}
	}

	if req.EndDate != "" {
		if t, err := time.Parse("2006-01-02", req.EndDate); err == nil {
			// 将结束日期设为当天的23:59:59
			t = t.Add(24*time.Hour - time.Second)
			filter.EndDate = &t
		}
	}

	// 设置分页默认值
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.PageSize < 1 {
		filter.PageSize = s.cfg.Pagination.DefaultPageSize
	}
	if filter.PageSize > s.cfg.Pagination.MaxPageSize {
		filter.PageSize = s.cfg.Pagination.MaxPageSize
	}

	// 调用服务层
	result, err := s.historyService.GetHistory(ctx, filter)
	if err != nil {
		log.Printf("GetHistory error: %v", err)
		return nil, status.Error(codes.Internal, "查询历史记录失败")
	}

	// 转换响应
	items := make([]*pb.HistoryItem, 0, len(result.Items))
	for _, h := range result.Items {
		item := &pb.HistoryItem{
			HistoryId: h.ID,
			TaskId:    h.TaskID,
			Url:       h.URL,
			Platform:  h.Platform,
			Title:     h.Title,
			Mode:      h.Mode,
			Quality:   h.Quality,
			FileSize:  h.FileSize,
			Status:    int32(h.Status),
			FilePath:  h.FilePath,
			FileName:  h.FileName,
			CreatedAt: h.CreatedAt.Format("2006-01-02 15:04:05"),
		}
		if h.CompletedAt != nil {
			item.CompletedAt = h.CompletedAt.Format("2006-01-02 15:04:05")
		}
		items = append(items, item)
	}

	return &pb.GetHistoryResponse{
		Total:    result.Total,
		Page:     int32(result.Page),
		PageSize: int32(result.PageSize),
		Items:    items,
	}, nil
}

// DeleteHistory 删除历史记录
func (s *GRPCServer) DeleteHistory(ctx context.Context, req *pb.DeleteHistoryRequest) (*pb.DeleteHistoryResponse, error) {
	if req.HistoryId == 0 || req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "历史ID和用户ID不能为空")
	}

	err := s.historyService.DeleteHistory(ctx, req.HistoryId, req.UserId)
	if err != nil {
		log.Printf("DeleteHistory error: %v", err)
		if err.Error() == "record not found" {
			return nil, status.Error(codes.NotFound, "记录不存在")
		}
		return nil, status.Error(codes.Internal, "删除记录失败")
	}

	return &pb.DeleteHistoryResponse{Success: true}, nil
}

// CreateHistory 创建历史记录
func (s *GRPCServer) CreateHistory(ctx context.Context, req *pb.CreateHistoryRequest) (*pb.CreateHistoryResponse, error) {
	log.Printf("[GRPCServer] CreateHistory called for user %s, task %s", req.UserId, req.TaskId)

	// 参数验证
	if req.UserId == "" || req.TaskId == "" || req.Url == "" {
		log.Printf("[GRPCServer] CreateHistory validation failed: missing required fields")
		return nil, status.Error(codes.InvalidArgument, "用户ID、任务ID和URL不能为空")
	}

	// 构建历史记录
	history := &models.DownloadHistory{
		UserID:    req.UserId,
		TaskID:    req.TaskId,
		URL:       req.Url,
		Platform:  req.Platform,
		Title:     req.Title,
		Mode:      req.Mode,
		Quality:   req.Quality,
		Thumbnail: req.Thumbnail,
		Duration:  req.Duration,
		Author:    req.Author,
		Status:    models.StatusPending, // 初始状态为待处理
	}

	// 调用服务层创建
	historyID, err := s.historyService.CreateHistory(ctx, history)
	if err != nil {
		log.Printf("[GRPCServer] CreateHistory error: %v", err)
		return nil, status.Error(codes.Internal, "创建历史记录失败")
	}

	log.Printf("[GRPCServer] CreateHistory success: historyID=%d", historyID)
	return &pb.CreateHistoryResponse{
		HistoryId: historyID,
	}, nil
}

// CheckQuota 检查配额
func (s *GRPCServer) CheckQuota(ctx context.Context, req *pb.CheckQuotaRequest) (*pb.CheckQuotaResponse, error) {
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "用户ID不能为空")
	}

	quota, err := s.quotaService.CheckQuota(ctx, req.UserId)
	if err != nil {
		log.Printf("CheckQuota error: %v", err)
		return nil, status.Error(codes.Internal, "检查配额失败")
	}

	remaining := s.quotaService.GetRemaining(quota)

	return &pb.CheckQuotaResponse{
		DailyLimit: int32(quota.DailyLimit),
		DailyUsed:  int32(quota.DailyUsed),
		Remaining:  int32(remaining),
		ResetAt:    quota.ResetAt.Format("2006-01-02 15:04:05"),
	}, nil
}

// ConsumeQuota 消费配额
func (s *GRPCServer) ConsumeQuota(ctx context.Context, req *pb.ConsumeQuotaRequest) (*pb.ConsumeQuotaResponse, error) {
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "用户ID不能为空")
	}

	quota, err := s.quotaService.ConsumeQuota(ctx, req.UserId)
	if err != nil {
		log.Printf("ConsumeQuota error: %v", err)
		if err.Error() == "daily quota exceeded" {
			return nil, status.Error(codes.ResourceExhausted, "每日配额已用完")
		}
		return nil, status.Error(codes.Internal, "消费配额失败")
	}

	remaining := s.quotaService.GetRemaining(quota)

	return &pb.ConsumeQuotaResponse{
		Success:   true,
		Remaining: int32(remaining),
	}, nil
}

// GetUserStats 获取用户统计
func (s *GRPCServer) GetUserStats(ctx context.Context, req *pb.GetUserStatsRequest) (*pb.GetUserStatsResponse, error) {
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "用户ID不能为空")
	}

	stats, err := s.statsService.GetUserStats(ctx, req.UserId)
	if err != nil {
		log.Printf("GetUserStats error: %v", err)
		return nil, status.Error(codes.Internal, "获取统计信息失败")
	}

	// 转换平台统计
	platforms := make([]*pb.PlatformStat, 0, len(stats.TopPlatforms))
	for _, p := range stats.TopPlatforms {
		platforms = append(platforms, &pb.PlatformStat{
			Platform: p.Platform,
			Count:    p.Count,
		})
	}

	// 转换日活统计
	activities := make([]*pb.DailyActivity, 0, len(stats.RecentActivity))
	for _, a := range stats.RecentActivity {
		activities = append(activities, &pb.DailyActivity{
			Date:  a.Date,
			Count: a.Count,
		})
	}

	return &pb.GetUserStatsResponse{
		TotalDownloads:   stats.TotalDownloads,
		SuccessDownloads: stats.SuccessDownloads,
		FailedDownloads:  stats.FailedDownloads,
		TotalSizeBytes:   stats.TotalSize,
		TopPlatforms:     platforms,
		RecentActivity:   activities,
	}, nil
}

// GetFileInfo 获取文件信息
func (s *GRPCServer) GetFileInfo(ctx context.Context, req *pb.GetFileInfoRequest) (*pb.GetFileInfoResponse, error) {
	if req.HistoryId == 0 || req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "历史ID和用户ID不能为空")
	}

	fileInfo, err := s.historyService.GetFileInfo(ctx, req.HistoryId, req.UserId)
	if err != nil {
		log.Printf("GetFileInfo error: %v", err)
		switch err.Error() {
		case "access denied":
			return nil, status.Error(codes.PermissionDenied, "无权访问该文件")
		case "download not completed":
			return nil, status.Error(codes.FailedPrecondition, "下载尚未完成")
		case "file not found":
			return nil, status.Error(codes.NotFound, "文件不存在")
		default:
			return nil, status.Error(codes.Internal, "获取文件信息失败")
		}
	}

	return &pb.GetFileInfoResponse{
		FilePath: fileInfo.FilePath,
		FileName: fileInfo.FileName,
		FileSize: fileInfo.FileSize,
		FileHash: fileInfo.FileHash,
	}, nil
}

// ========== 代理管理 ==========

func (s *GRPCServer) CreateProxy(ctx context.Context, req *pb.CreateProxyRequest) (*pb.CreateProxyResponse, error) {
	return s.proxyHandler.CreateProxy(ctx, req)
}

func (s *GRPCServer) UpdateProxy(ctx context.Context, req *pb.UpdateProxyRequest) (*pb.UpdateProxyResponse, error) {
	return s.proxyHandler.UpdateProxy(ctx, req)
}

func (s *GRPCServer) DeleteProxy(ctx context.Context, req *pb.DeleteProxyRequest) (*pb.DeleteProxyResponse, error) {
	return s.proxyHandler.DeleteProxy(ctx, req)
}

func (s *GRPCServer) GetProxy(ctx context.Context, req *pb.GetProxyRequest) (*pb.GetProxyResponse, error) {
	return s.proxyHandler.GetProxy(ctx, req)
}

func (s *GRPCServer) ListProxies(ctx context.Context, req *pb.ListProxiesRequest) (*pb.ListProxiesResponse, error) {
	return s.proxyHandler.ListProxies(ctx, req)
}

func (s *GRPCServer) CheckProxyHealth(ctx context.Context, req *pb.CheckProxyHealthRequest) (*pb.CheckProxyHealthResponse, error) {
	return s.proxyHandler.CheckProxyHealth(ctx, req)
}

func (s *GRPCServer) GetAvailableProxy(ctx context.Context, req *pb.GetAvailableProxyRequest) (*pb.GetAvailableProxyResponse, error) {
	return s.proxyHandler.GetAvailableProxy(ctx, req)
}

func (s *GRPCServer) ReportProxyUsage(ctx context.Context, req *pb.ReportProxyUsageRequest) (*pb.ReportProxyUsageResponse, error) {
	return s.proxyHandler.ReportProxyUsage(ctx, req)
}

// ========== Cookie 管理 ==========

func (s *GRPCServer) CreateCookie(ctx context.Context, req *pb.CreateCookieRequest) (*pb.CreateCookieResponse, error) {
	return s.cookieHandler.CreateCookie(ctx, req)
}

func (s *GRPCServer) UpdateCookie(ctx context.Context, req *pb.UpdateCookieRequest) (*pb.UpdateCookieResponse, error) {
	return s.cookieHandler.UpdateCookie(ctx, req)
}

func (s *GRPCServer) DeleteCookie(ctx context.Context, req *pb.DeleteCookieRequest) (*pb.DeleteCookieResponse, error) {
	return s.cookieHandler.DeleteCookie(ctx, req)
}

func (s *GRPCServer) GetCookie(ctx context.Context, req *pb.GetCookieRequest) (*pb.GetCookieResponse, error) {
	return s.cookieHandler.GetCookie(ctx, req)
}

func (s *GRPCServer) ListCookies(ctx context.Context, req *pb.ListCookiesRequest) (*pb.ListCookiesResponse, error) {
	return s.cookieHandler.ListCookies(ctx, req)
}

func (s *GRPCServer) GetAvailableCookie(ctx context.Context, req *pb.GetAvailableCookieRequest) (*pb.GetAvailableCookieResponse, error) {
	return s.cookieHandler.GetAvailableCookie(ctx, req)
}

func (s *GRPCServer) ReportCookieUsage(ctx context.Context, req *pb.ReportCookieUsageRequest) (*pb.ReportCookieUsageResponse, error) {
	return s.cookieHandler.ReportCookieUsage(ctx, req)
}

func (s *GRPCServer) FreezeCookie(ctx context.Context, req *pb.FreezeCookieRequest) (*pb.FreezeCookieResponse, error) {
	return s.cookieHandler.FreezeCookie(ctx, req)
}
