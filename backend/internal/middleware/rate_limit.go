package middleware

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// RateLimitConfig 限流配置
type RateLimitConfig struct {
	MaxRequests  int           // 最大请求次数
	Window       time.Duration // 时间窗口
	BlockDuration time.Duration // 封禁时长
}

// RateLimiter 限流器
type RateLimiter struct {
	redis *redis.Client
}

// NewRateLimiter 创建限流器
func NewRateLimiter(redisClient *redis.Client) *RateLimiter {
	return &RateLimiter{
		redis: redisClient,
	}
}

// RateLimit 通用限流中间件
func (rl *RateLimiter) RateLimit(config RateLimitConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 如果 Redis 不可用，跳过限流
		if rl.redis == nil {
			c.Next()
			return
		}

		// 获取客户端标识（IP）
		clientIP := c.ClientIP()

		// 检查是否被封禁
		blockedKey := fmt.Sprintf("rate_limit:blocked:%s", clientIP)
		blocked, _ := rl.redis.Get(context.Background(), blockedKey).Result()
		if blocked != "" {
			c.JSON(429, gin.H{
				"code":    429,
				"message": "请求过于频繁，请稍后再试",
			})
			c.Abort()
			return
		}

		// 检查限流
		key := fmt.Sprintf("rate_limit:%s:%s", c.FullPath(), clientIP)
		count, err := rl.redis.Incr(context.Background(), key).Result()
		if err != nil {
			// Redis 出错时不限流，降级处理
			c.Next()
			return
		}

		// 第一次访问，设置过期时间
		if count == 1 {
			rl.redis.Expire(context.Background(), key, config.Window)
		}

		// 超过限制
		if count > int64(config.MaxRequests) {
			// 封禁该 IP
			if config.BlockDuration > 0 {
				rl.redis.Set(context.Background(), blockedKey, "1", config.BlockDuration)
			}

			c.JSON(429, gin.H{
				"code":    429,
				"message": fmt.Sprintf("请求过于频繁，请在 %d 分钟后重试", int(config.BlockDuration.Minutes())),
			})
			c.Abort()
			return
		}

		// 设置响应头显示剩余次数
		c.Header("X-RateLimit-Limit", strconv.Itoa(config.MaxRequests))
		c.Header("X-RateLimit-Remaining", strconv.Itoa(config.MaxRequests-int(count)))
		c.Header("X-RateLimit-Reset", strconv.FormatInt(time.Now().Add(config.Window).Unix(), 10))

		c.Next()
	}
}

// RegisterRateLimit 注册专用限流
func RegisterRateLimit(redisClient *redis.Client) gin.HandlerFunc {
	rl := NewRateLimiter(redisClient)

	config := RateLimitConfig{
		MaxRequests:   3,                // 每小时最多3次注册
		Window:        1 * time.Hour,    // 1小时窗口
		BlockDuration: 24 * time.Hour,   // 封禁24小时
	}

	return rl.RateLimit(config)
}

// LoginRateLimit 登录专用限流（更严格）
func LoginRateLimit(redisClient *redis.Client) gin.HandlerFunc {
	rl := NewRateLimiter(redisClient)

	config := RateLimitConfig{
		MaxRequests:   10,               // 每15分钟最多10次登录
		Window:        15 * time.Minute, // 15分钟窗口
		BlockDuration: 1 * time.Hour,    // 封禁1小时
	}

	return rl.RateLimit(config)
}

// APIRateLimit 通用 API 限流
func APIRateLimit(redisClient *redis.Client) gin.HandlerFunc {
	rl := NewRateLimiter(redisClient)

	config := RateLimitConfig{
		MaxRequests:   100,               // 每分钟最多100次请求
		Window:        1 * time.Minute,   // 1分钟窗口
		BlockDuration: 10 * time.Minute,  // 封禁10分钟
	}

	return rl.RateLimit(config)
}

// GetRemainingAttempts 获取剩余尝试次数
func (rl *RateLimiter) GetRemainingAttempts(path, clientIP string) (int, error) {
	key := fmt.Sprintf("rate_limit:%s:%s", path, clientIP)
	val, err := rl.redis.Get(context.Background(), key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return 3, nil // 默认限制
		}
		return 0, err
	}

	count, _ := strconv.Atoi(val)
	return 3 - count, nil
}

// ResetRateLimit 重置限流（管理员功能）
func (rl *RateLimiter) ResetRateLimit(path, clientIP string) error {
	key := fmt.Sprintf("rate_limit:%s:%s", path, clientIP)
	return rl.redis.Del(context.Background(), key).Err()
}
