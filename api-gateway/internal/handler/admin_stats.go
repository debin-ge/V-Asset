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

const (
	adminTrendDefaultLimit = 7
	adminTrendMaxHourLimit = 168
	adminTrendMaxDayLimit  = 90
)

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
	granularity, limit := parseAdminTrendQuery(c)

	ctx, cancel := context.WithTimeout(c.Request.Context(), h.timeout)
	defer cancel()

	resp, err := h.adminClient.GetRequestTrend(ctx, &pb.AdminRequestTrendRequest{
		Granularity: granularity,
		Limit:       limit,
	})
	if err != nil {
		models.InternalError(c, grpcErrorMessage(err))
		return
	}

	points := make([]models.AdminTrendPoint, 0, len(resp.GetPoints()))
	for _, point := range resp.GetPoints() {
		points = append(points, models.AdminTrendPoint{
			Label:        point.GetLabel(),
			Count:        point.GetCount(),
			TotalCount:   point.GetTotalCount(),
			SuccessCount: point.GetSuccessCount(),
			FailedCount:  point.GetFailedCount(),
			SuccessRate:  point.GetSuccessRate(),
		})
	}

	models.Success(c, models.AdminRequestTrendResponse{
		Granularity: resp.GetGranularity(),
		Points:      points,
	})
}

func parseAdminTrendQuery(c *gin.Context) (string, int32) {
	granularity := c.DefaultQuery("granularity", "day")
	if granularity != "hour" {
		granularity = "day"
	}
	limit, err := strconv.Atoi(c.DefaultQuery("limit", "7"))
	if err != nil || limit <= 0 {
		limit = adminTrendDefaultLimit
	}
	maxLimit := adminTrendMaxDayLimit
	if granularity == "hour" {
		maxLimit = adminTrendMaxHourLimit
	}
	if limit > maxLimit {
		limit = maxLimit
	}
	return granularity, int32(limit)
}

func (h *AdminStatsHandler) DashboardHealth(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), h.timeout)
	defer cancel()

	resp, err := h.adminClient.GetDashboardHealth(ctx, &pb.AdminEmpty{})
	if err != nil {
		models.InternalError(c, grpcErrorMessage(err))
		return
	}

	models.Success(c, models.AdminDashboardHealthResponse{
		GeneratedAt: resp.GetGeneratedAt(),
		Downloads: models.AdminDashboardDownloads{
			Total:        resp.GetDownloads().GetTotal(),
			TodayTotal:   resp.GetDownloads().GetTodayTotal(),
			SuccessTotal: resp.GetDownloads().GetSuccessTotal(),
			FailedTotal:  resp.GetDownloads().GetFailedTotal(),
			SuccessRate:  resp.GetDownloads().GetSuccessRate(),
			FailureRate:  resp.GetDownloads().GetFailureRate(),
		},
		Users: models.AdminDashboardUsers{
			Total:        resp.GetUsers().GetTotal(),
			DailyActive:  resp.GetUsers().GetDailyActive(),
			WeeklyActive: resp.GetUsers().GetWeeklyActive(),
			DAUWAURate:   resp.GetUsers().GetDauWauRate(),
			WAUTotalRate: resp.GetUsers().GetWauTotalRate(),
		},
		Proxies: models.AdminDashboardProxies{
			Total:              resp.GetProxies().GetTotal(),
			Active:             resp.GetProxies().GetActive(),
			Available:          resp.GetProxies().GetAvailable(),
			Cooling:            resp.GetProxies().GetCooling(),
			Saturated:          resp.GetProxies().GetSaturated(),
			HighRisk:           resp.GetProxies().GetHighRisk(),
			RecentSuccess:      resp.GetProxies().GetRecentSuccess(),
			RecentFailure:      resp.GetProxies().GetRecentFailure(),
			RecentFailureRate:  resp.GetProxies().GetRecentFailureRate(),
			TopErrorCategories: adminDashboardCountsResponse(resp.GetProxies().GetTopErrorCategories()),
		},
		ProxySource: models.AdminDashboardProxySource{
			Healthy:           resp.GetProxySource().GetHealthy(),
			Mode:              resp.GetProxySource().GetMode(),
			Message:           resp.GetProxySource().GetMessage(),
			DynamicConfigured: resp.GetProxySource().GetDynamicConfigured(),
			ProxyLeaseID:      resp.GetProxySource().GetProxyLeaseId(),
			ProxyExpireAt:     resp.GetProxySource().GetProxyExpireAt(),
		},
		ProxyPolicy: models.AdminDashboardProxyPolicy{
			PrimarySource:   resp.GetProxyPolicy().GetPrimarySource(),
			FallbackSource:  resp.GetProxyPolicy().GetFallbackSource(),
			FallbackEnabled: resp.GetProxyPolicy().GetFallbackEnabled(),
		},
		Cookies: models.AdminDashboardCookies{
			Total:   resp.GetCookies().GetTotal(),
			Active:  resp.GetCookies().GetActive(),
			Expired: resp.GetCookies().GetExpired(),
			Frozen:  resp.GetCookies().GetFrozen(),
		},
		Billing: models.AdminDashboardBilling{
			ShortfallCount: resp.GetBilling().GetShortfallCount(),
		},
		Exceptions: adminDashboardExceptionsResponse(resp.GetExceptions()),
	})
}

func adminDashboardCountsResponse(items []*pb.AdminDashboardProxyErrorCategory) []models.AdminDashboardCount {
	result := make([]models.AdminDashboardCount, 0, len(items))
	for _, item := range items {
		result = append(result, models.AdminDashboardCount{
			Key:   item.GetKey(),
			Count: item.GetCount(),
		})
	}
	return result
}

func adminDashboardExceptionsResponse(items []*pb.AdminDashboardException) []models.AdminDashboardException {
	result := make([]models.AdminDashboardException, 0, len(items))
	for _, item := range items {
		result = append(result, models.AdminDashboardException{
			Area:        item.GetArea(),
			Severity:    item.GetSeverity(),
			Message:     item.GetMessage(),
			ActionLabel: item.GetActionLabel(),
			ActionHref:  item.GetActionHref(),
		})
	}
	return result
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
