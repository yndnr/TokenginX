# ASP.NET Core (C#) 生产环境指南

本指南帮助您在生产环境中部署和优化 ASP.NET Core 应用与 TokenginX 的集成。

## 前置要求

- .NET 6.0 或更高版本(.NET 8.0 推荐)
- 生产环境 TokenginX 服务器集群
- 监控和日志基础设施

## 生产级配置

### 1. 连接配置

在 `appsettings.Production.json` 中配置:

```json
{
  "TokenginX": {
    "Endpoints": [
      "tokenginx-node1.prod.example.com:6380",
      "tokenginx-node2.prod.example.com:6380",
      "tokenginx-node3.prod.example.com:6380"
    ],
    "Password": "${TOKENGINX_API_KEY}",
    "Ssl": true,
    "SslHost": "tokenginx.prod.example.com",
    "ConnectTimeout": 5000,
    "SyncTimeout": 3000,
    "ConnectRetry": 3,
    "AbortOnConnectFail": false,
    "KeepAlive": 60,
    "PoolSize": 50,
    "ClientName": "MyApp-Production"
  },
  "Logging": {
    "LogLevel": {
      "Default": "Information",
      "TokenginX": "Warning",
      "StackExchange.Redis": "Warning"
    }
  }
}
```

### 2. 高可用连接配置

创建 `TokenginXOptions.cs`:

```csharp
public class TokenginXOptions
{
    public string[] Endpoints { get; set; } = Array.Empty<string>();
    public string Password { get; set; } = string.Empty;
    public bool Ssl { get; set; } = true;
    public string? SslHost { get; set; }
    public int ConnectTimeout { get; set; } = 5000;
    public int SyncTimeout { get; set; } = 3000;
    public int ConnectRetry { get; set; } = 3;
    public bool AbortOnConnectFail { get; set; } = false;
    public int KeepAlive { get; set; } = 60;
    public int PoolSize { get; set; } = 50;
    public string ClientName { get; set; } = "MyApp";
}
```

创建 `TokenginXServiceCollectionExtensions.cs`:

```csharp
using Microsoft.Extensions.Configuration;
using Microsoft.Extensions.DependencyInjection;
using Microsoft.Extensions.Logging;
using StackExchange.Redis;
using System.Net.Security;
using System.Security.Cryptography.X509Certificates;

public static class TokenginXServiceCollectionExtensions
{
    public static IServiceCollection AddTokenginX(
        this IServiceCollection services,
        IConfiguration configuration)
    {
        var options = configuration.GetSection("TokenginX").Get<TokenginXOptions>()
            ?? throw new InvalidOperationException("TokenginX configuration not found");

        // 替换环境变量
        options.Password = Environment.ExpandEnvironmentVariables(options.Password);

        // 注册为单例
        services.AddSingleton<IConnectionMultiplexer>(sp =>
        {
            var logger = sp.GetRequiredService<ILogger<IConnectionMultiplexer>>();

            var configOptions = new ConfigurationOptions
            {
                ConnectTimeout = options.ConnectTimeout,
                SyncTimeout = options.SyncTimeout,
                ConnectRetry = options.ConnectRetry,
                AbortOnConnectFail = options.AbortOnConnectFail,
                KeepAlive = options.KeepAlive,
                Password = options.Password,
                ClientName = options.ClientName,
                Ssl = options.Ssl,
                SslHost = options.SslHost,

                // 高可用配置
                ReconnectRetryPolicy = new ExponentialRetry(5000),
                DefaultDatabase = 0,
            };

            // 添加端点
            foreach (var endpoint in options.Endpoints)
            {
                configOptions.EndPoints.Add(endpoint);
            }

            // mTLS 配置
            if (options.Ssl)
            {
                configOptions.CertificateSelection += (sender, targetHost, localCerts,
                    remoteCert, acceptableIssuers) =>
                {
                    // 加载客户端证书
                    var certPath = Environment.GetEnvironmentVariable("TOKENGINX_CERT_PATH");
                    var certPassword = Environment.GetEnvironmentVariable("TOKENGINX_CERT_PASSWORD");

                    if (!string.IsNullOrEmpty(certPath) && File.Exists(certPath))
                    {
                        return new X509Certificate2(certPath, certPassword);
                    }

                    return null;
                };

                configOptions.CertificateValidation += (sender, cert, chain, errors) =>
                {
                    // 自定义证书验证逻辑
                    if (errors == SslPolicyErrors.None)
                        return true;

                    logger.LogWarning("Certificate validation errors: {Errors}", errors);

                    // 生产环境应该严格验证证书
                    return false;
                };
            }

            var connection = ConnectionMultiplexer.Connect(configOptions);

            // 连接事件处理
            connection.ConnectionFailed += (sender, e) =>
            {
                logger.LogError("Connection failed: {Endpoint}, {FailureType}, {Exception}",
                    e.EndPoint, e.FailureType, e.Exception);
            };

            connection.ConnectionRestored += (sender, e) =>
            {
                logger.LogInformation("Connection restored: {Endpoint}", e.EndPoint);
            };

            connection.ErrorMessage += (sender, e) =>
            {
                logger.LogError("Redis error: {Message}", e.Message);
            };

            return connection;
        });

        // 注册 IDatabase
        services.AddScoped<IDatabase>(sp =>
        {
            var connection = sp.GetRequiredService<IConnectionMultiplexer>();
            return connection.GetDatabase();
        });

        // 注册业务服务
        services.AddScoped<TokenginXService>();

        return services;
    }
}
```

### 3. 生产级服务实现

创建 `TokenginXService.cs`:

```csharp
using Microsoft.Extensions.Logging;
using StackExchange.Redis;
using System.Text.Json;

public class TokenginXService
{
    private readonly IDatabase _db;
    private readonly ILogger<TokenginXService> _logger;
    private readonly IConnectionMultiplexer _connection;

    public TokenginXService(
        IDatabase database,
        ILogger<TokenginXService> logger,
        IConnectionMultiplexer connection)
    {
        _db = database;
        _logger = logger;
        _connection = connection;
    }

    /// <summary>
    /// 设置 OAuth Token(带重试和容错)
    /// </summary>
    public async Task<Result<bool>> SetOAuthTokenAsync(
        string tokenId,
        string userId,
        string scope,
        TimeSpan expiry,
        CancellationToken cancellationToken = default)
    {
        try
        {
            var tokenData = new
            {
                user_id = userId,
                scope = scope,
                created_at = DateTimeOffset.UtcNow.ToUnixTimeSeconds(),
                client_ip = GetClientIp()
            };

            var json = JsonSerializer.Serialize(tokenData);
            var key = $"oauth:token:{tokenId}";

            var success = await RetryPolicy.ExecuteAsync(async () =>
            {
                return await _db.StringSetAsync(key, json, expiry);
            }, cancellationToken);

            if (success)
            {
                _logger.LogInformation(
                    "OAuth token created: TokenId={TokenId}, UserId={UserId}, TTL={TTL}s",
                    tokenId, userId, expiry.TotalSeconds);
            }
            else
            {
                _logger.LogWarning(
                    "Failed to create OAuth token: TokenId={TokenId}, UserId={UserId}",
                    tokenId, userId);
            }

            return Result<bool>.Success(success);
        }
        catch (Exception ex)
        {
            _logger.LogError(ex,
                "Error setting OAuth token: TokenId={TokenId}, UserId={UserId}",
                tokenId, userId);
            return Result<bool>.Failure($"Failed to set token: {ex.Message}");
        }
    }

    /// <summary>
    /// 获取 OAuth Token(带缓存和降级)
    /// </summary>
    public async Task<Result<OAuthToken?>> GetOAuthTokenAsync(
        string tokenId,
        CancellationToken cancellationToken = default)
    {
        try
        {
            var key = $"oauth:token:{tokenId}";

            var value = await RetryPolicy.ExecuteAsync(async () =>
            {
                return await _db.StringGetAsync(key);
            }, cancellationToken);

            if (value.IsNullOrEmpty)
            {
                _logger.LogDebug("OAuth token not found: TokenId={TokenId}", tokenId);
                return Result<OAuthToken?>.Success(null);
            }

            var token = JsonSerializer.Deserialize<OAuthToken>(value!);

            _logger.LogDebug("OAuth token retrieved: TokenId={TokenId}", tokenId);

            return Result<OAuthToken?>.Success(token);
        }
        catch (Exception ex)
        {
            _logger.LogError(ex, "Error getting OAuth token: TokenId={TokenId}", tokenId);
            return Result<OAuthToken?>.Failure($"Failed to get token: {ex.Message}");
        }
    }

    /// <summary>
    /// 批量获取 Token(性能优化)
    /// </summary>
    public async Task<Result<Dictionary<string, OAuthToken?>>> GetMultipleTokensAsync(
        IEnumerable<string> tokenIds,
        CancellationToken cancellationToken = default)
    {
        try
        {
            var tokenIdList = tokenIds.ToList();
            var keys = tokenIdList.Select(id => (RedisKey)$"oauth:token:{id}").ToArray();

            var values = await RetryPolicy.ExecuteAsync(async () =>
            {
                return await _db.StringGetAsync(keys);
            }, cancellationToken);

            var result = new Dictionary<string, OAuthToken?>();
            for (int i = 0; i < tokenIdList.Count; i++)
            {
                if (!values[i].IsNullOrEmpty)
                {
                    result[tokenIdList[i]] = JsonSerializer.Deserialize<OAuthToken>(values[i]!);
                }
                else
                {
                    result[tokenIdList[i]] = null;
                }
            }

            _logger.LogDebug(
                "Batch retrieved {Total} tokens, {Found} found",
                tokenIdList.Count, result.Count(kv => kv.Value != null));

            return Result<Dictionary<string, OAuthToken?>>.Success(result);
        }
        catch (Exception ex)
        {
            _logger.LogError(ex, "Error getting multiple tokens");
            return Result<Dictionary<string, OAuthToken?>>.Failure($"Batch get failed: {ex.Message}");
        }
    }

    /// <summary>
    /// 健康检查
    /// </summary>
    public async Task<bool> IsHealthyAsync(CancellationToken cancellationToken = default)
    {
        try
        {
            var timeout = TimeSpan.FromSeconds(2);
            using var cts = CancellationTokenSource.CreateLinkedTokenSource(cancellationToken);
            cts.CancelAfter(timeout);

            await _db.PingAsync();
            return _connection.IsConnected;
        }
        catch
        {
            return false;
        }
    }

    private string? GetClientIp()
    {
        // 从 HTTP 上下文获取客户端 IP(如果可用)
        return null;
    }
}

// 重试策略
public static class RetryPolicy
{
    public static async Task<T> ExecuteAsync<T>(
        Func<Task<T>> operation,
        CancellationToken cancellationToken = default,
        int maxRetries = 3)
    {
        for (int i = 0; i < maxRetries; i++)
        {
            try
            {
                return await operation();
            }
            catch (RedisTimeoutException) when (i < maxRetries - 1)
            {
                await Task.Delay(TimeSpan.FromMilliseconds(100 * (i + 1)), cancellationToken);
            }
        }

        return await operation();
    }
}

// 结果类型
public class Result<T>
{
    public bool IsSuccess { get; }
    public T Value { get; }
    public string? Error { get; }

    private Result(bool isSuccess, T value, string? error)
    {
        IsSuccess = isSuccess;
        Value = value;
        Error = error;
    }

    public static Result<T> Success(T value) => new(true, value, null);
    public static Result<T> Failure(string error) => new(false, default!, error);
}

public class OAuthToken
{
    public string user_id { get; set; } = string.Empty;
    public string scope { get; set; } = string.Empty;
    public long created_at { get; set; }
    public string? client_ip { get; set; }
}
```

### 4. 注册服务

在 `Program.cs` 中:

```csharp
var builder = WebApplication.CreateBuilder(args);

// 添加 TokenginX
builder.Services.AddTokenginX(builder.Configuration);

// 添加健康检查
builder.Services.AddHealthChecks()
    .AddCheck<TokenginXHealthCheck>("tokenginx", tags: new[] { "ready", "live" });

// 添加分布式缓存(可选)
builder.Services.AddStackExchangeRedisCache(options =>
{
    var connection = builder.Services.BuildServiceProvider()
        .GetRequiredService<IConnectionMultiplexer>();
    options.ConnectionMultiplexerFactory = () => Task.FromResult(connection);
    options.InstanceName = "MyApp_";
});

var app = builder.Build();

// 健康检查端点
app.MapHealthChecks("/health/live", new HealthCheckOptions
{
    Predicate = check => check.Tags.Contains("live")
});

app.MapHealthChecks("/health/ready", new HealthCheckOptions
{
    Predicate = check => check.Tags.Contains("ready")
});

app.Run();
```

### 5. 健康检查实现

创建 `TokenginXHealthCheck.cs`:

```csharp
using Microsoft.Extensions.Diagnostics.HealthChecks;

public class TokenginXHealthCheck : IHealthCheck
{
    private readonly TokenginXService _tokenginx;

    public TokenginXHealthCheck(TokenginXService tokenginx)
    {
        _tokenginx = tokenginx;
    }

    public async Task<HealthCheckResult> CheckHealthAsync(
        HealthCheckContext context,
        CancellationToken cancellationToken = default)
    {
        try
        {
            var isHealthy = await _tokenginx.IsHealthyAsync(cancellationToken);

            return isHealthy
                ? HealthCheckResult.Healthy("TokenginX is healthy")
                : HealthCheckResult.Unhealthy("TokenginX is not responding");
        }
        catch (Exception ex)
        {
            return HealthCheckResult.Unhealthy("TokenginX check failed", ex);
        }
    }
}
```

## 监控和日志

### 1. Prometheus 指标

安装包:

```bash
dotnet add package prometheus-net.AspNetCore
```

配置:

```csharp
using Prometheus;

var app = builder.Build();

// 启用 HTTP 指标
app.UseHttpMetrics();

// Prometheus 端点
app.MapMetrics();
```

创建自定义指标:

```csharp
using Prometheus;

public class TokenginXMetrics
{
    private static readonly Counter TokenCreated = Metrics
        .CreateCounter("tokenginx_token_created_total", "Total tokens created");

    private static readonly Counter TokenRetrieved = Metrics
        .CreateCounter("tokenginx_token_retrieved_total", "Total tokens retrieved");

    private static readonly Histogram TokenOperationDuration = Metrics
        .CreateHistogram("tokenginx_operation_duration_seconds",
            "Duration of TokenginX operations",
            new HistogramConfiguration
            {
                Buckets = Histogram.ExponentialBuckets(0.001, 2, 10)
            });

    public static void RecordTokenCreated() => TokenCreated.Inc();

    public static void RecordTokenRetrieved() => TokenRetrieved.Inc();

    public static IDisposable MeasureOperation() =>
        TokenOperationDuration.NewTimer();
}
```

在服务中使用:

```csharp
public async Task<Result<bool>> SetOAuthTokenAsync(...)
{
    using (TokenginXMetrics.MeasureOperation())
    {
        // 执行操作
        var result = await ...;

        if (result)
        {
            TokenginXMetrics.RecordTokenCreated();
        }

        return result;
    }
}
```

### 2. 结构化日志

使用 Serilog:

```bash
dotnet add package Serilog.AspNetCore
dotnet add package Serilog.Sinks.Console
dotnet add package Serilog.Sinks.File
dotnet add package Serilog.Enrichers.Environment
```

配置:

```csharp
using Serilog;

Log.Logger = new LoggerConfiguration()
    .MinimumLevel.Information()
    .MinimumLevel.Override("Microsoft", LogEventLevel.Warning)
    .MinimumLevel.Override("StackExchange.Redis", LogEventLevel.Warning)
    .Enrich.FromLogContext()
    .Enrich.WithMachineName()
    .Enrich.WithEnvironmentName()
    .WriteTo.Console(
        outputTemplate: "[{Timestamp:HH:mm:ss} {Level:u3}] {Message:lj} {Properties:j}{NewLine}{Exception}")
    .WriteTo.File(
        path: "/var/log/myapp/log-.txt",
        rollingInterval: RollingInterval.Day,
        retainedFileCountLimit: 30)
    .CreateLogger();

builder.Host.UseSerilog();
```

## 性能优化

### 1. 连接池优化

```json
{
  "TokenginX": {
    "PoolSize": 50,
    "ConnectRetry": 3,
    "SyncTimeout": 3000,
    "KeepAlive": 60
  }
}
```

### 2. 使用管道

```csharp
public async Task<Dictionary<string, string>> GetMultipleValuesAsync(string[] keys)
{
    var batch = _db.CreateBatch();
    var tasks = keys.Select(key => batch.StringGetAsync(key)).ToArray();

    batch.Execute();

    var results = await Task.WhenAll(tasks);

    return keys.Zip(results, (key, value) => new { key, value })
        .Where(x => !x.value.IsNullOrEmpty)
        .ToDictionary(x => x.key, x => x.value.ToString());
}
```

### 3. 使用异步 I/O

确保所有 TokenginX 操作都是异步的:

```csharp
// ✅ 推荐
await _db.StringSetAsync(key, value);

// ❌ 不推荐(会阻塞线程)
_db.StringSet(key, value);
```

## 安全配置

### 1. 环境变量管理

使用 Azure Key Vault 或 AWS Secrets Manager:

```csharp
builder.Configuration.AddAzureKeyVault(
    new Uri($"https://{keyVaultName}.vault.azure.net/"),
    new DefaultAzureCredential());
```

### 2. mTLS 证书管理

```csharp
// 从证书存储加载
var store = new X509Store(StoreName.My, StoreLocation.CurrentUser);
store.Open(OpenFlags.ReadOnly);
var cert = store.Certificates
    .Find(X509FindType.FindByThumbprint, thumbprint, false)[0];
store.Close();
```

## Docker 部署

### Dockerfile

```dockerfile
FROM mcr.microsoft.com/dotnet/aspnet:8.0 AS base
WORKDIR /app
EXPOSE 80
EXPOSE 443

FROM mcr.microsoft.com/dotnet/sdk:8.0 AS build
WORKDIR /src
COPY ["MyApp/MyApp.csproj", "MyApp/"]
RUN dotnet restore "MyApp/MyApp.csproj"
COPY . .
WORKDIR "/src/MyApp"
RUN dotnet build "MyApp.csproj" -c Release -o /app/build

FROM build AS publish
RUN dotnet publish "MyApp.csproj" -c Release -o /app/publish

FROM base AS final
WORKDIR /app
COPY --from=publish /app/publish .

# 安全性:非 root 用户运行
RUN useradd -m -s /bin/bash appuser
USER appuser

ENTRYPOINT ["dotnet", "MyApp.dll"]
```

### docker-compose.yml

```yaml
version: '3.8'

services:
  myapp:
    image: myapp:latest
    build:
      context: .
      dockerfile: Dockerfile
    ports:
      - "8080:80"
    environment:
      - ASPNETCORE_ENVIRONMENT=Production
      - TOKENGINX_API_KEY=${TOKENGINX_API_KEY}
      - TOKENGINX_CERT_PATH=/certs/client-cert.pfx
      - TOKENGINX_CERT_PASSWORD=${CERT_PASSWORD}
    volumes:
      - ./certs:/certs:ro
      - ./logs:/var/log/myapp
    depends_on:
      - tokenginx
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost/health/live"]
      interval: 30s
      timeout: 10s
      retries: 3

  tokenginx:
    image: tokenginx/tokenginx-server:latest
    ports:
      - "6380:6380"
      - "9090:9090"
    environment:
      - TOKENGINX_CONFIG=/config/tokenginx.yaml
    volumes:
      - ./config:/config:ro
      - ./data:/data
    restart: unless-stopped
```

## Kubernetes 部署

### deployment.yaml

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: myapp
  labels:
    app: myapp
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
        - containerPort: 80
        env:
        - name: ASPNETCORE_ENVIRONMENT
          value: "Production"
        - name: TOKENGINX_API_KEY
          valueFrom:
            secretKeyRef:
              name: tokenginx-secret
              key: api-key
        - name: TOKENGINX_CERT_PATH
          value: "/certs/client-cert.pfx"
        - name: TOKENGINX_CERT_PASSWORD
          valueFrom:
            secretKeyRef:
              name: tokenginx-cert-secret
              key: password
        volumeMounts:
        - name: certs
          mountPath: /certs
          readOnly: true
        livenessProbe:
          httpGet:
            path: /health/live
            port: 80
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /health/ready
            port: 80
          initialDelaySeconds: 10
          periodSeconds: 5
        resources:
          requests:
            memory: "256Mi"
            cpu: "100m"
          limits:
            memory: "512Mi"
            cpu: "500m"
      volumes:
      - name: certs
        secret:
          secretName: tokenginx-cert-secret
---
apiVersion: v1
kind: Service
metadata:
  name: myapp
spec:
  selector:
    app: myapp
  ports:
  - protocol: TCP
    port: 80
    targetPort: 80
  type: LoadBalancer
```

## 最佳实践清单

- ✅ 使用连接池,避免每次请求创建新连接
- ✅ 启用 TLS/mTLS 加密通信
- ✅ 使用环境变量或密钥管理服务存储敏感信息
- ✅ 实现健康检查端点
- ✅ 添加 Prometheus 监控指标
- ✅ 使用结构化日志(Serilog)
- ✅ 实现重试和容错机制
- ✅ 设置合理的超时时间
- ✅ 使用异步 I/O
- ✅ 实现优雅关闭(Graceful Shutdown)
- ✅ 限制连接池大小,避免资源耗尽
- ✅ 监控连接失败和恢复事件

## 下一步

- 查看 [OAuth 2.0 集成](../protocols/oauth.md)
- 配置 [TLS/mTLS](../security/tls-mtls.md)
- 了解 [防重放攻击](../security/anti-replay.md)
