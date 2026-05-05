package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/user-system/backend/pkg/response"
	"gorm.io/gorm"

	"github.com/user-system/backend/internal/repository"
)

// RequireRole 检查当前用户是否拥有指定角色
func RequireRole(db *gorm.DB, roleCodes ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists {
			response.Unauthorized(c)
			c.Abort()
			return
		}

		userRepo := repository.NewUserRepository(db)
		roles, err := userRepo.GetUserRoles(userID.(uint))
		if err != nil {
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

// RequirePermission 检查当前用户是否拥有指定权限
func RequirePermission(db *gorm.DB, permissionCodes ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists {
			response.Unauthorized(c)
			c.Abort()
			return
		}

		userRepo := repository.NewUserRepository(db)
		roles, err := userRepo.GetUserRoles(userID.(uint))
		if err != nil {
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
