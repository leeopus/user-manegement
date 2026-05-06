package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/user-system/backend/internal/dto"
	"github.com/user-system/backend/internal/service"
	"github.com/user-system/backend/pkg/response"
)

type RoleHandler interface {
	ListRoles(c *gin.Context)
	GetRole(c *gin.Context)
	CreateRole(c *gin.Context)
	UpdateRole(c *gin.Context)
	DeleteRole(c *gin.Context)
	AssignPermission(c *gin.Context)
	RemovePermission(c *gin.Context)
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

type AssignPermissionRequest struct {
	PermissionID uint `json:"permission_id" binding:"required"`
}

func (h *roleHandler) ListRoles(c *gin.Context) {
	page, pageSize, offset := response.ParsePagination(c)

	roles, total, err := h.roleService.ListRoles(offset, pageSize)
	if err != nil {
		response.Error(c, err)
		return
	}

	roleResponses := make([]dto.RoleResponse, len(roles))
	for i := range roles {
		roleResponses[i] = dto.ToRoleResponse(&roles[i])
	}

	response.Success(c, gin.H{
		"roles":     roleResponses,
		"total":     total,
		"page":      page,
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
		response.Error(c, err)
		return
	}

	response.Success(c, dto.ToRoleResponse(role))
}

func (h *roleHandler) CreateRole(c *gin.Context) {
	var req CreateRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, err.Error())
		return
	}

	role, err := h.roleService.CreateRole(req.Name, req.Code, req.Description, getAuditContext(c))
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Created(c, dto.ToRoleResponse(role))
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

	role, err := h.roleService.UpdateRole(uint(id), req.Name, req.Code, req.Description, getAuditContext(c))
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, dto.ToRoleResponse(role))
}

func (h *roleHandler) DeleteRole(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.ValidationError(c, "invalid role id")
		return
	}

	if err := h.roleService.DeleteRole(uint(id), getAuditContext(c)); err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, gin.H{
		"message": "role deleted successfully",
	})
}

func (h *roleHandler) AssignPermission(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.ValidationError(c, "invalid role id")
		return
	}

	var req AssignPermissionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, err.Error())
		return
	}

	if err := h.roleService.AssignRolePermission(uint(id), req.PermissionID, getAuditContext(c)); err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, gin.H{
		"message": "permission assigned successfully",
	})
}

func (h *roleHandler) RemovePermission(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.ValidationError(c, "invalid role id")
		return
	}

	permissionID, err := strconv.ParseUint(c.Param("permissionId"), 10, 32)
	if err != nil {
		response.ValidationError(c, "invalid permission id")
		return
	}

	if err := h.roleService.RemoveRolePermission(uint(id), uint(permissionID), getAuditContext(c)); err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, gin.H{
		"message": "permission removed successfully",
	})
}
