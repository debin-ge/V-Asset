package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"youdlp/admin-service/internal/models"
)

type SessionService struct {
	redisClient *redis.Client
	ttl         time.Duration
}

func NewSessionService(redisClient *redis.Client, ttl time.Duration) *SessionService {
	return &SessionService{
		redisClient: redisClient,
		ttl:         ttl,
	}
}

func (s *SessionService) Create(ctx context.Context, user models.AdminUser) (*models.AdminSession, error) {
	session := &models.AdminSession{
		SessionID: uuid.NewString(),
		User:      user,
	}

	data, err := json.Marshal(session)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal session: %w", err)
	}

	key := fmt.Sprintf("admin:session:%s", session.SessionID)
	if err := s.redisClient.Set(ctx, key, data, s.ttl).Err(); err != nil {
		return nil, fmt.Errorf("failed to save session: %w", err)
	}

	return session, nil
}

func (s *SessionService) Get(ctx context.Context, sessionID string) (*models.AdminSession, error) {
	key := fmt.Sprintf("admin:session:%s", sessionID)
	data, err := s.redisClient.Get(ctx, key).Result()
	if err != nil {
		return nil, err
	}

	var session models.AdminSession
	if err := json.Unmarshal([]byte(data), &session); err != nil {
		return nil, fmt.Errorf("failed to unmarshal session: %w", err)
	}

	return &session, nil
}

func (s *SessionService) Delete(ctx context.Context, sessionID string) error {
	key := fmt.Sprintf("admin:session:%s", sessionID)
	return s.redisClient.Del(ctx, key).Err()
}
