package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	Redis    RedisConfig    `mapstructure:"redis"`
	JWT      JWTConfig      `mapstructure:"jwt"`
	CORS     CORSConfig     `mapstructure:"cors"`
	Frontend FrontendConfig `mapstructure:"frontend"`
	Security SecurityConfig `mapstructure:"security"`
}

type ServerConfig struct {
	Port    string `mapstructure:"port"`
	GinMode string `mapstructure:"gin_mode"`
}

type FrontendConfig struct {
	URL string `mapstructure:"url"`
}

type DatabaseConfig struct {
	URL string `mapstructure:"url"`
}

type RedisConfig struct {
	URL string `mapstructure:"url"`
}

type JWTConfig struct {
	Secret            string        `mapstructure:"secret"`
	Expiration        time.Duration `mapstructure:"expiration"`
	RefreshExpiration time.Duration `mapstructure:"refresh_expiration"`
}

type CORSConfig struct {
	Origins []string `mapstructure:"origins"`
}

// SecurityConfig 集中管理所有安全相关参数
type SecurityConfig struct {
	CookieSecure         bool `mapstructure:"cookie_secure"`
	MaxFailedAttempts    int  `mapstructure:"max_failed_attempts"`
	LockoutDurationMin   int  `mapstructure:"lockout_duration_min"`
	MaxTotalAttempts     int  `mapstructure:"max_total_attempts"`
	MaxSessionsPerUser   int  `mapstructure:"max_sessions_per_user"`
	AccessTokenMaxTTLMin int  `mapstructure:"access_token_max_ttl_min"`
	CSRFTokenTTLMin      int  `mapstructure:"csrf_token_ttl_min"`
}

var AppConfig *Config

// flatToNested 定义 .env 文件中的 flat key 到 viper 嵌套 key 的映射
var flatToNested = map[string]string{
	"DATABASE_URL":              "database.url",
	"REDIS_URL":                 "redis.url",
	"JWT_SECRET":                "jwt.secret",
	"JWT_EXPIRATION":            "jwt.expiration",
	"REFRESH_TOKEN_EXPIRATION":  "jwt.refresh_expiration",
	"SERVER_PORT":               "server.port",
	"SERVER_GIN_MODE":           "server.gin_mode",
	"FRONTEND_URL":              "frontend.url",
	"CORS_ORIGINS":              "cors.origins",
	"COOKIE_SECURE":             "security.cookie_secure",
	"MAX_FAILED_ATTEMPTS":       "security.max_failed_attempts",
	"LOCKOUT_DURATION_MIN":      "security.lockout_duration_min",
	"MAX_TOTAL_ATTEMPTS":        "security.max_total_attempts",
	"MAX_SESSIONS_PER_USER":     "security.max_sessions_per_user",
	"ACCESS_TOKEN_MAX_TTL_MIN":  "security.access_token_max_ttl_min",
	"CSRF_TOKEN_TTL_MIN":        "security.csrf_token_ttl_min",
}

func Load(configPath string) error {
	viper.SetConfigFile(configPath)
	viper.SetConfigType("env")

	// Set defaults
	viper.SetDefault("server.port", "8080")
	viper.SetDefault("server.gin_mode", "debug")
	viper.SetDefault("jwt.expiration", "1h")
	viper.SetDefault("jwt.refresh_expiration", "720h")
	viper.SetDefault("cors.origins", []string{"http://localhost:3000"})
	viper.SetDefault("redis.url", "redis://localhost:6379/0")
	viper.SetDefault("frontend.url", "http://localhost:3000")

	// Security defaults
	viper.SetDefault("security.cookie_secure", false)
	viper.SetDefault("security.max_failed_attempts", 5)
	viper.SetDefault("security.lockout_duration_min", 30)
	viper.SetDefault("security.max_total_attempts", 15)
	viper.SetDefault("security.max_sessions_per_user", 5)
	viper.SetDefault("security.access_token_max_ttl_min", 60)
	viper.SetDefault("security.csrf_token_ttl_min", 30)

	// Read from config file (.env)
	if err := viper.ReadInConfig(); err != nil {
		return err
	}

	// 将 .env 文件中的 flat key（如 DATABASE_URL）映射到嵌套 key（如 database.url）
	// Viper 读取 .env 时不会自动做下划线到点号的转换
	for flatKey, nestedKey := range flatToNested {
		if val := viper.GetString(flatKey); val != "" {
			viper.Set(nestedKey, val)
		}
	}

	// 真正的 OS 环境变量也可以覆盖（通过 BindEnv）
	viper.BindEnv("database.url", "DATABASE_URL")
	viper.BindEnv("redis.url", "REDIS_URL")
	viper.BindEnv("jwt.secret", "JWT_SECRET")
	viper.BindEnv("server.port", "SERVER_PORT")
	viper.BindEnv("frontend.url", "FRONTEND_URL")
	viper.BindEnv("security.cookie_secure", "COOKIE_SECURE")
	viper.BindEnv("security.max_failed_attempts", "MAX_FAILED_ATTEMPTS")
	viper.BindEnv("security.lockout_duration_min", "LOCKOUT_DURATION_MIN")
	viper.BindEnv("security.max_total_attempts", "MAX_TOTAL_ATTEMPTS")
	viper.BindEnv("security.max_sessions_per_user", "MAX_SESSIONS_PER_USER")
	viper.BindEnv("security.access_token_max_ttl_min", "ACCESS_TOKEN_MAX_TTL_MIN")
	viper.BindEnv("security.csrf_token_ttl_min", "CSRF_TOKEN_TTL_MIN")
	viper.AutomaticEnv()

	AppConfig = &Config{}
	if err := viper.Unmarshal(AppConfig); err != nil {
		return err
	}

	if len(AppConfig.JWT.Secret) < 32 {
		return fmt.Errorf("JWT_SECRET must be at least 32 bytes, got %d — generate with: openssl rand -hex 32", len(AppConfig.JWT.Secret))
	}

	// release 模式强制启用 Cookie Secure
	if AppConfig.Server.GinMode == "release" && !AppConfig.Security.CookieSecure {
		return fmt.Errorf("COOKIE_SECURE must be true in release mode — HTTP cookies without Secure flag expose tokens to network interception")
	}

	// 校验安全参数范围
	sec := AppConfig.Security
	if sec.MaxFailedAttempts < 3 || sec.MaxFailedAttempts > 20 {
		return fmt.Errorf("MAX_FAILED_ATTEMPTS must be between 3 and 20, got %d", sec.MaxFailedAttempts)
	}
	if sec.LockoutDurationMin < 5 || sec.LockoutDurationMin > 1440 {
		return fmt.Errorf("LOCKOUT_DURATION_MIN must be between 5 and 1440, got %d", sec.LockoutDurationMin)
	}
	if sec.MaxSessionsPerUser < 1 || sec.MaxSessionsPerUser > 20 {
		return fmt.Errorf("MAX_SESSIONS_PER_USER must be between 1 and 20, got %d", sec.MaxSessionsPerUser)
	}

	return nil
}

func Get() *Config {
	return AppConfig
}

// GetIntEnv 从环境变量读取整数，不存在或失败则返回默认值
func (c *Config) GetIntEnv(key string, defaultVal int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return defaultVal
}
