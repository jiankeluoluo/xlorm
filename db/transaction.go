package db

import (
	"database/sql"
	"fmt"
	"time"
)

// Transaction 事务管理器结构体
type Transaction struct {
	*sql.Tx
	db      *XgDB
	traceID string // 事务跟踪ID
}

// Commit 提交事务
func (tx *Transaction) Commit() error {
	if tx == nil || tx.Tx == nil {
		return fmt.Errorf("事务为空, trace_id:%s", tx.traceID)
	}

	startTime := time.Now()
	if tx.db.IsDebug() {
		tx.db.logger.Info("提交事务成功",
			"trace_id", tx.traceID,
			"duration", time.Since(startTime).Seconds(),
		)
	}
	if err := tx.Tx.Commit(); err != nil {
		tx.db.asyncMetrics.RecordError()
		return fmt.Errorf("提交事务失败: %v, trace_id:%s", err, tx.traceID)
	}

	tx.db.asyncMetrics.RecordQueryDuration("commit_transaction", time.Since(startTime))
	return nil
}

// Rollback 回滚事务
func (tx *Transaction) Rollback() error {
	if tx == nil || tx.Tx == nil {
		return fmt.Errorf("事务为空, trace_id:%s", tx.traceID)
	}

	startTime := time.Now()
	if tx.db.IsDebug() {
		tx.db.logger.Debug("回滚事务", "trace_id", tx.traceID)
	}
	if err := tx.Tx.Rollback(); err != nil {
		tx.db.asyncMetrics.RecordError()
		return fmt.Errorf("回滚事务失败: %v, trace_id:%s", err, tx.traceID)
	}

	if tx.db.IsDebug() {
		tx.db.logger.Info("回滚事务完成",
			"trace_id", tx.traceID,
			"duration", time.Since(startTime).Seconds(),
		)
	}
	tx.db.asyncMetrics.RecordQueryDuration("rollback_transaction", time.Since(startTime))
	return nil
}

// DB 获取数据库实例
func (tx *Transaction) DB() *XgDB {
	return tx.db
}
