# 核心功能参考

## 概述

TokenginX 提供高性能的 SSO 会话存储功能，支持多种协议和接口。本文档详细介绍核心功能和使用方法。

## 核心操作

### SET - 设置会话

设置一个会话数据，支持自动过期。

**语法**：
```
SET key value [EX seconds] [PX milliseconds] [EXAT unix-time-seconds] [PXAT unix-time-milliseconds]
```

**参数**：
- `key`: 会话键名，通常使用格式如 `oauth:token:{token_id}` 或 `saml:session:{session_id}`
- `value`: 会话数据（JSON 字符串或二进制数据）
- `EX seconds`: 设置过期时间（秒）
- `PX milliseconds`: 设置过期时间（毫秒）
- `EXAT timestamp`: 设置过期时间戳（秒）
- `PXAT timestamp`: 设置过期时间戳（毫秒）

**返回值**：
- `OK`: 成功
- 错误信息：失败

**示例**：
```bash
# 设置一个 1 小时过期的 OAuth token
SET oauth:token:abc123 '{"user_id":"user001","scope":"read write"}' EX 3600

# 设置一个 30 分钟过期的 SAML 会话
SET saml:session:xyz789 '{"name_id":"user@example.com","assertion":"..."}' EX 1800
```

### GET - 获取会话

获取指定键的会话数据。

**语法**：
```
GET key
```

**参数**：
- `key`: 会话键名

**返回值**：
- 会话数据：存在且未过期
- `(nil)`: 键不存在或已过期

**示例**：
```bash
GET oauth:token:abc123
# => '{"user_id":"user001","scope":"read write"}'

GET oauth:token:notexist
# => (nil)
```

### DEL - 删除会话

删除一个或多个会话。

**语法**：
```
DEL key [key ...]
```

**参数**：
- `key`: 要删除的键名，可以指定多个

**返回值**：
- 删除的键数量（整数）

**示例**：
```bash
DEL oauth:token:abc123
# => 1

DEL oauth:token:key1 oauth:token:key2 oauth:token:key3
# => 3
```

### EXISTS - 检查会话是否存在

检查一个或多个键是否存在。

**语法**：
```
EXISTS key [key ...]
```

**参数**：
- `key`: 要检查的键名

**返回值**：
- 存在的键数量（整数）

**示例**：
```bash
EXISTS oauth:token:abc123
# => 1

EXISTS oauth:token:abc123 oauth:token:notexist
# => 1
```

### TTL - 获取剩余生存时间

获取键的剩余生存时间（秒）。

**语法**：
```
TTL key
```

**参数**：
- `key`: 键名

**返回值**：
- 正整数：剩余秒数
- `-1`: 键存在但没有设置过期时间
- `-2`: 键不存在

**示例**：
```bash
SET oauth:token:abc123 "value" EX 3600
TTL oauth:token:abc123
# => 3600

TTL oauth:token:notexist
# => -2
```

### PTTL - 获取剩余生存时间（毫秒）

获取键的剩余生存时间（毫秒）。

**语法**：
```
PTTL key
```

**参数**：
- `key`: 键名

**返回值**：
- 正整数：剩余毫秒数
- `-1`: 键存在但没有设置过期时间
- `-2`: 键不存在

### EXPIRE - 设置过期时间

为已存在的键设置过期时间。

**语法**：
```
EXPIRE key seconds
```

**参数**：
- `key`: 键名
- `seconds`: 过期秒数

**返回值**：
- `1`: 成功设置
- `0`: 键不存在

**示例**：
```bash
SET oauth:token:abc123 "value"
EXPIRE oauth:token:abc123 3600
# => 1
```

### KEYS - 查找键

根据模式查找键名。

**语法**：
```
KEYS pattern
```

**参数**：
- `pattern`: 匹配模式，支持通配符：
  - `*`: 匹配任意字符
  - `?`: 匹配单个字符
  - `[abc]`: 匹配 a、b 或 c
  - `[^a]`: 匹配除 a 外的字符

**返回值**：
- 匹配的键名列表

**示例**：
```bash
KEYS oauth:token:*
# => ["oauth:token:abc123", "oauth:token:def456"]

KEYS saml:session:*
# => ["saml:session:xyz789"]
```

**注意**：在生产环境中谨慎使用 KEYS 命令，可能影响性能。建议使用 SCAN 命令。

### SCAN - 迭代扫描键

以游标方式迭代扫描键空间。

**语法**：
```
SCAN cursor [MATCH pattern] [COUNT count]
```

**参数**：
- `cursor`: 游标位置，初始为 0
- `MATCH pattern`: 可选，匹配模式
- `COUNT count`: 可选，每次返回的键数量提示

**返回值**：
- 下一个游标位置
- 匹配的键列表

**示例**：
```bash
SCAN 0 MATCH oauth:token:* COUNT 100
# => ["17", ["oauth:token:abc123", "oauth:token:def456"]]

SCAN 17 MATCH oauth:token:* COUNT 100
# => ["0", ["oauth:token:ghi789"]]  # 游标为 0 表示迭代结束
```

## 协议特定操作

### OAuth 2.0 操作

#### OAUTH.TOKEN.SET - 设置 OAuth Token

设置一个 OAuth 访问令牌。

**语法**：
```
OAUTH.TOKEN.SET token_id user_id scope [refresh_token refresh_token_id] [expires_in seconds]
```

**参数**：
- `token_id`: 访问令牌 ID
- `user_id`: 用户 ID
- `scope`: 授权范围（空格分隔）
- `refresh_token`: 可选，关联的刷新令牌 ID
- `expires_in`: 可选，过期秒数（默认 3600）

**返回值**：
- `OK`: 成功

**示例**：
```bash
OAUTH.TOKEN.SET abc123 user001 "read write" refresh_token xyz789 expires_in 3600
# => OK
```

#### OAUTH.TOKEN.GET - 获取 OAuth Token

获取 OAuth 访问令牌信息。

**语法**：
```
OAUTH.TOKEN.GET token_id
```

**返回值**：
- Token 信息（JSON 格式）

**示例**：
```bash
OAUTH.TOKEN.GET abc123
# => {"token_id":"abc123","user_id":"user001","scope":"read write","refresh_token":"xyz789","expires_at":1700000000}
```

#### OAUTH.TOKEN.INTROSPECT - 令牌内省（RFC 7662）

验证令牌并返回元数据。

**语法**：
```
OAUTH.TOKEN.INTROSPECT token_id
```

**返回值**：
- 令牌元数据（符合 RFC 7662 标准）

**示例**：
```bash
OAUTH.TOKEN.INTROSPECT abc123
# => {"active":true,"scope":"read write","client_id":"client001","username":"user001","exp":1700000000}
```

### SAML 2.0 操作

#### SAML.SESSION.SET - 设置 SAML 会话

设置一个 SAML 会话。

**语法**：
```
SAML.SESSION.SET session_index name_id assertion [expires_in seconds]
```

**参数**：
- `session_index`: 会话索引
- `name_id`: Name ID
- `assertion`: SAML 断言（Base64 编码）
- `expires_in`: 可选，过期秒数（默认 1800）

**返回值**：
- `OK`: 成功

**示例**：
```bash
SAML.SESSION.SET xyz789 "user@example.com" "PFNhbWw..." expires_in 1800
# => OK
```

#### SAML.SESSION.GET - 获取 SAML 会话

获取 SAML 会话信息。

**语法**：
```
SAML.SESSION.GET session_index
```

**返回值**：
- 会话信息（JSON 格式）

### CAS 操作

#### CAS.TGT.SET - 设置票据授予票据

设置一个 CAS TGT（Ticket Granting Ticket）。

**语法**：
```
CAS.TGT.SET tgt_id user_id [expires_in seconds]
```

**参数**：
- `tgt_id`: TGT ID
- `user_id`: 用户 ID
- `expires_in`: 可选，过期秒数（默认 7200）

**返回值**：
- `OK`: 成功

#### CAS.ST.SET - 设置服务票据

设置一个 CAS ST（Service Ticket）。

**语法**：
```
CAS.ST.SET st_id tgt_id service [expires_in seconds]
```

**参数**：
- `st_id`: ST ID
- `tgt_id`: 关联的 TGT ID
- `service`: 服务 URL
- `expires_in`: 可选，过期秒数（默认 300）

**返回值**：
- `OK`: 成功

## 批量操作

### MGET - 批量获取

批量获取多个键的值。

**语法**：
```
MGET key [key ...]
```

**参数**：
- `key`: 要获取的键名，可以指定多个

**返回值**：
- 值列表，不存在的键返回 nil

**示例**：
```bash
MGET oauth:token:abc123 oauth:token:def456
# => ["value1", "value2"]
```

### MSET - 批量设置

批量设置多个键值对。

**语法**：
```
MSET key value [key value ...]
```

**参数**：
- `key value`: 键值对，可以指定多个

**返回值**：
- `OK`: 成功

**示例**：
```bash
MSET oauth:token:abc123 "value1" oauth:token:def456 "value2"
# => OK
```

## 事务操作

### MULTI - 开始事务

标记事务块的开始。

**语法**：
```
MULTI
```

### EXEC - 执行事务

执行事务块内的所有命令。

**语法**：
```
EXEC
```

### DISCARD - 取消事务

取消事务，放弃执行事务块内的所有命令。

**语法**：
```
DISCARD
```

**示例**：
```bash
MULTI
SET oauth:token:abc123 "value1"
SET oauth:token:def456 "value2"
EXEC
# => [OK, OK]
```

## 性能指标

### INFO - 服务器信息

获取服务器统计信息和配置。

**语法**：
```
INFO [section]
```

**参数**：
- `section`: 可选，指定信息部分（server、stats、memory、cpu等）

**返回值**：
- 服务器信息文本

**示例**：
```bash
INFO stats
# => # Stats
# total_connections_received:1000
# total_commands_processed:50000
# instantaneous_ops_per_sec:10523
# ...
```

### DBSIZE - 数据库大小

返回当前数据库的键数量。

**语法**：
```
DBSIZE
```

**返回值**：
- 键数量（整数）

**示例**：
```bash
DBSIZE
# => 12345
```

## 最佳实践

### 键命名规范

建议使用以下键命名规范：

```
<protocol>:<type>:<id>
```

示例：
- `oauth:token:abc123`：OAuth 访问令牌
- `oauth:refresh:xyz789`：OAuth 刷新令牌
- `saml:session:session123`：SAML 会话
- `cas:tgt:tgt456`：CAS 票据授予票据
- `cas:st:st789`：CAS 服务票据

### 过期时间设置

根据协议和安全要求设置合适的过期时间：

- **OAuth Access Token**: 3600 秒（1 小时）
- **OAuth Refresh Token**: 2592000 秒（30 天）
- **SAML Session**: 1800 秒（30 分钟）
- **CAS TGT**: 7200 秒（2 小时）
- **CAS ST**: 300 秒（5 分钟）

### 错误处理

客户端应正确处理以下错误：

- **键不存在**: 返回 `(nil)`，表示会话已过期或不存在
- **权限不足**: ACL 拒绝访问
- **速率限制**: 超过 QPS 限制
- **网络错误**: 连接超时、连接断开

### 连接管理

- 使用连接池管理连接
- 设置合理的超时时间
- 实现自动重连机制
- 使用 TLS/mTLS 加密通信

## 性能优化

### 使用管道（Pipeline）

批量发送命令，减少网络往返：

```python
# Python 示例
pipe = client.pipeline()
pipe.set('oauth:token:1', 'value1')
pipe.set('oauth:token:2', 'value2')
pipe.set('oauth:token:3', 'value3')
pipe.execute()
```

### 使用批量操作

优先使用 MGET、MSET 等批量操作命令。

### 避免阻塞命令

在生产环境中避免使用 KEYS 等阻塞命令，使用 SCAN 替代。
