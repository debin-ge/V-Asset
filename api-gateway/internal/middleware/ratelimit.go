package middleware

import (
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"

	"youdlp/api-gateway/internal/config"
)

// RateLimiter 限流器
type RateLimiter struct {
	globalLimiter *rate.Limiter
	userLimiters  sync.Map
	ipLimiters    sync.Map
	userRPS       rate.Limit
	userBurst     int
	requestCount  atomic.Uint64
}

type limiterEntry struct {
	limiter  *rate.Limiter
	lastSeen atomic.Int64
}

const (
	limiterIdleTTL      = 15 * time.Minute
	limiterCleanupEvery = 256
)

// NewRateLimiter 创建限流器
func NewRateLimiter(cfg *config.RateLimitConfig) *RateLimiter {
	return &RateLimiter{
		globalLimiter: rate.NewLimiter(rate.Limit(cfg.GlobalRPS), cfg.Burst*2),
		userRPS:       rate.Limit(cfg.UserRPS),
		userBurst:     cfg.Burst,
	}
}

// getUserLimiter 获取用户限流器
func (rl *RateLimiter) getUserLimiter(userID string) *rate.Limiter {
	return rl.getLimiter(&rl.userLimiters, userID)
}

func (rl *RateLimiter) getIPLimiter(clientIP string) *rate.Limiter {
	return rl.getLimiter(&rl.ipLimiters, clientIP)
}

func (rl *RateLimiter) getLimiter(store *sync.Map, key string) *rate.Limiter {
	if limiter, ok := store.Load(key); ok {
		entry := limiter.(*limiterEntry)
		entry.touch()
		rl.maybeCleanup()
		return entry.limiter
	}

	entry := newLimiterEntry(rl.userRPS, rl.userBurst)
	actual, _ := store.LoadOrStore(key, entry)
	stored := actual.(*limiterEntry)
	stored.touch()
	rl.maybeCleanup()
	return stored.limiter
}

func newLimiterEntry(rps rate.Limit, burst int) *limiterEntry {
	entry := &limiterEntry{
		limiter: rate.NewLimiter(rps, burst),
	}
	entry.touch()
	return entry
}

func (e *limiterEntry) touch() {
	e.lastSeen.Store(time.Now().UnixNano())
}

func (rl *RateLimiter) maybeCleanup() {
	if rl.requestCount.Add(1)%limiterCleanupEvery != 0 {
		return
	}

	cutoff := time.Now().Add(-limiterIdleTTL).UnixNano()
	rl.cleanupExpired(&rl.userLimiters, cutoff)
	rl.cleanupExpired(&rl.ipLimiters, cutoff)
}

func (rl *RateLimiter) cleanupExpired(store *sync.Map, cutoff int64) {
	store.Range(func(key, value interface{}) bool {
		entry, ok := value.(*limiterEntry)
		if !ok || entry.lastSeen.Load() < cutoff {
			store.Delete(key)
		}
		return true
	})
}

// RateLimit 限流中间件
func RateLimit(rl *RateLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 全局限流
		if !rl.globalLimiter.Allow() {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"code":    429,
				"message": "global rate limit exceeded, please try again later",
			})
			c.Abort()
			return
		}

		// 用户限流 (仅对已认证用户)
		if userID, exists := c.Get("user_id"); exists {
			limiter := rl.getUserLimiter(userID.(string))
			if !limiter.Allow() {
				c.JSON(http.StatusTooManyRequests, gin.H{
					"code":    429,
					"message": "user rate limit exceeded, please try again later",
				})
				c.Abort()
				return
			}
		}

		c.Next()
	}
}

// IPRateLimit IP 限流中间件 (用于非认证接口)
func IPRateLimit(rl *RateLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 全局限流
		if !rl.globalLimiter.Allow() {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"code":    429,
				"message": "global rate limit exceeded, please try again later",
			})
			c.Abort()
			return
		}

		// IP 限流
		clientIP := c.ClientIP()
		limiter := rl.getIPLimiter(clientIP)

		if !limiter.Allow() {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"code":    429,
				"message": "ip rate limit exceeded, please try again later",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}
