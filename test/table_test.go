package test

import (
	"context"
	"testing"
	"time"
)

// TestTableWithContext 测试上下文设置
func TestTableWithContext(t *testing.T) {
	table := getTestTable(t)

	// 创建一个带有超时的上下文
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 设置上下文
	tableWithCtx := table.WithContext(ctx)

	// 验证上下文是否正确设置
	if tableWithCtx == nil {
		t.Error("上下文设置失败")
	}
}

// TestTableInsert 测试单条记录插入
func TestTableInsert(t *testing.T) {
	table := getTestTable(t)

	testUser := map[string]interface{}{
		"name": "test_user",
		"age":  25,
	}

	id, err := table.Insert(testUser)
	if err != nil {
		t.Fatalf("插入数据失败: %v", err)
	}

	if id == 0 {
		t.Errorf("插入影响行数不正确，期望: 1, 实际: %d", 0)
	}

	// 验证插入的数据
	user, err := table.Where("id = ?", id).Find()
	if err != nil {
		t.Fatalf("查询插入的数据失败: %v", err)
	}

	if user["name"] != testUser["name"] || user["age"] != testUser["age"] {
		t.Errorf("插入的数据与查询结果不一致")
	}
}

// TestTableUpdate 测试记录更新
func TestTableUpdate(t *testing.T) {
	table := getTestTable(t)

	// 先插入测试数据
	testUser := map[string]interface{}{
		"name": "initial_user",
		"age":  25,
	}
	id, err := table.Insert(testUser)
	if err != nil {
		t.Fatalf("插入初始数据失败: %v", err)
	}

	// 更新数据
	updateData := map[string]interface{}{
		"name": "updated_user",
		"age":  30,
	}
	affected, err := table.Where("id = ?", id).Update(updateData)
	if err != nil {
		t.Fatalf("更新数据失败: %v", err)
	}

	if affected != 1 {
		t.Errorf("更新影响行数不正确，期望: 1, 实际: %d", affected)
	}

	// 验证更新后的数据
	updatedUser, err := table.Where("id = ?", id).Find()
	if err != nil {
		t.Fatalf("查询更新后的数据失败: %v", err)
	}

	if updatedUser["name"] != updateData["name"] || updatedUser["age"] != updateData["age"] {
		t.Errorf("更新的数据与查询结果不一致")
	}
}

// TestTableDelete 测试记录删除
func TestTableDelete(t *testing.T) {
	table := getTestTable(t)

	// 先插入测试数据
	testUser := map[string]interface{}{
		"name": "to_be_deleted_user",
		"age":  25,
	}
	id, err := table.Insert(testUser)
	if err != nil {
		t.Fatalf("插入初始数据失败: %v", err)
	}

	// 删除数据
	affected, err := table.Where("id = ?", id).Delete()
	if err != nil {
		t.Fatalf("删除数据失败: %v", err)
	}

	if affected != 1 {
		t.Errorf("删除影响行数不正确，期望: 1, 实际: %d", affected)
	}

	// 验证数据是否已删除
	deletedUser, err := table.Where("id = ?", id).Find()
	if err == nil {
		t.Errorf("删除的数据仍然存在: %v", deletedUser)
	}
}

// TestTableSelect 测试查询功能
func TestTableSelect(t *testing.T) {
	table := getTestTable(t)

	// 插入多条测试数据
	testUsers := []map[string]interface{}{
		{"name": "user1", "age": 25},
		{"name": "user2", "age": 30},
		{"name": "user3", "age": 35},
	}
	_, err := table.BatchInsert(testUsers, 500)
	if err != nil {
		t.Fatalf("插入测试数据失败: %v", err)
	}

	// 测试条件查询
	users, err := table.Where("age > ?", 25).FindAll()
	if err != nil {
		t.Fatalf("条件查询失败: %v", err)
	}
	if len(users) != 2 {
		t.Errorf("条件查询结果不正确，期望: 2, 实际: %d", len(users))
	}

	// 测试排序查询
	users, err = table.OrderBy("age DESC").FindAll()
	if err != nil {
		t.Fatalf("排序查询失败: %v", err)
	}
	if len(users) != 3 || users[0]["age"].(int) != 35 {
		t.Errorf("排序查询结果不正确")
	}
}

// TestTableCount 测试记录计数
func TestTableCount(t *testing.T) {
	table := getTestTable(t)

	// 插入多条测试数据
	testUsers := []map[string]interface{}{
		{"name": "user1", "age": 25},
		{"name": "user2", "age": 30},
		{"name": "user3", "age": 35},
	}
	_, err := table.BatchInsert(testUsers, 500)
	if err != nil {
		t.Fatalf("插入测试数据失败: %v", err)
	}

	// 测试总数查询
	total, err := table.Count()
	if err != nil {
		t.Fatalf("获取总数失败: %v", err)
	}
	if total != 3 {
		t.Errorf("总数查询结果不正确，期望: 3, 实际: %d", total)
	}

	// 测试条件总数查询
	total, err = table.Where("age > ?", 25).Count()
	if err != nil {
		t.Fatalf("条件总数查询失败: %v", err)
	}
	if total != 2 {
		t.Errorf("条件总数查询结果不正确，期望: 2, 实际: %d", total)
	}
}
