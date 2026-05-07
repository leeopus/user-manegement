package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/user-system/backend/internal/dto"
	"github.com/user-system/backend/pkg/csrf"
)

const (
	csrfHeader = "X-CSRF-Token"
)

// CSRF 创建 CSRF 验证中间件（绑定 session，一次性 token）
func CSRF(redisClient *redis.Client) gin.HandlerFunc {
	csrfMgr := csrf.NewCSRFManager(redisClient)

	return func(c *gin.Context) {
		// 对于安全的方法（GET, HEAD, OPTIONS），跳过检查
		if c.Request.Method == "GET" || c.Request.Method == "HEAD" || c.Request.Method == "OPTIONS" {
			c.Next()
			return
		}

		// 仅从 header 获取 token（不使用 query param，防止 URL 泄漏）
		token := c.GetHeader(csrfHeader)

		// 获取当前 session 指纹用于验证绑定
		sessionID := dto.SessionFingerprint(c)

		// 验证一次性 token（使用后立即销毁，前端每次写请求前需获取新 token）
		if err := csrfMgr.ValidateToken(token, sessionID); err != nil {
			c.JSON(http.StatusForbidden, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "CSRF_TOKEN_INVALID_403",
					"message": "CSRF_TOKEN_INVALID",
					"details": gin.H{
						"reason": err.Error(),
					},
				},
			})
			c.Abort()
			return
		}

		c.Next()
	}
}
