package audit

import (
	"sync/atomic"
	"time"

	"github.com/user-system/backend/internal/repository"
	"go.uber.org/zap"
)

const bufferSize = 2048

// AsyncAuditLogRepository 异步审计日志 repository 包装器
// Create 方法异步写入，Shutdown 时排空队列
type AsyncAuditLogRepository struct {
	inner      repository.AuditLogRepository
	queue      chan *repository.AuditLog
	done       chan struct{}
	stopped    atomic.Bool
	droppedCnt atomic.Int64
}

// NewAsyncAuditLogRepository 创建异步审计日志 repository
func NewAsyncAuditLogRepository(inner repository.AuditLogRepository) *AsyncAuditLogRepository {
	r := &AsyncAuditLogRepository{
		inner: inner,
		queue: make(chan *repository.AuditLog, bufferSize),
		done:  make(chan struct{}),
	}
	go r.process()
	return r
}

func (r *AsyncAuditLogRepository) Create(log *repository.AuditLog) error {
	if r.stopped.Load() {
		return r.inner.Create(log)
	}
	select {
	case r.queue <- log:
		return nil
	default:
		// 队列满时回退到同步写入，安全事件不可丢弃
		zap.L().Warn("Audit log queue full, falling back to synchronous write")
		return r.inner.Create(log)
	}
}

// GetInner 返回内部的同步 repository，用于安全关键审计日志的同步写入
func (r *AsyncAuditLogRepository) GetInner() repository.AuditLogRepository {
	return r.inner
}

func (r *AsyncAuditLogRepository) FindByUserID(userID uint, offset, limit int) ([]repository.AuditLog, int64, error) {
	return r.inner.FindByUserID(userID, offset, limit)
}

func (r *AsyncAuditLogRepository) List(offset, limit int) ([]repository.AuditLog, int64, error) {
	return r.inner.List(offset, limit)
}

func (r *AsyncAuditLogRepository) ListFiltered(offset, limit int, filters repository.AuditLogFilters) ([]repository.AuditLog, int64, error) {
	return r.inner.ListFiltered(offset, limit, filters)
}

func (r *AsyncAuditLogRepository) CleanupOlderThan(retentionDays int) (int64, error) {
	return r.inner.CleanupOlderThan(retentionDays)
}

// Shutdown 优雅关闭：停止接受新日志，排空队列中已有日志，或超时退出
func (r *AsyncAuditLogRepository) Shutdown(timeout time.Duration) {
	r.stopped.Store(true)
	close(r.queue)

	if dropped := r.droppedCnt.Load(); dropped > 0 {
		zap.L().Warn("Audit logs were dropped during this session",
			zap.Int64("total_dropped", dropped),
		)
	}

	select {
	case <-r.done:
		zap.L().Info("Audit log queue drained successfully")
	case <-time.After(timeout):
		zap.L().Warn("Audit log shutdown timed out, some logs may be lost")
	}
}

func (r *AsyncAuditLogRepository) process() {
	defer close(r.done)
	for log := range r.queue {
		if err := r.inner.Create(log); err != nil {
			zap.L().Warn("Failed to create audit log", zap.Error(err))
		}
	}
}
