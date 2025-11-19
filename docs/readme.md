# TokenginX 文档

欢迎使用 TokenginX 文档！本文档将帮助您快速了解和使用 TokenginX。

## 文档目录

### 快速开始

根据您使用的技术栈选择相应的快速入门指南：

#### 主流 Web 开发语言

- [Python 快速指南](./quickstart/python.md) - Flask, Django, FastAPI 集成
- [Node.js/JavaScript 快速指南](./quickstart/nodejs.md) - Express, NestJS 集成
- [Ruby 快速指南](./quickstart/ruby.md) - Rails, Sinatra 集成
- [Java 快速指南](./quickstart/java.md) - Spring Boot 集成
- [PHP 快速指南](./quickstart/php.md) - Laravel, Symfony 集成
- [Go 快速指南](./quickstart/go.md) - 原生 Go 客户端
- [C# / ASP.NET Core 快速指南](./quickstart/aspnet-core.md)
- [Rust 快速指南](./quickstart/rust.md)

### API 参考文档

- [核心功能参考](./reference/core-features.md)
- [TCP (RESP) 协议参考](./reference/tcp-resp-api.md)
- [gRPC API 参考](./reference/grpc-api.md)
- [HTTP/REST API 参考](./reference/http-rest-api.md)
- [配置参考](./reference/configuration.md)

### 生产环境部署

- [Python 生产环境指南](./production/python.md)
- [Node.js 生产环境指南](./production/nodejs.md)
- [Ruby 生产环境指南](./production/ruby.md)
- [Java 生产环境指南](./production/java.md)
- [PHP 生产环境指南](./production/php.md)
- [Go 生产环境指南](./production/go.md)
- [C# / ASP.NET Core 生产环境指南](./production/aspnet-core.md)
- [Rust 生产环境指南](./production/rust.md)

### 协议支持

- [OAuth 2.0/OIDC 集成指南](./protocols/oauth.md)
- [SAML 2.0 集成指南](./protocols/saml.md)
- [CAS 集成指南](./protocols/cas.md)

### 安全性

- [TLS/mTLS 配置](./security/tls-mtls.md)
- [国密支持](./security/gm-crypto.md)
- [防重放攻击](./security/anti-replay.md)
- [访问控制 (ACL)](./security/acl.md)

### 容器化部署

- [GitHub 仓库初始化](./deployment/github-setup.md) - 连接到 GitHub 的完整指南
- [Docker 部署指南](./deployment/docker.md) - Docker 和 Docker Compose 完整指南
- [Podman 部署指南](./deployment/podman.md) - Rootless 容器和 Systemd 集成
- [Kubernetes 部署指南](./deployment/kubernetes.md) - 生产级 K8s 编排

### 项目管理和任务

- **[开始开发](./tasks/getting-started.md)** - 新手必读：开发流程、任务管理、代码规范
- [项目路线图](./tasks/roadmap.md) - 完整的版本规划和里程碑（v0.1.0 - v3.0.0）
- [GitHub Projects 配置](./tasks/github-projects.md) - 任务管理、看板、自动化工作流
- [v0.1.0 开发任务清单](./tasks/v0.1.0-dev-tasks.md) - MVP 版本开发任务详细清单
- [v0.1.0 测试任务清单](./tasks/v0.1.0-test-tasks.md) - MVP 版本测试任务和质量标准

## 项目概述

TokenginX 是一个专为单点登录（SSO）优化的高性能会话存储系统，提供以下核心特性：

- **高性能**：100K+ QPS，P99 < 1ms
- **多协议**：原生支持 OAuth 2.0/OIDC、SAML 2.0、CAS
- **多接口**：TCP (RESP)、gRPC、HTTP/REST 三种通信方式
- **安全优先**：TLS 1.3、mTLS、国密支持、防重放攻击
- **智能存储**：内存优先 + 冷数据自动溢出

## 获取帮助

- [GitHub Issues](https://github.com/your-org/tokenginx/issues)
- [社区讨论](https://github.com/your-org/tokenginx/discussions)
