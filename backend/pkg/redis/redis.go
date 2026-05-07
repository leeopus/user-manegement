package redis

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

var (
	Client *redis.Client
)

// InitRedis 初始化 Redis 连接
func InitRedis(url string) error {
	opt, err := redis.ParseURL(url)
	if err != nil {
		return fmt.Errorf("failed to parse redis url: %w", err)
	}

	Client = redis.NewClient(opt)

	// 测试连接
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := Client.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("failed to connect to redis: %w", err)
	}

	log.Println("Redis connected successfully")
	return nil
}

// IsAvailable 检查 Redis 客户端是否已初始化且可用
func IsAvailable() bool {
	return Client != nil
}

// Close 关闭 Redis 连接
func Close() error {
	if Client != nil {
		return Client.Close()
	}
	return nil
}
