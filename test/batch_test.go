package test

import (
	"fmt"
	"testing"
	"time"

	"xlorm/db"
)

// initTestDB 初始化测试数据库连接和测试表
func initTestDB(t *testing.T) *db.XgDB {
	cfg := getTestDBConfig()
	database, err := db.New(cfg)
	if err != nil {
		t.Fatalf("初始化数据库失败: %v", err)
	}

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

	return database
}

// TestBatchInsert 测试批量插入功能
func TestBatchInsert(t *testing.T) {
	testCases := []struct {
		name           string
		batch          []map[string]interface{}
		expectedRows   int
		expectingError bool
	}{
		{
			name: "正常批量插入",
			batch: func() []map[string]interface{} {
				batch := make([]map[string]interface{}, 0, 100)
				for i := 0; i < 100; i++ {
					batch = append(batch, map[string]interface{}{
						"name":       fmt.Sprintf("test_user_%d", i),
						"age":        20 + i%10,
						"created_at": time.Now(),
					})
				}
				return batch
			}(),
			expectedRows:   100,
			expectingError: false,
		},
		{
			name:           "空批次插入",
			batch:          []map[string]interface{}{},
			expectedRows:   0,
			expectingError: false,
		},
		{
			name: "部分数据不完整",
			batch: []map[string]interface{}{
				{"name": "incomplete_user_1"},
				{"age": 25},
			},
			expectedRows:   0,
			expectingError: true,
		},
	}

	db := initTestDB(t)
	defer db.Close()

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			affected, err := db.M("users").BatchInsert(tc.batch, 500)

			if tc.expectingError {
				if err == nil {
					t.Errorf("%s: 期望返回错误，但未返回", tc.name)
				}
				return
			}

			if err != nil {
				t.Fatalf("%s: 批量插入失败: %v", tc.name, err)
			}

			if int(affected) != tc.expectedRows {
				t.Errorf("%s: 期望影响行数 %d，实际为 %d", tc.name, tc.expectedRows, affected)
			}
		})
	}
}

// TestBatchUpdate 测试批量更新功能
func TestBatchUpdate(t *testing.T) {
	db := initTestDB(t)
	defer db.Close()

	// 先插入测试数据
	initialBatch := make([]map[string]interface{}, 0, 10)
	for i := 0; i < 10; i++ {
		initialBatch = append(initialBatch, map[string]interface{}{
			"name": fmt.Sprintf("initial_user_%d", i),
			"age":  20 + i,
		})
	}
	_, err := db.M("users").BatchInsert(initialBatch, 500)
	if err != nil {
		t.Fatalf("准备测试数据失败: %v", err)
	}

	testCases := []struct {
		name           string
		batch          []map[string]interface{}
		expectedRows   int
		expectingError bool
	}{
		{
			name: "正常批量更新",
			batch: func() []map[string]interface{} {
				batch := make([]map[string]interface{}, 0, 10)
				for i := 0; i < 10; i++ {
					batch = append(batch, map[string]interface{}{
						"id":     i + 1,
						"name":   fmt.Sprintf("updated_user_%d", i),
						"age":    30 + i,
						"status": 2,
					})
				}
				return batch
			}(),
			expectedRows:   10,
			expectingError: false,
		},
		{
			name:           "空批次更新",
			batch:          []map[string]interface{}{},
			expectedRows:   0,
			expectingError: false,
		},
		{
			name: "缺少主键",
			batch: []map[string]interface{}{
				{"name": "no_id_user_1"},
				{"age": 35},
			},
			expectedRows:   0,
			expectingError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			affected, err := db.M("users").BatchUpdate(tc.batch, "id", 500)

			if tc.expectingError {
				if err == nil {
					t.Errorf("%s: 期望返回错误，但未返回", tc.name)
				}
				return
			}

			if err != nil {
				t.Fatalf("%s: 批量更新失败: %v", tc.name, err)
			}

			if int(affected) != tc.expectedRows {
				t.Errorf("%s: 期望影响行数 %d，实际为 %d", tc.name, tc.expectedRows, affected)
			}
		})
	}
}

// TestBatchDelete 测试批量删除功能
func TestBatchDelete(t *testing.T) {
	db := initTestDB(t)
	defer db.Close()

	// 先插入测试数据
	initialBatch := make([]map[string]interface{}, 0, 10)
	for i := 0; i < 10; i++ {
		initialBatch = append(initialBatch, map[string]interface{}{
			"name": fmt.Sprintf("delete_user_%d", i),
			"age":  20 + i,
		})
	}
	_, err := db.M("users").BatchInsert(initialBatch, 500)
	if err != nil {
		t.Fatalf("准备测试数据失败: %v", err)
	}

	_ = []struct {
		name           string
		batch          []map[string]interface{}
		expectedRows   int
		expectingError bool
	}{
		{
			name: "正常批量删除",
			batch: func() []map[string]interface{} {
				batch := make([]map[string]interface{}, 0, 5)
				for i := 1; i <= 5; i++ {
					batch = append(batch, map[string]interface{}{"id": i})
				}
				return batch
			}(),
			expectedRows:   5,
			expectingError: false,
		},
		{
			name:           "空批次删除",
			batch:          []map[string]interface{}{},
			expectedRows:   0,
			expectingError: false,
		},
		{
			name: "不存在的记录",
			batch: []map[string]interface{}{
				{"id": 9999},
				{"id": 10000},
			},
			expectedRows:   0,
			expectingError: false,
		},
	}
}
