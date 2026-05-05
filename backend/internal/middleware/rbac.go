package middleware

import (
	"fmt"

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
	UserRepo    repository.UserRepository
	RedisClient *redis.Client
	CacheMgr    *auth.RBACCacheManager
	BlacklistMgr *auth.TokenBlacklistManager
}

func (c *RBACConfig) GetCacheMgr() *auth.RBACCacheManager {
	if c.CacheMgr == nil {
		c.CacheMgr = auth.NewRBACCacheManager(c.RedisClient)
	}
	return c.CacheMgr
}

var roleLoadSF singleflight.Group

func loadRolesWithSF(userRepo repository.UserRepository, cacheMgr *auth.RBACCacheManager, uid uint) ([]auth.RoleData, error) {
	key := fmt.Sprintf("rbac:%d", uid)
	v, err, _ := roleLoadSF.Do(key, func() (interface{}, error) {
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
	if err != nil {
		return nil, err
	}
	return v.([]auth.RoleData), nil
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

		for _, role := range roles {
			for _, perm := range role.Permissions {
				for _, required := range permissionCodes {
					if perm.Code == required {
						c.Set("user_roles", roles)
						c.Next()
						return
					}
				}
			}
		}

		response.Forbidden(c)
		c.Abort()
	}
}

func getOrLoadRoles(userRepo repository.UserRepository, cacheMgr *auth.RBACCacheManager, uid uint) []auth.RoleData {
	cachedRoles, err := cacheMgr.GetUserRoles(uid)
	if err == nil && cachedRoles != nil {
		return cachedRoles
	}

	roles, err := loadRolesWithSF(userRepo, cacheMgr, uid)
	if err != nil {
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
