package handler

import (
	"context"
	"log"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"

	"vasset/api-gateway/internal/middleware"
	"vasset/api-gateway/internal/models"
	pb "vasset/api-gateway/proto"
)

// AuthHandler 认证处理器
type AuthHandler struct {
	authClient  pb.AuthServiceClient
	redisClient *redis.Client
	timeout     time.Duration
}

// NewAuthHandler 创建认证处理器
func NewAuthHandler(authClient pb.AuthServiceClient, redisClient *redis.Client, timeout time.Duration) *AuthHandler {
	return &AuthHandler{
		authClient:  authClient,
		redisClient: redisClient,
		timeout:     timeout,
	}
}

// Register 用户注册
func (h *AuthHandler) Register(c *gin.Context) {
	var req models.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		models.BadRequest(c, "invalid request: "+err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), h.timeout)
	defer cancel()

	resp, err := h.authClient.Register(ctx, &pb.RegisterRequest{
		Email:    req.Email,
		Password: req.Password,
		Nickname: req.Nickname,
	})
	if err != nil {
		writeGRPCError(c, err)
		return
	}

	models.Created(c, models.UserResponse{
		UserID:   resp.UserId,
		Email:    resp.Email,
		Nickname: resp.Nickname,
	})
}

// Login 用户登录
func (h *AuthHandler) Login(c *gin.Context) {
	var req models.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		models.BadRequest(c, "invalid request: "+err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), h.timeout)
	defer cancel()

	resp, err := h.authClient.Login(ctx, &pb.LoginRequest{
		Email:      req.Email,
		Password:   req.Password,
		DeviceInfo: c.GetHeader("User-Agent"),
		IpAddress:  c.ClientIP(),
	})
	if err != nil {
		writeGRPCError(c, err)
		return
	}

	models.Success(c, models.LoginResponse{
		Token:        resp.Token,
		RefreshToken: resp.RefreshToken,
		ExpiresIn:    resp.ExpiresIn,
		User: models.UserResponse{
			UserID:    resp.User.UserId,
			Email:     resp.User.Email,
			Nickname:  resp.User.Nickname,
			AvatarURL: resp.User.AvatarUrl,
			Role:      resp.User.Role,
			CreatedAt: resp.User.CreatedAt,
		},
	})
}

// Logout 登出
func (h *AuthHandler) Logout(c *gin.Context) {
	token := middleware.GetToken(c)
	if token == "" {
		models.BadRequest(c, "token not found")
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), h.timeout)
	defer cancel()

	resp, err := h.authClient.Logout(ctx, &pb.LogoutRequest{
		Token: token,
	})
	if err != nil {
		writeGRPCError(c, err)
		return
	}
	if !resp.Success {
		models.InternalError(c, "logout failed, please try again later")
		return
	}

	if err := middleware.InvalidateTokenCache(c.Request.Context(), h.redisClient, token); err != nil {
		log.Printf("[Auth] Failed to invalidate token cache: %v", err)
	}

	models.Success(c, nil)
}

// GetProfile 获取用户信息
func (h *AuthHandler) GetProfile(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		models.Unauthorized(c, "user not authenticated")
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), h.timeout)
	defer cancel()

	resp, err := h.authClient.GetUserInfo(ctx, &pb.GetUserInfoRequest{
		UserId: userID,
	})
	if err != nil {
		writeGRPCError(c, err)
		return
	}

	models.Success(c, models.UserResponse{
		UserID:    resp.User.UserId,
		Email:     resp.User.Email,
		Nickname:  resp.User.Nickname,
		AvatarURL: resp.User.AvatarUrl,
		Role:      resp.User.Role,
		CreatedAt: resp.User.CreatedAt,
	})
}

// UpdateProfile 更新用户信息
func (h *AuthHandler) UpdateProfile(c *gin.Context) {
	var req models.UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		models.BadRequest(c, "invalid request: "+err.Error())
		return
	}

	userID := middleware.GetUserID(c)
	if userID == "" {
		models.Unauthorized(c, "user not authenticated")
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), h.timeout)
	defer cancel()

	resp, err := h.authClient.UpdateProfile(ctx, &pb.UpdateProfileRequest{
		UserId:   userID,
		Nickname: req.Nickname,
	})
	if err != nil {
		writeGRPCError(c, err)
		return
	}

	models.Success(c, models.UserResponse{
		UserID:    resp.User.UserId,
		Email:     resp.User.Email,
		Nickname:  resp.User.Nickname,
		AvatarURL: resp.User.AvatarUrl,
		Role:      resp.User.Role,
		CreatedAt: resp.User.CreatedAt,
	})
}

// ChangePassword 修改密码
func (h *AuthHandler) ChangePassword(c *gin.Context) {
	var req models.ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		models.BadRequest(c, "invalid request: "+err.Error())
		return
	}

	userID := middleware.GetUserID(c)
	if userID == "" {
		models.Unauthorized(c, "user not authenticated")
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), h.timeout)
	defer cancel()

	resp, err := h.authClient.ChangePassword(ctx, &pb.ChangePasswordRequest{
		UserId:      userID,
		OldPassword: req.OldPassword,
		NewPassword: req.NewPassword,
	})
	if err != nil {
		writeGRPCError(c, err)
		return
	}

	if !resp.Success {
		models.InternalError(c, "failed to change password")
		return
	}

	models.Success(c, nil)
}
