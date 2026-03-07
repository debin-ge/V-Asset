package handler

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"vasset/admin-service/internal/middleware"
	"vasset/admin-service/internal/models"
	"vasset/admin-service/internal/service"
)

type AuthHandler struct {
	authService  *service.AuthService
	cookieName   string
	secure       bool
	cookieDomain string
	sameSite     http.SameSite
}

func NewAuthHandler(authService *service.AuthService, cookieName string, secure bool, cookieDomain, sameSite string) *AuthHandler {
	return &AuthHandler{
		authService:  authService,
		cookieName:   cookieName,
		secure:       secure,
		cookieDomain: cookieDomain,
		sameSite:     parseSameSite(sameSite),
	}
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req models.AdminLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		models.BadRequest(c, "invalid request: "+err.Error())
		return
	}

	session, err := h.authService.Login(c.Request.Context(), req.Email, req.Password, c.Request.UserAgent(), c.ClientIP())
	if err != nil {
		models.Unauthorized(c, err.Error())
		return
	}

	c.SetSameSite(h.sameSite)
	c.SetCookie(h.cookieName, session.SessionID, 86400, "/", h.cookieDomain, h.secure, true)
	models.Success(c, models.LoginResponse{User: session.User})
}

func (h *AuthHandler) Logout(c *gin.Context) {
	sessionID, _ := c.Get("admin_session_id")
	if id, ok := sessionID.(string); ok && id != "" {
		_ = h.authService.Logout(c.Request.Context(), id)
	}

	c.SetSameSite(h.sameSite)
	c.SetCookie(h.cookieName, "", -1, "/", h.cookieDomain, h.secure, true)
	models.Success(c, gin.H{"success": true})
}

func (h *AuthHandler) Me(c *gin.Context) {
	user, ok := middleware.GetAdminUser(c)
	if !ok {
		models.Unauthorized(c, "missing admin user")
		return
	}

	models.Success(c, models.AdminMeResponse{User: user})
}

func parseSameSite(value string) http.SameSite {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "none":
		return http.SameSiteNoneMode
	case "strict":
		return http.SameSiteStrictMode
	default:
		return http.SameSiteLaxMode
	}
}
