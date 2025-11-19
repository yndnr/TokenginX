# PHP 生产环境指南

本指南帮助您在生产环境中部署和优化 PHP 应用与 TokenginX 的集成。

## 前置要求

- PHP 8.1 或更高版本(PHP 8.3 推荐)
- Composer 2.x
- 生产环境 TokenginX 服务器集群
- 监控和日志基础设施

## 生产级配置

### 1. Composer 依赖

在 `composer.json` 中:

```json
{
    "require": {
        "php": "^8.1",
        "predis/predis": "^2.2",
        "monolog/monolog": "^3.5",
        "prometheus/client_php": "^2.7",
        "symfony/cache": "^6.4"
    },
    "require-dev": {
        "phpunit/phpunit": "^10.5"
    },
    "config": {
        "optimize-autoloader": true,
        "apcu-autoloader": true,
        "sort-packages": true
    }
}
```

安装依赖:

```bash
composer install --no-dev --optimize-autoloader
```

### 2. 环境配置

创建 `.env.production`:

```env
# TokenginX 配置
TOKENGINX_SCHEME=tls
TOKENGINX_NODES=tokenginx-node1.prod.example.com:6380,tokenginx-node2.prod.example.com:6380,tokenginx-node3.prod.example.com:6380
TOKENGINX_PASSWORD=your-api-key-from-secret-manager
TOKENGINX_DATABASE=0
TOKENGINX_PREFIX=myapp:
TOKENGINX_READ_TIMEOUT=3
TOKENGINX_TIMEOUT=5

# SSL/TLS 配置
TOKENGINX_SSL_VERIFY_PEER=true
TOKENGINX_SSL_CAFILE=/etc/ssl/certs/tokenginx-ca.pem
TOKENGINX_SSL_CERTFILE=/etc/ssl/certs/client-cert.pem
TOKENGINX_SSL_KEYFILE=/etc/ssl/private/client-key.pem

# 连接池配置
TOKENGINX_POOL_SIZE=50
TOKENGINX_POOL_MIN_IDLE=10

# 应用配置
APP_ENV=production
APP_DEBUG=false
LOG_LEVEL=warning
```

### 3. 高可用连接类

创建 `src/TokenginX/ConnectionManager.php`:

```php
<?php

namespace App\TokenginX;

use Predis\Client;
use Predis\Connection\Parameters;
use Psr\Log\LoggerInterface;

class ConnectionManager
{
    private static ?Client $client = null;
    private LoggerInterface $logger;
    private array $config;

    public function __construct(LoggerInterface $logger, array $config = [])
    {
        $this->logger = $logger;
        $this->config = $config ?: $this->loadConfig();
    }

    private function loadConfig(): array
    {
        return [
            'scheme' => getenv('TOKENGINX_SCHEME') ?: 'tls',
            'cluster' => array_map(
                fn($node) => trim($node),
                explode(',', getenv('TOKENGINX_NODES'))
            ),
            'parameters' => [
                'password' => getenv('TOKENGINX_PASSWORD'),
                'database' => (int)(getenv('TOKENGINX_DATABASE') ?: 0),
                'read_write_timeout' => (float)(getenv('TOKENGINX_READ_TIMEOUT') ?: 3),
                'timeout' => (float)(getenv('TOKENGINX_TIMEOUT') ?: 5),
            ],
            'options' => [
                'cluster' => 'redis',
                'parameters' => [
                    'password' => getenv('TOKENGINX_PASSWORD'),
                ],
                'ssl' => [
                    'verify_peer' => filter_var(
                        getenv('TOKENGINX_SSL_VERIFY_PEER'),
                        FILTER_VALIDATE_BOOLEAN
                    ),
                    'verify_peer_name' => true,
                    'cafile' => getenv('TOKENGINX_SSL_CAFILE'),
                    'local_cert' => getenv('TOKENGINX_SSL_CERTFILE'),
                    'local_pk' => getenv('TOKENGINX_SSL_KEYFILE'),
                ],
            ],
        ];
    }

    public function getClient(): Client
    {
        if (self::$client === null) {
            self::$client = $this->createClient();
        }

        return self::$client;
    }

    private function createClient(): Client
    {
        try {
            $client = new Client(
                $this->config['cluster'],
                $this->config['options']
            );

            // 测试连接
            $client->ping();

            $this->logger->info('TokenginX connection established', [
                'nodes' => count($this->config['cluster']),
            ]);

            return $client;
        } catch (\Exception $e) {
            $this->logger->error('Failed to connect to TokenginX', [
                'error' => $e->getMessage(),
                'nodes' => $this->config['cluster'],
            ]);

            throw $e;
        }
    }

    public function disconnect(): void
    {
        if (self::$client !== null) {
            self::$client->disconnect();
            self::$client = null;
            $this->logger->info('TokenginX connection closed');
        }
    }
}
```

### 4. 生产级服务实现

创建 `src/TokenginX/TokenginXService.php`:

```php
<?php

namespace App\TokenginX;

use Predis\Client;
use Psr\Log\LoggerInterface;
use Prometheus\Counter;
use Prometheus\Histogram;
use Prometheus\CollectorRegistry;

class TokenginXService
{
    private Client $client;
    private LoggerInterface $logger;
    private string $prefix;

    // Prometheus 指标
    private Counter $tokenCreatedCounter;
    private Counter $tokenRetrievedCounter;
    private Histogram $operationDuration;

    public function __construct(
        Client $client,
        LoggerInterface $logger,
        CollectorRegistry $registry,
        string $prefix = 'myapp:'
    ) {
        $this->client = $client;
        $this->logger = $logger;
        $this->prefix = $prefix;

        // 初始化监控指标
        $this->tokenCreatedCounter = $registry->getOrRegisterCounter(
            'tokenginx',
            'token_created_total',
            'Total tokens created'
        );

        $this->tokenRetrievedCounter = $registry->getOrRegisterCounter(
            'tokenginx',
            'token_retrieved_total',
            'Total tokens retrieved'
        );

        $this->operationDuration = $registry->getOrRegisterHistogram(
            'tokenginx',
            'operation_duration_seconds',
            'Duration of TokenginX operations',
            ['operation'],
            [0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1.0]
        );
    }

    /**
     * 设置 OAuth Token(带重试和监控)
     */
    public function setOAuthToken(
        string $tokenId,
        string $userId,
        string $scope,
        int $ttlSeconds
    ): Result {
        $startTime = microtime(true);

        try {
            $tokenData = [
                'user_id' => $userId,
                'scope' => $scope,
                'created_at' => time(),
                'client_ip' => $_SERVER['REMOTE_ADDR'] ?? null,
            ];

            $key = $this->prefix . "oauth:token:{$tokenId}";
            $value = json_encode($tokenData, JSON_THROW_ON_ERROR);

            $success = $this->retryOperation(function () use ($key, $value, $ttlSeconds) {
                $result = $this->client->setex($key, $ttlSeconds, $value);
                return $result === 'OK';
            });

            if ($success) {
                $this->tokenCreatedCounter->inc();
                $this->logger->info('OAuth token created', [
                    'token_id' => $tokenId,
                    'user_id' => $userId,
                    'ttl' => $ttlSeconds,
                ]);
            } else {
                $this->logger->warning('Failed to create OAuth token', [
                    'token_id' => $tokenId,
                ]);
            }

            return Result::success($success);
        } catch (\Exception $e) {
            $this->logger->error('Error setting OAuth token', [
                'token_id' => $tokenId,
                'error' => $e->getMessage(),
            ]);

            return Result::failure('Failed to set token: ' . $e->getMessage());
        } finally {
            $duration = microtime(true) - $startTime;
            $this->operationDuration->observe($duration, ['set']);
        }
    }

    /**
     * 获取 OAuth Token
     */
    public function getOAuthToken(string $tokenId): Result
    {
        $startTime = microtime(true);

        try {
            $key = $this->prefix . "oauth:token:{$tokenId}";

            $value = $this->retryOperation(function () use ($key) {
                return $this->client->get($key);
            });

            if ($value === null) {
                $this->logger->debug('OAuth token not found', [
                    'token_id' => $tokenId,
                ]);

                return Result::failure('Token not found');
            }

            $token = json_decode($value, true, 512, JSON_THROW_ON_ERROR);
            $this->tokenRetrievedCounter->inc();

            $this->logger->debug('OAuth token retrieved', [
                'token_id' => $tokenId,
            ]);

            return Result::success($token);
        } catch (\Exception $e) {
            $this->logger->error('Error getting OAuth token', [
                'token_id' => $tokenId,
                'error' => $e->getMessage(),
            ]);

            return Result::failure('Failed to get token: ' . $e->getMessage());
        } finally {
            $duration = microtime(true) - $startTime;
            $this->operationDuration->observe($duration, ['get']);
        }
    }

    /**
     * 批量获取 Token(使用 Pipeline)
     */
    public function getMultipleTokens(array $tokenIds): Result
    {
        $startTime = microtime(true);

        try {
            $keys = array_map(
                fn($id) => $this->prefix . "oauth:token:{$id}",
                $tokenIds
            );

            $pipeline = $this->client->pipeline();
            foreach ($keys as $key) {
                $pipeline->get($key);
            }
            $values = $pipeline->execute();

            $result = [];
            foreach ($tokenIds as $index => $tokenId) {
                if ($values[$index] !== null) {
                    $result[$tokenId] = json_decode(
                        $values[$index],
                        true,
                        512,
                        JSON_THROW_ON_ERROR
                    );
                } else {
                    $result[$tokenId] = null;
                }
            }

            $this->logger->debug('Batch retrieved tokens', [
                'total' => count($tokenIds),
                'found' => count(array_filter($result)),
            ]);

            return Result::success($result);
        } catch (\Exception $e) {
            $this->logger->error('Error getting multiple tokens', [
                'error' => $e->getMessage(),
            ]);

            return Result::failure('Batch get failed: ' . $e->getMessage());
        } finally {
            $duration = microtime(true) - $startTime;
            $this->operationDuration->observe($duration, ['batch_get']);
        }
    }

    /**
     * 删除 Token
     */
    public function deleteToken(string $tokenId): Result
    {
        try {
            $key = $this->prefix . "oauth:token:{$tokenId}";
            $deleted = $this->client->del([$key]);

            $this->logger->info('Token deleted', [
                'token_id' => $tokenId,
                'success' => $deleted > 0,
            ]);

            return Result::success($deleted > 0);
        } catch (\Exception $e) {
            $this->logger->error('Error deleting token', [
                'token_id' => $tokenId,
                'error' => $e->getMessage(),
            ]);

            return Result::failure('Failed to delete token: ' . $e->getMessage());
        }
    }

    /**
     * 获取 Token TTL
     */
    public function getTokenTTL(string $tokenId): Result
    {
        try {
            $key = $this->prefix . "oauth:token:{$tokenId}";
            $ttl = $this->client->ttl($key);

            if ($ttl < 0) {
                return Result::failure('Token not found or no expiration');
            }

            return Result::success($ttl);
        } catch (\Exception $e) {
            $this->logger->error('Error getting TTL', [
                'token_id' => $tokenId,
                'error' => $e->getMessage(),
            ]);

            return Result::failure('Failed to get TTL: ' . $e->getMessage());
        }
    }

    /**
     * 健康检查
     */
    public function isHealthy(): bool
    {
        try {
            $pong = $this->client->ping();
            return $pong === 'PONG';
        } catch (\Exception $e) {
            $this->logger->error('Health check failed', [
                'error' => $e->getMessage(),
            ]);

            return false;
        }
    }

    /**
     * 重试机制
     */
    private function retryOperation(callable $operation, int $maxRetries = 3)
    {
        $lastException = null;

        for ($i = 0; $i < $maxRetries; $i++) {
            try {
                return $operation();
            } catch (\Predis\Connection\ConnectionException $e) {
                $lastException = $e;

                if ($i < $maxRetries - 1) {
                    $delay = 100 * ($i + 1); // 指数退避
                    usleep($delay * 1000);

                    $this->logger->warning('Retrying operation', [
                        'attempt' => $i + 1,
                        'max_retries' => $maxRetries,
                    ]);
                }
            }
        }

        throw $lastException;
    }
}

/**
 * 结果类
 */
class Result
{
    private bool $success;
    private mixed $data;
    private ?string $error;

    private function __construct(bool $success, mixed $data, ?string $error)
    {
        $this->success = $success;
        $this->data = $data;
        $this->error = $error;
    }

    public static function success(mixed $data): self
    {
        return new self(true, $data, null);
    }

    public static function failure(string $error): self
    {
        return new self(false, null, $error);
    }

    public function isSuccess(): bool
    {
        return $this->success;
    }

    public function getData(): mixed
    {
        return $this->data;
    }

    public function getError(): ?string
    {
        return $this->error;
    }
}
```

### 5. 依赖注入容器配置

创建 `src/Container.php`:

```php
<?php

namespace App;

use App\TokenginX\ConnectionManager;
use App\TokenginX\TokenginXService;
use Monolog\Logger;
use Monolog\Handler\StreamHandler;
use Monolog\Handler\SyslogHandler;
use Monolog\Formatter\JsonFormatter;
use Prometheus\CollectorRegistry;
use Prometheus\Storage\APC;

class Container
{
    private array $services = [];

    public function get(string $id)
    {
        if (!isset($this->services[$id])) {
            $this->services[$id] = $this->create($id);
        }

        return $this->services[$id];
    }

    private function create(string $id)
    {
        switch ($id) {
            case 'logger':
                return $this->createLogger();

            case 'prometheus':
                return new CollectorRegistry(new APC());

            case 'connection_manager':
                return new ConnectionManager($this->get('logger'));

            case 'tokenginx':
                return new TokenginXService(
                    $this->get('connection_manager')->getClient(),
                    $this->get('logger'),
                    $this->get('prometheus'),
                    getenv('TOKENGINX_PREFIX') ?: 'myapp:'
                );

            default:
                throw new \Exception("Service {$id} not found");
        }
    }

    private function createLogger(): Logger
    {
        $logger = new Logger('tokenginx');

        // 文件日志
        $fileHandler = new StreamHandler(
            '/var/log/myapp/application.log',
            Logger::WARNING
        );
        $fileHandler->setFormatter(new JsonFormatter());
        $logger->pushHandler($fileHandler);

        // Syslog
        if (getenv('APP_ENV') === 'production') {
            $syslogHandler = new SyslogHandler('myapp', LOG_USER, Logger::WARNING);
            $logger->pushHandler($syslogHandler);
        }

        return $logger;
    }
}
```

## 监控和日志

### 1. Prometheus 端点

创建 `public/metrics.php`:

```php
<?php

require_once __DIR__ . '/../vendor/autoload.php';

use Prometheus\RenderTextFormat;

$container = new App\Container();
$registry = $container->get('prometheus');

header('Content-Type: ' . RenderTextFormat::MIME_TYPE);
echo (new RenderTextFormat())->render($registry->getMetricFamilySamples());
```

### 2. 健康检查端点

创建 `public/health.php`:

```php
<?php

require_once __DIR__ . '/../vendor/autoload.php';

$container = new App\Container();
$tokenginx = $container->get('tokenginx');

header('Content-Type: application/json');

$healthy = $tokenginx->isHealthy();

http_response_code($healthy ? 200 : 503);

echo json_encode([
    'status' => $healthy ? 'healthy' : 'unhealthy',
    'timestamp' => date('c'),
], JSON_THROW_ON_ERROR);
```

## 性能优化

### 1. PHP-FPM 配置

在 `/etc/php/8.3/fpm/pool.d/www.conf`:

```ini
[www]
user = www-data
group = www-data

listen = /run/php/php8.3-fpm.sock
listen.owner = www-data
listen.group = www-data

pm = dynamic
pm.max_children = 50
pm.start_servers = 10
pm.min_spare_servers = 10
pm.max_spare_servers = 20
pm.max_requests = 500

; 性能优化
pm.process_idle_timeout = 10s
request_terminate_timeout = 30s

; OPcache 状态
php_admin_value[opcache.enable] = 1
php_admin_value[opcache.memory_consumption] = 256
php_admin_value[opcache.interned_strings_buffer] = 16
php_admin_value[opcache.max_accelerated_files] = 10000
php_admin_value[opcache.validate_timestamps] = 0
php_admin_value[opcache.save_comments] = 0
```

### 2. OPcache 配置

在 `php.ini`:

```ini
[opcache]
opcache.enable=1
opcache.enable_cli=0
opcache.memory_consumption=256
opcache.interned_strings_buffer=16
opcache.max_accelerated_files=10000
opcache.validate_timestamps=0
opcache.save_comments=0
opcache.fast_shutdown=1
```

### 3. APCu 配置

```ini
[apcu]
apc.enabled=1
apc.shm_size=128M
apc.ttl=7200
apc.gc_ttl=3600
apc.entries_hint=4096
```

## Nginx 配置

创建 `/etc/nginx/sites-available/myapp`:

```nginx
upstream php-fpm {
    server unix:/run/php/php8.3-fpm.sock;
    keepalive 32;
}

server {
    listen 80;
    listen [::]:80;
    server_name myapp.example.com;

    # 重定向到 HTTPS
    return 301 https://$server_name$request_uri;
}

server {
    listen 443 ssl http2;
    listen [::]:443 ssl http2;
    server_name myapp.example.com;

    root /var/www/myapp/public;
    index index.php;

    # SSL 配置
    ssl_certificate /etc/ssl/certs/myapp.crt;
    ssl_certificate_key /etc/ssl/private/myapp.key;
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers HIGH:!aNULL:!MD5;
    ssl_prefer_server_ciphers on;

    # 日志
    access_log /var/log/nginx/myapp.access.log;
    error_log /var/log/nginx/myapp.error.log;

    # 健康检查端点
    location /health {
        access_log off;
        try_files $uri /health.php$is_args$args;
    }

    # Prometheus 端点
    location /metrics {
        try_files $uri /metrics.php$is_args$args;
    }

    # PHP 处理
    location ~ \.php$ {
        fastcgi_pass php-fpm;
        fastcgi_index index.php;
        fastcgi_param SCRIPT_FILENAME $document_root$fastcgi_script_name;
        include fastcgi_params;

        # 性能优化
        fastcgi_keep_conn on;
        fastcgi_buffering on;
        fastcgi_buffer_size 32k;
        fastcgi_buffers 16 32k;
    }

    # 静态文件缓存
    location ~* \.(jpg|jpeg|png|gif|ico|css|js)$ {
        expires 1y;
        add_header Cache-Control "public, immutable";
    }
}
```

## Docker 部署

### Dockerfile

```dockerfile
FROM php:8.3-fpm-alpine AS base

RUN apk add --no-cache \
    linux-headers \
    libssl3 \
    openssl-dev \
    && docker-php-ext-install opcache \
    && pecl install apcu redis \
    && docker-php-ext-enable apcu redis

WORKDIR /var/www/html

FROM composer:2 AS build
WORKDIR /app
COPY composer.json composer.lock ./
RUN composer install --no-dev --optimize-autoloader --no-scripts
COPY . .
RUN composer dump-autoload --optimize

FROM base AS final
COPY --from=build /app /var/www/html

# PHP-FPM 配置
COPY docker/php-fpm.conf /usr/local/etc/php-fpm.d/www.conf
COPY docker/php.ini /usr/local/etc/php/php.ini

# 权限
RUN chown -R www-data:www-data /var/www/html

USER www-data

EXPOSE 9000

CMD ["php-fpm"]
```

### docker-compose.yml

```yaml
version: '3.8'

services:
  app:
    build:
      context: .
      dockerfile: Dockerfile
    volumes:
      - ./:/var/www/html:ro
      - ./logs:/var/log/myapp
    environment:
      - APP_ENV=production
      - TOKENGINX_NODES=${TOKENGINX_NODES}
      - TOKENGINX_PASSWORD=${TOKENGINX_PASSWORD}
    depends_on:
      - tokenginx
    restart: unless-stopped

  nginx:
    image: nginx:alpine
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - ./:/var/www/html:ro
      - ./docker/nginx.conf:/etc/nginx/conf.d/default.conf:ro
    depends_on:
      - app
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "wget", "-q", "--spider", "http://localhost/health"]
      interval: 30s
      timeout: 10s
      retries: 3

  tokenginx:
    image: tokenginx/tokenginx-server:latest
    ports:
      - "6380:6380"
    restart: unless-stopped
```

## Kubernetes 部署

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: myapp-php
spec:
  replicas: 3
  selector:
    matchLabels:
      app: myapp-php
  template:
    metadata:
      labels:
        app: myapp-php
    spec:
      containers:
      - name: php-fpm
        image: myapp-php:1.0.0
        ports:
        - containerPort: 9000
        env:
        - name: APP_ENV
          value: "production"
        - name: TOKENGINX_PASSWORD
          valueFrom:
            secretKeyRef:
              name: tokenginx-secret
              key: api-key
        resources:
          requests:
            memory: "256Mi"
            cpu: "100m"
          limits:
            memory: "512Mi"
            cpu: "500m"

      - name: nginx
        image: nginx:alpine
        ports:
        - containerPort: 80
        volumeMounts:
        - name: nginx-config
          mountPath: /etc/nginx/conf.d
        livenessProbe:
          httpGet:
            path: /health
            port: 80
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /health
            port: 80
          initialDelaySeconds: 10
          periodSeconds: 5

      volumes:
      - name: nginx-config
        configMap:
          name: nginx-config
```

## 最佳实践清单

- ✅ 使用连接单例,避免重复创建
- ✅ 启用 TLS/mTLS 加密通信
- ✅ 使用环境变量存储敏感信息
- ✅ 实现健康检查端点
- ✅ 添加 Prometheus 监控指标
- ✅ 使用结构化日志(JSON 格式)
- ✅ 实现重试机制
- ✅ 使用 Pipeline 批量操作
- ✅ 启用 OPcache 和 APCu
- ✅ 配置 PHP-FPM 连接池
- ✅ 使用 Nginx 反向代理
- ✅ 优化 Composer autoloader

## 下一步

- 查看 [OAuth 2.0 集成](../protocols/oauth.md)
- 配置 [TLS/mTLS](../security/tls-mtls.md)
- 了解 [防重放攻击](../security/anti-replay.md)
