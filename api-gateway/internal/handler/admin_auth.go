package handler

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"youdlp/api-gateway/internal/config"
	"youdlp/api-gateway/internal/models"
	pb "youdlp/api-gateway/proto"
)

type AdminAuthHandler struct {
	adminClient pb.AdminServiceClient
	timeout     time.Duration
	sessionCfg  *config.AdminSessionConfig
}

func NewAdminAuthHandler(adminClient pb.AdminServiceClient, timeout time.Duration, sessionCfg *config.AdminSessionConfig) *AdminAuthHandler {
	return &AdminAuthHandler{
		adminClient: adminClient,
		timeout:     timeout,
		sessionCfg:  sessionCfg,
	}
}

func (h *AdminAuthHandler) Login(c *gin.Context) {
	var req models.AdminLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		models.BadRequest(c, "invalid request: "+err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), h.timeout)
	defer cancel()

	resp, err := h.adminClient.Login(ctx, &pb.AdminLoginRequest{
		Email:     req.Email,
		Password:  req.Password,
		UserAgent: c.Request.UserAgent(),
		IpAddress: c.ClientIP(),
	})
	if err != nil {
		models.Unauthorized(c, grpcErrorMessage(err))
		return
	}

	setAdminSessionCookie(c, h.sessionCfg, resp.GetSessionId())
	models.Success(c, models.AdminLoginResponse{User: adminUserFromProto(resp.GetUser())})
}

func (h *AdminAuthHandler) Logout(c *gin.Context) {
	sessionID, _ := c.Get("admin_session_id")
	if sessionIDStr, ok := sessionID.(string); ok && sessionIDStr != "" {
		ctx, cancel := context.WithTimeout(c.Request.Context(), h.timeout)
		_, err := h.adminClient.Logout(ctx, &pb.AdminLogoutRequest{SessionId: sessionIDStr})
		cancel()
		if err != nil {
			models.InternalError(c, grpcErrorMessage(err))
			return
		}
	}

	clearAdminSessionCookie(c, h.sessionCfg)
	models.Success(c, gin.H{"success": true})
}

func (h *AdminAuthHandler) Me(c *gin.Context) {
	user, exists := c.Get("admin_user")
	if !exists {
		models.Unauthorized(c, "missing admin user")
		return
	}

	adminUser, ok := user.(*pb.AdminUser)
	if !ok {
		models.Unauthorized(c, "invalid admin user")
		return
	}

	models.Success(c, models.AdminMeResponse{User: adminUserFromProto(adminUser)})
}

func setAdminSessionCookie(c *gin.Context, cfg *config.AdminSessionConfig, sessionID string) {
	c.SetSameSite(parseSameSite(cfg.SameSite))
	maxAge := int(cfg.TTL / time.Second)
	c.SetCookie(cfg.CookieName, sessionID, maxAge, "/", cfg.CookieDomain, cfg.Secure, true)
}

func clearAdminSessionCookie(c *gin.Context, cfg *config.AdminSessionConfig) {
	c.SetSameSite(parseSameSite(cfg.SameSite))
	c.SetCookie(cfg.CookieName, "", -1, "/", cfg.CookieDomain, cfg.Secure, true)
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

func adminUserFromProto(user *pb.AdminUser) models.AdminUser {
	if user == nil {
		return models.AdminUser{}
	}

	return models.AdminUser{
		UserID:    user.GetUserId(),
		Email:     user.GetEmail(),
		Nickname:  user.GetNickname(),
		AvatarURL: user.GetAvatarUrl(),
		Role:      user.GetRole(),
		CreatedAt: user.GetCreatedAt(),
	}
}
