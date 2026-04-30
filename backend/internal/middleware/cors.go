package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/user-system/backend/internal/config"
)

func CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		cfg := config.Get()

		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		if len(cfg.CORS.Origins) > 0 {
			origin := c.Request.Header.Get("Origin")
			allowed := false
			for _, o := range cfg.CORS.Origins {
				if o == "*" || o == origin {
					allowed = true
					break
				}
			}
			if allowed {
				c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
			}
		}

		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}
