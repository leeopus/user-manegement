package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/user-system/backend/internal/dto"
	"github.com/user-system/backend/internal/repository"
	"github.com/user-system/backend/pkg/response"
)

type AuditHandler interface {
	ListAuditLogs(c *gin.Context)
}

type auditHandler struct {
	auditRepo repository.AuditLogRepository
}

func NewAuditHandler(auditRepo repository.AuditLogRepository) AuditHandler {
	return &auditHandler{auditRepo: auditRepo}
}

func (h *auditHandler) ListAuditLogs(c *gin.Context) {
	page, pageSize, offset := response.ParsePagination(c)

	filters := repository.AuditLogFilters{}
	if userIDStr := c.Query("user_id"); userIDStr != "" {
		if id, err := strconv.ParseUint(userIDStr, 10, 32); err == nil {
			filters.UserID = uint(id)
		}
	}
	if action := c.Query("action"); action != "" {
		filters.Action = action
	}
	if resource := c.Query("resource"); resource != "" {
		filters.Resource = resource
	}
	if search := c.Query("search"); search != "" {
		filters.Search = search
	}

	logs, total, err := h.auditRepo.ListFiltered(offset, pageSize, filters)
	if err != nil {
		response.Error(c, err)
		return
	}

	logResponses := make([]dto.AuditLogResponse, len(logs))
	for i := range logs {
		logResponses[i] = dto.ToAuditLogResponse(&logs[i])
	}

	response.Success(c, gin.H{
		"logs":      logResponses,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}
