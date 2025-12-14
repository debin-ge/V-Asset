package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"vasset/auth-service/internal/models"
)

// SessionRepository 会话数据访问层
type SessionRepository struct {
	db *sql.DB
}

// NewSessionRepository 创建会话仓储
func NewSessionRepository(db *sql.DB) *SessionRepository {
	return &SessionRepository{db: db}
}

// Create 创建会话
func (r *SessionRepository) Create(ctx context.Context, session *models.UserSession) error {
	query := `
		INSERT INTO user_sessions (user_id, refresh_token, token_hash, device_info, ip_address, expires_at, last_used_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id
	`

	now := time.Now()

	// 处理空 IP 地址：INET 类型不接受空字符串，需要传入 nil
	var ipAddress interface{}
	if session.IPAddress != "" {
		ipAddress = session.IPAddress
	}

	err := r.db.QueryRowContext(
		ctx, query,
		session.UserID,
		session.RefreshToken,
		session.TokenHash,
		session.DeviceInfo,
		ipAddress,
		session.ExpiresAt,
		now,
		now,
	).Scan(&session.ID)

	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}

	session.LastUsedAt = now
	session.CreatedAt = now
	return nil
}

// FindByRefreshToken 根据 Refresh Token 查询会话
func (r *SessionRepository) FindByRefreshToken(ctx context.Context, refreshToken string) (*models.UserSession, error) {
	query := `
		SELECT id, user_id, refresh_token, token_hash, device_info, ip_address, 
		       expires_at, last_used_at, created_at
		FROM user_sessions
		WHERE refresh_token = $1
	`

	session := &models.UserSession{}
	var ipAddress sql.NullString // 处理可能为 NULL 的 IP 地址

	err := r.db.QueryRowContext(ctx, query, refreshToken).Scan(
		&session.ID,
		&session.UserID,
		&session.RefreshToken,
		&session.TokenHash,
		&session.DeviceInfo,
		&ipAddress,
		&session.ExpiresAt,
		&session.LastUsedAt,
		&session.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find session by refresh token: %w", err)
	}

	// 转换 NullString 为普通字符串
	if ipAddress.Valid {
		session.IPAddress = ipAddress.String
	}

	return session, nil
}

// Update 更新会话
func (r *SessionRepository) Update(ctx context.Context, session *models.UserSession) error {
	query := `
		UPDATE user_sessions
		SET last_used_at = $1
		WHERE id = $2
	`

	_, err := r.db.ExecContext(ctx, query, session.LastUsedAt, session.ID)
	if err != nil {
		return fmt.Errorf("failed to update session: %w", err)
	}

	return nil
}

// DeleteByTokenHash 根据 Token Hash 删除会话
func (r *SessionRepository) DeleteByTokenHash(ctx context.Context, tokenHash string) error {
	query := `DELETE FROM user_sessions WHERE token_hash = $1`

	_, err := r.db.ExecContext(ctx, query, tokenHash)
	if err != nil {
		return fmt.Errorf("failed to delete session by token hash: %w", err)
	}

	return nil
}

// DeleteExpiredSessions 删除过期会话
func (r *SessionRepository) DeleteExpiredSessions(ctx context.Context) error {
	query := `DELETE FROM user_sessions WHERE expires_at < $1`

	result, err := r.db.ExecContext(ctx, query, time.Now())
	if err != nil {
		return fmt.Errorf("failed to delete expired sessions: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows > 0 {
		fmt.Printf("Deleted %d expired sessions\n", rows)
	}

	return nil
}

// CountUserSessions 统计用户会话数
func (r *SessionRepository) CountUserSessions(ctx context.Context, userID string) (int, error) {
	query := `SELECT COUNT(*) FROM user_sessions WHERE user_id = $1 AND expires_at > $2`

	var count int
	err := r.db.QueryRowContext(ctx, query, userID, time.Now()).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count user sessions: %w", err)
	}

	return count, nil
}

// DeleteOldestSession 删除用户最旧的会话
func (r *SessionRepository) DeleteOldestSession(ctx context.Context, userID string) error {
	query := `
		DELETE FROM user_sessions
		WHERE id = (
			SELECT id FROM user_sessions
			WHERE user_id = $1
			ORDER BY created_at ASC
			LIMIT 1
		)
	`

	_, err := r.db.ExecContext(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("failed to delete oldest session: %w", err)
	}

	return nil
}
