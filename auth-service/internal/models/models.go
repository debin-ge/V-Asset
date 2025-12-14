package models

import "time"

// UserRole 用户角色枚举
type UserRole int

const (
	RoleUser  UserRole = 1  // 普通用户
	RoleVIP   UserRole = 2  // VIP用户
	RoleAdmin UserRole = 99 // 管理员
)

// UserStatus 用户状态枚举
type UserStatus int

const (
	StatusDisabled UserStatus = 0 // 禁用
	StatusActive   UserStatus = 1 // 正常
)

// User 用户模型
type User struct {
	ID           string     `db:"id"`
	Email        string     `db:"email"`
	PasswordHash string     `db:"password_hash"`
	Nickname     string     `db:"nickname"`
	AvatarURL    string     `db:"avatar_url"`
	Role         UserRole   `db:"role"`
	Status       UserStatus `db:"status"`
	CreatedAt    time.Time  `db:"created_at"`
	UpdatedAt    time.Time  `db:"updated_at"`
	LastLoginAt  *time.Time `db:"last_login_at"`
}

// UserSession 用户会话模型
type UserSession struct {
	ID           int64     `db:"id"`
	UserID       string    `db:"user_id"`
	RefreshToken string    `db:"refresh_token"`
	TokenHash    string    `db:"token_hash"`
	DeviceInfo   string    `db:"device_info"`
	IPAddress    string    `db:"ip_address"`
	ExpiresAt    time.Time `db:"expires_at"`
	LastUsedAt   time.Time `db:"last_used_at"`
	CreatedAt    time.Time `db:"created_at"`
}
