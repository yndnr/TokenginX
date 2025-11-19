#!/bin/bash
# 快速创建 v0.1.0 Issues
# 使用保存的仓库配置

set -e

# 加载配置
if [ -f ".github-config" ]; then
    source .github-config
else
    echo "错误: 找不到 .github-config 文件"
    echo "请先运行 ./scripts/setup-github.sh"
    exit 1
fi

# 检查是否提供了 Token
if [ -z "$GITHUB_TOKEN" ]; then
    echo ""
    echo "请输入你的 GitHub Personal Access Token:"
    echo "（获取地址: https://github.com/settings/tokens）"
    echo "需要的权限: repo (完整仓库权限)"
    echo ""
    read -sp "Token: " GITHUB_TOKEN
    echo ""
fi

# 导出环境变量
export GITHUB_TOKEN
export REPO_OWNER
export REPO_NAME

echo ""
echo "仓库信息:"
echo "  所有者: $REPO_OWNER"
echo "  仓库名: $REPO_NAME"
echo "  Token: ${GITHUB_TOKEN:0:8}..."
echo ""

read -p "确认创建 20 个 v0.1.0 Issues? [y/N]: " confirm
if [[ ! "$confirm" =~ ^[Yy]$ ]]; then
    echo "已取消"
    exit 0
fi

# 运行创建脚本
./scripts/create-issues-api.sh
