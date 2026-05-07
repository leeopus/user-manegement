package handler

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/user-system/backend/internal/config"
	"github.com/user-system/backend/internal/dto"
	"github.com/user-system/backend/internal/service"
	"github.com/user-system/backend/pkg/jwt"
	"github.com/user-system/backend/pkg/response"
	apperrors "github.com/user-system/backend/pkg/errors"
)

// isSilentRegisterError 判断是否为防止邮箱枚举的静默成功错误
func isSilentRegisterError(err error) bool {
	if appErr, ok := apperrors.IsAppError(err); ok {
		return appErr.Code == "AUTH_REGISTER_SILENT_201"
	}
	return false
}

type AuthHandler interface {
	Register(c *gin.Context)
	Login(c *gin.Context)
	Logout(c *gin.Context)
	RefreshToken(c *gin.Context)
	GetCurrentUser(c *gin.Context)
	ChangePassword(c *gin.Context)
}

type authHandler struct {
	authService service.AuthService
}

func NewAuthHandler(authService service.AuthService) AuthHandler {
	return &authHandler{authService: authService}
}

type RegisterRequest struct {
	Email    string `json:"email" binding:"required,max=254"`
	Password string `json:"password" binding:"required,min=8,max=64"`
}

type LoginRequest struct {
	Email      string `json:"email" binding:"required,max=254"`
	Password   string `json:"password" binding:"required,max=64"`
	RememberMe bool   `json:"remember_me"`
}

type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

func (h *authHandler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, err.Error())
		return
	}

	auditCtx := dto.NewAuditContext(c, 0)

	_, err := h.authService.Register(req.Email, req.Password, auditCtx)
	if err != nil {
		if isSilentRegisterError(err) {
			response.Created(c, gin.H{
				"message": "registration_processed",
			})
			return
		}
		response.Error(c, err)
		return
	}

	// 统一响应结构，防止邮箱枚举
	response.Created(c, gin.H{
		"message": "registration_processed",
	})
}

func (h *authHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, err.Error())
		return
	}

	user, accessToken, refreshToken, err := h.authService.Login(req.Email, req.Password, c.ClientIP(), c.GetHeader("User-Agent"), req.RememberMe)
	if err != nil {
		response.Error(c, err)
		return
	}

	// 登录成功后轮换 CSRF session，防止会话固定攻击
	dto.RotateCSRFSession(c)

	cfg := config.Get()
	accessTTL := time.Duration(cfg.Security.AccessTokenMaxTTLMin) * time.Minute
	refreshTTL := cfg.GetRefreshTokenTTL()

	jwt.SetTokenCookie(c, jwt.AccessTokenCookie, accessToken, accessTTL)
	jwt.SetTokenCookie(c, jwt.RefreshTokenCookie, refreshToken, refreshTTL)

	response.Success(c, gin.H{
		"user": dto.ToUserWithRolesResponse(user),
	})
}

func (h *authHandler) Logout(c *gin.Context) {
	userIDVal, exists := c.Get("user_id")
	if !exists {
		response.Unauthorized(c)
		return
	}
	userID, ok := userIDVal.(uint)
	if !ok {
		response.Error(c, apperrors.ErrInternalServer)
		return
	}

	refreshToken, _ := jwt.GetTokenCookie(c, jwt.RefreshTokenCookie)
	accessToken, _ := jwt.GetTokenCookie(c, jwt.AccessTokenCookie)

	_ = h.authService.Logout(userID, refreshToken, accessToken)

	jwt.ClearAllTokenCookies(c)

	response.Success(c, gin.H{
		"message": "logged_out",
	})
}

func (h *authHandler) RefreshToken(c *gin.Context) {
	refreshToken, err := jwt.GetTokenCookie(c, jwt.RefreshTokenCookie)
	if err != nil {
		response.Error(c, apperrors.ErrInvalidRefreshToken)
		return
	}

	user, newAccessToken, newRefreshToken, _, err := h.authService.RefreshToken(refreshToken)
	if err != nil {
		response.Error(c, err)
		return
	}

	// Access Token Cookie 始终对齐 JWT TTL；RememberMe 仅影响 Refresh Token
	cfg := config.Get()
	accessTTL := time.Duration(cfg.Security.AccessTokenMaxTTLMin) * time.Minute
	refreshTTL := cfg.GetRefreshTokenTTL()

	jwt.SetTokenCookie(c, jwt.AccessTokenCookie, newAccessToken, accessTTL)
	jwt.SetTokenCookie(c, jwt.RefreshTokenCookie, newRefreshToken, refreshTTL)

	response.Success(c, gin.H{
		"user": dto.ToUserResponse(user),
	})
}

func (h *authHandler) GetCurrentUser(c *gin.Context) {
	userIDVal, _ := c.Get("user_id")
	userID, ok := userIDVal.(uint)
	if !ok {
		response.Unauthorized(c)
		return
	}

	user, err := h.authService.GetCurrentUser(userID)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, dto.ToUserWithRolesResponse(user))
}

type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password" binding:"required,max=64"`
	NewPassword     string `json:"new_password" binding:"required,min=8,max=64"`
}

func (h *authHandler) ChangePassword(c *gin.Context) {
	userIDVal, _ := c.Get("user_id")
	userID, ok := userIDVal.(uint)
	if !ok {
		response.Unauthorized(c)
		return
	}

	var req ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, err.Error())
		return
	}

	auditCtx := dto.NewAuditContext(c, userID)
	if err := h.authService.ChangePassword(userID, req.CurrentPassword, req.NewPassword, auditCtx); err != nil {
		response.Error(c, err)
		return
	}

	jwt.ClearAllTokenCookies(c)

	response.Success(c, gin.H{
		"message": "password_changed",
	})
}
