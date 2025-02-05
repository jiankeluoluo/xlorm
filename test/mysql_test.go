package test

import (
	"context"
	"testing"
	"time"

	"xlorm/db"
)

// getTestDBConfig 获取测试数据库配置
func getTestDBConfig() *db.Config {
	return &db.Config{
		Host:            "localhost",
		Port:            3306,
		Username:        "root",
		Password:        "root",
		DBName:          "test_db",
		Database:        "test_db",
		Charset:         "utf8mb4",
		TablePrefix:     "test_",
		LogDir:          "./logs",
		LogLevel:        "debug",
		Debug:           true,
		MaxOpenConns:    10,
		MaxIdleConns:    5,
		ConnTimeout:     5 * time.Second,
		ConnMaxLifetime: 30 * time.Minute,
	}
}

// getTestTable 获取测试用表
func getTestTable(t *testing.T) *db.Table {
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

	return database.M("users")
}

// TestNewDB 测试数据库连接初始化
func TestNewDB(t *testing.T) {
	cfg := getTestDBConfig()
	database, err := db.New(cfg)
	if err != nil {
		t.Fatalf("初始化数据库连接失败: %v", err)
	}
	defer database.Close()

	// 测试连接是否正常
	ctx := context.Background()
	err = database.Ping(ctx)
	if err != nil {
		t.Errorf("数据库连接测试失败: %v", err)
	}
}

// TestBasicCRUD 测试基本的CRUD操作
func TestBasicCRUD(t *testing.T) {
	cfg := getTestDBConfig()
	database, err := db.New(cfg)
	if err != nil {
		t.Fatalf("初始化数据库连接失败: %v", err)
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

	// 测试插入
	testUser := map[string]interface{}{
		"name": "test_user",
		"age":  25,
	}

	id, err := database.M("users").Insert(testUser)
	if err != nil {
		t.Fatalf("插入数据失败: %v", err)
	}

	// 获取插入的ID
	if id == 0 {
		t.Fatalf("获取插入ID失败: %v", err)
	}

	// 测试查询
	user, err := database.M("users").Where("id = ?", id).Find()
	if err != nil {
		t.Fatalf("查询数据失败: %v", err)
	}

	if user["name"] != testUser["name"] || user["age"] != testUser["age"] {
		t.Errorf("查询结果与插入数据不一致")
	}

	// 测试更新
	updateData := map[string]interface{}{
		"name": "updated_user",
		"age":  30,
	}
	affected, err := database.M("users").Where("id = ?", id).Update(updateData)
	if err != nil {
		t.Fatalf("更新数据失败: %v", err)
	}
	if affected != 1 {
		t.Errorf("更新影响行数不正确，期望: 1, 实际: %d", affected)
	}

	// 测试删除
	affected, err = database.M("users").Where("id = ?", id).Delete()
	if err != nil {
		t.Fatalf("删除数据失败: %v", err)
	}
	if affected != 1 {
		t.Errorf("删除影响行数不正确，期望: 1, 实际: %d", affected)
	}
}

// TestQueryBuilder 测试查询构建器功能
func TestQueryBuilder(t *testing.T) {
	cfg := getTestDBConfig()
	database, err := db.New(cfg)
	if err != nil {
		t.Fatalf("初始化数据库连接失败: %v", err)
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
		{"name": "user1", "age": 25},
		{"name": "user2", "age": 30},
		{"name": "user3", "age": 35},
	}
	_, err = database.M("users").BatchInsert(testUsers, 500)
	if err != nil {
		t.Fatalf("插入测试数据失败: %v", err)
	}

	// 测试条件查询
	users, err := database.M("users").Where("age > ?", 25).FindAll()
	if err != nil {
		t.Fatalf("条件查询失败: %v", err)
	}
	if len(users) != 2 {
		t.Errorf("条件查询结果不正确，期望: 2, 实际: %d", len(users))
	}

	// 测试排序
	users, err = database.M("users").OrderBy("age DESC").FindAll()
	if err != nil {
		t.Fatalf("排序查询失败: %v", err)
	}
	if len(users) != 3 || users[0]["age"].(int) != 35 {
		t.Errorf("排序查询结果不正确")
	}
}

// TestHasTotal 测试获取总数功能
func TestHasTotal(t *testing.T) {
	cfg := getTestDBConfig()
	database, err := db.New(cfg)
	if err != nil {
		t.Fatalf("初始化数据库连接失败: %v", err)
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
		{"name": "user1", "age": 25},
		{"name": "user2", "age": 30},
		{"name": "user3", "age": 35},
	}
	_, err = database.M("users").BatchInsert(testUsers, 500)
	if err != nil {
		t.Fatalf("插入测试数据失败: %v", err)
	}

	// 测试总数查询
	total, err := database.M("users").Count()
	if err != nil {
		t.Fatalf("获取总数失败: %v", err)
	}
	if total != 3 {
		t.Errorf("总数查询结果不正确，期望: 3, 实际: %d", total)
	}

	// 测试条件总数查询
	total, err = database.M("users").Where("age > ?", 25).Count()
	if err != nil {
		t.Fatalf("条件总数查询失败: %v", err)
	}
	if total != 2 {
		t.Errorf("条件总数查询结果不正确，期望: 2, 实际: %d", total)
	}
}

// TestMetrics 测试性能指标收集
func TestMetrics(t *testing.T) {
	cfg := getTestDBConfig()
	database, err := db.New(cfg)
	if err != nil {
		t.Fatalf("初始化数据库连接失败: %v", err)
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
	testUser := map[string]interface{}{
		"name": "metrics_test_user",
		"age":  25,
	}

	// 测试性能指标
	_, err = database.M("users").Insert(testUser)
	if err != nil {
		t.Fatalf("插入数据失败: %v", err)
	}

	// 检查性能指标
	metrics := database.DBMetrics()
	if metrics == nil {
		t.Fatal("性能指标未正确初始化")
	}

	// 验证指标是否被正确记录
	// if metrics.TotalQueries() == 0 {
	// 	t.Error("未正确记录查询总数")
	// }
}

// TestLogger 测试日志功能
func TestLogger(t *testing.T) {
	cfg := getTestDBConfig()
	database, err := db.New(cfg)
	if err != nil {
		t.Fatalf("初始化数据库连接失败: %v", err)
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

	// 测试日志记录
	testUser := map[string]interface{}{
		"name": "logger_test_user",
		"age":  25,
	}

	// 开启调试模式
	database.SetDebug(true)

	// 插入数据并触发日志记录
	_, err = database.M("users").Insert(testUser)
	if err != nil {
		t.Fatalf("插入数据失败: %v", err)
	}
}
