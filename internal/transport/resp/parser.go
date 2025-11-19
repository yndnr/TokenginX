package resp

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strconv"
)

// RESP 协议类型标识符
const (
	SimpleString = '+' // Simple String: +OK\r\n
	Error        = '-' // Error: -ERR message\r\n
	Integer      = ':' // Integer: :1000\r\n
	BulkString   = '$' // Bulk String: $6\r\nfoobar\r\n
	Array        = '*' // Array: *2\r\n$3\r\nfoo\r\n$3\r\nbar\r\n
)

const (
	// MaxBulkSize 是 Bulk String 的最大大小 (512MB)
	MaxBulkSize = 512 * 1024 * 1024

	// MaxArrayLength 是数组的最大长度
	MaxArrayLength = 1024 * 1024
)

var (
	// ErrInvalidFormat RESP 格式错误
	ErrInvalidFormat = errors.New("invalid RESP format")

	// ErrInvalidType 无效的 RESP 类型
	ErrInvalidType = errors.New("invalid RESP type")

	// ErrTooLarge 数据过大
	ErrTooLarge = errors.New("data too large")

	// ErrUnexpectedEOF 意外的 EOF
	ErrUnexpectedEOF = errors.New("unexpected EOF")
)

// Value 表示一个 RESP 值
//
// RESP 支持 5 种数据类型：
//   - Simple String: 简单字符串，如 +OK\r\n
//   - Error: 错误信息，如 -ERR message\r\n
//   - Integer: 整数，如 :1000\r\n
//   - Bulk String: 二进制安全的字符串，如 $6\r\nfoobar\r\n
//   - Array: 数组，可包含任意类型的元素
type Value struct {
	Type  byte        // RESP 类型标识符
	Str   string      // Simple String 或 Error 的值
	Int   int64       // Integer 的值
	Bulk  []byte      // Bulk String 的值（二进制安全）
	Array []Value     // Array 的值
	Null  bool        // 是否为 Null Bulk String ($-1\r\n)
}

// Parser RESP 协议解析器
type Parser struct {
	reader *bufio.Reader
}

// NewParser 创建一个新的 RESP 解析器
//
// 参数说明：
//   - reader: 要解析的数据源
//
// 返回值：
//   - *Parser: 解析器实例
//
// 示例：
//
//	conn, _ := net.Dial("tcp", "localhost:6380")
//	parser := NewParser(conn)
//	value, err := parser.Parse()
func NewParser(reader io.Reader) *Parser {
	return &Parser{
		reader: bufio.NewReader(reader),
	}
}

// Parse 解析一个 RESP 值
//
// 返回值：
//   - *Value: 解析后的 RESP 值
//   - error: 错误信息，nil 表示成功
//
// 示例：
//
//	value, err := parser.Parse()
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	switch value.Type {
//	case SimpleString:
//	    fmt.Println("Simple String:", value.Str)
//	case BulkString:
//	    fmt.Println("Bulk String:", string(value.Bulk))
//	case Array:
//	    fmt.Println("Array with", len(value.Array), "elements")
//	}
func (p *Parser) Parse() (*Value, error) {
	// 读取类型标识符
	typeByte, err := p.reader.ReadByte()
	if err != nil {
		if err == io.EOF {
			return nil, ErrUnexpectedEOF
		}
		return nil, err
	}

	// 根据类型解析
	switch typeByte {
	case SimpleString:
		return p.parseSimpleString()
	case Error:
		return p.parseError()
	case Integer:
		return p.parseInteger()
	case BulkString:
		return p.parseBulkString()
	case Array:
		return p.parseArray()
	default:
		return nil, fmt.Errorf("%w: unknown type '%c'", ErrInvalidType, typeByte)
	}
}

// parseSimpleString 解析 Simple String (+OK\r\n)
func (p *Parser) parseSimpleString() (*Value, error) {
	line, err := p.readLine()
	if err != nil {
		return nil, err
	}

	return &Value{
		Type: SimpleString,
		Str:  string(line),
	}, nil
}

// parseError 解析 Error (-ERR message\r\n)
func (p *Parser) parseError() (*Value, error) {
	line, err := p.readLine()
	if err != nil {
		return nil, err
	}

	return &Value{
		Type: Error,
		Str:  string(line),
	}, nil
}

// parseInteger 解析 Integer (:1000\r\n)
func (p *Parser) parseInteger() (*Value, error) {
	line, err := p.readLine()
	if err != nil {
		return nil, err
	}

	num, err := strconv.ParseInt(string(line), 10, 64)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid integer", ErrInvalidFormat)
	}

	return &Value{
		Type: Integer,
		Int:  num,
	}, nil
}

// parseBulkString 解析 Bulk String ($6\r\nfoobar\r\n 或 $-1\r\n)
func (p *Parser) parseBulkString() (*Value, error) {
	line, err := p.readLine()
	if err != nil {
		return nil, err
	}

	// 解析长度
	length, err := strconv.ParseInt(string(line), 10, 64)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid bulk string length", ErrInvalidFormat)
	}

	// Null Bulk String
	if length == -1 {
		return &Value{
			Type: BulkString,
			Null: true,
		}, nil
	}

	// 检查长度
	if length < 0 {
		return nil, fmt.Errorf("%w: negative bulk string length", ErrInvalidFormat)
	}
	if length > MaxBulkSize {
		return nil, fmt.Errorf("%w: bulk string too large (%d bytes)", ErrTooLarge, length)
	}

	// 读取内容
	bulk := make([]byte, length)
	if _, err := io.ReadFull(p.reader, bulk); err != nil {
		return nil, err
	}

	// 读取并验证 \r\n
	if err := p.expectCRLF(); err != nil {
		return nil, err
	}

	return &Value{
		Type: BulkString,
		Bulk: bulk,
	}, nil
}

// parseArray 解析 Array (*2\r\n$3\r\nfoo\r\n$3\r\nbar\r\n)
func (p *Parser) parseArray() (*Value, error) {
	line, err := p.readLine()
	if err != nil {
		return nil, err
	}

	// 解析数组长度
	length, err := strconv.ParseInt(string(line), 10, 64)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid array length", ErrInvalidFormat)
	}

	// Null Array
	if length == -1 {
		return &Value{
			Type: Array,
			Null: true,
		}, nil
	}

	// 检查长度
	if length < 0 {
		return nil, fmt.Errorf("%w: negative array length", ErrInvalidFormat)
	}
	if length > MaxArrayLength {
		return nil, fmt.Errorf("%w: array too large (%d elements)", ErrTooLarge, length)
	}

	// 解析数组元素
	array := make([]Value, length)
	for i := int64(0); i < length; i++ {
		value, err := p.Parse()
		if err != nil {
			return nil, err
		}
		array[i] = *value
	}

	return &Value{
		Type:  Array,
		Array: array,
	}, nil
}

// readLine 读取一行（直到 \r\n）
func (p *Parser) readLine() ([]byte, error) {
	line, err := p.reader.ReadBytes('\n')
	if err != nil {
		return nil, err
	}

	// 验证并移除 \r\n
	if len(line) < 2 || line[len(line)-2] != '\r' {
		return nil, fmt.Errorf("%w: missing CRLF", ErrInvalidFormat)
	}

	return line[:len(line)-2], nil
}

// expectCRLF 期望并读取 \r\n
func (p *Parser) expectCRLF() error {
	cr, err := p.reader.ReadByte()
	if err != nil {
		return err
	}
	if cr != '\r' {
		return fmt.Errorf("%w: expected CR, got %q", ErrInvalidFormat, cr)
	}

	lf, err := p.reader.ReadByte()
	if err != nil {
		return err
	}
	if lf != '\n' {
		return fmt.Errorf("%w: expected LF, got %q", ErrInvalidFormat, lf)
	}

	return nil
}

// Writer RESP 协议写入器
type Writer struct {
	writer *bufio.Writer
}

// NewWriter 创建一个新的 RESP 写入器
//
// 参数说明：
//   - writer: 要写入的目标
//
// 返回值：
//   - *Writer: 写入器实例
//
// 示例：
//
//	conn, _ := net.Dial("tcp", "localhost:6380")
//	writer := NewWriter(conn)
//	writer.WriteSimpleString("OK")
//	writer.Flush()
func NewWriter(writer io.Writer) *Writer {
	return &Writer{
		writer: bufio.NewWriter(writer),
	}
}

// WriteValue 写入一个 RESP 值
func (w *Writer) WriteValue(value *Value) error {
	switch value.Type {
	case SimpleString:
		return w.WriteSimpleString(value.Str)
	case Error:
		return w.WriteError(value.Str)
	case Integer:
		return w.WriteInteger(value.Int)
	case BulkString:
		if value.Null {
			return w.WriteNull()
		}
		return w.WriteBulkString(value.Bulk)
	case Array:
		if value.Null {
			return w.WriteNullArray()
		}
		return w.WriteArray(value.Array)
	default:
		return fmt.Errorf("%w: unknown type '%c'", ErrInvalidType, value.Type)
	}
}

// WriteSimpleString 写入 Simple String
func (w *Writer) WriteSimpleString(s string) error {
	if err := w.writer.WriteByte(SimpleString); err != nil {
		return err
	}
	if _, err := w.writer.WriteString(s); err != nil {
		return err
	}
	return w.writeCRLF()
}

// WriteError 写入 Error
func (w *Writer) WriteError(msg string) error {
	if err := w.writer.WriteByte(Error); err != nil {
		return err
	}
	if _, err := w.writer.WriteString(msg); err != nil {
		return err
	}
	return w.writeCRLF()
}

// WriteInteger 写入 Integer
func (w *Writer) WriteInteger(n int64) error {
	if err := w.writer.WriteByte(Integer); err != nil {
		return err
	}
	if _, err := w.writer.WriteString(strconv.FormatInt(n, 10)); err != nil {
		return err
	}
	return w.writeCRLF()
}

// WriteBulkString 写入 Bulk String
func (w *Writer) WriteBulkString(b []byte) error {
	if err := w.writer.WriteByte(BulkString); err != nil {
		return err
	}
	if _, err := w.writer.WriteString(strconv.Itoa(len(b))); err != nil {
		return err
	}
	if err := w.writeCRLF(); err != nil {
		return err
	}
	if _, err := w.writer.Write(b); err != nil {
		return err
	}
	return w.writeCRLF()
}

// WriteNull 写入 Null Bulk String
func (w *Writer) WriteNull() error {
	if err := w.writer.WriteByte(BulkString); err != nil {
		return err
	}
	if _, err := w.writer.WriteString("-1"); err != nil {
		return err
	}
	return w.writeCRLF()
}

// WriteArray 写入 Array
func (w *Writer) WriteArray(values []Value) error {
	if err := w.writer.WriteByte(Array); err != nil {
		return err
	}
	if _, err := w.writer.WriteString(strconv.Itoa(len(values))); err != nil {
		return err
	}
	if err := w.writeCRLF(); err != nil {
		return err
	}

	for _, value := range values {
		if err := w.WriteValue(&value); err != nil {
			return err
		}
	}

	return nil
}

// WriteNullArray 写入 Null Array
func (w *Writer) WriteNullArray() error {
	if err := w.writer.WriteByte(Array); err != nil {
		return err
	}
	if _, err := w.writer.WriteString("-1"); err != nil {
		return err
	}
	return w.writeCRLF()
}

// writeCRLF 写入 \r\n
func (w *Writer) writeCRLF() error {
	if err := w.writer.WriteByte('\r'); err != nil {
		return err
	}
	return w.writer.WriteByte('\n')
}

// Flush 刷新缓冲区
func (w *Writer) Flush() error {
	return w.writer.Flush()
}
