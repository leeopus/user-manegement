package repository

import (
	"gorm.io/gorm"
)

// BaseRepository 基础仓储，提供通用的事务处理方法
type BaseRepository struct {
	db *gorm.DB
}

// WithTransaction 执行事务操作
func (r *BaseRepository) WithTransaction(fn func(tx *gorm.DB) error) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		return fn(tx)
	})
}

// GetDB 获取数据库连接
func (r *BaseRepository) GetDB() *gorm.DB {
	return r.db
}

// GetTx 开始一个新事务
func (r *BaseRepository) GetTx() *gorm.DB {
	return r.db.Begin()
}
