# Node.js/JavaScript 快速指南

本指南帮助 Node.js 和 JavaScript 开发者快速集成 TokenginX 进行会话管理。

## 前置要求

- Node.js 16+ (推荐 LTS 版本)
- npm 或 yarn 或 pnpm
- TokenginX 服务器运行中

## 安装客户端库

TokenginX 支持多种 Node.js 客户端方式：

### 方式 1: Redis 客户端（推荐）

使用 ioredis 连接 TokenginX 的 TCP (RESP) 接口：

```bash
npm install ioredis
# 或
yarn add ioredis
# 或
pnpm add ioredis
```

### 方式 2: HTTP 客户端

```bash
npm install axios
# 或使用内置的 fetch (Node.js 18+)
```

### 方式 3: gRPC 客户端

```bash
npm install @grpc/grpc-js @grpc/proto-loader
```

## 快速开始

### 使用 Redis 客户端 (ioredis)

```javascript
const Redis = require('ioredis');

// 连接到 TokenginX
const client = new Redis({
  host: 'localhost',
  port: 6380,
  retryStrategy: (times) => {
    return Math.min(times * 50, 2000);
  }
});

// 设置会话（3600秒后过期）
async function setSession() {
  const sessionData = {
    userId: 'user123',
    username: 'john_doe',
    email: 'john@example.com',
    roles: ['user', 'admin'],
    createdAt: new Date().toISOString()
  };

  await client.setex(
    'oauth:token:abc123',
    3600, // TTL 秒数
    JSON.stringify(sessionData)
  );

  console.log('Session created');
}

// 获取会话
async function getSession(token) {
  const data = await client.get(`oauth:token:${token}`);

  if (!data) {
    console.log('Token not found or expired');
    return null;
  }

  const session = JSON.parse(data);
  console.log(`User: ${session.username}`);
  return session;
}

// 检查会话是否存在
async function sessionExists(token) {
  const exists = await client.exists(`oauth:token:${token}`);
  console.log(`Token exists: ${exists === 1}`);
  return exists === 1;
}

// 获取剩余 TTL
async function getSessionTTL(token) {
  const ttl = await client.ttl(`oauth:token:${token}`);
  console.log(`Token expires in: ${ttl} seconds`);
  return ttl;
}

// 删除会话
async function deleteSession(token) {
  await client.del(`oauth:token:${token}`);
  console.log('Session deleted');
}

// 使用示例
(async () => {
  await setSession();
  await getSession('abc123');
  await sessionExists('abc123');
  await getSessionTTL('abc123');
  await deleteSession('abc123');

  client.disconnect();
})();
```

### 使用 HTTP/REST 客户端 (Axios)

```javascript
const axios = require('axios');

class TokenginxClient {
  constructor(baseURL = 'http://localhost:8080/api/v1') {
    this.client = axios.create({
      baseURL,
      timeout: 5000
    });
  }

  async setSession(key, value, ttl = 3600) {
    try {
      const response = await this.client.post('/sessions', {
        key,
        value,
        ttl
      });
      return response.data;
    } catch (error) {
      console.error('Set session error:', error.message);
      throw error;
    }
  }

  async getSession(key) {
    try {
      const response = await this.client.get(`/sessions/${key}`);
      return response.data;
    } catch (error) {
      if (error.response?.status === 404) {
        return null;
      }
      throw error;
    }
  }

  async deleteSession(key) {
    try {
      const response = await this.client.delete(`/sessions/${key}`);
      return response.data;
    } catch (error) {
      console.error('Delete session error:', error.message);
      throw error;
    }
  }

  async exists(key) {
    try {
      await this.client.head(`/sessions/${key}`);
      return true;
    } catch (error) {
      if (error.response?.status === 404) {
        return false;
      }
      throw error;
    }
  }
}

// 使用示例
const client = new TokenginxClient();

(async () => {
  // 设置会话
  const sessionData = {
    userId: 'user123',
    username: 'john_doe',
    email: 'john@example.com'
  };

  await client.setSession('oauth:token:abc123', sessionData, 3600);

  // 获取会话
  const session = await client.getSession('oauth:token:abc123');
  if (session) {
    console.log(`User: ${session.value.username}`);
  }

  // 删除会话
  await client.deleteSession('oauth:token:abc123');
})();
```

## Express.js 集成

### 会话中间件

```javascript
const express = require('express');
const Redis = require('ioredis');
const crypto = require('crypto');

const app = express();
app.use(express.json());

// TokenginX 客户端
const tokenginx = new Redis({
  host: 'localhost',
  port: 6380
});

// 认证中间件
const requireAuth = async (req, res, next) => {
  try {
    // 从请求头获取 token
    const authHeader = req.headers.authorization;
    if (!authHeader || !authHeader.startsWith('Bearer ')) {
      return res.status(401).json({ error: 'Unauthorized' });
    }

    const token = authHeader.replace('Bearer ', '');

    // 从 TokenginX 获取会话
    const sessionKey = `oauth:token:${token}`;
    const sessionData = await tokenginx.get(sessionKey);

    if (!sessionData) {
      return res.status(401).json({ error: 'Invalid or expired token' });
    }

    // 将会话数据存储到 req 对象
    req.session = JSON.parse(sessionData);
    req.token = token;

    next();
  } catch (error) {
    console.error('Auth error:', error);
    res.status(500).json({ error: 'Internal server error' });
  }
};

// 登录端点
app.post('/api/login', async (req, res) => {
  try {
    const { username, password } = req.body;

    // 验证用户（示例）
    if (username === 'admin' && password === 'password') {
      // 生成 token
      const token = crypto.randomBytes(32).toString('hex');

      // 创建会话数据
      const sessionData = {
        userId: 'user123',
        username,
        roles: ['admin'],
        createdAt: new Date().toISOString()
      };

      // 存储到 TokenginX（1小时过期）
      const sessionKey = `oauth:token:${token}`;
      await tokenginx.setex(sessionKey, 3600, JSON.stringify(sessionData));

      return res.json({
        token,
        expiresIn: 3600
      });
    }

    res.status(401).json({ error: 'Invalid credentials' });
  } catch (error) {
    console.error('Login error:', error);
    res.status(500).json({ error: 'Internal server error' });
  }
});

// 受保护的端点
app.get('/api/profile', requireAuth, (req, res) => {
  res.json({
    userId: req.session.userId,
    username: req.session.username,
    roles: req.session.roles
  });
});

// 登出端点
app.post('/api/logout', requireAuth, async (req, res) => {
  try {
    const sessionKey = `oauth:token:${req.token}`;
    await tokenginx.del(sessionKey);
    res.json({ message: 'Logged out successfully' });
  } catch (error) {
    console.error('Logout error:', error);
    res.status(500).json({ error: 'Internal server error' });
  }
});

const PORT = 3000;
app.listen(PORT, () => {
  console.log(`Server running on port ${PORT}`);
});
```

## NestJS 集成

### TokenginX 模块

```typescript
// tokenginx.module.ts
import { Module, Global } from '@nestjs/common';
import { TokenginxService } from './tokenginx.service';

@Global()
@Module({
  providers: [TokenginxService],
  exports: [TokenginxService],
})
export class TokenginxModule {}
```

### TokenginX 服务

```typescript
// tokenginx.service.ts
import { Injectable, OnModuleDestroy } from '@nestjs/common';
import Redis from 'ioredis';

@Injectable()
export class TokenginxService implements OnModuleDestroy {
  private readonly client: Redis;

  constructor() {
    this.client = new Redis({
      host: process.env.TOKENGINX_HOST || 'localhost',
      port: parseInt(process.env.TOKENGINX_PORT || '6380'),
    });
  }

  async setSession(key: string, value: any, ttl: number = 3600): Promise<void> {
    await this.client.setex(key, ttl, JSON.stringify(value));
  }

  async getSession<T>(key: string): Promise<T | null> {
    const data = await this.client.get(key);
    if (!data) return null;
    return JSON.parse(data) as T;
  }

  async deleteSession(key: string): Promise<void> {
    await this.client.del(key);
  }

  async exists(key: string): Promise<boolean> {
    const result = await this.client.exists(key);
    return result === 1;
  }

  async getTTL(key: string): Promise<number> {
    return await this.client.ttl(key);
  }

  onModuleDestroy() {
    this.client.disconnect();
  }
}
```

### 认证守卫

```typescript
// auth.guard.ts
import {
  Injectable,
  CanActivate,
  ExecutionContext,
  UnauthorizedException,
} from '@nestjs/common';
import { TokenginxService } from './tokenginx.service';

interface SessionData {
  userId: string;
  username: string;
  roles: string[];
}

@Injectable()
export class AuthGuard implements CanActivate {
  constructor(private readonly tokenginx: TokenginxService) {}

  async canActivate(context: ExecutionContext): Promise<boolean> {
    const request = context.switchToHttp().getRequest();
    const authHeader = request.headers.authorization;

    if (!authHeader || !authHeader.startsWith('Bearer ')) {
      throw new UnauthorizedException('Missing or invalid authorization header');
    }

    const token = authHeader.replace('Bearer ', '');
    const sessionKey = `oauth:token:${token}`;

    const session = await this.tokenginx.getSession<SessionData>(sessionKey);

    if (!session) {
      throw new UnauthorizedException('Invalid or expired token');
    }

    request.user = session;
    request.token = token;

    return true;
  }
}
```

### 控制器

```typescript
// auth.controller.ts
import { Controller, Post, Get, Body, UseGuards, Request } from '@nestjs/common';
import { TokenginxService } from './tokenginx.service';
import { AuthGuard } from './auth.guard';
import * as crypto from 'crypto';

@Controller('api')
export class AuthController {
  constructor(private readonly tokenginx: TokenginxService) {}

  @Post('login')
  async login(@Body() body: { username: string; password: string }) {
    const { username, password } = body;

    // 验证用户（示例）
    if (username === 'admin' && password === 'password') {
      // 生成 token
      const token = crypto.randomBytes(32).toString('hex');

      // 创建会话数据
      const sessionData = {
        userId: 'user123',
        username,
        roles: ['admin'],
        createdAt: new Date().toISOString(),
      };

      // 存储到 TokenginX
      await this.tokenginx.setSession(
        `oauth:token:${token}`,
        sessionData,
        3600
      );

      return {
        token,
        expiresIn: 3600,
      };
    }

    throw new UnauthorizedException('Invalid credentials');
  }

  @Get('profile')
  @UseGuards(AuthGuard)
  getProfile(@Request() req) {
    return {
      userId: req.user.userId,
      username: req.user.username,
      roles: req.user.roles,
    };
  }

  @Post('logout')
  @UseGuards(AuthGuard)
  async logout(@Request() req) {
    await this.tokenginx.deleteSession(`oauth:token:${req.token}`);
    return { message: 'Logged out successfully' };
  }
}
```

## OAuth 2.0 Token Store

```javascript
const Redis = require('ioredis');
const crypto = require('crypto');

class OAuth2TokenStore {
  constructor(host = 'localhost', port = 6380) {
    this.client = new Redis({ host, port });
  }

  // 创建 Access Token
  async createAccessToken(userId, clientId, scopes, ttl = 3600) {
    const token = crypto.randomBytes(32).toString('hex');

    const tokenData = {
      userId,
      clientId,
      scopes,
      tokenType: 'Bearer',
      createdAt: new Date().toISOString()
    };

    const key = `oauth:access_token:${token}`;
    await this.client.setex(key, ttl, JSON.stringify(tokenData));

    return token;
  }

  // 创建 Refresh Token（30天）
  async createRefreshToken(userId, clientId, scopes, ttl = 2592000) {
    const token = crypto.randomBytes(32).toString('hex');

    const tokenData = {
      userId,
      clientId,
      scopes,
      tokenType: 'refresh',
      createdAt: new Date().toISOString()
    };

    const key = `oauth:refresh_token:${token}`;
    await this.client.setex(key, ttl, JSON.stringify(tokenData));

    return token;
  }

  // 验证 Access Token
  async verifyAccessToken(token) {
    const key = `oauth:access_token:${token}`;
    const data = await this.client.get(key);

    if (!data) return null;

    return JSON.parse(data);
  }

  // 刷新 Access Token
  async refreshAccessToken(refreshToken) {
    const key = `oauth:refresh_token:${refreshToken}`;
    const data = await this.client.get(key);

    if (!data) return null;

    const refreshData = JSON.parse(data);

    // 创建新的 Access Token
    const accessToken = await this.createAccessToken(
      refreshData.userId,
      refreshData.clientId,
      refreshData.scopes
    );

    return accessToken;
  }

  // 撤销 Token
  async revokeToken(token, tokenType = 'access') {
    const key = `oauth:${tokenType}_token:${token}`;
    await this.client.del(key);
  }

  // 创建 Authorization Code
  async createAuthorizationCode(userId, clientId, redirectUri, scopes, ttl = 300) {
    const code = crypto.randomBytes(32).toString('hex');

    const codeData = {
      userId,
      clientId,
      redirectUri,
      scopes,
      createdAt: new Date().toISOString()
    };

    const key = `oauth:code:${code}`;
    await this.client.setex(key, ttl, JSON.stringify(codeData));

    return code;
  }

  // 验证并消费 Authorization Code（一次性使用）
  async consumeAuthorizationCode(code) {
    const key = `oauth:code:${code}`;
    const data = await this.client.get(key);

    if (!data) return null;

    // 删除 code（一次性使用）
    await this.client.del(key);

    return JSON.parse(data);
  }
}

// 使用示例
(async () => {
  const store = new OAuth2TokenStore();

  // 创建 tokens
  const accessToken = await store.createAccessToken(
    'user123',
    'client_app',
    ['read', 'write']
  );

  const refreshToken = await store.createRefreshToken(
    'user123',
    'client_app',
    ['read', 'write']
  );

  console.log('Access Token:', accessToken);
  console.log('Refresh Token:', refreshToken);

  // 验证 token
  const tokenData = await store.verifyAccessToken(accessToken);
  if (tokenData) {
    console.log('Token valid for user:', tokenData.userId);
  }

  // 刷新 token
  const newAccessToken = await store.refreshAccessToken(refreshToken);
  console.log('New Access Token:', newAccessToken);

  store.client.disconnect();
})();
```

## TypeScript 支持

```typescript
// types.ts
export interface SessionData {
  userId: string;
  username: string;
  email?: string;
  roles: string[];
  createdAt: string;
}

export interface TokenData {
  userId: string;
  clientId: string;
  scopes: string[];
  tokenType: string;
  createdAt: string;
}

// client.ts
import Redis from 'ioredis';
import { SessionData } from './types';

export class TokenginxClient {
  private client: Redis;

  constructor(host = 'localhost', port = 6380) {
    this.client = new Redis({ host, port });
  }

  async setSession(
    key: string,
    value: SessionData,
    ttl: number = 3600
  ): Promise<void> {
    await this.client.setex(key, ttl, JSON.stringify(value));
  }

  async getSession(key: string): Promise<SessionData | null> {
    const data = await this.client.get(key);
    if (!data) return null;
    return JSON.parse(data) as SessionData;
  }

  async deleteSession(key: string): Promise<void> {
    await this.client.del(key);
  }

  async exists(key: string): Promise<boolean> {
    const result = await this.client.exists(key);
    return result === 1;
  }

  disconnect(): void {
    this.client.disconnect();
  }
}
```

## 连接池配置

```javascript
const Redis = require('ioredis');

// 创建连接池（Cluster 模式）
const cluster = new Redis.Cluster([
  {
    host: 'localhost',
    port: 6380
  }
], {
  redisOptions: {
    maxRetriesPerRequest: 3,
    retryStrategy: (times) => Math.min(times * 50, 2000)
  },
  clusterRetryStrategy: (times) => Math.min(times * 100, 2000)
});

// 或使用单实例连接池
const client = new Redis({
  host: 'localhost',
  port: 6380,
  maxRetriesPerRequest: 3,
  retryStrategy: (times) => Math.min(times * 50, 2000),
  lazyConnect: false,
  keepAlive: 30000
});
```

## 错误处理

```javascript
const Redis = require('ioredis');

const client = new Redis({
  host: 'localhost',
  port: 6380,
  retryStrategy: (times) => {
    if (times > 3) {
      // 超过 3 次重试，停止重试
      return null;
    }
    return Math.min(times * 50, 2000);
  }
});

// 监听错误
client.on('error', (err) => {
  console.error('Redis connection error:', err);
});

client.on('connect', () => {
  console.log('Connected to TokenginX');
});

client.on('ready', () => {
  console.log('TokenginX client ready');
});

// 安全地执行操作
async function safeGetSession(token) {
  try {
    const key = `oauth:token:${token}`;
    const data = await client.get(key);

    if (!data) return null;

    return JSON.parse(data);
  } catch (error) {
    console.error('Error getting session:', error.message);
    return null;
  }
}
```

## 性能优化

### Pipeline 批量操作

```javascript
// 使用 pipeline 批量操作
async function batchSetSessions(sessions) {
  const pipeline = client.pipeline();

  for (const session of sessions) {
    pipeline.setex(
      `oauth:token:${session.token}`,
      3600,
      JSON.stringify(session.data)
    );
  }

  const results = await pipeline.exec();
  return results;
}
```

### Lua 脚本

```javascript
// 使用 Lua 脚本实现原子操作
const getAndRefreshScript = `
  local key = KEYS[1]
  local ttl = ARGV[1]
  local value = redis.call('GET', key)
  if value then
    redis.call('EXPIRE', key, ttl)
    return value
  else
    return nil
  end
`;

async function getAndRefreshSession(token, ttl = 3600) {
  const key = `oauth:token:${token}`;
  const result = await client.eval(getAndRefreshScript, 1, key, ttl);

  if (!result) return null;

  return JSON.parse(result);
}
```

## 下一步

- 查看 [Node.js 生产环境指南](../production/nodejs.md) 了解生产部署
- 查看 [OAuth 2.0/OIDC 集成指南](../protocols/oauth.md) 了解协议集成
- 查看 [API 参考文档](../reference/http-rest-api.md) 了解完整 API
