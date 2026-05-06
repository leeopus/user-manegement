package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/user-system/backend/internal/dto"
	"github.com/user-system/backend/internal/repository"
	"github.com/user-system/backend/internal/service"
	"github.com/user-system/backend/pkg/response"
)

type UserHandler interface {
	ListUsers(c *gin.Context)
	GetUser(c *gin.Context)
	CreateUser(c *gin.Context)
	UpdateUser(c *gin.Context)
	UpdateUserStatus(c *gin.Context)
	DeleteUser(c *gin.Context)
	HardDeleteUser(c *gin.Context)
	AssignRole(c *gin.Context)
	RemoveRole(c *gin.Context)
}

type userHandler struct {
	userService service.UserService
}

func NewUserHandler(userService service.UserService) UserHandler {
	return &userHandler{userService: userService}
}

type CreateUserRequest struct {
	Username string `json:"username" binding:"required"`
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type UpdateUserRequest struct {
	Username string `json:"username" binding:"required"`
	Email    string `json:"email" binding:"required"`
}

type UpdateStatusRequest struct {
	Status string `json:"status" binding:"required,oneof=active disabled"`
}

type AssignRoleRequest struct {
	RoleID uint `json:"role_id" binding:"required"`
}

func getCurrentUserID(c *gin.Context) (uint, bool) {
	userIDVal, exists := c.Get("user_id")
	if !exists {
		return 0, false
	}
	userID, ok := userIDVal.(uint)
	return userID, ok
}

func getAuditContext(c *gin.Context) dto.AuditContext {
	userIDVal, _ := c.Get("user_id")
	userID, _ := userIDVal.(uint)
	return dto.NewAuditContext(c, userID)
}

func (h *userHandler) ListUsers(c *gin.Context) {
	page, pageSize, offset := response.ParsePagination(c)

	filters := repository.UserFilters{
		Status: c.Query("status"),
		Search: c.Query("search"),
	}
	if roleIDStr := c.Query("role_id"); roleIDStr != "" {
		if id, err := strconv.ParseUint(roleIDStr, 10, 32); err == nil {
			filters.RoleID = uint(id)
		}
	}

	users, total, err := h.userService.ListUsers(offset, pageSize, filters)
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

	currentUserID, ok := getCurrentUserID(c)
	if !ok {
		response.Unauthorized(c)
		return
	}

	user, err := h.userService.GetUser(uint(id), currentUserID)
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

	currentUserID, ok := getCurrentUserID(c)
	if !ok {
		response.Unauthorized(c)
		return
	}

	user, err := h.userService.UpdateUser(uint(id), req.Username, req.Email, currentUserID, getAuditContext(c))
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, dto.ToUserResponse(user))
}

func (h *userHandler) UpdateUserStatus(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.ValidationError(c, "invalid user id")
		return
	}

	var req UpdateStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, err.Error())
		return
	}

	currentUserID, ok := getCurrentUserID(c)
	if !ok {
		response.Unauthorized(c)
		return
	}

	if err := h.userService.UpdateUserStatus(uint(id), currentUserID, req.Status, getAuditContext(c)); err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, gin.H{
		"message": "user status updated",
	})
}

func (h *userHandler) DeleteUser(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.ValidationError(c, "invalid user id")
		return
	}

	currentUserID, ok := getCurrentUserID(c)
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

func (h *userHandler) HardDeleteUser(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.ValidationError(c, "invalid user id")
		return
	}

	currentUserID, ok := getCurrentUserID(c)
	if !ok {
		response.Unauthorized(c)
		return
	}

	if err := h.userService.HardDeleteUser(uint(id), currentUserID, getAuditContext(c)); err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, gin.H{
		"message": "user permanently deleted",
	})
}

func (h *userHandler) AssignRole(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.ValidationError(c, "invalid user id")
		return
	}

	var req AssignRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, err.Error())
		return
	}

	if err := h.userService.AssignRole(uint(id), req.RoleID, getAuditContext(c)); err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, gin.H{
		"message": "role assigned successfully",
	})
}

func (h *userHandler) RemoveRole(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.ValidationError(c, "invalid user id")
		return
	}

	roleID, err := strconv.ParseUint(c.Param("roleId"), 10, 32)
	if err != nil {
		response.ValidationError(c, "invalid role id")
		return
	}

	if err := h.userService.RemoveRole(uint(id), uint(roleID), getAuditContext(c)); err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, gin.H{
		"message": "role removed successfully",
	})
}
