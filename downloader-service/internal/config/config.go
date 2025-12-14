package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"gopkg.in/yaml.v3"
)

// Config 应用配置
type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Database DatabaseConfig `yaml:"database"`
	RabbitMQ RabbitMQConfig `yaml:"rabbitmq"`
	Redis    RedisConfig    `yaml:"redis"`
	Worker   WorkerConfig   `yaml:"worker"`
	YtDLP    YtDLPConfig    `yaml:"ytdlp"`
	Proxy    ProxyConfig    `yaml:"proxy"`
	Storage  StorageConfig  `yaml:"storage"`
	Cleanup  CleanupConfig  `yaml:"cleanup"`
	Retry    RetryConfig    `yaml:"retry"`
}

// ServerConfig 服务器配置
type ServerConfig struct {
	Port int `yaml:"port"`
}

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
	Host            string        `yaml:"host"`
	Port            int           `yaml:"port"`
	User            string        `yaml:"user"`
	Password        string        `yaml:"password"`
	DBName          string        `yaml:"dbname"`
	SSLMode         string        `yaml:"sslmode"`
	MaxOpenConns    int           `yaml:"max_open_conns"`
	MaxIdleConns    int           `yaml:"max_idle_conns"`
	ConnMaxLifetime time.Duration `yaml:"conn_max_lifetime"`
}

// GetDSN 获取数据库连接字符串
func (c *DatabaseConfig) GetDSN() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.DBName, c.SSLMode,
	)
}

// GetURL 获取数据库连接URL (用于golang-migrate)
func (c *DatabaseConfig) GetURL() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s&x-migrations-table=schema_migrations_downloader",
		c.User, c.Password, c.Host, c.Port, c.DBName, c.SSLMode,
	)
}

// RabbitMQConfig RabbitMQ 配置
type RabbitMQConfig struct {
	URL           string `yaml:"url"`
	Queue         string `yaml:"queue"`
	PrefetchCount int    `yaml:"prefetch_count"`
}

// RedisConfig Redis 配置
type RedisConfig struct {
	Addr     string `yaml:"addr"`
	Password string `yaml:"password"`
	DB       int    `yaml:"db"`
	PoolSize int    `yaml:"pool_size"`
}

// WorkerConfig Worker 池配置
type WorkerConfig struct {
	PoolSize      int `yaml:"pool_size"`
	MaxConcurrent int `yaml:"max_concurrent"`
}

// YtDLPConfig yt-dlp 配置
type YtDLPConfig struct {
	BinaryPath          string              `yaml:"binary_path"`
	Timeout             int                 `yaml:"timeout"`
	ConcurrentFragments int                 `yaml:"concurrent_fragments"`
	CookiesDir          string              `yaml:"cookies_dir"`
	DefaultArgs         []string            `yaml:"default_args"`
	PlatformArgs        map[string][]string `yaml:"platform_args"`
}

// ProxyConfig 代理配置
type ProxyConfig struct {
	Provider    string `yaml:"provider"`
	APIEndpoint string `yaml:"api_endpoint"`
	APIKey      string `yaml:"api_key"`
	Timeout     int    `yaml:"timeout"`
	RetryCount  int    `yaml:"retry_count"`
	StaticURL   string `yaml:"static_url"` // 静态代理URL
}

// StorageConfig 存储配置
type StorageConfig struct {
	BasePath string `yaml:"base_path"`
	TmpTTL   int    `yaml:"tmp_ttl"` // 秒
}

// CleanupConfig 清理配置
type CleanupConfig struct {
	Enabled   bool `yaml:"enabled"`
	Interval  int  `yaml:"interval"` // 秒
	BatchSize int  `yaml:"batch_size"`
}

// RetryConfig 重试配置
type RetryConfig struct {
	MaxAttempts     int `yaml:"max_attempts"`
	InitialInterval int `yaml:"initial_interval"` // 秒
	MaxInterval     int `yaml:"max_interval"`     // 秒
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
	// 数据库配置
	if dbHost := os.Getenv("DB_HOST"); dbHost != "" {
		cfg.Database.Host = dbHost
	}
	if dbPort := os.Getenv("DB_PORT"); dbPort != "" {
		if port, err := strconv.Atoi(dbPort); err == nil {
			cfg.Database.Port = port
		}
	}
	if dbUser := os.Getenv("DB_USER"); dbUser != "" {
		cfg.Database.User = dbUser
	}
	if dbPassword := os.Getenv("DB_PASSWORD"); dbPassword != "" {
		cfg.Database.Password = dbPassword
	}
	if dbName := os.Getenv("DB_NAME"); dbName != "" {
		cfg.Database.DBName = dbName
	}

	// Redis 配置
	if redisAddr := os.Getenv("REDIS_ADDR"); redisAddr != "" {
		cfg.Redis.Addr = redisAddr
	}
	if redisPassword := os.Getenv("REDIS_PASSWORD"); redisPassword != "" {
		cfg.Redis.Password = redisPassword
	}

	// RabbitMQ 配置
	if rabbitMQURL := os.Getenv("RABBITMQ_URL"); rabbitMQURL != "" {
		cfg.RabbitMQ.URL = rabbitMQURL
	}

	// 代理配置
	if proxyAPIKey := os.Getenv("PROXY_API_KEY"); proxyAPIKey != "" {
		cfg.Proxy.APIKey = proxyAPIKey
	}

	return &cfg, nil
}
