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

func getSessionTTL() time.Duration {
	cfg := config.Get()
	if cfg != nil {
		return cfg.GetRefreshTokenTTL()
	}
	return 30 * 24 * time.Hour
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
		return fmt.Errorf("refresh token store unavailable")
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
		int64(getSessionTTL().Seconds()),
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

// revokeScript 原子撤销单个 refresh token：DEL token + ZREM session
var revokeScript = redis.NewScript(`
	local tokenKey = KEYS[1]
	local sessionsKey = KEYS[2]
	local hash = ARGV[1]

	redis.call("DEL", tokenKey)
	redis.call("ZREM", sessionsKey, hash)
	return 1
`)

// Revoke 撤销单个 refresh token
func (m *RefreshTokenManager) Revoke(userID uint, token string) error {
	if m.redis == nil {
		return fmt.Errorf("refresh token store unavailable")
	}

	ctx, cancel := m.ctx()
	defer cancel()

	hash := refreshTokenHash(token)
	tokenKey := fmt.Sprintf("%s%s", refreshTokenPrefix, hash)
	sessionsKey := fmt.Sprintf("%s%d", userSessionsPrefix, userID)

	_, err := revokeScript.Run(ctx, m.redis,
		[]string{tokenKey, sessionsKey},
		hash,
	).Result()

	return err
}

// revokeAllScript 原子撤销用户所有 refresh token：ZRANGE + DEL all + DEL sorted set
var revokeAllScript = redis.NewScript(`
	local sessionsKey = KEYS[1]
	local tokenPrefix = ARGV[1]

	local members = redis.call("ZRANGE", sessionsKey, 0, -1)
	for _, hash in ipairs(members) do
		redis.call("DEL", tokenPrefix .. hash)
	end
	redis.call("DEL", sessionsKey)
	return 1
`)

// RevokeAll 撤销用户所有 refresh token（登出所有设备）
func (m *RefreshTokenManager) RevokeAll(userID uint) error {
	if m.redis == nil {
		return fmt.Errorf("refresh token store unavailable")
	}

	ctx, cancel := m.ctx()
	defer cancel()

	sessionsKey := fmt.Sprintf("%s%d", userSessionsPrefix, userID)

	_, err := revokeAllScript.Run(ctx, m.redis,
		[]string{sessionsKey},
		refreshTokenPrefix,
	).Result()

	return err
}

// rotateScript 原子化旋转 refresh token：validate-old → revoke-old → store-new → evict-excess
var rotateScript = redis.NewScript(`
	local oldTokenKey = KEYS[1]
	local sessionsKey = KEYS[2]
	local newTokenKey = KEYS[3]

	local expectedUserID = ARGV[1]
	local newHash = ARGV[2]
	local newTTL = tonumber(ARGV[3])
	local oldHash = ARGV[4]
	local sessionTTL = tonumber(ARGV[5])
	local maxSessions = tonumber(ARGV[6])
	local now = tonumber(ARGV[7])
	local tokenPrefix = ARGV[8]

	-- Step 1: Validate old token exists and matches user
	local storedUserID = redis.call("GET", oldTokenKey)
	if not storedUserID then
		return {-1, "token_not_found"}
	end
	if storedUserID ~= expectedUserID then
		return {-2, "user_mismatch"}
	end

	-- Step 2: Revoke old token
	redis.call("DEL", oldTokenKey)
	redis.call("ZREM", sessionsKey, oldHash)

	-- Step 3: Store new token
	redis.call("SET", newTokenKey, expectedUserID, "EX", newTTL)
	redis.call("ZADD", sessionsKey, now, newHash)
	redis.call("EXPIRE", sessionsKey, sessionTTL)

	-- Step 4: Evict excess sessions
	local count = redis.call("ZCARD", sessionsKey)
	if count > maxSessions then
		local excess = count - maxSessions
		local oldest = redis.call("ZRANGE", sessionsKey, 0, excess - 1)
		for _, old in ipairs(oldest) do
			redis.call("DEL", tokenPrefix .. old)
		end
		redis.call("ZREMRANGEBYRANK", sessionsKey, 0, excess - 1)
	end

	return {1, "ok"}
`)

// Rotate atomically validates the old refresh token, revokes it, and stores the new one.
func (m *RefreshTokenManager) Rotate(userID uint, oldToken, newToken string, newTTL time.Duration) error {
	if m.redis == nil {
		return fmt.Errorf("refresh token store unavailable")
	}

	ctx, cancel := m.ctx()
	defer cancel()

	oldHash := refreshTokenHash(oldToken)
	newHash := refreshTokenHash(newToken)
	oldTokenKey := fmt.Sprintf("%s%s", refreshTokenPrefix, oldHash)
	sessionsKey := fmt.Sprintf("%s%d", userSessionsPrefix, userID)
	newTokenKey := fmt.Sprintf("%s%s", refreshTokenPrefix, newHash)

	result, err := rotateScript.Run(ctx, m.redis,
		[]string{oldTokenKey, sessionsKey, newTokenKey},
		fmt.Sprintf("%d", userID),
		newHash,
		int64(newTTL.Seconds()),
		oldHash,
		int64(getSessionTTL().Seconds()),
		getMaxSessionsPerUser(),
		float64(time.Now().UnixNano()),
		refreshTokenPrefix,
	).Int64Slice()

	if err != nil {
		return fmt.Errorf("rotate script error: %w", err)
	}

	if len(result) < 1 || result[0] < 0 {
		if len(result) >= 2 && result[0] == -1 {
			return ErrTokenNotFound
		}
		return fmt.Errorf("rotate failed: user mismatch")
	}

	return nil
}
