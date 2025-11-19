# Multi-stage build for TokenginX
# Build stage
FROM golang:1.22-alpine AS builder

# 安装构建依赖
RUN apk add --no-cache git make ca-certificates tzdata

# 设置工作目录
WORKDIR /build

# 复制 go mod 文件并下载依赖
COPY go.mod go.sum ./
RUN go mod download && go mod verify

# 复制源代码
COPY . .

# 构建二进制文件
ARG VERSION=dev
ARG COMMIT_SHA=unknown
ARG BUILD_TIME
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s -X main.version=${VERSION} -X main.commitSHA=${COMMIT_SHA} -X main.buildTime=${BUILD_TIME}" \
    -o tokenginx-server \
    ./cmd/server

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s -X main.version=${VERSION}" \
    -o tokenginx-client \
    ./cmd/client

# Runtime stage
FROM alpine:3.19

# 安装运行时依赖
RUN apk add --no-cache ca-certificates tzdata

# 创建非 root 用户
RUN addgroup -g 1000 tokenginx && \
    adduser -D -u 1000 -G tokenginx tokenginx

# 创建必要的目录
RUN mkdir -p /var/lib/tokenginx /var/log/tokenginx /etc/tokenginx && \
    chown -R tokenginx:tokenginx /var/lib/tokenginx /var/log/tokenginx /etc/tokenginx

# 复制二进制文件
COPY --from=builder /build/tokenginx-server /usr/local/bin/
COPY --from=builder /build/tokenginx-client /usr/local/bin/

# 复制配置文件示例
COPY --from=builder /build/config/config.example.yaml /etc/tokenginx/config.yaml

# 设置权限
RUN chmod +x /usr/local/bin/tokenginx-server /usr/local/bin/tokenginx-client

# 切换到非 root 用户
USER tokenginx

# 设置工作目录
WORKDIR /home/tokenginx

# 暴露端口
EXPOSE 6380 9090 8080 9100 8081

# 健康检查
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD ["/usr/local/bin/tokenginx-client", "ping", "||", "exit", "1"]

# 启动命令
ENTRYPOINT ["/usr/local/bin/tokenginx-server"]
CMD ["-config", "/etc/tokenginx/config.yaml"]
