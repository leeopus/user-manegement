package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

const maxBodyBytes = 1 << 20 // 1 MB

// BodyLimit 限制请求体大小，防止 OOM 攻击
func BodyLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Body != nil {
			c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBodyBytes)
		}
		c.Next()
	}
}
