package service

import (
	"encoding/json"

	"github.com/user-system/backend/internal/dto"
	"github.com/user-system/backend/internal/repository"
	"go.uber.org/zap"
)

type AuditLogger struct {
	repo repository.AuditLogRepository
}

func NewAuditLogger(repo repository.AuditLogRepository) *AuditLogger {
	return &AuditLogger{repo: repo}
}

func (l *AuditLogger) Log(auditCtx *dto.AuditContext, action, resource string, details map[string]interface{}) {
	detailsJSON, _ := json.Marshal(details)
	auditLog := &repository.AuditLog{
		UserID:    auditCtx.UserID,
		Action:    action,
		Resource:  resource,
		Details:   string(detailsJSON),
		IPAddress: auditCtx.IPAddress,
		UserAgent: auditCtx.UserAgent,
		RequestID: auditCtx.RequestID,
	}
	if err := l.repo.Create(auditLog); err != nil {
		zap.L().Warn("Failed to create audit log", zap.String("request_id", auditCtx.RequestID), zap.Error(err))
	}
}
