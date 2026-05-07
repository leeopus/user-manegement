package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/user-system/backend/internal/config"
)

func CORS() gin.HandlerFunc {
	cfg := config.Get()
	allowedOrigins := make(map[string]bool, len(cfg.CORS.Origins))
	for _, o := range cfg.CORS.Origins {
		allowedOrigins[o] = true
	}
	hasConfiguredOrigins := len(cfg.CORS.Origins) > 0
	ginMode := cfg.Server.GinMode
	allowLocalhost := ginMode != "release"

	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")

		allowedOrigin := ""
		if hasConfiguredOrigins {
			if allowedOrigins[origin] {
				allowedOrigin = origin
			}
		} else if allowLocalhost {
			if strings.HasPrefix(origin, "http://localhost:") || strings.HasPrefix(origin, "http://127.0.0.1:") {
				allowedOrigin = origin
			}
		}

		if allowedOrigin != "" {
			c.Writer.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
			c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
			c.Writer.Header().Add("Vary", "Origin")
		}

		if c.Request.Method == http.MethodOptions {
			// 仅对合法 Origin 设置预检头，防止浏览器缓存未授权 Origin 的预检结果
			if allowedOrigin != "" {
				c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
				c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")
				c.Writer.Header().Set("Access-Control-Max-Age", "86400")
			}
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
