# XLORM Builder Methods Documentation

## Creating a Builder

### NewBuilder
- Create a query builder
- Signature: `NewBuilder(table string) *Builder`
- Example: `builder := db.NewBuilder("users")`

## Query Configuration Methods

### Fields
- Set query fields
- Signature: `Fields(fields ...string) *Builder`
- Example: `builder.Fields("id", "name", "age")`

### Where
- Add query conditions
- Signature: `Where(condition string, args ...interface{}) *Builder`
- Example: `builder.Where("age > ?", 18)`

### Join
- Add table join
- Signature: `Join(join string) *Builder`
- Example: `builder.Join("LEFT JOIN orders ON users.id = orders.user_id")`

### GroupBy
- Add grouping
- Signature: `GroupBy(groupBy string) *Builder`
- Example: `builder.GroupBy("category")`

### Having
- Add group filtering conditions
- Signature: `Having(having string) *Builder`
- Example: `builder.Having("count(*) > 10")`

### OrderBy
- Add sorting
- Signature: `OrderBy(orderBy string) *Builder`
- Example: `builder.OrderBy("created_at desc")`

### Limit
- Limit the number of records
- Signature: `Limit(limit int64) *Builder`
- Example: `builder.Limit(10)`

### Page
- Set pagination
- Signature: `Page(page, pageSize int64) *Builder`
- Example: `builder.Page(1, 20)` // Page 1, 20 records per page

### Offset
- Add offset
- Signature: `Offset(offset int64) *Builder`
- Example: `builder.Offset(10)` // Skip the first 10 records

### ForUpdate
- Add row lock
- Signature: `ForUpdate() *Builder`
- Example: `builder.ForUpdate()`

## Build Methods

### Build
- Build SQL statement
- Signature: `Build() (string, []interface{})`
- Example:
```go
query, args := builder.Build()
// query is the generated SQL statement
// args are the corresponding parameters
```

### ReleaseBuilder
- Release Builder object to the pool
- Signature: `ReleaseBuilder()`
- Example: `builder.ReleaseBuilder()`

## Usage Example

```go
query, args := db.NewBuilder("users").
    Fields("id", "name").
    Where("age > ?", 18).
    OrderBy("created_at desc").
    Limit(10).
    Build()
```

## Precautions
- Supports method chaining
- Flexible query condition configuration
- Uses object pool to manage Builder objects for performance optimization
- Automatically handles fields, conditions, grouping, sorting, and other query parameters
