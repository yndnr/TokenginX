package storage

import (
	"testing"
	"time"
)

// TestTTLManager_StartStop 测试启动和停止
func TestTTLManager_StartStop(t *testing.T) {
	sm := NewShardedMap(1024)
	ttlMgr := NewTTLManager(sm, nil)

	if ttlMgr.IsRunning() {
		t.Error("TTLManager should not be running initially")
	}

	ttlMgr.Start()
	if !ttlMgr.IsRunning() {
		t.Error("TTLManager should be running after Start")
	}

	ttlMgr.Stop()
	if ttlMgr.IsRunning() {
		t.Error("TTLManager should not be running after Stop")
	}

	// 测试重复 Start/Stop
	ttlMgr.Start()
	ttlMgr.Start() // 重复 Start 应该是安全的
	ttlMgr.Stop()
	ttlMgr.Stop() // 重复 Stop 应该是安全的
}

// TestTTLManager_PeriodicCleanup 测试定期清理
func TestTTLManager_PeriodicCleanup(t *testing.T) {
	sm := NewShardedMap(1024)

	config := &TTLManagerConfig{
		CleanupInterval: 100 * time.Millisecond,
		KeysPerScan:     100,
	}
	ttlMgr := NewTTLManager(sm, config)
	ttlMgr.Start()
	defer ttlMgr.Stop()

	// 插入一些很快过期的键
	for i := 0; i < 50; i++ {
		key := "expiring_key" + string(rune(i))
		sm.Set(key, i, 1) // 1 秒过期
	}

	// 插入一些永不过期的键
	for i := 0; i < 50; i++ {
		key := "permanent_key" + string(rune(i))
		sm.Set(key, i, 0) // 永不过期
	}

	initialLen := sm.Len()
	if initialLen != 100 {
		t.Errorf("Expected initial length 100, got %d", initialLen)
	}

	// 等待过期和清理
	time.Sleep(1500 * time.Millisecond)

	// 过期的键应该被清理
	finalLen := sm.Len()
	if finalLen > 50 {
		t.Errorf("Expected final length <= 50, got %d", finalLen)
	}
}

// TestTTLManager_CleanupPerformance 测试清理性能
func TestTTLManager_CleanupPerformance(t *testing.T) {
	sm := NewShardedMap(4096)

	config := &TTLManagerConfig{
		CleanupInterval: 50 * time.Millisecond,
		KeysPerScan:     1000,
	}
	ttlMgr := NewTTLManager(sm, config)
	ttlMgr.Start()
	defer ttlMgr.Stop()

	// 插入大量键
	count := 10000
	for i := 0; i < count; i++ {
		key := "key" + string(rune(i))
		sm.Set(key, i, 0)
	}

	// 添加一些很快过期的键
	for i := 0; i < 1000; i++ {
		key := "expiring" + string(rune(i))
		sm.Set(key, i, 1)
	}

	// 等待清理（清理不应该显著影响性能）
	time.Sleep(2 * time.Second)

	// 验证永久键仍然存在
	for i := 0; i < 100; i++ {
		key := "key" + string(rune(i))
		if !sm.Exists(key) {
			t.Errorf("Permanent key %s should exist", key)
		}
	}
}

// TestTTL 测试 TTL 函数
func TestTTL(t *testing.T) {
	sm := NewShardedMap(1024)

	// 测试不存在的键
	ttl := TTL(sm, "nonexistent")
	if ttl != -1 {
		t.Errorf("Expected TTL -1 for nonexistent key, got %d", ttl)
	}

	// 测试永不过期的键
	sm.Set("permanent", "value", 0)
	ttl = TTL(sm, "permanent")
	if ttl != -2 {
		t.Errorf("Expected TTL -2 for permanent key, got %d", ttl)
	}

	// 测试有 TTL 的键
	sm.Set("expiring", "value", 10) // 10 秒过期
	ttl = TTL(sm, "expiring")
	if ttl < 9 || ttl > 10 {
		t.Errorf("Expected TTL around 10, got %d", ttl)
	}

	// 等待一段时间
	time.Sleep(2 * time.Second)
	ttl = TTL(sm, "expiring")
	if ttl < 7 || ttl > 9 {
		t.Errorf("Expected TTL around 8, got %d", ttl)
	}
}

// TestExpire 测试 Expire 函数
func TestExpire(t *testing.T) {
	sm := NewShardedMap(1024)

	// 测试不存在的键
	updated := Expire(sm, "nonexistent", 100)
	if updated {
		t.Error("Expire should return false for nonexistent key")
	}

	// 测试更新已存在的键
	sm.Set("key1", "value1", 10) // 原本 10 秒过期

	ttl := TTL(sm, "key1")
	if ttl < 9 || ttl > 10 {
		t.Errorf("Expected initial TTL around 10, got %d", ttl)
	}

	// 延长到 100 秒
	updated = Expire(sm, "key1", 100)
	if !updated {
		t.Error("Expire should return true for existing key")
	}

	ttl = TTL(sm, "key1")
	if ttl < 99 || ttl > 100 {
		t.Errorf("Expected updated TTL around 100, got %d", ttl)
	}

	// 设置为永不过期
	Expire(sm, "key1", 0)
	ttl = TTL(sm, "key1")
	if ttl != -2 {
		t.Errorf("Expected TTL -2 after setting to permanent, got %d", ttl)
	}
}

// TestExpire_AfterExpiry 测试过期后更新 TTL
func TestExpire_AfterExpiry(t *testing.T) {
	sm := NewShardedMap(1024)

	sm.Set("key1", "value1", 1) // 1 秒过期

	// 等待过期
	time.Sleep(1100 * time.Millisecond)

	// 尝试更新过期键的 TTL（应该失败，因为键在 Get 时会被删除）
	sm.Get("key1") // 触发惰性删除

	updated := Expire(sm, "key1", 100)
	if updated {
		t.Error("Expire should fail for expired key")
	}
}

// TestTTLManager_GetStats 测试统计信息
func TestTTLManager_GetStats(t *testing.T) {
	sm := NewShardedMap(1024)

	config := &TTLManagerConfig{
		CleanupInterval: 500 * time.Millisecond,
		KeysPerScan:     200,
	}
	ttlMgr := NewTTLManager(sm, config)

	stats := ttlMgr.GetStats()
	if stats.Running {
		t.Error("Stats should show not running initially")
	}
	if stats.CleanupInterval != 500*time.Millisecond {
		t.Errorf("Expected interval 500ms, got %v", stats.CleanupInterval)
	}
	if stats.KeysPerScan != 200 {
		t.Errorf("Expected KeysPerScan 200, got %d", stats.KeysPerScan)
	}

	ttlMgr.Start()
	defer ttlMgr.Stop()

	stats = ttlMgr.GetStats()
	if !stats.Running {
		t.Error("Stats should show running after Start")
	}
}

// TestTTLManager_DefaultConfig 测试默认配置
func TestTTLManager_DefaultConfig(t *testing.T) {
	config := DefaultTTLManagerConfig()

	if config.CleanupInterval != 1*time.Second {
		t.Errorf("Expected default interval 1s, got %v", config.CleanupInterval)
	}

	if config.KeysPerScan != 100 {
		t.Errorf("Expected default KeysPerScan 100, got %d", config.KeysPerScan)
	}
}

// BenchmarkTTL 基准测试：TTL 查询
func BenchmarkTTL(b *testing.B) {
	sm := NewShardedMap(4096)
	sm.Set("key", "value", 3600)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		TTL(sm, "key")
	}
}

// BenchmarkExpire 基准测试：Expire 更新
func BenchmarkExpire(b *testing.B) {
	sm := NewShardedMap(4096)
	sm.Set("key", "value", 3600)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Expire(sm, "key", 7200)
	}
}
