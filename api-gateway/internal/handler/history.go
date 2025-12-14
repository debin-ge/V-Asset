package handler

import (
	"context"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"vasset/api-gateway/internal/middleware"
	"vasset/api-gateway/internal/models"
	pb "vasset/api-gateway/proto"
)

// HistoryHandler 历史处理器
type HistoryHandler struct {
	assetClient pb.AssetServiceClient
	timeout     time.Duration
}

// NewHistoryHandler 创建历史处理器
func NewHistoryHandler(assetClient pb.AssetServiceClient, timeout time.Duration) *HistoryHandler {
	return &HistoryHandler{
		assetClient: assetClient,
		timeout:     timeout,
	}
}

// GetHistory 获取下载历史
func (h *HistoryHandler) GetHistory(c *gin.Context) {
	var req models.HistoryRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		models.BadRequest(c, "invalid request: "+err.Error())
		return
	}

	userID := middleware.GetUserID(c)
	if userID == "" {
		models.Unauthorized(c, "user not authenticated")
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

	resp, err := h.assetClient.GetHistory(ctx, &pb.GetHistoryRequest{
		UserId:    userID,
		Status:    int32(req.Status),
		Platform:  req.Platform,
		StartDate: req.StartDate,
		EndDate:   req.EndDate,
		Page:      int32(req.Page),
		PageSize:  int32(req.PageSize),
		SortBy:    req.SortBy,
		SortOrder: req.SortOrder,
	})
	if err != nil {
		models.InternalError(c, "failed to get history: "+err.Error())
		return
	}

	// 转换结果
	items := make([]models.HistoryItem, 0, len(resp.Items))
	for _, item := range resp.Items {
		items = append(items, models.HistoryItem{
			HistoryID:   item.HistoryId,
			TaskID:      item.TaskId,
			URL:         item.Url,
			Platform:    item.Platform,
			Title:       item.Title,
			Mode:        item.Mode,
			Quality:     item.Quality,
			FileSize:    item.FileSize,
			Status:      item.Status,
			FileName:    item.FileName,
			CreatedAt:   item.CreatedAt,
			CompletedAt: item.CompletedAt,
		})
	}

	models.Success(c, models.PagedResponse{
		Total:    resp.Total,
		Page:     int(resp.Page),
		PageSize: int(resp.PageSize),
		Items:    items,
	})
}

// GetQuota 获取配额
func (h *HistoryHandler) GetQuota(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		models.Unauthorized(c, "user not authenticated")
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), h.timeout)
	defer cancel()

	resp, err := h.assetClient.CheckQuota(ctx, &pb.CheckQuotaRequest{
		UserId: userID,
	})
	if err != nil {
		models.InternalError(c, "failed to get quota: "+err.Error())
		return
	}

	models.Success(c, models.QuotaResponse{
		DailyLimit: resp.DailyLimit,
		DailyUsed:  resp.DailyUsed,
		Remaining:  resp.Remaining,
		ResetAt:    resp.ResetAt,
	})
}

// DeleteHistory 删除历史记录
func (h *HistoryHandler) DeleteHistory(c *gin.Context) {
	historyID := c.Param("id")
	if historyID == "" {
		models.BadRequest(c, "history_id is required")
		return
	}

	userID := middleware.GetUserID(c)
	if userID == "" {
		models.Unauthorized(c, "user not authenticated")
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), h.timeout)
	defer cancel()

	// 将字符串转换为 int64
	id, err := strconv.ParseInt(historyID, 10, 64)
	if err != nil {
		models.BadRequest(c, "invalid history_id format")
		return
	}

	resp, err := h.assetClient.DeleteHistory(ctx, &pb.DeleteHistoryRequest{
		HistoryId: id,
		UserId:    userID,
	})
	if err != nil {
		models.InternalError(c, "failed to delete history: "+err.Error())
		return
	}

	if !resp.Success {
		models.NotFound(c, "history not found or permission denied")
		return
	}

	models.Success(c, nil)
}

// GetUserStats 获取用户统计
func (h *HistoryHandler) GetUserStats(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		models.Unauthorized(c, "user not authenticated")
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), h.timeout)
	defer cancel()

	resp, err := h.assetClient.GetUserStats(ctx, &pb.GetUserStatsRequest{
		UserId: userID,
	})
	if err != nil {
		models.InternalError(c, "failed to get user stats: "+err.Error())
		return
	}

	// 转换平台统计
	platforms := make([]models.PlatformStat, 0, len(resp.TopPlatforms))
	for _, p := range resp.TopPlatforms {
		platforms = append(platforms, models.PlatformStat{
			Platform: p.Platform,
			Count:    p.Count,
		})
	}

	// 转换日活动
	activities := make([]models.DailyActivity, 0, len(resp.RecentActivity))
	for _, a := range resp.RecentActivity {
		activities = append(activities, models.DailyActivity{
			Date:  a.Date,
			Count: a.Count,
		})
	}

	models.Success(c, models.StatsResponse{
		TotalDownloads:   resp.TotalDownloads,
		SuccessDownloads: resp.SuccessDownloads,
		FailedDownloads:  resp.FailedDownloads,
		TotalSizeBytes:   resp.TotalSizeBytes,
		TopPlatforms:     platforms,
		RecentActivity:   activities,
	})
}
