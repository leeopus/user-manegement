package middleware

import (
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/user-system/backend/internal/config"
)

func CORS() gin.HandlerFunc {
	// 启动时一次性解析配置，避免每次请求重复读取
	cfg := config.Get()
	allowedOrigins := make(map[string]bool, len(cfg.CORS.Origins))
	for _, o := range cfg.CORS.Origins {
		allowedOrigins[o] = true
	}
	hasConfiguredOrigins := len(cfg.CORS.Origins) > 0
	ginMode := os.Getenv("GIN_MODE")
	// 需要显式启用 localhost CORS，而非仅依赖 GIN_MODE
	allowLocalhost := ginMode != "release" && os.Getenv("ALLOW_LOCALHOST_CORS") != "false"

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
			c.Writer.Header().Set("Vary", "Origin")
		}

		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")
		c.Writer.Header().Set("Access-Control-Max-Age", "86400")

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
