package config

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestLoadConfigOverridesCORSFromEnv(t *testing.T) {
	t.Setenv("CORS_ALLOWED_ORIGINS", "https://ytdlp.obstream.com, https://admin.obstream.com")
	t.Setenv("CORS_ALLOWED_METHODS", "GET, POST, OPTIONS")
	t.Setenv("CORS_ALLOWED_HEADERS", "Content-Type, Authorization, X-Request-ID")
	t.Setenv("WS_ALLOWED_ORIGINS", "https://ytdlp.obstream.com,https://www.ytdlp.obstream.com")

	cfg, err := LoadConfig(filepath.Join("..", "..", "config", "dev.yaml"))
	if err != nil {
		t.Fatalf("LoadConfig returned error: %v", err)
	}

	wantOrigins := []string{"https://ytdlp.obstream.com", "https://admin.obstream.com"}
	if !reflect.DeepEqual(cfg.CORS.AllowedOrigins, wantOrigins) {
		t.Fatalf("AllowedOrigins = %#v, want %#v", cfg.CORS.AllowedOrigins, wantOrigins)
	}

	wantMethods := []string{"GET", "POST", "OPTIONS"}
	if !reflect.DeepEqual(cfg.CORS.AllowedMethods, wantMethods) {
		t.Fatalf("AllowedMethods = %#v, want %#v", cfg.CORS.AllowedMethods, wantMethods)
	}

	wantHeaders := []string{"Content-Type", "Authorization", "X-Request-ID"}
	if !reflect.DeepEqual(cfg.CORS.AllowedHeaders, wantHeaders) {
		t.Fatalf("AllowedHeaders = %#v, want %#v", cfg.CORS.AllowedHeaders, wantHeaders)
	}

	wantWSOrigins := []string{"https://ytdlp.obstream.com", "https://www.ytdlp.obstream.com"}
	if !reflect.DeepEqual(cfg.WebSocket.AllowedOrigins, wantWSOrigins) {
		t.Fatalf("WebSocket.AllowedOrigins = %#v, want %#v", cfg.WebSocket.AllowedOrigins, wantWSOrigins)
	}
}

func TestSplitAndTrimSkipsEmptyValues(t *testing.T) {
	got := splitAndTrim(" GET, ,POST ,  , OPTIONS ")
	want := []string{"GET", "POST", "OPTIONS"}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("splitAndTrim() = %#v, want %#v", got, want)
	}
}

func TestLoadConfigUsesYamlWhenCORSEnvUnset(t *testing.T) {
	unsetEnv(t, "CORS_ALLOWED_ORIGINS")
	unsetEnv(t, "CORS_ALLOWED_METHODS")
	unsetEnv(t, "CORS_ALLOWED_HEADERS")
	unsetEnv(t, "WS_ALLOWED_ORIGINS")

	cfg, err := LoadConfig(filepath.Join("..", "..", "config", "dev.yaml"))
	if err != nil {
		t.Fatalf("LoadConfig returned error: %v", err)
	}

	if len(cfg.CORS.AllowedOrigins) != 0 {
		t.Fatalf("AllowedOrigins = %#v, want empty slice from yaml", cfg.CORS.AllowedOrigins)
	}
	if len(cfg.WebSocket.AllowedOrigins) != 0 {
		t.Fatalf("WebSocket.AllowedOrigins = %#v, want empty slice from yaml", cfg.WebSocket.AllowedOrigins)
	}

	if !cfg.Billing.Enabled {
		t.Fatal("Billing.Enabled = false, want true from yaml")
	}
}

func TestLoadConfigOverridesBillingEnabledFromEnv(t *testing.T) {
	t.Setenv("BILLING_ENABLED", "false")

	cfg, err := LoadConfig(filepath.Join("..", "..", "config", "dev.yaml"))
	if err != nil {
		t.Fatalf("LoadConfig returned error: %v", err)
	}

	if cfg.Billing.Enabled {
		t.Fatal("Billing.Enabled = true, want false from env override")
	}
}

func unsetEnv(t *testing.T, key string) {
	t.Helper()
	original, existed := os.LookupEnv(key)
	if err := os.Unsetenv(key); err != nil {
		t.Fatalf("Unsetenv(%q) returned error: %v", key, err)
	}

	t.Cleanup(func() {
		var err error
		if existed {
			err = os.Setenv(key, original)
		} else {
			err = os.Unsetenv(key)
		}
		if err != nil {
			t.Fatalf("cleanup env %q failed: %v", key, err)
		}
	})
}
