package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"vasset/asset-service/internal/models"
)

// CookieRepository Cookie 数据访问层
type CookieRepository struct {
	db *sql.DB
}

// NewCookieRepository 创建 Cookie 仓库
func NewCookieRepository(db *sql.DB) *CookieRepository {
	return &CookieRepository{db: db}
}

// Create 创建 Cookie
func (r *CookieRepository) Create(ctx context.Context, cookie *models.Cookie) (int64, error) {
	query := `
		INSERT INTO cookies (platform, name, content, expire_at, freeze_seconds, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id`

	now := time.Now()
	var id int64
	err := r.db.QueryRowContext(ctx, query,
		cookie.Platform,
		cookie.Name,
		cookie.Content,
		cookie.ExpireAt,
		cookie.FreezeSeconds,
		now,
		now,
	).Scan(&id)

	if err != nil {
		return 0, fmt.Errorf("create cookie failed: %w", err)
	}

	return id, nil
}

// GetByID 根据 ID 获取 Cookie
func (r *CookieRepository) GetByID(ctx context.Context, id int64) (*models.Cookie, error) {
	query := `
		SELECT id, platform, name, content, expire_at, frozen_until, freeze_seconds,
		       last_used_at, use_count, success_count, fail_count, created_at, updated_at
		FROM cookies WHERE id = $1`

	cookie := &models.Cookie{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&cookie.ID,
		&cookie.Platform,
		&cookie.Name,
		&cookie.Content,
		&cookie.ExpireAt,
		&cookie.FrozenUntil,
		&cookie.FreezeSeconds,
		&cookie.LastUsedAt,
		&cookie.UseCount,
		&cookie.SuccessCount,
		&cookie.FailCount,
		&cookie.CreatedAt,
		&cookie.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get cookie by id failed: %w", err)
	}

	return cookie, nil
}

// Update 更新 Cookie
func (r *CookieRepository) Update(ctx context.Context, cookie *models.Cookie) error {
	query := `
		UPDATE cookies 
		SET name = $2, content = $3, expire_at = $4, freeze_seconds = $5, updated_at = $6
		WHERE id = $1`

	_, err := r.db.ExecContext(ctx, query,
		cookie.ID,
		cookie.Name,
		cookie.Content,
		cookie.ExpireAt,
		cookie.FreezeSeconds,
		time.Now(),
	)

	if err != nil {
		return fmt.Errorf("update cookie failed: %w", err)
	}

	return nil
}

// Delete 删除 Cookie
func (r *CookieRepository) Delete(ctx context.Context, id int64) error {
	query := `DELETE FROM cookies WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete cookie failed: %w", err)
	}
	return nil
}

// List 列表查询
func (r *CookieRepository) List(ctx context.Context, filter *models.CookieFilter) (*models.CookieResult, error) {
	var conditions []string
	var args []interface{}
	argIdx := 1
	now := time.Now()

	if filter.Platform != nil {
		conditions = append(conditions, fmt.Sprintf("platform = $%d", argIdx))
		args = append(args, *filter.Platform)
		argIdx++
	}

	// 根据 OnlyAvailable 添加过期和冷冻条件
	if filter.OnlyAvailable {
		conditions = append(conditions, fmt.Sprintf("(expire_at IS NULL OR expire_at > $%d)", argIdx))
		args = append(args, now)
		argIdx++
		conditions = append(conditions, fmt.Sprintf("(frozen_until IS NULL OR frozen_until < $%d)", argIdx))
		args = append(args, now)
		argIdx++
	} else if filter.Status != nil {
		// 根据计算状态过滤
		switch *filter.Status {
		case models.CookieStatusExpired:
			conditions = append(conditions, fmt.Sprintf("expire_at IS NOT NULL AND expire_at <= $%d", argIdx))
			args = append(args, now)
			argIdx++
		case models.CookieStatusFrozen:
			// 冷冻中且未过期
			conditions = append(conditions, fmt.Sprintf("(expire_at IS NULL OR expire_at > $%d)", argIdx))
			args = append(args, now)
			argIdx++
			conditions = append(conditions, fmt.Sprintf("frozen_until IS NOT NULL AND frozen_until > $%d", argIdx))
			args = append(args, now)
			argIdx++
		case models.CookieStatusActive:
			// 未过期且未冷冻
			conditions = append(conditions, fmt.Sprintf("(expire_at IS NULL OR expire_at > $%d)", argIdx))
			args = append(args, now)
			argIdx++
			conditions = append(conditions, fmt.Sprintf("(frozen_until IS NULL OR frozen_until < $%d)", argIdx))
			args = append(args, now)
			argIdx++
		}
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	// 查询总数
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM cookies %s", whereClause)
	var total int64
	err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, fmt.Errorf("count cookies failed: %w", err)
	}

	// 查询数据
	offset := (filter.Page - 1) * filter.PageSize
	dataQuery := fmt.Sprintf(`
		SELECT id, platform, name, content, expire_at, frozen_until, freeze_seconds,
		       last_used_at, use_count, success_count, fail_count, created_at, updated_at
		FROM cookies %s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d`, whereClause, argIdx, argIdx+1)

	args = append(args, filter.PageSize, offset)
	rows, err := r.db.QueryContext(ctx, dataQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("list cookies failed: %w", err)
	}
	defer rows.Close()

	var items []models.Cookie
	for rows.Next() {
		var c models.Cookie
		err := rows.Scan(
			&c.ID, &c.Platform, &c.Name, &c.Content,
			&c.ExpireAt, &c.FrozenUntil, &c.FreezeSeconds,
			&c.LastUsedAt, &c.UseCount, &c.SuccessCount, &c.FailCount,
			&c.CreatedAt, &c.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan cookie failed: %w", err)
		}
		items = append(items, c)
	}

	return &models.CookieResult{
		Total:    total,
		Page:     filter.Page,
		PageSize: filter.PageSize,
		Items:    items,
	}, nil
}

// Freeze 冷冻 Cookie
func (r *CookieRepository) Freeze(ctx context.Context, id int64, freezeSeconds int) (*time.Time, error) {
	frozenUntil := time.Now().Add(time.Duration(freezeSeconds) * time.Second)

	query := `
		UPDATE cookies 
		SET frozen_until = $2, updated_at = $3
		WHERE id = $1`

	_, err := r.db.ExecContext(ctx, query, id, frozenUntil, time.Now())
	if err != nil {
		return nil, fmt.Errorf("freeze cookie failed: %w", err)
	}

	return &frozenUntil, nil
}

// UpdateUsage 更新使用统计并冷冻
func (r *CookieRepository) UpdateUsage(ctx context.Context, id int64, success bool, freezeSeconds int) error {
	now := time.Now()

	var query string
	var args []interface{}

	// 当 freezeSeconds 为 0 时，不更新 frozen_until
	if freezeSeconds == 0 {
		if success {
			query = `
				UPDATE cookies 
				SET use_count = use_count + 1, success_count = success_count + 1, 
				    last_used_at = $2, updated_at = $3
				WHERE id = $1`
		} else {
			query = `
				UPDATE cookies 
				SET use_count = use_count + 1, fail_count = fail_count + 1, 
				    last_used_at = $2, updated_at = $3
				WHERE id = $1`
		}
		args = []interface{}{id, now, now}
	} else {
		// freezeSeconds > 0 时，同时更新 frozen_until
		frozenUntil := now.Add(time.Duration(freezeSeconds) * time.Second)
		if success {
			query = `
				UPDATE cookies 
				SET use_count = use_count + 1, success_count = success_count + 1, 
				    last_used_at = $2, frozen_until = $3, updated_at = $4
				WHERE id = $1`
		} else {
			query = `
				UPDATE cookies 
				SET use_count = use_count + 1, fail_count = fail_count + 1, 
				    last_used_at = $2, frozen_until = $3, updated_at = $4
				WHERE id = $1`
		}
		args = []interface{}{id, now, frozenUntil, now}
	}

	_, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update cookie usage failed: %w", err)
	}

	return nil
}

// GetAvailableCookie 获取可用 Cookie（未过期、未冷冻、使用次数最少）
func (r *CookieRepository) GetAvailableCookie(ctx context.Context, platform string) (*models.Cookie, error) {
	now := time.Now()

	query := `
		SELECT id, platform, name, content, expire_at, frozen_until, freeze_seconds,
		       last_used_at, use_count, success_count, fail_count, created_at, updated_at
		FROM cookies
		WHERE platform = $1
		  AND (expire_at IS NULL OR expire_at > $2)
		  AND (frozen_until IS NULL OR frozen_until < $3)
		ORDER BY use_count ASC, last_used_at ASC NULLS FIRST
		LIMIT 1`

	cookie := &models.Cookie{}
	err := r.db.QueryRowContext(ctx, query, platform, now, now).Scan(
		&cookie.ID, &cookie.Platform, &cookie.Name, &cookie.Content,
		&cookie.ExpireAt, &cookie.FrozenUntil, &cookie.FreezeSeconds,
		&cookie.LastUsedAt, &cookie.UseCount, &cookie.SuccessCount, &cookie.FailCount,
		&cookie.CreatedAt, &cookie.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get available cookie failed: %w", err)
	}

	return cookie, nil
}
