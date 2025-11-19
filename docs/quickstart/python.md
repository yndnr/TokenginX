# Python 快速指南

本指南帮助 Python 开发者快速集成 TokenginX 进行会话管理。

## 前置要求

- Python 3.8+
- pip 包管理器
- TokenginX 服务器运行中

## 安装客户端库

TokenginX 支持多种 Python 客户端方式：

### 方式 1: Redis 客户端（推荐）

使用 redis-py 连接 TokenginX 的 TCP (RESP) 接口：

```bash
pip install redis
```

### 方式 2: HTTP 客户端

使用 requests 连接 TokenginX 的 HTTP/REST 接口：

```bash
pip install requests
```

### 方式 3: gRPC 客户端

```bash
pip install grpcio grpcio-tools
```

## 快速开始

### 使用 Redis 客户端

```python
import redis
import json
from datetime import datetime, timedelta

# 连接到 TokenginX
client = redis.Redis(
    host='localhost',
    port=6380,
    decode_responses=True  # 自动解码为字符串
)

# 设置会话（3600秒后过期）
session_data = {
    'user_id': 'user123',
    'username': 'john_doe',
    'email': 'john@example.com',
    'roles': ['user', 'admin'],
    'created_at': datetime.now().isoformat()
}

client.setex(
    name='oauth:token:abc123',
    time=3600,  # TTL 秒数
    value=json.dumps(session_data)
)

# 获取会话
token_data = client.get('oauth:token:abc123')
if token_data:
    session = json.loads(token_data)
    print(f"User: {session['username']}")
else:
    print("Token not found or expired")

# 检查会话是否存在
exists = client.exists('oauth:token:abc123')
print(f"Token exists: {bool(exists)}")

# 获取剩余 TTL
ttl = client.ttl('oauth:token:abc123')
print(f"Token expires in: {ttl} seconds")

# 删除会话
client.delete('oauth:token:abc123')
```

### 使用 HTTP/REST 客户端

```python
import requests
import json

BASE_URL = 'http://localhost:8080/api/v1'

class TokenginxClient:
    def __init__(self, base_url=BASE_URL):
        self.base_url = base_url
        self.session = requests.Session()

    def set_session(self, key, value, ttl=3600):
        """设置会话"""
        response = self.session.post(
            f'{self.base_url}/sessions',
            json={
                'key': key,
                'value': value,
                'ttl': ttl
            }
        )
        response.raise_for_status()
        return response.json()

    def get_session(self, key):
        """获取会话"""
        response = self.session.get(
            f'{self.base_url}/sessions/{key}'
        )
        if response.status_code == 404:
            return None
        response.raise_for_status()
        return response.json()

    def delete_session(self, key):
        """删除会话"""
        response = self.session.delete(
            f'{self.base_url}/sessions/{key}'
        )
        response.raise_for_status()
        return response.json()

    def exists(self, key):
        """检查会话是否存在"""
        response = self.session.head(
            f'{self.base_url}/sessions/{key}'
        )
        return response.status_code == 200

# 使用示例
client = TokenginxClient()

# 设置会话
session_data = {
    'user_id': 'user123',
    'username': 'john_doe',
    'email': 'john@example.com'
}

client.set_session('oauth:token:abc123', session_data, ttl=3600)

# 获取会话
session = client.get_session('oauth:token:abc123')
if session:
    print(f"User: {session['value']['username']}")

# 删除会话
client.delete_session('oauth:token:abc123')
```

## Flask 集成

### 会话中间件

```python
from flask import Flask, request, jsonify, g
import redis
import json
from functools import wraps

app = Flask(__name__)

# TokenginX 客户端
tokenginx = redis.Redis(
    host='localhost',
    port=6380,
    decode_responses=True
)

def require_auth(f):
    """认证装饰器"""
    @wraps(f)
    def decorated_function(*args, **kwargs):
        # 从请求头获取 token
        auth_header = request.headers.get('Authorization')
        if not auth_header or not auth_header.startswith('Bearer '):
            return jsonify({'error': 'Unauthorized'}), 401

        token = auth_header.replace('Bearer ', '')

        # 从 TokenginX 获取会话
        session_key = f'oauth:token:{token}'
        session_data = tokenginx.get(session_key)

        if not session_data:
            return jsonify({'error': 'Invalid or expired token'}), 401

        # 将会话数据存储到 g 对象
        g.session = json.loads(session_data)
        g.token = token

        return f(*args, **kwargs)

    return decorated_function

@app.route('/api/login', methods=['POST'])
def login():
    """登录端点"""
    data = request.json
    username = data.get('username')
    password = data.get('password')

    # 验证用户（示例）
    if username == 'admin' and password == 'password':
        # 生成 token
        import secrets
        token = secrets.token_urlsafe(32)

        # 创建会话数据
        session_data = {
            'user_id': 'user123',
            'username': username,
            'roles': ['admin']
        }

        # 存储到 TokenginX（1小时过期）
        session_key = f'oauth:token:{token}'
        tokenginx.setex(
            session_key,
            3600,
            json.dumps(session_data)
        )

        return jsonify({
            'token': token,
            'expires_in': 3600
        })

    return jsonify({'error': 'Invalid credentials'}), 401

@app.route('/api/profile', methods=['GET'])
@require_auth
def profile():
    """受保护的端点"""
    return jsonify({
        'user_id': g.session['user_id'],
        'username': g.session['username'],
        'roles': g.session['roles']
    })

@app.route('/api/logout', methods=['POST'])
@require_auth
def logout():
    """登出端点"""
    session_key = f'oauth:token:{g.token}'
    tokenginx.delete(session_key)
    return jsonify({'message': 'Logged out successfully'})

if __name__ == '__main__':
    app.run(debug=True)
```

## Django 集成

### 自定义会话后端

```python
# myapp/session_backend.py
from django.contrib.sessions.backends.base import SessionBase
import redis
import json

class TokenginxSessionStore(SessionBase):
    """TokenginX 会话后端"""

    def __init__(self, session_key=None):
        super().__init__(session_key)
        self.redis_client = redis.Redis(
            host='localhost',
            port=6380,
            decode_responses=True
        )

    def load(self):
        """加载会话数据"""
        session_key = f'django:session:{self.session_key}'
        data = self.redis_client.get(session_key)
        if data:
            return json.loads(data)
        return {}

    def create(self):
        """创建新会话"""
        self.session_key = self._get_new_session_key()
        self.modified = True
        self._session_cache = {}

    def save(self, must_create=False):
        """保存会话数据"""
        session_key = f'django:session:{self.session_key}'
        session_data = json.dumps(self._session_cache)

        # 设置过期时间（使用 Django 设置）
        ttl = self.get_expiry_age()

        self.redis_client.setex(session_key, ttl, session_data)

    def exists(self, session_key):
        """检查会话是否存在"""
        key = f'django:session:{session_key}'
        return self.redis_client.exists(key) > 0

    def delete(self, session_key=None):
        """删除会话"""
        if session_key is None:
            session_key = self.session_key
        key = f'django:session:{session_key}'
        self.redis_client.delete(key)
```

### 配置 Django

```python
# settings.py
SESSION_ENGINE = 'myapp.session_backend'
SESSION_COOKIE_AGE = 3600  # 1 小时
```

## FastAPI 集成

```python
from fastapi import FastAPI, Depends, HTTPException, status
from fastapi.security import HTTPBearer, HTTPAuthorizationCredentials
import redis
import json
from typing import Optional
import secrets

app = FastAPI()
security = HTTPBearer()

# TokenginX 客户端
tokenginx = redis.Redis(
    host='localhost',
    port=6380,
    decode_responses=True
)

class SessionData:
    def __init__(self, user_id: str, username: str, roles: list):
        self.user_id = user_id
        self.username = username
        self.roles = roles

async def get_current_user(
    credentials: HTTPAuthorizationCredentials = Depends(security)
) -> SessionData:
    """获取当前用户（依赖注入）"""
    token = credentials.credentials
    session_key = f'oauth:token:{token}'

    session_data = tokenginx.get(session_key)
    if not session_data:
        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED,
            detail="Invalid or expired token"
        )

    data = json.loads(session_data)
    return SessionData(
        user_id=data['user_id'],
        username=data['username'],
        roles=data.get('roles', [])
    )

@app.post("/api/login")
async def login(username: str, password: str):
    """登录端点"""
    # 验证用户（示例）
    if username == 'admin' and password == 'password':
        # 生成 token
        token = secrets.token_urlsafe(32)

        # 创建会话数据
        session_data = {
            'user_id': 'user123',
            'username': username,
            'roles': ['admin']
        }

        # 存储到 TokenginX
        session_key = f'oauth:token:{token}'
        tokenginx.setex(session_key, 3600, json.dumps(session_data))

        return {
            'token': token,
            'token_type': 'bearer',
            'expires_in': 3600
        }

    raise HTTPException(
        status_code=status.HTTP_401_UNAUTHORIZED,
        detail="Invalid credentials"
    )

@app.get("/api/profile")
async def profile(current_user: SessionData = Depends(get_current_user)):
    """受保护的端点"""
    return {
        'user_id': current_user.user_id,
        'username': current_user.username,
        'roles': current_user.roles
    }

@app.post("/api/logout")
async def logout(
    credentials: HTTPAuthorizationCredentials = Depends(security)
):
    """登出端点"""
    token = credentials.credentials
    session_key = f'oauth:token:{token}'
    tokenginx.delete(session_key)
    return {'message': 'Logged out successfully'}
```

## OAuth 2.0 集成示例

```python
import redis
import json
import secrets
from datetime import datetime, timedelta

class OAuth2TokenStore:
    """OAuth 2.0 Token 存储"""

    def __init__(self, host='localhost', port=6380):
        self.client = redis.Redis(
            host=host,
            port=port,
            decode_responses=True
        )

    def create_access_token(self, user_id, client_id, scopes, ttl=3600):
        """创建 Access Token"""
        token = secrets.token_urlsafe(32)

        token_data = {
            'user_id': user_id,
            'client_id': client_id,
            'scopes': scopes,
            'token_type': 'Bearer',
            'created_at': datetime.now().isoformat()
        }

        key = f'oauth:access_token:{token}'
        self.client.setex(key, ttl, json.dumps(token_data))

        return token

    def create_refresh_token(self, user_id, client_id, scopes, ttl=2592000):
        """创建 Refresh Token（30天）"""
        token = secrets.token_urlsafe(32)

        token_data = {
            'user_id': user_id,
            'client_id': client_id,
            'scopes': scopes,
            'token_type': 'refresh',
            'created_at': datetime.now().isoformat()
        }

        key = f'oauth:refresh_token:{token}'
        self.client.setex(key, ttl, json.dumps(token_data))

        return token

    def verify_access_token(self, token):
        """验证 Access Token"""
        key = f'oauth:access_token:{token}'
        data = self.client.get(key)

        if not data:
            return None

        return json.loads(data)

    def refresh_access_token(self, refresh_token):
        """使用 Refresh Token 刷新 Access Token"""
        key = f'oauth:refresh_token:{refresh_token}'
        data = self.client.get(key)

        if not data:
            return None

        refresh_data = json.loads(data)

        # 创建新的 Access Token
        access_token = self.create_access_token(
            user_id=refresh_data['user_id'],
            client_id=refresh_data['client_id'],
            scopes=refresh_data['scopes']
        )

        return access_token

    def revoke_token(self, token, token_type='access'):
        """撤销 Token"""
        key = f'oauth:{token_type}_token:{token}'
        self.client.delete(key)

# 使用示例
store = OAuth2TokenStore()

# 创建 tokens
access_token = store.create_access_token(
    user_id='user123',
    client_id='client_app',
    scopes=['read', 'write']
)

refresh_token = store.create_refresh_token(
    user_id='user123',
    client_id='client_app',
    scopes=['read', 'write']
)

print(f"Access Token: {access_token}")
print(f"Refresh Token: {refresh_token}")

# 验证 token
token_data = store.verify_access_token(access_token)
if token_data:
    print(f"Token valid for user: {token_data['user_id']}")

# 刷新 token
new_access_token = store.refresh_access_token(refresh_token)
print(f"New Access Token: {new_access_token}")
```

## 连接池配置

```python
import redis
from redis.connection import ConnectionPool

# 创建连接池
pool = ConnectionPool(
    host='localhost',
    port=6380,
    max_connections=50,
    decode_responses=True,
    socket_timeout=5,
    socket_connect_timeout=5,
    retry_on_timeout=True
)

# 使用连接池
client = redis.Redis(connection_pool=pool)

# 现在可以安全地在多线程环境中使用
def worker_function():
    client.setex('key', 60, 'value')
    value = client.get('key')
    return value
```

## 异步支持（Python 3.8+）

```python
import asyncio
import redis.asyncio as aioredis
import json

async def async_example():
    # 创建异步客户端
    client = await aioredis.from_url(
        'redis://localhost:6380',
        decode_responses=True
    )

    try:
        # 设置会话
        session_data = {
            'user_id': 'user123',
            'username': 'async_user'
        }

        await client.setex(
            'oauth:token:async123',
            3600,
            json.dumps(session_data)
        )

        # 获取会话
        data = await client.get('oauth:token:async123')
        if data:
            session = json.loads(data)
            print(f"User: {session['username']}")

        # 删除会话
        await client.delete('oauth:token:async123')

    finally:
        await client.close()

# 运行异步代码
asyncio.run(async_example())
```

## 错误处理

```python
import redis
from redis.exceptions import (
    ConnectionError,
    TimeoutError,
    ResponseError
)

def safe_get_session(token):
    """安全地获取会话"""
    try:
        client = redis.Redis(
            host='localhost',
            port=6380,
            socket_timeout=5,
            decode_responses=True
        )

        key = f'oauth:token:{token}'
        data = client.get(key)

        if data:
            return json.loads(data)
        return None

    except ConnectionError:
        print("Cannot connect to TokenginX")
        return None
    except TimeoutError:
        print("Request timed out")
        return None
    except ResponseError as e:
        print(f"Redis error: {e}")
        return None
    except json.JSONDecodeError:
        print("Invalid session data format")
        return None
```

## 下一步

- 查看 [Python 生产环境指南](../production/python.md) 了解生产部署
- 查看 [OAuth 2.0/OIDC 集成指南](../protocols/oauth.md) 了解协议集成
- 查看 [API 参考文档](../reference/http-rest-api.md) 了解完整 API
