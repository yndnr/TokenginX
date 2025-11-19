package tcp

import (
	"bufio"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/yndnr/tokenginx/internal/storage"
	"github.com/yndnr/tokenginx/internal/transport/resp"
)

// TestServer_StartStop 测试服务器启动和停止
func TestServer_StartStop(t *testing.T) {
	sm := storage.NewShardedMap(1024)
	server := NewServer(":0", sm) // 使用端口 0 让系统自动分配

	// Start server
	if err := server.Start(); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}

	stats := server.GetStats()
	if !stats.Running {
		t.Error("Server should be running")
	}

	// Stop server
	server.Stop()

	stats = server.GetStats()
	if stats.Running {
		t.Error("Server should not be running after Stop")
	}
}

// TestServer_MultipleStart 测试多次启动
func TestServer_MultipleStart(t *testing.T) {
	sm := storage.NewShardedMap(1024)
	server := NewServer(":0", sm)

	if err := server.Start(); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer server.Stop()

	// Second start should fail
	err := server.Start()
	if err == nil {
		t.Error("Expected error on second Start, got nil")
	}
}

// TestServer_BasicConnection 测试基本连接
func TestServer_BasicConnection(t *testing.T) {
	sm := storage.NewShardedMap(1024)
	server := NewServer("127.0.0.1:16380", sm)

	if err := server.Start(); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer server.Stop()

	// Wait for server to start
	time.Sleep(100 * time.Millisecond)

	// Connect to server
	conn, err := net.Dial("tcp", "127.0.0.1:16380")
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	// Send PING command
	writer := resp.NewWriter(conn)
	cmd := &resp.Value{
		Type: resp.Array,
		Array: []resp.Value{
			{Type: resp.BulkString, Bulk: []byte("PING")},
		},
	}

	if err := writer.WriteValue(cmd); err != nil {
		t.Fatalf("Failed to write command: %v", err)
	}
	writer.Flush()

	// Read response
	parser := resp.NewParser(conn)
	response, err := parser.Parse()
	if err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if response.Type != resp.SimpleString || response.Str != "PONG" {
		t.Errorf("Expected 'PONG', got %v", response)
	}
}

// TestServer_SetGetCommand 测试 SET/GET 命令
func TestServer_SetGetCommand(t *testing.T) {
	sm := storage.NewShardedMap(1024)
	server := NewServer("127.0.0.1:16381", sm)

	if err := server.Start(); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer server.Stop()

	time.Sleep(100 * time.Millisecond)

	conn, err := net.Dial("tcp", "127.0.0.1:16381")
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	writer := resp.NewWriter(conn)
	parser := resp.NewParser(conn)

	// SET key value
	setCmd := &resp.Value{
		Type: resp.Array,
		Array: []resp.Value{
			{Type: resp.BulkString, Bulk: []byte("SET")},
			{Type: resp.BulkString, Bulk: []byte("testkey")},
			{Type: resp.BulkString, Bulk: []byte("testvalue")},
		},
	}

	writer.WriteValue(setCmd)
	writer.Flush()

	response, _ := parser.Parse()
	if response.Type != resp.SimpleString || response.Str != "OK" {
		t.Errorf("Expected 'OK', got %v", response)
	}

	// GET key
	getCmd := &resp.Value{
		Type: resp.Array,
		Array: []resp.Value{
			{Type: resp.BulkString, Bulk: []byte("GET")},
			{Type: resp.BulkString, Bulk: []byte("testkey")},
		},
	}

	writer.WriteValue(getCmd)
	writer.Flush()

	response, _ = parser.Parse()
	if response.Type != resp.BulkString || string(response.Bulk) != "testvalue" {
		t.Errorf("Expected 'testvalue', got %v", response)
	}
}

// TestServer_MultipleClients 测试多客户端并发
func TestServer_MultipleClients(t *testing.T) {
	sm := storage.NewShardedMap(1024)
	server := NewServer("127.0.0.1:16382", sm)

	if err := server.Start(); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer server.Stop()

	time.Sleep(100 * time.Millisecond)

	// Create multiple clients
	numClients := 10
	done := make(chan bool, numClients)

	for i := 0; i < numClients; i++ {
		go func(id int) {
			conn, err := net.Dial("tcp", "127.0.0.1:16382")
			if err != nil {
				t.Errorf("Client %d failed to connect: %v", id, err)
				done <- false
				return
			}
			defer conn.Close()

			writer := resp.NewWriter(conn)
			parser := resp.NewParser(conn)

			// Send PING
			cmd := &resp.Value{
				Type: resp.Array,
				Array: []resp.Value{
					{Type: resp.BulkString, Bulk: []byte("PING")},
				},
			}

			writer.WriteValue(cmd)
			writer.Flush()

			response, err := parser.Parse()
			if err != nil || response.Type != resp.SimpleString || response.Str != "PONG" {
				t.Errorf("Client %d got unexpected response", id)
				done <- false
				return
			}

			done <- true
		}(i)
	}

	// Wait for all clients
	success := 0
	for i := 0; i < numClients; i++ {
		if <-done {
			success++
		}
	}

	if success != numClients {
		t.Errorf("Expected %d successful clients, got %d", numClients, success)
	}

	// Check stats
	stats := server.GetStats()
	if stats.TotalCommands < int64(numClients) {
		t.Errorf("Expected at least %d commands, got %d", numClients, stats.TotalCommands)
	}
}

// TestServer_Stats 测试统计信息
func TestServer_Stats(t *testing.T) {
	sm := storage.NewShardedMap(1024)
	server := NewServer("127.0.0.1:16383", sm)

	if err := server.Start(); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer server.Stop()

	time.Sleep(100 * time.Millisecond)

	initialStats := server.GetStats()

	// Connect and send command
	conn, _ := net.Dial("tcp", "127.0.0.1:16383")
	writer := resp.NewWriter(conn)
	parser := resp.NewParser(conn)

	cmd := &resp.Value{
		Type: resp.Array,
		Array: []resp.Value{
			{Type: resp.BulkString, Bulk: []byte("PING")},
		},
	}

	writer.WriteValue(cmd)
	writer.Flush()
	parser.Parse()

	finalStats := server.GetStats()

	if finalStats.TotalConnections != initialStats.TotalConnections+1 {
		t.Errorf("Expected TotalConnections to increase by 1")
	}

	if finalStats.TotalCommands != initialStats.TotalCommands+1 {
		t.Errorf("Expected TotalCommands to increase by 1")
	}

	conn.Close()
}

// TestServer_RawProtocol 测试原始 RESP 协议
func TestServer_RawProtocol(t *testing.T) {
	sm := storage.NewShardedMap(1024)
	server := NewServer("127.0.0.1:16384", sm)

	if err := server.Start(); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer server.Stop()

	time.Sleep(100 * time.Millisecond)

	conn, err := net.Dial("tcp", "127.0.0.1:16384")
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	// Send raw RESP command: *1\r\n$4\r\nPING\r\n
	rawCmd := "*1\r\n$4\r\nPING\r\n"
	conn.Write([]byte(rawCmd))

	// Read response
	reader := bufio.NewReader(conn)
	response, err := reader.ReadString('\n')
	if err != nil {
		t.Fatalf("Failed to read response: %v", err)
	}

	if !strings.HasPrefix(response, "+PONG") {
		t.Errorf("Expected '+PONG', got %s", response)
	}
}

// BenchmarkServer_PING 基准测试：PING 命令
func BenchmarkServer_PING(b *testing.B) {
	sm := storage.NewShardedMap(4096)
	server := NewServer("127.0.0.1:16385", sm)

	if err := server.Start(); err != nil {
		b.Fatalf("Failed to start server: %v", err)
	}
	defer server.Stop()

	time.Sleep(100 * time.Millisecond)

	conn, _ := net.Dial("tcp", "127.0.0.1:16385")
	defer conn.Close()

	writer := resp.NewWriter(conn)
	parser := resp.NewParser(conn)

	cmd := &resp.Value{
		Type: resp.Array,
		Array: []resp.Value{
			{Type: resp.BulkString, Bulk: []byte("PING")},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		writer.WriteValue(cmd)
		writer.Flush()
		parser.Parse()
	}
}

// BenchmarkServer_GET 基准测试：GET 命令
func BenchmarkServer_GET(b *testing.B) {
	sm := storage.NewShardedMap(4096)
	sm.Set("benchkey", "benchvalue", 0)

	server := NewServer("127.0.0.1:16386", sm)

	if err := server.Start(); err != nil {
		b.Fatalf("Failed to start server: %v", err)
	}
	defer server.Stop()

	time.Sleep(100 * time.Millisecond)

	conn, _ := net.Dial("tcp", "127.0.0.1:16386")
	defer conn.Close()

	writer := resp.NewWriter(conn)
	parser := resp.NewParser(conn)

	cmd := &resp.Value{
		Type: resp.Array,
		Array: []resp.Value{
			{Type: resp.BulkString, Bulk: []byte("GET")},
			{Type: resp.BulkString, Bulk: []byte("benchkey")},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		writer.WriteValue(cmd)
		writer.Flush()
		parser.Parse()
	}
}
