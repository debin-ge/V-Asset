package service

import (
	"context"
	"database/sql/driver"
	"encoding/json"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"

	"vasset/auth-service/internal/models"
	"vasset/auth-service/internal/repository"
	"vasset/auth-service/internal/utils"
)

func TestTokenServiceRefreshTokenInvalidatesPreviousCache(t *testing.T) {
	ctx := context.Background()
	redisServer := miniredis.RunT(t)
	redisClient := redis.NewClient(&redis.Options{Addr: redisServer.Addr()})
	defer redisClient.Close()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	jwtUtil := utils.NewJWTUtil("0123456789abcdef0123456789abcdef", 3600, 7200)
	service := NewTokenService(
		jwtUtil,
		redisClient,
		repository.NewSessionRepository(db),
		repository.NewUserRepository(db),
	)

	oldTokenHash := "old-token-hash"
	refreshToken := issueRefreshToken(t, jwtUtil, "user-1")
	expiresAt := time.Now().Add(time.Hour)
	createdAt := time.Now().Add(-time.Hour)
	lastUsedAt := time.Now().Add(-time.Minute)

	oldClaims, err := json.Marshal(utils.Claims{UserID: "user-1", Email: "user@example.com", Role: "1"})
	if err != nil {
		t.Fatalf("failed to marshal old claims: %v", err)
	}
	oldCacheKey := "auth:token:" + oldTokenHash
	redisServer.Set(oldCacheKey, string(oldClaims))

	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT id, user_id, refresh_token, token_hash, device_info, ip_address, 
		       expires_at, last_used_at, created_at
		FROM user_sessions
		WHERE refresh_token = $1
	`)).
		WithArgs(refreshToken).
		WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "refresh_token", "token_hash", "device_info", "ip_address", "expires_at", "last_used_at", "created_at"}).
			AddRow(int64(7), "user-1", refreshToken, oldTokenHash, "chrome", "127.0.0.1", expiresAt, lastUsedAt, createdAt))

	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT id, email, password_hash, nickname, avatar_url, role, status, 
		       created_at, updated_at, last_login_at
		FROM users
		WHERE id = $1
	`)).
		WithArgs("user-1").
		WillReturnRows(sqlmock.NewRows([]string{"id", "email", "password_hash", "nickname", "avatar_url", "role", "status", "created_at", "updated_at", "last_login_at"}).
			AddRow("user-1", "user@example.com", "hashed", "tester", "", models.RoleUser, models.StatusActive, createdAt, createdAt, nil))

	mock.ExpectExec(regexp.QuoteMeta(`
		UPDATE user_sessions
		SET token_hash = $1,
		    last_used_at = $2
		WHERE id = $3
	`)).
		WithArgs(sqlmock.AnyArg(), anyTime{}, int64(7)).
		WillReturnResult(sqlmock.NewResult(0, 1))

	newToken, err := service.RefreshToken(ctx, refreshToken)
	if err != nil {
		t.Fatalf("expected refresh token to succeed: %v", err)
	}
	if newToken == "" {
		t.Fatal("expected a new access token")
	}
	if redisServer.Exists(oldCacheKey) {
		t.Fatal("expected previous token cache to be deleted")
	}

	newCacheKey := "auth:token:" + utils.HashToken(newToken)
	newCachedClaims, err := redisClient.Get(ctx, newCacheKey).Result()
	if err != nil {
		t.Fatalf("expected refreshed token cache to exist: %v", err)
	}

	var claims utils.Claims
	if err := json.Unmarshal([]byte(newCachedClaims), &claims); err != nil {
		t.Fatalf("expected refreshed token cache to contain claims json: %v", err)
	}
	if claims.UserID != "user-1" || claims.Email != "user@example.com" || claims.Role != "1" {
		t.Fatalf("unexpected cached claims: %+v", claims)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func TestTokenServiceInvalidateTokenDeletesCacheAndSession(t *testing.T) {
	ctx := context.Background()
	redisServer := miniredis.RunT(t)
	redisClient := redis.NewClient(&redis.Options{Addr: redisServer.Addr()})
	defer redisClient.Close()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	service := NewTokenService(
		utils.NewJWTUtil("0123456789abcdef0123456789abcdef", 3600, 7200),
		redisClient,
		repository.NewSessionRepository(db),
		repository.NewUserRepository(db),
	)

	token := "access-token-value"
	cacheKey := "auth:token:" + utils.HashToken(token)
	redisServer.Set(cacheKey, `{"user_id":"user-1"}`)

	mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM user_sessions WHERE token_hash = $1`)).
		WithArgs(utils.HashToken(token)).
		WillReturnResult(sqlmock.NewResult(0, 1))

	if err := service.InvalidateToken(ctx, token); err != nil {
		t.Fatalf("expected token invalidation to succeed: %v", err)
	}
	if redisServer.Exists(cacheKey) {
		t.Fatal("expected token cache to be deleted")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func issueToken(t *testing.T, jwtUtil *utils.JWTUtil, userID, email, role string) string {
	t.Helper()

	token, err := jwtUtil.GenerateToken(userID, email, role)
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}
	return token
}

func issueRefreshToken(t *testing.T, jwtUtil *utils.JWTUtil, userID string) string {
	t.Helper()

	token, err := jwtUtil.GenerateRefreshToken(userID)
	if err != nil {
		t.Fatalf("failed to generate refresh token: %v", err)
	}
	return token
}

type anyTime struct{}

func (anyTime) Match(value driver.Value) bool {
	_, ok := value.(time.Time)
	return ok
}
