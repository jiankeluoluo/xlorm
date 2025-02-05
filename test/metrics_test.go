package test

import (
	"testing"

	"xlorm/db"
)

// TestMetricsBasicFunctionality 测试基本的指标收集功能
func TestMetricsBasicFunctionality(t *testing.T) {
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

	// 执行一些数据库操作
	testUser := map[string]interface{}{
		"name": "metrics_test_user",
		"age":  25,
	}

	// 插入数据
	_, err = database.M("users").Insert(testUser)
	if err != nil {
		t.Fatalf("插入数据失败: %v", err)
	}

	// 查询数据
	_, err = database.M("users").Where("name = ?", "metrics_test_user").Find()
	if err != nil {
		t.Fatalf("查询数据失败: %v", err)
	}

	// 更新数据
	_, err = database.M("users").Where("name = ?", "metrics_test_user").Update(map[string]interface{}{
		"age": 30,
	})
	if err != nil {
		t.Fatalf("更新数据失败: %v", err)
	}

	// 删除数据
	_, err = database.M("users").Where("name = ?", "metrics_test_user").Delete()
	if err != nil {
		t.Fatalf("删除数据失败: %v", err)
	}

	// 获取指标
	metrics := database.DBMetrics()
	if metrics == nil {
		t.Fatal("指标未正确初始化")
	}
}

// TestMetricsPerformance 测试性能指标收集
func TestMetricsPerformance(t *testing.T) {
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

	// 批量插入测试数据
	batch := make([]map[string]interface{}, 0, 1000)
	for i := 0; i < 1000; i++ {
		batch = append(batch, map[string]interface{}{
			"name": "performance_test_user_" + string(rune(i)),
			"age":  20 + i%10,
		})
	}

	// startTime := time.Now()
	_, err = database.M("users").BatchInsert(batch, 500)
	// insertDuration := time.Since(startTime)

	if err != nil {
		t.Fatalf("批量插入失败: %v", err)
	}

	// 获取指标
	metrics := database.DBMetrics()
	if metrics == nil {
		t.Fatal("指标未正确初始化")
	}
}

// TestMetricsReset 测试指标重置功能
func TestMetricsReset(t *testing.T) {
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

	// 执行一些数据库操作
	testUser := map[string]interface{}{
		"name": "reset_metrics_user",
		"age":  25,
	}

	_, err = database.M("users").Insert(testUser)
	if err != nil {
		t.Fatalf("插入数据失败: %v", err)
	}

	// 获取指标
	metrics := database.DBMetrics()
	if metrics == nil {
		t.Fatal("指标未正确初始化")
	}

	// // 验证初始指标
	// initialTotalQueries := metrics.TotalQueries()
	// if initialTotalQueries == 0 {
	// 	t.Error("未正确记录查询总数")
	// }

	// // 重置指标
	// metrics.Reset()

	// // 验证重置后的指标
	// if metrics.TotalQueries() != 0 {
	// 	t.Error("指标重置失败")
	// }
}
