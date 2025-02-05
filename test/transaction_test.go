package test

import (
	"fmt"
	"testing"

	"xlorm/db"
)

// TestTransaction 测试事务功能
func TestTransaction(t *testing.T) {
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

	testCases := []struct {
		name           string
		transactionFn  func(tx *db.Transaction) error
		expectingError bool
		verifyFn       func(t *testing.T, db *db.XgDB)
	}{
		{
			name: "成功的事务插入",
			transactionFn: func(tx *db.Transaction) error {
				batch := []map[string]interface{}{
					{"name": "tx_user_1", "age": 25},
					{"name": "tx_user_2", "age": 30},
				}
				_, err := tx.DB().M("users").BatchInsert(batch, 500)
				if err != nil {
					return err
				}
				return nil
			},
			expectingError: false,
			verifyFn: func(t *testing.T, db *db.XgDB) {
				count, err := db.M("users").Count()
				if err != nil {
					t.Fatalf("查询用户数量失败: %v", err)
				}
				if count != 2 {
					t.Errorf("事务插入后用户数量不正确，期望: 2, 实际: %d", count)
				}
			},
		},
		{
			name: "回滚事务",
			transactionFn: func(tx *db.Transaction) error {
				batch := []map[string]interface{}{
					{"name": "tx_rollback_user_1", "age": 35},
				}
				_, err := tx.DB().M("users").BatchInsert(batch, 500)
				if err != nil {
					return err
				}
				return fmt.Errorf("模拟错误，触发回滚")
			},
			expectingError: true,
			verifyFn: func(t *testing.T, db *db.XgDB) {
				count, err := db.M("users").Count()
				if err != nil {
					t.Fatalf("查询用户数量失败: %v", err)
				}
				if count != 0 {
					t.Errorf("事务回滚后用户数量不正确，期望: 0, 实际: %d", count)
				}
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// 清空表
			_, err = database.Exec("TRUNCATE TABLE users")
			if err != nil {
				t.Fatalf("清空表失败: %v", err)
			}

			tx, err := database.Begin()
			if err != nil {
				t.Fatalf("开启事务失败: %v", err)
			}

			err = tc.transactionFn(tx)

			if tc.expectingError {
				if err == nil {
					tx.Rollback()
					t.Errorf("%s: 期望返回错误，但未返回", tc.name)
				} else {
					tx.Rollback()
				}
			} else {
				if err != nil {
					tx.Rollback()
					t.Errorf("%s: 事务执行失败: %v", tc.name, err)
				} else {
					err = tx.Commit()
					if err != nil {
						t.Errorf("%s: 事务提交失败: %v", tc.name, err)
					}
				}
			}

			// 验证事务结果
			tc.verifyFn(t, database)
		})
	}
}

// TestTransactionIsolation 测试事务隔离性
func TestTransactionIsolation(t *testing.T) {
	cfg := getTestDBConfig()
	database, err := db.New(cfg)
	if err != nil {
		t.Fatalf("初始化数据库失败: %v", err)
	}
	defer database.Close()

	// 清理并重建测试表
	_, err = database.Exec("DROP TABLE IF EXISTS accounts")
	if err != nil {
		t.Fatalf("删除测试表失败: %v", err)
	}

	_, err = database.Exec(`
		CREATE TABLE accounts (
			id INT AUTO_INCREMENT PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			balance DECIMAL(10, 2) NOT NULL
		)
	`)
	if err != nil {
		t.Fatalf("创建测试表失败: %v", err)
	}

	// 初始化账户
	_, err = database.M("accounts").Insert(map[string]interface{}{
		"name":    "account1",
		"balance": 1000.00,
	})
	if err != nil {
		t.Fatalf("初始化account1失败: %v", err)
	}
	_, err = database.M("accounts").Insert(map[string]interface{}{
		"name":    "account2",
		"balance": 1000.00,
	})
	if err != nil {
		t.Fatalf("初始化account2失败: %v", err)
	}

	tx, err := database.Begin()
	if err != nil {
		t.Fatalf("开启事务失败: %v", err)
	}

	// 模拟转账操作
	_, err = tx.DB().M("accounts").
		Where("name = ?", "account1").
		Update(map[string]interface{}{
			"balance": 500.00,
		})
	if err != nil {
		tx.Rollback()
		t.Fatalf("更新account1余额失败: %v", err)
	}

	_, err = tx.DB().M("accounts").
		Where("name = ?", "account2").
		Update(map[string]interface{}{
			"balance": 1500.00,
		})
	if err != nil {
		tx.Rollback()
		t.Fatalf("更新account2余额失败: %v", err)
	}

	err = tx.Commit()
	if err != nil {
		t.Fatalf("转账事务执行失败: %v", err)
	}

	// 验证转账结果
	account1, err := database.M("accounts").Where("name = ?", "account1").Find()
	if err != nil {
		t.Fatalf("查询account1失败: %v", err)
	}

	account2, err := database.M("accounts").Where("name = ?", "account2").Find()
	if err != nil {
		t.Fatalf("查询account2失败: %v", err)
	}

	if account1["balance"] != 500.00 || account2["balance"] != 1500.00 {
		t.Errorf("转账结果不正确，account1: %v, account2: %v", account1["balance"], account2["balance"])
	}
}
