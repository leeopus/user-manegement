package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/user-system/backend/internal/config"
)

func CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		cfg := config.Get()
		origin := c.Request.Header.Get("Origin")

		allowedOrigin := ""
		if len(cfg.CORS.Origins) > 0 {
			for _, o := range cfg.CORS.Origins {
				if o == origin {
					allowedOrigin = origin
					break
				}
			}
		} else {
			// 默认允许 localhost 开发环境
			if strings.HasPrefix(origin, "http://localhost:") || strings.HasPrefix(origin, "http://127.0.0.1:") {
				allowedOrigin = origin
			}
		}

		if allowedOrigin != "" {
			c.Writer.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
			c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		}
		// 不设置 ACAO 和 Credentials 时，浏览器会阻止跨域请求，这是预期行为

		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
