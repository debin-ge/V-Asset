package repository

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"

	"youdlp/asset-service/internal/models"
)

func TestGetDashboardDownloadsComputesRates(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create sqlmock failed: %v", err)
	}
	defer db.Close()

	mock.ExpectQuery(`(?s)SELECT\s+COUNT\(\*\),\s+COUNT\(\*\) FILTER \(WHERE created_at >= CURRENT_DATE\),\s+COUNT\(\*\) FILTER \(WHERE status = \$1\),\s+COUNT\(\*\) FILTER \(WHERE status = \$2\)\s+FROM download_history`).
		WithArgs(models.StatusCompleted, models.StatusFailed).
		WillReturnRows(sqlmock.NewRows([]string{"total", "today_total", "success_total", "failed_total"}).AddRow(int64(10), int64(3), int64(8), int64(2)))

	repo := NewHistoryRepository(db)
	stats, err := repo.GetDashboardDownloads(context.Background())
	if err != nil {
		t.Fatalf("GetDashboardDownloads returned error: %v", err)
	}

	if stats.Total != 10 || stats.TodayTotal != 3 || stats.SuccessTotal != 8 || stats.FailedTotal != 2 {
		t.Fatalf("unexpected download stats: %+v", stats)
	}
	if stats.SuccessRate != 0.8 || stats.FailureRate != 0.2 {
		t.Fatalf("unexpected rates: success=%f failure=%f", stats.SuccessRate, stats.FailureRate)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func TestGetRequestTrendReturnsFixedBucketsWithBreakdown(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create sqlmock failed: %v", err)
	}
	defer db.Close()

	mock.ExpectQuery(`(?s)WITH series AS .*generate_series.*DATE_TRUNC\('day', created_at\).*LEFT JOIN aggregated`).
		WithArgs(7, models.StatusCompleted, models.StatusFailed).
		WillReturnRows(sqlmock.NewRows([]string{"label", "total_count", "success_count", "failed_count"}).
			AddRow("2026-04-28", int64(0), int64(0), int64(0)).
			AddRow("2026-04-29", int64(10), int64(8), int64(2)))

	repo := NewHistoryRepository(db)
	points, err := repo.GetRequestTrend(context.Background(), "day", 7)
	if err != nil {
		t.Fatalf("GetRequestTrend returned error: %v", err)
	}
	if len(points) != 2 {
		t.Fatalf("expected 2 trend points, got %d", len(points))
	}
	if points[0].Count != 0 || points[0].TotalCount != 0 || points[0].SuccessRate != 0 {
		t.Fatalf("expected zero-filled bucket, got %+v", points[0])
	}
	point := points[1]
	if point.Count != 10 || point.TotalCount != 10 || point.SuccessCount != 8 || point.FailedCount != 2 || point.SuccessRate != 0.8 {
		t.Fatalf("unexpected trend point: %+v", point)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func TestGetDashboardProxyStatsAggregatesPoolAndRecentUsage(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create sqlmock failed: %v", err)
	}
	defer db.Close()

	recentSince := time.Now().Add(-24 * time.Hour).Truncate(time.Second)
	mock.ExpectQuery(`(?s)SELECT\s+COUNT\(\*\) FILTER \(WHERE deleted_at IS NULL\),\s+COUNT\(\*\) FILTER \(WHERE deleted_at IS NULL AND status = \$1\),.*FROM proxies`).
		WithArgs(models.ProxyStatusActive, sqlmock.AnyArg(), models.ProxyRiskExcludeThreshold, 70).
		WillReturnRows(sqlmock.NewRows([]string{"total", "active", "available", "cooling", "saturated", "high_risk"}).AddRow(int64(5), int64(4), int64(3), int64(1), int64(2), int64(1)))
	mock.ExpectQuery(`(?s)SELECT\s+COUNT\(\*\) FILTER \(WHERE success = TRUE\),\s+COUNT\(\*\) FILTER \(WHERE success = FALSE\)\s+FROM proxy_usage_events\s+WHERE created_at >= \$1`).
		WithArgs(recentSince).
		WillReturnRows(sqlmock.NewRows([]string{"recent_success", "recent_failure"}).AddRow(int64(7), int64(3)))
	mock.ExpectQuery(`(?s)SELECT COALESCE\(error_category, 'unknown'\) AS key, COUNT\(\*\) AS count\s+FROM proxy_usage_events\s+WHERE created_at >= \$1\s+AND success = FALSE\s+GROUP BY key\s+ORDER BY count DESC, key ASC\s+LIMIT 10`).
		WithArgs(recentSince).
		WillReturnRows(sqlmock.NewRows([]string{"key", "count"}).
			AddRow(models.ErrorCategoryRateLimited, int64(2)).
			AddRow("unknown", int64(1)))

	repo := NewProxyRepository(db)
	stats, err := repo.GetDashboardStats(context.Background(), recentSince)
	if err != nil {
		t.Fatalf("GetDashboardStats returned error: %v", err)
	}

	if stats.Total != 5 || stats.Active != 4 || stats.Available != 3 || stats.Cooling != 1 || stats.Saturated != 2 || stats.HighRisk != 1 {
		t.Fatalf("unexpected proxy pool stats: %+v", stats)
	}
	if stats.RecentSuccess != 7 || stats.RecentFailure != 3 || stats.RecentFailureRate != 0.3 {
		t.Fatalf("unexpected recent proxy usage stats: %+v", stats)
	}
	if len(stats.TopErrorCategories) != 2 || stats.TopErrorCategories[0].Key != models.ErrorCategoryRateLimited || stats.TopErrorCategories[0].Count != 2 {
		t.Fatalf("unexpected top error categories: %#v", stats.TopErrorCategories)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func TestGetDashboardCookieStatsCountsActiveExpiredFrozen(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create sqlmock failed: %v", err)
	}
	defer db.Close()

	mock.ExpectQuery(`(?s)SELECT\s+COUNT\(\*\),\s+COUNT\(\*\) FILTER.*FROM cookies`).
		WithArgs(sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"total", "active", "expired", "frozen"}).AddRow(int64(6), int64(3), int64(2), int64(1)))

	repo := NewCookieRepository(db)
	stats, err := repo.GetDashboardStats(context.Background())
	if err != nil {
		t.Fatalf("GetDashboardStats returned error: %v", err)
	}

	if stats.Total != 6 || stats.Active != 3 || stats.Expired != 2 || stats.Frozen != 1 {
		t.Fatalf("unexpected cookie stats: %+v", stats)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func TestCountShortfallOrders(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create sqlmock failed: %v", err)
	}
	defer db.Close()

	mock.ExpectQuery(`(?s)SELECT COUNT\(\*\)\s+FROM billing_charge_orders\s+WHERE status = \$1\s+AND shortfall_yuan > 0`).
		WithArgs(models.BillingOrderStatusAwaitingShortfall).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(int64(4)))

	repo := NewBillingRepository(db)
	count, err := repo.CountShortfallOrders(context.Background())
	if err != nil {
		t.Fatalf("CountShortfallOrders returned error: %v", err)
	}
	if count != 4 {
		t.Fatalf("expected shortfall count 4, got %d", count)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}
