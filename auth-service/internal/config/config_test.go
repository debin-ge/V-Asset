package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfigRejectsWeakDefaultJWTSecret(t *testing.T) {
	t.Setenv("APP_ENV", "")
	t.Setenv("ENV", "")
	t.Setenv("GO_ENV", "")
	t.Setenv("JWT_SECRET", "")

	configPath := writeTempConfig(t, "your-secret-key-change-in-production")
	if _, err := LoadConfig(configPath); err == nil {
		t.Fatal("expected weak default jwt secret to be rejected")
	}
}

func TestLoadConfigRejectsShortJWTSecretInProduction(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Setenv("ENV", "")
	t.Setenv("GO_ENV", "")
	t.Setenv("JWT_SECRET", "short-secret")

	configPath := writeTempConfig(t, "ignored-by-env")
	if _, err := LoadConfig(configPath); err == nil {
		t.Fatal("expected short production jwt secret to be rejected")
	}
}

func TestLoadConfigAcceptsStrongJWTSecretInProduction(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Setenv("ENV", "")
	t.Setenv("GO_ENV", "")
	t.Setenv("JWT_SECRET", "0123456789abcdef0123456789abcdef")

	configPath := writeTempConfig(t, "ignored-by-env")
	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("expected strong production jwt secret to be accepted: %v", err)
	}
	if cfg.JWT.Secret != "0123456789abcdef0123456789abcdef" {
		t.Fatalf("unexpected jwt secret loaded: %q", cfg.JWT.Secret)
	}
}

func writeTempConfig(t *testing.T, secret string) string {
	t.Helper()

	content := "server:\n  port: 9001\ndatabase:\n  host: localhost\n  port: 5432\n  user: youdlp\n  password: password\n  dbname: youdlp\n  sslmode: disable\n  max_open_conns: 10\n  max_idle_conns: 5\n  conn_max_lifetime: 3600s\nredis:\n  addr: localhost:6379\n  password: \"\"\n  db: 0\n  pool_size: 10\njwt:\n  secret: \"" + secret + "\"\n  access_token_ttl: 86400\n  refresh_token_ttl: 604800\npassword:\n  bcrypt_cost: 10\n  min_length: 8\n  require_uppercase: true\n  require_lowercase: true\n  require_number: true\n  require_special: false\nsession:\n  max_sessions_per_user: 5\n  cleanup_interval: 3600\n"

	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("failed to write temp config: %v", err)
	}

	return path
}
