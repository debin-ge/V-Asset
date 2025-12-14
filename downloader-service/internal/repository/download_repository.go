package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"vasset/downloader-service/internal/models"
)

// DownloadRepository 下载记录数据访问层
type DownloadRepository struct {
	db *sql.DB
}

// NewDownloadRepository 创建下载记录仓储
func NewDownloadRepository(db *sql.DB) *DownloadRepository {
	return &DownloadRepository{db: db}
}

// Create 创建下载记录
func (r *DownloadRepository) Create(ctx context.Context, record *models.DownloadHistory) error {
	query := `
		INSERT INTO download_history (task_id, user_id, url, platform, title, mode, quality, status, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id
	`

	err := r.db.QueryRowContext(
		ctx, query,
		record.TaskID,
		record.UserID,
		record.URL,
		record.Platform,
		record.Title,
		record.Mode,
		record.Quality,
		models.StatusPending,
		time.Now(),
	).Scan(&record.ID)

	if err != nil {
		return fmt.Errorf("failed to create download record: %w", err)
	}

	return nil
}

// UpdateStatus 更新任务状态
func (r *DownloadRepository) UpdateStatus(ctx context.Context, taskID string, status int, errorMsg string) error {
	query := `
		UPDATE download_history
		SET status = $1, error_message = $2, updated_at = $3
		WHERE task_id = $4
	`

	var errMsgPtr *string
	if errorMsg != "" {
		errMsgPtr = &errorMsg
	}

	_, err := r.db.ExecContext(ctx, query, status, errMsgPtr, time.Now(), taskID)
	if err != nil {
		return fmt.Errorf("failed to update status: %w", err)
	}

	return nil
}

// UpdateProcessing 更新为处理中状态
func (r *DownloadRepository) UpdateProcessing(ctx context.Context, taskID string) error {
	query := `
		UPDATE download_history
		SET status = $1, started_at = $2
		WHERE task_id = $3
	`

	_, err := r.db.ExecContext(ctx, query, models.StatusProcessing, time.Now(), taskID)
	if err != nil {
		return fmt.Errorf("failed to update processing status: %w", err)
	}

	return nil
}

// UpdateComplete 更新为完成状态
func (r *DownloadRepository) UpdateComplete(ctx context.Context, taskID, filePath, fileName, fileHash string, fileSize int64, expireAt *time.Time) error {
	query := `
		UPDATE download_history
		SET status = $1, file_path = $2, file_name = $3, file_hash = $4, 
		    file_size = $5, expire_at = $6, completed_at = $7
		WHERE task_id = $8
	`

	status := models.StatusCompleted
	if expireAt != nil {
		status = models.StatusPendingCleanup
	}

	_, err := r.db.ExecContext(ctx, query, status, filePath, fileName, fileHash, fileSize, expireAt, time.Now(), taskID)
	if err != nil {
		return fmt.Errorf("failed to update complete status: %w", err)
	}

	return nil
}

// UpdateFailed 更新为失败状态
func (r *DownloadRepository) UpdateFailed(ctx context.Context, taskID, errorMsg string, retryCount int) error {
	query := `
		UPDATE download_history
		SET status = $1, error_message = $2, retry_count = $3
		WHERE task_id = $4
	`

	_, err := r.db.ExecContext(ctx, query, models.StatusFailed, errorMsg, retryCount, taskID)
	if err != nil {
		return fmt.Errorf("failed to update failed status: %w", err)
	}

	return nil
}

// IncrementRetry 增加重试次数
func (r *DownloadRepository) IncrementRetry(ctx context.Context, taskID string) error {
	query := `
		UPDATE download_history
		SET retry_count = retry_count + 1, status = $1
		WHERE task_id = $2
	`

	_, err := r.db.ExecContext(ctx, query, models.StatusPending, taskID)
	if err != nil {
		return fmt.Errorf("failed to increment retry count: %w", err)
	}

	return nil
}

// FindByTaskID 按任务 ID 查询
func (r *DownloadRepository) FindByTaskID(ctx context.Context, taskID string) (*models.DownloadHistory, error) {
	query := `
		SELECT id, task_id, user_id, url, platform, title, mode, quality,
		       file_path, file_name, file_size, file_hash, status, error_message,
		       retry_count, expire_at, created_at, started_at, completed_at
		FROM download_history
		WHERE task_id = $1
	`

	record := &models.DownloadHistory{}
	err := r.db.QueryRowContext(ctx, query, taskID).Scan(
		&record.ID, &record.TaskID, &record.UserID, &record.URL, &record.Platform,
		&record.Title, &record.Mode, &record.Quality, &record.FilePath, &record.FileName,
		&record.FileSize, &record.FileHash, &record.Status, &record.ErrorMessage,
		&record.RetryCount, &record.ExpireAt, &record.CreatedAt, &record.StartedAt, &record.CompletedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find by task id: %w", err)
	}

	return record, nil
}

// FindByUserID 按用户 ID 查询历史
// FindByUserID 按用户 ID 查询历史
func (r *DownloadRepository) FindByUserID(ctx context.Context, userID string, page, pageSize int, status *int) ([]*models.DownloadHistory, int64, error) {
	// 计算总数
	countQuery := `SELECT COUNT(*) FROM download_history WHERE user_id = $1`
	args := []interface{}{userID}

	if status != nil {
		countQuery += " AND status = $2"
		args = append(args, *status)
	}

	var total int64
	err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count records: %w", err)
	}

	// 查询列表
	offset := (page - 1) * pageSize
	listQuery := `
		SELECT id, task_id, user_id, url, platform, title, mode, quality,
		       file_path, file_name, file_size, file_hash, status, error_message,
		       retry_count, expire_at, created_at, started_at, completed_at
		FROM download_history
		WHERE user_id = $1
	`
	if status != nil {
		listQuery += " AND status = $2 ORDER BY created_at DESC LIMIT $3 OFFSET $4"
		args = append(args, pageSize, offset)
	} else {
		listQuery += " ORDER BY created_at DESC LIMIT $2 OFFSET $3"
		args = append(args, pageSize, offset)
	}

	rows, err := r.db.QueryContext(ctx, listQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query records: %w", err)
	}
	defer rows.Close()

	var records []*models.DownloadHistory
	for rows.Next() {
		record := &models.DownloadHistory{}
		err := rows.Scan(
			&record.ID, &record.TaskID, &record.UserID, &record.URL, &record.Platform,
			&record.Title, &record.Mode, &record.Quality, &record.FilePath, &record.FileName,
			&record.FileSize, &record.FileHash, &record.Status, &record.ErrorMessage,
			&record.RetryCount, &record.ExpireAt, &record.CreatedAt, &record.StartedAt, &record.CompletedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan record: %w", err)
		}
		records = append(records, record)
	}

	return records, total, nil
}

// FindExpiredRecords 查询待清理的记录
func (r *DownloadRepository) FindExpiredRecords(ctx context.Context, batchSize int) ([]*models.DownloadHistory, error) {
	query := `
		SELECT id, task_id, user_id, url, platform, title, mode, quality,
		       file_path, file_name, file_size, file_hash, status, error_message,
		       retry_count, expire_at, created_at, started_at, completed_at
		FROM download_history
		WHERE status = $1 AND expire_at < $2
		LIMIT $3
	`

	rows, err := r.db.QueryContext(ctx, query, models.StatusPendingCleanup, time.Now(), batchSize)
	if err != nil {
		return nil, fmt.Errorf("failed to query expired records: %w", err)
	}
	defer rows.Close()

	var records []*models.DownloadHistory
	for rows.Next() {
		record := &models.DownloadHistory{}
		err := rows.Scan(
			&record.ID, &record.TaskID, &record.UserID, &record.URL, &record.Platform,
			&record.Title, &record.Mode, &record.Quality, &record.FilePath, &record.FileName,
			&record.FileSize, &record.FileHash, &record.Status, &record.ErrorMessage,
			&record.RetryCount, &record.ExpireAt, &record.CreatedAt, &record.StartedAt, &record.CompletedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan record: %w", err)
		}
		records = append(records, record)
	}

	return records, nil
}

// MarkExpired 标记为已过期
func (r *DownloadRepository) MarkExpired(ctx context.Context, taskID string) error {
	query := `
		UPDATE download_history
		SET status = $1, file_path = NULL
		WHERE task_id = $2
	`

	_, err := r.db.ExecContext(ctx, query, models.StatusExpired, taskID)
	if err != nil {
		return fmt.Errorf("failed to mark expired: %w", err)
	}

	return nil
}
