package db

import (
	"fmt"
	"strings"
)

// Builder SQL查询构建器结构体
type Builder struct {
	groupBy   string        // GROUP BY 子句
	having    string        // HAVING 子句
	orderBy   string        // ORDER BY 子句
	table     string        // 表名
	fields    []string      // 字段列表
	where     []string      // WHERE 条件
	joins     []string      // JOIN 子句
	args      []interface{} // 查询参数
	limit     int64         // 查询限制
	offset    int64         // 查询偏移
	forUpdate bool          // 是否为 FOR UPDATE 查询
}

// NewBuilder 创建查询构建器
func (db *XgDB) NewBuilder(table string) *Builder {
	b := builderPool.Get().(*Builder)
	b.table = table
	b.fields = b.fields[:0]
	b.where = b.where[:0]
	b.args = b.args[:0]
	b.joins = b.joins[:0]
	b.groupBy = ""
	b.having = ""
	b.orderBy = ""
	b.limit = 0
	b.offset = 0
	b.forUpdate = false
	return b
}

// Fields 设置查询字段
func (b *Builder) Fields(fields ...string) *Builder {
	b.fields = fields
	return b
}

// Where 添加查询条件
func (b *Builder) Where(condition string, args ...interface{}) *Builder {
	b.where = append(b.where, condition)
	b.args = append(b.args, args...)
	return b
}

// Join 添加连接
func (b *Builder) Join(join string) *Builder {
	b.joins = append(b.joins, join)
	return b
}

// GroupBy 添加分组
func (b *Builder) GroupBy(groupBy string) *Builder {
	b.groupBy = groupBy
	return b
}

// Having 添加分组条件
func (b *Builder) Having(having string) *Builder {
	b.having = having
	return b
}

// OrderBy 添加排序
func (b *Builder) OrderBy(orderBy string) *Builder {
	b.orderBy = orderBy
	return b
}

// Limit 添加限制
func (b *Builder) Limit(limit int64) *Builder {
	b.limit = limit
	return b
}

// Page 设置分页
func (b *Builder) Page(page, pageSize int64) *Builder {
	b.limit = pageSize
	b.offset = (page - 1) * pageSize
	return b
}

// Offset 添加偏移
func (b *Builder) Offset(offset int64) *Builder {
	b.offset = offset
	return b
}

// ForUpdate 添加行锁
func (b *Builder) ForUpdate() *Builder {
	b.forUpdate = true
	return b
}

// Build 构建SQL语句
func (b *Builder) Build() (string, []interface{}) {
	var query strings.Builder
	query.WriteString("SELECT ")

	// 处理字段
	if len(b.fields) == 0 {
		query.WriteString("*")
	} else {
		query.WriteString(strings.Join(b.fields, ", "))
	}

	// 添加表名
	query.WriteString(fmt.Sprintf(" FROM %s", b.table))

	// 添加连接
	if len(b.joins) > 0 {
		query.WriteString(" " + strings.Join(b.joins, " "))
	}

	// 添加条件
	if len(b.where) > 0 {
		query.WriteString(" WHERE " + strings.Join(b.where, " AND "))
	}

	// 添加分组
	if b.groupBy != "" {
		query.WriteString(" GROUP BY " + b.groupBy)
	}

	// 添加分组条件
	if b.having != "" {
		query.WriteString(" HAVING " + b.having)
	}

	// 添加排序
	if b.orderBy != "" {
		query.WriteString(" ORDER BY " + b.orderBy)
	}

	// 添加限制
	if b.limit > 0 {
		query.WriteString(fmt.Sprintf(" LIMIT %d", b.limit))
	}

	// 添加偏移
	if b.offset > 0 {
		query.WriteString(fmt.Sprintf(" OFFSET %d", b.offset))
	}

	// 添加行锁
	if b.forUpdate {
		query.WriteString(" FOR UPDATE")
	}

	return query.String(), b.args
}

// ReleaseBuilder 释放Builder对象到池中
func (b *Builder) ReleaseBuilder() {
	builderPool.Put(b)
}
