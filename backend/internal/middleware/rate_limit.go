package middleware

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// RateLimitConfig 限流配置
type RateLimitConfig struct {
	MaxRequests   int
	Window        time.Duration
	BlockDuration time.Duration
}

// RateLimiter 限流器
type RateLimiter struct {
	redis      *redis.Client
	failClosed bool // true 时 Redis 不可用则拒绝请求
}

func NewRateLimiter(redisClient *redis.Client) *RateLimiter {
	return &RateLimiter{redis: redisClient}
}

// NewFailClosedRateLimiter 创建对安全关键端点的限流器，Redis 不可用时拒绝请求
func NewFailClosedRateLimiter(redisClient *redis.Client) *RateLimiter {
	return &RateLimiter{redis: redisClient, failClosed: true}
}

// slidingWindowScript 滑动窗口限流 Lua 脚本
// 使用 Sorted Set 以时间戳为 score，统计窗口内的请求数
// 自动淘汰过期条目，避免内存无限增长
var slidingWindowScript = redis.NewScript(`
	local key = KEYS[1]
	local now = tonumber(ARGV[1])
	local window_ms = tonumber(ARGV[2])
	local max_requests = tonumber(ARGV[3])
	local member = ARGV[4]

	local window_start = now - window_ms

	-- 移除窗口外的旧记录
	redis.call("ZREMRANGEBYSCORE", key, "-inf", window_start)

	-- 统计当前窗口内的请求数
	local count = redis.call("ZCARD", key)

	if count >= max_requests then
		return {count, 0}
	end

	-- 添加当前请求（score=now, member=now:random 避免重复）
	redis.call("ZADD", key, now, member)

	-- 设置 key 过期时间为窗口大小（自动清理）
	redis.call("PEXPIRE", key, window_ms)

	return {count + 1, 1}
`)

// RateLimit 通用限流中间件（滑动窗口算法）
func (rl *RateLimiter) RateLimit(config RateLimitConfig) gin.HandlerFunc {
	return rl.RateLimitWithKey(config, func(c *gin.Context) string {
		return c.ClientIP()
	})
}

// RateLimitWithKey 支持自定义限流 key 的限流中间件（用于用户维度等场景）
func (rl *RateLimiter) RateLimitWithKey(config RateLimitConfig, keyFn func(c *gin.Context) string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if rl.redis == nil {
			if rl.failClosed {
				// 安全关键端点：Redis 不可用时拒绝请求，防止暴力破解
				zap.L().Warn("Rate limiter unavailable (Redis not connected), rejecting request (fail-closed)",
					zap.String("path", c.FullPath()),
					zap.String("client_ip", c.ClientIP()),
				)
				c.JSON(503, gin.H{
					"success": false,
					"error": gin.H{
						"code":    "RATE_LIMIT_UNAVAILABLE",
						"message": "RATE_LIMIT_UNAVAILABLE",
						"details": gin.H{
							"reason": "security service temporarily unavailable, please try again later",
						},
					},
				})
				c.Abort()
				return
			}
			// 非安全关键端点：放行请求，避免自我 DoS
			zap.L().Warn("Rate limiter unavailable (Redis not connected), skipping rate limit",
				zap.String("path", c.FullPath()),
				zap.String("client_ip", c.ClientIP()),
			)
			c.Next()
			return
		}

		limitKey := keyFn(c)
		if limitKey == "" {
			c.Next()
			return
		}

		ctx := c.Request.Context()

		// 检查是否被封禁
		blockedKey := fmt.Sprintf("rate_limit:blocked:%s:%s", c.FullPath(), limitKey)
		blocked, _ := rl.redis.Get(ctx, blockedKey).Result()
		if blocked != "" {
			c.JSON(429, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "RATE_LIMIT_BLOCKED_429",
					"message": "RATE_LIMIT_BLOCKED",
					"details": gin.H{
						"reason":        "too many requests",
						"block_duration": int(config.BlockDuration.Minutes()),
					},
				},
			})
			c.Abort()
			return
		}

		// 滑动窗口限流
		key := fmt.Sprintf("rate_limit:%s:%s", c.FullPath(), limitKey)
		now := time.Now().UnixMilli()
		member := fmt.Sprintf("%d:%s", now, c.GetString("request_id"))
		windowMS := config.Window.Milliseconds()

		result, err := slidingWindowScript.Run(ctx, rl.redis, []string{key},
			now,
			windowMS,
			config.MaxRequests,
			member,
		).Int64Slice()

		if err != nil {
			zap.L().Error("Rate limit script error, rejecting request for security", zap.Error(err))
			c.JSON(503, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "RATE_LIMIT_ERROR_503",
					"message": "RATE_LIMIT_ERROR",
					"details": gin.H{
						"reason": "rate limiter encountered an error, request rejected for security",
					},
				},
			})
			c.Abort()
			return
		}

		count := result[0]

		if count > int64(config.MaxRequests) {
			if config.BlockDuration > 0 {
				rl.redis.Set(ctx, blockedKey, "1", config.BlockDuration)
			}

			c.JSON(429, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "RATE_LIMIT_EXCEEDED_429",
					"message": "RATE_LIMIT_EXCEEDED",
					"details": gin.H{
						"max_requests":   config.MaxRequests,
						"window":         config.Window.String(),
						"block_duration": int(config.BlockDuration.Minutes()),
					},
				},
			})
			c.Abort()
			return
		}

		c.Header("X-RateLimit-Limit", strconv.Itoa(config.MaxRequests))
		remaining := config.MaxRequests - int(count)
		if remaining < 0 {
			remaining = 0
		}
		c.Header("X-RateLimit-Remaining", strconv.Itoa(remaining))
		c.Header("X-RateLimit-Reset", strconv.FormatInt(time.Now().Add(config.Window).Unix(), 10))

		c.Next()
	}
}

// RegisterRateLimit 注册专用限流（fail-closed：Redis 不可用时拒绝请求）
func RegisterRateLimit(redisClient *redis.Client) gin.HandlerFunc {
	rl := NewFailClosedRateLimiter(redisClient)
	return rl.RateLimit(RateLimitConfig{
		MaxRequests:   10,
		Window:        10 * time.Minute,
		BlockDuration: 1 * time.Hour,
	})
}

// LoginRateLimit 登录专用限流（更严格，fail-closed）
func LoginRateLimit(redisClient *redis.Client) gin.HandlerFunc {
	rl := NewFailClosedRateLimiter(redisClient)
	return rl.RateLimit(RateLimitConfig{
		MaxRequests:   10,
		Window:        10 * time.Minute,
		BlockDuration: 30 * time.Minute,
	})
}

// APIRateLimit 通用 API 限流
func APIRateLimit(redisClient *redis.Client) gin.HandlerFunc {
	rl := NewRateLimiter(redisClient)
	return rl.RateLimit(RateLimitConfig{
		MaxRequests:   100,
		Window:        1 * time.Minute,
		BlockDuration: 10 * time.Minute,
	})
}

// OAuthTokenRateLimit OAuth token 端点专用限流（防止暴力破解 authorization code / client secret，fail-closed）
func OAuthTokenRateLimit(redisClient *redis.Client) gin.HandlerFunc {
	rl := NewFailClosedRateLimiter(redisClient)
	return rl.RateLimit(RateLimitConfig{
		MaxRequests:   20,
		Window:        1 * time.Minute,
		BlockDuration: 1 * time.Hour,
	})
}

// PasswordResetRateLimit 密码重置端点专用限流（防止邮件轰炸和邮箱枚举，fail-closed）
func PasswordResetRateLimit(redisClient *redis.Client, maxRequests int, blockDuration time.Duration) gin.HandlerFunc {
	rl := NewFailClosedRateLimiter(redisClient)
	return rl.RateLimit(RateLimitConfig{
		MaxRequests:   maxRequests,
		Window:        1 * time.Hour,
		BlockDuration: blockDuration,
	})
}

// RefreshRateLimit token 刷新端点专用限流（fail-closed：防止 token 被盗后无限重试）
func RefreshRateLimit(redisClient *redis.Client) gin.HandlerFunc {
	rl := NewFailClosedRateLimiter(redisClient)
	return rl.RateLimit(RateLimitConfig{
		MaxRequests:   30,
		Window:        15 * time.Minute,
		BlockDuration: 1 * time.Hour,
	})
}

// CSRFTokenRateLimit CSRF token 端点专用限流（防止 token 滥用消耗 Redis 内存）
func CSRFTokenRateLimit(redisClient *redis.Client) gin.HandlerFunc {
	rl := NewRateLimiter(redisClient)
	return rl.RateLimit(RateLimitConfig{
		MaxRequests:   30,
		Window:        1 * time.Minute,
		BlockDuration: 10 * time.Minute,
	})
}

// PasswordChangeRateLimit 密码修改端点专用限流（IP + 用户双维度，fail-closed）
// 先检查 IP 维度限流，再检查用户维度限流，任一维度超限即拒绝
func PasswordChangeRateLimit(redisClient *redis.Client) gin.HandlerFunc {
	rl := NewFailClosedRateLimiter(redisClient)
	ipLimit := rl.RateLimit(RateLimitConfig{
		MaxRequests:   5,
		Window:        15 * time.Minute,
		BlockDuration: 1 * time.Hour,
	})
	userLimit := rl.RateLimitWithKey(RateLimitConfig{
		MaxRequests:   3,
		Window:        15 * time.Minute,
		BlockDuration: 1 * time.Hour,
	}, func(c *gin.Context) string {
		userID, exists := c.Get("user_id")
		if !exists {
			return ""
		}
		return fmt.Sprintf("user:%v", userID)
	})

	return func(c *gin.Context) {
		ipLimit(c)
		if c.IsAborted() {
			return
		}
		userLimit(c)
	}
}

// GetRemainingAttempts 获取剩余尝试次数
func (rl *RateLimiter) GetRemainingAttempts(path, clientIP string, maxRequests int, window time.Duration) (int, error) {
	if rl.redis == nil {
		return maxRequests, nil
	}

	ctx := context.Background()
	key := fmt.Sprintf("rate_limit:%s:%s", path, clientIP)

	// 使用 Lua 脚本原子清理+计数，与 RateLimit 逻辑一致
	var getRemainingScript = redis.NewScript(`
		local key = KEYS[1]
		local now = tonumber(ARGV[1])
		local window_ms = tonumber(ARGV[2])

		redis.call("ZREMRANGEBYSCORE", key, "-inf", tostring(now - window_ms))
		return redis.call("ZCARD", key)
	`)

	now := time.Now().UnixMilli()
	windowMS := window.Milliseconds()

	count, err := getRemainingScript.Run(ctx, rl.redis, []string{key}, now, windowMS).Int64()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return maxRequests, nil
		}
		return 0, err
	}

	remaining := maxRequests - int(count)
	if remaining < 0 {
		remaining = 0
	}
	return remaining, nil
}

// ResetRateLimit 重置限流（管理员功能）
func (rl *RateLimiter) ResetRateLimit(path, clientIP string) error {
	if rl.redis == nil {
		return nil
	}

	ctx := context.Background()
	key := fmt.Sprintf("rate_limit:%s:%s", path, clientIP)
	blockedKey := fmt.Sprintf("rate_limit:blocked:%s", clientIP)

	rl.redis.Del(ctx, key)
	return rl.redis.Del(ctx, blockedKey).Err()
}
