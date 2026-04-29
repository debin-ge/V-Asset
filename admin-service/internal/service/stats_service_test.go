package service

import (
	"testing"

	"youdlp/admin-service/internal/models"
)

func TestBuildDashboardExceptionsCoversHealthRules(t *testing.T) {
	t.Parallel()

	exceptions := buildDashboardExceptions(
		models.DashboardDownloads{Total: 100, SuccessRate: 0.89},
		models.DashboardProxies{Total: 2, Available: 0, HighRisk: 1},
		models.DashboardProxySource{Healthy: false},
		models.DashboardCookies{Total: 3, Active: 0},
		models.DashboardBilling{ShortfallCount: 2},
	)

	want := map[string]string{
		"Downloads":    "critical",
		"Proxy Source": "critical",
		"Manual Pool":  "warning",
		"Proxy Risk":   "warning",
		"Billing":      "warning",
		"Cookies":      "warning",
	}
	if len(exceptions) != len(want) {
		t.Fatalf("expected %d exceptions, got %d: %#v", len(want), len(exceptions), exceptions)
	}
	for _, item := range exceptions {
		severity, ok := want[item.Area]
		if !ok {
			t.Fatalf("unexpected exception area %q in %#v", item.Area, exceptions)
		}
		if item.Severity != severity {
			t.Fatalf("expected %s severity %q, got %q", item.Area, severity, item.Severity)
		}
		if item.Area == "Manual Pool" && item.ActionHref != "/proxies" {
			t.Fatalf("expected manual pool action href /proxies, got %q", item.ActionHref)
		}
	}
}

func TestBuildDashboardExceptionsSkipsHealthyState(t *testing.T) {
	t.Parallel()

	exceptions := buildDashboardExceptions(
		models.DashboardDownloads{Total: 100, SuccessRate: 0.98},
		models.DashboardProxies{Total: 3, Available: 2},
		models.DashboardProxySource{Healthy: true},
		models.DashboardCookies{Total: 2, Active: 1},
		models.DashboardBilling{},
	)
	if len(exceptions) != 0 {
		t.Fatalf("expected no exceptions, got %#v", exceptions)
	}
}
