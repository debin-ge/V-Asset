package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"youdlp/asset-service/internal/models"
)

// HistoryRepository 历史记录仓储
type HistoryRepository struct {
	db *sql.DB
}

// NewHistoryRepository 创建历史记录仓储
func NewHistoryRepository(db *sql.DB) *HistoryRepository {
	return &HistoryRepository{db: db}
}

// GetByID 根据ID获取历史记录
func (r *HistoryRepository) GetByID(ctx context.Context, id int64) (*models.DownloadHistory, error) {
	query := `
		SELECT id, task_id, user_id, url, platform, title, mode, quality,
		       file_size, file_path, file_name, file_hash, status, error_message,
		       created_at, started_at, completed_at
		FROM download_history
		WHERE id = $1
	`

	var h models.DownloadHistory
	var completedAt sql.NullTime

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&h.ID, &h.TaskID, &h.UserID, &h.URL, &h.Platform, &h.Title,
		&h.Mode, &h.Quality, &h.FileSize, &h.FilePath, &h.FileName,
		&h.FileHash, &h.Status, &h.ErrorMessage, &h.CreatedAt, &h.StartedAt, &completedAt,
	)

	if err != nil {
		return nil, err
	}

	if completedAt.Valid {
		h.CompletedAt = &completedAt.Time
	}

	return &h, nil
}

// GetByIDAndUserID 根据ID和用户ID获取历史记录(权限校验)
func (r *HistoryRepository) GetByIDAndUserID(ctx context.Context, id int64, userID string) (*models.DownloadHistory, error) {
	query := `
		SELECT id, task_id, user_id, url, platform, title, mode, quality, 
		       file_size, file_path, file_name, file_hash, status, error_message, 
		       created_at, started_at, completed_at
		FROM download_history
		WHERE id = $1 AND user_id = $2
	`

	var h models.DownloadHistory
	var completedAt sql.NullTime

	err := r.db.QueryRowContext(ctx, query, id, userID).Scan(
		&h.ID, &h.TaskID, &h.UserID, &h.URL, &h.Platform, &h.Title,
		&h.Mode, &h.Quality, &h.FileSize, &h.FilePath, &h.FileName,
		&h.FileHash, &h.Status, &h.ErrorMessage, &h.CreatedAt, &h.StartedAt, &completedAt,
	)

	if err != nil {
		return nil, err
	}

	if completedAt.Valid {
		h.CompletedAt = &completedAt.Time
	}

	return &h, nil
}

// GetByTaskIDAndUserID 根据任务ID和用户ID获取历史记录(权限校验)
func (r *HistoryRepository) GetByTaskIDAndUserID(ctx context.Context, taskID, userID string) (*models.DownloadHistory, error) {
	query := `
			SELECT id, task_id, user_id, url, platform, title, mode, quality,
			       file_size, file_path, file_name, file_hash, status, error_message,
			       created_at, started_at, completed_at
			FROM download_history
			WHERE task_id = $1 AND user_id = $2
		`

	var h models.DownloadHistory
	var completedAt sql.NullTime

	err := r.db.QueryRowContext(ctx, query, taskID, userID).Scan(
		&h.ID, &h.TaskID, &h.UserID, &h.URL, &h.Platform, &h.Title,
		&h.Mode, &h.Quality, &h.FileSize, &h.FilePath, &h.FileName,
		&h.FileHash, &h.Status, &h.ErrorMessage, &h.CreatedAt, &h.StartedAt, &completedAt,
	)
	if err != nil {
		return nil, err
	}

	if completedAt.Valid {
		h.CompletedAt = &completedAt.Time
	}

	return &h, nil
}

// Create 创建历史记录
func (r *HistoryRepository) Create(ctx context.Context, history *models.DownloadHistory) (int64, error) {
	query := `
		INSERT INTO download_history (
			task_id, user_id, url, platform, title, mode, quality, 
			thumbnail, duration, author, status, created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12
		) RETURNING id
	`

	var id int64
	err := r.db.QueryRowContext(ctx, query,
		history.TaskID, history.UserID, history.URL, history.Platform,
		history.Title, history.Mode, history.Quality, history.Thumbnail,
		history.Duration, history.Author, history.Status, time.Now(),
	).Scan(&id)

	if err != nil {
		return 0, fmt.Errorf("failed to create history: %w", err)
	}

	return id, nil
}

// UpdateCompletionByTaskID 按任务 ID 更新完成状态和文件信息
func (r *HistoryRepository) UpdateCompletionByTaskID(ctx context.Context, taskID string, status models.HistoryStatus, fileInfo *models.FileInfo) error {
	query := `
		UPDATE download_history
		SET status = $1,
		    file_path = $2,
		    file_name = $3,
		    file_size = $4,
		    file_hash = $5,
		    error_message = NULL,
		    completed_at = $6
		WHERE task_id = $7
	`

	result, err := r.db.ExecContext(
		ctx,
		query,
		status,
		fileInfo.FilePath,
		fileInfo.FileName,
		fileInfo.FileSize,
		fileInfo.FileHash,
		time.Now(),
		taskID,
	)
	if err != nil {
		return fmt.Errorf("failed to update completion status: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to inspect completion update result: %w", err)
	}
	if rows == 0 {
		return sql.ErrNoRows
	}

	return nil
}

// UpdateFailureByTaskID 按任务 ID 更新失败状态
func (r *HistoryRepository) UpdateFailureByTaskID(ctx context.Context, taskID string, errorMessage string) error {
	query := `
		UPDATE download_history
		SET status = $1,
		    error_message = $2
		WHERE task_id = $3
	`

	result, err := r.db.ExecContext(ctx, query, models.StatusFailed, errorMessage, taskID)
	if err != nil {
		return fmt.Errorf("failed to update failure status: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to inspect failure update result: %w", err)
	}
	if rows == 0 {
		return sql.ErrNoRows
	}

	return nil
}

// Query 查询历史记录(支持过滤、分页、排序)
func (r *HistoryRepository) Query(ctx context.Context, filter *models.HistoryFilter) (*models.HistoryResult, error) {
	// 构建WHERE子句
	conditions := []string{"user_id = $1"}
	args := []interface{}{filter.UserID}
	argIndex := 2

	if filter.Status != nil {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argIndex))
		args = append(args, *filter.Status)
		argIndex++
	}

	if filter.Platform != nil && *filter.Platform != "" {
		conditions = append(conditions, fmt.Sprintf("platform = $%d", argIndex))
		args = append(args, *filter.Platform)
		argIndex++
	}

	if filter.StartDate != nil {
		conditions = append(conditions, fmt.Sprintf("created_at >= $%d", argIndex))
		args = append(args, *filter.StartDate)
		argIndex++
	}

	if filter.EndDate != nil {
		conditions = append(conditions, fmt.Sprintf("created_at <= $%d", argIndex))
		args = append(args, *filter.EndDate)
		argIndex++
	}

	whereClause := strings.Join(conditions, " AND ")

	// 计算总数
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM download_history WHERE %s", whereClause)
	var total int64
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, fmt.Errorf("failed to count records: %w", err)
	}

	// 构建排序
	sortBy := "created_at"
	if filter.SortBy != "" {
		// 防止SQL注入,只允许特定字段
		allowedSorts := map[string]bool{
			"created_at": true,
			"file_size":  true,
			"status":     true,
			"platform":   true,
		}
		if allowedSorts[filter.SortBy] {
			sortBy = filter.SortBy
		}
	}

	sortOrder := "DESC"
	if filter.SortOrder == "asc" {
		sortOrder = "ASC"
	}

	// 计算分页
	page := filter.Page
	if page < 1 {
		page = 1
	}
	pageSize := filter.PageSize
	if pageSize < 1 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	// 查询数据
	dataQuery := fmt.Sprintf(`
		SELECT id, task_id, user_id, url, platform, title, mode, quality, 
		       file_size, file_path, file_name, file_hash, status, error_message, 
		       created_at, started_at, completed_at
		FROM download_history
		WHERE %s
		ORDER BY %s %s
		LIMIT $%d OFFSET $%d
	`, whereClause, sortBy, sortOrder, argIndex, argIndex+1)

	args = append(args, pageSize, offset)

	rows, err := r.db.QueryContext(ctx, dataQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query records: %w", err)
	}
	defer rows.Close()

	var items []models.DownloadHistory
	for rows.Next() {
		var h models.DownloadHistory
		var completedAt sql.NullTime

		if err := rows.Scan(
			&h.ID, &h.TaskID, &h.UserID, &h.URL, &h.Platform, &h.Title,
			&h.Mode, &h.Quality, &h.FileSize, &h.FilePath, &h.FileName,
			&h.FileHash, &h.Status, &h.ErrorMessage, &h.CreatedAt, &h.StartedAt, &completedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan record: %w", err)
		}

		if completedAt.Valid {
			h.CompletedAt = &completedAt.Time
		}

		items = append(items, h)
	}

	return &models.HistoryResult{
		Total:    total,
		Page:     page,
		PageSize: pageSize,
		Items:    items,
	}, nil
}

// Delete 删除历史记录
func (r *HistoryRepository) Delete(ctx context.Context, id int64, userID string) error {
	query := `DELETE FROM download_history WHERE id = $1 AND user_id = $2`
	result, err := r.db.ExecContext(ctx, query, id, userID)
	if err != nil {
		return err
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return sql.ErrNoRows
	}

	return nil
}

// GetTotalCount 获取用户总下载数
func (r *HistoryRepository) GetTotalCount(ctx context.Context, userID string) (int64, error) {
	var count int64
	query := `SELECT COUNT(*) FROM download_history WHERE user_id = $1`
	err := r.db.QueryRowContext(ctx, query, userID).Scan(&count)
	return count, err
}

// GetCountByStatus 按状态统计
func (r *HistoryRepository) GetCountByStatus(ctx context.Context, userID string, status models.HistoryStatus) (int64, error) {
	var count int64
	query := `SELECT COUNT(*) FROM download_history WHERE user_id = $1 AND status = $2`
	err := r.db.QueryRowContext(ctx, query, userID, status).Scan(&count)
	return count, err
}

// GetTotalSize 获取总文件大小
func (r *HistoryRepository) GetTotalSize(ctx context.Context, userID string) (int64, error) {
	var size sql.NullInt64
	query := `SELECT COALESCE(SUM(file_size), 0) FROM download_history WHERE user_id = $1 AND status = $2`
	err := r.db.QueryRowContext(ctx, query, userID, models.StatusCompleted).Scan(&size)
	if err != nil {
		return 0, err
	}
	return size.Int64, nil
}

// GetPlatformStats 获取平台统计
func (r *HistoryRepository) GetPlatformStats(ctx context.Context, userID string, limit int) ([]models.PlatformStat, error) {
	query := `
		SELECT platform, COUNT(*) as count
		FROM download_history
		WHERE user_id = $1
		GROUP BY platform
		ORDER BY count DESC
		LIMIT $2
	`

	rows, err := r.db.QueryContext(ctx, query, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stats []models.PlatformStat
	for rows.Next() {
		var s models.PlatformStat
		if err := rows.Scan(&s.Platform, &s.Count); err != nil {
			return nil, err
		}
		stats = append(stats, s)
	}

	return stats, nil
}

// GetDailyActivity 获取每日活动统计
func (r *HistoryRepository) GetDailyActivity(ctx context.Context, userID string, days int) ([]models.DailyActivity, error) {
	startDate := time.Now().AddDate(0, 0, -days)

	query := `
		SELECT DATE(created_at) as date, COUNT(*) as count
		FROM download_history
		WHERE user_id = $1 AND created_at >= $2
		GROUP BY DATE(created_at)
		ORDER BY date DESC
	`

	rows, err := r.db.QueryContext(ctx, query, userID, startDate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var activities []models.DailyActivity
	for rows.Next() {
		var a models.DailyActivity
		var date time.Time
		if err := rows.Scan(&date, &a.Count); err != nil {
			return nil, err
		}
		a.Date = date.Format("2006-01-02")
		activities = append(activities, a)
	}

	return activities, nil
}

// GetPlatformTotalCount 获取平台总下载请求数
func (r *HistoryRepository) GetPlatformTotalCount(ctx context.Context) (int64, error) {
	var count int64
	query := `SELECT COUNT(*) FROM download_history`
	err := r.db.QueryRowContext(ctx, query).Scan(&count)
	return count, err
}

// GetPlatformCountByStatus 获取平台按状态统计
func (r *HistoryRepository) GetPlatformCountByStatus(ctx context.Context, status models.HistoryStatus) (int64, error) {
	var count int64
	query := `SELECT COUNT(*) FROM download_history WHERE status = $1`
	err := r.db.QueryRowContext(ctx, query, status).Scan(&count)
	return count, err
}

// GetPlatformDownloadsToday 获取平台今日下载请求数
func (r *HistoryRepository) GetPlatformDownloadsToday(ctx context.Context) (int64, error) {
	var count int64
	query := `SELECT COUNT(*) FROM download_history WHERE created_at >= CURRENT_DATE`
	err := r.db.QueryRowContext(ctx, query).Scan(&count)
	return count, err
}

func (r *HistoryRepository) GetDashboardDownloads(ctx context.Context) (models.DashboardDownloads, error) {
	var downloads models.DashboardDownloads
	query := `
		SELECT
			COUNT(*),
			COUNT(*) FILTER (WHERE created_at >= CURRENT_DATE),
			COUNT(*) FILTER (WHERE status = $1),
			COUNT(*) FILTER (WHERE status = $2)
		FROM download_history`
	if err := r.db.QueryRowContext(ctx, query, models.StatusCompleted, models.StatusFailed).Scan(
		&downloads.Total,
		&downloads.TodayTotal,
		&downloads.SuccessTotal,
		&downloads.FailedTotal,
	); err != nil {
		return models.DashboardDownloads{}, fmt.Errorf("failed to query dashboard downloads: %w", err)
	}

	if downloads.Total > 0 {
		downloads.SuccessRate = float64(downloads.SuccessTotal) / float64(downloads.Total)
		downloads.FailureRate = float64(downloads.FailedTotal) / float64(downloads.Total)
	}
	return downloads, nil
}

// GetActiveUserCount 获取活跃用户数
func (r *HistoryRepository) GetActiveUserCount(ctx context.Context, since time.Time) (int64, error) {
	var count int64
	query := `
		SELECT COUNT(DISTINCT user_id)
		FROM download_history
		WHERE user_id <> '' AND created_at >= $1
	`
	err := r.db.QueryRowContext(ctx, query, since).Scan(&count)
	return count, err
}

// GetRequestTrend 获取平台请求趋势
func (r *HistoryRepository) GetRequestTrend(ctx context.Context, granularity string, limit int) ([]models.TrendPoint, error) {
	if limit <= 0 {
		limit = 7
	}

	var (
		query string
		args  []interface{}
	)

	switch granularity {
	case "hour":
		query = `
			WITH series AS (
				SELECT generate_series(
					DATE_TRUNC('hour', NOW()) - (($1::int - 1) * INTERVAL '1 hour'),
					DATE_TRUNC('hour', NOW()),
					INTERVAL '1 hour'
				) AS bucket
			),
			aggregated AS (
				SELECT
					DATE_TRUNC('hour', created_at) AS bucket,
					COUNT(*) AS total_count,
					COUNT(*) FILTER (WHERE status = $2) AS success_count,
					COUNT(*) FILTER (WHERE status = $3) AS failed_count
				FROM download_history
				WHERE created_at >= DATE_TRUNC('hour', NOW()) - (($1::int - 1) * INTERVAL '1 hour')
				  AND created_at < DATE_TRUNC('hour', NOW()) + INTERVAL '1 hour'
				GROUP BY bucket
			)
			SELECT
				TO_CHAR(series.bucket, 'YYYY-MM-DD HH24:00') AS label,
				COALESCE(aggregated.total_count, 0) AS total_count,
				COALESCE(aggregated.success_count, 0) AS success_count,
				COALESCE(aggregated.failed_count, 0) AS failed_count
			FROM series
			LEFT JOIN aggregated ON aggregated.bucket = series.bucket
			ORDER BY series.bucket ASC
		`
		args = []interface{}{limit, models.StatusCompleted, models.StatusFailed}
	default:
		query = `
			WITH series AS (
				SELECT generate_series(
					CURRENT_DATE - (($1::int - 1) * INTERVAL '1 day'),
					CURRENT_DATE,
					INTERVAL '1 day'
				) AS bucket
			),
			aggregated AS (
				SELECT
					DATE_TRUNC('day', created_at) AS bucket,
					COUNT(*) AS total_count,
					COUNT(*) FILTER (WHERE status = $2) AS success_count,
					COUNT(*) FILTER (WHERE status = $3) AS failed_count
				FROM download_history
				WHERE created_at >= CURRENT_DATE - (($1::int - 1) * INTERVAL '1 day')
				  AND created_at < CURRENT_DATE + INTERVAL '1 day'
				GROUP BY bucket
			)
			SELECT
				TO_CHAR(series.bucket, 'YYYY-MM-DD') AS label,
				COALESCE(aggregated.total_count, 0) AS total_count,
				COALESCE(aggregated.success_count, 0) AS success_count,
				COALESCE(aggregated.failed_count, 0) AS failed_count
			FROM series
			LEFT JOIN aggregated ON aggregated.bucket = series.bucket
			ORDER BY series.bucket ASC
		`
		args = []interface{}{limit, models.StatusCompleted, models.StatusFailed}
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query request trend: %w", err)
	}
	defer rows.Close()

	points := make([]models.TrendPoint, 0, limit)
	for rows.Next() {
		var point models.TrendPoint
		if err := rows.Scan(&point.Label, &point.TotalCount, &point.SuccessCount, &point.FailedCount); err != nil {
			return nil, fmt.Errorf("failed to scan request trend: %w", err)
		}
		point.Count = point.TotalCount
		if point.TotalCount > 0 {
			point.SuccessRate = float64(point.SuccessCount) / float64(point.TotalCount)
		}
		points = append(points, point)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate request trend: %w", err)
	}

	return points, nil
}
