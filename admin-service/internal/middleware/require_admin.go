package middleware

import (
	"github.com/gin-gonic/gin"

	"vasset/admin-service/internal/models"
)

func RequireAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		user, ok := GetAdminUser(c)
		if !ok || user.Role != 99 {
			models.Forbidden(c, "admin access required")
			c.Abort()
			return
		}

		c.Next()
	}
}
