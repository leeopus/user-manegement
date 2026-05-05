package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/user-system/backend/internal/dto"
	"github.com/user-system/backend/internal/service"
	"github.com/user-system/backend/pkg/response"
)

type UserHandler interface {
	ListUsers(c *gin.Context)
	GetUser(c *gin.Context)
	CreateUser(c *gin.Context)
	UpdateUser(c *gin.Context)
	DeleteUser(c *gin.Context)
}

type userHandler struct {
	userService service.UserService
}

func NewUserHandler(userService service.UserService) UserHandler {
	return &userHandler{userService: userService}
}

type CreateUserRequest struct {
	Username string `json:"username" binding:"required,min=3,max=32"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8,max=64"`
}

type UpdateUserRequest struct {
	Username string `json:"username" binding:"required,min=3,max=32"`
	Email    string `json:"email" binding:"required,email"`
}

func getAuditContext(c *gin.Context) dto.AuditContext {
	userIDVal, _ := c.Get("user_id")
	userID, _ := userIDVal.(uint)
	return dto.NewAuditContext(c, userID)
}

func (h *userHandler) ListUsers(c *gin.Context) {
	page, pageSize, offset := response.ParsePagination(c)

	users, total, err := h.userService.ListUsers(offset, pageSize)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, gin.H{
		"users":     dto.ToUserResponseList(users),
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

func (h *userHandler) GetUser(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.ValidationError(c, "invalid user id")
		return
	}

	user, err := h.userService.GetUser(uint(id))
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, dto.ToUserResponse(user))
}

func (h *userHandler) CreateUser(c *gin.Context) {
	var req CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, err.Error())
		return
	}

	user, err := h.userService.CreateUser(req.Username, req.Email, req.Password, getAuditContext(c))
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Created(c, dto.ToUserResponse(user))
}

func (h *userHandler) UpdateUser(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.ValidationError(c, "invalid user id")
		return
	}

	var req UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, err.Error())
		return
	}

	user, err := h.userService.UpdateUser(uint(id), req.Username, req.Email, getAuditContext(c))
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, dto.ToUserResponse(user))
}

func (h *userHandler) DeleteUser(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.ValidationError(c, "invalid user id")
		return
	}

	currentUserIDVal, _ := c.Get("user_id")
	currentUserID, ok := currentUserIDVal.(uint)
	if !ok {
		response.Unauthorized(c)
		return
	}

	if err := h.userService.DeleteUser(uint(id), currentUserID, getAuditContext(c)); err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, gin.H{
		"message": "user deleted successfully",
	})
}
