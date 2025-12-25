package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"vasset/asset-service/internal/models"
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
