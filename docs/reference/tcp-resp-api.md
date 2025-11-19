# TCP (RESP) 协议参考

## 概述

TokenginX 支持 Redis RESP (REdis Serialization Protocol) 协议,可以使用任何标准的 Redis 客户端连接。RESP 是一个二进制安全的文本协议,具有高性能和易于实现的特点。

**默认端口**: `6380`

**协议版本**: RESP2 (兼容 Redis 6.x)

## 连接方式

### 基本连接

```bash
# 使用 redis-cli 连接
redis-cli -h localhost -p 6380

# 使用密码认证
redis-cli -h localhost -p 6380 -a your-api-key

# 使用 TLS 连接
redis-cli -h localhost -p 6380 --tls --cacert /path/to/ca.pem
```

### 客户端库连接

各语言客户端库示例参见:
- [ASP.NET Core 快速指南](../quickstart/aspnet-core.md)
- [Java 快速指南](../quickstart/java.md)
- [PHP 快速指南](../quickstart/php.md)
- [Go 快速指南](../quickstart/go.md)
- [Rust 快速指南](../quickstart/rust.md)

## 认证命令

### AUTH

使用密码认证。

**语法**:
```
AUTH password
```

**参数**:
- `password`: API Key 或密码

**返回值**:
- 成功: `OK`
- 失败: 错误信息

**示例**:
```
AUTH your-api-key
```

**错误**:
- `(error) ERR invalid password`: 密码错误
- `(error) NOAUTH Authentication required`: 需要认证

## 基本键值操作

### SET

设置键值对。

**语法**:
```
SET key value [EX seconds|PX milliseconds] [NX|XX]
```

**参数**:
- `key`: 键名
- `value`: 值(字符串)
- `EX seconds`: 设置过期时间(秒)
- `PX milliseconds`: 设置过期时间(毫秒)
- `NX`: 仅当键不存在时设置
- `XX`: 仅当键已存在时设置

**返回值**:
- 成功: `OK`
- 失败: `(nil)` (当使用 NX/XX 条件不满足时)

**示例**:
```
# 设置简单键值对
SET mykey "Hello World"

# 设置带 TTL 的键值对(3600 秒)
SET oauth:token:abc123 "{\"user_id\":\"user001\"}" EX 3600

# 仅当键不存在时设置
SET mykey "value" NX

# 设置带毫秒级 TTL
SET session:xyz "{\"data\":\"...\"}" PX 300000
```

**性能**:
- 时间复杂度: O(1)
- 典型延迟: P99 < 0.5ms

### GET

获取键的值。

**语法**:
```
GET key
```

**参数**:
- `key`: 键名

**返回值**:
- 存在: 键对应的值
- 不存在或已过期: `(nil)`

**示例**:
```
GET oauth:token:abc123
# 返回: "{\"user_id\":\"user001\",\"scope\":\"read write\"}"

GET nonexistent
# 返回: (nil)
```

**性能**:
- 时间复杂度: O(1)
- 典型延迟: P99 < 0.3ms

### DEL

删除一个或多个键。

**语法**:
```
DEL key [key ...]
```

**参数**:
- `key`: 一个或多个键名

**返回值**:
- 整数: 被删除的键数量

**示例**:
```
DEL oauth:token:abc123
# 返回: (integer) 1

DEL key1 key2 key3
# 返回: (integer) 3

DEL nonexistent
# 返回: (integer) 0
```

**性能**:
- 时间复杂度: O(N),N 为键的数量
- 典型延迟: P99 < 0.5ms (单个键)

### EXISTS

检查键是否存在。

**语法**:
```
EXISTS key [key ...]
```

**参数**:
- `key`: 一个或多个键名

**返回值**:
- 整数: 存在的键数量

**示例**:
```
EXISTS oauth:token:abc123
# 返回: (integer) 1

EXISTS key1 key2 key3
# 返回: (integer) 2 (假设 key1 和 key2 存在)

EXISTS nonexistent
# 返回: (integer) 0
```

**性能**:
- 时间复杂度: O(N),N 为键的数量
- 典型延迟: P99 < 0.3ms

### TTL

获取键的剩余生存时间(秒)。

**语法**:
```
TTL key
```

**参数**:
- `key`: 键名

**返回值**:
- 正整数: 剩余秒数
- `-1`: 键存在但没有设置过期时间
- `-2`: 键不存在

**示例**:
```
TTL oauth:token:abc123
# 返回: (integer) 3456

TTL persistent_key
# 返回: (integer) -1

TTL nonexistent
# 返回: (integer) -2
```

**性能**:
- 时间复杂度: O(1)
- 典型延迟: P99 < 0.2ms

### PTTL

获取键的剩余生存时间(毫秒)。

**语法**:
```
PTTL key
```

**参数**:
- `key`: 键名

**返回值**:
- 正整数: 剩余毫秒数
- `-1`: 键存在但没有设置过期时间
- `-2`: 键不存在

**示例**:
```
PTTL oauth:token:abc123
# 返回: (integer) 3456789
```

### EXPIRE

设置键的过期时间(秒)。

**语法**:
```
EXPIRE key seconds
```

**参数**:
- `key`: 键名
- `seconds`: 过期时间(秒)

**返回值**:
- `1`: 成功设置
- `0`: 键不存在

**示例**:
```
EXPIRE oauth:token:abc123 3600
# 返回: (integer) 1
```

### PEXPIRE

设置键的过期时间(毫秒)。

**语法**:
```
PEXPIRE key milliseconds
```

**参数**:
- `key`: 键名
- `milliseconds`: 过期时间(毫秒)

**返回值**:
- `1`: 成功设置
- `0`: 键不存在

**示例**:
```
PEXPIRE session:xyz 300000
# 返回: (integer) 1
```

## 批量操作

### MGET

批量获取多个键的值。

**语法**:
```
MGET key [key ...]
```

**参数**:
- `key`: 一个或多个键名

**返回值**:
- 数组: 每个键对应的值,不存在的键返回 `(nil)`

**示例**:
```
MGET oauth:token:abc123 oauth:token:def456 oauth:token:ghi789
# 返回:
# 1) "{\"user_id\":\"user001\"}"
# 2) "{\"user_id\":\"user002\"}"
# 3) (nil)
```

**性能**:
- 时间复杂度: O(N),N 为键的数量
- 典型延迟: P99 < 1ms (10 个键)

### MSET

批量设置多个键值对。

**语法**:
```
MSET key value [key value ...]
```

**参数**:
- `key value`: 键值对

**返回值**:
- `OK`

**示例**:
```
MSET key1 "value1" key2 "value2" key3 "value3"
# 返回: OK
```

**注意**: MSET 不支持设置 TTL,如需 TTL 请使用多个 SET 命令。

**性能**:
- 时间复杂度: O(N),N 为键的数量
- 典型延迟: P99 < 2ms (10 个键)

## 扫描操作

### SCAN

迭代扫描键。

**语法**:
```
SCAN cursor [MATCH pattern] [COUNT count]
```

**参数**:
- `cursor`: 游标(初始为 0)
- `MATCH pattern`: 匹配模式(可选)
- `COUNT count`: 每次迭代返回的数量提示(可选,默认 10)

**返回值**:
- 数组:
  1. 新游标(0 表示迭代结束)
  2. 键列表

**示例**:
```
# 第一次扫描
SCAN 0 MATCH oauth:token:* COUNT 100
# 返回:
# 1) "17"
# 2) 1) "oauth:token:abc123"
#    2) "oauth:token:def456"

# 继续扫描
SCAN 17 MATCH oauth:token:* COUNT 100
# 返回:
# 1) "0"
# 2) 1) "oauth:token:ghi789"
```

**性能**:
- 时间复杂度: O(1) 每次迭代
- 典型延迟: P99 < 1ms

**注意事项**:
- SCAN 保证最终会遍历所有键,但可能返回重复键
- 不阻塞服务器,适合大数据集
- COUNT 只是提示,实际返回数量可能不同

### KEYS

查找所有匹配模式的键(不推荐在生产环境使用)。

**语法**:
```
KEYS pattern
```

**参数**:
- `pattern`: 匹配模式

**返回值**:
- 数组: 匹配的键列表

**示例**:
```
KEYS oauth:token:*
# 返回:
# 1) "oauth:token:abc123"
# 2) "oauth:token:def456"
# 3) "oauth:token:ghi789"
```

**警告**:
- ⚠️ 时间复杂度: O(N),N 为数据库中的键总数
- ⚠️ 会阻塞服务器,不适合大数据集
- ✅ 生产环境请使用 SCAN 命令

## 管道(Pipelining)

支持管道操作以减少网络往返次数。

**示例(使用 redis-cli)**:
```bash
echo -e "SET key1 value1\nSET key2 value2\nGET key1\nGET key2" | redis-cli -p 6380 --pipe
```

**示例(使用 Go 客户端)**:
```go
pipe := client.Pipeline()
pipe.Set(ctx, "key1", "value1", 0)
pipe.Set(ctx, "key2", "value2", 0)
pipe.Get(ctx, "key1")
pipe.Get(ctx, "key2")
_, err := pipe.Exec(ctx)
```

**性能优势**:
- 减少网络 RTT
- 10 个命令的管道操作延迟 ≈ 单个命令延迟

## 事务(Transactions)

支持 MULTI/EXEC 事务。

**语法**:
```
MULTI
command1
command2
...
EXEC
```

**示例**:
```
MULTI
SET key1 "value1"
SET key2 "value2"
GET key1
EXEC
# 返回:
# 1) OK
# 2) OK
# 3) "value1"
```

**DISCARD**: 取消事务
```
MULTI
SET key1 "value1"
DISCARD
```

**注意事项**:
- 事务中的命令会原子性执行
- 不支持回滚(rollback)
- 命令错误会在 EXEC 时返回

## 连接管理

### PING

测试连接。

**语法**:
```
PING [message]
```

**返回值**:
- 无参数: `PONG`
- 有参数: 返回参数值

**示例**:
```
PING
# 返回: PONG

PING "Hello"
# 返回: "Hello"
```

### ECHO

回显消息。

**语法**:
```
ECHO message
```

**返回值**:
- 返回消息内容

**示例**:
```
ECHO "Hello World"
# 返回: "Hello World"
```

### QUIT

关闭连接。

**语法**:
```
QUIT
```

**返回值**:
- `OK`

## 服务器信息

### INFO

获取服务器信息。

**语法**:
```
INFO [section]
```

**参数**:
- `section`: 信息段(可选),如 `server`、`stats`、`memory`

**返回值**:
- 服务器信息(文本格式)

**示例**:
```
INFO server
# 返回:
# # Server
# tokenginx_version:1.0.0
# os:Linux 5.15.0-1234-generic x86_64
# process_id:12345
# uptime_in_seconds:86400
# uptime_in_days:1

INFO stats
# 返回:
# # Stats
# total_commands_processed:1000000
# instantaneous_ops_per_sec:15234
# total_keys:12345
```

### DBSIZE

获取数据库键总数。

**语法**:
```
DBSIZE
```

**返回值**:
- 整数: 键总数

**示例**:
```
DBSIZE
# 返回: (integer) 12345
```

### CONFIG GET

获取配置参数。

**语法**:
```
CONFIG GET parameter
```

**参数**:
- `parameter`: 配置参数名(支持通配符)

**返回值**:
- 数组: 参数名和值

**示例**:
```
CONFIG GET maxmemory
# 返回:
# 1) "maxmemory"
# 2) "2147483648"

CONFIG GET max*
# 返回:
# 1) "maxmemory"
# 2) "2147483648"
# 3) "maxclients"
# 4) "10000"
```

**权限**: 需要 admin 权限

## OAuth 2.0 扩展命令

TokenginX 提供了 OAuth 2.0 的扩展命令,简化令牌管理。

### OAUTH.SET

设置 OAuth Token。

**语法**:
```
OAUTH.SET token_id user_id scope ttl
```

**参数**:
- `token_id`: Token 唯一标识
- `user_id`: 用户 ID
- `scope`: 授权范围
- `ttl`: 过期时间(秒)

**返回值**:
- `OK`

**示例**:
```
OAUTH.SET abc123 user001 "read write" 3600
# 返回: OK
```

**等价操作**:
```
SET oauth:token:abc123 "{\"user_id\":\"user001\",\"scope\":\"read write\"}" EX 3600
```

### OAUTH.GET

获取 OAuth Token。

**语法**:
```
OAUTH.GET token_id
```

**参数**:
- `token_id`: Token 唯一标识

**返回值**:
- 数组: `[user_id, scope, created_at, expires_at]`
- `(nil)`: Token 不存在或已过期

**示例**:
```
OAUTH.GET abc123
# 返回:
# 1) "user001"
# 2) "read write"
# 3) (integer) 1700000000
# 4) (integer) 1700003600
```

### OAUTH.INTROSPECT

Token 内省(RFC 7662)。

**语法**:
```
OAUTH.INTROSPECT token_id
```

**参数**:
- `token_id`: Token 唯一标识

**返回值**:
- 数组: `[active, user_id, scope, exp]`

**示例**:
```
OAUTH.INTROSPECT abc123
# 返回:
# 1) (integer) 1  # active
# 2) "user001"    # user_id
# 3) "read write" # scope
# 4) (integer) 1700003600  # exp
```

### OAUTH.REVOKE

撤销 OAuth Token。

**语法**:
```
OAUTH.REVOKE token_id
```

**参数**:
- `token_id`: Token 唯一标识

**返回值**:
- `1`: 成功撤销
- `0`: Token 不存在

**示例**:
```
OAUTH.REVOKE abc123
# 返回: (integer) 1
```

## SAML 2.0 扩展命令

### SAML.SET

设置 SAML 会话。

**语法**:
```
SAML.SET session_index name_id assertion ttl
```

**参数**:
- `session_index`: 会话索引
- `name_id`: 用户名称标识
- `assertion`: SAML 断言(Base64 编码)
- `ttl`: 过期时间(秒)

**返回值**:
- `OK`

**示例**:
```
SAML.SET xyz789 "user@example.com" "PFNhbWwuLi4=" 1800
# 返回: OK
```

### SAML.GET

获取 SAML 会话。

**语法**:
```
SAML.GET session_index
```

**参数**:
- `session_index`: 会话索引

**返回值**:
- 数组: `[name_id, assertion, created_at, expires_at]`
- `(nil)`: 会话不存在或已过期

**示例**:
```
SAML.GET xyz789
# 返回:
# 1) "user@example.com"
# 2) "PFNhbWwuLi4="
# 3) (integer) 1700000000
# 4) (integer) 1700001800
```

## CAS 扩展命令

### CAS.SET_TGT

设置 TGT (Ticket Granting Ticket)。

**语法**:
```
CAS.SET_TGT tgt_id user_id ttl
```

**参数**:
- `tgt_id`: TGT 唯一标识
- `user_id`: 用户 ID
- `ttl`: 过期时间(秒)

**返回值**:
- `OK`

**示例**:
```
CAS.SET_TGT TGT-1-abc123 user001 7200
# 返回: OK
```

### CAS.SET_ST

设置 ST (Service Ticket)。

**语法**:
```
CAS.SET_ST st_id tgt_id service ttl
```

**参数**:
- `st_id`: ST 唯一标识
- `tgt_id`: 关联的 TGT ID
- `service`: 服务 URL
- `ttl`: 过期时间(秒)

**返回值**:
- `OK`

**示例**:
```
CAS.SET_ST ST-1-xyz789 TGT-1-abc123 "https://app.example.com" 300
# 返回: OK
```

### CAS.VALIDATE_ST

验证 ST。

**语法**:
```
CAS.VALIDATE_ST st_id service
```

**参数**:
- `st_id`: ST 唯一标识
- `service`: 服务 URL

**返回值**:
- 数组: `[valid, user_id]`

**示例**:
```
CAS.VALIDATE_ST ST-1-xyz789 "https://app.example.com"
# 返回:
# 1) (integer) 1  # valid
# 2) "user001"    # user_id
```

## RESP 协议格式

TokenginX 使用 RESP2 协议进行通信。

### 数据类型

- **简单字符串**: `+OK\r\n`
- **错误**: `-(error) ERR message\r\n`
- **整数**: `:1000\r\n`
- **批量字符串**: `$6\r\nfoobar\r\n`
- **数组**: `*2\r\n$3\r\nfoo\r\n$3\r\nbar\r\n`
- **空值**: `$-1\r\n`

### 示例

**请求**: `SET mykey "Hello"`
```
*3\r\n
$3\r\n
SET\r\n
$5\r\n
mykey\r\n
$5\r\n
Hello\r\n
```

**响应**: `OK`
```
+OK\r\n
```

## 性能基准

### 延迟分布

| 操作 | P50 | P95 | P99 | P999 |
|------|-----|-----|-----|------|
| GET  | 0.15ms | 0.25ms | 0.3ms | 1ms |
| SET  | 0.2ms | 0.35ms | 0.5ms | 2ms |
| DEL  | 0.18ms | 0.3ms | 0.45ms | 1.5ms |
| MGET(10) | 0.4ms | 0.7ms | 1ms | 3ms |
| SCAN | 0.5ms | 1ms | 1.5ms | 5ms |

### 吞吐量

- 单连接: 20,000 QPS
- 10 并发连接: 120,000 QPS
- 管道(100 命令): 500,000+ QPS

## 错误码

- `WRONGTYPE`: 操作与键类型不匹配
- `NOAUTH`: 需要认证
- `ERR`: 通用错误
- `NOPERM`: 权限不足
- `NOTFOUND`: 键不存在
- `EXPIRED`: 键已过期

## 最佳实践

### 1. 使用连接池

```python
# Python 示例
import redis

pool = redis.ConnectionPool(
    host='localhost',
    port=6380,
    password='your-api-key',
    max_connections=50
)
client = redis.Redis(connection_pool=pool)
```

### 2. 使用管道减少 RTT

```python
pipe = client.pipeline()
pipe.set('key1', 'value1')
pipe.set('key2', 'value2')
pipe.get('key1')
pipe.execute()
```

### 3. 使用 SCAN 而非 KEYS

```python
# ❌ 不推荐
keys = client.keys('oauth:token:*')

# ✅ 推荐
cursor = 0
keys = []
while True:
    cursor, batch = client.scan(cursor, match='oauth:token:*', count=100)
    keys.extend(batch)
    if cursor == 0:
        break
```

### 4. 设置合理的超时

```python
client = redis.Redis(
    host='localhost',
    port=6380,
    socket_connect_timeout=5,
    socket_timeout=5
)
```

## 下一步

- 查看 [核心功能参考](./core-features.md)
- 了解 [HTTP/REST API](./http-rest-api.md)
- 配置 [TLS/mTLS](../security/tls-mtls.md)
