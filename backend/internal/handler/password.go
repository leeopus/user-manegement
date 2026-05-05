package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/user-system/backend/internal/service"
	"github.com/user-system/backend/pkg/response"
)

type PasswordHandler interface {
	RequestReset(c *gin.Context)
	ResetPassword(c *gin.Context)
	ValidateToken(c *gin.Context)
}

type passwordHandler struct {
	passwordService service.PasswordResetService
}

func NewPasswordHandler(passwordService service.PasswordResetService) PasswordHandler {
	return &passwordHandler{
		passwordService: passwordService,
	}
}

type RequestResetRequest struct {
	Email string `json:"email" binding:"required,email"`
}

type ResetPasswordRequest struct {
	Token       string `json:"token" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=8,max=64"`
}

type ValidateTokenRequest struct {
	Token string `json:"token" binding:"required"`
}

// RequestReset 请求密码重置
func (h *passwordHandler) RequestReset(c *gin.Context) {
	var req RequestResetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, err.Error())
		return
	}

	// 请求重置（总是返回成功，不透露用户是否存在）
	err := h.passwordService.RequestPasswordReset(req.Email)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, gin.H{
		"message": "如果该邮箱存在，您将收到密码重置邮件",
	})
}

// ResetPassword 重置密码
func (h *passwordHandler) ResetPassword(c *gin.Context) {
	var req ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, err.Error())
		return
	}

	err := h.passwordService.ResetPassword(req.Token, req.NewPassword)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, gin.H{
		"message": "密码重置成功，请使用新密码登录",
	})
}

// ValidateToken 验证重置令牌
func (h *passwordHandler) ValidateToken(c *gin.Context) {
	var req ValidateTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, err.Error())
		return
	}

	valid, err := h.passwordService.ValidateResetToken(req.Token)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, gin.H{
		"valid": valid,
	})
}
