package db

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"
)

const (
	defaultBatchSize = 1000
)

// BatchInsert 批量插入数据，使用事务确保原子性和性能
// data 批量插入的数据
// batchSize 单词批量插入的数据量，默认：1000
// totalAffecteds 返回影响的行数
// err 返回错误信息
func (t *Table) BatchInsert(data []map[string]interface{}, batchSize int) (totalAffecteds int64, err error) {
	if batchSize == 0 {
		batchSize = defaultBatchSize
	}
	dataLen := len(data)
	// 检查数据是否为空
	if dataLen == 0 {
		return 0, nil
	}

	// 记录开始时间
	startTime := time.Now()

	// 提取字段和值
	checkFields, err := extractBatchFields(data)
	if err != nil {
		return 0, err
	}
	if t.db.IsDebug() {
		t.db.logger.Debug("批量插入开始",
			"table", t.tableName,
			"count", dataLen,
			"success", 0,
		)
	}
	checkFieldsLen := len(checkFields)
	// 分批处理大量数据
	var totalAffectedRows int64
	var fields []string

	for i := 0; i < dataLen; i += batchSize {
		end := i + batchSize
		if end > dataLen {
			end = dataLen
		}
		batchData := data[i:end]

		// 提取字段和值
		fields, err = extractBatchFields(batchData)
		if err != nil {
			if t.db.IsDebug() {
				t.db.logger.Error("批量插入失败",
					"error", err,
					"batch_start", i,
					"batch_end", end)
			}
			return totalAffectedRows, err
		}
		if len(fields) != checkFieldsLen {
			err = errors.New("字段数量不匹配")
			if t.db.IsDebug() {
				t.db.logger.Error("批量插入失败",
					"error", err,
					"batch_start", i,
					"batch_end", end)
			}
			return totalAffectedRows, err
		}

		// 构建批量插入SQL
		placeholderStr := getCachedPlaceholder(len(fields), t.db.placeholderCache)
		// 安全地构建 placeholders
		placeholders := make([]string, len(batchData))
		for j := range placeholders {
			placeholders[j] = placeholderStr
		}
		query := fmt.Sprintf(
			"INSERT INTO %s (%s) VALUES %s",
			t.tableName,
			strings.Join(fields, ", "),
			strings.Join(placeholders, ","),
		)

		// 准备批量插入的参数
		var args []interface{}
		for _, item := range batchData {
			for _, field := range fields {
				// 移除 ` 符号
				cleanField := strings.Trim(field, "`")
				args = append(args, item[cleanField])
			}
		}
		// 使用事务执行批量插入
		var batchAffectedRows int64
		err := t.db.ExecTx(func(tx *Transaction) error {
			// 使用事务执行插入
			result, err := tx.Exec(query, args...)
			if err != nil {
				t.db.logger.Error("批量插入失败(批次 %d-%d)SQL:%s: %v", slog.Int("batch_start", i), slog.Int("batch_end", end), query, err)
				return fmt.Errorf("批量插入失败(批次 %d-%d): %v", i, end, err)
			}

			// 获取影响的行数
			rowsAffected, err := result.RowsAffected()
			if err != nil {
				t.db.logger.Error("批量插入失败(批次 %d-%d)SQL:%s: %v", slog.Int("batch_start", i), slog.Int("batch_end", end), query, err)
				return fmt.Errorf("获取影响行数失败(批次 %d-%d): %v", i, end, err)
			}
			batchAffectedRows = rowsAffected

			return nil
		})

		// 记录性能指标
		if err == nil {
			totalAffectedRows += batchAffectedRows
			t.db.asyncMetrics.RecordAffectedRows(batchAffectedRows)
		} else {
			t.db.asyncMetrics.RecordError()
			if t.db.IsDebug() {
				t.db.logger.Error("批量插入失败",
					"table", t.tableName,
					"batch_start", i,
					"batch_end", end,
					"error", err)
			}
			return totalAffectedRows, err
		}
	}

	// 记录总体性能指标
	duration := time.Since(startTime)
	t.db.asyncMetrics.RecordQueryDuration("batch_insert", duration)

	if t.db.IsDebug() {
		t.db.logger.Debug("批量插入结束",
			"table", t.tableName,
			"count", dataLen,
			"affected", totalAffectedRows,
			"duration", duration.Seconds(),
		)
	}
	return totalAffectedRows, nil
}

// BatchUpdate 批量更新数据
// 返回更新的行数和错误
func (t *Table) BatchUpdate(records []map[string]interface{}, keyField string, batchSize int) (totalAffecteds int64, err error) {
	if batchSize == 0 {
		batchSize = defaultBatchSize
	}
	recordsLen := len(records)
	if recordsLen == 0 {
		return 0, nil
	}
	if keyField == "" {
		return 0, errors.New("必须指定主键字段")
	}

	startTime := time.Now()
	if t.db.IsDebug() {
		t.db.logger.Debug("开始批量更新",
			"table", t.tableName,
			"count", recordsLen,
		)
	}
	// 开启事务
	tx, err := t.db.Begin()
	if err != nil {
		return 0, fmt.Errorf("开启事务失败: %v", err)
	}
	defer tx.Rollback()

	var totalAffected int64
	for i := 0; i < recordsLen; i += batchSize {
		end := i + batchSize
		if end > recordsLen {
			end = recordsLen
		}

		batch := records[i:end]
		affected, err := t.updateBatch(tx, batch, keyField)
		if err != nil {
			return totalAffected, err
		}
		totalAffected += affected
	}

	// 提交事务
	if err := tx.Commit(); err != nil {
		return totalAffected, fmt.Errorf("提交事务失败: %v", err)
	}

	duration := time.Since(startTime)
	// 记录性能指标
	t.db.asyncMetrics.RecordQueryDuration("batch_update", duration)
	t.db.asyncMetrics.RecordAffectedRows(totalAffected)

	if t.db.IsDebug() {
		t.db.logger.Info("批量更新完成",
			"table", t.tableName,
			"count", recordsLen,
			"affected", totalAffected,
			"duration", duration.Seconds(),
		)
	}
	return totalAffected, nil
}

// updateBatch 更新一批数据
func (t *Table) updateBatch(tx *Transaction, records []map[string]interface{}, keyField string) (int64, error) {
	if len(records) == 0 {
		return 0, nil
	}

	// 提取更新字段
	var updateFields []string
	for field := range records[0] {
		if field != keyField {
			updateFields = append(updateFields, field)
		}
	}
	if len(updateFields) == 0 {
		return 0, errors.New("没有要更新的字段")
	}

	// 构建CASE语句
	var query strings.Builder
	query.WriteString(fmt.Sprintf("UPDATE %s SET ", t.tableName))

	var args []interface{}
	for i, field := range updateFields {
		if i > 0 {
			query.WriteString(", ")
		}
		query.WriteString(fmt.Sprintf("%s = CASE %s ", field, keyField))

		for _, record := range records {
			keyValue, ok := record[keyField]
			if !ok {
				return 0, fmt.Errorf("记录缺少主键字段: %s", keyField)
			}

			value, ok := record[field]
			if !ok {
				return 0, fmt.Errorf("记录缺少更新字段: %s", field)
			}

			query.WriteString("WHEN ? THEN ? ")
			args = append(args, keyValue, value)
		}
		query.WriteString("END")
	}

	// 添加WHERE条件
	query.WriteString(" WHERE ")
	query.WriteString(keyField)
	query.WriteString(" IN (")

	for i, record := range records {
		if i > 0 {
			query.WriteString(",")
		}
		query.WriteString("?")
		args = append(args, record[keyField])
	}
	query.WriteString(")")

	// 执行SQL
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()

	result, err := tx.ExecContext(ctx, query.String(), args...)
	if err != nil {
		return 0, fmt.Errorf("执行SQL失败: %v", err)
	}

	return result.RowsAffected()
}
