# Podman 部署指南

本指南介绍如何使用 Podman 部署 TokenginX。Podman 是一个无守护进程的容器引擎，提供与 Docker 兼容的 CLI，更加安全和轻量。

## 前置要求

- Podman 3.0+
- Podman Compose 1.0+ (可选)
- 至少 2GB 可用内存
- 至少 10GB 可用磁盘空间

## 安装 Podman

### RHEL/CentOS/Fedora

```bash
# Fedora
sudo dnf install podman podman-compose

# CentOS 8/RHEL 8
sudo dnf install podman podman-compose

# CentOS 7/RHEL 7
sudo yum install podman
```

### Ubuntu/Debian

```bash
# Ubuntu 20.10+
sudo apt-get update
sudo apt-get install podman podman-compose

# Ubuntu 20.04
source /etc/os-release
echo "deb https://download.opensuse.org/repositories/devel:/kubic:/libcontainers:/stable/xUbuntu_${VERSION_ID}/ /" | \
  sudo tee /etc/apt/sources.list.d/devel:kubic:libcontainers:stable.list
curl -L "https://download.opensuse.org/repositories/devel:/kubic:/libcontainers:/stable/xUbuntu_${VERSION_ID}/Release.key" | \
  sudo apt-key add -
sudo apt-get update
sudo apt-get install podman podman-compose
```

### macOS

```bash
# 使用 Homebrew
brew install podman

# 初始化 Podman Machine
podman machine init
podman machine start
```

## Docker vs Podman 对比

| 特性 | Docker | Podman |
|-----|--------|--------|
| 守护进程 | 需要 dockerd | 无守护进程 |
| Root 权限 | 默认需要 | 支持 rootless |
| 安全性 | 较好 | 更好（无守护进程） |
| CLI 兼容 | - | 兼容 Docker CLI |
| Systemd 集成 | 第三方 | 原生支持 |
| Kubernetes YAML | 不支持 | 支持生成 |

## 快速开始

### 方式 1: 从 Docker Hub 拉取

```bash
# Podman 兼容 Docker 镜像
podman pull docker.io/tokenginx/tokenginx:latest

# 运行容器
podman run -d \
  --name tokenginx \
  -p 6380:6380 \
  -p 9090:9090 \
  -p 8080:8080 \
  docker.io/tokenginx/tokenginx:latest
```

### 方式 2: 从源码构建

```bash
# 克隆仓库
git clone https://github.com/your-org/tokenginx.git
cd tokenginx

# 使用 Podman 构建
podman build -t tokenginx:local .

# 运行容器
podman run -d \
  --name tokenginx \
  -p 6380:6380 \
  -p 9090:9090 \
  -p 8080:8080 \
  tokenginx:local
```

### 方式 3: 使用 Podman Compose

```bash
# 安装 podman-compose
pip3 install podman-compose

# 或使用系统包管理器
sudo dnf install podman-compose

# 启动服务
podman-compose up -d

# 查看日志
podman-compose logs -f tokenginx

# 停止服务
podman-compose down
```

## Rootless 模式（推荐）

Podman 的核心优势是支持无 root 权限运行容器，提高安全性。

### 配置 Rootless 模式

```bash
# 确认当前用户有 subuid 和 subgid
cat /etc/subuid
cat /etc/subgid

# 如果没有，添加配置
echo "$(whoami):100000:65536" | sudo tee -a /etc/subuid
echo "$(whoami):100000:65536" | sudo tee -a /etc/subgid

# 启用 lingering（允许用户容器在登出后继续运行）
loginctl enable-linger $(whoami)
```

### Rootless 运行容器

```bash
# 作为普通用户运行（无需 sudo）
podman run -d \
  --name tokenginx \
  -p 6380:6380 \
  -p 9090:9090 \
  -p 8080:8080 \
  tokenginx:latest

# 查看运行的容器
podman ps

# 查看日志
podman logs tokenginx
```

### 端口映射注意事项

Rootless 模式下，< 1024 的端口需要特殊配置：

```bash
# 方式 1: 使用高端口
podman run -d \
  --name tokenginx \
  -p 8380:6380 \
  -p 9090:9090 \
  -p 8080:8080 \
  tokenginx:latest

# 方式 2: 配置端口转发
sudo sysctl net.ipv4.ip_unprivileged_port_start=80

# 方式 3: 使用 slirp4netns 端口转发
podman run -d \
  --name tokenginx \
  --network slirp4netns:port_handler=slirp4netns \
  -p 6380:6380 \
  tokenginx:latest
```

## 配置管理

### 使用环境变量

```bash
podman run -d \
  --name tokenginx \
  -p 6380:6380 \
  -e TOKENGINX_LOG_LEVEL=debug \
  -e TOKENGINX_TCP_ADDR=0.0.0.0:6380 \
  -e TOKENGINX_MASTER_KEY=your-secret-key-here \
  tokenginx:latest
```

### 使用配置文件

```bash
# 创建配置文件
mkdir -p ~/.config/tokenginx
cat > ~/.config/tokenginx/config.yaml <<EOF
server:
  tcp_addr: "0.0.0.0:6380"
  grpc_addr: "0.0.0.0:9090"
  http_addr: "0.0.0.0:8080"

storage:
  enable_persistence: true
  data_dir: "/var/lib/tokenginx"

logging:
  level: "info"
EOF

# 挂载配置文件
podman run -d \
  --name tokenginx \
  -p 6380:6380 \
  -v ~/.config/tokenginx/config.yaml:/etc/tokenginx/config.yaml:ro,Z \
  tokenginx:latest -config /etc/tokenginx/config.yaml
```

**注意**: `:Z` 标志用于 SELinux 上下文，在 RHEL/CentOS/Fedora 上必需。

## 数据持久化

### 使用命名卷

```bash
# 创建卷
podman volume create tokenginx-data

# 运行容器
podman run -d \
  --name tokenginx \
  -p 6380:6380 \
  -v tokenginx-data:/var/lib/tokenginx:Z \
  tokenginx:latest

# 查看卷信息
podman volume inspect tokenginx-data
```

### 使用绑定挂载

```bash
# 创建数据目录
mkdir -p ~/tokenginx/data

# 运行容器
podman run -d \
  --name tokenginx \
  -p 6380:6380 \
  -v ~/tokenginx/data:/var/lib/tokenginx:Z \
  tokenginx:latest
```

## Systemd 集成

Podman 原生支持 Systemd，可以将容器作为系统服务管理。

### 生成 Systemd 单元文件

```bash
# 启动容器
podman run -d \
  --name tokenginx \
  -p 6380:6380 \
  -p 9090:9090 \
  -p 8080:8080 \
  -v tokenginx-data:/var/lib/tokenginx:Z \
  tokenginx:latest

# 生成 systemd 单元文件
podman generate systemd --new --name tokenginx > ~/.config/systemd/user/tokenginx.service

# 或者为 root 用户生成
sudo podman generate systemd --new --name tokenginx > /etc/systemd/system/tokenginx.service
```

### 用户级 Systemd 服务（Rootless）

```bash
# 创建 systemd 目录
mkdir -p ~/.config/systemd/user

# 生成服务文件
podman generate systemd --new --name tokenginx \
  > ~/.config/systemd/user/tokenginx.service

# 停止并删除容器（systemd 将重新创建）
podman stop tokenginx
podman rm tokenginx

# 重新加载 systemd
systemctl --user daemon-reload

# 启用并启动服务
systemctl --user enable tokenginx.service
systemctl --user start tokenginx.service

# 查看状态
systemctl --user status tokenginx.service

# 查看日志
journalctl --user -u tokenginx.service -f
```

### 系统级 Systemd 服务（Root）

```bash
# 生成服务文件（需要 root）
sudo podman generate systemd --new --name tokenginx \
  > /etc/systemd/system/tokenginx.service

# 停止并删除容器
sudo podman stop tokenginx
sudo podman rm tokenginx

# 重新加载 systemd
sudo systemctl daemon-reload

# 启用并启动服务
sudo systemctl enable tokenginx.service
sudo systemctl start tokenginx.service

# 查看状态
sudo systemctl status tokenginx.service
```

### 自定义 Systemd 服务文件

```ini
# ~/.config/systemd/user/tokenginx.service
[Unit]
Description=TokenginX Session Storage
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
Restart=on-failure
RestartSec=5s
TimeoutStopSec=70
ExecStartPre=/usr/bin/podman pull docker.io/tokenginx/tokenginx:latest
ExecStart=/usr/bin/podman run \
  --rm \
  --name tokenginx \
  -p 6380:6380 \
  -p 9090:9090 \
  -p 8080:8080 \
  -v tokenginx-data:/var/lib/tokenginx:Z \
  docker.io/tokenginx/tokenginx:latest
ExecStop=/usr/bin/podman stop -t 10 tokenginx

[Install]
WantedBy=default.target
```

## Pod 管理

Podman 支持 Pod 概念（类似 Kubernetes），可以将多个容器组合在一起。

### 创建 Pod

```bash
# 创建 pod
podman pod create \
  --name tokenginx-pod \
  -p 6380:6380 \
  -p 9090:9090 \
  -p 8080:8080 \
  -p 9091:9090

# 在 pod 中运行 TokenginX
podman run -d \
  --pod tokenginx-pod \
  --name tokenginx \
  -v tokenginx-data:/var/lib/tokenginx:Z \
  tokenginx:latest

# 在同一 pod 中运行 Prometheus（共享网络）
podman run -d \
  --pod tokenginx-pod \
  --name prometheus \
  -v ./prometheus.yml:/etc/prometheus/prometheus.yml:ro,Z \
  prom/prometheus:latest

# 查看 pod
podman pod ps

# 查看 pod 中的容器
podman ps --pod
```

### 管理 Pod

```bash
# 启动 pod
podman pod start tokenginx-pod

# 停止 pod
podman pod stop tokenginx-pod

# 重启 pod
podman pod restart tokenginx-pod

# 删除 pod（会删除所有容器）
podman pod rm -f tokenginx-pod

# 查看 pod 日志
podman logs -f tokenginx
```

### 生成 Kubernetes YAML

```bash
# 从 pod 生成 Kubernetes YAML
podman generate kube tokenginx-pod > tokenginx-k8s.yaml

# 查看生成的 YAML
cat tokenginx-k8s.yaml

# 使用 YAML 部署 pod
podman play kube tokenginx-k8s.yaml

# 删除 pod
podman play kube --down tokenginx-k8s.yaml
```

## Podman Compose

Podman Compose 提供与 Docker Compose 兼容的接口。

### 安装 Podman Compose

```bash
# 方式 1: pip 安装
pip3 install podman-compose

# 方式 2: 系统包管理器
sudo dnf install podman-compose  # Fedora
sudo apt install podman-compose  # Ubuntu 22.04+
```

### 使用 docker-compose.yml

```yaml
# docker-compose.yml (与 Docker Compose 相同)
version: '3.8'

services:
  tokenginx:
    image: tokenginx:latest
    container_name: tokenginx-server
    restart: unless-stopped
    ports:
      - "6380:6380"
      - "9090:9090"
      - "8080:8080"
    volumes:
      - tokenginx-data:/var/lib/tokenginx
      - ./config.yaml:/etc/tokenginx/config.yaml:ro
    environment:
      - TOKENGINX_LOG_LEVEL=info

volumes:
  tokenginx-data:
```

### Podman Compose 命令

```bash
# 启动服务
podman-compose up -d

# 查看状态
podman-compose ps

# 查看日志
podman-compose logs -f

# 停止服务
podman-compose down

# 重启服务
podman-compose restart

# 扩展服务
podman-compose up -d --scale tokenginx=3
```

## 网络配置

### 创建自定义网络

```bash
# 创建网络
podman network create tokenginx-net

# 查看网络
podman network ls

# 运行容器并指定网络
podman run -d \
  --name tokenginx \
  --network tokenginx-net \
  -p 6380:6380 \
  tokenginx:latest

# 查看网络详情
podman network inspect tokenginx-net
```

### 容器间通信

```bash
# 创建网络
podman network create tokenginx-net

# 运行 TokenginX
podman run -d \
  --name tokenginx \
  --network tokenginx-net \
  tokenginx:latest

# 运行测试容器
podman run -it --rm \
  --network tokenginx-net \
  alpine sh

# 在测试容器中访问 TokenginX
nc -zv tokenginx 6380
```

## 监控和日志

### 查看容器日志

```bash
# 实时日志
podman logs -f tokenginx

# 最近 100 行
podman logs --tail 100 tokenginx

# 带时间戳
podman logs --timestamps tokenginx

# 特定时间范围
podman logs --since 2h tokenginx
```

### 查看资源使用

```bash
# 实时统计
podman stats tokenginx

# 一次性统计
podman stats --no-stream tokenginx

# 所有容器
podman stats --no-stream
```

### 检查容器

```bash
# 容器详细信息
podman inspect tokenginx

# 特定字段
podman inspect --format='{{.State.Status}}' tokenginx

# 网络信息
podman inspect --format='{{.NetworkSettings.IPAddress}}' tokenginx
```

## 备份与恢复

### 备份容器

```bash
# 导出容器为 tar
podman export tokenginx > tokenginx-container.tar

# 提交容器为镜像
podman commit tokenginx tokenginx:backup

# 保存镜像
podman save -o tokenginx-backup.tar tokenginx:backup
```

### 备份卷

```bash
# 停止容器
podman stop tokenginx

# 备份卷数据
podman run --rm \
  -v tokenginx-data:/source:ro \
  -v $(pwd):/backup:Z \
  alpine tar czf /backup/tokenginx-data-$(date +%Y%m%d).tar.gz -C /source .

# 重启容器
podman start tokenginx
```

### 恢复数据

```bash
# 创建新卷
podman volume create tokenginx-data-new

# 恢复数据
podman run --rm \
  -v tokenginx-data-new:/target \
  -v $(pwd):/backup:ro,Z \
  alpine tar xzf /backup/tokenginx-data-20250119.tar.gz -C /target

# 使用新卷启动容器
podman run -d \
  --name tokenginx-new \
  -p 6380:6380 \
  -v tokenginx-data-new:/var/lib/tokenginx:Z \
  tokenginx:latest
```

## 安全最佳实践

### 1. 使用 Rootless 模式

```bash
# 以普通用户运行（推荐）
podman run -d --name tokenginx -p 6380:6380 tokenginx:latest
```

### 2. SELinux 标签

```bash
# 正确使用 SELinux 标签
podman run -d \
  --name tokenginx \
  -v ~/data:/var/lib/tokenginx:Z \  # 私有标签
  tokenginx:latest

# 或使用共享标签（多容器共享）
podman run -d \
  --name tokenginx \
  -v ~/data:/var/lib/tokenginx:z \
  tokenginx:latest
```

### 3. 只读根文件系统

```bash
podman run -d \
  --name tokenginx \
  --read-only \
  --tmpfs /tmp \
  -v tokenginx-data:/var/lib/tokenginx:Z \
  tokenginx:latest
```

### 4. 限制 Capabilities

```bash
podman run -d \
  --name tokenginx \
  --cap-drop=ALL \
  --cap-add=NET_BIND_SERVICE \
  tokenginx:latest
```

### 5. 扫描镜像

```bash
# 使用 Podman 内置扫描
podman scan tokenginx:latest

# 使用 Trivy
podman run --rm \
  -v /var/run/podman/podman.sock:/var/run/docker.sock:ro \
  aquasec/trivy image tokenginx:latest
```

## 故障排查

### 检查 Podman 版本

```bash
podman version
podman info
```

### 权限问题

```bash
# 检查 subuid/subgid
cat /etc/subuid
cat /etc/subgid

# 重置 rootless 配置
podman system reset

# 重新配置
podman system migrate
```

### 网络问题

```bash
# 检查网络
podman network ls
podman network inspect bridge

# 重置网络
podman network prune

# 测试连接
podman run --rm alpine ping -c 3 8.8.8.8
```

### SELinux 问题

```bash
# 检查 SELinux 状态
getenforce

# 临时禁用（调试用）
sudo setenforce 0

# 查看 SELinux 拒绝
sudo ausearch -m avc -ts recent

# 生成 SELinux 策略
sudo ausearch -m avc -ts recent | audit2allow -M mypolicy
sudo semodule -i mypolicy.pp
```

## 从 Docker 迁移到 Podman

### 命令别名

```bash
# 添加到 ~/.bashrc 或 ~/.zshrc
alias docker=podman
alias docker-compose=podman-compose

# 重新加载配置
source ~/.bashrc
```

### API 兼容性

```bash
# 启动 Podman API 服务（Docker API 兼容）
podman system service --time=0 unix:///tmp/podman.sock

# 设置 DOCKER_HOST
export DOCKER_HOST=unix:///tmp/podman.sock

# 现在可以使用 Docker 客户端
docker ps
docker-compose up -d
```

### 迁移镜像

```bash
# 从 Docker 导出
docker save tokenginx:latest | gzip > tokenginx.tar.gz

# 导入到 Podman
podman load < tokenginx.tar.gz
```

## 常用命令对比

| 操作 | Docker | Podman |
|-----|--------|--------|
| 运行容器 | `docker run` | `podman run` |
| 列出容器 | `docker ps` | `podman ps` |
| 构建镜像 | `docker build` | `podman build` |
| 查看日志 | `docker logs` | `podman logs` |
| 进入容器 | `docker exec` | `podman exec` |
| 删除容器 | `docker rm` | `podman rm` |
| 创建编排 | `docker-compose` | `podman-compose` |
| 生成 K8s YAML | 不支持 | `podman generate kube` |

## 下一步

- 查看 [Kubernetes 部署指南](./kubernetes.md) 了解生产级编排
- 查看 [Docker 部署指南](./docker.md) 了解 Docker 部署
- 查看 [安全配置指南](../security/) 了解安全加固
