package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/user-system/backend/pkg/jwt"
	"github.com/user-system/backend/pkg/response"
	"github.com/user-system/backend/pkg/utils"
)

func Auth() gin.HandlerFunc {
	return func(c *gin.Context) {
		var tokenString string

		// 1. 首先尝试从 Authorization header 获取 token
		authHeader := c.GetHeader("Authorization")
		if authHeader != "" && len(authHeader) >= 7 && authHeader[:6] == "Bearer" {
			tokenString = authHeader[7:]
		} else {
			// 2. 如果没有 header，尝试从 cookie 获取 token（支持"记住我"功能）
			token, err := jwt.GetTokenCookie(c, jwt.AccessTokenCookie)
			if err == nil && token != "" {
				tokenString = token
			} else {
				// 3. 如果 access_token 过期，尝试使用 refresh_token
				refreshToken, err := jwt.GetTokenCookie(c, jwt.RefreshTokenCookie)
				if err == nil && refreshToken != "" {
					// 验证 refresh token 并自动刷新
					claims, err := utils.ParseToken(refreshToken)
					if err == nil {
						// 生成新的 access token
						newAccessToken, err := utils.GenerateToken(claims.UserID, claims.Username, claims.Email)
						if err == nil {
							// 设置新的 access token cookie (15分钟)
							jwt.SetTokenCookie(c, jwt.AccessTokenCookie, newAccessToken, 15*time.Minute)
							tokenString = newAccessToken
						}
					}
				}
			}
		}

		// 如果都没有获取到 token，返回未授权
		if tokenString == "" {
			response.Unauthorized(c)
			c.Abort()
			return
		}

		// 验证 token
		claims, err := utils.ParseToken(tokenString)
		if err != nil {
			response.Unauthorized(c)
			c.Abort()
			return
		}

		// 将用户信息存入 context
		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)
		c.Set("email", claims.Email)

		c.Next()
	}
}

func OAuthAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			response.Unauthorized(c)
			c.Abort()
			return
		}

		if len(authHeader) < 7 || authHeader[:6] != "Bearer" {
			response.Unauthorized(c)
			c.Abort()
			return
		}

		tokenString := authHeader[7:]

		// 验证 token
		claims, err := utils.ParseToken(tokenString)
		if err != nil {
			response.Unauthorized(c)
			c.Abort()
			return
		}

		// 将用户信息存入 context
		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)
		c.Set("email", claims.Email)

		c.Next()
	}
}
