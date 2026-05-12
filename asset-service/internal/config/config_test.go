package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfigDefaultsProxyBindingReconcileInterval(t *testing.T) {
	t.Parallel()

	cfg, err := LoadConfig(writeTempConfig(t, "server:\n  port: 9004\n"))
	if err != nil {
		t.Fatalf("LoadConfig returned error: %v", err)
	}
	if cfg.Proxy.BindingReconcileIntervalSeconds != 60 {
		t.Fatalf("expected default binding reconcile interval 60 seconds, got %d", cfg.Proxy.BindingReconcileIntervalSeconds)
	}
}

func TestLoadConfigAllowsDisablingProxyBindingReconcileFromEnv(t *testing.T) {
	t.Setenv("PROXY_BINDING_RECONCILE_INTERVAL_SECONDS", "0")

	cfg, err := LoadConfig(writeTempConfig(t, "server:\n  port: 9004\n"))
	if err != nil {
		t.Fatalf("LoadConfig returned error: %v", err)
	}
	if cfg.Proxy.BindingReconcileIntervalSeconds != 0 {
		t.Fatalf("expected env override to disable reconcile interval, got %d", cfg.Proxy.BindingReconcileIntervalSeconds)
	}
}

func writeTempConfig(t *testing.T, content string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatalf("write temp config failed: %v", err)
	}
	return path
}
