package auth

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/user-system/backend/internal/config"
)

const (
	attemptWindow = 15 * time.Minute
)

// AccountLockoutManager 账户锁定管理器（IP + Email 双维度 + Email 全局聚合）
type AccountLockoutManager struct {
	redis *redis.Client
}

func NewAccountLockoutManager(redisClient *redis.Client) *AccountLockoutManager {
	return &AccountLockoutManager{redis: redisClient}
}

func getLockoutConfig() (maxFailed int, maxTotal int, lockoutDur time.Duration) {
	cfg := config.Get()
	if cfg != nil {
		return cfg.Security.MaxFailedAttempts, cfg.Security.MaxTotalAttempts,
			time.Duration(cfg.Security.LockoutDurationMin) * time.Minute
	}
	return 5, 15, 30 * time.Minute
}

func (m *AccountLockoutManager) ctx() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), redisOpTimeout)
}

func failedAttemptsKey(email, ip string) string {
	return fmt.Sprintf("auth:failed_attempts:%s:%s", email, ip)
}

func totalFailedAttemptsKey(email string) string {
	return fmt.Sprintf("auth:total_failed:%s", email)
}

func lockedKey(email string) string {
	return fmt.Sprintf("auth:locked:%s", email)
}

// RecordFailedAttempt 记录来自特定 IP 对特定 email 的失败尝试
func (m *AccountLockoutManager) RecordFailedAttempt(email, ip string) error {
	if m.redis == nil {
		return nil
	}

	ctx, cancel := m.ctx()
	defer cancel()
	key := failedAttemptsKey(email, ip)

	count, err := m.redis.Incr(ctx, key).Result()
	if err != nil {
		return err
	}

	if count == 1 {
		if err := m.redis.Expire(ctx, key, attemptWindow).Err(); err != nil {
			return fmt.Errorf("failed to set attempt window expiry: %w", err)
		}
	}

	maxFailed, maxTotal, lockoutDur := getLockoutConfig()

	// 增加 email 维度总计数
	totalKey := totalFailedAttemptsKey(email)
	total, err := m.redis.Incr(ctx, totalKey).Result()
	if err != nil {
		return err
	}
	if total == 1 {
		if err := m.redis.Expire(ctx, totalKey, lockoutDur).Err(); err != nil {
			return fmt.Errorf("failed to set total attempt window expiry: %w", err)
		}
	}

	// 单 IP 超限 或 email 维度总计数超限 都触发锁定
	if count >= int64(maxFailed) || total >= int64(maxTotal) {
		lockK := lockedKey(email)
		if err := m.redis.Set(ctx, lockK, strconv.FormatInt(total, 10), lockoutDur).Err(); err != nil {
			return fmt.Errorf("failed to set lockout: %w", err)
		}
		m.redis.Del(ctx, key)
	}

	return nil
}

// IsAccountLocked 检查账户是否被锁定（全局维度）
func (m *AccountLockoutManager) IsAccountLocked(email string) (bool, time.Duration, error) {
	if m.redis == nil {
		return false, 0, fmt.Errorf("account lockout check unavailable: cannot verify account status")
	}

	ctx, cancel := m.ctx()
	defer cancel()
	lockK := lockedKey(email)

	ttl := m.redis.TTL(ctx, lockK).Val()
	if ttl > 0 {
		return true, ttl, nil
	}

	return false, 0, nil
}

// ClearFailedAttempts 清除来自特定 IP 的失败尝试和 email 总计数
func (m *AccountLockoutManager) ClearFailedAttempts(email, ip string) error {
	if m.redis == nil {
		return nil
	}

	ctx, cancel := m.ctx()
	defer cancel()
	key := failedAttemptsKey(email, ip)
	m.redis.Del(ctx, key)

	// 登录成功时清除 email 维度总计数
	totalKey := totalFailedAttemptsKey(email)
	m.redis.Del(ctx, totalKey)

	return nil
}

// GetRemainingAttempts 获取来自特定 IP 的剩余尝试次数
func (m *AccountLockoutManager) GetRemainingAttempts(email, ip string) (int, error) {
	if m.redis == nil {
		maxFailed, _, _ := getLockoutConfig()
		return maxFailed, nil
	}

	ctx, cancel := m.ctx()
	defer cancel()
	key := failedAttemptsKey(email, ip)

	val, err := m.redis.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			maxFailed, _, _ := getLockoutConfig()
			return maxFailed, nil
		}
		return 0, err
	}

	count, _ := strconv.Atoi(val)
	maxFailed, _, _ := getLockoutConfig()
	remaining := maxFailed - count
	if remaining < 0 {
		remaining = 0
	}

	return remaining, nil
}

// UnlockAccount 解锁账户（管理员功能）
func (m *AccountLockoutManager) UnlockAccount(email string) error {
	if m.redis == nil {
		return nil
	}

	ctx, cancel := m.ctx()
	defer cancel()
	lockK := lockedKey(email)
	m.redis.Del(ctx, lockK)
	m.redis.Del(ctx, totalFailedAttemptsKey(email))

	// 使用 SCAN 清除所有 IP 维度的失败计数
	pattern := fmt.Sprintf("auth:failed_attempts:%s:*", email)
	iter := m.redis.Scan(ctx, 0, pattern, 100).Iterator()
	for iter.Next(ctx) {
		m.redis.Del(ctx, iter.Val())
	}

	return iter.Err()
}
