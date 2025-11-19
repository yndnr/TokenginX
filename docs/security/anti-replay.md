# 防重放攻击

TokenginX 内置完整的防重放攻击机制,保护系统免受重放攻击威胁。

## 什么是重放攻击

重放攻击（Replay Attack）是指攻击者截获并重新发送合法的请求，试图欺骗系统执行未授权的操作。

**攻击场景示例**:
1. 用户 A 向 TokenginX 发送设置会话的请求
2. 攻击者截获该请求
3. 攻击者在稍后时间重新发送该请求
4. 如果没有防重放机制，系统会再次执行该操作

## TokenginX 防重放策略

TokenginX 采用多层防御策略:

### 1. 基于时间戳的防重放

每个请求必须携带时间戳，服务器拒绝超出时间窗口的请求。

#### 工作原理

```
当前时间: 2025-11-19 12:00:00
时间窗口: 5 分钟

接受的请求时间范围:
  最早: 2025-11-19 11:55:00 (当前时间 - 5分钟)
  最晚: 2025-11-19 12:05:00 (当前时间 + 5分钟，容忍时钟偏移)
```

#### 配置

```yaml
security:
  anti_replay:
    enabled: true
    # 时间窗口（秒）
    window_seconds: 300  # 5 分钟
    # 时钟偏移容忍（秒）
    clock_skew_seconds: 30
```

#### 客户端实现（HTTP）

```go
import (
    "fmt"
    "net/http"
    "time"
)

func makeRequest() {
    // 1. 生成时间戳（Unix 秒）
    timestamp := fmt.Sprintf("%d", time.Now().Unix())

    // 2. 创建请求
    req, _ := http.NewRequest("POST", "https://tokenginx.example.com/api/v1/sessions", body)

    // 3. 添加时间戳头
    req.Header.Set("X-Timestamp", timestamp)
    req.Header.Set("Content-Type", "application/json")

    // 4. 发送请求
    resp, err := http.DefaultClient.Do(req)
    // ...
}
```

#### 客户端实现（gRPC）

```go
import (
    "context"
    "time"

    "google.golang.org/grpc/metadata"
)

func makeGRPCRequest(client pb.SessionServiceClient) {
    // 1. 创建带有时间戳的 metadata
    ctx := context.Background()
    timestamp := fmt.Sprintf("%d", time.Now().Unix())
    md := metadata.Pairs("x-timestamp", timestamp)
    ctx = metadata.NewOutgoingContext(ctx, md)

    // 2. 发送请求
    resp, err := client.Set(ctx, &pb.SetRequest{...})
    // ...
}
```

### 2. 基于 Nonce 的防重放

Nonce（Number used once）是一次性随机数，确保每个请求唯一。

#### 工作原理

```
1. 客户端生成唯一的 Nonce（如 UUID）
2. 请求携带 Nonce
3. 服务器检查 Nonce 是否已使用过
4. 如果已使用，拒绝请求
5. 如果未使用，记录 Nonce 并处理请求
6. 超出时间窗口的 Nonce 自动清理
```

#### 配置

```yaml
security:
  anti_replay:
    enabled: true
    window_seconds: 300
    # Nonce 缓存大小（个数）
    nonce_cache_size: 100000
    # Nonce 缓存实现：memory | redis
    nonce_cache_backend: "memory"
```

#### 客户端实现

```go
import (
    "crypto/rand"
    "encoding/hex"
    "fmt"
    "net/http"
    "time"
)

// 生成随机 Nonce
func generateNonce() string {
    b := make([]byte, 16)
    rand.Read(b)
    return hex.EncodeToString(b)
}

func makeRequestWithNonce() {
    // 1. 生成时间戳和 Nonce
    timestamp := fmt.Sprintf("%d", time.Now().Unix())
    nonce := generateNonce()

    // 2. 创建请求
    req, _ := http.NewRequest("POST", "https://tokenginx.example.com/api/v1/sessions", body)

    // 3. 添加头
    req.Header.Set("X-Timestamp", timestamp)
    req.Header.Set("X-Nonce", nonce)
    req.Header.Set("Content-Type", "application/json")

    // 4. 发送请求
    resp, err := http.DefaultClient.Do(req)
    // ...
}
```

### 3. 请求签名

使用 HMAC 签名确保请求完整性和防止篡改。

#### 签名算法

TokenginX 支持以下签名算法:
- **HMAC-SHA256**（商密）
- **HMAC-SM3**（国密）

#### 签名流程

**签名字符串构造**:
```
signString = method + "\n" + uri + "\n" + timestamp + "\n" + nonce + "\n" + body
```

**HMAC 计算**:
```
signature = HMAC(secretKey, signString)
```

#### 配置

```yaml
security:
  anti_replay:
    enabled: true
    window_seconds: 300
    nonce_cache_size: 100000
    # 签名算法
    signature_algorithm: "hmac-sha256"  # hmac-sha256 | hmac-sm3
    # 签名验证
    require_signature: true
```

#### 客户端实现（HMAC-SHA256）

```go
import (
    "crypto/hmac"
    "crypto/sha256"
    "encoding/hex"
    "fmt"
    "net/http"
    "time"
)

func signRequest(method, uri, body string, secretKey []byte) (timestamp, nonce, signature string) {
    // 1. 生成时间戳和 Nonce
    timestamp = fmt.Sprintf("%d", time.Now().Unix())
    nonce = generateNonce()

    // 2. 构造签名字符串
    signString := fmt.Sprintf("%s\n%s\n%s\n%s\n%s",
        method, uri, timestamp, nonce, body)

    // 3. 计算 HMAC-SHA256
    h := hmac.New(sha256.New, secretKey)
    h.Write([]byte(signString))
    signature = hex.EncodeToString(h.Sum(nil))

    return timestamp, nonce, signature
}

func makeSignedRequest() {
    method := "POST"
    uri := "/api/v1/sessions"
    body := `{"key":"oauth:token:abc123","value":"...","ttl":3600}`
    secretKey := []byte("your-secret-key-here")

    // 1. 签名
    timestamp, nonce, signature := signRequest(method, uri, body, secretKey)

    // 2. 创建请求
    req, _ := http.NewRequest(method, "https://tokenginx.example.com"+uri, strings.NewReader(body))

    // 3. 添加签名头
    req.Header.Set("X-Timestamp", timestamp)
    req.Header.Set("X-Nonce", nonce)
    req.Header.Set("X-Signature", signature)
    req.Header.Set("Content-Type", "application/json")

    // 4. 发送请求
    resp, err := http.DefaultClient.Do(req)
    // ...
}
```

#### 客户端实现（HMAC-SM3，国密）

```go
import (
    "crypto/hmac"
    "encoding/hex"
    "fmt"

    "github.com/tjfoc/gmsm/sm3"
)

func signRequestGM(method, uri, body string, secretKey []byte) (timestamp, nonce, signature string) {
    timestamp = fmt.Sprintf("%d", time.Now().Unix())
    nonce = generateNonce()

    signString := fmt.Sprintf("%s\n%s\n%s\n%s\n%s",
        method, uri, timestamp, nonce, body)

    // 使用 HMAC-SM3
    h := hmac.New(sm3.New, secretKey)
    h.Write([]byte(signString))
    signature = hex.EncodeToString(h.Sum(nil))

    return timestamp, nonce, signature
}
```

### 4. 序列号机制（可选）

适用于有序请求场景，确保请求按顺序执行。

#### 工作原理

```
客户端:
  请求1: seq=1
  请求2: seq=2
  请求3: seq=3

服务器:
  记录最后收到的序列号: lastSeq=2
  收到请求: seq=3 -> 接受（3 > 2）
  收到请求: seq=2 -> 拒绝（2 <= 2，疑似重放）
```

#### 配置

```yaml
security:
  anti_replay:
    enabled: true
    # 启用序列号验证
    enable_sequence: true
    # 允许的序列号跳跃（防止丢包导致拒绝）
    sequence_tolerance: 10
```

#### 客户端实现

```go
type Client struct {
    sequence uint64
    mu       sync.Mutex
}

func (c *Client) makeSequencedRequest() {
    c.mu.Lock()
    c.sequence++
    seq := c.sequence
    c.mu.Unlock()

    req, _ := http.NewRequest("POST", url, body)
    req.Header.Set("X-Sequence", fmt.Sprintf("%d", seq))
    // ...
}
```

## 完整示例

### 服务器配置

```yaml
# config.yaml
security:
  anti_replay:
    # 启用防重放
    enabled: true

    # 时间窗口（秒）
    window_seconds: 300

    # 时钟偏移容忍（秒）
    clock_skew_seconds: 30

    # Nonce 配置
    nonce_cache_size: 100000
    nonce_cache_backend: "memory"  # memory | redis

    # 签名配置
    require_signature: true
    signature_algorithm: "hmac-sha256"  # hmac-sha256 | hmac-sm3

    # 序列号（可选）
    enable_sequence: false
    sequence_tolerance: 10

    # 日志
    log_rejected_requests: true

# 密钥管理
secrets:
  # 客户端密钥（示例）
  clients:
    - client_id: "app1"
      secret_key: "secret-key-for-app1"
    - client_id: "app2"
      secret_key: "secret-key-for-app2"
```

### Go 客户端完整实现

```go
package main

import (
    "crypto/hmac"
    "crypto/rand"
    "crypto/sha256"
    "encoding/hex"
    "fmt"
    "io"
    "net/http"
    "strings"
    "time"
)

type TokenginxClient struct {
    baseURL   string
    clientID  string
    secretKey []byte
    httpClient *http.Client
}

func NewClient(baseURL, clientID, secretKey string) *TokenginxClient {
    return &TokenginxClient{
        baseURL:   baseURL,
        clientID:  clientID,
        secretKey: []byte(secretKey),
        httpClient: &http.Client{Timeout: 10 * time.Second},
    }
}

func (c *TokenginxClient) generateNonce() string {
    b := make([]byte, 16)
    rand.Read(b)
    return hex.EncodeToString(b)
}

func (c *TokenginxClient) signRequest(method, uri, body string) (timestamp, nonce, signature string) {
    // 1. 生成时间戳和 Nonce
    timestamp = fmt.Sprintf("%d", time.Now().Unix())
    nonce = c.generateNonce()

    // 2. 构造签名字符串
    signString := fmt.Sprintf("%s\n%s\n%s\n%s\n%s",
        method, uri, timestamp, nonce, body)

    // 3. 计算 HMAC-SHA256
    h := hmac.New(sha256.New, c.secretKey)
    h.Write([]byte(signString))
    signature = hex.EncodeToString(h.Sum(nil))

    return
}

func (c *TokenginxClient) SetSession(key string, value interface{}, ttl int) error {
    method := "POST"
    uri := "/api/v1/sessions"
    body := fmt.Sprintf(`{"key":"%s","value":%v,"ttl":%d}`, key, value, ttl)

    // 1. 签名请求
    timestamp, nonce, signature := c.signRequest(method, uri, body)

    // 2. 创建请求
    req, err := http.NewRequest(method, c.baseURL+uri, strings.NewReader(body))
    if err != nil {
        return err
    }

    // 3. 添加头
    req.Header.Set("X-Client-ID", c.clientID)
    req.Header.Set("X-Timestamp", timestamp)
    req.Header.Set("X-Nonce", nonce)
    req.Header.Set("X-Signature", signature)
    req.Header.Set("Content-Type", "application/json")

    // 4. 发送请求
    resp, err := c.httpClient.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    // 5. 检查响应
    if resp.StatusCode != http.StatusOK {
        bodyBytes, _ := io.ReadAll(resp.Body)
        return fmt.Errorf("request failed: %d %s", resp.StatusCode, string(bodyBytes))
    }

    return nil
}

func main() {
    client := NewClient(
        "https://tokenginx.example.com",
        "app1",
        "secret-key-for-app1",
    )

    err := client.SetSession("oauth:token:abc123", `{"user_id":"user001"}`, 3600)
    if err != nil {
        fmt.Printf("Error: %v\n", err)
        return
    }

    fmt.Println("Session set successfully")
}
```

### Java 客户端实现

```java
import javax.crypto.Mac;
import javax.crypto.spec.SecretKeySpec;
import java.net.http.*;
import java.nio.charset.StandardCharsets;
import java.security.SecureRandom;
import java.time.Instant;
import java.util.HexFormat;

public class TokenginxClient {
    private final String baseURL;
    private final String clientID;
    private final byte[] secretKey;
    private final HttpClient httpClient;
    private final SecureRandom random;

    public TokenginxClient(String baseURL, String clientID, String secretKey) {
        this.baseURL = baseURL;
        this.clientID = clientID;
        this.secretKey = secretKey.getBytes(StandardCharsets.UTF_8);
        this.httpClient = HttpClient.newHttpClient();
        this.random = new SecureRandom();
    }

    private String generateNonce() {
        byte[] bytes = new byte[16];
        random.nextBytes(bytes);
        return HexFormat.of().formatHex(bytes);
    }

    private SignatureResult signRequest(String method, String uri, String body) throws Exception {
        // 1. 生成时间戳和 Nonce
        String timestamp = String.valueOf(Instant.now().getEpochSecond());
        String nonce = generateNonce();

        // 2. 构造签名字符串
        String signString = String.format("%s\n%s\n%s\n%s\n%s",
            method, uri, timestamp, nonce, body);

        // 3. 计算 HMAC-SHA256
        Mac mac = Mac.getInstance("HmacSHA256");
        SecretKeySpec keySpec = new SecretKeySpec(secretKey, "HmacSHA256");
        mac.init(keySpec);
        byte[] signBytes = mac.doFinal(signString.getBytes(StandardCharsets.UTF_8));
        String signature = HexFormat.of().formatHex(signBytes);

        return new SignatureResult(timestamp, nonce, signature);
    }

    public void setSession(String key, String value, int ttl) throws Exception {
        String method = "POST";
        String uri = "/api/v1/sessions";
        String body = String.format("{\"key\":\"%s\",\"value\":%s,\"ttl\":%d}",
            key, value, ttl);

        // 1. 签名
        SignatureResult sig = signRequest(method, uri, body);

        // 2. 创建请求
        HttpRequest request = HttpRequest.newBuilder()
            .uri(URI.create(baseURL + uri))
            .header("X-Client-ID", clientID)
            .header("X-Timestamp", sig.timestamp)
            .header("X-Nonce", sig.nonce)
            .header("X-Signature", sig.signature)
            .header("Content-Type", "application/json")
            .POST(HttpRequest.BodyPublishers.ofString(body))
            .build();

        // 3. 发送请求
        HttpResponse<String> response = httpClient.send(request,
            HttpResponse.BodyHandlers.ofString());

        if (response.statusCode() != 200) {
            throw new RuntimeException("Request failed: " + response.statusCode() +
                " " + response.body());
        }
    }

    private static class SignatureResult {
        String timestamp;
        String nonce;
        String signature;

        SignatureResult(String timestamp, String nonce, String signature) {
            this.timestamp = timestamp;
            this.nonce = nonce;
            this.signature = signature;
        }
    }

    public static void main(String[] args) throws Exception {
        TokenginxClient client = new TokenginxClient(
            "https://tokenginx.example.com",
            "app1",
            "secret-key-for-app1"
        );

        client.setSession("oauth:token:abc123", "{\"user_id\":\"user001\"}", 3600);
        System.out.println("Session set successfully");
    }
}
```

## 错误处理

### 常见错误响应

#### 时间戳过期

**HTTP 响应**:
```
HTTP/1.1 401 Unauthorized
Content-Type: application/json

{
  "error": "timestamp_expired",
  "message": "Request timestamp is outside the acceptable window",
  "timestamp_received": "1700383200",
  "server_time": "1700383800"
}
```

**处理方式**:
- 检查客户端时钟是否同步
- 使用 NTP 同步时间
- 调整 `clock_skew_seconds` 配置

#### Nonce 重复

**HTTP 响应**:
```
HTTP/1.1 409 Conflict
Content-Type: application/json

{
  "error": "nonce_reused",
  "message": "The nonce has already been used",
  "nonce": "abc123def456..."
}
```

**处理方式**:
- 确保 Nonce 生成器使用加密安全的随机数
- 检查是否有请求重试逻辑导致重复

#### 签名错误

**HTTP 响应**:
```
HTTP/1.1 401 Unauthorized
Content-Type: application/json

{
  "error": "invalid_signature",
  "message": "Request signature verification failed"
}
```

**处理方式**:
- 检查签名字符串构造是否正确
- 确认密钥是否正确
- 检查签名算法是否匹配（HMAC-SHA256 vs HMAC-SM3）

## 性能优化

### Nonce 缓存优化

对于高并发场景,Nonce 缓存可能成为瓶颈:

#### 使用 Redis 作为 Nonce 缓存

```yaml
security:
  anti_replay:
    nonce_cache_backend: "redis"
    redis:
      addr: "localhost:6379"
      password: ""
      db: 0
      # 使用 SET NX EX 原子操作
      key_prefix: "tokenginx:nonce:"
```

#### 分片 Nonce 缓存

```yaml
security:
  anti_replay:
    nonce_cache_backend: "memory"
    # 使用多个分片减少锁竞争
    nonce_cache_shards: 256
```

### 签名验证优化

- **批量验证**:对于批量操作,一次签名覆盖多个请求
- **异步验证**:非关键路径的签名验证可以异步进行
- **缓存验证结果**:短时间内重复的签名验证结果可以缓存

## 安全建议

1. **始终使用 HTTPS/TLS**:防重放攻击配合 TLS 使用效果最佳
2. **合理设置时间窗口**:太长增加风险,太短影响可用性（推荐 5 分钟）
3. **定期轮换密钥**:密钥泄露风险,定期轮换（建议每 90 天）
4. **监控异常**:记录和监控被拒绝的请求,及时发现攻击
5. **使用强随机源**:Nonce 必须使用加密安全的随机数生成器

## 故障排查

### 启用调试日志

```yaml
logging:
  level: "debug"
  modules:
    - "anti_replay"
```

### 常见问题

**问题1: 时钟不同步导致大量请求被拒绝**

解决:
```bash
# 安装 NTP 客户端
apt-get install ntp

# 同步时间
ntpdate pool.ntp.org

# 启动 NTP 服务
systemctl enable ntp
systemctl start ntp
```

**问题2: Nonce 缓存占用内存过大**

解决:
- 减少 `window_seconds`
- 减少 `nonce_cache_size`
- 使用 Redis 作为外部缓存

**问题3: 签名验证性能瓶颈**

解决:
- 启用签名验证缓存
- 使用更快的哈希算法（如 BLAKE3）
- 考虑硬件加速（如 Intel AES-NI）

## 参考资料

- [OWASP Replay Attack](https://owasp.org/www-community/attacks/Replay_Attack)
- [RFC 6749 - OAuth 2.0 (Section 10.5)](https://tools.ietf.org/html/rfc6749#section-10.5)
- [AWS Signature Version 4](https://docs.aws.amazon.com/general/latest/gr/signature-version-4.html)

---

**下一步**: 配置 [访问控制列表 (ACL)](./acl.md) 进一步增强安全性。
