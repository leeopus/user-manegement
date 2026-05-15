package handler

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/user-system/backend/internal/dto"
	"github.com/user-system/backend/internal/service"
	"github.com/user-system/backend/pkg/response"
)

type ProfileHandler interface {
	GetProfile(c *gin.Context)
	UpdateProfile(c *gin.Context)
	UploadAvatar(c *gin.Context)
	DeleteAccount(c *gin.Context)
}

type profileHandler struct {
	profileService service.ProfileService
	uploadService  service.UploadService
}

func NewProfileHandler(profileService service.ProfileService, uploadService service.UploadService) ProfileHandler {
	return &profileHandler{profileService: profileService, uploadService: uploadService}
}

type UpdateProfileRequest struct {
	Nickname string `json:"nickname" binding:"omitempty,max=50"`
	Bio      string `json:"bio" binding:"omitempty,max=500"`
	Avatar   string `json:"avatar" binding:"omitempty,url,max=255"`
}

func (h *profileHandler) GetProfile(c *gin.Context) {
	userID, ok := getCurrentUserID(c)
	if !ok {
		response.Unauthorized(c)
		return
	}

	user, err := h.profileService.GetProfile(userID)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, dto.ToUserWithRolesResponse(user))
}

func (h *profileHandler) UpdateProfile(c *gin.Context) {
	userID, ok := getCurrentUserID(c)
	if !ok {
		response.Unauthorized(c)
		return
	}

	var req UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, err.Error())
		return
	}

	user, err := h.profileService.UpdateProfile(
		userID,
		req.Nickname,
		req.Bio,
		req.Avatar,
		getAuditContext(c),
	)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, dto.ToUserResponse(user))
}

func (h *profileHandler) UploadAvatar(c *gin.Context) {
	userID, ok := getCurrentUserID(c)
	if !ok {
		response.Unauthorized(c)
		return
	}

	file, header, err := c.Request.FormFile("avatar")
	if err != nil {
		response.ValidationError(c, "avatar file is required")
		return
	}
	defer file.Close()

	avatarURL, err := h.uploadService.UploadAvatar(userID, file, header.Filename, header.Size)
	if err != nil {
		response.Error(c, err)
		return
	}

	user, err := h.profileService.UpdateProfile(userID, "", "", avatarURL, getAuditContext(c))
	if err != nil {
		cleanPath := filepath.Clean("." + avatarURL)
		if !strings.Contains(cleanPath, "..") {
			os.Remove(cleanPath)
		}
		response.Error(c, err)
		return
	}

	response.Success(c, gin.H{"avatar": user.Avatar})
}

type DeleteAccountRequest struct {
	Password string `json:"password" binding:"required,min=8,max=64"`
	Confirm  bool   `json:"confirm" binding:"required,eq=true"`
}

func (h *profileHandler) DeleteAccount(c *gin.Context) {
	userID, ok := getCurrentUserID(c)
	if !ok {
		response.Unauthorized(c)
		return
	}

	var req DeleteAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, err.Error())
		return
	}

	if err := h.profileService.DeleteAccount(userID, req.Password, getAuditContext(c)); err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, gin.H{"message": "account deleted"})
}
