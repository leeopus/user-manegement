package handler

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/user-system/backend/internal/service"
	"github.com/user-system/backend/pkg/jwt"
	"github.com/user-system/backend/pkg/response"
	apperrors "github.com/user-system/backend/pkg/errors"
)

type AuthHandler interface {
	Register(c *gin.Context)
	Login(c *gin.Context)
	Logout(c *gin.Context)
	RefreshToken(c *gin.Context)
	GetCurrentUser(c *gin.Context)
}

type authHandler struct {
	authService service.AuthService
}

func NewAuthHandler(authService service.AuthService) AuthHandler {
	return &authHandler{authService: authService}
}

type RegisterRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8,max=64"`
}

type LoginRequest struct {
	Email      string `json:"email" binding:"required,email"`
	Password   string `json:"password" binding:"required"`
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

	clientIP := c.ClientIP()
	userAgent := c.GetHeader("User-Agent")

	user, err := h.authService.Register(req.Email, req.Password, clientIP, userAgent)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Created(c, gin.H{
		"user": user,
	})
}

func (h *authHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, err.Error())
		return
	}

	user, accessToken, refreshToken, err := h.authService.Login(req.Email, req.Password)
	if err != nil {
		response.Error(c, err)
		return
	}

	// 根据"记住我"设置不同的cookie过期时间
	if req.RememberMe {
		jwt.SetTokenCookie(c, jwt.AccessTokenCookie, accessToken, 7*24*time.Hour)
		jwt.SetTokenCookie(c, jwt.RefreshTokenCookie, refreshToken, 30*24*time.Hour)
	} else {
		jwt.SetTokenCookie(c, jwt.AccessTokenCookie, accessToken, 15*time.Minute)
		jwt.SetTokenCookie(c, jwt.RefreshTokenCookie, refreshToken, 7*24*time.Hour)
	}

	response.Success(c, gin.H{
		"user": user,
	})
}

func (h *authHandler) Logout(c *gin.Context) {
	userID, _ := c.Get("user_id")

	// 从 cookie 获取 refresh token 用于撤销
	refreshToken, _ := jwt.GetTokenCookie(c, jwt.RefreshTokenCookie)

	_ = h.authService.Logout(userID.(uint), refreshToken)

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

	user, newToken, err := h.authService.RefreshToken(refreshToken)
	if err != nil {
		response.Error(c, err)
		return
	}

	jwt.SetTokenCookie(c, jwt.AccessTokenCookie, newToken, 15*time.Minute)

	response.Success(c, gin.H{
		"user": user,
	})
}

func (h *authHandler) GetCurrentUser(c *gin.Context) {
	userID, _ := c.Get("user_id")

	user, err := h.authService.GetCurrentUser(userID.(uint))
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, user)
}
