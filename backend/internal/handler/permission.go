package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/user-system/backend/internal/service"
	"github.com/user-system/backend/pkg/response"
)

type PermissionHandler interface {
	ListPermissions(c *gin.Context)
	GetPermission(c *gin.Context)
	CreatePermission(c *gin.Context)
	UpdatePermission(c *gin.Context)
	DeletePermission(c *gin.Context)
}

type permissionHandler struct {
	permissionService service.PermissionService
}

func NewPermissionHandler(permissionService service.PermissionService) PermissionHandler {
	return &permissionHandler{permissionService: permissionService}
}

type CreatePermissionRequest struct {
	Name        string `json:"name" binding:"required"`
	Code        string `json:"code" binding:"required"`
	Resource    string `json:"resource" binding:"required"`
	Action      string `json:"action" binding:"required"`
	Description string `json:"description"`
}

func (h *permissionHandler) ListPermissions(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))

	permissions, total, err := h.permissionService.ListPermissions(page, pageSize)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, gin.H{
		"permissions": permissions,
		"total":       total,
		"page":        page,
		"page_size":   pageSize,
	})
}

func (h *permissionHandler) GetPermission(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.ValidationError(c, "invalid permission id")
		return
	}

	permission, err := h.permissionService.GetPermission(uint(id))
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, permission)
}

func (h *permissionHandler) CreatePermission(c *gin.Context) {
	var req CreatePermissionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, err.Error())
		return
	}

	permission, err := h.permissionService.CreatePermission(req.Name, req.Code, req.Resource, req.Action, req.Description)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, permission)
}

func (h *permissionHandler) UpdatePermission(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.ValidationError(c, "invalid permission id")
		return
	}

	var req CreatePermissionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, err.Error())
		return
	}

	permission, err := h.permissionService.UpdatePermission(uint(id), req.Name, req.Code, req.Resource, req.Action, req.Description)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, permission)
}

func (h *permissionHandler) DeletePermission(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.ValidationError(c, "invalid permission id")
		return
	}

	if err := h.permissionService.DeletePermission(uint(id)); err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, gin.H{
		"message": "permission deleted successfully",
	})
}
