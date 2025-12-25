package handler

import (
	"context"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"vasset/api-gateway/internal/models"
	pb "vasset/api-gateway/proto"
)

// CookieHandler Cookie 管理处理器
type CookieHandler struct {
	assetClient pb.AssetServiceClient
	timeout     time.Duration
}

// NewCookieHandler 创建 Cookie 管理处理器
func NewCookieHandler(assetClient pb.AssetServiceClient, timeout time.Duration) *CookieHandler {
	return &CookieHandler{
		assetClient: assetClient,
		timeout:     timeout,
	}
}

// CreateCookie 创建 Cookie
func (h *CookieHandler) CreateCookie(c *gin.Context) {
	var req models.CreateCookieRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		models.BadRequest(c, "invalid request: "+err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), h.timeout)
	defer cancel()

	resp, err := h.assetClient.CreateCookie(ctx, &pb.CreateCookieRequest{
		Platform:      req.Platform,
		Name:          req.Name,
		Content:       req.Content,
		ExpireAt:      req.ExpireAt,
		FreezeSeconds: req.FreezeSeconds,
	})
	if err != nil {
		models.InternalError(c, "failed to create cookie: "+err.Error())
		return
	}

	models.Success(c, models.CreateCookieResponse{
		ID: resp.Id,
	})
}

// UpdateCookie 更新 Cookie
func (h *CookieHandler) UpdateCookie(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		models.BadRequest(c, "invalid cookie id")
		return
	}

	var req models.UpdateCookieRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		models.BadRequest(c, "invalid request: "+err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), h.timeout)
	defer cancel()

	_, err = h.assetClient.UpdateCookie(ctx, &pb.UpdateCookieRequest{
		Id:            id,
		Name:          req.Name,
		Content:       req.Content,
		ExpireAt:      req.ExpireAt,
		FreezeSeconds: req.FreezeSeconds,
	})
	if err != nil {
		models.InternalError(c, "failed to update cookie: "+err.Error())
		return
	}

	models.Success(c, nil)
}

// DeleteCookie 删除 Cookie
func (h *CookieHandler) DeleteCookie(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		models.BadRequest(c, "invalid cookie id")
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), h.timeout)
	defer cancel()

	_, err = h.assetClient.DeleteCookie(ctx, &pb.DeleteCookieRequest{
		Id: id,
	})
	if err != nil {
		models.InternalError(c, "failed to delete cookie: "+err.Error())
		return
	}

	models.Success(c, nil)
}

// GetCookie 获取 Cookie 详情
func (h *CookieHandler) GetCookie(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		models.BadRequest(c, "invalid cookie id")
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), h.timeout)
	defer cancel()

	resp, err := h.assetClient.GetCookie(ctx, &pb.GetCookieRequest{
		Id: id,
	})
	if err != nil {
		models.InternalError(c, "failed to get cookie: "+err.Error())
		return
	}

	models.Success(c, models.CookieInfo{
		ID:            resp.Cookie.Id,
		Platform:      resp.Cookie.Platform,
		Name:          resp.Cookie.Name,
		Content:       resp.Cookie.Content,
		Status:        resp.Cookie.Status,
		ExpireAt:      resp.Cookie.ExpireAt,
		FrozenUntil:   resp.Cookie.FrozenUntil,
		FreezeSeconds: resp.Cookie.FreezeSeconds,
		LastUsedAt:    resp.Cookie.LastUsedAt,
		UseCount:      int64(resp.Cookie.UseCount),
		SuccessCount:  int64(resp.Cookie.SuccessCount),
		FailCount:     int64(resp.Cookie.FailCount),
		CreatedAt:     resp.Cookie.CreatedAt,
		UpdatedAt:     resp.Cookie.UpdatedAt,
	})
}

// ListCookies 列出 Cookie
func (h *CookieHandler) ListCookies(c *gin.Context) {
	var req models.ListCookiesRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		models.BadRequest(c, "invalid request: "+err.Error())
		return
	}

	// 设置默认值
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 20
	}
	if req.PageSize > 100 {
		req.PageSize = 100
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), h.timeout)
	defer cancel()

	resp, err := h.assetClient.ListCookies(ctx, &pb.ListCookiesRequest{
		Platform: req.Platform,
		Status:   req.Status,
		Page:     int32(req.Page),
		PageSize: int32(req.PageSize),
	})
	if err != nil {
		models.InternalError(c, "failed to list cookies: "+err.Error())
		return
	}

	// 转换结果
	items := make([]models.CookieInfo, 0, len(resp.Items))
	for _, item := range resp.Items {
		items = append(items, models.CookieInfo{
			ID:            item.Id,
			Platform:      item.Platform,
			Name:          item.Name,
			Content:       item.Content,
			Status:        item.Status,
			ExpireAt:      item.ExpireAt,
			FrozenUntil:   item.FrozenUntil,
			FreezeSeconds: item.FreezeSeconds,
			LastUsedAt:    item.LastUsedAt,
			UseCount:      int64(item.UseCount),
			SuccessCount:  int64(item.SuccessCount),
			FailCount:     int64(item.FailCount),
			CreatedAt:     item.CreatedAt,
			UpdatedAt:     item.UpdatedAt,
		})
	}

	models.Success(c, models.PagedResponse{
		Total:    resp.Total,
		Page:     int(resp.Page),
		PageSize: int(resp.PageSize),
		Items:    items,
	})
}

// FreezeCookie 冻结 Cookie
func (h *CookieHandler) FreezeCookie(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		models.BadRequest(c, "invalid cookie id")
		return
	}

	var req models.FreezeCookieRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		models.BadRequest(c, "invalid request: "+err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), h.timeout)
	defer cancel()

	resp, err := h.assetClient.FreezeCookie(ctx, &pb.FreezeCookieRequest{
		CookieId:      id,
		FreezeSeconds: req.FreezeSeconds,
	})
	if err != nil {
		models.InternalError(c, "failed to freeze cookie: "+err.Error())
		return
	}

	models.Success(c, models.FreezeCookieResponse{
		Success:     resp.Success,
		FrozenUntil: resp.FrozenUntil,
	})
}
