# Go 快速指南

本指南帮助您快速在 Go 应用中集成 TokenginX。

## 前置要求

- Go 1.19 或更高版本
- TokenginX 服务器已运行

## 安装客户端库

TokenginX 支持标准的 Redis 协议,可以使用 go-redis 客户端。

```bash
go get github.com/redis/go-redis/v9
```

## 使用 go-redis 客户端

### 1. 创建客户端

创建 `tokenginx/client.go`:

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
)

// Client TokenginX 客户端封装
type Client struct {
    rdb *redis.Client
}

// Config 客户端配置
type Config struct {
    Addr     string
    Password string
    TLSConfig *tls.Config
}

// NewClient 创建 TokenginX 客户端
//
// 参数说明:
//   - cfg: 客户端配置
//
// 返回值:
//   - *Client: 客户端实例
//   - error: 错误信息,nil 表示成功
func NewClient(cfg Config) (*Client, error) {
    rdb := redis.NewClient(&redis.Options{
        Addr:         cfg.Addr,
        Password:     cfg.Password,
        DB:           0,
        DialTimeout:  5 * time.Second,
        ReadTimeout:  5 * time.Second,
        WriteTimeout: 5 * time.Second,
        PoolSize:     10,
        MinIdleConns: 5,
        TLSConfig:    cfg.TLSConfig,
    })

    // 测试连接
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    if err := rdb.Ping(ctx).Err(); err != nil {
        return nil, fmt.Errorf("failed to connect to TokenginX: %w", err)
    }

    return &Client{rdb: rdb}, nil
}

// Close 关闭客户端连接
func (c *Client) Close() error {
    return c.rdb.Close()
}

// OAuthToken OAuth Token 数据结构
type OAuthToken struct {
    UserID    string `json:"user_id"`
    Scope     string `json:"scope"`
    CreatedAt int64  `json:"created_at"`
}

// SetOAuthToken 设置 OAuth Token
//
// 参数说明:
//   - ctx: 上下文
//   - tokenID: Token 唯一标识
//   - token: Token 数据
//   - ttl: 过期时间
//
// 返回值:
//   - error: 错误信息,nil 表示成功
func (c *Client) SetOAuthToken(ctx context.Context, tokenID string, token *OAuthToken, ttl time.Duration) error {
    key := fmt.Sprintf("oauth:token:%s", tokenID)

    data, err := json.Marshal(token)
    if err != nil {
        return fmt.Errorf("failed to marshal token: %w", err)
    }

    if err := c.rdb.Set(ctx, key, data, ttl).Err(); err != nil {
        return fmt.Errorf("failed to set token: %w", err)
    }

    return nil
}

// GetOAuthToken 获取 OAuth Token
//
// 参数说明:
//   - ctx: 上下文
//   - tokenID: Token 唯一标识
//
// 返回值:
//   - *OAuthToken: Token 数据
//   - error: 错误信息,nil 表示成功
func (c *Client) GetOAuthToken(ctx context.Context, tokenID string) (*OAuthToken, error) {
    key := fmt.Sprintf("oauth:token:%s", tokenID)

    data, err := c.rdb.Get(ctx, key).Bytes()
    if err == redis.Nil {
        return nil, nil // Token 不存在或已过期
    }
    if err != nil {
        return nil, fmt.Errorf("failed to get token: %w", err)
    }

    var token OAuthToken
    if err := json.Unmarshal(data, &token); err != nil {
        return nil, fmt.Errorf("failed to unmarshal token: %w", err)
    }

    return &token, nil
}

// DeleteToken 删除 Token
//
// 参数说明:
//   - ctx: 上下文
//   - tokenID: Token 唯一标识
//
// 返回值:
//   - error: 错误信息,nil 表示成功
func (c *Client) DeleteToken(ctx context.Context, tokenID string) error {
    key := fmt.Sprintf("oauth:token:%s", tokenID)
    return c.rdb.Del(ctx, key).Err()
}

// TokenExists 检查 Token 是否存在
//
// 参数说明:
//   - ctx: 上下文
//   - tokenID: Token 唯一标识
//
// 返回值:
//   - bool: 是否存在
//   - error: 错误信息,nil 表示成功
func (c *Client) TokenExists(ctx context.Context, tokenID string) (bool, error) {
    key := fmt.Sprintf("oauth:token:%s", tokenID)
    count, err := c.rdb.Exists(ctx, key).Result()
    if err != nil {
        return false, err
    }
    return count > 0, nil
}

// GetTokenTTL 获取 Token 剩余 TTL
//
// 参数说明:
//   - ctx: 上下文
//   - tokenID: Token 唯一标识
//
// 返回值:
//   - time.Duration: 剩余时间
//   - error: 错误信息,nil 表示成功
func (c *Client) GetTokenTTL(ctx context.Context, tokenID string) (time.Duration, error) {
    key := fmt.Sprintf("oauth:token:%s", tokenID)
    return c.rdb.TTL(ctx, key).Result()
}

// GetMultipleTokens 批量获取 Tokens
//
// 参数说明:
//   - ctx: 上下文
//   - tokenIDs: Token ID 列表
//
// 返回值:
//   - map[string]*OAuthToken: Token 映射,不存在的 key 对应 nil
//   - error: 错误信息,nil 表示成功
func (c *Client) GetMultipleTokens(ctx context.Context, tokenIDs []string) (map[string]*OAuthToken, error) {
    keys := make([]string, len(tokenIDs))
    for i, id := range tokenIDs {
        keys[i] = fmt.Sprintf("oauth:token:%s", id)
    }

    values, err := c.rdb.MGet(ctx, keys...).Result()
    if err != nil {
        return nil, fmt.Errorf("failed to get multiple tokens: %w", err)
    }

    result := make(map[string]*OAuthToken)
    for i, tokenID := range tokenIDs {
        if values[i] == nil {
            result[tokenID] = nil
            continue
        }

        data, ok := values[i].(string)
        if !ok {
            result[tokenID] = nil
            continue
        }

        var token OAuthToken
        if err := json.Unmarshal([]byte(data), &token); err != nil {
            result[tokenID] = nil
            continue
        }

        result[tokenID] = &token
    }

    return result, nil
}

// ScanKeys 扫描匹配的键
//
// 参数说明:
//   - ctx: 上下文
//   - pattern: 匹配模式
//   - count: 每次迭代返回的数量提示
//
// 返回值:
//   - []string: 匹配的键列表
//   - error: 错误信息,nil 表示成功
func (c *Client) ScanKeys(ctx context.Context, pattern string, count int64) ([]string, error) {
    var keys []string
    var cursor uint64

    for {
        var batch []string
        var err error

        batch, cursor, err = c.rdb.Scan(ctx, cursor, pattern, count).Result()
        if err != nil {
            return nil, fmt.Errorf("failed to scan keys: %w", err)
        }

        keys = append(keys, batch...)

        if cursor == 0 {
            break
        }
    }

    return keys, nil
}

// LoadTLSConfig 加载 TLS 配置
//
// 参数说明:
//   - caFile: CA 证书文件路径
//   - certFile: 客户端证书文件路径(mTLS 时需要)
//   - keyFile: 客户端私钥文件路径(mTLS 时需要)
//
// 返回值:
//   - *tls.Config: TLS 配置
//   - error: 错误信息,nil 表示成功
func LoadTLSConfig(caFile, certFile, keyFile string) (*tls.Config, error) {
    tlsConfig := &tls.Config{
        MinVersion: tls.VersionTLS13,
    }

    // 加载 CA 证书
    if caFile != "" {
        caCert, err := os.ReadFile(caFile)
        if err != nil {
            return nil, fmt.Errorf("failed to read ca file: %w", err)
        }

        caCertPool := x509.NewCertPool()
        if !caCertPool.AppendCertsFromPEM(caCert) {
            return nil, fmt.Errorf("failed to parse ca certificate")
        }

        tlsConfig.RootCAs = caCertPool
    }

    // 加载客户端证书(mTLS)
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

### 2. 在应用中使用

创建 `main.go`:

```go
package main

import (
    "context"
    "crypto/rand"
    "encoding/hex"
    "fmt"
    "log"
    "os"
    "time"

    "your-module/tokenginx"
)

func main() {
    // 加载 TLS 配置
    tlsConfig, err := tokenginx.LoadTLSConfig(
        "/path/to/ca.pem",
        "/path/to/client-cert.pem",
        "/path/to/client-key.pem",
    )
    if err != nil {
        log.Fatalf("Failed to load TLS config: %v", err)
    }

    // 创建客户端
    client, err := tokenginx.NewClient(tokenginx.Config{
        Addr:      "localhost:6380",
        Password:  os.Getenv("TOKENGINX_API_KEY"),
        TLSConfig: tlsConfig,
    })
    if err != nil {
        log.Fatalf("Failed to create client: %v", err)
    }
    defer client.Close()

    ctx := context.Background()

    // 创建 Token
    tokenID := generateTokenID()
    token := &tokenginx.OAuthToken{
        UserID:    "user001",
        Scope:     "read write",
        CreatedAt: time.Now().Unix(),
    }

    if err := client.SetOAuthToken(ctx, tokenID, token, time.Hour); err != nil {
        log.Fatalf("Failed to set token: %v", err)
    }
    fmt.Printf("Token created: %s\n", tokenID)

    // 验证 Token
    retrievedToken, err := client.GetOAuthToken(ctx, tokenID)
    if err != nil {
        log.Fatalf("Failed to get token: %v", err)
    }

    if retrievedToken != nil {
        fmt.Printf("Token valid:\n")
        fmt.Printf("  User ID: %s\n", retrievedToken.UserID)
        fmt.Printf("  Scope: %s\n", retrievedToken.Scope)

        ttl, _ := client.GetTokenTTL(ctx, tokenID)
        fmt.Printf("  TTL: %v\n", ttl)
    } else {
        fmt.Println("Token not found or expired")
    }

    // 撤销 Token
    if err := client.DeleteToken(ctx, tokenID); err != nil {
        log.Fatalf("Failed to delete token: %v", err)
    }
    fmt.Println("Token revoked")
}

func generateTokenID() string {
    b := make([]byte, 16)
    rand.Read(b)
    return hex.EncodeToString(b)
}
```

## HTTP 服务器集成

使用 Gin 框架创建 OAuth 服务:

```go
package main

import (
    "context"
    "net/http"
    "time"

    "github.com/gin-gonic/gin"
    "your-module/tokenginx"
)

type AuthService struct {
    tokenginx *tokenginx.Client
}

func NewAuthService(client *tokenginx.Client) *AuthService {
    return &AuthService{tokenginx: client}
}

// CreateTokenRequest 创建 Token 请求
type CreateTokenRequest struct {
    UserID string `json:"user_id" binding:"required"`
    Scope  string `json:"scope" binding:"required"`
}

// CreateTokenResponse 创建 Token 响应
type CreateTokenResponse struct {
    AccessToken string `json:"access_token"`
}

// CreateToken 创建 Token 端点
func (s *AuthService) CreateToken(c *gin.Context) {
    var req CreateTokenRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    tokenID := generateTokenID()
    token := &tokenginx.OAuthToken{
        UserID:    req.UserID,
        Scope:     req.Scope,
        CreatedAt: time.Now().Unix(),
    }

    ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
    defer cancel()

    if err := s.tokenginx.SetOAuthToken(ctx, tokenID, token, time.Hour); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create token"})
        return
    }

    c.JSON(http.StatusOK, CreateTokenResponse{AccessToken: tokenID})
}

// IntrospectTokenRequest Token 内省请求
type IntrospectTokenRequest struct {
    Token string `json:"token" binding:"required"`
}

// IntrospectTokenResponse Token 内省响应
type IntrospectTokenResponse struct {
    Active bool   `json:"active"`
    UserID string `json:"user_id,omitempty"`
    Scope  string `json:"scope,omitempty"`
}

// IntrospectToken Token 内省端点
func (s *AuthService) IntrospectToken(c *gin.Context) {
    var req IntrospectTokenRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
    defer cancel()

    token, err := s.tokenginx.GetOAuthToken(ctx, req.Token)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to introspect token"})
        return
    }

    if token == nil {
        c.JSON(http.StatusOK, IntrospectTokenResponse{Active: false})
        return
    }

    c.JSON(http.StatusOK, IntrospectTokenResponse{
        Active: true,
        UserID: token.UserID,
        Scope:  token.Scope,
    })
}

// RevokeTokenRequest 撤销 Token 请求
type RevokeTokenRequest struct {
    Token string `json:"token" binding:"required"`
}

// RevokeToken 撤销 Token 端点
func (s *AuthService) RevokeToken(c *gin.Context) {
    var req RevokeTokenRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
    defer cancel()

    if err := s.tokenginx.DeleteToken(ctx, req.Token); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to revoke token"})
        return
    }

    c.JSON(http.StatusOK, gin.H{"message": "Token revoked"})
}

func main() {
    // 创建 TokenginX 客户端
    client, err := tokenginx.NewClient(tokenginx.Config{
        Addr:     "localhost:6380",
        Password: os.Getenv("TOKENGINX_API_KEY"),
    })
    if err != nil {
        log.Fatalf("Failed to create client: %v", err)
    }
    defer client.Close()

    // 创建认证服务
    authService := NewAuthService(client)

    // 创建 Gin 路由
    r := gin.Default()

    r.POST("/oauth/token", authService.CreateToken)
    r.POST("/oauth/introspect", authService.IntrospectToken)
    r.POST("/oauth/revoke", authService.RevokeToken)

    r.Run(":8080")
}
```

## 中间件集成

创建认证中间件:

```go
package middleware

import (
    "context"
    "net/http"
    "strings"
    "time"

    "github.com/gin-gonic/gin"
    "your-module/tokenginx"
)

// TokenginXAuth TokenginX 认证中间件
func TokenginXAuth(client *tokenginx.Client) gin.HandlerFunc {
    return func(c *gin.Context) {
        authHeader := c.GetHeader("Authorization")

        if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
            c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
            c.Abort()
            return
        }

        token := strings.TrimPrefix(authHeader, "Bearer ")

        ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
        defer cancel()

        tokenData, err := client.GetOAuthToken(ctx, token)
        if err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
            c.Abort()
            return
        }

        if tokenData == nil {
            c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
            c.Abort()
            return
        }

        // 将用户信息添加到上下文
        c.Set("user_id", tokenData.UserID)
        c.Set("scope", tokenData.Scope)

        c.Next()
    }
}

// 使用中间件
func main() {
    client, _ := tokenginx.NewClient(tokenginx.Config{
        Addr: "localhost:6380",
    })

    r := gin.Default()

    // 受保护的路由
    protected := r.Group("/api")
    protected.Use(TokenginXAuth(client))
    {
        protected.GET("/profile", func(c *gin.Context) {
            userID := c.GetString("user_id")
            c.JSON(http.StatusOK, gin.H{"user_id": userID})
        })
    }

    r.Run(":8080")
}
```

## 错误处理与重试

```go
package tokenginx

import (
    "context"
    "errors"
    "time"
)

// GetOAuthTokenWithRetry 带重试的获取 Token
//
// 参数说明:
//   - ctx: 上下文
//   - tokenID: Token 唯一标识
//   - maxRetries: 最大重试次数
//
// 返回值:
//   - *OAuthToken: Token 数据
//   - error: 错误信息,nil 表示成功
func (c *Client) GetOAuthTokenWithRetry(ctx context.Context, tokenID string, maxRetries int) (*OAuthToken, error) {
    var lastErr error

    for i := 0; i < maxRetries; i++ {
        token, err := c.GetOAuthToken(ctx, tokenID)
        if err == nil {
            return token, nil
        }

        lastErr = err

        // 如果是超时错误,进行重试
        if errors.Is(err, context.DeadlineExceeded) {
            backoff := time.Duration(i+1) * 100 * time.Millisecond
            time.Sleep(backoff)
            continue
        }

        // 其他错误直接返回
        return nil, err
    }

    return nil, lastErr
}
```

## 连接池配置

```go
package main

import (
    "github.com/redis/go-redis/v9"
)

func NewOptimizedClient() *redis.Client {
    return redis.NewClient(&redis.Options{
        Addr:     "localhost:6380",
        Password: os.Getenv("TOKENGINX_API_KEY"),

        // 连接池配置
        PoolSize:     50,              // 最大连接数
        MinIdleConns: 10,              // 最小空闲连接数
        MaxIdleConns: 20,              // 最大空闲连接数
        PoolTimeout:  4 * time.Second, // 连接池超时

        // 超时配置
        DialTimeout:  5 * time.Second,
        ReadTimeout:  3 * time.Second,
        WriteTimeout: 3 * time.Second,

        // 重试配置
        MaxRetries:      3,
        MinRetryBackoff: 8 * time.Millisecond,
        MaxRetryBackoff: 512 * time.Millisecond,
    })
}
```

## 使用集群模式(v2.0.0+)

```go
package main

import (
    "github.com/redis/go-redis/v9"
)

func NewClusterClient() *redis.ClusterClient {
    return redis.NewClusterClient(&redis.ClusterOptions{
        Addrs: []string{
            "node1:6380",
            "node2:6380",
            "node3:6380",
        },
        Password: os.Getenv("TOKENGINX_API_KEY"),

        // 连接池配置
        PoolSize:     50,
        MinIdleConns: 10,

        // 超时配置
        DialTimeout:  5 * time.Second,
        ReadTimeout:  3 * time.Second,
        WriteTimeout: 3 * time.Second,
    })
}
```

## 下一步

- 查看 [Go 生产环境指南](../production/go.md)
- 了解 [OAuth 2.0 集成](../protocols/oauth.md)
- 配置 [TLS/mTLS](../security/tls-mtls.md)
