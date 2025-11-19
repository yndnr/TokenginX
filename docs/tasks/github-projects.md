# GitHub Projects 配置指南

本文档介绍如何使用 GitHub Projects 进行 TokenginX 项目的任务管理和进度跟踪。

## 项目面板设置

### 创建项目面板

1. **访问项目页面**
   - 进入 GitHub 仓库
   - 点击 "Projects" 标签
   - 点击 "New project"

2. **选择模板**
   - 推荐使用 "Team backlog" 模板
   - 或从空白项目开始自定义

3. **项目命名**
   - 项目名称：`TokenginX Development`
   - 描述：`High-performance SSO session storage system`

## 视图配置

### Board 视图（看板视图）

**列配置**：

```
📋 Backlog      - 待办任务
🎯 Todo         - 已计划任务
🔄 In Progress  - 进行中
✅ Done         - 已完成
🚫 Blocked      - 已阻塞
```

**自动化规则**：

- Issue 创建时自动进入 Backlog
- 分配给成员时自动移至 Todo
- PR 合并后自动移至 Done
- 标记为 blocked 时自动移至 Blocked

### Table 视图（表格视图）

**字段配置**：

| 字段 | 类型 | 说明 |
|-----|------|------|
| Title | 文本 | 任务标题 |
| Status | 单选 | Backlog/Todo/In Progress/Done/Blocked |
| Priority | 单选 | P0/P1/P2/P3 |
| Version | 单选 | v0.1.0/v0.5.0/v1.0.0/v2.0.0/v3.0.0 |
| Module | 单选 | Storage/Transport/Protocol/Security/... |
| Estimate | 数字 | 预估工时（小时） |
| Assignees | 人员 | 负责人 |
| Labels | 标签 | bug/enhancement/documentation/... |
| Due date | 日期 | 截止日期 |

### Roadmap 视图（路线图视图）

**时间线配置**：

- 显示所有版本的里程碑
- 按 Version 字段分组
- 显示任务依赖关系
- 高亮关键路径

## 里程碑配置

### v0.1.0 - MVP（最小可行产品）

**时间**：2025-12 ~ 2026-01（6 周）

**目标**：
- 单机存储引擎
- TCP (RESP) 接口
- OAuth 2.0 基础支持
- 基础配置和测试

**任务**：
- 共 45 个 Issue（参考 roadmap.md）
- 优先级：28 个 P0，12 个 P1，5 个 P2

### v0.5.0 - 完整协议支持

**时间**：2026-02 ~ 2026-04（10 周）

**目标**：
- SAML 2.0 支持
- CAS 支持
- gRPC 和 HTTP/REST 接口
- 完整的安全特性

### v1.0.0 - 生产可用

**时间**：2026-05 ~ 2026-07（12 周）

**目标**：
- 国密支持
- 安全加固
- 性能优化
- 完整文档

### v2.0.0 - 分布式集群

**时间**：2026-08 ~ 2026-12（20 周）

**目标**：
- Gossip 协议
- Quorum 副本同步
- 自动故障转移

### v3.0.0 - 企业版

**时间**：2027-01 ~ 2027-06（24 周）

**目标**：
- Prometheus 监控
- 运维工具
- 企业级支持

## 标签（Labels）体系

### 类型标签

```yaml
bug:
  color: "d73a4a"
  description: "Bug 修复"

enhancement:
  color: "a2eeef"
  description: "新功能"

documentation:
  color: "0075ca"
  description: "文档更新"

refactor:
  color: "fbca04"
  description: "代码重构"

test:
  color: "1d76db"
  description: "测试相关"

performance:
  color: "5319e7"
  description: "性能优化"

security:
  color: "d93f0b"
  description: "安全相关"
```

### 优先级标签

```yaml
P0:
  color: "b60205"
  description: "关键优先级（阻塞发布）"

P1:
  color: "d93f0b"
  description: "高优先级（重要功能）"

P2:
  color: "fbca04"
  description: "中优先级（增强功能）"

P3:
  color: "0e8a16"
  description: "低优先级（可选功能）"
```

### 模块标签

```yaml
module:storage:
  color: "c5def5"
  description: "存储引擎模块"

module:transport:
  color: "c5def5"
  description: "传输层模块"

module:protocol:
  color: "c5def5"
  description: "协议模块"

module:security:
  color: "c5def5"
  description: "安全模块"

module:cluster:
  color: "c5def5"
  description: "集群模块"

module:monitoring:
  color: "c5def5"
  description: "监控模块"
```

### 状态标签

```yaml
blocked:
  color: "000000"
  description: "已阻塞"

needs-review:
  color: "fbca04"
  description: "需要审查"

good first issue:
  color: "7057ff"
  description: "适合新手"

help wanted:
  color: "008672"
  description: "需要帮助"
```

## Issue 模板配置

### Bug Report

路径：`.github/ISSUE_TEMPLATE/bug_report.md`

**自动标签**：`bug`
**自动分配**：进入 Backlog

### Feature Request

路径：`.github/ISSUE_TEMPLATE/feature_request.md`

**自动标签**：`enhancement`
**自动分配**：进入 Backlog

### Task Template

创建新模板 `.github/ISSUE_TEMPLATE/task.md`：

```markdown
---
name: 开发任务
about: 创建一个开发任务
title: '[TASK] '
labels: ''
assignees: ''
---

## 任务描述

简要描述任务内容

## 模块

- [ ] Storage
- [ ] Transport
- [ ] Protocol
- [ ] Security
- [ ] Cluster
- [ ] Monitoring

## 版本

- [ ] v0.1.0
- [ ] v0.5.0
- [ ] v1.0.0
- [ ] v2.0.0
- [ ] v3.0.0

## 优先级

- [ ] P0 - 关键
- [ ] P1 - 高
- [ ] P2 - 中
- [ ] P3 - 低

## 任务清单

- [ ] 子任务 1
- [ ] 子任务 2
- [ ] 子任务 3

## 预估工时

预估：X 小时

## 依赖

依赖的其他 Issue：#XXX

## 验收标准

- [ ] 标准 1
- [ ] 标准 2
- [ ] 单元测试通过
- [ ] 文档已更新
```

## 自动化工作流

### Issue 创建自动化

创建 `.github/workflows/issue-automation.yml`：

```yaml
name: Issue Automation

on:
  issues:
    types: [opened, labeled]

jobs:
  auto-assign-project:
    runs-on: ubuntu-latest
    steps:
      - name: Add to project
        uses: actions/add-to-project@v0.5.0
        with:
          project-url: https://github.com/orgs/your-org/projects/1
          github-token: ${{ secrets.ADD_TO_PROJECT_PAT }}

  auto-set-priority:
    runs-on: ubuntu-latest
    if: contains(github.event.issue.labels.*.name, 'bug')
    steps:
      - name: Add P1 label to bugs
        uses: actions-ecosystem/action-add-labels@v1
        with:
          labels: P1
```

### PR 合并自动化

创建 `.github/workflows/pr-automation.yml`：

```yaml
name: PR Automation

on:
  pull_request:
    types: [closed]

jobs:
  auto-close-issues:
    if: github.event.pull_request.merged == true
    runs-on: ubuntu-latest
    steps:
      - name: Close linked issues
        uses: actions/github-script@v6
        with:
          script: |
            const pr = context.payload.pull_request;
            const body = pr.body || '';
            const regex = /(?:close|closes|closed|fix|fixes|fixed|resolve|resolves|resolved)\s+#(\d+)/gi;
            let match;
            while ((match = regex.exec(body)) !== null) {
              const issueNumber = match[1];
              await github.rest.issues.update({
                owner: context.repo.owner,
                repo: context.repo.repo,
                issue_number: issueNumber,
                state: 'closed'
              });
            }
```

## 任务导入

### 批量创建 v0.1.0 任务

使用 GitHub CLI 批量创建任务：

```bash
#!/bin/bash
# scripts/create-v0.1.0-issues.sh

# 存储引擎任务
gh issue create --title "[Storage] 实现 ShardedMap 基础结构" \
  --body "实现 256 分片的核心存储结构" \
  --label "module:storage,P0" \
  --milestone "v0.1.0"

gh issue create --title "[Storage] 实现 TTL 管理机制" \
  --body "实现惰性删除和定期清理" \
  --label "module:storage,P0" \
  --milestone "v0.1.0"

gh issue create --title "[Storage] 实现基础 LRU 缓存" \
  --body "实现简化版 LRU 缓存淘汰策略" \
  --label "module:storage,P1" \
  --milestone "v0.1.0"

# TCP 传输层任务
gh issue create --title "[Transport] 实现 RESP 协议解析器" \
  --body "实现 Redis RESP 协议的解析" \
  --label "module:transport,P0" \
  --milestone "v0.1.0"

gh issue create --title "[Transport] 实现基础命令处理" \
  --body "实现 GET、SET、DEL 命令" \
  --label "module:transport,P0" \
  --milestone "v0.1.0"

# ... 更多任务
```

执行脚本：

```bash
chmod +x scripts/create-v0.1.0-issues.sh
./scripts/create-v0.1.0-issues.sh
```

### CSV 导入（备选方案）

创建 `docs/tasks/v0.1.0-tasks.csv`：

```csv
Title,Labels,Milestone,Priority,Module,Estimate
实现 ShardedMap 基础结构,enhancement,v0.1.0,P0,Storage,6
实现 TTL 管理机制,enhancement,v0.1.0,P0,Storage,4
实现基础 LRU 缓存,enhancement,v0.1.0,P1,Storage,4
实现 RESP 协议解析器,enhancement,v0.1.0,P0,Transport,6
实现基础命令处理,enhancement,v0.1.0,P0,Transport,4
```

使用工具导入（如 GitHub Importer 插件）

## 进度跟踪

### 周报模板

创建 `docs/tasks/weekly-report-template.md`：

```markdown
# TokenginX 开发周报 - Week XX

## 时间范围

YYYY-MM-DD ~ YYYY-MM-DD

## 本周完成

- [x] #123 - 实现 ShardedMap 基础结构
- [x] #124 - 实现 TTL 管理机制
- [x] #125 - 实现 RESP 协议解析器

**完成任务数**：3
**完成工时**：16h
**代码变更**：+1200 -50 lines

## 本周进行中

- [ ] #126 - 实现基础 LRU 缓存（80% 完成）
- [ ] #127 - 实现基础命令处理（60% 完成）

## 下周计划

- [ ] #126 - 完成基础 LRU 缓存
- [ ] #127 - 完成基础命令处理
- [ ] #128 - 实现并发测试

## 遇到的问题

1. **问题**：分片锁竞争导致性能下降
   - **影响**：QPS 低于目标 20%
   - **解决方案**：优化锁粒度，使用读写锁

2. **问题**：RESP 协议解析内存分配过多
   - **影响**：GC 压力大
   - **解决方案**：使用 sync.Pool 复用缓冲区

## 风险和阻塞

- 无重大风险
- 无阻塞任务

## 下周资源需求

- 需要安全专家审查 TLS 实现
```

### 月度总结模板

创建 `docs/tasks/monthly-summary-template.md`：

```markdown
# TokenginX 月度总结 - YYYY-MM

## 总体进度

**当前版本**：v0.1.0
**进度**：45 / 45 任务（100%）
**代码统计**：
- 新增代码：+12,000 lines
- 删除代码：-500 lines
- 单元测试覆盖率：85%

## 里程碑完成情况

- [x] 存储引擎（14h，已完成）
- [x] TCP 传输层（20h，已完成）
- [ ] OAuth 2.0 支持（10h，进行中 80%）
- [ ] 配置系统（6h，待开始）

## 关键成果

1. **性能达标**：QPS 120,000+，超出目标 20%
2. **测试覆盖**：单元测试覆盖率 85%
3. **文档完善**：完成所有核心模块文档

## 下月计划

1. 完成 OAuth 2.0 支持
2. 完成配置系统
3. 完成 CLI 工具
4. 开始 v0.2.0 规划

## 团队

- 开发：2 人
- 测试：1 人
- 总工时：240h
```

## 最佳实践

### Issue 创建

1. **标题清晰**：使用 `[模块] 动词 + 描述` 格式
   - ✅ `[Storage] 实现 ShardedMap 基础结构`
   - ❌ `修复问题`

2. **描述详细**：包含背景、目标、验收标准
3. **标签完整**：至少包含类型、优先级、模块
4. **关联 Milestone**：所有任务必须关联版本
5. **预估工时**：提供合理的时间估算

### Pull Request

1. **关联 Issue**：使用 `Closes #123` 关键字
2. **描述变更**：清晰说明做了什么、为什么
3. **测试证明**：提供测试结果截图或日志
4. **代码审查**：至少一个 Reviewer 批准

### 进度更新

1. **每日更新**：在 Issue 中评论进度
2. **周报**：每周五发布周报
3. **月报**：每月最后一天发布月报
4. **即时沟通**：遇到阻塞及时在 Issue 中标记

## 相关资源

- [GitHub Projects 文档](https://docs.github.com/en/issues/planning-and-tracking-with-projects)
- [GitHub CLI 文档](https://cli.github.com/manual/)
- [项目路线图](./roadmap.md)
- [贡献指南](../../contributing.md)
