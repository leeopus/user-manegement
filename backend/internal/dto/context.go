package dto

import (
	"crypto/sha256"
	"encoding/hex"

	"github.com/gin-gonic/gin"
)

// AuditContext 携带请求级别的审计上下文信息
type AuditContext struct {
	UserID     uint
	IPAddress  string
	UserAgent  string
	RequestID  string
}

// NewAuditContext 从 gin.Context 提取审计上下文
func NewAuditContext(c *gin.Context, userID uint) AuditContext {
	requestID := ""
	if id, ok := c.Get("request_id"); ok {
		if s, ok := id.(string); ok {
			requestID = s
		}
	}
	return AuditContext{
		UserID:    userID,
		IPAddress: c.ClientIP(),
		UserAgent: c.GetHeader("User-Agent"),
		RequestID: requestID,
	}
}

// SessionFingerprint 生成匿名 session 指纹（IP + User-Agent），供 CSRF 绑定使用
func SessionFingerprint(c *gin.Context) string {
	ip := c.ClientIP()
	ua := c.GetHeader("User-Agent")
	h := sha256.Sum256([]byte(ip + "|" + ua))
	return hex.EncodeToString(h[:16])
}
