# OAuth 2.0/OIDC 集成指南

本指南详细说明如何使用 TokenginX 实现完整的 OAuth 2.0 和 OpenID Connect (OIDC) 认证授权系统。

## 概述

OAuth 2.0 是一个行业标准的授权框架,OpenID Connect 在 OAuth 2.0 基础上添加了身份认证层。TokenginX 提供高性能的会话存储,支持所有 OAuth 2.0 授权流程。

### 支持的授权流程

- **Authorization Code Flow** (授权码模式) - 推荐,最安全
- **Authorization Code Flow with PKCE** - 移动应用和 SPA 推荐
- **Client Credentials Flow** - 机器对机器通信
- **Implicit Flow** - 已弃用,不推荐
- **Resource Owner Password Credentials Flow** - 特殊场景

### 支持的 Token 类型

- **Access Token** - 访问令牌
- **Refresh Token** - 刷新令牌
- **ID Token** (OIDC) - 身份令牌
- **Authorization Code** - 授权码

## 数据结构设计

### Access Token 存储

```
Key: oauth:access_token:{token_id}
Value: {
  "token_type": "Bearer",
  "client_id": "client_app_001",
  "user_id": "user_12345",
  "scope": "read write profile",
  "issued_at": 1700000000,
  "expires_at": 1700003600,
  "refresh_token_id": "refresh_abc123"
}
TTL: 3600 seconds (1 hour)
```

### Refresh Token 存储

```
Key: oauth:refresh_token:{token_id}
Value: {
  "client_id": "client_app_001",
  "user_id": "user_12345",
  "scope": "read write profile",
  "issued_at": 1700000000,
  "expires_at": 1702678400,
  "access_token_id": "access_xyz789"
}
TTL: 2678400 seconds (31 days)
```

### Authorization Code 存储

```
Key: oauth:auth_code:{code}
Value: {
  "client_id": "client_app_001",
  "redirect_uri": "https://app.example.com/callback",
  "user_id": "user_12345",
  "scope": "read write",
  "code_challenge": "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM",
  "code_challenge_method": "S256",
  "state": "random_state_value",
  "nonce": "random_nonce_value",
  "issued_at": 1700000000
}
TTL: 600 seconds (10 minutes)
```

### ID Token Claims 存储 (OIDC)

```
Key: oauth:id_token:{token_id}
Value: {
  "iss": "https://auth.example.com",
  "sub": "user_12345",
  "aud": "client_app_001",
  "exp": 1700003600,
  "iat": 1700000000,
  "nonce": "random_nonce_value",
  "name": "John Doe",
  "email": "john@example.com",
  "email_verified": true,
  "picture": "https://example.com/avatar.jpg"
}
TTL: 3600 seconds (1 hour)
```

## Authorization Code Flow 实现

### 1. 授权端点 (/authorize)

```go
// Go 实现示例
func handleAuthorize(w http.ResponseWriter, r *http.Request) {
    // 解析请求参数
    clientID := r.URL.Query().Get("client_id")
    redirectURI := r.URL.Query().Get("redirect_uri")
    scope := r.URL.Query().Get("scope")
    state := r.URL.Query().Get("state")
    responseType := r.URL.Query().Get("response_type")
    codeChallenge := r.URL.Query().Get("code_challenge")
    codeChallengeMethod := r.URL.Query().Get("code_challenge_method")
    nonce := r.URL.Query().Get("nonce")

    // 验证客户端
    client, err := validateClient(clientID, redirectURI)
    if err != nil {
        http.Error(w, "Invalid client", http.StatusBadRequest)
        return
    }

    // 验证 response_type
    if responseType != "code" {
        redirectError(w, redirectURI, "unsupported_response_type", state)
        return
    }

    // 用户认证(假设已完成)
    userID := getUserFromSession(r)
    if userID == "" {
        // 重定向到登录页面
        redirectToLogin(w, r)
        return
    }

    // 用户授权确认(假设已完成)
    if !userHasConsented(userID, clientID, scope) {
        // 显示授权同意页面
        showConsentPage(w, client, scope)
        return
    }

    // 生成授权码
    code := generateSecureToken(32)

    // 存储授权码到 TokenginX
    authCodeData := map[string]interface{}{
        "client_id":             clientID,
        "redirect_uri":          redirectURI,
        "user_id":               userID,
        "scope":                 scope,
        "code_challenge":        codeChallenge,
        "code_challenge_method": codeChallengeMethod,
        "state":                 state,
        "nonce":                 nonce,
        "issued_at":             time.Now().Unix(),
    }

    key := fmt.Sprintf("oauth:auth_code:%s", code)
    value, _ := json.Marshal(authCodeData)

    ctx := context.Background()
    err = redisClient.Set(ctx, key, value, 10*time.Minute).Err()
    if err != nil {
        http.Error(w, "Internal error", http.StatusInternalServerError)
        return
    }

    // 重定向到回调 URI
    callbackURL := fmt.Sprintf("%s?code=%s&state=%s", redirectURI, code, state)
    http.Redirect(w, r, callbackURL, http.StatusFound)
}
```

### 2. Token 端点 (/token)

```go
func handleToken(w http.ResponseWriter, r *http.Request) {
    grantType := r.FormValue("grant_type")

    switch grantType {
    case "authorization_code":
        handleAuthorizationCodeGrant(w, r)
    case "refresh_token":
        handleRefreshTokenGrant(w, r)
    case "client_credentials":
        handleClientCredentialsGrant(w, r)
    default:
        writeError(w, "unsupported_grant_type", http.StatusBadRequest)
    }
}

func handleAuthorizationCodeGrant(w http.ResponseWriter, r *http.Request) {
    // 解析参数
    code := r.FormValue("code")
    redirectURI := r.FormValue("redirect_uri")
    codeVerifier := r.FormValue("code_verifier")
    clientID := r.FormValue("client_id")
    clientSecret := r.FormValue("client_secret")

    // 验证客户端
    if !validateClientCredentials(clientID, clientSecret) {
        writeError(w, "invalid_client", http.StatusUnauthorized)
        return
    }

    // 从 TokenginX 获取授权码
    ctx := context.Background()
    key := fmt.Sprintf("oauth:auth_code:%s", code)

    authCodeJSON, err := redisClient.Get(ctx, key).Result()
    if err == redis.Nil {
        writeError(w, "invalid_grant", http.StatusBadRequest)
        return
    }

    var authCodeData map[string]interface{}
    json.Unmarshal([]byte(authCodeJSON), &authCodeData)

    // 验证 redirect_uri
    if authCodeData["redirect_uri"] != redirectURI {
        writeError(w, "invalid_grant", http.StatusBadRequest)
        return
    }

    // 验证 PKCE (如果使用)
    if codeChallenge := authCodeData["code_challenge"]; codeChallenge != "" {
        if !verifyPKCE(codeVerifier, codeChallenge.(string),
                       authCodeData["code_challenge_method"].(string)) {
            writeError(w, "invalid_grant", http.StatusBadRequest)
            return
        }
    }

    // 删除已使用的授权码(一次性使用)
    redisClient.Del(ctx, key)

    // 生成 Access Token 和 Refresh Token
    accessTokenID := generateSecureToken(32)
    refreshTokenID := generateSecureToken(32)

    userID := authCodeData["user_id"].(string)
    scope := authCodeData["scope"].(string)

    // 存储 Access Token
    accessTokenData := map[string]interface{}{
        "token_type":       "Bearer",
        "client_id":        clientID,
        "user_id":          userID,
        "scope":            scope,
        "issued_at":        time.Now().Unix(),
        "expires_at":       time.Now().Add(1 * time.Hour).Unix(),
        "refresh_token_id": refreshTokenID,
    }

    accessKey := fmt.Sprintf("oauth:access_token:%s", accessTokenID)
    accessValue, _ := json.Marshal(accessTokenData)
    redisClient.Set(ctx, accessKey, accessValue, 1*time.Hour)

    // 存储 Refresh Token
    refreshTokenData := map[string]interface{}{
        "client_id":        clientID,
        "user_id":          userID,
        "scope":            scope,
        "issued_at":        time.Now().Unix(),
        "expires_at":       time.Now().Add(30 * 24 * time.Hour).Unix(),
        "access_token_id":  accessTokenID,
    }

    refreshKey := fmt.Sprintf("oauth:refresh_token:%s", refreshTokenID)
    refreshValue, _ := json.Marshal(refreshTokenData)
    redisClient.Set(ctx, refreshKey, refreshValue, 30*24*time.Hour)

    // 生成 ID Token (OIDC)
    var idToken string
    if strings.Contains(scope, "openid") {
        idToken = generateIDToken(userID, clientID, authCodeData["nonce"].(string))
    }

    // 返回响应
    response := map[string]interface{}{
        "access_token":  accessTokenID,
        "token_type":    "Bearer",
        "expires_in":    3600,
        "refresh_token": refreshTokenID,
        "scope":         scope,
    }

    if idToken != "" {
        response["id_token"] = idToken
    }

    w.Header().Set("Content-Type", "application/json")
    w.Header().Set("Cache-Control", "no-store")
    w.Header().Set("Pragma", "no-cache")
    json.NewEncoder(w).Encode(response)
}
```

### 3. Token 刷新

```go
func handleRefreshTokenGrant(w http.ResponseWriter, r *http.Request) {
    refreshToken := r.FormValue("refresh_token")
    scope := r.FormValue("scope")
    clientID := r.FormValue("client_id")
    clientSecret := r.FormValue("client_secret")

    // 验证客户端
    if !validateClientCredentials(clientID, clientSecret) {
        writeError(w, "invalid_client", http.StatusUnauthorized)
        return
    }

    // 从 TokenginX 获取 Refresh Token
    ctx := context.Background()
    key := fmt.Sprintf("oauth:refresh_token:%s", refreshToken)

    refreshTokenJSON, err := redisClient.Get(ctx, key).Result()
    if err == redis.Nil {
        writeError(w, "invalid_grant", http.StatusBadRequest)
        return
    }

    var refreshTokenData map[string]interface{}
    json.Unmarshal([]byte(refreshTokenJSON), &refreshTokenData)

    // 验证客户端匹配
    if refreshTokenData["client_id"] != clientID {
        writeError(w, "invalid_grant", http.StatusBadRequest)
        return
    }

    // 验证作用域(如果提供)
    originalScope := refreshTokenData["scope"].(string)
    if scope != "" && !isScopeSubset(scope, originalScope) {
        writeError(w, "invalid_scope", http.StatusBadRequest)
        return
    }
    if scope == "" {
        scope = originalScope
    }

    // 撤销旧的 Access Token
    oldAccessTokenID := refreshTokenData["access_token_id"].(string)
    redisClient.Del(ctx, fmt.Sprintf("oauth:access_token:%s", oldAccessTokenID))

    // 生成新的 Access Token
    newAccessTokenID := generateSecureToken(32)
    userID := refreshTokenData["user_id"].(string)

    accessTokenData := map[string]interface{}{
        "token_type":       "Bearer",
        "client_id":        clientID,
        "user_id":          userID,
        "scope":            scope,
        "issued_at":        time.Now().Unix(),
        "expires_at":       time.Now().Add(1 * time.Hour).Unix(),
        "refresh_token_id": refreshToken,
    }

    accessKey := fmt.Sprintf("oauth:access_token:%s", newAccessTokenID)
    accessValue, _ := json.Marshal(accessTokenData)
    redisClient.Set(ctx, accessKey, accessValue, 1*time.Hour)

    // 更新 Refresh Token 关联
    refreshTokenData["access_token_id"] = newAccessTokenID
    refreshValue, _ := json.Marshal(refreshTokenData)
    redisClient.Set(ctx, key, refreshValue, 30*24*time.Hour)

    // 返回响应
    response := map[string]interface{}{
        "access_token": newAccessTokenID,
        "token_type":   "Bearer",
        "expires_in":   3600,
        "scope":        scope,
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
}
```

## Token 内省 (RFC 7662)

```go
func handleIntrospection(w http.ResponseWriter, r *http.Request) {
    token := r.FormValue("token")
    tokenTypeHint := r.FormValue("token_type_hint")

    // 验证调用者身份(Resource Server 认证)
    if !validateResourceServer(r) {
        writeError(w, "invalid_client", http.StatusUnauthorized)
        return
    }

    ctx := context.Background()
    var tokenData map[string]interface{}
    var found bool

    // 根据 hint 优先查找
    if tokenTypeHint == "access_token" || tokenTypeHint == "" {
        key := fmt.Sprintf("oauth:access_token:%s", token)
        if data, err := redisClient.Get(ctx, key).Result(); err == nil {
            json.Unmarshal([]byte(data), &tokenData)
            found = true
        }
    }

    if !found && (tokenTypeHint == "refresh_token" || tokenTypeHint == "") {
        key := fmt.Sprintf("oauth:refresh_token:%s", token)
        if data, err := redisClient.Get(ctx, key).Result(); err == nil {
            json.Unmarshal([]byte(data), &tokenData)
            found = true
        }
    }

    if !found {
        // Token 不存在或已过期
        response := map[string]interface{}{"active": false}
        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(response)
        return
    }

    // 返回 Token 信息
    response := map[string]interface{}{
        "active":     true,
        "scope":      tokenData["scope"],
        "client_id":  tokenData["client_id"],
        "username":   tokenData["user_id"],
        "token_type": "Bearer",
        "exp":        tokenData["expires_at"],
        "iat":        tokenData["issued_at"],
        "sub":        tokenData["user_id"],
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
}
```

## Token 撤销 (RFC 7009)

```go
func handleRevocation(w http.ResponseWriter, r *http.Request) {
    token := r.FormValue("token")
    tokenTypeHint := r.FormValue("token_type_hint")
    clientID := r.FormValue("client_id")
    clientSecret := r.FormValue("client_secret")

    // 验证客户端
    if !validateClientCredentials(clientID, clientSecret) {
        writeError(w, "invalid_client", http.StatusUnauthorized)
        return
    }

    ctx := context.Background()

    // 撤销 Token
    if tokenTypeHint == "access_token" || tokenTypeHint == "" {
        key := fmt.Sprintf("oauth:access_token:%s", token)
        redisClient.Del(ctx, key)
    }

    if tokenTypeHint == "refresh_token" || tokenTypeHint == "" {
        key := fmt.Sprintf("oauth:refresh_token:%s", token)

        // 获取关联的 Access Token 并一起撤销
        if data, err := redisClient.Get(ctx, key).Result(); err == nil {
            var tokenData map[string]interface{}
            json.Unmarshal([]byte(data), &tokenData)

            if accessTokenID, ok := tokenData["access_token_id"].(string); ok {
                redisClient.Del(ctx, fmt.Sprintf("oauth:access_token:%s", accessTokenID))
            }
        }

        redisClient.Del(ctx, key)
    }

    // RFC 7009: 成功撤销返回 200
    w.WriteHeader(http.StatusOK)
}
```

## PKCE 实现

```go
// 验证 PKCE
func verifyPKCE(verifier, challenge, method string) bool {
    if method == "S256" {
        // SHA256 哈希
        hash := sha256.Sum256([]byte(verifier))
        computed := base64.RawURLEncoding.EncodeToString(hash[:])
        return computed == challenge
    } else if method == "plain" {
        // 明文比较(不推荐)
        return verifier == challenge
    }
    return false
}

// 生成 code_verifier
func generateCodeVerifier() string {
    bytes := make([]byte, 32)
    rand.Read(bytes)
    return base64.RawURLEncoding.EncodeToString(bytes)
}

// 生成 code_challenge
func generateCodeChallenge(verifier string) string {
    hash := sha256.Sum256([]byte(verifier))
    return base64.RawURLEncoding.EncodeToString(hash[:])
}
```

## OpenID Connect (OIDC) 支持

### UserInfo 端点

```go
func handleUserInfo(w http.ResponseWriter, r *http.Request) {
    // 从 Authorization header 获取 Access Token
    authHeader := r.Header.Get("Authorization")
    if !strings.HasPrefix(authHeader, "Bearer ") {
        writeError(w, "invalid_token", http.StatusUnauthorized)
        return
    }

    accessToken := strings.TrimPrefix(authHeader, "Bearer ")

    // 从 TokenginX 获取 Token 信息
    ctx := context.Background()
    key := fmt.Sprintf("oauth:access_token:%s", accessToken)

    tokenJSON, err := redisClient.Get(ctx, key).Result()
    if err == redis.Nil {
        writeError(w, "invalid_token", http.StatusUnauthorized)
        return
    }

    var tokenData map[string]interface{}
    json.Unmarshal([]byte(tokenJSON), &tokenData)

    // 验证作用域包含 openid
    scope := tokenData["scope"].(string)
    if !strings.Contains(scope, "openid") {
        writeError(w, "insufficient_scope", http.StatusForbidden)
        return
    }

    // 获取用户信息
    userID := tokenData["user_id"].(string)
    userInfo := getUserInfo(userID, scope)

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(userInfo)
}

func getUserInfo(userID, scope string) map[string]interface{} {
    // 从数据库获取用户信息
    user := fetchUserFromDB(userID)

    // 根据 scope 返回相应的 claims
    info := map[string]interface{}{
        "sub": userID,
    }

    if strings.Contains(scope, "profile") {
        info["name"] = user.Name
        info["given_name"] = user.GivenName
        info["family_name"] = user.FamilyName
        info["picture"] = user.Picture
    }

    if strings.Contains(scope, "email") {
        info["email"] = user.Email
        info["email_verified"] = user.EmailVerified
    }

    if strings.Contains(scope, "phone") {
        info["phone_number"] = user.PhoneNumber
        info["phone_number_verified"] = user.PhoneVerified
    }

    return info
}
```

### ID Token 生成 (JWT)

```go
import "github.com/golang-jwt/jwt/v5"

func generateIDToken(userID, clientID, nonce string) string {
    claims := jwt.MapClaims{
        "iss":   "https://auth.example.com",
        "sub":   userID,
        "aud":   clientID,
        "exp":   time.Now().Add(1 * time.Hour).Unix(),
        "iat":   time.Now().Unix(),
        "nonce": nonce,
    }

    // 添加用户信息
    user := fetchUserFromDB(userID)
    claims["name"] = user.Name
    claims["email"] = user.Email
    claims["email_verified"] = user.EmailVerified
    claims["picture"] = user.Picture

    token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)

    // 使用私钥签名
    privateKey := loadPrivateKey()
    signedToken, _ := token.SignedString(privateKey)

    return signedToken
}
```

## 安全最佳实践

### 1. Token 安全

- ✅ 使用加密安全的随机数生成器
- ✅ Access Token 有效期 ≤ 1 小时
- ✅ Refresh Token 有效期 ≤ 90 天
- ✅ 授权码有效期 ≤ 10 分钟
- ✅ 授权码仅可使用一次
- ✅ 实施 Token 绑定(Token Binding)

### 2. PKCE 强制使用

```go
// 对公共客户端强制要求 PKCE
if client.IsPublic && codeChallenge == "" {
    writeError(w, "invalid_request", http.StatusBadRequest)
    return
}
```

### 3. 状态参数验证

```go
// 客户端必须验证 state 参数
if receivedState != originalState {
    return errors.New("state mismatch - possible CSRF attack")
}
```

### 4. Nonce 验证 (OIDC)

```go
// 验证 ID Token 中的 nonce
if idTokenNonce != originalNonce {
    return errors.New("nonce mismatch")
}
```

### 5. Redirect URI 严格验证

```go
func validateRedirectURI(provided, registered string) bool {
    // 必须完全匹配(包括查询参数)
    return provided == registered
}
```

## 客户端示例

### ASP.NET Core 客户端

```csharp
// Startup.cs
services.AddAuthentication(options =>
{
    options.DefaultScheme = "Cookies";
    options.DefaultChallengeScheme = "oidc";
})
.AddCookie("Cookies")
.AddOpenIdConnect("oidc", options =>
{
    options.Authority = "https://auth.example.com";
    options.ClientId = "my-client-id";
    options.ClientSecret = "my-client-secret";
    options.ResponseType = "code";
    options.SaveTokens = true;
    options.GetClaimsFromUserInfoEndpoint = true;

    options.Scope.Add("openid");
    options.Scope.Add("profile");
    options.Scope.Add("email");
});
```

### JavaScript (SPA) 客户端

```javascript
// 使用 PKCE
async function login() {
    // 生成 code_verifier 和 code_challenge
    const codeVerifier = generateRandomString(128);
    const codeChallenge = await generateCodeChallenge(codeVerifier);

    sessionStorage.setItem('code_verifier', codeVerifier);

    const params = new URLSearchParams({
        client_id: 'my-spa-client',
        redirect_uri: 'https://myapp.com/callback',
        response_type: 'code',
        scope: 'openid profile email',
        code_challenge: codeChallenge,
        code_challenge_method: 'S256',
        state: generateRandomString(32)
    });

    window.location = `https://auth.example.com/authorize?${params}`;
}

async function handleCallback() {
    const params = new URLSearchParams(window.location.search);
    const code = params.get('code');
    const codeVerifier = sessionStorage.getItem('code_verifier');

    const response = await fetch('https://auth.example.com/token', {
        method: 'POST',
        headers: {'Content-Type': 'application/x-www-form-urlencoded'},
        body: new URLSearchParams({
            grant_type: 'authorization_code',
            code: code,
            redirect_uri: 'https://myapp.com/callback',
            client_id: 'my-spa-client',
            code_verifier: codeVerifier
        })
    });

    const tokens = await response.json();
    // 存储 tokens
}
```

## 性能优化

### 1. 批量 Token 验证

```go
func validateMultipleTokens(tokens []string) map[string]bool {
    ctx := context.Background()
    pipe := redisClient.Pipeline()

    cmds := make([]*redis.StringCmd, len(tokens))
    for i, token := range tokens {
        key := fmt.Sprintf("oauth:access_token:%s", token)
        cmds[i] = pipe.Get(ctx, key)
    }

    pipe.Exec(ctx)

    result := make(map[string]bool)
    for i, cmd := range cmds {
        _, err := cmd.Result()
        result[tokens[i]] = (err == nil)
    }

    return result
}
```

### 2. Token 缓存

在应用层缓存 Token 验证结果(短时间):

```go
var tokenCache = cache.New(5*time.Minute, 10*time.Minute)

func isTokenValid(token string) bool {
    if cached, found := tokenCache.Get(token); found {
        return cached.(bool)
    }

    valid := checkTokenInTokenginX(token)
    tokenCache.Set(token, valid, cache.DefaultExpiration)

    return valid
}
```

## 监控指标

```go
// Prometheus 指标
var (
    tokenIssued = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "oauth_tokens_issued_total",
            Help: "Total number of tokens issued",
        },
        []string{"grant_type", "client_id"},
    )

    tokenIntrospection = prometheus.NewHistogram(
        prometheus.HistogramOpts{
            Name:    "oauth_introspection_duration_seconds",
            Help:    "Token introspection duration",
            Buckets: prometheus.ExponentialBuckets(0.001, 2, 10),
        },
    )
)
```

## 下一步

- 查看 [SAML 2.0 集成指南](./saml.md)
- 查看 [CAS 集成指南](./cas.md)
- 配置 [TLS/mTLS](../security/tls-mtls.md)
- 了解 [防重放攻击](../security/anti-replay.md)
