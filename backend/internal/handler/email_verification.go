package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/user-system/backend/internal/dto"
	"github.com/user-system/backend/internal/service"
	"github.com/user-system/backend/pkg/response"
)

type EmailVerificationHandler interface {
	VerifyEmail(c *gin.Context)
	ResendVerification(c *gin.Context)
}

type emailVerificationHandler struct {
	emailVerifyService service.EmailVerificationService
}

func NewEmailVerificationHandler(emailVerifyService service.EmailVerificationService) EmailVerificationHandler {
	return &emailVerificationHandler{emailVerifyService: emailVerifyService}
}

type VerifyEmailRequest struct {
	Token string `json:"token" binding:"required"`
}

func (h *emailVerificationHandler) VerifyEmail(c *gin.Context) {
	var req VerifyEmailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, "token is required")
		return
	}

	user, err := h.emailVerifyService.VerifyEmail(req.Token)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, dto.ToUserResponse(user))
}

func (h *emailVerificationHandler) ResendVerification(c *gin.Context) {
	userID, ok := getCurrentUserID(c)
	if !ok {
		response.Unauthorized(c)
		return
	}

	if err := h.emailVerifyService.ResendVerification(userID); err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, gin.H{"message": "verification email sent"})
}
