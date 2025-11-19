# Go 生产环境指南

本指南帮助您在生产环境中部署和优化 Go 应用与 TokenginX 的集成。

## 前置要求

- Go 1.21 或更高版本(Go 1.22+ 推荐)
- 生产环境 TokenginX 服务器集群
- 监控和日志基础设施

## 生产级配置

### 1. 项目结构

```
myapp/
├── cmd/
│   └── server/
│       └── main.go
├── internal/
│   ├── config/
│   │   └── config.go
│   ├── tokenginx/
│   │   ├── client.go
│   │   └── metrics.go
│   └── server/
│       └── server.go
├── pkg/
│   └── logger/
│       └── logger.go
├── config/
│   └── production.yaml
├── Dockerfile
├── go.mod
└── go.sum
```

### 2. 配置管理

创建 `internal/config/config.go`:

```go
package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server    ServerConfig    `yaml:"server"`
	TokenginX TokenginXConfig `yaml:"tokenginx"`
	Log       LogConfig       `yaml:"log"`
	Metrics   MetricsConfig   `yaml:"metrics"`
}

type ServerConfig struct {
	Port         int           `yaml:"port"`
	ReadTimeout  time.Duration `yaml:"read_timeout"`
	WriteTimeout time.Duration `yaml:"write_timeout"`
	IdleTimeout  time.Duration `yaml:"idle_timeout"`
}

type TokenginXConfig struct {
	Addrs        []string      `yaml:"addrs"`
	Password     string        `yaml:"password"`
	PoolSize     int           `yaml:"pool_size"`
	MinIdleConns int           `yaml:"min_idle_conns"`
	DialTimeout  time.Duration `yaml:"dial_timeout"`
	ReadTimeout  time.Duration `yaml:"read_timeout"`
	WriteTimeout time.Duration `yaml:"write_timeout"`
	TLS          TLSConfig     `yaml:"tls"`
}

type TLSConfig struct {
	Enabled  bool   `yaml:"enabled"`
	CertFile string `yaml:"cert_file"`
	KeyFile  string `yaml:"key_file"`
	CAFile   string `yaml:"ca_file"`
}

type LogConfig struct {
	Level  string `yaml:"level"`
	Format string `yaml:"format"`
	Output string `yaml:"output"`
}

type MetricsConfig struct {
	Enabled bool   `yaml:"enabled"`
	Path    string `yaml:"path"`
}

func Load(filename string) (*Config, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// 替换环境变量
	data = []byte(os.ExpandEnv(string(data)))

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	return &cfg, nil
}
```

创建 `config/production.yaml`:

```yaml
server:
  port: 8080
  read_timeout: 10s
  write_timeout: 10s
  idle_timeout: 120s

tokenginx:
  addrs:
    - tokenginx-node1.prod.example.com:6380
    - tokenginx-node2.prod.example.com:6380
    - tokenginx-node3.prod.example.com:6380
  password: ${TOKENGINX_API_KEY}
  pool_size: 50
  min_idle_conns: 10
  dial_timeout: 5s
  read_timeout: 3s
  write_timeout: 3s
  tls:
    enabled: true
    cert_file: /certs/client-cert.pem
    key_file: /certs/client-key.pem
    ca_file: /certs/ca.pem

log:
  level: info
  format: json
  output: /var/log/myapp/application.log

metrics:
  enabled: true
  path: /metrics
```

### 3. TokenginX 客户端封装

创建 `internal/tokenginx/client.go`:

```go
package tokenginx

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

type Client struct {
	rdb     *redis.ClusterClient
	logger  *zap.Logger
	metrics *Metrics
}

type Config struct {
	Addrs        []string
	Password     string
	PoolSize     int
	MinIdleConns int
	DialTimeout  time.Duration
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	TLSConfig    *tls.Config
}

func NewClient(cfg *Config, logger *zap.Logger, metrics *Metrics) (*Client, error) {
	rdb := redis.NewClusterClient(&redis.ClusterOptions{
		Addrs:        cfg.Addrs,
		Password:     cfg.Password,
		PoolSize:     cfg.PoolSize,
		MinIdleConns: cfg.MinIdleConns,
		DialTimeout:  cfg.DialTimeout,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		TLSConfig:    cfg.TLSConfig,

		// 高可用配置
		MaxRetries:      3,
		MinRetryBackoff: 8 * time.Millisecond,
		MaxRetryBackoff: 512 * time.Millisecond,

		// 连接池配置
		PoolTimeout: 4 * time.Second,
		MaxIdleConns: 20,

		// 健康检查
		OnConnect: func(ctx context.Context, cn *redis.Conn) error {
			logger.Info("New connection established")
			return nil
		},
	})

	// 测试连接
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to TokenginX: %w", err)
	}

	logger.Info("Connected to TokenginX cluster",
		zap.Int("nodes", len(cfg.Addrs)))

	return &Client{
		rdb:     rdb,
		logger:  logger,
		metrics: metrics,
	}, nil
}

func (c *Client) Close() error {
	return c.rdb.Close()
}

type OAuthToken struct {
	UserID    string `json:"user_id"`
	Scope     string `json:"scope"`
	CreatedAt int64  `json:"created_at"`
	ClientIP  string `json:"client_ip,omitempty"`
}

func (c *Client) SetOAuthToken(ctx context.Context, tokenID string, token *OAuthToken, ttl time.Duration) error {
	start := time.Now()
	defer func() {
		c.metrics.RecordOperationDuration("set", time.Since(start))
	}()

	key := fmt.Sprintf("oauth:token:%s", tokenID)

	data, err := json.Marshal(token)
	if err != nil {
		return fmt.Errorf("failed to marshal token: %w", err)
	}

	if err := c.rdb.Set(ctx, key, data, ttl).Err(); err != nil {
		c.logger.Error("Failed to set OAuth token",
			zap.String("token_id", tokenID),
			zap.Error(err))
		return fmt.Errorf("failed to set token: %w", err)
	}

	c.metrics.IncTokenCreated()
	c.logger.Info("OAuth token created",
		zap.String("token_id", tokenID),
		zap.String("user_id", token.UserID),
		zap.Duration("ttl", ttl))

	return nil
}

func (c *Client) GetOAuthToken(ctx context.Context, tokenID string) (*OAuthToken, error) {
	start := time.Now()
	defer func() {
		c.metrics.RecordOperationDuration("get", time.Since(start))
	}()

	key := fmt.Sprintf("oauth:token:%s", tokenID)

	data, err := c.rdb.Get(ctx, key).Bytes()
	if err == redis.Nil {
		c.logger.Debug("OAuth token not found", zap.String("token_id", tokenID))
		return nil, nil
	}
	if err != nil {
		c.logger.Error("Failed to get OAuth token",
			zap.String("token_id", tokenID),
			zap.Error(err))
		return nil, fmt.Errorf("failed to get token: %w", err)
	}

	var token OAuthToken
	if err := json.Unmarshal(data, &token); err != nil {
		return nil, fmt.Errorf("failed to unmarshal token: %w", err)
	}

	c.metrics.IncTokenRetrieved()
	c.logger.Debug("OAuth token retrieved", zap.String("token_id", tokenID))

	return &token, nil
}

func (c *Client) GetMultipleTokens(ctx context.Context, tokenIDs []string) (map[string]*OAuthToken, error) {
	start := time.Now()
	defer func() {
		c.metrics.RecordOperationDuration("batch_get", time.Since(start))
	}()

	keys := make([]string, len(tokenIDs))
	for i, id := range tokenIDs {
		keys[i] = fmt.Sprintf("oauth:token:%s", id)
	}

	// 使用 Pipeline
	pipe := c.rdb.Pipeline()
	cmds := make([]*redis.StringCmd, len(keys))
	for i, key := range keys {
		cmds[i] = pipe.Get(ctx, key)
	}

	if _, err := pipe.Exec(ctx); err != nil && err != redis.Nil {
		return nil, fmt.Errorf("failed to execute pipeline: %w", err)
	}

	result := make(map[string]*OAuthToken)
	for i, cmd := range cmds {
		data, err := cmd.Bytes()
		if err == redis.Nil {
			result[tokenIDs[i]] = nil
			continue
		}
		if err != nil {
			c.logger.Warn("Failed to get token in batch",
				zap.String("token_id", tokenIDs[i]),
				zap.Error(err))
			result[tokenIDs[i]] = nil
			continue
		}

		var token OAuthToken
		if err := json.Unmarshal(data, &token); err != nil {
			c.logger.Warn("Failed to unmarshal token",
				zap.String("token_id", tokenIDs[i]),
				zap.Error(err))
			result[tokenIDs[i]] = nil
			continue
		}

		result[tokenIDs[i]] = &token
	}

	c.logger.Debug("Batch retrieved tokens",
		zap.Int("total", len(tokenIDs)),
		zap.Int("found", len(result)))

	return result, nil
}

func (c *Client) DeleteToken(ctx context.Context, tokenID string) error {
	key := fmt.Sprintf("oauth:token:%s", tokenID)

	if err := c.rdb.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("failed to delete token: %w", err)
	}

	c.logger.Info("Token deleted", zap.String("token_id", tokenID))
	return nil
}

func (c *Client) GetTokenTTL(ctx context.Context, tokenID string) (time.Duration, error) {
	key := fmt.Sprintf("oauth:token:%s", tokenID)
	return c.rdb.TTL(ctx, key).Result()
}

func (c *Client) IsHealthy(ctx context.Context) bool {
	err := c.rdb.Ping(ctx).Err()
	return err == nil
}

func LoadTLSConfig(certFile, keyFile, caFile string) (*tls.Config, error) {
	tlsConfig := &tls.Config{
		MinVersion: tls.VersionTLS13,
	}

	// 加载 CA 证书
	if caFile != "" {
		caCert, err := os.ReadFile(caFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read CA file: %w", err)
		}

		caCertPool := x509.NewCertPool()
		if !caCertPool.AppendCertsFromPEM(caCert) {
			return nil, fmt.Errorf("failed to parse CA certificate")
		}

		tlsConfig.RootCAs = caCertPool
	}

	// 加载客户端证书
	if certFile != "" && keyFile != "" {
		cert, err := tls.LoadX509KeyPair(certFile, keyFile)
		if err != nil {
			return nil, fmt.Errorf("failed to load client certificate: %w", err)
		}

		tlsConfig.Certificates = []tls.Certificate{cert}
	}

	return tlsConfig, nil
}
```

### 4. Prometheus 指标

创建 `internal/tokenginx/metrics.go`:

```go
package tokenginx

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type Metrics struct {
	tokenCreated        prometheus.Counter
	tokenRetrieved      prometheus.Counter
	operationDuration   *prometheus.HistogramVec
	connectionPoolSize  prometheus.Gauge
	connectionPoolIdle  prometheus.Gauge
}

func NewMetrics() *Metrics {
	return &Metrics{
		tokenCreated: promauto.NewCounter(prometheus.CounterOpts{
			Name: "tokenginx_token_created_total",
			Help: "Total number of tokens created",
		}),
		tokenRetrieved: promauto.NewCounter(prometheus.CounterOpts{
			Name: "tokenginx_token_retrieved_total",
			Help: "Total number of tokens retrieved",
		}),
		operationDuration: promauto.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "tokenginx_operation_duration_seconds",
			Help:    "Duration of TokenginX operations",
			Buckets: prometheus.ExponentialBuckets(0.001, 2, 10),
		}, []string{"operation"}),
		connectionPoolSize: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "tokenginx_connection_pool_size",
			Help: "Current size of connection pool",
		}),
		connectionPoolIdle: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "tokenginx_connection_pool_idle",
			Help: "Current number of idle connections",
		}),
	}
}

func (m *Metrics) IncTokenCreated() {
	m.tokenCreated.Inc()
}

func (m *Metrics) IncTokenRetrieved() {
	m.tokenRetrieved.Inc()
}

func (m *Metrics) RecordOperationDuration(operation string, duration time.Duration) {
	m.operationDuration.WithLabelValues(operation).Observe(duration.Seconds())
}

func (m *Metrics) SetPoolSize(size int) {
	m.connectionPoolSize.Set(float64(size))
}

func (m *Metrics) SetPoolIdle(idle int) {
	m.connectionPoolIdle.Set(float64(idle))
}
```

### 5. HTTP 服务器

创建 `internal/server/server.go`:

```go
package server

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"

	"myapp/internal/tokenginx"
)

type Server struct {
	router     *mux.Router
	httpServer *http.Server
	tokenginx  *tokenginx.Client
	logger     *zap.Logger
}

func NewServer(cfg *Config, tgx *tokenginx.Client, logger *zap.Logger) *Server {
	s := &Server{
		router:    mux.NewRouter(),
		tokenginx: tgx,
		logger:    logger,
	}

	s.setupRoutes()

	s.httpServer = &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Port),
		Handler:      s.router,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		IdleTimeout:  cfg.IdleTimeout,
	}

	return s
}

func (s *Server) setupRoutes() {
	// 健康检查
	s.router.HandleFunc("/health/live", s.handleLiveness).Methods("GET")
	s.router.HandleFunc("/health/ready", s.handleReadiness).Methods("GET")

	// Prometheus 指标
	s.router.Handle("/metrics", promhttp.Handler())

	// API 路由
	api := s.router.PathPrefix("/api/v1").Subrouter()
	api.HandleFunc("/oauth/token", s.handleCreateToken).Methods("POST")
	api.HandleFunc("/oauth/introspect", s.handleIntrospectToken).Methods("POST")
	api.HandleFunc("/oauth/revoke", s.handleRevokeToken).Methods("POST")
}

func (s *Server) Start() error {
	s.logger.Info("Starting HTTP server", zap.String("addr", s.httpServer.Addr))
	return s.httpServer.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.Info("Shutting down HTTP server")
	return s.httpServer.Shutdown(ctx)
}

func (s *Server) handleLiveness(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "alive"})
}

func (s *Server) handleReadiness(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	if s.tokenginx.IsHealthy(ctx) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "ready"})
	} else {
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]string{"status": "not ready"})
	}
}

// 其他 API 处理函数...
```

### 6. 主程序

创建 `cmd/server/main.go`:

```go
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"

	"myapp/internal/config"
	"myapp/internal/server"
	"myapp/internal/tokenginx"
)

func main() {
	configFile := flag.String("config", "config/production.yaml", "Config file path")
	flag.Parse()

	// 加载配置
	cfg, err := config.Load(*configFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		os.Exit(1)
	}

	// 初始化日志
	logger, err := initLogger(cfg.Log)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to init logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	// 初始化 Prometheus 指标
	metrics := tokenginx.NewMetrics()

	// 加载 TLS 配置
	var tlsConfig *tls.Config
	if cfg.TokenginX.TLS.Enabled {
		tlsConfig, err = tokenginx.LoadTLSConfig(
			cfg.TokenginX.TLS.CertFile,
			cfg.TokenginX.TLS.KeyFile,
			cfg.TokenginX.TLS.CAFile,
		)
		if err != nil {
			logger.Fatal("Failed to load TLS config", zap.Error(err))
		}
	}

	// 创建 TokenginX 客户端
	tgxClient, err := tokenginx.NewClient(&tokenginx.Config{
		Addrs:        cfg.TokenginX.Addrs,
		Password:     cfg.TokenginX.Password,
		PoolSize:     cfg.TokenginX.PoolSize,
		MinIdleConns: cfg.TokenginX.MinIdleConns,
		DialTimeout:  cfg.TokenginX.DialTimeout,
		ReadTimeout:  cfg.TokenginX.ReadTimeout,
		WriteTimeout: cfg.TokenginX.WriteTimeout,
		TLSConfig:    tlsConfig,
	}, logger, metrics)
	if err != nil {
		logger.Fatal("Failed to create TokenginX client", zap.Error(err))
	}
	defer tgxClient.Close()

	// 创建 HTTP 服务器
	srv := server.NewServer(&cfg.Server, tgxClient, logger)

	// 启动服务器
	go func() {
		if err := srv.Start(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("HTTP server error", zap.Error(err))
		}
	}()

	// 优雅关闭
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("Server forced to shutdown", zap.Error(err))
	}

	logger.Info("Server exited")
}

func initLogger(cfg config.LogConfig) (*zap.Logger, error) {
	var zapCfg zap.Config

	if cfg.Format == "json" {
		zapCfg = zap.NewProductionConfig()
	} else {
		zapCfg = zap.NewDevelopmentConfig()
	}

	zapCfg.OutputPaths = []string{cfg.Output}

	level := zap.NewAtomicLevel()
	if err := level.UnmarshalText([]byte(cfg.Level)); err != nil {
		return nil, err
	}
	zapCfg.Level = level

	return zapCfg.Build()
}
```

## Docker 部署

### Dockerfile

```dockerfile
FROM golang:1.22-alpine AS builder

WORKDIR /build

# 安装依赖
RUN apk add --no-cache git ca-certificates tzdata

# 复制 go.mod 和 go.sum
COPY go.mod go.sum ./
RUN go mod download

# 复制源代码
COPY . .

# 构建
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s" \
    -o /app/server \
    ./cmd/server

FROM scratch

# 从 builder 复制必要文件
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /app/server /server
COPY --from=builder /build/config /config

EXPOSE 8080

ENTRYPOINT ["/server"]
CMD ["-config", "/config/production.yaml"]
```

## Kubernetes 部署

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: myapp-go
spec:
  replicas: 3
  selector:
    matchLabels:
      app: myapp-go
  template:
    metadata:
      labels:
        app: myapp-go
    spec:
      containers:
      - name: myapp
        image: myapp-go:1.0.0
        ports:
        - containerPort: 8080
        env:
        - name: TOKENGINX_API_KEY
          valueFrom:
            secretKeyRef:
              name: tokenginx-secret
              key: api-key
        - name: GOMAXPROCS
          value: "2"
        - name: GOMEMLIMIT
          value: "512MiB"
        livenessProbe:
          httpGet:
            path: /health/live
            port: 8080
          initialDelaySeconds: 10
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /health/ready
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 5
        resources:
          requests:
            memory: "256Mi"
            cpu: "100m"
          limits:
            memory: "512Mi"
            cpu: "500m"
```

## 最佳实践清单

- ✅ 使用 context 控制超时和取消
- ✅ 启用 TLS/mTLS 加密通信
- ✅ 使用结构化日志(zap)
- ✅ 添加 Prometheus 监控指标
- ✅ 实现优雅关闭(Graceful Shutdown)
- ✅ 使用连接池配置
- ✅ 使用 Pipeline 批量操作
- ✅ 设置 GOMAXPROCS 和 GOMEMLIMIT
- ✅ 使用健康检查(Liveness/Readiness)
- ✅ 实现错误重试机制
- ✅ 使用 scratch 基础镜像减小镜像大小

## 下一步

- 查看 [OAuth 2.0 集成](../protocols/oauth.md)
- 配置 [TLS/mTLS](../security/tls-mtls.md)
- 了解 [防重放攻击](../security/anti-replay.md)
