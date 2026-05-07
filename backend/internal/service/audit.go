package service

import (
	"encoding/json"

	"github.com/user-system/backend/internal/dto"
	"github.com/user-system/backend/internal/repository"
	"github.com/user-system/backend/pkg/audit"
	"go.uber.org/zap"
)

type AuditLogger struct {
	repo     repository.AuditLogRepository
	syncRepo repository.AuditLogRepository
}

func NewAuditLogger(repo repository.AuditLogRepository) *AuditLogger {
	return &AuditLogger{repo: repo, syncRepo: unwrapAsync(repo)}
}

// unwrapAsync 获取 AsyncAuditLogRepository 内部的同步 repo，用于 LogSync 场景
func unwrapAsync(repo repository.AuditLogRepository) repository.AuditLogRepository {
	if async, ok := repo.(*audit.AsyncAuditLogRepository); ok {
		return async.GetInner()
	}
	return repo
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

// LogSync 安全关键审计日志同步写入（登录、密码修改、权限变更等），确保服务崩溃不丢失
func (l *AuditLogger) LogSync(auditCtx *dto.AuditContext, action, resource string, details map[string]interface{}) error {
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
	if err := l.syncRepo.Create(auditLog); err != nil {
		zap.L().Error("Failed to sync-write critical audit log", zap.String("action", action), zap.Error(err))
		return err
	}
	return nil
}
