package handler

import (
	"context"
	"log"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"vasset/asset-service/internal/models"
	"vasset/asset-service/internal/service"
	pb "vasset/asset-service/proto"
)

// CookieHandler Cookie gRPC 处理器
type CookieHandler struct {
	cookieService *service.CookieService
}

// NewCookieHandler 创建 Cookie 处理器
func NewCookieHandler(cookieService *service.CookieService) *CookieHandler {
	return &CookieHandler{cookieService: cookieService}
}

// CreateCookie 创建 Cookie
func (h *CookieHandler) CreateCookie(ctx context.Context, req *pb.CreateCookieRequest) (*pb.CreateCookieResponse, error) {
	if req.Platform == "" || req.Name == "" || req.Content == "" {
		return nil, status.Error(codes.InvalidArgument, "平台、名称和内容不能为空")
	}

	cookie := &models.Cookie{
		Platform:      req.Platform,
		Name:          req.Name,
		Content:       req.Content,
		FreezeSeconds: int(req.FreezeSeconds),
	}

	// 解析过期时间
	if req.ExpireAt != "" {
		expireAt, err := time.Parse("2006-01-02 15:04:05", req.ExpireAt)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "过期时间格式错误，应为 YYYY-MM-DD HH:MM:SS")
		}
		cookie.ExpireAt = &expireAt
	}

	id, err := h.cookieService.Create(ctx, cookie)
	if err != nil {
		log.Printf("CreateCookie error: %v", err)
		return nil, status.Error(codes.Internal, "创建 Cookie 失败")
	}

	return &pb.CreateCookieResponse{Id: id}, nil
}

// UpdateCookie 更新 Cookie
func (h *CookieHandler) UpdateCookie(ctx context.Context, req *pb.UpdateCookieRequest) (*pb.UpdateCookieResponse, error) {
	if req.Id == 0 {
		return nil, status.Error(codes.InvalidArgument, "Cookie ID 不能为空")
	}

	cookie := &models.Cookie{
		ID:            req.Id,
		Name:          req.Name,
		Content:       req.Content,
		FreezeSeconds: int(req.FreezeSeconds),
	}

	if req.ExpireAt != "" {
		expireAt, err := time.Parse("2006-01-02 15:04:05", req.ExpireAt)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "过期时间格式错误")
		}
		cookie.ExpireAt = &expireAt
	}

	if err := h.cookieService.Update(ctx, cookie); err != nil {
		log.Printf("UpdateCookie error: %v", err)
		return nil, status.Error(codes.Internal, "更新 Cookie 失败")
	}

	return &pb.UpdateCookieResponse{Success: true}, nil
}

// DeleteCookie 删除 Cookie
func (h *CookieHandler) DeleteCookie(ctx context.Context, req *pb.DeleteCookieRequest) (*pb.DeleteCookieResponse, error) {
	if req.Id == 0 {
		return nil, status.Error(codes.InvalidArgument, "Cookie ID 不能为空")
	}

	if err := h.cookieService.Delete(ctx, req.Id); err != nil {
		log.Printf("DeleteCookie error: %v", err)
		return nil, status.Error(codes.Internal, "删除 Cookie 失败")
	}

	return &pb.DeleteCookieResponse{Success: true}, nil
}

// GetCookie 获取 Cookie
func (h *CookieHandler) GetCookie(ctx context.Context, req *pb.GetCookieRequest) (*pb.GetCookieResponse, error) {
	if req.Id == 0 {
		return nil, status.Error(codes.InvalidArgument, "Cookie ID 不能为空")
	}

	cookie, err := h.cookieService.GetByID(ctx, req.Id)
	if err != nil {
		log.Printf("GetCookie error: %v", err)
		return nil, status.Error(codes.Internal, "获取 Cookie 失败")
	}
	if cookie == nil {
		return nil, status.Error(codes.NotFound, "Cookie 不存在")
	}

	return &pb.GetCookieResponse{
		Cookie: cookieToProto(cookie),
	}, nil
}

// ListCookies 列表 Cookie
func (h *CookieHandler) ListCookies(ctx context.Context, req *pb.ListCookiesRequest) (*pb.ListCookiesResponse, error) {
	filter := &models.CookieFilter{
		Page:     int(req.Page),
		PageSize: int(req.PageSize),
	}

	if req.Platform != "" {
		filter.Platform = &req.Platform
	}
	if req.Status != 0 {
		s := models.CookieStatus(req.Status)
		filter.Status = &s
	}

	result, err := h.cookieService.List(ctx, filter)
	if err != nil {
		log.Printf("ListCookies error: %v", err)
		return nil, status.Error(codes.Internal, "查询 Cookie 列表失败")
	}

	items := make([]*pb.CookieInfo, 0, len(result.Items))
	for _, c := range result.Items {
		items = append(items, cookieToProto(&c))
	}

	return &pb.ListCookiesResponse{
		Total:    result.Total,
		Page:     int32(result.Page),
		PageSize: int32(result.PageSize),
		Items:    items,
	}, nil
}

// GetAvailableCookie 获取可用 Cookie
func (h *CookieHandler) GetAvailableCookie(ctx context.Context, req *pb.GetAvailableCookieRequest) (*pb.GetAvailableCookieResponse, error) {
	if req.Platform == "" {
		return nil, status.Error(codes.InvalidArgument, "平台不能为空")
	}

	cookieID, content, err := h.cookieService.GetAvailableCookie(ctx, req.Platform)
	if err != nil {
		log.Printf("GetAvailableCookie error: %v", err)
		return nil, status.Error(codes.NotFound, "没有可用的 Cookie")
	}

	return &pb.GetAvailableCookieResponse{
		CookieId: cookieID,
		Content:  content,
	}, nil
}

// ReportCookieUsage 报告 Cookie 使用结果
func (h *CookieHandler) ReportCookieUsage(ctx context.Context, req *pb.ReportCookieUsageRequest) (*pb.ReportCookieUsageResponse, error) {
	if req.CookieId == 0 {
		return nil, status.Error(codes.InvalidArgument, "Cookie ID 不能为空")
	}

	if err := h.cookieService.ReportUsage(ctx, req.CookieId, req.Success); err != nil {
		log.Printf("ReportCookieUsage error: %v", err)
		return nil, status.Error(codes.Internal, "报告使用结果失败")
	}

	return &pb.ReportCookieUsageResponse{Success: true}, nil
}

// FreezeCookie 冷冻 Cookie
func (h *CookieHandler) FreezeCookie(ctx context.Context, req *pb.FreezeCookieRequest) (*pb.FreezeCookieResponse, error) {
	if req.CookieId == 0 {
		return nil, status.Error(codes.InvalidArgument, "Cookie ID 不能为空")
	}

	frozenUntil, err := h.cookieService.Freeze(ctx, req.CookieId, int(req.FreezeSeconds))
	if err != nil {
		log.Printf("FreezeCookie error: %v", err)
		return nil, status.Error(codes.Internal, "冷冻 Cookie 失败")
	}

	return &pb.FreezeCookieResponse{
		Success:     true,
		FrozenUntil: frozenUntil.Format("2006-01-02 15:04:05"),
	}, nil
}

// cookieToProto 将模型转换为 Proto
func cookieToProto(c *models.Cookie) *pb.CookieInfo {
	info := &pb.CookieInfo{
		Id:            c.ID,
		Platform:      c.Platform,
		Name:          c.Name,
		Content:       c.Content,
		Status:        int32(c.GetEffectiveStatus()),
		FreezeSeconds: int32(c.FreezeSeconds),
		UseCount:      int32(c.UseCount),
		SuccessCount:  int32(c.SuccessCount),
		FailCount:     int32(c.FailCount),
		CreatedAt:     c.CreatedAt.Format("2006-01-02 15:04:05"),
		UpdatedAt:     c.UpdatedAt.Format("2006-01-02 15:04:05"),
	}

	if c.ExpireAt != nil {
		info.ExpireAt = c.ExpireAt.Format("2006-01-02 15:04:05")
	}
	if c.FrozenUntil != nil {
		info.FrozenUntil = c.FrozenUntil.Format("2006-01-02 15:04:05")
	}
	if c.LastUsedAt != nil {
		info.LastUsedAt = c.LastUsedAt.Format("2006-01-02 15:04:05")
	}

	return info
}
