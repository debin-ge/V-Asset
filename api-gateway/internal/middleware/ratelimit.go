package middleware

import (
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"

	"vasset/api-gateway/internal/config"
)

// RateLimiter 限流器
type RateLimiter struct {
	globalLimiter *rate.Limiter
	userLimiters  sync.Map
	userRPS       rate.Limit
	userBurst     int
}

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
	if limiter, ok := rl.userLimiters.Load(userID); ok {
		return limiter.(*rate.Limiter)
	}

	// 创建新的限流器
	limiter := rate.NewLimiter(rl.userRPS, rl.userBurst)
	rl.userLimiters.Store(userID, limiter)
	return limiter
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
	ipLimiters := sync.Map{}

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
		var limiter *rate.Limiter
		if l, ok := ipLimiters.Load(clientIP); ok {
			limiter = l.(*rate.Limiter)
		} else {
			limiter = rate.NewLimiter(rl.userRPS, rl.userBurst)
			ipLimiters.Store(clientIP, limiter)
		}

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
