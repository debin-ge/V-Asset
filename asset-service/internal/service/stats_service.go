package service

import (
	"context"

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
