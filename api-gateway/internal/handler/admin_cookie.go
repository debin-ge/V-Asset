package handler

import (
	"context"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"youdlp/api-gateway/internal/models"
	pb "youdlp/api-gateway/proto"
)

type AdminCookieHandler struct {
	adminClient pb.AdminServiceClient
	timeout     time.Duration
}

func NewAdminCookieHandler(adminClient pb.AdminServiceClient, timeout time.Duration) *AdminCookieHandler {
	return &AdminCookieHandler{adminClient: adminClient, timeout: timeout}
}

func (h *AdminCookieHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	status, _ := strconv.Atoi(c.DefaultQuery("status", "0"))

	ctx, cancel := context.WithTimeout(c.Request.Context(), h.timeout)
	defer cancel()

	resp, err := h.adminClient.ListCookies(ctx, &pb.AdminListCookiesRequest{
		Platform: c.Query("platform"),
		Status:   int32(status),
		Page:     int32(page),
		PageSize: int32(pageSize),
	})
	if err != nil {
		models.InternalError(c, grpcErrorMessage(err))
		return
	}

	items := make([]models.CookieInfo, 0, len(resp.GetItems()))
	for _, item := range resp.GetItems() {
		items = append(items, models.CookieInfo{
			ID:            item.GetId(),
			Platform:      item.GetPlatform(),
			Name:          item.GetName(),
			Content:       item.GetContent(),
			Status:        item.GetStatus(),
			ExpireAt:      item.GetExpireAt(),
			FrozenUntil:   item.GetFrozenUntil(),
			FreezeSeconds: item.GetFreezeSeconds(),
			LastUsedAt:    item.GetLastUsedAt(),
			UseCount:      item.GetUseCount(),
			SuccessCount:  item.GetSuccessCount(),
			FailCount:     item.GetFailCount(),
			CreatedAt:     item.GetCreatedAt(),
			UpdatedAt:     item.GetUpdatedAt(),
		})
	}

	models.Success(c, models.CookieListResponse{
		Total:    resp.GetTotal(),
		Page:     int(resp.GetPage()),
		PageSize: int(resp.GetPageSize()),
		Items:    items,
	})
}

func (h *AdminCookieHandler) Get(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		models.BadRequest(c, "invalid cookie id")
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), h.timeout)
	defer cancel()

	resp, err := h.adminClient.GetCookie(ctx, &pb.AdminGetCookieRequest{Id: id})
	if err != nil {
		writeGRPCError(c, err)
		return
	}

	item := resp.GetCookie()
	if item == nil {
		models.NotFound(c, "cookie not found")
		return
	}

	models.Success(c, models.CookieInfo{
		ID:            item.GetId(),
		Platform:      item.GetPlatform(),
		Name:          item.GetName(),
		Content:       item.GetContent(),
		Status:        item.GetStatus(),
		ExpireAt:      item.GetExpireAt(),
		FrozenUntil:   item.GetFrozenUntil(),
		FreezeSeconds: item.GetFreezeSeconds(),
		LastUsedAt:    item.GetLastUsedAt(),
		UseCount:      item.GetUseCount(),
		SuccessCount:  item.GetSuccessCount(),
		FailCount:     item.GetFailCount(),
		CreatedAt:     item.GetCreatedAt(),
		UpdatedAt:     item.GetUpdatedAt(),
	})
}

func (h *AdminCookieHandler) Create(c *gin.Context) {
	var req models.CreateCookieRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		models.BadRequest(c, "invalid request: "+err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), h.timeout)
	defer cancel()

	resp, err := h.adminClient.CreateCookie(ctx, &pb.AdminCreateCookieRequest{
		Platform:      req.Platform,
		Name:          req.Name,
		Content:       req.Content,
		ExpireAt:      req.ExpireAt,
		FreezeSeconds: req.FreezeSeconds,
	})
	if err != nil {
		models.InternalError(c, grpcErrorMessage(err))
		return
	}

	models.Success(c, models.CreateCookieResponse{ID: resp.GetId()})
}

func (h *AdminCookieHandler) Update(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
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

	_, err = h.adminClient.UpdateCookie(ctx, &pb.AdminUpdateCookieRequest{
		Id:            id,
		Name:          req.Name,
		Content:       req.Content,
		ExpireAt:      req.ExpireAt,
		FreezeSeconds: req.FreezeSeconds,
	})
	if err != nil {
		models.InternalError(c, grpcErrorMessage(err))
		return
	}

	models.Success(c, gin.H{"success": true})
}

func (h *AdminCookieHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		models.BadRequest(c, "invalid cookie id")
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), h.timeout)
	defer cancel()

	_, err = h.adminClient.DeleteCookie(ctx, &pb.AdminDeleteRequest{Id: id})
	if err != nil {
		models.InternalError(c, grpcErrorMessage(err))
		return
	}

	models.Success(c, gin.H{"success": true})
}

func (h *AdminCookieHandler) Freeze(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
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

	resp, err := h.adminClient.FreezeCookie(ctx, &pb.AdminFreezeCookieRequest{
		Id:            id,
		FreezeSeconds: req.FreezeSeconds,
	})
	if err != nil {
		models.InternalError(c, grpcErrorMessage(err))
		return
	}

	models.Success(c, models.FreezeCookieResponse{
		Success:     resp.GetSuccess(),
		FrozenUntil: resp.GetFrozenUntil(),
	})
}
