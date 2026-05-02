package middleware

import (
	"net/http"
	"runtime/debug"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Recovery 全局错误恢复中间件
func Recovery(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				// 获取调用栈
				stack := debug.Stack()

				// 记录详细错误信息
				logger.Error("Panic recovered",
					zap.Any("error", err),
					zap.String("path", c.Request.URL.Path),
					zap.String("method", c.Request.Method),
					zap.String("client_ip", c.ClientIP()),
					zap.String("user_agent", c.Request.UserAgent()),
					zap.ByteString("stack", stack),
				)

				// 返回500错误给客户端
				c.JSON(http.StatusInternalServerError, gin.H{
					"success": false,
					"error": gin.H{
						"code":    "INTERNAL_SERVER_ERROR_500",
						"message": "INTERNAL_SERVER_ERROR",
						"details": gin.H{
							"reason": "Internal server error occurred",
						},
					},
				})
				c.Abort()
			}
		}()

		c.Next()
	}
}
