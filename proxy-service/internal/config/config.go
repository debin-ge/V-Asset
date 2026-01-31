package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Config 应用配置
type Config struct {
	Server      ServerConfig      `yaml:"server"`
	Redis       RedisConfig       `yaml:"redis"`
	Cache       CacheConfig       `yaml:"cache"`
	YtDLPAPI    YtDLPAPIConfig    `yaml:"ytdlp_api"`
	HealthCheck HealthCheckConfig `yaml:"health_check"`
}

// ServerConfig 服务器配置
type ServerConfig struct {
	Port int    `yaml:"port"`
	Mode string `yaml:"mode"` // debug, release
}

// RedisConfig Redis 配置
type RedisConfig struct {
	Addr     string `yaml:"addr"`
	Password string `yaml:"password"`
	DB       int    `yaml:"db"`
	PoolSize int    `yaml:"pool_size"`
}

// CacheConfig 缓存配置
type CacheConfig struct {
	TTL int `yaml:"ttl"` // 缓存过期时间(秒)
}

// GetCacheTTL 获取缓存 TTL
func (c *CacheConfig) GetCacheTTL() time.Duration {
	if c.TTL <= 0 {
		return 30 * time.Minute
	}
	return time.Duration(c.TTL) * time.Second
}

// YtDLPAPIConfig 第三方 yt-dlp API 配置
type YtDLPAPIConfig struct {
	BaseURL string `yaml:"base_url"`
	APIKey  string `yaml:"api_key"`
	Timeout int    `yaml:"timeout"`
}

// HealthCheckConfig 健康检查配置
type HealthCheckConfig struct {
	Interval int  `yaml:"interval"` // 检查间隔(秒)，默认 60 秒
	Enabled  bool `yaml:"enabled"`  // 是否启用
}

// GetInterval 获取检查间隔
func (c *HealthCheckConfig) GetInterval() time.Duration {
	if c.Interval <= 0 {
		return 60 * time.Second
	}
	return time.Duration(c.Interval) * time.Second
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

	// 环境变量覆盖
	if redisAddr := os.Getenv("REDIS_ADDR"); redisAddr != "" {
		cfg.Redis.Addr = redisAddr
	}
	if ytdlpBaseURL := os.Getenv("YTDLP_API_BASE_URL"); ytdlpBaseURL != "" {
		cfg.YtDLPAPI.BaseURL = ytdlpBaseURL
	}
	if ytdlpAPIKey := os.Getenv("YTDLP_API_KEY"); ytdlpAPIKey != "" {
		cfg.YtDLPAPI.APIKey = ytdlpAPIKey
	}

	return &cfg, nil
}
