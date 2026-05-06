package middleware

import (
	"context"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/user-system/backend/internal/repository"
	"github.com/user-system/backend/pkg/auth"
	"github.com/user-system/backend/pkg/response"
	"go.uber.org/zap"
	"golang.org/x/sync/singleflight"
)

// RBACConfig RBAC 中间件配置
type RBACConfig struct {
	UserRepo     repository.UserRepository
	RedisClient  *redis.Client
	CacheMgr     *auth.RBACCacheManager
	BlacklistMgr *auth.TokenBlacklistManager
}

func (c *RBACConfig) GetCacheMgr() *auth.RBACCacheManager {
	if c.CacheMgr == nil {
		c.CacheMgr = auth.NewRBACCacheManager(c.RedisClient)
	}
	return c.CacheMgr
}

var roleLoadSF singleflight.Group

const rbacDBTimeout = 3 * time.Second

func loadRolesWithSF(userRepo repository.UserRepository, cacheMgr *auth.RBACCacheManager, uid uint) ([]auth.RoleData, error) {
	key := fmt.Sprintf("rbac:%d", uid)
	ctx, cancel := context.WithTimeout(context.Background(), rbacDBTimeout)
	defer cancel()

	resultCh := roleLoadSF.DoChan(key, func() (interface{}, error) {
		// singleflight 回调内部无法直接受 context 取消，
		// 但 DB 查询本身会在 rbacDBTimeout 后由外部逻辑丢弃结果
		dbRoles, dbErr := userRepo.GetUserRoles(uid)
		if dbErr != nil {
			return nil, dbErr
		}

		roleDataList := convertToRoleData(dbRoles)

		if cacheErr := cacheMgr.SetUserRoles(uid, roleDataList); cacheErr != nil {
			zap.L().Warn("RBAC cache write failed", zap.Uint("user_id", uid), zap.Error(cacheErr))
		}

		return roleDataList, nil
	})

	select {
	case result := <-resultCh:
		if result.Err != nil {
			return nil, result.Err
		}
		return result.Val.([]auth.RoleData), nil
	case <-ctx.Done():
		zap.L().Error("RBAC role loading timed out", zap.Uint("user_id", uid), zap.Duration("timeout", rbacDBTimeout))
		return nil, fmt.Errorf("rbac load timeout for user %d", uid)
	}
}

// RequireRole 检查当前用户是否拥有指定角色（带 Redis 缓存）
func RequireRole(cfg RBACConfig, roleCodes ...string) gin.HandlerFunc {
	cacheMgr := cfg.GetCacheMgr()

	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists {
			response.Unauthorized(c)
			c.Abort()
			return
		}

		uid, ok := userID.(uint)
		if !ok {
			response.Unauthorized(c)
			c.Abort()
			return
		}

		roles := getOrLoadRoles(cfg.UserRepo, cacheMgr, uid)
		if roles == nil {
			response.Forbidden(c)
			c.Abort()
			return
		}

		for _, role := range roles {
			for _, required := range roleCodes {
				if role.Code == required {
					c.Set("user_roles", roles)
					c.Next()
					return
				}
			}
		}

		response.Forbidden(c)
		c.Abort()
	}
}

// RequirePermission 检查当前用户是否拥有指定权限（带 Redis 缓存）
func RequirePermission(cfg RBACConfig, permissionCodes ...string) gin.HandlerFunc {
	cacheMgr := cfg.GetCacheMgr()

	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists {
			response.Unauthorized(c)
			c.Abort()
			return
		}

		uid, ok := userID.(uint)
		if !ok {
			response.Unauthorized(c)
			c.Abort()
			return
		}

		roles := getOrLoadRoles(cfg.UserRepo, cacheMgr, uid)
		if roles == nil {
			response.Forbidden(c)
			c.Abort()
			return
		}

		// 预构建 map 索引，将 O(R*P*N) 降为 O(P+N)
		permMap := buildPermissionMap(roles)
		for _, required := range permissionCodes {
			if permMap[required] {
				c.Set("user_roles", roles)
				c.Next()
				return
			}
		}

		response.Forbidden(c)
		c.Abort()
	}
}

// buildPermissionMap 将角色-权限展平为 map[code]bool，用于 O(1) 查找
func buildPermissionMap(roles []auth.RoleData) map[string]bool {
	m := make(map[string]bool, len(roles)*4)
	for _, role := range roles {
		for _, perm := range role.Permissions {
			m[perm.Code] = true
		}
	}
	return m
}

func getOrLoadRoles(userRepo repository.UserRepository, cacheMgr *auth.RBACCacheManager, uid uint) []auth.RoleData {
	cachedRoles, err := cacheMgr.GetUserRoles(uid)
	if err == nil && cachedRoles != nil {
		return cachedRoles
	}

	roles, err := loadRolesWithSF(userRepo, cacheMgr, uid)
	if err != nil {
		zap.L().Error("CRITICAL: RBAC role loading failed (both Redis cache miss and DB error), all permission checks will deny access",
			zap.Uint("user_id", uid),
			zap.Error(err),
		)
		return nil
	}
	return roles
}

func convertToRoleData(roles []repository.Role) []auth.RoleData {
	roleDataList := make([]auth.RoleData, 0, len(roles))
	for _, role := range roles {
		permDataList := make([]auth.PermissionData, 0, len(role.Permissions))
		for _, perm := range role.Permissions {
			permDataList = append(permDataList, auth.PermissionData{
				ID:   perm.ID,
				Code: perm.Code,
				Name: perm.Name,
			})
		}
		roleDataList = append(roleDataList, auth.RoleData{
			ID:          role.ID,
			Code:        role.Code,
			Name:        role.Name,
			Permissions: permDataList,
		})
	}
	return roleDataList
}
