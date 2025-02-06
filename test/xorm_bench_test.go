package test

import (
	"fmt"
	"log"
	"testing"

	_ "github.com/go-sql-driver/mysql"
	"xorm.io/xorm"
)

var xormDB *xorm.Engine

type XUser struct {
	ID   int    `xorm:"'id' pk autoincr"`
	Name string `xorm:"'name'"`
	Age  int    `xorm:"'age'"`
}

func init() {
	dsn := "root:770880@tcp(localhost:3306)/test_db?charset=utf8mb4"
	var err error
	xormDB, err = xorm.NewEngine("mysql", dsn)
	if err != nil {
		log.Fatal(err)
	}
}

func BenchmarkXORM_Insert(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		user := XUser{Name: fmt.Sprintf("测试用户_%d", i), Age: 25}
		if _, err := xormDB.Insert(&user); err != nil {
			b.Fatalf("插入失败: %v", err)
		}
	}
}

func BenchmarkXORM_BatchInsert(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		users := []XUser{
			{Name: fmt.Sprintf("用户1_%d", i), Age: 20},
			{Name: fmt.Sprintf("用户2_%d", i), Age: 21},
			{Name: fmt.Sprintf("用户3_%d", i), Age: 22},
		}
		if _, err := xormDB.Insert(&users); err != nil {
			b.Fatalf("批量插入失败: %v", err)
		}
	}
}

func BenchmarkXORM_Find(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var user XUser
		if _, err := xormDB.Cols("id", "name", "age").Where("id = ?", 2).Get(&user); err != nil {
			b.Fatalf("查询失败: %v", err)
		}
	}
}

func BenchmarkXORM_FindAll(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var users []XUser
		if err := xormDB.Where("age > ?", 30).Desc("age").Find(&users); err != nil {
			b.Fatalf("查询失败: %v", err)
		}
	}
}

func BenchmarkXORM_Update(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := xormDB.Table("users").Where("id = ?", 1).Update(&XUser{Age: 26 + i%10}); err != nil {
			b.Fatalf("更新失败: %v", err)
		}
	}
}

func BenchmarkXORM_BatchUpdate(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		session := xormDB.NewSession()
		defer session.Close()

		if err := session.Begin(); err != nil {
			b.Fatalf("开启事务失败: %v", err)
		}

		if _, err := session.ID(1).Update(&XUser{Name: fmt.Sprintf("张三(更新_%d)", i), Age: 30 + i%10}); err != nil {
			session.Rollback()
			b.Fatalf("批量更新失败: %v", err)
		}

		if _, err := session.ID(2).Update(&XUser{Name: fmt.Sprintf("李四(更新_%d)", i), Age: 31 + i%10}); err != nil {
			session.Rollback()
			b.Fatalf("批量更新失败: %v", err)
		}

		if err := session.Commit(); err != nil {
			b.Fatalf("事务提交失败: %v", err)
		}
	}
}

func BenchmarkXORM_Delete(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := xormDB.Where("id = ?", 1+i%10).Delete(&XUser{}); err != nil {
			b.Fatalf("删除失败: %v", err)
		}
	}
}

func BenchmarkXORM_Transaction(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		session := xormDB.NewSession()
		defer session.Close()

		if err := session.Begin(); err != nil {
			b.Fatalf("开启事务失败: %v", err)
		}

		newUsers := []XUser{
			{Name: fmt.Sprintf("赵六_%d", i), Age: 23},
		}
		if _, err := session.Insert(&newUsers); err != nil {
			session.Rollback()
			b.Fatalf("事务中批量插入失败: %v", err)
		}

		if err := session.Commit(); err != nil {
			b.Fatalf("事务提交失败: %v", err)
		}
	}
}
