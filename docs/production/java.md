# Java 生产环境指南

本指南帮助您在生产环境中部署和优化 Java 应用与 TokenginX 的集成。

## 前置要求

- Java 17 或更高版本(Java 21 LTS 推荐)
- Spring Boot 3.x 或更高版本
- 生产环境 TokenginX 服务器集群
- 监控和日志基础设施

## 生产级配置

### 1. Maven 依赖

在 `pom.xml` 中添加:

```xml
<dependencies>
    <!-- Redis 客户端 -->
    <dependency>
        <groupId>io.lettuce</groupId>
        <artifactId>lettuce-core</artifactId>
        <version>6.3.0.RELEASE</version>
    </dependency>

    <!-- Spring Boot Redis -->
    <dependency>
        <groupId>org.springframework.boot</groupId>
        <artifactId>spring-boot-starter-data-redis</artifactId>
    </dependency>

    <!-- 连接池 -->
    <dependency>
        <groupId>org.apache.commons</groupId>
        <artifactId>commons-pool2</artifactId>
    </dependency>

    <!-- JSON 处理 -->
    <dependency>
        <groupId>com.google.code.gson</groupId>
        <artifactId>gson</artifactId>
        <version>2.10.1</version>
    </dependency>

    <!-- 监控 -->
    <dependency>
        <groupId>org.springframework.boot</groupId>
        <artifactId>spring-boot-starter-actuator</artifactId>
    </dependency>

    <!-- Micrometer Prometheus -->
    <dependency>
        <groupId>io.micrometer</groupId>
        <artifactId>micrometer-registry-prometheus</artifactId>
    </dependency>
</dependencies>
```

### 2. 应用配置

在 `application-production.yml` 中:

```yaml
spring:
  application:
    name: myapp-production

  data:
    redis:
      cluster:
        nodes:
          - tokenginx-node1.prod.example.com:6380
          - tokenginx-node2.prod.example.com:6380
          - tokenginx-node3.prod.example.com:6380
        max-redirects: 3
      password: ${TOKENGINX_API_KEY}
      ssl:
        enabled: true
        bundle: tokenginx
      timeout: 3000ms
      connect-timeout: 5000ms
      lettuce:
        pool:
          max-active: 50
          max-idle: 20
          min-idle: 10
          max-wait: 2000ms
        cluster:
          refresh:
            adaptive: true
            period: 60s

  ssl:
    bundle:
      jks:
        tokenginx:
          key:
            alias: client
          keystore:
            location: file:/certs/client-keystore.jks
            password: ${KEYSTORE_PASSWORD}
            type: JKS
          truststore:
            location: file:/certs/truststore.jks
            password: ${TRUSTSTORE_PASSWORD}
            type: JKS

# 监控配置
management:
  endpoints:
    web:
      exposure:
        include: health,info,prometheus,metrics
  endpoint:
    health:
      show-details: when-authorized
  metrics:
    distribution:
      percentiles-histogram:
        http.server.requests: true
    tags:
      application: ${spring.application.name}
      environment: production

# 日志配置
logging:
  level:
    root: INFO
    com.myapp: INFO
    io.lettuce.core: WARN
  pattern:
    console: "%d{yyyy-MM-dd HH:mm:ss} [%thread] %-5level %logger{36} - %msg%n"
  file:
    name: /var/log/myapp/application.log
    max-size: 100MB
    max-history: 30
```

### 3. 连接配置类

创建 `TokenginXConfig.java`:

```java
package com.myapp.config;

import io.lettuce.core.ClientOptions;
import io.lettuce.core.SocketOptions;
import io.lettuce.core.TimeoutOptions;
import io.lettuce.core.cluster.ClusterClientOptions;
import io.lettuce.core.cluster.ClusterTopologyRefreshOptions;
import org.springframework.boot.autoconfigure.data.redis.LettuceClientConfigurationBuilderCustomizer;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.data.redis.connection.RedisConnectionFactory;
import org.springframework.data.redis.connection.lettuce.LettuceConnectionFactory;
import org.springframework.data.redis.core.RedisTemplate;
import org.springframework.data.redis.serializer.GenericJackson2JsonRedisSerializer;
import org.springframework.data.redis.serializer.StringRedisSerializer;

import java.time.Duration;

@Configuration
public class TokenginXConfig {

    @Bean
    public LettuceClientConfigurationBuilderCustomizer lettuceClientConfigurationBuilderCustomizer() {
        return clientConfigurationBuilder -> {
            // Socket 配置
            SocketOptions socketOptions = SocketOptions.builder()
                .connectTimeout(Duration.ofSeconds(5))
                .keepAlive(true)
                .build();

            // 超时配置
            TimeoutOptions timeoutOptions = TimeoutOptions.builder()
                .fixedTimeout(Duration.ofSeconds(3))
                .build();

            // 集群拓扑刷新配置
            ClusterTopologyRefreshOptions refreshOptions = ClusterTopologyRefreshOptions.builder()
                .enablePeriodicRefresh(Duration.ofSeconds(60))
                .enableAllAdaptiveRefreshTriggers()
                .build();

            // 客户端配置
            ClusterClientOptions clientOptions = ClusterClientOptions.builder()
                .socketOptions(socketOptions)
                .timeoutOptions(timeoutOptions)
                .topologyRefreshOptions(refreshOptions)
                .autoReconnect(true)
                .cancelCommandsOnReconnectFailure(false)
                .disconnectedBehavior(ClientOptions.DisconnectedBehavior.REJECT_COMMANDS)
                .build();

            clientConfigurationBuilder.clientOptions(clientOptions);
        };
    }

    @Bean
    public RedisTemplate<String, Object> redisTemplate(RedisConnectionFactory connectionFactory) {
        RedisTemplate<String, Object> template = new RedisTemplate<>();
        template.setConnectionFactory(connectionFactory);

        // 序列化配置
        StringRedisSerializer stringSerializer = new StringRedisSerializer();
        GenericJackson2JsonRedisSerializer jsonSerializer = new GenericJackson2JsonRedisSerializer();

        template.setKeySerializer(stringSerializer);
        template.setValueSerializer(jsonSerializer);
        template.setHashKeySerializer(stringSerializer);
        template.setHashValueSerializer(jsonSerializer);

        template.afterPropertiesSet();
        return template;
    }
}
```

### 4. 生产级服务实现

创建 `TokenginXService.java`:

```java
package com.myapp.service;

import com.google.gson.Gson;
import io.micrometer.core.instrument.Counter;
import io.micrometer.core.instrument.MeterRegistry;
import io.micrometer.core.instrument.Timer;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.dao.DataAccessException;
import org.springframework.data.redis.RedisConnectionFailureException;
import org.springframework.data.redis.core.RedisOperations;
import org.springframework.data.redis.core.RedisTemplate;
import org.springframework.data.redis.core.SessionCallback;
import org.springframework.retry.annotation.Backoff;
import org.springframework.retry.annotation.Retryable;
import org.springframework.stereotype.Service;

import java.time.Duration;
import java.time.Instant;
import java.util.HashMap;
import java.util.List;
import java.util.Map;
import java.util.concurrent.TimeUnit;

@Service
public class TokenginXService {

    private static final Logger log = LoggerFactory.getLogger(TokenginXService.class);
    private final RedisTemplate<String, Object> redisTemplate;
    private final Gson gson;

    // Prometheus 指标
    private final Counter tokenCreatedCounter;
    private final Counter tokenRetrievedCounter;
    private final Timer operationTimer;

    public TokenginXService(
            RedisTemplate<String, Object> redisTemplate,
            MeterRegistry meterRegistry) {
        this.redisTemplate = redisTemplate;
        this.gson = new Gson();

        // 初始化监控指标
        this.tokenCreatedCounter = Counter.builder("tokenginx.token.created")
            .description("Total tokens created")
            .register(meterRegistry);

        this.tokenRetrievedCounter = Counter.builder("tokenginx.token.retrieved")
            .description("Total tokens retrieved")
            .register(meterRegistry);

        this.operationTimer = Timer.builder("tokenginx.operation.duration")
            .description("Duration of TokenginX operations")
            .publishPercentiles(0.5, 0.95, 0.99)
            .register(meterRegistry);
    }

    /**
     * 设置 OAuth Token(带重试和监控)
     */
    @Retryable(
        retryFor = {RedisConnectionFailureException.class},
        maxAttempts = 3,
        backoff = @Backoff(delay = 100, multiplier = 2)
    )
    public Result<Boolean> setOAuthToken(
            String tokenId,
            String userId,
            String scope,
            int ttlSeconds) {

        return operationTimer.record(() -> {
            try {
                OAuthToken token = new OAuthToken();
                token.setUserId(userId);
                token.setScope(scope);
                token.setCreatedAt(Instant.now().getEpochSecond());

                String key = "oauth:token:" + tokenId;
                String value = gson.toJson(token);

                Boolean success = redisTemplate.opsForValue()
                    .set(key, value, Duration.ofSeconds(ttlSeconds));

                if (Boolean.TRUE.equals(success)) {
                    tokenCreatedCounter.increment();
                    log.info("OAuth token created: tokenId={}, userId={}, ttl={}s",
                        tokenId, userId, ttlSeconds);
                    return Result.success(true);
                } else {
                    log.warn("Failed to create OAuth token: tokenId={}", tokenId);
                    return Result.failure("Failed to set token");
                }
            } catch (Exception e) {
                log.error("Error setting OAuth token: tokenId={}", tokenId, e);
                return Result.failure("Error: " + e.getMessage());
            }
        });
    }

    /**
     * 获取 OAuth Token
     */
    @Retryable(
        retryFor = {RedisConnectionFailureException.class},
        maxAttempts = 3,
        backoff = @Backoff(delay = 100, multiplier = 2)
    )
    public Result<OAuthToken> getOAuthToken(String tokenId) {
        return operationTimer.record(() -> {
            try {
                String key = "oauth:token:" + tokenId;
                Object value = redisTemplate.opsForValue().get(key);

                if (value == null) {
                    log.debug("OAuth token not found: tokenId={}", tokenId);
                    return Result.failure("Token not found");
                }

                OAuthToken token = gson.fromJson(value.toString(), OAuthToken.class);
                tokenRetrievedCounter.increment();

                log.debug("OAuth token retrieved: tokenId={}", tokenId);
                return Result.success(token);
            } catch (Exception e) {
                log.error("Error getting OAuth token: tokenId={}", tokenId, e);
                return Result.failure("Error: " + e.getMessage());
            }
        });
    }

    /**
     * 批量获取 Token(使用 Pipeline)
     */
    public Result<Map<String, OAuthToken>> getMultipleTokens(List<String> tokenIds) {
        return operationTimer.record(() -> {
            try {
                List<Object> results = redisTemplate.executePipelined(
                    new SessionCallback<Object>() {
                        @Override
                        public <K, V> Object execute(RedisOperations<K, V> operations)
                                throws DataAccessException {
                            RedisOperations<String, Object> ops =
                                (RedisOperations<String, Object>) operations;

                            for (String tokenId : tokenIds) {
                                String key = "oauth:token:" + tokenId;
                                ops.opsForValue().get(key);
                            }
                            return null;
                        }
                    }
                );

                Map<String, OAuthToken> tokenMap = new HashMap<>();
                for (int i = 0; i < tokenIds.size(); i++) {
                    Object value = results.get(i);
                    if (value != null) {
                        OAuthToken token = gson.fromJson(value.toString(), OAuthToken.class);
                        tokenMap.put(tokenIds.get(i), token);
                    }
                }

                log.debug("Batch retrieved {} tokens, {} found",
                    tokenIds.size(), tokenMap.size());

                return Result.success(tokenMap);
            } catch (Exception e) {
                log.error("Error getting multiple tokens", e);
                return Result.failure("Batch get failed: " + e.getMessage());
            }
        });
    }

    /**
     * 删除 Token
     */
    @Retryable(
        retryFor = {RedisConnectionFailureException.class},
        maxAttempts = 3,
        backoff = @Backoff(delay = 100, multiplier = 2)
    )
    public Result<Boolean> deleteToken(String tokenId) {
        try {
            String key = "oauth:token:" + tokenId;
            Boolean deleted = redisTemplate.delete(key);

            log.info("Token deleted: tokenId={}, success={}", tokenId, deleted);
            return Result.success(Boolean.TRUE.equals(deleted));
        } catch (Exception e) {
            log.error("Error deleting token: tokenId={}", tokenId, e);
            return Result.failure("Error: " + e.getMessage());
        }
    }

    /**
     * 获取 Token TTL
     */
    public Result<Long> getTokenTTL(String tokenId) {
        try {
            String key = "oauth:token:" + tokenId;
            Long ttl = redisTemplate.getExpire(key, TimeUnit.SECONDS);

            if (ttl == null || ttl < 0) {
                return Result.failure("Token not found or no expiration");
            }

            return Result.success(ttl);
        } catch (Exception e) {
            log.error("Error getting TTL: tokenId={}", tokenId, e);
            return Result.failure("Error: " + e.getMessage());
        }
    }

    /**
     * 健康检查
     */
    public boolean isHealthy() {
        try {
            String pong = redisTemplate.getConnectionFactory()
                .getConnection()
                .ping();
            return "PONG".equals(pong);
        } catch (Exception e) {
            log.error("Health check failed", e);
            return false;
        }
    }

    // 数据类
    public static class OAuthToken {
        private String userId;
        private String scope;
        private long createdAt;

        // Getters and Setters
        public String getUserId() { return userId; }
        public void setUserId(String userId) { this.userId = userId; }
        public String getScope() { return scope; }
        public void setScope(String scope) { this.scope = scope; }
        public long getCreatedAt() { return createdAt; }
        public void setCreatedAt(long createdAt) { this.createdAt = createdAt; }
    }

    // 结果类
    public static class Result<T> {
        private final boolean success;
        private final T data;
        private final String error;

        private Result(boolean success, T data, String error) {
            this.success = success;
            this.data = data;
            this.error = error;
        }

        public static <T> Result<T> success(T data) {
            return new Result<>(true, data, null);
        }

        public static <T> Result<T> failure(String error) {
            return new Result<>(false, null, error);
        }

        public boolean isSuccess() { return success; }
        public T getData() { return data; }
        public String getError() { return error; }
    }
}
```

### 5. 健康检查

创建 `TokenginXHealthIndicator.java`:

```java
package com.myapp.health;

import com.myapp.service.TokenginXService;
import org.springframework.boot.actuate.health.Health;
import org.springframework.boot.actuate.health.HealthIndicator;
import org.springframework.stereotype.Component;

@Component("tokenginx")
public class TokenginXHealthIndicator implements HealthIndicator {

    private final TokenginXService tokenginxService;

    public TokenginXHealthIndicator(TokenginXService tokenginxService) {
        this.tokenginxService = tokenginxService;
    }

    @Override
    public Health health() {
        boolean isHealthy = tokenginxService.isHealthy();

        return isHealthy
            ? Health.up().withDetail("status", "Connected").build()
            : Health.down().withDetail("status", "Disconnected").build();
    }
}
```

### 6. 启用重试

创建 `RetryConfig.java`:

```java
package com.myapp.config;

import org.springframework.context.annotation.Configuration;
import org.springframework.retry.annotation.EnableRetry;

@Configuration
@EnableRetry
public class RetryConfig {
}
```

## 监控和日志

### 1. Logback 配置

创建 `logback-spring.xml`:

```xml
<?xml version="1.0" encoding="UTF-8"?>
<configuration>
    <include resource="org/springframework/boot/logging/logback/defaults.xml"/>

    <springProperty scope="context" name="APP_NAME" source="spring.application.name"/>

    <!-- Console Appender -->
    <appender name="CONSOLE" class="ch.qos.logback.core.ConsoleAppender">
        <encoder>
            <pattern>%d{yyyy-MM-dd HH:mm:ss.SSS} [%thread] %-5level %logger{36} - %msg%n</pattern>
        </encoder>
    </appender>

    <!-- File Appender -->
    <appender name="FILE" class="ch.qos.logback.core.rolling.RollingFileAppender">
        <file>/var/log/${APP_NAME}/application.log</file>
        <rollingPolicy class="ch.qos.logback.core.rolling.TimeBasedRollingPolicy">
            <fileNamePattern>/var/log/${APP_NAME}/application-%d{yyyy-MM-dd}.%i.log</fileNamePattern>
            <timeBasedFileNamingAndTriggeringPolicy
                    class="ch.qos.logback.core.rolling.SizeAndTimeBasedFNATP">
                <maxFileSize>100MB</maxFileSize>
            </timeBasedFileNamingAndTriggeringPolicy>
            <maxHistory>30</maxHistory>
        </rollingPolicy>
        <encoder>
            <pattern>%d{yyyy-MM-dd HH:mm:ss.SSS} [%thread] %-5level %logger{36} - %msg%n</pattern>
        </encoder>
    </appender>

    <!-- JSON Appender (for production) -->
    <appender name="JSON" class="ch.qos.logback.core.rolling.RollingFileAppender">
        <file>/var/log/${APP_NAME}/application.json</file>
        <rollingPolicy class="ch.qos.logback.core.rolling.TimeBasedRollingPolicy">
            <fileNamePattern>/var/log/${APP_NAME}/application-%d{yyyy-MM-dd}.%i.json</fileNamePattern>
            <timeBasedFileNamingAndTriggeringPolicy
                    class="ch.qos.logback.core.rolling.SizeAndTimeBasedFNATP">
                <maxFileSize>100MB</maxFileSize>
            </timeBasedFileNamingAndTriggeringPolicy>
            <maxHistory>30</maxHistory>
        </rollingPolicy>
        <encoder class="net.logstash.logback.encoder.LogstashEncoder">
            <includeMdcKeyName>traceId</includeMdcKeyName>
            <includeMdcKeyName>userId</includeMdcKeyName>
        </encoder>
    </appender>

    <root level="INFO">
        <appender-ref ref="CONSOLE"/>
        <appender-ref ref="FILE"/>
        <appender-ref ref="JSON"/>
    </root>

    <logger name="com.myapp" level="INFO"/>
    <logger name="io.lettuce.core" level="WARN"/>
</configuration>
```

### 2. Prometheus 指标导出

访问 `/actuator/prometheus` 获取指标。

示例指标:

```
# HELP tokenginx_token_created_total Total tokens created
# TYPE tokenginx_token_created_total counter
tokenginx_token_created_total{application="myapp-production",environment="production",} 12345.0

# HELP tokenginx_operation_duration_seconds Duration of TokenginX operations
# TYPE tokenginx_operation_duration_seconds summary
tokenginx_operation_duration_seconds{application="myapp-production",quantile="0.5",} 0.001234
tokenginx_operation_duration_seconds{application="myapp-production",quantile="0.95",} 0.005678
tokenginx_operation_duration_seconds{application="myapp-production",quantile="0.99",} 0.012345
```

## 性能优化

### 1. 连接池配置

```yaml
spring:
  data:
    redis:
      lettuce:
        pool:
          max-active: 50    # 最大活跃连接数
          max-idle: 20      # 最大空闲连接数
          min-idle: 10      # 最小空闲连接数
          max-wait: 2000ms  # 最大等待时间
```

### 2. JVM 参数优化

```bash
java -jar myapp.jar \
  -Xms2g -Xmx2g \
  -XX:+UseG1GC \
  -XX:MaxGCPauseMillis=200 \
  -XX:+HeapDumpOnOutOfMemoryError \
  -XX:HeapDumpPath=/var/log/myapp/heapdump.hprof
```

### 3. 使用虚拟线程(Java 21+)

```yaml
spring:
  threads:
    virtual:
      enabled: true
```

## Docker 部署

### Dockerfile

```dockerfile
FROM eclipse-temurin:21-jre-jammy AS base
WORKDIR /app

FROM maven:3.9-eclipse-temurin-21 AS build
WORKDIR /build
COPY pom.xml .
RUN mvn dependency:go-offline
COPY src ./src
RUN mvn package -DskipTests

FROM base AS final
COPY --from=build /build/target/*.jar app.jar

# 安全性:非 root 用户
RUN useradd -m -s /bin/bash appuser && chown -R appuser:appuser /app
USER appuser

EXPOSE 8080

ENTRYPOINT ["java", \
    "-XX:+UseContainerSupport", \
    "-XX:MaxRAMPercentage=75.0", \
    "-Djava.security.egd=file:/dev/./urandom", \
    "-jar", "app.jar"]
```

## Kubernetes 部署

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: myapp
spec:
  replicas: 3
  selector:
    matchLabels:
      app: myapp
  template:
    metadata:
      labels:
        app: myapp
    spec:
      containers:
      - name: myapp
        image: myapp:1.0.0
        ports:
        - containerPort: 8080
        env:
        - name: SPRING_PROFILES_ACTIVE
          value: "production"
        - name: TOKENGINX_API_KEY
          valueFrom:
            secretKeyRef:
              name: tokenginx-secret
              key: api-key
        - name: JAVA_OPTS
          value: "-Xms2g -Xmx2g -XX:+UseG1GC"
        livenessProbe:
          httpGet:
            path: /actuator/health/liveness
            port: 8080
          initialDelaySeconds: 60
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /actuator/health/readiness
            port: 8080
          initialDelaySeconds: 30
          periodSeconds: 5
        resources:
          requests:
            memory: "2Gi"
            cpu: "500m"
          limits:
            memory: "4Gi"
            cpu: "2000m"
```

## 最佳实践清单

- ✅ 使用连接池,配置合理的连接数
- ✅ 启用 TLS/mTLS 加密通信
- ✅ 使用环境变量存储敏感信息
- ✅ 实现健康检查(Liveness 和 Readiness)
- ✅ 添加 Prometheus 监控指标
- ✅ 使用结构化日志(JSON 格式)
- ✅ 实现重试机制(@Retryable)
- ✅ 设置合理的超时时间
- ✅ 使用 Pipeline 批量操作
- ✅ 配置 JVM 参数优化性能
- ✅ 使用虚拟线程(Java 21+)提升并发性能

## 下一步

- 查看 [OAuth 2.0 集成](../protocols/oauth.md)
- 配置 [TLS/mTLS](../security/tls-mtls.md)
- 了解 [防重放攻击](../security/anti-replay.md)
