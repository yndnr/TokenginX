# Docker 部署指南

本指南介绍如何使用 Docker 部署 TokenginX。

## 前置要求

- Docker 20.10+
- Docker Compose 2.0+ (可选，用于编排)
- 至少 2GB 可用内存
- 至少 10GB 可用磁盘空间

## 快速开始

### 方式 1: 使用预构建镜像

```bash
# 拉取镜像
docker pull tokenginx/tokenginx:latest

# 运行容器
docker run -d \
  --name tokenginx \
  -p 6380:6380 \
  -p 9090:9090 \
  -p 8080:8080 \
  tokenginx/tokenginx:latest
```

### 方式 2: 从源码构建

```bash
# 克隆仓库
git clone https://github.com/your-org/tokenginx.git
cd tokenginx

# 构建镜像
docker build -t tokenginx:local .

# 运行容器
docker run -d \
  --name tokenginx \
  -p 6380:6380 \
  -p 9090:9090 \
  -p 8080:8080 \
  tokenginx:local
```

### 方式 3: 使用 Docker Compose

```bash
# 启动服务
docker-compose up -d

# 查看日志
docker-compose logs -f tokenginx

# 停止服务
docker-compose down
```

## Dockerfile 详解

TokenginX 使用多阶段构建优化镜像大小：

```dockerfile
# 构建阶段
FROM golang:1.22-alpine AS builder
WORKDIR /build
COPY . .
RUN go build -o tokenginx-server ./cmd/server

# 运行阶段
FROM alpine:3.19
COPY --from=builder /build/tokenginx-server /usr/local/bin/
EXPOSE 6380 9090 8080
ENTRYPOINT ["/usr/local/bin/tokenginx-server"]
```

**特点**:
- **多阶段构建**: 最终镜像只包含运行时必需文件
- **Alpine 基础镜像**: 镜像大小 < 50MB
- **非 root 用户**: 以 `tokenginx` 用户运行，提高安全性
- **健康检查**: 内置健康检查机制

## 端口说明

| 端口 | 协议 | 说明 |
|-----|------|------|
| 6380 | TCP (RESP) | Redis 兼容协议 |
| 9090 | gRPC | gRPC 服务 |
| 8080 | HTTP | REST API |
| 9100 | HTTP | Prometheus metrics |
| 8081 | HTTP | 健康检查端点 |

## 配置管理

### 使用环境变量

```bash
docker run -d \
  --name tokenginx \
  -p 6380:6380 \
  -e TOKENGINX_LOG_LEVEL=debug \
  -e TOKENGINX_TCP_ADDR=0.0.0.0:6380 \
  -e TOKENGINX_MASTER_KEY=your-secret-key-here \
  tokenginx:latest
```

**支持的环境变量**:
- `TOKENGINX_LOG_LEVEL`: 日志级别 (debug/info/warn/error)
- `TOKENGINX_TCP_ADDR`: TCP 监听地址
- `TOKENGINX_GRPC_ADDR`: gRPC 监听地址
- `TOKENGINX_HTTP_ADDR`: HTTP 监听地址
- `TOKENGINX_DATA_DIR`: 数据目录
- `TOKENGINX_MASTER_KEY`: 加密主密钥
- `TOKENGINX_TLS_ENABLED`: 启用 TLS (true/false)

### 使用配置文件

```bash
# 创建配置文件
cat > config.yaml <<EOF
server:
  tcp_addr: "0.0.0.0:6380"
  grpc_addr: "0.0.0.0:9090"
  http_addr: "0.0.0.0:8080"

storage:
  enable_persistence: true
  data_dir: "/var/lib/tokenginx"

logging:
  level: "info"
  format: "json"
EOF

# 挂载配置文件
docker run -d \
  --name tokenginx \
  -p 6380:6380 \
  -v $(pwd)/config.yaml:/etc/tokenginx/config.yaml:ro \
  tokenginx:latest -config /etc/tokenginx/config.yaml
```

## 数据持久化

### 使用命名卷

```bash
# 创建数据卷
docker volume create tokenginx-data

# 运行容器并挂载卷
docker run -d \
  --name tokenginx \
  -p 6380:6380 \
  -v tokenginx-data:/var/lib/tokenginx \
  tokenginx:latest
```

### 使用绑定挂载

```bash
# 创建数据目录
mkdir -p /data/tokenginx

# 运行容器
docker run -d \
  --name tokenginx \
  -p 6380:6380 \
  -v /data/tokenginx:/var/lib/tokenginx \
  tokenginx:latest
```

## Docker Compose 完整配置

```yaml
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
      - "9100:9100"
      - "8081:8081"
    volumes:
      - tokenginx-data:/var/lib/tokenginx
      - tokenginx-logs:/var/log/tokenginx
      - ./config.yaml:/etc/tokenginx/config.yaml:ro
    environment:
      - TOKENGINX_LOG_LEVEL=info
      - TOKENGINX_MASTER_KEY=${MASTER_KEY}
    networks:
      - tokenginx-network
    healthcheck:
      test: ["CMD", "/usr/local/bin/tokenginx-client", "ping"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 10s
    deploy:
      resources:
        limits:
          cpus: '2'
          memory: 2G
        reservations:
          cpus: '0.5'
          memory: 512M

volumes:
  tokenginx-data:
  tokenginx-logs:

networks:
  tokenginx-network:
    driver: bridge
```

## TLS/mTLS 配置

### 挂载证书

```bash
# 准备证书
mkdir -p certs
# 将证书放入 certs 目录

# 运行容器
docker run -d \
  --name tokenginx \
  -p 6380:6380 \
  -v $(pwd)/certs:/etc/tokenginx/certs:ro \
  -v $(pwd)/config-tls.yaml:/etc/tokenginx/config.yaml:ro \
  tokenginx:latest
```

### TLS 配置文件

```yaml
# config-tls.yaml
security:
  tls:
    enabled: true
    cert_file: "/etc/tokenginx/certs/server.crt"
    key_file: "/etc/tokenginx/certs/server.key"
    ca_file: "/etc/tokenginx/certs/ca.crt"
    client_auth: "require"
```

## 监控集成

### 添加 Prometheus 监控

```yaml
# docker-compose.yml
services:
  tokenginx:
    # ... tokenginx 配置 ...

  prometheus:
    image: prom/prometheus:latest
    ports:
      - "9091:9090"
    volumes:
      - ./prometheus.yml:/etc/prometheus/prometheus.yml:ro
      - prometheus-data:/prometheus
    networks:
      - tokenginx-network

  grafana:
    image: grafana/grafana:latest
    ports:
      - "3000:3000"
    volumes:
      - grafana-data:/var/lib/grafana
    environment:
      - GF_SECURITY_ADMIN_PASSWORD=admin
    networks:
      - tokenginx-network
    depends_on:
      - prometheus

volumes:
  prometheus-data:
  grafana-data:
```

### Prometheus 配置

```yaml
# prometheus.yml
global:
  scrape_interval: 15s
  evaluation_interval: 15s

scrape_configs:
  - job_name: 'tokenginx'
    static_configs:
      - targets: ['tokenginx:9100']
        labels:
          instance: 'tokenginx-server'
```

## 日志管理

### 查看日志

```bash
# 查看实时日志
docker logs -f tokenginx

# 查看最近 100 行
docker logs --tail 100 tokenginx

# 查看特定时间段
docker logs --since 2h tokenginx
```

### 日志驱动配置

```yaml
# docker-compose.yml
services:
  tokenginx:
    image: tokenginx:latest
    logging:
      driver: "json-file"
      options:
        max-size: "100m"
        max-file: "10"
        compress: "true"
```

### 集成 ELK

```yaml
services:
  tokenginx:
    image: tokenginx:latest
    logging:
      driver: "gelf"
      options:
        gelf-address: "udp://logstash:12201"
        tag: "tokenginx"
```

## 健康检查

### 内置健康检查

```bash
# 检查容器健康状态
docker inspect --format='{{.State.Health.Status}}' tokenginx

# 查看健康检查日志
docker inspect --format='{{json .State.Health}}' tokenginx | jq
```

### 自定义健康检查

```dockerfile
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:8081/health || exit 1
```

## 网络配置

### 创建自定义网络

```bash
# 创建桥接网络
docker network create \
  --driver bridge \
  --subnet 172.20.0.0/16 \
  --gateway 172.20.0.1 \
  tokenginx-net

# 运行容器
docker run -d \
  --name tokenginx \
  --network tokenginx-net \
  --ip 172.20.0.10 \
  -p 6380:6380 \
  tokenginx:latest
```

### Host 网络模式

```bash
# 使用 host 网络（高性能，但失去网络隔离）
docker run -d \
  --name tokenginx \
  --network host \
  tokenginx:latest
```

## 资源限制

### CPU 和内存限制

```bash
docker run -d \
  --name tokenginx \
  --cpus=2 \
  --memory=2g \
  --memory-reservation=512m \
  -p 6380:6380 \
  tokenginx:latest
```

### Docker Compose 中的资源限制

```yaml
services:
  tokenginx:
    image: tokenginx:latest
    deploy:
      resources:
        limits:
          cpus: '2'
          memory: 2G
        reservations:
          cpus: '0.5'
          memory: 512M
```

## 备份与恢复

### 备份数据

```bash
# 停止容器（可选，确保数据一致性）
docker stop tokenginx

# 备份数据卷
docker run --rm \
  -v tokenginx-data:/source:ro \
  -v $(pwd)/backup:/backup \
  alpine tar czf /backup/tokenginx-backup-$(date +%Y%m%d).tar.gz -C /source .

# 启动容器
docker start tokenginx
```

### 恢复数据

```bash
# 停止并删除容器
docker stop tokenginx
docker rm tokenginx

# 创建新的数据卷
docker volume create tokenginx-data-new

# 恢复数据
docker run --rm \
  -v tokenginx-data-new:/target \
  -v $(pwd)/backup:/backup:ro \
  alpine tar xzf /backup/tokenginx-backup-20250119.tar.gz -C /target

# 启动新容器
docker run -d \
  --name tokenginx \
  -p 6380:6380 \
  -v tokenginx-data-new:/var/lib/tokenginx \
  tokenginx:latest
```

## 多实例部署

### 使用不同端口

```bash
# 实例 1
docker run -d \
  --name tokenginx-1 \
  -p 6381:6380 \
  -p 9091:9090 \
  -p 8081:8080 \
  tokenginx:latest

# 实例 2
docker run -d \
  --name tokenginx-2 \
  -p 6382:6380 \
  -p 9092:9090 \
  -p 8082:8080 \
  tokenginx:latest
```

### 使用 Docker Compose Scale

```yaml
# docker-compose.yml
services:
  tokenginx:
    image: tokenginx:latest
    ports:
      - "6380-6389:6380"
    deploy:
      replicas: 3
```

```bash
# 扩展到 5 个实例
docker-compose up -d --scale tokenginx=5
```

## 安全最佳实践

### 1. 使用非 root 用户

Dockerfile 已默认配置：

```dockerfile
RUN addgroup -g 1000 tokenginx && \
    adduser -D -u 1000 -G tokenginx tokenginx
USER tokenginx
```

### 2. 只读文件系统

```bash
docker run -d \
  --name tokenginx \
  --read-only \
  --tmpfs /tmp \
  -v tokenginx-data:/var/lib/tokenginx \
  -p 6380:6380 \
  tokenginx:latest
```

### 3. 限制 Capabilities

```bash
docker run -d \
  --name tokenginx \
  --cap-drop=ALL \
  --cap-add=NET_BIND_SERVICE \
  -p 6380:6380 \
  tokenginx:latest
```

### 4. 使用 Secrets

```bash
# 创建 secret
echo "my-secret-key" | docker secret create tokenginx_master_key -

# 使用 secret (Docker Swarm)
docker service create \
  --name tokenginx \
  --secret tokenginx_master_key \
  -e TOKENGINX_MASTER_KEY_FILE=/run/secrets/tokenginx_master_key \
  tokenginx:latest
```

### 5. 扫描镜像漏洞

```bash
# 使用 Trivy 扫描
docker run --rm \
  -v /var/run/docker.sock:/var/run/docker.sock \
  aquasec/trivy image tokenginx:latest

# 使用 Docker Scout
docker scout cves tokenginx:latest
```

## 故障排查

### 容器无法启动

```bash
# 查看容器日志
docker logs tokenginx

# 查看容器详细信息
docker inspect tokenginx

# 交互式运行容器
docker run -it --rm \
  -p 6380:6380 \
  tokenginx:latest /bin/sh
```

### 性能问题

```bash
# 查看资源使用
docker stats tokenginx

# 查看容器进程
docker top tokenginx

# 进入容器调试
docker exec -it tokenginx /bin/sh
```

### 网络问题

```bash
# 检查端口映射
docker port tokenginx

# 检查网络配置
docker network inspect tokenginx-network

# 测试连接
docker run --rm --network tokenginx-network \
  alpine ping tokenginx
```

## 常用命令

```bash
# 构建镜像
docker build -t tokenginx:latest .

# 启动容器
docker run -d --name tokenginx -p 6380:6380 tokenginx:latest

# 停止容器
docker stop tokenginx

# 重启容器
docker restart tokenginx

# 删除容器
docker rm -f tokenginx

# 查看日志
docker logs -f tokenginx

# 进入容器
docker exec -it tokenginx /bin/sh

# 查看容器资源使用
docker stats tokenginx

# 导出镜像
docker save tokenginx:latest | gzip > tokenginx-latest.tar.gz

# 导入镜像
docker load < tokenginx-latest.tar.gz
```

## 下一步

- 查看 [Podman 部署指南](./podman.md) 了解无守护进程容器运行
- 查看 [Kubernetes 部署指南](./kubernetes.md) 了解生产级编排
- 查看 [监控指南](../production/monitoring.md) 了解完整监控方案
