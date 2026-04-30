package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/user-system/backend/internal/service"
	"github.com/user-system/backend/pkg/response"
)

type RoleHandler interface {
	ListRoles(c *gin.Context)
	GetRole(c *gin.Context)
	CreateRole(c *gin.Context)
	UpdateRole(c *gin.Context)
	DeleteRole(c *gin.Context)
}

type roleHandler struct {
	roleService service.RoleService
}

func NewRoleHandler(roleService service.RoleService) RoleHandler {
	return &roleHandler{roleService: roleService}
}

type CreateRoleRequest struct {
	Name        string `json:"name" binding:"required"`
	Code        string `json:"code" binding:"required"`
	Description string `json:"description"`
}

func (h *roleHandler) ListRoles(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))

	roles, total, err := h.roleService.ListRoles(page, pageSize)
	if err != nil {
		response.Error(c, 500, err.Error())
		return
	}

	response.Success(c, gin.H{
		"roles": roles,
		"total": total,
		"page":  page,
		"page_size": pageSize,
	})
}

func (h *roleHandler) GetRole(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.ValidationError(c, "invalid role id")
		return
	}

	role, err := h.roleService.GetRole(uint(id))
	if err != nil {
		response.Error(c, 404, "role not found")
		return
	}

	response.Success(c, role)
}

func (h *roleHandler) CreateRole(c *gin.Context) {
	var req CreateRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, err.Error())
		return
	}

	role, err := h.roleService.CreateRole(req.Name, req.Code, req.Description)
	if err != nil {
		response.Error(c, 400, err.Error())
		return
	}

	response.Success(c, role)
}

func (h *roleHandler) UpdateRole(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.ValidationError(c, "invalid role id")
		return
	}

	var req CreateRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, err.Error())
		return
	}

	role, err := h.roleService.UpdateRole(uint(id), req.Name, req.Code, req.Description)
	if err != nil {
		response.Error(c, 400, err.Error())
		return
	}

	response.Success(c, role)
}

func (h *roleHandler) DeleteRole(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.ValidationError(c, "invalid role id")
		return
	}

	if err := h.roleService.DeleteRole(uint(id)); err != nil {
		response.Error(c, 400, err.Error())
		return
	}

	response.Success(c, gin.H{
		"message": "role deleted successfully",
	})
}
