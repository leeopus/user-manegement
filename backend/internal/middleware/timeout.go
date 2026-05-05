package middleware

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// Timeout 为请求设置 context 超时，下游 DB/Redis 操作会尊重 context 取消
// 不使用 goroutine，避免 gin.Context 的并发安全问题
func Timeout(duration time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), duration)
		defer cancel()

		c.Request = c.Request.WithContext(ctx)
		c.Next()

		// Context 超时且 handler 未写入响应时，返回超时错误
		if ctx.Err() == context.DeadlineExceeded && !c.Writer.Written() {
			c.JSON(http.StatusRequestTimeout, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "REQUEST_TIMEOUT_408",
					"message": "REQUEST_TIMEOUT",
				},
			})
			c.Abort()
		}
	}
}
