package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Config 应用配置
type Config struct {
	Server    ServerConfig              `yaml:"server"`
	Redis     RedisConfig               `yaml:"redis"`
	YTDLP     YTDLPConfig               `yaml:"ytdlp"`
	Cache     CacheConfig               `yaml:"cache"`
	Platforms map[string]PlatformConfig `yaml:"platforms"`
}

// ServerConfig 服务器配置
type ServerConfig struct {
	Port int `yaml:"port"`
}

// RedisConfig Redis配置
type RedisConfig struct {
	Addr     string `yaml:"addr"`
	Password string `yaml:"password"`
	DB       int    `yaml:"db"`
	PoolSize int    `yaml:"pool_size"`
}

// YTDLPConfig yt-dlp配置
type YTDLPConfig struct {
	BinaryPath    string   `yaml:"binary_path"`
	Timeout       int      `yaml:"timeout"`        // 解析超时(秒)
	MaxConcurrent int      `yaml:"max_concurrent"` // 最大并发解析数
	CookiesDir    string   `yaml:"cookies_dir"`
	Proxy         string   `yaml:"proxy"`        // 代理地址
	DefaultArgs   []string `yaml:"default_args"` // 默认参数
}

// CacheConfig 缓存配置
type CacheConfig struct {
	TTL     int `yaml:"ttl"`      // 缓存TTL(秒)
	MaxSize int `yaml:"max_size"` // 最大缓存条目数
}

// PlatformConfig 平台特定配置
type PlatformConfig struct {
	Enabled    bool     `yaml:"enabled"`
	ExtraArgs  []string `yaml:"extra_args"`
	CookieFile string   `yaml:"cookie_file"`
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
	// Redis 配置
	if redisAddr := os.Getenv("REDIS_ADDR"); redisAddr != "" {
		cfg.Redis.Addr = redisAddr
	}
	if redisPassword := os.Getenv("REDIS_PASSWORD"); redisPassword != "" {
		cfg.Redis.Password = redisPassword
	}

	// 设置默认值
	if cfg.YTDLP.BinaryPath == "" {
		cfg.YTDLP.BinaryPath = "yt-dlp"
	}
	if cfg.YTDLP.Timeout == 0 {
		cfg.YTDLP.Timeout = 30
	}
	if cfg.YTDLP.MaxConcurrent == 0 {
		cfg.YTDLP.MaxConcurrent = 10
	}
	if cfg.Cache.TTL == 0 {
		cfg.Cache.TTL = 3600
	}

	return &cfg, nil
}

// GetCacheTTL 获取缓存TTL时间
func (c *CacheConfig) GetCacheTTL() time.Duration {
	return time.Duration(c.TTL) * time.Second
}

// GetTimeout 获取超时时间
func (c *YTDLPConfig) GetTimeout() time.Duration {
	return time.Duration(c.Timeout) * time.Second
}
