# HTTP/REST API 参考

## 概述

TokenginX 提供 RESTful HTTP API，方便各种编程语言和平台接入。所有 API 端点支持 TLS 加密和 mTLS 双向认证。

**Base URL**: `https://your-tokenginx-server:8443`

## 认证

### API Key 认证

在请求头中包含 API Key：

```http
Authorization: Bearer YOUR_API_KEY
```

### mTLS 认证

使用客户端证书进行双向 TLS 认证（推荐）。

### 请求签名认证

使用 HMAC-SHA256 签名防重放攻击：

```http
X-Timestamp: 1700000000
X-Nonce: random-nonce-value
X-Signature: hmac-sha256-signature
```

## 会话管理 API

### 设置会话

**端点**: `POST /api/v1/sessions`

**请求体**:
```json
{
  "key": "oauth:token:abc123",
  "value": {
    "user_id": "user001",
    "scope": "read write"
  },
  "ttl": 3600
}
```

**响应**:
```json
{
  "status": "success",
  "message": "Session created",
  "data": {
    "key": "oauth:token:abc123",
    "expires_at": "2025-11-19T12:00:00Z"
  }
}
```

**状态码**:
- `201 Created`: 成功创建
- `400 Bad Request`: 请求参数错误
- `401 Unauthorized`: 认证失败
- `403 Forbidden`: 权限不足
- `429 Too Many Requests`: 超过速率限制

### 获取会话

**端点**: `GET /api/v1/sessions/{key}`

**路径参数**:
- `key`: 会话键名（URL 编码）

**响应**:
```json
{
  "status": "success",
  "data": {
    "key": "oauth:token:abc123",
    "value": {
      "user_id": "user001",
      "scope": "read write"
    },
    "ttl": 3456,
    "created_at": "2025-11-19T11:00:00Z",
    "expires_at": "2025-11-19T12:00:00Z"
  }
}
```

**状态码**:
- `200 OK`: 成功
- `404 Not Found`: 会话不存在或已过期

### 删除会话

**端点**: `DELETE /api/v1/sessions/{key}`

**响应**:
```json
{
  "status": "success",
  "message": "Session deleted"
}
```

**状态码**:
- `204 No Content`: 成功删除
- `404 Not Found`: 会话不存在

### 检查会话是否存在

**端点**: `HEAD /api/v1/sessions/{key}`

**响应**:
- 无响应体

**状态码**:
- `200 OK`: 会话存在
- `404 Not Found`: 会话不存在

### 更新会话 TTL

**端点**: `PATCH /api/v1/sessions/{key}/ttl`

**请求体**:
```json
{
  "ttl": 7200
}
```

**响应**:
```json
{
  "status": "success",
  "data": {
    "key": "oauth:token:abc123",
    "ttl": 7200,
    "expires_at": "2025-11-19T14:00:00Z"
  }
}
```

## 批量操作 API

### 批量获取会话

**端点**: `POST /api/v1/sessions/batch/get`

**请求体**:
```json
{
  "keys": [
    "oauth:token:abc123",
    "oauth:token:def456",
    "oauth:token:ghi789"
  ]
}
```

**响应**:
```json
{
  "status": "success",
  "data": {
    "oauth:token:abc123": {
      "user_id": "user001",
      "scope": "read write"
    },
    "oauth:token:def456": {
      "user_id": "user002",
      "scope": "read"
    },
    "oauth:token:ghi789": null
  }
}
```

### 批量设置会话

**端点**: `POST /api/v1/sessions/batch/set`

**请求体**:
```json
{
  "sessions": [
    {
      "key": "oauth:token:abc123",
      "value": {"user_id": "user001"},
      "ttl": 3600
    },
    {
      "key": "oauth:token:def456",
      "value": {"user_id": "user002"},
      "ttl": 3600
    }
  ]
}
```

**响应**:
```json
{
  "status": "success",
  "data": {
    "created": 2,
    "failed": 0
  }
}
```

### 批量删除会话

**端点**: `POST /api/v1/sessions/batch/delete`

**请求体**:
```json
{
  "keys": [
    "oauth:token:abc123",
    "oauth:token:def456"
  ]
}
```

**响应**:
```json
{
  "status": "success",
  "data": {
    "deleted": 2
  }
}
```

## OAuth 2.0 API

### 存储 Access Token

**端点**: `POST /api/v1/oauth/tokens`

**请求体**:
```json
{
  "access_token": "abc123",
  "token_type": "Bearer",
  "expires_in": 3600,
  "refresh_token": "xyz789",
  "scope": "read write",
  "user_id": "user001",
  "client_id": "client001"
}
```

**响应**:
```json
{
  "status": "success",
  "data": {
    "access_token": "abc123",
    "expires_at": "2025-11-19T12:00:00Z"
  }
}
```

### Token 内省（RFC 7662）

**端点**: `POST /api/v1/oauth/introspect`

**请求体**:
```json
{
  "token": "abc123",
  "token_type_hint": "access_token"
}
```

**响应**:
```json
{
  "active": true,
  "scope": "read write",
  "client_id": "client001",
  "username": "user001",
  "token_type": "Bearer",
  "exp": 1700000000,
  "iat": 1699996400,
  "sub": "user001"
}
```

### 撤销 Token（RFC 7009）

**端点**: `POST /api/v1/oauth/revoke`

**请求体**:
```json
{
  "token": "abc123",
  "token_type_hint": "access_token"
}
```

**响应**:
```json
{
  "status": "success"
}
```

## SAML 2.0 API

### 存储 SAML 会话

**端点**: `POST /api/v1/saml/sessions`

**请求体**:
```json
{
  "session_index": "xyz789",
  "name_id": "user@example.com",
  "name_id_format": "urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress",
  "assertion": "PFNhbWwuLi4=",
  "expires_in": 1800,
  "attributes": {
    "email": "user@example.com",
    "display_name": "User Name"
  }
}
```

**响应**:
```json
{
  "status": "success",
  "data": {
    "session_index": "xyz789",
    "expires_at": "2025-11-19T11:30:00Z"
  }
}
```

### 获取 SAML 会话

**端点**: `GET /api/v1/saml/sessions/{session_index}`

**响应**:
```json
{
  "status": "success",
  "data": {
    "session_index": "xyz789",
    "name_id": "user@example.com",
    "assertion": "PFNhbWwuLi4=",
    "attributes": {
      "email": "user@example.com"
    },
    "created_at": "2025-11-19T11:00:00Z",
    "expires_at": "2025-11-19T11:30:00Z"
  }
}
```

## CAS API

### 创建 TGT（Ticket Granting Ticket）

**端点**: `POST /api/v1/cas/tgts`

**请求体**:
```json
{
  "tgt_id": "TGT-1-abc123",
  "user_id": "user001",
  "expires_in": 7200
}
```

**响应**:
```json
{
  "status": "success",
  "data": {
    "tgt_id": "TGT-1-abc123",
    "expires_at": "2025-11-19T13:00:00Z"
  }
}
```

### 创建 ST（Service Ticket）

**端点**: `POST /api/v1/cas/sts`

**请求体**:
```json
{
  "st_id": "ST-1-xyz789",
  "tgt_id": "TGT-1-abc123",
  "service": "https://app.example.com",
  "expires_in": 300
}
```

**响应**:
```json
{
  "status": "success",
  "data": {
    "st_id": "ST-1-xyz789",
    "service": "https://app.example.com",
    "expires_at": "2025-11-19T11:05:00Z"
  }
}
```

### 验证 ST

**端点**: `POST /api/v1/cas/validate`

**请求体**:
```json
{
  "st_id": "ST-1-xyz789",
  "service": "https://app.example.com"
}
```

**响应**:
```json
{
  "status": "success",
  "data": {
    "valid": true,
    "user_id": "user001",
    "attributes": {}
  }
}
```

## 查询和统计 API

### 搜索键

**端点**: `GET /api/v1/keys?pattern=oauth:token:*&cursor=0&count=100`

**查询参数**:
- `pattern`: 匹配模式（支持通配符）
- `cursor`: 游标位置（初始为 0）
- `count`: 返回数量提示

**响应**:
```json
{
  "status": "success",
  "data": {
    "cursor": 17,
    "keys": [
      "oauth:token:abc123",
      "oauth:token:def456"
    ]
  }
}
```

### 获取统计信息

**端点**: `GET /api/v1/stats`

**响应**:
```json
{
  "status": "success",
  "data": {
    "total_keys": 12345,
    "memory_used_bytes": 524288000,
    "total_commands_processed": 1000000,
    "instantaneous_ops_per_sec": 15234,
    "connected_clients": 42,
    "uptime_seconds": 86400
  }
}
```

### 健康检查

**端点**: `GET /api/v1/health`

**响应**:
```json
{
  "status": "healthy",
  "version": "v1.0.0",
  "uptime_seconds": 86400
}
```

**状态码**:
- `200 OK`: 服务健康
- `503 Service Unavailable`: 服务不可用

## 管理 API

### 获取配置

**端点**: `GET /api/v1/admin/config`

**需要权限**: `admin`

**响应**:
```json
{
  "status": "success",
  "data": {
    "max_memory": "2GB",
    "default_ttl": 3600,
    "max_connections": 10000
  }
}
```

### 刷新数据库

**端点**: `POST /api/v1/admin/flushdb`

**需要权限**: `admin`

**响应**:
```json
{
  "status": "success",
  "message": "Database flushed"
}
```

## 错误响应

所有错误响应遵循统一格式：

```json
{
  "status": "error",
  "error": {
    "code": "INVALID_REQUEST",
    "message": "Invalid request parameters",
    "details": {
      "field": "ttl",
      "reason": "must be a positive integer"
    }
  }
}
```

### 错误码

- `INVALID_REQUEST`: 请求参数错误
- `UNAUTHORIZED`: 未授权
- `FORBIDDEN`: 权限不足
- `NOT_FOUND`: 资源不存在
- `RATE_LIMIT_EXCEEDED`: 超过速率限制
- `INTERNAL_ERROR`: 服务器内部错误

## 速率限制

API 响应头包含速率限制信息：

```http
X-RateLimit-Limit: 10000
X-RateLimit-Remaining: 9523
X-RateLimit-Reset: 1700000000
```

## 分页

使用游标分页：

**请求**:
```http
GET /api/v1/keys?cursor=0&count=100
```

**响应**:
```json
{
  "data": {
    "cursor": 17,
    "keys": [...]
  }
}
```

当 `cursor` 返回 `0` 时表示已到末尾。

## 最佳实践

### 使用连接池

HTTP 客户端应使用连接池和 Keep-Alive。

### 启用压缩

支持 gzip 压缩：

```http
Accept-Encoding: gzip
```

### 幂等性

所有 PUT、DELETE 操作都是幂等的，可以安全重试。

### 超时设置

建议设置：
- 连接超时：5 秒
- 读取超时：10 秒
- 写入超时：5 秒

### TLS 配置

生产环境强制使用 TLS 1.3：

- 禁用 TLS 1.0/1.1
- 使用强密码套件
- 启用 OCSP Stapling
- 使用 mTLS 双向认证（推荐）
