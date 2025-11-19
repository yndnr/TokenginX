package resp

import (
	"bytes"
	"strings"
	"testing"
)

// TestParser_SimpleString 测试解析 Simple String
func TestParser_SimpleString(t *testing.T) {
	input := "+OK\r\n"
	parser := NewParser(strings.NewReader(input))

	value, err := parser.Parse()
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if value.Type != SimpleString {
		t.Errorf("Expected type SimpleString, got %c", value.Type)
	}

	if value.Str != "OK" {
		t.Errorf("Expected 'OK', got '%s'", value.Str)
	}
}

// TestParser_Error 测试解析 Error
func TestParser_Error(t *testing.T) {
	input := "-ERR unknown command\r\n"
	parser := NewParser(strings.NewReader(input))

	value, err := parser.Parse()
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if value.Type != Error {
		t.Errorf("Expected type Error, got %c", value.Type)
	}

	if value.Str != "ERR unknown command" {
		t.Errorf("Expected 'ERR unknown command', got '%s'", value.Str)
	}
}

// TestParser_Integer 测试解析 Integer
func TestParser_Integer(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
	}{
		{":0\r\n", 0},
		{":1000\r\n", 1000},
		{":-1000\r\n", -1000},
		{":9223372036854775807\r\n", 9223372036854775807}, // Max int64
	}

	for _, tt := range tests {
		parser := NewParser(strings.NewReader(tt.input))
		value, err := parser.Parse()
		if err != nil {
			t.Fatalf("Parse failed for %s: %v", tt.input, err)
		}

		if value.Type != Integer {
			t.Errorf("Expected type Integer, got %c", value.Type)
		}

		if value.Int != tt.expected {
			t.Errorf("Expected %d, got %d", tt.expected, value.Int)
		}
	}
}

// TestParser_BulkString 测试解析 Bulk String
func TestParser_BulkString(t *testing.T) {
	input := "$6\r\nfoobar\r\n"
	parser := NewParser(strings.NewReader(input))

	value, err := parser.Parse()
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if value.Type != BulkString {
		t.Errorf("Expected type BulkString, got %c", value.Type)
	}

	if string(value.Bulk) != "foobar" {
		t.Errorf("Expected 'foobar', got '%s'", string(value.Bulk))
	}

	if value.Null {
		t.Error("Expected non-null bulk string")
	}
}

// TestParser_NullBulkString 测试解析 Null Bulk String
func TestParser_NullBulkString(t *testing.T) {
	input := "$-1\r\n"
	parser := NewParser(strings.NewReader(input))

	value, err := parser.Parse()
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if value.Type != BulkString {
		t.Errorf("Expected type BulkString, got %c", value.Type)
	}

	if !value.Null {
		t.Error("Expected null bulk string")
	}
}

// TestParser_EmptyBulkString 测试解析空 Bulk String
func TestParser_EmptyBulkString(t *testing.T) {
	input := "$0\r\n\r\n"
	parser := NewParser(strings.NewReader(input))

	value, err := parser.Parse()
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if value.Type != BulkString {
		t.Errorf("Expected type BulkString, got %c", value.Type)
	}

	if len(value.Bulk) != 0 {
		t.Errorf("Expected empty bulk string, got length %d", len(value.Bulk))
	}
}

// TestParser_Array 测试解析 Array
func TestParser_Array(t *testing.T) {
	input := "*2\r\n$3\r\nfoo\r\n$3\r\nbar\r\n"
	parser := NewParser(strings.NewReader(input))

	value, err := parser.Parse()
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if value.Type != Array {
		t.Errorf("Expected type Array, got %c", value.Type)
	}

	if len(value.Array) != 2 {
		t.Errorf("Expected array length 2, got %d", len(value.Array))
	}

	if string(value.Array[0].Bulk) != "foo" {
		t.Errorf("Expected 'foo', got '%s'", string(value.Array[0].Bulk))
	}

	if string(value.Array[1].Bulk) != "bar" {
		t.Errorf("Expected 'bar', got '%s'", string(value.Array[1].Bulk))
	}
}

// TestParser_EmptyArray 测试解析空数组
func TestParser_EmptyArray(t *testing.T) {
	input := "*0\r\n"
	parser := NewParser(strings.NewReader(input))

	value, err := parser.Parse()
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if value.Type != Array {
		t.Errorf("Expected type Array, got %c", value.Type)
	}

	if len(value.Array) != 0 {
		t.Errorf("Expected empty array, got length %d", len(value.Array))
	}
}

// TestParser_NullArray 测试解析 Null Array
func TestParser_NullArray(t *testing.T) {
	input := "*-1\r\n"
	parser := NewParser(strings.NewReader(input))

	value, err := parser.Parse()
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if value.Type != Array {
		t.Errorf("Expected type Array, got %c", value.Type)
	}

	if !value.Null {
		t.Error("Expected null array")
	}
}

// TestParser_NestedArray 测试解析嵌套数组
func TestParser_NestedArray(t *testing.T) {
	input := "*2\r\n*3\r\n:1\r\n:2\r\n:3\r\n*2\r\n+Foo\r\n-Bar\r\n"
	parser := NewParser(strings.NewReader(input))

	value, err := parser.Parse()
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if value.Type != Array {
		t.Errorf("Expected type Array, got %c", value.Type)
	}

	if len(value.Array) != 2 {
		t.Fatalf("Expected array length 2, got %d", len(value.Array))
	}

	// 第一个子数组 [1, 2, 3]
	subArray1 := value.Array[0]
	if len(subArray1.Array) != 3 {
		t.Errorf("Expected first sub-array length 3, got %d", len(subArray1.Array))
	}
	if subArray1.Array[0].Int != 1 || subArray1.Array[1].Int != 2 || subArray1.Array[2].Int != 3 {
		t.Error("First sub-array values incorrect")
	}

	// 第二个子数组 ["Foo", Error("Bar")]
	subArray2 := value.Array[1]
	if len(subArray2.Array) != 2 {
		t.Errorf("Expected second sub-array length 2, got %d", len(subArray2.Array))
	}
}

// TestParser_MixedArray 测试解析混合类型数组
func TestParser_MixedArray(t *testing.T) {
	input := "*5\r\n+simple\r\n-error\r\n:100\r\n$4\r\nbulk\r\n*0\r\n"
	parser := NewParser(strings.NewReader(input))

	value, err := parser.Parse()
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(value.Array) != 5 {
		t.Fatalf("Expected array length 5, got %d", len(value.Array))
	}

	if value.Array[0].Type != SimpleString || value.Array[0].Str != "simple" {
		t.Error("Array[0] incorrect")
	}
	if value.Array[1].Type != Error || value.Array[1].Str != "error" {
		t.Error("Array[1] incorrect")
	}
	if value.Array[2].Type != Integer || value.Array[2].Int != 100 {
		t.Error("Array[2] incorrect")
	}
	if value.Array[3].Type != BulkString || string(value.Array[3].Bulk) != "bulk" {
		t.Error("Array[3] incorrect")
	}
	if value.Array[4].Type != Array || len(value.Array[4].Array) != 0 {
		t.Error("Array[4] incorrect")
	}
}

// TestParser_RediCommand 测试解析 Redis 命令
func TestParser_RediCommand(t *testing.T) {
	// SET key value
	input := "*3\r\n$3\r\nSET\r\n$3\r\nkey\r\n$5\r\nvalue\r\n"
	parser := NewParser(strings.NewReader(input))

	value, err := parser.Parse()
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(value.Array) != 3 {
		t.Fatalf("Expected 3 arguments, got %d", len(value.Array))
	}

	if string(value.Array[0].Bulk) != "SET" {
		t.Errorf("Expected command 'SET', got '%s'", string(value.Array[0].Bulk))
	}
	if string(value.Array[1].Bulk) != "key" {
		t.Errorf("Expected key 'key', got '%s'", string(value.Array[1].Bulk))
	}
	if string(value.Array[2].Bulk) != "value" {
		t.Errorf("Expected value 'value', got '%s'", string(value.Array[2].Bulk))
	}
}

// TestParser_InvalidFormat 测试无效格式
func TestParser_InvalidFormat(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"Missing CRLF", "+OK"},
		{"Invalid type", "?invalid\r\n"},
		{"Invalid integer", ":abc\r\n"},
		{"Invalid bulk length", "$abc\r\ndata\r\n"},
		{"Negative bulk length", "$-2\r\n"},
		{"Invalid array length", "*abc\r\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewParser(strings.NewReader(tt.input))
			_, err := parser.Parse()
			if err == nil {
				t.Error("Expected error, got nil")
			}
		})
	}
}

// TestWriter_SimpleString 测试写入 Simple String
func TestWriter_SimpleString(t *testing.T) {
	var buf bytes.Buffer
	writer := NewWriter(&buf)

	err := writer.WriteSimpleString("OK")
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	writer.Flush()

	expected := "+OK\r\n"
	if buf.String() != expected {
		t.Errorf("Expected %q, got %q", expected, buf.String())
	}
}

// TestWriter_Error 测试写入 Error
func TestWriter_Error(t *testing.T) {
	var buf bytes.Buffer
	writer := NewWriter(&buf)

	err := writer.WriteError("ERR unknown")
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	writer.Flush()

	expected := "-ERR unknown\r\n"
	if buf.String() != expected {
		t.Errorf("Expected %q, got %q", expected, buf.String())
	}
}

// TestWriter_Integer 测试写入 Integer
func TestWriter_Integer(t *testing.T) {
	var buf bytes.Buffer
	writer := NewWriter(&buf)

	err := writer.WriteInteger(1000)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	writer.Flush()

	expected := ":1000\r\n"
	if buf.String() != expected {
		t.Errorf("Expected %q, got %q", expected, buf.String())
	}
}

// TestWriter_BulkString 测试写入 Bulk String
func TestWriter_BulkString(t *testing.T) {
	var buf bytes.Buffer
	writer := NewWriter(&buf)

	err := writer.WriteBulkString([]byte("foobar"))
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	writer.Flush()

	expected := "$6\r\nfoobar\r\n"
	if buf.String() != expected {
		t.Errorf("Expected %q, got %q", expected, buf.String())
	}
}

// TestWriter_Null 测试写入 Null
func TestWriter_Null(t *testing.T) {
	var buf bytes.Buffer
	writer := NewWriter(&buf)

	err := writer.WriteNull()
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	writer.Flush()

	expected := "$-1\r\n"
	if buf.String() != expected {
		t.Errorf("Expected %q, got %q", expected, buf.String())
	}
}

// TestWriter_Array 测试写入 Array
func TestWriter_Array(t *testing.T) {
	var buf bytes.Buffer
	writer := NewWriter(&buf)

	array := []Value{
		{Type: BulkString, Bulk: []byte("foo")},
		{Type: BulkString, Bulk: []byte("bar")},
	}

	err := writer.WriteArray(array)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	writer.Flush()

	expected := "*2\r\n$3\r\nfoo\r\n$3\r\nbar\r\n"
	if buf.String() != expected {
		t.Errorf("Expected %q, got %q", expected, buf.String())
	}
}

// TestRoundTrip 测试读写往返
func TestRoundTrip(t *testing.T) {
	tests := []struct {
		name  string
		value Value
	}{
		{"Simple String", Value{Type: SimpleString, Str: "OK"}},
		{"Error", Value{Type: Error, Str: "ERR message"}},
		{"Integer", Value{Type: Integer, Int: 42}},
		{"Bulk String", Value{Type: BulkString, Bulk: []byte("hello")}},
		{"Null Bulk", Value{Type: BulkString, Null: true}},
		{"Empty Array", Value{Type: Array, Array: []Value{}}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer

			// Write
			writer := NewWriter(&buf)
			err := writer.WriteValue(&tt.value)
			if err != nil {
				t.Fatalf("Write failed: %v", err)
			}
			writer.Flush()

			// Read
			parser := NewParser(&buf)
			value, err := parser.Parse()
			if err != nil {
				t.Fatalf("Parse failed: %v", err)
			}

			// Compare
			if value.Type != tt.value.Type {
				t.Errorf("Type mismatch: expected %c, got %c", tt.value.Type, value.Type)
			}
		})
	}
}

// BenchmarkParser_SimpleString 基准测试：解析 Simple String
func BenchmarkParser_SimpleString(b *testing.B) {
	input := "+OK\r\n"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		parser := NewParser(strings.NewReader(input))
		parser.Parse()
	}
}

// BenchmarkParser_BulkString 基准测试：解析 Bulk String
func BenchmarkParser_BulkString(b *testing.B) {
	input := "$6\r\nfoobar\r\n"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		parser := NewParser(strings.NewReader(input))
		parser.Parse()
	}
}

// BenchmarkParser_Array 基准测试：解析 Array
func BenchmarkParser_Array(b *testing.B) {
	input := "*3\r\n$3\r\nSET\r\n$3\r\nkey\r\n$5\r\nvalue\r\n"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		parser := NewParser(strings.NewReader(input))
		parser.Parse()
	}
}

// BenchmarkWriter_BulkString 基准测试：写入 Bulk String
func BenchmarkWriter_BulkString(b *testing.B) {
	var buf bytes.Buffer
	buf.Grow(1024)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf.Reset()
		writer := NewWriter(&buf)
		writer.WriteBulkString([]byte("foobar"))
		writer.Flush()
	}
}
