#!/bin/bash
# Script to create GitHub Issues for v0.1.0 MVP tasks
#
# Prerequisites:
# - GitHub CLI (gh) installed: https://cli.github.com/
# - Authenticated: gh auth login
# - Repository must exist
#
# Usage:
#   chmod +x scripts/create-v0.1.0-issues.sh
#   ./scripts/create-v0.1.0-issues.sh

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}===== Creating GitHub Issues for v0.1.0 MVP =====${NC}"

# Check if gh is installed
if ! command -v gh &> /dev/null; then
    echo -e "${RED}Error: GitHub CLI (gh) is not installed${NC}"
    echo "Install from: https://cli.github.com/"
    exit 1
fi

# Check if authenticated
if ! gh auth status &> /dev/null; then
    echo -e "${RED}Error: Not authenticated with GitHub${NC}"
    echo "Run: gh auth login"
    exit 1
fi

# Milestone name
MILESTONE="v0.1.0"

# Check if milestone exists, create if not
if ! gh api repos/:owner/:repo/milestones --jq ".[] | select(.title==\"$MILESTONE\")" | grep -q "$MILESTONE"; then
    echo -e "${YELLOW}Creating milestone: $MILESTONE${NC}"
    gh api repos/:owner/:repo/milestones -X POST -f title="$MILESTONE" -f description="MVP - 最小可行产品" -f due_on="2026-01-31T23:59:59Z"
fi

echo -e "${GREEN}Milestone '$MILESTONE' is ready${NC}"
echo ""

# ===== 1. Storage Engine (存储引擎) =====
echo -e "${GREEN}[1/7] Creating Storage Engine issues...${NC}"

gh issue create \
  --title "[Storage] 实现 ShardedMap 基础结构" \
  --body "## 任务描述

实现 256 分片的核心存储结构 \`ShardedMap\`，这是 TokenginX 存储引擎的核心组件。

## 实现要点

- 定义 \`ShardedMap\` 结构体（256 个分片）
- 实现哈希分片函数（一致性哈希）
- 实现 Set/Get/Delete/Exists 方法
- 每个分片使用 \`sync.RWMutex\` 保证并发安全
- 支持基础统计（Len 方法）

## 验收标准

- [ ] 所有单元测试通过（覆盖率 > 90%）
- [ ] 无数据竞争（\`go test -race\` 通过）
- [ ] 性能达标：单分片 GET > 1M ops/s
- [ ] 代码符合 Go 规范（\`go vet\` 通过）

## 相关文档

- [开发任务清单](docs/tasks/v0.1.0-dev-tasks.md#11-shardedmap-实现6h)
- [测试任务清单](docs/tasks/v0.1.0-test-tasks.md#shardedmap-测试3h)

预估工时：6h" \
  --label "module:storage,P0,enhancement" \
  --milestone "$MILESTONE"

gh issue create \
  --title "[Storage] 实现 TTL 管理机制" \
  --body "## 任务描述

实现会话过期的 TTL（Time-To-Live）管理机制，包括惰性删除和定期清理。

## 实现要点

- 在 \`Get()\` 时检查过期（惰性删除）
- 后台 Goroutine 定期扫描过期键（定期清理）
- 实现 \`TTL(key)\` 获取剩余时间
- 实现 \`Expire(key, ttl)\` 更新 TTL
- 清理策略可配置（清理间隔、每次清理数量）

## 验收标准

- [ ] 惰性删除正常工作
- [ ] 定期清理不影响性能（CPU < 5%）
- [ ] 过期键能及时清理（延迟 < 1s）
- [ ] 单元测试覆盖率 > 85%

## 相关文档

- [开发任务清单](docs/tasks/v0.1.0-dev-tasks.md#12-ttl-管理4h)
- [测试任务清单](docs/tasks/v0.1.0-test-tasks.md#ttl-管理测试2h)

## 依赖

- #1 ShardedMap 基础结构

预估工时：4h" \
  --label "module:storage,P0,enhancement" \
  --milestone "$MILESTONE"

gh issue create \
  --title "[Storage] 实现基础 LRU 缓存" \
  --body "## 任务描述

实现简化版的 LRU（Least Recently Used）缓存淘汰策略，为 v0.5.0 的 W-TinyLFU 做准备。

## 实现要点

- 使用链表 + 哈希表实现 LRU
- 支持容量限制
- 访问时移动到链表头
- 容量满时淘汰链表尾
- 集成到 \`ShardedMap\`（每个分片可选 LRU）

## 验收标准

- [ ] LRU 淘汰策略正确
- [ ] 缓存命中率 > 90%（模拟测试）
- [ ] 性能开销 < 10%
- [ ] 单元测试通过

## 相关文档

- [开发任务清单](docs/tasks/v0.1.0-dev-tasks.md#13-基础-lru-缓存4h)
- [测试任务清单](docs/tasks/v0.1.0-test-tasks.md#lru-缓存测试1h)

## 依赖

- #1 ShardedMap 基础结构

预估工时：4h" \
  --label "module:storage,P1,enhancement" \
  --milestone "$MILESTONE"

echo -e "${GREEN}Storage Engine issues created (3/3)${NC}"
echo ""

# ===== 2. Transport Layer (传输层) =====
echo -e "${GREEN}[2/7] Creating Transport Layer issues...${NC}"

gh issue create \
  --title "[Transport] 实现 RESP 协议解析器" \
  --body "## 任务描述

实现 Redis RESP（REdis Serialization Protocol）协议的解析器和序列化器，确保兼容 Redis 客户端。

## 实现要点

- 定义 RESP 数据类型（Simple String, Error, Integer, Bulk String, Array）
- 实现 \`ParseRESP(reader)\` 解析函数
- 实现 \`WriteRESP(writer, value)\` 序列化函数
- 处理边界情况（大文件、空值、错误格式）
- 零拷贝优化（使用 \`[]byte\` 避免字符串转换）

## 验收标准

- [ ] 兼容 Redis RESP 协议
- [ ] 解析速度 > 100K ops/s
- [ ] 无内存泄漏
- [ ] 单元测试覆盖率 > 95%
- [ ] 支持所有 RESP 类型

## 相关文档

- [开发任务清单](docs/tasks/v0.1.0-dev-tasks.md#21-resp-协议解析器6h)
- [测试任务清单](docs/tasks/v0.1.0-test-tasks.md#resp-协议测试3h)
- [TCP RESP API 参考](docs/reference/tcp-resp-api.md)

预估工时：6h" \
  --label "module:transport,P0,enhancement" \
  --milestone "$MILESTONE"

gh issue create \
  --title "[Transport] 实现 TCP 服务器" \
  --body "## 任务描述

实现 TCP 服务器，监听端口并处理客户端连接，支持 10,000+ 并发连接。

## 实现要点

- 实现 \`TCPServer\` 结构体
- 实现 \`Start()\` 启动服务器
- 实现 \`Stop()\` 优雅关闭（等待现有连接完成）
- 实现 \`handleConnection(conn)\` 处理单个连接
- 连接超时（读超时、写超时、空闲超时）
- 连接池管理（最大连接数限制）
- 连接统计（当前连接数、总连接数）

## 验收标准

- [ ] 支持 10,000+ 并发连接
- [ ] 优雅关闭无连接丢失
- [ ] 连接超时正常工作
- [ ] 可使用 \`redis-cli\` 连接测试

## 相关文档

- [开发任务清单](docs/tasks/v0.1.0-dev-tasks.md#22-tcp-服务器6h)
- [测试任务清单](docs/tasks/v0.1.0-test-tasks.md#21-tcp-服务器集成测试6h)

## 依赖

- #4 RESP 协议解析器

预估工时：6h" \
  --label "module:transport,P0,enhancement" \
  --milestone "$MILESTONE"

gh issue create \
  --title "[Transport] 实现基础命令处理器" \
  --body "## 任务描述

实现基础的 Redis 命令处理器，支持 GET、SET、DEL 等常用命令。

## 实现要点

- 定义 \`CommandHandler\` 结构体
- 实现命令注册机制 \`RegisterCommand(name, fn)\`
- 实现以下命令：
  - \`GET key\`
  - \`SET key value [EX seconds]\`
  - \`DEL key [key ...]\`
  - \`EXISTS key\`
  - \`TTL key\`
  - \`EXPIRE key seconds\`
  - \`PING\`
  - \`ECHO message\`
- 错误处理（参数错误、类型错误）

## 验收标准

- [ ] 所有命令符合 Redis 语义
- [ ] 错误信息清晰
- [ ] 参数验证完整
- [ ] 每个命令有独立测试

## 相关文档

- [开发任务清单](docs/tasks/v0.1.0-dev-tasks.md#23-命令处理器4h)
- [测试任务清单](docs/tasks/v0.1.0-test-tasks.md#命令处理器测试3h)
- [TCP RESP API 参考](docs/reference/tcp-resp-api.md)

## 依赖

- #1 ShardedMap 基础结构
- #4 RESP 协议解析器

预估工时：4h" \
  --label "module:transport,P0,enhancement" \
  --milestone "$MILESTONE"

gh issue create \
  --title "[Transport] 实现连接管理和监控" \
  --body "## 任务描述

实现连接管理功能，包括连接列表、连接关闭、服务器统计等。

## 实现要点

- 实现 \`CLIENT LIST\` 命令（列出所有连接）
- 实现 \`CLIENT KILL ip:port\` 命令（关闭指定连接）
- 实现 \`INFO\` 命令（服务器统计信息）
- 连接心跳检测
- 连接速率限制（每个 IP 最大连接数）
- 监控指标（连接数、命令数、错误数）

## 验收标准

- [ ] CLIENT 命令正常工作
- [ ] INFO 输出格式正确（符合 Redis 格式）
- [ ] 速率限制有效
- [ ] 监控指标准确

## 相关文档

- [开发任务清单](docs/tasks/v0.1.0-dev-tasks.md#24-连接管理4h)

## 依赖

- #5 TCP 服务器

预估工时：4h" \
  --label "module:transport,P1,enhancement" \
  --milestone "$MILESTONE"

echo -e "${GREEN}Transport Layer issues created (4/4)${NC}"
echo ""

# ===== 3. OAuth 2.0 Protocol (协议层) =====
echo -e "${GREEN}[3/7] Creating OAuth 2.0 Protocol issues...${NC}"

gh issue create \
  --title "[Protocol] 实现 OAuth 2.0 Token 存储" \
  --body "## 任务描述

实现 OAuth 2.0 Token 的存储和管理，包括 Access Token 和 Refresh Token。

## 实现要点

- 定义 \`OAuthToken\` 数据结构
- 实现 \`StoreAccessToken(token, ttl)\` 存储访问令牌
- 实现 \`GetAccessToken(accessToken)\` 获取访问令牌
- 实现 \`DeleteAccessToken(accessToken)\` 删除访问令牌
- 实现 \`StoreRefreshToken(token, ttl)\` 存储刷新令牌
- 实现 \`GetRefreshToken(refreshToken)\` 获取刷新令牌
- Token 自动过期（基于 TTL）
- Token 序列化/反序列化（JSON）

## 验收标准

- [ ] Token 存储和检索正确
- [ ] TTL 自动过期生效
- [ ] 支持 JSON 序列化
- [ ] 单元测试覆盖率 > 85%

## 相关文档

- [开发任务清单](docs/tasks/v0.1.0-dev-tasks.md#31-token-存储4h)
- [测试任务清单](docs/tasks/v0.1.0-test-tasks.md#token-存储测试2h)
- [OAuth 2.0 集成指南](docs/protocols/oauth.md)

## 依赖

- #1 ShardedMap 基础结构

预估工时：4h" \
  --label "module:protocol,P0,enhancement" \
  --milestone "$MILESTONE"

gh issue create \
  --title "[Protocol] 实现 OAuth 2.0 Authorization Code 流程" \
  --body "## 任务描述

实现 OAuth 2.0 Authorization Code Flow 的授权码存储和管理。

## 实现要点

- 定义 \`AuthorizationCode\` 数据结构
- 实现 \`StoreAuthCode(code, ttl)\` 存储授权码
- 实现 \`GetAuthCode(code)\` 获取授权码
- 实现 \`DeleteAuthCode(code)\` 删除授权码（一次性使用）
- 授权码过期（默认 10 分钟）
- 防重放检测（授权码只能使用一次）

## 验收标准

- [ ] 授权码只能使用一次
- [ ] 过期授权码无法使用
- [ ] 防重放检测有效
- [ ] 单元测试通过

## 相关文档

- [开发任务清单](docs/tasks/v0.1.0-dev-tasks.md#32-authorization-code3h)
- [测试任务清单](docs/tasks/v0.1.0-test-tasks.md#authorization-code-测试1h)
- [OAuth 2.0 集成指南](docs/protocols/oauth.md)

## 依赖

- #1 ShardedMap 基础结构

预估工时：3h" \
  --label "module:protocol,P0,enhancement" \
  --milestone "$MILESTONE"

gh issue create \
  --title "[Protocol] 实现 Token Introspection (RFC 7662)" \
  --body "## 任务描述

实现 OAuth 2.0 Token Introspection 功能，用于验证 Token 有效性。

## 实现要点

- 实现 \`IntrospectToken(token)\` 检查令牌有效性
- 返回 \`IntrospectionResponse\` 结构（符合 RFC 7662）
- Token 有效性检查（是否存在、是否过期）
- Token 元数据返回（scope, client_id, username, exp, iat）
- 添加 HTTP/REST API 接口（POST /oauth/introspect）

## 验收标准

- [ ] Introspection 响应符合 RFC 7662
- [ ] 有效性检查准确
- [ ] API 接口正常工作
- [ ] 单元测试覆盖率 > 90%

## 相关文档

- [开发任务清单](docs/tasks/v0.1.0-dev-tasks.md#33-token-introspection3h)
- [测试任务清单](docs/tasks/v0.1.0-test-tasks.md#token-introspection-测试1h)
- [OAuth 2.0 集成指南](docs/protocols/oauth.md)
- [RFC 7662](https://tools.ietf.org/html/rfc7662)

## 依赖

- #8 Token 存储

预估工时：3h" \
  --label "module:protocol,P1,enhancement" \
  --milestone "$MILESTONE"

echo -e "${GREEN}OAuth 2.0 Protocol issues created (3/3)${NC}"
echo ""

# ===== 4. Configuration System (配置系统) =====
echo -e "${GREEN}[4/7] Creating Configuration System issues...${NC}"

gh issue create \
  --title "[Config] 实现配置文件解析" \
  --body "## 任务描述

实现 YAML 格式的配置文件解析，支持所有核心配置项。

## 实现要点

- 定义配置结构体（Server, Storage, Cache, OAuth, Security）
- 实现 \`LoadConfig(path)\` 加载 YAML 配置
- 配置验证（必填项检查、范围检查）
- 默认配置（无配置文件时使用默认值）
- 配置文档完善

## 验收标准

- [ ] YAML 解析正确
- [ ] 配置验证有效
- [ ] 默认配置可用
- [ ] 单元测试通过

## 相关文档

- [开发任务清单](docs/tasks/v0.1.0-dev-tasks.md#41-配置文件解析3h)
- [配置参考文档](docs/reference/configuration.md)
- [配置示例](config/config.example.yaml)

预估工时：3h" \
  --label "module:config,P0,enhancement" \
  --milestone "$MILESTONE"

gh issue create \
  --title "[Config] 支持环境变量配置" \
  --body "## 任务描述

支持通过环境变量覆盖配置文件，遵循 12-factor app 原则。

## 实现要点

- 实现环境变量命名规则（\`TOKENGINX_MODULE_KEY\`）
- 实现 \`LoadFromEnv()\` 从环境变量加载
- 优先级：环境变量 > 配置文件 > 默认值
- 环境变量文档完善

## 验收标准

- [ ] 环境变量正常工作
- [ ] 优先级正确
- [ ] 文档完整
- [ ] 单元测试通过

## 相关文档

- [开发任务清单](docs/tasks/v0.1.0-dev-tasks.md#42-环境变量支持2h)
- [配置参考文档](docs/reference/configuration.md)

## 依赖

- #11 配置文件解析

预估工时：2h" \
  --label "module:config,P1,enhancement" \
  --milestone "$MILESTONE"

echo -e "${GREEN}Configuration System issues created (2/2)${NC}"
echo ""

# ===== 5. CLI Tools (CLI 工具) =====
echo -e "${GREEN}[5/7] Creating CLI Tools issues...${NC}"

gh issue create \
  --title "[CLI] 实现 tokenginx-server 主程序" \
  --body "## 任务描述

实现服务器主程序，整合所有核心模块，提供完整的启动和关闭流程。

## 实现要点

- 实现 \`main()\` 函数（\`cmd/server/main.go\`）
- 命令行参数解析（\`-config\`, \`-addr\`, \`-version\`, \`-help\`）
- 版本信息和帮助信息
- 优雅启动和关闭
- 信号处理（SIGINT, SIGTERM）
- 日志初始化

## 验收标准

- [ ] 命令行参数正常工作
- [ ] 优雅关闭无数据丢失
- [ ] 日志输出正确
- [ ] 可使用 \`redis-cli\` 连接

## 相关文档

- [开发任务清单](docs/tasks/v0.1.0-dev-tasks.md#51-tokenginx-server6h)
- [快速开始指南](docs/quickstart/)

## 依赖

- #1 ShardedMap
- #5 TCP 服务器
- #6 命令处理器
- #11 配置文件解析

预估工时：6h" \
  --label "module:cli,P0,enhancement" \
  --milestone "$MILESTONE"

gh issue create \
  --title "[CLI] 实现 tokenginx-client 客户端工具" \
  --body "## 任务描述

实现命令行客户端工具，用于连接服务器、执行命令和调试。

## 实现要点

- 实现 \`main()\` 函数（\`cmd/client/main.go\`）
- 连接服务器（\`-server localhost:6380\`）
- 交互式 REPL（Read-Eval-Print Loop）
- 单命令执行（\`-c \"GET key\"\`）
- 批量执行（从文件读取命令）
- 输出格式化

## 验收标准

- [ ] 可连接服务器
- [ ] REPL 正常工作
- [ ] 命令执行正确
- [ ] 文档完整

## 相关文档

- [开发任务清单](docs/tasks/v0.1.0-dev-tasks.md#52-tokenginx-client4h)
- [快速开始指南](docs/quickstart/)

## 依赖

- #4 RESP 协议解析器

预估工时：4h" \
  --label "module:cli,P1,enhancement" \
  --milestone "$MILESTONE"

echo -e "${GREEN}CLI Tools issues created (2/2)${NC}"
echo ""

# ===== 6. Monitoring & Logging (监控和日志) =====
echo -e "${GREEN}[6/7] Creating Monitoring & Logging issues...${NC}"

gh issue create \
  --title "[Monitoring] 实现结构化日志" \
  --body "## 任务描述

实现结构化日志系统，支持多级别、多格式、日志轮转。

## 实现要点

- 集成日志库（推荐 \`zap\` 或 \`logrus\`）
- 日志级别（DEBUG, INFO, WARN, ERROR）
- 日志格式（JSON 或 Text）
- 日志输出（stdout 或文件）
- 日志轮转（按大小或时间）
- 关键操作日志（启动、关闭、错误）

## 验收标准

- [ ] 日志输出正常
- [ ] JSON 格式正确
- [ ] 轮转策略有效
- [ ] 性能开销 < 1%

## 相关文档

- [开发任务清单](docs/tasks/v0.1.0-dev-tasks.md#61-日志系统2h)

预估工时：2h" \
  --label "module:monitoring,P0,enhancement" \
  --milestone "$MILESTONE"

gh issue create \
  --title "[Monitoring] 实现基础性能指标" \
  --body "## 任务描述

实现基础的性能指标统计，包括连接数、命令数、QPS、延迟等。

## 实现要点

- 实现 \`Stats\` 结构体（统计指标）
- 实现 \`INFO stats\` 命令（输出统计信息）
- QPS 计算（每秒命令数）
- 延迟统计（P50, P99, P999）
- 内存使用统计
- 指标导出（为后续 Prometheus 集成做准备）

## 验收标准

- [ ] 统计指标准确
- [ ] INFO 命令输出正确
- [ ] 性能开销 < 1%

## 相关文档

- [开发任务清单](docs/tasks/v0.1.0-dev-tasks.md#62-性能指标2h)

## 依赖

- #6 命令处理器

预估工时：2h" \
  --label "module:monitoring,P1,enhancement" \
  --milestone "$MILESTONE"

echo -e "${GREEN}Monitoring & Logging issues created (2/2)${NC}"
echo ""

# ===== 7. Documentation & Testing (文档和测试) =====
echo -e "${GREEN}[7/7] Creating Documentation & Testing issues...${NC}"

gh issue create \
  --title "[Docs] 完善 API 参考文档" \
  --body "## 任务描述

完善 API 参考文档，确保所有命令都有详细说明和示例。

## 实现要点

- 完善 \`docs/reference/tcp-resp-api.md\`（所有命令）
- 添加命令示例
- 添加错误码说明
- 添加性能建议

## 验收标准

- [ ] 文档覆盖所有命令
- [ ] 示例可执行
- [ ] 格式统一

## 相关文档

- [开发任务清单](docs/tasks/v0.1.0-dev-tasks.md#71-api-文档2h)
- [TCP RESP API 参考](docs/reference/tcp-resp-api.md)

## 依赖

- #6 命令处理器

预估工时：2h" \
  --label "documentation,P1" \
  --milestone "$MILESTONE"

gh issue create \
  --title "[Docs] 更新快速开始指南和 GoDoc 注释" \
  --body "## 任务描述

更新快速开始指南，确保新手可以快速上手，并完善 GoDoc 代码注释。

## 实现要点

### 快速开始指南
- 更新 \`readme.md\`（v0.1.0 功能）
- 更新安装说明
- 更新快速示例
- 添加故障排查

### GoDoc 注释
- 为所有导出函数添加 GoDoc 注释
- 为所有导出类型添加注释
- 为所有包添加包级注释
- 添加使用示例（Example 测试）

## 验收标准

- [ ] 新手可按文档快速上手
- [ ] 示例可正常运行
- [ ] GoDoc 注释覆盖率 100%
- [ ] 示例代码可运行

## 相关文档

- [开发任务清单](docs/tasks/v0.1.0-dev-tasks.md#72-快速开始指南2h)
- [CLAUDE.md 注释规范](claude.md)

## 依赖

- 所有核心模块

预估工时：4h" \
  --label "documentation,P1" \
  --milestone "$MILESTONE"

gh issue create \
  --title "[Test] 单元测试覆盖率达到 80%+" \
  --body "## 任务描述

为所有核心模块编写单元测试，确保测试覆盖率达到 80% 以上。

## 测试模块

- 存储引擎（ShardedMap, TTL, LRU）
- 传输层（RESP 解析器, 命令处理器）
- 协议层（OAuth Token, AuthCode, Introspection）
- 配置系统
- 监控日志

## 验收标准

- [ ] 所有单元测试通过
- [ ] 覆盖率 > 80%（整体）
- [ ] 存储引擎覆盖率 > 90%
- [ ] 传输层覆盖率 > 90%
- [ ] 协议层覆盖率 > 85%
- [ ] \`go test -race\` 通过（无数据竞争）

## 相关文档

- [测试任务清单](docs/tasks/v0.1.0-test-tasks.md#1-单元测试16-小时)

预估工时：16h" \
  --label "test,P0" \
  --milestone "$MILESTONE"

gh issue create \
  --title "[Test] 集成测试和性能测试" \
  --body "## 任务描述

编写集成测试和性能测试，确保系统稳定性和性能达标。

## 测试内容

### 集成测试
- TCP 服务器集成测试
- OAuth 2.0 端到端测试
- CLI 工具集成测试

### 性能测试
- 基准测试（QPS > 100K, P99 < 1ms）
- 压力测试（10,000 并发连接）
- 长时间稳定性测试
- 内存泄漏检测

## 验收标准

- [ ] 所有集成测试通过
- [ ] 性能达标：QPS > 100,000
- [ ] 延迟达标：P99 < 1ms, P999 < 5ms
- [ ] 无内存泄漏
- [ ] 无 Goroutine 泄漏
- [ ] 长时间运行稳定（24h+）

## 相关文档

- [测试任务清单](docs/tasks/v0.1.0-test-tasks.md#2-集成测试12-小时)
- [测试任务清单](docs/tasks/v0.1.0-test-tasks.md#3-性能测试6-小时)

预估工时：18h" \
  --label "test,P0" \
  --milestone "$MILESTONE"

echo -e "${GREEN}Documentation & Testing issues created (4/4)${NC}"
echo ""

# Summary
echo -e "${GREEN}===== Summary =====${NC}"
echo -e "Total issues created: ${YELLOW}20${NC}"
echo ""
echo -e "${GREEN}Module breakdown:${NC}"
echo "  - Storage Engine:        3 issues"
echo "  - Transport Layer:       4 issues"
echo "  - OAuth 2.0 Protocol:    3 issues"
echo "  - Configuration System:  2 issues"
echo "  - CLI Tools:             2 issues"
echo "  - Monitoring & Logging:  2 issues"
echo "  - Documentation:         2 issues"
echo "  - Testing:               2 issues"
echo ""
echo -e "${GREEN}Priority breakdown:${NC}"
echo "  - P0 (Critical):  10 issues"
echo "  - P1 (Important):  9 issues"
echo "  - P2 (Nice-to-have): 1 issue"
echo ""
echo -e "${GREEN}Estimated total hours:${NC} 70h development + 34h testing = ${YELLOW}104h${NC}"
echo ""
echo -e "${GREEN}Next steps:${NC}"
echo "  1. Review issues on GitHub: gh issue list --milestone '$MILESTONE'"
echo "  2. Assign issues to team members"
echo "  3. Set up GitHub Projects board"
echo "  4. Start development!"
echo ""
echo -e "${GREEN}Done! All issues created successfully.${NC}"
