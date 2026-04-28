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
	query := `(?s)WITH candidate AS \(\s*SELECT id\s+FROM proxies\s+WHERE status = \$1 AND deleted_at IS NULL AND protocol = \$2 AND region = \$3 AND id <> \$4\s+ORDER BY last_used_at ASC NULLS FIRST,\s+priority DESC,\s+success_count DESC,\s+id ASC\s+LIMIT 1\s+FOR UPDATE SKIP LOCKED\s+\)\s+UPDATE proxies p\s+SET last_used_at = \$5,\s+updated_at = \$5\s+FROM candidate\s+WHERE p\.id = candidate\.id\s+RETURNING p\.id, p\.host`

	mock.ExpectQuery(query).
		WithArgs(models.ProxyStatusActive, protocol, region, excludedID, sqlmock.AnyArg()).
		WillReturnRows(proxyRepositoryRows().AddRow(
			int64(8), nil, "127.0.0.8", 8080, nil, nil, string(models.ProxyProtocolHTTP), region,
			3, nil, nil, models.ProxyStatusActive, nil, nil, 2, 1, now, nil, now, now,
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

	query := `(?s)WITH candidate AS \(\s*SELECT id\s+FROM proxies\s+WHERE status = \$1 AND deleted_at IS NULL\s+ORDER BY last_used_at ASC NULLS FIRST,\s+priority DESC,\s+success_count DESC,\s+id ASC\s+LIMIT 1\s+FOR UPDATE SKIP LOCKED\s+\)\s+UPDATE proxies p\s+SET last_used_at = \$2,\s+updated_at = \$2\s+FROM candidate\s+WHERE p\.id = candidate\.id\s+RETURNING p\.id, p\.host`

	mock.ExpectQuery(query).
		WithArgs(models.ProxyStatusActive, sqlmock.AnyArg()).
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

	mock.ExpectExec(`UPDATE proxies\s+SET success_count = success_count \+ 1, updated_at = \$2\s+WHERE id = \$1`).
		WithArgs(int64(8), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))

	repo := NewProxyRepository(db)
	if err := repo.UpdateUsage(context.Background(), 8, true); err != nil {
		t.Fatalf("UpdateUsage returned error: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func proxyRepositoryRows() *sqlmock.Rows {
	return sqlmock.NewRows([]string{
		"id", "host", "ip", "port", "username", "password", "protocol", "region", "priority",
		"platform_tags", "remark", "status", "last_check_at", "last_check_result",
		"success_count", "fail_count", "last_used_at", "deleted_at", "created_at", "updated_at",
	})
}
