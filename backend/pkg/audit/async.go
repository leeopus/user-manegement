package audit

import (
	"sync"
	"time"

	"github.com/user-system/backend/internal/repository"
	"go.uber.org/zap"
)

const bufferSize = 2048

// AsyncAuditLogRepository 异步审计日志 repository 包装器
// Create 方法异步写入，Shutdown 时排空队列
type AsyncAuditLogRepository struct {
	inner   repository.AuditLogRepository
	queue   chan *repository.AuditLog
	done    chan struct{}
	mu      sync.Mutex
	stopped bool
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
	r.mu.Lock()
	if r.stopped {
		r.mu.Unlock()
		return r.inner.Create(log)
	}
	select {
	case r.queue <- log:
		r.mu.Unlock()
		return nil
	default:
		r.mu.Unlock()
		zap.L().Warn("Audit log queue full, falling back to sync write")
		return r.inner.Create(log)
	}
}

func (r *AsyncAuditLogRepository) FindByUserID(userID uint, offset, limit int) ([]repository.AuditLog, int64, error) {
	return r.inner.FindByUserID(userID, offset, limit)
}

func (r *AsyncAuditLogRepository) List(offset, limit int) ([]repository.AuditLog, int64, error) {
	return r.inner.List(offset, limit)
}

// Shutdown 优雅关闭：停止接受新日志，排空队列中已有日志，或超时退出
func (r *AsyncAuditLogRepository) Shutdown(timeout time.Duration) {
	r.mu.Lock()
	r.stopped = true
	close(r.queue)
	r.mu.Unlock()

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
