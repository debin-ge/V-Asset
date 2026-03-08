package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server  ServerConfig  `yaml:"server"`
	GRPC    GRPCConfig    `yaml:"grpc"`
	Redis   RedisConfig   `yaml:"redis"`
	Session SessionConfig `yaml:"session"`
	CORS    CORSConfig    `yaml:"cors"`
}

type ServerConfig struct {
	Port           int           `yaml:"port"`
	Mode           string        `yaml:"mode"`
	ReadTimeout    time.Duration `yaml:"read_timeout"`
	WriteTimeout   time.Duration `yaml:"write_timeout"`
	MaxHeaderBytes int           `yaml:"max_header_bytes"`
}

type GRPCConfig struct {
	AuthService  string        `yaml:"auth_service"`
	AssetService string        `yaml:"asset_service"`
	Timeout      time.Duration `yaml:"timeout"`
}

type RedisConfig struct {
	Addr     string `yaml:"addr"`
	Password string `yaml:"password"`
	DB       int    `yaml:"db"`
	PoolSize int    `yaml:"pool_size"`
}

type SessionConfig struct {
	Secret       string        `yaml:"secret"`
	TTL          time.Duration `yaml:"ttl"`
	CookieName   string        `yaml:"cookie_name"`
	Secure       bool          `yaml:"secure"`
	CookieDomain string        `yaml:"cookie_domain"`
	SameSite     string        `yaml:"same_site"`
}

type CORSConfig struct {
	AllowedOrigins   []string `yaml:"allowed_origins"`
	AllowedMethods   []string `yaml:"allowed_methods"`
	AllowedHeaders   []string `yaml:"allowed_headers"`
	AllowCredentials bool     `yaml:"allow_credentials"`
}

func LoadConfig(configPath string) (*Config, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	if authAddr := os.Getenv("AUTH_SERVICE_ADDR"); authAddr != "" {
		cfg.GRPC.AuthService = authAddr
	}
	if assetAddr := os.Getenv("ASSET_SERVICE_ADDR"); assetAddr != "" {
		cfg.GRPC.AssetService = assetAddr
	}
	if redisAddr := os.Getenv("REDIS_ADDR"); redisAddr != "" {
		cfg.Redis.Addr = redisAddr
	}
	if redisPassword := os.Getenv("REDIS_PASSWORD"); redisPassword != "" {
		cfg.Redis.Password = redisPassword
	}
	if corsAllowedOrigins := os.Getenv("CORS_ALLOWED_ORIGINS"); corsAllowedOrigins != "" {
		cfg.CORS.AllowedOrigins = splitAndTrim(corsAllowedOrigins)
	}
	if corsAllowedMethods := os.Getenv("CORS_ALLOWED_METHODS"); corsAllowedMethods != "" {
		cfg.CORS.AllowedMethods = splitAndTrim(corsAllowedMethods)
	}
	if corsAllowedHeaders := os.Getenv("CORS_ALLOWED_HEADERS"); corsAllowedHeaders != "" {
		cfg.CORS.AllowedHeaders = splitAndTrim(corsAllowedHeaders)
	}
	if sessionSecure := os.Getenv("SESSION_SECURE"); sessionSecure != "" {
		cfg.Session.Secure = strings.EqualFold(sessionSecure, "true") || sessionSecure == "1"
	}
	if sessionCookieDomain := os.Getenv("SESSION_COOKIE_DOMAIN"); sessionCookieDomain != "" {
		cfg.Session.CookieDomain = sessionCookieDomain
	}
	if sessionSameSite := os.Getenv("SESSION_SAME_SITE"); sessionSameSite != "" {
		cfg.Session.SameSite = sessionSameSite
	}

	if cfg.Server.Port == 0 {
		cfg.Server.Port = 9005
	}
	if cfg.Server.Mode == "" {
		cfg.Server.Mode = "debug"
	}
	if cfg.Server.ReadTimeout == 0 {
		cfg.Server.ReadTimeout = 30 * time.Second
	}
	if cfg.Server.WriteTimeout == 0 {
		cfg.Server.WriteTimeout = 30 * time.Second
	}
	if cfg.GRPC.Timeout == 0 {
		cfg.GRPC.Timeout = 5 * time.Second
	}
	if cfg.Session.TTL == 0 {
		cfg.Session.TTL = 24 * time.Hour
	}
	if cfg.Session.CookieName == "" {
		cfg.Session.CookieName = "vasset_admin_session"
	}
	if cfg.Session.SameSite == "" {
		cfg.Session.SameSite = "Lax"
	}
	if len(cfg.CORS.AllowedMethods) == 0 {
		cfg.CORS.AllowedMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}
	}
	if len(cfg.CORS.AllowedHeaders) == 0 {
		cfg.CORS.AllowedHeaders = []string{"Content-Type", "Authorization"}
	}

	return &cfg, nil
}

func splitAndTrim(value string) []string {
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}
