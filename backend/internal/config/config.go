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
}

type ServerConfig struct {
	Port    string `mapstructure:"port"`
	GinMode string `mapstructure:"gin_mode"`
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

	// Read from environment
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		return err
	}

	AppConfig = &Config{}
	if err := viper.Unmarshal(AppConfig); err != nil {
		return err
	}

	return nil
}

func Get() *Config {
	return AppConfig
}
