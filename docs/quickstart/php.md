# PHP 快速指南

本指南帮助您快速在 PHP 应用中集成 TokenginX。

## 前置要求

- PHP 7.4 或更高版本
- Composer
- TokenginX 服务器已运行

## 安装客户端库

TokenginX 支持标准的 Redis 协议,可以使用 Predis 或 PhpRedis 客户端。

### 使用 Predis (推荐)

```bash
composer require predis/predis
```

### 使用 PhpRedis (C 扩展,性能更高)

```bash
pecl install redis
# 在 php.ini 中启用
extension=redis.so
```

## 使用 Predis 客户端

### 1. 创建连接

创建 `TokenginXClient.php`:

```php
<?php

use Predis\Client;

class TokenginXClient
{
    private Client $client;

    public function __construct(array $config = [])
    {
        $defaultConfig = [
            'scheme' => 'tls',
            'host'   => 'localhost',
            'port'   => 6380,
            'password' => getenv('TOKENGINX_API_KEY'),
            'database' => 0,
            'read_write_timeout' => 5,
            'ssl' => [
                'verify_peer' => true,
                'verify_peer_name' => true,
                'cafile' => '/path/to/ca.pem',
            ],
        ];

        $this->client = new Client(array_merge($defaultConfig, $config));
    }

    /**
     * 设置 OAuth Token
     *
     * @param string $tokenId Token 唯一标识
     * @param array $tokenData Token 数据
     * @param int $ttl 过期时间(秒)
     * @return bool 是否成功
     */
    public function setOAuthToken(string $tokenId, array $tokenData, int $ttl): bool
    {
        try {
            $key = "oauth:token:{$tokenId}";
            $value = json_encode([
                'user_id' => $tokenData['user_id'],
                'scope' => $tokenData['scope'],
                'created_at' => time(),
            ]);

            $result = $this->client->setex($key, $ttl, $value);
            return $result === 'OK';
        } catch (\Exception $e) {
            error_log("Failed to set OAuth token: " . $e->getMessage());
            return false;
        }
    }

    /**
     * 获取 OAuth Token
     *
     * @param string $tokenId Token 唯一标识
     * @return array|null Token 数据,不存在时返回 null
     */
    public function getOAuthToken(string $tokenId): ?array
    {
        try {
            $key = "oauth:token:{$tokenId}";
            $value = $this->client->get($key);

            if ($value === null) {
                return null;
            }

            return json_decode($value, true);
        } catch (\Exception $e) {
            error_log("Failed to get OAuth token: " . $e->getMessage());
            return null;
        }
    }

    /**
     * 删除 Token
     *
     * @param string $tokenId Token 唯一标识
     * @return bool 是否成功
     */
    public function deleteToken(string $tokenId): bool
    {
        $key = "oauth:token:{$tokenId}";
        return $this->client->del([$key]) > 0;
    }

    /**
     * 检查 Token 是否存在
     *
     * @param string $tokenId Token 唯一标识
     * @return bool 是否存在
     */
    public function tokenExists(string $tokenId): bool
    {
        $key = "oauth:token:{$tokenId}";
        return $this->client->exists($key) === 1;
    }

    /**
     * 获取剩余 TTL
     *
     * @param string $tokenId Token 唯一标识
     * @return int|null 剩余秒数,不存在时返回 null
     */
    public function getTokenTTL(string $tokenId): ?int
    {
        $key = "oauth:token:{$tokenId}";
        $ttl = $this->client->ttl($key);

        return $ttl > 0 ? $ttl : null;
    }

    /**
     * 批量获取 Tokens
     *
     * @param array $tokenIds Token ID 列表
     * @return array 键为 tokenId,值为 token 数据的关联数组
     */
    public function getMultipleTokens(array $tokenIds): array
    {
        $keys = array_map(fn($id) => "oauth:token:{$id}", $tokenIds);
        $values = $this->client->mget($keys);

        $result = [];
        foreach ($tokenIds as $index => $tokenId) {
            if ($values[$index] !== null) {
                $result[$tokenId] = json_decode($values[$index], true);
            } else {
                $result[$tokenId] = null;
            }
        }

        return $result;
    }

    /**
     * 搜索匹配的键
     *
     * @param string $pattern 匹配模式
     * @param int $count 每次迭代返回的数量提示
     * @return array 匹配的键列表
     */
    public function scanKeys(string $pattern, int $count = 100): array
    {
        $keys = [];
        $cursor = 0;

        do {
            [$cursor, $batch] = $this->client->scan($cursor, [
                'MATCH' => $pattern,
                'COUNT' => $count,
            ]);

            $keys = array_merge($keys, $batch);
        } while ($cursor !== 0);

        return $keys;
    }
}
```

### 2. 在应用中使用

```php
<?php

require_once __DIR__ . '/vendor/autoload.php';
require_once __DIR__ . '/TokenginXClient.php';

// 创建客户端实例
$tokenginx = new TokenginXClient();

// 创建 Token
$tokenId = bin2hex(random_bytes(16));
$success = $tokenginx->setOAuthToken($tokenId, [
    'user_id' => 'user001',
    'scope' => 'read write',
], 3600);

if ($success) {
    echo "Token created: {$tokenId}\n";
} else {
    echo "Failed to create token\n";
}

// 验证 Token
$token = $tokenginx->getOAuthToken($tokenId);
if ($token) {
    echo "Token valid:\n";
    echo "  User ID: {$token['user_id']}\n";
    echo "  Scope: {$token['scope']}\n";

    $ttl = $tokenginx->getTokenTTL($tokenId);
    echo "  TTL: {$ttl} seconds\n";
} else {
    echo "Token not found or expired\n";
}

// 撤销 Token
$tokenginx->deleteToken($tokenId);
echo "Token revoked\n";
```

## 使用 PhpRedis 扩展

```php
<?php

class TokenginXPhpRedis
{
    private Redis $redis;

    public function __construct()
    {
        $this->redis = new Redis();

        // TLS 连接
        $this->redis->connect(
            'tls://localhost',
            6380,
            5.0 // 超时时间
        );

        // 认证
        $this->redis->auth(getenv('TOKENGINX_API_KEY'));

        // 设置选项
        $this->redis->setOption(Redis::OPT_SERIALIZER, Redis::SERIALIZER_JSON);
        $this->redis->setOption(Redis::OPT_READ_TIMEOUT, 5);
    }

    public function setOAuthToken(string $tokenId, array $tokenData, int $ttl): bool
    {
        $key = "oauth:token:{$tokenId}";
        return $this->redis->setex($key, $ttl, $tokenData);
    }

    public function getOAuthToken(string $tokenId): ?array
    {
        $key = "oauth:token:{$tokenId}";
        $value = $this->redis->get($key);

        return $value !== false ? $value : null;
    }

    public function deleteToken(string $tokenId): bool
    {
        $key = "oauth:token:{$tokenId}";
        return $this->redis->del($key) > 0;
    }
}
```

## Laravel 集成

### 1. 配置连接

在 `config/database.php` 中添加:

```php
'redis' => [
    'client' => env('REDIS_CLIENT', 'predis'),

    'tokenginx' => [
        'url' => env('TOKENGINX_URL'),
        'host' => env('TOKENGINX_HOST', 'localhost'),
        'password' => env('TOKENGINX_PASSWORD'),
        'port' => env('TOKENGINX_PORT', 6380),
        'database' => env('TOKENGINX_DB', 0),
        'scheme' => 'tls',
    ],
],
```

在 `.env` 中配置:

```env
TOKENGINX_HOST=localhost
TOKENGINX_PORT=6380
TOKENGINX_PASSWORD=your-api-key
```

### 2. 创建服务类

```php
<?php

namespace App\Services;

use Illuminate\Support\Facades\Redis;
use Illuminate\Support\Facades\Log;

class TokenginXService
{
    private $redis;

    public function __construct()
    {
        $this->redis = Redis::connection('tokenginx');
    }

    public function setOAuthToken(string $tokenId, array $tokenData, int $ttl): bool
    {
        try {
            $key = "oauth:token:{$tokenId}";
            $value = json_encode([
                'user_id' => $tokenData['user_id'],
                'scope' => $tokenData['scope'],
                'created_at' => time(),
            ]);

            return $this->redis->setex($key, $ttl, $value) === 'OK';
        } catch (\Exception $e) {
            Log::error('Failed to set OAuth token', ['error' => $e->getMessage()]);
            return false;
        }
    }

    public function getOAuthToken(string $tokenId): ?array
    {
        try {
            $key = "oauth:token:{$tokenId}";
            $value = $this->redis->get($key);

            return $value ? json_decode($value, true) : null;
        } catch (\Exception $e) {
            Log::error('Failed to get OAuth token', ['error' => $e->getMessage()]);
            return null;
        }
    }

    public function deleteToken(string $tokenId): bool
    {
        $key = "oauth:token:{$tokenId}";
        return $this->redis->del($key) > 0;
    }
}
```

### 3. 在控制器中使用

```php
<?php

namespace App\Http\Controllers;

use App\Services\TokenginXService;
use Illuminate\Http\Request;
use Illuminate\Support\Str;

class AuthController extends Controller
{
    private TokenginXService $tokenginx;

    public function __construct(TokenginXService $tokenginx)
    {
        $this->tokenginx = $tokenginx;
    }

    public function createToken(Request $request)
    {
        $request->validate([
            'user_id' => 'required|string',
            'scope' => 'required|string',
        ]);

        $tokenId = Str::random(32);
        $success = $this->tokenginx->setOAuthToken($tokenId, [
            'user_id' => $request->user_id,
            'scope' => $request->scope,
        ], 3600);

        if (!$success) {
            return response()->json(['error' => 'Failed to create token'], 500);
        }

        return response()->json(['access_token' => $tokenId]);
    }

    public function introspectToken(Request $request)
    {
        $request->validate([
            'token' => 'required|string',
        ]);

        $token = $this->tokenginx->getOAuthToken($request->token);

        if (!$token) {
            return response()->json(['active' => false]);
        }

        return response()->json([
            'active' => true,
            'scope' => $token['scope'],
            'user_id' => $token['user_id'],
        ]);
    }

    public function revokeToken(Request $request)
    {
        $request->validate([
            'token' => 'required|string',
        ]);

        $this->tokenginx->deleteToken($request->token);

        return response()->json(['message' => 'Token revoked']);
    }
}
```

### 4. 中间件集成

创建认证中间件 `app/Http/Middleware/TokenginXAuth.php`:

```php
<?php

namespace App\Http\Middleware;

use App\Services\TokenginXService;
use Closure;
use Illuminate\Http\Request;

class TokenginXAuth
{
    private TokenginXService $tokenginx;

    public function __construct(TokenginXService $tokenginx)
    {
        $this->tokenginx = $tokenginx;
    }

    public function handle(Request $request, Closure $next)
    {
        $authHeader = $request->header('Authorization');

        if (!$authHeader || !str_starts_with($authHeader, 'Bearer ')) {
            return response()->json(['error' => 'Unauthorized'], 401);
        }

        $token = substr($authHeader, 7);
        $tokenData = $this->tokenginx->getOAuthToken($token);

        if (!$tokenData) {
            return response()->json(['error' => 'Invalid or expired token'], 401);
        }

        // 将用户信息添加到请求
        $request->merge([
            'user_id' => $tokenData['user_id'],
            'scope' => $tokenData['scope'],
        ]);

        return $next($request);
    }
}
```

## 使用 mTLS 认证

### 使用 Predis

```php
<?php

$client = new Predis\Client([
    'scheme' => 'tls',
    'host'   => 'localhost',
    'port'   => 6380,
    'ssl' => [
        'verify_peer' => true,
        'verify_peer_name' => true,
        'cafile' => '/path/to/ca.pem',
        'local_cert' => '/path/to/client-cert.pem',
        'local_pk' => '/path/to/client-key.pem',
    ],
]);
```

### 使用 PhpRedis

```php
<?php

$redis = new Redis();

$context = stream_context_create([
    'ssl' => [
        'verify_peer' => true,
        'verify_peer_name' => true,
        'cafile' => '/path/to/ca.pem',
        'local_cert' => '/path/to/client-cert.pem',
        'local_pk' => '/path/to/client-key.pem',
    ],
]);

$redis->connect(
    'tls://localhost',
    6380,
    5.0,
    null,
    0,
    0,
    $context
);
```

## 错误处理

```php
<?php

class TokenginXClient
{
    private const MAX_RETRIES = 3;
    private const RETRY_DELAY_MS = 100;

    public function getOAuthTokenWithRetry(string $tokenId): ?array
    {
        $lastException = null;

        for ($i = 0; $i < self::MAX_RETRIES; $i++) {
            try {
                return $this->getOAuthToken($tokenId);
            } catch (Predis\Connection\ConnectionException $e) {
                $lastException = $e;
                error_log("Connection error, retry {$i}/{self::MAX_RETRIES}");

                if ($i < self::MAX_RETRIES - 1) {
                    usleep(self::RETRY_DELAY_MS * 1000 * ($i + 1));
                }
            }
        }

        throw $lastException;
    }
}
```

## 连接池(使用 Swoole)

```php
<?php

use Swoole\Coroutine\Channel;

class TokenginXPool
{
    private Channel $pool;
    private int $size;

    public function __construct(int $size = 10)
    {
        $this->size = $size;
        $this->pool = new Channel($size);

        for ($i = 0; $i < $size; $i++) {
            $this->pool->push($this->createClient());
        }
    }

    private function createClient(): Predis\Client
    {
        return new Predis\Client([
            'scheme' => 'tls',
            'host'   => 'localhost',
            'port'   => 6380,
            'password' => getenv('TOKENGINX_API_KEY'),
        ]);
    }

    public function get(): Predis\Client
    {
        return $this->pool->pop();
    }

    public function put(Predis\Client $client): void
    {
        $this->pool->push($client);
    }

    public function execute(callable $callback)
    {
        $client = $this->get();

        try {
            return $callback($client);
        } finally {
            $this->put($client);
        }
    }
}

// 使用示例
$pool = new TokenginXPool(20);

$token = $pool->execute(function($client) use ($tokenId) {
    return $client->get("oauth:token:{$tokenId}");
});
```

## 下一步

- 查看 [PHP 生产环境指南](../production/php.md)
- 了解 [OAuth 2.0 集成](../protocols/oauth.md)
- 配置 [TLS/mTLS](../security/tls-mtls.md)
