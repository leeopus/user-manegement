package jwt

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	AccessTokenCookie  = "access_token"
	RefreshTokenCookie = "refresh_token"
)

// isSecureRequest 判断是否为安全连接（HTTPS），支持反向代理场景
func isSecureRequest(c *gin.Context) bool {
	// 直接 TLS 连接
	if c.Request.TLS != nil {
		return true
	}
	// 反向代理场景：检查 X-Forwarded-Proto header
	if c.GetHeader("X-Forwarded-Proto") == "https" {
		return true
	}
	return false
}

// SetTokenCookie 设置 HTTP-only cookie
func SetTokenCookie(c *gin.Context, name, token string, maxAge time.Duration) {
	isSecure := isSecureRequest(c)

	// 生产环境必须使用 Secure 和 SameSite
	c.SetSameSite(http.SameSiteLaxMode) // Lax 比 Strict 更适合 SSO 场景（允许跨站 GET 导航带 cookie）
	c.SetCookie(
		name,
		token,
		int(maxAge.Seconds()),
		"/",       // path
		"",        // domain (空字符串表示当前域名)
		isSecure,  // secure (根据协议自动设置)
		true,      // httpOnly (prevent XSS)
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

	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(
		name,
		"",
		-1,        // maxAge (负值表示立即删除)
		"/",       // path
		"",        // domain
		isSecure,  // secure
		true,      // httpOnly
	)
}

// ClearAllTokenCookies 清除所有 token cookies
func ClearAllTokenCookies(c *gin.Context) {
	ClearTokenCookie(c, AccessTokenCookie)
	ClearTokenCookie(c, RefreshTokenCookie)
}
