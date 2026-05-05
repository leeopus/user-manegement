package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/user-system/backend/internal/service"
	apperrors "github.com/user-system/backend/pkg/errors"
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

func (h *userHandler) ListUsers(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))

	// 分页安全限制
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}
	if pageSize > 100 {
		pageSize = 100
	}

	users, total, err := h.userService.ListUsers(page, pageSize)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, gin.H{
		"users":     users,
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

	response.Success(c, user)
}

func (h *userHandler) CreateUser(c *gin.Context) {
	var req CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, err.Error())
		return
	}

	user, err := h.userService.CreateUser(req.Username, req.Email, req.Password)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Created(c, user)
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

	user, err := h.userService.UpdateUser(uint(id), req.Username, req.Email)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, user)
}

func (h *userHandler) DeleteUser(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.ValidationError(c, "invalid user id")
		return
	}

	if err := h.userService.DeleteUser(uint(id)); err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, gin.H{
		"message": "user deleted successfully",
	})
}

// requiresAdmin 是一个辅助函数，检查用户是否是管理员
func requiresAdmin(c *gin.Context) bool {
	roles, exists := c.Get("user_roles")
	if !exists {
		return false
	}

	userRoles, ok := roles.([]interface{})
	if !ok {
		return false
	}

	for _, role := range userRoles {
		if r, ok := role.(map[string]interface{}); ok {
			if code, ok := r["code"].(string); ok && code == "admin" {
				return true
			}
		}
	}

	return false
}

// ensureAdminOrSelf 确保用户是管理员或正在操作自己的资源
func ensureAdminOrSelf(c *gin.Context, targetUserID uint) bool {
	if requiresAdmin(c) {
		return true
	}

	userID, _ := c.Get("user_id")
	return userID.(uint) == targetUserID
}

// ErrorResponse 返回一个标准错误响应
func ErrorResponse(c *gin.Context, err error) {
	appErr, ok := apperrors.IsAppError(err)
	if ok {
		response.Error(c, appErr)
	} else {
		response.Error(c, apperrors.ErrInternalServer)
	}
}
