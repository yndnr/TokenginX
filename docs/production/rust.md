# Rust 生产环境指南

本指南帮助您在生产环境中部署和优化 Rust 应用与 TokenginX 的集成。

## 前置要求

- Rust 1.75 或更高版本
- Cargo
- 生产环境 TokenginX 服务器集群
- 监控和日志基础设施

## 生产级配置

### 1. Cargo.toml

```toml
[package]
name = "myapp"
version = "1.0.0"
edition = "2021"

[dependencies]
# Redis 客户端
redis = { version = "0.24", features = ["tokio-comp", "tls-native-tls", "cluster"] }

# 异步运行时
tokio = { version = "1", features = ["full"] }

# HTTP 服务器
axum = "0.7"
tower = "0.4"
tower-http = { version = "0.5", features = ["trace", "compression-gzip"] }

# 序列化
serde = { version = "1.0", features = ["derive"] }
serde_json = "1.0"

# 日志
tracing = "0.1"
tracing-subscriber = { version = "0.3", features = ["env-filter", "json"] }

# 监控
metrics = "0.22"
metrics-exporter-prometheus = "0.13"

# 配置
config = "0.14"

# 错误处理
anyhow = "1.0"
thiserror = "1.0"

[profile.release]
opt-level = 3
lto = true
codegen-units = 1
strip = true
```

### 2. 配置管理

创建 `src/config.rs`:

```rust
use anyhow::Result;
use config::{Config, ConfigError, Environment, File};
use serde::Deserialize;
use std::time::Duration;

#[derive(Debug, Deserialize, Clone)]
pub struct AppConfig {
    pub server: ServerConfig,
    pub tokenginx: TokenginXConfig,
    pub log: LogConfig,
    pub metrics: MetricsConfig,
}

#[derive(Debug, Deserialize, Clone)]
pub struct ServerConfig {
    pub host: String,
    pub port: u16,
    #[serde(with = "humantime_serde")]
    pub read_timeout: Duration,
    #[serde(with = "humantime_serde")]
    pub write_timeout: Duration,
}

#[derive(Debug, Deserialize, Clone)]
pub struct TokenginXConfig {
    pub nodes: Vec<String>,
    pub password: String,
    pub pool_size: u32,
    pub min_idle: u32,
    #[serde(with = "humantime_serde")]
    pub connection_timeout: Duration,
    #[serde(with = "humantime_serde")]
    pub response_timeout: Duration,
    pub tls: TlsConfig,
}

#[derive(Debug, Deserialize, Clone)]
pub struct TlsConfig {
    pub enabled: bool,
    pub cert_file: Option<String>,
    pub key_file: Option<String>,
    pub ca_file: Option<String>,
}

#[derive(Debug, Deserialize, Clone)]
pub struct LogConfig {
    pub level: String,
    pub format: String,
}

#[derive(Debug, Deserialize, Clone)]
pub struct MetricsConfig {
    pub enabled: bool,
    pub path: String,
}

impl AppConfig {
    pub fn load(config_path: &str) -> Result<Self, ConfigError> {
        Config::builder()
            .add_source(File::with_name(config_path))
            .add_source(Environment::with_prefix("APP").separator("__"))
            .build()?
            .try_deserialize()
    }
}
```

创建 `config/production.toml`:

```toml
[server]
host = "0.0.0.0"
port = 8080
read_timeout = "10s"
write_timeout = "10s"

[tokenginx]
nodes = [
    "tokenginx-node1.prod.example.com:6380",
    "tokenginx-node2.prod.example.com:6380",
    "tokenginx-node3.prod.example.com:6380"
]
password = "${TOKENGINX_API_KEY}"
pool_size = 50
min_idle = 10
connection_timeout = "5s"
response_timeout = "3s"

[tokenginx.tls]
enabled = true
cert_file = "/certs/client-cert.pem"
key_file = "/certs/client-key.pem"
ca_file = "/certs/ca.pem"

[log]
level = "info"
format = "json"

[metrics]
enabled = true
path = "/metrics"
```

### 3. TokenginX 客户端

创建 `src/tokenginx/client.rs`:

```rust
use anyhow::{Context, Result};
use redis::aio::ConnectionManager;
use redis::{AsyncCommands, Client, Cmd};
use serde::{Deserialize, Serialize};
use std::sync::Arc;
use std::time::Duration;
use tracing::{debug, error, info, warn};

use super::metrics::TokenginXMetrics;

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct OAuthToken {
    pub user_id: String,
    pub scope: String,
    pub created_at: i64,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub client_ip: Option<String>,
}

#[derive(Clone)]
pub struct TokenginXClient {
    conn: ConnectionManager,
    metrics: Arc<TokenginXMetrics>,
}

impl TokenginXClient {
    pub async fn new(
        nodes: Vec<String>,
        password: String,
        metrics: Arc<TokenginXMetrics>,
    ) -> Result<Self> {
        let client = Client::open(format!(
            "redis://{}/?password={}",
            nodes.join(","),
            password
        ))
        .context("Failed to create Redis client")?;

        let conn = ConnectionManager::new(client)
            .await
            .context("Failed to create connection manager")?;

        // 测试连接
        let mut test_conn = conn.clone();
        redis::cmd("PING")
            .query_async::<_, String>(&mut test_conn)
            .await
            .context("Failed to ping TokenginX")?;

        info!("Connected to TokenginX cluster");

        Ok(Self { conn, metrics })
    }

    pub async fn set_oauth_token(
        &self,
        token_id: &str,
        token: &OAuthToken,
        ttl: Duration,
    ) -> Result<()> {
        let start = std::time::Instant::now();

        let key = format!("oauth:token:{}", token_id);
        let value = serde_json::to_string(token)
            .context("Failed to serialize token")?;

        let mut conn = self.conn.clone();
        conn.set_ex(&key, value, ttl.as_secs() as usize)
            .await
            .context("Failed to set token")?;

        self.metrics.token_created.increment(1);
        self.metrics
            .operation_duration
            .record(start.elapsed().as_secs_f64());

        info!(
            token_id = %token_id,
            user_id = %token.user_id,
            ttl_secs = %ttl.as_secs(),
            "OAuth token created"
        );

        Ok(())
    }

    pub async fn get_oauth_token(&self, token_id: &str) -> Result<Option<OAuthToken>> {
        let start = std::time::Instant::now();

        let key = format!("oauth:token:{}", token_id);

        let mut conn = self.conn.clone();
        let value: Option<String> = conn
            .get(&key)
            .await
            .context("Failed to get token")?;

        self.metrics
            .operation_duration
            .record(start.elapsed().as_secs_f64());

        match value {
            Some(v) => {
                let token: OAuthToken = serde_json::from_str(&v)
                    .context("Failed to deserialize token")?;

                self.metrics.token_retrieved.increment(1);
                debug!(token_id = %token_id, "OAuth token retrieved");

                Ok(Some(token))
            }
            None => {
                debug!(token_id = %token_id, "OAuth token not found");
                Ok(None)
            }
        }
    }

    pub async fn get_multiple_tokens(
        &self,
        token_ids: &[String],
    ) -> Result<Vec<Option<OAuthToken>>> {
        let start = std::time::Instant::now();

        let keys: Vec<String> = token_ids
            .iter()
            .map(|id| format!("oauth:token:{}", id))
            .collect();

        let mut conn = self.conn.clone();
        let values: Vec<Option<String>> = conn
            .get(&keys)
            .await
            .context("Failed to get multiple tokens")?;

        let tokens: Result<Vec<Option<OAuthToken>>> = values
            .into_iter()
            .map(|v| match v {
                Some(value) => {
                    let token: OAuthToken = serde_json::from_str(&value)
                        .context("Failed to deserialize token")?;
                    Ok(Some(token))
                }
                None => Ok(None),
            })
            .collect();

        self.metrics
            .operation_duration
            .record(start.elapsed().as_secs_f64());

        let result = tokens?;
        let found_count = result.iter().filter(|t| t.is_some()).count();

        debug!(
            total = token_ids.len(),
            found = found_count,
            "Batch retrieved tokens"
        );

        Ok(result)
    }

    pub async fn delete_token(&self, token_id: &str) -> Result<bool> {
        let key = format!("oauth:token:{}", token_id);

        let mut conn = self.conn.clone();
        let deleted: u32 = conn
            .del(&key)
            .await
            .context("Failed to delete token")?;

        info!(token_id = %token_id, deleted = deleted > 0, "Token deleted");

        Ok(deleted > 0)
    }

    pub async fn get_token_ttl(&self, token_id: &str) -> Result<Option<i64>> {
        let key = format!("oauth:token:{}", token_id);

        let mut conn = self.conn.clone();
        let ttl: i64 = conn.ttl(&key).await.context("Failed to get TTL")?;

        Ok(if ttl > 0 { Some(ttl) } else { None })
    }

    pub async fn is_healthy(&self) -> bool {
        let mut conn = self.conn.clone();
        match redis::cmd("PING")
            .query_async::<_, String>(&mut conn)
            .await
        {
            Ok(pong) => pong == "PONG",
            Err(e) => {
                error!(error = %e, "Health check failed");
                false
            }
        }
    }
}
```

### 4. Prometheus 指标

创建 `src/tokenginx/metrics.rs`:

```rust
use metrics::{counter, describe_counter, describe_histogram, histogram, Counter, Histogram};

pub struct TokenginXMetrics {
    pub token_created: Counter,
    pub token_retrieved: Counter,
    pub operation_duration: Histogram,
}

impl TokenginXMetrics {
    pub fn new() -> Self {
        describe_counter!("tokenginx_token_created_total", "Total tokens created");
        describe_counter!("tokenginx_token_retrieved_total", "Total tokens retrieved");
        describe_histogram!(
            "tokenginx_operation_duration_seconds",
            "Duration of TokenginX operations"
        );

        Self {
            token_created: counter!("tokenginx_token_created_total"),
            token_retrieved: counter!("tokenginx_token_retrieved_total"),
            operation_duration: histogram!("tokenginx_operation_duration_seconds"),
        }
    }
}

impl Default for TokenginXMetrics {
    fn default() -> Self {
        Self::new()
    }
}
```

### 5. HTTP 服务器

创建 `src/server.rs`:

```rust
use axum::{
    extract::{Json, State},
    http::StatusCode,
    response::IntoResponse,
    routing::{get, post},
    Router,
};
use metrics_exporter_prometheus::{Matcher, PrometheusBuilder, PrometheusHandle};
use serde::{Deserialize, Serialize};
use std::sync::Arc;
use std::time::Duration;
use tower_http::trace::TraceLayer;
use tracing::info;

use crate::tokenginx::{OAuthToken, TokenginXClient};

#[derive(Clone)]
pub struct AppState {
    pub tokenginx: TokenginXClient,
    pub prometheus: PrometheusHandle,
}

pub fn create_router(state: AppState) -> Router {
    Router::new()
        // 健康检查
        .route("/health/live", get(handle_liveness))
        .route("/health/ready", get(handle_readiness))
        // Prometheus 指标
        .route("/metrics", get(handle_metrics))
        // API 路由
        .route("/api/v1/oauth/token", post(handle_create_token))
        .route("/api/v1/oauth/introspect", post(handle_introspect_token))
        .route("/api/v1/oauth/revoke", post(handle_revoke_token))
        .layer(TraceLayer::new_for_http())
        .with_state(state)
}

async fn handle_liveness() -> impl IntoResponse {
    Json(serde_json::json!({"status": "alive"}))
}

async fn handle_readiness(State(state): State<AppState>) -> impl IntoResponse {
    if state.tokenginx.is_healthy().await {
        (
            StatusCode::OK,
            Json(serde_json::json!({"status": "ready"})),
        )
    } else {
        (
            StatusCode::SERVICE_UNAVAILABLE,
            Json(serde_json::json!({"status": "not ready"})),
        )
    }
}

async fn handle_metrics(State(state): State<AppState>) -> String {
    state.prometheus.render()
}

#[derive(Deserialize)]
struct CreateTokenRequest {
    user_id: String,
    scope: String,
    ttl_seconds: u64,
}

#[derive(Serialize)]
struct CreateTokenResponse {
    access_token: String,
}

async fn handle_create_token(
    State(state): State<AppState>,
    Json(req): Json<CreateTokenRequest>,
) -> Result<Json<CreateTokenResponse>, AppError> {
    let token_id = uuid::Uuid::new_v4().to_string();

    let token = OAuthToken {
        user_id: req.user_id,
        scope: req.scope,
        created_at: chrono::Utc::now().timestamp(),
        client_ip: None,
    };

    state
        .tokenginx
        .set_oauth_token(&token_id, &token, Duration::from_secs(req.ttl_seconds))
        .await?;

    Ok(Json(CreateTokenResponse {
        access_token: token_id,
    }))
}

#[derive(Deserialize)]
struct IntrospectTokenRequest {
    token: String,
}

#[derive(Serialize)]
struct IntrospectTokenResponse {
    active: bool,
    #[serde(skip_serializing_if = "Option::is_none")]
    user_id: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    scope: Option<String>,
}

async fn handle_introspect_token(
    State(state): State<AppState>,
    Json(req): Json<IntrospectTokenRequest>,
) -> Result<Json<IntrospectTokenResponse>, AppError> {
    let token = state.tokenginx.get_oauth_token(&req.token).await?;

    match token {
        Some(t) => Ok(Json(IntrospectTokenResponse {
            active: true,
            user_id: Some(t.user_id),
            scope: Some(t.scope),
        })),
        None => Ok(Json(IntrospectTokenResponse {
            active: false,
            user_id: None,
            scope: None,
        })),
    }
}

#[derive(Deserialize)]
struct RevokeTokenRequest {
    token: String,
}

async fn handle_revoke_token(
    State(state): State<AppState>,
    Json(req): Json<RevokeTokenRequest>,
) -> Result<StatusCode, AppError> {
    state.tokenginx.delete_token(&req.token).await?;
    Ok(StatusCode::OK)
}

// 错误处理
struct AppError(anyhow::Error);

impl IntoResponse for AppError {
    fn into_response(self) -> axum::response::Response {
        (
            StatusCode::INTERNAL_SERVER_ERROR,
            Json(serde_json::json!({
                "error": self.0.to_string()
            })),
        )
            .into_response()
    }
}

impl<E> From<E> for AppError
where
    E: Into<anyhow::Error>,
{
    fn from(err: E) -> Self {
        Self(err.into())
    }
}
```

### 6. 主程序

创建 `src/main.rs`:

```rust
mod config;
mod server;
mod tokenginx;

use anyhow::Result;
use metrics_exporter_prometheus::PrometheusBuilder;
use std::sync::Arc;
use tokio::signal;
use tracing::info;
use tracing_subscriber::{layer::SubscriberExt, util::SubscriberInitExt};

use crate::config::AppConfig;
use crate::server::{create_router, AppState};
use crate::tokenginx::{TokenginXClient, TokenginXMetrics};

#[tokio::main]
async fn main() -> Result<()> {
    // 加载配置
    let config = AppConfig::load("config/production.toml")?;

    // 初始化日志
    init_tracing(&config.log)?;

    // 初始化 Prometheus
    let prometheus = PrometheusBuilder::new().install_recorder()?;

    // 初始化 TokenginX 客户端
    let metrics = Arc::new(TokenginXMetrics::new());
    let tokenginx = TokenginXClient::new(
        config.tokenginx.nodes.clone(),
        config.tokenginx.password.clone(),
        metrics,
    )
    .await?;

    // 创建应用状态
    let state = AppState {
        tokenginx,
        prometheus,
    };

    // 创建路由
    let app = create_router(state);

    // 启动服务器
    let addr = format!("{}:{}", config.server.host, config.server.port);
    let listener = tokio::net::TcpListener::bind(&addr).await?;

    info!("Starting server on {}", addr);

    axum::serve(listener, app)
        .with_graceful_shutdown(shutdown_signal())
        .await?;

    Ok(())
}

fn init_tracing(config: &config::LogConfig) -> Result<()> {
    let env_filter = tracing_subscriber::EnvFilter::try_from_default_env()
        .unwrap_or_else(|_| tracing_subscriber::EnvFilter::new(&config.level));

    if config.format == "json" {
        tracing_subscriber::registry()
            .with(env_filter)
            .with(tracing_subscriber::fmt::layer().json())
            .init();
    } else {
        tracing_subscriber::registry()
            .with(env_filter)
            .with(tracing_subscriber::fmt::layer())
            .init();
    }

    Ok(())
}

async fn shutdown_signal() {
    let ctrl_c = async {
        signal::ctrl_c()
            .await
            .expect("failed to install Ctrl+C handler");
    };

    #[cfg(unix)]
    let terminate = async {
        signal::unix::signal(signal::unix::SignalKind::terminate())
            .expect("failed to install signal handler")
            .recv()
            .await;
    };

    #[cfg(not(unix))]
    let terminate = std::future::pending::<()>();

    tokio::select! {
        _ = ctrl_c => {},
        _ = terminate => {},
    }

    info!("Shutting down gracefully");
}
```

## Docker 部署

### Dockerfile

```dockerfile
FROM rust:1.75-alpine AS builder

WORKDIR /build

RUN apk add --no-cache musl-dev openssl-dev openssl-libs-static

# 复制 Cargo 文件
COPY Cargo.toml Cargo.lock ./
RUN mkdir src && echo "fn main() {}" > src/main.rs
RUN cargo build --release
RUN rm -rf src

# 复制源代码
COPY src ./src
RUN touch src/main.rs
RUN cargo build --release

FROM alpine:latest

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /build/target/release/myapp /app/myapp
COPY config /app/config

# 非 root 用户
RUN adduser -D -u 1000 appuser
USER appuser

EXPOSE 8080

ENTRYPOINT ["/app/myapp"]
```

## Kubernetes 部署

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: myapp-rust
spec:
  replicas: 3
  selector:
    matchLabels:
      app: myapp-rust
  template:
    metadata:
      labels:
        app: myapp-rust
    spec:
      containers:
      - name: myapp
        image: myapp-rust:1.0.0
        ports:
        - containerPort: 8080
        env:
        - name: TOKENGINX_API_KEY
          valueFrom:
            secretKeyRef:
              name: tokenginx-secret
              key: api-key
        - name: RUST_LOG
          value: "info"
        livenessProbe:
          httpGet:
            path: /health/live
            port: 8080
          initialDelaySeconds: 10
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /health/ready
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 5
        resources:
          requests:
            memory: "128Mi"
            cpu: "100m"
          limits:
            memory: "256Mi"
            cpu: "500m"
```

## 最佳实践清单

- ✅ 使用 Tokio 异步运行时
- ✅ 启用 TLS/mTLS 加密通信
- ✅ 使用结构化日志(tracing)
- ✅ 添加 Prometheus 监控指标
- ✅ 实现优雅关闭(Graceful Shutdown)
- ✅ 使用 Redis 连接管理器
- ✅ 启用 LTO 和代码优化
- ✅ 使用 Alpine 基础镜像减小镜像大小
- ✅ 实现健康检查(Liveness/Readiness)
- ✅ 使用 tower-http 中间件
- ✅ 零成本抽象,高性能

## 下一步

- 查看 [OAuth 2.0 集成](../protocols/oauth.md)
- 配置 [TLS/mTLS](../security/tls-mtls.md)
- 了解 [防重放攻击](../security/anti-replay.md)
