package tcp

import (
	"testing"

	"github.com/yndnr/tokenginx/internal/storage"
	"github.com/yndnr/tokenginx/internal/transport/resp"
)

// TestCommandHandler_Ping 测试 PING 命令
func TestCommandHandler_Ping(t *testing.T) {
	sm := storage.NewShardedMap(1024)
	handler := NewCommandHandler(sm)

	// PING without message
	cmd := &resp.Value{
		Type: resp.Array,
		Array: []resp.Value{
			{Type: resp.BulkString, Bulk: []byte("PING")},
		},
	}

	response := handler.HandleCommand(cmd)
	if response.Type != resp.SimpleString || response.Str != "PONG" {
		t.Errorf("Expected 'PONG', got %v", response)
	}

	// PING with message
	cmd = &resp.Value{
		Type: resp.Array,
		Array: []resp.Value{
			{Type: resp.BulkString, Bulk: []byte("PING")},
			{Type: resp.BulkString, Bulk: []byte("hello")},
		},
	}

	response = handler.HandleCommand(cmd)
	if response.Type != resp.BulkString || string(response.Bulk) != "hello" {
		t.Errorf("Expected 'hello', got %v", response)
	}
}

// TestCommandHandler_Echo 测试 ECHO 命令
func TestCommandHandler_Echo(t *testing.T) {
	sm := storage.NewShardedMap(1024)
	handler := NewCommandHandler(sm)

	cmd := &resp.Value{
		Type: resp.Array,
		Array: []resp.Value{
			{Type: resp.BulkString, Bulk: []byte("ECHO")},
			{Type: resp.BulkString, Bulk: []byte("test message")},
		},
	}

	response := handler.HandleCommand(cmd)
	if response.Type != resp.BulkString || string(response.Bulk) != "test message" {
		t.Errorf("Expected 'test message', got %v", response)
	}
}

// TestCommandHandler_Get 测试 GET 命令
func TestCommandHandler_Get(t *testing.T) {
	sm := storage.NewShardedMap(1024)
	handler := NewCommandHandler(sm)

	// GET non-existent key
	cmd := &resp.Value{
		Type: resp.Array,
		Array: []resp.Value{
			{Type: resp.BulkString, Bulk: []byte("GET")},
			{Type: resp.BulkString, Bulk: []byte("key1")},
		},
	}

	response := handler.HandleCommand(cmd)
	if response.Type != resp.BulkString || !response.Null {
		t.Errorf("Expected null bulk string, got %v", response)
	}

	// Set a key and get it
	sm.Set("key1", "value1", 0)

	response = handler.HandleCommand(cmd)
	if response.Type != resp.BulkString || string(response.Bulk) != "value1" {
		t.Errorf("Expected 'value1', got %v", response)
	}
}

// TestCommandHandler_Set 测试 SET 命令
func TestCommandHandler_Set(t *testing.T) {
	sm := storage.NewShardedMap(1024)
	handler := NewCommandHandler(sm)

	// SET without TTL
	cmd := &resp.Value{
		Type: resp.Array,
		Array: []resp.Value{
			{Type: resp.BulkString, Bulk: []byte("SET")},
			{Type: resp.BulkString, Bulk: []byte("key1")},
			{Type: resp.BulkString, Bulk: []byte("value1")},
		},
	}

	response := handler.HandleCommand(cmd)
	if response.Type != resp.SimpleString || response.Str != "OK" {
		t.Errorf("Expected 'OK', got %v", response)
	}

	// Verify the value was set
	value, exists := sm.Get("key1")
	if !exists {
		t.Error("Key should exist")
	}
	if string(value.([]byte)) != "value1" {
		t.Errorf("Expected 'value1', got %v", value)
	}

	// SET with EX option
	cmd = &resp.Value{
		Type: resp.Array,
		Array: []resp.Value{
			{Type: resp.BulkString, Bulk: []byte("SET")},
			{Type: resp.BulkString, Bulk: []byte("key2")},
			{Type: resp.BulkString, Bulk: []byte("value2")},
			{Type: resp.BulkString, Bulk: []byte("EX")},
			{Type: resp.BulkString, Bulk: []byte("10")},
		},
	}

	response = handler.HandleCommand(cmd)
	if response.Type != resp.SimpleString || response.Str != "OK" {
		t.Errorf("Expected 'OK', got %v", response)
	}

	// Verify TTL was set
	ttl := storage.TTL(sm, "key2")
	if ttl <= 0 || ttl > 10 {
		t.Errorf("Expected TTL around 10, got %d", ttl)
	}
}

// TestCommandHandler_Del 测试 DEL 命令
func TestCommandHandler_Del(t *testing.T) {
	sm := storage.NewShardedMap(1024)
	handler := NewCommandHandler(sm)

	// Set some keys
	sm.Set("key1", "value1", 0)
	sm.Set("key2", "value2", 0)
	sm.Set("key3", "value3", 0)

	// DEL multiple keys
	cmd := &resp.Value{
		Type: resp.Array,
		Array: []resp.Value{
			{Type: resp.BulkString, Bulk: []byte("DEL")},
			{Type: resp.BulkString, Bulk: []byte("key1")},
			{Type: resp.BulkString, Bulk: []byte("key2")},
			{Type: resp.BulkString, Bulk: []byte("nonexistent")},
		},
	}

	response := handler.HandleCommand(cmd)
	if response.Type != resp.Integer || response.Int != 2 {
		t.Errorf("Expected 2 deleted keys, got %v", response)
	}

	// Verify keys were deleted
	if sm.Exists("key1") || sm.Exists("key2") {
		t.Error("Keys should be deleted")
	}
	if !sm.Exists("key3") {
		t.Error("key3 should still exist")
	}
}

// TestCommandHandler_Exists 测试 EXISTS 命令
func TestCommandHandler_Exists(t *testing.T) {
	sm := storage.NewShardedMap(1024)
	handler := NewCommandHandler(sm)

	// Set some keys
	sm.Set("key1", "value1", 0)
	sm.Set("key2", "value2", 0)

	// EXISTS multiple keys
	cmd := &resp.Value{
		Type: resp.Array,
		Array: []resp.Value{
			{Type: resp.BulkString, Bulk: []byte("EXISTS")},
			{Type: resp.BulkString, Bulk: []byte("key1")},
			{Type: resp.BulkString, Bulk: []byte("key2")},
			{Type: resp.BulkString, Bulk: []byte("nonexistent")},
		},
	}

	response := handler.HandleCommand(cmd)
	if response.Type != resp.Integer || response.Int != 2 {
		t.Errorf("Expected 2 existing keys, got %v", response)
	}
}

// TestCommandHandler_TTL 测试 TTL 命令
func TestCommandHandler_TTL(t *testing.T) {
	sm := storage.NewShardedMap(1024)
	handler := NewCommandHandler(sm)

	// TTL for non-existent key
	cmd := &resp.Value{
		Type: resp.Array,
		Array: []resp.Value{
			{Type: resp.BulkString, Bulk: []byte("TTL")},
			{Type: resp.BulkString, Bulk: []byte("nonexistent")},
		},
	}

	response := handler.HandleCommand(cmd)
	if response.Type != resp.Integer || response.Int != -1 {
		t.Errorf("Expected -1, got %v", response)
	}

	// TTL for permanent key
	sm.Set("key1", "value1", 0)

	cmd = &resp.Value{
		Type: resp.Array,
		Array: []resp.Value{
			{Type: resp.BulkString, Bulk: []byte("TTL")},
			{Type: resp.BulkString, Bulk: []byte("key1")},
		},
	}

	response = handler.HandleCommand(cmd)
	if response.Type != resp.Integer || response.Int != -2 {
		t.Errorf("Expected -2, got %v", response)
	}

	// TTL for key with expiration
	sm.Set("key2", "value2", 10)

	cmd = &resp.Value{
		Type: resp.Array,
		Array: []resp.Value{
			{Type: resp.BulkString, Bulk: []byte("TTL")},
			{Type: resp.BulkString, Bulk: []byte("key2")},
		},
	}

	response = handler.HandleCommand(cmd)
	if response.Type != resp.Integer || response.Int <= 0 || response.Int > 10 {
		t.Errorf("Expected TTL around 10, got %v", response)
	}
}

// TestCommandHandler_Expire 测试 EXPIRE 命令
func TestCommandHandler_Expire(t *testing.T) {
	sm := storage.NewShardedMap(1024)
	handler := NewCommandHandler(sm)

	// EXPIRE non-existent key
	cmd := &resp.Value{
		Type: resp.Array,
		Array: []resp.Value{
			{Type: resp.BulkString, Bulk: []byte("EXPIRE")},
			{Type: resp.BulkString, Bulk: []byte("nonexistent")},
			{Type: resp.BulkString, Bulk: []byte("10")},
		},
	}

	response := handler.HandleCommand(cmd)
	if response.Type != resp.Integer || response.Int != 0 {
		t.Errorf("Expected 0, got %v", response)
	}

	// EXPIRE existing key
	sm.Set("key1", "value1", 0)

	cmd = &resp.Value{
		Type: resp.Array,
		Array: []resp.Value{
			{Type: resp.BulkString, Bulk: []byte("EXPIRE")},
			{Type: resp.BulkString, Bulk: []byte("key1")},
			{Type: resp.BulkString, Bulk: []byte("10")},
		},
	}

	response = handler.HandleCommand(cmd)
	if response.Type != resp.Integer || response.Int != 1 {
		t.Errorf("Expected 1, got %v", response)
	}

	// Verify TTL was updated
	ttl := storage.TTL(sm, "key1")
	if ttl <= 0 || ttl > 10 {
		t.Errorf("Expected TTL around 10, got %d", ttl)
	}
}

// TestCommandHandler_UnknownCommand 测试未知命令
func TestCommandHandler_UnknownCommand(t *testing.T) {
	sm := storage.NewShardedMap(1024)
	handler := NewCommandHandler(sm)

	cmd := &resp.Value{
		Type: resp.Array,
		Array: []resp.Value{
			{Type: resp.BulkString, Bulk: []byte("UNKNOWN")},
		},
	}

	response := handler.HandleCommand(cmd)
	if response.Type != resp.Error {
		t.Errorf("Expected error, got %v", response)
	}
}

// TestCommandHandler_InvalidFormat 测试无效命令格式
func TestCommandHandler_InvalidFormat(t *testing.T) {
	sm := storage.NewShardedMap(1024)
	handler := NewCommandHandler(sm)

	// Not an array
	cmd := &resp.Value{
		Type: resp.SimpleString,
		Str:  "GET key",
	}

	response := handler.HandleCommand(cmd)
	if response.Type != resp.Error {
		t.Errorf("Expected error, got %v", response)
	}

	// Empty array
	cmd = &resp.Value{
		Type:  resp.Array,
		Array: []resp.Value{},
	}

	response = handler.HandleCommand(cmd)
	if response.Type != resp.Error {
		t.Errorf("Expected error, got %v", response)
	}
}
