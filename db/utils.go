package db

import (
	"errors"
	"fmt"
	"log/slog"
	"reflect"
	"strings"
	"time"
	"unicode"
)

// 新增：SQL标识符转义函数
func escapeSQLIdentifier(name string) string {
	// 添加对保留字的过滤
	reservedWords := map[string]bool{
		"select": true,
		"insert": true,
		"update": true,
		"delete": true,
	}
	if reservedWords[strings.ToLower(name)] {
		return "`invalid`"
	}

	// 过滤非法字符，仅允许字母、数字、下划线和点
	var safeName strings.Builder
	for _, r := range name {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '.' {
			safeName.WriteRune(r)
		}
	}
	if safeName.Len() == 0 {
		return "``"
	}
	return "`" + safeName.String() + "`"
}

// safeTimeout 带最小值的超时时间
func safeTimeout(d time.Duration) string {
	if d <= 1 {
		return "1s"
	}
	return fmt.Sprintf("%vs", d.Seconds())
}

func getCachedPlaceholder(fieldCount int, placeholderCache *ShardedCache) string {
	keyName := fmt.Sprintf("placeholder:%d", fieldCount)
	if v, ok := placeholderCache.Get(keyName); ok {
		return v[0] // 直接返回第一个元素
	}
	s := "(" + strings.Repeat("?,", fieldCount-1) + "?)"
	placeholderCache.Set(keyName, []string{s})
	return s
}

func getCacheTableName(tableName, perfix string) string {
	var builder strings.Builder
	builder.WriteString("`")
	builder.WriteString(perfix)
	builder.WriteString(tableName)
	builder.WriteString("`")
	return builder.String()
}

// 生成结构体唯一标识
func genStructKey(typ reflect.Type) string {
	return typ.PkgPath() + "." + typ.Name() + "@" + typ.String()
}

// extractFromStructSlice 从结构体切片提取字段和值
func extractFromStructSlice(val reflect.Value) ([]string, []interface{}) {
	var fields []string
	var values []interface{}

	firstItem := val.Index(0)
	if firstItem.Kind() == reflect.Ptr {
		firstItem = firstItem.Elem()
	}
	typ := firstItem.Type()

	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		if tag := field.Tag.Get("db"); tag != "" {
			tagParts := strings.Split(tag, ",")
			fields = append(fields, escapeSQLIdentifier(tagParts[0]))
		}
	}

	for i := 0; i < val.Len(); i++ {
		item := val.Index(i)
		if item.Kind() == reflect.Ptr {
			item = item.Elem()
		}
		for j := 0; j < item.NumField(); j++ {
			if tag := typ.Field(j).Tag.Get("db"); tag != "" {
				values = append(values, item.Field(j).Interface())
			}
		}
	}

	return fields, values
}

// extractBatchFields 从批量数据中提取字段
func extractBatchFields(data []map[string]interface{}) ([]string, error) {
	if len(data) == 0 {
		return nil, errors.New("数据为空")
	}

	// 从第一条记录提取字段
	fields := make([]string, 0, len(data[0]))
	for field := range data[0] {
		// 转义字段名
		escapedField := escapeSQLIdentifier(field)
		fields = append(fields, escapedField)
	}

	// 验证所有记录的字段一致性
	for _, item := range data[1:] {
		if len(item) != len(fields) {
			return nil, fmt.Errorf("批量插入数据字段不一致：第一条记录有 %d 个字段，当前记录有 %d 个字段", len(fields), len(item))
		}
	}

	return fields, nil
}

func parseLogLevel(level string) (slog.Level, error) {
	l, ok := logLevelMap[strings.ToLower(level)]
	if !ok || level == "" {
		return slog.LevelInfo, fmt.Errorf("无效的日志级别: %s,可选值:debug|info|warn|error", level)
	}
	return l, nil
}
