package middleware

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"

	pb "youdlp/api-gateway/proto"
)

// JWTAuth JWT 认证中间件
func JWTAuth(authClient pb.AuthServiceClient, redisClient *redis.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")

		claims, err := AuthenticateToken(c.Request.Context(), authClient, redisClient, authHeader)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"code":    401,
				"message": err.Error(),
			})
			c.Abort()
			return
		}

		c.Set("user_id", claims.UserID)
		c.Set("user_email", claims.Email)
		c.Set("user_role", claims.Role)
		c.Set("token", claims.Token)

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
		if value, ok := userID.(string); ok {
			return value
		}
	}
	return ""
}

// GetUserEmail 从上下文获取用户邮箱
func GetUserEmail(c *gin.Context) string {
	if email, exists := c.Get("user_email"); exists {
		if value, ok := email.(string); ok {
			return value
		}
	}
	return ""
}

func GetToken(c *gin.Context) string {
	if token, exists := c.Get("token"); exists {
		if value, ok := token.(string); ok {
			return value
		}
	}
	return ""
}
