package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"

	"vasset/auth-service/internal/models"
)

// UserRepository 用户数据访问层
type UserRepository struct {
	db *sql.DB
}

// NewUserRepository 创建用户仓储
func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

// Create 创建用户
func (r *UserRepository) Create(ctx context.Context, user *models.User) error {
	// 生成UUID
	user.ID = uuid.New().String()

	query := `
		INSERT INTO users (id, email, password_hash, nickname, avatar_url, role, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	now := time.Now()
	_, err := r.db.ExecContext(
		ctx, query,
		user.ID,
		user.Email,
		user.PasswordHash,
		user.Nickname,
		user.AvatarURL,
		user.Role,
		user.Status,
		now,
		now,
	)

	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	user.CreatedAt = now
	user.UpdatedAt = now
	return nil
}

// FindByID 根据 ID 查询用户
func (r *UserRepository) FindByID(ctx context.Context, id string) (*models.User, error) {
	query := `
		SELECT id, email, password_hash, nickname, avatar_url, role, status, 
		       created_at, updated_at, last_login_at
		FROM users
		WHERE id = $1
	`

	user := &models.User{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.Nickname,
		&user.AvatarURL,
		&user.Role,
		&user.Status,
		&user.CreatedAt,
		&user.UpdatedAt,
		&user.LastLoginAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find user by id: %w", err)
	}

	return user, nil
}

// FindByEmail 根据邮箱查询用户
func (r *UserRepository) FindByEmail(ctx context.Context, email string) (*models.User, error) {
	query := `
		SELECT id, email, password_hash, nickname, avatar_url, role, status, 
		       created_at, updated_at, last_login_at
		FROM users
		WHERE email = $1
	`

	user := &models.User{}
	err := r.db.QueryRowContext(ctx, query, email).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.Nickname,
		&user.AvatarURL,
		&user.Role,
		&user.Status,
		&user.CreatedAt,
		&user.UpdatedAt,
		&user.LastLoginAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find user by email: %w", err)
	}

	return user, nil
}

// EmailExists 检查邮箱是否存在
func (r *UserRepository) EmailExists(ctx context.Context, email string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM users WHERE email = $1)`

	var exists bool
	err := r.db.QueryRowContext(ctx, query, email).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check email existence: %w", err)
	}

	return exists, nil
}

// Update 更新用户信息
func (r *UserRepository) Update(ctx context.Context, user *models.User) error {
	query := `
		UPDATE users
		SET nickname = $1, avatar_url = $2, updated_at = $3
		WHERE id = $4
	`

	_, err := r.db.ExecContext(ctx, query, user.Nickname, user.AvatarURL, time.Now(), user.ID)
	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	return nil
}

// UpdateLastLogin 更新最后登录时间
func (r *UserRepository) UpdateLastLogin(ctx context.Context, userID string) error {
	query := `UPDATE users SET last_login_at = $1 WHERE id = $2`

	_, err := r.db.ExecContext(ctx, query, time.Now(), userID)
	if err != nil {
		return fmt.Errorf("failed to update last login: %w", err)
	}

	return nil
}

// UpdateNickname 更新用户昵称
func (r *UserRepository) UpdateNickname(ctx context.Context, userID string, nickname string) error {
	query := `UPDATE users SET nickname = $1, updated_at = $2 WHERE id = $3`

	_, err := r.db.ExecContext(ctx, query, nickname, time.Now(), userID)
	if err != nil {
		return fmt.Errorf("failed to update nickname: %w", err)
	}

	return nil
}

// UpdatePassword 更新用户密码
func (r *UserRepository) UpdatePassword(ctx context.Context, userID string, passwordHash string) error {
	query := `UPDATE users SET password_hash = $1, updated_at = $2 WHERE id = $3`

	_, err := r.db.ExecContext(ctx, query, passwordHash, time.Now(), userID)
	if err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	return nil
}
