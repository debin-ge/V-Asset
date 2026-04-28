package service

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"

	"youdlp/asset-service/internal/config"
	"youdlp/asset-service/internal/models"
	"youdlp/asset-service/internal/repository"
)

func TestReportUsageMarksParseFailureBindingFailed(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create sqlmock failed: %v", err)
	}
	defer db.Close()

	taskID := "task-1"
	proxyID := int64(7)
	now := time.Now()

	mock.ExpectQuery(`SELECT id, task_id, source_type`).
		WithArgs(taskID).
		WillReturnRows(taskProxyBindingRows().AddRow(
			int64(1), taskID, string(models.ProxySourceTypeManualPool), int64(10), proxyID, "static-7",
			"http://proxy-a:8080", string(models.ProxyProtocolHTTP), nil, "youtube", nil,
			string(models.TaskProxyBindStatusBound), false, nil, nil, nil, nil, now, now,
		))
	mock.ExpectExec(`UPDATE task_proxy_bindings\s+SET last_report_stage`).
		WithArgs(taskID, proxyUsageStageParse, false, sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`UPDATE task_proxy_bindings\s+SET bind_status`).
		WithArgs(taskID, models.TaskProxyBindStatusFailed, sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`UPDATE proxies\s+SET fail_count`).
		WithArgs(proxyID, sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))

	svc := newProxyServiceForTest(db)
	if err := svc.ReportUsage(context.Background(), taskID, "static-7", proxyUsageStageParse, false); err != nil {
		t.Fatalf("ReportUsage returned error: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func TestGetAvailableProxyUsesAtomicManualAcquire(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create sqlmock failed: %v", err)
	}
	defer db.Close()

	now := time.Now()

	mock.ExpectQuery(atomicAcquireNoFilterQuery()).
		WithArgs(models.ProxyStatusActive, sqlmock.AnyArg()).
		WillReturnRows(proxyRows().AddRow(
			int64(8), nil, "127.0.0.8", 8080, nil, nil, string(models.ProxyProtocolHTTP), nil,
			1, nil, nil, models.ProxyStatusActive, nil, nil, 0, 0, now, nil, now, now,
		))

	svc := newProxyServiceForTest(db)
	proxyURL, leaseID, expireAt, err := svc.GetAvailableProxy(context.Background(), nil, nil)
	if err != nil {
		t.Fatalf("GetAvailableProxy returned error: %v", err)
	}
	if proxyURL != "http://127.0.0.8:8080" {
		t.Fatalf("expected proxy url http://127.0.0.8:8080, got %s", proxyURL)
	}
	if leaseID != "static-8" {
		t.Fatalf("expected lease id static-8, got %s", leaseID)
	}
	if expireAt != "" {
		t.Fatalf("expected empty expire_at, got %s", expireAt)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func TestAcquireProxyForTaskCreatesManualBindingWithAtomicAcquire(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create sqlmock failed: %v", err)
	}
	defer db.Close()

	taskID := "task-new"
	proxyID := int64(8)
	policyID := int64(10)
	now := time.Now()

	mock.ExpectQuery(`SELECT id, task_id, source_type`).
		WithArgs(taskID).
		WillReturnError(sql.ErrNoRows)
	mock.ExpectQuery(`FROM proxy_source_policies`).
		WithArgs("global", "").
		WillReturnRows(proxyPolicyRows().AddRow(
			policyID, "global", nil, string(models.ProxySourceTypeManualPool), nil, false,
			3000, 2, 60, 600, "lru", 0, now, now,
		))
	mock.ExpectQuery(atomicAcquireNoFilterQuery()).
		WithArgs(models.ProxyStatusActive, sqlmock.AnyArg()).
		WillReturnRows(proxyRows().AddRow(
			proxyID, nil, "127.0.0.8", 8080, nil, nil, string(models.ProxyProtocolHTTP), nil,
			1, nil, nil, models.ProxyStatusActive, nil, nil, 0, 0, now, nil, now, now,
		))
	mock.ExpectExec(`INSERT INTO task_proxy_bindings`).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(`SELECT id, task_id, source_type`).
		WithArgs(taskID).
		WillReturnRows(taskProxyBindingRows().AddRow(
			int64(1), taskID, string(models.ProxySourceTypeManualPool), policyID, proxyID, "static-8",
			"http://127.0.0.8:8080", string(models.ProxyProtocolHTTP), nil, nil, nil,
			string(models.TaskProxyBindStatusBound), false, nil, nil, nil, nil, now, now,
		))

	svc := newProxyServiceForTest(db)
	binding, err := svc.AcquireProxyForTask(context.Background(), taskID, nil, nil, nil)
	if err != nil {
		t.Fatalf("AcquireProxyForTask returned error: %v", err)
	}
	if binding.ProxyID == nil || *binding.ProxyID != proxyID {
		t.Fatalf("expected proxy id %d, got %+v", proxyID, binding.ProxyID)
	}
	if binding.ProxyLeaseID == nil || *binding.ProxyLeaseID != "static-8" {
		t.Fatalf("expected static lease id, got %+v", binding.ProxyLeaseID)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func TestAcquireProxyForTaskRebindsFailedManualProxy(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create sqlmock failed: %v", err)
	}
	defer db.Close()

	taskID := "task-2"
	oldProxyID := int64(7)
	newProxyID := int64(8)
	policyID := int64(10)
	now := time.Now()

	mock.ExpectQuery(`SELECT id, task_id, source_type`).
		WithArgs(taskID).
		WillReturnRows(taskProxyBindingRows().AddRow(
			int64(1), taskID, string(models.ProxySourceTypeManualPool), policyID, oldProxyID, "static-7",
			"http://proxy-a:8080", string(models.ProxyProtocolHTTP), nil, "youtube", nil,
			string(models.TaskProxyBindStatusFailed), false, nil, proxyUsageStageParse, false, now, now, now,
		))
	mock.ExpectQuery(`FROM proxy_source_policies`).
		WithArgs("platform", "youtube").
		WillReturnRows(sqlmock.NewRows(proxyPolicyColumns()))
	mock.ExpectQuery(`FROM proxy_source_policies`).
		WithArgs("global", "").
		WillReturnRows(proxyPolicyRows().AddRow(
			policyID, "global", nil, string(models.ProxySourceTypeManualPool), nil, false,
			3000, 2, 60, 600, "lru", 0, now, now,
		))
	mock.ExpectQuery(atomicAcquireExcludingQuery()).
		WithArgs(models.ProxyStatusActive, oldProxyID, sqlmock.AnyArg()).
		WillReturnRows(proxyRows().AddRow(
			newProxyID, nil, "127.0.0.8", 8080, nil, nil, string(models.ProxyProtocolHTTP), nil,
			1, nil, nil, models.ProxyStatusActive, nil, nil, 0, 0, nil, nil, now, now,
		))
	mock.ExpectExec(`UPDATE task_proxy_bindings\s+SET source_type`).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(`SELECT id, task_id, source_type`).
		WithArgs(taskID).
		WillReturnRows(taskProxyBindingRows().AddRow(
			int64(1), taskID, string(models.ProxySourceTypeManualPool), policyID, newProxyID, "static-8",
			"http://127.0.0.8:8080", string(models.ProxyProtocolHTTP), nil, "youtube", nil,
			string(models.TaskProxyBindStatusBound), false, nil, nil, nil, nil, now, now,
		))

	platform := "youtube"
	svc := newProxyServiceForTest(db)
	binding, err := svc.AcquireProxyForTask(context.Background(), taskID, nil, nil, &platform)
	if err != nil {
		t.Fatalf("AcquireProxyForTask returned error: %v", err)
	}
	if binding.ProxyID == nil || *binding.ProxyID != newProxyID {
		t.Fatalf("expected proxy id %d, got %+v", newProxyID, binding.ProxyID)
	}
	if binding.BindStatus != models.TaskProxyBindStatusBound {
		t.Fatalf("expected rebound status, got %s", binding.BindStatus)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func newProxyServiceForTest(db *sql.DB) *ProxyService {
	return NewProxyService(
		repository.NewProxyRepository(db),
		repository.NewProxyPolicyRepository(db),
		repository.NewTaskProxyBindingRepository(db),
		&config.Config{},
	)
}

func taskProxyBindingRows() *sqlmock.Rows {
	return sqlmock.NewRows([]string{
		"id", "task_id", "source_type", "source_policy_id", "proxy_id", "proxy_lease_id",
		"proxy_url_snapshot", "protocol", "region", "platform", "expire_at", "bind_status",
		"is_degraded", "degrade_reason", "last_report_stage", "last_report_success",
		"last_report_at", "created_at", "updated_at",
	})
}

func proxyPolicyColumns() []string {
	return []string{
		"id", "scope_type", "scope_value", "primary_source", "fallback_source", "fallback_enabled",
		"dynamic_timeout_ms", "dynamic_retry_count", "dynamic_circuit_breaker_sec",
		"min_lease_ttl_sec", "manual_selection_strategy", "status", "created_at", "updated_at",
	}
}

func proxyPolicyRows() *sqlmock.Rows {
	return sqlmock.NewRows(proxyPolicyColumns())
}

func proxyRows() *sqlmock.Rows {
	return sqlmock.NewRows([]string{
		"id", "host", "ip", "port", "username", "password", "protocol", "region", "priority",
		"platform_tags", "remark", "status", "last_check_at", "last_check_result",
		"success_count", "fail_count", "last_used_at", "deleted_at", "created_at", "updated_at",
	})
}

func atomicAcquireNoFilterQuery() string {
	return `(?s)WITH candidate AS \(\s*SELECT id\s+FROM proxies\s+WHERE status = \$1 AND deleted_at IS NULL\s+ORDER BY last_used_at ASC NULLS FIRST,\s+priority DESC,\s+success_count DESC,\s+id ASC\s+LIMIT 1\s+FOR UPDATE SKIP LOCKED\s+\)\s+UPDATE proxies p\s+SET last_used_at = \$2,\s+updated_at = \$2\s+FROM candidate\s+WHERE p\.id = candidate\.id\s+RETURNING p\.id, p\.host`
}

func atomicAcquireExcludingQuery() string {
	return `(?s)WITH candidate AS \(\s*SELECT id\s+FROM proxies\s+WHERE status = \$1 AND deleted_at IS NULL AND id <> \$2\s+ORDER BY last_used_at ASC NULLS FIRST,\s+priority DESC,\s+success_count DESC,\s+id ASC\s+LIMIT 1\s+FOR UPDATE SKIP LOCKED\s+\)\s+UPDATE proxies p\s+SET last_used_at = \$3,\s+updated_at = \$3\s+FROM candidate\s+WHERE p\.id = candidate\.id\s+RETURNING p\.id, p\.host`
}
