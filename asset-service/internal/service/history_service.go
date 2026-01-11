package service

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"os"

	"vasset/asset-service/internal/models"
	"vasset/asset-service/internal/repository"
)

// HistoryService 历史服务
type HistoryService struct {
	historyRepo *repository.HistoryRepository
}

// NewHistoryService 创建历史服务
func NewHistoryService(historyRepo *repository.HistoryRepository) *HistoryService {
	return &HistoryService{
		historyRepo: historyRepo,
	}
}

// GetHistory 获取历史记录
func (s *HistoryService) GetHistory(ctx context.Context, filter *models.HistoryFilter) (*models.HistoryResult, error) {
	return s.historyRepo.Query(ctx, filter)
}

// CreateHistory 创建历史记录
func (s *HistoryService) CreateHistory(ctx context.Context, history *models.DownloadHistory) (int64, error) {
	log.Printf("[HistoryService] Creating history for task %s, user %s", history.TaskID, history.UserID)
	return s.historyRepo.Create(ctx, history)
}

// DeleteHistory 删除历史记录
func (s *HistoryService) DeleteHistory(ctx context.Context, historyID int64, userID string) error {
	// 1. 获取记录信息
	record, err := s.historyRepo.GetByIDAndUserID(ctx, historyID, userID)
	if err != nil {
		if err == sql.ErrNoRows {
			return errors.New("record not found")
		}
		return err
	}

	// 2. 如果文件存在,尝试删除物理文件
	if record.FilePath.Valid && record.FilePath.String != "" && record.Status == models.StatusCompleted {
		if err := os.Remove(record.FilePath.String); err != nil && !os.IsNotExist(err) {
			log.Printf("Warning: failed to delete file %s: %v", record.FilePath.String, err)
		}
	}

	// 3. 删除数据库记录
	return s.historyRepo.Delete(ctx, historyID, userID)
}

// GetFileInfo 获取文件信息(带权限验证)
func (s *HistoryService) GetFileInfo(ctx context.Context, historyID int64, userID string) (*models.FileInfo, error) {
	// 1. 获取记录(同时验证权限)
	record, err := s.historyRepo.GetByIDAndUserID(ctx, historyID, userID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("access denied")
		}
		return nil, err
	}

	// 2. 检查状态 - 允许 StatusCompleted 和 StatusPendingCleanup (quick_download 完成后)
	if record.Status != models.StatusCompleted && record.Status != models.StatusPendingCleanup {
		return nil, errors.New("download not completed")
	}

	// 3. 检查文件是否存在
	if !record.FilePath.Valid || record.FilePath.String == "" {
		return nil, errors.New("file not found")
	}
	if _, err := os.Stat(record.FilePath.String); os.IsNotExist(err) {
		return nil, errors.New("file not found")
	}

	return &models.FileInfo{
		FilePath: record.FilePath.String,
		FileName: record.FileName.String,
		FileSize: record.FileSize.Int64,
		FileHash: record.FileHash.String,
	}, nil
}
