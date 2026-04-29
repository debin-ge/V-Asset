package service

import (
	"context"

	"youdlp/admin-service/internal/models"
	pb "youdlp/admin-service/proto"
)

type StatsService struct {
	authClient  pb.AuthServiceClient
	assetClient pb.AssetServiceClient
}

func NewStatsService(authClient pb.AuthServiceClient, assetClient pb.AssetServiceClient) *StatsService {
	return &StatsService{
		authClient:  authClient,
		assetClient: assetClient,
	}
}

func (s *StatsService) GetOverview(ctx context.Context) (*models.OverviewResponse, error) {
	userResp, err := s.authClient.GetPlatformUserStats(ctx, &pb.GetPlatformUserStatsRequest{})
	if err != nil {
		return nil, err
	}

	statsResp, err := s.assetClient.GetPlatformStats(ctx, &pb.GetPlatformStatsRequest{})
	if err != nil {
		return nil, err
	}

	proxiesResp, err := s.assetClient.ListProxies(ctx, &pb.ListProxiesRequest{Status: -1, Page: 1, PageSize: 1})
	if err != nil {
		return nil, err
	}

	activeProxiesResp, err := s.assetClient.ListProxies(ctx, &pb.ListProxiesRequest{Status: 0, Page: 1, PageSize: 1})
	if err != nil {
		return nil, err
	}

	return &models.OverviewResponse{
		TotalUsers:          userResp.TotalUsers,
		DailyActiveUsers:    statsResp.DailyActiveUsers,
		WeeklyActiveUsers:   statsResp.WeeklyActiveUsers,
		TotalDownloads:      statsResp.TotalDownloads,
		DownloadsToday:      statsResp.DownloadsToday,
		SuccessDownloads:    statsResp.SuccessDownloads,
		FailedDownloads:     statsResp.FailedDownloads,
		ActiveManualProxies: activeProxiesResp.Total,
		TotalManualProxies:  proxiesResp.Total,
	}, nil
}

func (s *StatsService) GetRequestTrend(ctx context.Context, granularity string, limit int32) (*models.RequestTrendResponse, error) {
	resp, err := s.assetClient.GetRequestTrend(ctx, &pb.GetRequestTrendRequest{
		Granularity: granularity,
		Limit:       limit,
	})
	if err != nil {
		return nil, err
	}

	points := make([]models.TrendPoint, 0, len(resp.Points))
	for _, point := range resp.Points {
		points = append(points, models.TrendPoint{
			Label:        point.Label,
			Count:        point.Count,
			TotalCount:   point.TotalCount,
			SuccessCount: point.SuccessCount,
			FailedCount:  point.FailedCount,
			SuccessRate:  point.SuccessRate,
		})
	}

	return &models.RequestTrendResponse{
		Granularity: resp.Granularity,
		Points:      points,
	}, nil
}

func (s *StatsService) GetDashboardHealth(ctx context.Context) (*models.DashboardHealthResponse, error) {
	userResp, err := s.authClient.GetPlatformUserStats(ctx, &pb.GetPlatformUserStatsRequest{})
	if err != nil {
		return nil, err
	}

	assetResp, err := s.assetClient.GetDashboardHealth(ctx, &pb.GetDashboardHealthRequest{})
	if err != nil {
		return nil, err
	}

	downloads := dashboardDownloadsFromProto(assetResp.GetDownloads())
	users := models.DashboardUsers{
		Total:        userResp.GetTotalUsers(),
		DailyActive:  assetResp.GetUsers().GetDailyActive(),
		WeeklyActive: assetResp.GetUsers().GetWeeklyActive(),
	}
	users.DAUWAURate = safeRate(users.DailyActive, users.WeeklyActive)
	users.WAUTotalRate = safeRate(users.WeeklyActive, users.Total)

	proxies := dashboardProxiesFromProto(assetResp.GetProxies())
	proxySource := dashboardProxySourceFromProto(assetResp.GetProxySource())
	proxyPolicy := dashboardProxyPolicyFromProto(assetResp.GetProxyPolicy())
	cookies := dashboardCookiesFromProto(assetResp.GetCookies())
	billing := dashboardBillingFromProto(assetResp.GetBilling())

	return &models.DashboardHealthResponse{
		GeneratedAt: assetResp.GetGeneratedAt(),
		Downloads:   downloads,
		Users:       users,
		Proxies:     proxies,
		ProxySource: proxySource,
		ProxyPolicy: proxyPolicy,
		Cookies:     cookies,
		Billing:     billing,
		Exceptions:  buildDashboardExceptions(downloads, proxies, proxySource, cookies, billing),
	}, nil
}

func dashboardDownloadsFromProto(item *pb.AssetDashboardDownloads) models.DashboardDownloads {
	if item == nil {
		return models.DashboardDownloads{}
	}
	return models.DashboardDownloads{
		Total:        item.GetTotal(),
		TodayTotal:   item.GetTodayTotal(),
		SuccessTotal: item.GetSuccessTotal(),
		FailedTotal:  item.GetFailedTotal(),
		SuccessRate:  item.GetSuccessRate(),
		FailureRate:  item.GetFailureRate(),
	}
}

func dashboardProxiesFromProto(item *pb.AssetDashboardProxies) models.DashboardProxies {
	if item == nil {
		return models.DashboardProxies{}
	}
	return models.DashboardProxies{
		Total:              item.GetTotal(),
		Active:             item.GetActive(),
		Available:          item.GetAvailable(),
		Cooling:            item.GetCooling(),
		Saturated:          item.GetSaturated(),
		HighRisk:           item.GetHighRisk(),
		RecentSuccess:      item.GetRecentSuccess(),
		RecentFailure:      item.GetRecentFailure(),
		RecentFailureRate:  item.GetRecentFailureRate(),
		TopErrorCategories: dashboardCountsFromProto(item.GetTopErrorCategories()),
	}
}

func dashboardCountsFromProto(items []*pb.AssetDashboardProxyErrorCategory) []models.DashboardCount {
	result := make([]models.DashboardCount, 0, len(items))
	for _, item := range items {
		result = append(result, models.DashboardCount{Key: item.GetKey(), Count: item.GetCount()})
	}
	return result
}

func dashboardProxySourceFromProto(item *pb.AssetDashboardProxySource) models.DashboardProxySource {
	if item == nil {
		return models.DashboardProxySource{}
	}
	return models.DashboardProxySource{
		Healthy:           item.GetHealthy(),
		Mode:              item.GetMode(),
		Message:           item.GetMessage(),
		DynamicConfigured: item.GetDynamicConfigured(),
		ProxyLeaseID:      item.GetProxyLeaseId(),
		ProxyExpireAt:     item.GetProxyExpireAt(),
	}
}

func dashboardProxyPolicyFromProto(item *pb.AssetDashboardProxyPolicy) models.DashboardProxyPolicy {
	if item == nil {
		return models.DashboardProxyPolicy{}
	}
	return models.DashboardProxyPolicy{
		PrimarySource:   item.GetPrimarySource(),
		FallbackSource:  item.GetFallbackSource(),
		FallbackEnabled: item.GetFallbackEnabled(),
	}
}

func dashboardCookiesFromProto(item *pb.AssetDashboardCookies) models.DashboardCookies {
	if item == nil {
		return models.DashboardCookies{}
	}
	return models.DashboardCookies{
		Total:   item.GetTotal(),
		Active:  item.GetActive(),
		Expired: item.GetExpired(),
		Frozen:  item.GetFrozen(),
	}
}

func dashboardBillingFromProto(item *pb.AssetDashboardBilling) models.DashboardBilling {
	if item == nil {
		return models.DashboardBilling{}
	}
	return models.DashboardBilling{ShortfallCount: item.GetShortfallCount()}
}

func safeRate(numerator, denominator int64) float64 {
	if denominator <= 0 {
		return 0
	}
	return float64(numerator) / float64(denominator)
}

func buildDashboardExceptions(
	downloads models.DashboardDownloads,
	proxies models.DashboardProxies,
	proxySource models.DashboardProxySource,
	cookies models.DashboardCookies,
	billing models.DashboardBilling,
) []models.DashboardException {
	exceptions := make([]models.DashboardException, 0)

	if downloads.Total > 0 && downloads.SuccessRate < 0.90 {
		exceptions = append(exceptions, models.DashboardException{
			Area:        "Downloads",
			Severity:    "critical",
			Message:     "Overall download success rate is below 90%.",
			ActionLabel: "Review trend",
		})
	} else if downloads.Total > 0 && downloads.SuccessRate < 0.95 {
		exceptions = append(exceptions, models.DashboardException{
			Area:        "Downloads",
			Severity:    "warning",
			Message:     "Overall download success rate is below 95%.",
			ActionLabel: "Review trend",
		})
	}

	if !proxySource.Healthy {
		exceptions = append(exceptions, models.DashboardException{
			Area:        "Proxy Source",
			Severity:    "critical",
			Message:     "Proxy source is currently unhealthy.",
			ActionLabel: "Manage proxies",
			ActionHref:  "/proxies",
		})
	}
	if proxies.Total == 0 {
		exceptions = append(exceptions, models.DashboardException{
			Area:        "Manual Pool",
			Severity:    "critical",
			Message:     "No manual proxies are configured.",
			ActionLabel: "Add proxies",
			ActionHref:  "/proxies",
		})
	} else if proxies.Available == 0 {
		exceptions = append(exceptions, models.DashboardException{
			Area:        "Manual Pool",
			Severity:    "warning",
			Message:     "Manual proxy pool has no currently selectable proxies.",
			ActionLabel: "Manage proxies",
			ActionHref:  "/proxies",
		})
	}
	if proxies.HighRisk > 0 {
		exceptions = append(exceptions, models.DashboardException{
			Area:        "Proxy Risk",
			Severity:    "warning",
			Message:     "High-risk proxies are present in the manual pool.",
			ActionLabel: "Review proxies",
			ActionHref:  "/proxies",
		})
	}
	if billing.ShortfallCount > 0 {
		exceptions = append(exceptions, models.DashboardException{
			Area:        "Billing",
			Severity:    "warning",
			Message:     "Billing shortfall orders are awaiting reconciliation.",
			ActionLabel: "Review billing",
			ActionHref:  "/billing",
		})
	}
	if cookies.Total > 0 && cookies.Active == 0 {
		exceptions = append(exceptions, models.DashboardException{
			Area:        "Cookies",
			Severity:    "warning",
			Message:     "No active cookies are currently available.",
			ActionLabel: "Review cookies",
			ActionHref:  "/cookies",
		})
	}

	return exceptions
}

func (s *StatsService) GetUserStats(ctx context.Context) (*models.UserStatsResponse, error) {
	userResp, err := s.authClient.GetPlatformUserStats(ctx, &pb.GetPlatformUserStatsRequest{})
	if err != nil {
		return nil, err
	}

	statsResp, err := s.assetClient.GetPlatformStats(ctx, &pb.GetPlatformStatsRequest{})
	if err != nil {
		return nil, err
	}

	return &models.UserStatsResponse{
		TotalUsers:        userResp.TotalUsers,
		DailyActiveUsers:  statsResp.DailyActiveUsers,
		WeeklyActiveUsers: statsResp.WeeklyActiveUsers,
	}, nil
}
