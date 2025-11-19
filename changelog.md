# Changelog

本文件记录 TokenginX 项目的所有重要变更。

格式基于 [Keep a Changelog](https://keepachangelog.com/zh-CN/1.0.0/)，
版本号遵循 [Semantic Versioning](https://semver.org/lang/zh-CN/)。

## [Unreleased]

### 计划中
- OAuth 2.0/OIDC 完整实现
- SAML 2.0 基础支持
- CAS 协议基础支持
- gRPC 接口实现
- HTTP/REST 接口实现
- 国密算法集成（SM2/SM3/SM4）
- TLS 1.3 和 mTLS 支持
- 访问控制列表（ACL）
- 防重放攻击机制
- 审计日志系统

## [0.1.0] - TBD (MVP)

### 概述
第一个最小可用版本（MVP），提供单机部署的基础会话存储功能和 OAuth 2.0 基础支持。

### 新增 (Added)
- **核心存储引擎**
  - 256 分片的高性能哈希表
  - W-TinyLFU 缓存淘汰算法
  - TTL 自动过期管理（惰性删除 + 定期清理）
  - 基本的内存存储实现

- **传输层**
  - TCP (RESP 协议) 支持，兼容 Redis 客户端
  - 基本命令：GET、SET、DEL、EXISTS、TTL

- **OAuth 2.0 基础支持**
  - Access Token 存储和验证
  - Refresh Token 管理
  - Authorization Code 存储
  - 基础的 Token Introspection

- **配置系统**
  - YAML 配置文件支持
  - 环境变量配置支持
  - 基本的服务器配置（地址、端口、超时等）

- **命令行工具**
  - `tokenginx-server`: 服务器端程序
  - `tokenginx-client`: 基础客户端工具

- **文档**
  - 项目 readme.md
  - 贡献指南 contributing.md
  - API 参考文档
  - 快速入门指南

### 技术指标
- QPS: > 50,000 次/秒（单节点）
- P99 延迟: < 2ms
- 缓存命中率: > 95%

### 已知限制
- 仅支持内存存储，无持久化
- 单机部署，无集群支持
- OAuth 2.0 功能不完整
- 缺少 SAML 和 CAS 支持
- 缺少 gRPC 和 HTTP 接口
- 无安全加固（TLS、ACL 等）

---

## [0.5.0] - TBD

### 概述
完整协议支持版本，添加 SAML 2.0、CAS 协议，以及 gRPC 和 HTTP/REST 接口。

### 计划新增 (Planned Added)
- **协议支持**
  - 完整的 OAuth 2.0/OIDC 实现
    - Implicit Flow
    - Client Credentials Flow
    - PKCE 支持
    - ID Token 管理
  - SAML 2.0 完整支持
    - SP-initiated SSO
    - IdP-initiated SSO
    - Artifact Binding
    - SAML Assertion 存储和验证
  - CAS 协议完整支持
    - TGT (Ticket Granting Ticket)
    - ST (Service Ticket)
    - PT (Proxy Ticket)

- **传输层扩展**
  - gRPC 接口实现
  - HTTP/REST API 实现
  - 多接口统一管理

- **持久化**
  - mmap 内存映射文件支持
  - 冷热数据分离
  - 基础的 WAL 日志

- **监控和运维**
  - 基础健康检查接口
  - 统计信息查询
  - 日志系统完善

### 技术指标提升
- QPS: > 80,000 次/秒
- P99 延迟: < 1.5ms
- 支持 3 种传输协议

---

## [1.0.0] - TBD

### 概述
生产可用版本，完整的安全特性和稳定性保障。

### 计划新增 (Planned Added)
- **安全特性**
  - TLS 1.3 支持
  - mTLS 双向认证
  - 国密算法支持（SM2、SM3、SM4）
  - 国密 TLS（TLCP 双证书模式）
  - 访问控制列表（ACL）
    - 基于角色的访问控制（RBAC）
    - 命令级权限控制
    - 键级权限控制
    - IP 白名单/黑名单
  - 防重放攻击
    - 时间戳验证
    - Nonce 机制
    - 请求签名（HMAC-SHA256/SM3）
  - 数据加密
    - 内存数据加密（AES-256-GCM / SM4-GCM）
    - 持久化数据加密
    - 密钥管理和轮换
  - 审计日志
    - 结构化 JSON 日志
    - 完整的安全事件记录
    - Syslog 集成

- **持久化增强**
  - WAL 完整实现
  - 数据压缩（LZ4/Snappy）
  - 快照和恢复
  - 数据完整性校验

- **性能优化**
  - 零拷贝优化
  - 内存池优化
  - 批量操作支持

- **运维工具**
  - 数据备份和恢复
  - 配置热更新
  - 性能诊断工具

- **文档完善**
  - 完整的安全配置指南
  - 生产环境部署指南
  - 故障排查指南
  - 最佳实践文档

### 技术指标
- QPS: > 100,000 次/秒
- P99 延迟: < 1ms
- P999 延迟: < 5ms
- 缓存命中率: > 99%
- 单元测试覆盖率: > 80%

### 兼容性
- 向后兼容 v0.5.0 的所有 API
- 配置文件格式可能有破坏性变更（提供迁移工具）

---

## [2.0.0] - TBD

### 概述
分布式集群版本，支持高可用和水平扩展。

### 计划新增 (Planned Added)
- **分布式集群**
  - Gossip 协议节点发现
  - Quorum 副本同步
  - 自动故障转移
  - 数据分片和路由
  - 一致性哈希

- **高可用**
  - 主从复制
  - 读写分离
  - 多副本支持
  - 自动故障恢复

- **扩展性**
  - 水平扩展
  - 动态节点添加/移除
  - 数据再平衡

### 技术指标
- 集群 QPS: > 1,000,000 次/秒
- 可用性: 99.99%
- 支持 100+ 节点集群

### 兼容性
- 向后兼容 v1.0.0 的所有 API
- 单机模式可无缝升级到集群模式

---

## [3.0.0] - TBD

### 概述
企业版，完整的监控和运维工具。

### 计划新增 (Planned Added)
- **监控系统**
  - Prometheus metrics 导出
  - 完整的性能指标
  - 告警规则模板
  - Grafana 仪表板

- **运维工具**
  - 集群管理 CLI
  - 可视化管理界面
  - 自动化运维脚本
  - 滚动升级支持

- **企业特性**
  - 硬件国密卡支持（HSM）
  - KMS 集成
  - 多租户支持
  - 企业级技术支持

### 技术指标
- 支持 1000+ 节点超大规模集群
- 可用性: 99.999%

---

## 版本号规范

TokenginX 遵循 [Semantic Versioning 2.0.0](https://semver.org/lang/zh-CN/)：

- **主版本号（Major）**：不兼容的 API 变更
- **次版本号（Minor）**：向下兼容的功能性新增
- **修订号（Patch）**：向下兼容的问题修正

### 版本状态标识

- **Alpha**: 内部测试版本，功能不完整
- **Beta**: 公开测试版本，功能基本完整但可能有 Bug
- **RC (Release Candidate)**: 发布候选版本，准备正式发布
- **Stable**: 稳定版本，推荐生产使用

### 示例

- `v0.1.0-alpha.1`: 第一个 Alpha 测试版本
- `v0.1.0-beta.2`: 第二个 Beta 测试版本
- `v0.1.0-rc.1`: 第一个候选发布版本
- `v0.1.0`: 正式稳定版本
- `v0.1.1`: 第一个补丁版本

---

## 变更类型说明

- **Added（新增）**: 新功能
- **Changed（变更）**: 现有功能的变更
- **Deprecated（弃用）**: 即将移除的功能
- **Removed（移除）**: 已移除的功能
- **Fixed（修复）**: Bug 修复
- **Security（安全）**: 安全漏洞修复

---

**注**: 本变更日志将在每个版本发布前更新。未发布的版本内容可能会根据开发进度调整。
