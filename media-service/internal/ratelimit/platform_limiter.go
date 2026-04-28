package ratelimit

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	StageParse    = "parse"
	StageDownload = "download"
)

type PlatformLimiter struct {
	redis          *redis.Client
	parsePerMin    int64
	downloadPerMin int64
}

func NewPlatformLimiter(redisClient *redis.Client) *PlatformLimiter {
	return &PlatformLimiter{
		redis:          redisClient,
		parsePerMin:    30,
		downloadPerMin: 10,
	}
}

func (l *PlatformLimiter) Allow(ctx context.Context, platform, stage string) (bool, error) {
	if l == nil || l.redis == nil || platform == "" {
		return true, nil
	}

	limit := l.limitFor(stage)
	if limit <= 0 {
		return true, nil
	}

	now := time.Now().UTC()
	key := fmt.Sprintf("risk:platform:%s:%s:%s", platform, stage, now.Format("200601021504"))
	count, err := l.redis.Incr(ctx, key).Result()
	if err != nil {
		return true, err
	}
	if count == 1 {
		_ = l.redis.Expire(ctx, key, 2*time.Minute).Err()
	}
	return count <= limit, nil
}

func (l *PlatformLimiter) limitFor(stage string) int64 {
	switch stage {
	case StageParse:
		return l.parsePerMin
	case StageDownload:
		return l.downloadPerMin
	default:
		return 0
	}
}
