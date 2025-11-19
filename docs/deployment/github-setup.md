# GitHub 仓库初始化指南

本文档介绍如何将 TokenginX 项目连接到 GitHub。

## 快速开始

### 自动化脚本（推荐）

我们提供了一个交互式脚本来自动化整个过程：

```bash
# 运行 GitHub 初始化脚本
./scripts/setup-github.sh
```

脚本会引导你完成以下步骤：
1. 配置 Git 用户信息
2. 配置 GitHub 仓库信息
3. 选择认证方式（HTTPS 或 SSH）
4. 初始化本地 Git 仓库
5. 推送代码到 GitHub

### 手动步骤

如果你更喜欢手动操作，请按照以下步骤进行。

## 前置要求

- Git 已安装（`git --version` 检查）
- GitHub 账户
- 选择认证方式：
  - **HTTPS**：需要 Personal Access Token
  - **SSH**：需要配置 SSH Key

## 步骤 1: 创建 GitHub 仓库

1. 访问 https://github.com/new
2. 填写仓库信息：
   - **Repository name**: `tokenginx`
   - **Description**: 专为单点登录（SSO）优化的高性能会话存储系统
   - **Visibility**: Public 或 Private
   - **重要**: 不要勾选 "Add a README file"、"Add .gitignore" 或 "Choose a license"（本地已有）
3. 点击 "Create repository"

## 步骤 2: 配置认证

### 方式 1: HTTPS (Personal Access Token)

**推荐用于新手和临时访问**

1. 获取 Personal Access Token：
   - 访问 https://github.com/settings/tokens
   - 点击 "Generate new token" → "Generate new token (classic)"
   - 填写 Note: `TokenginX Development`
   - 选择 Expiration: 自定义（建议 90 天）
   - 勾选权限：
     - ✅ `repo` (完整仓库权限)
   - 点击 "Generate token"
   - **重要**: 复制 Token 并保存（只显示一次）

2. 使用 Token 克隆/推送：
   ```bash
   # 推送时会提示输入用户名和密码
   # Username: 你的 GitHub 用户名
   # Password: 粘贴 Personal Access Token（不是 GitHub 密码）
   ```

### 方式 2: SSH Key

**推荐用于长期开发**

1. 生成 SSH Key（如果没有）：
   ```bash
   ssh-keygen -t ed25519 -C "your_email@example.com"
   # 或使用 RSA
   ssh-keygen -t rsa -b 4096 -C "your_email@example.com"
   ```

2. 启动 SSH Agent：
   ```bash
   eval "$(ssh-agent -s)"
   ssh-add ~/.ssh/id_ed25519
   # 或 ssh-add ~/.ssh/id_rsa
   ```

3. 复制公钥：
   ```bash
   cat ~/.ssh/id_ed25519.pub
   # 或 cat ~/.ssh/id_rsa.pub
   ```

4. 添加 SSH Key 到 GitHub：
   - 访问 https://github.com/settings/keys
   - 点击 "New SSH key"
   - Title: `TokenginX Development`
   - Key: 粘贴公钥内容
   - 点击 "Add SSH key"

5. 测试连接：
   ```bash
   ssh -T git@github.com
   # 应该看到: Hi username! You've successfully authenticated...
   ```

## 步骤 3: 初始化本地仓库

```bash
# 进入项目目录
cd /home/yangsen/codes/tokenginx

# 初始化 Git 仓库（如果还没有）
git init

# 配置用户信息
git config user.name "Your Name"
git config user.email "your.email@example.com"

# 设置默认分支为 main
git branch -M main

# 添加所有文件
git add .

# 创建初始提交
git commit -m "chore: 初始化 TokenginX 项目

- 添加项目文档和配置
- 添加存储引擎、传输层、协议层设计
- 添加部署配置（Docker, Podman, Kubernetes）
- 添加任务管理和路线图文档

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude <noreply@anthropic.com>"
```

## 步骤 4: 连接远程仓库

### 使用 HTTPS

```bash
# 替换 your-username 和 your-repo
git remote add origin https://github.com/your-username/tokenginx.git

# 推送代码
git push -u origin main
# 输入用户名和 Personal Access Token
```

### 使用 SSH

```bash
# 替换 your-username
git remote add origin git@github.com:your-username/tokenginx.git

# 推送代码
git push -u origin main
```

## 步骤 5: 验证

访问你的 GitHub 仓库查看代码是否已推送：
```
https://github.com/your-username/tokenginx
```

## 后续步骤

### 1. 创建开发任务

```bash
# 确保已安装 GitHub CLI
gh auth login

# 创建所有 v0.1.0 任务（20 个 Issue）
./scripts/create-v0.1.0-issues.sh
```

### 2. 配置 GitHub Projects

参考 [GitHub Projects 配置指南](../tasks/github-projects.md) 设置项目看板。

### 3. 配置分支保护

建议为 `main` 分支设置保护规则：

1. 访问仓库 Settings → Branches
2. 点击 "Add rule"
3. Branch name pattern: `main`
4. 勾选：
   - ✅ Require a pull request before merging
   - ✅ Require status checks to pass before merging
   - ✅ Require branches to be up to date before merging
   - ✅ Include administrators
5. 点击 "Create"

### 4. 启用 GitHub Actions

GitHub Actions 会自动运行（`.github/workflows/ci.yml` 已配置）：
- 每次 push 到 `main` 或 `develop`
- 每次创建 Pull Request
- 运行测试、代码检查、覆盖率上传

## 常见问题

### Q: 推送时提示 "Authentication failed"

**HTTPS 用户**:
- 确保使用的是 Personal Access Token，而不是 GitHub 密码
- Token 可能已过期，重新生成一个

**SSH 用户**:
- 运行 `ssh -T git@github.com` 测试连接
- 确保 SSH Key 已添加到 GitHub
- 检查 SSH Agent 是否运行：`ssh-add -l`

### Q: 推送时提示 "Permission denied"

- 检查仓库权限（是否有写权限）
- 检查 Token 权限（是否勾选了 `repo`）
- 检查仓库 URL 是否正确

### Q: 如何切换 HTTPS 和 SSH？

```bash
# 查看当前 URL
git remote -v

# 切换到 HTTPS
git remote set-url origin https://github.com/username/tokenginx.git

# 切换到 SSH
git remote set-url origin git@github.com:username/tokenginx.git
```

### Q: 如何保存 HTTPS 凭据（避免每次输入）？

```bash
# 永久保存（明文存储，不安全）
git config --global credential.helper store

# 缓存 15 分钟
git config --global credential.helper cache

# 缓存 1 小时
git config --global credential.helper 'cache --timeout=3600'

# 使用 Git Credential Manager（推荐，支持 Windows/macOS/Linux）
# 下载：https://github.com/GitCredentialManager/git-credential-manager
```

### Q: 推送时提示 "Repository not found"

- 检查仓库 URL 是否正确
- 检查仓库是否已创建
- 检查用户名/组织名是否正确
- 检查是否有访问权限（私有仓库）

### Q: 如何更新远程 URL？

```bash
# 删除旧的 origin
git remote remove origin

# 添加新的 origin
git remote add origin <new-url>

# 或直接修改
git remote set-url origin <new-url>
```

## 安全最佳实践

### Personal Access Token

- ✅ 设置合理的过期时间（建议 90 天）
- ✅ 只授予必要的权限
- ✅ 定期轮换 Token
- ❌ 不要在代码中硬编码 Token
- ❌ 不要分享 Token

### SSH Key

- ✅ 使用 Ed25519 算法（更安全）
- ✅ 设置 SSH Key 密码短语
- ✅ 定期轮换 SSH Key
- ❌ 不要分享私钥（`id_ed25519` 或 `id_rsa`）

### Git 配置

```bash
# 全局忽略敏感文件
cat >> ~/.gitignore_global << EOF
.env
.env.local
*.key
*.pem
credentials.json
EOF

git config --global core.excludesfile ~/.gitignore_global
```

## 脚本使用示例

### 运行自动化脚本

```bash
$ ./scripts/setup-github.sh

=====================================
  TokenginX GitHub 仓库初始化
=====================================

[INFO] 检查环境...
[SUCCESS] 环境检查通过

步骤 1/5: 配置 Git 用户信息

请输入你的 GitHub 用户名: yangsen
请输入你的 Git 邮箱: yangsen@example.com
请输入你的 Git 用户名 [默认: yangsen]:

步骤 2/5: 配置 GitHub 仓库信息

请输入 GitHub 仓库所有者（用户名或组织名） [默认: yangsen]:
请输入仓库名称 [默认: tokenginx]:
选择仓库可见性 (public/private) [默认: public]:

步骤 3/5: 选择认证方式

GitHub 认证方式：
  1) HTTPS (Personal Access Token) - 推荐
  2) SSH (SSH Key)

请选择认证方式 (1/2) [默认: 1]: 1

[INFO] 使用 HTTPS 认证需要 Personal Access Token
[INFO] 获取 Token: https://github.com/settings/tokens
[INFO] 需要的权限: repo (完整仓库权限)

请输入你的 Personal Access Token: ********

步骤 4/5: 确认配置

配置摘要：
  Git 用户名: yangsen
  Git 邮箱: yangsen@example.com
  GitHub 用户名: yangsen
  仓库所有者: yangsen
  仓库名称: tokenginx
  仓库可见性: public
  仓库 URL: https://github.com/yangsen/tokenginx.git
  认证方式: HTTPS (Personal Access Token)

确认以上信息是否正确？ [y/N]: y

步骤 5/5: 初始化 Git 仓库

[INFO] 初始化 Git 仓库...
[SUCCESS] Git 仓库初始化完成
[INFO] 配置 Git 用户信息...
[SUCCESS] Git 用户信息配置完成
[INFO] 创建 .gitignore 文件...
[SUCCESS] .gitignore 文件已创建
[INFO] 设置默认分支为 main...
[SUCCESS] 默认分支已设置为 main
[INFO] 添加文件到 Git...
[SUCCESS] 文件已添加
[INFO] 创建初始提交...
[SUCCESS] 初始提交已创建
[INFO] 配置远程仓库...
[SUCCESS] 远程仓库已添加
[INFO] 远程仓库 URL: https://yangsen:***@github.com/yangsen/tokenginx.git

注意：在推送代码前，请确保 GitHub 仓库已创建

如果仓库不存在，请访问以下链接创建：
  https://github.com/new

创建仓库时：
  - 仓库名称: tokenginx
  - 可见性: public
  - 不要初始化 README、.gitignore 或 LICENSE（本地已有）

GitHub 仓库是否已创建？ [y/N]: y

[INFO] 推送代码到 GitHub...
是否立即推送代码到 GitHub？ [y/N]: y

[INFO] 首次推送，设置 upstream...
[SUCCESS] 代码已推送到 GitHub！

=====================================
     GitHub 仓库初始化完成！
=====================================

仓库地址: https://github.com/yangsen/tokenginx

下一步：
  1. 访问仓库查看代码
  2. 运行 ./scripts/create-v0.1.0-issues.sh 创建开发任务
  3. 配置 GitHub Projects 看板
  4. 开始开发！

[INFO] 保存配置到 .github-config（仅本地）...
[SUCCESS] 配置已保存到 .github-config
[SUCCESS] 所有操作完成！
```

## 相关资源

- [GitHub 文档](https://docs.github.com/)
- [Git 官方文档](https://git-scm.com/doc)
- [GitHub CLI 文档](https://cli.github.com/manual/)
- [GitHub Actions 文档](https://docs.github.com/en/actions)
- [贡献指南](../../contributing.md)
- [开发入门](../tasks/getting-started.md)
