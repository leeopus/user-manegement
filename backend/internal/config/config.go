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
	Secret             string        `mapstructure:"secret"`
	Expiration         time.Duration `mapstructure:"expiration"`
	RefreshExpiration  time.Duration `mapstructure:"refresh_expiration"`
}

type CORSConfig struct {
	Origins []string `mapstructure:"origins"`
}

var AppConfig *Config

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
	viper.SetDefault("frontend.url", "http://localhost:3000") // Add default for frontend URL
	viper.SetDefault("database.url", "postgres://admin:admin123@localhost:5432/user_system?sslmode=disable") // Add default for database URL

	// Read from config file
	if err := viper.ReadInConfig(); err != nil {
		return err
	}

	// Bind environment variables for nested structures
	viper.BindEnv("frontend.url", "FRONTEND_URL")
	viper.BindEnv("database.url", "DATABASE_URL") // 绑定数据库URL环境变量
	viper.BindEnv("redis.url", "REDIS_URL")       // 绑定Redis URL环境变量

	// Read from environment (will override config file)
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
