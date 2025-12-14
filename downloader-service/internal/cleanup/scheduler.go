package cleanup

import (
	"context"
	"log"
	"time"

	"vasset/downloader-service/internal/config"
	"vasset/downloader-service/internal/repository"
	"vasset/downloader-service/internal/storage"
)

// Scheduler 清理调度器
type Scheduler struct {
	repo        *repository.DownloadRepository
	fileManager *storage.FileManager
	interval    time.Duration
	batchSize   int
	enabled     bool
}

// NewScheduler 创建清理调度器
func NewScheduler(cfg *config.CleanupConfig, repo *repository.DownloadRepository, fileManager *storage.FileManager) *Scheduler {
	return &Scheduler{
		repo:        repo,
		fileManager: fileManager,
		interval:    time.Duration(cfg.Interval) * time.Second,
		batchSize:   cfg.BatchSize,
		enabled:     cfg.Enabled,
	}
}

// Start 启动清理调度器
func (s *Scheduler) Start(ctx context.Context) {
	if !s.enabled {
		log.Println("[Cleanup] Scheduler is disabled")
		return
	}

	log.Printf("[Cleanup] Starting scheduler, interval: %v, batch_size: %d", s.interval, s.batchSize)

	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	// 启动时先执行一次
	s.cleanup(ctx)

	for {
		select {
		case <-ticker.C:
			s.cleanup(ctx)
		case <-ctx.Done():
			log.Println("[Cleanup] Scheduler stopped")
			return
		}
	}
}

// cleanup 执行清理
func (s *Scheduler) cleanup(ctx context.Context) {
	log.Println("[Cleanup] Starting cleanup task")
	startTime := time.Now()

	// 查询待清理的记录
	records, err := s.repo.FindExpiredRecords(ctx, s.batchSize)
	if err != nil {
		log.Printf("[Cleanup] Failed to find expired records: %v", err)
		return
	}

	if len(records) == 0 {
		log.Println("[Cleanup] No expired files to clean")
		return
	}

	log.Printf("[Cleanup] Found %d expired files", len(records))

	deletedCount := 0
	failedCount := 0

	for _, record := range records {
		// 检查上下文是否取消
		select {
		case <-ctx.Done():
			log.Println("[Cleanup] Context cancelled, stopping cleanup")
			return
		default:
		}

		// 获取文件路径
		if !record.FilePath.Valid || record.FilePath.String == "" {
			// 文件路径为空,直接标记为已过期
			if err := s.repo.MarkExpired(ctx, record.TaskID); err != nil {
				log.Printf("[Cleanup] Failed to mark expired: %v", err)
				failedCount++
			} else {
				deletedCount++
			}
			continue
		}

		filePath := record.FilePath.String

		// 删除物理文件
		if err := s.fileManager.DeleteFile(filePath); err != nil {
			log.Printf("[Cleanup] Failed to delete file %s: %v", filePath, err)
			failedCount++
			continue
		}

		// 更新数据库状态
		if err := s.repo.MarkExpired(ctx, record.TaskID); err != nil {
			log.Printf("[Cleanup] Failed to mark expired: %v", err)
			failedCount++
			continue
		}

		deletedCount++
		log.Printf("[Cleanup] Cleaned up: %s", filePath)
	}

	elapsed := time.Since(startTime)
	log.Printf("[Cleanup] Completed in %v: deleted=%d, failed=%d", elapsed, deletedCount, failedCount)
}
