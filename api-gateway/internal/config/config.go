package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Config 应用配置
type Config struct {
	Server       ServerConfig       `yaml:"server"`
	GRPC         GRPCConfig         `yaml:"grpc"`
	Redis        RedisConfig        `yaml:"redis"`
	RabbitMQ     RabbitMQConfig     `yaml:"rabbitmq"`
	CORS         CORSConfig         `yaml:"cors"`
	RateLimit    RateLimitConfig    `yaml:"rate_limit"`
	FileDownload FileDownloadConfig `yaml:"file_download"`
	Logging      LoggingConfig      `yaml:"logging"`
}

// ServerConfig 服务器配置
type ServerConfig struct {
	Port           int           `yaml:"port"`
	Mode           string        `yaml:"mode"` // debug, release
	ReadTimeout    time.Duration `yaml:"read_timeout"`
	WriteTimeout   time.Duration `yaml:"write_timeout"`
	MaxHeaderBytes int           `yaml:"max_header_bytes"`
}

// GRPCConfig gRPC 服务配置
type GRPCConfig struct {
	AuthService   string        `yaml:"auth_service"`
	ParserService string        `yaml:"parser_service"`
	AssetService  string        `yaml:"asset_service"`
	Timeout       time.Duration `yaml:"timeout"`
}

// RedisConfig Redis 配置
type RedisConfig struct {
	Addr     string `yaml:"addr"`
	Password string `yaml:"password"`
	DB       int    `yaml:"db"`
	PoolSize int    `yaml:"pool_size"`
}

// RabbitMQConfig RabbitMQ 配置
type RabbitMQConfig struct {
	URL        string `yaml:"url"`
	Exchange   string `yaml:"exchange"`
	Queue      string `yaml:"queue"`
	RoutingKey string `yaml:"routing_key"`
}

// CORSConfig CORS 配置
type CORSConfig struct {
	AllowedOrigins []string `yaml:"allowed_origins"`
	AllowedMethods []string `yaml:"allowed_methods"`
	AllowedHeaders []string `yaml:"allowed_headers"`
	MaxAge         int      `yaml:"max_age"`
}

// RateLimitConfig 限流配置
type RateLimitConfig struct {
	GlobalRPS int `yaml:"global_rps"`
	UserRPS   int `yaml:"user_rps"`
	Burst     int `yaml:"burst"`
}

// FileDownloadConfig 文件下载配置
type FileDownloadConfig struct {
	MaxConcurrent int `yaml:"max_concurrent"`
	BufferSize    int `yaml:"buffer_size"`
}

// LoggingConfig 日志配置
type LoggingConfig struct {
	Level  string `yaml:"level"`
	Format string `yaml:"format"`
}

// LoadConfig 加载配置文件
func LoadConfig(configPath string) (*Config, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// 从环境变量覆盖配置
	// gRPC 服务地址
	if authAddr := os.Getenv("AUTH_SERVICE_ADDR"); authAddr != "" {
		cfg.GRPC.AuthService = authAddr
	}
	if parserAddr := os.Getenv("PARSER_SERVICE_ADDR"); parserAddr != "" {
		cfg.GRPC.ParserService = parserAddr
	}
	if assetAddr := os.Getenv("ASSET_SERVICE_ADDR"); assetAddr != "" {
		cfg.GRPC.AssetService = assetAddr
	}

	// Redis
	if redisAddr := os.Getenv("REDIS_ADDR"); redisAddr != "" {
		cfg.Redis.Addr = redisAddr
	}
	if redisPassword := os.Getenv("REDIS_PASSWORD"); redisPassword != "" {
		cfg.Redis.Password = redisPassword
	}

	// RabbitMQ
	if rabbitmqURL := os.Getenv("RABBITMQ_URL"); rabbitmqURL != "" {
		cfg.RabbitMQ.URL = rabbitmqURL
	}

	// 设置默认值
	if cfg.Server.Port == 0 {
		cfg.Server.Port = 8080
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
	if cfg.RateLimit.GlobalRPS == 0 {
		cfg.RateLimit.GlobalRPS = 200
	}
	if cfg.RateLimit.UserRPS == 0 {
		cfg.RateLimit.UserRPS = 10
	}
	if cfg.RateLimit.Burst == 0 {
		cfg.RateLimit.Burst = 20
	}
	if cfg.FileDownload.BufferSize == 0 {
		cfg.FileDownload.BufferSize = 32768 // 32KB
	}

	return &cfg, nil
}
