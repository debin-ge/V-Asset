package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"vasset/auth-service/internal/models"
	"vasset/auth-service/internal/repository"
	"vasset/auth-service/internal/utils"

	"github.com/redis/go-redis/v9"
)

// TokenService Token 服务
type TokenService struct {
	jwtUtil     *utils.JWTUtil
	redis       *redis.Client
	sessionRepo *repository.SessionRepository
	userRepo    *repository.UserRepository
}

// NewTokenService 创建 Token 服务
func NewTokenService(
	jwtUtil *utils.JWTUtil,
	redis *redis.Client,
	sessionRepo *repository.SessionRepository,
	userRepo *repository.UserRepository,
) *TokenService {
	return &TokenService{
		jwtUtil:     jwtUtil,
		redis:       redis,
		sessionRepo: sessionRepo,
		userRepo:    userRepo,
	}
}

// GenerateTokenPair 生成 Token 对(Access Token 和 Refresh Token)
func (s *TokenService) GenerateTokenPair(ctx context.Context, user *models.User, deviceInfo, ipAddress string) (accessToken, refreshToken string, err error) {
	// 生成 Access Token
	roleStr := fmt.Sprintf("%d", user.Role)
	accessToken, err = s.jwtUtil.GenerateToken(user.ID, user.Email, roleStr)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate access token: %w", err)
	}

	// 生成 Refresh Token
	refreshToken, err = s.jwtUtil.GenerateRefreshToken(user.ID)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate refresh token: %w", err)
	}

	// 创建会话记录
	session := &models.UserSession{
		UserID:       user.ID,
		RefreshToken: refreshToken,
		TokenHash:    utils.HashToken(accessToken),
		DeviceInfo:   deviceInfo,
		IPAddress:    ipAddress,
		ExpiresAt:    time.Now().Add(time.Duration(s.jwtUtil.GetRefreshTokenTTL()) * time.Second),
	}

	if err := s.sessionRepo.Create(ctx, session); err != nil {
		return "", "", fmt.Errorf("failed to create session: %w", err)
	}

	// 缓存 Token Claims
	if err := s.CacheTokenClaims(ctx, accessToken, user.ID, user.Email, roleStr); err != nil {
		// 缓存失败不影响主流程,只记录日志
		fmt.Printf("Warning: failed to cache token claims: %v\n", err)
	}

	return accessToken, refreshToken, nil
}

// VerifyToken 验证 Token
func (s *TokenService) VerifyToken(ctx context.Context, tokenString string) (*utils.Claims, error) {
	// 1. 检查 Redis 缓存
	cacheKey := fmt.Sprintf("auth:token:%s", utils.HashToken(tokenString))
	cached, err := s.redis.Get(ctx, cacheKey).Result()

	if err == nil {
		// 缓存命中
		var claims utils.Claims
		if err := json.Unmarshal([]byte(cached), &claims); err == nil {
			return &claims, nil
		}
	}

	// 2. 缓存未命中,解析 JWT
	claims, err := s.jwtUtil.ParseToken(tokenString)
	if err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	// 3. 验证用户是否存在
	user, err := s.userRepo.FindByID(ctx, claims.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to verify user: %w", err)
	}
	if user == nil {
		return nil, fmt.Errorf("user not found")
	}

	// 4. 写入缓存(TTL=5分钟)
	claimsJSON, _ := json.Marshal(claims)
	s.redis.Set(ctx, cacheKey, claimsJSON, 5*time.Minute)

	return claims, nil
}

// RefreshToken 刷新 Access Token
func (s *TokenService) RefreshToken(ctx context.Context, refreshToken string) (string, error) {
	// 1. 查询会话
	session, err := s.sessionRepo.FindByRefreshToken(ctx, refreshToken)
	if err != nil {
		return "", fmt.Errorf("failed to find session: %w", err)
	}
	if session == nil {
		return "", fmt.Errorf("invalid refresh token")
	}

	// 2. 检查是否过期
	if session.ExpiresAt.Before(time.Now()) {
		return "", fmt.Errorf("refresh token expired")
	}

	// 3. 获取用户信息
	user, err := s.userRepo.FindByID(ctx, session.UserID)
	if err != nil {
		return "", fmt.Errorf("failed to get user: %w", err)
	}
	if user == nil {
		return "", fmt.Errorf("user not found")
	}

	// 4. 生成新的 Access Token
	roleStr := fmt.Sprintf("%d", user.Role)
	newToken, err := s.jwtUtil.GenerateToken(user.ID, user.Email, roleStr)
	if err != nil {
		return "", fmt.Errorf("failed to generate new token: %w", err)
	}

	// 5. 更新会话的最后使用时间
	session.LastUsedAt = time.Now()
	s.sessionRepo.Update(ctx, session)

	// 6. 缓存新 Token
	s.CacheTokenClaims(ctx, newToken, user.ID, user.Email, roleStr)

	return newToken, nil
}

// InvalidateToken 使 Token 失效
func (s *TokenService) InvalidateToken(ctx context.Context, tokenString string) error {
	// 1. 删除 Redis 缓存
	cacheKey := fmt.Sprintf("auth:token:%s", utils.HashToken(tokenString))
	s.redis.Del(ctx, cacheKey)

	// 2. 删除数据库 session 记录
	tokenHash := utils.HashToken(tokenString)
	if err := s.sessionRepo.DeleteByTokenHash(ctx, tokenHash); err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}

	return nil
}

// CacheTokenClaims 缓存 Token Claims
func (s *TokenService) CacheTokenClaims(ctx context.Context, token string, userID string, email, role string) error {
	claims := utils.Claims{
		UserID: userID,
		Email:  email,
		Role:   role,
	}

	claimsJSON, err := json.Marshal(claims)
	if err != nil {
		return err
	}

	cacheKey := fmt.Sprintf("auth:token:%s", utils.HashToken(token))
	return s.redis.Set(ctx, cacheKey, claimsJSON, 5*time.Minute).Err()
}

// GetAccessTokenTTL 获取 Access Token 有效期
func (s *TokenService) GetAccessTokenTTL() int64 {
	return s.jwtUtil.GetAccessTokenTTL()
}
