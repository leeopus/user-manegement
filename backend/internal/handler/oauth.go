package handler

import (
	"fmt"
	"net/url"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/user-system/backend/internal/config"
	"github.com/user-system/backend/internal/dto"
	"github.com/user-system/backend/internal/service"
	"github.com/user-system/backend/pkg/response"
)

type OAuthHandler interface {
	Authorize(c *gin.Context)
	AuthorizePage(c *gin.Context)
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
	ClientID            string `json:"client_id" binding:"required"`
	RedirectURI         string `json:"redirect_uri" binding:"required"`
	State               string `json:"state" binding:"required"`
	Scope               string `json:"scope"`
	CodeChallenge       string `json:"code_challenge"`
	CodeChallengeMethod string `json:"code_challenge_method"`
}

type TokenRequest struct {
	ClientID     string `json:"client_id" binding:"required"`
	ClientSecret string `json:"client_secret" binding:"required"`
	Code         string `json:"code" binding:"required"`
	RedirectURI  string `json:"redirect_uri" binding:"required"`
	CodeVerifier string `json:"code_verifier"`
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

	code, err := h.oauthService.Authorize(userID, req.ClientID, req.RedirectURI, req.State, req.Scope, req.CodeChallenge, req.CodeChallengeMethod, c.ClientIP(), c.GetHeader("User-Agent"))
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, gin.H{
		"code":  code,
		"state": req.State,
	})
}

// AuthorizePage handles browser-based OAuth2 authorization (GET request).
// If the user is authenticated, generates a code and redirects to callback.
// If not authenticated, redirects to the login page with return URL.
func (h *oauthHandler) AuthorizePage(c *gin.Context) {
	clientID := c.Query("client_id")
	redirectURI := c.Query("redirect_uri")
	state := c.Query("state")
	scope := c.Query("scope")

	if clientID == "" || redirectURI == "" || state == "" {
		response.ValidationError(c, "client_id, redirect_uri, and state are required")
		return
	}

	// Check if user is authenticated
	userIDVal, exists := c.Get("user_id")
	if !exists {
		// Not logged in — redirect to login page with return URL
		cfg := config.Get()
		frontendURL := "http://localhost:3000"
		if cfg != nil && cfg.Frontend.URL != "" {
			frontendURL = cfg.Frontend.URL
		}
		// Build full backend URL so redirect after login hits the backend, not the frontend
		scheme := "http"
		if c.Request.TLS != nil || c.GetHeader("X-Forwarded-Proto") == "https" {
			scheme = "https"
		}
		returnURL := fmt.Sprintf("%s://%s%s", scheme, c.Request.Host, c.Request.URL.String())
		loginURL := fmt.Sprintf("%s/zh/login?redirect=%s", frontendURL, url.QueryEscape(returnURL))
		c.Redirect(302, loginURL)
		return
	}

	userID, ok := userIDVal.(uint)
	if !ok {
		response.Unauthorized(c)
		return
	}

	// Generate authorization code
	code, err := h.oauthService.Authorize(userID, clientID, redirectURI, state, scope, "", "", c.ClientIP(), c.GetHeader("User-Agent"))
	if err != nil {
		response.Error(c, err)
		return
	}

	// Redirect back to client callback with code and state
	callbackURL, _ := url.Parse(redirectURI)
	q := callbackURL.Query()
	q.Set("code", code)
	q.Set("state", state)
	callbackURL.RawQuery = q.Encode()
	c.Redirect(302, callbackURL.String())
}

func (h *oauthHandler) Token(c *gin.Context) {
	var req TokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, err.Error())
		return
	}

	accessToken, _, err := h.oauthService.Token(req.ClientID, req.ClientSecret, req.Code, req.RedirectURI, req.CodeVerifier)
	if err != nil {
		response.Error(c, err)
		return
	}

	cfg := config.Get()
	expiresIn := 900
	if cfg != nil && cfg.Security.AccessTokenMaxTTLMin > 0 {
		expiresIn = cfg.Security.AccessTokenMaxTTLMin * 60
	}

	response.Success(c, gin.H{
		"access_token": accessToken,
		"token_type":   "Bearer",
		"expires_in":   expiresIn,
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
