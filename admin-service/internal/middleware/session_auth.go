package middleware

import (
	"github.com/gin-gonic/gin"

	"vasset/admin-service/internal/models"
	"vasset/admin-service/internal/service"
)

const adminUserContextKey = "admin_user"

func SessionAuth(sessionService *service.SessionService, cookieName string) gin.HandlerFunc {
	return func(c *gin.Context) {
		sessionID, err := c.Cookie(cookieName)
		if err != nil || sessionID == "" {
			models.Unauthorized(c, "missing admin session")
			c.Abort()
			return
		}

		session, err := sessionService.Get(c.Request.Context(), sessionID)
		if err != nil {
			models.Unauthorized(c, "invalid admin session")
			c.Abort()
			return
		}

		c.Set(adminUserContextKey, session.User)
		c.Set("admin_session_id", session.SessionID)
		c.Next()
	}
}

func GetAdminUser(c *gin.Context) (models.AdminUser, bool) {
	user, ok := c.Get(adminUserContextKey)
	if !ok {
		return models.AdminUser{}, false
	}

	adminUser, ok := user.(models.AdminUser)
	return adminUser, ok
}
