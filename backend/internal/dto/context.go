package dto

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/user-system/backend/internal/config"
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

const csrfSessionCookie = "csrf_session"

// SessionFingerprint 获取或创建 CSRF session ID（通过 HttpOnly cookie）
// 每个浏览器实例拥有唯一、随机的 session ID，不受 NAT/User-Agent 影响
func SessionFingerprint(c *gin.Context) string {
	sessionID, err := c.Cookie(csrfSessionCookie)
	if err == nil && sessionID != "" {
		return sessionID
	}

	// 生成新的随机 session ID（32 字节 = 256 bit 熵）
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		// 回退到 IP+UA（仅在 rand.Read 失败时）
		return fallbackSessionFingerprint(c)
	}
	sessionID = hex.EncodeToString(b)

	// 设置 HttpOnly, SameSite=Strict cookie
	cookieSecure := false
	if cfg := config.Get(); cfg != nil && cfg.Security.CookieSecure {
		cookieSecure = true
	}

	c.SetSameSite(http.SameSiteStrictMode)
	c.SetCookie(
		csrfSessionCookie,
		sessionID,
		86400, // 24 小时
		"/",
		"",
		cookieSecure,
		true, // HttpOnly: JavaScript 无法读取
	)

	return sessionID
}

// fallbackSessionFingerprint 仅在随机数生成失败时使用
func fallbackSessionFingerprint(c *gin.Context) string {
	return c.ClientIP()
}
