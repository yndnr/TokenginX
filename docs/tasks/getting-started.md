# 开始开发 TokenginX

欢迎加入 TokenginX 项目！本文档将帮助您快速开始开发工作。

## 项目概述

TokenginX 是一个专为单点登录（SSO）优化的高性能会话存储系统，使用 Go 语言开发。

**核心特性**：
- 高性能：100K+ QPS，P99 < 1ms
- 多协议：OAuth 2.0/OIDC、SAML 2.0、CAS
- 多接口：TCP (RESP)、gRPC、HTTP/REST
- 安全优先：TLS 1.3、mTLS、国密支持
- 智能存储：256 分片 + W-TinyLFU 缓存

**当前版本**：v0.1.0（MVP 开发中）

## 快速开始

### 1. 克隆仓库

```bash
git clone https://github.com/your-org/tokenginx.git
cd tokenginx
```

### 2. 安装依赖

```bash
# 确保 Go 版本 >= 1.21
go version

# 下载依赖
go mod download

# 验证依赖
go mod verify
```

### 3. 运行测试

```bash
# 运行所有测试
go test ./...

# 运行带覆盖率的测试
go test -cover ./...

# 数据竞争检测
go test -race ./...
```

### 4. 构建项目

```bash
# 构建服务器
go build -o bin/tokenginx-server ./cmd/server

# 构建客户端
go build -o bin/tokenginx-client ./cmd/client

# 运行服务器
./bin/tokenginx-server -config config/config.example.yaml
```

## 开发流程

### 分支策略

```bash
# 主要分支
main        # 稳定版本（生产环境）
develop     # 开发分支（日常开发）

# 创建功能分支
git checkout -b feature/your-feature-name develop

# 创建修复分支
git checkout -b bugfix/issue-123 develop
```

### 提交规范

遵循 [Conventional Commits](https://www.conventionalcommits.org/) 规范：

```bash
# 格式
<type>(<scope>): <subject>

# 示例
feat(storage): 实现 ShardedMap 基础结构
fix(transport): 修复 RESP 协议解析错误
docs(readme): 更新安装说明
test(storage): 添加 TTL 管理单元测试
```

**类型说明**：
- `feat`: 新功能
- `fix`: Bug 修复
- `docs`: 文档更新
- `style`: 代码格式
- `refactor`: 重构
- `perf`: 性能优化
- `test`: 测试相关
- `chore`: 构建或工具变动

### 代码审查

1. 提交 Pull Request 到 `develop` 分支
2. 填写 PR 模板（变更描述、测试结果）
3. 等待至少一个维护者审查
4. 解决审查意见
5. CI 检查全部通过后合并

## 任务管理

### 查看任务

我们使用 GitHub Issues 和 Projects 进行任务管理。

**重要文档**：
- [项目路线图](./roadmap.md) - 完整的版本规划
- [v0.1.0 开发任务](./v0.1.0-dev-tasks.md) - MVP 开发任务详细清单
- [v0.1.0 测试任务](./v0.1.0-test-tasks.md) - MVP 测试任务清单
- [GitHub Projects 配置](./github-projects.md) - 看板和自动化

**查看 Issue**：

```bash
# 安装 GitHub CLI
# https://cli.github.com/

# 列出 v0.1.0 的所有任务
gh issue list --milestone "v0.1.0"

# 查看 P0 优先级任务
gh issue list --label "P0"

# 查看存储引擎相关任务
gh issue list --label "module:storage"
```

### 创建 GitHub Issues

我们提供了自动化脚本来创建 v0.1.0 的所有任务：

```bash
# 确保已安装并登录 GitHub CLI
gh auth login

# 运行脚本创建 20 个 Issue
./scripts/create-v0.1.0-issues.sh
```

**脚本会创建**：
- 3 个存储引擎任务
- 4 个传输层任务
- 3 个 OAuth 2.0 任务
- 2 个配置系统任务
- 2 个 CLI 工具任务
- 2 个监控日志任务
- 2 个文档任务
- 2 个测试任务

**总计**：20 个 Issue，预估 104 小时

### 认领任务

1. 在 GitHub Issue 中评论 "I'll take this"
2. 维护者会将 Issue 分配给你
3. 开始开发前阅读任务清单中的详细要求
4. 在开发过程中更新 Issue 进度

## 开发指南

### 目录结构

```
tokenginx/
├── cmd/
│   ├── server/          # 服务器主程序
│   └── client/          # 客户端工具
├── internal/            # 内部包（不可导出）
│   ├── storage/         # 存储引擎
│   ├── transport/       # 传输层
│   ├── protocol/        # 协议实现
│   ├── security/        # 安全模块
│   └── monitoring/      # 监控模块
├── pkg/                 # 公共库（可导出）
├── api/                 # API 定义（proto, OpenAPI）
├── config/              # 配置文件示例
├── scripts/             # 构建和部署脚本
├── tests/               # 集成测试和基准测试
│   ├── integration/
│   └── benchmark/
├── docs/                # 文档
└── deploy/              # 部署配置
    ├── docker/
    └── kubernetes/
```

### 代码规范

**遵循 Go 官方规范**：
- [Effective Go](https://go.dev/doc/effective_go)
- [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)

**关键要求**：
- 所有导出的函数、类型、常量必须有 GoDoc 注释
- 注释以声明的名称开头
- 提供实际可运行的示例
- 说明并发安全性和性能特点

**示例**：

```go
// NewShardedMap 创建一个新的分片哈希表实例
//
// 参数说明：
//   - initialCapacity: 每个分片的初始容量，建议设置为预期总容量的 1/256
//
// 返回值：
//   - *ShardedMap: 分片哈希表实例
//
// 示例：
//
//	// 创建一个预期存储 100 万个键值对的分片哈希表
//	sm := NewShardedMap(4096) // 4096 * 256 ≈ 1,000,000
//	sm.Set("key1", "value1", 3600)
//
// 注意事项：
//   - initialCapacity 为 0 时使用默认容量
//   - 该方法是并发安全的
func NewShardedMap(initialCapacity int) *ShardedMap {
    // 实现
}
```

### 测试要求

**单元测试**：
- 覆盖率 > 80%（整体）
- 核心模块覆盖率 > 90%
- 所有公共 API 必须有测试

**并发测试**：
```bash
# 必须通过数据竞争检测
go test -race ./...
```

**基准测试**：
```bash
# 性能关键路径需要基准测试
go test -bench=. -benchmem ./internal/storage/
```

**示例测试**：

```go
func TestShardedMap_SetGet(t *testing.T) {
    sm := NewShardedMap(1024)

    err := sm.Set("key1", "value1", 0)
    assert.NoError(t, err)

    value, found := sm.Get("key1")
    assert.True(t, found)
    assert.Equal(t, "value1", value)
}

func BenchmarkShardedMap_Get(b *testing.B) {
    sm := NewShardedMap(1024)
    sm.Set("key", "value", 0)

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        sm.Get("key")
    }
}
```

### 性能优化

**关键原则**：
- 使用 `sync.Pool` 减少内存分配
- 避免在热路径上使用反射
- 优先使用 `[]byte` 而非 `string`
- 使用 `atomic` 包进行无锁操作
- 性能关键路径使用 `//go:inline` 提示

**示例**：

```go
// 使用 sync.Pool 复用缓冲区
var bufferPool = sync.Pool{
    New: func() interface{} {
        return make([]byte, 4096)
    },
}

func processData(data []byte) {
    buf := bufferPool.Get().([]byte)
    defer bufferPool.Put(buf)

    // 使用 buf 处理数据
}
```

## v0.1.0 MVP 开发计划

### 时间规划

**总时长**：6 周（2025-12 ~ 2026-01）
**工时**：约 140h 开发 + 38h 测试 = 178h

### 团队配置

- 2 名开发工程师
- 1 名测试工程师

### 周计划

| 周次 | 开发内容 | 测试内容 |
|-----|---------|---------|
| Week 1 | 存储引擎（ShardedMap, TTL）<br>配置系统 | 边开发边单元测试 |
| Week 2 | 传输层（RESP, TCP, 命令处理） | 边开发边单元测试 |
| Week 3 | OAuth 2.0 协议<br>CLI 工具 | 单元测试补充 |
| Week 4 | 监控日志<br>文档完善 | 集成测试 |
| Week 5 | Bug 修复<br>性能优化 | 性能测试 |
| Week 6 | 最终测试<br>发布准备 | 安全测试<br>压力测试 |

### 里程碑

**Milestone 1（Week 2 结束）**：
- [x] 存储引擎完成
- [x] 基础 TCP 服务器可运行
- [x] 可用 `redis-cli` 连接

**Milestone 2（Week 4 结束）**：
- [ ] OAuth 2.0 支持完成
- [ ] CLI 工具完成
- [ ] 集成测试通过

**Milestone 3（Week 6 结束）**：
- [ ] 所有测试通过
- [ ] 性能达标（QPS > 100K, P99 < 1ms）
- [ ] 文档完整
- [ ] v0.1.0 正式发布

### 关键依赖

```
ShardedMap
    ↓
TTL 管理 + RESP 解析器
    ↓
TCP 服务器 + 命令处理器
    ↓
OAuth Token 存储
    ↓
配置系统
    ↓
tokenginx-server
    ↓
集成测试
```

**关键路径**：约 47 小时

## 常见问题

### Q: 如何选择任务？

**A**: 优先选择：
1. P0 优先级任务（关键路径）
2. 你熟悉的模块
3. 没有依赖的任务

查看 [v0.1.0 开发任务清单](./v0.1.0-dev-tasks.md) 了解任务依赖关系。

### Q: 如何运行单个模块的测试？

**A**:
```bash
# 运行存储模块测试
go test ./internal/storage/...

# 运行单个测试函数
go test -run TestShardedMap_SetGet ./internal/storage/

# 运行基准测试
go test -bench=BenchmarkShardedMap ./internal/storage/
```

### Q: 如何调试性能问题？

**A**:
```bash
# CPU 性能分析
go test -cpuprofile=cpu.prof -bench=. ./internal/storage/
go tool pprof cpu.prof

# 内存分析
go test -memprofile=mem.prof -bench=. ./internal/storage/
go tool pprof mem.prof

# 在线分析（运行中的服务器）
go tool pprof http://localhost:6060/debug/pprof/profile
```

### Q: 如何提交 Pull Request？

**A**:
1. 确保所有测试通过：`go test ./...`
2. 确保无数据竞争：`go test -race ./...`
3. 确保代码格式化：`go fmt ./...`
4. 确保代码检查通过：`go vet ./...`
5. 提交 PR 并填写模板
6. 等待 CI 检查通过
7. 等待代码审查

### Q: 遇到问题怎么办？

**A**:
- 阅读相关文档（[docs/](../../docs/)）
- 搜索 GitHub Issues
- 在 Issue 中提问
- 参加社区讨论（GitHub Discussions）

## 资源链接

### 文档

- [项目 README](../../readme.md)
- [贡献指南](../../contributing.md)
- [CLAUDE.md](../../claude.md) - Claude Code 开发指南
- [API 参考](../reference/)
- [安全文档](../security/)
- [部署文档](../deployment/)

### 任务管理

- [项目路线图](./roadmap.md)
- [GitHub Projects 配置](./github-projects.md)
- [v0.1.0 开发任务](./v0.1.0-dev-tasks.md)
- [v0.1.0 测试任务](./v0.1.0-test-tasks.md)

### 工具

- [GitHub Issues](https://github.com/your-org/tokenginx/issues)
- [GitHub Projects](https://github.com/your-org/tokenginx/projects)
- [GitHub Discussions](https://github.com/your-org/tokenginx/discussions)
- [GitHub Actions](https://github.com/your-org/tokenginx/actions)

### 外部资源

- [Go 官方文档](https://go.dev/doc/)
- [Effective Go](https://go.dev/doc/effective_go)
- [Redis RESP 协议](https://redis.io/docs/reference/protocol-spec/)
- [OAuth 2.0 RFC 6749](https://tools.ietf.org/html/rfc6749)
- [Conventional Commits](https://www.conventionalcommits.org/)

## 联系方式

- **GitHub Issues**: 报告 Bug 和功能请求
- **GitHub Discussions**: 技术讨论和问答
- **维护者**: @your-github-username

---

**欢迎贡献！** 让我们一起打造高性能的 SSO 会话存储系统！
