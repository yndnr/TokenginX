package storage

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

// TestNewShardedMap 测试构造函数
func TestNewShardedMap(t *testing.T) {
	sm := NewShardedMap(1024)
	if sm == nil {
		t.Fatal("NewShardedMap returned nil")
	}

	if len(sm.shards) != DefaultShardCount {
		t.Errorf("Expected %d shards, got %d", DefaultShardCount, len(sm.shards))
	}

	if sm.Len() != 0 {
		t.Errorf("Expected empty map, got length %d", sm.Len())
	}
}

// TestShardedMap_SetGet 测试设置和获取
func TestShardedMap_SetGet(t *testing.T) {
	sm := NewShardedMap(1024)

	// 测试基本的 Set/Get
	err := sm.Set("key1", "value1", 0)
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	value, found := sm.Get("key1")
	if !found {
		t.Fatal("Key not found")
	}

	if value != "value1" {
		t.Errorf("Expected 'value1', got '%v'", value)
	}
}

// TestShardedMap_SetOverwrite 测试覆盖已存在的键
func TestShardedMap_SetOverwrite(t *testing.T) {
	sm := NewShardedMap(1024)

	sm.Set("key1", "value1", 0)
	sm.Set("key1", "value2", 0)

	value, found := sm.Get("key1")
	if !found {
		t.Fatal("Key not found")
	}

	if value != "value2" {
		t.Errorf("Expected 'value2', got '%v'", value)
	}
}

// TestShardedMap_GetNonExistent 测试获取不存在的键
func TestShardedMap_GetNonExistent(t *testing.T) {
	sm := NewShardedMap(1024)

	value, found := sm.Get("nonexistent")
	if found {
		t.Error("Expected key not found, but it was found")
	}

	if value != nil {
		t.Errorf("Expected nil value, got %v", value)
	}
}

// TestShardedMap_Delete 测试删除
func TestShardedMap_Delete(t *testing.T) {
	sm := NewShardedMap(1024)

	sm.Set("key1", "value1", 0)

	// 删除存在的键
	deleted := sm.Delete("key1")
	if !deleted {
		t.Error("Expected Delete to return true")
	}

	// 验证键已被删除
	_, found := sm.Get("key1")
	if found {
		t.Error("Key should have been deleted")
	}

	// 删除不存在的键
	deleted = sm.Delete("nonexistent")
	if deleted {
		t.Error("Expected Delete to return false for non-existent key")
	}
}

// TestShardedMap_Exists 测试存在性检查
func TestShardedMap_Exists(t *testing.T) {
	sm := NewShardedMap(1024)

	sm.Set("key1", "value1", 0)

	if !sm.Exists("key1") {
		t.Error("Expected key1 to exist")
	}

	if sm.Exists("nonexistent") {
		t.Error("Expected nonexistent key to not exist")
	}
}

// TestShardedMap_TTL 测试 TTL 过期
func TestShardedMap_TTL(t *testing.T) {
	sm := NewShardedMap(1024)

	// 设置 1 秒过期
	sm.Set("key1", "value1", 1)

	// 立即获取应该成功
	value, found := sm.Get("key1")
	if !found {
		t.Fatal("Key should exist immediately after Set")
	}
	if value != "value1" {
		t.Errorf("Expected 'value1', got '%v'", value)
	}

	// 等待过期
	time.Sleep(1100 * time.Millisecond)

	// 应该已过期
	_, found = sm.Get("key1")
	if found {
		t.Error("Key should have expired")
	}

	// Exists 也应该返回 false
	if sm.Exists("key1") {
		t.Error("Expired key should not exist")
	}
}

// TestShardedMap_ZeroTTL 测试 TTL=0（永不过期）
func TestShardedMap_ZeroTTL(t *testing.T) {
	sm := NewShardedMap(1024)

	sm.Set("key1", "value1", 0)

	// 等待一段时间
	time.Sleep(100 * time.Millisecond)

	// 应该仍然存在
	_, found := sm.Get("key1")
	if !found {
		t.Error("Key with TTL=0 should never expire")
	}
}

// TestShardedMap_Len 测试长度统计
func TestShardedMap_Len(t *testing.T) {
	sm := NewShardedMap(1024)

	if sm.Len() != 0 {
		t.Errorf("Expected length 0, got %d", sm.Len())
	}

	sm.Set("key1", "value1", 0)
	sm.Set("key2", "value2", 0)
	sm.Set("key3", "value3", 0)

	if sm.Len() != 3 {
		t.Errorf("Expected length 3, got %d", sm.Len())
	}

	sm.Delete("key1")

	if sm.Len() != 2 {
		t.Errorf("Expected length 2 after delete, got %d", sm.Len())
	}
}

// TestShardedMap_Clear 测试清空
func TestShardedMap_Clear(t *testing.T) {
	sm := NewShardedMap(1024)

	sm.Set("key1", "value1", 0)
	sm.Set("key2", "value2", 0)

	sm.Clear()

	if sm.Len() != 0 {
		t.Errorf("Expected length 0 after Clear, got %d", sm.Len())
	}

	_, found := sm.Get("key1")
	if found {
		t.Error("Key should not exist after Clear")
	}
}

// TestShardedMap_ManyKeys 测试大量键
func TestShardedMap_ManyKeys(t *testing.T) {
	sm := NewShardedMap(4096)

	// 插入 10000 个键
	count := 10000
	for i := 0; i < count; i++ {
		key := fmt.Sprintf("key%d", i)
		sm.Set(key, i, 0)
	}

	if sm.Len() != count {
		t.Errorf("Expected length %d, got %d", count, sm.Len())
	}

	// 验证所有键都能找到
	for i := 0; i < count; i++ {
		key := fmt.Sprintf("key%d", i)
		value, found := sm.Get(key)
		if !found {
			t.Errorf("Key %s not found", key)
		}
		if value != i {
			t.Errorf("Expected value %d, got %v", i, value)
		}
	}
}

// TestShardedMap_ConcurrentSet 测试并发写入
func TestShardedMap_ConcurrentSet(t *testing.T) {
	sm := NewShardedMap(1024)
	var wg sync.WaitGroup

	goroutines := 100
	keysPerGoroutine := 100

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			for j := 0; j < keysPerGoroutine; j++ {
				key := fmt.Sprintf("key%d_%d", n, j)
				sm.Set(key, j, 0)
			}
		}(i)
	}

	wg.Wait()

	expected := goroutines * keysPerGoroutine
	if sm.Len() != expected {
		t.Errorf("Expected length %d, got %d", expected, sm.Len())
	}
}

// TestShardedMap_ConcurrentGet 测试并发读取
func TestShardedMap_ConcurrentGet(t *testing.T) {
	sm := NewShardedMap(1024)

	// 预先插入数据
	count := 1000
	for i := 0; i < count; i++ {
		key := fmt.Sprintf("key%d", i)
		sm.Set(key, i, 0)
	}

	var wg sync.WaitGroup
	goroutines := 100

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < count; j++ {
				key := fmt.Sprintf("key%d", j)
				_, found := sm.Get(key)
				if !found {
					t.Errorf("Key %s not found during concurrent read", key)
				}
			}
		}()
	}

	wg.Wait()
}

// TestShardedMap_ConcurrentMixed 测试混合读写
func TestShardedMap_ConcurrentMixed(t *testing.T) {
	sm := NewShardedMap(1024)
	var wg sync.WaitGroup

	// 并发写入
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 1000; i++ {
			key := fmt.Sprintf("write_key%d", i)
			sm.Set(key, i, 0)
		}
	}()

	// 并发读取
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 1000; i++ {
			key := fmt.Sprintf("read_key%d", i%100)
			sm.Get(key)
		}
	}()

	// 并发删除
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 1000; i++ {
			key := fmt.Sprintf("delete_key%d", i%100)
			sm.Delete(key)
		}
	}()

	wg.Wait()
}

// TestShardedMap_DifferentTypes 测试不同类型的值
func TestShardedMap_DifferentTypes(t *testing.T) {
	sm := NewShardedMap(1024)

	// 字符串
	sm.Set("string_key", "value", 0)
	if v, found := sm.Get("string_key"); !found || v != "value" {
		t.Error("String value failed")
	}

	// 整数
	sm.Set("int_key", 42, 0)
	if v, found := sm.Get("int_key"); !found || v != 42 {
		t.Error("Int value failed")
	}

	// 结构体
	type TestStruct struct {
		Name string
		Age  int
	}
	testStruct := TestStruct{Name: "Alice", Age: 30}
	sm.Set("struct_key", testStruct, 0)
	if v, found := sm.Get("struct_key"); !found {
		t.Error("Struct value not found")
	} else if v != testStruct {
		t.Error("Struct value mismatch")
	}

	// Map
	testMap := map[string]int{"a": 1, "b": 2}
	sm.Set("map_key", testMap, 0)
	if v, found := sm.Get("map_key"); !found {
		t.Error("Map value not found")
	} else {
		vm := v.(map[string]int)
		if vm["a"] != 1 || vm["b"] != 2 {
			t.Error("Map value mismatch")
		}
	}
}

// BenchmarkShardedMap_Set 基准测试：Set 操作
func BenchmarkShardedMap_Set(b *testing.B) {
	sm := NewShardedMap(4096)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("key%d", i)
		sm.Set(key, i, 0)
	}
}

// BenchmarkShardedMap_Get 基准测试：Get 操作
func BenchmarkShardedMap_Get(b *testing.B) {
	sm := NewShardedMap(4096)

	// 预先插入数据
	for i := 0; i < 10000; i++ {
		key := fmt.Sprintf("key%d", i)
		sm.Set(key, i, 0)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("key%d", i%10000)
		sm.Get(key)
	}
}

// BenchmarkShardedMap_Delete 基准测试：Delete 操作
func BenchmarkShardedMap_Delete(b *testing.B) {
	sm := NewShardedMap(4096)

	// 预先插入数据
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("key%d", i)
		sm.Set(key, i, 0)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("key%d", i)
		sm.Delete(key)
	}
}

// BenchmarkShardedMap_Parallel 基准测试：并发操作
func BenchmarkShardedMap_Parallel(b *testing.B) {
	sm := NewShardedMap(4096)

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			key := fmt.Sprintf("key%d", i)
			sm.Set(key, i, 0)
			sm.Get(key)
			i++
		}
	})
}
