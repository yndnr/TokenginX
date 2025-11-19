# Java 快速指南

本指南帮助您快速在 Java 应用中集成 TokenginX。

## 前置要求

- Java 11 或更高版本
- Maven 或 Gradle
- TokenginX 服务器已运行

## 安装客户端库

### Maven

在 `pom.xml` 中添加依赖：

```xml
<dependencies>
    <!-- Jedis（推荐） -->
    <dependency>
        <groupId>redis.clients</groupId>
        <artifactId>jedis</artifactId>
        <version>5.1.0</version>
    </dependency>

    <!-- 或者使用 Lettuce -->
    <dependency>
        <groupId>io.lettuce</groupId>
        <artifactId>lettuce-core</artifactId>
        <version>6.3.0.RELEASE</version>
    </dependency>

    <!-- JSON 处理 -->
    <dependency>
        <groupId>com.google.code.gson</groupId>
        <artifactId>gson</artifactId>
        <version>2.10.1</version>
    </dependency>
</dependencies>
```

### Gradle

在 `build.gradle` 中添加：

```gradle
dependencies {
    implementation 'redis.clients:jedis:5.1.0'
    // 或 implementation 'io.lettuce:lettuce-core:6.3.0.RELEASE'
    implementation 'com.google.code.gson:gson:2.10.1'
}
```

## 使用 Jedis 客户端（推荐）

### 1. 创建连接池

```java
import redis.clients.jedis.Jedis;
import redis.clients.jedis.JedisPool;
import redis.clients.jedis.JedisPoolConfig;
import redis.clients.jedis.DefaultJedisClientConfig;
import redis.clients.jedis.HostAndPort;

import javax.net.ssl.SSLContext;
import javax.net.ssl.TrustManagerFactory;
import java.io.FileInputStream;
import java.security.KeyStore;

public class TokenginXConfig {
    private static JedisPool jedisPool;

    public static void initialize() {
        JedisPoolConfig poolConfig = new JedisPoolConfig();
        poolConfig.setMaxTotal(128);
        poolConfig.setMaxIdle(64);
        poolConfig.setMinIdle(16);
        poolConfig.setTestOnBorrow(true);
        poolConfig.setTestOnReturn(true);
        poolConfig.setTestWhileIdle(true);
        poolConfig.setNumTestsPerEvictionRun(3);
        poolConfig.setBlockWhenExhausted(true);

        // TLS 配置
        DefaultJedisClientConfig clientConfig = DefaultJedisClientConfig.builder()
            .password("your-api-key")
            .ssl(true)
            .sslSocketFactory(createSSLContext().getSocketFactory())
            .connectionTimeoutMillis(5000)
            .socketTimeoutMillis(5000)
            .build();

        jedisPool = new JedisPool(
            poolConfig,
            new HostAndPort("your-tokenginx-server", 6380),
            clientConfig
        );
    }

    private static SSLContext createSSLContext() {
        try {
            KeyStore trustStore = KeyStore.getInstance("JKS");
            try (FileInputStream fis = new FileInputStream("truststore.jks")) {
                trustStore.load(fis, "changeit".toCharArray());
            }

            TrustManagerFactory tmf = TrustManagerFactory.getInstance(
                TrustManagerFactory.getDefaultAlgorithm());
            tmf.init(trustStore);

            SSLContext sslContext = SSLContext.getInstance("TLS");
            sslContext.init(null, tmf.getTrustManagers(), null);
            return sslContext;
        } catch (Exception e) {
            throw new RuntimeException("Failed to create SSL context", e);
        }
    }

    public static JedisPool getPool() {
        return jedisPool;
    }

    public static void shutdown() {
        if (jedisPool != null) {
            jedisPool.close();
        }
    }
}
```

### 2. 创建服务类

```java
import redis.clients.jedis.Jedis;
import redis.clients.jedis.JedisPool;
import redis.clients.jedis.params.SetParams;
import com.google.gson.Gson;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.util.HashMap;
import java.util.List;
import java.util.Map;

public class TokenginXService {
    private static final Logger logger = LoggerFactory.getLogger(TokenginXService.class);
    private final JedisPool jedisPool;
    private final Gson gson;

    public TokenginXService(JedisPool jedisPool) {
        this.jedisPool = jedisPool;
        this.gson = new Gson();
    }

    // 设置 OAuth Token
    public boolean setOAuthToken(String tokenId, OAuthToken token, int ttlSeconds) {
        try (Jedis jedis = jedisPool.getResource()) {
            String key = "oauth:token:" + tokenId;
            String value = gson.toJson(token);

            SetParams params = SetParams.setParams().ex(ttlSeconds);
            String result = jedis.set(key, value, params);

            return "OK".equals(result);
        } catch (Exception e) {
            logger.error("Failed to set OAuth token", e);
            return false;
        }
    }

    // 获取 OAuth Token
    public OAuthToken getOAuthToken(String tokenId) {
        try (Jedis jedis = jedisPool.getResource()) {
            String key = "oauth:token:" + tokenId;
            String value = jedis.get(key);

            if (value == null) {
                return null;
            }

            return gson.fromJson(value, OAuthToken.class);
        } catch (Exception e) {
            logger.error("Failed to get OAuth token", e);
            return null;
        }
    }

    // 删除 Token
    public boolean deleteToken(String tokenId) {
        try (Jedis jedis = jedisPool.getResource()) {
            String key = "oauth:token:" + tokenId;
            return jedis.del(key) > 0;
        } catch (Exception e) {
            logger.error("Failed to delete token", e);
            return false;
        }
    }

    // 检查 Token 是否存在
    public boolean tokenExists(String tokenId) {
        try (Jedis jedis = jedisPool.getResource()) {
            String key = "oauth:token:" + tokenId;
            return jedis.exists(key);
        } catch (Exception e) {
            logger.error("Failed to check token existence", e);
            return false;
        }
    }

    // 获取剩余 TTL
    public Long getTokenTTL(String tokenId) {
        try (Jedis jedis = jedisPool.getResource()) {
            String key = "oauth:token:" + tokenId;
            return jedis.ttl(key);
        } catch (Exception e) {
            logger.error("Failed to get token TTL", e);
            return -2L;
        }
    }

    // 批量获取 Tokens
    public Map<String, OAuthToken> getMultipleTokens(List<String> tokenIds) {
        Map<String, OAuthToken> result = new HashMap<>();

        try (Jedis jedis = jedisPool.getResource()) {
            String[] keys = tokenIds.stream()
                .map(id -> "oauth:token:" + id)
                .toArray(String[]::new);

            List<String> values = jedis.mget(keys);

            for (int i = 0; i < tokenIds.size(); i++) {
                String value = values.get(i);
                if (value != null) {
                    result.put(tokenIds.get(i), gson.fromJson(value, OAuthToken.class));
                }
            }
        } catch (Exception e) {
            logger.error("Failed to get multiple tokens", e);
        }

        return result;
    }

    // 使用管道批量设置
    public boolean setMultipleTokens(Map<String, OAuthToken> tokens, int ttlSeconds) {
        try (Jedis jedis = jedisPool.getResource()) {
            var pipeline = jedis.pipelined();

            tokens.forEach((tokenId, token) -> {
                String key = "oauth:token:" + tokenId;
                String value = gson.toJson(token);
                pipeline.setex(key, ttlSeconds, value);
            });

            pipeline.sync();
            return true;
        } catch (Exception e) {
            logger.error("Failed to set multiple tokens", e);
            return false;
        }
    }
}

// Token 数据类
public class OAuthToken {
    private String userId;
    private String scope;
    private String clientId;
    private long createdAt;

    // Constructors
    public OAuthToken() {}

    public OAuthToken(String userId, String scope, String clientId) {
        this.userId = userId;
        this.scope = scope;
        this.clientId = clientId;
        this.createdAt = System.currentTimeMillis() / 1000;
    }

    // Getters and Setters
    public String getUserId() { return userId; }
    public void setUserId(String userId) { this.userId = userId; }

    public String getScope() { return scope; }
    public void setScope(String scope) { this.scope = scope; }

    public String getClientId() { return clientId; }
    public void setClientId(String clientId) { this.clientId = clientId; }

    public long getCreatedAt() { return createdAt; }
    public void setCreatedAt(long createdAt) { this.createdAt = createdAt; }
}
```

### 3. 在 Spring Boot 中使用

创建配置类：

```java
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import redis.clients.jedis.JedisPool;

@Configuration
public class TokenginXConfiguration {

    @Bean
    public JedisPool jedisPool() {
        TokenginXConfig.initialize();
        return TokenginXConfig.getPool();
    }

    @Bean
    public TokenginXService tokenginXService(JedisPool jedisPool) {
        return new TokenginXService(jedisPool);
    }
}
```

创建 REST 控制器：

```java
import org.springframework.web.bind.annotation.*;
import org.springframework.http.ResponseEntity;
import org.springframework.http.HttpStatus;

import java.util.HashMap;
import java.util.Map;
import java.util.UUID;

@RestController
@RequestMapping("/api/auth")
public class AuthController {

    private final TokenginXService tokenginXService;

    public AuthController(TokenginXService tokenginXService) {
        this.tokenginXService = tokenginXService;
    }

    @PostMapping("/token")
    public ResponseEntity<Map<String, String>> createToken(@RequestBody TokenRequest request) {
        String tokenId = UUID.randomUUID().toString();

        OAuthToken token = new OAuthToken(
            request.getUserId(),
            request.getScope(),
            request.getClientId()
        );

        boolean success = tokenginXService.setOAuthToken(tokenId, token, 3600);

        if (!success) {
            return ResponseEntity.status(HttpStatus.INTERNAL_SERVER_ERROR).build();
        }

        Map<String, String> response = new HashMap<>();
        response.put("access_token", tokenId);
        response.put("token_type", "Bearer");
        response.put("expires_in", "3600");

        return ResponseEntity.ok(response);
    }

    @PostMapping("/introspect")
    public ResponseEntity<Map<String, Object>> introspectToken(
            @RequestBody IntrospectRequest request) {

        OAuthToken token = tokenginXService.getOAuthToken(request.getToken());
        Map<String, Object> response = new HashMap<>();

        if (token == null) {
            response.put("active", false);
            return ResponseEntity.ok(response);
        }

        Long ttl = tokenginXService.getTokenTTL(request.getToken());

        response.put("active", true);
        response.put("scope", token.getScope());
        response.put("user_id", token.getUserId());
        response.put("client_id", token.getClientId());
        response.put("exp", System.currentTimeMillis() / 1000 + (ttl != null ? ttl : 0));

        return ResponseEntity.ok(response);
    }

    @PostMapping("/revoke")
    public ResponseEntity<Void> revokeToken(@RequestBody RevokeRequest request) {
        tokenginXService.deleteToken(request.getToken());
        return ResponseEntity.ok().build();
    }
}

// Request DTOs
class TokenRequest {
    private String userId;
    private String scope;
    private String clientId;

    // Getters and Setters
    public String getUserId() { return userId; }
    public void setUserId(String userId) { this.userId = userId; }
    public String getScope() { return scope; }
    public void setScope(String scope) { this.scope = scope; }
    public String getClientId() { return clientId; }
    public void setClientId(String clientId) { this.clientId = clientId; }
}

class IntrospectRequest {
    private String token;

    public String getToken() { return token; }
    public void setToken(String token) { this.token = token; }
}

class RevokeRequest {
    private String token;

    public String getToken() { return token; }
    public void setToken(String token) { this.token = token; }
}
```

## 使用 Lettuce 客户端（异步）

```java
import io.lettuce.core.RedisClient;
import io.lettuce.core.RedisURI;
import io.lettuce.core.SslOptions;
import io.lettuce.core.api.StatefulRedisConnection;
import io.lettuce.core.api.async.RedisAsyncCommands;

import java.time.Duration;
import java.util.concurrent.CompletableFuture;

public class LettuceTokenginXService {
    private final RedisClient redisClient;

    public LettuceTokenginXService() {
        RedisURI redisUri = RedisURI.builder()
            .withHost("your-tokenginx-server")
            .withPort(6380)
            .withPassword("your-api-key".toCharArray())
            .withSsl(true)
            .withTimeout(Duration.ofSeconds(5))
            .build();

        SslOptions sslOptions = SslOptions.builder()
            .truststore(new File("truststore.jks"), "changeit")
            .build();

        this.redisClient = RedisClient.create(redisUri);
        this.redisClient.setOptions(ClientOptions.builder()
            .sslOptions(sslOptions)
            .build());
    }

    public CompletableFuture<Boolean> setOAuthTokenAsync(
            String tokenId, OAuthToken token, int ttlSeconds) {

        try (StatefulRedisConnection<String, String> connection = redisClient.connect()) {
            RedisAsyncCommands<String, String> async = connection.async();
            String key = "oauth:token:" + tokenId;
            String value = new Gson().toJson(token);

            return async.setex(key, ttlSeconds, value)
                .thenApply("OK"::equals)
                .toCompletableFuture();
        }
    }

    public CompletableFuture<OAuthToken> getOAuthTokenAsync(String tokenId) {
        try (StatefulRedisConnection<String, String> connection = redisClient.connect()) {
            RedisAsyncCommands<String, String> async = connection.async();
            String key = "oauth:token:" + tokenId;

            return async.get(key)
                .thenApply(value -> {
                    if (value == null) return null;
                    return new Gson().fromJson(value, OAuthToken.class);
                })
                .toCompletableFuture();
        }
    }

    public void shutdown() {
        redisClient.shutdown();
    }
}
```

## 使用 mTLS 认证

```java
import javax.net.ssl.*;
import java.io.FileInputStream;
import java.security.KeyStore;

public class MTLSConfig {
    public static SSLSocketFactory createMTLSSocketFactory(
            String keystorePath,
            String keystorePassword,
            String truststorePath,
            String truststorePassword) throws Exception {

        // 加载客户端证书
        KeyStore keyStore = KeyStore.getInstance("JKS");
        try (FileInputStream kis = new FileInputStream(keystorePath)) {
            keyStore.load(kis, keystorePassword.toCharArray());
        }

        KeyManagerFactory kmf = KeyManagerFactory.getInstance(
            KeyManagerFactory.getDefaultAlgorithm());
        kmf.init(keyStore, keystorePassword.toCharArray());

        // 加载信任库
        KeyStore trustStore = KeyStore.getInstance("JKS");
        try (FileInputStream tis = new FileInputStream(truststorePath)) {
            trustStore.load(tis, truststorePassword.toCharArray());
        }

        TrustManagerFactory tmf = TrustManagerFactory.getInstance(
            TrustManagerFactory.getDefaultAlgorithm());
        tmf.init(trustStore);

        // 创建 SSL 上下文
        SSLContext sslContext = SSLContext.getInstance("TLS");
        sslContext.init(kmf.getKeyManagers(), tmf.getTrustManagers(), null);

        return sslContext.getSocketFactory();
    }
}
```

## Spring Boot 配置属性

在 `application.yml` 中：

```yaml
tokenginx:
  host: your-tokenginx-server
  port: 6380
  password: your-api-key
  ssl:
    enabled: true
    truststore: classpath:truststore.jks
    truststore-password: changeit
  pool:
    max-total: 128
    max-idle: 64
    min-idle: 16
```

## 错误处理和重试

```java
import redis.clients.jedis.exceptions.JedisConnectionException;
import redis.clients.jedis.exceptions.JedisDataException;

public OAuthToken getOAuthTokenWithRetry(String tokenId, int maxRetries) {
    int retries = 0;
    while (retries < maxRetries) {
        try {
            return getOAuthToken(tokenId);
        } catch (JedisConnectionException e) {
            retries++;
            logger.warn("Connection error, retry {}/{}", retries, maxRetries, e);
            if (retries >= maxRetries) {
                throw e;
            }
            try {
                Thread.sleep(100 * retries);
            } catch (InterruptedException ie) {
                Thread.currentThread().interrupt();
                throw new RuntimeException(ie);
            }
        } catch (JedisDataException e) {
            logger.error("Data error", e);
            throw e;
        }
    }
    return null;
}
```

## 下一步

- 查看 [Java 生产环境指南](../production/java.md)
- 了解 [gRPC API 参考](../reference/grpc-api.md)
- 配置 [国密支持](../security/gm-crypto.md)
