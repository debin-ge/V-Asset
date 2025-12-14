package service

import (
	"context"
	"fmt"
	"time"

	"vasset/auth-service/internal/config"
	"vasset/auth-service/internal/models"
	"vasset/auth-service/internal/repository"
	"vasset/auth-service/internal/utils"
)

// UserService 用户服务
type UserService struct {
	userRepo  *repository.UserRepository
	pwdConfig *config.PasswordConfig
}

// NewUserService 创建用户服务
func NewUserService(userRepo *repository.UserRepository, pwdConfig *config.PasswordConfig) *UserService {
	return &UserService{
		userRepo:  userRepo,
		pwdConfig: pwdConfig,
	}
}

// CreateUser 创建用户
func (s *UserService) CreateUser(ctx context.Context, email, password, nickname string) (*models.User, error) {
	// 1. 验证邮箱
	if err := utils.ValidateEmail(email); err != nil {
		return nil, err
	}

	// 2. 验证密码强度
	if err := utils.ValidatePasswordStrength(
		password,
		s.pwdConfig.MinLength,
		s.pwdConfig.RequireUppercase,
		s.pwdConfig.RequireLowercase,
		s.pwdConfig.RequireNumber,
		s.pwdConfig.RequireSpecial,
	); err != nil {
		return nil, err
	}

	// 3. 验证昵称
	if err := utils.ValidateNickname(nickname); err != nil {
		return nil, err
	}

	// 4. 检查邮箱是否已存在
	exists, err := s.userRepo.EmailExists(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("failed to check email: %w", err)
	}
	if exists {
		return nil, fmt.Errorf("邮箱已被注册")
	}

	// 5. 加密密码
	hashedPassword, err := utils.HashPassword(password, s.pwdConfig.BcryptCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// 6. 创建用户
	user := &models.User{
		Email:        email,
		PasswordHash: hashedPassword,
		Nickname:     nickname,
		Role:         models.RoleUser,
		Status:       models.StatusActive,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return user, nil
}

// GetUserByID 根据 ID 获取用户
func (s *UserService) GetUserByID(ctx context.Context, userID string) (*models.User, error) {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	if user == nil {
		return nil, fmt.Errorf("用户不存在")
	}
	return user, nil
}

// GetUserByEmail 根据邮箱获取用户
func (s *UserService) GetUserByEmail(ctx context.Context, email string) (*models.User, error) {
	user, err := s.userRepo.FindByEmail(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	return user, nil
}

// UpdateUser 更新用户信息
func (s *UserService) UpdateUser(ctx context.Context, user *models.User) error {
	if err := utils.ValidateNickname(user.Nickname); err != nil {
		return err
	}

	if err := s.userRepo.Update(ctx, user); err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	return nil
}

// UpdateLastLogin 更新最后登录时间
func (s *UserService) UpdateLastLogin(ctx context.Context, userID string) error {
	return s.userRepo.UpdateLastLogin(ctx, userID)
}

// UpdateNickname 更新用户昵称
func (s *UserService) UpdateNickname(ctx context.Context, userID string, nickname string) error {
	if err := utils.ValidateNickname(nickname); err != nil {
		return err
	}
	return s.userRepo.UpdateNickname(ctx, userID, nickname)
}

// UpdatePassword 更新用户密码
func (s *UserService) UpdatePassword(ctx context.Context, userID string, passwordHash string) error {
	return s.userRepo.UpdatePassword(ctx, userID, passwordHash)
}
