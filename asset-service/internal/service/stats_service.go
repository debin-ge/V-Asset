package service

import (
	"context"
	"time"

	"vasset/asset-service/internal/models"
	"vasset/asset-service/internal/repository"
)

// StatsService 统计服务
type StatsService struct {
	historyRepo *repository.HistoryRepository
}

// NewStatsService 创建统计服务
func NewStatsService(historyRepo *repository.HistoryRepository) *StatsService {
	return &StatsService{
		historyRepo: historyRepo,
	}
}

// GetUserStats 获取用户统计
func (s *StatsService) GetUserStats(ctx context.Context, userID string) (*models.UserStats, error) {
	stats := &models.UserStats{}

	// 总下载数
	total, err := s.historyRepo.GetTotalCount(ctx, userID)
	if err != nil {
		return nil, err
	}
	stats.TotalDownloads = total

	// 成功下载数
	success, err := s.historyRepo.GetCountByStatus(ctx, userID, models.StatusCompleted)
	if err != nil {
		return nil, err
	}
	stats.SuccessDownloads = success

	// 失败下载数
	failed, err := s.historyRepo.GetCountByStatus(ctx, userID, models.StatusFailed)
	if err != nil {
		return nil, err
	}
	stats.FailedDownloads = failed

	// 总文件大小
	size, err := s.historyRepo.GetTotalSize(ctx, userID)
	if err != nil {
		return nil, err
	}
	stats.TotalSize = size

	// 平台排名(Top 5)
	platforms, err := s.historyRepo.GetPlatformStats(ctx, userID, 5)
	if err != nil {
		return nil, err
	}
	stats.TopPlatforms = platforms

	// 最近30天活动
	activities, err := s.historyRepo.GetDailyActivity(ctx, userID, 30)
	if err != nil {
		return nil, err
	}
	stats.RecentActivity = activities

	return stats, nil
}

// GetPlatformStats 获取平台统计
func (s *StatsService) GetPlatformStats(ctx context.Context) (*models.PlatformStats, error) {
	stats := &models.PlatformStats{}

	total, err := s.historyRepo.GetPlatformTotalCount(ctx)
	if err != nil {
		return nil, err
	}
	stats.TotalDownloads = total

	success, err := s.historyRepo.GetPlatformCountByStatus(ctx, models.StatusCompleted)
	if err != nil {
		return nil, err
	}
	stats.SuccessDownloads = success

	failed, err := s.historyRepo.GetPlatformCountByStatus(ctx, models.StatusFailed)
	if err != nil {
		return nil, err
	}
	stats.FailedDownloads = failed

	downloadsToday, err := s.historyRepo.GetPlatformDownloadsToday(ctx)
	if err != nil {
		return nil, err
	}
	stats.DownloadsToday = downloadsToday

	now := time.Now()
	dau, err := s.historyRepo.GetActiveUserCount(ctx, now.Add(-24*time.Hour))
	if err != nil {
		return nil, err
	}
	stats.DailyActiveUsers = dau

	wau, err := s.historyRepo.GetActiveUserCount(ctx, now.Add(-7*24*time.Hour))
	if err != nil {
		return nil, err
	}
	stats.WeeklyActiveUsers = wau

	return stats, nil
}

// GetRequestTrend 获取平台请求趋势
func (s *StatsService) GetRequestTrend(ctx context.Context, granularity string, limit int) ([]models.TrendPoint, error) {
	return s.historyRepo.GetRequestTrend(ctx, granularity, limit)
}
