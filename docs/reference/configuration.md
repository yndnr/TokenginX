# 配置参考

TokenginX 使用 YAML 格式的配置文件。本文档详细说明所有配置选项。

## 配置文件位置

TokenginX 按以下顺序查找配置文件：

1. 命令行指定：`./tokenginx-server -config /path/to/config.yaml`
2. 当前目录：`./config.yaml`
3. `/etc/tokenginx/config.yaml`
4. `$HOME/.tokenginx/config.yaml`

## 完整配置示例

完整配置示例请参考 [config/config.example.yaml](../../config/config.example.yaml)。

## 配置章节

### 服务器配置 (server)

服务器监听地址和超时设置。

```yaml
server:
  tcp_addr: "0.0.0.0:6380"     # TCP (RESP) 监听地址
  grpc_addr: "0.0.0.0:9090"    # gRPC 监听地址
  http_addr: "0.0.0.0:8080"    # HTTP/REST 监听地址
  read_timeout: 30             # 读取超时（秒）
  write_timeout: 30            # 写入超时（秒）
  idle_timeout: 120            # 空闲超时（秒）
  max_connections: 10000       # 最大并发连接数
```

**参数说明**：

| 参数 | 类型 | 默认值 | 说明 |
|-----|------|--------|------|
| `tcp_addr` | string | `0.0.0.0:6380` | TCP (RESP) 监听地址 |
| `grpc_addr` | string | `0.0.0.0:9090` | gRPC 监听地址 |
| `http_addr` | string | `0.0.0.0:8080` | HTTP/REST 监听地址 |
| `read_timeout` | int | `30` | 读取超时（秒） |
| `write_timeout` | int | `30` | 写入超时（秒） |
| `idle_timeout` | int | `120` | 空闲超时（秒） |
| `max_connections` | int | `10000` | 最大并发连接数 |

### 存储配置 (storage)

存储引擎和持久化设置。

```yaml
storage:
  shard_count: 256              # 分片数量
  initial_capacity: 4096        # 每个分片初始容量
  enable_persistence: false     # 启用持久化
  data_dir: "/var/lib/tokenginx"  # 数据目录
```

**参数说明**：

| 参数 | 类型 | 默认值 | 说明 |
|-----|------|--------|------|
| `shard_count` | int | `256` | 分片数量，推荐 256 |
| `initial_capacity` | int | `4096` | 每个分片的初始容量 |
| `enable_persistence` | bool | `false` | 是否启用持久化 |
| `data_dir` | string | `/var/lib/tokenginx` | 数据存储目录 |

#### 持久化配置 (storage.persistence)

```yaml
storage:
  persistence:
    mmap_size_mb: 1024          # mmap 文件大小（MB）
    wal:
      enabled: true             # 启用 WAL
      dir: "/var/lib/tokenginx/wal"  # WAL 目录
      max_size_mb: 64           # WAL 文件最大大小（MB）
      sync_policy: "everysec"   # 同步策略
    compression:
      enabled: false            # 启用压缩
      algorithm: "lz4"          # 压缩算法
    encryption:
      enabled: false            # 启用加密
      algorithm: "aes-256-gcm"  # 加密算法
```

**sync_policy 选项**：
- `always`: 每次写入都同步到磁盘（最安全，性能最低）
- `everysec`: 每秒同步一次（推荐）
- `no`: 由操作系统决定何时同步（性能最高，风险最大）

**compression.algorithm 选项**：
- `lz4`: 快速压缩（推荐）
- `snappy`: 平衡压缩

### 缓存配置 (cache)

缓存算法和淘汰策略。

```yaml
cache:
  algorithm: "w-tinylfu"        # 缓存算法
  max_memory_mb: 1024           # 最大内存（MB）
  eviction_policy: "lru"        # 淘汰策略
  enable_stats: true            # 启用统计
```

**algorithm 选项**：
- `w-tinylfu`: W-TinyLFU 算法（推荐，命中率最高）
- `lru`: LRU 算法
- `lfu`: LFU 算法

**eviction_policy 选项**：
- `lru`: Least Recently Used
- `lfu`: Least Frequently Used
- `random`: 随机淘汰

### TTL 配置 (ttl)

过期时间管理。

```yaml
ttl:
  enabled: true                 # 启用 TTL
  default_ttl: 0                # 默认 TTL（秒，0=永不过期）
  cleanup_interval: 60          # 清理间隔（秒）
  cleanup_batch_size: 1000      # 每次清理数量
```

### 安全配置 (security)

#### 加密模式 (security.crypto_mode)

```yaml
security:
  crypto_mode: "auto"  # auto | gm | sm | hybrid
```

**选项说明**：
- `auto`: 根据客户端能力自动选择国密或商密
- `gm`: 仅使用国密算法（SM2/SM3/SM4）
- `sm`: 仅使用商密算法（RSA/AES/SHA）
- `hybrid`: 同时支持国密和商密

#### TLS 配置 (security.tls)

```yaml
security:
  tls:
    enabled: false              # 启用 TLS
    version: "1.3"              # TLS 版本
    cert_file: "/path/to/cert.pem"  # 商密证书
    key_file: "/path/to/key.pem"    # 商密私钥
    gm_sign_cert: "/path/to/sm2_sign.pem"  # 国密签名证书
    gm_sign_key: "/path/to/sm2_sign_key.pem"  # 国密签名私钥
    gm_enc_cert: "/path/to/sm2_enc.pem"  # 国密加密证书
    gm_enc_key: "/path/to/sm2_enc_key.pem"  # 国密加密私钥
    ca_file: "/path/to/ca.pem"  # CA 证书
    client_auth: "none"         # 客户端认证模式
```

**version 选项**：
- `1.2`: TLS 1.2
- `1.3`: TLS 1.3（推荐）
- `tlcp-1.1`: 国密 TLS 1.1
- `gm-tls-1.3`: 国密 TLS 1.3

**client_auth 选项**：
- `none`: 不要求客户端证书
- `request`: 请求但不强制客户端证书
- `require`: 强制要求客户端证书（mTLS）

详见 [TLS/mTLS 配置](../security/tls-mtls.md) 和 [国密支持](../security/gm-crypto.md)。

#### 数据加密 (security.encryption)

```yaml
security:
  encryption:
    enabled: false              # 启用数据加密
    algorithm: "auto"           # 加密算法
    master_key_source: "env"    # 主密钥来源
    master_key_file: "/secure/path/master.key"  # 主密钥文件
    key_rotation_days: 90       # 密钥轮换周期（天）
    key_retention_count: 3      # 保留旧密钥数量
```

**algorithm 选项**：
- `auto`: 根据 crypto_mode 自动选择
- `aes-256-gcm`: AES-256-GCM（商密）
- `sm4-gcm`: SM4-GCM（国密）

**master_key_source 选项**：
- `env`: 从环境变量读取（`TOKENGINX_MASTER_KEY`）
- `file`: 从文件读取
- `kms`: 从云 KMS 读取
- `hsm`: 从硬件安全模块读取（企业版）

##### KMS 配置

```yaml
security:
  encryption:
    master_key_source: "kms"
    kms:
      provider: "aliyun"        # aliyun | tencent | huawei | aws
      region: "cn-beijing"
      key_id: "your-kms-key-id"
      access_key_id: "${KMS_ACCESS_KEY_ID}"
      access_key_secret: "${KMS_ACCESS_KEY_SECRET}"
```

#### 防重放攻击 (security.anti_replay)

```yaml
security:
  anti_replay:
    enabled: false              # 启用防重放
    window_seconds: 300         # 时间窗口（秒）
    clock_skew_seconds: 30      # 时钟偏移容忍（秒）
    nonce_cache_size: 100000    # Nonce 缓存大小
    nonce_cache_backend: "memory"  # Nonce 缓存后端
    require_signature: false    # 要求签名
    signature_algorithm: "hmac-sha256"  # 签名算法
    enable_sequence: false      # 启用序列号
    sequence_tolerance: 10      # 序列号容忍度
    log_rejected_requests: true # 记录被拒绝的请求
```

**nonce_cache_backend 选项**：
- `memory`: 内存缓存
- `redis`: Redis 缓存

**signature_algorithm 选项**：
- `hmac-sha256`: HMAC-SHA256（商密）
- `hmac-sm3`: HMAC-SM3（国密）

详见 [防重放攻击](../security/anti-replay.md)。

#### 访问控制 (security.acl)

```yaml
security:
  acl:
    enabled: false              # 启用 ACL
    default_deny: true          # 默认拒绝策略
    rules_file: "/etc/tokenginx/acl.yaml"  # ACL 规则文件
    audit_denied: true          # 审计被拒绝的请求
    cache:
      enabled: true             # 启用 ACL 缓存
      ttl_seconds: 60           # 缓存 TTL（秒）
      max_entries: 10000        # 最大缓存条目
```

详见 [访问控制 (ACL)](../security/acl.md)。

#### 审计日志 (security.audit)

```yaml
security:
  audit:
    enabled: false              # 启用审计
    format: "json"              # 日志格式
    output: "file"              # 日志输出
    file_path: "/var/log/tokenginx/audit.log"  # 日志文件路径
    encrypt_log: false          # 加密日志
    event_types:                # 记录的事件类型
      - "authentication"
      - "authorization"
      - "data_access"
      - "configuration_change"
      - "security_event"
```

**format 选项**：
- `json`: JSON 格式（推荐）
- `text`: 文本格式

**output 选项**：
- `file`: 文件输出
- `syslog`: Syslog 输出
- `stdout`: 标准输出

##### 日志轮转

```yaml
security:
  audit:
    rotation:
      max_size_mb: 100          # 最大文件大小（MB）
      max_backups: 10           # 保留文件数量
      max_age_days: 30          # 保留天数
      compress: true            # 是否压缩
```

##### Syslog 配置

```yaml
security:
  audit:
    output: "syslog"
    syslog:
      network: "udp"            # udp | tcp
      address: "localhost:514"
      tag: "tokenginx"
```

#### 速率限制 (security.rate_limit)

```yaml
security:
  rate_limit:
    enabled: false              # 启用速率限制
    global_qps: 100000          # 全局 QPS
    per_user_qps: 10000         # 单用户 QPS
    per_ip_qps: 5000            # 单 IP QPS
    per_command:                # 单命令 QPS
      SET: 50000
      GET: 100000
      DELETE: 20000
    algorithm: "token_bucket"   # 限流算法
    response:
      http_status: 429
      message: "Rate limit exceeded"
```

**algorithm 选项**：
- `token_bucket`: 令牌桶算法
- `sliding_window`: 滑动窗口算法

### 协议配置 (protocols)

#### OAuth 2.0/OIDC (protocols.oauth)

```yaml
protocols:
  oauth:
    enabled: true               # 启用 OAuth
    default_token_ttl: 3600     # 默认 Access Token TTL（秒）
    default_refresh_token_ttl: 2592000  # 默认 Refresh Token TTL（秒）
    default_code_ttl: 300       # 默认 Authorization Code TTL（秒）
    key_prefix: "oauth"         # Token 键前缀
```

详见 [OAuth 2.0/OIDC 集成指南](../protocols/oauth.md)。

#### SAML 2.0 (protocols.saml)

```yaml
protocols:
  saml:
    enabled: false              # 启用 SAML
    default_assertion_ttl: 1800 # 默认 Assertion TTL（秒）
    default_session_ttl: 28800  # 默认 Session TTL（秒）
    key_prefix: "saml"          # Assertion 键前缀
```

详见 [SAML 2.0 集成指南](../protocols/saml.md)。

#### CAS (protocols.cas)

```yaml
protocols:
  cas:
    enabled: false              # 启用 CAS
    default_tgt_ttl: 7200       # 默认 TGT TTL（秒）
    default_st_ttl: 300         # 默认 ST TTL（秒）
    default_pt_ttl: 300         # 默认 PT TTL（秒）
    key_prefix: "cas"           # Ticket 键前缀
```

详见 [CAS 集成指南](../protocols/cas.md)。

### 监控配置 (monitoring)

#### Prometheus (monitoring.prometheus)

```yaml
monitoring:
  prometheus:
    enabled: false              # 启用 Prometheus
    addr: "0.0.0.0:9100"        # Metrics 端点地址
    path: "/metrics"            # Metrics 路径
```

#### 健康检查 (monitoring.health_check)

```yaml
monitoring:
  health_check:
    enabled: true               # 启用健康检查
    addr: "0.0.0.0:8081"        # 健康检查端点地址
    path: "/health"             # 健康检查路径
```

**健康检查响应**：

```json
{
  "status": "healthy",
  "timestamp": "2025-11-19T12:34:56Z",
  "version": "v0.1.0",
  "uptime_seconds": 86400,
  "checks": {
    "storage": "ok",
    "cache": "ok",
    "ttl": "ok"
  }
}
```

#### 性能分析 (monitoring.pprof)

```yaml
monitoring:
  pprof:
    enabled: false              # 启用 pprof
    addr: "localhost:6060"      # pprof 端点地址
```

**访问 pprof**：

```bash
# CPU profile
curl http://localhost:6060/debug/pprof/profile?seconds=30 > cpu.prof
go tool pprof cpu.prof

# Heap profile
curl http://localhost:6060/debug/pprof/heap > heap.prof
go tool pprof heap.prof

# Goroutine profile
curl http://localhost:6060/debug/pprof/goroutine > goroutine.prof
go tool pprof goroutine.prof
```

### 日志配置 (logging)

```yaml
logging:
  level: "info"                 # 日志级别
  format: "json"                # 日志格式
  output: "stdout"              # 日志输出
  file_path: "/var/log/tokenginx/tokenginx.log"  # 日志文件路径
  modules:                      # 调试的模块
    - "storage"
    - "security"
```

**level 选项**：
- `debug`: 调试级别
- `info`: 信息级别（推荐）
- `warn`: 警告级别
- `error`: 错误级别

**format 选项**：
- `json`: JSON 格式（推荐生产环境）
- `text`: 文本格式（推荐开发环境）

**output 选项**：
- `stdout`: 标准输出
- `file`: 文件输出

### 集群配置 (cluster)

> **注意**：集群功能在 v2.0.0+ 版本可用。

```yaml
cluster:
  enabled: false                # 启用集群模式
  node_id: "node1"              # 节点 ID
  gossip:
    bind_addr: "0.0.0.0:7946"   # Gossip 绑定地址
    seed_nodes:                 # 初始节点列表
      - "10.0.0.1:7946"
      - "10.0.0.2:7946"
  replication:
    replicas: 3                 # 副本数量
    quorum: 2                   # 法定人数
  sharding:
    enabled: true               # 启用分片
    virtual_nodes: 256          # 虚拟节点数量
```

## 环境变量

TokenginX 支持通过环境变量覆盖配置：

| 环境变量 | 配置路径 | 说明 |
|---------|---------|------|
| `TOKENGINX_TCP_ADDR` | `server.tcp_addr` | TCP 监听地址 |
| `TOKENGINX_GRPC_ADDR` | `server.grpc_addr` | gRPC 监听地址 |
| `TOKENGINX_HTTP_ADDR` | `server.http_addr` | HTTP 监听地址 |
| `TOKENGINX_DATA_DIR` | `storage.data_dir` | 数据目录 |
| `TOKENGINX_TLS_ENABLED` | `security.tls.enabled` | 启用 TLS |
| `TOKENGINX_TLS_CERT` | `security.tls.cert_file` | TLS 证书路径 |
| `TOKENGINX_TLS_KEY` | `security.tls.key_file` | TLS 私钥路径 |
| `TOKENGINX_MASTER_KEY` | `security.encryption.master_key_source=env` | 主密钥 |
| `TOKENGINX_LOG_LEVEL` | `logging.level` | 日志级别 |
| `KMS_ACCESS_KEY_ID` | `security.encryption.kms.access_key_id` | KMS 访问密钥 ID |
| `KMS_ACCESS_KEY_SECRET` | `security.encryption.kms.access_key_secret` | KMS 访问密钥 |
| `HSM_PIN` | `security.encryption.hsm.pin` | HSM PIN 码 |

**使用示例**：

```bash
# 设置监听地址
export TOKENGINX_TCP_ADDR="0.0.0.0:6380"
export TOKENGINX_GRPC_ADDR="0.0.0.0:9090"
export TOKENGINX_HTTP_ADDR="0.0.0.0:8080"

# 设置主密钥
export TOKENGINX_MASTER_KEY="0123456789abcdef..."

# 启动服务器
./tokenginx-server -config config.yaml
```

## 配置验证

使用 `--validate` 标志验证配置文件：

```bash
./tokenginx-server --validate -config config.yaml
```

**输出示例**：

```
Configuration validation result:
✓ Server configuration is valid
✓ Storage configuration is valid
✓ Security configuration is valid
! Warning: TLS is disabled (security.tls.enabled=false)
! Warning: ACL is disabled (security.acl.enabled=false)
✓ Protocol configuration is valid
✓ Monitoring configuration is valid
✓ Logging configuration is valid

Configuration is valid with 2 warnings.
```

## 配置最佳实践

### 开发环境配置

```yaml
server:
  tcp_addr: "localhost:6380"
  grpc_addr: "localhost:9090"
  http_addr: "localhost:8080"

storage:
  enable_persistence: false

security:
  tls:
    enabled: false
  acl:
    enabled: false

logging:
  level: "debug"
  format: "text"
  output: "stdout"
```

### 生产环境配置

```yaml
server:
  tcp_addr: "0.0.0.0:6380"
  grpc_addr: "0.0.0.0:9090"
  http_addr: "0.0.0.0:8080"
  max_connections: 10000

storage:
  enable_persistence: true
  persistence:
    wal:
      enabled: true
      sync_policy: "everysec"

security:
  crypto_mode: "auto"
  tls:
    enabled: true
    version: "1.3"
    client_auth: "require"
  encryption:
    enabled: true
    master_key_source: "kms"
  anti_replay:
    enabled: true
  acl:
    enabled: true
    default_deny: true
  audit:
    enabled: true
  rate_limit:
    enabled: true

monitoring:
  prometheus:
    enabled: true
  health_check:
    enabled: true

logging:
  level: "info"
  format: "json"
  output: "file"
```

## 故障排查

### 配置错误

**问题**: 启动时报配置错误

**解决**:
1. 使用 `--validate` 验证配置
2. 检查 YAML 语法（缩进、引号等）
3. 查看详细错误信息

### TLS 证书问题

**问题**: TLS 握手失败

**解决**:
1. 验证证书路径是否正确
2. 检查证书和私钥是否匹配
3. 确认证书未过期
4. 检查 CA 证书是否配置正确

### 权限问题

**问题**: 无法写入数据目录

**解决**:
```bash
# 确保数据目录存在且有正确权限
mkdir -p /var/lib/tokenginx
chown -R tokenginx:tokenginx /var/lib/tokenginx
chmod 750 /var/lib/tokenginx
```

## 相关文档

- [TLS/mTLS 配置](../security/tls-mtls.md)
- [国密支持](../security/gm-crypto.md)
- [防重放攻击](../security/anti-replay.md)
- [访问控制 (ACL)](../security/acl.md)
- [OAuth 2.0/OIDC 集成指南](../protocols/oauth.md)
- [SAML 2.0 集成指南](../protocols/saml.md)
- [CAS 集成指南](../protocols/cas.md)
