package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"time"
)

var safeOrderBy = regexp.MustCompile(`^[a-zA-Z0-9_, ]+$`)

// Table 表操作结构体
type Table struct {
	db        *XgDB
	tableName string
	fields    string
	orderBy   string
	groupBy   string
	having    string
	where     []string
	joins     []string
	args      []interface{}
	total     int64 // 记录集总数
	limit     int64
	offset    int64
	hasTotal  bool // 是否需要获取总数
}

// Release 释放Table对象到池中
func (t *Table) Release() {
	if t.db.IsDebug() {
		t.db.logger.Debug("释放Table对象", "table", t.tableName)
	}
	t.Reset()
	tablePool.Put(t)
}

// Reset 重置Table对象的状态
func (t *Table) Reset() {
	t.db = nil
	t.tableName = ""
	t.orderBy = ""
	t.limit = 0
	t.offset = 0
	t.fields = ""
	t.groupBy = ""
	t.having = ""
	t.where = nil
	t.args = nil
	t.joins = nil
	t.hasTotal = false
	t.total = 0
}

// ========== 表操作相关公开方法 ==========

// Where 添加查询条件
func (t *Table) Where(condition string, args ...interface{}) *Table {
	if condition == "" {
		return t
	}

	// 增强校验：检查是否有未参数化的值
	if strings.Count(condition, "?") != len(args) {
		t.db.logger.Error("条件参数数量不匹配",
			"condition", condition,
			"args_count", len(args),
		)
		return t
	}

	// 检查SQL注入
	if strings.ContainsAny(condition, ";\x00") {
		t.db.logger.Error("检测到可能的SQL注入尝试", "condition", condition)
		return t
	}

	t.where = append(t.where, condition)
	t.args = append(t.args, args...)
	return t

}

// OrderBy 添加排序条件
func (t *Table) OrderBy(order string) *Table {
	if order == "" {
		return t
	}
	if !safeOrderBy.MatchString(order) {
		t.db.logger.Error("非法排序字段", "order", order)
		return t
	}
	// 检查SQL注入
	if strings.ContainsAny(order, ";\x00") {
		t.db.logger.Error("检测到可能的SQL注入尝试", "order", order)
		return t
	}

	t.orderBy = order
	return t
}

// Limit 添加限制条件
func (t *Table) Limit(limit int64) *Table {
	if limit < 0 {
		t.db.logger.Error("limit不能为负数", "limit", limit)
		return t
	}
	t.limit = limit
	return t
}

// Page 设置分页
func (t *Table) Page(page, pageSize int64) *Table {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	t.limit = pageSize
	t.offset = (page - 1) * pageSize
	return t
}

// Offset 添加偏移量
func (t *Table) Offset(offset int64) *Table {
	if offset < 0 {
		t.db.logger.Error("offset不能为负数", "offset", offset)
		return t
	}
	t.offset = offset
	return t
}

// Fields 设置查询字段
func (t *Table) Fields(fields string) *Table {
	if fields == "" {
		return t
	}

	// 检查SQL注入
	if strings.ContainsAny(fields, ";\x00") {
		t.db.logger.Error("检测到可能的SQL注入尝试", "fields", fields)
		return t
	}

	t.fields = fields
	return t
}

// Join 添加表连接
func (t *Table) Join(join string) *Table {
	if join == "" {
		return t
	}

	// 检查SQL注入
	if strings.ContainsAny(join, ";\x00") {
		t.db.logger.Error("检测到可能的SQL注入尝试", "join", join)
		return t
	}

	t.joins = append(t.joins, join)
	return t
}

// GroupBy 添加分组条件
func (t *Table) GroupBy(groupBy string) *Table {
	if groupBy == "" {
		return t
	}

	// 检查SQL注入
	if strings.ContainsAny(groupBy, ";\x00") {
		t.db.logger.Error("检测到可能的SQL注入尝试", "groupBy", groupBy)
		return t
	}

	t.groupBy = groupBy
	return t
}

// Having 添加分组过滤条件
func (t *Table) Having(having string) *Table {
	if having == "" {
		return t
	}

	// 检查SQL注入
	if strings.ContainsAny(having, ";\x00") {
		t.db.logger.Error("检测到可能的SQL注入尝试", "having", having)
		return t
	}

	t.having = having
	return t
}

// Count 获取记录数
func (t *Table) Count() (int64, error) {
	startTime := time.Now()
	query, args := t.buildQuery("COUNT")
	var count int64
	err := t.db.QueryRow(query, args...).Scan(&count)
	if err != nil {
		t.db.asyncMetrics.RecordError()
		t.db.logger.Error("执行查询失败", "Count", query, "args", args, "error", err)
		return 0, fmt.Errorf("执行查询失败: %v", err)
	}
	t.db.asyncMetrics.RecordQueryDuration("count", time.Since(startTime))
	return count, nil
}

// Find 查询单条记录
func (t *Table) Find() (map[string]interface{}, error) {
	t.limit = 1
	t.hasTotal = false
	records, err := t.FindAll()
	if err != nil {
		return nil, err
	}

	if len(records) == 0 {
		return nil, sql.ErrNoRows
	}
	return records[0], nil
}

func (t *Table) WithContext(ctx context.Context) *Table {
	t.db.ctxMu.Lock()
	defer t.db.ctxMu.Unlock()
	t.db.ctx = ctx
	return t
}

// FindAll 查询多条记录
// 如果之前调用了HasTotal(true)，会先执行一次Count查询获取总数
// 返回值：
//   - []map[string]interface{}: 查询结果集，每个map代表一条记录，key为字段名，value为字段值
//   - error: 如果发生错误，返回具体的错误信息
//
// 可能的错误：
//   - "构建查询SQL失败": 生成SQL语句时发生错误，通常是由于查询条件不正确
//   - "执行查询失败": 执行SQL时发生错误，可能是由于网络问题或SQL语法错误
//   - "获取列信息失败": 无法获取结果集的列信息，可能是由于表结构发生变化
//   - "扫描数据失败": 将数据库返回的数据转换为Go类型时失败
func (t *Table) FindAll() ([]map[string]interface{}, error) {
	return t.FindAllWithContext(context.Background())
}

func (t *Table) FindAllWithContext(ctx context.Context) ([]map[string]interface{}, error) {
	defer t.Release()
	startTime := time.Now()
	// 如果需要获取总数，先执行 Count 查询
	if t.hasTotal {
		// 创建一个新的Table对象用于Count查询，避免影响当前查询
		countTable := t.db.M(t.tableName)
		// 复制查询条件
		t.copyQueryConditions(countTable)

		// 执行Count查询
		total, err := countTable.Count()
		if err != nil {
			return nil, fmt.Errorf("获取记录总数失败: %v", err)
		}
		t.total = total
	}

	// 构建查询SQL
	query, args := t.buildQuery("SELECT")

	if t.db.IsDebug() {
		t.db.logger.Debug("执行SQL", "FindAllWithContext", query, "args", args)
	}

	// 执行查询
	rows, err := t.db.QueryContext(ctx, query, args...)
	if err != nil {
		t.db.asyncMetrics.RecordError()
		t.db.logger.Error("执行查询失败", "FindAllWithContext", query, "args", args, "error", err)
		return nil, fmt.Errorf("执行查询失败: %v", err)
	}
	defer rows.Close()

	// 获取列名
	columns, err := rows.Columns()
	if err != nil {
		t.db.asyncMetrics.RecordError()
		t.db.logger.Error("获取列信息失败", "FindAllWithContext", query, "args", args, "error", err)
		return nil, fmt.Errorf("获取列信息失败: %v", err)
	}

	columnsLen := len(columns)

	// 预分配结果集切片，减少扩容
	var results []map[string]interface{}
	if t.limit > 0 {
		results = make([]map[string]interface{}, 0, t.limit)
	} else {
		// 如果没有limit，给一个合理的初始容量
		results = make([]map[string]interface{}, 0, 64)
	}

	// 准备扫描目标
	values := make([]interface{}, columnsLen)
	scanArgs := make([]interface{}, columnsLen)
	for i := range values {
		scanArgs[i] = &values[i]
	}

	// 扫描结果
	for rows.Next() {
		// 扫描数据
		if err := rows.Scan(scanArgs...); err != nil {
			t.db.asyncMetrics.RecordError()
			t.db.logger.Error("扫描数据失败", "FindAllWithContext", query, "args", args, "error", err)
			return nil, fmt.Errorf("扫描数据失败: %v", err)
		}

		row := make(map[string]interface{}, columnsLen)
		for i, col := range columns {
			val := values[i]
			if val == nil {
				row[col] = nil
				continue
			}

			// 处理特殊类型
			switch v := val.(type) {
			case []byte:
				// 尝试将[]byte转换为字符串
				row[col] = string(v)
			default:
				row[col] = v
			}
		}

		results = append(results, row)
	}

	// 检查遍历错误
	if err = rows.Err(); err != nil {
		t.db.asyncMetrics.RecordError()
		t.db.logger.Error("遍历结果集失败", "FindAllWithContext", query, "args", args, "error", err)
		return nil, fmt.Errorf("遍历结果集失败: %v", err)
	}

	// 记录慢查询
	duration := time.Since(startTime)

	// 记录查询耗时
	findType := "findAll"
	if t.limit == 1 {
		findType = "find"
	}
	t.db.asyncMetrics.RecordQueryDuration(findType, duration)

	if duration >= t.db.slowQueryThreshold {
		t.db.asyncMetrics.RecordSlowQuery()
		t.db.logger.Warn("慢查询",
			"query", query,
			"args", args,
			"duration", duration.Seconds(),
			"threshold", t.db.slowQueryThreshold,
			"rows", len(results),
		)
	}

	return results, nil
}

// FindAllWithCursor 使用游标逐行读取数据，减少内存占用
// handler 是处理每一行记录的回调函数，返回error时会中止处理
func (t *Table) FindAllWithCursor(ctx context.Context, handler func(map[string]interface{}) error) error {
	defer t.Release()
	startTime := time.Now()
	query, args := t.buildQuery("SELECT")

	// 执行查询
	rows, err := t.db.QueryContext(ctx, query, args...)
	if err != nil {
		t.db.asyncMetrics.RecordError()
		t.db.logger.Error("执行查询失败", "FindAllWithCursor", query, "args", args, "error", err)
		return fmt.Errorf("执行查询失败: %v", err)
	}
	defer rows.Close()

	// 获取列信息
	columns, err := rows.Columns()
	if err != nil {
		t.db.asyncMetrics.RecordError()
		t.db.logger.Error("获取列信息失败", "FindAllWithCursor", query, "args", args, "error", err)
		return fmt.Errorf("获取列信息失败: %v", err)
	}

	columnsLen := len(columns)

	// 准备扫描缓冲
	values := make([]interface{}, columnsLen)
	scanArgs := make([]interface{}, columnsLen)
	for i := range values {
		scanArgs[i] = &values[i]
	}

	// 逐行处理
	for rows.Next() {
		// 扫描数据
		if err := rows.Scan(scanArgs...); err != nil {
			t.db.asyncMetrics.RecordError()
			t.db.logger.Error("扫描数据失败", "FindAllWithCursor", query, "args", args, "error", err)
			return fmt.Errorf("扫描数据失败: %v", err)
		}

		// 转换为map
		record := make(map[string]interface{}, columnsLen)
		for i, col := range columns {
			val := values[i]
			switch v := val.(type) {
			case []byte:
				record[col] = string(v)
			default:
				record[col] = v
			}
		}

		// 调用处理函数
		if err := handler(record); err != nil {
			return err // 允许调用方中止处理流程
		}
	}

	// 检查遍历错误
	if err := rows.Err(); err != nil {
		t.db.asyncMetrics.RecordError()
		t.db.logger.Error("遍历结果集失败", "FindAllWithCursor", query, "args", args, "error", err)
		return fmt.Errorf("遍历结果集失败: %v", err)
	}

	// 记录慢查询
	duration := time.Since(startTime)
	t.db.asyncMetrics.RecordQueryDuration("findAllWithCursor", duration)

	if duration >= t.db.slowQueryThreshold {
		t.db.asyncMetrics.RecordSlowQuery()
		t.db.logger.Warn("慢查询",
			"query", query,
			"args", args,
			"duration", duration.Seconds(),
			"threshold", t.db.slowQueryThreshold,
		)
	}

	return nil
}

// HasTotal 设置是否需要获取总数
// 当设置为true时，在执行FindAll时会自动执行一次Count查询获取符合条件的记录总数
// 可以通过GetTotal方法获取查询结果
func (t *Table) HasTotal(need bool) *Table {
	t.hasTotal = need
	return t
}

// GetTotal 获取记录集总数
// 仅当在执行FindAll之前调用HasTotal(true)时才会返回有效值
// 否则返回0
func (t *Table) GetTotal() int64 {
	defer t.Release()
	return t.total
}

// Insert 插入记录
// lastInsertId 返回插入的记录的ID
// err 返回错误信息
func (t *Table) Insert(data interface{}) (lastInsertId int64, err error) {
	defer t.Release()
	return t.insert(context.Background(), data, "INSERT")
}

// InsertWithContext 插入记录
// lastInsertId 返回插入的记录的ID
// err 返回错误信息
func (t *Table) InsertWithContext(ctx context.Context, data interface{}) (lastInsertId int64, err error) {
	defer t.Release()
	return t.insert(ctx, data, "INSERT")
}

// insert 内部插入方法
func (t *Table) insert(ctx context.Context, data interface{}, insertType string) (int64, error) {
	startTime := time.Now()
	fields, values, err := t.extractFieldsAndValues(data)
	if err != nil {
		return 0, err
	}

	if len(fields) == 0 {
		return 0, errors.New("插入的数据不能为空，字段名为空")
	}

	// 构建SQL语句
	var sql strings.Builder
	sql.WriteString(fmt.Sprintf("%s INTO %s (`", insertType, t.tableName))
	sql.WriteString(strings.Join(fields, "`,`"))
	sql.WriteString("`) VALUES ")
	sql.WriteString(strings.Join(t.buildPlaceholders(len(fields), 1), ","))

	query := sql.String()

	if t.db.IsDebug() {
		t.db.logger.Debug("执行SQL", "insert", query, "args", values)
	}

	// 执行SQL
	result, err := t.db.ExecContext(ctx, query, values...)
	if err != nil {
		t.db.asyncMetrics.RecordError()
		t.db.logger.Error("执行SQL失败", "insert", query, "args", values, "error", err)
		return 0, err
	}

	// 获取最后插入的ID
	lastInsertId, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}

	t.db.asyncMetrics.RecordQueryDuration("insert", time.Since(startTime))
	return lastInsertId, nil
}

// Update 更新记录
func (t *Table) Update(data interface{}) (rowsAffected int64, err error) {
	defer t.Release()
	return t.update(context.Background(), data)
}

// UpdateWithContext 更新记录
func (t *Table) UpdateWithContext(ctx context.Context, data interface{}) (rowsAffected int64, err error) {
	defer t.Release()
	return t.update(ctx, data)
}

func (t *Table) update(ctx context.Context, data interface{}) (int64, error) {
	startTime := time.Now()
	fields, values, err := t.extractFieldsAndValues(data)
	if err != nil {
		return 0, err
	}

	// 检查是否有 WHERE 子句
	whereClause, whereArgs := t.GetWhere()
	if whereClause == "" {
		t.db.logger.Warn("更新操作未指定 WHERE 条件，拒绝执行")
		return 0, fmt.Errorf("更新操作必须指定 WHERE 条件")
	}

	// 构建SET子句
	setClause := make([]string, len(fields))
	for i, field := range fields {
		setClause[i] = fmt.Sprintf("`%s` = ?", field) //已经转义过，请勿重复转义
	}

	// 构建SQL语句
	query := fmt.Sprintf("UPDATE %s SET %s WHERE %s", t.tableName, strings.Join(setClause, ","), whereClause)

	// 合并参数
	args := append(values, whereArgs...)

	if t.db.IsDebug() {
		t.db.logger.Debug("执行SQL", "update", query, "args", args)
	}

	// 执行SQL
	result, err := t.db.ExecContext(ctx, query, args...)
	if err != nil {
		t.db.asyncMetrics.RecordError()
		t.db.logger.Error("执行SQL失败", "update", query, "args", args, "error", err)
		return 0, err
	}

	rowsAffected, _ := result.RowsAffected()
	if t.db.IsDebug() {
		t.db.logger.Debug("更新操作结果", "rowsAffected", rowsAffected)
	}

	t.db.asyncMetrics.RecordQueryDuration("update", time.Since(startTime))
	return rowsAffected, nil
}

// Delete 删除记录
func (t *Table) Delete() (rowsAffected int64, err error) {
	defer t.Release()
	return t.delete(context.Background())
}

// DeleteWithContext 删除记录
func (t *Table) DeleteWithContext(ctx context.Context) (rowsAffected int64, err error) {
	defer t.Release()
	return t.delete(ctx)
}

func (t *Table) delete(ctx context.Context) (int64, error) {
	startTime := time.Now()
	query, args := t.buildQuery("DELETE")
	if query == "" || args == nil {
		return 0, errors.New("构建查询语句失败，查询语句或参数为空")
	}
	if t.db.IsDebug() {
		t.db.logger.Debug("执行SQL", "delete", query, "args", args)
	}
	// 执行SQL
	result, err := t.db.ExecContext(ctx, query, args...)
	if err != nil {
		t.db.asyncMetrics.RecordError()
		t.db.logger.Error("执行SQL失败", "delete", query, "args", args, "error", err)
		return 0, err
	}

	rowsAffected, _ := result.RowsAffected()
	if t.db.IsDebug() {
		t.db.logger.Debug("删除操作结果", "rowsAffected", rowsAffected)
	}
	t.db.asyncMetrics.RecordQueryDuration("delete", time.Since(startTime))
	return rowsAffected, nil
}

// GetWhere 获取WHERE子句
func (t *Table) GetWhere() (string, []interface{}) {
	return strings.Join(t.where, " AND "), t.args
}

// buildQuery 构建查询语句
func (t *Table) buildQuery(queryType string) (string, []interface{}) {
	// 预估SQL长度，避免频繁扩容
	query := strings.Builder{}
	query.Grow(256)

	var args []interface{}
	if len(t.args) > 0 {
		args = make([]interface{}, 0, len(t.args))
	}

	// 构建基础查询
	switch queryType {
	case "SELECT":
		query.WriteString("SELECT ")
		if t.fields != "" {
			query.WriteString(t.fields)
		} else {
			query.WriteByte('*')
		}
		query.WriteString(" FROM ")
		query.WriteString(t.tableName)

	case "COUNT":
		query.WriteString("SELECT COUNT(*) FROM ")
		query.WriteString(t.tableName)

	case "DELETE":
		query.WriteString("DELETE FROM ")
		query.WriteString(t.tableName)

	default:
		t.db.logger.Error("不支持的查询类型", "type", queryType)
		return "", nil
	}

	// 添加连接
	if len(t.joins) > 0 {
		for _, join := range t.joins {
			query.WriteByte(' ')
			query.WriteString(join)
		}
	}

	// 添加条件
	if len(t.where) > 0 {
		query.WriteString(" WHERE ")
		query.WriteString(strings.Join(t.where, " AND "))
		args = append(args, t.args...)
	}

	// 添加分组
	if t.groupBy != "" {
		query.WriteString(" GROUP BY ")
		query.WriteString(t.groupBy)

		if t.having != "" {
			query.WriteString(" HAVING ")
			query.WriteString(t.having)
		}
	}

	// 添加排序
	if t.orderBy != "" {
		query.WriteString(" ORDER BY ")
		query.WriteString(t.orderBy)
	}

	// 添加限制和偏移
	if t.limit > 0 {
		query.WriteString(" LIMIT ?")
		args = append(args, t.limit)

		if t.offset > 0 {
			query.WriteString(" OFFSET ?")
			args = append(args, t.offset)
		}
	}

	return query.String(), args
}

// extractFieldsAndValues 提取字段和值
func (t *Table) extractFieldsAndValues(data interface{}) ([]string, []interface{}, error) {
	var fields []string
	var values []interface{}

	switch v := data.(type) {
	case map[string]interface{}:
		// 先收集并排序字段名，确保字段顺序一致
		for field := range v {
			fields = append(fields, field)
		}
		sort.Strings(fields)
		// 按照排序后的字段顺序收集值
		for _, field := range fields {
			values = append(values, v[field])
		}

	case []map[string]interface{}:
		if len(v) == 0 {
			return nil, nil, errors.New("数据不能为空")
		}
		// 先收集并排序字段名
		for field := range v[0] {
			fields = append(fields, field)
		}
		sort.Strings(fields)
		// 按照排序后的字段顺序收集每条记录的值
		for _, record := range v {
			for _, field := range fields {
				values = append(values, record[field])
			}
		}

	default:
		val := reflect.ValueOf(data)
		if val.Kind() == reflect.Ptr {
			val = val.Elem()
		}

		switch val.Kind() {
		case reflect.Struct:
			fields, values = t.extractFromStruct(val)

		case reflect.Slice:
			if val.Len() == 0 {
				return nil, nil, errors.New("数据不能为空")
			}
			fields, values = extractFromStructSlice(val)

		default:
			return nil, nil, errors.New("数据类型必须是 map、struct 或它们的切片")
		}
	}

	if len(fields) == 0 {
		return nil, nil, errors.New("没有需要操作的字段")
	}

	return fields, values, nil
}

// extractFromStruct 从结构体提取字段和值
func (t *Table) extractFromStruct(val reflect.Value) ([]string, []interface{}) {
	typ := val.Type()
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}
	// 生成包含嵌套结构的唯一key
	cacheKey := genStructKey(typ)
	// 尝试从缓存获取字段
	if fields, ok := t.db.structFieldsCache.Get(cacheKey); ok {
		values := make([]interface{}, len(fields))
		for i, f := range fields {
			values[i] = val.FieldByName(f).Interface()
		}
		return fields, values
	}
	var fields []string
	var fieldNames []string
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		if tag := field.Tag.Get("db"); tag != "" {
			tagParts := strings.Split(tag, ",")
			fieldName := tagParts[0]
			fields = append(fields, escapeSQLIdentifier(fieldName))
			fieldNames = append(fieldNames, fieldName)
		}
	}

	// 缓存字段信息
	if len(fieldNames) > 0 {
		t.db.structFieldsCache.Set(cacheKey, fieldNames)
	}

	values := make([]interface{}, len(fieldNames))
	for i, name := range fieldNames {
		values[i] = val.FieldByName(name).Interface()
	}

	return fields, values
}

// buildPlaceholders 构建占位符
func (t *Table) buildPlaceholders(fieldCount, recordCount int) []string {
	// 2. 直接创建目标切片
	placeholders := make([]string, recordCount)

	// 3. 并行填充（Go 1.21+ 新增清析语法）
	clear(placeholders) // 显式初始化（非必须）

	// 4. 内存预分配优化
	if recordCount > 0 {
		placeholders[0] = getCachedPlaceholder(fieldCount, t.db.placeholderCache) //生成带括号的单记录占位符
		for i := 1; i < recordCount; i *= 2 {
			copy(placeholders[i:], placeholders[:i])
		}
	}

	return placeholders
}

// copyQueryConditions 复制查询条件到目标Table对象
// 用于在不影响原查询的情况下执行Count等操作
func (t *Table) copyQueryConditions(target *Table) {
	if len(t.where) > 0 {
		target.where = make([]string, len(t.where))
		copy(target.where, t.where)
	}

	if len(t.args) > 0 {
		target.args = make([]interface{}, len(t.args))
		copy(target.args, t.args)
	}

	if len(t.joins) > 0 {
		target.joins = make([]string, len(t.joins))
		copy(target.joins, t.joins)
	}

	target.groupBy = t.groupBy
	target.having = t.having
}
