package storage

import (
	"hash/fnv"
	"sync"
	"time"
)

const (
	// DefaultShardCount 是分片哈希表的默认分片数量
	//
	// 使用 256 个分片可以在大多数场景下提供良好的并发性能，
	// 同时避免过多分片导致的内存开销。
	DefaultShardCount = 256

	// DefaultInitialCapacity 是每个分片的默认初始容量
	DefaultInitialCapacity = 4096
)

// item 表示存储的单个数据项
type item struct {
	value     interface{} // 存储的值
	expiresAt int64       // 过期时间戳（Unix 秒），0 表示永不过期
	createdAt int64       // 创建时间戳
}

// mapShard 表示单个分片
//
// 每个分片独立管理一部分数据，使用独立的读写锁来减少锁竞争。
type mapShard struct {
	mu    sync.RWMutex       // 读写锁，保证并发安全
	items map[string]*item   // 存储的键值对
}

// ShardedMap 是一个线程安全的分片哈希表，用于高并发场景下的键值存储
//
// ShardedMap 使用 256 个分片来减少锁竞争，每个分片独立管理一部分数据。
// 通过哈希函数将键分配到不同的分片，从而实现并发访问时的性能优化。
//
// 示例：
//
//	sm := NewShardedMap(4096)
//	sm.Set("key1", "value1", 3600) // 设置 1 小时后过期
//	value, found := sm.Get("key1")
//	if found {
//	    fmt.Println(value)
//	}
type ShardedMap struct {
	shards [DefaultShardCount]*mapShard // 256 个分片
}

// NewShardedMap 创建一个新的分片哈希表实例
//
// 参数说明：
//   - initialCapacity: 每个分片的初始容量，建议设置为预期总容量的 1/256
//
// 返回值：
//   - *ShardedMap: 分片哈希表实例
//
// 示例：
//
//	// 创建一个预期存储 100 万个键值对的分片哈希表
//	sm := NewShardedMap(4096) // 4096 * 256 ≈ 1,000,000
//
// 注意事项：
//   - initialCapacity 为 0 时使用默认容量
//   - 该方法是并发安全的
func NewShardedMap(initialCapacity int) *ShardedMap {
	if initialCapacity <= 0 {
		initialCapacity = DefaultInitialCapacity
	}

	sm := &ShardedMap{}
	for i := 0; i < DefaultShardCount; i++ {
		sm.shards[i] = &mapShard{
			items: make(map[string]*item, initialCapacity),
		}
	}

	return sm
}

// getShard 根据键的哈希值返回对应的分片索引
//
// 使用 FNV-1a 哈希算法将键映射到 0-255 的分片索引。
func (sm *ShardedMap) getShard(key string) *mapShard {
	h := fnv.New32a()
	h.Write([]byte(key))
	index := h.Sum32() % DefaultShardCount
	return sm.shards[index]
}

// Set 在分片哈希表中设置键值对，并指定过期时间
//
// 参数说明：
//   - key: 要设置的键
//   - value: 要设置的值
//   - ttl: 生存时间（秒），0 表示永不过期
//
// 返回值：
//   - error: 错误信息，nil 表示成功
//
// 示例：
//
//	// 设置一个 1 小时后过期的会话
//	err := sm.Set("session:abc123", sessionData, 3600)
//	if err != nil {
//	    log.Printf("设置会话失败: %v", err)
//	}
//
// 注意事项：
//   - 该方法是并发安全的
//   - 如果 key 已存在，将覆盖旧值
//   - ttl 使用惰性删除 + 定期清理策略
func (sm *ShardedMap) Set(key string, value interface{}, ttl int) error {
	shard := sm.getShard(key)

	shard.mu.Lock()
	defer shard.mu.Unlock()

	now := time.Now().Unix()
	var expiresAt int64
	if ttl > 0 {
		expiresAt = now + int64(ttl)
	}

	shard.items[key] = &item{
		value:     value,
		expiresAt: expiresAt,
		createdAt: now,
	}

	return nil
}

// Get 从分片哈希表中获取指定键的值
//
// 参数说明：
//   - key: 要获取的键
//
// 返回值：
//   - interface{}: 键对应的值
//   - bool: 是否找到该键（true 表示找到且未过期）
//
// 示例：
//
//	value, found := sm.Get("session:abc123")
//	if !found {
//	    log.Println("会话不存在或已过期")
//	    return
//	}
//	session := value.(*SessionData)
//
// 注意事项：
//   - 该方法是并发安全的
//   - 访问时会检查 TTL，如果已过期则删除并返回 false（惰性删除）
//   - 类型断言需要调用方自行处理
func (sm *ShardedMap) Get(key string) (interface{}, bool) {
	shard := sm.getShard(key)

	shard.mu.Lock()
	defer shard.mu.Unlock()

	item, exists := shard.items[key]
	if !exists {
		return nil, false
	}

	// 检查是否过期（惰性删除）
	if item.expiresAt > 0 && time.Now().Unix() >= item.expiresAt {
		// 删除过期的键
		delete(shard.items, key)
		return nil, false
	}

	return item.value, true
}

// Delete 从分片哈希表中删除指定的键
//
// 参数说明：
//   - key: 要删除的键
//
// 返回值：
//   - bool: 是否成功删除（true 表示键存在并被删除）
//
// 示例：
//
//	deleted := sm.Delete("session:abc123")
//	if deleted {
//	    log.Println("会话已删除")
//	}
//
// 注意事项：
//   - 该方法是并发安全的
//   - 如果键不存在，返回 false
func (sm *ShardedMap) Delete(key string) bool {
	shard := sm.getShard(key)

	shard.mu.Lock()
	defer shard.mu.Unlock()

	_, exists := shard.items[key]
	if !exists {
		return false
	}

	delete(shard.items, key)
	return true
}

// Exists 检查指定的键是否存在且未过期
//
// 参数说明：
//   - key: 要检查的键
//
// 返回值：
//   - bool: 键是否存在且未过期
//
// 示例：
//
//	if sm.Exists("session:abc123") {
//	    log.Println("会话存在")
//	}
//
// 注意事项：
//   - 该方法是并发安全的
//   - 会检查 TTL，过期的键返回 false
func (sm *ShardedMap) Exists(key string) bool {
	_, exists := sm.Get(key)
	return exists
}

// Len 返回分片哈希表中的键值对总数
//
// 返回值：
//   - int: 键值对总数（包括已过期但未清理的键）
//
// 示例：
//
//	count := sm.Len()
//	log.Printf("当前存储了 %d 个键值对", count)
//
// 注意事项：
//   - 该方法是并发安全的
//   - 返回的数量包括已过期但未清理的键
//   - 性能开销 O(分片数)，建议不要频繁调用
func (sm *ShardedMap) Len() int {
	count := 0
	for i := 0; i < DefaultShardCount; i++ {
		sm.shards[i].mu.RLock()
		count += len(sm.shards[i].items)
		sm.shards[i].mu.RUnlock()
	}
	return count
}

// Clear 清空分片哈希表中的所有数据
//
// 注意事项：
//   - 该方法是并发安全的
//   - 会删除所有键值对，包括未过期的
func (sm *ShardedMap) Clear() {
	for i := 0; i < DefaultShardCount; i++ {
		sm.shards[i].mu.Lock()
		sm.shards[i].items = make(map[string]*item, DefaultInitialCapacity)
		sm.shards[i].mu.Unlock()
	}
}
