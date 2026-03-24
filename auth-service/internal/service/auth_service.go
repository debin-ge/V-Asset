package service

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/grpc"
	assetpb "youdlp/asset-service/proto"
	"youdlp/auth-service/internal/config"
	"youdlp/auth-service/internal/models"
	"youdlp/auth-service/internal/repository"
	"youdlp/auth-service/internal/utils"

	"github.com/redis/go-redis/v9"
)

type authUserService interface {
	CreateUser(ctx context.Context, email, password, nickname string) (*models.User, error)
	GetUserByEmail(ctx context.Context, email string) (*models.User, error)
	UpdateLastLogin(ctx context.Context, userID string) error
	GetUserByID(ctx context.Context, userID string) (*models.User, error)
	UpdateNickname(ctx context.Context, userID string, nickname string) error
	UpdatePassword(ctx context.Context, userID string, passwordHash string) error
}

type welcomeCreditClient interface {
	GrantWelcomeCredit(ctx context.Context, in *assetpb.GrantWelcomeCreditRequest, opts ...grpc.CallOption) (*assetpb.GrantWelcomeCreditResponse, error)
}

// AuthService 认证服务
type AuthService struct {
	userService   authUserService
	tokenService  *TokenService
	sessionRepo   *repository.SessionRepository
	redis         *redis.Client
	sessionConfig *config.SessionConfig
	pwdConfig     *config.PasswordConfig
	welcomeCredit welcomeCreditClient
}

// NewAuthService 创建认证服务
func NewAuthService(
	userService authUserService,
	tokenService *TokenService,
	sessionRepo *repository.SessionRepository,
	redis *redis.Client,
	sessionConfig *config.SessionConfig,
	pwdConfig *config.PasswordConfig,
	welcomeCredit welcomeCreditClient,
) *AuthService {
	return &AuthService{
		userService:   userService,
		tokenService:  tokenService,
		sessionRepo:   sessionRepo,
		redis:         redis,
		sessionConfig: sessionConfig,
		pwdConfig:     pwdConfig,
		welcomeCredit: welcomeCredit,
	}
}

// Register 用户注册
func (s *AuthService) Register(ctx context.Context, email, password, nickname string) (*models.User, error) {
	user, err := s.userService.CreateUser(ctx, email, password, nickname)
	if err != nil {
		return nil, err
	}

	if s.welcomeCredit == nil {
		return nil, fmt.Errorf("failed to grant welcome credit: client not configured")
	}

	operationID := fmt.Sprintf("welcome_credit:%s", user.ID)
	if _, err := s.welcomeCredit.GrantWelcomeCredit(ctx, &assetpb.GrantWelcomeCreditRequest{
		UserId:      user.ID,
		OperationId: operationID,
	}); err != nil {
		return nil, fmt.Errorf("failed to grant welcome credit: %w", err)
	}

	return user, nil
}

// Login 用户登录
func (s *AuthService) Login(ctx context.Context, email, password, deviceInfo, ipAddress string) (accessToken, refreshToken string, user *models.User, err error) {
	// 1. 检查登录失败次数
	if err := s.CheckLoginAttempts(ctx, email); err != nil {
		return "", "", nil, err
	}

	// 2. 查询用户
	user, err = s.userService.GetUserByEmail(ctx, email)
	if err != nil {
		return "", "", nil, fmt.Errorf("failed to get user: %w", err)
	}
	if user == nil {
		s.RecordFailedLogin(ctx, email)
		return "", "", nil, fmt.Errorf("用户不存在或密码错误")
	}

	// 3. 验证密码
	if err := utils.ComparePassword(user.PasswordHash, password); err != nil {
		s.RecordFailedLogin(ctx, email)
		return "", "", nil, fmt.Errorf("用户不存在或密码错误")
	}

	// 4. 检查用户状态
	if user.Status != models.StatusActive {
		return "", "", nil, fmt.Errorf("账号已被禁用")
	}

	// 5. 检查会话数量限制
	sessionCount, err := s.sessionRepo.CountUserSessions(ctx, user.ID)
	if err == nil && sessionCount >= s.sessionConfig.MaxSessionsPerUser {
		// 删除最旧的会话
		s.sessionRepo.DeleteOldestSession(ctx, user.ID)
	}

	// 6. 生成 Token
	accessToken, refreshToken, err = s.tokenService.GenerateTokenPair(ctx, user, deviceInfo, ipAddress)
	if err != nil {
		return "", "", nil, fmt.Errorf("failed to generate token: %w", err)
	}

	// 7. 更新最后登录时间
	s.userService.UpdateLastLogin(ctx, user.ID)

	// 8. 清除登录失败记录
	s.ClearLoginAttempts(ctx, email)

	return accessToken, refreshToken, user, nil
}

// Logout 用户登出
func (s *AuthService) Logout(ctx context.Context, token string) error {
	return s.tokenService.InvalidateToken(ctx, token)
}

// CheckLoginAttempts 检查登录失败次数
func (s *AuthService) CheckLoginAttempts(ctx context.Context, email string) error {
	key := fmt.Sprintf("login:attempts:%s", email)
	attempts, err := s.redis.Get(ctx, key).Int()
	if err != nil && err != redis.Nil {
		// Redis 错误不影响登录流程
		return nil
	}

	if attempts >= 5 {
		return fmt.Errorf("登录失败次数过多,账号已锁定30分钟,请稍后重试")
	}

	return nil
}

// RecordFailedLogin 记录登录失败
func (s *AuthService) RecordFailedLogin(ctx context.Context, email string) {
	key := fmt.Sprintf("login:attempts:%s", email)
	s.redis.Incr(ctx, key)
	s.redis.Expire(ctx, key, 30*time.Minute)
}

// ClearLoginAttempts 清除登录失败记录
func (s *AuthService) ClearLoginAttempts(ctx context.Context, email string) {
	key := fmt.Sprintf("login:attempts:%s", email)
	s.redis.Del(ctx, key)
}

// UpdateProfile 更新用户信息
func (s *AuthService) UpdateProfile(ctx context.Context, userID string, nickname string) (*models.User, error) {
	// 更新昵称
	if err := s.userService.UpdateNickname(ctx, userID, nickname); err != nil {
		return nil, err
	}

	// 返回更新后的用户信息
	return s.userService.GetUserByID(ctx, userID)
}

// ChangePassword 修改密码
func (s *AuthService) ChangePassword(ctx context.Context, userID string, oldPassword, newPassword string) error {
	// 1. 获取用户信息
	user, err := s.userService.GetUserByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}
	if user == nil {
		return fmt.Errorf("用户不存在")
	}

	// 2. 验证旧密码
	if err := utils.ComparePassword(user.PasswordHash, oldPassword); err != nil {
		return fmt.Errorf("旧密码不正确")
	}

	// 3. 哈希新密码
	newHash, err := utils.HashPassword(newPassword, s.pwdConfig.BcryptCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	// 4. 更新密码
	if err := s.userService.UpdatePassword(ctx, userID, newHash); err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	return nil
}
