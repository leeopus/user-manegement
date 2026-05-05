package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/user-system/backend/internal/config"
)

const (
	tokenBlacklistPrefix = "auth:token_blacklist:"
	userRevokedPrefix    = "auth:user_revoked:"
	userStatusPrefix     = "auth:user_status:"
	redisOpTimeout       = 2 * time.Second
)

func getAccessTokenMaxTTL() time.Duration {
	cfg := config.Get()
	if cfg != nil && cfg.Security.AccessTokenMaxTTLMin > 0 {
		return time.Duration(cfg.Security.AccessTokenMaxTTLMin) * time.Minute
	}
	return 1 * time.Hour
}

// TokenBlacklistManager 管理 JWT token 黑名单
type TokenBlacklistManager struct {
	redis *redis.Client
}

// NewTokenBlacklistManager 创建 token 黑名单管理器
func NewTokenBlacklistManager(redisClient *redis.Client) *TokenBlacklistManager {
	return &TokenBlacklistManager{redis: redisClient}
}

func (m *TokenBlacklistManager) ctx() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), redisOpTimeout)
}

// AddToBlacklist 将 token 加入黑名单，TTL 为 token 剩余有效期
func (m *TokenBlacklistManager) AddToBlacklist(jti string, remainingTTL time.Duration) error {
	if m.redis == nil || jti == "" {
		return nil
	}
	ctx, cancel := m.ctx()
	defer cancel()

	key := fmt.Sprintf("%s%s", tokenBlacklistPrefix, jti)
	return m.redis.Set(ctx, key, "1", remainingTTL).Err()
}

// IsBlacklisted 检查 token 是否在黑名单中
func (m *TokenBlacklistManager) IsBlacklisted(jti string) (bool, error) {
	if m.redis == nil || jti == "" {
		return false, nil
	}
	ctx, cancel := m.ctx()
	defer cancel()

	key := fmt.Sprintf("%s%s", tokenBlacklistPrefix, jti)
	_, err := m.redis.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// CheckTokenStatus 使用 Pipeline 一次性检查用户级吊销 + token 级黑名单
func (m *TokenBlacklistManager) CheckTokenStatus(ctx context.Context, userID uint, jti string) (userRevoked bool, tokenBlacklisted bool, err error) {
	if m.redis == nil || userID == 0 {
		// Redis 不可用时拒绝请求，防止已吊销的 token 继续使用
		return false, false, fmt.Errorf("token blacklist unavailable: security check cannot be skipped")
	}

	var cancel context.CancelFunc
	if _, ok := ctx.Deadline(); !ok {
		ctx, cancel = context.WithTimeout(ctx, redisOpTimeout)
		defer cancel()
	}

	userKey := fmt.Sprintf("%s%d", userRevokedPrefix, userID)
	tokenKey := fmt.Sprintf("%s%s", tokenBlacklistPrefix, jti)

	pipe := m.redis.Pipeline()
	userCmd := pipe.Get(ctx, userKey)
	tokenCmd := pipe.Get(ctx, tokenKey)
	if _, execErr := pipe.Exec(ctx); execErr != nil && execErr != redis.Nil {
		// Pipeline 执行失败（非 key 不存在），拒绝请求
		return false, false, fmt.Errorf("token blacklist check failed: %w", execErr)
	}

	_, userErr := userCmd.Result()
	_, tokenErr := tokenCmd.Result()

	userRevoked = userErr == nil
	tokenBlacklisted = tokenErr == nil
	return userRevoked, tokenBlacklisted, nil
}

// RevokeAllUserTokens 吊销用户所有已发出的 access token（通过用户级吊销标记）
func (m *TokenBlacklistManager) RevokeAllUserTokens(userID uint) error {
	if m.redis == nil || userID == 0 {
		return nil
	}
	ctx, cancel := m.ctx()
	defer cancel()

	key := fmt.Sprintf("%s%d", userRevokedPrefix, userID)
	return m.redis.Set(ctx, key, fmt.Sprintf("%d", time.Now().Unix()), getAccessTokenMaxTTL()).Err()
}

// IsUserRevoked 检查用户是否被全量吊销
func (m *TokenBlacklistManager) IsUserRevoked(userID uint) (bool, error) {
	if m.redis == nil || userID == 0 {
		return false, nil
	}
	ctx, cancel := m.ctx()
	defer cancel()

	key := fmt.Sprintf("%s%d", userRevokedPrefix, userID)
	_, err := m.redis.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// SetUserStatus 缓存用户状态到 Redis（状态变更时调用）
func (m *TokenBlacklistManager) SetUserStatus(userID uint, status string) error {
	if m.redis == nil || userID == 0 {
		return nil
	}
	ctx, cancel := m.ctx()
	defer cancel()

	key := fmt.Sprintf("%s%d", userStatusPrefix, userID)
	// 状态缓存 TTL 与 access token 最长有效期对齐
	return m.redis.Set(ctx, key, status, getAccessTokenMaxTTL()).Err()
}

// CheckUserStatus 检查用户状态是否允许访问（active 以外的状态均拒绝）
func (m *TokenBlacklistManager) CheckUserStatus(userID uint) (bool, error) {
	if m.redis == nil || userID == 0 {
		// Redis 不可用时放行，避免锁死所有用户
		return true, nil
	}
	ctx, cancel := m.ctx()
	defer cancel()

	key := fmt.Sprintf("%s%d", userStatusPrefix, userID)
	val, err := m.redis.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			// 无缓存记录 = 未被标记为非活跃，放行
			return true, nil
		}
		return false, err
	}
	return val == "active", nil
}
