package service

import (
	"context"

	"vasset/downloader-service/internal/models"
	"vasset/downloader-service/internal/repository"
)

// DownloaderService 下载服务
type DownloaderService struct {
	repo *repository.DownloadRepository
}

// NewDownloaderService 创建下载服务
func NewDownloaderService(repo *repository.DownloadRepository) *DownloaderService {
	return &DownloaderService{repo: repo}
}

// GetTaskStatus 获取任务状态
func (s *DownloaderService) GetTaskStatus(ctx context.Context, taskID string) (*models.DownloadHistory, error) {
	return s.repo.FindByTaskID(ctx, taskID)
}

// GetDownloadHistory 获取用户下载历史
func (s *DownloaderService) GetDownloadHistory(ctx context.Context, userID string, page, pageSize int, status *int) ([]*models.DownloadHistory, int64, error) {
	return s.repo.FindByUserID(ctx, userID, page, pageSize, status)
}

// CancelTask 取消任务
func (s *DownloaderService) CancelTask(ctx context.Context, taskID string, userID string) (bool, error) {
	// 获取任务
	task, err := s.repo.FindByTaskID(ctx, taskID)
	if err != nil {
		return false, err
	}

	if task == nil {
		return false, nil
	}

	// 检查权限
	if task.UserID != userID {
		return false, nil
	}

	// 只有待处理状态的任务可以取消
	if task.Status != models.StatusPending {
		return false, nil
	}

	// 更新为失败状态
	err = s.repo.UpdateStatus(ctx, taskID, models.StatusFailed, "Task cancelled by user")
	if err != nil {
		return false, err
	}

	return true, nil
}
