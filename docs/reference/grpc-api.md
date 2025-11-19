# gRPC API 参考

## 概述

TokenginX 提供高性能的 gRPC 接口,适合微服务架构和需要强类型定义的场景。gRPC 基于 HTTP/2 协议,支持双向流、流控、头部压缩等特性。

**默认端口**: `9090`

**协议**: gRPC over HTTP/2

**序列化**: Protocol Buffers (protobuf)

## Proto 定义

### 服务定义

完整的 proto 定义文件位于 `api/proto/tokenginx.proto`:

```protobuf
syntax = "proto3";

package tokenginx.v1;

option go_package = "github.com/yourorg/tokenginx/api/v1;v1";

// TokenginX 服务定义
service TokenginXService {
  // 基本键值操作
  rpc Set(SetRequest) returns (SetResponse);
  rpc Get(GetRequest) returns (GetResponse);
  rpc Delete(DeleteRequest) returns (DeleteResponse);
  rpc Exists(ExistsRequest) returns (ExistsResponse);
  rpc GetTTL(GetTTLRequest) returns (GetTTLResponse);

  // 批量操作
  rpc BatchGet(BatchGetRequest) returns (BatchGetResponse);
  rpc BatchSet(BatchSetRequest) returns (BatchSetResponse);
  rpc BatchDelete(BatchDeleteRequest) returns (BatchDeleteResponse);

  // 扫描操作
  rpc Scan(ScanRequest) returns (stream ScanResponse);

  // OAuth 2.0 操作
  rpc SetOAuthToken(SetOAuthTokenRequest) returns (SetOAuthTokenResponse);
  rpc GetOAuthToken(GetOAuthTokenRequest) returns (GetOAuthTokenResponse);
  rpc IntrospectToken(IntrospectTokenRequest) returns (IntrospectTokenResponse);
  rpc RevokeToken(RevokeTokenRequest) returns (RevokeTokenResponse);

  // SAML 2.0 操作
  rpc SetSAMLSession(SetSAMLSessionRequest) returns (SetSAMLSessionResponse);
  rpc GetSAMLSession(GetSAMLSessionRequest) returns (GetSAMLSessionResponse);

  // CAS 操作
  rpc SetTGT(SetTGTRequest) returns (SetTGTResponse);
  rpc SetST(SetSTRequest) returns (SetSTResponse);
  rpc ValidateST(ValidateSTRequest) returns (ValidateSTResponse);

  // 服务器信息
  rpc GetStats(GetStatsRequest) returns (GetStatsResponse);
  rpc HealthCheck(HealthCheckRequest) returns (HealthCheckResponse);
}

// 消息定义

message SetRequest {
  string key = 1;
  bytes value = 2;
  int64 ttl_seconds = 3;  // 0 表示永不过期
}

message SetResponse {
  bool success = 1;
  string error = 2;
}

message GetRequest {
  string key = 1;
}

message GetResponse {
  bytes value = 1;
  bool found = 2;
  int64 ttl_seconds = 3;
}

message DeleteRequest {
  string key = 1;
}

message DeleteResponse {
  bool deleted = 1;
}

message ExistsRequest {
  string key = 1;
}

message ExistsResponse {
  bool exists = 1;
}

message GetTTLRequest {
  string key = 1;
}

message GetTTLResponse {
  int64 ttl_seconds = 1;  // -1: 永不过期, -2: 不存在
}

message BatchGetRequest {
  repeated string keys = 1;
}

message BatchGetResponse {
  map<string, bytes> values = 1;  // 不存在的键不会出现在 map 中
}

message BatchSetRequest {
  message KeyValue {
    string key = 1;
    bytes value = 2;
    int64 ttl_seconds = 3;
  }
  repeated KeyValue items = 1;
}

message BatchSetResponse {
  int32 success_count = 1;
  int32 failed_count = 2;
}

message BatchDeleteRequest {
  repeated string keys = 1;
}

message BatchDeleteResponse {
  int32 deleted_count = 1;
}

message ScanRequest {
  string pattern = 1;
  int32 count = 2;  // 每次迭代返回的数量提示
}

message ScanResponse {
  repeated string keys = 1;
}

// OAuth 2.0 消息

message SetOAuthTokenRequest {
  string token_id = 1;
  string user_id = 2;
  string scope = 3;
  string client_id = 4;
  int64 ttl_seconds = 5;
  map<string, string> metadata = 6;  // 额外元数据
}

message SetOAuthTokenResponse {
  bool success = 1;
  string error = 2;
}

message GetOAuthTokenRequest {
  string token_id = 1;
}

message GetOAuthTokenResponse {
  bool found = 1;
  string user_id = 2;
  string scope = 3;
  string client_id = 4;
  int64 created_at = 5;
  int64 expires_at = 6;
  map<string, string> metadata = 7;
}

message IntrospectTokenRequest {
  string token = 1;
  string token_type_hint = 2;  // "access_token" 或 "refresh_token"
}

message IntrospectTokenResponse {
  bool active = 1;
  string scope = 2;
  string client_id = 3;
  string username = 4;
  string token_type = 5;
  int64 exp = 6;
  int64 iat = 7;
  string sub = 8;
}

message RevokeTokenRequest {
  string token = 1;
  string token_type_hint = 2;
}

message RevokeTokenResponse {
  bool success = 1;
}

// SAML 2.0 消息

message SetSAMLSessionRequest {
  string session_index = 1;
  string name_id = 2;
  string name_id_format = 3;
  bytes assertion = 4;  // Base64 编码的 SAML 断言
  int64 ttl_seconds = 5;
  map<string, string> attributes = 6;
}

message SetSAMLSessionResponse {
  bool success = 1;
  string error = 2;
}

message GetSAMLSessionRequest {
  string session_index = 1;
}

message GetSAMLSessionResponse {
  bool found = 1;
  string name_id = 2;
  string name_id_format = 3;
  bytes assertion = 4;
  int64 created_at = 5;
  int64 expires_at = 6;
  map<string, string> attributes = 7;
}

// CAS 消息

message SetTGTRequest {
  string tgt_id = 1;
  string user_id = 2;
  int64 ttl_seconds = 3;
  map<string, string> attributes = 4;
}

message SetTGTResponse {
  bool success = 1;
  string error = 2;
}

message SetSTRequest {
  string st_id = 1;
  string tgt_id = 2;
  string service = 3;
  int64 ttl_seconds = 4;
}

message SetSTResponse {
  bool success = 1;
  string error = 2;
}

message ValidateSTRequest {
  string st_id = 1;
  string service = 2;
}

message ValidateSTResponse {
  bool valid = 1;
  string user_id = 2;
  map<string, string> attributes = 3;
}

// 服务器信息消息

message GetStatsRequest {}

message GetStatsResponse {
  int64 total_keys = 1;
  int64 memory_used_bytes = 2;
  int64 total_commands_processed = 3;
  int32 instantaneous_ops_per_sec = 4;
  int32 connected_clients = 5;
  int64 uptime_seconds = 6;
}

message HealthCheckRequest {}

message HealthCheckResponse {
  enum Status {
    UNKNOWN = 0;
    HEALTHY = 1;
    UNHEALTHY = 2;
  }
  Status status = 1;
  string version = 2;
  int64 uptime_seconds = 3;
}
```

## 客户端示例

### Go 客户端

```go
package main

import (
    "context"
    "crypto/tls"
    "crypto/x509"
    "fmt"
    "io/ioutil"
    "log"
    "time"

    "google.golang.org/grpc"
    "google.golang.org/grpc/credentials"
    pb "your-module/api/v1"
)

func main() {
    // 加载 TLS 证书
    tlsConfig, err := loadTLSConfig(
        "/path/to/ca.pem",
        "/path/to/client-cert.pem",
        "/path/to/client-key.pem",
    )
    if err != nil {
        log.Fatal(err)
    }

    // 创建 gRPC 连接
    creds := credentials.NewTLS(tlsConfig)
    conn, err := grpc.Dial(
        "localhost:9090",
        grpc.WithTransportCredentials(creds),
        grpc.WithBlock(),
        grpc.WithTimeout(5*time.Second),
    )
    if err != nil {
        log.Fatalf("Failed to connect: %v", err)
    }
    defer conn.Close()

    // 创建客户端
    client := pb.NewTokenginXServiceClient(conn)

    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    // 设置键值对
    setResp, err := client.Set(ctx, &pb.SetRequest{
        Key:        "oauth:token:abc123",
        Value:      []byte(`{"user_id":"user001","scope":"read write"}`),
        TtlSeconds: 3600,
    })
    if err != nil {
        log.Fatalf("Set failed: %v", err)
    }
    fmt.Printf("Set success: %v\n", setResp.Success)

    // 获取键值
    getResp, err := client.Get(ctx, &pb.GetRequest{
        Key: "oauth:token:abc123",
    })
    if err != nil {
        log.Fatalf("Get failed: %v", err)
    }
    if getResp.Found {
        fmt.Printf("Value: %s\n", string(getResp.Value))
        fmt.Printf("TTL: %d seconds\n", getResp.TtlSeconds)
    }

    // 使用 OAuth 专用方法
    oauthResp, err := client.SetOAuthToken(ctx, &pb.SetOAuthTokenRequest{
        TokenId:    "xyz789",
        UserId:     "user002",
        Scope:      "read",
        ClientId:   "client001",
        TtlSeconds: 3600,
    })
    if err != nil {
        log.Fatalf("SetOAuthToken failed: %v", err)
    }
    fmt.Printf("OAuth token set: %v\n", oauthResp.Success)

    // Token 内省
    introspectResp, err := client.IntrospectToken(ctx, &pb.IntrospectTokenRequest{
        Token:         "xyz789",
        TokenTypeHint: "access_token",
    })
    if err != nil {
        log.Fatalf("Introspect failed: %v", err)
    }
    if introspectResp.Active {
        fmt.Printf("Token active: user=%s, scope=%s\n",
            introspectResp.Username, introspectResp.Scope)
    }
}

func loadTLSConfig(caFile, certFile, keyFile string) (*tls.Config, error) {
    // 加载 CA 证书
    caCert, err := ioutil.ReadFile(caFile)
    if err != nil {
        return nil, err
    }
    caCertPool := x509.NewCertPool()
    caCertPool.AppendCertsFromPEM(caCert)

    // 加载客户端证书
    cert, err := tls.LoadX509KeyPair(certFile, keyFile)
    if err != nil {
        return nil, err
    }

    return &tls.Config{
        Certificates: []tls.Certificate{cert},
        RootCAs:      caCertPool,
        MinVersion:   tls.VersionTLS13,
    }, nil
}
```

### Java 客户端

```java
import io.grpc.ManagedChannel;
import io.grpc.ManagedChannelBuilder;
import io.grpc.netty.GrpcSslContexts;
import io.grpc.netty.NettyChannelBuilder;
import io.netty.handler.ssl.SslContext;

import java.io.File;
import java.util.concurrent.TimeUnit;

public class TokenginXClient {
    private final ManagedChannel channel;
    private final TokenginXServiceGrpc.TokenginXServiceBlockingStub blockingStub;

    public TokenginXClient(String host, int port, SslContext sslContext) {
        this.channel = NettyChannelBuilder.forAddress(host, port)
            .sslContext(sslContext)
            .build();

        this.blockingStub = TokenginXServiceGrpc.newBlockingStub(channel);
    }

    public void shutdown() throws InterruptedException {
        channel.shutdown().awaitTermination(5, TimeUnit.SECONDS);
    }

    public void set(String key, byte[] value, long ttl) {
        SetRequest request = SetRequest.newBuilder()
            .setKey(key)
            .setValue(ByteString.copyFrom(value))
            .setTtlSeconds(ttl)
            .build();

        SetResponse response = blockingStub.set(request);
        System.out.println("Set success: " + response.getSuccess());
    }

    public byte[] get(String key) {
        GetRequest request = GetRequest.newBuilder()
            .setKey(key)
            .build();

        GetResponse response = blockingStub.get(request);
        if (response.getFound()) {
            return response.getValue().toByteArray();
        }
        return null;
    }

    public void setOAuthToken(String tokenId, String userId, String scope, long ttl) {
        SetOAuthTokenRequest request = SetOAuthTokenRequest.newBuilder()
            .setTokenId(tokenId)
            .setUserId(userId)
            .setScope(scope)
            .setTtlSeconds(ttl)
            .build();

        SetOAuthTokenResponse response = blockingStub.setOAuthToken(request);
        System.out.println("OAuth token set: " + response.getSuccess());
    }

    public static void main(String[] args) throws Exception {
        // 创建 SSL 上下文
        SslContext sslContext = GrpcSslContexts.forClient()
            .trustManager(new File("/path/to/ca.pem"))
            .keyManager(
                new File("/path/to/client-cert.pem"),
                new File("/path/to/client-key.pem")
            )
            .build();

        TokenginXClient client = new TokenginXClient("localhost", 9090, sslContext);

        try {
            client.set("mykey", "myvalue".getBytes(), 3600);
            byte[] value = client.get("mykey");
            System.out.println("Value: " + new String(value));

            client.setOAuthToken("token123", "user001", "read write", 3600);
        } finally {
            client.shutdown();
        }
    }
}
```

### Python 客户端

```python
import grpc
from tokenginx.v1 import tokenginx_pb2, tokenginx_pb2_grpc

def load_credentials(ca_file, cert_file, key_file):
    with open(ca_file, 'rb') as f:
        ca_cert = f.read()
    with open(cert_file, 'rb') as f:
        client_cert = f.read()
    with open(key_file, 'rb') as f:
        client_key = f.read()

    return grpc.ssl_channel_credentials(
        root_certificates=ca_cert,
        private_key=client_key,
        certificate_chain=client_cert
    )

def main():
    # 创建 TLS 凭证
    credentials = load_credentials(
        '/path/to/ca.pem',
        '/path/to/client-cert.pem',
        '/path/to/client-key.pem'
    )

    # 创建 gRPC 通道
    channel = grpc.secure_channel('localhost:9090', credentials)
    stub = tokenginx_pb2_grpc.TokenginXServiceStub(channel)

    # 设置键值对
    response = stub.Set(tokenginx_pb2.SetRequest(
        key='oauth:token:abc123',
        value=b'{"user_id":"user001"}',
        ttl_seconds=3600
    ))
    print(f"Set success: {response.success}")

    # 获取键值
    response = stub.Get(tokenginx_pb2.GetRequest(
        key='oauth:token:abc123'
    ))
    if response.found:
        print(f"Value: {response.value.decode()}")
        print(f"TTL: {response.ttl_seconds}")

    # 设置 OAuth Token
    response = stub.SetOAuthToken(tokenginx_pb2.SetOAuthTokenRequest(
        token_id='xyz789',
        user_id='user002',
        scope='read',
        ttl_seconds=3600
    ))
    print(f"OAuth token set: {response.success}")

    # Token 内省
    response = stub.IntrospectToken(tokenginx_pb2.IntrospectTokenRequest(
        token='xyz789',
        token_type_hint='access_token'
    ))
    if response.active:
        print(f"Token active: user={response.username}, scope={response.scope}")

if __name__ == '__main__':
    main()
```

## 流式操作

### 服务端流式扫描

```go
// Go 示例
stream, err := client.Scan(ctx, &pb.ScanRequest{
    Pattern: "oauth:token:*",
    Count:   100,
})
if err != nil {
    log.Fatal(err)
}

for {
    resp, err := stream.Recv()
    if err == io.EOF {
        break
    }
    if err != nil {
        log.Fatal(err)
    }

    for _, key := range resp.Keys {
        fmt.Println(key)
    }
}
```

```python
# Python 示例
for response in stub.Scan(tokenginx_pb2.ScanRequest(
    pattern='oauth:token:*',
    count=100
)):
    for key in response.keys:
        print(key)
```

## 元数据和认证

### 使用元数据传递 API Key

```go
// Go 示例
md := metadata.New(map[string]string{
    "authorization": "Bearer your-api-key",
})
ctx := metadata.NewOutgoingContext(context.Background(), md)

resp, err := client.Set(ctx, &pb.SetRequest{
    Key:   "mykey",
    Value: []byte("myvalue"),
})
```

```python
# Python 示例
metadata = [('authorization', 'Bearer your-api-key')]

response = stub.Set(
    tokenginx_pb2.SetRequest(key='mykey', value=b'myvalue'),
    metadata=metadata
)
```

### 使用拦截器自动添加认证

```go
// Go 拦截器
func authInterceptor(apiKey string) grpc.UnaryClientInterceptor {
    return func(
        ctx context.Context,
        method string,
        req, reply interface{},
        cc *grpc.ClientConn,
        invoker grpc.UnaryInvoker,
        opts ...grpc.CallOption,
    ) error {
        md := metadata.New(map[string]string{
            "authorization": "Bearer " + apiKey,
        })
        ctx = metadata.NewOutgoingContext(ctx, md)
        return invoker(ctx, method, req, reply, cc, opts...)
    }
}

// 使用拦截器
conn, err := grpc.Dial(
    "localhost:9090",
    grpc.WithTransportCredentials(creds),
    grpc.WithUnaryInterceptor(authInterceptor("your-api-key")),
)
```

## 错误处理

gRPC 使用标准的状态码:

```go
import (
    "google.golang.org/grpc/codes"
    "google.golang.org/grpc/status"
)

resp, err := client.Get(ctx, &pb.GetRequest{Key: "mykey"})
if err != nil {
    st, ok := status.FromError(err)
    if ok {
        switch st.Code() {
        case codes.NotFound:
            fmt.Println("Key not found")
        case codes.Unauthenticated:
            fmt.Println("Authentication required")
        case codes.PermissionDenied:
            fmt.Println("Permission denied")
        case codes.DeadlineExceeded:
            fmt.Println("Request timeout")
        default:
            fmt.Printf("Error: %v\n", st.Message())
        }
    }
}
```

### 常用状态码

- `OK` (0): 成功
- `CANCELLED` (1): 操作被取消
- `INVALID_ARGUMENT` (3): 无效参数
- `DEADLINE_EXCEEDED` (4): 超时
- `NOT_FOUND` (5): 资源不存在
- `ALREADY_EXISTS` (6): 资源已存在
- `PERMISSION_DENIED` (7): 权限不足
- `UNAUTHENTICATED` (16): 未认证
- `RESOURCE_EXHAUSTED` (8): 资源耗尽(如超过速率限制)
- `INTERNAL` (13): 内部错误
- `UNAVAILABLE` (14): 服务不可用

## 性能优化

### 1. 连接复用

```go
// 复用单个连接
conn, err := grpc.Dial("localhost:9090", opts...)
defer conn.Close()

client := pb.NewTokenginXServiceClient(conn)
// 多次调用复用同一连接
```

### 2. 使用连接池

```go
type ClientPool struct {
    clients []pb.TokenginXServiceClient
    conns   []*grpc.ClientConn
    current int32
    mu      sync.Mutex
}

func NewClientPool(addr string, size int) (*ClientPool, error) {
    pool := &ClientPool{
        clients: make([]pb.TokenginXServiceClient, size),
        conns:   make([]*grpc.ClientConn, size),
    }

    for i := 0; i < size; i++ {
        conn, err := grpc.Dial(addr, opts...)
        if err != nil {
            return nil, err
        }
        pool.conns[i] = conn
        pool.clients[i] = pb.NewTokenginXServiceClient(conn)
    }

    return pool, nil
}

func (p *ClientPool) Get() pb.TokenginXServiceClient {
    idx := atomic.AddInt32(&p.current, 1) % int32(len(p.clients))
    return p.clients[idx]
}
```

### 3. 设置合理的超时

```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

resp, err := client.Get(ctx, &pb.GetRequest{Key: "mykey"})
```

### 4. 启用压缩

```go
import "google.golang.org/grpc/encoding/gzip"

resp, err := client.Get(ctx, req, grpc.UseCompressor(gzip.Name))
```

## 性能基准

### 延迟分布

| 操作 | P50 | P95 | P99 | P999 |
|------|-----|-----|-----|------|
| Set  | 0.3ms | 0.6ms | 1ms | 3ms |
| Get  | 0.25ms | 0.5ms | 0.8ms | 2.5ms |
| BatchGet(10) | 0.5ms | 1ms | 1.5ms | 4ms |
| Scan(stream) | 0.8ms | 1.5ms | 2ms | 5ms |

### 吞吐量

- 单连接: 15,000 QPS
- 10 并发连接: 100,000 QPS
- 连接池(10 连接): 120,000 QPS

## 生成客户端代码

### Go

```bash
protoc --go_out=. --go_opt=paths=source_relative \
    --go-grpc_out=. --go-grpc_opt=paths=source_relative \
    api/proto/tokenginx.proto
```

### Java

```bash
protoc --java_out=src/main/java \
    --grpc-java_out=src/main/java \
    api/proto/tokenginx.proto
```

### Python

```bash
python -m grpc_tools.protoc -I. \
    --python_out=. \
    --grpc_python_out=. \
    api/proto/tokenginx.proto
```

### C#

```bash
protoc --csharp_out=. --grpc_out=. \
    --plugin=protoc-gen-grpc=grpc_csharp_plugin \
    api/proto/tokenginx.proto
```

## 下一步

- 查看 [核心功能参考](./core-features.md)
- 查看 [HTTP/REST API](./http-rest-api.md)
- 配置 [TLS/mTLS](../security/tls-mtls.md)
- 了解 [生产环境部署](../production/)
