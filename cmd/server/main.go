package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/yndnr/tokenginx/internal/storage"
	"github.com/yndnr/tokenginx/internal/transport/tcp"
)

const (
	// Version 版本号
	Version = "v0.1.0-dev"

	// DefaultAddr 默认监听地址
	DefaultAddr = ":6380"

	// DefaultShardCount 默认分片数
	DefaultShardCount = 4096

	// DefaultCleanupInterval 默认 TTL 清理间隔
	DefaultCleanupInterval = 1 * time.Second

	// DefaultKeysPerScan 默认每次扫描的键数
	DefaultKeysPerScan = 100
)

var (
	// 命令行参数
	addr            = flag.String("addr", DefaultAddr, "监听地址 (例如: :6380 或 0.0.0.0:6380)")
	shardCount      = flag.Int("shards", DefaultShardCount, "分片数量 (2的幂次)")
	cleanupInterval = flag.Duration("cleanup-interval", DefaultCleanupInterval, "TTL 清理间隔")
	keysPerScan     = flag.Int("keys-per-scan", DefaultKeysPerScan, "每次扫描清理的键数")
	showVersion     = flag.Bool("version", false, "显示版本信息")
	showHelp        = flag.Bool("help", false, "显示帮助信息")
)

func main() {
	flag.Parse()

	// 显示版本信息
	if *showVersion {
		fmt.Printf("TokenginX %s\n", Version)
		fmt.Println("专为 SSO 优化的高性能会话存储系统")
		os.Exit(0)
	}

	// 显示帮助信息
	if *showHelp {
		printHelp()
		os.Exit(0)
	}

	// 打印启动信息
	printBanner()
	log.Printf("[INFO] TokenginX %s 正在启动...", Version)
	log.Printf("[INFO] 配置: 监听地址=%s, 分片数=%d, 清理间隔=%v, 每次扫描=%d",
		*addr, *shardCount, *cleanupInterval, *keysPerScan)

	// 创建存储引擎
	log.Println("[INFO] 初始化存储引擎...")
	sm := storage.NewShardedMap(*shardCount)

	// 创建并启动 TTL 管理器
	log.Println("[INFO] 启动 TTL 管理器...")
	ttlConfig := &storage.TTLManagerConfig{
		CleanupInterval: *cleanupInterval,
		KeysPerScan:     *keysPerScan,
	}
	ttlManager := storage.NewTTLManager(sm, ttlConfig)
	ttlManager.Start()
	defer ttlManager.Stop()

	// 创建并启动 TCP 服务器
	log.Println("[INFO] 启动 TCP 服务器...")
	server := tcp.NewServer(*addr, sm)
	if err := server.Start(); err != nil {
		log.Fatalf("[FATAL] 服务器启动失败: %v", err)
	}
	defer server.Stop()

	log.Printf("[INFO] ✓ TokenginX 启动成功!")
	log.Printf("[INFO] ✓ 监听地址: %s", *addr)
	log.Printf("[INFO] ✓ 使用 redis-cli 或其他 Redis 客户端连接")
	log.Printf("[INFO] ✓ 示例: redis-cli -h 127.0.0.1 -p 6380")

	// 启动统计信息输出 Goroutine
	go printStats(server)

	// 等待退出信号
	waitForShutdown()

	log.Println("[INFO] TokenginX 已退出")
}

// printBanner 打印启动横幅
func printBanner() {
	banner := `
  _____     _               _____ _     __  __
 |_   _|__ | | _____ _ __ |  ___(_)_ _\ \/ /
   | |/ _ \| |/ / _ \ '_ \| |_  | | '_ \\  /
   | | (_) |   <  __/ | | |  _| | | | | /  \
   |_|\___/|_|\_\___|_| |_|_|   |_|_| |_/_/\_\

   专为 SSO 优化的高性能会话存储系统
   https://github.com/yndnr/tokenginx
`
	fmt.Println(banner)
}

// printHelp 打印帮助信息
func printHelp() {
	fmt.Println("TokenginX - 专为 SSO 优化的高性能会话存储系统")
	fmt.Println()
	fmt.Println("用法:")
	fmt.Printf("  %s [选项]\n", os.Args[0])
	fmt.Println()
	fmt.Println("选项:")
	flag.PrintDefaults()
	fmt.Println()
	fmt.Println("示例:")
	fmt.Printf("  %s                                # 使用默认配置启动\n", os.Args[0])
	fmt.Printf("  %s -addr :6380                    # 指定监听端口\n", os.Args[0])
	fmt.Printf("  %s -shards 8192                   # 使用 8192 个分片\n", os.Args[0])
	fmt.Printf("  %s -cleanup-interval 500ms        # 每 500ms 清理一次过期键\n", os.Args[0])
	fmt.Printf("  %s -keys-per-scan 200             # 每次扫描 200 个键\n", os.Args[0])
	fmt.Println()
	fmt.Println("环境变量:")
	fmt.Println("  无")
	fmt.Println()
	fmt.Println("支持的命令:")
	fmt.Println("  PING [message]           - 测试连接")
	fmt.Println("  ECHO message             - 回显消息")
	fmt.Println("  GET key                  - 获取键值")
	fmt.Println("  SET key value [EX sec]   - 设置键值（可选过期时间）")
	fmt.Println("  DEL key [key ...]        - 删除键")
	fmt.Println("  EXISTS key [key ...]     - 检查键是否存在")
	fmt.Println("  TTL key                  - 获取键的剩余生存时间")
	fmt.Println("  EXPIRE key seconds       - 设置键的过期时间")
	fmt.Println()
	fmt.Println("连接示例:")
	fmt.Println("  redis-cli -h 127.0.0.1 -p 6380")
	fmt.Println("  redis-cli -h 127.0.0.1 -p 6380 PING")
	fmt.Println("  redis-cli -h 127.0.0.1 -p 6380 SET mykey myvalue")
	fmt.Println("  redis-cli -h 127.0.0.1 -p 6380 GET mykey")
}

// waitForShutdown 等待退出信号
func waitForShutdown() {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)

	sig := <-sigCh
	log.Printf("[INFO] 收到信号: %v，正在关闭...", sig)
}

// printStats 定期打印统计信息
func printStats(server *tcp.Server) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		stats := server.GetStats()
		if !stats.Running {
			return
		}

		log.Printf("[STATS] 总连接: %d, 活跃连接: %d, 总命令: %d",
			stats.TotalConnections, stats.ActiveClients, stats.TotalCommands)
	}
}
