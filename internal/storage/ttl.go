package storage

import (
	"sync"
	"time"
)

// TTLManager 管理过期键的定期清理
//
// TTLManager 运行一个后台 Goroutine，定期扫描并清理过期的键。
// 这与 Get 方法中的惰性删除配合使用，确保过期键能够及时清理。
type TTLManager struct {
	sm              *ShardedMap   // 要管理的 ShardedMap
	cleanupInterval time.Duration // 清理间隔
	keysPerScan     int           // 每次扫描清理的键数
	stopCh          chan struct{} // 停止信号
	wg              sync.WaitGroup
	running         bool
	mu              sync.Mutex
}

// TTLManagerConfig TTL 管理器配置
type TTLManagerConfig struct {
	// CleanupInterval 清理间隔，默认 1 秒
	CleanupInterval time.Duration

	// KeysPerScan 每次扫描清理的键数，默认 100
	// 设置过大可能影响性能，设置过小可能导致清理不及时
	KeysPerScan int
}

// DefaultTTLManagerConfig 返回默认的 TTL 管理器配置
func DefaultTTLManagerConfig() *TTLManagerConfig {
	return &TTLManagerConfig{
		CleanupInterval: 1 * time.Second,
		KeysPerScan:     100,
	}
}

// NewTTLManager 创建一个新的 TTL 管理器
//
// 参数说明：
//   - sm: 要管理的 ShardedMap
//   - config: TTL 管理器配置，如果为 nil 则使用默认配置
//
// 返回值：
//   - *TTLManager: TTL 管理器实例
//
// 示例：
//
//	sm := NewShardedMap(4096)
//	ttlMgr := NewTTLManager(sm, nil) // 使用默认配置
//	ttlMgr.Start()
//	defer ttlMgr.Stop()
//
// 注意事项：
//   - 需要调用 Start() 启动清理任务
//   - 使用完毕后应调用 Stop() 停止清理任务
func NewTTLManager(sm *ShardedMap, config *TTLManagerConfig) *TTLManager {
	if config == nil {
		config = DefaultTTLManagerConfig()
	}

	return &TTLManager{
		sm:              sm,
		cleanupInterval: config.CleanupInterval,
		keysPerScan:     config.KeysPerScan,
		stopCh:          make(chan struct{}),
		running:         false,
	}
}

// Start 启动 TTL 清理任务
//
// 启动一个后台 Goroutine 定期扫描并清理过期的键。
//
// 注意事项：
//   - 重复调用 Start 不会启动多个清理任务
//   - 非阻塞调用
func (tm *TTLManager) Start() {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	if tm.running {
		return // 已经在运行
	}

	tm.running = true
	tm.stopCh = make(chan struct{})

	tm.wg.Add(1)
	go tm.cleanupLoop()
}

// Stop 停止 TTL 清理任务
//
// 停止后台清理 Goroutine 并等待其退出。
//
// 注意事项：
//   - 调用 Stop 后会阻塞直到清理任务完全停止
//   - 多次调用 Stop 是安全的
func (tm *TTLManager) Stop() {
	tm.mu.Lock()
	if !tm.running {
		tm.mu.Unlock()
		return // 未在运行
	}

	close(tm.stopCh)
	tm.running = false
	tm.mu.Unlock()

	// 等待清理任务退出
	tm.wg.Wait()
}

// IsRunning 检查 TTL 管理器是否正在运行
func (tm *TTLManager) IsRunning() bool {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	return tm.running
}

// cleanupLoop 清理循环
func (tm *TTLManager) cleanupLoop() {
	defer tm.wg.Done()

	ticker := time.NewTicker(tm.cleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-tm.stopCh:
			return
		case <-ticker.C:
			tm.cleanup()
		}
	}
}

// cleanup 执行一次清理操作
//
// 遍历所有分片，从每个分片中随机选择一些键检查是否过期，
// 如果过期则删除。这种采样方式既能及时清理过期键，
// 又不会因为扫描所有键而影响性能。
func (tm *TTLManager) cleanup() {
	now := time.Now().Unix()
	keysPerShard := tm.keysPerScan / DefaultShardCount
	if keysPerShard < 1 {
		keysPerShard = 1
	}

	for i := 0; i < DefaultShardCount; i++ {
		tm.cleanupShard(tm.sm.shards[i], keysPerShard, now)
	}
}

// cleanupShard 清理单个分片中的过期键
func (tm *TTLManager) cleanupShard(shard *mapShard, maxKeys int, now int64) {
	shard.mu.Lock()
	defer shard.mu.Unlock()

	// 收集过期的键
	expiredKeys := make([]string, 0, maxKeys)
	count := 0

	for key, item := range shard.items {
		// 检查是否过期
		if item.expiresAt > 0 && now >= item.expiresAt {
			expiredKeys = append(expiredKeys, key)
			count++

			// 达到最大清理数量，停止扫描
			if count >= maxKeys {
				break
			}
		}

		// 达到最大扫描数量，停止扫描
		if count >= maxKeys*2 {
			break
		}
	}

	// 删除过期的键
	for _, key := range expiredKeys {
		delete(shard.items, key)
	}
}

// GetStats 获取 TTL 管理器的统计信息
type TTLStats struct {
	Running         bool          // 是否正在运行
	CleanupInterval time.Duration // 清理间隔
	KeysPerScan     int           // 每次扫描的键数
}

// GetStats 返回 TTL 管理器的统计信息
func (tm *TTLManager) GetStats() TTLStats {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	return TTLStats{
		Running:         tm.running,
		CleanupInterval: tm.cleanupInterval,
		KeysPerScan:     tm.keysPerScan,
	}
}

// TTL 获取键的剩余生存时间（秒）
//
// 参数说明：
//   - key: 要查询的键
//
// 返回值：
//   - int64: 剩余 TTL（秒），-1 表示键不存在，-2 表示永不过期
//
// 示例：
//
//	ttl := TTL(sm, "session:abc123")
//	if ttl == -1 {
//	    log.Println("键不存在")
//	} else if ttl == -2 {
//	    log.Println("永不过期")
//	} else {
//	    log.Printf("剩余 %d 秒", ttl)
//	}
func TTL(sm *ShardedMap, key string) int64 {
	shard := sm.getShard(key)

	shard.mu.RLock()
	defer shard.mu.RUnlock()

	item, exists := shard.items[key]
	if !exists {
		return -1 // 键不存在
	}

	if item.expiresAt == 0 {
		return -2 // 永不过期
	}

	now := time.Now().Unix()
	remaining := item.expiresAt - now

	if remaining <= 0 {
		return 0 // 已过期或即将过期
	}

	return remaining
}

// Expire 更新键的过期时间
//
// 参数说明：
//   - key: 要更新的键
//   - ttl: 新的生存时间（秒），0 表示永不过期
//
// 返回值：
//   - bool: 是否成功更新（false 表示键不存在）
//
// 示例：
//
//	// 将会话有效期延长到 1 小时
//	updated := Expire(sm, "session:abc123", 3600)
//	if !updated {
//	    log.Println("会话不存在")
//	}
func Expire(sm *ShardedMap, key string, ttl int) bool {
	shard := sm.getShard(key)

	shard.mu.Lock()
	defer shard.mu.Unlock()

	item, exists := shard.items[key]
	if !exists {
		return false
	}

	if ttl > 0 {
		item.expiresAt = time.Now().Unix() + int64(ttl)
	} else {
		item.expiresAt = 0 // 永不过期
	}

	return true
}
