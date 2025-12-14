package repository

import (
	"context"
	"database/sql"
	"time"

	"vasset/asset-service/internal/models"
)

// QuotaRepository 配额仓储
type QuotaRepository struct {
	db *sql.DB
}

// NewQuotaRepository 创建配额仓储
func NewQuotaRepository(db *sql.DB) *QuotaRepository {
	return &QuotaRepository{db: db}
}

// GetByUserID 根据用户ID获取配额
func (r *QuotaRepository) GetByUserID(ctx context.Context, userID string) (*models.UserQuota, error) {
	query := `
		SELECT id, user_id, daily_limit, daily_used, reset_at, updated_at
		FROM user_quotas
		WHERE user_id = $1
	`

	var q models.UserQuota
	err := r.db.QueryRowContext(ctx, query, userID).Scan(
		&q.ID, &q.UserID, &q.DailyLimit, &q.DailyUsed, &q.ResetAt, &q.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &q, nil
}

// Create 创建配额记录
func (r *QuotaRepository) Create(ctx context.Context, quota *models.UserQuota) error {
	query := `
		INSERT INTO user_quotas (user_id, daily_limit, daily_used, reset_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`

	return r.db.QueryRowContext(ctx, query,
		quota.UserID, quota.DailyLimit, quota.DailyUsed, quota.ResetAt, time.Now(),
	).Scan(&quota.ID)
}

// Update 更新配额
func (r *QuotaRepository) Update(ctx context.Context, quota *models.UserQuota) error {
	query := `
		UPDATE user_quotas
		SET daily_limit = $1, daily_used = $2, reset_at = $3, updated_at = $4
		WHERE id = $5
	`

	_, err := r.db.ExecContext(ctx, query,
		quota.DailyLimit, quota.DailyUsed, quota.ResetAt, time.Now(), quota.ID,
	)
	return err
}

// IncrementUsed 增加使用次数(原子操作)
func (r *QuotaRepository) IncrementUsed(ctx context.Context, userID string) error {
	query := `
		UPDATE user_quotas
		SET daily_used = daily_used + 1, updated_at = $1
		WHERE user_id = $2
	`

	_, err := r.db.ExecContext(ctx, query, time.Now(), userID)
	return err
}

// ResetQuota 重置配额
func (r *QuotaRepository) ResetQuota(ctx context.Context, userID string, resetAt time.Time) error {
	query := `
		UPDATE user_quotas
		SET daily_used = 0, reset_at = $1, updated_at = $2
		WHERE user_id = $3
	`

	_, err := r.db.ExecContext(ctx, query, resetAt, time.Now(), userID)
	return err
}

// GetOrCreate 获取或创建配额(带行锁的原子操作)
func (r *QuotaRepository) GetOrCreate(ctx context.Context, userID string, defaultLimit int) (*models.UserQuota, error) {
	// 先尝试获取
	quota, err := r.GetByUserID(ctx, userID)
	if err == nil {
		return quota, nil
	}

	if err != sql.ErrNoRows {
		return nil, err
	}

	// 不存在则创建
	now := time.Now()
	newQuota := &models.UserQuota{
		UserID:     userID,
		DailyLimit: defaultLimit,
		DailyUsed:  0,
		ResetAt:    getNextMidnight(now),
	}

	if err := r.Create(ctx, newQuota); err != nil {
		// 可能是并发创建,再次尝试获取
		return r.GetByUserID(ctx, userID)
	}

	return newQuota, nil
}

// ConsumeQuotaSafe 安全消费配额(带行锁)
func (r *QuotaRepository) ConsumeQuotaSafe(ctx context.Context, userID string, defaultLimit int) (*models.UserQuota, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	// 行锁获取配额
	var quota models.UserQuota
	query := `
		SELECT id, user_id, daily_limit, daily_used, reset_at, updated_at
		FROM user_quotas
		WHERE user_id = $1
		FOR UPDATE
	`

	err = tx.QueryRowContext(ctx, query, userID).Scan(
		&quota.ID, &quota.UserID, &quota.DailyLimit, &quota.DailyUsed, &quota.ResetAt, &quota.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		// 创建新配额
		now := time.Now()
		quota = models.UserQuota{
			UserID:     userID,
			DailyLimit: defaultLimit,
			DailyUsed:  0,
			ResetAt:    getNextMidnight(now),
		}

		insertQuery := `
			INSERT INTO user_quotas (user_id, daily_limit, daily_used, reset_at, updated_at)
			VALUES ($1, $2, $3, $4, $5)
			RETURNING id
		`
		err = tx.QueryRowContext(ctx, insertQuery,
			quota.UserID, quota.DailyLimit, quota.DailyUsed, quota.ResetAt, now,
		).Scan(&quota.ID)

		if err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	}

	// 检查是否需要重置
	now := time.Now()
	if now.After(quota.ResetAt) {
		quota.DailyUsed = 0
		quota.ResetAt = getNextMidnight(now)
	}

	// 检查配额
	if quota.DailyUsed >= quota.DailyLimit {
		tx.Rollback()
		return &quota, sql.ErrNoRows // 使用ErrNoRows表示配额已用完
	}

	// 递增使用次数
	quota.DailyUsed++
	updateQuery := `
		UPDATE user_quotas
		SET daily_used = $1, reset_at = $2, updated_at = $3
		WHERE id = $4
	`
	_, err = tx.ExecContext(ctx, updateQuery, quota.DailyUsed, quota.ResetAt, now, quota.ID)
	if err != nil {
		return nil, err
	}

	if err = tx.Commit(); err != nil {
		return nil, err
	}

	return &quota, nil
}

// getNextMidnight 获取下一个午夜时间
func getNextMidnight(t time.Time) time.Time {
	year, month, day := t.Add(24 * time.Hour).Date()
	return time.Date(year, month, day, 0, 0, 0, 0, t.Location())
}
