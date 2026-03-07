package config

import (
	"fmt"
	"os"
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
	Secret     string        `yaml:"secret"`
	TTL        time.Duration `yaml:"ttl"`
	CookieName string        `yaml:"cookie_name"`
	Secure     bool          `yaml:"secure"`
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

	if cfg.Server.Port == 0 {
		cfg.Server.Port = 8081
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

	return &cfg, nil
}
