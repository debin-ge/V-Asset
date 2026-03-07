package service

import (
	"context"
	"fmt"

	"vasset/admin-service/internal/models"
	pb "vasset/admin-service/proto"
)

type AuthService struct {
	authClient     pb.AuthServiceClient
	sessionService *SessionService
}

func NewAuthService(authClient pb.AuthServiceClient, sessionService *SessionService) *AuthService {
	return &AuthService{
		authClient:     authClient,
		sessionService: sessionService,
	}
}

func (s *AuthService) Login(ctx context.Context, email, password, userAgent, ip string) (*models.AdminSession, error) {
	resp, err := s.authClient.Login(ctx, &pb.LoginRequest{
		Email:      email,
		Password:   password,
		DeviceInfo: userAgent,
		IpAddress:  ip,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to login: %w", err)
	}

	if resp.User == nil {
		return nil, fmt.Errorf("empty user returned from auth service")
	}
	if resp.User.Role != 99 {
		return nil, fmt.Errorf("admin access required")
	}

	return s.sessionService.Create(ctx, models.AdminUser{
		UserID:    resp.User.UserId,
		Email:     resp.User.Email,
		Nickname:  resp.User.Nickname,
		AvatarURL: resp.User.AvatarUrl,
		Role:      resp.User.Role,
		CreatedAt: resp.User.CreatedAt,
	})
}

func (s *AuthService) Logout(ctx context.Context, sessionID string) error {
	return s.sessionService.Delete(ctx, sessionID)
}

func (s *AuthService) GetCurrentUser(ctx context.Context, sessionID string) (*models.AdminUser, error) {
	session, err := s.sessionService.Get(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	return &session.User, nil
}
