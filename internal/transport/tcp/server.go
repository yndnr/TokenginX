package tcp

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/yndnr/tokenginx/internal/storage"
	"github.com/yndnr/tokenginx/internal/transport/resp"
)

// Server TCP 服务器
//
// Server 实现了一个基于 RESP 协议的 TCP 服务器，用于处理客户端连接和命令请求。
// 每个客户端连接由独立的 Goroutine 处理，支持高并发。
//
// 示例：
//
//	sm := storage.NewShardedMap(4096)
//	server := NewServer(":6380", sm)
//	if err := server.Start(); err != nil {
//	    log.Fatal(err)
//	}
//	defer server.Stop()
type Server struct {
	addr     string              // 监听地址
	listener net.Listener        // TCP 监听器
	sm       *storage.ShardedMap // 存储引擎
	handler  *CommandHandler     // 命令处理器

	// 状态管理
	running  atomic.Bool   // 服务器是否运行中
	wg       sync.WaitGroup // 等待所有连接关闭
	stopCh   chan struct{} // 停止信号
	ctx      context.Context
	cancelFn context.CancelFunc

	// 统计信息
	totalConnections atomic.Int64 // 总连接数
	activeClients    atomic.Int64 // 当前活跃连接数
	totalCommands    atomic.Int64 // 总命令数
}

// NewServer 创建一个新的 TCP 服务器
//
// 参数说明：
//   - addr: 监听地址，格式为 "host:port"，如 ":6380" 或 "0.0.0.0:6380"
//   - sm: 存储引擎实例
//
// 返回值：
//   - *Server: TCP 服务器实例
//
// 示例：
//
//	sm := storage.NewShardedMap(4096)
//	server := NewServer(":6380", sm)
//	if err := server.Start(); err != nil {
//	    log.Fatalf("服务器启动失败: %v", err)
//	}
//
// 注意事项：
//   - 创建后需要调用 Start() 启动服务器
//   - 使用完毕后应调用 Stop() 优雅关闭
func NewServer(addr string, sm *storage.ShardedMap) *Server {
	ctx, cancel := context.WithCancel(context.Background())

	return &Server{
		addr:     addr,
		sm:       sm,
		handler:  NewCommandHandler(sm),
		stopCh:   make(chan struct{}),
		ctx:      ctx,
		cancelFn: cancel,
	}
}

// Start 启动 TCP 服务器
//
// 返回值：
//   - error: 启动失败时的错误信息
//
// 示例：
//
//	if err := server.Start(); err != nil {
//	    log.Fatalf("启动失败: %v", err)
//	}
//
// 注意事项：
//   - 该方法是非阻塞的，服务器在后台运行
//   - 如果端口已被占��，会返回错误
//   - 多次调用 Start 不会启动多个服务器
func (s *Server) Start() error {
	if s.running.Load() {
		return fmt.Errorf("服务器已在运行")
	}

	listener, err := net.Listen("tcp", s.addr)
	if err != nil {
		return fmt.Errorf("监听失败: %w", err)
	}

	s.listener = listener
	s.running.Store(true)

	log.Printf("[INFO] TCP 服务器启动成功，监听地址: %s", s.addr)

	// 启动接受连接的 Goroutine
	s.wg.Add(1)
	go s.acceptLoop()

	return nil
}

// Stop 停止 TCP 服务器
//
// 该方法会优雅地关闭服务器：
//   1. 停止接受新连接
//   2. 等待所有现有连接处理完成
//   3. 关闭监听器
//
// 注意事项：
//   - 该方法会阻塞直到所有连接关闭
//   - 多次调用 Stop 是安全的
func (s *Server) Stop() {
	if !s.running.Load() {
		return
	}

	log.Println("[INFO] 正在停止 TCP 服务器...")

	// 取消上下文，通知所有连接关闭
	s.cancelFn()

	// 关闭监听器，停止接受新连接
	if s.listener != nil {
		s.listener.Close()
	}

	// 等待所有连接处理完成
	s.wg.Wait()

	s.running.Store(false)
	log.Println("[INFO] TCP 服务器已停止")
}

// acceptLoop 接受客户端连接的循环
func (s *Server) acceptLoop() {
	defer s.wg.Done()

	for {
		conn, err := s.listener.Accept()
		if err != nil {
			// 检查是否因为服务器关闭而出错
			select {
			case <-s.ctx.Done():
				return
			default:
				log.Printf("[ERROR] 接受连接失败: %v", err)
				continue
			}
		}

		// 为每个连接启动独立的 Goroutine
		s.wg.Add(1)
		go s.handleConnection(conn)
	}
}

// handleConnection 处理单个客户端连接
func (s *Server) handleConnection(conn net.Conn) {
	defer s.wg.Done()
	defer conn.Close()

	// 更新统计信息
	s.totalConnections.Add(1)
	s.activeClients.Add(1)
	defer s.activeClients.Add(-1)

	clientAddr := conn.RemoteAddr().String()
	log.Printf("[INFO] 新连接: %s (活跃连接: %d)", clientAddr, s.activeClients.Load())

	// 创建 RESP 解析器和写入器
	parser := resp.NewParser(conn)
	writer := resp.NewWriter(conn)

	// 连接上下文（用于超时控制）
	ctx, cancel := context.WithCancel(s.ctx)
	defer cancel()

	// 读取命令循环
	for {
		select {
		case <-ctx.Done():
			// 服务器正在关闭或连接超时
			return
		default:
		}

		// 设置读取超时（5 分钟无活动则断开）
		conn.SetReadDeadline(time.Now().Add(5 * time.Minute))

		// 解析 RESP 命令
		value, err := parser.Parse()
		if err != nil {
			if err == io.EOF {
				// 客户端关闭连接
				log.Printf("[INFO] 连接关闭: %s", clientAddr)
				return
			}

			// 解析错误
			log.Printf("[ERROR] 解析命令失败 (%s): %v", clientAddr, err)
			writer.WriteError(fmt.Sprintf("ERR 协议错误: %v", err))
			writer.Flush()
			return
		}

		// 处理命令并返回响应
		response := s.handler.HandleCommand(value)
		if err := writer.WriteValue(response); err != nil {
			log.Printf("[ERROR] 写入响应失败 (%s): %v", clientAddr, err)
			return
		}

		if err := writer.Flush(); err != nil {
			log.Printf("[ERROR] 刷新缓冲区失败 (%s): %v", clientAddr, err)
			return
		}

		// 更新统计信息
		s.totalCommands.Add(1)
	}
}

// GetStats 获取服务器统计信息
type ServerStats struct {
	Running          bool   // 是否运行中
	ListenAddr       string // 监听地址
	TotalConnections int64  // 总连接数
	ActiveClients    int64  // 当前活跃连接数
	TotalCommands    int64  // 总命令数
}

// GetStats 返回服务器的统计信息
//
// 返回值：
//   - ServerStats: 服务器统计信息
//
// 示例：
//
//	stats := server.GetStats()
//	log.Printf("活跃连接: %d, 总命令: %d", stats.ActiveClients, stats.TotalCommands)
func (s *Server) GetStats() ServerStats {
	return ServerStats{
		Running:          s.running.Load(),
		ListenAddr:       s.addr,
		TotalConnections: s.totalConnections.Load(),
		ActiveClients:    s.activeClients.Load(),
		TotalCommands:    s.totalCommands.Load(),
	}
}
