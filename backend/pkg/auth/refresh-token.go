package auth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/user-system/backend/internal/config"
)

const (
	refreshTokenPrefix = "auth:refresh_token:"
	userSessionsPrefix = "auth:user_sessions:"
)

func getMaxSessionsPerUser() int {
	cfg := config.Get()
	if cfg != nil && cfg.Security.MaxSessionsPerUser > 0 {
		return cfg.Security.MaxSessionsPerUser
	}
	return 5
}

// ErrTokenNotFound 表示 token 在 Redis 中不存在（已过期或已撤销）
var ErrTokenNotFound = fmt.Errorf("refresh token not found or expired")

// RefreshTokenManager 管理服务端 refresh token 生命周期
type RefreshTokenManager struct {
	redis *redis.Client
}

func NewRefreshTokenManager(redisClient *redis.Client) *RefreshTokenManager {
	return &RefreshTokenManager{redis: redisClient}
}

func (m *RefreshTokenManager) ctx() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), redisOpTimeout)
}

func refreshTokenHash(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}

// storeScript 原子化存储 refresh token：SET token + ZADD session + 自动淘汰超限旧 token
var storeScript = redis.NewScript(`
	local tokenKey = KEYS[1]
	local sessionsKey = KEYS[2]
	local userID = ARGV[1]
	local tokenTTL = tonumber(ARGV[2])
	local sessionTTL = tonumber(ARGV[3])
	local maxSessions = tonumber(ARGV[4])
	local hash = ARGV[5]
	local now = tonumber(ARGV[6])

	redis.call("SET", tokenKey, userID, "EX", tokenTTL)

	redis.call("ZADD", sessionsKey, now, hash)
	redis.call("EXPIRE", sessionsKey, sessionTTL)

	local count = redis.call("ZCARD", sessionsKey)
	if count > maxSessions then
		local excess = count - maxSessions
		local oldest = redis.call("ZRANGE", sessionsKey, 0, excess - 1)
		for _, old in ipairs(oldest) do
			redis.call("DEL", KEYS[3] .. old)
		end
		redis.call("ZREMRANGEBYRANK", sessionsKey, 0, excess - 1)
	end

	return 1
`)

// Store 存储 refresh token，使用 Lua 脚本保证原子性
func (m *RefreshTokenManager) Store(userID uint, token string, ttl time.Duration) error {
	if m.redis == nil {
		return nil
	}

	ctx, cancel := m.ctx()
	defer cancel()

	hash := refreshTokenHash(token)
	tokenKey := fmt.Sprintf("%s%s", refreshTokenPrefix, hash)
	sessionsKey := fmt.Sprintf("%s%d", userSessionsPrefix, userID)

	_, err := storeScript.Run(ctx, m.redis,
		[]string{tokenKey, sessionsKey, refreshTokenPrefix},
		fmt.Sprintf("%d", userID),
		int64(ttl.Seconds()),
		int64(30*24*time.Hour.Seconds()),
		getMaxSessionsPerUser(),
		hash,
		float64(time.Now().UnixNano()),
	).Result()

	return err
}

// Validate 校验 refresh token 是否有效，区分 Redis 不可用、token 不存在和 token 过期
func (m *RefreshTokenManager) Validate(token string) (uint, error) {
	if m.redis == nil {
		return 0, fmt.Errorf("refresh token store unavailable")
	}

	ctx, cancel := m.ctx()
	defer cancel()

	hash := refreshTokenHash(token)
	tokenKey := fmt.Sprintf("%s%s", refreshTokenPrefix, hash)

	val, err := m.redis.Get(ctx, tokenKey).Result()
	if err != nil {
		if err == redis.Nil {
			return 0, ErrTokenNotFound
		}
		return 0, fmt.Errorf("refresh token store error: %w", err)
	}

	var userID uint
	fmt.Sscanf(val, "%d", &userID)
	return userID, nil
}

// Revoke 撤销单个 refresh token
func (m *RefreshTokenManager) Revoke(userID uint, token string) error {
	if m.redis == nil {
		return nil
	}

	ctx, cancel := m.ctx()
	defer cancel()

	hash := refreshTokenHash(token)
	tokenKey := fmt.Sprintf("%s%s", refreshTokenPrefix, hash)
	sessionsKey := fmt.Sprintf("%s%d", userSessionsPrefix, userID)

	m.redis.Del(ctx, tokenKey)
	m.redis.ZRem(ctx, sessionsKey, hash)
	return nil
}

// RevokeAll 撤销用户所有 refresh token（登出所有设备）
func (m *RefreshTokenManager) RevokeAll(userID uint) error {
	if m.redis == nil {
		return nil
	}

	ctx, cancel := m.ctx()
	defer cancel()

	sessionsKey := fmt.Sprintf("%s%d", userSessionsPrefix, userID)

	members := m.redis.ZRange(ctx, sessionsKey, 0, -1).Val()
	for _, hash := range members {
		m.redis.Del(ctx, fmt.Sprintf("%s%s", refreshTokenPrefix, hash))
	}

	return m.redis.Del(ctx, sessionsKey).Err()
}
