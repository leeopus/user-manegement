package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/user-system/backend/pkg/csrf"
)

const (
	csrfHeader    = "X-CSRF-Token"
	csrfQueryParam = "csrf_token"
)

// CSRF 中间件 - 验证 CSRF token
func CSRF() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 对于安全的方法（GET, HEAD, OPTIONS），跳过检查
		if c.Request.Method == "GET" || c.Request.Method == "HEAD" || c.Request.Method == "OPTIONS" {
			c.Next()
			return
		}

		// 获取 token
		token := c.GetHeader(csrfHeader)
		if token == "" {
			token = c.PostForm(csrfQueryParam)
		}
		if token == "" {
			token = c.Query(csrfQueryParam)
		}

		// 验证 token
		if err := csrf.DefaultManager.ValidateToken(token); err != nil {
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

// CSRFExempt 排除某些路由的 CSRF 检查
func CSRFExempt(paths ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		for _, path := range paths {
			if strings.HasPrefix(c.Request.URL.Path, path) {
				c.Next()
				return
			}
		}
		CSRF()(c)
	}
}
