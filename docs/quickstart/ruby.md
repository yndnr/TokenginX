# Ruby 快速指南

本指南帮助 Ruby 开发者快速集成 TokenginX 进行会话管理。

## 前置要求

- Ruby 2.7+ (推荐 3.0+)
- Bundler
- TokenginX 服务器运行中

## 安装客户端库

TokenginX 支持多种 Ruby 客户端方式：

### 方式 1: Redis 客户端（推荐）

将以下内容添加到 `Gemfile`:

```ruby
gem 'redis', '~> 5.0'
gem 'connection_pool', '~> 2.4'
```

然后运行：

```bash
bundle install
```

### 方式 2: HTTP 客户端

```ruby
gem 'httparty', '~> 0.21'
# 或
gem 'faraday', '~> 2.7'
```

## 快速开始

### 使用 Redis 客户端

```ruby
require 'redis'
require 'json'

# 连接到 TokenginX
client = Redis.new(
  host: 'localhost',
  port: 6380,
  timeout: 5,
  reconnect_attempts: 3
)

# 设置会话（3600秒后过期）
session_data = {
  user_id: 'user123',
  username: 'john_doe',
  email: 'john@example.com',
  roles: ['user', 'admin'],
  created_at: Time.now.iso8601
}

client.setex(
  'oauth:token:abc123',
  3600, # TTL 秒数
  session_data.to_json
)

puts 'Session created'

# 获取会话
token_data = client.get('oauth:token:abc123')
if token_data
  session = JSON.parse(token_data, symbolize_names: true)
  puts "User: #{session[:username]}"
else
  puts 'Token not found or expired'
end

# 检查会话是否存在
exists = client.exists?('oauth:token:abc123')
puts "Token exists: #{exists}"

# 获取剩余 TTL
ttl = client.ttl('oauth:token:abc123')
puts "Token expires in: #{ttl} seconds"

# 删除会话
client.del('oauth:token:abc123')
puts 'Session deleted'
```

### 使用 HTTP/REST 客户端 (HTTParty)

```ruby
require 'httparty'
require 'json'

class TokenginxClient
  include HTTParty
  base_uri 'http://localhost:8080/api/v1'

  def initialize
    self.class.default_timeout(5)
  end

  def set_session(key, value, ttl = 3600)
    response = self.class.post('/sessions',
      body: {
        key: key,
        value: value,
        ttl: ttl
      }.to_json,
      headers: { 'Content-Type' => 'application/json' }
    )

    raise "Failed to set session: #{response.code}" unless response.success?

    response.parsed_response
  end

  def get_session(key)
    response = self.class.get("/sessions/#{key}")

    return nil if response.code == 404

    raise "Failed to get session: #{response.code}" unless response.success?

    response.parsed_response
  end

  def delete_session(key)
    response = self.class.delete("/sessions/#{key}")

    raise "Failed to delete session: #{response.code}" unless response.success?

    response.parsed_response
  end

  def exists?(key)
    response = self.class.head("/sessions/#{key}")
    response.success?
  end
end

# 使用示例
client = TokenginxClient.new

# 设置会话
session_data = {
  user_id: 'user123',
  username: 'john_doe',
  email: 'john@example.com'
}

client.set_session('oauth:token:abc123', session_data, 3600)

# 获取会话
session = client.get_session('oauth:token:abc123')
puts "User: #{session['value']['username']}" if session

# 删除会话
client.delete_session('oauth:token:abc123')
```

## Rails 集成

### 自定义会话存储

```ruby
# config/initializers/session_store.rb
require 'redis'
require 'json'

module ActionDispatch
  module Session
    class TokenginxStore < AbstractSecureStore
      def initialize(app, options = {})
        super

        @redis = Redis.new(
          host: options[:host] || 'localhost',
          port: options[:port] || 6380,
          timeout: options[:timeout] || 5
        )

        @key_prefix = options[:key_prefix] || 'rails:session:'
        @default_ttl = options[:expire_after] || 3600
      end

      private

      def find_session(env, sid)
        unless sid && (session = get_session_with_fallback(sid))
          sid, session = generate_sid, {}
        end
        [sid, session]
      end

      def write_session(env, sid, session, options)
        key = session_key(sid)
        ttl = options[:expire_after] || @default_ttl

        @redis.setex(key, ttl, session.to_json)

        sid
      rescue Redis::BaseError => e
        Rails.logger.error "Session write failed: #{e.message}"
        false
      end

      def delete_session(env, sid, options)
        @redis.del(session_key(sid))
        generate_sid
      rescue Redis::BaseError => e
        Rails.logger.error "Session delete failed: #{e.message}"
        nil
      end

      def get_session_with_fallback(sid)
        data = @redis.get(session_key(sid))
        return nil unless data

        JSON.parse(data)
      rescue JSON::ParserError, Redis::BaseError => e
        Rails.logger.error "Session read failed: #{e.message}"
        nil
      end

      def session_key(sid)
        "#{@key_prefix}#{sid}"
      end
    end
  end
end
```

### 配置 Rails

```ruby
# config/application.rb
config.session_store :tokenginx_store,
  key: '_myapp_session',
  host: ENV.fetch('TOKENGINX_HOST', 'localhost'),
  port: ENV.fetch('TOKENGINX_PORT', 6380).to_i,
  expire_after: 1.hour
```

### 认证模块

```ruby
# app/controllers/concerns/authentication.rb
module Authentication
  extend ActiveSupport::Concern

  included do
    before_action :authenticate_user!
  end

  private

  def authenticate_user!
    token = extract_token_from_header

    unless token && (session = get_session_from_token(token))
      render json: { error: 'Unauthorized' }, status: :unauthorized
      return
    end

    @current_user_session = session
    @current_token = token
  end

  def extract_token_from_header
    auth_header = request.headers['Authorization']
    return nil unless auth_header&.start_with?('Bearer ')

    auth_header.split(' ').last
  end

  def get_session_from_token(token)
    redis = Redis.new(
      host: ENV.fetch('TOKENGINX_HOST', 'localhost'),
      port: ENV.fetch('TOKENGINX_PORT', 6380).to_i
    )

    data = redis.get("oauth:token:#{token}")
    return nil unless data

    JSON.parse(data, symbolize_names: true)
  rescue Redis::BaseError, JSON::ParserError => e
    Rails.logger.error "Auth error: #{e.message}"
    nil
  end

  def current_user
    @current_user_session
  end

  def current_token
    @current_token
  end
end
```

### 认证控制器

```ruby
# app/controllers/api/auth_controller.rb
class Api::AuthController < ApplicationController
  skip_before_action :authenticate_user!, only: [:login]

  def login
    username = params[:username]
    password = params[:password]

    # 验证用户（示例）
    if username == 'admin' && password == 'password'
      # 生成 token
      token = SecureRandom.hex(32)

      # 创建会话数据
      session_data = {
        user_id: 'user123',
        username: username,
        roles: ['admin'],
        created_at: Time.now.iso8601
      }

      # 存储到 TokenginX（1小时过期）
      redis = Redis.new(
        host: ENV.fetch('TOKENGINX_HOST', 'localhost'),
        port: ENV.fetch('TOKENGINX_PORT', 6380).to_i
      )

      redis.setex(
        "oauth:token:#{token}",
        3600,
        session_data.to_json
      )

      render json: {
        token: token,
        expires_in: 3600
      }
    else
      render json: { error: 'Invalid credentials' }, status: :unauthorized
    end
  end

  def profile
    render json: {
      user_id: current_user[:user_id],
      username: current_user[:username],
      roles: current_user[:roles]
    }
  end

  def logout
    redis = Redis.new(
      host: ENV.fetch('TOKENGINX_HOST', 'localhost'),
      port: ENV.fetch('TOKENGINX_PORT', 6380).to_i
    )

    redis.del("oauth:token:#{current_token}")

    render json: { message: 'Logged out successfully' }
  end
end
```

## Sinatra 集成

```ruby
require 'sinatra'
require 'redis'
require 'json'
require 'securerandom'

# TokenginX 客户端
set :tokenginx, Redis.new(
  host: ENV.fetch('TOKENGINX_HOST', 'localhost'),
  port: ENV.fetch('TOKENGINX_PORT', 6380).to_i
)

# 认证辅助方法
helpers do
  def authenticate!
    auth_header = request.env['HTTP_AUTHORIZATION']

    unless auth_header && auth_header.start_with?('Bearer ')
      halt 401, json(error: 'Unauthorized')
    end

    token = auth_header.split(' ').last
    session_key = "oauth:token:#{token}"

    data = settings.tokenginx.get(session_key)

    unless data
      halt 401, json(error: 'Invalid or expired token')
    end

    @current_user = JSON.parse(data, symbolize_names: true)
    @current_token = token
  end

  def current_user
    @current_user
  end

  def json(data)
    content_type :json
    data.to_json
  end
end

# 登录端点
post '/api/login' do
  request.body.rewind
  data = JSON.parse(request.body.read, symbolize_names: true)

  username = data[:username]
  password = data[:password]

  # 验证用户（示例）
  if username == 'admin' && password == 'password'
    # 生成 token
    token = SecureRandom.hex(32)

    # 创建会话数据
    session_data = {
      user_id: 'user123',
      username: username,
      roles: ['admin'],
      created_at: Time.now.iso8601
    }

    # 存储到 TokenginX
    settings.tokenginx.setex(
      "oauth:token:#{token}",
      3600,
      session_data.to_json
    )

    json(
      token: token,
      expires_in: 3600
    )
  else
    status 401
    json(error: 'Invalid credentials')
  end
end

# 受保护的端点
get '/api/profile' do
  authenticate!

  json(
    user_id: current_user[:user_id],
    username: current_user[:username],
    roles: current_user[:roles]
  )
end

# 登出端点
post '/api/logout' do
  authenticate!

  settings.tokenginx.del("oauth:token:#{@current_token}")

  json(message: 'Logged out successfully')
end
```

## OAuth 2.0 Token Store

```ruby
require 'redis'
require 'json'
require 'securerandom'

class OAuth2TokenStore
  def initialize(host: 'localhost', port: 6380)
    @redis = Redis.new(host: host, port: port)
  end

  # 创建 Access Token
  def create_access_token(user_id, client_id, scopes, ttl: 3600)
    token = SecureRandom.hex(32)

    token_data = {
      user_id: user_id,
      client_id: client_id,
      scopes: scopes,
      token_type: 'Bearer',
      created_at: Time.now.iso8601
    }

    key = "oauth:access_token:#{token}"
    @redis.setex(key, ttl, token_data.to_json)

    token
  end

  # 创建 Refresh Token（30天）
  def create_refresh_token(user_id, client_id, scopes, ttl: 2_592_000)
    token = SecureRandom.hex(32)

    token_data = {
      user_id: user_id,
      client_id: client_id,
      scopes: scopes,
      token_type: 'refresh',
      created_at: Time.now.iso8601
    }

    key = "oauth:refresh_token:#{token}"
    @redis.setex(key, ttl, token_data.to_json)

    token
  end

  # 验证 Access Token
  def verify_access_token(token)
    key = "oauth:access_token:#{token}"
    data = @redis.get(key)

    return nil unless data

    JSON.parse(data, symbolize_names: true)
  end

  # 刷新 Access Token
  def refresh_access_token(refresh_token)
    key = "oauth:refresh_token:#{refresh_token}"
    data = @redis.get(key)

    return nil unless data

    refresh_data = JSON.parse(data, symbolize_names: true)

    # 创建新的 Access Token
    create_access_token(
      refresh_data[:user_id],
      refresh_data[:client_id],
      refresh_data[:scopes]
    )
  end

  # 撤销 Token
  def revoke_token(token, token_type: 'access')
    key = "oauth:#{token_type}_token:#{token}"
    @redis.del(key)
  end

  # 创建 Authorization Code
  def create_authorization_code(user_id, client_id, redirect_uri, scopes, ttl: 300)
    code = SecureRandom.hex(32)

    code_data = {
      user_id: user_id,
      client_id: client_id,
      redirect_uri: redirect_uri,
      scopes: scopes,
      created_at: Time.now.iso8601
    }

    key = "oauth:code:#{code}"
    @redis.setex(key, ttl, code_data.to_json)

    code
  end

  # 验证并消费 Authorization Code（一次性使用）
  def consume_authorization_code(code)
    key = "oauth:code:#{code}"
    data = @redis.get(key)

    return nil unless data

    # 删除 code（一次性使用）
    @redis.del(key)

    JSON.parse(data, symbolize_names: true)
  end
end

# 使用示例
store = OAuth2TokenStore.new

# 创建 tokens
access_token = store.create_access_token(
  'user123',
  'client_app',
  ['read', 'write']
)

refresh_token = store.create_refresh_token(
  'user123',
  'client_app',
  ['read', 'write']
)

puts "Access Token: #{access_token}"
puts "Refresh Token: #{refresh_token}"

# 验证 token
token_data = store.verify_access_token(access_token)
if token_data
  puts "Token valid for user: #{token_data[:user_id]}"
end

# 刷新 token
new_access_token = store.refresh_access_token(refresh_token)
puts "New Access Token: #{new_access_token}"
```

## 连接池配置

```ruby
require 'redis'
require 'connection_pool'

# 创建连接池
TOKENGINX_POOL = ConnectionPool.new(size: 10, timeout: 5) do
  Redis.new(
    host: ENV.fetch('TOKENGINX_HOST', 'localhost'),
    port: ENV.fetch('TOKENGINX_PORT', 6380).to_i,
    timeout: 5,
    reconnect_attempts: 3
  )
end

# 使用连接池
def get_session(token)
  TOKENGINX_POOL.with do |redis|
    data = redis.get("oauth:token:#{token}")
    return nil unless data

    JSON.parse(data, symbolize_names: true)
  end
end

def set_session(token, data, ttl = 3600)
  TOKENGINX_POOL.with do |redis|
    redis.setex("oauth:token:#{token}", ttl, data.to_json)
  end
end
```

## 错误处理

```ruby
require 'redis'
require 'json'

def safe_get_session(token)
  redis = Redis.new(
    host: 'localhost',
    port: 6380,
    timeout: 5
  )

  key = "oauth:token:#{token}"
  data = redis.get(key)

  return nil unless data

  JSON.parse(data, symbolize_names: true)

rescue Redis::BaseConnectionError => e
  Rails.logger.error "Cannot connect to TokenginX: #{e.message}"
  nil
rescue Redis::TimeoutError => e
  Rails.logger.error "Request timed out: #{e.message}"
  nil
rescue Redis::CommandError => e
  Rails.logger.error "Redis error: #{e.message}"
  nil
rescue JSON::ParserError => e
  Rails.logger.error "Invalid session data format: #{e.message}"
  nil
ensure
  redis&.close
end
```

## Pipeline 批量操作

```ruby
# 批量设置会话
def batch_set_sessions(sessions)
  redis = Redis.new(host: 'localhost', port: 6380)

  redis.pipelined do |pipeline|
    sessions.each do |session|
      pipeline.setex(
        "oauth:token:#{session[:token]}",
        3600,
        session[:data].to_json
      )
    end
  end
end

# 批量获取会话
def batch_get_sessions(tokens)
  redis = Redis.new(host: 'localhost', port: 6380)

  keys = tokens.map { |token| "oauth:token:#{token}" }

  values = redis.mget(*keys)

  values.map do |value|
    value ? JSON.parse(value, symbolize_names: true) : nil
  end
end
```

## Lua 脚本

```ruby
# 使用 Lua 脚本实现原子操作
GET_AND_REFRESH_SCRIPT = <<~LUA
  local key = KEYS[1]
  local ttl = ARGV[1]
  local value = redis.call('GET', key)
  if value then
    redis.call('EXPIRE', key, ttl)
    return value
  else
    return nil
  end
LUA

def get_and_refresh_session(token, ttl = 3600)
  redis = Redis.new(host: 'localhost', port: 6380)

  key = "oauth:token:#{token}"
  result = redis.eval(GET_AND_REFRESH_SCRIPT, [key], [ttl])

  return nil unless result

  JSON.parse(result, symbolize_names: true)
end
```

## RSpec 测试

```ruby
# spec/support/tokenginx_helper.rb
module TokenginxHelper
  def mock_tokenginx
    @mock_redis = instance_double(Redis)
    allow(Redis).to receive(:new).and_return(@mock_redis)
    @mock_redis
  end

  def stub_valid_session(token, user_data)
    session_data = {
      user_id: user_data[:user_id],
      username: user_data[:username],
      roles: user_data[:roles]
    }.to_json

    allow(@mock_redis).to receive(:get)
      .with("oauth:token:#{token}")
      .and_return(session_data)
  end

  def stub_expired_session(token)
    allow(@mock_redis).to receive(:get)
      .with("oauth:token:#{token}")
      .and_return(nil)
  end
end

RSpec.configure do |config|
  config.include TokenginxHelper
end

# spec/controllers/api/auth_controller_spec.rb
require 'rails_helper'

RSpec.describe Api::AuthController, type: :controller do
  before do
    mock_tokenginx
  end

  describe 'POST #login' do
    it 'creates a session and returns token' do
      expect(@mock_redis).to receive(:setex)
        .with(/oauth:token:/, 3600, anything)

      post :login, params: { username: 'admin', password: 'password' }

      expect(response).to have_http_status(:success)
      expect(JSON.parse(response.body)).to have_key('token')
    end
  end

  describe 'GET #profile' do
    it 'returns user profile for valid token' do
      stub_valid_session('valid_token', {
        user_id: 'user123',
        username: 'john_doe',
        roles: ['admin']
      })

      request.headers['Authorization'] = 'Bearer valid_token'
      get :profile

      expect(response).to have_http_status(:success)
      json = JSON.parse(response.body)
      expect(json['username']).to eq('john_doe')
    end

    it 'returns unauthorized for invalid token' do
      stub_expired_session('invalid_token')

      request.headers['Authorization'] = 'Bearer invalid_token'
      get :profile

      expect(response).to have_http_status(:unauthorized)
    end
  end
end
```

## 下一步

- 查看 [Ruby 生产环境指南](../production/ruby.md) 了解生产部署
- 查看 [OAuth 2.0/OIDC 集成指南](../protocols/oauth.md) 了解协议集成
- 查看 [API 参考文档](../reference/http-rest-api.md) 了解完整 API
