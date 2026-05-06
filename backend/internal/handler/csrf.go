package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/user-system/backend/internal/config"
	"github.com/user-system/backend/internal/dto"
	"github.com/user-system/backend/pkg/csrf"
)

type CSRFHandler interface {
	GetToken(c *gin.Context)
}

type csrfHandler struct {
	csrfManager *csrf.Manager
}

func NewCSRFHandler(redisClient *redis.Client) CSRFHandler {
	return &csrfHandler{
		csrfManager: csrf.NewCSRFManager(redisClient),
	}
}

// GetToken 获取 CSRF token（绑定到当前 session 指纹）
func (h *csrfHandler) GetToken(c *gin.Context) {
	sessionID := dto.SessionFingerprint(c)
	token, err := h.csrfManager.GenerateToken(sessionID)
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

	cfg := config.Get()
	ttlMin := 30
	if cfg != nil && cfg.Security.CSRFTokenTTLMin > 0 {
		ttlMin = cfg.Security.CSRFTokenTTLMin
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"csrf_token": token,
			"expires_at": time.Now().Add(time.Duration(ttlMin) * time.Minute).Format(time.RFC3339),
		},
	})
}
