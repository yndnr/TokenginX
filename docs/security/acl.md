# 访问控制列表 (ACL)

TokenginX 提供企业级访问控制列表（Access Control List, ACL）功能,实现细粒度的权限管理。

## 概述

ACL 允许您控制:
- **谁**可以访问系统（用户/客户端认证）
- **可以执行什么操作**（命令级权限）
- **可以访问哪些数据**（键级权限）
- **从哪里访问**（IP 白名单/黑名单）
- **什么时候可以访问**（时间窗口控制）

## 核心概念

### 主体（Subject）

访问系统的实体:
- **用户**:人类用户
- **客户端**:应用程序/服务
- **IP 地址**:来源 IP

### 角色（Role）

权限的集合:
- **预定义角色**:`admin`, `operator`, `readonly`
- **自定义角色**:根据需求定义

### 权限（Permission）

具体的操作权限:
- **命令权限**:`GET`, `SET`, `DELETE`, `EXISTS` 等
- **键权限**:允许访问的键前缀
- **操作权限**:`read`, `write`, `delete`

### 策略（Policy）

将主体、角色、权限关联起来的规则。

## 配置 ACL

### 启用 ACL

```yaml
# config.yaml
security:
  acl:
    # 启用 ACL
    enabled: true

    # 默认拒绝策略（推荐）
    # true: 默认拒绝所有,需明确授权
    # false: 默认允许所有,需明确拒绝
    default_deny: true

    # ACL 规则文件
    rules_file: "/etc/tokenginx/acl.yaml"

    # 审计被拒绝的请求
    audit_denied: true
```

### 定义角色和权限

```yaml
# /etc/tokenginx/acl.yaml

# 角色定义
roles:
  # 管理员角色
  admin:
    description: "Full administrative access"
    commands:
      - "*"  # 所有命令
    keys:
      - "*"  # 所有键
    operations:
      - "read"
      - "write"
      - "delete"

  # 操作员角色
  operator:
    description: "Operational access for session management"
    commands:
      - "GET"
      - "SET"
      - "DELETE"
      - "EXISTS"
      - "TTL"
    keys:
      - "oauth:*"
      - "saml:*"
      - "cas:*"
    operations:
      - "read"
      - "write"
      - "delete"

  # 只读角色
  readonly:
    description: "Read-only access"
    commands:
      - "GET"
      - "EXISTS"
      - "TTL"
    keys:
      - "*"
    operations:
      - "read"

  # OAuth 管理员
  oauth_admin:
    description: "OAuth session management"
    commands:
      - "GET"
      - "SET"
      - "DELETE"
      - "EXISTS"
    keys:
      - "oauth:*"  # 仅 OAuth 相关键
    operations:
      - "read"
      - "write"
      - "delete"

  # SAML 只读
  saml_readonly:
    description: "SAML read-only access"
    commands:
      - "GET"
      - "EXISTS"
    keys:
      - "saml:*"
    operations:
      - "read"

# 用户定义
users:
  # 管理员用户
  - username: "admin"
    password_hash: "$2a$10$..." # bcrypt hash
    roles:
      - "admin"
    ip_whitelist:
      - "10.0.0.0/8"
      - "192.168.0.0/16"

  # OAuth 服务账号
  - username: "oauth_service"
    api_key: "oauth-api-key-12345"
    roles:
      - "oauth_admin"
    ip_whitelist:
      - "10.1.0.0/24"

  # 只读审计账号
  - username: "auditor"
    certificate_fingerprint: "SHA256:abcdef..."
    roles:
      - "readonly"

# 客户端定义
clients:
  # OAuth 服务器
  - client_id: "oauth_server"
    secret_key: "secret-key-here"
    roles:
      - "oauth_admin"
    ip_whitelist:
      - "10.1.0.10"
      - "10.1.0.11"

  # SAML IdP
  - client_id: "saml_idp"
    secret_key: "another-secret-key"
    roles:
      - "operator"
    ip_whitelist:
      - "10.2.0.0/24"

# IP 黑名单（全局）
ip_blacklist:
  - "192.168.1.100"  # 已知恶意 IP
  - "203.0.113.0/24"  # 黑名单 IP 段

# 时间窗口控制（可选）
time_windows:
  # 工作时间访问
  - name: "business_hours"
    days: ["Monday", "Tuesday", "Wednesday", "Thursday", "Friday"]
    start_time: "08:00"
    end_time: "18:00"
    timezone: "Asia/Shanghai"
    applies_to:
      users: ["operator1", "operator2"]
```

## 认证方式

TokenginX 支持多种认证方式:

### 1. 用户名/密码

```yaml
users:
  - username: "user1"
    password_hash: "$2a$10$..."  # bcrypt hash
    roles:
      - "operator"
```

**生成密码哈希**:

```bash
# 使用 bcrypt
$ htpasswd -bnBC 10 "" your-password | tr -d ':\n'
$2a$10$...

# 或使用 Go
package main
import (
    "fmt"
    "golang.org/x/crypto/bcrypt"
)
func main() {
    hash, _ := bcrypt.GenerateFromPassword([]byte("your-password"), bcrypt.DefaultCost)
    fmt.Println(string(hash))
}
```

**客户端使用**:

```go
// HTTP Basic Auth
req.SetBasicAuth("user1", "your-password")

// 或自定义头
req.Header.Set("X-Username", "user1")
req.Header.Set("X-Password", "your-password")
```

### 2. API Key

```yaml
users:
  - username: "api_user"
    api_key: "api-key-1234567890"
    roles:
      - "oauth_admin"
```

**客户端使用**:

```go
req.Header.Set("X-API-Key", "api-key-1234567890")
```

### 3. 证书认证（推荐）

```yaml
users:
  - username: "cert_user"
    certificate_fingerprint: "SHA256:1234567890abcdef..."
    roles:
      - "admin"
```

**生成证书指纹**:

```bash
# 从证书文件获取指纹
openssl x509 -in client_cert.pem -noout -fingerprint -sha256
SHA256 Fingerprint=12:34:56:78:90:AB:CD:EF:...

# 格式化为 SHA256:...
echo "SHA256:1234567890abcdef..."
```

**客户端使用**:

```go
// 配置 mTLS
cert, _ := tls.LoadX509KeyPair("client_cert.pem", "client_key.pem")
tlsConfig := &tls.Config{
    Certificates: []tls.Certificate{cert},
}
```

### 4. JWT Token

```yaml
users:
  - username: "jwt_user"
    jwt_issuer: "https://auth.example.com"
    jwt_audience: "tokenginx"
    roles:
      - "operator"
```

**客户端使用**:

```go
token := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
req.Header.Set("Authorization", "Bearer "+token)
```

## 权限检查

### 命令级权限

控制用户可以执行哪些命令:

```yaml
roles:
  limited_user:
    commands:
      - "GET"    # 允许 GET
      - "EXISTS" # 允许 EXISTS
      # 不允许 SET, DELETE 等写操作
```

### 键级权限

控制用户可以访问哪些键:

```yaml
roles:
  oauth_user:
    commands:
      - "*"
    keys:
      - "oauth:token:*"       # 允许访问所有 OAuth token
      - "oauth:code:*"        # 允许访问所有 OAuth code
      # 不允许访问 saml:*, cas:* 等其他键
```

**键匹配规则**:
- `*`: 匹配任意字符（通配符）
- `oauth:*`: 匹配以 `oauth:` 开头的所有键
- `oauth:token:*`: 匹配以 `oauth:token:` 开头的所有键
- `oauth:token:user123`: 精确匹配

### 操作级权限

控制用户可以执行的操作类型:

```yaml
roles:
  readonly_user:
    operations:
      - "read"  # 只读
      # 不允许 write, delete
```

**操作类型映射**:
- `GET`, `EXISTS`, `TTL` → `read`
- `SET`, `SETEX` → `write`
- `DELETE`, `DEL` → `delete`

## IP 白名单/黑名单

### 用户级 IP 白名单

```yaml
users:
  - username: "restricted_user"
    roles:
      - "operator"
    ip_whitelist:
      - "10.1.0.0/24"      # 允许整个子网
      - "192.168.1.100"    # 允许单个 IP
```

### 全局 IP 黑名单

```yaml
ip_blacklist:
  - "203.0.113.50"       # 阻止单个 IP
  - "198.51.100.0/24"    # 阻止整个子网
```

### IP 检查优先级

```
1. 检查全局黑名单 -> 如果匹配,拒绝
2. 检查用户白名单 -> 如果不匹配,拒绝
3. 允许访问
```

## 时间窗口控制

限制用户访问时间:

```yaml
time_windows:
  # 工作时间
  - name: "business_hours"
    days: ["Monday", "Tuesday", "Wednesday", "Thursday", "Friday"]
    start_time: "08:00"
    end_time: "18:00"
    timezone: "Asia/Shanghai"
    applies_to:
      users: ["operator1"]
      roles: ["operator"]

  # 维护窗口（禁止访问）
  - name: "maintenance"
    days: ["Sunday"]
    start_time: "02:00"
    end_time: "04:00"
    timezone: "Asia/Shanghai"
    action: "deny"  # deny 或 allow
```

## 动态 ACL 管理

### 通过 API 管理 ACL

```bash
# 添加用户
curl -X POST https://tokenginx.example.com/api/v1/acl/users \
  -H "Authorization: Bearer admin-token" \
  -H "Content-Type: application/json" \
  -d '{
    "username": "newuser",
    "password": "password123",
    "roles": ["operator"]
  }'

# 修改用户角色
curl -X PUT https://tokenginx.example.com/api/v1/acl/users/newuser/roles \
  -H "Authorization: Bearer admin-token" \
  -H "Content-Type: application/json" \
  -d '{
    "roles": ["operator", "oauth_admin"]
  }'

# 删除用户
curl -X DELETE https://tokenginx.example.com/api/v1/acl/users/newuser \
  -H "Authorization: Bearer admin-token"

# 添加 IP 到黑名单
curl -X POST https://tokenginx.example.com/api/v1/acl/ip-blacklist \
  -H "Authorization: Bearer admin-token" \
  -H "Content-Type: application/json" \
  -d '{
    "ip": "192.168.1.100"
  }'
```

### 热更新 ACL 规则

```bash
# 重新加载 ACL 配置（无需重启）
curl -X POST https://tokenginx.example.com/api/v1/acl/reload \
  -H "Authorization: Bearer admin-token"

# 或使用客户端工具
./tokenginx-client acl reload
```

## 审计日志

ACL 决策会记录在审计日志中:

```json
{
  "timestamp": "2025-11-19T12:34:56.789Z",
  "event_type": "acl_check",
  "client_ip": "192.168.1.100",
  "username": "oauth_service",
  "command": "SET",
  "key": "oauth:token:abc123",
  "result": "allowed",
  "reason": "user has oauth_admin role with write permission",
  "latency_ms": 0.15
}
```

**拒绝访问示例**:

```json
{
  "timestamp": "2025-11-19T12:35:00.123Z",
  "event_type": "acl_check",
  "client_ip": "192.168.1.200",
  "username": "readonly_user",
  "command": "DELETE",
  "key": "oauth:token:xyz789",
  "result": "denied",
  "reason": "user readonly_user does not have delete permission",
  "latency_ms": 0.08
}
```

## 最佳实践

### 1. 使用默认拒绝策略

```yaml
security:
  acl:
    default_deny: true  # 推荐
```

### 2. 最小权限原则

只授予完成任务所需的最小权限:

```yaml
# ❌ 不好：授予过多权限
roles:
  app_user:
    commands:
      - "*"
    keys:
      - "*"

# ✅ 好：最小权限
roles:
  app_user:
    commands:
      - "GET"
      - "SET"
    keys:
      - "oauth:token:*"
```

### 3. 使用证书认证

证书认证比密码更安全:

```yaml
# ✅ 推荐：证书认证
users:
  - username: "prod_service"
    certificate_fingerprint: "SHA256:..."
    roles:
      - "operator"

# ⚠️ 不推荐：密码认证（仅适用于人类用户）
users:
  - username: "admin"
    password_hash: "$2a$10$..."
    roles:
      - "admin"
```

### 4. 使用 IP 白名单

限制访问来源:

```yaml
users:
  - username: "api_user"
    api_key: "..."
    roles:
      - "oauth_admin"
    # 仅允许从应用服务器访问
    ip_whitelist:
      - "10.1.0.0/24"
```

### 5. 定期审计

定期检查 ACL 配置和审计日志:

```bash
# 查看当前用户列表
./tokenginx-client acl users list

# 查看被拒绝的请求
./tokenginx-client acl audit --result denied --last 24h

# 查看特定用户的活动
./tokenginx-client acl audit --user oauth_service --last 7d
```

### 6. 分离职责

不同服务使用不同账号:

```yaml
# OAuth 服务
clients:
  - client_id: "oauth_server"
    roles:
      - "oauth_admin"

# SAML 服务
clients:
  - client_id: "saml_idp"
    roles:
      - "operator"  # 不同角色

# 监控服务
clients:
  - client_id: "monitoring"
    roles:
      - "readonly"  # 只读权限
```

## 性能考虑

### ACL 缓存

TokenginX 会缓存 ACL 决策结果,减少检查开销:

```yaml
security:
  acl:
    enabled: true
    # ACL 决策缓存
    cache:
      enabled: true
      ttl_seconds: 60
      max_entries: 10000
```

### 权限检查性能

典型性能指标:
- ACL 检查延迟: < 0.1ms (缓存命中)
- ACL 检查延迟: < 1ms (缓存未命中)
- 对总体 QPS 影响: < 5%

## 故障排查

### 调试 ACL

启用 ACL 调试日志:

```yaml
logging:
  level: "debug"
  modules:
    - "acl"
```

### 常见问题

#### 问题 1: 用户无法访问

**症状**: `401 Unauthorized` 或 `403 Forbidden`

**排查步骤**:

1. 检查用户是否存在:
   ```bash
   ./tokenginx-client acl users get username
   ```

2. 检查用户角色:
   ```bash
   ./tokenginx-client acl users get username --show-roles
   ```

3. 检查角色权限:
   ```bash
   ./tokenginx-client acl roles get role_name
   ```

4. 检查审计日志:
   ```bash
   ./tokenginx-client acl audit --user username --last 1h
   ```

#### 问题 2: IP 白名单问题

**症状**: 从某些 IP 无法访问

**排查**:

```bash
# 检查用户 IP 白名单
./tokenginx-client acl users get username --show-ip-whitelist

# 测试 IP 是否在白名单中
./tokenginx-client acl test-ip username 192.168.1.100
```

#### 问题 3: ACL 配置未生效

**解决**: 重新加载 ACL 配置

```bash
./tokenginx-client acl reload
```

## 示例场景

### 场景 1: OAuth 服务器部署

```yaml
# OAuth 服务器只能访问 OAuth 相关键
clients:
  - client_id: "oauth_server"
    secret_key: "oauth-secret-key"
    roles:
      - "oauth_admin"
    ip_whitelist:
      - "10.1.0.10"

roles:
  oauth_admin:
    commands:
      - "GET"
      - "SET"
      - "DELETE"
      - "EXISTS"
      - "TTL"
    keys:
      - "oauth:*"
    operations:
      - "read"
      - "write"
      - "delete"
```

### 场景 2: 多租户环境

```yaml
# 租户 A
clients:
  - client_id: "tenant_a"
    secret_key: "..."
    roles:
      - "tenant_a_access"

roles:
  tenant_a_access:
    commands:
      - "*"
    keys:
      - "tenant_a:*"  # 仅访问自己的数据
    operations:
      - "read"
      - "write"
      - "delete"

# 租户 B
clients:
  - client_id: "tenant_b"
    secret_key: "..."
    roles:
      - "tenant_b_access"

roles:
  tenant_b_access:
    commands:
      - "*"
    keys:
      - "tenant_b:*"  # 仅访问自己的数据
    operations:
      - "read"
      - "write"
      - "delete"
```

### 场景 3: 开发/测试/生产环境

```yaml
# 开发环境：宽松策略
users:
  - username: "dev_user"
    password_hash: "..."
    roles:
      - "admin"  # 开发环境给予完全权限

# 生产环境：严格策略
users:
  - username: "prod_service"
    certificate_fingerprint: "SHA256:..."
    roles:
      - "oauth_admin"
    ip_whitelist:
      - "10.100.0.0/24"  # 仅生产网络
    time_windows:
      - name: "no_maintenance"
        # 在维护窗口禁止访问
```

## 参考资料

- [OAuth 2.0 Security Best Practices](https://datatracker.ietf.org/doc/html/draft-ietf-oauth-security-topics)
- [NIST SP 800-63B - Digital Identity Guidelines](https://pages.nist.gov/800-63-3/sp800-63b.html)
- [OWASP Access Control Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Access_Control_Cheat_Sheet.html)

---

**下一步**: 配置 [TLS/mTLS](./tls-mtls.md) 保护传输层安全。
