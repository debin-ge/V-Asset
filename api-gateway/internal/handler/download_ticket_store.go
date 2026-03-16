package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const downloadTicketPrefix = "download:ticket:"

var errDownloadTicketNotFound = errors.New("download ticket not found")

type downloadTicketPayload struct {
	UserID    string `json:"user_id"`
	HistoryID int64  `json:"history_id"`
}

type downloadTicketStore interface {
	Save(ctx context.Context, ticket string, payload *downloadTicketPayload, ttl time.Duration) error
	Load(ctx context.Context, ticket string) (*downloadTicketPayload, error)
}

type redisDownloadTicketStore struct {
	client *redis.Client
}

func NewRedisDownloadTicketStore(client *redis.Client) downloadTicketStore {
	if client == nil {
		return nil
	}
	return &redisDownloadTicketStore{client: client}
}

func (s *redisDownloadTicketStore) Save(ctx context.Context, ticket string, payload *downloadTicketPayload, ttl time.Duration) error {
	if s == nil || s.client == nil {
		return errors.New("download ticket store unavailable")
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal download ticket: %w", err)
	}

	if err := s.client.Set(ctx, downloadTicketPrefix+ticket, data, ttl).Err(); err != nil {
		return fmt.Errorf("save download ticket: %w", err)
	}

	return nil
}

func (s *redisDownloadTicketStore) Load(ctx context.Context, ticket string) (*downloadTicketPayload, error) {
	if s == nil || s.client == nil {
		return nil, errors.New("download ticket store unavailable")
	}

	data, err := s.client.Get(ctx, downloadTicketPrefix+ticket).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, errDownloadTicketNotFound
		}
		return nil, fmt.Errorf("load download ticket: %w", err)
	}

	var payload downloadTicketPayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, fmt.Errorf("unmarshal download ticket: %w", err)
	}

	return &payload, nil
}
