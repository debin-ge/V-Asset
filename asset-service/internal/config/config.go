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
	Server     ServerConfig     `yaml:"server"`
	Database   DatabaseConfig   `yaml:"database"`
	Quota      QuotaConfig      `yaml:"quota"`
	Pagination PaginationConfig `yaml:"pagination"`
	Storage    StorageConfig    `yaml:"storage"`
	Proxy      ProxyConfig      `yaml:"proxy"`
	Cookie     CookieConfig     `yaml:"cookie"`
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

// QuotaConfig 配额配置
type QuotaConfig struct {
	DefaultDailyLimit int `yaml:"default_daily_limit"`
	VIPDailyLimit     int `yaml:"vip_daily_limit"`
	ResetHour         int `yaml:"reset_hour"`
}

// PaginationConfig 分页配置
type PaginationConfig struct {
	DefaultPageSize int `yaml:"default_page_size"`
	MaxPageSize     int `yaml:"max_page_size"`
}

// StorageConfig 存储配置
type StorageConfig struct {
	BasePath string `yaml:"base_path"`
}

// ProxyConfig 代理配置
type ProxyConfig struct {
	HealthCheckTimeout int    `yaml:"health_check_timeout"` // 健康检查超时（秒）
	TestURL            string `yaml:"test_url"`             // 测试 URL
}

// CookieConfig Cookie 配置
type CookieConfig struct {
	DefaultFreezeSeconds int `yaml:"default_freeze_seconds"` // 默认冷冻时间（秒）
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

	// 设置默认值
	if cfg.Pagination.DefaultPageSize == 0 {
		cfg.Pagination.DefaultPageSize = 20
	}
	if cfg.Pagination.MaxPageSize == 0 {
		cfg.Pagination.MaxPageSize = 100
	}
	if cfg.Quota.DefaultDailyLimit == 0 {
		cfg.Quota.DefaultDailyLimit = 10
	}
	if cfg.Quota.VIPDailyLimit == 0 {
		cfg.Quota.VIPDailyLimit = 100
	}

	// 代理默认值
	if cfg.Proxy.HealthCheckTimeout == 0 {
		cfg.Proxy.HealthCheckTimeout = 10
	}
	if cfg.Proxy.TestURL == "" {
		cfg.Proxy.TestURL = "https://www.google.com"
	}

	// Cookie 默认值
	if cfg.Cookie.DefaultFreezeSeconds == 0 {
		cfg.Cookie.DefaultFreezeSeconds = 0 // 不冷冻
	}

	return &cfg, nil
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
		"postgres://%s:%s@%s:%d/%s?sslmode=%s&x-migrations-table=schema_migrations_asset",
		c.User, c.Password, c.Host, c.Port, c.DBName, c.SSLMode,
	)
}
