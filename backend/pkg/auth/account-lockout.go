package auth

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	// 登录失败配置
	MaxFailedAttempts = 5                     // 最大失败次数
	LockoutDuration    = 30 * time.Minute    // 锁定时长
	AttemptWindow     = 15 * time.Minute     // 失败尝试的时间窗口
)

// AccountLockoutManager 账户锁定管理器
type AccountLockoutManager struct {
	redis *redis.Client
}

// NewAccountLockoutManager 创建账户锁定管理器
func NewAccountLockoutManager(redisClient *redis.Client) *AccountLockoutManager {
	return &AccountLockoutManager{
		redis: redisClient,
	}
}

// RecordFailedAttempt 记录失败尝试
func (m *AccountLockoutManager) RecordFailedAttempt(email string) error {
	if m.redis == nil {
		return nil // Redis 不可用时不记录
	}

	ctx := context.Background()

	// 记录失败次数
	key := fmt.Sprintf("auth:failed_attempts:%s", email)
	count, err := m.redis.Incr(ctx, key).Result()
	if err != nil {
		return err
	}

	// 设置过期时间（第一次失败时）
	if count == 1 {
		m.redis.Expire(ctx, key, AttemptWindow)
	}

	// 检查是否需要锁定账户
	if count >= MaxFailedAttempts {
		// 锁定账户
		lockKey := fmt.Sprintf("auth:locked:%s", email)
		m.redis.Set(ctx, lockKey, strconv.FormatInt(count, 10), LockoutDuration)

		// 清除失败计数
		m.redis.Del(ctx, key)
	}

	return nil
}

// IsAccountLocked 检查账户是否被锁定
func (m *AccountLockoutManager) IsAccountLocked(email string) (bool, time.Duration, error) {
	if m.redis == nil {
		return false, 0, nil // Redis 不可用时不锁定
	}

	ctx := context.Background()
	lockKey := fmt.Sprintf("auth:locked:%s", email)

	ttl := m.redis.TTL(ctx, lockKey).Val()
	if ttl > 0 {
		return true, ttl, nil
	}

	return false, 0, nil
}

// ClearFailedAttempts 清除失败尝试（登录成功时调用）
func (m *AccountLockoutManager) ClearFailedAttempts(email string) error {
	if m.redis == nil {
		return nil
	}

	ctx := context.Background()
	key := fmt.Sprintf("auth:failed_attempts:%s", email)
	return m.redis.Del(ctx, key).Err()
}

// GetRemainingAttempts 获取剩余尝试次数
func (m *AccountLockoutManager) GetRemainingAttempts(email string) (int, error) {
	if m.redis == nil {
		return MaxFailedAttempts, nil // Redis 不可用时返回默认值
	}

	ctx := context.Background()
	key := fmt.Sprintf("auth:failed_attempts:%s", email)

	val, err := m.redis.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return MaxFailedAttempts, nil // 没有失败记录
		}
		return 0, err
	}

	count, _ := strconv.Atoi(val)
	remaining := MaxFailedAttempts - count
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

	ctx := context.Background()
	lockKey := fmt.Sprintf("auth:locked:%s", email)
	attemptKey := fmt.Sprintf("auth:failed_attempts:%s", email)

	// 清除锁定和失败计数
	m.redis.Del(ctx, lockKey)
	return m.redis.Del(ctx, attemptKey).Err()
}
