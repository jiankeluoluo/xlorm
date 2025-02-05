# XLORM Builder 方法文档

## 创建构建器

### NewBuilder
- 创建查询构建器
- 签名：`NewBuilder(table string) *Builder`
- 示例：`builder := db.NewBuilder("users")`

## 查询配置方法

### Fields
- 设置查询字段
- 签名：`Fields(fields ...string) *Builder`
- 示例：`builder.Fields("id", "name", "age")`

### Where
- 添加查询条件
- 签名：`Where(condition string, args ...interface{}) *Builder`
- 示例：`builder.Where("age > ?", 18)`

### Join
- 添加连接
- 签名：`Join(join string) *Builder`
- 示例：`builder.Join("LEFT JOIN orders ON users.id = orders.user_id")`

### GroupBy
- 添加分组
- 签名：`GroupBy(groupBy string) *Builder`
- 示例：`builder.GroupBy("category")`

### Having
- 添加分组条件
- 签名：`Having(having string) *Builder`
- 示例：`builder.Having("count(*) > 10")`

### OrderBy
- 添加排序
- 签名：`OrderBy(orderBy string) *Builder`
- 示例：`builder.OrderBy("created_at desc")`

### Limit
- 添加记录数限制
- 签名：`Limit(limit int64) *Builder`
- 示例：`builder.Limit(10)`

### Page
- 设置分页
- 签名：`Page(page, pageSize int64) *Builder`
- 示例：`builder.Page(1, 20)` // 第1页，每页20条记录

### Offset
- 添加偏移
- 签名：`Offset(offset int64) *Builder`
- 示例：`builder.Offset(10)` // 跳过前10条记录

### ForUpdate
- 添加行锁
- 签名：`ForUpdate() *Builder`
- 示例：`builder.ForUpdate()`

## 构建方法

### Build
- 构建SQL语句
- 签名：`Build() (string, []interface{})`
- 示例：
```go
query, args := builder.Build()
// query 为生成的 SQL 语句
// args 为对应的参数
```

### ReleaseBuilder
- 释放Builder对象到池中
- 签名：`ReleaseBuilder()`
- 示例：`builder.ReleaseBuilder()`

## 使用示例

```go
query, args := db.NewBuilder("users").
    Fields("id", "name").
    Where("age > ?", 18).
    OrderBy("created_at desc").
    Limit(10).
    Build()
```

## 注意事项
- 支持链式调用
- 可以灵活配置查询条件
- 使用对象池管理 Builder 对象，提高性能
- 自动处理字段、条件、分组、排序等查询参数
