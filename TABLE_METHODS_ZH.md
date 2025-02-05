# XLORM Table 方法文档

[中文版](TABLE_METHODS_ZH.md "访问中文版")

[English](TABLE_METHODS_EN.md "Access English Version")

## 查询条件方法

### Where
- 添加查询条件
- 签名：`Where(condition string, args ...interface{}) *Table`
- 示例：`table.Where("id = ?", 1)`

### OrderBy
- 添加排序条件
- 签名：`OrderBy(order string) *Table`
- 示例：`table.OrderBy("created_at desc")`

### Limit
- 添加记录数限制
- 签名：`Limit(limit int64) *Table`
- 示例：`table.Limit(10)`

### Page
- 设置分页
- 签名：`Page(page, pageSize int64) *Table`
- 示例：`table.Page(1, 20)` // 第1页，每页20条记录

### Offset
- 添加偏移量
- 签名：`Offset(offset int64) *Table`
- 示例：`table.Offset(10)` // 跳过前10条记录

### Fields
- 设置查询字段
- 签名：`Fields(fields string) *Table`
- 示例：`table.Fields("id, name, age")`

### Join
- 添加表连接
- 签名：`Join(join string) *Table`
- 示例：`table.Join("LEFT JOIN users ON users.id = orders.user_id")`

### GroupBy
- 添加分组条件
- 签名：`GroupBy(groupBy string) *Table`
- 示例：`table.GroupBy("category")`

### Having
- 添加分组过滤条件
- 签名：`Having(having string) *Table`
- 示例：`table.Having("count(*) > 10")`

## 查询方法

### Count
- 获取记录数
- 签名：`Count() (int64, error)`
- 示例：`count, err := table.Count()`

### Find
- 查询单条记录
- 签名：`Find() (map[string]interface{}, error)`
- 示例：`record, err := table.Find()`

### FindAll
- 查询多条记录
- 签名：`FindAll() ([]map[string]interface{}, error)`
- 示例：`records, err := table.FindAll()`

### FindAllWithCursor
- 使用游标逐行读取数据
- 签名：`FindAllWithCursor(ctx context.Context, handler func(map[string]interface{}) error) error`
- 示例：
```go
err := table.FindAllWithCursor(ctx, func(record map[string]interface{}) error {
    // 处理每一行记录
    return nil
})
```

## 上下文方法

### WithContext
- 设置上下文
- 签名：`WithContext(ctx context.Context) *Table`
- 示例：`table.WithContext(ctx)`

### FindAllWithContext
- 带上下文的多记录查询
- 签名：`FindAllWithContext(ctx context.Context) ([]map[string]interface{}, error)`
- 示例：`records, err := table.FindAllWithContext(ctx)`

## 总数控制方法

### HasTotal
- 设置是否需要获取总数
- 签名：`HasTotal(need bool) *Table`
- 示例：`table.HasTotal(true)`

### GetTotal
- 获取记录集总数
- 签名：`GetTotal() int64`
- 示例：`total := table.GetTotal()`

## 数据操作方法

### Insert
- 插入记录
- 签名：`Insert(data interface{}) (lastInsertId int64, err error)`
- 示例：`id, err := table.Insert(data)`

### InsertWithContext
- 带上下文的插入记录
- 签名：`InsertWithContext(ctx context.Context, data interface{}) (lastInsertId int64, err error)`
- 示例：`id, err := table.InsertWithContext(ctx, data)`

### Update
- 更新记录
- 签名：`Update(data interface{}) (rowsAffected int64, err error)`
- 示例：`affected, err := table.Update(data)`

### UpdateWithContext
- 带上下文的更新记录
- 签名：`UpdateWithContext(ctx context.Context, data interface{}) (rowsAffected int64, err error)`
- 示例：`affected, err := table.UpdateWithContext(ctx, data)`

### Delete
- 删除记录
- 签名：`Delete() (rowsAffected int64, err error)`
- 示例：`affected, err := table.Delete()`

### DeleteWithContext
- 带上下文的删除记录
- 签名：`DeleteWithContext(ctx context.Context) (rowsAffected int64, err error)`
- 示例：`affected, err := table.DeleteWithContext(ctx)`

## 批量操作方法

### BatchInsert
- 批量插入记录
- 签名：`BatchInsert(data []map[string]interface{}, batchSize int) (totalAffecteds int64, err error)`
- 示例：
```go
users := []map[string]interface{}{
    {"name": "Alice", "age": 25},
    {"name": "Bob", "age": 30},
}
affected, err := table.BatchInsert(users, 100)
```

### BatchUpdate
- 批量更新记录
- 签名：`BatchUpdate(records []map[string]interface{}, keyField string, batchSize int) (totalAffecteds int64, err error)`
- 示例：
```go
users := []map[string]interface{}{
    {"id": 1, "name": "Alice Updated", "age": 26},
    {"id": 2, "name": "Bob Updated", "age": 31},
}
affected, err := table.BatchUpdate(users, "id", 100)
```

## 批量操作注意事项
- 批量操作支持大规模数据处理
- 可以自定义批次大小
- 支持灵活的数据处理

## 注意事项
- 大多数方法支持链式调用
- 内置 SQL 注入防护机制
- 支持上下文和非上下文版本的方法
