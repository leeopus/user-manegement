package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/user-system/backend/internal/config"
)

// SecurityHeaders adds standard security headers to all responses.
func SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "SAMEORIGIN")
		c.Header("X-XSS-Protection", "0")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Header("Permissions-Policy", "camera=(), microphone=(), geolocation=()")

		if cfg := config.Get(); cfg != nil && cfg.Server.GinMode == "release" {
			c.Header("Strict-Transport-Security", "max-age=63072000; includeSubDomains; preload")
		}

		c.Next()
	}
}
