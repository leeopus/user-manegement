package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/user-system/backend/internal/repository"
	"github.com/user-system/backend/pkg/auth"
	"github.com/user-system/backend/pkg/response"
	"github.com/user-system/backend/pkg/utils"
	"go.uber.org/zap"
)

const (
	accessTokenCookie  = "access_token"
	refreshTokenCookie = "refresh_token"
)

// tokenSource 控制 token 提取来源
type tokenSource int

const (
	tokenSourceAll    tokenSource = iota // header + cookie fallback
	tokenSourceHeaderOnly                // header only (OAuth)
)

// tokenRestriction 控制 token 类型限制
type tokenRestriction int

const (
	restrictionNone      tokenRestriction = iota // 无限制
	restrictionNoOAuth                            // 拒绝 OAuth token（常规 API 用）
	restrictionOAuthOnly                          // 仅允许 OAuth token（OAuth userinfo 用）
)

// extractBearerToken 从 Authorization header 提取 Bearer token
func extractBearerToken(authHeader string) string {
	if !strings.HasPrefix(authHeader, "Bearer ") {
		return ""
	}
	return strings.TrimPrefix(authHeader, "Bearer ")
}

// getTokenCookie 从 cookie 获取 token
func getTokenCookie(c *gin.Context, name string) (string, error) {
	token, err := c.Cookie(name)
	if err != nil {
		return "", err
	}
	return token, nil
}

// authenticate 通用认证逻辑（Auth 和 OAuthAuth 共用）
func authenticate(c *gin.Context, blacklistMgr *auth.TokenBlacklistManager, userRepo repository.UserRepository, source tokenSource, restriction tokenRestriction) {
	var tokenString string

	authHeader := c.GetHeader("Authorization")
	if tokenString = extractBearerToken(authHeader); tokenString == "" {
		if source == tokenSourceAll {
			token, err := getTokenCookie(c, accessTokenCookie)
			if err == nil && token != "" {
				tokenString = token
			}
		}
	}

	if tokenString == "" {
		response.Unauthorized(c)
		c.Abort()
		return
	}

	claims, err := utils.ParseToken(tokenString)
	if err != nil {
		response.Unauthorized(c)
		c.Abort()
		return
	}

	if claims.TokenType == "refresh" {
		response.Unauthorized(c)
		c.Abort()
		return
	}

	// OAuth token 隔离：根据路由限制 token 类型
	if restriction == restrictionNoOAuth && claims.ClientID != "" {
		response.Forbidden(c)
		c.Abort()
		return
	}
	if restriction == restrictionOAuthOnly && claims.ClientID == "" {
		response.Forbidden(c)
		c.Abort()
		return
	}

	revoked, blacklisted, err := blacklistMgr.CheckTokenStatus(c.Request.Context(), claims.UserID, claims.JTI)
	if err != nil {
		zap.L().Warn("Token status check failed, rejecting request for security", zap.Error(err))
		response.Unauthorized(c)
		c.Abort()
		return
	}
	if revoked || blacklisted {
		response.Unauthorized(c)
		c.Abort()
		return
	}

	// 检查用户状态：Redis 缓存命中时直接判断，缓存 miss 时回源 DB 确认
	active, cached, statusErr := blacklistMgr.CheckUserStatus(claims.UserID)
	if statusErr != nil {
		zap.L().Warn("User status check failed, rejecting request for security", zap.Error(statusErr))
		response.Unauthorized(c)
		c.Abort()
		return
	}
	if cached && !active {
		response.Forbidden(c)
		c.Abort()
		return
	}
	// 缓存 miss：回源 DB 检查用户是否仍为 active 状态
	if !cached && userRepo != nil {
		user, dbErr := userRepo.FindByID(claims.UserID)
		if dbErr != nil {
			// 用户不存在，拒绝访问
			response.Unauthorized(c)
			c.Abort()
			return
		}
		if user.Status != "active" {
			// 用户已被禁用/删除，写入缓存以加速后续判断，并拒绝本次访问
			_ = blacklistMgr.SetUserStatus(user.ID, user.Status)
			response.Forbidden(c)
			c.Abort()
			return
		}
		// 用户状态正常，回填缓存
		_ = blacklistMgr.SetUserStatus(user.ID, "active")
	}

	c.Set("user_id", claims.UserID)
	c.Set("username", claims.Username)
	c.Set("email", claims.Email)

	c.Next()
}

// AuthConfig 认证中间件配置
type AuthConfig struct {
	BlacklistMgr *auth.TokenBlacklistManager
	UserRepo     repository.UserRepository
}

// Auth 创建认证中间件（支持 Bearer header + cookie fallback，拒绝 OAuth token）
func Auth(cfg AuthConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		authenticate(c, cfg.BlacklistMgr, cfg.UserRepo, tokenSourceAll, restrictionNoOAuth)
	}
}

// OAuthAuth 创建 OAuth 认证中间件（仅 Bearer header，仅允许 OAuth token）
func OAuthAuth(cfg AuthConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		authenticate(c, cfg.BlacklistMgr, cfg.UserRepo, tokenSourceHeaderOnly, restrictionOAuthOnly)
	}
}
