package middleware

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"vasset/api-gateway/internal/config"
	pb "vasset/api-gateway/proto"
)

func AdminSession(adminClient pb.AdminServiceClient, cfg *config.AdminSessionConfig, timeout time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		cookie, err := c.Cookie(cfg.CookieName)
		if err != nil || strings.TrimSpace(cookie) == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"code": http.StatusUnauthorized, "message": "missing admin session"})
			c.Abort()
			return
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), timeout)
		defer cancel()

		resp, err := adminClient.GetCurrentUser(ctx, &pb.AdminSessionRequest{SessionId: cookie})
		if err != nil || resp.GetUser() == nil {
			c.JSON(http.StatusUnauthorized, gin.H{"code": http.StatusUnauthorized, "message": "invalid admin session"})
			c.Abort()
			return
		}

		if resp.User.Role != 99 {
			c.JSON(http.StatusForbidden, gin.H{"code": http.StatusForbidden, "message": "admin access required"})
			c.Abort()
			return
		}

		c.Set("admin_session_id", cookie)
		c.Set("admin_user", resp.User)
		c.Next()
	}
}
