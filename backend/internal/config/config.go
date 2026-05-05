package config

import (
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
	viper.AutomaticEnv()

	AppConfig = &Config{}
	if err := viper.Unmarshal(AppConfig); err != nil {
		return err
	}

	return nil
}

func Get() *Config {
	return AppConfig
}
