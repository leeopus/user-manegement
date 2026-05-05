package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	rbacCachePrefix = "rbac:roles:"
	rbacCacheTTL    = 10 * time.Minute
)

type RoleData struct {
	ID          uint             `json:"id"`
	Code        string           `json:"code"`
	Name        string           `json:"name"`
	Permissions []PermissionData `json:"permissions"`
}

type PermissionData struct {
	ID   uint   `json:"id"`
	Code string `json:"code"`
	Name string `json:"name"`
}

type RBACCacheManager struct {
	redis *redis.Client
}

func NewRBACCacheManager(redisClient *redis.Client) *RBACCacheManager {
	return &RBACCacheManager{redis: redisClient}
}

func (m *RBACCacheManager) ctx() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), redisOpTimeout)
}

func (m *RBACCacheManager) GetUserRoles(userID uint) ([]RoleData, error) {
	if m.redis == nil {
		return nil, nil
	}

	ctx, cancel := m.ctx()
	defer cancel()

	key := fmt.Sprintf("%s%d", rbacCachePrefix, userID)
	val, err := m.redis.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, err
	}

	var roles []RoleData
	if err := json.Unmarshal([]byte(val), &roles); err != nil {
		return nil, err
	}
	return roles, nil
}

func (m *RBACCacheManager) SetUserRoles(userID uint, roles []RoleData) error {
	if m.redis == nil {
		return nil
	}

	ctx, cancel := m.ctx()
	defer cancel()

	key := fmt.Sprintf("%s%d", rbacCachePrefix, userID)
	data, err := json.Marshal(roles)
	if err != nil {
		return err
	}
	return m.redis.Set(ctx, key, string(data), rbacCacheTTL).Err()
}

func (m *RBACCacheManager) InvalidateUserRoles(userID uint) error {
	if m.redis == nil {
		return nil
	}
	ctx, cancel := m.ctx()
	defer cancel()

	key := fmt.Sprintf("%s%d", rbacCachePrefix, userID)
	return m.redis.Del(ctx, key).Err()
}
