package test

import (
	"fmt"
	"log"
	"testing"
	"xlorm/db"
)

var testEngine *db.XgDB

func init() {
	cfg := &db.Config{
		DBName:             "master",
		Driver:             "MySQL",
		Host:               "localhost",
		Port:               3306,
		Database:           "test_db",
		Username:           "root",
		Password:           "root",
		Charset:            "utf8mb4",
		TablePrefix:        "test_",
		LogDir:             "./logs",
		LogLevel:           "debug",
		Debug:              false,
		EnablePoolStats:    false,
		LogRotationEnabled: true,
		LogRotationMaxAge:  30,
	}

	var err error
	testEngine, err = db.New(cfg)
	if err != nil {
		log.Fatal(err)
	}
}

func BenchmarkInsert(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		data := map[string]interface{}{
			"name": fmt.Sprintf("测试用户_%d", i),
			"age":  25,
		}
		_, err := testEngine.M("users").Insert(data)
		if err != nil {
			b.Fatalf("插入失败: %v", err)
		}
	}
}

func BenchmarkBatchInsert(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		records := []map[string]interface{}{
			{"name": fmt.Sprintf("用户1_%d", i), "age": 20},
			{"name": fmt.Sprintf("用户2_%d", i), "age": 21},
			{"name": fmt.Sprintf("用户3_%d", i), "age": 22},
		}
		_, err := testEngine.M("users").BatchInsert(records, 500)
		if err != nil {
			b.Fatalf("批量插入失败: %v", err)
		}
	}
}

func BenchmarkFind(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := testEngine.M("users").Fields("`id`,`name`,`age`").Where("id = ?", 2).Find()
		if err != nil {
			b.Fatalf("查询失败: %v", err)
		}
	}
}

func BenchmarkFindAll(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := testEngine.M("users").
			Where("age > ?", 30).
			OrderBy("age DESC").
			FindAll()
		if err != nil {
			b.Fatalf("查询失败: %v", err)
		}
	}
}

func BenchmarkUpdate(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		data := map[string]interface{}{
			"age": 26 + i%10,
		}
		_, err := testEngine.M("users").Where("id = ?", 1).Update(data)
		if err != nil {
			b.Fatalf("更新失败: %v", err)
		}
	}
}

func BenchmarkBatchUpdate(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		updates := []map[string]interface{}{
			{
				"id":   1,
				"name": fmt.Sprintf("张三(更新_%d)", i),
				"age":  30 + i%10,
			},
			{
				"id":   2,
				"name": fmt.Sprintf("李四(更新_%d)", i),
				"age":  31 + i%10,
			},
		}
		_, err := testEngine.M("users").BatchUpdate(updates, "id", 500)
		if err != nil {
			b.Fatalf("批量更新失败: %v", err)
		}
	}
}

func BenchmarkDelete(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := testEngine.M("users").Where("id = ?", 1+i%10).Delete()
		if err != nil {
			b.Fatalf("删除失败: %v", err)
		}
	}
}

func BenchmarkTransaction(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tx, err := testEngine.Begin()
		if err != nil {
			b.Fatalf("开启事务失败: %v", err)
		}

		newRecords := []map[string]interface{}{
			{
				"name": fmt.Sprintf("赵六_%d", i),
				"age":  23,
			},
		}

		_, err = tx.DB().M("users").BatchInsert(newRecords, 500)
		if err != nil {
			tx.Rollback()
			b.Fatalf("事务中批量插入失败: %v", err)
		}

		err = tx.Commit()
		if err != nil {
			b.Fatalf("事务提交失败: %v", err)
		}
	}
}
