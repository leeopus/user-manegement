package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/user-system/backend/internal/dto"
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
	State       string `json:"state" binding:"required"`
	Scope       string `json:"scope"`
}

type TokenRequest struct {
	ClientID     string `json:"client_id" binding:"required"`
	ClientSecret string `json:"client_secret" binding:"required"`
	Code         string `json:"code" binding:"required"`
	RedirectURI  string `json:"redirect_uri" binding:"required"`
}

type CreateApplicationRequest struct {
	Name         string `json:"name" binding:"required"`
	RedirectURIs string `json:"redirect_uris" binding:"required"`
	Scopes       string `json:"scopes"`
}

type UpdateApplicationRequest struct {
	Name         string `json:"name" binding:"required"`
	RedirectURIs string `json:"redirect_uris" binding:"required"`
}

func (h *oauthHandler) Authorize(c *gin.Context) {
	var req AuthorizeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, err.Error())
		return
	}

	userIDVal, exists := c.Get("user_id")
	if !exists {
		response.Unauthorized(c)
		return
	}
	userID, ok := userIDVal.(uint)
	if !ok {
		response.Unauthorized(c)
		return
	}

	code, err := h.oauthService.Authorize(userID, req.ClientID, req.RedirectURI, req.State, req.Scope, c.ClientIP(), c.GetHeader("User-Agent"))
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, gin.H{
		"code":  code,
		"state": req.State,
	})
}

func (h *oauthHandler) Token(c *gin.Context) {
	var req TokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, err.Error())
		return
	}

	accessToken, _, err := h.oauthService.Token(req.ClientID, req.ClientSecret, req.Code, req.RedirectURI)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, gin.H{
		"access_token": accessToken,
		"token_type":   "Bearer",
		"expires_in":   3600,
		// refresh_token 仅在需要时返回，此处省略以减少暴露面
	})
}

func (h *oauthHandler) Userinfo(c *gin.Context) {
	// 使用中间件已验证的 user_id，避免重复解析 token
	userIDVal, exists := c.Get("user_id")
	if !exists {
		response.Unauthorized(c)
		return
	}
	userID, ok := userIDVal.(uint)
	if !ok {
		response.Unauthorized(c)
		return
	}

	user, err := h.oauthService.UserinfoByID(userID)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, dto.ToUserResponse(user))
}

func (h *oauthHandler) ListApplications(c *gin.Context) {
	page, pageSize, offset := response.ParsePagination(c)

	applications, total, err := h.oauthService.ListApplications(offset, pageSize)
	if err != nil {
		response.Error(c, err)
		return
	}

	appResponses := make([]dto.OAuthApplicationResponse, len(applications))
	for i := range applications {
		appResponses[i] = dto.ToOAuthApplicationResponse(&applications[i])
	}

	response.Success(c, gin.H{
		"applications": appResponses,
		"total":        total,
		"page":         page,
		"page_size":    pageSize,
	})
}

func (h *oauthHandler) GetApplication(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.ValidationError(c, "invalid application id")
		return
	}

	app, err := h.oauthService.GetApplication(uint(id))
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, dto.ToOAuthApplicationResponse(app))
}

func (h *oauthHandler) CreateApplication(c *gin.Context) {
	var req CreateApplicationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, err.Error())
		return
	}

	app, rawSecret, err := h.oauthService.CreateApplication(req.Name, req.RedirectURIs, req.Scopes, getAuditContext(c))
	if err != nil {
		response.Error(c, err)
		return
	}

	// client_secret 仅在创建时返回一次
	resp := dto.ToOAuthApplicationResponse(app)
	resp.ClientSecret = rawSecret

	response.Created(c, resp)
}

func (h *oauthHandler) UpdateApplication(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.ValidationError(c, "invalid application id")
		return
	}

	var req UpdateApplicationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, err.Error())
		return
	}

	app, err := h.oauthService.UpdateApplication(uint(id), req.Name, req.RedirectURIs, getAuditContext(c))
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, dto.ToOAuthApplicationResponse(app))
}

func (h *oauthHandler) DeleteApplication(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.ValidationError(c, "invalid application id")
		return
	}

	if err := h.oauthService.DeleteApplication(uint(id), getAuditContext(c)); err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, gin.H{
		"message": "application deleted",
	})
}
