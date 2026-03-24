package service

import (
	"context"
	"fmt"
	"testing"

	"google.golang.org/grpc"

	assetpb "vasset/asset-service/proto"
	"vasset/auth-service/internal/config"
	"vasset/auth-service/internal/models"
)

type stubRegisterUserService struct {
	user *models.User
}

func (s *stubRegisterUserService) CreateUser(_ context.Context, _, _, _ string) (*models.User, error) {
	if s.user == nil {
		return nil, fmt.Errorf("user is required")
	}
	copy := *s.user
	return &copy, nil
}

func (s *stubRegisterUserService) GetUserByEmail(context.Context, string) (*models.User, error) {
	return nil, nil
}

func (s *stubRegisterUserService) UpdateLastLogin(context.Context, string) error {
	return nil
}

func (s *stubRegisterUserService) GetUserByID(context.Context, string) (*models.User, error) {
	return nil, nil
}

func (s *stubRegisterUserService) UpdateNickname(context.Context, string, string) error {
	return nil
}

func (s *stubRegisterUserService) UpdatePassword(context.Context, string, string) error {
	return nil
}

type stubWelcomeCreditClient struct {
	calls       int
	grantEvents int
	requests    []*assetpb.GrantWelcomeCreditRequest
	seenOps     map[string]struct{}
}

func (c *stubWelcomeCreditClient) GrantWelcomeCredit(_ context.Context, in *assetpb.GrantWelcomeCreditRequest, _ ...grpc.CallOption) (*assetpb.GrantWelcomeCreditResponse, error) {
	c.calls++
	reqCopy := &assetpb.GrantWelcomeCreditRequest{
		UserId:      in.GetUserId(),
		OperationId: in.GetOperationId(),
	}
	c.requests = append(c.requests, reqCopy)

	if c.seenOps == nil {
		c.seenOps = make(map[string]struct{})
	}

	_, seen := c.seenOps[in.GetOperationId()]
	if !seen {
		c.seenOps[in.GetOperationId()] = struct{}{}
		c.grantEvents++
	}

	return &assetpb.GrantWelcomeCreditResponse{
		Success: true,
		Granted: !seen,
	}, nil
}

func newRegisterTestAuthService(userID string, welcomeClient *stubWelcomeCreditClient) *AuthService {
	return NewAuthService(
		&stubRegisterUserService{user: &models.User{ID: userID, Email: "new@example.com", Nickname: "newbie"}},
		nil,
		nil,
		nil,
		&config.SessionConfig{MaxSessionsPerUser: 5, CleanupInterval: 3600},
		&config.PasswordConfig{BcryptCost: 10, MinLength: 8, RequireUppercase: true, RequireLowercase: true, RequireNumber: true},
		welcomeClient,
	)
}

func TestRegisterTriggersWelcomeCredit(t *testing.T) {
	client := &stubWelcomeCreditClient{}
	svc := newRegisterTestAuthService("user-1", client)

	user, err := svc.Register(context.Background(), "new@example.com", "Passw0rd", "newbie")
	if err != nil {
		t.Fatalf("Register returned error: %v", err)
	}
	if user.ID != "user-1" {
		t.Fatalf("expected user id user-1, got %q", user.ID)
	}
	if client.calls != 1 {
		t.Fatalf("expected GrantWelcomeCredit to be called once, got %d", client.calls)
	}
}

func TestRegisterUsesDeterministicWelcomeCreditOperationID(t *testing.T) {
	client := &stubWelcomeCreditClient{}
	svc := newRegisterTestAuthService("user-42", client)

	if _, err := svc.Register(context.Background(), "new@example.com", "Passw0rd", "newbie"); err != nil {
		t.Fatalf("Register returned error: %v", err)
	}

	if len(client.requests) != 1 {
		t.Fatalf("expected one welcome credit request, got %d", len(client.requests))
	}
	if client.requests[0].GetOperationId() != "welcome_credit:user-42" {
		t.Fatalf("expected deterministic operation id welcome_credit:user-42, got %q", client.requests[0].GetOperationId())
	}
}

func TestRegisterDoesNotGrantTwiceOnRetry(t *testing.T) {
	client := &stubWelcomeCreditClient{}
	svc := newRegisterTestAuthService("retry-user", client)

	if _, err := svc.Register(context.Background(), "retry@example.com", "Passw0rd", "retry"); err != nil {
		t.Fatalf("first Register returned error: %v", err)
	}
	if _, err := svc.Register(context.Background(), "retry@example.com", "Passw0rd", "retry"); err != nil {
		t.Fatalf("retry Register returned error: %v", err)
	}

	if client.calls != 2 {
		t.Fatalf("expected GrantWelcomeCredit to be invoked for each register attempt, got %d calls", client.calls)
	}
	if client.grantEvents != 1 {
		t.Fatalf("expected exactly one effective grant event after retry, got %d", client.grantEvents)
	}
	if client.requests[0].GetOperationId() != "welcome_credit:retry-user" || client.requests[1].GetOperationId() != "welcome_credit:retry-user" {
		t.Fatalf("expected same deterministic operation id on retries, got %q and %q", client.requests[0].GetOperationId(), client.requests[1].GetOperationId())
	}
}
