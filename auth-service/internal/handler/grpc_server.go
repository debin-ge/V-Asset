package handler

import (
	"context"
	"fmt"
	"strings"

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
		return nil, authStatusError(err)
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
		return nil, status.Error(codes.Internal, err.Error())
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

// GetPlatformUserStats 获取平台用户统计
func (s *GRPCServer) GetPlatformUserStats(ctx context.Context, req *pb.GetPlatformUserStatsRequest) (*pb.GetPlatformUserStatsResponse, error) {
	totalUsers, err := s.userService.GetPlatformUserStats(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &pb.GetPlatformUserStatsResponse{
		TotalUsers: totalUsers,
	}, nil
}

// UpdateProfile 更新用户信息
func (s *GRPCServer) UpdateProfile(ctx context.Context, req *pb.UpdateProfileRequest) (*pb.UpdateProfileResponse, error) {
	LogRequest("UpdateProfile", req)

	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}
	if req.Nickname == "" {
		return nil, status.Error(codes.InvalidArgument, "nickname is required")
	}

	user, err := s.authService.UpdateProfile(ctx, req.UserId, req.Nickname)
	if err != nil {
		return nil, authStatusError(err)
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
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}
	if req.OldPassword == "" {
		return nil, status.Error(codes.InvalidArgument, "old_password is required")
	}
	if req.NewPassword == "" {
		return nil, status.Error(codes.InvalidArgument, "new_password is required")
	}

	err := s.authService.ChangePassword(ctx, req.UserId, req.OldPassword, req.NewPassword)
	if err != nil {
		return nil, authStatusError(err)
	}

	return &pb.ChangePasswordResponse{
		Success: true,
	}, nil
}

func (s *GRPCServer) SearchUsers(ctx context.Context, req *pb.SearchUsersRequest) (*pb.SearchUsersResponse, error) {
	page := req.GetPage()
	if page <= 0 {
		page = 1
	}
	pageSize := req.GetPageSize()
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	users, total, err := s.userService.SearchUsers(ctx, req.GetQuery(), int(page), int(pageSize))
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	items := make([]*pb.User, 0, len(users))
	for _, user := range users {
		items = append(items, &pb.User{
			UserId:    user.ID,
			Email:     user.Email,
			Nickname:  user.Nickname,
			AvatarUrl: user.AvatarURL,
			Role:      int32(user.Role),
			CreatedAt: user.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	return &pb.SearchUsersResponse{
		Total:    total,
		Page:     page,
		PageSize: pageSize,
		Users:    items,
	}, nil
}

func (s *GRPCServer) BatchGetUsers(ctx context.Context, req *pb.BatchGetUsersRequest) (*pb.BatchGetUsersResponse, error) {
	users, err := s.userService.BatchGetUsers(ctx, req.GetUserIds())
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	items := make([]*pb.User, 0, len(users))
	for _, user := range users {
		items = append(items, &pb.User{
			UserId:    user.ID,
			Email:     user.Email,
			Nickname:  user.Nickname,
			AvatarUrl: user.AvatarURL,
			Role:      int32(user.Role),
			CreatedAt: user.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	return &pb.BatchGetUsersResponse{Users: items}, nil
}

// HealthCheck 健康检查
func (s *GRPCServer) HealthCheck(ctx context.Context) error {
	// 可以添加数据库和 Redis 连接检查
	return nil
}

// LogRequest 记录请求日志
func LogRequest(method string, req any) {
	fmt.Printf("[gRPC] Method: %s, Request: %+v\n", method, req)
}

func authStatusError(err error) error {
	if err == nil {
		return nil
	}
	if _, ok := status.FromError(err); ok {
		return err
	}

	message := strings.TrimSpace(err.Error())
	switch {
	case message == "":
		return status.Error(codes.Internal, "internal server error")
	case strings.Contains(message, "邮箱已被注册"):
		return status.Error(codes.AlreadyExists, message)
	case strings.Contains(message, "不能为空"),
		strings.Contains(message, "格式不正确"),
		strings.Contains(message, "长度必须"),
		strings.Contains(message, "旧密码不正确"),
		strings.Contains(message, "nickname is required"),
		strings.Contains(message, "user_id is required"),
		strings.Contains(message, "old_password is required"),
		strings.Contains(message, "new_password is required"):
		return status.Error(codes.InvalidArgument, message)
	case strings.Contains(message, "用户不存在"):
		return status.Error(codes.NotFound, message)
	default:
		return status.Error(codes.Internal, message)
	}
}
