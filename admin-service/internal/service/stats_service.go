package service

import (
	"context"

	"vasset/admin-service/internal/models"
	pb "vasset/admin-service/proto"
)

type StatsService struct {
	authClient  pb.AuthServiceClient
	assetClient pb.AssetServiceClient
}

func NewStatsService(authClient pb.AuthServiceClient, assetClient pb.AssetServiceClient) *StatsService {
	return &StatsService{
		authClient:  authClient,
		assetClient: assetClient,
	}
}

func (s *StatsService) GetOverview(ctx context.Context) (*models.OverviewResponse, error) {
	userResp, err := s.authClient.GetPlatformUserStats(ctx, &pb.GetPlatformUserStatsRequest{})
	if err != nil {
		return nil, err
	}

	statsResp, err := s.assetClient.GetPlatformStats(ctx, &pb.GetPlatformStatsRequest{})
	if err != nil {
		return nil, err
	}

	proxiesResp, err := s.assetClient.ListProxies(ctx, &pb.ListProxiesRequest{Status: -1})
	if err != nil {
		return nil, err
	}

	var activeManualProxies int64
	for _, item := range proxiesResp.Items {
		if item.Status == 0 {
			activeManualProxies++
		}
	}

	return &models.OverviewResponse{
		TotalUsers:          userResp.TotalUsers,
		DailyActiveUsers:    statsResp.DailyActiveUsers,
		WeeklyActiveUsers:   statsResp.WeeklyActiveUsers,
		TotalDownloads:      statsResp.TotalDownloads,
		DownloadsToday:      statsResp.DownloadsToday,
		SuccessDownloads:    statsResp.SuccessDownloads,
		FailedDownloads:     statsResp.FailedDownloads,
		ActiveManualProxies: activeManualProxies,
		TotalManualProxies:  int64(len(proxiesResp.Items)),
	}, nil
}

func (s *StatsService) GetRequestTrend(ctx context.Context, granularity string, limit int32) (*models.RequestTrendResponse, error) {
	resp, err := s.assetClient.GetRequestTrend(ctx, &pb.GetRequestTrendRequest{
		Granularity: granularity,
		Limit:       limit,
	})
	if err != nil {
		return nil, err
	}

	points := make([]models.TrendPoint, 0, len(resp.Points))
	for _, point := range resp.Points {
		points = append(points, models.TrendPoint{
			Label: point.Label,
			Count: point.Count,
		})
	}

	return &models.RequestTrendResponse{
		Granularity: resp.Granularity,
		Points:      points,
	}, nil
}

func (s *StatsService) GetUserStats(ctx context.Context) (*models.UserStatsResponse, error) {
	userResp, err := s.authClient.GetPlatformUserStats(ctx, &pb.GetPlatformUserStatsRequest{})
	if err != nil {
		return nil, err
	}

	statsResp, err := s.assetClient.GetPlatformStats(ctx, &pb.GetPlatformStatsRequest{})
	if err != nil {
		return nil, err
	}

	return &models.UserStatsResponse{
		TotalUsers:        userResp.TotalUsers,
		DailyActiveUsers:  statsResp.DailyActiveUsers,
		WeeklyActiveUsers: statsResp.WeeklyActiveUsers,
	}, nil
}
