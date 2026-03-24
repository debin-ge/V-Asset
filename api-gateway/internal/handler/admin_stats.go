package handler

import (
	"context"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"youdlp/api-gateway/internal/models"
	pb "youdlp/api-gateway/proto"
)

type AdminStatsHandler struct {
	adminClient pb.AdminServiceClient
	timeout     time.Duration
}

func NewAdminStatsHandler(adminClient pb.AdminServiceClient, timeout time.Duration) *AdminStatsHandler {
	return &AdminStatsHandler{adminClient: adminClient, timeout: timeout}
}

func (h *AdminStatsHandler) Overview(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), h.timeout)
	defer cancel()

	resp, err := h.adminClient.GetOverview(ctx, &pb.AdminEmpty{})
	if err != nil {
		models.InternalError(c, grpcErrorMessage(err))
		return
	}

	models.Success(c, models.AdminOverviewResponse{
		TotalUsers:          resp.GetTotalUsers(),
		DailyActiveUsers:    resp.GetDailyActiveUsers(),
		WeeklyActiveUsers:   resp.GetWeeklyActiveUsers(),
		TotalDownloads:      resp.GetTotalDownloads(),
		DownloadsToday:      resp.GetDownloadsToday(),
		SuccessDownloads:    resp.GetSuccessDownloads(),
		FailedDownloads:     resp.GetFailedDownloads(),
		ActiveManualProxies: resp.GetActiveManualProxies(),
		TotalManualProxies:  resp.GetTotalManualProxies(),
	})
}

func (h *AdminStatsHandler) RequestTrend(c *gin.Context) {
	granularity := c.DefaultQuery("granularity", "day")
	limit, err := strconv.Atoi(c.DefaultQuery("limit", "7"))
	if err != nil || limit <= 0 {
		limit = 7
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), h.timeout)
	defer cancel()

	resp, err := h.adminClient.GetRequestTrend(ctx, &pb.AdminRequestTrendRequest{
		Granularity: granularity,
		Limit:       int32(limit),
	})
	if err != nil {
		models.InternalError(c, grpcErrorMessage(err))
		return
	}

	points := make([]models.AdminTrendPoint, 0, len(resp.GetPoints()))
	for _, point := range resp.GetPoints() {
		points = append(points, models.AdminTrendPoint{Label: point.GetLabel(), Count: point.GetCount()})
	}

	models.Success(c, models.AdminRequestTrendResponse{
		Granularity: resp.GetGranularity(),
		Points:      points,
	})
}

func (h *AdminStatsHandler) Users(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), h.timeout)
	defer cancel()

	resp, err := h.adminClient.GetUserStats(ctx, &pb.AdminEmpty{})
	if err != nil {
		models.InternalError(c, grpcErrorMessage(err))
		return
	}

	models.Success(c, models.AdminUserStatsResponse{
		TotalUsers:        resp.GetTotalUsers(),
		DailyActiveUsers:  resp.GetDailyActiveUsers(),
		WeeklyActiveUsers: resp.GetWeeklyActiveUsers(),
	})
}
