# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## 文档索引

TokenginX 文档已拆分为多个专题文件，便于维护和查阅。

### Specs 规范文档

详细的项目规范和技术说明文档位于 `specs/` 目录：

- [01-项目概述](./specs/01-项目概述.md) - 项目定位、核心特性、命令行包
- [02-开发命令](./specs/02-开发命令.md) - 构建、测试、Git 工作流、CI/CD
- [03-核心架构](./specs/03-核心架构.md) - 分片存储、缓存、TTL、持久化、协议支持
- [04-安全特性](./specs/04-安全特性.md) - 传输层安全、认证授权、防重放、数据加密、审计、国密商密双体系
- [05-性能与版本](./specs/05-性能与版本.md) - 性能目标、版本规划
- [06-代码组织](./specs/06-代码组织.md) - 目录结构、关键模块说明
- [07-开发注意事项](./specs/07-开发注意事项.md) - 性能优化、并发安全、测试要求
- [08-开源协作规范](./specs/08-开源协作规范.md) - Code Review、Issue 管理、文档要求

### 用户文档

完整的用户文档位于 `docs/` 目录：

#### API 参考文档
- [核心功能参考](./docs/reference/core-features.md) - 所有命令和操作详解
- [HTTP/REST API 参考](./docs/reference/http-rest-api.md) - RESTful API 完整文档
- [TCP (RESP) 协议参考](./docs/reference/tcp-resp-api.md) - Redis 兼容协议
- [gRPC API 参考](./docs/reference/grpc-api.md) - gRPC 服务定义

#### 快速入门指南
- [ASP.NET Core (C#) 快速指南](./docs/quickstart/aspnet-core.md)
- [Java 快速指南](./docs/quickstart/java.md)
- [PHP 快速指南](./docs/quickstart/php.md)
- [Go 快速指南](./docs/quickstart/go.md)
- [Rust 快速指南](./docs/quickstart/rust.md)

#### 生产环境部署
- [ASP.NET Core (C#) 生产环境指南](./docs/production/aspnet-core.md)
- [Java 生产环境指南](./docs/production/java.md)
- [PHP 生产环境指南](./docs/production/php.md)
- [Go 生产环境指南](./docs/production/go.md)
- [Rust 生产环境指南](./docs/production/rust.md)

#### 协议集成
- [OAuth 2.0/OIDC 集成指南](./docs/protocols/oauth.md)
- [SAML 2.0 集成指南](./docs/protocols/saml.md)
- [CAS 集成指南](./docs/protocols/cas.md)

#### 安全配置
- [TLS/mTLS 配置](./docs/security/tls-mtls.md)
- [国密支持](./docs/security/gm-crypto.md)
- [防重放攻击](./docs/security/anti-replay.md)
- [访问控制 (ACL)](./docs/security/acl.md)

## 快速导航

### 我是开发者
1. 阅读 [项目概述](./specs/01-项目概述.md) 了解项目定位
2. 查看 [开发命令](./specs/02-开发命令.md) 学习构建和测试
3. 熟悉 [核心架构](./specs/03-核心架构.md) 理解系统设计
4. 查阅 [安全特性](./specs/04-安全特性.md) 了解安全机制
5. 遵循 [开源协作规范](./specs/08-开源协作规范.md) 贡献代码

### 我是用户
1. 根据技术栈选择对应的快速指南（`docs/quickstart/`）
2. 查阅 [API 参考文档](./docs/reference/) 了解详细用法
3. 参考生产环境指南（`docs/production/`）进行部署

### 我要集成 SSO
1. 选择您使用的协议（OAuth/SAML/CAS）
2. 阅读对应的协议集成指南（`docs/protocols/`）
3. 配置安全特性（`docs/security/`）

## 文档更新策略

**重要**：后续所有任务执行都会同步更新文档。

当添加新功能或修改现有功能时：
1. 更新相应的 specs 文档
2. 更新 API 参考文档
3. 更新快速指南和生产环境指南
4. 更新协议集成指南（如适用）

## 获取帮助

- [GitHub Issues](https://github.com/your-org/tokenginx/issues)
- [社区讨论](https://github.com/your-org/tokenginx/discussions)
- [完整文档目录](./docs/readme.md)
