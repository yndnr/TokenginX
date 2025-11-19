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
