package db

import (
	"fmt"
	"strconv"
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

	// 新增位运算相关字段
	conditionFlags uint64
	conditionIndex int
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

	// 重置位运算相关字段
	b.conditionFlags = 0
	b.conditionIndex = 0
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

	// 更新位标记和索引
	if b.conditionIndex == 0 {
		b.conditionFlags |= condAND // 第一个条件默认为 AND
	}
	b.conditionIndex++

	return b
}

// OrWhere 添加 OR 查询条件
func (b *Builder) OrWhere(condition string, args ...interface{}) *Builder {
	b.where = append(b.where, condition)
	b.args = append(b.args, args...)

	// 更新位标记和索引
	b.conditionFlags |= condOR
	b.conditionIndex++

	return b
}

// NotWhere 添加 NOT 查询条件
func (b *Builder) NotWhere(condition string, args ...interface{}) *Builder {
	// 为 NOT 条件添加 NOT 前缀
	notCondition := fmt.Sprintf("NOT (%s)", condition)
	b.where = append(b.where, notCondition)
	b.args = append(b.args, args...)

	// 更新位标记和索引
	b.conditionFlags |= condNOT
	b.conditionIndex++

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
	defer b.ReleaseBuilder()
	var query strings.Builder
	query.WriteString("SELECT ")

	// 处理字段
	if len(b.fields) == 0 {
		query.WriteString("*")
	} else {
		query.WriteString("`")
		query.WriteString(strings.Join(b.fields, "`, `"))
		query.WriteString("`")
	}

	// 添加表名
	query.WriteString(" FROM ")
	query.WriteString(b.table)

	// 添加连接
	if len(b.joins) > 0 {
		query.WriteByte(' ')
		query.WriteString(strings.Join(b.joins, " "))
	}

	// 添加条件
	if len(b.where) > 0 {
		query.WriteString(" WHERE ")

		// 使用位运算快速判断条件类型
		switch {
		case b.conditionFlags&condOR != 0:
			// 存在 OR 条件，使用括号确保正确性
			query.WriteByte('(')
			for i, condition := range b.where {
				if i > 0 {
					query.WriteString(" OR ")
				}
				query.WriteString(condition)
			}
			query.WriteByte(')')

		case b.conditionFlags&condNOT != 0:
			// 存在 NOT 条件，使用括号确保正确性
			query.WriteByte('(')
			for i, condition := range b.where {
				if i > 0 {
					query.WriteString(" AND ")
				}
				query.WriteString(condition)
			}
			query.WriteByte(')')

		default:
			// 纯 AND 条件，直接连接
			query.WriteString(strings.Join(b.where, " AND "))
		}

		// 重置条件标记
		b.conditionFlags = 0
		b.conditionIndex = 0
	}

	// 添加分组
	if b.groupBy != "" {
		query.WriteString(" GROUP BY ")
		query.WriteString(b.groupBy)
	}

	// 添加分组条件
	if b.having != "" {
		query.WriteString(" HAVING ")
		query.WriteString(b.having)
	}

	// 添加排序
	if b.orderBy != "" {
		query.WriteString(" ORDER BY ")
		query.WriteString(b.orderBy)
	}

	// 添加限制
	if b.limit > 0 {
		query.WriteString(" LIMIT ")
		query.WriteString(strconv.FormatInt(b.limit, 10))
	}

	// 添加偏移
	if b.offset > 0 {
		query.WriteString(" OFFSET ")
		query.WriteString(strconv.FormatInt(b.offset, 10))
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
