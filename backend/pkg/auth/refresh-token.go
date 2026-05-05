package auth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	refreshTokenPrefix = "auth:refresh_token:"
	userSessionsPrefix = "auth:user_sessions:"
	maxSessionsPerUser = 5
)

// RefreshTokenManager 管理服务端 refresh token 生命周期
type RefreshTokenManager struct {
	redis *redis.Client
}

// NewRefreshTokenManager 创建 refresh token 管理器
func NewRefreshTokenManager(redisClient *redis.Client) *RefreshTokenManager {
	return &RefreshTokenManager{redis: redisClient}
}

// tokenHash 对 token 做哈希，避免明文存储
func tokenHash(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}

// Store 存储 refresh token，关联到用户（支持多设备）
func (m *RefreshTokenManager) Store(userID uint, token string, ttl time.Duration) error {
	if m.redis == nil {
		return nil
	}

	ctx := context.Background()
	hash := tokenHash(token)
	tokenKey := fmt.Sprintf("%s%s", refreshTokenPrefix, hash)
	sessionsKey := fmt.Sprintf("%s%d", userSessionsPrefix, userID)

	// 1. 存储token到用户会话集合
	m.redis.SAdd(ctx, sessionsKey, hash)
	m.redis.Expire(ctx, sessionsKey, 30*24*time.Hour) // 最长30天

	// 限制每用户最多 maxSessionsPerUser 个会话
	count := m.redis.SCard(ctx, sessionsKey).Val()
	if count > int64(maxSessionsPerUser) {
		// 移除最早的一个（FIFO）
		members := m.redis.SPopN(ctx, sessionsKey, count-int64(maxSessionsPerUser)).Val()
		for _, old := range members {
			m.redis.Del(ctx, fmt.Sprintf("%s%s", refreshTokenPrefix, old))
		}
	}

	// 2. 存储token详情（用于校验）
	return m.redis.Set(ctx, tokenKey, fmt.Sprintf("%d", userID), ttl).Err()
}

// Validate 校验 refresh token 是否有效
func (m *RefreshTokenManager) Validate(token string) (uint, error) {
	if m.redis == nil {
		return 0, nil // Redis 不可用时跳过校验
	}

	ctx := context.Background()
	hash := tokenHash(token)
	tokenKey := fmt.Sprintf("%s%s", refreshTokenPrefix, hash)

	val, err := m.redis.Get(ctx, tokenKey).Result()
	if err != nil {
		if err == redis.Nil {
			return 0, fmt.Errorf("refresh token not found or revoked")
		}
		return 0, err
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

	ctx := context.Background()
	hash := tokenHash(token)
	tokenKey := fmt.Sprintf("%s%s", refreshTokenPrefix, hash)
	sessionsKey := fmt.Sprintf("%s%d", userSessionsPrefix, userID)

	m.redis.Del(ctx, tokenKey)
	m.redis.SRem(ctx, sessionsKey, hash)
	return nil
}

// RevokeAll 撤销用户所有 refresh token（登出所有设备）
func (m *RefreshTokenManager) RevokeAll(userID uint) error {
	if m.redis == nil {
		return nil
	}

	ctx := context.Background()
	sessionsKey := fmt.Sprintf("%s%d", userSessionsPrefix, userID)

	// 获取所有 session token hash
	members := m.redis.SMembers(ctx, sessionsKey).Val()
	for _, hash := range members {
		m.redis.Del(ctx, fmt.Sprintf("%s%s", refreshTokenPrefix, hash))
	}

	return m.redis.Del(ctx, sessionsKey).Err()
}
