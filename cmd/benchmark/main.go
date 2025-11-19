package main

import (
	"flag"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-redis/redis/v8"
	"golang.org/x/net/context"
)

var (
	addr        = flag.String("addr", "127.0.0.1:6380", "服务器地址")
	clients     = flag.Int("clients", 10, "并发客户端数")
	requests    = flag.Int("requests", 10000, "每个客户端的请求数")
	keySize     = flag.Int("keysize", 10, "键名长度")
	valueSize   = flag.Int("valuesize", 100, "值长度")
	testCommand = flag.String("command", "set", "测试命令 (set|get|mixed)")
)

type BenchmarkResult struct {
	TotalRequests int64
	TotalTime     time.Duration
	MinLatency    time.Duration
	MaxLatency    time.Duration
	AvgLatency    time.Duration
	QPS           float64
	Errors        int64
}

func main() {
	flag.Parse()

	fmt.Println("TokenginX 性能测试工具")
	fmt.Println("====================")
	fmt.Printf("服务器地址: %s\n", *addr)
	fmt.Printf("并发客户端: %d\n", *clients)
	fmt.Printf("每客户端请求数: %d\n", *requests)
	fmt.Printf("键大小: %d 字节\n", *keySize)
	fmt.Printf("值大小: %d 字节\n", *valueSize)
	fmt.Printf("测试命令: %s\n", *testCommand)
	fmt.Println()

	// 测试连接
	ctx := context.Background()
	client := redis.NewClient(&redis.Options{
		Addr: *addr,
	})

	if err := client.Ping(ctx).Err(); err != nil {
		log.Fatalf("无法连接到服务器: %v", err)
	}
	client.Close()

	fmt.Println("开始性能测试...")
	fmt.Println()

	switch *testCommand {
	case "set":
		result := benchmarkSet()
		printResult("SET", result)
	case "get":
		result := benchmarkGet()
		printResult("GET", result)
	case "mixed":
		result := benchmarkMixed()
		printResult("Mixed (50% SET + 50% GET)", result)
	default:
		log.Fatalf("未知命令: %s", *testCommand)
	}
}

func benchmarkSet() BenchmarkResult {
	return runBenchmark(func(client *redis.Client, ctx context.Context, key, value string) error {
		return client.Set(ctx, key, value, 0).Err()
	})
}

func benchmarkGet() BenchmarkResult {
	// 先填充一些数据
	ctx := context.Background()
	client := redis.NewClient(&redis.Options{Addr: *addr})
	for i := 0; i < *requests; i++ {
		key := fmt.Sprintf("key:%d", i)
		value := generateValue(*valueSize)
		client.Set(ctx, key, value, 0)
	}
	client.Close()

	return runBenchmark(func(client *redis.Client, ctx context.Context, key, value string) error {
		return client.Get(ctx, key).Err()
	})
}

func benchmarkMixed() BenchmarkResult {
	return runBenchmark(func(client *redis.Client, ctx context.Context, key, value string) error {
		// 50% SET, 50% GET
		if time.Now().UnixNano()%2 == 0 {
			return client.Set(ctx, key, value, 0).Err()
		}
		return client.Get(ctx, key).Err()
	})
}

func runBenchmark(op func(*redis.Client, context.Context, string, string) error) BenchmarkResult {
	var (
		totalRequests atomic.Int64
		totalErrors   atomic.Int64
		totalLatency  atomic.Int64
		minLatency    int64 = 1<<63 - 1
		maxLatency    int64
	)

	start := time.Now()
	var wg sync.WaitGroup

	for i := 0; i < *clients; i++ {
		wg.Add(1)
		go func(clientID int) {
			defer wg.Done()

			client := redis.NewClient(&redis.Options{
				Addr: *addr,
			})
			defer client.Close()

			ctx := context.Background()

			for j := 0; j < *requests; j++ {
				key := fmt.Sprintf("key:%d:%d", clientID, j)
				value := generateValue(*valueSize)

				reqStart := time.Now()
				err := op(client, ctx, key, value)
				latency := time.Since(reqStart)

				totalRequests.Add(1)
				totalLatency.Add(int64(latency))

				if err != nil {
					totalErrors.Add(1)
				}

				// 更新最小/最大延迟
				lat := int64(latency)
				for {
					old := atomic.LoadInt64(&minLatency)
					if lat >= old || atomic.CompareAndSwapInt64(&minLatency, old, lat) {
						break
					}
				}
				for {
					old := atomic.LoadInt64(&maxLatency)
					if lat <= old || atomic.CompareAndSwapInt64(&maxLatency, old, lat) {
						break
					}
				}
			}
		}(i)
	}

	wg.Wait()
	duration := time.Since(start)

	requests := totalRequests.Load()
	avgLatency := time.Duration(totalLatency.Load() / requests)
	qps := float64(requests) / duration.Seconds()

	return BenchmarkResult{
		TotalRequests: requests,
		TotalTime:     duration,
		MinLatency:    time.Duration(minLatency),
		MaxLatency:    time.Duration(maxLatency),
		AvgLatency:    avgLatency,
		QPS:           qps,
		Errors:        totalErrors.Load(),
	}
}

func generateValue(size int) string {
	const chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, size)
	for i := range b {
		b[i] = chars[i%len(chars)]
	}
	return string(b)
}

func printResult(name string, result BenchmarkResult) {
	fmt.Printf("=== %s 性能测试结果 ===\n", name)
	fmt.Printf("总请求数: %d\n", result.TotalRequests)
	fmt.Printf("总耗时: %v\n", result.TotalTime)
	fmt.Printf("QPS: %.2f 请求/秒\n", result.QPS)
	fmt.Printf("平均延迟: %v\n", result.AvgLatency)
	fmt.Printf("最小延迟: %v\n", result.MinLatency)
	fmt.Printf("最大延迟: %v\n", result.MaxLatency)
	fmt.Printf("错误数: %d (%.2f%%)\n", result.Errors, float64(result.Errors)/float64(result.TotalRequests)*100)
	fmt.Println()
}
