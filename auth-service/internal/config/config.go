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
	Redis    RedisConfig    `yaml:"redis"`
	JWT      JWTConfig      `yaml:"jwt"`
	Password PasswordConfig `yaml:"password"`
	Session  SessionConfig  `yaml:"session"`
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

// RedisConfig Redis配置
type RedisConfig struct {
	Addr     string `yaml:"addr"`
	Password string `yaml:"password"`
	DB       int    `yaml:"db"`
	PoolSize int    `yaml:"pool_size"`
}

// JWTConfig JWT配置
type JWTConfig struct {
	Secret          string `yaml:"secret"`
	AccessTokenTTL  int64  `yaml:"access_token_ttl"`  // 秒
	RefreshTokenTTL int64  `yaml:"refresh_token_ttl"` // 秒
}

// PasswordConfig 密码策略配置
type PasswordConfig struct {
	BcryptCost       int  `yaml:"bcrypt_cost"`
	MinLength        int  `yaml:"min_length"`
	RequireUppercase bool `yaml:"require_uppercase"`
	RequireLowercase bool `yaml:"require_lowercase"`
	RequireNumber    bool `yaml:"require_number"`
	RequireSpecial   bool `yaml:"require_special"`
}

// SessionConfig 会话配置
type SessionConfig struct {
	MaxSessionsPerUser int `yaml:"max_sessions_per_user"`
	CleanupInterval    int `yaml:"cleanup_interval"` // 秒
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

	// JWT 配置
	if jwtSecret := os.Getenv("JWT_SECRET"); jwtSecret != "" {
		cfg.JWT.Secret = jwtSecret
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
		"postgres://%s:%s@%s:%d/%s?sslmode=%s&x-migrations-table=schema_migrations_auth",
		c.User, c.Password, c.Host, c.Port, c.DBName, c.SSLMode,
	)
}
