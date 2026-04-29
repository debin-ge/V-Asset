package service

import (
	"context"
	"database/sql"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"

	"youdlp/asset-service/internal/config"
	"youdlp/asset-service/internal/models"
	"youdlp/asset-service/internal/repository"
)

func TestSanitizeProxyUsageErrorMessageRedactsCommonSecrets(t *testing.T) {
	t.Parallel()

	message := strings.Join([]string{
		`GET http://proxy-user:proxy-pass@127.0.0.1:8080`,
		`Authorization: Bearer bearer-token`,
		`Proxy-Authorization: Basic proxy-basic-token`,
		`Cookie: sessionid=cookie-secret; csrf=csrf-secret`,
		`X-Api-Key: api-key-secret`,
		`url=https://example.test/watch?token=query-token&password=query-password`,
		`{"token":"json-token","password":"json-password","cookie":"json-cookie"}`,
	}, "\n")

	sanitized := sanitizeProxyUsageErrorMessage(message)
	for _, secret := range []string{
		"proxy-pass",
		"bearer-token",
		"proxy-basic-token",
		"cookie-secret",
		"csrf-secret",
		"api-key-secret",
		"query-token",
		"query-password",
		"json-token",
		"json-password",
		"json-cookie",
	} {
		if strings.Contains(sanitized, secret) {
			t.Fatalf("expected secret %q to be redacted from %q", secret, sanitized)
		}
	}
	if !strings.Contains(sanitized, "proxy-user:***@127.0.0.1:8080") {
		t.Fatalf("expected proxy password redaction, got %q", sanitized)
	}
}

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
			string(models.TaskProxyBindStatusBound), false, nil, nil, nil, nil, nil, 0, nil, nil, 1, now, now,
		))
	mock.ExpectExec(`INSERT INTO proxy_usage_events`).
		WithArgs(taskID, sqlmock.AnyArg(), "static-7", string(models.ProxySourceTypeManualPool), proxyUsageStageParse, "youtube", false, models.ErrorCategoryNetworkTimeout, "timeout", sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`UPDATE task_proxy_bindings\s+SET last_report_stage`).
		WithArgs(taskID, proxyUsageStageParse, false, sqlmock.AnyArg(), models.ErrorCategoryNetworkTimeout).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`UPDATE proxies\s+SET fail_count`).
		WithArgs(proxyID, 10, models.ProxyRiskMaxScore, sqlmock.AnyArg(), models.ErrorCategoryNetworkTimeout, sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(`UPDATE task_proxy_bindings\s+SET bind_status`).
		WithArgs(taskID, models.TaskProxyBindStatusFailed, sqlmock.AnyArg(), models.ErrorCategoryNetworkTimeout, models.TaskProxyBindStatusBound).
		WillReturnRows(taskProxyBindingRows().AddRow(
			int64(1), taskID, string(models.ProxySourceTypeManualPool), int64(10), proxyID, "static-7",
			"http://proxy-a:8080", string(models.ProxyProtocolHTTP), nil, "youtube", nil,
			string(models.TaskProxyBindStatusFailed), false, nil, proxyUsageStageParse, false, now, models.ErrorCategoryNetworkTimeout, 0, now, models.ErrorCategoryNetworkTimeout, 1, now, now,
		))
	mock.ExpectExec(`UPDATE proxies\s+SET active_task_count`).
		WithArgs(proxyID, sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`UPDATE task_proxy_bindings\s+SET bind_status`).
		WithArgs(taskID, models.TaskProxyBindStatusFailed, models.ErrorCategoryNetworkTimeout, sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))

	svc := newProxyServiceForTest(db)
	if err := svc.ReportUsage(context.Background(), taskID, "static-7", proxyUsageStageParse, false, models.ErrorCategoryNetworkTimeout, "timeout"); err != nil {
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
		WithArgs(models.ProxyStatusActive, sqlmock.AnyArg(), models.ProxyRiskExcludeThreshold, sqlmock.AnyArg()).
		WillReturnRows(proxyRows().AddRow(
			int64(8), nil, "127.0.0.8", 8080, nil, nil, string(models.ProxyProtocolHTTP), nil,
			1, nil, nil, models.ProxyStatusActive, nil, nil, 0, 0, now, nil, 0, 0, nil, nil, 1, 0, nil, now, now,
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
		WithArgs(models.ProxyStatusActive, sqlmock.AnyArg(), models.ProxyRiskExcludeThreshold, sqlmock.AnyArg()).
		WillReturnRows(proxyRows().AddRow(
			proxyID, nil, "127.0.0.8", 8080, nil, nil, string(models.ProxyProtocolHTTP), nil,
			1, nil, nil, models.ProxyStatusActive, nil, nil, 0, 0, now, nil, 0, 0, nil, nil, 1, 1, nil, now, now,
		))
	mock.ExpectExec(`INSERT INTO task_proxy_bindings`).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(`SELECT id, task_id, source_type`).
		WithArgs(taskID).
		WillReturnRows(taskProxyBindingRows().AddRow(
			int64(1), taskID, string(models.ProxySourceTypeManualPool), policyID, proxyID, "static-8",
			"http://127.0.0.8:8080", string(models.ProxyProtocolHTTP), nil, nil, nil,
			string(models.TaskProxyBindStatusBound), false, nil, nil, nil, nil, nil, 0, nil, nil, 1, now, now,
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
			string(models.TaskProxyBindStatusFailed), false, nil, proxyUsageStageParse, false, now, models.ErrorCategoryNetworkTimeout, 1, now, models.ErrorCategoryNetworkTimeout, 1, now, now,
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
		WithArgs(models.ProxyStatusActive, sqlmock.AnyArg(), models.ProxyRiskExcludeThreshold, oldProxyID, sqlmock.AnyArg()).
		WillReturnRows(proxyRows().AddRow(
			newProxyID, nil, "127.0.0.8", 8080, nil, nil, string(models.ProxyProtocolHTTP), nil,
			1, nil, nil, models.ProxyStatusActive, nil, nil, 0, 0, nil, nil, 0, 0, nil, nil, 1, 1, nil, now, now,
		))
	mock.ExpectExec(`UPDATE task_proxy_bindings\s+SET source_type`).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(`SELECT id, task_id, source_type`).
		WithArgs(taskID).
		WillReturnRows(taskProxyBindingRows().AddRow(
			int64(1), taskID, string(models.ProxySourceTypeManualPool), policyID, newProxyID, "static-8",
			"http://127.0.0.8:8080", string(models.ProxyProtocolHTTP), nil, "youtube", nil,
			string(models.TaskProxyBindStatusBound), false, nil, nil, nil, nil, nil, 0, nil, nil, 2, now, now,
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

func TestListProxiesCapsPaginationAtServiceBoundary(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create sqlmock failed: %v", err)
	}
	defer db.Close()

	const expectedMaxPage = 10000
	expectedOffset := (expectedMaxPage - 1) * models.ProxyListMaxPageSize

	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM proxies p WHERE deleted_at IS NULL`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(int64(0)))
	mock.ExpectQuery(`(?s)SELECT p\.id,.*FROM proxies p\s+WHERE deleted_at IS NULL\s+ORDER BY p\.status ASC, p\.risk_score ASC, p\.priority DESC, p\.created_at DESC\s+LIMIT \$1 OFFSET \$2`).
		WithArgs(models.ProxyListMaxPageSize, expectedOffset).
		WillReturnRows(proxyRows())

	svc := newProxyServiceForTest(db)
	result, err := svc.ListProxies(context.Background(), models.ProxyListFilter{
		Page:     expectedMaxPage + 1,
		PageSize: models.ProxyListMaxPageSize + 1,
	})
	if err != nil {
		t.Fatalf("ListProxies returned error: %v", err)
	}
	if result.Page != expectedMaxPage || result.PageSize != models.ProxyListMaxPageSize {
		t.Fatalf("expected capped pagination page=%d page_size=%d, got page=%d page_size=%d", expectedMaxPage, models.ProxyListMaxPageSize, result.Page, result.PageSize)
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
		"last_report_at", "last_error_category", "failure_count", "released_at",
		"expired_reason", "binding_generation", "created_at", "updated_at",
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
		"success_count", "fail_count", "last_used_at", "cooldown_until", "consecutive_fail_count",
		"risk_score", "last_error_category", "last_fail_at", "max_concurrent", "active_task_count",
		"deleted_at", "created_at", "updated_at",
	})
}

func atomicAcquireNoFilterQuery() string {
	return `(?s)WITH candidate AS \(\s*SELECT id\s+FROM proxies\s+WHERE status = \$1 AND deleted_at IS NULL AND \(cooldown_until IS NULL OR cooldown_until <= \$2\) AND risk_score < \$3 AND active_task_count < max_concurrent\s+ORDER BY risk_score ASC,\s+last_used_at ASC NULLS FIRST,\s+priority DESC,\s+success_count DESC,\s+consecutive_fail_count ASC,\s+id ASC\s+LIMIT 1\s+FOR UPDATE SKIP LOCKED\s+\)\s+UPDATE proxies p\s+SET last_used_at = \$4,\s+updated_at = \$4(?:,\s+active_task_count = p\.active_task_count \+ 1)?\s+FROM candidate\s+WHERE p\.id = candidate\.id\s+RETURNING p\.id, p\.host`
}

func atomicAcquireExcludingQuery() string {
	return `(?s)WITH candidate AS \(\s*SELECT id\s+FROM proxies\s+WHERE status = \$1 AND deleted_at IS NULL AND \(cooldown_until IS NULL OR cooldown_until <= \$2\) AND risk_score < \$3 AND active_task_count < max_concurrent AND id <> \$4\s+ORDER BY risk_score ASC,\s+last_used_at ASC NULLS FIRST,\s+priority DESC,\s+success_count DESC,\s+consecutive_fail_count ASC,\s+id ASC\s+LIMIT 1\s+FOR UPDATE SKIP LOCKED\s+\)\s+UPDATE proxies p\s+SET last_used_at = \$5,\s+updated_at = \$5,\s+active_task_count = p\.active_task_count \+ 1\s+FROM candidate\s+WHERE p\.id = candidate\.id\s+RETURNING p\.id, p\.host`
}
