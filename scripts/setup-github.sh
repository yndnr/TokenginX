#!/bin/bash
# TokenginX GitHub 仓库初始化脚本
#
# 本脚本将帮助你：
# 1. 初始化本地 Git 仓库
# 2. 创建 .gitignore 文件
# 3. 连接到 GitHub 远程仓库
# 4. 推送代码到 GitHub
#
# 使用方法：
#   chmod +x scripts/setup-github.sh
#   ./scripts/setup-github.sh

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 打印带颜色的消息
print_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# 打印标题
print_header() {
    echo ""
    echo -e "${GREEN}=====================================${NC}"
    echo -e "${GREEN}  TokenginX GitHub 仓库初始化${NC}"
    echo -e "${GREEN}=====================================${NC}"
    echo ""
}

# 检查命令是否存在
check_command() {
    if ! command -v $1 &> /dev/null; then
        print_error "$1 未安装，请先安装"
        echo "安装方法: https://git-scm.com/downloads"
        exit 1
    fi
}

# 检查是否在项目根目录
check_project_root() {
    if [ ! -f "claude.md" ] || [ ! -f "readme.md" ]; then
        print_error "请在项目根目录运行此脚本"
        exit 1
    fi
}

# 获取用户输入（带默认值）
get_input() {
    local prompt="$1"
    local default="$2"
    local var_name="$3"

    if [ -n "$default" ]; then
        read -p "$(echo -e ${BLUE}${prompt}${NC} [默认: ${YELLOW}${default}${NC}]: )" input
        eval $var_name=\"${input:-$default}\"
    else
        read -p "$(echo -e ${BLUE}${prompt}${NC}: )" input
        eval $var_name=\"$input\"
    fi
}

# 获取密码输入（隐藏输入）
get_password() {
    local prompt="$1"
    local var_name="$2"

    read -sp "$(echo -e ${BLUE}${prompt}${NC}: )" input
    echo ""
    eval $var_name=\"$input\"
}

# 确认操作
confirm() {
    local prompt="$1"
    read -p "$(echo -e ${YELLOW}${prompt}${NC} [y/N]: )" response
    case "$response" in
        [yY][eE][sS]|[yY])
            return 0
            ;;
        *)
            return 1
            ;;
    esac
}

# ============================================
# 主程序开始
# ============================================

print_header

# 1. 检查环境
print_info "检查环境..."
check_command git
check_project_root
print_success "环境检查通过"

# 2. 收集用户信息
echo ""
echo -e "${GREEN}步骤 1/5: 配置 Git 用户信息${NC}"
echo ""

get_input "请输入你的 GitHub 用户名" "" GITHUB_USERNAME
get_input "请输入你的 Git 邮箱" "" GIT_EMAIL
get_input "请输入你的 Git 用户名" "$GITHUB_USERNAME" GIT_NAME

# 3. 收集仓库信息
echo ""
echo -e "${GREEN}步骤 2/5: 配置 GitHub 仓库信息${NC}"
echo ""

get_input "请输入 GitHub 仓库所有者（用户名或组织名）" "$GITHUB_USERNAME" REPO_OWNER
get_input "请输入仓库名称" "tokenginx" REPO_NAME
get_input "选择仓库可见性 (public/private)" "public" REPO_VISIBILITY

REPO_URL="https://github.com/${REPO_OWNER}/${REPO_NAME}.git"

# 4. 选择认证方式
echo ""
echo -e "${GREEN}步骤 3/5: 选择认证方式${NC}"
echo ""
echo "GitHub 认证方式："
echo "  1) HTTPS (Personal Access Token) - 推荐"
echo "  2) SSH (SSH Key)"
echo ""

get_input "请选择认证方式 (1/2)" "1" AUTH_METHOD

if [ "$AUTH_METHOD" = "1" ]; then
    echo ""
    print_info "使用 HTTPS 认证需要 Personal Access Token"
    print_info "获取 Token: https://github.com/settings/tokens"
    print_info "需要的权限: repo (完整仓库权限)"
    echo ""
    get_password "请输入你的 Personal Access Token" GITHUB_TOKEN

    # 使用 token 构造 URL
    REPO_URL_WITH_AUTH="https://${GITHUB_USERNAME}:${GITHUB_TOKEN}@github.com/${REPO_OWNER}/${REPO_NAME}.git"
elif [ "$AUTH_METHOD" = "2" ]; then
    REPO_URL="git@github.com:${REPO_OWNER}/${REPO_NAME}.git"

    print_info "使用 SSH 认证"
    print_info "确保你已经添加 SSH Key 到 GitHub"
    print_info "查看 SSH Key: cat ~/.ssh/id_rsa.pub"
    print_info "添加 SSH Key: https://github.com/settings/keys"
    echo ""

    if ! confirm "是否已经配置好 SSH Key？"; then
        print_error "请先配置 SSH Key，然后重新运行脚本"
        exit 1
    fi
else
    print_error "无效的选择"
    exit 1
fi

# 5. 显示配置摘要
echo ""
echo -e "${GREEN}步骤 4/5: 确认配置${NC}"
echo ""
echo "配置摘要："
echo "  Git 用户名: $GIT_NAME"
echo "  Git 邮箱: $GIT_EMAIL"
echo "  GitHub 用户名: $GITHUB_USERNAME"
echo "  仓库所有者: $REPO_OWNER"
echo "  仓库名称: $REPO_NAME"
echo "  仓库可见性: $REPO_VISIBILITY"
echo "  仓库 URL: $REPO_URL"
if [ "$AUTH_METHOD" = "1" ]; then
    echo "  认证方式: HTTPS (Personal Access Token)"
else
    echo "  认证方式: SSH"
fi
echo ""

if ! confirm "确认以上信息是否正确？"; then
    print_error "操作已取消"
    exit 1
fi

# 6. 初始化 Git 仓库
echo ""
echo -e "${GREEN}步骤 5/5: 初始化 Git 仓库${NC}"
echo ""

# 检查是否已经是 Git 仓库
if [ -d ".git" ]; then
    print_warning "已存在 .git 目录"

    if confirm "是否要删除现有 Git 仓库并重新初始化？"; then
        print_info "删除现有 .git 目录..."
        rm -rf .git
        print_success ".git 目录已删除"
    else
        print_info "保留现有 Git 仓库，继续配置..."
    fi
fi

# 初始化 Git 仓库（如果需要）
if [ ! -d ".git" ]; then
    print_info "初始化 Git 仓库..."
    git init
    print_success "Git 仓库初始化完成"
fi

# 配置 Git 用户信息
print_info "配置 Git 用户信息..."
git config user.name "$GIT_NAME"
git config user.email "$GIT_EMAIL"
print_success "Git 用户信息配置完成"

# 创建或更新 .gitignore
print_info "检查 .gitignore 文件..."
if [ ! -f ".gitignore" ]; then
    print_info "创建 .gitignore 文件..."
    cat > .gitignore << 'EOF'
# Binaries for programs and plugins
*.exe
*.exe~
*.dll
*.so
*.dylib
bin/
*.test
*.out

# Go workspace file
go.work

# Dependency directories
vendor/

# Test coverage
*.coverprofile
coverage.out
coverage.html

# IDE and editor files
.idea/
.vscode/
*.swp
*.swo
*~
.DS_Store

# Environment and config files
.env
.env.local
*.local.yaml
*.local.yml

# Log files
*.log
logs/

# Data and cache
data/
*.db
*.sqlite
*.mmap

# Temporary files
tmp/
temp/
*.tmp

# Build artifacts
dist/
build/
*.tar.gz
*.zip

# Debug files
debug
__debug_bin

# OS specific
Thumbs.db
EOF
    print_success ".gitignore 文件已创建"
else
    print_success ".gitignore 文件已存在"
fi

# 设置默认分支为 main
print_info "设置默认分支为 main..."
git branch -M main
print_success "默认分支已设置为 main"

# 添加所有文件
print_info "添加文件到 Git..."
git add .
print_success "文件已添加"

# 创建初始提交
print_info "创建初始提交..."
if git rev-parse HEAD >/dev/null 2>&1; then
    print_warning "已存在提交历史"
else
    git commit -m "chore: 初始化 TokenginX 项目

- 添加项目文档和配置
- 添加存储引擎、传输层、协议层设计
- 添加部署配置（Docker, Podman, Kubernetes）
- 添加任务管理和路线图文档

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude <noreply@anthropic.com>"
    print_success "初始提交已创建"
fi

# 添加远程仓库
print_info "配置远程仓库..."

# 检查是否已存在 origin
if git remote | grep -q "^origin$"; then
    print_warning "远程仓库 'origin' 已存在"

    if confirm "是否要更新 origin 的 URL？"; then
        git remote remove origin
        print_info "已删除旧的 origin"
    else
        print_info "保留现有 origin"
    fi
fi

# 添加 origin（如果不存在）
if ! git remote | grep -q "^origin$"; then
    if [ "$AUTH_METHOD" = "1" ]; then
        git remote add origin "$REPO_URL_WITH_AUTH"
    else
        git remote add origin "$REPO_URL"
    fi
    print_success "远程仓库已添加"
fi

# 显示远程仓库（隐藏 token）
DISPLAY_URL=$(git remote get-url origin | sed 's/:.*@/:***@/')
print_info "远程仓库 URL: $DISPLAY_URL"

# 询问是否创建 GitHub 仓库
echo ""
print_warning "注意：在推送代码前，请确保 GitHub 仓库已创建"
echo ""
echo "如果仓库不存在，请访问以下链接创建："
echo "  https://github.com/new"
echo ""
echo "创建仓库时："
echo "  - 仓库名称: $REPO_NAME"
echo "  - 可见性: $REPO_VISIBILITY"
echo "  - 不要初始化 README、.gitignore 或 LICENSE（本地已有）"
echo ""

if ! confirm "GitHub 仓库是否已创建？"; then
    print_warning "请先创建 GitHub 仓库，然后重新运行脚本或手动推送代码"
    echo ""
    echo "手动推送命令："
    echo "  git push -u origin main"
    exit 0
fi

# 推送到 GitHub
echo ""
print_info "推送代码到 GitHub..."

if confirm "是否立即推送代码到 GitHub？"; then
    # 检查是否有 upstream
    if git rev-parse --abbrev-ref --symbolic-full-name @{u} >/dev/null 2>&1; then
        print_info "推送到远程仓库..."
        git push
    else
        print_info "首次推送，设置 upstream..."
        git push -u origin main
    fi

    print_success "代码已推送到 GitHub！"

    echo ""
    echo -e "${GREEN}=====================================${NC}"
    echo -e "${GREEN}     GitHub 仓库初始化完成！${NC}"
    echo -e "${GREEN}=====================================${NC}"
    echo ""
    echo "仓库地址: https://github.com/${REPO_OWNER}/${REPO_NAME}"
    echo ""
    echo "下一步："
    echo "  1. 访问仓库查看代码"
    echo "  2. 运行 ./scripts/create-v0.1.0-issues.sh 创建开发任务"
    echo "  3. 配置 GitHub Projects 看板"
    echo "  4. 开始开发！"
    echo ""
else
    print_warning "跳过推送，你可以稍后手动推送："
    echo ""
    echo "  git push -u origin main"
    echo ""
fi

# 保存配置到文件（供后续脚本使用）
print_info "保存配置到 .github-config（仅本地）..."
cat > .github-config << EOF
GITHUB_USERNAME=$GITHUB_USERNAME
REPO_OWNER=$REPO_OWNER
REPO_NAME=$REPO_NAME
REPO_URL=$REPO_URL
EOF

# 确保 .github-config 在 .gitignore 中
if ! grep -q "^\.github-config$" .gitignore; then
    echo ".github-config" >> .gitignore
    print_info "已将 .github-config 添加到 .gitignore"
fi

print_success "配置已保存到 .github-config"

echo ""
print_success "所有操作完成！"
