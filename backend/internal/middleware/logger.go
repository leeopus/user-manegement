package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/user-system/backend/pkg/logger"
)

func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		c.Next()

		end := time.Now()
		latency := end.Sub(start)

		logger.Info("HTTP Request",
			logger.Logger.With(
				"method", c.Request.Method,
				"path", path,
				"query", query,
				"status", c.Writer.Status(),
				"latency", latency,
				"ip", c.ClientIP(),
				"user_agent", c.Request.UserAgent(),
			),
		)
	}
}
