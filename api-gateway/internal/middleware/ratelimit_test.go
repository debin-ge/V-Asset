package middleware

import (
	"testing"
	"time"

	"youdlp/api-gateway/internal/config"
)

func TestMaybeCleanupRemovesExpiredUserAndIPLimiters(t *testing.T) {
	t.Parallel()

	rl := NewRateLimiter(&config.RateLimitConfig{
		GlobalRPS: 100,
		UserRPS:   10,
		Burst:     5,
	})

	userEntry := newLimiterEntry(rl.userRPS, rl.userBurst)
	userEntry.lastSeen.Store(time.Now().Add(-limiterIdleTTL - time.Minute).UnixNano())
	rl.userLimiters.Store("stale-user", userEntry)

	ipEntry := newLimiterEntry(rl.userRPS, rl.userBurst)
	ipEntry.lastSeen.Store(time.Now().Add(-limiterIdleTTL - time.Minute).UnixNano())
	rl.ipLimiters.Store("203.0.113.10", ipEntry)

	rl.requestCount.Store(limiterCleanupEvery - 1)
	rl.getUserLimiter("fresh-user")

	if _, ok := rl.userLimiters.Load("stale-user"); ok {
		t.Fatalf("expected stale user limiter to be evicted")
	}

	if _, ok := rl.ipLimiters.Load("203.0.113.10"); ok {
		t.Fatalf("expected stale ip limiter to be evicted")
	}

	if _, ok := rl.userLimiters.Load("fresh-user"); !ok {
		t.Fatalf("expected fresh user limiter to remain")
	}
}

func TestMaybeCleanupKeepsRecentlyTouchedLimiters(t *testing.T) {
	t.Parallel()

	rl := NewRateLimiter(&config.RateLimitConfig{
		GlobalRPS: 100,
		UserRPS:   10,
		Burst:     5,
	})

	entry := newLimiterEntry(rl.userRPS, rl.userBurst)
	entry.lastSeen.Store(time.Now().UnixNano())
	rl.userLimiters.Store("active-user", entry)

	rl.requestCount.Store(limiterCleanupEvery - 1)
	rl.getIPLimiter("198.51.100.2")

	if _, ok := rl.userLimiters.Load("active-user"); !ok {
		t.Fatalf("expected active user limiter to survive cleanup")
	}
}
