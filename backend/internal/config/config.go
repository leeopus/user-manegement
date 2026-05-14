package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/spf13/viper"
	"go.uber.org/zap"
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
	Secret string `mapstructure:"secret"`
}

type CORSConfig struct {
	Origins []string `mapstructure:"origins"`
}

// SecurityConfig 集中管理所有安全相关参数
type SecurityConfig struct {
	CookieSecure                 bool `mapstructure:"cookie_secure"`
	MaxFailedAttempts            int  `mapstructure:"max_failed_attempts"`
	LockoutDurationMin           int  `mapstructure:"lockout_duration_min"`
	MaxTotalAttempts             int  `mapstructure:"max_total_attempts"`
	MaxSessionsPerUser           int  `mapstructure:"max_sessions_per_user"`
	AccessTokenMaxTTLMin         int  `mapstructure:"access_token_max_ttl_min"`
	RefreshTokenTTLDays          int  `mapstructure:"refresh_token_ttl_days"`
	RefreshTokenTTLDaysNoRemember int `mapstructure:"refresh_token_ttl_days_no_remember"`
	CSRFTokenTTLMin              int  `mapstructure:"csrf_token_ttl_min"`
	AuditRetentionDays     int    `mapstructure:"audit_retention_days"`
	PasswordResetSecret    string `mapstructure:"password_reset_secret"`
}

var AppConfig *Config

// flatToNested 定义 .env 文件中的 flat key 到 viper 嵌套 key 的映射
var flatToNested = map[string]string{
	"DATABASE_URL":              "database.url",
	"REDIS_URL":                 "redis.url",
	"JWT_SECRET":                "jwt.secret",
	"SERVER_PORT":               "server.port",
	"SERVER_GIN_MODE":           "server.gin_mode",
	"GIN_MODE":                  "server.gin_mode",
	"FRONTEND_URL":              "frontend.url",
	"CORS_ORIGINS":              "cors.origins",
	"COOKIE_SECURE":             "security.cookie_secure",
	"MAX_FAILED_ATTEMPTS":       "security.max_failed_attempts",
	"LOCKOUT_DURATION_MIN":      "security.lockout_duration_min",
	"MAX_TOTAL_ATTEMPTS":        "security.max_total_attempts",
	"MAX_SESSIONS_PER_USER":     "security.max_sessions_per_user",
	"ACCESS_TOKEN_MAX_TTL_MIN":  "security.access_token_max_ttl_min",
	"REFRESH_TOKEN_TTL_DAYS":             "security.refresh_token_ttl_days",
	"REFRESH_TOKEN_TTL_DAYS_NO_REMEMBER": "security.refresh_token_ttl_days_no_remember",
	"CSRF_TOKEN_TTL_MIN":                 "security.csrf_token_ttl_min",
	"AUDIT_RETENTION_DAYS":      "security.audit_retention_days",
	"PASSWORD_RESET_SECRET":     "security.password_reset_secret",
}

func Load(configPath string) error {
	viper.SetConfigFile(configPath)
	viper.SetConfigType("env")

	// Set defaults
	viper.SetDefault("server.port", "8080")
	viper.SetDefault("server.gin_mode", "debug")
	viper.SetDefault("cors.origins", []string{"http://localhost:3000"})
	viper.SetDefault("redis.url", "redis://localhost:6379/0")
	viper.SetDefault("frontend.url", "http://localhost:3000")

	// Security defaults
	viper.SetDefault("security.cookie_secure", false)
	viper.SetDefault("security.max_failed_attempts", 5)
	viper.SetDefault("security.lockout_duration_min", 30)
	viper.SetDefault("security.max_total_attempts", 15)
	viper.SetDefault("security.max_sessions_per_user", 5)
	viper.SetDefault("security.access_token_max_ttl_min", 15)
	viper.SetDefault("security.refresh_token_ttl_days", 30)
	viper.SetDefault("security.refresh_token_ttl_days_no_remember", 7)
	viper.SetDefault("security.csrf_token_ttl_min", 30)
	viper.SetDefault("security.audit_retention_days", 90)

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
	viper.BindEnv("server.gin_mode", "GIN_MODE")
	viper.BindEnv("frontend.url", "FRONTEND_URL")
	viper.BindEnv("security.cookie_secure", "COOKIE_SECURE")
	viper.BindEnv("security.max_failed_attempts", "MAX_FAILED_ATTEMPTS")
	viper.BindEnv("security.lockout_duration_min", "LOCKOUT_DURATION_MIN")
	viper.BindEnv("security.max_total_attempts", "MAX_TOTAL_ATTEMPTS")
	viper.BindEnv("security.max_sessions_per_user", "MAX_SESSIONS_PER_USER")
	viper.BindEnv("security.access_token_max_ttl_min", "ACCESS_TOKEN_MAX_TTL_MIN")
	viper.BindEnv("security.refresh_token_ttl_days", "REFRESH_TOKEN_TTL_DAYS")
	viper.BindEnv("security.refresh_token_ttl_days_no_remember", "REFRESH_TOKEN_TTL_DAYS_NO_REMEMBER")
	viper.BindEnv("security.csrf_token_ttl_min", "CSRF_TOKEN_TTL_MIN")
	viper.BindEnv("security.audit_retention_days", "AUDIT_RETENTION_DAYS")
	viper.BindEnv("security.password_reset_secret", "PASSWORD_RESET_SECRET")
	viper.AutomaticEnv()

	AppConfig = &Config{}
	if err := viper.Unmarshal(AppConfig); err != nil {
		return err
	}

	if len(AppConfig.JWT.Secret) < 32 {
		return fmt.Errorf("JWT_SECRET must be at least 32 bytes, got %d — generate with: openssl rand -hex 32", len(AppConfig.JWT.Secret))
	}

	if err := validateJWTSecretEntropy(AppConfig.JWT.Secret); err != nil {
		return err
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
	if sec.AccessTokenMaxTTLMin < 1 || sec.AccessTokenMaxTTLMin > 1440 {
		return fmt.Errorf("ACCESS_TOKEN_MAX_TTL_MIN must be between 1 and 1440, got %d", sec.AccessTokenMaxTTLMin)
	}
	if sec.RefreshTokenTTLDays < 1 || sec.RefreshTokenTTLDays > 90 {
		return fmt.Errorf("REFRESH_TOKEN_TTL_DAYS must be between 1 and 90, got %d", sec.RefreshTokenTTLDays)
	}
	if sec.RefreshTokenTTLDaysNoRemember < 1 || sec.RefreshTokenTTLDaysNoRemember > 90 {
		return fmt.Errorf("REFRESH_TOKEN_TTL_DAYS_NO_REMEMBER must be between 1 and 90, got %d", sec.RefreshTokenTTLDaysNoRemember)
	}
	if sec.RefreshTokenTTLDaysNoRemember > sec.RefreshTokenTTLDays {
		return fmt.Errorf("REFRESH_TOKEN_TTL_DAYS_NO_REMEMBER (%d) must not exceed REFRESH_TOKEN_TTL_DAYS (%d)", sec.RefreshTokenTTLDaysNoRemember, sec.RefreshTokenTTLDays)
	}

	// 密码重置密钥：未设置时回退到 JWT Secret 并告警，release 模式强制要求独立设置
	if sec.PasswordResetSecret == "" {
		if AppConfig.Server.GinMode == "release" {
			return fmt.Errorf("PASSWORD_RESET_SECRET is required in release mode — generate with: openssl rand -hex 32")
		}
		zap.L().Warn("PASSWORD_RESET_SECRET not set, falling back to JWT_SECRET (recommended to set a separate secret)")
		AppConfig.Security.PasswordResetSecret = AppConfig.JWT.Secret
	} else if len(sec.PasswordResetSecret) < 32 {
		return fmt.Errorf("PASSWORD_RESET_SECRET must be at least 32 bytes, got %d — generate with: openssl rand -hex 32", len(sec.PasswordResetSecret))
	}

	return nil
}

func Get() *Config {
	return AppConfig
}

// GetRefreshTokenTTL 返回统一的 refresh token 有效期
func (c *Config) GetRefreshTokenTTL() time.Duration {
	days := c.Security.RefreshTokenTTLDays
	if days <= 0 {
		days = 30
	}
	return time.Duration(days) * 24 * time.Hour
}

// GetRefreshTokenTTLForRememberMe 根据 RememberMe 返回 refresh token 有效期
func (c *Config) GetRefreshTokenTTLForRememberMe(rememberMe bool) time.Duration {
	if rememberMe {
		return c.GetRefreshTokenTTL()
	}
	days := c.Security.RefreshTokenTTLDaysNoRemember
	if days <= 0 {
		days = 7
	}
	return time.Duration(days) * 24 * time.Hour
}

// GetIntEnv 从环境变量读取整数，不存在或失败则返回默认值
func (c *Config) GetIntEnv(key string, defaultVal int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	if v := viper.GetString(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return defaultVal
}

// GetBoolEnv 从环境变量或 .env 配置读取布尔值
func (c *Config) GetBoolEnv(key string, defaultVal bool) bool {
	if v := os.Getenv(key); v != "" {
		return strings.EqualFold(v, "true")
	}
	if v := viper.GetString(key); v != "" {
		return strings.EqualFold(v, "true")
	}
	return defaultVal
}

// wellKnownWeakSecrets 常见的弱 JWT Secret 占位符
var wellKnownWeakSecrets = []string{
	"your-super-secret-jwt-key-change-in-production",
	"replace_me_generate_with_openssl_rand_hex_32",
	"secret", "jwt-secret", "my-secret", "change-me",
	"super-secret", "jwt_secret", "your-secret-key",
	"example-secret", "test-secret", "default-secret",
}

// validateJWTSecretEntropy 校验 JWT Secret 的熵和安全性
func validateJWTSecretEntropy(secret string) error {
	lower := strings.ToLower(strings.TrimSpace(secret))

	for _, weak := range wellKnownWeakSecrets {
		if lower == strings.ToLower(weak) {
			return fmt.Errorf("JWT_SECRET is a known weak value (%q) — generate a strong random key with: openssl rand -hex 32", weak)
		}
	}

	// 检查字符多样性：至少需要 3 种字符类型（大写、小写、数字、特殊字符）
	var hasUpper, hasLower, hasDigit, hasSpecial bool
	for _, r := range secret {
		switch {
		case unicode.IsUpper(r):
			hasUpper = true
		case unicode.IsLower(r):
			hasLower = true
		case unicode.IsDigit(r):
			hasDigit = true
		default:
			hasSpecial = true
		}
	}

	charTypes := 0
	for _, has := range []bool{hasUpper, hasLower, hasDigit, hasSpecial} {
		if has {
			charTypes++
		}
	}

	if charTypes < 3 {
		return fmt.Errorf("JWT_SECRET has insufficient character diversity (only %d types: upper=%v lower=%v digit=%v special=%v) — use a high-entropy random key from: openssl rand -hex 32",
			charTypes, hasUpper, hasLower, hasDigit, hasSpecial)
	}

	return nil
}
