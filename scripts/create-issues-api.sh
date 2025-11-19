#!/bin/bash
# 使用 GitHub API 创建 v0.1.0 Issues（不需要 GitHub CLI）
#
# 使用方法：
#   1. 获取 Personal Access Token (https://github.com/settings/tokens)
#      权限：repo (完整仓库权限)
#   2. 运行脚本：
#      export GITHUB_TOKEN="your_token_here"
#      export REPO_OWNER="your_username"
#      export REPO_NAME="tokenginx"
#      ./scripts/create-issues-api.sh

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

print_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

print_header() {
    echo ""
    echo -e "${GREEN}=====================================${NC}"
    echo -e "${GREEN}  创建 v0.1.0 GitHub Issues${NC}"
    echo -e "${GREEN}=====================================${NC}"
    echo ""
}

# 检查环境变量
check_env() {
    if [ -z "$GITHUB_TOKEN" ]; then
        print_error "请设置 GITHUB_TOKEN 环境变量"
        echo ""
        echo "获取 Token: https://github.com/settings/tokens"
        echo "需要的权限: repo (完整仓库权限)"
        echo ""
        echo "设置方法:"
        echo "  export GITHUB_TOKEN=\"your_token_here\""
        exit 1
    fi

    if [ -z "$REPO_OWNER" ]; then
        print_error "请设置 REPO_OWNER 环境变量（仓库所有者）"
        echo "  export REPO_OWNER=\"your_username\""
        exit 1
    fi

    if [ -z "$REPO_NAME" ]; then
        REPO_NAME="tokenginx"
        print_info "使用默认仓库名: $REPO_NAME"
    fi
}

# 创建或获取 milestone
create_milestone() {
    local milestone_title="v0.1.0"
    local milestone_desc="MVP - 最小可行产品"
    local due_date="2026-01-31T23:59:59Z"

    print_info "检查 milestone: $milestone_title"

    # 获取现有 milestones
    local milestone_number=$(curl -s -H "Authorization: token $GITHUB_TOKEN" \
        "https://api.github.com/repos/$REPO_OWNER/$REPO_NAME/milestones" \
        | grep -B3 "\"title\": \"$milestone_title\"" \
        | grep "\"number\":" \
        | head -1 \
        | sed 's/.*: \([0-9]*\).*/\1/')

    if [ -n "$milestone_number" ]; then
        print_success "Milestone 已存在: $milestone_title (编号: $milestone_number)"
        echo "$milestone_number"
    else
        print_info "创建 milestone: $milestone_title"

        local response=$(curl -s -X POST \
            -H "Authorization: token $GITHUB_TOKEN" \
            -H "Accept: application/vnd.github.v3+json" \
            "https://api.github.com/repos/$REPO_OWNER/$REPO_NAME/milestones" \
            -d "{
                \"title\": \"$milestone_title\",
                \"description\": \"$milestone_desc\",
                \"due_on\": \"$due_date\"
            }")

        milestone_number=$(echo "$response" | grep '"number":' | head -1 | sed 's/.*: \([0-9]*\).*/\1/')

        if [ -n "$milestone_number" ]; then
            print_success "Milestone 创建成功 (编号: $milestone_number)"
            echo "$milestone_number"
        else
            print_error "Milestone 创建失败"
            echo "$response"
            exit 1
        fi
    fi
}

# 创建 Issue
create_issue() {
    local title="$1"
    local body="$2"
    local labels="$3"
    local milestone="$4"

    print_info "创建 Issue: $title"

    local response=$(curl -s -X POST \
        -H "Authorization: token $GITHUB_TOKEN" \
        -H "Accept: application/vnd.github.v3+json" \
        "https://api.github.com/repos/$REPO_OWNER/$REPO_NAME/issues" \
        -d "{
            \"title\": \"$title\",
            \"body\": $(echo "$body" | jq -Rs .),
            \"labels\": [$(echo "$labels" | sed 's/,/","/g' | sed 's/^/"/' | sed 's/$/"/')]
            $([ -n "$milestone" ] && echo ", \"milestone\": $milestone" || echo "")
        }")

    local issue_number=$(echo "$response" | grep '"number":' | head -1 | sed 's/.*: \([0-9]*\).*/\1/')

    if [ -n "$issue_number" ]; then
        print_success "Issue #$issue_number 创建成功"
    else
        print_error "Issue 创建失败: $title"
        echo "$response" | head -20
    fi
}

# ============================================
# 主程序
# ============================================

print_header

# 检查依赖
if ! command -v jq &> /dev/null; then
    print_error "需要安装 jq"
    echo "安装方法: sudo apt install jq"
    exit 1
fi

if ! command -v curl &> /dev/null; then
    print_error "需要安装 curl"
    exit 1
fi

# 检查环境变量
check_env

print_info "仓库: $REPO_OWNER/$REPO_NAME"
print_info "Token: ${GITHUB_TOKEN:0:8}..."
echo ""

# 创建 milestone
MILESTONE=$(create_milestone)
echo ""

# 创建 Issues
ISSUE_COUNT=0

# 1. 存储引擎
print_info "创建存储引擎 Issues..."

create_issue \
    "[Storage] 实现 ShardedMap 基础结构" \
    "## 任务描述

实现 256 分片的核心存储结构 \\\`ShardedMap\\\`，这是 TokenginX 存储引擎的核心组件。

## 实现要点

- 定义 \\\`ShardedMap\\\` 结构体（256 个分片）
- 实现哈希分片函数（一致性哈希）
- 实现 Set/Get/Delete/Exists 方法
- 每个分片使用 \\\`sync.RWMutex\\\` 保证并发安全
- 支持基础统计（Len 方法）

## 验收标准

- [ ] 所有单元测试通过（覆盖率 > 90%）
- [ ] 无数据竞争（\\\`go test -race\\\` 通过）
- [ ] 性能达标：单分片 GET > 1M ops/s
- [ ] 代码符合 Go 规范（\\\`go vet\\\` 通过）

## 相关文档

- [开发任务清单](docs/tasks/v0.1.0-dev-tasks.md#11-shardedmap-实现6h)
- [测试任务清单](docs/tasks/v0.1.0-test-tasks.md#shardedmap-测试3h)

预估工时：6h" \
    "module:storage,P0,enhancement" \
    "$MILESTONE"
((ISSUE_COUNT++))

create_issue \
    "[Storage] 实现 TTL 管理机制" \
    "## 任务描述

实现会话过期的 TTL（Time-To-Live）管理机制，包括惰性删除和定期清理。

## 实现要点

- 在 \\\`Get()\\\` 时检查过期（惰性删除）
- 后台 Goroutine 定期扫描过期键（定期清理）
- 实现 \\\`TTL(key)\\\` 获取剩余时间
- 实现 \\\`Expire(key, ttl)\\\` 更新 TTL
- 清理策略可配置（清理间隔、每次清理数量）

## 验收标准

- [ ] 惰性删除正常工作
- [ ] 定期清理不影响性能（CPU < 5%）
- [ ] 过期键能及时清理（延迟 < 1s）
- [ ] 单元测试覆盖率 > 85%

预估工时：4h" \
    "module:storage,P0,enhancement" \
    "$MILESTONE"
((ISSUE_COUNT++))

create_issue \
    "[Storage] 实现基础 LRU 缓存" \
    "## 任务描述

实现简化版的 LRU（Least Recently Used）缓存淘汰策略。

## 验收标准

- [ ] LRU 淘汰策略正确
- [ ] 缓存命中率 > 90%（模拟测试）
- [ ] 性能开销 < 10%

预估工时：4h" \
    "module:storage,P1,enhancement" \
    "$MILESTONE"
((ISSUE_COUNT++))

# 2. 传输层
print_info "创建传输层 Issues..."

create_issue \
    "[Transport] 实现 RESP 协议解析器" \
    "## 任务描述

实现 Redis RESP 协议的解析器和序列化器。

## 验收标准

- [ ] 兼容 Redis RESP 协议
- [ ] 解析速度 > 100K ops/s
- [ ] 单元测试覆盖率 > 95%

预估工时：6h" \
    "module:transport,P0,enhancement" \
    "$MILESTONE"
((ISSUE_COUNT++))

create_issue \
    "[Transport] 实现 TCP 服务器" \
    "## 任务描述

实现 TCP 服务器，支持 10,000+ 并发连接。

## 验收标准

- [ ] 支持 10,000+ 并发连接
- [ ] 优雅关闭无连接丢失
- [ ] 可使用 redis-cli 连接测试

预估工时：6h" \
    "module:transport,P0,enhancement" \
    "$MILESTONE"
((ISSUE_COUNT++))

create_issue \
    "[Transport] 实现基础命令处理器" \
    "## 任务描述

实现基础 Redis 命令：GET、SET、DEL、EXISTS、TTL、EXPIRE、PING、ECHO。

## 验收标准

- [ ] 所有命令符合 Redis 语义
- [ ] 错误处理完整
- [ ] 每个命令有独立测试

预估工时：4h" \
    "module:transport,P0,enhancement" \
    "$MILESTONE"
((ISSUE_COUNT++))

create_issue \
    "[Transport] 实现连接管理和监控" \
    "## 任务描述

实现 CLIENT LIST、CLIENT KILL、INFO 命令和连接管理。

## 验收标准

- [ ] CLIENT 命令正常工作
- [ ] INFO 输出格式正确
- [ ] 速率限制有效

预估工时：4h" \
    "module:transport,P1,enhancement" \
    "$MILESTONE"
((ISSUE_COUNT++))

# 3. OAuth 2.0
print_info "创建 OAuth 2.0 Issues..."

create_issue \
    "[Protocol] 实现 OAuth 2.0 Token 存储" \
    "## 任务描述

实现 OAuth 2.0 Token 的存储和管理。

## 验收标准

- [ ] Token 存储和检索正确
- [ ] TTL 自动过期生效
- [ ] 支持 JSON 序列化

预估工时：4h" \
    "module:protocol,P0,enhancement" \
    "$MILESTONE"
((ISSUE_COUNT++))

create_issue \
    "[Protocol] 实现 OAuth 2.0 Authorization Code" \
    "## 任务描述

实现 OAuth 2.0 Authorization Code Flow。

## 验收标准

- [ ] 授权码只能使用一次
- [ ] 过期授权码无法使用
- [ ] 防重放检测有效

预估工时：3h" \
    "module:protocol,P0,enhancement" \
    "$MILESTONE"
((ISSUE_COUNT++))

create_issue \
    "[Protocol] 实现 Token Introspection (RFC 7662)" \
    "## 任务描述

实现 OAuth 2.0 Token Introspection。

## 验收标准

- [ ] 符合 RFC 7662
- [ ] 有效性检查准确
- [ ] API 接口正常工作

预估工时：3h" \
    "module:protocol,P1,enhancement" \
    "$MILESTONE"
((ISSUE_COUNT++))

# 4. 配置系统
print_info "创建配置系统 Issues..."

create_issue \
    "[Config] 实现配置文件解析" \
    "## 任务描述

实现 YAML 格式配置文件解析。

## 验收标准

- [ ] YAML 解析正确
- [ ] 配置验证有效
- [ ] 默认配置可用

预估工时：3h" \
    "module:config,P0,enhancement" \
    "$MILESTONE"
((ISSUE_COUNT++))

create_issue \
    "[Config] 支持环境变量配置" \
    "## 任务描述

支持通过环境变量覆盖配置。

## 验收标准

- [ ] 环境变量正常工作
- [ ] 优先级正确（环境变量 > 配置文件 > 默认值）

预估工时：2h" \
    "module:config,P1,enhancement" \
    "$MILESTONE"
((ISSUE_COUNT++))

# 5. CLI 工具
print_info "创建 CLI 工具 Issues..."

create_issue \
    "[CLI] 实现 tokenginx-server 主程序" \
    "## 任务描述

实现服务器主程序，整合所有核心模块。

## 验收标准

- [ ] 命令行参数正常工作
- [ ] 优雅关闭无数据丢失
- [ ] 可使用 redis-cli 连接

预估工时：6h" \
    "module:cli,P0,enhancement" \
    "$MILESTONE"
((ISSUE_COUNT++))

create_issue \
    "[CLI] 实现 tokenginx-client 客户端工具" \
    "## 任务描述

实现命令行客户端工具。

## 验收标准

- [ ] 可连接服务器
- [ ] REPL 正常工作
- [ ] 命令执行正确

预估工时：4h" \
    "module:cli,P1,enhancement" \
    "$MILESTONE"
((ISSUE_COUNT++))

# 6. 监控日志
print_info "创建监控日志 Issues..."

create_issue \
    "[Monitoring] 实现结构化日志" \
    "## 任务描述

实现结构化日志系统。

## 验收标准

- [ ] 日志输出正常
- [ ] JSON 格式正确
- [ ] 轮转策略有效

预估工时：2h" \
    "module:monitoring,P0,enhancement" \
    "$MILESTONE"
((ISSUE_COUNT++))

create_issue \
    "[Monitoring] 实现基础性能指标" \
    "## 任务描述

实现基础性能指标统计。

## 验收标准

- [ ] 统计指标准确
- [ ] INFO 命令输出正确
- [ ] 性能开销 < 1%

预估工时：2h" \
    "module:monitoring,P1,enhancement" \
    "$MILESTONE"
((ISSUE_COUNT++))

# 7. 文档和测试
print_info "创建文档和测试 Issues..."

create_issue \
    "[Docs] 完善 API 参考文档" \
    "## 任务描述

完善 API 参考文档，确保所有命令都有详细说明。

## 验收标准

- [ ] 文档覆盖所有命令
- [ ] 示例可执行

预估工时：2h" \
    "documentation,P1" \
    "$MILESTONE"
((ISSUE_COUNT++))

create_issue \
    "[Docs] 更新快速开始指南和 GoDoc" \
    "## 任务描述

更新快速开始指南和 GoDoc 代码注释。

## 验收标准

- [ ] 新手可按文档快速上手
- [ ] GoDoc 注释覆盖率 100%

预估工时：4h" \
    "documentation,P1" \
    "$MILESTONE"
((ISSUE_COUNT++))

create_issue \
    "[Test] 单元测试覆盖率达到 80%+" \
    "## 任务描述

为所有核心模块编写单元测试。

## 验收标准

- [ ] 覆盖率 > 80%
- [ ] go test -race 通过

预估工时：16h" \
    "test,P0" \
    "$MILESTONE"
((ISSUE_COUNT++))

create_issue \
    "[Test] 集成测试和性能测试" \
    "## 任务描述

编写集成测试和性能测试。

## 验收标准

- [ ] QPS > 100,000
- [ ] P99 < 1ms
- [ ] 无内存泄漏

预估工时：18h" \
    "test,P0" \
    "$MILESTONE"
((ISSUE_COUNT++))

# 总结
echo ""
echo -e "${GREEN}=====================================${NC}"
echo -e "${GREEN}     Issues 创建完成！${NC}"
echo -e "${GREEN}=====================================${NC}"
echo ""
echo -e "创建的 Issues 数量: ${YELLOW}$ISSUE_COUNT${NC}"
echo ""
echo "查看 Issues:"
echo "  https://github.com/$REPO_OWNER/$REPO_NAME/issues"
echo ""
echo "查看 Milestone:"
echo "  https://github.com/$REPO_OWNER/$REPO_NAME/milestone/$MILESTONE"
echo ""
print_success "完成！"
