# 贡献指南

感谢您对 TokenginX 项目的关注！我们欢迎所有形式的贡献，包括但不限于：

- 报告 Bug
- 提出功能请求
- 提交代码改进
- 完善文档
- 分享使用经验

## 行为准则

参与本项目即表示您同意遵守我们的行为准则：

- **尊重他人**：尊重所有贡献者，无论其背景、经验水平如何
- **建设性反馈**：提供具体、有帮助的建议，而非单纯批评
- **开放协作**：欢迎不同观点，通过讨论达成共识
- **专业态度**：保持技术讨论的专业性和客观性

## 如何贡献

### 报告 Bug

如果您发现了 Bug，请通过 [GitHub Issues](https://github.com/your-org/tokenginx/issues) 报告，并提供以下信息：

1. **Bug 描述**：清晰简洁地描述问题
2. **复现步骤**：详细的步骤说明
   ```
   1. 启动服务器 './tokenginx-server -config config.yaml'
   2. 执行命令 '...'
   3. 观察到错误 '...'
   ```
3. **预期行为**：您期望发生什么
4. **实际行为**：实际发生了什么
5. **环境信息**：
   - 操作系统（Linux/macOS/Windows）
   - Go 版本（`go version`）
   - TokenginX 版本
   - 相关配置文件内容

**Bug 报告模板**：

```markdown
**描述**
简短描述 Bug

**复现步骤**
1. ...
2. ...
3. ...

**预期行为**
描述您期望的结果

**实际行为**
描述实际发生的情况

**环境**
- OS: [e.g. Ubuntu 22.04]
- Go 版本: [e.g. 1.21.5]
- TokenginX 版本: [e.g. v0.1.0]

**配置文件**
```yaml
# 粘贴相关配置
```

**日志输出**
```
# 粘贴相关日志
```

**附加信息**
其他有助于理解问题的信息
```

### 提出功能请求

如果您有新功能的想法，请通过 [GitHub Issues](https://github.com/your-org/tokenginx/issues) 提交功能请求：

1. **功能描述**：清晰描述您希望添加的功能
2. **使用场景**：说明为什么需要这个功能
3. **预期实现**：您认为应该如何实现（可选）
4. **替代方案**：是否考虑过其他解决方案（可选）

**功能请求模板**：

```markdown
**功能描述**
简短描述新功能

**使用场景**
描述这个功能解决什么问题，为什么需要它

**预期实现**
描述您期望的实现方式（可选）

**替代方案**
描述您考虑过的其他解决方案（可选）

**附加信息**
其他有助于理解需求的信息
```

## 开发环境设置

### 前置要求

- **Go**: >= 1.21
- **Git**: >= 2.x
- **Make**: 用于构建自动化（可选）
- **golangci-lint**: 用于代码检查（可选）

### 克隆仓库

```bash
# 克隆主仓库
git clone https://github.com/your-org/tokenginx.git
cd tokenginx

# 如果您没有写权限，先 Fork 后克隆
git clone https://github.com/YOUR_USERNAME/tokenginx.git
cd tokenginx

# 添加上游仓库
git remote add upstream https://github.com/your-org/tokenginx.git
```

### 安装依赖

```bash
# 下载依赖
go mod download

# 验证依赖
go mod verify
```

### 运行测试

```bash
# 运行所有测试
go test ./...

# 运行带覆盖率的测试
go test -cover ./...

# 运行数据竞争检测
go test -race ./...

# 运行基准测试
go test -bench=. -benchmem ./...
```

### 代码检查

```bash
# 格式化代码
go fmt ./...

# 静态分析
go vet ./...

# 使用 golangci-lint（推荐）
golangci-lint run
```

## Git 工作流

我们遵循标准的 Git Flow 工作流程：

### 分支策略

- **main**: 稳定版本，用于生产发布
- **develop**: 开发分支，日常开发的主分支
- **feature/***: 新功能分支
- **bugfix/***: Bug 修复分支
- **hotfix/***: 紧急修复分支
- **release/***: 发布准备分支

### 创建功能分支

```bash
# 确保 develop 分支是最新的
git checkout develop
git pull upstream develop

# 创建功能分支
git checkout -b feature/your-feature-name

# 或者修复 Bug
git checkout -b bugfix/issue-123-session-expiry
```

### 提交代码

我们遵循 [Conventional Commits](https://www.conventionalcommits.org/) 规范：

**格式**：`<type>(<scope>): <subject>`

**类型（type）**：
- `feat`: 新功能
- `fix`: Bug 修复
- `docs`: 文档更新
- `style`: 代码格式（不影响功能）
- `refactor`: 重构（既不是新功能也不是修复）
- `perf`: 性能优化
- `test`: 测试相关
- `chore`: 构建过程或辅助工具的变动
- `ci`: CI/CD 配置文件和脚本的变动

**作用域（scope）**（可选）：
- `storage`: 存储引擎
- `protocol`: 协议实现
- `transport`: 传输层
- `security`: 安全模块
- `oauth`, `saml`, `cas`: 具体协议

**示例**：

```bash
# 新功能
git commit -m "feat(oauth): add authorization code flow support"

# Bug 修复
git commit -m "fix(storage): resolve race condition in shard locking"

# 性能优化
git commit -m "perf(cache): optimize w-tinylfu eviction algorithm"

# 文档更新
git commit -m "docs(readme): update installation instructions"

# 测试
git commit -m "test(transport): add grpc integration tests"
```

### 提交代码前的检查清单

- [ ] 代码已格式化（`go fmt ./...`）
- [ ] 通过静态分析（`go vet ./...`）
- [ ] 所有测试通过（`go test ./...`）
- [ ] 无数据竞争（`go test -race ./...`）
- [ ] 添加了必要的单元测试
- [ ] 添加了必要的 GoDoc 注释
- [ ] 更新了相关文档

### 推送并创建 Pull Request

```bash
# 推送到您的 Fork
git push origin feature/your-feature-name

# 如果是直接贡献者
git push origin feature/your-feature-name
```

然后在 GitHub 上创建 Pull Request：

1. 访问项目仓库页面
2. 点击 "Pull requests" -> "New pull request"
3. 选择您的分支
4. 填写 PR 标题和描述（见下文）
5. 提交 Pull Request

## Pull Request 规范

### PR 标题

PR 标题应遵循 Conventional Commits 格式：

```
feat(oauth): add PKCE support for authorization code flow
fix(storage): prevent memory leak in shard cleanup
docs(security): add national cryptography configuration guide
```

### PR 描述模板

```markdown
## 摘要
简短描述这个 PR 的目的和主要变更

## 变更类型
- [ ] Bug 修复
- [ ] 新功能
- [ ] 代码重构
- [ ] 性能优化
- [ ] 文档更新
- [ ] 测试改进
- [ ] CI/CD 改进

## 关联 Issue
Closes #123
Fixes #456

## 变更说明
详细说明做了哪些变更：
- 添加了 XXX 功能
- 修复了 YYY 问题
- 优化了 ZZZ 性能

## 测试
描述如何测试这些变更：
- [ ] 添加了单元测试
- [ ] 添加了集成测试
- [ ] 手动测试通过
- [ ] 性能基准测试（如适用）

## 文档
- [ ] 更新了 API 文档
- [ ] 更新了用户文档
- [ ] 更新了 changelog.md
- [ ] 添加了代码注释

## 检查清单
- [ ] 代码已格式化
- [ ] 通过所有测试
- [ ] 无数据竞争
- [ ] 更新了文档
- [ ] 遵循项目代码风格

## 截图（如适用）
如果有 UI 变更或性能改进，可以添加截图或性能对比

## 附加信息
其他需要审查者注意的信息
```

## 代码审查

所有代码必须经过至少一名维护者的审查才能合并。

### 审查重点

审查者会关注以下方面：

1. **架构和设计**
   - 是否符合项目整体架构
   - 是否遵循设计模式
   - 是否有更好的实现方式

2. **代码质量**
   - 代码是否清晰易读
   - 变量/函数命名是否恰当
   - 是否有适当的注释
   - 是否有代码重复

3. **测试覆盖**
   - 是否有充分的单元测试
   - 测试是否覆盖边界情况
   - 是否有集成测试（如适用）

4. **性能影响**
   - 是否影响性能（正面或负面）
   - 是否需要性能基准测试
   - 是否有内存泄漏风险

5. **安全性**
   - 是否引入安全漏洞
   - 是否有输入验证
   - 是否有权限检查

6. **兼容性**
   - 是否破坏向后兼容性
   - 是否需要更新 API 版本
   - 是否需要迁移指南

### 如何响应审查意见

- **及时响应**：尽快回复审查意见
- **积极讨论**：如有不同意见，礼貌地讨论
- **接受建议**：虚心接受合理的改进建议
- **标记完成**：修改后标记对话为已解决

## 代码风格指南

### Go 代码规范

我们遵循 Go 官方代码风格：

- [Effective Go](https://go.dev/doc/effective_go)
- [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)

### 命名规范

**包名**：小写，单词，不使用下划线或驼峰
```go
package storage  // ✅
package storageEngine  // ❌
```

**变量/函数名**：驼峰命名
```go
var sessionTimeout int  // ✅
var session_timeout int  // ❌

func getUserSession() {}  // ✅
func get_user_session() {}  // ❌
```

**常量**：驼峰或全大写（根据导出情况）
```go
const DefaultTimeout = 300  // 导出常量
const maxRetries = 3  // 内部常量
```

### 注释规范

所有导出的函数、类型、常量都必须有注释。详细规范见 [specs/08-开源协作规范.md](./specs/08-开源协作规范.md#代码注释规范遵循-go-和-github-标准)。

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
//	sm := NewShardedMap(4096)
//	sm.Set("key1", "value1", 3600)
func NewShardedMap(initialCapacity int) *ShardedMap {
    // 实现
}
```

### 错误处理

**返回错误，不要 panic**：

```go
// ✅ 好的做法
func getSession(key string) (*Session, error) {
    if key == "" {
        return nil, errors.New("key cannot be empty")
    }
    // ...
}

// ❌ 不好的做法
func getSession(key string) *Session {
    if key == "" {
        panic("key cannot be empty")
    }
    // ...
}
```

**错误信息使用小写，不以标点符号结尾**：

```go
errors.New("session not found")  // ✅
errors.New("Session not found.")  // ❌
```

### 并发安全

**文档化并发安全性**：

```go
// ShardedMap 是一个线程安全的分片哈希表
// 所有公共方法都是并发安全的
type ShardedMap struct {
    // ...
}
```

**使用 sync.RWMutex 进行读写分离**：

```go
type mapShard struct {
    mu sync.RWMutex
    data map[string]interface{}
}

func (s *mapShard) Get(key string) (interface{}, bool) {
    s.mu.RLock()
    defer s.mu.RUnlock()
    val, ok := s.data[key]
    return val, ok
}
```

## 测试指南

### 单元测试

- 测试文件命名：`xxx_test.go`
- 测试函数命名：`TestXxx` 或 `TestXxx_Scenario`
- 覆盖率目标：> 80%

**示例**：

```go
func TestShardedMap_Set(t *testing.T) {
    sm := NewShardedMap(10)
    err := sm.Set("key1", "value1", 3600)
    if err != nil {
        t.Errorf("Set() error = %v", err)
    }
}

func TestShardedMap_Get_NotFound(t *testing.T) {
    sm := NewShardedMap(10)
    _, found := sm.Get("nonexistent")
    if found {
        t.Error("Get() should return false for nonexistent key")
    }
}
```

### 基准测试

性能关键的代码需要基准测试：

```go
func BenchmarkShardedMap_Set(b *testing.B) {
    sm := NewShardedMap(1000)
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        key := fmt.Sprintf("key%d", i)
        sm.Set(key, "value", 3600)
    }
}
```

### 数据竞争检测

```bash
go test -race ./...
```

## 文档贡献

### 文档结构

- **specs/**: 项目规范和技术说明（面向开发者）
- **docs/**: 用户文档（面向使用者）
- **readme.md**: 项目主说明文件
- **contributing.md**: 本文件
- **changelog.md**: 版本变更日志

### 文档更新原则

当您贡献代码时，请同步更新相关文档：

1. **新功能**：
   - 更新 API 参考文档
   - 更新快速指南或生产环境指南
   - 更新 changelog.md

2. **协议支持**：
   - 更新协议集成指南
   - 更新 specs/03-核心架构.md

3. **安全特性**：
   - 更新安全文档
   - 更新 specs/04-安全特性.md

4. **配置变更**：
   - 更新配置参考文档
   - 更新配置示例文件

### Markdown 规范

- 所有文件名使用小写（如 `readme.md`, `contributing.md`）
- 使用中文时注意标点符号使用全角
- 代码块指定语言：\`\`\`go, \`\`\`bash, \`\`\`yaml
- 使用相对链接引用其他文档

## 发布流程

发布流程由维护者执行，但了解流程有助于贡献者理解版本管理：

1. **创建发布分支**
   ```bash
   git checkout -b release/v0.1.0 develop
   ```

2. **更新版本号和 changelog**
   - 编辑 `version.go` 或 `Makefile` 中的版本号
   - 更新 `changelog.md`

3. **提交版本更新**
   ```bash
   git commit -am "chore(release): bump version to v0.1.0"
   ```

4. **合并到 main 并打标签**
   ```bash
   git checkout main
   git merge --no-ff release/v0.1.0
   git tag -a v0.1.0 -m "Release version 0.1.0"
   git push origin main --tags
   ```

5. **合并回 develop**
   ```bash
   git checkout develop
   git merge --no-ff release/v0.1.0
   git push origin develop
   ```

6. **GitHub Actions 自动化**
   - 运行完整测试套件
   - 构建多平台二进制文件
   - 创建 GitHub Release
   - 上传构建产物

## 获取帮助

如果您在贡献过程中遇到问题：

- **技术问题**：在 [GitHub Issues](https://github.com/your-org/tokenginx/issues) 提问
- **讨论想法**：在 [GitHub Discussions](https://github.com/your-org/tokenginx/discussions) 讨论
- **文档问题**：直接提交 PR 修复

## 感谢您的贡献

每一个贡献都让 TokenginX 变得更好。我们感谢您花时间为本项目做出贡献！

---

**问题反馈**：如果您对本贡献指南有任何建议，欢迎提交 Issue 或 PR。
