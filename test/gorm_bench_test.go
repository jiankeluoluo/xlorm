package test

import (
	"fmt"
	"log"
	"testing"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var gormDB *gorm.DB

type GUser struct {
	ID   int    `gorm:"column:id;primaryKey"`
	Name string `gorm:"column:name"`
	Age  int    `gorm:"column:age"`
}

func init() {
	dsn := "root:770880@tcp(localhost:3306)/test_db?charset=utf8mb4&parseTime=True&loc=Local"
	var err error
	gormDB, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal(err)
	}
}

func BenchmarkGORM_Insert(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		user := GUser{Name: fmt.Sprintf("测试用户_%d", i), Age: 25}
		if err := gormDB.Create(&user).Error; err != nil {
			b.Fatalf("插入失败: %v", err)
		}
	}
}

func BenchmarkGORM_BatchInsert(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		users := []GUser{
			{Name: fmt.Sprintf("用户1_%d", i), Age: 20},
			{Name: fmt.Sprintf("用户2_%d", i), Age: 21},
			{Name: fmt.Sprintf("用户3_%d", i), Age: 22},
		}
		if err := gormDB.CreateInBatches(users, 500).Error; err != nil {
			b.Fatalf("批量插入失败: %v", err)
		}
	}
}

func BenchmarkGORM_Find(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var user GUser
		if err := gormDB.Select("id, name, age").Where("id = ?", 2).First(&user).Error; err != nil {
			b.Fatalf("查询失败: %v", err)
		}
	}
}

func BenchmarkGORM_FindAll(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var users []GUser
		if err := gormDB.Where("age > ?", 30).Order("age DESC").Find(&users).Error; err != nil {
			b.Fatalf("查询失败: %v", err)
		}
	}
}

func BenchmarkGORM_Update(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := gormDB.Model(&GUser{}).Where("id = ?", 1).Update("age", 26+i%10).Error; err != nil {
			b.Fatalf("更新失败: %v", err)
		}
	}
}

func BenchmarkGORM_BatchUpdate(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		updates := []GUser{
			{ID: 1, Name: fmt.Sprintf("张三(更新_%d)", i), Age: 30 + i%10},
			{ID: 2, Name: fmt.Sprintf("李四(更新_%d)", i), Age: 31 + i%10},
		}
		for _, user := range updates {
			if err := gormDB.Save(&user).Error; err != nil {
				b.Fatalf("批量更新失败: %v", err)
			}
		}
	}
}

func BenchmarkGORM_Delete(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := gormDB.Where("id = ?", 1+i%10).Delete(&GUser{}).Error; err != nil {
			b.Fatalf("删除失败: %v", err)
		}
	}
}

func BenchmarkGORM_Transaction(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := gormDB.Transaction(func(tx *gorm.DB) error {
			newUsers := []GUser{
				{Name: fmt.Sprintf("赵六_%d", i), Age: 23},
			}
			if err := tx.Create(&newUsers).Error; err != nil {
				return err
			}
			return nil
		})
		if err != nil {
			b.Fatalf("事务失败: %v", err)
		}
	}
}
