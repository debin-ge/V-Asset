package middleware

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"

	pb "vasset/api-gateway/proto"
)

// JWTAuth JWT 认证中间件
func JWTAuth(authClient pb.AuthServiceClient, redisClient *redis.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. 提取 Token
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"code":    401,
				"message": "missing authorization header",
			})
			c.Abort()
			return
		}

		if !strings.HasPrefix(authHeader, "Bearer ") {
			c.JSON(http.StatusUnauthorized, gin.H{
				"code":    401,
				"message": "invalid authorization header format",
			})
			c.Abort()
			return
		}

		token := strings.TrimPrefix(authHeader, "Bearer ")
		if token == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"code":    401,
				"message": "empty token",
			})
			c.Abort()
			return
		}

		// 2. 检查 Redis 缓存
		cacheKey := fmt.Sprintf("auth:token:%s", hashToken(token))
		ctx := context.Background()

		cachedData, err := redisClient.HGetAll(ctx, cacheKey).Result()
		if err == nil && len(cachedData) > 0 {
			// 缓存命中
			userID := cachedData["user_id"]
			email := cachedData["email"]
			role := cachedData["role"]

			if userID != "" {
				c.Set("user_id", userID)
				c.Set("user_email", email)
				c.Set("user_role", role)
				c.Set("token", token)
				c.Next()
				return
			}
		}

		// 3. 调用 Auth Service 验证
		verifyCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()

		resp, err := authClient.VerifyToken(verifyCtx, &pb.VerifyTokenRequest{
			Token: token,
		})

		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"code":    401,
				"message": "token verification failed",
			})
			c.Abort()
			return
		}

		if !resp.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{
				"code":    401,
				"message": "invalid token",
			})
			c.Abort()
			return
		}

		// 4. 写入缓存 (TTL = 5分钟)
		redisClient.HSet(ctx, cacheKey, map[string]interface{}{
			"user_id": resp.UserId,
			"email":   resp.Email,
			"role":    resp.Role,
		})
		redisClient.Expire(ctx, cacheKey, 5*time.Minute)

		// 5. 设置上下文
		c.Set("user_id", resp.UserId)
		c.Set("user_email", resp.Email)
		c.Set("user_role", resp.Role)
		c.Set("token", token)

		c.Next()
	}
}

// hashToken 对 Token 进行哈希处理
func hashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

// GetUserID 从上下文获取用户 ID (UUID字符串)
func GetUserID(c *gin.Context) string {
	if userID, exists := c.Get("user_id"); exists {
		return userID.(string)
	}
	return ""
}

// GetUserEmail 从上下文获取用户邮箱
func GetUserEmail(c *gin.Context) string {
	if email, exists := c.Get("user_email"); exists {
		return email.(string)
	}
	return ""
}
