package tcp

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/yndnr/tokenginx/internal/storage"
	"github.com/yndnr/tokenginx/internal/transport/resp"
)

// CommandHandler 命令处理器
//
// CommandHandler 负责解析和执行 Redis 兼容的命令，包括：
//   - GET key
//   - SET key value [EX seconds]
//   - DEL key [key ...]
//   - EXISTS key [key ...]
//   - TTL key
//   - EXPIRE key seconds
//   - PING [message]
//   - ECHO message
//
// 示例：
//
//	sm := storage.NewShardedMap(4096)
//	handler := NewCommandHandler(sm)
//	response := handler.HandleCommand(commandValue)
type CommandHandler struct {
	sm *storage.ShardedMap // 存储引擎
}

// NewCommandHandler 创建一个新的命令处理器
//
// 参数说明：
//   - sm: 存储引擎实例
//
// 返回值：
//   - *CommandHandler: 命令处理器实例
func NewCommandHandler(sm *storage.ShardedMap) *CommandHandler {
	return &CommandHandler{
		sm: sm,
	}
}

// HandleCommand 处理 RESP 命令并返回响应
//
// 参数说明：
//   - value: 解析后的 RESP 命令（通常是 Array 类型）
//
// 返回值：
//   - *resp.Value: RESP 响应
//
// 示例：
//
//	// 处理 SET key value 命令
//	command := &resp.Value{
//	    Type: resp.Array,
//	    Array: []resp.Value{
//	        {Type: resp.BulkString, Bulk: []byte("SET")},
//	        {Type: resp.BulkString, Bulk: []byte("key1")},
//	        {Type: resp.BulkString, Bulk: []byte("value1")},
//	    },
//	}
//	response := handler.HandleCommand(command)
func (h *CommandHandler) HandleCommand(value *resp.Value) *resp.Value {
	// 命令必须是 Array 类型
	if value.Type != resp.Array {
		return &resp.Value{
			Type: resp.Error,
			Str:  "ERR 命令格式错误，期望 Array",
		}
	}

	// 空命令
	if len(value.Array) == 0 {
		return &resp.Value{
			Type: resp.Error,
			Str:  "ERR 空命令",
		}
	}

	// 提取命令名（第一个元素）
	cmdValue := value.Array[0]
	if cmdValue.Type != resp.BulkString {
		return &resp.Value{
			Type: resp.Error,
			Str:  "ERR 命令名必须是 Bulk String",
		}
	}

	cmdName := strings.ToUpper(string(cmdValue.Bulk))
	args := value.Array[1:]

	// 根据命令名分发
	switch cmdName {
	case "PING":
		return h.handlePing(args)
	case "ECHO":
		return h.handleEcho(args)
	case "GET":
		return h.handleGet(args)
	case "SET":
		return h.handleSet(args)
	case "DEL":
		return h.handleDel(args)
	case "EXISTS":
		return h.handleExists(args)
	case "TTL":
		return h.handleTTL(args)
	case "EXPIRE":
		return h.handleExpire(args)
	case "DBSIZE":
		return h.handleDBSize(args)
	case "FLUSHALL":
		return h.handleFlushAll(args)
	case "KEYS":
		return h.handleKeys(args)
	case "INFO":
		return h.handleInfo(args)
	default:
		return &resp.Value{
			Type: resp.Error,
			Str:  fmt.Sprintf("ERR 未知命令: %s", cmdName),
		}
	}
}

// handlePing 处理 PING 命令
//
// 格式：PING [message]
// 返回：如果没有 message，返回 "PONG"；否则返回 message
func (h *CommandHandler) handlePing(args []resp.Value) *resp.Value {
	if len(args) == 0 {
		return &resp.Value{
			Type: resp.SimpleString,
			Str:  "PONG",
		}
	}

	if args[0].Type != resp.BulkString {
		return &resp.Value{
			Type: resp.Error,
			Str:  "ERR PING 参数必须是 Bulk String",
		}
	}

	return &resp.Value{
		Type: resp.BulkString,
		Bulk: args[0].Bulk,
	}
}

// handleEcho 处理 ECHO 命令
//
// 格式：ECHO message
// 返回：message
func (h *CommandHandler) handleEcho(args []resp.Value) *resp.Value {
	if len(args) != 1 {
		return &resp.Value{
			Type: resp.Error,
			Str:  "ERR ECHO 命令需要 1 个参数",
		}
	}

	if args[0].Type != resp.BulkString {
		return &resp.Value{
			Type: resp.Error,
			Str:  "ERR ECHO 参数必须是 Bulk String",
		}
	}

	return &resp.Value{
		Type: resp.BulkString,
		Bulk: args[0].Bulk,
	}
}

// handleGet 处理 GET 命令
//
// 格式：GET key
// 返回：键对应的值，或 Null Bulk String（如果不存在）
func (h *CommandHandler) handleGet(args []resp.Value) *resp.Value {
	if len(args) != 1 {
		return &resp.Value{
			Type: resp.Error,
			Str:  "ERR GET 命令需要 1 个参数",
		}
	}

	if args[0].Type != resp.BulkString {
		return &resp.Value{
			Type: resp.Error,
			Str:  "ERR 键名必须是 Bulk String",
		}
	}

	key := string(args[0].Bulk)
	value, exists := h.sm.Get(key)
	if !exists {
		return &resp.Value{
			Type: resp.BulkString,
			Null: true,
		}
	}

	// 将值转换为字符串
	var strValue string
	switch v := value.(type) {
	case string:
		strValue = v
	case []byte:
		strValue = string(v)
	default:
		strValue = fmt.Sprintf("%v", v)
	}

	return &resp.Value{
		Type: resp.BulkString,
		Bulk: []byte(strValue),
	}
}

// handleSet 处理 SET 命令
//
// 格式：SET key value [EX seconds]
// 返回：+OK 或错误
func (h *CommandHandler) handleSet(args []resp.Value) *resp.Value {
	if len(args) < 2 {
		return &resp.Value{
			Type: resp.Error,
			Str:  "ERR SET 命令至少需要 2 个参数",
		}
	}

	if args[0].Type != resp.BulkString || args[1].Type != resp.BulkString {
		return &resp.Value{
			Type: resp.Error,
			Str:  "ERR 键和值必须是 Bulk String",
		}
	}

	key := string(args[0].Bulk)
	value := args[1].Bulk

	// 解析 TTL（如果有 EX 选项）
	ttl := 0
	if len(args) >= 4 {
		option := strings.ToUpper(string(args[2].Bulk))
		if option == "EX" {
			if args[3].Type != resp.BulkString {
				return &resp.Value{
					Type: resp.Error,
					Str:  "ERR EX 参数必须是数字",
				}
			}

			seconds, err := strconv.Atoi(string(args[3].Bulk))
			if err != nil || seconds < 0 {
				return &resp.Value{
					Type: resp.Error,
					Str:  "ERR EX 参数必须是非负整数",
				}
			}
			ttl = seconds
		}
	}

	// 存储键值对
	if err := h.sm.Set(key, value, ttl); err != nil {
		return &resp.Value{
			Type: resp.Error,
			Str:  fmt.Sprintf("ERR 设置失败: %v", err),
		}
	}

	return &resp.Value{
		Type: resp.SimpleString,
		Str:  "OK",
	}
}

// handleDel 处理 DEL 命令
//
// 格式：DEL key [key ...]
// 返回：删除的键数量
func (h *CommandHandler) handleDel(args []resp.Value) *resp.Value {
	if len(args) == 0 {
		return &resp.Value{
			Type: resp.Error,
			Str:  "ERR DEL 命令至少需要 1 个参数",
		}
	}

	count := 0
	for _, arg := range args {
		if arg.Type != resp.BulkString {
			return &resp.Value{
				Type: resp.Error,
				Str:  "ERR 键名必须是 Bulk String",
			}
		}

		key := string(arg.Bulk)
		if h.sm.Delete(key) {
			count++
		}
	}

	return &resp.Value{
		Type: resp.Integer,
		Int:  int64(count),
	}
}

// handleExists 处理 EXISTS 命令
//
// 格式：EXISTS key [key ...]
// 返回：存在的键数量
func (h *CommandHandler) handleExists(args []resp.Value) *resp.Value {
	if len(args) == 0 {
		return &resp.Value{
			Type: resp.Error,
			Str:  "ERR EXISTS 命令至少需要 1 个参数",
		}
	}

	count := 0
	for _, arg := range args {
		if arg.Type != resp.BulkString {
			return &resp.Value{
				Type: resp.Error,
				Str:  "ERR 键名必须是 Bulk String",
			}
		}

		key := string(arg.Bulk)
		if h.sm.Exists(key) {
			count++
		}
	}

	return &resp.Value{
		Type: resp.Integer,
		Int:  int64(count),
	}
}

// handleTTL 处理 TTL 命令
//
// 格式：TTL key
// 返回：剩余 TTL（秒），-1 表示不存在，-2 表示永不过期
func (h *CommandHandler) handleTTL(args []resp.Value) *resp.Value {
	if len(args) != 1 {
		return &resp.Value{
			Type: resp.Error,
			Str:  "ERR TTL 命令需要 1 个参数",
		}
	}

	if args[0].Type != resp.BulkString {
		return &resp.Value{
			Type: resp.Error,
			Str:  "ERR 键名必须是 Bulk String",
		}
	}

	key := string(args[0].Bulk)
	ttl := storage.TTL(h.sm, key)

	return &resp.Value{
		Type: resp.Integer,
		Int:  ttl,
	}
}

// handleExpire 处理 EXPIRE 命令
//
// 格式：EXPIRE key seconds
// 返回：1 表示成功，0 表示键不存在
func (h *CommandHandler) handleExpire(args []resp.Value) *resp.Value {
	if len(args) != 2 {
		return &resp.Value{
			Type: resp.Error,
			Str:  "ERR EXPIRE 命令需要 2 个参数",
		}
	}

	if args[0].Type != resp.BulkString || args[1].Type != resp.BulkString {
		return &resp.Value{
			Type: resp.Error,
			Str:  "ERR 参数必须是 Bulk String",
		}
	}

	key := string(args[0].Bulk)
	seconds, err := strconv.Atoi(string(args[1].Bulk))
	if err != nil || seconds < 0 {
		return &resp.Value{
			Type: resp.Error,
			Str:  "ERR 过期时间必须是非负整数",
		}
	}

	updated := storage.Expire(h.sm, key, seconds)
	result := int64(0)
	if updated {
		result = 1
	}

	return &resp.Value{
		Type: resp.Integer,
		Int:  result,
	}
}

// handleDBSize 处理 DBSIZE 命令
//
// 格式：DBSIZE
// 返回：数据库中键的数量
func (h *CommandHandler) handleDBSize(args []resp.Value) *resp.Value {
	if len(args) != 0 {
		return &resp.Value{
			Type: resp.Error,
			Str:  "ERR DBSIZE 命令不需要参数",
		}
	}

	size := h.sm.Len()

	return &resp.Value{
		Type: resp.Integer,
		Int:  int64(size),
	}
}

// handleFlushAll 处理 FLUSHALL 命令
//
// 格式：FLUSHALL
// 返回：+OK
func (h *CommandHandler) handleFlushAll(args []resp.Value) *resp.Value {
	if len(args) != 0 {
		return &resp.Value{
			Type: resp.Error,
			Str:  "ERR FLUSHALL 命令不需要参数",
		}
	}

	h.sm.Clear()

	return &resp.Value{
		Type: resp.SimpleString,
		Str:  "OK",
	}
}

// handleKeys 处理 KEYS 命令
//
// 格式：KEYS pattern
// 返回：匹配的键列表
// 注意：这是一个简化实现，仅支持 * 通配符
func (h *CommandHandler) handleKeys(args []resp.Value) *resp.Value {
	if len(args) != 1 {
		return &resp.Value{
			Type: resp.Error,
			Str:  "ERR KEYS 命令需要 1 个参数",
		}
	}

	if args[0].Type != resp.BulkString {
		return &resp.Value{
			Type: resp.Error,
			Str:  "ERR 模式必须是 Bulk String",
		}
	}

	pattern := string(args[0].Bulk)

	// 简化实现：仅支持 "*" 模式（返回所有键）
	if pattern != "*" {
		return &resp.Value{
			Type: resp.Error,
			Str:  "ERR 当前仅支持 KEYS * 模式",
		}
	}

	// 获取所有键（简化实现）
	keys := h.getAllKeys()

	// 构建响应数组
	result := make([]resp.Value, len(keys))
	for i, key := range keys {
		result[i] = resp.Value{
			Type: resp.BulkString,
			Bulk: []byte(key),
		}
	}

	return &resp.Value{
		Type:  resp.Array,
		Array: result,
	}
}

// handleInfo 处理 INFO 命令
//
// 格式：INFO [section]
// 返回：服务器信息
func (h *CommandHandler) handleInfo(args []resp.Value) *resp.Value {
	section := "all"
	if len(args) > 0 {
		if args[0].Type != resp.BulkString {
			return &resp.Value{
				Type: resp.Error,
				Str:  "ERR section 必须是 Bulk String",
			}
		}
		section = strings.ToLower(string(args[0].Bulk))
	}

	info := h.buildInfoString(section)

	return &resp.Value{
		Type: resp.BulkString,
		Bulk: []byte(info),
	}
}

// getAllKeys 获取所有键（简化实现，仅用于 KEYS 命令）
//
// 注意：这是一个 O(n) 操作，在生产环境中应避免频繁使用
func (h *CommandHandler) getAllKeys() []string {
	keys := make([]string, 0, 100)

	// 遍历所有分片收集键
	// 注意：这里访问了 ShardedMap 的内部实现
	// 在实际生产代码中，应该在 ShardedMap 中提供一个专门的方法
	for i := 0; i < 256; i++ {
		shard := h.sm.GetShardForIndex(i)
		if shard == nil {
			continue
		}

		shardKeys := shard.GetAllKeys()
		keys = append(keys, shardKeys...)
	}

	return keys
}

// buildInfoString 构建 INFO 命令的响应字符串
func (h *CommandHandler) buildInfoString(section string) string {
	var info strings.Builder

	if section == "all" || section == "server" {
		info.WriteString("# Server\r\n")
		info.WriteString("tokenginx_version:0.1.0-dev\r\n")
		info.WriteString("tokenginx_mode:standalone\r\n")
		info.WriteString("os:Linux\r\n")
		info.WriteString("arch_bits:64\r\n")
		info.WriteString("\r\n")
	}

	if section == "all" || section == "memory" {
		info.WriteString("# Memory\r\n")
		// 这里可以添加更详细的内存统计
		info.WriteString("\r\n")
	}

	if section == "all" || section == "stats" {
		info.WriteString("# Stats\r\n")
		size := h.sm.Len()
		info.WriteString(fmt.Sprintf("total_keys:%d\r\n", size))
		info.WriteString("\r\n")
	}

	if section == "all" || section == "keyspace" {
		info.WriteString("# Keyspace\r\n")
		size := h.sm.Len()
		info.WriteString(fmt.Sprintf("db0:keys=%d\r\n", size))
		info.WriteString("\r\n")
	}

	if info.Len() == 0 {
		return fmt.Sprintf("# %s section not supported\r\n", section)
	}

	return info.String()
}
