package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/user-system/backend/pkg/csrf"
)

type CSRFHandler interface {
	GetToken(c *gin.Context)
}

type csrfHandler struct{}

func NewCSRFHandler() CSRFHandler {
	return &csrfHandler{}
}

// GetToken 获取 CSRF token
func (h *csrfHandler) GetToken(c *gin.Context) {
	token, err := csrf.DefaultManager.GenerateToken()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "CSRF_TOKEN_GENERATION_FAILED_500",
				"message": "CSRF_TOKEN_GENERATION_FAILED",
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"csrf_token": token,
			"expires_at": "1h",
		},
	})
}
