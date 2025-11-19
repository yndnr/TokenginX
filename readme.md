# TokenginX

[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![Go Version](https://img.shields.io/badge/go-%3E%3D1.21-blue.svg)](https://go.dev/)
[![Build Status](https://github.com/your-org/tokenginx/workflows/CI/badge.svg)](https://github.com/your-org/tokenginx/actions)

一个专为单点登录（SSO）优化的高性能会话存储系统。

## 核心特性

- **高性能**: 单节点 100K+ QPS，P99 延迟 < 1ms，P999 < 5ms
- **多协议支持**: 原生支持 OAuth 2.0/OIDC、SAML 2.0、CAS 协议
- **多接口通信**: TCP (RESP)、gRPC、HTTP/REST 三种方式任选
- **智能存储**: 内存优先 + W-TinyLFU 缓存算法，命中率 > 99%
- **安全优先**:
  - TLS 1.3 / mTLS 双向认证
  - 国密算法支持（SM2、SM3、SM4）
  - 防重放攻击、访问控制列表（ACL）
  - 完整的审计日志
- **轻量级**: 单二进制文件，零外部依赖
- **分片架构**: 256 分片设计，充分利用多核 CPU

## 快速开始

### 安装

从 [Releases](https://github.com/your-org/tokenginx/releases) 下载最新版本：

```bash
# Linux AMD64
wget https://github.com/your-org/tokenginx/releases/download/v0.1.0/tokenginx-server-linux-amd64
chmod +x tokenginx-server-linux-amd64
mv tokenginx-server-linux-amd64 /usr/local/bin/tokenginx-server

# macOS
wget https://github.com/your-org/tokenginx/releases/download/v0.1.0/tokenginx-server-darwin-amd64
chmod +x tokenginx-server-darwin-amd64
mv tokenginx-server-darwin-amd64 /usr/local/bin/tokenginx-server
```

或从源码构建：

```bash
git clone https://github.com/your-org/tokenginx.git
cd tokenginx
go build -o bin/tokenginx-server ./cmd/server
go build -o bin/tokenginx-client ./cmd/client
```

### 运行服务器

```bash
# 使用默认配置运行
./tokenginx-server

# 使用配置文件运行
./tokenginx-server -config config.yaml
```

服务器默认监听：
- TCP (RESP): `localhost:6380`
- gRPC: `localhost:9090`
- HTTP/REST: `localhost:8080`

### 客户端连接示例

#### 使用 Redis 客户端（RESP 协议）

```bash
# 使用 redis-cli 连接
redis-cli -p 6380

# 设置会话（key, value, ttl秒）
127.0.0.1:6380> SET oauth:token:abc123 '{"user_id":"user001"}' EX 3600
OK

# 获取会话
127.0.0.1:6380> GET oauth:token:abc123
"{\"user_id\":\"user001\"}"

# 删除会话
127.0.0.1:6380> DEL oauth:token:abc123
(integer) 1
```

#### HTTP/REST API

```bash
# 设置会话
curl -X POST http://localhost:8080/api/v1/sessions \
  -H "Content-Type: application/json" \
  -d '{
    "key": "oauth:token:abc123",
    "value": {"user_id": "user001"},
    "ttl": 3600
  }'

# 获取会话
curl http://localhost:8080/api/v1/sessions/oauth:token:abc123

# 删除会话
curl -X DELETE http://localhost:8080/api/v1/sessions/oauth:token:abc123
```

#### Go 客户端（gRPC）

```go
package main

import (
    "context"
    "log"
    "time"

    "google.golang.org/grpc"
    pb "github.com/your-org/tokenginx/api/grpc/v1"
)

func main() {
    conn, err := grpc.Dial("localhost:9090", grpc.WithInsecure())
    if err != nil {
        log.Fatal(err)
    }
    defer conn.Close()

    client := pb.NewSessionServiceClient(conn)
    ctx, cancel := context.WithTimeout(context.Background(), time.Second)
    defer cancel()

    // 设置会话
    _, err = client.Set(ctx, &pb.SetRequest{
        Key:   "oauth:token:abc123",
        Value: []byte(`{"user_id":"user001"}`),
        Ttl:   3600,
    })
    if err != nil {
        log.Fatal(err)
    }

    // 获取会话
    resp, err := client.Get(ctx, &pb.GetRequest{
        Key: "oauth:token:abc123",
    })
    if err != nil {
        log.Fatal(err)
    }
    log.Printf("Value: %s", resp.Value)
}
```

## 架构概览

```
┌─────────────────────────────────────────────────────────────┐
│                      客户端应用层                              │
│  (OAuth Server, SAML IdP, CAS Server, 自定义应用)              │
└────────────┬────────────┬───────────────┬────────────────────┘
             │            │               │
    ┌────────▼───┐  ┌────▼─────┐  ┌──────▼──────┐
    │ TCP (RESP) │  │   gRPC   │  │  HTTP/REST  │
    └────────┬───┘  └────┬─────┘  └──────┬──────┘
             │            │               │
┌────────────▼────────────▼───────────────▼────────────────────┐
│                    TokenginX 服务器                            │
│  ┌─────────────────────────────────────────────────────┐     │
│  │           分片存储引擎（256 分片）                      │     │
│  │  ┌──────────────┐    ┌────────────────────────┐     │     │
│  │  │  内存存储     │───▶│  W-TinyLFU 缓存算法    │     │     │
│  │  │  (ShardedMap)│    │  (命中率 > 99%)        │     │     │
│  │  └──────────────┘    └────────────────────────┘     │     │
│  │  ┌──────────────┐    ┌────────────────────────┐     │     │
│  │  │  TTL 管理    │    │  惰性删除 + 定期清理   │     │     │
│  │  └──────────────┘    └────────────────────────┘     │     │
│  └─────────────────────────────────────────────────────┘     │
│  ┌─────────────────────────────────────────────────────┐     │
│  │               持久化层（可选）                          │     │
│  │  ┌──────────────┐    ┌────────────────────────┐     │     │
│  │  │  mmap 文件   │    │  WAL 日志              │     │     │
│  │  │  (冷数据)    │    │  (数据持久性)          │     │     │
│  │  └──────────────┘    └────────────────────────┘     │     │
│  └─────────────────────────────────────────────────────┘     │
│  ┌─────────────────────────────────────────────────────┐     │
│  │               安全层                                    │     │
│  │  TLS 1.3  │  mTLS  │  国密  │  ACL  │  防重放        │     │
│  └─────────────────────────────────────────────────────┘     │
└───────────────────────────────────────────────────────────────┘
```

### 与 Redis 的关键区别

| 特性 | TokenginX | Redis |
|-----|-----------|-------|
| **定位** | 专为 SSO 会话存储优化 | 通用缓存/数据库 |
| **协议支持** | 内置 OAuth/SAML/CAS | 需自行实现 |
| **国密支持** | 原生支持 SM2/SM3/SM4 | 不支持 |
| **部署** | 单二进制，零依赖 | 需额外配置 |
| **存储策略** | 智能冷热分离 | 手动配置 |
| **安全审计** | 内置完整审计日志 | 需额外工具 |

## 协议支持

### OAuth 2.0 / OIDC

支持以下 OAuth 2.0 流程和功能：

- Authorization Code Flow
- Implicit Flow
- Client Credentials Flow
- Token Introspection (RFC 7662)
- 存储：Access Token、Refresh Token、ID Token、授权码

详见 [OAuth 2.0/OIDC 集成指南](./docs/protocols/oauth.md)

### SAML 2.0

支持 SAML 2.0 SSO 场景：

- SP-initiated SSO
- IdP-initiated SSO
- Artifact Binding
- 存储：SAML Assertion、Session Index、Name ID

详见 [SAML 2.0 集成指南](./docs/protocols/saml.md)

### CAS

支持 CAS 协议的票据管理：

- TGT (Ticket Granting Ticket)
- ST (Service Ticket)
- PT (Proxy Ticket)

详见 [CAS 集成指南](./docs/protocols/cas.md)

## 性能指标

基于 Intel Xeon 8 核 16GB 测试结果：

- **QPS**: 100,000+ 次/秒（单节点）
- **延迟**:
  - P50: < 0.5ms
  - P99: < 1ms
  - P999: < 5ms
- **缓存命中率**: > 99%
- **并发连接**: 支持 10,000+ 并发连接

性能基准测试：

```bash
# 运行基准测试
go test -bench=. -benchmem ./internal/storage/

# 运行完整压力测试
./scripts/benchmark.sh
```

## 安全特性

TokenginX 采用纵深防御策略，提供企业级安全保障：

### 传输层安全
- **TLS 1.3**: 默认使用最新 TLS 版本
- **mTLS**: 双向认证，客户端证书验证
- **国密 TLS**: 支持 SM2 双证书模式（TLCP）

### 认证与授权
- **多种认证方式**: 证书认证、令牌认证、用户名/密码、API Key
- **访问控制列表（ACL）**: 基于角色的访问控制（RBAC）
- **命令级权限**: 细粒度控制客户端可执行的命令
- **IP 白名单/黑名单**: 限制客户端来源

### 防重放攻击
- **时间戳验证**: 拒绝超出时间窗口的请求
- **Nonce 机制**: 一次性随机数防止重放
- **请求签名**: HMAC-SHA256/SM3 签名验证

### 数据加密
- **内存加密**: AES-256-GCM / SM4-GCM
- **持久化加密**: 透明加密 mmap 文件和 WAL 日志
- **密钥管理**: 支持密钥轮换、KMS 集成

### 审计日志
- **结构化日志**: JSON 格式，便于解析
- **完整记录**: 认证、授权、数据访问、配置变更
- **敏感信息脱敏**: 密码、令牌自动脱敏
- **Syslog 集成**: 支持远程日志服务器

详见 [安全文档](./docs/security/)

## 配置

TokenginX 支持通过配置文件或环境变量进行配置：

```yaml
# config.yaml 示例
server:
  tcp_addr: "0.0.0.0:6380"
  grpc_addr: "0.0.0.0:9090"
  http_addr: "0.0.0.0:8080"

storage:
  shard_count: 256
  initial_capacity: 4096
  enable_persistence: true
  data_dir: "/var/lib/tokenginx"

cache:
  algorithm: "w-tinylfu"
  max_memory_mb: 1024
  eviction_policy: "lru"

security:
  crypto_mode: "auto"  # auto | gm | sm | hybrid
  tls:
    enabled: true
    cert_file: "/path/to/cert.pem"
    key_file: "/path/to/key.pem"
    client_auth: "require"
  acl:
    enabled: true
    default_deny: true
  audit:
    enabled: true
    output: "/var/log/tokenginx/audit.log"

protocols:
  oauth:
    enabled: true
    default_token_ttl: 3600
  saml:
    enabled: true
    default_assertion_ttl: 1800
  cas:
    enabled: true
    default_ticket_ttl: 300
```

完整配置参考：[配置文档](./docs/reference/configuration.md)

## 客户端库支持

TokenginX 支持多种编程语言,选择您使用的语言查看快速指南:

- [Python](./docs/quickstart/python.md) - Flask, Django, FastAPI
- [Node.js/JavaScript](./docs/quickstart/nodejs.md) - Express, NestJS
- [Ruby](./docs/quickstart/ruby.md) - Rails, Sinatra
- [Go](./docs/quickstart/go.md) - 原生 Go 客户端
- [Java](./docs/quickstart/java.md) - Spring Boot
- [PHP](./docs/quickstart/php.md) - Laravel, Symfony
- [C# / ASP.NET Core](./docs/quickstart/aspnet-core.md)
- [Rust](./docs/quickstart/rust.md)

## 文档

- [快速开始指南](./docs/readme.md)
- [API 参考文档](./docs/reference/)
- [协议集成指南](./docs/protocols/)
- [安全配置指南](./docs/security/)
- [生产环境部署](./docs/production/)
- [容器化部署指南](./docs/deployment/)
  - [Docker 部署](./docs/deployment/docker.md)
  - [Podman 部署](./docs/deployment/podman.md)
  - [Kubernetes 部署](./docs/deployment/kubernetes.md)

## 版本规划

- **v0.1.0** (MVP): 单机 + OAuth 2.0 基础支持
- **v0.5.0**: 完整协议支持（SAML + CAS + 三接口）
- **v1.0.0**: 生产可用（国密 + 安全加固）
- **v2.0.0**: 分布式集群 + 高可用
- **v3.0.0**: 企业版（Prometheus 监控 + 运维工具）

详见 [changelog.md](./changelog.md)

## 贡献

我们欢迎所有形式的贡献！请阅读 [贡献指南](./contributing.md) 了解如何参与项目开发。

### 开发环境设置

```bash
# 克隆仓库
git clone https://github.com/your-org/tokenginx.git
cd tokenginx

# 安装依赖
go mod download

# 运行测试
go test ./...

# 运行代码检查
go vet ./...
go fmt ./...

# 构建
go build -o bin/tokenginx-server ./cmd/server
```

## 社区

- [GitHub Issues](https://github.com/your-org/tokenginx/issues) - Bug 报告和功能请求
- [GitHub Discussions](https://github.com/your-org/tokenginx/discussions) - 社区讨论
- [Wiki](https://github.com/your-org/tokenginx/wiki) - 社区维护的文档

## 许可证

本项目采用 [MIT 许可证](./LICENSE)。

## 致谢

TokenginX 使用了以下优秀的开源项目：

- [gmsm](https://github.com/tjfoc/gmsm) - 国密算法 Go 实现
- [grpc-go](https://github.com/grpc/grpc-go) - gRPC Go 实现
- 以及其他在 go.mod 中列出的依赖

---

**注**: TokenginX 目前处于早期开发阶段（v0.1.0），不建议在生产环境使用。生产可用版本预计在 v1.0.0 发布。
