package test

import (
	"testing"

	"xlorm/db"
)

// TestBuilderSelect 测试查询构建器
func TestBuilderSelect(t *testing.T) {
	cfg := getTestDBConfig()
	database, err := db.New(cfg)
	if err != nil {
		t.Fatalf("初始化数据库失败: %v", err)
	}
	defer database.Close()

	// 清理并重建测试表
	_, err = database.Exec("DROP TABLE IF EXISTS users")
	if err != nil {
		t.Fatalf("删除测试表失败: %v", err)
	}

	_, err = database.Exec(`
		CREATE TABLE users (
			id INT AUTO_INCREMENT PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			age INT NOT NULL,
			status INT DEFAULT 1,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		t.Fatalf("创建测试表失败: %v", err)
	}

	// 插入测试数据
	testUsers := []map[string]interface{}{
		{"name": "user1", "age": 25, "status": 1},
		{"name": "user2", "age": 30, "status": 2},
		{"name": "user3", "age": 35, "status": 1},
	}
	_, err = database.M("users").BatchInsert(testUsers, 500)
	if err != nil {
		t.Fatalf("插入测试数据失败: %v", err)
	}

	testCases := []struct {
		name           string
		builderFn      func(*db.Table) *db.Table
		expectedCount  int
		expectedFirst  string
		expectedValues map[string]interface{}
	}{
		{
			name: "简单条件查询",
			builderFn: func(table *db.Table) *db.Table {
				return table.Where("age > ?", 25)
			},
			expectedCount: 2,
			expectedFirst: "user2",
		},
		{
			name: "多条件查询",
			builderFn: func(table *db.Table) *db.Table {
				return table.Where("age > ? AND status = ?", 25, 1)
			},
			expectedCount: 1,
			expectedFirst: "user3",
		},
		{
			name: "排序查询",
			builderFn: func(table *db.Table) *db.Table {
				return table.OrderBy("age DESC")
			},
			expectedCount: 3,
			expectedFirst: "user3",
			expectedValues: map[string]interface{}{
				"name": "user3",
				"age":  35,
			},
		},
		{
			name: "限制查询",
			builderFn: func(table *db.Table) *db.Table {
				return table.Limit(1).OrderBy("age ASC")
			},
			expectedCount: 1,
			expectedFirst: "user1",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			builder := tc.builderFn(database.M("users"))
			results, err := builder.FindAll()
			if err != nil {
				t.Fatalf("%s: 查询失败: %v", tc.name, err)
			}

			if len(results) != tc.expectedCount {
				t.Errorf("%s: 查询结果数量不正确，期望: %d, 实际: %d", tc.name, tc.expectedCount, len(results))
			}

			if tc.expectedFirst != "" && results[0]["name"] != tc.expectedFirst {
				t.Errorf("%s: 第一条记录名称不正确，期望: %s, 实际: %s", tc.name, tc.expectedFirst, results[0]["name"])
			}

			if tc.expectedValues != nil {
				for k, v := range tc.expectedValues {
					if results[0][k] != v {
						t.Errorf("%s: 字段 %s 值不正确，期望: %v, 实际: %v", tc.name, k, v, results[0][k])
					}
				}
			}
		})
	}
}

// TestBuilderAggregate 测试聚合函数
func TestBuilderAggregate(t *testing.T) {
	cfg := getTestDBConfig()
	database, err := db.New(cfg)
	if err != nil {
		t.Fatalf("初始化数据库失败: %v", err)
	}
	defer database.Close()

	// 清理并重建测试表
	_, err = database.Exec("DROP TABLE IF EXISTS users")
	if err != nil {
		t.Fatalf("删除测试表失败: %v", err)
	}

	_, err = database.Exec(`
		CREATE TABLE users (
			id INT AUTO_INCREMENT PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			age INT NOT NULL,
			salary DECIMAL(10, 2) NOT NULL,
			status INT DEFAULT 1,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		t.Fatalf("创建测试表失败: %v", err)
	}

	// 插入测试数据
	testUsers := []map[string]interface{}{
		{"name": "user1", "age": 25, "salary": 5000.00, "status": 1},
		{"name": "user2", "age": 30, "salary": 7500.50, "status": 2},
		{"name": "user3", "age": 35, "salary": 6200.75, "status": 1},
	}
	_, err = database.M("users").BatchInsert(testUsers, 500)
	if err != nil {
		t.Fatalf("插入测试数据失败: %v", err)
	}
}
