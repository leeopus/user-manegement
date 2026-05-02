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

// SetTokenCookie 设置 HTTP-only cookie
func SetTokenCookie(c *gin.Context, name, token string, maxAge time.Duration) {
	// 检测是否为HTTPS连接
	isSecure := c.Request.TLS != nil

	// 生产环境必须使用 Secure 和 SameSite
	c.SetSameSite(http.SameSiteStrictMode)
	c.SetCookie(
		name,
		token,
		int(maxAge.Seconds()),
		"/",        // path
		"",         // domain (空字符串表示当前域名)
		isSecure,   // secure (根据协议自动设置)
		true,       // httpOnly (prevent XSS)
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
	// 检测是否为HTTPS连接
	isSecure := c.Request.TLS != nil

	c.SetSameSite(http.SameSiteStrictMode)
	c.SetCookie(
		name,
		"",
		-1,         // maxAge (负值表示立即删除)
		"/",        // path
		"",         // domain
		isSecure,   // secure (根据协议自动设置)
		true,       // httpOnly
	)
}

// ClearAllTokenCookies 清除所有 token cookies
func ClearAllTokenCookies(c *gin.Context) {
	ClearTokenCookie(c, AccessTokenCookie)
	ClearTokenCookie(c, RefreshTokenCookie)
}
