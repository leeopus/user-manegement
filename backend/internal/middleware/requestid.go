package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"regexp"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

const requestIDKey = "request_id"

// validRequestIDPattern 外部传入的 X-Request-ID 仅允许 hex 字符，32 或 36 字符（含连字符的 UUID）
var validRequestIDPattern = regexp.MustCompile(`^[0-9a-fA-F\-]{1,64}$`)

// RequestID 为每个请求生成唯一 ID
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" || !validRequestIDPattern.MatchString(requestID) {
			requestID = generateRequestID()
		}

		c.Set(requestIDKey, requestID)
		c.Header("X-Request-ID", requestID)
		c.Next()
	}
}

// GetRequestID 从 gin context 获取 request ID
func GetRequestID(c *gin.Context) string {
	if id, ok := c.Get(requestIDKey); ok {
		if s, ok := id.(string); ok {
			return s
		}
	}
	return ""
}

// LoggerWithRequestID 创建带 request ID 的子 logger
func LoggerWithRequestID(c *gin.Context) *zap.Logger {
	return zap.L().With(zap.String("request_id", GetRequestID(c)))
}

func generateRequestID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		// 降级：使用时间戳
		return "fallback"
	}
	return hex.EncodeToString(b)
}
