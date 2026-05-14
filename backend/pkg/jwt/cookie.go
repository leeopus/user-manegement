package jwt

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/user-system/backend/internal/config"
)

const (
	AccessTokenCookie  = "access_token"
	RefreshTokenCookie = "refresh_token"
)

// isSecureRequest 判断是否应使用 Secure cookie 标志
func isSecureRequest(c *gin.Context) bool {
	cfg := config.Get()
	if cfg != nil && cfg.Security.CookieSecure {
		return true
	}
	// 仅根据实际 TLS 连接判断，不信任 X-Forwarded-Proto（可被伪造）
	if c.Request.TLS != nil {
		return true
	}
	return false
}

// SetTokenCookie 设置 HTTP-only cookie
func SetTokenCookie(c *gin.Context, name, token string, maxAge time.Duration) {
	isSecure := isSecureRequest(c)

	c.SetSameSite(http.SameSiteStrictMode)
	c.SetCookie(
		name,
		token,
		int(maxAge.Seconds()),
		"/",
		"",
		isSecure,
		true,
	)
}

// SetSessionTokenCookie 设置 HTTP-only session cookie（MaxAge=0，关闭浏览器即清除）
func SetSessionTokenCookie(c *gin.Context, name, token string) {
	isSecure := isSecureRequest(c)

	c.SetSameSite(http.SameSiteStrictMode)
	c.SetCookie(
		name,
		token,
		0,
		"/",
		"",
		isSecure,
		true,
	)
}

// GetTokenCookie 获取 cookie 中的 token
func GetTokenCookie(c *gin.Context, name string) (string, error) {
	token, err := c.Cookie(name)
	if err != nil {
		return "", err
	}
	return token, nil
}

// ClearTokenCookie 清除 cookie
func ClearTokenCookie(c *gin.Context, name string) {
	isSecure := isSecureRequest(c)

	c.SetSameSite(http.SameSiteStrictMode)
	c.SetCookie(
		name,
		"",
		-1,
		"/",
		"",
		isSecure,
		true,
	)
}

// ClearAllTokenCookies 清除所有 token cookies
func ClearAllTokenCookies(c *gin.Context) {
	ClearTokenCookie(c, AccessTokenCookie)
	ClearTokenCookie(c, RefreshTokenCookie)
}
