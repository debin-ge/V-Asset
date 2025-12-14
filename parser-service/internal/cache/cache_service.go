package cache

import (
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"vasset/parser-service/internal/utils"
)

// ParseResult 解析结果
type ParseResult struct {
	VideoID     string                   `json:"video_id"`
	Platform    string                   `json:"platform"`
	Title       string                   `json:"title"`
	Description string                   `json:"description"`
	Duration    int64                    `json:"duration"`
	Thumbnail   string                   `json:"thumbnail"`
	Author      string                   `json:"author"`
	UploadDate  string                   `json:"upload_date"`
	ViewCount   int64                    `json:"view_count"`
	Formats     []utils.NormalizedFormat `json:"formats"`
}

// Service 缓存服务
type Service struct {
	redis *redis.Client
	ttl   time.Duration
}

// NewService 创建缓存服务
func NewService(redisClient *redis.Client, ttl time.Duration) *Service {
	return &Service{
		redis: redisClient,
		ttl:   ttl,
	}
}

// Get 从缓存获取解析结果
func (s *Service) Get(ctx context.Context, url string) (*ParseResult, error) {
	key := generateCacheKey(url)

	data, err := s.redis.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return nil, utils.ErrCacheMiss
	}
	if err != nil {
		return nil, fmt.Errorf("redis get failed: %w", err)
	}

	var result ParseResult
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal cache data: %w", err)
	}

	return &result, nil
}

// Set 将解析结果写入缓存
func (s *Service) Set(ctx context.Context, url string, result *ParseResult) error {
	key := generateCacheKey(url)

	data, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("failed to marshal result: %w", err)
	}

	if err := s.redis.Set(ctx, key, data, s.ttl).Err(); err != nil {
		return fmt.Errorf("redis set failed: %w", err)
	}

	return nil
}

// Delete 删除缓存
func (s *Service) Delete(ctx context.Context, url string) error {
	key := generateCacheKey(url)
	return s.redis.Del(ctx, key).Err()
}

// generateCacheKey 生成缓存key
func generateCacheKey(url string) string {
	hash := md5.Sum([]byte(url))
	return fmt.Sprintf("parser:url:%x", hash)
}
