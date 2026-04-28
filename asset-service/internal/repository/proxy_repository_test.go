package repository

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"

	"youdlp/asset-service/internal/models"
)

func TestAcquireAvailableProxyExcludingAllocatesAtomically(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create sqlmock failed: %v", err)
	}
	defer db.Close()

	protocol := models.ProxyProtocolHTTP
	region := "US"
	excludedID := int64(7)
	now := time.Now()
	query := `(?s)WITH candidate AS \(\s*SELECT id\s+FROM proxies\s+WHERE status = \$1 AND deleted_at IS NULL AND \(cooldown_until IS NULL OR cooldown_until <= \$2\) AND risk_score < \$3 AND active_task_count < max_concurrent AND protocol = \$4 AND region = \$5 AND id <> \$6\s+ORDER BY risk_score ASC,\s+last_used_at ASC NULLS FIRST,\s+priority DESC,\s+success_count DESC,\s+consecutive_fail_count ASC,\s+id ASC\s+LIMIT 1\s+FOR UPDATE SKIP LOCKED\s+\)\s+UPDATE proxies p\s+SET last_used_at = \$7,\s+updated_at = \$7\s+FROM candidate\s+WHERE p\.id = candidate\.id\s+RETURNING p\.id, p\.host`

	mock.ExpectQuery(query).
		WithArgs(models.ProxyStatusActive, sqlmock.AnyArg(), models.ProxyRiskExcludeThreshold, protocol, region, excludedID, sqlmock.AnyArg()).
		WillReturnRows(proxyRepositoryRows().AddRow(
			int64(8), nil, "127.0.0.8", 8080, nil, nil, string(models.ProxyProtocolHTTP), region,
			3, nil, nil, models.ProxyStatusActive, nil, nil, 2, 1, now, nil, 0, 0, nil, nil, 1, 0, nil, now, now,
		))

	repo := NewProxyRepository(db)
	proxy, err := repo.AcquireAvailableProxyExcluding(context.Background(), &protocol, &region, &excludedID)
	if err != nil {
		t.Fatalf("AcquireAvailableProxyExcluding returned error: %v", err)
	}
	if proxy == nil {
		t.Fatal("expected proxy, got nil")
	}
	if proxy.ID != 8 {
		t.Fatalf("expected proxy id 8, got %d", proxy.ID)
	}
	if proxy.LastUsedAt == nil || !proxy.LastUsedAt.Equal(now) {
		t.Fatalf("expected allocated last_used_at %v, got %v", now, proxy.LastUsedAt)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func TestAcquireAvailableProxyReturnsNilWhenNoCandidate(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create sqlmock failed: %v", err)
	}
	defer db.Close()

	query := `(?s)WITH candidate AS \(\s*SELECT id\s+FROM proxies\s+WHERE status = \$1 AND deleted_at IS NULL AND \(cooldown_until IS NULL OR cooldown_until <= \$2\) AND risk_score < \$3 AND active_task_count < max_concurrent\s+ORDER BY risk_score ASC,\s+last_used_at ASC NULLS FIRST,\s+priority DESC,\s+success_count DESC,\s+consecutive_fail_count ASC,\s+id ASC\s+LIMIT 1\s+FOR UPDATE SKIP LOCKED\s+\)\s+UPDATE proxies p\s+SET last_used_at = \$4,\s+updated_at = \$4\s+FROM candidate\s+WHERE p\.id = candidate\.id\s+RETURNING p\.id, p\.host`

	mock.ExpectQuery(query).
		WithArgs(models.ProxyStatusActive, sqlmock.AnyArg(), models.ProxyRiskExcludeThreshold, sqlmock.AnyArg()).
		WillReturnError(sql.ErrNoRows)

	repo := NewProxyRepository(db)
	proxy, err := repo.AcquireAvailableProxy(context.Background(), nil, nil)
	if err != nil {
		t.Fatalf("AcquireAvailableProxy returned error: %v", err)
	}
	if proxy != nil {
		t.Fatalf("expected nil proxy, got %+v", proxy)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func TestUpdateUsageOnlyUpdatesResultCounters(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create sqlmock failed: %v", err)
	}
	defer db.Close()

	mock.ExpectExec(`UPDATE proxies\s+SET success_count = success_count \+ 1,\s+consecutive_fail_count = 0,\s+risk_score = GREATEST\(risk_score - 5, 0\),\s+updated_at = \$2\s+WHERE id = \$1`).
		WithArgs(int64(8), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))

	repo := NewProxyRepository(db)
	if err := repo.UpdateUsage(context.Background(), 8, true, "", 0, nil); err != nil {
		t.Fatalf("UpdateUsage returned error: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func TestListUsageEventsSummarizesFailureCategoriesOnly(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create sqlmock failed: %v", err)
	}
	defer db.Close()

	start := time.Now().Add(-24 * time.Hour).Truncate(time.Second)
	end := time.Now().Truncate(time.Second)

	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM proxy_usage_events e WHERE e\.created_at >= \$1 AND e\.created_at <= \$2`).
		WithArgs(start, end).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(int64(3)))
	mock.ExpectQuery(`(?s)SELECT\s+COUNT\(\*\) FILTER \(WHERE e\.success = TRUE\),\s+COUNT\(\*\) FILTER \(WHERE e\.success = FALSE\)\s+FROM proxy_usage_events e\s+WHERE e\.created_at >= \$1 AND e\.created_at <= \$2`).
		WithArgs(start, end).
		WillReturnRows(sqlmock.NewRows([]string{"success_count", "failure_count"}).AddRow(int64(2), int64(1)))
	mock.ExpectQuery(`(?s)SELECT COALESCE\(e\.error_category, 'unknown'\) AS key, COUNT\(\*\) AS count\s+FROM proxy_usage_events e\s+WHERE \(e\.created_at >= \$1 AND e\.created_at <= \$2\) AND e\.success = FALSE`).
		WithArgs(start, end).
		WillReturnRows(sqlmock.NewRows([]string{"key", "count"}).AddRow(models.ErrorCategoryRateLimited, int64(1)))
	mock.ExpectQuery(`(?s)SELECT e\.stage AS key, COUNT\(\*\) AS count\s+FROM proxy_usage_events e\s+WHERE e\.created_at >= \$1 AND e\.created_at <= \$2`).
		WithArgs(start, end).
		WillReturnRows(sqlmock.NewRows([]string{"key", "count"}).AddRow("parse", int64(3)))
	mock.ExpectQuery(`(?s)SELECT COALESCE\(e\.platform, 'unknown'\) AS key, COUNT\(\*\) AS count\s+FROM proxy_usage_events e\s+WHERE e\.created_at >= \$1 AND e\.created_at <= \$2`).
		WithArgs(start, end).
		WillReturnRows(sqlmock.NewRows([]string{"key", "count"}).AddRow("youtube", int64(3)))
	mock.ExpectQuery(`(?s)SELECT\s+e\.id,.*FROM proxy_usage_events e\s+LEFT JOIN proxies p ON p\.id = e\.proxy_id\s+WHERE e\.created_at >= \$1 AND e\.created_at <= \$2\s+ORDER BY e\.created_at DESC, e\.id DESC\s+LIMIT \$3 OFFSET \$4`).
		WithArgs(start, end, 20, 0).
		WillReturnRows(proxyUsageEventRows())

	repo := NewProxyRepository(db)
	result, err := repo.ListUsageEvents(context.Background(), models.ProxyUsageEventFilter{
		StartTime: start,
		EndTime:   end,
		Page:      1,
		PageSize:  20,
		SortOrder: models.ProxyUsageSortOrderDesc,
	})
	if err != nil {
		t.Fatalf("ListUsageEvents returned error: %v", err)
	}
	if result.Total != 3 {
		t.Fatalf("expected total 3, got %d", result.Total)
	}
	if got := result.Summary.CategoryCounts; len(got) != 1 || got[0].Key != models.ErrorCategoryRateLimited || got[0].Count != 1 {
		t.Fatalf("expected failure-only category counts, got %#v", got)
	}
	if result.Summary.FailureRate != float64(1)/float64(3) {
		t.Fatalf("expected failure rate 1/3, got %f", result.Summary.FailureRate)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func proxyRepositoryRows() *sqlmock.Rows {
	return sqlmock.NewRows([]string{
		"id", "host", "ip", "port", "username", "password", "protocol", "region", "priority",
		"platform_tags", "remark", "status", "last_check_at", "last_check_result",
		"success_count", "fail_count", "last_used_at", "cooldown_until", "consecutive_fail_count",
		"risk_score", "last_error_category", "last_fail_at", "max_concurrent", "active_task_count",
		"deleted_at", "created_at", "updated_at",
	})
}

func proxyUsageEventRows() *sqlmock.Rows {
	return sqlmock.NewRows([]string{
		"id", "task_id", "proxy_id", "proxy_lease_id", "source_type", "stage", "platform",
		"success", "error_category", "error_message", "created_at", "proxy_host", "proxy_port",
		"proxy_protocol", "proxy_region", "proxy_risk_score", "proxy_cooldown_until",
		"proxy_active_task_count", "proxy_max_concurrent",
	})
}
