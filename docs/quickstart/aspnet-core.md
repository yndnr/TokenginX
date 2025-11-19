# ASP.NET Core (C#) 快速指南

本指南帮助您快速在 ASP.NET Core 应用中集成 TokenginX。

## 前置要求

- .NET 6.0 或更高版本
- Visual Studio 2022 或 VS Code
- TokenginX 服务器已运行

## 安装客户端库

TokenginX 支持标准的 Redis 协议，可以使用 StackExchange.Redis 客户端。

```bash
dotnet add package StackExchange.Redis
```

或者使用 HTTP API 客户端：

```bash
dotnet add package RestSharp
```

## 使用 Redis 客户端（推荐）

### 1. 配置连接

在 `appsettings.json` 中配置连接字符串：

```json
{
  "TokenginX": {
    "ConnectionString": "localhost:6380,password=your-api-key,ssl=true,abortConnect=false"
  }
}
```

### 2. 注册服务

在 `Program.cs` 中注册 TokenginX 服务：

```csharp
using StackExchange.Redis;

var builder = WebApplication.CreateBuilder(args);

// 注册 TokenginX
builder.Services.AddSingleton<IConnectionMultiplexer>(sp =>
{
    var configuration = ConfigurationOptions.Parse(
        builder.Configuration["TokenginX:ConnectionString"]!);

    // TLS 配置
    configuration.Ssl = true;
    configuration.SslHost = "your-tokenginx-server";

    // 连接池配置
    configuration.ConnectRetry = 3;
    configuration.ConnectTimeout = 5000;
    configuration.SyncTimeout = 5000;

    return ConnectionMultiplexer.Connect(configuration);
});

builder.Services.AddScoped<IDatabase>(sp =>
{
    var redis = sp.GetRequiredService<IConnectionMultiplexer>();
    return redis.GetDatabase();
});

var app = builder.Build();
```

### 3. 创建服务类

创建 `TokenginXService.cs`：

```csharp
using StackExchange.Redis;
using System.Text.Json;

public class TokenginXService
{
    private readonly IDatabase _db;
    private readonly ILogger<TokenginXService> _logger;

    public TokenginXService(IDatabase database, ILogger<TokenginXService> logger)
    {
        _db = database;
        _logger = logger;
    }

    // 设置 OAuth Token
    public async Task<bool> SetOAuthTokenAsync(
        string tokenId,
        string userId,
        string scope,
        TimeSpan expiry)
    {
        try
        {
            var tokenData = new
            {
                user_id = userId,
                scope = scope,
                created_at = DateTimeOffset.UtcNow.ToUnixTimeSeconds()
            };

            var json = JsonSerializer.Serialize(tokenData);
            var key = $"oauth:token:{tokenId}";

            return await _db.StringSetAsync(key, json, expiry);
        }
        catch (Exception ex)
        {
            _logger.LogError(ex, "Failed to set OAuth token");
            return false;
        }
    }

    // 获取 OAuth Token
    public async Task<OAuthToken?> GetOAuthTokenAsync(string tokenId)
    {
        try
        {
            var key = $"oauth:token:{tokenId}";
            var value = await _db.StringGetAsync(key);

            if (value.IsNullOrEmpty)
                return null;

            return JsonSerializer.Deserialize<OAuthToken>(value!);
        }
        catch (Exception ex)
        {
            _logger.LogError(ex, "Failed to get OAuth token");
            return null;
        }
    }

    // 删除 Token
    public async Task<bool> DeleteTokenAsync(string tokenId)
    {
        var key = $"oauth:token:{tokenId}";
        return await _db.KeyDeleteAsync(key);
    }

    // 检查 Token 是否存在
    public async Task<bool> TokenExistsAsync(string tokenId)
    {
        var key = $"oauth:token:{tokenId}";
        return await _db.KeyExistsAsync(key);
    }

    // 获取剩余 TTL
    public async Task<TimeSpan?> GetTokenTTLAsync(string tokenId)
    {
        var key = $"oauth:token:{tokenId}";
        return await _db.KeyTimeToLiveAsync(key);
    }

    // 批量获取 Tokens
    public async Task<Dictionary<string, OAuthToken?>> GetMultipleTokensAsync(
        IEnumerable<string> tokenIds)
    {
        var keys = tokenIds.Select(id => (RedisKey)$"oauth:token:{id}").ToArray();
        var values = await _db.StringGetAsync(keys);

        var result = new Dictionary<string, OAuthToken?>();
        for (int i = 0; i < tokenIds.Count(); i++)
        {
            var tokenId = tokenIds.ElementAt(i);
            if (!values[i].IsNullOrEmpty)
            {
                result[tokenId] = JsonSerializer.Deserialize<OAuthToken>(values[i]!);
            }
            else
            {
                result[tokenId] = null;
            }
        }

        return result;
    }
}

public class OAuthToken
{
    public string user_id { get; set; } = string.Empty;
    public string scope { get; set; } = string.Empty;
    public long created_at { get; set; }
}
```

### 4. 注册服务

在 `Program.cs` 中注册自定义服务：

```csharp
builder.Services.AddScoped<TokenginXService>();
```

### 5. 在控制器中使用

```csharp
using Microsoft.AspNetCore.Mvc;

[ApiController]
[Route("api/[controller]")]
public class AuthController : ControllerBase
{
    private readonly TokenginXService _tokenginx;

    public AuthController(TokenginXService tokenginx)
    {
        _tokenginx = tokenginx;
    }

    [HttpPost("token")]
    public async Task<IActionResult> CreateToken([FromBody] TokenRequest request)
    {
        var tokenId = Guid.NewGuid().ToString();
        var success = await _tokenginx.SetOAuthTokenAsync(
            tokenId,
            request.UserId,
            request.Scope,
            TimeSpan.FromHours(1)
        );

        if (!success)
            return StatusCode(500, "Failed to create token");

        return Ok(new { access_token = tokenId });
    }

    [HttpPost("introspect")]
    public async Task<IActionResult> IntrospectToken([FromBody] IntrospectRequest request)
    {
        var token = await _tokenginx.GetOAuthTokenAsync(request.Token);

        if (token == null)
        {
            return Ok(new { active = false });
        }

        var ttl = await _tokenginx.GetTokenTTLAsync(request.Token);

        return Ok(new
        {
            active = true,
            scope = token.scope,
            user_id = token.user_id,
            exp = DateTimeOffset.UtcNow.Add(ttl ?? TimeSpan.Zero).ToUnixTimeSeconds()
        });
    }

    [HttpPost("revoke")]
    public async Task<IActionResult> RevokeToken([FromBody] RevokeRequest request)
    {
        await _tokenginx.DeleteTokenAsync(request.Token);
        return Ok();
    }
}

public record TokenRequest(string UserId, string Scope);
public record IntrospectRequest(string Token);
public record RevokeRequest(string Token);
```

## 使用 HTTP API 客户端

### 1. 安装依赖

```bash
dotnet add package RestSharp
```

### 2. 创建 HTTP 客户端

```csharp
using RestSharp;
using RestSharp.Authenticators;

public class TokenginXHttpClient
{
    private readonly RestClient _client;
    private readonly ILogger<TokenginXHttpClient> _logger;

    public TokenginXHttpClient(IConfiguration configuration, ILogger<TokenginXHttpClient> logger)
    {
        var baseUrl = configuration["TokenginX:BaseUrl"]!;
        var apiKey = configuration["TokenginX:ApiKey"]!;

        var options = new RestClientOptions(baseUrl)
        {
            Authenticator = new JwtAuthenticator(apiKey),
            ThrowOnAnyError = false,
            Timeout = TimeSpan.FromSeconds(10)
        };

        _client = new RestClient(options);
        _logger = logger;
    }

    public async Task<bool> SetSessionAsync(string key, object value, int ttl)
    {
        var request = new RestRequest("/api/v1/sessions", Method.Post);
        request.AddJsonBody(new
        {
            key,
            value,
            ttl
        });

        var response = await _client.ExecuteAsync(request);
        return response.IsSuccessful;
    }

    public async Task<T?> GetSessionAsync<T>(string key)
    {
        var encodedKey = Uri.EscapeDataString(key);
        var request = new RestRequest($"/api/v1/sessions/{encodedKey}", Method.Get);

        var response = await _client.ExecuteAsync<SessionResponse<T>>(request);

        if (!response.IsSuccessful || response.Data == null)
            return default;

        return response.Data.data.value;
    }

    public async Task<bool> DeleteSessionAsync(string key)
    {
        var encodedKey = Uri.EscapeDataString(key);
        var request = new RestRequest($"/api/v1/sessions/{encodedKey}", Method.Delete);

        var response = await _client.ExecuteAsync(request);
        return response.IsSuccessful;
    }
}

public class SessionResponse<T>
{
    public string status { get; set; } = string.Empty;
    public SessionData<T> data { get; set; } = new();
}

public class SessionData<T>
{
    public string key { get; set; } = string.Empty;
    public T value { get; set; } = default!;
    public int ttl { get; set; }
}
```

## 使用 mTLS 认证

```csharp
var options = new ConfigurationOptions
{
    EndPoints = { "your-tokenginx-server:6380" },
    Ssl = true,
    SslHost = "your-tokenginx-server"
};

// 加载客户端证书
var clientCert = new X509Certificate2("client-cert.pfx", "password");
options.CertificateSelection += delegate { return clientCert; };

// 验证服务器证书
options.CertificateValidation += (sender, cert, chain, errors) =>
{
    // 自定义证书验证逻辑
    return errors == SslPolicyErrors.None;
};

var connection = ConnectionMultiplexer.Connect(options);
```

## ASP.NET Core 中间件集成

创建一个中间件来验证每个请求的 Token：

```csharp
public class TokenValidationMiddleware
{
    private readonly RequestDelegate _next;
    private readonly TokenginXService _tokenginx;

    public TokenValidationMiddleware(RequestDelegate next, TokenginXService tokenginx)
    {
        _next = next;
        _tokenginx = tokenginx;
    }

    public async Task InvokeAsync(HttpContext context)
    {
        var authHeader = context.Request.Headers.Authorization.FirstOrDefault();

        if (authHeader?.StartsWith("Bearer ") == true)
        {
            var token = authHeader.Substring("Bearer ".Length).Trim();
            var tokenData = await _tokenginx.GetOAuthTokenAsync(token);

            if (tokenData != null)
            {
                // 将用户信息添加到 HttpContext
                context.Items["UserId"] = tokenData.user_id;
                context.Items["Scope"] = tokenData.scope;
            }
            else
            {
                context.Response.StatusCode = 401;
                await context.Response.WriteAsync("Invalid or expired token");
                return;
            }
        }

        await _next(context);
    }
}

// 在 Program.cs 中注册
app.UseMiddleware<TokenValidationMiddleware>();
```

## 错误处理

```csharp
public async Task<OAuthToken?> GetOAuthTokenWithRetryAsync(string tokenId, int maxRetries = 3)
{
    for (int i = 0; i < maxRetries; i++)
    {
        try
        {
            return await GetOAuthTokenAsync(tokenId);
        }
        catch (RedisTimeoutException ex)
        {
            _logger.LogWarning(ex, $"Timeout getting token, retry {i + 1}/{maxRetries}");
            if (i == maxRetries - 1) throw;
            await Task.Delay(TimeSpan.FromMilliseconds(100 * (i + 1)));
        }
        catch (RedisConnectionException ex)
        {
            _logger.LogError(ex, "Redis connection error");
            throw;
        }
    }

    return null;
}
```

## 依赖注入最佳实践

```csharp
// 添加健康检查
builder.Services.AddHealthChecks()
    .AddRedis(
        builder.Configuration["TokenginX:ConnectionString"]!,
        name: "tokenginx",
        tags: new[] { "ready" }
    );

// 添加分布式缓存（可选）
builder.Services.AddStackExchangeRedisCache(options =>
{
    options.Configuration = builder.Configuration["TokenginX:ConnectionString"];
    options.InstanceName = "MyApp_";
});
```

## 下一步

- 查看 [ASP.NET Core 生产环境指南](../production/aspnet-core.md)
- 了解 [OAuth 2.0 集成](../protocols/oauth.md)
- 配置 [TLS/mTLS](../security/tls-mtls.md)
