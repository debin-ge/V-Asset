package handler

import (
	"context"
	"fmt"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"vasset/auth-service/internal/service"
	pb "vasset/auth-service/proto"
)

// GRPCServer gRPC 服务器
type GRPCServer struct {
	pb.UnimplementedAuthServiceServer
	authService  *service.AuthService
	userService  *service.UserService
	tokenService *service.TokenService
}

// NewGRPCServer 创建 gRPC 服务器
func NewGRPCServer(
	authService *service.AuthService,
	userService *service.UserService,
	tokenService *service.TokenService,
) *GRPCServer {
	return &GRPCServer{
		authService:  authService,
		userService:  userService,
		tokenService: tokenService,
	}
}

// Register 用户注册
func (s *GRPCServer) Register(ctx context.Context, req *pb.RegisterRequest) (*pb.RegisterResponse, error) {
	// 参数验证
	if req.Email == "" || req.Password == "" {
		return nil, status.Error(codes.InvalidArgument, "邮箱和密码不能为空")
	}

	// 调用服务层
	user, err := s.authService.Register(ctx, req.Email, req.Password, req.Nickname)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &pb.RegisterResponse{
		UserId:   user.ID,
		Email:    user.Email,
		Nickname: user.Nickname,
	}, nil
}

// Login 用户登录
func (s *GRPCServer) Login(ctx context.Context, req *pb.LoginRequest) (*pb.LoginResponse, error) {
	// 参数验证
	if req.Email == "" || req.Password == "" {
		return nil, status.Error(codes.InvalidArgument, "邮箱和密码不能为空")
	}

	// 调用服务层
	accessToken, refreshToken, user, err := s.authService.Login(
		ctx,
		req.Email,
		req.Password,
		req.DeviceInfo,
		req.IpAddress,
	)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, err.Error())
	}

	return &pb.LoginResponse{
		Token:        accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    s.tokenService.GetAccessTokenTTL(),
		User: &pb.User{
			UserId:    user.ID,
			Email:     user.Email,
			Nickname:  user.Nickname,
			AvatarUrl: user.AvatarURL,
			Role:      int32(user.Role),
			CreatedAt: user.CreatedAt.Format("2006-01-02 15:04:05"),
		},
	}, nil
}

// VerifyToken Token 验证
func (s *GRPCServer) VerifyToken(ctx context.Context, req *pb.VerifyTokenRequest) (*pb.VerifyTokenResponse, error) {
	if req.Token == "" {
		return nil, status.Error(codes.InvalidArgument, "Token 不能为空")
	}

	claims, err := s.tokenService.VerifyToken(ctx, req.Token)
	if err != nil {
		return &pb.VerifyTokenResponse{
			Valid: false,
		}, nil
	}

	return &pb.VerifyTokenResponse{
		Valid:  true,
		UserId: claims.UserID,
		Email:  claims.Email,
		Role:   claims.Role,
	}, nil
}

// RefreshToken Token 刷新
func (s *GRPCServer) RefreshToken(ctx context.Context, req *pb.RefreshTokenRequest) (*pb.RefreshTokenResponse, error) {
	if req.RefreshToken == "" {
		return nil, status.Error(codes.InvalidArgument, "Refresh Token 不能为空")
	}

	newToken, err := s.tokenService.RefreshToken(ctx, req.RefreshToken)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, err.Error())
	}

	return &pb.RefreshTokenResponse{
		Token:     newToken,
		ExpiresIn: s.tokenService.GetAccessTokenTTL(),
	}, nil
}

// Logout 用户登出
func (s *GRPCServer) Logout(ctx context.Context, req *pb.LogoutRequest) (*pb.LogoutResponse, error) {
	if req.Token == "" {
		return nil, status.Error(codes.InvalidArgument, "Token 不能为空")
	}

	err := s.authService.Logout(ctx, req.Token)
	if err != nil {
		return &pb.LogoutResponse{Success: false}, nil
	}

	return &pb.LogoutResponse{Success: true}, nil
}

// GetUserInfo 获取用户信息
func (s *GRPCServer) GetUserInfo(ctx context.Context, req *pb.GetUserInfoRequest) (*pb.GetUserInfoResponse, error) {
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "用户 ID 不能为空")
	}

	user, err := s.userService.GetUserByID(ctx, req.UserId)
	if err != nil {
		return nil, status.Error(codes.NotFound, err.Error())
	}

	return &pb.GetUserInfoResponse{
		User: &pb.User{
			UserId:    user.ID,
			Email:     user.Email,
			Nickname:  user.Nickname,
			AvatarUrl: user.AvatarURL,
			Role:      int32(user.Role),
			CreatedAt: user.CreatedAt.Format("2006-01-02 15:04:05"),
		},
	}, nil
}

// UpdateProfile 更新用户信息
func (s *GRPCServer) UpdateProfile(ctx context.Context, req *pb.UpdateProfileRequest) (*pb.UpdateProfileResponse, error) {
	LogRequest("UpdateProfile", req)

	if req.UserId == "" {
		return nil, fmt.Errorf("user_id is required")
	}
	if req.Nickname == "" {
		return nil, fmt.Errorf("nickname is required")
	}

	user, err := s.authService.UpdateProfile(ctx, req.UserId, req.Nickname)
	if err != nil {
		return nil, err
	}

	return &pb.UpdateProfileResponse{
		User: &pb.User{
			UserId:    user.ID,
			Email:     user.Email,
			Nickname:  user.Nickname,
			AvatarUrl: user.AvatarURL,
			Role:      int32(user.Role),
			CreatedAt: user.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		},
	}, nil
}

// ChangePassword 修改密码
func (s *GRPCServer) ChangePassword(ctx context.Context, req *pb.ChangePasswordRequest) (*pb.ChangePasswordResponse, error) {
	LogRequest("ChangePassword", "userID:"+req.UserId)

	if req.UserId == "" {
		return nil, fmt.Errorf("user_id is required")
	}
	if req.OldPassword == "" {
		return nil, fmt.Errorf("old_password is required")
	}
	if req.NewPassword == "" {
		return nil, fmt.Errorf("new_password is required")
	}

	err := s.authService.ChangePassword(ctx, req.UserId, req.OldPassword, req.NewPassword)
	if err != nil {
		return nil, err
	}

	return &pb.ChangePasswordResponse{
		Success: true,
	}, nil
}

// HealthCheck 健康检查
func (s *GRPCServer) HealthCheck(ctx context.Context) error {
	// 可以添加数据库和 Redis 连接检查
	return nil
}

// LogRequest 记录请求日志
func LogRequest(method string, req interface{}) {
	fmt.Printf("[gRPC] Method: %s, Request: %+v\n", method, req)
}
