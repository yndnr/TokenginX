# Rust 快速指南

本指南帮助您快速在 Rust 应用中集成 TokenginX。

## 前置要求

- Rust 1.70 或更高版本
- Cargo
- TokenginX 服务器已运行

## 安装客户端库

在 `Cargo.toml` 中添加依赖:

```toml
[dependencies]
redis = { version = "0.24", features = ["tokio-comp", "tls-native-tls"] }
tokio = { version = "1", features = ["full"] }
serde = { version = "1.0", features = ["derive"] }
serde_json = "1.0"
anyhow = "1.0"
```

## 使用 Redis 客户端

### 1. 创建客户端模块

创建 `src/tokenginx.rs`:

```rust
use anyhow::{Context, Result};
use redis::{Client, Commands, Connection};
use serde::{Deserialize, Serialize};
use std::time::Duration;

/// OAuth Token 数据结构
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct OAuthToken {
    pub user_id: String,
    pub scope: String,
    pub created_at: i64,
}

/// TokenginX 客户端
pub struct TokenginXClient {
    client: Client,
}

impl TokenginXClient {
    /// 创建新的 TokenginX 客户端
    ///
    /// # 参数
    ///
    /// * `addr` - 服务器地址 (例如: "redis://localhost:6380")
    /// * `password` - API Key
    ///
    /// # 返回
    ///
    /// * `Result<Self>` - 客户端实例或错误
    ///
    /// # 示例
    ///
    /// ```
    /// let client = TokenginXClient::new("redis://localhost:6380", "your-api-key")?;
    /// ```
    pub fn new(addr: &str, password: &str) -> Result<Self> {
        let client = Client::open(format!("{}?password={}", addr, password))
            .context("Failed to create Redis client")?;

        Ok(Self { client })
    }

    /// 创建带 TLS 的客户端
    ///
    /// # 参数
    ///
    /// * `addr` - 服务器地址 (例如: "rediss://localhost:6380")
    /// * `password` - API Key
    ///
    /// # 返回
    ///
    /// * `Result<Self>` - 客户端实例或错误
    pub fn new_with_tls(addr: &str, password: &str) -> Result<Self> {
        let client = Client::open(format!("{}?password={}", addr, password))
            .context("Failed to create Redis client with TLS")?;

        Ok(Self { client })
    }

    /// 获取连接
    fn get_connection(&self) -> Result<Connection> {
        self.client
            .get_connection()
            .context("Failed to get connection")
    }

    /// 设置 OAuth Token
    ///
    /// # 参数
    ///
    /// * `token_id` - Token 唯一标识
    /// * `token` - Token 数据
    /// * `ttl` - 过期时间(秒)
    ///
    /// # 返回
    ///
    /// * `Result<()>` - 成功或错误
    ///
    /// # 示例
    ///
    /// ```
    /// let token = OAuthToken {
    ///     user_id: "user001".to_string(),
    ///     scope: "read write".to_string(),
    ///     created_at: chrono::Utc::now().timestamp(),
    /// };
    /// client.set_oauth_token("abc123", &token, 3600)?;
    /// ```
    pub fn set_oauth_token(&self, token_id: &str, token: &OAuthToken, ttl: usize) -> Result<()> {
        let mut conn = self.get_connection()?;
        let key = format!("oauth:token:{}", token_id);
        let value = serde_json::to_string(token).context("Failed to serialize token")?;

        conn.set_ex(&key, value, ttl)
            .context("Failed to set token")?;

        Ok(())
    }

    /// 获取 OAuth Token
    ///
    /// # 参数
    ///
    /// * `token_id` - Token 唯一标识
    ///
    /// # 返回
    ///
    /// * `Result<Option<OAuthToken>>` - Token 数据,不存在时返回 None
    ///
    /// # 示例
    ///
    /// ```
    /// match client.get_oauth_token("abc123")? {
    ///     Some(token) => println!("User ID: {}", token.user_id),
    ///     None => println!("Token not found or expired"),
    /// }
    /// ```
    pub fn get_oauth_token(&self, token_id: &str) -> Result<Option<OAuthToken>> {
        let mut conn = self.get_connection()?;
        let key = format!("oauth:token:{}", token_id);

        let value: Option<String> = conn.get(&key).context("Failed to get token")?;

        match value {
            Some(v) => {
                let token: OAuthToken =
                    serde_json::from_str(&v).context("Failed to deserialize token")?;
                Ok(Some(token))
            }
            None => Ok(None),
        }
    }

    /// 删除 Token
    ///
    /// # 参数
    ///
    /// * `token_id` - Token 唯一标识
    ///
    /// # 返回
    ///
    /// * `Result<bool>` - 是否成功删除
    pub fn delete_token(&self, token_id: &str) -> Result<bool> {
        let mut conn = self.get_connection()?;
        let key = format!("oauth:token:{}", token_id);

        let count: i32 = conn.del(&key).context("Failed to delete token")?;

        Ok(count > 0)
    }

    /// 检查 Token 是否存在
    ///
    /// # 参数
    ///
    /// * `token_id` - Token 唯一标识
    ///
    /// # 返回
    ///
    /// * `Result<bool>` - 是否存在
    pub fn token_exists(&self, token_id: &str) -> Result<bool> {
        let mut conn = self.get_connection()?;
        let key = format!("oauth:token:{}", token_id);

        conn.exists(&key).context("Failed to check token existence")
    }

    /// 获取 Token 剩余 TTL
    ///
    /// # 参数
    ///
    /// * `token_id` - Token 唯一标识
    ///
    /// # 返回
    ///
    /// * `Result<Option<i64>>` - 剩余秒数,不存在时返回 None
    pub fn get_token_ttl(&self, token_id: &str) -> Result<Option<i64>> {
        let mut conn = self.get_connection()?;
        let key = format!("oauth:token:{}", token_id);

        let ttl: i64 = conn.ttl(&key).context("Failed to get TTL")?;

        Ok(if ttl > 0 { Some(ttl) } else { None })
    }

    /// 批量获取 Tokens
    ///
    /// # 参数
    ///
    /// * `token_ids` - Token ID 列表
    ///
    /// # 返回
    ///
    /// * `Result<Vec<Option<OAuthToken>>>` - Token 列表,不存在的为 None
    pub fn get_multiple_tokens(&self, token_ids: &[String]) -> Result<Vec<Option<OAuthToken>>> {
        let mut conn = self.get_connection()?;

        let keys: Vec<String> = token_ids
            .iter()
            .map(|id| format!("oauth:token:{}", id))
            .collect();

        let values: Vec<Option<String>> = conn.get(&keys).context("Failed to get multiple tokens")?;

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

        tokens
    }

    /// 扫描匹配的键
    ///
    /// # 参数
    ///
    /// * `pattern` - 匹配模式
    ///
    /// # 返回
    ///
    /// * `Result<Vec<String>>` - 匹配的键列表
    pub fn scan_keys(&self, pattern: &str) -> Result<Vec<String>> {
        let mut conn = self.get_connection()?;
        let mut keys = Vec::new();
        let mut cursor = 0;

        loop {
            let (new_cursor, batch): (u64, Vec<String>) = redis::cmd("SCAN")
                .arg(cursor)
                .arg("MATCH")
                .arg(pattern)
                .arg("COUNT")
                .arg(100)
                .query(&mut conn)
                .context("Failed to scan keys")?;

            keys.extend(batch);
            cursor = new_cursor;

            if cursor == 0 {
                break;
            }
        }

        Ok(keys)
    }
}
```

### 2. 异步版本(使用 Tokio)

创建 `src/tokenginx_async.rs`:

```rust
use anyhow::{Context, Result};
use redis::aio::Connection;
use redis::{AsyncCommands, Client};
use serde::{Deserialize, Serialize};

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct OAuthToken {
    pub user_id: String,
    pub scope: String,
    pub created_at: i64,
}

/// 异步 TokenginX 客户端
pub struct TokenginXAsyncClient {
    client: Client,
}

impl TokenginXAsyncClient {
    /// 创建新的异步客户端
    pub fn new(addr: &str, password: &str) -> Result<Self> {
        let client = Client::open(format!("{}?password={}", addr, password))
            .context("Failed to create Redis client")?;

        Ok(Self { client })
    }

    /// 获取异步连接
    async fn get_connection(&self) -> Result<Connection> {
        self.client
            .get_async_connection()
            .await
            .context("Failed to get async connection")
    }

    /// 设置 OAuth Token
    pub async fn set_oauth_token(
        &self,
        token_id: &str,
        token: &OAuthToken,
        ttl: usize,
    ) -> Result<()> {
        let mut conn = self.get_connection().await?;
        let key = format!("oauth:token:{}", token_id);
        let value = serde_json::to_string(token).context("Failed to serialize token")?;

        conn.set_ex(&key, value, ttl)
            .await
            .context("Failed to set token")?;

        Ok(())
    }

    /// 获取 OAuth Token
    pub async fn get_oauth_token(&self, token_id: &str) -> Result<Option<OAuthToken>> {
        let mut conn = self.get_connection().await?;
        let key = format!("oauth:token:{}", token_id);

        let value: Option<String> = conn.get(&key).await.context("Failed to get token")?;

        match value {
            Some(v) => {
                let token: OAuthToken =
                    serde_json::from_str(&v).context("Failed to deserialize token")?;
                Ok(Some(token))
            }
            None => Ok(None),
        }
    }

    /// 删除 Token
    pub async fn delete_token(&self, token_id: &str) -> Result<bool> {
        let mut conn = self.get_connection().await?;
        let key = format!("oauth:token:{}", token_id);

        let count: i32 = conn.del(&key).await.context("Failed to delete token")?;

        Ok(count > 0)
    }

    /// 检查 Token 是否存在
    pub async fn token_exists(&self, token_id: &str) -> Result<bool> {
        let mut conn = self.get_connection().await?;
        let key = format!("oauth:token:{}", token_id);

        conn.exists(&key)
            .await
            .context("Failed to check token existence")
    }

    /// 获取 Token 剩余 TTL
    pub async fn get_token_ttl(&self, token_id: &str) -> Result<Option<i64>> {
        let mut conn = self.get_connection().await?;
        let key = format!("oauth:token:{}", token_id);

        let ttl: i64 = conn.ttl(&key).await.context("Failed to get TTL")?;

        Ok(if ttl > 0 { Some(ttl) } else { None })
    }
}
```

### 3. 在应用中使用

创建 `src/main.rs`:

```rust
mod tokenginx;

use anyhow::Result;
use tokenginx::{OAuthToken, TokenginXClient};

fn main() -> Result<()> {
    // 创建客户端
    let client = TokenginXClient::new_with_tls(
        "rediss://localhost:6380",
        &std::env::var("TOKENGINX_API_KEY")?,
    )?;

    // 创建 Token
    let token_id = generate_token_id();
    let token = OAuthToken {
        user_id: "user001".to_string(),
        scope: "read write".to_string(),
        created_at: chrono::Utc::now().timestamp(),
    };

    client.set_oauth_token(&token_id, &token, 3600)?;
    println!("Token created: {}", token_id);

    // 验证 Token
    match client.get_oauth_token(&token_id)? {
        Some(retrieved_token) => {
            println!("Token valid:");
            println!("  User ID: {}", retrieved_token.user_id);
            println!("  Scope: {}", retrieved_token.scope);

            if let Some(ttl) = client.get_token_ttl(&token_id)? {
                println!("  TTL: {} seconds", ttl);
            }
        }
        None => println!("Token not found or expired"),
    }

    // 撤销 Token
    client.delete_token(&token_id)?;
    println!("Token revoked");

    Ok(())
}

fn generate_token_id() -> String {
    use rand::Rng;
    let mut rng = rand::thread_rng();
    let bytes: Vec<u8> = (0..16).map(|_| rng.gen()).collect();
    hex::encode(bytes)
}
```

### 4. 异步版本示例

```rust
mod tokenginx_async;

use anyhow::Result;
use tokenginx_async::{OAuthToken, TokenginXAsyncClient};

#[tokio::main]
async fn main() -> Result<()> {
    // 创建异步客户端
    let client = TokenginXAsyncClient::new(
        "rediss://localhost:6380",
        &std::env::var("TOKENGINX_API_KEY")?,
    )?;

    // 创建 Token
    let token_id = generate_token_id();
    let token = OAuthToken {
        user_id: "user001".to_string(),
        scope: "read write".to_string(),
        created_at: chrono::Utc::now().timestamp(),
    };

    client.set_oauth_token(&token_id, &token, 3600).await?;
    println!("Token created: {}", token_id);

    // 验证 Token
    match client.get_oauth_token(&token_id).await? {
        Some(retrieved_token) => {
            println!("Token valid:");
            println!("  User ID: {}", retrieved_token.user_id);
            println!("  Scope: {}", retrieved_token.scope);
        }
        None => println!("Token not found or expired"),
    }

    // 撤销 Token
    client.delete_token(&token_id).await?;
    println!("Token revoked");

    Ok(())
}
```

## Actix-web 集成

创建 HTTP 服务:

```rust
use actix_web::{web, App, HttpResponse, HttpServer, Result};
use serde::{Deserialize, Serialize};
use std::sync::Arc;
use tokenginx_async::{OAuthToken, TokenginXAsyncClient};

/// 应用状态
struct AppState {
    tokenginx: TokenginXAsyncClient,
}

#[derive(Deserialize)]
struct CreateTokenRequest {
    user_id: String,
    scope: String,
}

#[derive(Serialize)]
struct CreateTokenResponse {
    access_token: String,
}

/// 创建 Token 端点
async fn create_token(
    data: web::Data<Arc<AppState>>,
    req: web::Json<CreateTokenRequest>,
) -> Result<HttpResponse> {
    let token_id = generate_token_id();
    let token = OAuthToken {
        user_id: req.user_id.clone(),
        scope: req.scope.clone(),
        created_at: chrono::Utc::now().timestamp(),
    };

    data.tokenginx
        .set_oauth_token(&token_id, &token, 3600)
        .await
        .map_err(|e| actix_web::error::ErrorInternalServerError(e))?;

    Ok(HttpResponse::Ok().json(CreateTokenResponse {
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

/// Token 内省端点
async fn introspect_token(
    data: web::Data<Arc<AppState>>,
    req: web::Json<IntrospectTokenRequest>,
) -> Result<HttpResponse> {
    let token = data
        .tokenginx
        .get_oauth_token(&req.token)
        .await
        .map_err(|e| actix_web::error::ErrorInternalServerError(e))?;

    match token {
        Some(t) => Ok(HttpResponse::Ok().json(IntrospectTokenResponse {
            active: true,
            user_id: Some(t.user_id),
            scope: Some(t.scope),
        })),
        None => Ok(HttpResponse::Ok().json(IntrospectTokenResponse {
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

/// 撤销 Token 端点
async fn revoke_token(
    data: web::Data<Arc<AppState>>,
    req: web::Json<RevokeTokenRequest>,
) -> Result<HttpResponse> {
    data.tokenginx
        .delete_token(&req.token)
        .await
        .map_err(|e| actix_web::error::ErrorInternalServerError(e))?;

    Ok(HttpResponse::Ok().json(serde_json::json!({"message": "Token revoked"})))
}

#[actix_web::main]
async fn main() -> std::io::Result<()> {
    // 创建 TokenginX 客户端
    let client = TokenginXAsyncClient::new(
        "rediss://localhost:6380",
        &std::env::var("TOKENGINX_API_KEY").unwrap(),
    )
    .unwrap();

    let app_state = Arc::new(AppState { tokenginx: client });

    HttpServer::new(move || {
        App::new()
            .app_data(web::Data::new(app_state.clone()))
            .route("/oauth/token", web::post().to(create_token))
            .route("/oauth/introspect", web::post().to(introspect_token))
            .route("/oauth/revoke", web::post().to(revoke_token))
    })
    .bind(("127.0.0.1", 8080))?
    .run()
    .await
}
```

## 使用 mTLS 认证

```rust
use redis::Client;
use std::fs::File;
use std::io::Read;

fn create_tls_client() -> Result<Client> {
    // 读取客户端证书
    let mut cert_file = File::open("/path/to/client-cert.pem")?;
    let mut cert = Vec::new();
    cert_file.read_to_end(&mut cert)?;

    // 读取客户端私钥
    let mut key_file = File::open("/path/to/client-key.pem")?;
    let mut key = Vec::new();
    key_file.read_to_end(&mut key)?;

    // 创建 TLS 配置
    let tls_config = rustls::ClientConfig::builder()
        .with_safe_defaults()
        .with_root_certificates(load_ca_certs()?)
        .with_client_auth_cert(cert, key)?;

    // 创建 Redis 客户端
    let client = Client::open("rediss://localhost:6380")?;

    Ok(client)
}
```

## 错误处理与重试

```rust
use anyhow::Result;
use std::time::Duration;
use tokio::time::sleep;

impl TokenginXAsyncClient {
    /// 带重试的获取 Token
    pub async fn get_oauth_token_with_retry(
        &self,
        token_id: &str,
        max_retries: usize,
    ) -> Result<Option<OAuthToken>> {
        let mut last_error = None;

        for i in 0..max_retries {
            match self.get_oauth_token(token_id).await {
                Ok(token) => return Ok(token),
                Err(e) => {
                    eprintln!("Retry {}/{}: {}", i + 1, max_retries, e);
                    last_error = Some(e);

                    if i < max_retries - 1 {
                        sleep(Duration::from_millis(100 * (i as u64 + 1))).await;
                    }
                }
            }
        }

        Err(last_error.unwrap())
    }
}
```

## 连接池配置

```rust
use deadpool_redis::{Config, Pool, Runtime};

async fn create_pool() -> Result<Pool> {
    let cfg = Config::from_url("rediss://localhost:6380");

    let pool = cfg
        .builder()?
        .max_size(50)
        .build()?;

    Ok(pool)
}

async fn use_pool(pool: &Pool) -> Result<()> {
    let mut conn = pool.get().await?;

    let value: String = redis::cmd("GET")
        .arg("oauth:token:abc123")
        .query_async(&mut conn)
        .await?;

    Ok(())
}
```

## 测试

创建 `tests/integration_test.rs`:

```rust
#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_set_and_get_token() -> Result<()> {
        let client = TokenginXClient::new("redis://localhost:6380", "test-key")?;

        let token_id = "test-token";
        let token = OAuthToken {
            user_id: "user001".to_string(),
            scope: "read write".to_string(),
            created_at: chrono::Utc::now().timestamp(),
        };

        client.set_oauth_token(token_id, &token, 60)?;

        let retrieved = client.get_oauth_token(token_id)?;
        assert!(retrieved.is_some());

        let retrieved_token = retrieved.unwrap();
        assert_eq!(retrieved_token.user_id, "user001");
        assert_eq!(retrieved_token.scope, "read write");

        client.delete_token(token_id)?;
        Ok(())
    }

    #[tokio::test]
    async fn test_async_operations() -> Result<()> {
        let client = TokenginXAsyncClient::new("redis://localhost:6380", "test-key")?;

        let token_id = "async-test-token";
        let token = OAuthToken {
            user_id: "user002".to_string(),
            scope: "read".to_string(),
            created_at: chrono::Utc::now().timestamp(),
        };

        client.set_oauth_token(token_id, &token, 60).await?;

        assert!(client.token_exists(token_id).await?);

        let ttl = client.get_token_ttl(token_id).await?;
        assert!(ttl.is_some());
        assert!(ttl.unwrap() > 0);

        client.delete_token(token_id).await?;
        assert!(!client.token_exists(token_id).await?);

        Ok(())
    }
}
```

## 下一步

- 查看 [Rust 生产环境指南](../production/rust.md)
- 了解 [OAuth 2.0 集成](../protocols/oauth.md)
- 配置 [TLS/mTLS](../security/tls-mtls.md)
