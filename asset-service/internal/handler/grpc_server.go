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
	cfg            *config.Config
}

// NewGRPCServer 创建 gRPC 服务器
func NewGRPCServer(
	historyService *service.HistoryService,
	quotaService *service.QuotaService,
	statsService *service.StatsService,
	cfg *config.Config,
) *GRPCServer {
	return &GRPCServer{
		historyService: historyService,
		quotaService:   quotaService,
		statsService:   statsService,
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
