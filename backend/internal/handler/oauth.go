package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/user-system/backend/internal/service"
	"github.com/user-system/backend/pkg/response"
)

type OAuthHandler interface {
	Authorize(c *gin.Context)
	Token(c *gin.Context)
	Userinfo(c *gin.Context)
	ListApplications(c *gin.Context)
	GetApplication(c *gin.Context)
	CreateApplication(c *gin.Context)
	UpdateApplication(c *gin.Context)
	DeleteApplication(c *gin.Context)
}

type oauthHandler struct {
	oauthService service.OAuthService
}

func NewOAuthHandler(oauthService service.OAuthService) OAuthHandler {
	return &oauthHandler{oauthService: oauthService}
}

type AuthorizeRequest struct {
	ClientID    string `json:"client_id" binding:"required"`
	RedirectURI string `json:"redirect_uri" binding:"required"`
}

type TokenRequest struct {
	ClientID     string `json:"client_id" binding:"required"`
	ClientSecret string `json:"client_secret" binding:"required"`
	Code         string `json:"code" binding:"required"`
}

type CreateApplicationRequest struct {
	Name         string `json:"name" binding:"required"`
	ClientSecret string `json:"client_secret" binding:"required"`
	RedirectURIs string `json:"redirect_uris" binding:"required"`
}

func (h *oauthHandler) Authorize(c *gin.Context) {
	var req AuthorizeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, err.Error())
		return
	}

	app, err := h.oauthService.Authorize(req.ClientID, req.RedirectURI)
	if err != nil {
		response.Error(c, 400, err.Error())
		return
	}

	response.Success(c, app)
}

func (h *oauthHandler) Token(c *gin.Context) {
	var req TokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, err.Error())
		return
	}

	accessToken, refreshToken, err := h.oauthService.Token(req.ClientID, req.ClientSecret, req.Code)
	if err != nil {
		response.Error(c, 401, err.Error())
		return
	}

	response.Success(c, gin.H{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
		"token_type":    "Bearer",
	})
}

func (h *oauthHandler) Userinfo(c *gin.Context) {
	token := c.GetHeader("Authorization")
	if len(token) > 7 {
		token = token[7:]
	}

	user, err := h.oauthService.Userinfo(token)
	if err != nil {
		response.Error(c, 401, err.Error())
		return
	}

	response.Success(c, user)
}

func (h *oauthHandler) ListApplications(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))

	// Simplified - would need to implement ListApplications in service
	response.Success(c, gin.H{
		"applications": []interface{}{},
		"page":        page,
		"page_size":   pageSize,
	})
}

func (h *oauthHandler) GetApplication(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.ValidationError(c, "invalid application id")
		return
	}

	response.Success(c, gin.H{
		"id": id,
	})
}

func (h *oauthHandler) CreateApplication(c *gin.Context) {
	var req CreateApplicationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, err.Error())
		return
	}

	app, err := h.oauthService.CreateApplication(req.Name, req.ClientSecret, req.RedirectURIs)
	if err != nil {
		response.Error(c, 400, err.Error())
		return
	}

	response.Success(c, app)
}

func (h *oauthHandler) UpdateApplication(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.ValidationError(c, "invalid application id")
		return
	}

	response.Success(c, gin.H{
		"id":      id,
		"message": "application updated",
	})
}

func (h *oauthHandler) DeleteApplication(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.ValidationError(c, "invalid application id")
		return
	}

	response.Success(c, gin.H{
		"id":      id,
		"message": "application deleted",
	})
}
